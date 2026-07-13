package session

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// CallHandle represents a single pending RPC call. It is safe for concurrent
// use: the receive loop calls complete (via Resolve/Reject) while the caller
// goroutine waits on Done.
type CallHandle struct {
	done      chan struct{} // closed exactly once on completion
	once      sync.Once     // ensures single-shot close(done)
	mu        sync.Mutex    // protects result, rawResult, err, payload
	result    tg.TLObject   // stored decoded TL result
	rawResult []byte        // stored raw result bytes
	err       error         // stored error
	isRaw     bool          // true for SendRaw waiters
	sentAt    int64         // unix-nano timestamp when the query was sent (for state checks)
	acked     atomic.Bool   // true when the server acknowledged receipt
	payload   []byte        // encrypted payload for re-send on msg_resend_req
}

// SentAt returns the time the query was sent.
func (h *CallHandle) SentAt() time.Time {
	return time.Unix(0, h.sentAt)
}

// IsAcked reports whether the server has acknowledged receipt of this query.
func (h *CallHandle) IsAcked() bool {
	return h.acked.Load()
}

// Done returns a channel that is closed when the handle is completed.
func (h *CallHandle) Done() <-chan struct{} {
	return h.done
}

// Result returns the stored result, raw result, and error.
// Must only be called after Done() is closed.
func (h *CallHandle) Result() (tg.TLObject, []byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.result, h.rawResult, h.err
}

// StorePayload stores the encrypted payload for later re-send on msg_resend_req.
func (h *CallHandle) StorePayload(p []byte) {
	h.mu.Lock()
	h.payload = p
	h.mu.Unlock()
}

// GetPayload returns the stored encrypted payload, or nil if none was stored.
func (h *CallHandle) GetPayload() []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.payload
}

// complete is the shared completion path. fn is called inside the mutex to
// store the result or error, then done is closed via sync.Once.
// Returns true if this was the first completion.
func (h *CallHandle) complete(fn func()) bool {
	completed := false
	h.once.Do(func() {
		h.mu.Lock()
		fn()
		h.mu.Unlock()
		close(h.done)
		completed = true
	})
	return completed
}

// PendingManager owns the lifecycle of all pending RPC calls for a session.
// Resolve/Reject/Cancel never block on caller behavior.
type PendingManager struct {
	mu           sync.Mutex
	pending      map[int64]*CallHandle
	maxPending   int64
	totalPending atomic.Int64
	rawPending   atomic.Int64
}

func NewPendingManager() *PendingManager {
	return &PendingManager{
		pending:    make(map[int64]*CallHandle),
		maxPending: 1024,
	}
}

func (pm *PendingManager) SetMaxPending(n int64) {
	pm.mu.Lock()
	pm.maxPending = n
	pm.mu.Unlock()
}

func (pm *PendingManager) MaxPending() int64 {
	pm.mu.Lock()
	n := pm.maxPending
	pm.mu.Unlock()
	return n
}

func (pm *PendingManager) Register(msgID int64, isRaw bool) (*CallHandle, error) {
	pm.mu.Lock()
	if pm.maxPending > 0 && int64(len(pm.pending)) >= pm.maxPending {
		pm.mu.Unlock()
		return nil, ErrBusy
	}
	h := &CallHandle{
		done:   make(chan struct{}),
		isRaw:  isRaw,
		sentAt: time.Now().UnixNano(),
	}
	pm.pending[msgID] = h
	pm.mu.Unlock()
	pm.totalPending.Add(1)
	if isRaw {
		pm.rawPending.Add(1)
	}
	return h, nil
}

// Resolve stores a decoded TL result and completes the handle.
// Returns true if accepted, false if already completed or not found.
func (pm *PendingManager) Resolve(msgID int64, obj tg.TLObject) bool {
	h := pm.remove(msgID)
	if h == nil {
		return false
	}
	if h.complete(func() { h.result = obj }) {
		pm.dec(h)
		return true
	}
	return false
}

// ResolveRaw stores raw result bytes and completes the handle.
func (pm *PendingManager) ResolveRaw(msgID int64, data []byte) bool {
	h := pm.remove(msgID)
	if h == nil {
		return false
	}
	if h.complete(func() { h.rawResult = data }) {
		pm.dec(h)
		return true
	}
	return false
}

// ResolveRPCResult handles an incoming raw RPC result payload for reqMsgID.
// It resolves raw waiters immediately. Returns true if a decoded waiter
// exists — the caller should decode the payload and call Resolve.
func (pm *PendingManager) ResolveRPCResult(reqMsgID int64, rawPayload []byte) bool {
	pm.mu.Lock()
	h, ok := pm.pending[reqMsgID]
	if !ok {
		pm.mu.Unlock()
		return false
	}

	if h.isRaw {
		delete(pm.pending, reqMsgID)
		pm.mu.Unlock()
		h.complete(func() {
			cp := make([]byte, len(rawPayload))
			copy(cp, rawPayload)
			h.rawResult = cp
		})
		pm.dec(h)
		return false
	}

	// Decoded waiter — leave in map for the caller to Resolve after decoding.
	pm.mu.Unlock()
	return true
}

// Reject stores an error and completes the handle.
func (pm *PendingManager) Reject(msgID int64, err error) bool {
	h := pm.remove(msgID)
	if h == nil {
		return false
	}
	if h.complete(func() { h.err = err }) {
		pm.dec(h)
		return true
	}
	return false
}

// Cancel removes the handle from the map without closing done.
// The caller is already returning via ctx.Err() or timeout.
func (pm *PendingManager) Cancel(msgID int64) bool {
	h := pm.remove(msgID)
	if h == nil {
		return false
	}
	pm.dec(h)
	return true
}

// RejectAll rejects every active pending call with the given error.
func (pm *PendingManager) RejectAll(err error) {
	pm.mu.Lock()
	handles := make([]*CallHandle, 0, len(pm.pending))
	var total, raw int64
	for msgID, h := range pm.pending {
		handles = append(handles, h)
		delete(pm.pending, msgID)
		total++
		if h.isRaw {
			raw++
		}
	}
	pm.mu.Unlock()
	for _, h := range handles {
		h.complete(func() { h.err = err })
	}
	pm.totalPending.Add(-total)
	pm.rawPending.Add(-raw)
}

// Has reports whether a specific msgID has a pending handle.
func (pm *PendingManager) Has(msgID int64) bool {
	pm.mu.Lock()
	_, ok := pm.pending[msgID]
	pm.mu.Unlock()
	return ok
}

// HasRaw reports whether a specific msgID has a raw pending handle.
func (pm *PendingManager) HasRaw(msgID int64) bool {
	pm.mu.Lock()
	h, ok := pm.pending[msgID]
	pm.mu.Unlock()
	return ok && h.isRaw
}

// HasAny reports whether any pending calls exist.
func (pm *PendingManager) HasAny() bool {
	return pm.totalPending.Load() > 0
}

// HasAnyRaw reports whether any raw pending calls exist.
func (pm *PendingManager) HasAnyRaw() bool {
	return pm.rawPending.Load() > 0
}

// Count returns the number of active pending calls.
func (pm *PendingManager) Count() int64 {
	return pm.totalPending.Load()
}

// MarkAcked marks the pending handle for msgID as acknowledged by the server.
// No-op if the message is not pending or already completed.
func (pm *PendingManager) MarkAcked(msgID int64) {
	pm.mu.Lock()
	h, ok := pm.pending[msgID]
	pm.mu.Unlock()
	if ok {
		h.acked.Store(true)
	}
}

// GetPayload returns the stored encrypted payload for msgID, or nil if not found.
func (pm *PendingManager) GetPayload(msgID int64) []byte {
	pm.mu.Lock()
	h, ok := pm.pending[msgID]
	pm.mu.Unlock()
	if !ok {
		return nil
	}
	return h.GetPayload()
}

// OverduePending returns the message IDs of content-related pending queries
// that have been sent more than threshold ago and have not been acknowledged.
// These are candidates for msgs_state_req reconciliation.
func (pm *PendingManager) OverduePending(threshold time.Duration) []int64 {
	cutoff := time.Now().Add(-threshold).UnixNano()
	pm.mu.Lock()
	defer pm.mu.Unlock()
	var ids []int64
	for msgID, h := range pm.pending {
		if h.isRaw {
			continue
		}
		if h.acked.Load() {
			continue
		}
		if h.sentAt > cutoff {
			continue
		}
		ids = append(ids, msgID)
	}
	return ids
}

// remove extracts and deletes the handle from the map.
func (pm *PendingManager) remove(msgID int64) *CallHandle {
	pm.mu.Lock()
	h, ok := pm.pending[msgID]
	if ok {
		delete(pm.pending, msgID)
	}
	pm.mu.Unlock()
	return h
}

// decrements counters for a completed handle.
func (pm *PendingManager) dec(h *CallHandle) {
	pm.totalPending.Add(-1)
	if h.isRaw {
		pm.rawPending.Add(-1)
	}
}
