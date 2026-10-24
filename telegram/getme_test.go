package telegram

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// getMeMockInvoker answers users.getFullUser with a fixed user, counting
// server round-trips.
type getMeMockInvoker struct {
	user  *tg.User
	calls atomic.Int32
	err   error
}

func (m *getMeMockInvoker) RPCInvoke(_ context.Context, input tg.TLObject, _ func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	if m.err != nil {
		return nil, m.err
	}
	req, ok := input.(*tg.UsersGetFullUserRequest)
	if !ok {
		return nil, errors.New("unexpected request type")
	}
	if _, ok := req.ID.(*tg.InputUserSelf); !ok {
		return nil, errors.New("expected InputUserSelf")
	}
	m.calls.Add(1)
	return &tg.UsersUserFull{
		FullUser: &tg.UserFull{ID: m.user.ID},
		Users:    []tg.UserClass{m.user},
	}, nil
}

func (m *getMeMockInvoker) RPCInvokeRaw(context.Context, tg.TLObject) ([]byte, error) {
	return nil, errors.New("unexpected raw invoke")
}

// newPartialRestoreClient builds a connected client whose storage holds a
// user id but no profile fields — the state left behind by an externally
// minted session string.
func newPartialRestoreClient(t *testing.T, inv tg.Invoker) (*Client, *MemoryStorage) {
	t.Helper()
	c, _ := NewClient(1, "h", nil)
	st := NewMemoryStorage()
	if err := st.SetUserID(6946061806); err != nil {
		t.Fatal(err)
	}
	c.storage = st
	c.state.setConnected(true)
	if inv != nil {
		c.testInvoker = inv
	}
	return c, st
}

func TestGetMeRefreshesPartiallyRestoredUser(t *testing.T) {
	full := &tg.User{ID: 6946061806, FirstName: "Full", LastName: "Name", Username: "fresh"}
	mock := &getMeMockInvoker{user: full}
	c, st := newPartialRestoreClient(t, mock)

	if err := c.restoreAuthorizedUser(context.Background(), st, c.authGeneration.Load()); err != nil {
		t.Fatalf("restoreAuthorizedUser() error = %v", err)
	}

	c.mu.RLock()
	me := c.me
	partial := c.mePartial
	c.mu.RUnlock()
	if me == nil || me.ID != full.ID {
		t.Fatalf("restored me = %v, want id %d", me, full.ID)
	}
	if !partial {
		t.Fatal("restored user not marked partial")
	}
	if calls := mock.calls.Load(); calls != 0 {
		t.Fatalf("restore performed %d RPCs, want 0", calls)
	}

	got, err := c.GetMe(context.Background())
	if err != nil {
		t.Fatalf("GetMe() error = %v", err)
	}
	if got.FirstName != "Full" || got.LastName != "Name" || got.Username != "fresh" {
		t.Fatalf("GetMe() = %+v, want healed profile", got)
	}
	if calls := mock.calls.Load(); calls != 1 {
		t.Fatalf("GetMe() performed %d RPCs, want 1", calls)
	}

	// The refresh must heal the blank storage row, not just the cache.
	if fn, _ := st.FirstName(); fn != "Full" {
		t.Errorf("stored first name = %q, want %q", fn, "Full")
	}
	if un, _ := st.Username(); un != "fresh" {
		t.Errorf("stored username = %q, want %q", un, "fresh")
	}

	c.mu.RLock()
	partial = c.mePartial
	c.mu.RUnlock()
	if partial {
		t.Fatal("partial flag not cleared after refresh")
	}

	// After the one refresh, GetMe is served from cache even if the server
	// becomes unreachable.
	mock.err = errors.New("server unreachable")
	cached, err := c.GetMe(context.Background())
	if err != nil {
		t.Fatalf("cached GetMe() error = %v", err)
	}
	if cached.Username != "fresh" {
		t.Fatalf("cached GetMe() = %+v, want healed user", cached)
	}
}

func TestGetMePartialRefreshFailureIsRetryable(t *testing.T) {
	full := &tg.User{ID: 6946061806, FirstName: "Full", Username: "fresh"}
	mock := &getMeMockInvoker{user: full}
	c, _ := newPartialRestoreClient(t, mock)

	if err := c.restoreAuthorizedUser(context.Background(), c.storage, c.authGeneration.Load()); err != nil {
		t.Fatalf("restoreAuthorizedUser() error = %v", err)
	}

	mock.err = errors.New("transient failure")
	if _, err := c.GetMe(context.Background()); err == nil {
		t.Fatal("GetMe() = nil error, want refresh failure surfaced")
	}

	c.mu.RLock()
	partial := c.mePartial
	c.mu.RUnlock()
	if !partial {
		t.Fatal("partial flag cleared despite refresh failure")
	}

	mock.err = nil
	got, err := c.GetMe(context.Background())
	if err != nil {
		t.Fatalf("GetMe() after recovery error = %v", err)
	}
	if got.Username != "fresh" {
		t.Fatalf("GetMe() = %+v, want healed user", got)
	}
}

func TestMeRefreshesPartiallyRestoredUser(t *testing.T) {
	full := &tg.User{ID: 6946061806, FirstName: "Full", Username: "fresh"}
	mock := &getMeMockInvoker{user: full}
	c, _ := newPartialRestoreClient(t, mock)

	if err := c.restoreAuthorizedUser(context.Background(), c.storage, c.authGeneration.Load()); err != nil {
		t.Fatalf("restoreAuthorizedUser() error = %v", err)
	}

	got := c.Me()
	if got == nil || got.Username != "fresh" {
		t.Fatalf("Me() = %+v, want healed user", got)
	}
	if calls := mock.calls.Load(); calls != 1 {
		t.Fatalf("Me() performed %d RPCs, want 1", calls)
	}
}

func TestAuthenticateUserUserAccountMarksMePartial(t *testing.T) {
	mock := &getMeMockInvoker{err: errors.New("no RPC expected during restore")}
	c, _ := NewClient(1, "h", &Config{BotToken: "123:token"})
	st := NewMemoryStorage()
	if err := st.SetUserID(6946061806); err != nil {
		t.Fatal(err)
	}
	c.storage = st
	c.state.setConnected(true)
	c.testInvoker = mock

	if err := c.authenticateUser(st, time.Second); err != nil {
		t.Fatalf("authenticateUser() error = %v", err)
	}

	c.mu.RLock()
	me := c.me
	partial := c.mePartial
	c.mu.RUnlock()
	if me == nil || me.ID != 6946061806 {
		t.Fatalf("restored me = %v, want id 6946061806", me)
	}
	if !partial {
		t.Fatal("user-account restore did not mark me partial")
	}

	// A full user committed on a real authorization path clears the flag.
	c.SetMe(&types.User{ID: 6946061806, FirstName: "Full"})
	c.mu.RLock()
	partial = c.mePartial
	c.mu.RUnlock()
	if partial {
		t.Fatal("SetMe did not clear partial flag")
	}
}
