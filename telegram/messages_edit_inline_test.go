package telegram

import (
	"context"
	"sync"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

type capturingDCInvoker struct {
	mu    sync.Mutex
	calls []tg.TLObject
}

func (c *capturingDCInvoker) RPCInvoke(_ context.Context, input tg.TLObject, _ func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, input)
	return &tg.BoolTrue{}, nil
}

func (c *capturingDCInvoker) RPCInvokeRaw(_ context.Context, _ tg.TLObject) ([]byte, error) {
	return nil, nil
}

func (c *capturingDCInvoker) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.calls)
}

func (c *capturingDCInvoker) lastCall() tg.TLObject {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.calls) == 0 {
		return nil
	}
	return c.calls[len(c.calls)-1]
}

// seedDCSession registers a fake authorized session entry for a foreign DC.
func seedDCSession(t *testing.T, client *Client, dcID int, invoker *capturingDCInvoker) {
	t.Helper()
	_, generation := client.dcSessions.getInitLock(dcID)
	entry := &dcSessionEntry{rpc: tg.NewRPCClient(invoker)}
	if !client.dcSessions.putIfGeneration(dcID, entry, generation) {
		t.Fatal("failed to seed DC session entry")
	}
}

// TestEditInlineRoutesToInlineMessageDC verifies that inline message edits are
// sent to the DC embedded in the inline message ID, not the home DC. Telegram
// rejects edits invoked on any other DC with MESSAGE_ID_INVALID.
func TestEditInlineRoutesToInlineMessageDC(t *testing.T) {
	ids := map[string]tg.InputBotInlineMessageIDClass{
		"inputBotInlineMessageID":   &tg.InputBotInlineMessageID{DCID: 4, ID: 7, AccessHash: 99},
		"inputBotInlineMessageID64": &tg.InputBotInlineMessageID64{DCID: 4, OwnerID: 7, ID: 7, AccessHash: 99},
	}
	for name, inlineID := range ids {
		t.Run(name, func(t *testing.T) {
			client, err := NewClient(1, "hash", nil)
			if err != nil {
				t.Fatal(err)
			}
			setDCPoolClientConnected(t, client, 2)

			dcInvoker := &capturingDCInvoker{}
			seedDCSession(t, client, 4, dcInvoker)

			ok, err := client.EditInlineText(context.Background(), inlineID, "updated")
			if err != nil {
				t.Fatalf("EditInlineText() error: %v", err)
			}
			if !ok {
				t.Fatal("EditInlineText() = false, want true")
			}
			if got := dcInvoker.callCount(); got != 1 {
				t.Fatalf("foreign DC invoker calls = %d, want 1", got)
			}
			req, ok := dcInvoker.lastCall().(*tg.MessagesEditInlineBotMessageRequest)
			if !ok {
				t.Fatalf("foreign DC call type = %T, want *tg.MessagesEditInlineBotMessageRequest", dcInvoker.lastCall())
			}
			if req.Message != "updated" {
				t.Fatalf("req.Message = %q, want %q", req.Message, "updated")
			}
			if req.ID != inlineID {
				t.Fatalf("req.ID = %v, want %v", req.ID, inlineID)
			}
		})
	}
}

// TestEditInlineMediaRoutesToInlineMessageDC covers the media variant.
func TestEditInlineMediaRoutesToInlineMessageDC(t *testing.T) {
	client, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatal(err)
	}
	setDCPoolClientConnected(t, client, 2)

	dcInvoker := &capturingDCInvoker{}
	seedDCSession(t, client, 4, dcInvoker)

	inlineID := &tg.InputBotInlineMessageID64{DCID: 4, OwnerID: 7, ID: 7, AccessHash: 99}
	media := &tg.InputMediaPhoto{ID: &tg.InputPhoto{ID: 1, AccessHash: 2}}
	ok, err := client.EditInlineMedia(context.Background(), inlineID, media)
	if err != nil {
		t.Fatalf("EditInlineMedia() error: %v", err)
	}
	if !ok {
		t.Fatal("EditInlineMedia() = false, want true")
	}
	req, ok := dcInvoker.lastCall().(*tg.MessagesEditInlineBotMessageRequest)
	if !ok {
		t.Fatalf("foreign DC call type = %T, want *tg.MessagesEditInlineBotMessageRequest", dcInvoker.lastCall())
	}
	if req.Media != tg.InputMediaClass(media) {
		t.Fatalf("req.Media = %v, want %v", req.Media, media)
	}
}

// TestEditInlineHomeDCUsesMainInvoker verifies that edits whose inline message
// ID points at the home DC (or carries no usable DC) stay on the main invoker.
func TestEditInlineHomeDCUsesMainInvoker(t *testing.T) {
	ids := map[string]tg.InputBotInlineMessageIDClass{
		"zeroDC": &tg.InputBotInlineMessageID{DCID: 0, ID: 7, AccessHash: 99},
		"homeDC": &tg.InputBotInlineMessageID64{DCID: 2, OwnerID: 7, ID: 7, AccessHash: 99},
	}
	for name, inlineID := range ids {
		t.Run(name, func(t *testing.T) {
			client, mock := newClientWithMock(t)

			ok, err := client.EditInlineCaption(context.Background(), inlineID, "caption")
			if err != nil {
				t.Fatalf("EditInlineCaption() error: %v", err)
			}
			if !ok {
				t.Fatal("EditInlineCaption() = false, want true")
			}
			if got := mock.callCount(); got != 1 {
				t.Fatalf("main invoker calls = %d, want 1", got)
			}
			req, ok := mock.lastCall().(*tg.MessagesEditInlineBotMessageRequest)
			if !ok {
				t.Fatalf("main invoker call type = %T, want *tg.MessagesEditInlineBotMessageRequest", mock.lastCall())
			}
			if req.Message != "caption" {
				t.Fatalf("req.Message = %q, want %q", req.Message, "caption")
			}
		})
	}
}

// TestInlineMessageDCExtraction covers the dc_id extraction for both inline
// message ID constructors.
func TestInlineMessageDCExtraction(t *testing.T) {
	dc, ok := inlineMessageDC(&tg.InputBotInlineMessageID{DCID: 5, ID: 1, AccessHash: 2})
	if !ok || dc != 5 {
		t.Fatalf("inlineMessageDC(InputBotInlineMessageID) = (%d, %v), want (5, true)", dc, ok)
	}
	dc, ok = inlineMessageDC(&tg.InputBotInlineMessageID64{DCID: 3, OwnerID: 1, ID: 1, AccessHash: 2})
	if !ok || dc != 3 {
		t.Fatalf("inlineMessageDC(InputBotInlineMessageID64) = (%d, %v), want (3, true)", dc, ok)
	}
}
