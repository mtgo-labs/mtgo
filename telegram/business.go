package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// GetBusinessConnection retrieves a business connection by its ID.
// It returns the BotBusinessConnection associated with the given connectionID,
// or an error if the connection ID is empty, the request fails, or no
// matching connection is found in the response updates.
//
// Example:
//
//	conn, err := client.GetBusinessConnection(ctx, "conn_12345")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Business connection: %+v\n", conn)
func (c *BusinessMethods) GetBusinessConnection(ctx context.Context, connectionID string) (*tg.BotBusinessConnection, error) {
	if connectionID == "" {
		return nil, ErrBusinessConnIDRequired
	}

	c.Log.Debugf("GetBusinessConnection id=%s", connectionID)

	rpc := c.Raw()
	result, err := rpc.AccountGetBotBusinessConnection(ctx, &tg.AccountGetBotBusinessConnectionRequest{
		ConnectionID: connectionID,
	})
	if err != nil {
		return nil, err
	}

	switch v := result.(type) {
	case *tg.Updates:
		for _, u := range v.Updates {
			if upd, ok := u.(*tg.UpdateBotBusinessConnect); ok {
				return upd.Connection, nil
			}
		}
		return nil, ErrNoBusinessConnection
	default:
		return nil, fmt.Errorf("business: unexpected updates type %T", result)
	}
}

// SetBusinessWorkHours sets the logged-in user's business opening hours shown
// on their profile. Pass nil to remove them.
func (c *BusinessMethods) SetBusinessWorkHours(ctx context.Context, hours *tg.BusinessWorkHours) error {
	if hours != nil {
		c.Log.Debugf("SetBusinessWorkHours tz=%s", hours.TimezoneID)
	} else {
		c.Log.Debug("SetBusinessWorkHours clear")
	}
	_, err := c.Raw().AccountUpdateBusinessWorkHours(ctx, &tg.AccountUpdateBusinessWorkHoursRequest{
		BusinessWorkHours: hours,
	})
	if err != nil {
		return fmt.Errorf("set business work hours: %w", err)
	}
	return nil
}

// SetBusinessLocation sets the business location shown on the logged-in user's
// profile. Provide coordinates, a text address, or both; leaving both empty
// clears the location.
func (c *BusinessMethods) SetBusinessLocation(ctx context.Context, address string, lat, lng float64) error {
	c.Log.Debugf("SetBusinessLocation address=%q geo=%v,%v", address, lat, lng)
	req := &tg.AccountUpdateBusinessLocationRequest{
		Address: address,
	}
	if lat != 0 || lng != 0 {
		req.GeoPoint = &tg.InputGeoPoint{Lat: lat, Long: lng}
	}
	_, err := c.Raw().AccountUpdateBusinessLocation(ctx, req)
	if err != nil {
		return fmt.Errorf("set business location: %w", err)
	}
	return nil
}

// SetBusinessGreetingMessage sets the automated greeting message sent to new
// business chats. Pass nil to disable it.
func (c *BusinessMethods) SetBusinessGreetingMessage(ctx context.Context, message *tg.InputBusinessGreetingMessage) error {
	c.Log.Debug("SetBusinessGreetingMessage")
	_, err := c.Raw().AccountUpdateBusinessGreetingMessage(ctx, &tg.AccountUpdateBusinessGreetingMessageRequest{
		Message: message,
	})
	if err != nil {
		return fmt.Errorf("set business greeting message: %w", err)
	}
	return nil
}

// SetBusinessAwayMessage sets the automated away message sent while the
// business account is closed. Pass nil to disable it.
func (c *BusinessMethods) SetBusinessAwayMessage(ctx context.Context, message *tg.InputBusinessAwayMessage) error {
	c.Log.Debug("SetBusinessAwayMessage")
	_, err := c.Raw().AccountUpdateBusinessAwayMessage(ctx, &tg.AccountUpdateBusinessAwayMessageRequest{
		Message: message,
	})
	if err != nil {
		return fmt.Errorf("set business away message: %w", err)
	}
	return nil
}

// SetBusinessIntro sets the intro new customers see when opening the business
// chat. Pass nil to reset it to the default.
func (c *BusinessMethods) SetBusinessIntro(ctx context.Context, intro *tg.InputBusinessIntro) error {
	c.Log.Debug("SetBusinessIntro")
	_, err := c.Raw().AccountUpdateBusinessIntro(ctx, &tg.AccountUpdateBusinessIntroRequest{
		Intro: intro,
	})
	if err != nil {
		return fmt.Errorf("set business intro: %w", err)
	}
	return nil
}

// GetBusinessChatLinks lists the chat links configured for the logged-in
// business account.
func (c *BusinessMethods) GetBusinessChatLinks(ctx context.Context) ([]*tg.BusinessChatLink, error) {
	c.Log.Debug("GetBusinessChatLinks")
	result, err := c.Raw().AccountGetBusinessChatLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("get business chat links: %w", err)
	}
	return result.Links, nil
}

// CreateBusinessChatLink creates a new chat link for the logged-in business
// account from the given link definition.
func (c *BusinessMethods) CreateBusinessChatLink(ctx context.Context, link *tg.InputBusinessChatLink) (*tg.BusinessChatLink, error) {
	c.Log.Debug("CreateBusinessChatLink")
	result, err := c.Raw().AccountCreateBusinessChatLink(ctx, &tg.AccountCreateBusinessChatLinkRequest{
		Link: link,
	})
	if err != nil {
		return nil, fmt.Errorf("create business chat link: %w", err)
	}
	return result, nil
}

// EditBusinessChatLink replaces the definition of an existing business chat
// link, identified by its slug.
func (c *BusinessMethods) EditBusinessChatLink(ctx context.Context, slug string, link *tg.InputBusinessChatLink) (*tg.BusinessChatLink, error) {
	c.Log.Debugf("EditBusinessChatLink slug=%s", slug)
	result, err := c.Raw().AccountEditBusinessChatLink(ctx, &tg.AccountEditBusinessChatLinkRequest{
		Slug: slug,
		Link: link,
	})
	if err != nil {
		return nil, fmt.Errorf("edit business chat link: %w", err)
	}
	return result, nil
}

// DeleteBusinessChatLink deletes a business chat link by its slug.
func (c *BusinessMethods) DeleteBusinessChatLink(ctx context.Context, slug string) error {
	c.Log.Debugf("DeleteBusinessChatLink slug=%s", slug)
	_, err := c.Raw().AccountDeleteBusinessChatLink(ctx, &tg.AccountDeleteBusinessChatLinkRequest{
		Slug: slug,
	})
	if err != nil {
		return fmt.Errorf("delete business chat link: %w", err)
	}
	return nil
}

// ResolveBusinessChatLink resolves a business chat link slug and returns the
// destination peer and greeting message.
func (c *BusinessMethods) ResolveBusinessChatLink(ctx context.Context, slug string) (*tg.AccountResolvedBusinessChatLinks, error) {
	c.Log.Debugf("ResolveBusinessChatLink slug=%s", slug)
	result, err := c.Raw().AccountResolveBusinessChatLink(ctx, &tg.AccountResolveBusinessChatLinkRequest{
		Slug: slug,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve business chat link: %w", err)
	}
	return result, nil
}
