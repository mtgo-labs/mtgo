package session

import "sync/atomic"

// SeqNoGenerator generates monotonically increasing MTProto sequence numbers
// using atomic operations for safe concurrent access.
type SeqNoGenerator struct {
	value int32
}

// NewSeqNoGenerator returns a new SeqNoGenerator initialized to zero.
func NewSeqNoGenerator() *SeqNoGenerator {
	return &SeqNoGenerator{}
}

// Next returns the next sequence number. If contentRelated is true the
// internal content-message counter is advanced (content-related messages use
// odd numbers); otherwise the current even sequence number is returned.
func (g *SeqNoGenerator) Next(contentRelated bool) int32 {
	if contentRelated {
		n := atomic.AddInt32(&g.value, 1) - 1
		return n*2 + 1
	}
	return atomic.LoadInt32(&g.value) * 2
}

// Reset sets the sequence number counter back to zero.
func (g *SeqNoGenerator) Reset() {
	atomic.StoreInt32(&g.value, 0)
}
