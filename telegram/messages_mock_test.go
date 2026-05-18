package telegram

import (
	"context"
	"sync"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

type mockMessagesInvoker struct {
	mu      sync.Mutex
	t       testing.TB
	calls   []tg.TLObject
	results map[uint32]tg.TLObject
}

func newMockMessagesInvoker(t testing.TB) *mockMessagesInvoker {
	return &mockMessagesInvoker{
		t:       t,
		results: make(map[uint32]tg.TLObject),
	}
}

func (m *mockMessagesInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, input)
	if result, ok := m.results[input.ConstructorID()]; ok {
		return result, nil
	}
	return updatesWithMessage(), nil
}
func (m *mockMessagesInvoker) RPCInvokeRaw(_ context.Context, _ tg.TLObject) ([]byte, error) {
	return nil, nil
}


func (m *mockMessagesInvoker) lastCall() tg.TLObject {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.calls) == 0 {
		m.t.Fatal("no RPC calls recorded")
	}
	return m.calls[len(m.calls)-1]
}

func (m *mockMessagesInvoker) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockMessagesInvoker) setResult(constructorID uint32, result tg.TLObject) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results[constructorID] = result
}

func newClientWithMock(t testing.TB) (*Client, *mockMessagesInvoker) {
	c, _ := NewClient(1, "test", nil)
	c.state.setConnected(true)
	inv := newMockMessagesInvoker(t)
	c.testInvoker = inv
	return c, inv
}

func updatesWithMessage() tg.TLObject {
	return &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateNewMessage{
				Message: &tg.Message{
					ID:     1,
					PeerID: &tg.PeerChannel{ChannelID: 10},
				},
			},
		},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	}
}
