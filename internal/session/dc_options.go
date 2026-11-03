package session

import (
	"errors"
	"math/rand"
	"sort"
	"sync"
	"syscall"
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
	// familyUnreachable records when an address family last proved
	// unroutable on this host (keyed by the DataCenter.IPv6 flag).
	familyUnreachable map[bool]time.Time
	// familyProbeInterval is how long a family stays suppressed after
	// proving unroutable; afterwards candidates include it again so a
	// family that appears later (e.g. IPv6 on a roaming host) is retried.
	familyProbeInterval time.Duration
	mu                  sync.RWMutex
}

// defaultFamilyProbeInterval is the default suppression window for an
// address family whose dials fail at the routing level. Long enough to keep
// quiet across reconnect backoff loops, short enough to pick up a network
// change that introduces the family without waiting for ResetHealth.
const defaultFamilyProbeInterval = 5 * time.Minute

// NewDCOptionPool creates a new endpoint pool for the given DC.
func NewDCOptionPool(dcID int, coolDown time.Duration) *DCOptionPool {
	return &DCOptionPool{
		dcID:                dcID,
		coolDown:            coolDown,
		familyUnreachable:   make(map[bool]time.Time),
		familyProbeInterval: defaultFamilyProbeInterval,
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

// RecordDialFailure records a failed connection attempt. When the dial
// error proves the host cannot route the endpoint's address family at all
// (network/host unreachable, family unsupported) rather than the endpoint
// being down, the family is additionally marked unreachable so Candidates
// skips its endpoints for the probe interval while the other family still
// offers candidates. This keeps IPv4-only hosts from re-racing IPv6
// endpoints (and vice versa) on every connect.
func (p *DCOptionPool) RecordDialFailure(option DataCenter, err error) {
	p.RecordFailure(option)
	if err == nil || !isFamilyUnreachableError(err) {
		return
	}
	p.mu.Lock()
	p.familyUnreachable[option.IPv6] = time.Now()
	p.mu.Unlock()
}

// isFamilyUnreachableError reports whether err indicates the host has no
// route to (or kernel support for) the dialed address family, as opposed to
// one endpoint being down. Such dials fail instantly at the kernel level,
// e.g. dialing an IPv6 Telegram DC on an IPv4-only host.
func isFamilyUnreachableError(err error) bool {
	return errors.Is(err, syscall.ENETUNREACH) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.EAFNOSUPPORT)
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
// Address-family suppression is cleared too: a network change is exactly
// when a previously unroutable family may become available.
func (p *DCOptionPool) ResetHealth() {
	p.mu.Lock()
	for i := range p.options {
		p.options[i].Health = EndpointHealth{State: HealthUntested}
	}
	clear(p.familyUnreachable)
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
	// Drop endpoints of address families that recently proved unroutable
	// (e.g. IPv6 on an IPv4-only host), but only while the other family
	// still has candidates: if filtering would remove every endpoint, keep
	// them all — connectivity beats quietness.
	if filtered := p.excludeUnreachableFamilies(candidates, now); len(filtered) > 0 {
		candidates = filtered
	}
	return candidates, nil
}

// excludeUnreachableFamilies removes candidates whose address family is
// currently suppressed; it returns candidates unchanged when no family is
// suppressed. Callers must treat an empty result as "keep the unfiltered
// list" rather than "no endpoints".
func (p *DCOptionPool) excludeUnreachableFamilies(candidates []DataCenter, now time.Time) []DataCenter {
	var suppressIPv4, suppressIPv6 bool
	if t, ok := p.familyUnreachable[false]; ok && now.Sub(t) < p.familyProbeInterval {
		suppressIPv4 = true
	}
	if t, ok := p.familyUnreachable[true]; ok && now.Sub(t) < p.familyProbeInterval {
		suppressIPv6 = true
	}
	if !suppressIPv4 && !suppressIPv6 {
		return candidates
	}
	filtered := make([]DataCenter, 0, len(candidates))
	for _, candidate := range candidates {
		if (candidate.IPv6 && suppressIPv6) || (!candidate.IPv6 && suppressIPv4) {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
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
