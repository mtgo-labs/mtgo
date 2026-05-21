package telegram

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type mockTDLibInvoker struct {
	lastReq tg.TLObject
	result  tg.TLObject
	err     error
	rpcErr  *tgerr.Error
}

func (m *mockTDLibInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	m.lastReq = input
	if m.err != nil {
		return nil, m.err
	}
	if m.rpcErr != nil {
		return nil, m.rpcErr
	}
	return m.result, nil
}

func (m *mockTDLibInvoker) RPCInvokeRaw(_ context.Context, _ tg.TLObject) ([]byte, error) {
	return nil, nil
}

func tdlibClient(mock *mockTDLibInvoker) *Client {
	return &Client{testInvoker: mock}
}

func TestInvokeRawJSONBasic(t *testing.T) {
	mock := &mockTDLibInvoker{
		result: &tg.UserEmpty{ID: 42},
	}
	client := tdlibClient(mock)

	req := `{"@type": "sendMessage", "peer": {"@type": "inputPeerSelf"}, "message": "hello", "random_id": "12345"}`
	result, err := client.InvokeRawJSON(context.Background(), []byte(req))
	if err != nil {
		t.Fatalf("InvokeRawJSON: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if m["@type"] != "userEmpty" {
		t.Fatalf("expected @type=userEmpty, got %v", m["@type"])
	}
	if _, has := m["_"]; has {
		t.Fatal("should not contain _ field")
	}
}

func TestInvokeRawJSONShortName(t *testing.T) {
	mock := &mockTDLibInvoker{
		result: &tg.UserEmpty{ID: 1},
	}
	client := tdlibClient(mock)

	req := `{"@type": "getUsers", "id": [{"@type": "inputUserSelf"}]}`
	result, err := client.InvokeRawJSON(context.Background(), []byte(req))
	if err != nil {
		t.Fatalf("InvokeRawJSON: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("result: %v", err)
	}
	if m["@type"] != "userEmpty" {
		t.Fatalf("expected @type=userEmpty, got %v", m["@type"])
	}
}

func TestInvokeRawJSONFullName(t *testing.T) {
	mock := &mockTDLibInvoker{
		result: &tg.UserEmpty{ID: 1},
	}
	client := tdlibClient(mock)

	req := `{"@type": "users.getUsers", "id": [{"@type": "inputUserSelf"}]}`
	result, err := client.InvokeRawJSON(context.Background(), []byte(req))
	if err != nil {
		t.Fatalf("InvokeRawJSON: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("result: %v", err)
	}
	if m["@type"] != "userEmpty" {
		t.Fatalf("expected @type=userEmpty, got %v", m["@type"])
	}
}

func TestInvokeRawJSONUnderscoreFallback(t *testing.T) {
	mock := &mockTDLibInvoker{
		result: &tg.UserEmpty{ID: 1},
	}
	client := tdlibClient(mock)

	req := `{"_": "users.getUsers", "id": [{"_": "inputUserSelf"}]}`
	result, err := client.InvokeRawJSON(context.Background(), []byte(req))
	if err != nil {
		t.Fatalf("InvokeRawJSON: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("result: %v", err)
	}
	if m["@type"] != "userEmpty" {
		t.Fatalf("expected @type=userEmpty, got %v", m["@type"])
	}
}

func TestInvokeRawJSONRPCError(t *testing.T) {
	mock := &mockTDLibInvoker{
		rpcErr: tgerr.New(420, "FLOOD_WAIT_3"),
	}
	client := tdlibClient(mock)

	req := `{"@type": "sendMessage", "peer": {"@type": "inputPeerSelf"}, "message": "hi"}`
	result, err := client.InvokeRawJSON(context.Background(), []byte(req))
	if err != nil {
		t.Fatalf("should not return Go error for RPC errors: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("result: %v", err)
	}
	if m["@type"] != "error" {
		t.Fatalf("expected @type=error, got %v", m["@type"])
	}
	if m["code"] != float64(420) {
		t.Fatalf("expected code=420, got %v", m["code"])
	}
}

func TestInvokeRawJSONMissingType(t *testing.T) {
	mock := &mockTDLibInvoker{}
	client := tdlibClient(mock)

	req := `{"message": "hello"}`
	result, err := client.InvokeRawJSON(context.Background(), []byte(req))
	if err != nil {
		t.Fatalf("should not return Go error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("result: %v", err)
	}
	if m["@type"] != "error" {
		t.Fatalf("expected error response, got %v", m["@type"])
	}
}

func TestInvokeRawJSONUnknownMethod(t *testing.T) {
	mock := &mockTDLibInvoker{}
	client := tdlibClient(mock)

	req := `{"@type": "nonexistentMethod"}`
	result, err := client.InvokeRawJSON(context.Background(), []byte(req))
	if err != nil {
		t.Fatalf("should not return Go error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("result: %v", err)
	}
	if m["@type"] != "error" {
		t.Fatalf("expected error response, got %v", m["@type"])
	}
}

func TestInvokeRawJSONInvalidJSON(t *testing.T) {
	mock := &mockTDLibInvoker{}
	client := tdlibClient(mock)

	result, err := client.InvokeRawJSON(context.Background(), []byte("{invalid}"))
	if err != nil {
		t.Fatalf("should not return Go error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("result: %v", err)
	}
	if m["@type"] != "error" {
		t.Fatalf("expected error response, got %v", m["@type"])
	}
}

func TestTDLibShortNames(t *testing.T) {
	mock := &mockTDLibInvoker{}
	client := tdlibClient(mock)

	names := client.TDLibShortNames()
	if len(names) == 0 {
		t.Fatal("expected non-empty short names")
	}

	found := false
	for _, n := range names {
		if n == "sendMessage" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected sendMessage in short names")
	}
}

func TestInvokeRawJSONExistingInvokeStillWorks(t *testing.T) {
	mock := &mockTDLibInvoker{
		result: &tg.UserEmpty{ID: 42},
	}
	client := tdlibClient(mock)

	payload := []byte(`{"peer": {"_": "inputPeerSelf"}, "message": "hello", "random_id": 12345}`)
	result, err := client.InvokeJSON(context.Background(), "messages.sendMessage", payload, false)
	if err != nil {
		t.Fatalf("existing InvokeJSON should still work: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("result: %v", err)
	}
	if m["_"] != "userEmpty" {
		t.Fatalf("expected _=userEmpty, got %v", m["_"])
	}
}
