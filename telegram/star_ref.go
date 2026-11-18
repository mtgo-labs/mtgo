package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// UpdateStarRefProgram configures a bot's affiliate program: the commission
// (in thousandths) affiliates earn and, optionally, how many months the
// commission keeps accruing. Only the bot's owner may call it.
func (c *BotsMethods) UpdateStarRefProgram(ctx context.Context, botID int64, commissionPermille int32, durationMonths int32) (*tg.StarRefProgram, error) {
	c.Log.Debugf("UpdateStarRefProgram bot=%d commission=%d", botID, commissionPermille)
	bot, err := resolveUserID(ctx, c.Client, botID)
	if err != nil {
		return nil, fmt.Errorf("resolve bot: %w", err)
	}
	req := &tg.BotsUpdateStarRefProgramRequest{
		Bot:                bot,
		CommissionPermille: commissionPermille,
	}
	if durationMonths > 0 {
		req.DurationMonths = durationMonths
	}
	result, err := c.Raw().BotsUpdateStarRefProgram(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("update star ref program: %w", err)
	}
	return result, nil
}

// ConnectedStarRef is one affiliate connection: the referral link, its
// commission terms, and the revenue it has generated.
type ConnectedStarRef struct {
	*tg.ConnectedBotStarRef
	// Bot is the program's bot.
	Bot *types.User
}

func (c *Client) parseConnectedStarRefs(result *tg.PaymentsConnectedStarRefBots) []ConnectedStarRef {
	out := make([]ConnectedStarRef, 0, len(result.ConnectedBots))
	users := make(map[int64]*types.User, len(result.Users))
	for _, u := range result.Users {
		if user, ok := u.(*tg.User); ok {
			users[user.ID] = types.ParseUser(u)
		}
	}
	for _, connected := range result.ConnectedBots {
		entry := ConnectedStarRef{ConnectedBotStarRef: connected, Bot: users[connected.BotID]}
		out = append(out, entry)
	}
	return out
}

// GetConnectedStarRefs lists the affiliate programs a peer has joined,
// newest first. offsetDate and offsetLink continue a previous page.
func (c *PaymentMethods) GetConnectedStarRefs(ctx context.Context, peerID int64, offsetDate int32, offsetLink string, limit int32) ([]ConnectedStarRef, error) {
	c.Log.Debugf("GetConnectedStarRefs peer=%d", peerID)
	peer, err := resolvePeer(ctx, c.Client, peerID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().PaymentsGetConnectedStarRefBots(ctx, &tg.PaymentsGetConnectedStarRefBotsRequest{
		Peer:       peer,
		OffsetDate: offsetDate,
		OffsetLink: offsetLink,
		Limit:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get connected star refs: %w", err)
	}
	return c.parseConnectedStarRefs(result), nil
}

// GetConnectedStarRef returns the peer's affiliate connection with one bot,
// or an empty slice when the peer has not joined that program.
func (c *PaymentMethods) GetConnectedStarRef(ctx context.Context, peerID, botID int64) ([]ConnectedStarRef, error) {
	c.Log.Debugf("GetConnectedStarRef peer=%d bot=%d", peerID, botID)
	peer, err := resolvePeer(ctx, c.Client, peerID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	bot, err := resolveUserID(ctx, c.Client, botID)
	if err != nil {
		return nil, fmt.Errorf("resolve bot: %w", err)
	}
	result, err := c.Raw().PaymentsGetConnectedStarRefBot(ctx, &tg.PaymentsGetConnectedStarRefBotRequest{
		Peer: peer,
		Bot:  bot,
	})
	if err != nil {
		return nil, fmt.Errorf("get connected star ref: %w", err)
	}
	return c.parseConnectedStarRefs(result), nil
}

// SuggestedStarRefBots is one page of [PaymentMethods.GetSuggestedStarRefs]
// results: affiliate programs the peer has not joined yet.
type SuggestedStarRefBots struct {
	// Count is the total number of suggested programs.
	Count int32
	// Programs with their bots.
	Programs []*tg.StarRefProgram
	// Bots maps bot IDs to their users.
	Bots *types.PeerMap
	// NextOffset pages the list; empty at the end.
	NextOffset string
}

// GetSuggestedStarRefs lists affiliate programs worth joining, ordered by
// revenue or by recency.
func (c *PaymentMethods) GetSuggestedStarRefs(ctx context.Context, peerID int64, orderByRevenue bool, offset string, limit int32) (*SuggestedStarRefBots, error) {
	c.Log.Debugf("GetSuggestedStarRefs peer=%d", peerID)
	peer, err := resolvePeer(ctx, c.Client, peerID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().PaymentsGetSuggestedStarRefBots(ctx, &tg.PaymentsGetSuggestedStarRefBotsRequest{
		OrderByRevenue: orderByRevenue,
		Peer:           peer,
		Offset:         offset,
		Limit:          limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get suggested star refs: %w", err)
	}
	return &SuggestedStarRefBots{
		Count:      result.Count,
		Programs:   result.SuggestedBots,
		Bots:       types.NewPeerMapFromClasses(result.Users, nil),
		NextOffset: result.NextOffset,
	}, nil
}

// ConnectStarRef joins a bot's affiliate program, returning the peer's
// referral link.
func (c *PaymentMethods) ConnectStarRef(ctx context.Context, peerID, botID int64) ([]ConnectedStarRef, error) {
	c.Log.Debugf("ConnectStarRef peer=%d bot=%d", peerID, botID)
	peer, err := resolvePeer(ctx, c.Client, peerID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	bot, err := resolveUserID(ctx, c.Client, botID)
	if err != nil {
		return nil, fmt.Errorf("resolve bot: %w", err)
	}
	result, err := c.Raw().PaymentsConnectStarRefBot(ctx, &tg.PaymentsConnectStarRefBotRequest{
		Peer: peer,
		Bot:  bot,
	})
	if err != nil {
		return nil, fmt.Errorf("connect star ref: %w", err)
	}
	return c.parseConnectedStarRefs(result), nil
}

// EditConnectedStarRef revokes or restores one of the peer's referral links.
func (c *PaymentMethods) EditConnectedStarRef(ctx context.Context, peerID int64, link string, revoked bool) ([]ConnectedStarRef, error) {
	c.Log.Debugf("EditConnectedStarRef peer=%d revoked=%v", peerID, revoked)
	peer, err := resolvePeer(ctx, c.Client, peerID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().PaymentsEditConnectedStarRefBot(ctx, &tg.PaymentsEditConnectedStarRefBotRequest{
		Revoked: revoked,
		Peer:    peer,
		Link:    link,
	})
	if err != nil {
		return nil, fmt.Errorf("edit connected star ref: %w", err)
	}
	return c.parseConnectedStarRefs(result), nil
}
