package tgerr

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNew_plainMessage(t *testing.T) {
	e := New(400, "USERNAME_NOT_OCCUPIED")
	if e.Code != 400 {
		t.Errorf("Code = %d, want 400", e.Code)
	}
	if e.Message != "USERNAME_NOT_OCCUPIED" {
		t.Errorf("Message = %q, want %q", e.Message, "USERNAME_NOT_OCCUPIED")
	}
	if e.Type != "USERNAME_NOT_OCCUPIED" {
		t.Errorf("Type = %q, want %q", e.Type, "USERNAME_NOT_OCCUPIED")
	}
	if e.Argument != 0 {
		t.Errorf("Argument = %d, want 0", e.Argument)
	}
}

func TestNew_parameterizedMessage(t *testing.T) {
	e := New(420, "FLOOD_WAIT_30")
	if e.Code != 420 {
		t.Errorf("Code = %d, want 420", e.Code)
	}
	if e.Message != "FLOOD_WAIT_30" {
		t.Errorf("Message = %q, want %q", e.Message, "FLOOD_WAIT_30")
	}
	if e.Type != "FLOOD_WAIT" {
		t.Errorf("Type = %q, want %q", e.Type, "FLOOD_WAIT")
	}
	if e.Argument != 30 {
		t.Errorf("Argument = %d, want 30", e.Argument)
	}
}

func TestNew_middleNumber(t *testing.T) {
	e := New(400, "GO_1337_METERS_AWAY")
	if e.Type != "GO_METERS_AWAY" {
		t.Errorf("Type = %q, want %q", e.Type, "GO_METERS_AWAY")
	}
	if e.Argument != 1337 {
		t.Errorf("Argument = %d, want 1337", e.Argument)
	}
}

func TestNew_emptyMessage(t *testing.T) {
	e := New(500, "")
	if e.Type != "" {
		t.Errorf("Type = %q, want empty", e.Type)
	}
	if e.Argument != 0 {
		t.Errorf("Argument = %d, want 0", e.Argument)
	}
}

func TestNew_singlePart(t *testing.T) {
	e := New(400, "BADREQUEST")
	if e.Type != "BADREQUEST" {
		t.Errorf("Type = %q, want %q", e.Type, "BADREQUEST")
	}
	if e.Argument != 0 {
		t.Errorf("Argument = %d, want 0", e.Argument)
	}
}

func TestError_stringWithArgument(t *testing.T) {
	e := New(420, "FLOOD_WAIT_30")
	got := e.Error()
	want := "rpc error code 420: FLOOD_WAIT (30)"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestError_stringWithoutArgument(t *testing.T) {
	e := New(400, "USERNAME_NOT_OCCUPIED")
	got := e.Error()
	want := "rpc error code 400: USERNAME_NOT_OCCUPIED"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestError_codes(t *testing.T) {
	cases := []struct {
		code int
		msg  string
	}{
		{400, "BAD_REQUEST"},
		{401, "SESSION_PASSWORD_NEEDED"},
		{403, "FORBIDDEN"},
		{420, "FLOOD_WAIT_60"},
		{500, "INTERNAL_SERVER_ERROR"},
	}
	for _, tc := range cases {
		e := New(tc.code, tc.msg)
		if !e.IsCode(tc.code) {
			t.Errorf("IsCode(%d) = false, want true for msg %q", tc.code, tc.msg)
		}
	}
}

func TestAs(t *testing.T) {
	e := New(400, "USERNAME_NOT_OCCUPIED")
	rpcErr, ok := As(e)
	if !ok {
		t.Fatal("As() ok = false, want true")
	}
	if rpcErr.Type != "USERNAME_NOT_OCCUPIED" {
		t.Errorf("Type = %q, want %q", rpcErr.Type, "USERNAME_NOT_OCCUPIED")
	}
}

func TestAs_wrapped(t *testing.T) {
	e := New(400, "USERNAME_NOT_OCCUPIED")
	wrapped := fmt.Errorf("wrap: %w", e)
	rpcErr, ok := As(wrapped)
	if !ok {
		t.Fatal("As() ok = false, want true for wrapped error")
	}
	if rpcErr.Type != "USERNAME_NOT_OCCUPIED" {
		t.Errorf("Type = %q, want %q", rpcErr.Type, "USERNAME_NOT_OCCUPIED")
	}
}

func TestAs_nonRPCError(t *testing.T) {
	_, ok := As(fmt.Errorf("some error"))
	if ok {
		t.Error("As() ok = true for non-RPC error, want false")
	}
}

func TestAsType(t *testing.T) {
	e := New(400, "USERNAME_NOT_OCCUPIED")
	rpcErr, ok := AsType(e, "USERNAME_NOT_OCCUPIED")
	if !ok {
		t.Fatal("AsType() ok = false, want true")
	}
	if rpcErr.Code != 400 {
		t.Errorf("Code = %d, want 400", rpcErr.Code)
	}
}

func TestAsType_wrongType(t *testing.T) {
	e := New(400, "USERNAME_NOT_OCCUPIED")
	_, ok := AsType(e, "FLOOD_WAIT")
	if ok {
		t.Error("AsType() ok = true for wrong type, want false")
	}
}

func TestAsType_wrapped(t *testing.T) {
	e := New(400, "USERNAME_NOT_OCCUPIED")
	wrapped := fmt.Errorf("wrap: %w", e)
	_, ok := AsType(wrapped, "USERNAME_NOT_OCCUPIED")
	if !ok {
		t.Error("AsType() ok = false for wrapped error, want true")
	}
}

func TestIs(t *testing.T) {
	e := New(400, "USERNAME_NOT_OCCUPIED")
	if !Is(e, "USERNAME_NOT_OCCUPIED") {
		t.Error("Is() = false, want true")
	}
	if Is(e, "FLOOD_WAIT") {
		t.Error("Is() = true for wrong type, want false")
	}
}

func TestIs_multipleTypes(t *testing.T) {
	e := New(400, "USERNAME_NOT_OCCUPIED")
	if !Is(e, "FLOOD_WAIT", "USERNAME_NOT_OCCUPIED") {
		t.Error("Is() = false, want true for one-of match")
	}
}

func TestIs_nonRPCError(t *testing.T) {
	if Is(fmt.Errorf("some error"), "FLOOD_WAIT") {
		t.Error("Is() = true for non-RPC error, want false")
	}
}

func TestIsCode(t *testing.T) {
	e := New(420, "FLOOD_WAIT_30")
	if !IsCode(e, 420) {
		t.Error("IsCode(420) = false, want true")
	}
	if IsCode(e, 400) {
		t.Error("IsCode(400) = true, want false")
	}
}

func TestIsCode_multipleCodes(t *testing.T) {
	e := New(401, "SESSION_PASSWORD_NEEDED")
	if !IsCode(e, 400, 401, 403) {
		t.Error("IsCode(400, 401, 403) = false, want true")
	}
}

func TestIsType_nil(t *testing.T) {
	var e *Error
	if e.IsType("anything") {
		t.Error("IsType on nil = true, want false")
	}
}

func TestIsCode_nil(t *testing.T) {
	var e *Error
	if e.IsCode(400) {
		t.Error("IsCode on nil = true, want false")
	}
}

func TestIsOneOf_nil(t *testing.T) {
	var e *Error
	if e.IsOneOf("anything") {
		t.Error("IsOneOf on nil = true, want false")
	}
}

func TestAsFloodWait(t *testing.T) {
	e := New(420, "FLOOD_WAIT_30")
	d, ok := AsFloodWait(e)
	if !ok {
		t.Fatal("AsFloodWait() ok = false, want true")
	}
	if d != 30*time.Second {
		t.Errorf("duration = %v, want %v", d, 30*time.Second)
	}
}

func TestAsFloodWait_premium(t *testing.T) {
	e := New(420, "FLOOD_PREMIUM_WAIT_60")
	d, ok := AsFloodWait(e)
	if !ok {
		t.Fatal("AsFloodWait() ok = false for premium, want true")
	}
	if d != 60*time.Second {
		t.Errorf("duration = %v, want %v", d, 60*time.Second)
	}
}

func TestAsFloodWait_nonFlood(t *testing.T) {
	e := New(400, "USERNAME_NOT_OCCUPIED")
	_, ok := AsFloodWait(e)
	if ok {
		t.Error("AsFloodWait() ok = true for non-flood, want false")
	}
}

func TestAsFloodWait_wrapped(t *testing.T) {
	e := New(420, "FLOOD_WAIT_10")
	wrapped := fmt.Errorf("layer: %w", e)
	d, ok := AsFloodWait(wrapped)
	if !ok {
		t.Fatal("AsFloodWait() ok = false for wrapped, want true")
	}
	if d != 10*time.Second {
		t.Errorf("duration = %v, want %v", d, 10*time.Second)
	}
}

func TestFloodWait_success(t *testing.T) {
	e := New(420, "FLOOD_WAIT_1")
	ctx := context.Background()
	waited, err := FloodWait(ctx, e)
	if !waited {
		t.Error("FloodWait() waited = false, want true")
	}
	if err == nil {
		t.Error("FloodWait() err = nil, want the original error")
	}
}

func TestFloodWait_cancelled(t *testing.T) {
	e := New(420, "FLOOD_WAIT_300")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	waited, err := FloodWait(ctx, e)
	if waited {
		t.Error("FloodWait() waited = true for cancelled context, want false")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("FloodWait() err = %v, want DeadlineExceeded", err)
	}
}

func TestFloodWait_nonFlood(t *testing.T) {
	e := New(400, "USERNAME_NOT_OCCUPIED")
	waited, err := FloodWait(context.Background(), e)
	if waited {
		t.Error("FloodWait() waited = true for non-flood, want false")
	}
	if err == nil {
		t.Error("FloodWait() err = nil for non-flood, want the original error")
	}
}
