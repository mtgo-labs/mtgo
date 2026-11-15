package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// EditLiveLocation updates the coordinates of a live-location message.
func (c *MessageMethods) EditLiveLocation(ctx context.Context, chatID int64, messageID int32, lat, lng float64) (*types.Message, error) {
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, err
	}
	result, err := c.Raw().MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
		Peer: peer,
		ID:   messageID,
		Media: &tg.InputMediaGeoLive{
			GeoPoint: &tg.InputGeoPoint{Lat: lat, Long: lng},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("edit message: %w", err)
	}
	return extractSingleMessage(result, c)
}

// StopLiveLocation terminates a live-location message, keeping the last known
// coordinates.
func (c *MessageMethods) StopLiveLocation(ctx context.Context, chatID int64, messageID int32) (*types.Message, error) {
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, err
	}
	result, err := c.Raw().MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
		Peer: peer,
		ID:   messageID,
		Media: &tg.InputMediaGeoLive{
			Flags:   1 << 0,
			Stopped: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("edit message: %w", err)
	}
	return extractSingleMessage(result, c)
}

// SummarizeMessage asks the server to summarize a message and returns the
// summary as formatted text.
func (c *MessageMethods) SummarizeMessage(ctx context.Context, chatID int64, messageID int32) (*tg.TextWithEntities, error) {
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, err
	}
	result, err := c.Raw().MessagesSummarizeText(ctx, &tg.MessagesSummarizeTextRequest{
		Peer: peer,
		ID:   messageID,
	})
	if err != nil {
		return nil, fmt.Errorf("summarize message: %w", err)
	}
	return result, nil
}

// SendScreenshotNotification sends the secret-chat screenshot notification
// service message into a chat.
func (c *MessageMethods) SendScreenshotNotification(ctx context.Context, chatID int64, replyToMessageID int32) error {
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return err
	}
	var replyTo tg.InputReplyToClass
	if replyToMessageID != 0 {
		replyTo = &tg.InputReplyToMessage{ReplyToMsgID: replyToMessageID}
	}
	_, err = c.Raw().MessagesSendScreenshotNotification(ctx, &tg.MessagesSendScreenshotNotificationRequest{
		Peer:     peer,
		ReplyTo:  replyTo,
		RandomID: c.RandomID(),
	})
	if err != nil {
		return fmt.Errorf("send screenshot notification: %w", err)
	}
	return nil
}

// GetScheduledMessages retrieves scheduled messages of a chat. When IDs are
// empty, every scheduled message is returned.
func (c *MessageMethods) GetScheduledMessages(ctx context.Context, chatID int64, messageIDs []int32) ([]*types.Message, error) {
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, err
	}
	result, err := c.Raw().MessagesGetScheduledMessages(ctx, &tg.MessagesGetScheduledMessagesRequest{
		Peer: peer,
		ID:   messageIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("get scheduled messages: %w", err)
	}
	return extractMessagesFromMessagesClass(result, c)
}

// TranslateText translates raw formatted texts into the target language and
// returns the translated texts in input order.
func (c *MessageMethods) TranslateText(ctx context.Context, texts []*tg.TextWithEntities, toLang string) ([]*tg.TextWithEntities, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	result, err := c.Raw().MessagesTranslateText(ctx, &tg.MessagesTranslateTextRequest{
		Text:   texts,
		ToLang: toLang,
	})
	if err != nil {
		return nil, fmt.Errorf("translate text: %w", err)
	}
	return result.Result, nil
}

// TranslateMessageText translates the given messages of a chat into the target
// language and returns the translated texts in input order.
func (c *MessageMethods) TranslateMessageText(ctx context.Context, chatID int64, messageIDs []int32, toLang string) ([]*tg.TextWithEntities, error) {
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, err
	}
	result, err := c.Raw().MessagesTranslateText(ctx, &tg.MessagesTranslateTextRequest{
		Peer:   peer,
		ID:     messageIDs,
		ToLang: toLang,
	})
	if err != nil {
		return nil, fmt.Errorf("translate message text: %w", err)
	}
	return result.Result, nil
}

// SearchPostsResult is one page of the post search.
type SearchPostsResult struct {
	// Messages are the matching posts in chronological order.
	Messages []*types.Message
	// Count is the total number of matching posts.
	Count int32
	// NextOffsetRate and NextOffsetID position the next page.
	NextOffsetRate int32
	NextOffsetID   int32
	// NextOffsetPeer is the peer cursor for the next page.
	NextOffsetPeer tg.InputPeerClass
}

// SearchPosts searches public posts by hashtag or free-text query, ordered
// from oldest to newest. Pass an empty query to search by hashtag only.
func (c *MessageMethods) SearchPosts(ctx context.Context, query, hashtag string, offsetRate int32, offsetID int32, offsetPeer int64, limit int) (*SearchPostsResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	var offsetPeerInput tg.InputPeerClass
	if offsetPeer != 0 {
		p, err := resolvePeer(ctx, c.Client, offsetPeer)
		if err != nil {
			return nil, fmt.Errorf("search posts: resolve offset peer: %w", err)
		}
		offsetPeerInput = p
	}
	result, err := c.Raw().ChannelsSearchPosts(ctx, &tg.ChannelsSearchPostsRequest{
		Hashtag:    hashtag,
		Query:      query,
		OffsetRate: offsetRate,
		OffsetPeer: offsetPeerInput,
		OffsetID:   offsetID,
		Limit:      int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search posts: %w", err)
	}
	msgs, ok := result.(*tg.MessagesMessages)
	if ok {
		return &SearchPostsResult{
			Messages: parseMessageClasses(msgs.Messages, types.NewPeerMapFromClasses(msgs.Users, msgs.Chats), c),
			Count:    int32(len(msgs.Messages)),
		}, nil
	}
	slice, ok := result.(*tg.MessagesMessagesSlice)
	if !ok {
		return nil, fmt.Errorf("search posts: unexpected result %T", result)
	}
	return &SearchPostsResult{
		Messages:       parseMessageClasses(slice.Messages, types.NewPeerMapFromClasses(slice.Users, slice.Chats), c),
		Count:          slice.Count,
		NextOffsetRate: slice.NextRate,
	}, nil
}

// StartBot starts a conversation with a bot in the given chat, delivering the
// deep-link start parameter.
func (c *MessageMethods) StartBot(ctx context.Context, botID int64, chatID int64, startParam string) error {
	bot, err := resolveUserID(ctx, c.Client, botID)
	if err != nil {
		return err
	}
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return err
	}
	_, err = c.Raw().MessagesStartBot(ctx, &tg.MessagesStartBotRequest{
		Bot:        bot,
		Peer:       peer,
		RandomID:   c.RandomID(),
		StartParam: startParam,
	})
	if err != nil {
		return fmt.Errorf("start bot: %w", err)
	}
	return nil
}

// AddChecklistTasks appends tasks to a checklist (todo-list) message.
func (c *MessageMethods) AddChecklistTasks(ctx context.Context, chatID int64, messageID int32, tasks []*tg.TodoItem) error {
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return err
	}
	_, err = c.Raw().MessagesAppendTodoList(ctx, &tg.MessagesAppendTodoListRequest{
		Peer:  peer,
		MsgID: messageID,
		List:  tasks,
	})
	if err != nil {
		return fmt.Errorf("add checklist tasks: %w", err)
	}
	return nil
}

// MarkChecklistTasksAsDone toggles the completion state of checklist tasks:
// the tasks in done are marked completed, the tasks in undone are reverted.
func (c *MessageMethods) MarkChecklistTasksAsDone(ctx context.Context, chatID int64, messageID int32, done, undone []int32) error {
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return err
	}
	_, err = c.Raw().MessagesToggleTodoCompleted(ctx, &tg.MessagesToggleTodoCompletedRequest{
		Peer:        peer,
		MsgID:       messageID,
		Completed:   done,
		Incompleted: undone,
	})
	if err != nil {
		return fmt.Errorf("mark checklist tasks done: %w", err)
	}
	return nil
}

// SendLivePhoto sends a photo marked as a self-destructing live photo with the
// given time-to-live in seconds.
func (c *MessageMethods) SendLivePhoto(ctx context.Context, chatID int64, file *InputFile, ttlSeconds int32, caption string, opts ...*params.SendPhoto) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendPhoto{}, opts...)
	media, err := resolveFile(ctx, file, c.Client, mediaKindPhoto)
	if err != nil {
		return nil, fmt.Errorf("send_live_photo: %w", err)
	}
	if photo, ok := media.(*tg.InputMediaPhoto); ok {
		photo.TTLSeconds = ttlSeconds
	}
	return c.SendMedia(ctx, chatID, media, caption, opt.ToSendMsg())
}

// ResolveStarGiftOffer accepts or declines a star-gift purchase offer message
// and returns the updated message.
func (c *PaymentMethods) ResolveStarGiftOffer(ctx context.Context, offerMessageID int32, decline bool) (*types.Message, error) {
	result, err := c.Raw().PaymentsResolveStarGiftOffer(ctx, &tg.PaymentsResolveStarGiftOfferRequest{
		OfferMsgID: offerMessageID,
		Decline:    decline,
	})
	if err != nil {
		return nil, fmt.Errorf("payments resolve star gift offer: %w", err)
	}
	return extractSingleMessage(result, c)
}
