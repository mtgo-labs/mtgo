package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestSendRichMessage_FlagAndField(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	rm := RichMessageMarkdown("**hi**", false)
	_, err := c.SendRichMessage(context.Background(), 10, rm,
		&params.SendMessage{NoForwards: true})
	if err != nil {
		t.Fatalf("SendRichMessage() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMessageRequest, got %T", inv.lastCall())
	}
	if !req.Flags.Has(23) {
		t.Fatalf("expected rich_message flag 23, flags=%032b", req.Flags)
	}
	if req.RichMessage != rm {
		t.Fatalf("req.RichMessage = %v, want the InputRichMessageMarkdown passed in", req.RichMessage)
	}
	if req.Message != "" {
		t.Fatalf("req.Message = %q, want empty plain text alongside the rich message", req.Message)
	}
}

func TestSendRichMessage_NilRichMessage(t *testing.T) {
	c, _ := newClientWithMock(t)
	if _, err := c.SendRichMessage(context.Background(), 10, nil); err == nil {
		t.Fatal("SendRichMessage(nil) should error")
	}
}

func TestRichMessageConstructors(t *testing.T) {
	html := RichMessageHTML("<b>x</b>", true)
	if html.HTML != "<b>x</b>" || !html.Rtl {
		t.Fatalf("RichMessageHTML = %+v", html)
	}
	md := RichMessageMarkdown("**x**", false)
	if md.Markdown != "**x**" || md.Rtl {
		t.Fatalf("RichMessageMarkdown = %+v", md)
	}
	blocks := RichMessageBlocks(
		[]tg.PageBlockClass{&tg.PageBlockParagraph{Text: &tg.TextPlain{Text: "p"}}},
		nil, nil, nil, false, true,
	)
	if len(blocks.Blocks) != 1 || !blocks.Noautolink {
		t.Fatalf("RichMessageBlocks = %+v", blocks)
	}
}

func TestSendRichMessageHTMLAndMarkdownConvenience(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	if _, err := c.SendRichMessageHTML(context.Background(), 10, "<i>x</i>"); err != nil {
		t.Fatalf("SendRichMessageHTML() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMessageRequest, got %T", inv.lastCall())
	}
	rm, ok := req.RichMessage.(*tg.InputRichMessageHTML)
	if !ok || rm.HTML != "<i>x</i>" {
		t.Fatalf("rich message = %#v, want InputRichMessageHTML", req.RichMessage)
	}

	if _, err := c.SendRichMessageMarkdown(context.Background(), 10, "*x*"); err != nil {
		t.Fatalf("SendRichMessageMarkdown() error: %v", err)
	}
	req, ok = inv.lastCall().(*tg.MessagesSendMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMessageRequest, got %T", inv.lastCall())
	}
	if _, ok := req.RichMessage.(*tg.InputRichMessageMarkdown); !ok {
		t.Fatalf("rich message = %T, want *tg.InputRichMessageMarkdown", req.RichMessage)
	}
}

func TestEditMessageText_RichMessage(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	rm := RichMessageHTML("<p>new</p>", false)
	_, err := c.EditMessageText(context.Background(), 10, 7, "", &params.EditMessage{RichMessage: rm})
	if err != nil {
		t.Fatalf("EditMessageText() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesEditMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesEditMessageRequest, got %T", inv.lastCall())
	}
	if !req.Flags.Has(23) || req.RichMessage != rm {
		t.Fatalf("rich_message flag/field not set, flags=%032b rm=%v", req.Flags, req.RichMessage)
	}
}

func TestRichMessageDraft_StreamAndStop(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	ctx := context.Background()

	d, err := c.StartRichMessageDraft(ctx, 10, &DraftOpts{CanStop: true, KeepOnStop: true})
	if err != nil {
		t.Fatalf("StartRichMessageDraft() error: %v", err)
	}
	if d.RandomID() == 0 {
		t.Fatal("draft random ID must be non-zero")
	}

	if err := d.Send(ctx, RichMessageHTML("<p>part</p>", false)); err != nil {
		t.Fatalf("draft.Send() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSetTypingRequest)
	if !ok {
		t.Fatalf("expected MessagesSetTypingRequest, got %T", inv.lastCall())
	}
	action, ok := req.Action.(*tg.InputSendMessageRichMessageDraftAction)
	if !ok {
		t.Fatalf("action = %T, want *tg.InputSendMessageRichMessageDraftAction", req.Action)
	}
	if action.RandomID != d.RandomID() {
		t.Fatalf("action.RandomID = %d, want draft ID %d", action.RandomID, d.RandomID())
	}
	if !action.CanStop || !action.KeepOnStop {
		t.Fatalf("action flags = can_stop:%v keep_on_stop:%v, want both true", action.CanStop, action.KeepOnStop)
	}
	if _, ok := action.RichMessage.(*tg.InputRichMessageHTML); !ok {
		t.Fatalf("action.RichMessage = %T, want *tg.InputRichMessageHTML", action.RichMessage)
	}

	if err := d.SendMarkdown(ctx, "**more**"); err != nil {
		t.Fatalf("draft.SendMarkdown() error: %v", err)
	}
	req, ok = inv.lastCall().(*tg.MessagesSetTypingRequest)
	if !ok {
		t.Fatalf("expected MessagesSetTypingRequest, got %T", inv.lastCall())
	}
	action, ok = req.Action.(*tg.InputSendMessageRichMessageDraftAction)
	if !ok || action.RandomID != d.RandomID() {
		t.Fatalf("second stream call lost the draft identity: %#v", req.Action)
	}

	if err := d.Stop(ctx); err != nil {
		t.Fatalf("draft.Stop() error: %v", err)
	}
	req, ok = inv.lastCall().(*tg.MessagesSetTypingRequest)
	if !ok {
		t.Fatalf("expected MessagesSetTypingRequest, got %T", inv.lastCall())
	}
	stop, ok := req.Action.(*tg.SendMessageStopDraftAction)
	if !ok {
		t.Fatalf("action = %T, want *tg.SendMessageStopDraftAction", req.Action)
	}
	if stop.RandomID != d.RandomID() {
		t.Fatalf("stop.RandomID = %d, want draft ID %d", stop.RandomID, d.RandomID())
	}
}

func TestRichMessageDraft_SendNilRichMessage(t *testing.T) {
	c, _ := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	d, err := c.StartRichMessageDraft(context.Background(), 10)
	if err != nil {
		t.Fatalf("StartRichMessageDraft() error: %v", err)
	}
	if err := d.Send(context.Background(), nil); err == nil {
		t.Fatal("draft.Send(nil) should error")
	}
}

func TestMessageDraft_SendAndStop(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	ctx := context.Background()

	d, err := c.StartMessageDraft(ctx, 10)
	if err != nil {
		t.Fatalf("StartMessageDraft() error: %v", err)
	}
	entities := []tg.MessageEntityClass{&tg.MessageEntityBold{Offset: 0, Length: 4}}
	if err := d.Send(ctx, "part", entities); err != nil {
		t.Fatalf("draft.Send() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSetTypingRequest)
	if !ok {
		t.Fatalf("expected MessagesSetTypingRequest, got %T", inv.lastCall())
	}
	action, ok := req.Action.(*tg.SendMessageTextDraftAction)
	if !ok {
		t.Fatalf("action = %T, want *tg.SendMessageTextDraftAction", req.Action)
	}
	if action.RandomID != d.RandomID() {
		t.Fatalf("action.RandomID = %d, want draft ID %d", action.RandomID, d.RandomID())
	}
	if action.Text == nil || action.Text.Text != "part" || len(action.Text.Entities) != 1 {
		t.Fatalf("action.Text = %#v", action.Text)
	}

	if err := d.Stop(ctx); err != nil {
		t.Fatalf("draft.Stop() error: %v", err)
	}
	req, ok = inv.lastCall().(*tg.MessagesSetTypingRequest)
	if !ok {
		t.Fatalf("expected MessagesSetTypingRequest, got %T", inv.lastCall())
	}
	if stop, ok := req.Action.(*tg.SendMessageStopDraftAction); !ok || stop.RandomID != d.RandomID() {
		t.Fatalf("stop action = %#v", req.Action)
	}
}

func TestBoundSendRich_ForwardsReplyTo(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	rm := RichMessageMarkdown("m", false)
	_, err := c.BoundSendRich(context.Background(), 10, rm, 42)
	if err != nil {
		t.Fatalf("BoundSendRich() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMessageRequest, got %T", inv.lastCall())
	}
	if req.ReplyTo == nil {
		t.Fatal("reply_to should be set from replyTo argument")
	}
	if !req.Flags.Has(23) || req.RichMessage != rm {
		t.Fatal("rich message not propagated through BoundSendRich")
	}
}
