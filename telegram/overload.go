package telegram

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

// ErrOverload is the sentinel error returned when the client cannot admit more
// work. Use errors.Is(err, ErrOverload) to detect overload and apply retry logic.
var ErrOverload = errors.New("telegram: client overloaded")

// OverloadError provides details about an overload rejection. It wraps
// ErrOverload so errors.Is works. Callers can inspect Info for metrics.
type OverloadError struct {
	Info OverloadInfo
}

// OverloadInfo carries diagnostic context about an overload rejection.
type OverloadInfo struct {
	QueueDepth      int
	InFlight        int
	Priority        int // 0=Low, 1=High
	AdmissionWaited time.Duration
	Reason          string // "low_priority_fast_fail" | "admission_deadline_exceeded"
}

func (e *OverloadError) Error() string {
	return "telegram: client overloaded (" + e.Info.Reason + ")"
}

func (e *OverloadError) Unwrap() error { return ErrOverload }

// ThrottleLevel represents the current degradation tier of the overload
// controller. As load increases, the level ascends: Normal → ThrottlingLow →
// Shedding → Critical.
type ThrottleLevel int

const (
	ThrottleNormal        ThrottleLevel = iota // all traffic admitted
	ThrottleThrottlingLow                      // low-priority RPCs fast-fail
	ThrottleShedding                           // + inbound queue shedding
	ThrottleCritical                           // high-priority admission deadline exceeded
)

// ResourceCeilings are configurable limits enforced by OverloadController and
// the OutboundBatcher. All-zero = disabled (backward compat).
type ResourceCeilings struct {
	MaxInFlightRPCs         int
	MaxInboundDepth         int
	MaxOutboundDepth        int
	MaxBackgroundGoroutines int
}

// OverloadConfig configures an OverloadController.
type OverloadConfig struct {
	Ceilings          ResourceCeilings
	AdmissionDeadline time.Duration // High-priority bounded wait; 0 = 5s default
	Log               *Logger
}

// OverloadController gates RPC admission by priority against ResourceCeilings.
// Low-priority RPCs fast-fail when at capacity; high-priority RPCs wait up to
// AdmissionDeadline for a slot. All counters are atomic for zero-contention
// admission checks.
//
// Ported conceptually from TDLib net/NetQueryDispatcher.h priority admission.
type OverloadController struct {
	maxInFlight       int
	admissionDeadline time.Duration
	log               *Logger

	// Semaphore: buffered channel of struct{}; capacity = maxInFlight.
	sem chan struct{}

	// Metrics (atomic).
	inFlight      atomic.Int64
	totalAdmitted atomic.Int64
	totalRejected atomic.Int64

	closed atomic.Bool
}

// NewOverloadController creates a controller. When Ceilings.MaxInFlightRPCs <= 0,
// the controller is disabled and Admit always succeeds (backward compat).
func NewOverloadController(cfg OverloadConfig) *OverloadController {
	maxInFlight := cfg.Ceilings.MaxInFlightRPCs
	if maxInFlight <= 0 {
		return &OverloadController{log: cfg.Log}
	}
	deadline := cfg.AdmissionDeadline
	if deadline <= 0 {
		deadline = 5 * time.Second
	}
	return &OverloadController{
		maxInFlight:       maxInFlight,
		admissionDeadline: deadline,
		log:               cfg.Log,
		sem:               make(chan struct{}, maxInFlight),
	}
}

// Enabled reports whether the controller is active (MaxInFlightRPCs > 0).
func (c *OverloadController) Enabled() bool { return c != nil && c.maxInFlight > 0 }

// Admit attempts to admit an RPC of the given priority. Returns a release
// function (MUST be deferred by the caller) and nil on success, or
// *OverloadError on rejection.
//
//   - PriorityLow: fast-fails immediately if at capacity.
//   - PriorityHigh: waits up to AdmissionDeadline for a slot; fails if exceeded.
func (c *OverloadController) Admit(ctx context.Context, priority int) (func(), error) {
	if c.closed.Load() {
		return nil, &OverloadError{Info: OverloadInfo{Reason: "controller_closed"}}
	}
	if !c.Enabled() {
		return func() {}, nil // disabled: always admit
	}

	// Low priority: non-blocking acquire.
	if priority == 0 { // PriorityLow
		select {
		case c.sem <- struct{}{}:
			c.inFlight.Add(1)
			c.totalAdmitted.Add(1)
			return c.release, nil
		default:
			c.totalRejected.Add(1)
			return nil, &OverloadError{Info: OverloadInfo{
				InFlight: int(c.inFlight.Load()),
				Priority: priority,
				Reason:   "low_priority_fast_fail",
			}}
		}
	}

	// High priority: bounded wait.
	waitStart := time.Now()
	timer := time.NewTimer(c.admissionDeadline)
	defer timer.Stop()
	select {
	case c.sem <- struct{}{}:
		c.inFlight.Add(1)
		c.totalAdmitted.Add(1)
		return c.release, nil
	case <-timer.C:
		c.totalRejected.Add(1)
		return nil, &OverloadError{Info: OverloadInfo{
			InFlight:        int(c.inFlight.Load()),
			Priority:        priority,
			AdmissionWaited: time.Since(waitStart),
			Reason:          "admission_deadline_exceeded",
		}}
	case <-ctx.Done():
		c.totalRejected.Add(1)
		return nil, ctx.Err()
	}
}

func (c *OverloadController) release() {
	<-c.sem
	c.inFlight.Add(-1)
}

// ThrottleLevel returns the current degradation tier based on in-flight count.
func (c *OverloadController) ThrottleLevel() ThrottleLevel {
	if !c.Enabled() {
		return ThrottleNormal
	}
	inFlight := int(c.inFlight.Load())
	switch {
	case inFlight >= c.maxInFlight:
		// At capacity — low-priority fast-failing.
		return ThrottleThrottlingLow
	case inFlight >= c.maxInFlight*3/4:
		return ThrottleShedding
	default:
		return ThrottleNormal
	}
}

// OverloadSnapshot is a read-only view of the controller's state.
type OverloadSnapshot struct {
	InFlight       int64
	TotalAdmitted  int64
	TotalRejected  int64
	ThrottleLevel  ThrottleLevel
	OverloadActive bool
}

func (c *OverloadController) Snapshot() OverloadSnapshot {
	if !c.Enabled() {
		return OverloadSnapshot{}
	}
	return OverloadSnapshot{
		InFlight:       c.inFlight.Load(),
		TotalAdmitted:  c.totalAdmitted.Load(),
		TotalRejected:  c.totalRejected.Load(),
		ThrottleLevel:  c.ThrottleLevel(),
		OverloadActive: int(c.inFlight.Load()) >= c.maxInFlight,
	}
}

// Close releases resources. After Close, Admit always returns an error.
func (c *OverloadController) Close() {
	if c != nil {
		c.closed.Store(true)
	}
}

// Priority constants for the Admit call (mirror session.Priority without import cycle).
const (
	overloadPriorityLow  = 0
	overloadPriorityHigh = 1
)
