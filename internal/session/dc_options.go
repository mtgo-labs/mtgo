package session

import (
	"math/rand"
	"sync"
	"time"
)

// HealthState represents the observed reliability state of a DC endpoint.
// Ported from td/td/telegram/net/DcOptionsSet.h:47-68 (Stat states).
type HealthState int

const (
	// HealthUntested is the initial state for endpoints not yet tried.
	HealthUntested HealthState = iota
	// HealthChecking indicates a connection attempt is in progress.
	HealthChecking
	// HealthOk indicates the endpoint was recently used successfully.
	HealthOk
	// HealthError indicates the endpoint recently failed.
	HealthError
)

// EndpointHealth tracks per-endpoint reliability timestamps.
// Ported from td/td/telegram/net/DcOptionsSet.h:47-68 (Stat with ok_at/error_at/check_at).
type EndpointHealth struct {
	State   HealthState
	OkAt    time.Time // Last successful connection
	ErrorAt time.Time // Last failed connection
	CheckAt time.Time // Last check attempt
}

// DCOptionWithHealth pairs a data center endpoint with its health state.
type DCOptionWithHealth struct {
	Option DataCenter
	Health EndpointHealth
}

// DCOptionPool manages candidate endpoints for a single DC with health scoring.
// Ported from td/td/telegram/net/DcOptionsSet.h (find_connection).
type DCOptionPool struct {
	dcID     int
	options  []DCOptionWithHealth
	coolDown time.Duration // cool-down after failure before retry
	mu       sync.RWMutex
}

// NewDCOptionPool creates a new endpoint pool for the given DC.
func NewDCOptionPool(dcID int, coolDown time.Duration) *DCOptionPool {
	return &DCOptionPool{
		dcID:     dcID,
		coolDown: coolDown,
	}
}

// FindBest returns the healthiest available endpoint.
// Scoring: Ok (most recent ok_at) > Untested (random) > Error with cool-down expired.
// Ported from td/td/telegram/net/DcOptionsSet.h:64-68 find_connection.
func (p *DCOptionPool) FindBest() (DataCenter, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.options) == 0 {
		return DataCenter{}, ErrNoEndpoints
	}

	// 1. Prefer Ok endpoints (most recent ok_at first)
	var okEndpoints []DCOptionWithHealth
	for _, ep := range p.options {
		if ep.Health.State == HealthOk {
			okEndpoints = append(okEndpoints, ep)
		}
	}
	if len(okEndpoints) > 0 {
		best := okEndpoints[0]
		for _, ep := range okEndpoints[1:] {
			if ep.Health.OkAt.After(best.Health.OkAt) {
				best = ep
			}
		}
		return best.Option, nil
	}

	// 2. Then Untested endpoints (random order to spread load)
	var untestedEndpoints []DCOptionWithHealth
	for _, ep := range p.options {
		if ep.Health.State == HealthUntested {
			untestedEndpoints = append(untestedEndpoints, ep)
		}
	}
	if len(untestedEndpoints) > 0 {
		idx := rand.Intn(len(untestedEndpoints))
		return untestedEndpoints[idx].Option, nil
	}

	// 3. Then Error endpoints with cool-down expired
	now := time.Now()
	for _, ep := range p.options {
		if ep.Health.State == HealthError && ep.Health.ErrorAt.Add(p.coolDown).Before(now) {
			return ep.Option, nil
		}
	}

	return DataCenter{}, ErrAllEndpointsFailing
}

// RecordSuccess marks an endpoint as healthy.
func (p *DCOptionPool) RecordSuccess(option DataCenter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.options {
		if p.options[i].Option == option {
			p.options[i].Health.State = HealthOk
			p.options[i].Health.OkAt = time.Now()
			return
		}
	}
}

// RecordFailure marks an endpoint as failing.
func (p *DCOptionPool) RecordFailure(option DataCenter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.options {
		if p.options[i].Option == option {
			p.options[i].Health.State = HealthError
			p.options[i].Health.ErrorAt = time.Now()
			return
		}
	}
}

// AddOption adds a new endpoint to the pool with HealthUntested state.
func (p *DCOptionPool) AddOption(option DataCenter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Don't add duplicates
	for _, ep := range p.options {
		if ep.Option == option {
			return
		}
	}
	p.options = append(p.options, DCOptionWithHealth{
		Option: option,
		Health: EndpointHealth{State: HealthUntested},
	})
}

// UpdateOptions merges server-provided DC options into the pool.
// Existing options are updated if changed; new options are added.
// Ported from td/td/telegram/net/DcOptionsSet.h (on_dc_options_update).
func (p *DCOptionPool) UpdateOptions(options []DataCenter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	existing := make(map[DataCenter]bool)
	for _, ep := range p.options {
		existing[ep.Option] = true
	}
	for _, opt := range options {
		if !existing[opt] {
			p.options = append(p.options, DCOptionWithHealth{
				Option: opt,
				Health: EndpointHealth{State: HealthUntested},
			})
		}
	}
}

// OptionCount returns the number of endpoints in the pool.
func (p *DCOptionPool) OptionCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.options)
}

// Errors for DCOptionPool.
var (
	ErrNoEndpoints         = NewSessionError("dc_options: no endpoints available")
	ErrAllEndpointsFailing = NewSessionError("dc_options: all endpoints failing (cool-down active)")
)

// NewSessionError creates a new session error.
func NewSessionError(msg string) error {
	return sessionError(msg)
}

type sessionError string

func (e sessionError) Error() string { return string(e) }
