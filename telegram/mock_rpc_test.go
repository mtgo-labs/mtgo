package telegram

import (
	"context"
	"sync"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

type mockBotRPCInvoke struct {
	mu      sync.Mutex
	calls   []tg.TLObject
	results map[uint32]tg.TLObject
	errors  map[uint32]error
}

func newMockBotRPCInvoke() *mockBotRPCInvoke {
	return &mockBotRPCInvoke{
		results: make(map[uint32]tg.TLObject),
		errors:  make(map[uint32]error),
	}
}

func (m *mockBotRPCInvoke) RPCInvoke(_ context.Context, req tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, req)
	id := req.ConstructorID()
	if err, ok := m.errors[id]; ok {
		return nil, err
	}
	if res, ok := m.results[id]; ok {
		return res, nil
	}
	return nil, nil
}

func (m *mockBotRPCInvoke) RPCInvokeRaw(_ context.Context, _ tg.TLObject) ([]byte, error) {
	return nil, nil
}

func (m *mockBotRPCInvoke) setResult(constructorID uint32, result tg.TLObject) {
	m.mu.Lock()
	m.results[constructorID] = result
	m.mu.Unlock()
}

func (m *mockBotRPCInvoke) setError(constructorID uint32, err error) {
	m.mu.Lock()
	m.errors[constructorID] = err
	m.mu.Unlock()
}

func (m *mockBotRPCInvoke) lastCall() tg.TLObject {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.calls) == 0 {
		return nil
	}
	return m.calls[len(m.calls)-1]
}

func (m *mockBotRPCInvoke) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func newClientWithBotRPCMock(_ testing.TB) (*Client, *mockBotRPCInvoke) {
	c, _ := NewClient(1, "test", nil)
	c.state.setConnected(true)
	mock := newMockBotRPCInvoke()
	c.testInvoker = mock
	return c, mock
}
