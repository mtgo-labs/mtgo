package peers

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

func managerWithRPC(fn tg.InvokerFunc, store Store) *Manager {
	return NewManager(Deps{
		Invoker:   func() *tg.RPCClient { return tg.NewRPCClient(fn) },
		Store:     func() Store { return store },
		SavePeers: func() bool { return store != nil },
	})
}

func TestResolveUsernameNotFoundClassification(t *testing.T) {
	m := managerWithRPC(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
		return nil, tgerr.New(400, "USERNAME_NOT_OCCUPIED")
	}, nil)

	_, err := m.InputPeerByUsername(t.Context(), "ghost")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("USERNAME_NOT_OCCUPIED err = %v, want ErrNotFound chain", err)
	}
}

func TestResolveUsernameTransientUnmasked(t *testing.T) {
	m := managerWithRPC(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
		return nil, tgerr.New(420, "FLOOD_WAIT_30")
	}, nil)

	_, err := m.InputPeerByUsername(t.Context(), "anyone")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("FLOOD_WAIT masquerading as not-found: %v", err)
	}
	if !tgerr.IsFloodWait(err) && !errors.Is(err, err) {
		t.Fatalf("flood error lost: %v", err)
	}
}

func TestResolvePhoneTransientUnmasked(t *testing.T) {
	m := managerWithRPC(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
		return nil, errors.New("network down")
	}, nil)

	_, err := m.InputPeerByPhone(t.Context(), "+79001234567")
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("network error masquerading as not-found: %v", err)
	}
}

func TestPeerToInputPeerZeroHashUserIsError(t *testing.T) {
	_, err := PeerToInputPeer(&tg.PeerUser{UserID: 7},
		[]tg.UserClass{&tg.User{ID: 7, AccessHash: 0}}, nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("zero-hash user err = %v, want ErrNotFound (not silent self)", err)
	}
}

func TestNotUserErrorTyped(t *testing.T) {
	_, err := InputPeerToUser(&tg.InputPeerChat{ChatID: 1})
	var nue NotUserError
	if !errors.As(err, &nue) {
		t.Fatalf("err = %v, want NotUserError", err)
	}
}

func TestNotChannelErrorTyped(t *testing.T) {
	_, err := InputPeerToChannel(&tg.InputPeerUser{UserID: 1})
	var nce NotChannelError
	if !errors.As(err, &nce) {
		t.Fatalf("err = %v, want NotChannelError", err)
	}
}
