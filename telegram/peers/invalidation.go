package peers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// staleHashErrors are server rejections of a cached access hash. They are
// safe to retry: the server rejected the call without executing it.
const (
	errPeerIDInvalid  = "PEER_ID_INVALID"
	errChannelInvalid = "CHANNEL_INVALID"
)

// InvalidateOnStaleHash returns an invoker middleware that watches for
// PEER_ID_INVALID / CHANNEL_INVALID responses, drops the referenced peers
// from the cache, and replays the request once. A rejected request was not
// executed server-side, so the replay cannot double-apply mutations.
//
// Register it once on the client's invoker chain:
//
//	client.UseInvokerMiddleware(client.Peers().InvalidateOnStaleHash())
func (m *Manager) InvalidateOnStaleHash() func(next tg.Invoker) tg.Invoker {
	return func(next tg.Invoker) tg.Invoker {
		return staleHashInvoker{mgr: m, next: next}
	}
}

type staleHashInvoker struct {
	mgr  *Manager
	next tg.Invoker
}

func (s staleHashInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	res, err := s.next.RPCInvoke(ctx, input, decode)
	if err == nil || !tgerr.Is(err, errPeerIDInvalid, errChannelInvalid) {
		return res, err
	}
	ids := requestPeerIDs(input)
	if len(ids) == 0 {
		return res, err
	}
	for _, id := range ids {
		s.mgr.Invalidate(id)
	}
	retried, rerr := s.next.RPCInvoke(ctx, input, decode)
	if tgerr.Is(rerr, errPeerIDInvalid, errChannelInvalid) {
		// Still rejected after a fresh resolution attempt: this client can no
		// longer address the peer.
		return retried, fmt.Errorf("%w: peer %v rejected after re-resolution: %w", ErrInvalid, ids, rerr)
	}
	return retried, rerr
}

func (s staleHashInvoker) RPCInvokeRaw(ctx context.Context, input tg.TLObject) ([]byte, error) {
	res, err := s.next.RPCInvokeRaw(ctx, input)
	if err == nil || !tgerr.Is(err, errPeerIDInvalid, errChannelInvalid) {
		return res, err
	}
	ids := requestPeerIDs(input)
	if len(ids) == 0 {
		return res, err
	}
	for _, id := range ids {
		s.mgr.Invalidate(id)
	}
	retried, rerr := s.next.RPCInvokeRaw(ctx, input)
	if tgerr.Is(rerr, errPeerIDInvalid, errChannelInvalid) {
		return retried, fmt.Errorf("%w: peer %v rejected after re-resolution: %w", ErrInvalid, ids, rerr)
	}
	return retried, rerr
}

// requestPeerIDs extracts every peer ID referenced by an RPC request by
// walking its fields (one level of nesting deep). Used to know which cache
// entries to drop when the server rejects an access hash.
func requestPeerIDs(input tg.TLObject) []int64 {
	if input == nil {
		return nil
	}
	v := reflect.Indirect(reflect.ValueOf(input))
	if v.Kind() != reflect.Struct {
		return nil
	}
	var ids []int64
	collectPeerIDs(v, &ids, 0)
	return ids
}

func collectPeerIDs(v reflect.Value, ids *[]int64, depth int) {
	if depth > 2 {
		return
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			return
		}
		if id, ok := peerIDOfLeaf(v); ok {
			*ids = append(*ids, id)
			return
		}
		collectPeerIDs(v.Elem(), ids, depth)
	case reflect.Struct:
		if id, ok := peerIDOfLeaf(v); ok {
			*ids = append(*ids, id)
			return
		}
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if !f.CanInterface() {
				continue
			}
			collectPeerIDs(f, ids, depth+1)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			collectPeerIDs(v.Index(i), ids, depth+1)
		}
	}
}

// peerIDOfLeaf reports the canonical cache key for concrete peer values.
func peerIDOfLeaf(v reflect.Value) (int64, bool) {
	if !v.IsValid() || !v.CanInterface() {
		return 0, false
	}
	switch p := v.Interface().(type) {
	case *tg.InputPeerUser:
		return p.UserID, true
	case *tg.InputPeerChannel:
		return p.ChannelID, true
	case *tg.InputPeerChat:
		return p.ChatID, true
	case *tg.InputUser:
		return p.UserID, true
	case *tg.InputChannel:
		return p.ChannelID, true
	case *tg.PeerUser:
		return p.UserID, true
	case *tg.PeerChannel:
		return p.ChannelID, true
	case *tg.PeerChat:
		return p.ChatID, true
	case *tg.Channel:
		return p.ID, true
	case *tg.User:
		return p.ID, true
	case *tg.Chat:
		return p.ID, true
	}
	return 0, false
}

// Invalidate drops every cached form of a peer — the in-memory entry, its
// username and phone index mappings, and the persisted store record — and is
// used when the server rejects a cached access hash. Lookups normalize the
// marked (-100-prefixed) form automatically.
func (m *Manager) Invalidate(id int64) {
	keys := peerLookupIDs(id)

	m.mu.Lock()
	for _, key := range keys {
		if username, ok := m.idToUsername[key]; ok {
			delete(m.usernameToID, username)
			delete(m.idToUsername, key)
		}
		if phone, ok := m.idToPhone[key]; ok {
			delete(m.phoneToID, phone)
			delete(m.idToPhone, key)
		}
		delete(m.byID, key)
	}
	m.mu.Unlock()

	if ps := m.store(); ps != nil {
		for _, key := range keys {
			_ = ps.DeletePeer(key)
		}
	}
}
