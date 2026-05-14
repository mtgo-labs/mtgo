package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestApplyBoost_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.ApplyBoost(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for unresolvable peer")
	}
}

func TestApplyBoost_SlotOption(t *testing.T) {
	cfg := &ApplyBoostOption{Slots: []int32{1, 2, 3}}
	if len(cfg.Slots) != 3 {
		t.Fatalf("expected 3 slots, got %d", len(cfg.Slots))
	}
	if cfg.Slots[0] != 1 || cfg.Slots[1] != 2 || cfg.Slots[2] != 3 {
		t.Errorf("Slots = %v, want [1 2 3]", cfg.Slots)
	}
}

func TestApplyBoost_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	mock.setResult(tg.PremiumApplyBoostTypeID, &tg.PremiumMyBoosts{
		MyBoosts: []*tg.MyBoost{
			{Slot: 1},
			{Slot: 2},
		},
	})

	boosts, err := c.ApplyBoost(context.Background(), 10, &ApplyBoostOption{Slots: []int32{1, 2}})
	if err != nil {
		t.Fatalf("ApplyBoost() error: %v", err)
	}
	if len(boosts) != 2 {
		t.Fatalf("expected 2 boosts, got %d", len(boosts))
	}
	if boosts[0].Slot != 1 {
		t.Errorf("Slot[0] = %d, want 1", boosts[0].Slot)
	}
	req, ok := mock.lastCall().(*tg.PremiumApplyBoostRequest)
	if !ok {
		t.Fatalf("expected PremiumApplyBoostRequest, got %T", mock.lastCall())
	}
	if req.Flags&(1<<0) == 0 {
		t.Error("expected slots flag to be set")
	}
}

func TestApplyBoost_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	mock.setError(tg.PremiumApplyBoostTypeID, errors.New("rpc error"))

	_, err := c.ApplyBoost(context.Background(), 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetBoostsStatus_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.GetBoostsStatus(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for unresolvable peer")
	}
}

func TestGetBoostsStatus_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	mock.setResult(tg.PremiumGetBoostsStatusTypeID, &tg.PremiumBoostsStatus{
		Boosts: 5,
	})

	status, err := c.GetBoostsStatus(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetBoostsStatus() error: %v", err)
	}
	if status.Boosts != 5 {
		t.Errorf("Boosts = %d, want 5", status.Boosts)
	}
}

func TestGetBoostsStatus_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	mock.setError(tg.PremiumGetBoostsStatusTypeID, errors.New("rpc error"))

	_, err := c.GetBoostsStatus(context.Background(), 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetBoosts_BasicOption(t *testing.T) {
	cfg := &GetBoostsOption{Offset: "abc"}
	if cfg.Offset != "abc" {
		t.Errorf("Offset = %q, want %q", cfg.Offset, "abc")
	}
}

func TestGetBoosts_LimitOption(t *testing.T) {
	cfg := &GetBoostsOption{Limit: 50}
	if cfg.Limit != 50 {
		t.Errorf("Limit = %d, want 50", cfg.Limit)
	}
}

func TestGetBoosts_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setResult(tg.PremiumGetMyBoostsTypeID, &tg.PremiumMyBoosts{
		MyBoosts: []*tg.MyBoost{
			{Slot: 1},
		},
	})

	boosts, err := c.GetBoosts(context.Background())
	if err != nil {
		t.Fatalf("GetBoosts() error: %v", err)
	}
	if len(boosts) != 1 {
		t.Fatalf("expected 1 boost, got %d", len(boosts))
	}
}

func TestGetBoosts_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setError(tg.PremiumGetMyBoostsTypeID, errors.New("rpc error"))

	_, err := c.GetBoosts(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
