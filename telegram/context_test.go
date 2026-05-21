package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestNewContext_SetsClient(t *testing.T) {
	c, _ := newClientWithBotRPCMock(t)

	ctx := c.NewContext(context.Background())
	if ctx.Client != c {
		t.Error("Client not set")
	}
	if ctx.Stopped {
		t.Error("Stopped should be false")
	}
}

func TestContext_AnswerInlineQuery_Delegates(t *testing.T) {
	c, _ := newClientWithBotRPCMock(t)

	ctx := c.NewContext(context.Background())
	ctx.InlineQuery = &types.InlineQuery{
		ID:     999,
		UserID: 42,
		Query:  "test",
	}

	if ctx.InlineQuery == nil {
		t.Fatal("InlineQuery should be set")
	}
	if ctx.InlineQuery.ID != 999 {
		t.Errorf("InlineQuery.ID = %d, want 999", ctx.InlineQuery.ID)
	}
}

func TestContext_AnswerCallbackQuery_Delegates(t *testing.T) {
	c, _ := newClientWithBotRPCMock(t)

	ctx := c.NewContext(context.Background())
	ctx.CallbackQuery = &types.CallbackQuery{
		ID:     777,
		UserID: 42,
	}

	if ctx.CallbackQuery == nil {
		t.Fatal("CallbackQuery should be set")
	}
	if ctx.CallbackQuery.ID != 777 {
		t.Errorf("CallbackQuery.ID = %d, want 777", ctx.CallbackQuery.ID)
	}
}

func TestContext_CtxField(t *testing.T) {
	bg := context.Background()
	ctx := (&Client{}).NewContext(bg)
	if ctx.Ctx != bg {
		t.Error("Ctx field should match the passed context")
	}

	derived, cancel := context.WithCancel(bg)
	defer cancel()
	ctx2 := (&Client{}).NewContext(derived)
	if ctx2.Ctx != derived {
		t.Error("Ctx field should match the derived context")
	}
}

func TestContext_CtxDefault(t *testing.T) {
	ctx := (&Client{}).NewContext(context.Background())
	if ctx.Ctx == nil {
		t.Error("Ctx should not be nil when context.Background() is passed")
	}
}

func TestNewContext_TODOCtx(t *testing.T) {
	base := context.TODO()
	ctx := (&Client{}).NewContext(base)
	if ctx.Ctx == nil {
		t.Error("Ctx should not be nil when context.TODO() is passed")
	}
	if ctx.Ctx != base {
		t.Error("Ctx field should match the passed context")
	}
}

func TestContext_Reply_NilMessage(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}}
	_, err := c.Reply("hello", nil)
	if err == nil {
		t.Fatal("expected error for nil message")
	}
}

func TestContext_Reply_MessageNilChat(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}, Message: &types.Message{}}
	_, err := c.Reply("hello", nil)
	if err == nil {
		t.Fatal("expected error for message with no chat")
	}
}

func TestContext_Edit_NilMessage(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}}
	_, err := c.Edit("new text", nil)
	if err == nil {
		t.Fatal("expected error for nil message")
	}
}

func TestContext_Delete_NilMessage(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}}
	_, err := c.Delete(nil)
	if err == nil {
		t.Fatal("expected error for nil message")
	}
}

func TestContext_Forward_NilMessage(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}}
	_, err := c.Forward(123, nil)
	if err == nil {
		t.Fatal("expected error for nil message")
	}
}

func TestContext_Copy_NilMessage(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}}
	_, err := c.Copy(123, nil)
	if err == nil {
		t.Fatal("expected error for nil message")
	}
}

func TestContext_Send_NilClient(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: nil}
	_, err := c.Send(123, "hello", nil)
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestContext_React_NilMessage(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}}
	err := c.React(types.Reaction{Emoji: "👍"})
	if err == nil {
		t.Fatal("expected error for nil message")
	}
}

func TestContext_Read_NilMessage(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}}
	err := c.Read()
	if err == nil {
		t.Fatal("expected error for nil message")
	}
}

func TestContext_SendMedia_NilClient(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: nil}
	_, err := c.SendMedia(123, &tg.InputMediaPhoto{ID: &tg.InputPhoto{ID: 1}}, "cap", nil)
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestContext_GetChat_NilClient(t *testing.T) {
	c := &Context{Ctx: context.Background()}
	_, err := c.GetChat()
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestContext_GetChat_NoChat(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}}
	_, err := c.GetChat()
	if err == nil {
		t.Fatal("expected error for no chat in context")
	}
}

func TestContext_AnswerCallback_NilCallback(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}}
	err := c.AnswerCallback("text", false)
	if err != nil {
		t.Fatalf("expected nil for nil callback query, got: %v", err)
	}
}

func TestContext_AnswerInline_NilQuery(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}}
	err := c.AnswerInline(nil, nil)
	if err != nil {
		t.Fatalf("expected nil for nil inline query, got: %v", err)
	}
}

func TestContext_Ban_NilMessage(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}}
	err := c.Ban(123)
	if err == nil {
		t.Fatal("expected error for no chat")
	}
}

func TestContext_Download_NilMessage(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}}
	_, err := c.DownloadMedia()
	if err == nil {
		t.Fatal("expected error for nil message")
	}
}

func TestContext_Download_NilMedia(t *testing.T) {
	c := &Context{Ctx: context.Background(), Client: &Client{}, Message: &types.Message{}}
	_, err := c.DownloadMedia()
	if err == nil {
		t.Fatal("expected error for message with no media")
	}
}

func TestContext_SendStory_NilClient(t *testing.T) {
	c := &Context{Ctx: context.Background()}
	_, err := c.SendStory(123, &tg.InputMediaPhoto{ID: &tg.InputPhoto{ID: 1}})
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestContext_GetStories_NilClient(t *testing.T) {
	c := &Context{Ctx: context.Background()}
	_, err := c.GetStories(123, []int32{1})
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestContext_GetBoostsStatus_NilClient(t *testing.T) {
	c := &Context{Ctx: context.Background()}
	_, err := c.GetBoostsStatus(123)
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestContext_GetPaymentForm_NilClient(t *testing.T) {
	c := &Context{Ctx: context.Background()}
	_, err := c.GetPaymentForm(123, 1, nil)
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestContext_GetBusinessConnection_NilClient(t *testing.T) {
	c := &Context{Ctx: context.Background()}
	_, err := c.GetBusinessConnection("conn_123")
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}
