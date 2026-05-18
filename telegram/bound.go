package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// BoundSend sends a text message to the specified chat. This is a bound-method
// convenience wrapper that creates a background context and delegates to
// [Client.SendMessage]. The replyTo parameter sets the message ID to reply to;
// it can be overridden via opts.
//
// Parameters:
//   - chatID: target chat identifier (user ID, group ID, or channel ID)
//   - text: message text to send
//   - replyTo: message ID to reply to (0 for no reply)
//   - opts: optional [params.SendMessage] overrides
//
// Returns the sent message or an error if the send fails.
func (c *Client) BoundSend(chatID int64, text string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.SendMessage{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendMessage(ctx, chatID, text, opt)
}

// BoundSendMedia sends a media attachment to the specified chat with an optional
// caption. This is a bound-method convenience wrapper that creates a background
// context and delegates to [Client.SendMedia].
//
// Parameters:
//   - chatID: target chat identifier
//   - media: Telegram input media object describing the attachment
//   - caption: optional caption text for the media
//   - replyTo: message ID to reply to (0 for no reply)
//   - opts: optional [params.SendMessage] overrides
//
// Returns the sent message or an error if the send fails.
func (c *Client) BoundSendMedia(chatID int64, media tg.InputMediaClass, caption string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.SendMessage{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendMedia(ctx, chatID, media, caption, opt)
}

// BoundForward forwards a single message from one chat to another. This is a
// bound-method convenience wrapper around [Client.ForwardMessages] that handles
// single-message forwarding.
//
// Parameters:
//   - chatID: destination chat identifier
//   - fromChatID: source chat identifier
//   - msgID: message ID to forward
//   - opts: optional [params.ForwardMessages] overrides
//
// Returns the forwarded message, nil if no messages were returned, or an error
// if the forward fails.
func (c *Client) BoundForward(chatID int64, fromChatID int64, msgID int32, opts ...*params.ForwardMessages) (*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.ForwardMessages{}, opts...)
	msgs, err := c.ForwardMessages(ctx, chatID, fromChatID, []int32{msgID}, opt)
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, nil
	}
	return msgs[0], nil
}

// BoundCopy copies a single message from one chat to another without the
// forward header. This is a bound-method convenience wrapper around
// [Client.CopyMessage].
//
// Parameters:
//   - chatID: destination chat identifier
//   - fromChatID: source chat identifier
//   - msgID: message ID to copy
//   - opts: optional [params.CopyMessage] overrides
//
// Returns the new message ID or an error if the copy fails.
func (c *Client) BoundCopy(chatID int64, fromChatID int64, msgID int32, opts ...*params.CopyMessage) (int64, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.CopyMessage{}, opts...)
	return c.CopyMessage(ctx, chatID, fromChatID, msgID, opt)
}

// BoundEdit edits the text of an existing message in the specified chat. This
// is a bound-method convenience wrapper around [Client.EditMessageText].
//
// Parameters:
//   - chatID: chat containing the message to edit
//   - msgID: ID of the message to edit
//   - text: new text content
//   - opts: optional [params.EditMessage] overrides
//
// Returns the edited message or an error if the edit fails.
func (c *Client) BoundEdit(chatID int64, msgID int32, text string, opts ...*params.EditMessage) (*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.EditMessage{}, opts...)
	return c.EditMessageText(ctx, chatID, msgID, text, opt)
}

// BoundEditCaption edits the caption of an existing media message. This is a
// bound-method convenience wrapper around [Client.EditMessageCaption].
//
// Parameters:
//   - chatID: chat containing the message to edit
//   - msgID: ID of the media message whose caption to edit
//   - caption: new caption text
//   - opts: optional [params.EditMessage] overrides
//
// Returns the edited message or an error if the edit fails.
func (c *Client) BoundEditCaption(chatID int64, msgID int32, caption string, opts ...*params.EditMessage) (*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.EditMessage{}, opts...)
	return c.EditMessageCaption(ctx, chatID, msgID, caption, opt)
}

// BoundEditMedia replaces the media content of an existing message. This is a
// bound-method convenience wrapper around [Client.EditMessageMedia].
//
// Parameters:
//   - chatID: chat containing the message to edit
//   - msgID: ID of the message whose media to replace
//   - media: new media content
//
// Returns the edited message or an error if the edit fails.
func (c *Client) BoundEditMedia(chatID int64, msgID int32, media tg.InputMediaClass) (*types.Message, error) {
	ctx := context.Background()
	return c.EditMessageMedia(ctx, chatID, msgID, media)
}

// BoundEditReplyMarkup changes the inline keyboard of an existing message. This
// is a bound-method convenience wrapper around [Client.EditMessageReplyMarkup].
//
// Parameters:
//   - chatID: chat containing the message to edit
//   - msgID: ID of the message whose reply markup to change
//   - markup: new reply markup (inline keyboard)
//
// Returns the edited message or an error if the edit fails.
func (c *Client) BoundEditReplyMarkup(chatID int64, msgID int32, markup tg.ReplyMarkupClass) (*types.Message, error) {
	ctx := context.Background()
	return c.EditMessageReplyMarkup(ctx, chatID, msgID, markup)
}

func (c *Client) BoundEditInline(inlineMessageID tg.InputBotInlineMessageIDClass, text string, opts ...*params.EditMessage) (bool, error) {
	ctx := context.Background()
	return c.EditInlineText(ctx, inlineMessageID, text)
}

func (c *Client) BoundEditInlineCaption(inlineMessageID tg.InputBotInlineMessageIDClass, caption string, opts ...*params.EditMessage) (bool, error) {
	ctx := context.Background()
	return c.EditInlineCaption(ctx, inlineMessageID, caption)
}

func (c *Client) BoundEditInlineMedia(inlineMessageID tg.InputBotInlineMessageIDClass, media tg.InputMediaClass) (bool, error) {
	ctx := context.Background()
	return c.EditInlineMedia(ctx, inlineMessageID, media)
}

func (c *Client) BoundEditInlineReplyMarkup(inlineMessageID tg.InputBotInlineMessageIDClass, markup tg.ReplyMarkupClass) (bool, error) {
	ctx := context.Background()
	return c.EditInlineReplyMarkup(ctx, inlineMessageID, markup)
}

// BoundDelete deletes one or more messages from the specified chat. This is a
// bound-method convenience wrapper around [Client.DeleteMessages].
//
// Parameters:
//   - chatID: chat containing the messages to delete
//   - msgIDs: slice of message IDs to delete
//   - opts: optional [params.DeleteMessages] overrides
//
// Returns the number of messages deleted or an error if the deletion fails.
func (c *Client) BoundDelete(chatID int64, msgIDs []int32, opts ...*params.DeleteMessages) (int, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.DeleteMessages{}, opts...)
	return c.DeleteMessages(ctx, chatID, msgIDs, opt)
}

// BoundReact adds emoji reactions to a message. This is a bound-method
// convenience wrapper that converts emoji strings to [tg.ReactionEmoji] objects
// and delegates to [Client.SendReaction].
//
// Parameters:
//   - chatID: chat containing the target message
//   - msgID: ID of the message to react to
//   - emojis: slice of emoji strings to react with (e.g. "👍", "❤️")
//
// Returns an error if the reaction fails.
func (c *Client) BoundReact(chatID int64, msgID int32, opts ...*params.React) error {
	ctx := context.Background()
	o := params.GetOptDef(&params.React{}, opts...)
	reactions := []tg.ReactionClass{&tg.ReactionEmoji{Emoticon: o.Emoji}}
	return c.SendReaction(ctx, chatID, msgID, reactions...)
}

// BoundPin pins a message in the specified chat. This is a bound-method
// convenience wrapper around [Client.PinMessage].
//
// Parameters:
//   - chatID: chat where the message should be pinned
//   - msgID: ID of the message to pin
//   - opts: optional [params.PinMessage] overrides
//
// Returns an error if the pin operation fails.
func (c *Client) BoundPin(chatID int64, msgID int32, opts ...*params.PinMessage) error {
	ctx := context.Background()
	opt := params.GetOptDef(&params.PinMessage{}, opts...)
	_, err := c.PinMessage(ctx, chatID, msgID, opt)
	return err
}

// BoundUnpin removes a pinned message from the specified chat. This is a
// bound-method convenience wrapper around [Client.UnpinMessage].
//
// Parameters:
//   - chatID: chat where the message is pinned
//   - msgID: ID of the message to unpin
//   - opts: optional [params.PinMessage] overrides (currently unused)
//
// Returns an error if the unpin operation fails.
func (c *Client) BoundUnpin(chatID int64, msgID int32, opts ...*params.PinMessage) error {
	ctx := context.Background()
	_, err := c.UnpinMessage(ctx, chatID, msgID)
	return err
}

// BoundRead marks messages as read up to the specified message ID in the given
// chat. This is a bound-method convenience wrapper around
// [Client.ReadHistory].
//
// Parameters:
//   - chatID: chat whose history to mark as read
//   - msgID: message ID up to which messages are considered read
//
// Returns an error if the operation fails.
func (c *Client) BoundRead(chatID int64, msgID int32) error {
	ctx := context.Background()
	return c.ReadHistory(ctx, chatID, msgID)
}

// BoundAnswerCallback sends an answer to a callback query triggered by an
// inline button press. This is a bound-method convenience wrapper around
// [Client.AnswerCallbackQuery].
//
// Parameters:
//   - queryID: unique identifier of the callback query
//   - text: notification text shown to the user
//   - showAlert: if true, the text is shown as a popup alert instead of a toast
//   - url: optional deep-link URL to open when the answer is received
//   - cacheTime: maximum seconds the client may cache the answer
//
// Returns an error if the answer fails to send.
func (c *Client) BoundAnswerCallback(queryID int64, opts ...*params.AnswerCallback) error {
	ctx := context.Background()
	o := params.GetOptDef(&params.AnswerCallback{}, opts...)
	return c.AnswerCallbackQuery(ctx, queryID, o.Text, o.ShowAlert, o.URL, int(o.CacheTime))
}

// BoundDownload downloads the media attached to a specific message. It first
// retrieves the message by ID, then downloads its media content. This is a
// bound-method convenience wrapper around [Client.GetMessages] and
// [Client.DownloadMedia].
//
// Parameters:
//   - chatID: chat containing the message with media
//   - msgID: ID of the message whose media to download
//   - opts: optional [params.Download] overrides
//
// Returns the downloaded media bytes, nil if the message has no media, or an
// error if retrieval or download fails.
func (c *Client) BoundDownload(chatID int64, msgID int32, opts ...*params.Download) ([]byte, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.Download{}, opts...)
	msgs, err := c.GetMessages(ctx, chatID, []int32{msgID})
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 || msgs[0].Media == nil {
		return nil, nil
	}
	return c.DownloadMedia(ctx, msgs[0].Media, "", opt)
}

func (c *Client) BoundDownloadTo(chatID int64, msgID int32, fileName string, opts ...*params.Download) (string, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.Download{FileName: fileName}, opts...)
	if opt.FileName == "" {
		opt.FileName = fileName
	}
	msgs, err := c.GetMessages(ctx, chatID, []int32{msgID})
	if err != nil {
		return "", err
	}
	if len(msgs) == 0 || msgs[0].Media == nil {
		return "", ErrNoDownloadableMedia
	}
	return c.downloadToPath(ctx, msgs[0].Media, opt.FileName, opt.Progress)
}

// BoundSendContact sends a contact card to the specified chat. This is a
// bound-method convenience wrapper around [Client.SendContact].
//
// Parameters:
//   - chatID: target chat identifier
//   - phone: contact phone number
//   - firstName: contact first name
//   - lastName: contact last name
//   - replyTo: message ID to reply to (0 for no reply)
//   - opts: optional [params.SendMessage] overrides
//
// Returns the sent message or an error if the send fails.
func (c *Client) BoundSendContact(chatID int64, phone, firstName, lastName string, replyTo int32, opts ...*params.SendContact) (*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.SendContact{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendContact(ctx, chatID, phone, firstName, lastName, opt.ToSendMsg())
}

// BoundSendLocation sends a geographic location to the specified chat. This is
// a bound-method convenience wrapper around [Client.SendLocation].
//
// Parameters:
//   - chatID: target chat identifier
//   - lat: latitude of the location
//   - lng: longitude of the location
//   - replyTo: message ID to reply to (0 for no reply)
//   - opts: optional [params.SendMessage] overrides
//
// Returns the sent message or an error if the send fails.
func (c *Client) BoundSendLocation(chatID int64, lat, lng float64, replyTo int32, opts ...*params.SendLocation) (*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.SendLocation{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendLocation(ctx, chatID, lat, lng, opt.ToSendMsg())
}

// BoundSendVenue sends a venue (location with title and address) to the
// specified chat. This is a bound-method convenience wrapper around
// [Client.SendVenue].
//
// Parameters:
//   - chatID: target chat identifier
//   - lat: latitude of the venue
//   - lng: longitude of the venue
//   - title: name of the venue
//   - address: physical address of the venue
//   - replyTo: message ID to reply to (0 for no reply)
//   - opts: optional [params.SendMessage] overrides
//
// Returns the sent message or an error if the send fails.
func (c *Client) BoundSendVenue(chatID int64, lat, lng float64, title, address string, replyTo int32, opts ...*params.SendVenue) (*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.SendVenue{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendVenue(ctx, chatID, lat, lng, title, address, opt.ToSendMsg())
}

// BoundSendPoll sends a poll question to the specified chat. This is a
// bound-method convenience wrapper around [Client.SendPoll].
//
// Parameters:
//   - chatID: target chat identifier
//   - question: the poll question text
//   - options: slice of answer option strings
//   - replyTo: message ID to reply to (0 for no reply)
//   - opts: optional [params.SendMessage] overrides
//
// Returns the sent message containing the poll or an error if the send fails.
func (c *Client) BoundSendPoll(chatID int64, question string, options []string, replyTo int32, opts ...*params.SendPoll) (*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.SendPoll{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendPoll(ctx, chatID, question, options, opt.ToSendMsg())
}

// BoundSendDice sends a dice message (animated emoji with a random value) to
// the specified chat. This is a bound-method convenience wrapper around
// [Client.SendDice].
//
// Parameters:
//   - chatID: target chat identifier
//   - emoji: dice emoji to send (e.g. "🎲", "🎯", "🏀", "⚽", "🎰", " bowling")
//   - replyTo: message ID to reply to (0 for no reply)
//   - opts: optional [params.SendMessage] overrides (currently unused)
//
// Returns the sent message or an error if the send fails.
func (c *Client) BoundSendDice(chatID int64, emoji string, replyTo int32, opts ...*params.SendDice) (*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.SendDice{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	emoticon := "\U0001F3B2"
	if emoji != "" {
		emoticon = emoji
	}
	if opt.Emoticon != "" {
		emoticon = opt.Emoticon
	}
	media := &tg.InputMediaDice{Emoticon: emoticon}
	return c.SendMedia(ctx, chatID, media, "", opt.ToSendMsg())
}

// BoundSendGame sends a game message to the specified chat. Only bots can send
// games. This is a bound-method convenience wrapper around [Client.SendGame].
//
// Parameters:
//   - chatID: target chat identifier (must be a private chat)
//   - gameShortName: short name of the game to send
//   - replyTo: message ID to reply to (0 for no reply)
//   - opts: optional [params.SendMessage] overrides
//
// Returns the sent message or an error if the send fails.
func (c *Client) BoundSendGame(chatID int64, gameShortName string, replyTo int32, opts ...*params.SendGame) (*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.SendGame{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendGame(ctx, chatID, gameShortName, opt.ToSendMsg())
}

// BoundSendMediaGroup sends a group of media items as a single album message.
// All items are grouped together and displayed as an album in the chat. This is
// a bound-method convenience wrapper around [Client.SendMediaGroup].
//
// Parameters:
//   - chatID: target chat identifier
//   - media: slice of input media objects to send as a group
//   - replyTo: message ID to reply to (0 for no reply)
//   - opts: optional [params.SendMessage] overrides
//
// Returns the slice of sent messages or an error if the send fails.
func (c *Client) BoundSendMediaGroup(chatID int64, media []tg.InputMediaClass, replyTo int32, opts ...*params.SendMediaGroup) ([]*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.SendMediaGroup{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	items := make([]*tg.InputSingleMedia, len(media))
	for i, m := range media {
		items[i] = &tg.InputSingleMedia{
			Media:    m,
			RandomID: c.RandomID(),
		}
	}
	return c.SendMediaGroup(ctx, chatID, items, opt.ToSendMsg())
}

// BoundSendChatAction sends a chat action indicator (e.g. "typing",
// "uploading photo") to the specified chat. This is a bound-method convenience
// wrapper around [Client.SendChatAction].
//
// Parameters:
//   - chatID: target chat identifier
//   - action: the chat action to display (e.g. [*tg.SendMessageTypingAction])
//
// Returns an error if the action fails to send.
func (c *Client) BoundSendChatAction(chatID int64, action tg.SendMessageActionClass) error {
	ctx := context.Background()
	return c.SendChatAction(ctx, chatID, action)
}

// BoundSendInlineBotResult sends an inline bot result to a chat. Use this after
// calling [Client.GetInlineBotResults] to obtain queryID and resultID. This is
// a bound-method convenience wrapper around [Client.SendInlineBotResult].
//
// Parameters:
//   - chatID: target chat identifier
//   - queryID: unique identifier of the inline query
//   - resultID: unique identifier of the inline result to send
//   - replyTo: message ID to reply to (0 for no reply)
//   - opts: optional [params.SendMessage] overrides
//
// Returns the sent message or an error if the send fails.
func (c *Client) BoundSendInlineBotResult(chatID int64, queryID int64, resultID string, replyTo int32, opts ...*params.SendInlineBotResult) (*types.Message, error) {
	ctx := context.Background()
	opt := params.GetOptDef(&params.SendInlineBotResult{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	inlineOpt := &SendInlineBotResultOption{
		ReplyTo:      int64(opt.ReplyToMessageID),
		Silent:       opt.DisableNotification || opt.Silent,
		HideVia:      opt.HideVia,
		ScheduleDate: opt.ScheduleDate,
		ClearDraft:   opt.ClearDraft,
	}
	return c.SendInlineBotResult(ctx, chatID, queryID, resultID, inlineOpt)
}

// BoundVote casts a vote in a poll. This is a bound-method convenience wrapper
// around [Client.VotePoll].
//
// Parameters:
//   - chatID: chat containing the poll
//   - msgID: ID of the poll message
//   - options: slice of option byte arrays to vote for
//
// Returns an error if the vote fails.
func (c *Client) BoundVote(chatID int64, msgID int32, options [][]byte) error {
	ctx := context.Background()
	return c.VotePoll(ctx, chatID, msgID, options)
}

// BoundRetractVote retracts the user's vote in a poll. This is a bound-method
// convenience wrapper around [Client.RetractVote].
//
// Parameters:
//   - chatID: chat containing the poll
//   - msgID: ID of the poll message
//
// Returns an error if the retraction fails.
func (c *Client) BoundRetractVote(chatID int64, msgID int32) error {
	ctx := context.Background()
	return c.RetractVote(ctx, chatID, msgID)
}

// BoundGetMediaGroup retrieves all messages belonging to the same media group
// (album) as the specified message. This is a bound-method convenience wrapper
// around [Client.GetMediaGroup].
//
// Parameters:
//   - chatID: chat containing the grouped media
//   - msgID: ID of any message in the media group
//
// Returns the slice of messages in the group or an error if retrieval fails.
func (c *Client) BoundGetMediaGroup(chatID int64, msgID int32) ([]*types.Message, error) {
	ctx := context.Background()
	return c.GetMediaGroup(ctx, chatID, msgID)
}

// BoundCopyMediaGroup copies an entire media group (album) from one chat to
// another. It first retrieves the source message to find the group ID, then
// copies all grouped messages. This is a bound-method convenience wrapper
// around [Client.GetMessages] and [Client.CopyMediaGroup].
//
// Parameters:
//   - chatID: destination chat identifier
//   - fromChatID: source chat identifier
//   - msgID: ID of any message in the source media group
//
// Returns the copied messages, nil if no messages found, or an error if the
// operation fails.
func (c *Client) BoundCopyMediaGroup(chatID int64, fromChatID int64, msgID int32) ([]*types.Message, error) {
	ctx := context.Background()
	msgs, err := c.GetMessages(ctx, fromChatID, []int32{msgID})
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, nil
	}
	return c.CopyMediaGroup(ctx, chatID, fromChatID, msgs[0].GroupedID)
}

// BoundStub returns an error indicating that the given method is not
// implemented. Use this as a placeholder for bound methods that have not yet
// been implemented.
//
// Parameters:
//   - method: name of the unimplemented method
//
// Returns an error with the message "method <method> not implemented".
func (c *Client) BoundStub(method string) error {
	return fmt.Errorf("method %s not implemented", method)
}

// BoundAnswerInline sends inline query results back to the user. This is a
// bound-method convenience wrapper around [Client.SetInlineBotResults].
//
// Parameters:
//   - queryID: unique identifier of the inline query to answer
//   - results: slice of inline result objects to display
//   - cacheTime: seconds the client may cache the results
//   - gallery: enable gallery-style layout for the results
//   - private: mark results as private to the querying user
//   - nextOffset: pagination offset for fetching more results
//   - switchPM: start parameter for a switch-to-PM button
//   - switchPMText: label shown on the switch-to-PM button
//
// Returns an error if the answer fails to send.
func (c *Client) BoundAnswerInline(queryID int64, results []tg.InputBotInlineResultClass, opts ...*params.InlineQuery) error {
	ctx := context.Background()
	o := params.GetOptDef(&params.InlineQuery{}, opts...)
	req := &tg.MessagesSetInlineBotResultsRequest{
		QueryID:   queryID,
		Results:   results,
		CacheTime: int32(o.CacheTime),
		Gallery:   o.Gallery,
		Private:   o.Private,
	}
	if o.NextOffset != "" {
		req.NextOffset = o.NextOffset
	}
	if o.SwitchPM != "" {
		req.SwitchPm = &tg.InlineBotSwitchPm{
			Text:       o.SwitchPMText,
			StartParam: o.SwitchPM,
		}
	}
	_, err := c.Raw().MessagesSetInlineBotResults(ctx, req)
	return err
}

// BoundBlock blocks a user. This is a bound-method convenience wrapper around
// [Client.BlockUser].
//
// Parameters:
//   - userID: ID of the user to block
//
// Returns an error if the block operation fails.
func (c *Client) BoundBlock(userID int64) error {
	return c.BlockUser(context.Background(), userID)
}

// BoundUnblock removes a user from the block list. This is a bound-method
// convenience wrapper around [Client.UnblockUser].
//
// Parameters:
//   - userID: ID of the user to unblock
//
// Returns an error if the unblock operation fails.
func (c *Client) BoundUnblock(userID int64) error {
	return c.UnblockUser(context.Background(), userID)
}

// BoundGetCommonChats returns the list of common chats shared with a user. This
// is a bound-method convenience wrapper around
// [messages.getCommonChats].
//
// Parameters:
//   - userID: ID of the target user
//   - limit: maximum number of chats to return
//
// Returns a slice of Chat objects or an error if the request fails.
func (c *Client) BoundGetCommonChats(userID int64, limit int) ([]*types.Chat, error) {
	ctx := context.Background()
	peer, err := resolveUserID(c, userID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	result, err := c.Raw().MessagesGetCommonChats(ctx, &tg.MessagesGetCommonChatsRequest{
		UserID: peer,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	return parseCommonChats(result), nil
}

func parseCommonChats(result tg.ChatsClass) []*types.Chat {
	var raw []tg.ChatClass
	switch v := result.(type) {
	case *tg.MessagesChats:
		raw = v.Chats
	case *tg.MessagesChatsSlice:
		raw = v.Chats
	}
	chats := make([]*types.Chat, 0, len(raw))
	for _, ch := range raw {
		if parsed := types.ParseChatFromChat(ch); parsed != nil {
			chats = append(chats, parsed)
		}
	}
	return chats
}

// BoundArchiveUser archives a user chat (moves it to the archived folder). This
// is a bound-method convenience wrapper that delegates to BoundArchive.
//
// Parameters:
//   - chatID: ID of the user chat to archive
//
// Returns an error if the archive operation fails.
func (c *Client) BoundArchiveUser(chatID int64) error {
	return c.BoundArchive(chatID)
}

// BoundUnarchiveUser unarchives a user chat (moves it back to the main list).
// This is a bound-method convenience wrapper that delegates to BoundUnarchive.
//
// Parameters:
//   - chatID: ID of the user chat to unarchive
//
// Returns an error if the unarchive operation fails.
func (c *Client) BoundUnarchiveUser(chatID int64) error {
	return c.BoundUnarchive(chatID)
}

func (c *Client) BoundAnswerPreCheckout(queryID int64, opts ...*params.AnswerPreCheckout) error {
	o := params.GetOptDef(&params.AnswerPreCheckout{}, opts...)
	return c.AnswerPreCheckoutQuery(context.Background(), queryID, o.Ok, o.ErrorMsg)
}

func (c *Client) BoundAnswerShipping(queryID int64, opts ...*params.AnswerShipping) error {
	o := params.GetOptDef(&params.AnswerShipping{}, opts...)
	var shippingOpts []*tg.ShippingOption
	if o.ShippingOptions != nil {
		shippingOpts = o.ShippingOptions.([]*tg.ShippingOption)
	}
	return c.AnswerShippingQuery(context.Background(), queryID, o.Ok, shippingOpts)
}

func (c *Client) BoundApproveJoinRequest(chatID int64, userID int64) error {
	return c.ApproveChatJoinRequest(context.Background(), chatID, userID)
}

func (c *Client) BoundDeclineJoinRequest(chatID int64, userID int64) error {
	return c.DeclineChatJoinRequest(context.Background(), chatID, userID)
}

func (c *Client) BoundStoryReply(peerID int64, storyID int32, text string, opts ...*params.SendMessage) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendMessage{}, opts...)
	opt.ReplyTo = &tg.InputReplyToStory{
		Peer:    &tg.InputPeerUser{UserID: peerID},
		StoryID: storyID,
	}
	return c.SendMessage(context.Background(), peerID, text, opt)
}

func (c *Client) BoundStoryReplyMedia(peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendMessage{}, opts...)
	opt.ReplyTo = &tg.InputReplyToStory{
		Peer:    &tg.InputPeerUser{UserID: peerID},
		StoryID: storyID,
	}
	return c.SendMedia(context.Background(), peerID, media, caption, opt)
}

func (c *Client) BoundStoryForward(fromChatID int64, storyID int32, chatID int64, opts ...*params.StoryForward) (*types.Message, error) {
	ctx := context.Background()
	return c.ForwardStory(ctx, chatID, fromChatID, storyID)
}

func (c *Client) BoundStoryRead(peerID int64, storyID int32) error {
	ctx := context.Background()
	peer, err := c.ResolvePeer(ctx, peerID)
	if err != nil {
		return err
	}
	_, err = c.Raw().StoriesReadStories(ctx, &tg.StoriesReadStoriesRequest{
		Peer:  peer,
		MaxID: storyID,
	})
	return err
}

func (c *Client) BoundStoryDelete(peerID int64, storyID int32) error {
	return c.DeleteStories(context.Background(), peerID, []int32{storyID})
}

func (c *Client) BoundStoryEditCaption(peerID int64, storyID int32, opts ...*params.EditCaption) (*types.Story, error) {
	o := params.GetOptDef(&params.EditCaption{}, opts...)
	return c.EditStoryCaption(context.Background(), peerID, storyID, o.Caption)
}

func (c *Client) BoundStoryEditMedia(peerID int64, storyID int32, media tg.InputMediaClass) (*types.Story, error) {
	return c.EditStoryMedia(context.Background(), peerID, storyID, media)
}

func (c *Client) BoundStoryEditPrivacy(peerID int64, storyID int32, opts ...*params.EditPrivacy) (*types.Story, error) {
	ctx := context.Background()
	peer, err := c.ResolvePeer(ctx, peerID)
	if err != nil {
		return nil, err
	}
	result, err := c.Raw().StoriesEditStory(ctx, &tg.StoriesEditStoryRequest{
		Peer: peer,
		ID:   storyID,
	})
	if err != nil {
		return nil, err
	}
	return extractStoryFromUpdates(result)
}

func (c *Client) BoundStoryReact(peerID int64, storyID int32, opts ...*params.React) error {
	ctx := context.Background()
	o := params.GetOptDef(&params.React{}, opts...)
	peer, err := c.ResolvePeer(ctx, peerID)
	if err != nil {
		return err
	}
	_, err = c.Raw().StoriesSendReaction(ctx, &tg.StoriesSendReactionRequest{
		Peer:    peer,
		StoryID: storyID,
		Reaction: &tg.ReactionEmoji{
			Emoticon: o.Emoji,
		},
	})
	return err
}

func (c *Client) BoundStoryDownload(peerID int64, storyID int32, opts ...*params.Download) ([]byte, error) {
	ctx := context.Background()
	stories, err := c.GetStories(ctx, peerID, []int32{storyID})
	if err != nil {
		return nil, err
	}
	if len(stories) == 0 || stories[0].Media == nil {
		return nil, nil
	}
	opt := params.GetOptDef(&params.Download{}, opts...)
	return c.DownloadMedia(ctx, stories[0].Media, "", opt)
}
