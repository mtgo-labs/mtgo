package telegram

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestKeyboardEmpty(t *testing.T) {
	kb := Keyboard()
	if kb.Build() != nil {
		t.Error("empty builder should return nil")
	}
	if kb.BuildReply() != nil {
		t.Error("empty builder should return nil for reply")
	}
}

func TestKeyboardCallback(t *testing.T) {
	markup := Keyboard().
		Callback("Click", "data123").
		Build()

	inner, ok := markup.(*tg.ReplyInlineMarkup)
	if !ok {
		t.Fatal("expected ReplyInlineMarkup")
	}
	if len(inner.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(inner.Rows))
	}
	if len(inner.Rows[0].Buttons) != 1 {
		t.Fatalf("expected 1 button, got %d", len(inner.Rows[0].Buttons))
	}
	btn, ok := inner.Rows[0].Buttons[0].(*tg.KeyboardButtonCallback)
	if !ok {
		t.Fatal("expected KeyboardButtonCallback")
	}
	if btn.Text != "Click" {
		t.Errorf("text = %q, want %q", btn.Text, "Click")
	}
	if string(btn.Data) != "data123" {
		t.Errorf("data = %q, want %q", btn.Data, "data123")
	}
}

func TestKeyboardCallbackTruncation(t *testing.T) {
	longData := make([]byte, 128)
	for i := range longData {
		longData[i] = 'x'
	}
	markup := Keyboard().
		Callback("Btn", string(longData)).
		Build()

	inner := markup.(*tg.ReplyInlineMarkup)
	btn := inner.Rows[0].Buttons[0].(*tg.KeyboardButtonCallback)
	if len(btn.Data) != 64 {
		t.Errorf("data length = %d, want 64", len(btn.Data))
	}
}

func TestKeyboardURL(t *testing.T) {
	markup := Keyboard().URL("Link", "https://example.com").Build()
	inner := markup.(*tg.ReplyInlineMarkup)
	btn := inner.Rows[0].Buttons[0].(*tg.KeyboardButtonURL)
	if btn.Text != "Link" || btn.URL != "https://example.com" {
		t.Errorf("got text=%q url=%q", btn.Text, btn.URL)
	}
}

func TestKeyboardTextReply(t *testing.T) {
	markup := Keyboard().
		Text("A").
		Text("B").
		BuildReply(ReplyOpts{Resize: true, OneTime: true})

	inner, ok := markup.(*tg.ReplyKeyboardMarkup)
	if !ok {
		t.Fatal("expected ReplyKeyboardMarkup")
	}
	if !inner.Resize || !inner.SingleUse {
		t.Error("expected Resize=true, SingleUse=true")
	}
	if len(inner.Rows) != 1 || len(inner.Rows[0].Buttons) != 2 {
		t.Fatalf("expected 1 row with 2 buttons, got %d rows", len(inner.Rows))
	}
}

func TestKeyboardNext(t *testing.T) {
	markup := Keyboard().
		Callback("R1A", "a").
		Callback("R1B", "b").
		Next().
		Callback("R2A", "c").
		Build()

	inner := markup.(*tg.ReplyInlineMarkup)
	if len(inner.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(inner.Rows))
	}
	if len(inner.Rows[0].Buttons) != 2 {
		t.Errorf("row 0: expected 2 buttons, got %d", len(inner.Rows[0].Buttons))
	}
	if len(inner.Rows[1].Buttons) != 1 {
		t.Errorf("row 1: expected 1 button, got %d", len(inner.Rows[1].Buttons))
	}
}

func TestKeyboardRow(t *testing.T) {
	markup := Keyboard().
		Row(
			&tg.KeyboardButtonCallback{Text: "A", Data: []byte("a")},
			&tg.KeyboardButtonCallback{Text: "B", Data: []byte("b")},
		).
		Row(
			&tg.KeyboardButtonCallback{Text: "C", Data: []byte("c")},
		).
		Build()

	inner := markup.(*tg.ReplyInlineMarkup)
	if len(inner.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(inner.Rows))
	}
}

func TestKeyboardReplyPlaceholder(t *testing.T) {
	markup := Keyboard().
		Text("OK").
		BuildReply(ReplyOpts{Placeholder: "Pick one"})

	inner := markup.(*tg.ReplyKeyboardMarkup)
	if inner.Placeholder != "Pick one" {
		t.Error("expected placeholder to be set")
	}
}

func TestKeyboardMultipleTypes(t *testing.T) {
	markup := Keyboard().
		Callback("Cb", "cb").
		URL("Link", "https://example.com").
		Copy("Copy", "text").
		Game("Play").
		Build()

	inner := markup.(*tg.ReplyInlineMarkup)
	if len(inner.Rows[0].Buttons) != 4 {
		t.Fatalf("expected 4 buttons, got %d", len(inner.Rows[0].Buttons))
	}
	if _, ok := inner.Rows[0].Buttons[0].(*tg.KeyboardButtonCallback); !ok {
		t.Error("button 0 should be Callback")
	}
	if _, ok := inner.Rows[0].Buttons[1].(*tg.KeyboardButtonURL); !ok {
		t.Error("button 1 should be URL")
	}
	if _, ok := inner.Rows[0].Buttons[2].(*tg.KeyboardButtonCopy); !ok {
		t.Error("button 2 should be Copy")
	}
	if _, ok := inner.Rows[0].Buttons[3].(*tg.KeyboardButtonGame); !ok {
		t.Error("button 3 should be Game")
	}
}

func TestKeyboardReplyButtons(t *testing.T) {
	markup := Keyboard().
		Text("A").
		RequestPhone("Phone").
		RequestGeo("Location").
		RequestPoll("Poll", false).
		BuildReply()

	inner := markup.(*tg.ReplyKeyboardMarkup)
	btns := inner.Rows[0].Buttons
	if _, ok := btns[0].(*tg.KeyboardButton); !ok {
		t.Error("button 0 should be Text")
	}
	if _, ok := btns[1].(*tg.KeyboardButtonRequestPhone); !ok {
		t.Error("button 1 should be RequestPhone")
	}
	if _, ok := btns[2].(*tg.KeyboardButtonRequestGeoLocation); !ok {
		t.Error("button 2 should be RequestGeo")
	}
	if _, ok := btns[3].(*tg.KeyboardButtonRequestPoll); !ok {
		t.Error("button 3 should be RequestPoll")
	}
}

func TestForceReplyMarkup(t *testing.T) {
	m := ForceReplyMarkup()
	if m == nil {
		t.Fatal("expected non-nil")
	}
}

func TestRemoveKeyboard(t *testing.T) {
	m := RemoveKeyboard()
	if m == nil {
		t.Fatal("expected non-nil")
	}
}

func TestKeyboardSwitch(t *testing.T) {
	markup := Keyboard().Switch("Share", false, "query").Build()
	inner := markup.(*tg.ReplyInlineMarkup)
	btn := inner.Rows[0].Buttons[0].(*tg.KeyboardButtonSwitchInline)
	if btn.Text != "Share" || btn.Query != "query" || btn.SamePeer {
		t.Errorf("unexpected switch button: %+v", btn)
	}
}

func TestKeyboardWebApp(t *testing.T) {
	markup := Keyboard().WebApp("Open", "https://app.com").Build()
	inner := markup.(*tg.ReplyInlineMarkup)
	btn := inner.Rows[0].Buttons[0].(*tg.KeyboardButtonWebView)
	if btn.Text != "Open" || btn.URL != "https://app.com" {
		t.Errorf("unexpected webapp button: %+v", btn)
	}
}

func TestKeyboardBuy(t *testing.T) {
	markup := Keyboard().Buy("Pay").Build()
	inner := markup.(*tg.ReplyInlineMarkup)
	if _, ok := inner.Rows[0].Buttons[0].(*tg.KeyboardButtonBuy); !ok {
		t.Error("expected Buy button")
	}
}

func TestKeyboardNextNoopOnEmpty(t *testing.T) {
	markup := Keyboard().Next().Next().Callback("A", "a").Build()
	inner := markup.(*tg.ReplyInlineMarkup)
	if len(inner.Rows) != 1 || len(inner.Rows[0].Buttons) != 1 {
		t.Error("Next() on empty row should be noop")
	}
}

func TestKeyboardRowEmpty(t *testing.T) {
	markup := Keyboard().Row().Build()
	if markup != nil {
		t.Error("empty Row() should produce nil Build()")
	}
}

func TestKeyboardRequestPeer(t *testing.T) {
	markup := Keyboard().
		RequestPeer("Channel", 1, &tg.RequestPeerTypeBroadcast{}, 1).
		RequestPeer("User", 2, &tg.RequestPeerTypeUser{}, 1).
		BuildReply(ReplyOpts{Resize: true})

	inner := markup.(*tg.ReplyKeyboardMarkup)
	btns := inner.Rows[0].Buttons
	if len(btns) != 2 {
		t.Fatalf("expected 2 buttons, got %d", len(btns))
	}
	ch, ok := btns[0].(*tg.InputKeyboardButtonRequestPeer)
	if !ok {
		t.Fatal("button 0 should be InputKeyboardButtonRequestPeer")
	}
	if ch.Text != "Channel" || ch.ButtonID != 1 {
		t.Errorf("got text=%q buttonID=%d", ch.Text, ch.ButtonID)
	}
	usr := btns[1].(*tg.InputKeyboardButtonRequestPeer)
	if usr.ButtonID != 2 {
		t.Errorf("got buttonID=%d", usr.ButtonID)
	}
}

func TestKeyboardRequestPeerEncodesInputButtonAndBoolFilter(t *testing.T) {
	markup := Keyboard().
		RequestPeer("Bot", 4, &tg.RequestPeerTypeUser{Bot: true}, 1).
		BuildReply()

	btn := markup.(*tg.ReplyKeyboardMarkup).Rows[0].Buttons[0]
	var buf bytes.Buffer
	if err := btn.Encode(&buf); err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	data := buf.Bytes()
	if got := binary.LittleEndian.Uint32(data[:4]); got != tg.InputKeyboardButtonRequestPeerTypeID {
		t.Fatalf("constructor = 0x%08x, want 0x%08x", got, tg.InputKeyboardButtonRequestPeerTypeID)
	}

	boolTrue := make([]byte, 4)
	binary.LittleEndian.PutUint32(boolTrue, tg.BoolTrueID)
	if !bytes.Contains(data, boolTrue) {
		t.Fatal("expected requestPeerTypeUser Bot filter to encode boolTrue")
	}
}

func TestKeyboardStyleOnNilStyle(t *testing.T) {
	markup := Keyboard().
		Callback("Yes", "yes").
		Success().
		Build()

	inner := markup.(*tg.ReplyInlineMarkup)
	btn := inner.Rows[0].Buttons[0].(*tg.KeyboardButtonCallback)
	if btn.Style == nil {
		t.Fatal("Style should be initialized after Success()")
	}
	if !btn.Style.BgSuccess {
		t.Error("BgSuccess should be true")
	}
}

func TestKeyboardRequestUser(t *testing.T) {
	markup := Keyboard().
		RequestUser("Pick User", 1, 5).
		BuildReply()

	btn := markup.(*tg.ReplyKeyboardMarkup).Rows[0].Buttons[0]
	peer := btn.(*tg.InputKeyboardButtonRequestPeer)
	pt, ok := peer.PeerType.(*tg.RequestPeerTypeUser)
	if !ok {
		t.Fatal("expected RequestPeerTypeUser")
	}
	if pt.Bot {
		t.Error("RequestUser should not set Bot=true by default")
	}
	if peer.MaxQuantity != 5 {
		t.Errorf("MaxQuantity = %d, want 5", peer.MaxQuantity)
	}
}

func TestKeyboardRequestUserBot(t *testing.T) {
	markup := Keyboard().
		RequestUser("Pick Bot", 2, 1, PeerUserOpts{Bot: true, Premium: true}).
		BuildReply()

	btn := markup.(*tg.ReplyKeyboardMarkup).Rows[0].Buttons[0]
	peer := btn.(*tg.InputKeyboardButtonRequestPeer)
	pt, ok := peer.PeerType.(*tg.RequestPeerTypeUser)
	if !ok {
		t.Fatal("expected RequestPeerTypeUser")
	}
	if !pt.Bot {
		t.Error("expected Bot=true")
	}
	if !pt.Premium {
		t.Error("expected Premium=true")
	}
}

func TestKeyboardRequestGroup(t *testing.T) {
	markup := Keyboard().
		RequestGroup("Pick Group", 3).
		BuildReply()

	btn := markup.(*tg.ReplyKeyboardMarkup).Rows[0].Buttons[0]
	peer := btn.(*tg.InputKeyboardButtonRequestPeer)
	if _, ok := peer.PeerType.(*tg.RequestPeerTypeChat); !ok {
		t.Fatal("expected RequestPeerTypeChat")
	}
}

func TestKeyboardRequestGroupWithOptions(t *testing.T) {
	markup := Keyboard().
		RequestGroup("Pick Forum", 3, PeerGroupOpts{Creator: true, Forum: true, HasUsername: true}).
		BuildReply()

	btn := markup.(*tg.ReplyKeyboardMarkup).Rows[0].Buttons[0]
	peer := btn.(*tg.InputKeyboardButtonRequestPeer)
	pt, ok := peer.PeerType.(*tg.RequestPeerTypeChat)
	if !ok {
		t.Fatal("expected RequestPeerTypeChat")
	}
	if !pt.Creator || !pt.Forum || !pt.HasUsername {
		t.Errorf("got creator=%v forum=%v hasUsername=%v", pt.Creator, pt.Forum, pt.HasUsername)
	}
}

func TestKeyboardRequestChannel(t *testing.T) {
	markup := Keyboard().
		RequestChannel("Pick Channel", 4).
		BuildReply()

	btn := markup.(*tg.ReplyKeyboardMarkup).Rows[0].Buttons[0]
	peer := btn.(*tg.InputKeyboardButtonRequestPeer)
	if _, ok := peer.PeerType.(*tg.RequestPeerTypeBroadcast); !ok {
		t.Fatal("expected RequestPeerTypeBroadcast")
	}
}

func TestKeyboardRequestChannelWithOptions(t *testing.T) {
	markup := Keyboard().
		RequestChannel("Pick Channel", 4, PeerChannelOpts{Creator: true}).
		BuildReply()

	btn := markup.(*tg.ReplyKeyboardMarkup).Rows[0].Buttons[0]
	peer := btn.(*tg.InputKeyboardButtonRequestPeer)
	pt, ok := peer.PeerType.(*tg.RequestPeerTypeBroadcast)
	if !ok {
		t.Fatal("expected RequestPeerTypeBroadcast")
	}
	if !pt.Creator {
		t.Error("expected Creator=true")
	}
}
