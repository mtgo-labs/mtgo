package peers

import (
	"context"
	"reflect"
	"sync"

	"github.com/mtgo-labs/mtgo/tg"
)

// IngestResponses returns an invoker middleware that funnels users and chats
// found in successful RPC responses into the cache. Every entity the client
// ever observes — not just updates and dialogs — becomes a future cache hit.
//
// The walk uses reflection, but the field layout of each response type is
// computed once and cached, so steady-state cost per response is a cached
// map lookup plus direct field reads — no per-call type discovery.
func (m *Manager) IngestResponses() func(next tg.Invoker) tg.Invoker {
	return func(next tg.Invoker) tg.Invoker {
		return responseIngestInvoker{mgr: m, next: next}
	}
}

type responseIngestInvoker struct {
	mgr  *Manager
	next tg.Invoker
}

func (r responseIngestInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	res, err := r.next.RPCInvoke(ctx, input, decode)
	if err == nil {
		r.mgr.ingestResult(res)
	}
	return res, err
}

func (r responseIngestInvoker) RPCInvokeRaw(ctx context.Context, input tg.TLObject) ([]byte, error) {
	// Raw results are undecoded bytes; nothing to ingest.
	return r.next.RPCInvokeRaw(ctx, input)
}

// entityKind identifies which cache lane a response field feeds.
type entityKind int

const (
	kindUsers entityKind = iota + 1
	kindChats
)

// responseLayout records which depth-1 struct fields of a response type hold
// user/chat entity slices.
type responseLayout struct {
	fields []layoutField
}

type layoutField struct {
	index int // struct field index
	kind  entityKind
}

var (
	layoutMu    sync.RWMutex
	layoutCache = make(map[reflect.Type]*responseLayout)
	noEntities  = make(map[reflect.Type]bool) // types known to carry none
)

var (
	userClassType = reflect.TypeOf((*tg.UserClass)(nil)).Elem()
	chatClassType = reflect.TypeOf((*tg.ChatClass)(nil)).Elem()
)

// layoutFor returns the cached entity layout of a response type, or nil when
// the type carries no entity slices.
func layoutFor(t reflect.Type) *responseLayout {
	layoutMu.RLock()
	if l, ok := layoutCache[t]; ok {
		layoutMu.RUnlock()
		return l
	}
	skip := noEntities[t]
	layoutMu.RUnlock()
	if skip {
		return nil
	}

	var l *responseLayout
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Type.Kind() != reflect.Slice {
				continue
			}
			var kind entityKind
			switch {
			case f.Type.Elem().Implements(userClassType):
				kind = kindUsers
			case f.Type.Elem().Implements(chatClassType):
				kind = kindChats
			default:
				continue
			}
			if l == nil {
				l = &responseLayout{}
			}
			l.fields = append(l.fields, layoutField{index: i, kind: kind})
		}
	}

	layoutMu.Lock()
	if l == nil {
		noEntities[t] = true
	} else {
		layoutCache[t] = l
	}
	layoutMu.Unlock()
	return l
}

// ingestResult walks a successful RPC result and caches every user/chat
// entity slice found in it.
func (m *Manager) ingestResult(result tg.TLObject) {
	if result == nil {
		return
	}
	v := reflect.ValueOf(result)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return
	}
	l := layoutFor(v.Type())
	if l == nil {
		return
	}
	structVal := v.Elem()
	var users []tg.UserClass
	var chats []tg.ChatClass
	for _, f := range l.fields {
		fieldVal := structVal.Field(f.index)
		switch f.kind {
		case kindUsers:
			slice, ok := fieldVal.Interface().([]tg.UserClass)
			if ok && len(slice) > 0 {
				users = append(users, slice...)
			}
		case kindChats:
			slice, ok := fieldVal.Interface().([]tg.ChatClass)
			if ok && len(slice) > 0 {
				chats = append(chats, slice...)
			}
		}
	}
	if len(users) > 0 || len(chats) > 0 {
		m.Ingest(users, chats)
	}
}
