package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

// Issue #107 repro: messages returned by SendMessage/Reply must be bound.
func TestIssue107ReturnedMessagesAreBound(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesSendMessageTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdateNewMessage{
			Message: &tg.Message{ID: 7, PeerID: &tg.PeerUser{UserID: 10}},
		}},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})

	msg, err := c.SendMessage(t.Context(), 10, "hello")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	inv.setResult(tg.MessagesEditMessageTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdateEditMessage{
			Message: &tg.Message{ID: 7, PeerID: &tg.PeerUser{UserID: 10}, Message: "done"},
		}},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})
	if _, err := msg.EditText("done"); err != nil {
		t.Fatalf("EditText on returned message: %v (issue #107)", err)
	}
}
