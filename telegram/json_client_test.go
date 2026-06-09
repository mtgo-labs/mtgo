package telegram

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

type mockJSONInvoker struct {
	lastReq tg.TLObject
	result  tg.TLObject
	err     error
}

func (m *mockJSONInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	m.lastReq = input
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func (m *mockJSONInvoker) RPCInvokeRaw(_ context.Context, _ tg.TLObject) ([]byte, error) {
	return nil, nil
}

func TestInvokeJSONBasic(t *testing.T) {
	mock := &mockJSONInvoker{
		result: &tg.UserEmpty{ID: 42},
	}
	client := NewJSONClient(tg.NewRPCClient(mock))

	jsonPayload := `{"peer": {"_": "inputPeerSelf"}, "message": "hello", "random_id": 12345}`

	ctx := context.Background()
	result, err := client.InvokeJSON(ctx, "messages.sendMessage", []byte(jsonPayload), false)
	if err != nil {
		t.Fatalf("InvokeJSON failed: %v", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(result, &resultMap); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if resultMap["_"] != "userEmpty" {
		t.Fatalf("expected _=userEmpty, got %v", resultMap["_"])
	}
}

func TestInvokeJSONUnknownFunction(t *testing.T) {
	mock := &mockJSONInvoker{}
	client := NewJSONClient(tg.NewRPCClient(mock))

	ctx := context.Background()
	_, err := client.InvokeJSON(ctx, "nonexistent.function", nil, false)
	if err == nil {
		t.Fatal("expected error for unknown function")
	}
}

func TestInvokeJSONSnakeCase(t *testing.T) {
	mock := &mockJSONInvoker{
		result: &tg.UserEmpty{ID: 99},
	}
	client := NewJSONClient(tg.NewRPCClient(mock))

	jsonPayload := `{"peer": {"_": "inputPeerSelf"}, "message": "hello", "random_id": 12345}`

	ctx := context.Background()
	result, err := client.InvokeJSON(ctx, "messages.sendMessage", []byte(jsonPayload), true)
	if err != nil {
		t.Fatalf("InvokeJSON with snakeCase failed: %v", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(result, &resultMap); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if resultMap["_"] != "user_empty" {
		t.Fatalf("expected _=user_empty (snake_case), got %v", resultMap["_"])
	}
}

func TestInvokeJSONInvalidJSON(t *testing.T) {
	mock := &mockJSONInvoker{}
	client := NewJSONClient(tg.NewRPCClient(mock))

	ctx := context.Background()
	_, err := client.InvokeJSON(ctx, "messages.sendMessage", []byte("{invalid}"), false)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
