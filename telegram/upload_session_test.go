package telegram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"testing"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

func TestUploadRPCUsesMainInvoker(t *testing.T) {
	client, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	invoker := newMockRPCInvoker()
	client.testInvoker = invoker

	result, err := client.uploadRPC().UploadSaveFilePart(context.Background(), &tg.UploadSaveFilePartRequest{
		FileID:   1,
		FilePart: 0,
		Bytes:    []byte("part"),
	})
	if err != nil {
		t.Fatalf("UploadSaveFilePart() error: %v", err)
	}
	if !result {
		t.Fatal("UploadSaveFilePart() = false, want true")
	}
	invoker.mu.Lock()
	defer invoker.mu.Unlock()
	if got := string(invoker.savedParts[0]); got != "part" {
		t.Fatalf("saved part = %q, want %q", got, "part")
	}
}

func TestTransferRetryContextReplaysUploadPart(t *testing.T) {
	client, err := NewClient(1, "hash", &Config{InMemory: true})
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	client.state.SetConnecting(2)
	client.state.SetConnected()

	calls := 0
	query := &tg.UploadSaveBigFilePartRequest{
		FileID:         1,
		FilePart:       2,
		FileTotalParts: 3,
		Bytes:          []byte("part"),
	}
	err = client.retrySessionErr(withTransferRetry(context.Background()), func(*session.Session) error {
		calls++
		if calls == 1 {
			return &session.DeliveryError{
				State: session.DeliveryUnknown,
				Err:   session.ErrSessionClosed,
			}
		}
		return nil
	}, query)
	if err != nil {
		t.Fatalf("retrySessionErr() error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestUploadPartsAreReplaySafe(t *testing.T) {
	client, err := NewClient(1, "hash", &Config{InMemory: true})
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	for _, query := range []tg.TLObject{
		&tg.UploadSaveFilePartRequest{},
		&tg.UploadSaveBigFilePartRequest{},
	} {
		if !client.replaySafe(query) {
			t.Fatalf("%T should be replay-safe", query)
		}
	}
	if client.replaySafe(&tg.MessagesSendMessageRequest{}) {
		t.Fatal("messages.sendMessage should remain delivery-uncertain")
	}
}

func TestIsSessionClosedErrUsesTypedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "session closed", err: session.ErrSessionClosed, want: true},
		{name: "wrapped draining", err: fmt.Errorf("invoke: %w", session.ErrDraining), want: true},
		{name: "network closed", err: net.ErrClosed, want: true},
		{name: "EOF", err: io.EOF, want: true},
		{name: "connection reset", err: syscall.ECONNRESET, want: true},
		{name: "broken pipe", err: syscall.EPIPE, want: true},
		{name: "string lookalike", err: errors.New("broken pipe"), want: false},
		{name: "unrelated", err: errors.New("rpc failed"), want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isSessionClosedErr(test.err); got != test.want {
				t.Fatalf("isSessionClosedErr(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}

func TestIsTransferSessionDeadErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "send timeout", err: session.ErrSendTimeout, want: true},
		{name: "auth key unregistered", err: tgerr.New(401, "AUTH_KEY_UNREGISTERED"), want: true},
		{name: "permanent key empty", err: tgerr.New(401, "AUTH_KEY_PERM_EMPTY"), want: true},
		{name: "unrelated", err: errors.New("rpc failed"), want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isTransferSessionDeadErr(test.err); got != test.want {
				t.Fatalf("isTransferSessionDeadErr(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}
