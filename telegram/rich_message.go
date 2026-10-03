package telegram

import (
	"context"
	"errors"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// Rich message support (Bot API 10.1+, MTProto layer 229).
//
// Rich messages carry formatted block content alongside (or instead of) plain
// text. The server parses markup for the HTML/Markdown input variants, so no
// client-side entity parsing is involved. Mappings mirror the official Bot API
// server: sendRichMessage → messages.sendMessage with rich_message (flag 23);
// sendRichMessageDraft → messages.setTyping with
// inputSendMessageRichMessageDraftAction.

// RichMessageHTML builds a rich message from Telegram HTML markup.
//
// Example:
//
//	rm := telegram.RichMessageHTML("<h1>Report</h1><p>See <b>details</b>.</p>", false)
//	msg, err := client.SendRichMessage(ctx, chatID, rm)
func RichMessageHTML(html string, rtl bool) *tg.InputRichMessageHTML {
	return &tg.InputRichMessageHTML{Rtl: rtl, HTML: html}
}

// RichMessageMarkdown builds a rich message from Telegram Markdown markup.
func RichMessageMarkdown(markdown string, rtl bool) *tg.InputRichMessageMarkdown {
	return &tg.InputRichMessageMarkdown{Rtl: rtl, Markdown: markdown}
}

// RichMessageBlocks builds a rich message from pre-built page blocks with the
// media/user inputs they reference. This is the Bot API 10.2 "blocks" variant.
func RichMessageBlocks(blocks []tg.PageBlockClass, photos []tg.InputPhotoClass, documents []tg.InputDocumentClass, users []tg.InputUserClass, rtl, noAutoLink bool) *tg.InputRichMessage {
	return &tg.InputRichMessage{
		Rtl:        rtl,
		Noautolink: noAutoLink,
		Blocks:     blocks,
		Photos:     photos,
		Documents:  documents,
		Users:      users,
	}
}

// SendRichMessage sends a rich message (Bot API sendRichMessage). The rich
// message replaces the plain text body; all standard SendMessage options
// (reply, markup, schedule, silent, …) still apply.
//
// Example:
//
//	msg, err := client.SendRichMessage(ctx, chatID,
//	    telegram.RichMessageMarkdown("# Title\n**bold** text", false),
//	    &params.SendMessage{ReplyToMessageID: 42},
//	)
func (c *Client) SendRichMessage(ctx context.Context, chatID int64, rm tg.InputRichMessageClass, opts ...*params.SendMessage) (*types.Message, error) {
	if rm == nil {
		return nil, errors.New("send rich message: rich message is nil")
	}
	opt := params.GetOptDef(&params.SendMessage{}, opts...)
	opt.RichMessage = rm
	return c.SendMessage(ctx, chatID, "", opt)
}

// SendRichMessageHTML sends a rich message built from Telegram HTML markup.
func (c *Client) SendRichMessageHTML(ctx context.Context, chatID int64, html string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendRichMessage(ctx, chatID, RichMessageHTML(html, false), opts...)
}

// SendRichMessageMarkdown sends a rich message built from Telegram Markdown
// markup.
func (c *Client) SendRichMessageMarkdown(ctx context.Context, chatID int64, markdown string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendRichMessage(ctx, chatID, RichMessageMarkdown(markdown, false), opts...)
}

// GetRichMessage fetches the rendered rich message content of an existing
// message via messages.getRichMessage. Partial rich messages (delivered with
// part=true in updates) are completed by this call.
func (c *Client) GetRichMessage(ctx context.Context, chatID int64, messageID int32) (*tg.RichMessage, error) {
	c.Log.Debugf("GetRichMessage chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := c.draftPeer(ctx, chatID)
	if err != nil {
		return nil, err
	}
	result, err := c.Raw().MessagesGetRichMessage(ctx, &tg.MessagesGetRichMessageRequest{
		Peer: peer,
		ID:   messageID,
	})
	if err != nil {
		return nil, fmt.Errorf("get rich message: %w", err)
	}
	var messages []tg.MessageClass
	switch v := result.(type) {
	case *tg.MessagesMessagesSlice:
		messages = v.Messages
	case *tg.MessagesMessages:
		messages = v.Messages
	}
	for _, m := range messages {
		if msg, ok := m.(*tg.Message); ok && msg.ID == messageID {
			if msg.RichMessage == nil {
				return nil, fmt.Errorf("get rich message: message %d has no rich content", messageID)
			}
			return msg.RichMessage, nil
		}
	}
	return nil, fmt.Errorf("get rich message: message %d not found in response", messageID)
}

// DraftOpts configures streaming drafts (Bot API sendMessageDraft /
// sendRichMessageDraft).
type DraftOpts struct {
	// ThreadID targets a forum topic or comment thread (top_msg_id).
	ThreadID int32
	// CanStop allows the user to stop the streaming draft (Bot API can_stop,
	// Bot API 10.3).
	CanStop bool
	// KeepOnStop keeps the partial content when the draft is stopped (Bot API
	// keep_on_stop).
	KeepOnStop bool
}

func getDraftOpts(opts ...*DraftOpts) *DraftOpts {
	if len(opts) == 0 || opts[0] == nil {
		return &DraftOpts{}
	}
	return opts[0]
}

// RichMessageDraft streams a rich message into a chat chunk by chunk (Bot API
// sendRichMessageDraft). Obtain one from [Client.StartRichMessageDraft], push
// partial content with Send, and finish with Stop (or let the user stop it
// when DraftOpts.CanStop is set). Every Send re-sends the draft built so far —
// the server renders the accumulated result.
type RichMessageDraft struct {
	c        *Client
	peer     tg.InputPeerClass
	randomID int64
	opts     *DraftOpts
}

// RandomID is the draft identifier shared by every Send/Stop call. It matches
// the Bot API draft_id and can be reused across reconnects.
func (d *RichMessageDraft) RandomID() int64 { return d.randomID }

// Send streams one partial rich message update.
func (d *RichMessageDraft) Send(ctx context.Context, rm tg.InputRichMessageClass) error {
	if d == nil || d.c == nil {
		return errors.New("rich message draft: not started")
	}
	if rm == nil {
		return errors.New("rich message draft: rich message is nil")
	}
	_, err := d.c.Raw().MessagesSetTyping(ctx, &tg.MessagesSetTypingRequest{
		Peer: d.peer,
		Action: &tg.InputSendMessageRichMessageDraftAction{
			CanStop:     d.opts.CanStop,
			KeepOnStop:  d.opts.KeepOnStop,
			RandomID:    d.randomID,
			RichMessage: rm,
		},
	})
	return err
}

// SendHTML streams one partial update built from Telegram HTML markup.
func (d *RichMessageDraft) SendHTML(ctx context.Context, html string) error {
	return d.Send(ctx, RichMessageHTML(html, false))
}

// SendMarkdown streams one partial update built from Telegram Markdown markup.
func (d *RichMessageDraft) SendMarkdown(ctx context.Context, markdown string) error {
	return d.Send(ctx, RichMessageMarkdown(markdown, false))
}

// Stop terminates the streaming draft, publishing the last streamed content.
func (d *RichMessageDraft) Stop(ctx context.Context) error {
	if d == nil || d.c == nil {
		return errors.New("rich message draft: not started")
	}
	_, err := d.c.Raw().MessagesSetTyping(ctx, &tg.MessagesSetTypingRequest{
		Peer:   d.peer,
		Action: &tg.SendMessageStopDraftAction{RandomID: d.randomID},
	})
	return err
}

// StartRichMessageDraft resolves the target chat and returns a streaming rich
// message draft bound to it.
func (c *Client) StartRichMessageDraft(ctx context.Context, chatID int64, opts ...*DraftOpts) (*RichMessageDraft, error) {
	c.Log.Debugf("StartRichMessageDraft chat_id=%d", chatID)
	peer, err := c.draftPeer(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return &RichMessageDraft{c: c, peer: peer, randomID: c.RandomID(), opts: getDraftOpts(opts...)}, nil
}

// MessageDraft streams plain formatted text into a chat chunk by chunk (Bot
// API sendMessageDraft). Text updates accumulate server-side the same way rich
// message drafts do.
type MessageDraft struct {
	c        *Client
	peer     tg.InputPeerClass
	randomID int64
	opts     *DraftOpts
}

// RandomID is the draft identifier shared by every Send/Stop call.
func (d *MessageDraft) RandomID() int64 { return d.randomID }

// Send streams one partial text update. Entities are optional; the text must
// already be plain (parse it beforehand when using markup).
func (d *MessageDraft) Send(ctx context.Context, text string, entities []tg.MessageEntityClass) error {
	if d == nil || d.c == nil {
		return errors.New("message draft: not started")
	}
	if len(entities) == 0 {
		entities = nil
	}
	_, err := d.c.Raw().MessagesSetTyping(ctx, &tg.MessagesSetTypingRequest{
		Peer: d.peer,
		Action: &tg.SendMessageTextDraftAction{
			CanStop:    d.opts.CanStop,
			KeepOnStop: d.opts.KeepOnStop,
			RandomID:   d.randomID,
			Text:       &tg.TextWithEntities{Text: text, Entities: entities},
		},
	})
	return err
}

// Stop terminates the streaming draft, publishing the last streamed content.
func (d *MessageDraft) Stop(ctx context.Context) error {
	if d == nil || d.c == nil {
		return errors.New("message draft: not started")
	}
	_, err := d.c.Raw().MessagesSetTyping(ctx, &tg.MessagesSetTypingRequest{
		Peer:   d.peer,
		Action: &tg.SendMessageStopDraftAction{RandomID: d.randomID},
	})
	return err
}

// StartMessageDraft resolves the target chat and returns a streaming plain
// text draft bound to it.
func (c *Client) StartMessageDraft(ctx context.Context, chatID int64, opts ...*DraftOpts) (*MessageDraft, error) {
	c.Log.Debugf("StartMessageDraft chat_id=%d", chatID)
	peer, err := c.draftPeer(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return &MessageDraft{c: c, peer: peer, randomID: c.RandomID(), opts: getDraftOpts(opts...)}, nil
}

func (c *Client) draftPeer(ctx context.Context, chatID int64) (tg.InputPeerClass, error) {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		peer, err = c.ResolvePeer(ctx, chatID)
		if err != nil {
			return nil, fmt.Errorf("resolve peer: %w", err)
		}
		return peer, nil
	}
	if c.IsBot() {
		peer, err = c.peersManager().EnsureUsable(ctx, peer)
		if err != nil {
			return nil, fmt.Errorf("resolve peer: %w", err)
		}
	}
	return peer, nil
}
