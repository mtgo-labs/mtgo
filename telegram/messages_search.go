package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// SearchMessages searches for messages within a specific chat matching the given
// query string. Supports filtering by sender, date range, message type, and forum
// topic, as well as pagination via offset.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat to search within
//   - query: the search string; use empty string to match all messages (useful with filters)
//   - opts: optional SearchMessagesOption for pagination, date range, sender, and type filters
//
// Returns the matching messages on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the search query is rejected by the server
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	msgs, err := client.SearchMessages(ctx, chatID, "hello",
//	    &SearchMessagesOption{Limit: 10},
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, m := range msgs {
//	    fmt.Println(m.ID, m.Text)
//	}
func (c *Client) SearchMessages(ctx context.Context, chatID int64, query string, opts ...*SearchMessagesOption) ([]*types.Message, error) {
	c.Log.Debugf("SearchMessages chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	opt := getOptDef(&SearchMessagesOption{}, opts...)

	limit := int32(100)
	if opt.Limit > 0 {
		limit = int32(opt.Limit)
	}

	var flags tg.Fields
	if opt.FromID != nil {
		flags.Set(0)
	}
	if opt.TopMsgID != nil {
		flags.Set(1)
	}

	filter := opt.Filter
	if filter == nil {
		filter = &tg.InputMessagesFilterEmpty{}
	}

	req := &tg.MessagesSearchRequest{
		Flags:    flags,
		Peer:     peer,
		Q:        query,
		FromID:   opt.FromID,
		Filter:   filter,
		MinDate:  opt.MinDate,
		MaxDate:  opt.MaxDate,
		OffsetID: opt.OffsetID,
		Limit:    limit,
	}
	if opt.TopMsgID != nil {
		req.TopMsgID = *opt.TopMsgID
	}

	rpc := c.Raw()
	result, err := rpc.MessagesSearch(ctx, req)
	if err != nil {
		return nil, err
	}
	return extractMessagesFromMessagesClass(result, c)
}

// SearchGlobal performs a global search across all of the user's chats for messages
// matching the given query. Results can be scoped to channels only, groups only, or
// a specific chat folder.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - query: the search string
//   - opts: optional SearchGlobalOption for pagination, date range, chat type, and folder filters
//
// Returns the matching messages on success.
//
// Returns an error if:
//   - the search query is rejected by the server
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	msgs, err := client.SearchGlobal(ctx, "important announcement",
//	    &SearchGlobalOption{Limit: 20},
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, m := range msgs {
//	    fmt.Println(m.ID, m.Text)
//	}
func (c *Client) SearchGlobal(ctx context.Context, query string, opts ...*SearchGlobalOption) ([]*types.Message, error) {
	c.Log.Debug("SearchGlobal")
	opt := getOptDef(&SearchGlobalOption{}, opts...)

	limit := int32(100)
	if opt.Limit > 0 {
		limit = int32(opt.Limit)
	}

	var flags tg.Fields
	if opt.BroadcastsOnly {
		flags.Set(1)
	}
	if opt.GroupsOnly {
		flags.Set(2)
	}

	offsetPeer := opt.OffsetPeer
	if offsetPeer == nil {
		offsetPeer = &tg.InputPeerEmpty{}
	}

	filter := opt.Filter
	if filter == nil {
		filter = &tg.InputMessagesFilterEmpty{}
	}

	req := &tg.MessagesSearchGlobalRequest{
		Flags:      flags,
		Q:          query,
		Filter:     filter,
		MinDate:    opt.MinDate,
		MaxDate:    opt.MaxDate,
		OffsetRate: opt.OffsetRate,
		OffsetPeer: offsetPeer,
		OffsetID:   opt.OffsetID,
		Limit:      limit,
	}
	if opt.FolderID != nil {
		req.FolderID = *opt.FolderID
	}

	rpc := c.Raw()
	result, err := rpc.MessagesSearchGlobal(ctx, req)
	if err != nil {
		return nil, err
	}
	return extractMessagesFromMessagesClass(result, c)
}

// SearchMessagesCount returns the total number of messages matching the query within
// a specific chat. It performs a minimal search (limit=1) to extract the total count
// from the server response without fetching full message data.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat to search within
//   - query: the search string
//
// Returns the total number of matching messages on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails
func (c *Client) SearchMessagesCount(ctx context.Context, chatID int64, query string) (int32, error) {
	c.Log.Debugf("SearchMessagesCount chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return 0, fmt.Errorf("resolve peer: %w", err)
	}

	req := &tg.MessagesSearchRequest{
		Peer:   peer,
		Q:      query,
		Filter: &tg.InputMessagesFilterEmpty{},
		Limit:  1,
	}

	rpc := c.Raw()
	result, err := rpc.MessagesSearch(ctx, req)
	if err != nil {
		return 0, err
	}
	return extractMessagesCount(result)
}

// SearchGlobalCount returns the total number of messages matching the query across
// all of the user's chats. It performs a minimal search (limit=1) to extract the
// total count without fetching full message data.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - query: the search string
//
// Returns the total number of matching messages on success.
//
// Returns an error if:
//   - the RPC call fails
func (c *Client) SearchGlobalCount(ctx context.Context, query string) (int32, error) {
	c.Log.Debug("SearchGlobalCount")
	req := &tg.MessagesSearchGlobalRequest{
		Q:          query,
		Filter:     &tg.InputMessagesFilterEmpty{},
		Limit:      1,
		OffsetPeer: &tg.InputPeerEmpty{},
	}

	rpc := c.Raw()
	result, err := rpc.MessagesSearchGlobal(ctx, req)
	if err != nil {
		return 0, err
	}
	return extractMessagesCount(result)
}

func extractMessagesCount(result tg.MessagesClass) (int32, error) {
	switch v := result.(type) {
	case *tg.MessagesMessages:
		return int32(len(v.Messages)), nil
	case *tg.MessagesMessagesSlice:
		return v.Count, nil
	case *tg.MessagesChannelMessages:
		return v.Count, nil
	case *tg.MessagesMessagesNotModified:
		return 0, nil
	default:
		return 0, fmt.Errorf("unexpected messages type %T", result)
	}
}
