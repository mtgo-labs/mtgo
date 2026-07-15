package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// ApplyBoostOption configures optional parameters when applying a boost to a
// chat using [Client.ApplyBoost].
type ApplyBoostOption struct {
	// Slots specifies which boost slots to use. If empty, the next available
	// slot is used automatically.
	Slots []int32
}

// GetBoostsOption configures optional parameters when retrieving the current
// user's boosts using [Client.GetBoosts].
type GetBoostsOption struct {
	// Offset is the pagination offset string for fetching the next batch of
	// boosts.
	Offset string
	// Limit is the maximum number of boosts to return.
	Limit int32
}

// ApplyBoost applies one or more Telegram Premium boosts to the specified chat.
// Boosts help the chat unlock additional features (higher upload limits, custom
// colors, etc.).
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat to boost
//   - opts: optional [ApplyBoostOption] to specify which slots to use
//
// Returns the updated list of MyBoost objects, or an error if the peer cannot
// be resolved or the boost fails.
//
// Example:
//
//	ctx := context.Background()
//	boosts, err := client.ApplyBoost(ctx, chatID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Active boosts: %d\n", len(boosts))
func (c *Client) ApplyBoost(ctx context.Context, chatID int64, opts ...*ApplyBoostOption) ([]*tg.MyBoost, error) {
	c.Log.Debugf("ApplyBoost chat_id=%d", chatID)
	peer, err := resolvePeer(c.clientPeerResolver(), chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	opt := getOptDef(&ApplyBoostOption{}, opts...)

	req := &tg.PremiumApplyBoostRequest{
		Peer: peer,
	}
	if len(opt.Slots) > 0 {
		req.Flags |= (1 << 0)
		req.Slots = opt.Slots
	}

	rpc := c.Raw()
	result, err := rpc.PremiumApplyBoost(ctx, req)
	if err != nil {
		return nil, err
	}
	return result.MyBoosts, nil
}

// GetBoostsStatus retrieves the boost status for the specified chat, including
// the current boost level and how many more boosts are needed for the next
// level.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat to check
//
// Returns the PremiumBoostsStatus object or an error if the peer cannot be
// resolved or the RPC call fails.
//
// Example:
//
//	ctx := context.Background()
//	status, err := client.GetBoostsStatus(ctx, chatID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Boost level: %d\n", status.Level)
func (c *Client) GetBoostsStatus(ctx context.Context, chatID int64) (*tg.PremiumBoostsStatus, error) {
	c.Log.Debugf("GetBoostsStatus chat_id=%d", chatID)
	peer, err := resolvePeer(c.clientPeerResolver(), chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	return rpc.PremiumGetBoostsStatus(ctx, &tg.PremiumGetBoostsStatusRequest{
		Peer: peer,
	})
}

// GetBoosts retrieves all active boosts applied by the current user across all
// chats.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - opts: optional [GetBoostsOption] for pagination
//
// Returns a slice of MyBoost objects, or an error if the RPC call fails.
//
// Example:
//
//	ctx := context.Background()
//	boosts, err := client.GetBoosts(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, b := range boosts {
//	    fmt.Printf("Boost in chat %d, slot %d\n", b.Peer.PeerID, b.Slot)
//	}
func (c *Client) GetBoosts(ctx context.Context, opts ...*GetBoostsOption) ([]*tg.MyBoost, error) {
	c.Log.Debug("GetBoosts")

	rpc := c.Raw()
	result, err := rpc.PremiumGetMyBoosts(ctx)
	if err != nil {
		return nil, err
	}
	return result.MyBoosts, nil
}
