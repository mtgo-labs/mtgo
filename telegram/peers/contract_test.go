package peers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

func TestNotFoundErrorIdentity(t *testing.T) {
	err := &NotFoundError{Ref: "@ghost", Cause: tgerr.New(400, "USERNAME_NOT_OCCUPIED")}
	if !errors.Is(err, ErrNotFound) {
		t.Fatal("NotFoundError must satisfy errors.Is(err, ErrNotFound)")
	}
	if !errors.Is(err, telegramErrNotFoundAliasCheck()) {
		t.Fatal("NotFoundError must satisfy the telegram.ErrPeerNotFound alias")
	}
	var rpc *tgerr.Error
	if !errors.As(err, &rpc) {
		t.Fatal("cause RPC error must stay reachable via errors.As")
	}
}

func telegramErrNotFoundAliasCheck() error { return ErrNotFound }

func TestStaleHashInvokerReportsErrInvalidAfterFailedReplay(t *testing.T) {
	m := newTestManager(0, false, nil)
	calls := 0
	var next tg.Invoker = tg.InvokerFunc(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
		calls++
		return nil, tgerr.New(400, "PEER_ID_INVALID")
	})
	inv := m.InvalidateOnStaleHash()(next)

	_, err := inv.RPCInvoke(t.Context(),
		&tg.MessagesSendMessageRequest{Peer: &tg.InputPeerChannel{ChannelID: 55}}, nil)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid chain", err)
	}
	if !tgerr.Is(err, "PEER_ID_INVALID") {
		t.Fatalf("underlying rejection lost: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestCoalescerWaiterHonorsContext(t *testing.T) {
	m := newTestManager(0, false, nil)
	started := make(chan struct{})
	release := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())

	// Leader: takes the key and blocks in fn.
	leaderDone := make(chan struct{})
	go func() {
		defer close(leaderDone)
		_, _ = coalesce(m, t.Context(), "username:slow", func() (tg.InputPeerClass, error) {
			close(started)
			<-release
			return &tg.InputPeerSelf{}, nil
		})
	}()

	<-started
	// Waiter: joins the in-flight call with a cancellable context.
	waiterDone := make(chan error, 1)
	go func() {
		_, err := coalesce(m, ctx, "username:slow", func() (tg.InputPeerClass, error) {
			return &tg.InputPeerSelf{}, nil
		})
		waiterDone <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-waiterDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waiter err = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not honor ctx cancellation")
	}
	close(release)
	<-leaderDone
}

func TestPhoneReverseIndexOnInvalidate(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.Cache(9, &tg.InputPeerUser{UserID: 9, AccessHash: 1})
	m.CachePhone("+7900", 9)

	m.Invalidate(9)

	if _, ok := m.cachedByPhone("7900"); ok {
		t.Fatal("phone index survived Invalidate")
	}
}

func TestPhoneRebindDropsOldNumber(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.Cache(9, &tg.InputPeerUser{UserID: 9, AccessHash: 1})
	m.CachePhone("+7900", 9)
	m.CachePhone("+7800", 9)

	if _, ok := m.cachedByPhone("7900"); ok {
		t.Fatal("old phone number still indexed after rebind")
	}
	if _, ok := m.cachedByPhone("7800"); !ok {
		t.Fatal("new phone number missing")
	}
}
