package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestSendWidget(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	_, err := c.SendWidget(context.Background(), 10, 320, 200, "<b>Hi</b>",
		&params.SendWidget{AllowQueries: true, SharedWidget: true, PublicData: `{"v":1}`})
	if err != nil {
		t.Fatalf("SendWidget() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendMediaRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMediaRequest, got %T", inv.lastCall())
	}
	media, ok := req.Media.(*tg.InputMediaWidget)
	if !ok {
		t.Fatalf("expected InputMediaWidget, got %T", req.Media)
	}
	if media.Width != 320 || media.Height != 200 || media.HTML != "<b>Hi</b>" {
		t.Errorf("media = %#v, want 320x200 widget", media)
	}
	if !media.AllowQueries || !media.SharedWidget || media.PublicData != `{"v":1}` {
		t.Errorf("media flags = %#v, want queries+shared+public data", media)
	}
}

func TestEditMessageWidgetData(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesEditMessageWidgetDataTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateEditMessage{
				Message: &tg.Message{ID: 55, PeerID: &tg.PeerChannel{ChannelID: 10}},
			},
		},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})

	msg, err := c.EditMessageWidgetData(context.Background(), 10, 55, `{"v":2}`, "secret")
	if err != nil {
		t.Fatalf("EditMessageWidgetData() error: %v", err)
	}
	if msg == nil || msg.ID != 55 {
		t.Fatalf("msg = %#v, want updated message 55", msg)
	}
	req, ok := inv.lastCall().(*tg.MessagesEditMessageWidgetDataRequest)
	if !ok {
		t.Fatalf("expected MessagesEditMessageWidgetDataRequest, got %T", inv.lastCall())
	}
	if req.ID != 55 || req.PublicData != `{"v":2}` || req.ChatData != "secret" {
		t.Errorf("request = %#v, want message 55 with both data fields", req)
	}
}

func TestEditWidgetData_Shared(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesEditWidgetDataTypeID, &tg.BoolTrue{})

	if err := c.EditWidgetData(context.Background(), 77, `{"v":3}`, 300); err != nil {
		t.Fatalf("EditWidgetData() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesEditWidgetDataRequest)
	if !ok {
		t.Fatalf("expected MessagesEditWidgetDataRequest, got %T", inv.lastCall())
	}
	if req.WidgetID != 77 || req.PublicData != `{"v":3}` || req.CacheTime != 300 {
		t.Errorf("request = %#v, want widget 77 with cache time", req)
	}
}

func TestGetWidgetPublicData(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesGetWidgetPublicDataTypeID, &tg.MessagesWidgetPublicData{
		Hash: 7, Data: `{"v":1}`, Period: 60,
	})

	data, err := c.GetWidgetPublicData(context.Background(), 10, 55, 0)
	if err != nil {
		t.Fatalf("GetWidgetPublicData() error: %v", err)
	}
	if data.NotModified || data.Data != `{"v":1}` || data.Hash != 7 || data.Period != 60 {
		t.Fatalf("data = %#v, want full payload", data)
	}

	inv.setResult(tg.MessagesGetWidgetPublicDataTypeID, &tg.MessagesWidgetPublicDataNotModified{Period: 60})
	data, err = c.GetWidgetPublicData(context.Background(), 10, 55, 7)
	if err != nil {
		t.Fatalf("GetWidgetPublicData(not modified) error: %v", err)
	}
	if !data.NotModified || data.Period != 60 {
		t.Fatalf("data = %#v, want not-modified with period", data)
	}
}

func TestAnswerWidgetQuery(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesSetBotWidgetQueryAnswerTypeID, &tg.BoolTrue{})

	if err := c.AnswerWidgetQuery(context.Background(), 99, `{"rows":[1]}`); err != nil {
		t.Fatalf("AnswerWidgetQuery() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSetBotWidgetQueryAnswerRequest)
	if !ok {
		t.Fatalf("expected MessagesSetBotWidgetQueryAnswerRequest, got %T", inv.lastCall())
	}
	if req.QueryID != 99 || req.Result != `{"rows":[1]}` {
		t.Errorf("request = %#v", req)
	}
}

func TestParseWidgetQuery_AndBoundAnswer(t *testing.T) {
	b := &mockWidgetBinder{}
	q := types.ParseWidgetQuery(&tg.UpdateBotWidgetQuery{
		QueryID:   5,
		UserID:    10,
		Peer:      &tg.PeerChannel{ChannelID: 100},
		MsgID:     55,
		ChatType:  "private",
		Data:      []byte("ask"),
		WidgetID:  77,
		QueryType: &tg.WidgetQueryTypeData{},
	})
	if q == nil {
		t.Fatal("ParseWidgetQuery() = nil")
	}
	if q.ID != 5 || q.UserID != 10 || q.MessageID != 55 || q.WidgetID != 77 {
		t.Fatalf("query = %#v", q)
	}
	if q.ChatID != -1000000000100 {
		t.Errorf("chat ID = %d, want marked channel", q.ChatID)
	}

	q.SetBinder(b)
	if err := q.Answer("done"); err != nil {
		t.Fatalf("Answer() error: %v", err)
	}
	if b.queryID != 5 || b.result != "done" {
		t.Errorf("answered = %d/%q", b.queryID, b.result)
	}
}

type mockWidgetBinder struct {
	queryID int64
	result  string
}

func (m *mockWidgetBinder) AnswerWidgetQuery(ctx context.Context, queryID int64, result string) error {
	m.queryID, m.result = queryID, result
	return nil
}

func TestMessageEditWidgetData_Bound(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesEditMessageWidgetDataTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateEditMessage{
				Message: &tg.Message{ID: 42, PeerID: &tg.PeerChannel{ChannelID: 10}},
			},
		},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})
	m := &types.Message{ID: 42, ChatID: 10}
	m.SetBinder(c)

	updated, err := m.EditWidgetData(`{"v":9}`, "chat")
	if err != nil {
		t.Fatalf("EditWidgetData() error: %v", err)
	}
	if updated == nil || updated.ID != 42 {
		t.Fatalf("updated = %#v", updated)
	}
	req, ok := inv.lastCall().(*tg.MessagesEditMessageWidgetDataRequest)
	if !ok {
		t.Fatalf("expected MessagesEditMessageWidgetDataRequest, got %T", inv.lastCall())
	}
	if req.PublicData != `{"v":9}` || req.ChatData != "chat" {
		t.Errorf("request = %#v", req)
	}
}
