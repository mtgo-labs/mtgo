package telegram

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// scriptedInvoker is a tg.Invoker that records every request and answers
// from a scripted handler, standing in for an auxiliary-DC session.
type scriptedInvoker struct {
	mu     sync.Mutex
	calls  []tg.TLObject
	script func(input tg.TLObject) (tg.TLObject, error)
}

func (s *scriptedInvoker) RPCInvoke(_ context.Context, input tg.TLObject, _ func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	s.mu.Lock()
	s.calls = append(s.calls, input)
	s.mu.Unlock()
	return s.script(input)
}

func (s *scriptedInvoker) RPCInvokeRaw(_ context.Context, input tg.TLObject) ([]byte, error) {
	return nil, fmt.Errorf("unexpected raw invoke %T", input)
}

func (s *scriptedInvoker) recorded() []tg.TLObject {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]tg.TLObject(nil), s.calls...)
}

func TestBotAuthImportRedirect(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantTarget    int
		wantRedirects bool
	}{
		{
			name:          "user migrate with target",
			err:           tgerr.New(303, "USER_MIGRATE_4"),
			wantTarget:    4,
			wantRedirects: true,
		},
		{
			name:          "wrapped user migrate",
			err:           fmt.Errorf("import bot auth: %w", tgerr.New(303, "USER_MIGRATE_2")),
			wantTarget:    2,
			wantRedirects: true,
		},
		{
			name:          "user migrate without target",
			err:           tgerr.New(303, "USER_MIGRATE"),
			wantRedirects: false,
		},
		{
			name:          "phone migrate is not a bot auth redirect",
			err:           tgerr.New(303, "PHONE_MIGRATE_4"),
			wantRedirects: false,
		},
		{
			name:          "file migrate is not a bot auth redirect",
			err:           tgerr.New(303, "FILE_MIGRATE_4"),
			wantRedirects: false,
		},
		{
			name:          "non-migrate code",
			err:           tgerr.New(400, "USER_MIGRATE_4"),
			wantRedirects: false,
		},
		{
			name:          "plain error",
			err:           errors.New("boom"),
			wantRedirects: false,
		},
		{
			name:          "nil error",
			err:           nil,
			wantRedirects: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, redirected := botAuthImportRedirect(tt.err)
			if redirected != tt.wantRedirects {
				t.Fatalf("botAuthImportRedirect(%v) redirected = %v, want %v", tt.err, redirected, tt.wantRedirects)
			}
			if redirected && target != tt.wantTarget {
				t.Fatalf("botAuthImportRedirect(%v) target = %d, want %d", tt.err, target, tt.wantTarget)
			}
		})
	}
}

// newBotTransferClient returns a connected bot client whose home session
// talks to a scripted test server, plus a dialer-bound fake for the
// auxiliary-DC import side.
func newBotTransferClient(t *testing.T, script func(input tg.TLObject) (tg.TLObject, error)) (*Client, *testServer, *scriptedInvoker) {
	t.Helper()
	client, server := newTestClient(1, "hash", Config{
		NoUpdates:        true,
		ReconnectEnabled: false,
		BotToken:         "123456:ABC",
	})
	st := client.testStorage
	if err := st.SetUserID(42); err != nil {
		t.Fatal(err)
	}
	if err := st.SetIsBot(true); err != nil {
		t.Fatal(err)
	}
	if err := client.Connect(5 * time.Second); err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect()
		server.Close()
	})
	return client, server, &scriptedInvoker{script: script}
}

func TestAuthorizeBotDCSessionFallsBackOnUserMigrate(t *testing.T) {
	client, server, fake := newBotTransferClient(t, func(input tg.TLObject) (tg.TLObject, error) {
		switch input.(type) {
		case *tg.AuthImportBotAuthorizationRequest:
			return nil, tgerr.New(303, "USER_MIGRATE_4")
		case *tg.AuthImportAuthorizationRequest:
			return &tg.AuthAuthorization{}, nil
		default:
			return nil, fmt.Errorf("unexpected request %T", input)
		}
	})

	var exported tg.TLObject = &tg.AuthExportedAuthorization{ID: 7, Bytes: []byte{1, 2, 3}}
	server.rpcResult.Store(&exported)

	if err := client.authorizeBotDCSession(context.Background(), tg.NewRPCClient(fake), 2); err != nil {
		t.Fatalf("authorizeBotDCSession() = %v, want nil after USER_MIGRATE fallback", err)
	}

	calls := fake.recorded()
	if len(calls) != 2 {
		t.Fatalf("import session calls = %d (%v), want 2", len(calls), calls)
	}
	if _, ok := calls[0].(*tg.AuthImportBotAuthorizationRequest); !ok {
		t.Fatalf("first call = %T, want *tg.AuthImportBotAuthorizationRequest", calls[0])
	}
	importReq, ok := calls[1].(*tg.AuthImportAuthorizationRequest)
	if !ok {
		t.Fatalf("second call = %T, want *tg.AuthImportAuthorizationRequest", calls[1])
	}
	if importReq.ID != 7 {
		t.Errorf("import ID = %d, want 7 (exported by home DC)", importReq.ID)
	}
	if !bytes.Equal(importReq.Bytes, []byte{1, 2, 3}) {
		t.Errorf("import bytes = %v, want exported bytes [1 2 3]", importReq.Bytes)
	}
}

func TestAuthorizeBotDCSessionDirectImportSuccess(t *testing.T) {
	client, _, fake := newBotTransferClient(t, func(input tg.TLObject) (tg.TLObject, error) {
		if _, ok := input.(*tg.AuthImportBotAuthorizationRequest); !ok {
			return nil, fmt.Errorf("unexpected request %T", input)
		}
		return &tg.AuthAuthorization{}, nil
	})

	if err := client.authorizeBotDCSession(context.Background(), tg.NewRPCClient(fake), 2); err != nil {
		t.Fatalf("authorizeBotDCSession() = %v, want nil", err)
	}
	if calls := fake.recorded(); len(calls) != 1 {
		t.Fatalf("import session calls = %d, want 1 (no fallback on direct success)", len(calls))
	}
}

func TestAuthorizeBotDCSessionUserMigrateFallbackFailure(t *testing.T) {
	client, _, fake := newBotTransferClient(t, func(input tg.TLObject) (tg.TLObject, error) {
		switch input.(type) {
		case *tg.AuthImportBotAuthorizationRequest:
			return nil, tgerr.New(303, "USER_MIGRATE_4")
		default:
			return nil, fmt.Errorf("unexpected request %T", input)
		}
	})
	// No rpcResult stored: the home-DC export fails with MOCK_SERVER, so the
	// auth transfer fallback must fail.

	err := client.authorizeBotDCSession(context.Background(), tg.NewRPCClient(fake), 2)
	if err == nil {
		t.Fatal("authorizeBotDCSession() = nil, want fallback failure")
	}
	var rpcErr *tgerr.Error
	if !errors.As(err, &rpcErr) || !rpcErr.IsType("USER_MIGRATE") {
		t.Fatalf("authorizeBotDCSession() = %v, want wrapped USER_MIGRATE", err)
	}
	if !strings.Contains(err.Error(), "redirected to DC 4") {
		t.Errorf("error %q does not expose the redirected DC", err)
	}
	calls := fake.recorded()
	if len(calls) != 1 {
		t.Fatalf("import session calls = %d, want 1 (no import after failed export)", len(calls))
	}
}

func TestAuthorizeBotDCSessionUserMigrateFallbackImportFailure(t *testing.T) {
	client, server, fake := newBotTransferClient(t, func(input tg.TLObject) (tg.TLObject, error) {
		switch input.(type) {
		case *tg.AuthImportBotAuthorizationRequest:
			return nil, tgerr.New(303, "USER_MIGRATE_4")
		case *tg.AuthImportAuthorizationRequest:
			// Export succeeded but the server rejected the imported bytes.
			return nil, tgerr.New(400, "AUTH_BYTES_INVALID")
		default:
			return nil, fmt.Errorf("unexpected request %T", input)
		}
	})
	var exported tg.TLObject = &tg.AuthExportedAuthorization{ID: 9, Bytes: []byte{4, 5}}
	server.rpcResult.Store(&exported)

	err := client.authorizeBotDCSession(context.Background(), tg.NewRPCClient(fake), 2)
	if err == nil {
		t.Fatal("authorizeBotDCSession() = nil, want import failure")
	}
	if want := "auth transfer fallback failed: import auth on DC 2"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q does not contain %q", err, want)
	}
	var rpcErr *tgerr.Error
	if !errors.As(err, &rpcErr) || !rpcErr.IsType("USER_MIGRATE") {
		t.Fatalf("authorizeBotDCSession() = %v, want wrapped USER_MIGRATE preserved", err)
	}
	if calls := fake.recorded(); len(calls) != 2 {
		t.Fatalf("import session calls = %d, want 2", len(calls))
	}
}

func TestAuthorizeBotDCSessionNonRedirectErrorHasNoFallback(t *testing.T) {
	client, _, fake := newBotTransferClient(t, func(input tg.TLObject) (tg.TLObject, error) {
		switch input.(type) {
		case *tg.AuthImportBotAuthorizationRequest:
			return nil, tgerr.New(400, "AUTH_BYTES_INVALID")
		default:
			return nil, fmt.Errorf("unexpected request %T", input)
		}
	})

	err := client.authorizeBotDCSession(context.Background(), tg.NewRPCClient(fake), 2)
	if err == nil {
		t.Fatal("authorizeBotDCSession() = nil, want error")
	}
	if want := "import bot auth on DC 2"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q does not contain %q", err, want)
	}
	if strings.Contains(err.Error(), "redirected") {
		t.Errorf("error %q unexpectedly mentions a redirect", err)
	}
	if calls := fake.recorded(); len(calls) != 1 {
		t.Fatalf("import session calls = %d, want 1", len(calls))
	}
}

func TestAuthorizeSessionViaTransferPropagatesExportFailure(t *testing.T) {
	client, _, fake := newBotTransferClient(t, func(input tg.TLObject) (tg.TLObject, error) {
		if _, ok := input.(*tg.AuthImportAuthorizationRequest); !ok {
			return nil, fmt.Errorf("unexpected request %T", input)
		}
		return &tg.AuthAuthorization{}, nil
	})
	// No rpcResult: auth.exportAuthorization on the home DC fails.

	err := client.authorizeSessionViaTransfer(context.Background(), tg.NewRPCClient(fake), 3)
	if err == nil {
		t.Fatal("authorizeSessionViaTransfer() = nil, want export failure")
	}
	if want := "export auth for DC 3"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q does not contain %q", err, want)
	}
	if calls := fake.recorded(); len(calls) != 0 {
		t.Fatalf("import session calls = %d, want 0 after failed export", len(calls))
	}
}
