package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// seedEngine points the engine at a known state without going through the
// network.
func seedEngine(r *updateRecovery, pts, qts, date, seq int32) {
	r.mu.Lock()
	r.hasState = true
	r.state = storage.UpdateState{Pts: pts, Qts: qts, Date: date, Seq: seq}
	r.mu.Unlock()
}

func engineState(r *updateRecovery) storage.UpdateState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state
}

func TestUpdateRecoveryTracksPtsAndDetectsGaps(t *testing.T) {
	c, _ := newClientWithMock(t)
	r := c.updateRec
	seedEngine(r, 10, 0, 100, 1)

	// Contiguous advance applies silently.
	if r.trackPTS(12, 2) {
		t.Fatal("contiguous pts flagged as gap")
	}
	if got := engineState(r).Pts; got != 12 {
		t.Fatalf("pts = %d, want 12", got)
	}
	// Old/duplicate updates are ignored.
	if r.trackPTS(11, 1) {
		t.Fatal("old pts flagged as gap")
	}
	// A jump past the expected value is a gap.
	if !r.trackPTS(20, 2) {
		t.Fatal("pts jump not detected as gap")
	}

	// seq/date tracking: contiguous seq applies, a jump is a gap.
	if r.trackSeq(2, 101) {
		t.Fatal("contiguous seq flagged as gap")
	}
	if !r.trackSeq(9, 102) {
		t.Fatal("seq jump not detected as gap")
	}

	// qts advances one event at a time.
	if r.trackQTS(1) {
		t.Fatal("contiguous qts flagged as gap")
	}
	if !r.trackQTS(4) {
		t.Fatal("qts jump not detected as gap")
	}
}

func TestUpdateRecoveryTrackBatchDetectsGap(t *testing.T) {
	c, _ := newClientWithMock(t)
	r := c.updateRec
	seedEngine(r, 10, 0, 100, 1)

	// Simulate the receive hook seeing a batch whose first update jumps pts.
	r.trackBatch(&tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdateNewMessage{
			Message:  &tg.Message{ID: 5, PeerID: &tg.PeerUser{UserID: 1}},
			PTS:      15,
			PTSCount: 1,
		}},
		Seq:  2,
		Date: 101,
	})
	if got := engineState(r).Pts; got != 10 {
		t.Fatalf("pts advanced over a gap: %d", got)
	}

	// UpdatesTooLong always means "fetch the difference".
	if !r.trackUpdate(&tg.UpdatePTSChanged{}) {
		t.Fatal("updatePtsChanged not treated as resync signal")
	}
}

func TestUpdateRecoveryAppliesDifference(t *testing.T) {
	c, inv := newClientWithMock(t)
	r := c.updateRec
	seedEngine(r, 10, 0, 100, 1)

	page := &tg.UpdatesDifference{
		NewMessages: []tg.MessageClass{&tg.Message{
			ID:     7,
			PeerID: &tg.PeerUser{UserID: 55},
		}},
		OtherUpdates: []tg.UpdateClass{&tg.UpdateDeleteMessages{
			Messages: []int32{1, 2},
			PTS:      11,
		}},
		Users: []tg.UserClass{&tg.User{ID: 55, AccessHash: 9, Username: "recovered"}},
		State: &tg.UpdatesState{PTS: 12, Qts: 0, Date: 101, Seq: 2},
	}
	inv.setResult(tg.UpdatesGetDifferenceTypeID, page)
	inv.setResult(tg.UpdatesDifferenceEmptyTypeID, nil) // not used; safety

	// The second call (after the page) must report the state caught up: the
	// mock returns the same page, whose pts no longer moves → converge.
	r.pass()

	if got := engineState(r).Pts; got != 12 {
		t.Fatalf("pts = %d, want 12 after difference", got)
	}
	if got := engineState(r).Seq; got != 2 {
		t.Fatalf("seq = %d, want 2", got)
	}
	// The recovered message round-tripped through HandleUpdates: its user
	// fed the peer cache.
	if un := c.peersManager().LookupUsername(55); un != "recovered" {
		t.Fatalf("recovered user not ingested: username = %q", un)
	}
	if inv.callCount() < 2 {
		t.Fatalf("expected at least two getDifference calls, got %d", inv.callCount())
	}
}

func TestUpdateRecoveryTooLongFastForwards(t *testing.T) {
	c, inv := newClientWithMock(t)
	r := c.updateRec
	seedEngine(r, 10, 0, 100, 1)
	inv.setResult(tg.UpdatesGetDifferenceTypeID, &tg.UpdatesDifferenceTooLong{PTS: 5000})

	r.pass()
	if got := engineState(r).Pts; got != 5000 {
		t.Fatalf("pts = %d, want 5000 after tooLong", got)
	}
}

func TestUpdateRecoveryReseedsOnPersistentTimestampError(t *testing.T) {
	c, inv := newClientWithBotRPCMock(t)
	r := c.updateRec
	seedEngine(r, 10, 0, 100, 1)
	inv.setError(tg.UpdatesGetDifferenceTypeID, tgerr.New(400, "PERSISTENT_TIMESTAMP_OUTDATED"))
	inv.setResult(tg.UpdatesGetStateTypeID, &tg.UpdatesState{PTS: 77, Qts: 3, Date: 200, Seq: 5})
	inv.setResult(tg.UpdatesDifferenceEmptyTypeID, nil)

	err := r.recoverDifference(context.Background())
	if err != nil {
		t.Fatalf("recoverDifference: %v", err)
	}
	if got := engineState(r).Pts; got != 77 {
		t.Fatalf("pts = %d, want 77 after reseed", got)
	}
}

func TestUpdateRecoveryChannelGap(t *testing.T) {
	c, inv := newClientWithMock(t)
	r := c.updateRec
	seedEngine(r, 10, 0, 100, 1)

	const chID = int64(100)
	c.CachePeer(chID, &tg.InputPeerChannel{ChannelID: chID, AccessHash: 42})

	// First channel update starts tracking; the next one jumps.
	r.trackChannelPTS(chID, 5, 1)
	if !r.trackChannelPTS(chID, 9, 1) {
		t.Fatal("channel pts jump not detected")
	}

	inv.setResult(tg.UpdatesGetDifferenceTypeID, &tg.UpdatesDifferenceEmpty{})
	inv.setResult(tg.UpdatesGetChannelDifferenceTypeID, &tg.UpdatesChannelDifference{
		Final: true,
		PTS:   9,
		NewMessages: []tg.MessageClass{&tg.Message{
			ID:     3,
			PeerID: &tg.PeerChannel{ChannelID: chID},
		}},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})

	if err := r.recoverAll(context.Background()); err != nil {
		t.Fatalf("recoverAll: %v", err)
	}
	if got := r.channelPTS(chID); got != 9 {
		t.Fatalf("channel pts = %d, want 9", got)
	}
	// The gap request addressed the cached channel with its access hash.
	last := inv.lastCall()
	req, ok := last.(*tg.UpdatesGetChannelDifferenceRequest)
	if !ok {
		t.Fatalf("last call = %T, want getChannelDifference", last)
	}
	if ch, ok := req.Channel.(*tg.InputChannel); !ok || ch.ChannelID != chID || ch.AccessHash != 42 {
		t.Fatalf("channel input = %+v, want channel 100 hash 42", req.Channel)
	}
}

func TestUpdateRecoveryAffectedMessagesMiddleware(t *testing.T) {
	c, _ := newClientWithMock(t)
	r := c.updateRec
	seedEngine(r, 10, 0, 100, 1)

	var next tg.Invoker = tg.InvokerFunc(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
		return &tg.MessagesAffectedMessages{PTS: 13, PTSCount: 3}, nil
	})
	inv := r.trackAffectedResponses()(next)
	res, err := inv.RPCInvoke(context.Background(), &tg.MessagesDeleteMessagesRequest{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.(*tg.MessagesAffectedMessages); !ok {
		t.Fatalf("result not passed through: %T", res)
	}
	if got := engineState(r).Pts; got != 13 {
		t.Fatalf("pts = %d, want 13 from affectedMessages", got)
	}
}

func TestUpdateRecoverySeedPrefersPersistedState(t *testing.T) {
	c, _ := newClientWithMock(t)
	r := c.updateRec
	seedEngine(r, 42, 1, 100, 3)

	// A server state older than the persisted one must not clobber it.
	r.seedFromServer(&tg.UpdatesState{PTS: 10, Qts: 0, Date: 99, Seq: 1})
	if got := engineState(r).Pts; got != 42 {
		t.Fatalf("persisted pts clobbered by server seed: %d", got)
	}
}

func TestUpdateRecoveryChannelDifferenceTooLong(t *testing.T) {
	c, inv := newClientWithMock(t)
	r := c.updateRec
	seedEngine(r, 10, 0, 100, 1)

	const chID = int64(200)
	c.CachePeer(chID, &tg.InputPeerChannel{ChannelID: chID, AccessHash: 7})
	r.mu.Lock()
	r.channels[chID] = 5
	r.pending[chID] = true
	r.mu.Unlock()

	inv.setResult(tg.UpdatesGetDifferenceTypeID, &tg.UpdatesDifferenceEmpty{})
	inv.setResult(tg.UpdatesGetChannelDifferenceTypeID, &tg.UpdatesChannelDifferenceTooLong{
		Timeout: 400,
		Messages: []tg.MessageClass{&tg.Message{
			ID:     9,
			PeerID: &tg.PeerChannel{ChannelID: chID},
		}},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})

	if err := r.recoverAll(context.Background()); err != nil {
		t.Fatalf("recoverAll: %v", err)
	}
	if got := r.channelPTS(chID); got != 400 {
		t.Fatalf("channel pts = %d, want 400 after tooLong", got)
	}
}
