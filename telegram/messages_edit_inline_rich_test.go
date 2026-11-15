package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestEditInlineTextParsesParseMode(t *testing.T) {
	client, mock := newClientWithMock(t)
	inlineID := &tg.InputBotInlineMessageID{DCID: 0, ID: 7, AccessHash: 99}

	ok, err := client.EditInlineText(context.Background(), inlineID, "<b>bold</b>",
		&EditInlineOpts{ParseMode: params.ParseModeHTML})
	if err != nil {
		t.Fatalf("EditInlineText() error: %v", err)
	}
	if !ok {
		t.Fatal("EditInlineText() = false, want true")
	}
	req, ok := mock.lastCall().(*tg.MessagesEditInlineBotMessageRequest)
	if !ok {
		t.Fatalf("main invoker call type = %T", mock.lastCall())
	}
	if req.Message != "bold" {
		t.Fatalf("req.Message = %q, want parsed plain text %q", req.Message, "bold")
	}
	if !req.Flags.Has(3) {
		t.Fatalf("entities flag 3 not set, flags=%032b", req.Flags)
	}
	if len(req.Entities) != 1 {
		t.Fatalf("req.Entities = %v, want one bold entity", req.Entities)
	}
	if _, isBold := req.Entities[0].(*tg.MessageEntityBold); !isBold {
		t.Fatalf("entity type = %T, want *tg.MessageEntityBold", req.Entities[0])
	}
}

func TestEditInlineCaptionParsesParseMode(t *testing.T) {
	client, mock := newClientWithMock(t)
	inlineID := &tg.InputBotInlineMessageID{DCID: 0, ID: 7, AccessHash: 99}

	ok, err := client.EditInlineCaption(context.Background(), inlineID, "*hi*",
		&EditInlineOpts{ParseMode: params.MarkdownV2})
	if err != nil {
		t.Fatalf("EditInlineCaption() error: %v", err)
	}
	if !ok {
		t.Fatal("EditInlineCaption() = false, want true")
	}
	req, ok := mock.lastCall().(*tg.MessagesEditInlineBotMessageRequest)
	if !ok {
		t.Fatalf("main invoker call type = %T", mock.lastCall())
	}
	if req.Message != "hi" {
		t.Fatalf("req.Message = %q, want %q", req.Message, "hi")
	}
	if !req.Flags.Has(3) || len(req.Entities) != 1 {
		t.Fatalf("parsed entities missing, flags=%032b entities=%v", req.Flags, req.Entities)
	}
}

func TestEditInlineRichMessage(t *testing.T) {
	client, mock := newClientWithMock(t)
	inlineID := &tg.InputBotInlineMessageID{DCID: 0, ID: 7, AccessHash: 99}

	rm := RichMessageMarkdown("**rich**", false)
	ok, err := client.EditInlineRichMessage(context.Background(), inlineID, rm,
		&EditInlineOpts{ReplyMarkup: &tg.ReplyInlineMarkup{}})
	if err != nil {
		t.Fatalf("EditInlineRichMessage() error: %v", err)
	}
	if !ok {
		t.Fatal("EditInlineRichMessage() = false, want true")
	}
	req, ok := mock.lastCall().(*tg.MessagesEditInlineBotMessageRequest)
	if !ok {
		t.Fatalf("main invoker call type = %T", mock.lastCall())
	}
	if !req.Flags.Has(23) || req.RichMessage != rm {
		t.Fatalf("rich_message flag/field not set, flags=%032b rm=%v", req.Flags, req.RichMessage)
	}
	if !req.Flags.Has(2) {
		t.Fatalf("reply_markup flag 2 not set, flags=%032b", req.Flags)
	}

	if _, err := client.EditInlineRichMessage(context.Background(), inlineID, nil); err == nil {
		t.Fatal("EditInlineRichMessage(nil) should error")
	}
}

func TestEditInlineTextWithRichMessageOption(t *testing.T) {
	client, mock := newClientWithMock(t)
	inlineID := &tg.InputBotInlineMessageID{DCID: 0, ID: 7, AccessHash: 99}

	rm := RichMessageHTML("<p>x</p>", false)
	ok, err := client.EditInlineText(context.Background(), inlineID, "", &EditInlineOpts{RichMessage: rm})
	if err != nil {
		t.Fatalf("EditInlineText() error: %v", err)
	}
	if !ok {
		t.Fatal("EditInlineText() = false, want true")
	}
	req, _ := mock.lastCall().(*tg.MessagesEditInlineBotMessageRequest)
	if !req.Flags.Has(23) || req.RichMessage != rm {
		t.Fatalf("rich_message flag/field not set via EditInlineOpts, flags=%032b", req.Flags)
	}
}

func TestBoundEditInlineForwardsOpts(t *testing.T) {
	client, mock := newClientWithMock(t)
	inlineID := &tg.InputBotInlineMessageID{DCID: 0, ID: 7, AccessHash: 99}

	ok, err := client.BoundEditInline(context.Background(), inlineID, "<i>it</i>", &params.EditMessage{
		ParseMode:             params.ParseModeHTML,
		DisableWebPagePreview: true,
	})
	if err != nil {
		t.Fatalf("BoundEditInline() error: %v", err)
	}
	if !ok {
		t.Fatal("BoundEditInline() = false, want true")
	}
	req, ok := mock.lastCall().(*tg.MessagesEditInlineBotMessageRequest)
	if !ok {
		t.Fatalf("main invoker call type = %T", mock.lastCall())
	}
	if req.Message != "it" {
		t.Fatalf("req.Message = %q, want parsed %q", req.Message, "it")
	}
	if !req.Flags.Has(1) {
		t.Fatalf("no_webpage flag 1 not forwarded, flags=%032b", req.Flags)
	}
	if !req.Flags.Has(3) || len(req.Entities) != 1 {
		t.Fatalf("parse-mode entities not forwarded, flags=%032b entities=%v", req.Flags, req.Entities)
	}

	// Rich message forwarding through the bound path.
	rm := RichMessageMarkdown("m", false)
	if _, err := client.BoundEditInline(context.Background(), inlineID, "", &params.EditMessage{RichMessage: rm}); err != nil {
		t.Fatalf("BoundEditInline(rich) error: %v", err)
	}
	req, ok = mock.lastCall().(*tg.MessagesEditInlineBotMessageRequest)
	if !ok || !req.Flags.Has(23) || req.RichMessage != rm {
		t.Fatalf("rich message not forwarded through BoundEditInline: flags=%032b", req.Flags)
	}
}

func TestBoundEditInlineCaptionForwardsOpts(t *testing.T) {
	client, mock := newClientWithMock(t)
	inlineID := &tg.InputBotInlineMessageID{DCID: 0, ID: 7, AccessHash: 99}

	ok, err := client.BoundEditInlineCaption(context.Background(), inlineID, "*c*", &params.EditMessage{
		ParseMode: params.MarkdownV2,
	})
	if err != nil {
		t.Fatalf("BoundEditInlineCaption() error: %v", err)
	}
	if !ok {
		t.Fatal("BoundEditInlineCaption() = false, want true")
	}
	req, ok := mock.lastCall().(*tg.MessagesEditInlineBotMessageRequest)
	if !ok {
		t.Fatalf("main invoker call type = %T", mock.lastCall())
	}
	if req.Message != "c" || !req.Flags.Has(3) {
		t.Fatalf("caption parse not forwarded, message=%q flags=%032b", req.Message, req.Flags)
	}
}
