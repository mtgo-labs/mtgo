package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// Reply sends a text message in reply to the message that triggered the current context.
// The reply automatically references the original message ID. If the context has no
// message, an error is returned.
//
// Parameters:
//   - text: the reply text to send
//   - opts: optional [params.SendMessage] parameters for formatting, buttons, and other options
//
// Returns:
//   - *types.Message: the sent reply message
//   - error: non-nil if there is no message to reply to, the chat ID cannot be resolved, or the send fails
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Reply("Hello! I received your message.")
//	})
func (c *Context) Reply(text string, opts ...*params.SendMessage) (*types.Message, error) {
	if c.Message == nil {
		return nil, ErrContextNoReply
	}
	chatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	opt := params.GetOptDef(&params.SendMessage{}, opts...)
	opt.ReplyToMessageID = c.Message.ID

	// Pre-resolve the peer from the update's Users/Chats maps so the
	// access hash is available even on first interaction with a user.
	if opt.ReplyTo == nil && chatID > 0 {
		if _, cacheErr := resolvePeer(c.Client, chatID); cacheErr != nil {
			if u, ok := c.Update.Users[chatID]; ok && u.AccessHash != 0 {
				c.Client.CachePeer(chatID, &tg.InputPeerUser{UserID: chatID, AccessHash: u.AccessHash})
			}
		}
	}

	return c.Client.SendMessage(c.Ctx, chatID, text, opt)
}

// Edit modifies the text of the message that triggered the current context. Works for
// both regular messages and edited messages that the bot previously sent.
//
// Parameters:
//   - text: the new message text
//   - opts: optional [params.EditMessage] parameters for formatting and other options
//
// Returns:
//   - *types.Message: the edited message
//   - error: non-nil if there is no message to edit, the chat/message ID cannot be resolved, or the edit fails
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    msg, _ := ctx.Reply("Loading...")
//	    ctx.Edit("Done processing!")
//	})
func (c *Context) Edit(text string, opts ...*params.EditMessage) (*types.Message, error) {
	if c.Message == nil && c.EditedMessage == nil {
		return nil, ErrContextNoEdit
	}
	chatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	msgID, err := c.messageID()
	if err != nil {
		return nil, err
	}
	return c.Client.EditMessageText(c.Ctx, chatID, msgID, text, opts...)
}

// EditText is an alias for [Context.Edit] for explicit clarity when editing message text.
func (c *Context) EditText(text string, opts ...*params.EditMessage) (*types.Message, error) {
	return c.Edit(text, opts...)
}

// EditCaption changes the caption of the media message that triggered the current context.
//
// Parameters:
//   - caption: the new caption text
//   - opts: optional [params.EditMessage] parameters for formatting and other options
//
// Returns:
//   - *types.Message: the edited message
//   - error: non-nil if the chat/message ID cannot be resolved or the edit fails
func (c *Context) EditCaption(caption string, opts ...*params.EditMessage) (*types.Message, error) {
	chatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	msgID, err := c.messageID()
	if err != nil {
		return nil, err
	}
	return c.Client.EditMessageCaption(c.Ctx, chatID, msgID, caption, opts...)
}

// EditMedia replaces the media attachment of the message that triggered the current context.
// Use this to swap photos, videos, documents, or other media types.
//
// Parameters:
//   - media: the new media to upload as an [tg.InputMediaClass]
//   - opts: optional [params.EditMessage] parameters for additional configuration
//
// Returns:
//   - *types.Message: the edited message
//   - error: non-nil if the chat/message ID cannot be resolved or the edit fails
func (c *Context) EditMedia(media tg.InputMediaClass, opts ...*params.EditMessage) (*types.Message, error) {
	chatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	msgID, err := c.messageID()
	if err != nil {
		return nil, err
	}
	return c.Client.EditMessageMedia(c.Ctx, chatID, msgID, media, opts...)
}

// EditReplyMarkup replaces the inline keyboard of the message that triggered the current
// context. Use this to update button labels, remove buttons, or change the layout after
// user interaction.
//
// Parameters:
//   - replyMarkup: the new inline keyboard markup
//
// Returns:
//   - *types.Message: the edited message
//   - error: non-nil if the chat/message ID cannot be resolved or the edit fails
func (c *Context) EditReplyMarkup(replyMarkup tg.ReplyMarkupClass) (*types.Message, error) {
	chatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	msgID, err := c.messageID()
	if err != nil {
		return nil, err
	}
	return c.Client.EditMessageReplyMarkup(c.Ctx, chatID, msgID, replyMarkup)
}

// Delete removes the message that triggered the current context.
//
// Parameters:
//   - opts: optional [params.DeleteMessages] parameters (e.g. for revoke control)
//
// Returns:
//   - int: the number of messages deleted
//   - error: non-nil if the chat/message ID cannot be resolved or the deletion fails
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Delete()
//	})
func (c *Context) Delete(opts ...*params.DeleteMessages) (int, error) {
	msgID, err := c.messageID()
	if err != nil {
		return 0, err
	}
	chatID, err := c.chatID()
	if err != nil {
		return 0, err
	}
	return c.Client.DeleteMessages(c.Ctx, chatID, []int32{msgID}, opts...)
}

// Forward forwards the message that triggered the current context to the specified chat.
// The forwarded message retains the original sender attribution.
//
// Parameters:
//   - chatID: the target chat ID to forward the message to
//   - opts: optional [params.ForwardMessages] parameters (e.g. for silent forwarding)
//
// Returns:
//   - *types.Message: the forwarded message in the target chat
//   - error: non-nil if there is no message to forward, the chat ID cannot be resolved, or the forward fails
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Forward(logChatID)
//	})
func (c *Context) Forward(chatID int64, opts ...*params.ForwardMessages) (*types.Message, error) {
	if c.Message == nil {
		return nil, ErrContextNoForward
	}
	fromChatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	msgs, err := c.Client.ForwardMessages(c.Ctx, chatID, fromChatID, []int32{c.Message.ID}, opts...)
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, ErrContextNoForwardResult
	}
	return msgs[0], nil
}

// Copy sends a copy of the message that triggered the current context to the specified
// chat. Unlike [Context.Forward], the copy is sent as a new message without the original
// sender attribution.
//
// Parameters:
//   - chatID: the target chat ID to copy the message to
//   - opts: optional [params.CopyMessage] parameters for additional configuration
//
// Returns:
//   - int64: the ID of the sent message in the target chat
//   - error: non-nil if there is no message to copy, the chat ID cannot be resolved, or the copy fails
func (c *Context) Copy(chatID int64, opts ...*params.CopyMessage) (int64, error) {
	if c.Message == nil {
		return 0, ErrContextNoCopy
	}
	fromChatID, err := c.chatID()
	if err != nil {
		return 0, err
	}
	return c.Client.CopyMessage(c.Ctx, chatID, fromChatID, c.Message.ID, opts...)
}

// Send sends a text message to the specified chat. Unlike [Context.Reply], this does not
// reference any previous message.
//
// Parameters:
//   - chatID: the target chat ID
//   - text: the message text to send
//   - opts: optional [params.SendMessage] parameters for formatting, buttons, and other options
//
// Returns:
//   - *types.Message: the sent message
//   - error: non-nil if the context has no client or the send fails
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Send(adminChatID, "User sent a message")
//	})
func (c *Context) Send(chatID int64, text string, opts ...*params.SendMessage) (*types.Message, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.SendMessage(c.Ctx, chatID, text, opts...)
}

// SendMedia sends a media attachment (photo, video, document, etc.) with an optional
// caption to the specified chat.
//
// Parameters:
//   - chatID: the target chat ID
//   - media: the media to send as an [tg.InputMediaClass]
//   - caption: optional caption text for the media
//   - opts: optional [params.SendMessage] parameters for additional configuration
//
// Returns:
//   - *types.Message: the sent media message
//   - error: non-nil if the context has no client or the send fails
func (c *Context) SendMedia(chatID int64, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.SendMedia(c.Ctx, chatID, media, caption, opts...)
}

// React adds one or more emoji reactions to the message that triggered the current context.
//
// Parameters:
//   - reaction: one or more [tg.ReactionClass] values representing the reactions to add
//
// Returns:
//   - error: non-nil if there is no message, the chat ID cannot be resolved, or the reaction fails
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.React(types.Reaction{Emoji: "👍"})
//	})
func (c *Context) React(reactions ...types.Reaction) error {
	if c.Message == nil {
		return ErrContextNoReact
	}
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.SendReaction(c.Ctx, chatID, c.Message.ID, reactions)
}

// Read marks the message that triggered the current context as read. If no specific message
// ID can be resolved, it marks the entire chat as read from the beginning.
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved or the read operation fails
func (c *Context) Read() error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	msgID, err := c.messageID()
	if err != nil {
		return c.Client.ReadHistory(c.Ctx, chatID, 0)
	}
	return c.Client.ReadHistory(c.Ctx, chatID, msgID)
}

// DownloadMedia downloads the media attached to the current context message and returns
// it as a byte slice.
//
// Returns:
//   - []byte: the raw media content
//   - error: non-nil if there is no message, the message has no media, or the download fails
func (c *Context) DownloadMedia() ([]byte, error) {
	if c.Message == nil {
		return nil, ErrContextNoMessage
	}
	if c.Message.Media == nil {
		return nil, ErrContextNoMedia
	}
	return c.Client.DownloadMedia(c.Ctx, c.Message.Media, "", nil)
}

// DownloadMediaToFile downloads the media attached to the current context message and
// writes it directly to the specified file path on disk.
//
// Parameters:
//   - filePath: the destination file path on disk
//   - fileSize: expected file size in bytes for progress tracking; use 0 if unknown
//
// Returns:
//   - error: non-nil if there is no message, the message has no media, or the download fails
func (c *Context) DownloadMediaToFile(filePath string, fileSize int64) error {
	if c.Message == nil {
		return ErrContextNoMessage
	}
	if c.Message.Media == nil {
		return ErrContextNoMedia
	}
	return c.Client.DownloadMediaToFile(c.Ctx, c.Message.Media, "", filePath, fileSize, nil)
}

// Pin pins the message that triggered the current context to the top of the chat.
//
// Parameters:
//   - opts: optional [params.PinMessage] parameters (e.g. for silent pinning)
//
// Returns:
//   - error: non-nil if the chat/message ID cannot be resolved or the pin fails
func (c *Context) Pin(opts ...*params.PinMessage) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	msgID, err := c.messageID()
	if err != nil {
		return err
	}
	_, err = c.Client.PinMessage(c.Ctx, chatID, msgID, opts...)
	return err
}

// Unpin removes the message that triggered the current context from the pinned messages
// in the chat.
//
// Returns:
//   - error: non-nil if the chat/message ID cannot be resolved or the unpin fails
func (c *Context) Unpin() error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	msgID, err := c.messageID()
	if err != nil {
		return err
	}
	_, err = c.Client.UnpinMessage(c.Ctx, chatID, msgID)
	return err
}

// SendChatAction sends a typing or other activity indicator to the chat. Use this to
// show the user that the bot is performing an action (e.g. typing, uploading a photo).
//
// Parameters:
//   - action: the chat action to display (e.g. [tg.SendMessageTypingAction])
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved or the action send fails
func (c *Context) SendChatAction(action tg.SendMessageActionClass) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.SendChatAction(c.Ctx, chatID, action)
}

// GetMediaGroup retrieves all messages that are part of the same media group (album) as
// the current context message. Media groups are sets of photos or videos sent together.
//
// Returns:
//   - []*types.Message: all messages belonging to the same media group
//   - error: non-nil if there is no message, the message is not part of a media group, or the request fails
func (c *Context) GetMediaGroup() ([]*types.Message, error) {
	if c.Message == nil {
		return nil, ErrContextNoMessage
	}
	if c.Message.GroupedID == 0 {
		return nil, ErrContextNotMediaGroup
	}
	chatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	return c.Client.GetMediaGroup(c.Ctx, chatID, c.Message.ID)
}

// Vote casts a vote in the poll attached to the current context message.
//
// Parameters:
//   - options: slice of byte arrays identifying the poll options to vote for
//
// Returns:
//   - error: non-nil if there is no message, the chat ID cannot be resolved, or the vote fails
func (c *Context) Vote(options [][]byte) error {
	if c.Message == nil {
		return ErrContextNoMessage
	}
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.VotePoll(c.Ctx, chatID, c.Message.ID, options)
}

// StopPoll closes the poll attached to the current context message, preventing further
// votes.
//
// Returns:
//   - error: non-nil if there is no message, the chat ID cannot be resolved, or the stop fails
func (c *Context) StopPoll() error {
	if c.Message == nil {
		return ErrContextNoMessage
	}
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.StopPoll(c.Ctx, chatID, c.Message.ID)
}

// RetractVote withdraws the current user's vote from the poll attached to the context message.
//
// Returns:
//   - error: non-nil if there is no message, the chat ID cannot be resolved, or the retraction fails
func (c *Context) RetractVote() error {
	if c.Message == nil {
		return ErrContextNoMessage
	}
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.RetractVote(c.Ctx, chatID, c.Message.ID)
}

// SendPaidReaction sends a paid reaction (using Telegram Stars) to the message that
// triggered the current context.
//
// Parameters:
//   - amount: the number of Stars to spend on this reaction
//
// Returns:
//   - error: non-nil if there is no message, the chat ID cannot be resolved, or the reaction fails
func (c *Context) SendPaidReaction(amount int64) error {
	if c.Message == nil {
		return ErrContextNoMessage
	}
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.SendPaidReaction(c.Ctx, chatID, c.Message.ID, amount)
}

// View is an alias for [Context.Read]. It marks the message that triggered the current
// context as read.
func (c *Context) View() error {
	return c.Read()
}

// DeleteChatHistory deletes messages from the chat history up to and including the message
// that triggered the current context. If no message ID is available, all history is deleted.
//
// Parameters:
//   - revoke: if true, deletes the messages for both sides (when supported by the chat type)
//
// Returns:
//   - int: the number of messages deleted
//   - error: non-nil if the chat ID cannot be resolved or the deletion fails
func (c *Context) DeleteChatHistory(revoke bool) (int, error) {
	chatID, err := c.chatID()
	if err != nil {
		return 0, err
	}
	msgID, err := c.messageID()
	if err != nil {
		return c.Client.DeleteChatHistory(c.Ctx, chatID, 0, revoke)
	}
	return c.Client.DeleteChatHistory(c.Ctx, chatID, msgID, revoke)
}

// GetChatHistoryCount returns the total number of messages in the chat associated with
// the current context.
//
// Returns:
//   - int: the total message count
//   - error: non-nil if the chat ID cannot be resolved or the request fails
func (c *Context) GetChatHistoryCount() (int, error) {
	chatID, err := c.chatID()
	if err != nil {
		return 0, err
	}
	return c.Client.GetChatHistoryCount(c.Ctx, chatID)
}

// ForwardMediaGroup forwards the entire media group (album) that includes the current
// context message to the specified chat.
//
// Parameters:
//   - chatID: the target chat ID to forward the media group to
//   - opts: optional [params.ForwardMessages] parameters for additional configuration
//
// Returns:
//   - []*types.Message: the forwarded messages in the target chat
//   - error: non-nil if there is no message, the chat ID cannot be resolved, or the forward fails
func (c *Context) ForwardMediaGroup(chatID int64, opts ...*params.ForwardMessages) ([]*types.Message, error) {
	if c.Message == nil {
		return nil, ErrContextNoMessage
	}
	fromChatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	return c.Client.ForwardMediaGroup(c.Ctx, chatID, fromChatID, []int32{c.Message.ID}, opts...)
}

// SendGame sends a game message to the specified chat. Games are a special message type
// that launch an inline game when the user taps the Play button.
//
// Parameters:
//   - chatID: the target chat ID
//   - gameShortName: the short name of the game to send
//   - opts: optional [params.SendMessage] parameters for additional configuration
//
// Returns:
//   - *types.Message: the sent game message
//   - error: non-nil if the context has no client or the send fails
func (c *Context) SendGame(chatID int64, gameShortName string, opts ...*params.SendMessage) (*types.Message, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.SendGame(c.Ctx, chatID, gameShortName, opts...)
}

// ReadMentions marks all mentions in the chat as read, clearing the unread mention badge.
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved or the operation fails
func (c *Context) ReadMentions() error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.ReadMentions(c.Ctx, chatID)
}

// ReadReactions marks all reaction notifications in the chat as read.
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved or the operation fails
func (c *Context) ReadReactions() error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.ReadReactions(c.Ctx, chatID)
}
