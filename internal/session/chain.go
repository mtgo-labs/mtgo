package session

import (
	"context"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// chainCtxKey is the context key used to pass a chain ID through Invoke.
type chainCtxKey struct{}

// WithChainID returns a new context that carries the given chainID. When the
// resulting context is passed to [Session.Invoke], the query is wrapped in
// invokeAfterMsg referencing the chain's last sent msg_id, ensuring
// server-side sequential processing.
func WithChainID(ctx context.Context, chainID int64) context.Context {
	if chainID == 0 {
		return ctx
	}
	return context.WithValue(ctx, chainCtxKey{}, chainID)
}

// ChainIDFromContext extracts the chain ID from ctx. Returns (0, false) when
// no chain ID is present.
func ChainIDFromContext(ctx context.Context) (int64, bool) {
	v := ctx.Value(chainCtxKey{})
	if v == nil {
		return 0, false
	}
	return v.(int64), true
}

// SetChain records the last sent msg_id for a chain. Subsequent calls to
// [Session.InvokeChained] (or Invoke with [WithChainID]) for the same chainID
// will wrap their query in invokeAfterMsg referencing msgID.
func (s *Session) SetChain(msgID int64, chainID int64) {
	s.chainMu.Lock()
	s.chains[chainID] = msgID
	s.chainMu.Unlock()
}

// ClearChain removes the chain entry for chainID. The next InvokeChained call
// for this chain will send the query without an invokeAfterMsg wrapper.
func (s *Session) ClearChain(chainID int64) {
	s.chainMu.Lock()
	delete(s.chains, chainID)
	s.chainMu.Unlock()
}

// ChainLastMsgID returns the last sent msg_id for the given chain, or 0 and
// false when the chain has no entry.
func (s *Session) ChainLastMsgID(chainID int64) (int64, bool) {
	s.chainMu.Lock()
	last, ok := s.chains[chainID]
	s.chainMu.Unlock()
	return last, ok
}

// wrapChain returns the query wrapped in invokeAfterMsg if chainID has a
// prior msg_id, or the original query if the chain is empty/new.
func (s *Session) wrapChain(chainID int64, query tg.TLObject) tg.TLObject {
	s.chainMu.Lock()
	lastMsgID, ok := s.chains[chainID]
	s.chainMu.Unlock()
	if !ok || lastMsgID == 0 {
		return query
	}
	return &tg.InvokeAfterMsgRequest{MsgID: lastMsgID, Query: query}
}

// InvokeChained sends an RPC query as part of an ordered chain. If chainID
// has a prior message, the query is wrapped in invokeAfterMsg so the server
// processes it only after the prior message. On success, the chain is updated
// with the new msg_id so the next InvokeChained call references it.
//
// ChainID 0 is a no-op: the query is sent without any wrapper. This makes
// chains fully opt-in — callers that never pass a chainID get standard
// unordered behavior.
func (s *Session) InvokeChained(ctx context.Context, chainID int64, query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error) {
	return s.Invoke(WithChainID(ctx, chainID), query, retries, timeout)
}
