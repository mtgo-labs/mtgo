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
// user's boosts using [Client.GetMyBoosts].
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
func (c *PremiumMethods) ApplyBoost(ctx context.Context, chatID int64, opts ...*ApplyBoostOption) ([]*tg.MyBoost, error) {
	c.Log.Debugf("ApplyBoost chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
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
		return nil, fmt.Errorf("premium apply boost: %w", err)
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
func (c *PremiumMethods) GetBoostsStatus(ctx context.Context, chatID int64) (*tg.PremiumBoostsStatus, error) {
	c.Log.Debugf("GetBoostsStatus chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	return rpc.PremiumGetBoostsStatus(ctx, &tg.PremiumGetBoostsStatusRequest{
		Peer: peer,
	})
}

// GetMyBoosts retrieves all active boosts applied by the current user across all
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
//	boosts, err := client.GetMyBoosts(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, b := range boosts {
//	    fmt.Printf("Boost in chat %d, slot %d\n", b.Peer.PeerID, b.Slot)
//	}
func (c *PremiumMethods) GetMyBoosts(ctx context.Context, opts ...*GetBoostsOption) ([]*tg.MyBoost, error) {
	c.Log.Debug("GetBoosts")

	rpc := c.Raw()
	result, err := rpc.PremiumGetMyBoosts(ctx)
	if err != nil {
		return nil, fmt.Errorf("premium get my boosts: %w", err)
	}
	return result.MyBoosts, nil
}

// ChatBoosts is one page of the boosters of a chat.
type ChatBoosts struct {
	// Count is the total number of boosts applied to the chat.
	Count int32
	// Boosts are the boosts on this page, most recent first.
	Boosts []*tg.Boost
	// NextOffset fetches the next page; empty on the last page.
	NextOffset string
	// Users maps the boosters referenced by Boosts.
	Users []tg.UserClass
}

// GetChatBoosts lists the boosts applied to a chat, most recent first. Pass
// the previous page's next offset to continue; giftsOnly lists only gift and
// giveaway boosts.
func (c *PremiumMethods) GetChatBoosts(ctx context.Context, chatID int64, offset string, limit int, giftsOnly bool) (*ChatBoosts, error) {
	c.Log.Debugf("GetChatBoosts chat_id=%d offset=%q limit=%d gifts=%v", chatID, offset, limit, giftsOnly)
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	req := &tg.PremiumGetBoostsListRequest{
		Peer:   peer,
		Offset: offset,
		Limit:  int32(limit),
		Gifts:  giftsOnly,
	}
	result, err := c.Raw().PremiumGetBoostsList(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get chat boosts: %w", err)
	}
	return &ChatBoosts{
		Count:      result.Count,
		Boosts:     result.Boosts,
		NextOffset: result.NextOffset,
		Users:      result.Users,
	}, nil
}

// GetUserBoosts lists the boosts a specific user has applied to a chat.
func (c *PremiumMethods) GetUserBoosts(ctx context.Context, chatID, userID int64) ([]*tg.Boost, error) {
	c.Log.Debugf("GetUserBoosts chat_id=%d user_id=%d", chatID, userID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	user, err := resolveUserID(ctx, c.Client, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}
	result, err := c.Raw().PremiumGetUserBoosts(ctx, &tg.PremiumGetUserBoostsRequest{
		Peer:   peer,
		UserID: user,
	})
	if err != nil {
		return nil, fmt.Errorf("get user boosts: %w", err)
	}
	return result.Boosts, nil
}
