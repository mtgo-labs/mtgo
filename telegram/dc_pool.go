package telegram

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
)

type dcSessionStats struct {
	inFlight       atomic.Int64
	latencyEWMA    atomic.Int64
	failures       atomic.Uint32
	unhealthyUntil atomic.Int64
}

func newDCSessionEntry(sess *session.Session, closer ioCloser, client *Client) *dcSessionEntry {
	entry := &dcSessionEntry{sess: sess, closer: closer}
	entry.rpc = tg.NewRPCClient(&dcSessionInvoker{sess: sess, client: client, entry: entry})
	return entry
}

func (e *dcSessionEntry) close() {
	if e == nil {
		return
	}
	e.closeOnce.Do(func() {
		if e.sess != nil {
			e.sess.Stop()
		}
		if e.closer != nil {
			_ = e.closer.Close()
		}
	})
}

func (e *dcSessionEntry) retire() {
	if e == nil {
		return
	}
	e.retired.Store(true)
	if e.stats.inFlight.Load() == 0 {
		e.close()
	}
}

func (e *dcSessionEntry) beginRequest() time.Time {
	e.stats.inFlight.Add(1)
	return time.Now()
}

func (e *dcSessionEntry) endRequest(start time.Time, err error, cooldown time.Duration) {
	e.stats.observe(time.Since(start), err, cooldown)
	if e.stats.inFlight.Add(-1) == 0 && e.retired.Load() {
		e.close()
	}
}

func (s *dcSessionStats) observe(latency time.Duration, err error, cooldown time.Duration) {
	if isDCConnectionFailure(err) {
		s.failures.Add(1)
		s.unhealthyUntil.Store(time.Now().Add(cooldown).UnixNano())
		return
	}
	s.failures.Store(0)
	s.unhealthyUntil.Store(0)
	updateEWMA(&s.latencyEWMA, latency.Nanoseconds())
}

func updateEWMA(dst *atomic.Int64, sample int64) {
	for {
		current := dst.Load()
		next := sample
		if current > 0 {
			next = (current*7 + sample) / 8
		}
		if dst.CompareAndSwap(current, next) {
			return
		}
	}
}

func isDCConnectionFailure(err error) bool {
	if err == nil {
		return false
	}
	switch session.ClassifyError(err) {
	case session.ClassTransient, session.ClassPermanent, session.ClassClosed:
		return true
	}
	return errors.Is(err, session.ErrNotConnected) ||
		errors.Is(err, session.ErrSendTimeout) ||
		errors.Is(err, ErrNotConnected) ||
		errors.Is(err, ErrReconnectFailed)
}

func (p *dcSessionPool) len() int {
	p.mu.RLock()
	n := len(p.entries)
	p.mu.RUnlock()
	return n
}

func (p *dcSessionPool) snapshot(limit int) []*dcSessionEntry {
	p.mu.RLock()
	if limit <= 0 || limit > len(p.entries) {
		limit = len(p.entries)
	}
	entries := append([]*dcSessionEntry(nil), p.entries[:limit]...)
	p.mu.RUnlock()
	return entries
}

func (p *dcSessionPool) rpcClients(limit int) []*tg.RPCClient {
	entries := p.snapshot(limit)
	rpcs := make([]*tg.RPCClient, len(entries))
	for i, entry := range entries {
		rpcs[i] = entry.rpc
	}
	return rpcs
}

func (p *dcSessionPool) replace(index int, entry *dcSessionEntry) *dcSessionEntry {
	p.mu.Lock()
	index %= len(p.entries)
	old := p.entries[index]
	p.entries[index] = entry
	p.mu.Unlock()
	return old
}

func (p *dcSessionPool) entry(index int) *dcSessionEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.entries) == 0 {
		return nil
	}
	return p.entries[index%len(p.entries)]
}

func (p *dcSessionPool) selectEntry() (*dcSessionEntry, int) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.entries) == 0 {
		return nil, -1
	}

	now := time.Now().UnixNano()
	start := int(p.next.Add(1)-1) % len(p.entries)
	var best *dcSessionEntry
	bestIndex := -1
	for offset := range len(p.entries) {
		index := (start + offset) % len(p.entries)
		entry := p.entries[index]
		if entry == nil || entry.retired.Load() || entry.sess == nil || !entry.sess.IsConnected() {
			continue
		}
		if entry.stats.unhealthyUntil.Load() > now {
			continue
		}
		if betterDCSession(entry, best) {
			best = entry
			bestIndex = index
		}
	}
	if best != nil {
		return best, bestIndex
	}

	for offset := range len(p.entries) {
		index := (start + offset) % len(p.entries)
		entry := p.entries[index]
		if entry != nil && !entry.retired.Load() && betterDCSession(entry, best) {
			best = entry
			bestIndex = index
		}
	}
	return best, bestIndex
}

func betterDCSession(candidate, current *dcSessionEntry) bool {
	if current == nil {
		return true
	}
	candidateLoad := candidate.stats.inFlight.Load()
	currentLoad := current.stats.inFlight.Load()
	if candidateLoad != currentLoad {
		return candidateLoad < currentLoad
	}
	candidateLatency := candidate.stats.latencyEWMA.Load()
	currentLatency := current.stats.latencyEWMA.Load()
	if candidateLatency == 0 {
		return currentLatency != 0
	}
	return currentLatency != 0 && candidateLatency < currentLatency
}

type dcPoolInvoker struct {
	pool   *dcSessionPool
	client *Client
	dcID   int
}

func (d *dcPoolInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	entry, index := d.pool.selectEntry()
	if entry == nil {
		return nil, ErrNotConnected
	}
	result, err := entry.rpc.Invoke(ctx, input, decode)
	d.repair(ctx, entry, index, err)
	return result, err
}

func (d *dcPoolInvoker) RPCInvokeRaw(ctx context.Context, input tg.TLObject) ([]byte, error) {
	entry, index := d.pool.selectEntry()
	if entry == nil {
		return nil, ErrNotConnected
	}
	result, err := entry.rpc.InvokeWithRawResult(ctx, input)
	d.repair(ctx, entry, index, err)
	return result, err
}

func (d *dcPoolInvoker) repair(ctx context.Context, entry *dcSessionEntry, index int, err error) {
	if d.client == nil || !isDCConnectionFailure(err) {
		return
	}
	_, repairErr := d.client.replaceDCRPCPoolEntryIfCurrent(ctx, d.dcID, d.pool.len(), index, entry)
	if repairErr != nil && d.client.Log != nil {
		d.client.Log.Warnf("DC %d pool repair failed: %v", d.dcID, repairErr)
	}
}
