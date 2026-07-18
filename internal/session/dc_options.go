package session

import (
	"math/rand"
	"sort"
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

// DCOptionPool manages candidate endpoints with per-DC health scoring.
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
	candidates, err := p.Candidates(1)
	if err != nil {
		return DataCenter{}, err
	}
	return candidates[0], nil
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

// ResetHealth marks every endpoint untested after a network change so stale
// failures and successes from the previous network do not affect selection.
func (p *DCOptionPool) ResetHealth() {
	p.mu.Lock()
	for i := range p.options {
		p.options[i].Health = EndpointHealth{State: HealthUntested}
	}
	p.mu.Unlock()
}

// Candidates returns up to max endpoints ordered by current health.
// Order: Ok by newest success, Untested shuffled, then Error after cool-down.
// A max <= 0 returns every currently eligible endpoint.
func (p *DCOptionPool) Candidates(max int) ([]DataCenter, error) {
	return p.CandidatesForDC(p.dcID, max)
}

// CandidatesForDC returns up to max endpoints for dcID ordered by current health.
func (p *DCOptionPool) CandidatesForDC(dcID, max int) ([]DataCenter, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.options) == 0 {
		return nil, ErrNoEndpoints
	}

	now := time.Now()
	okEndpoints := make([]DCOptionWithHealth, 0, len(p.options))
	untestedEndpoints := make([]DCOptionWithHealth, 0, len(p.options))
	retryEndpoints := make([]DCOptionWithHealth, 0, len(p.options))
	for _, ep := range p.options {
		if ep.Option.ID != dcID {
			continue
		}
		switch ep.Health.State {
		case HealthOk:
			okEndpoints = append(okEndpoints, ep)
		case HealthUntested, HealthChecking:
			untestedEndpoints = append(untestedEndpoints, ep)
		case HealthError:
			if ep.Health.ErrorAt.Add(p.coolDown).Before(now) {
				retryEndpoints = append(retryEndpoints, ep)
			}
		}
	}

	sort.Slice(okEndpoints, func(i, j int) bool {
		return okEndpoints[i].Health.OkAt.After(okEndpoints[j].Health.OkAt)
	})
	rand.Shuffle(len(untestedEndpoints), func(i, j int) {
		untestedEndpoints[i], untestedEndpoints[j] = untestedEndpoints[j], untestedEndpoints[i]
	})
	sort.Slice(retryEndpoints, func(i, j int) bool {
		return retryEndpoints[i].Health.ErrorAt.Before(retryEndpoints[j].Health.ErrorAt)
	})

	candidates := make([]DataCenter, 0, len(p.options))
	appendEndpoint := func(ep DCOptionWithHealth) {
		if max > 0 && len(candidates) >= max {
			return
		}
		candidates = append(candidates, ep.Option)
	}
	for _, ep := range okEndpoints {
		appendEndpoint(ep)
	}
	for _, ep := range untestedEndpoints {
		appendEndpoint(ep)
	}
	for _, ep := range retryEndpoints {
		appendEndpoint(ep)
	}

	if len(candidates) == 0 {
		return nil, ErrAllEndpointsFailing
	}
	return candidates, nil
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
