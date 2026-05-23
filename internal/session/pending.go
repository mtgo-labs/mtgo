package session

import (
	"sync"
	"sync/atomic"

	"github.com/mtgo-labs/mtgo/tg"
)

// CallHandle represents a single pending RPC call. It is safe for concurrent
// use: the receive loop calls complete (via Resolve/Reject) while the caller
// goroutine waits on Done.
type CallHandle struct {
	done      chan struct{} // closed exactly once on completion
	once      sync.Once     // ensures single-shot close(done)
	mu        sync.Mutex    // protects result, rawResult, err
	result    tg.TLObject   // stored decoded TL result
	rawResult []byte        // stored raw result bytes
	err       error         // stored error
	isRaw     bool          // true for SendRaw waiters
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
	mu      sync.Mutex
	pending map[int64]*CallHandle

	totalPending atomic.Int64
	rawPending   atomic.Int64
}

// NewPendingManager creates a PendingManager ready to use.
func NewPendingManager() *PendingManager {
	return &PendingManager{
		pending: make(map[int64]*CallHandle),
	}
}

// Register creates a new CallHandle for the given msgID.
func (pm *PendingManager) Register(msgID int64, isRaw bool) *CallHandle {
	h := &CallHandle{
		done:  make(chan struct{}),
		isRaw: isRaw,
	}
	pm.mu.Lock()
	pm.pending[msgID] = h
	pm.mu.Unlock()
	pm.totalPending.Add(1)
	if isRaw {
		pm.rawPending.Add(1)
	}
	return h
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
	for msgID, h := range pm.pending {
		handles = append(handles, h)
		delete(pm.pending, msgID)
	}
	pm.mu.Unlock()
	for _, h := range handles {
		h.complete(func() { h.err = err })
	}
	pm.totalPending.Store(0)
	pm.rawPending.Store(0)
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
