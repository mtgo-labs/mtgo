package telegram

// Regression tests for update-subscription restoration after a server-side
// session replacement (mtgo-bug-session-death-update-dispatch-deaf.md).
//
// Per https://core.telegram.org/api/updates, update events are pushed only to
// connections that have run initConnection and an API method. When the server
// replaces the session (new_session_created — MTProto server garbage
// collection or reboot) that identification is lost server-side, so the
// client must re-identify and re-subscribe or the server silently stops
// pushing updates while pings keep being answered.

import (
	"fmt"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

func ctorName(id uint32) string {
	switch id {
	case tg.PingTypeID:
		return "ping"
	case tg.InvokeWithLayerTypeID:
		return "invokeWithLayer"
	case tg.InitConnectionTypeID:
		return "initConnection"
	case tg.UpdatesGetStateTypeID:
		return "updates.getState"
	case tg.HelpGetConfigTypeID:
		return "help.getConfig"
	case tg.MsgsAckTypeID:
		return "msgs_ack"
	case tg.NewSessionCreatedTypeID:
		return "new_session_created"
	default:
		return fmt.Sprintf("0x%08x", id)
	}
}

// waitForRequest blocks until the recorded client traffic contains a request
// with the given constructor ID (and, when requireLayer, wrapped in
// invokeWithLayer+initConnection).
func waitForRequest(t *testing.T, srv *testServer, constructor uint32, requireLayer bool) recordedRequest {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		for _, r := range srv.recorded() {
			if r.constructor == constructor && (!requireLayer || r.withLayer) {
				return r
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for client request %s (requireLayer=%v); recorded: %+v",
		ctorName(constructor), requireLayer, srv.recorded())
	return recordedRequest{}
}

func waitForCond(t *testing.T, what string, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func connectAndIdentify(t *testing.T, client *Client, srv *testServer) {
	t.Helper()
	srv.startRecording()
	if err := client.Connect(5 * time.Second); err != nil {
		t.Fatalf("connect: %v", err)
	}
	waitForCond(t, "client to become connected", 5*time.Second, client.IsConnected)
	// postConnect's identification/state RPCs.
	waitForRequest(t, srv, tg.UpdatesGetStateTypeID, false)
	time.Sleep(200 * time.Millisecond)
	srv.clearRecorded()
}

// A silent new_session_created on a live, passive client must trigger
// re-identification of the replaced server-side session (initConnection-wrapped
// updates.getState) and fire the reconnect hooks for gap recovery. Previously
// the only outbound traffic was the msgs_ack for the notification itself, and
// the client stayed deaf to updates indefinitely.
func TestSilentNewSessionCreatedResubscribes(t *testing.T) {
	client, srv := newTestClient(1, "hash", Config{ReconnectEnabled: true})
	defer srv.Close()
	defer client.Close()

	hookFired := make(chan struct{}, 4)
	client.OnReconnect(func(*Client) { hookFired <- struct{}{} })

	connectAndIdentify(t, client, srv)

	if err := srv.push(&tg.NewSessionCreated{FirstMsgID: 1, UniqueID: 42, ServerSalt: 7}); err != nil {
		t.Fatalf("push new_session_created: %v", err)
	}

	// The resubscribe must re-identify the session: the state fetch goes out
	// wrapped in invokeWithLayer+initConnection.
	waitForRequest(t, srv, tg.UpdatesGetStateTypeID, true)
	select {
	case <-hookFired:
	case <-time.After(5 * time.Second):
		t.Fatal("reconnect hook did not fire after new_session_created")
	}
}

// After a session death and automatic reconnect, the re-created session must
// re-subscribe (updates.getState), not merely re-identify via the DC-option
// config refresh.
func TestSessionDeathReconnectResubscribes(t *testing.T) {
	client, srv := newTestClient(1, "hash", Config{
		ReconnectEnabled:     true,
		ReconnectBaseDelay:   100 * time.Millisecond,
		UpdateSilenceTimeout: 0, // isolate the reconnect path from the watchdog
	})
	defer srv.Close()
	defer client.Close()

	connectAndIdentify(t, client, srv)

	srv.connMu.Lock()
	conn := srv.curConn
	srv.connMu.Unlock()
	if conn == nil {
		t.Fatal("no current connection")
	}
	conn.Close()

	waitForCond(t, "reconnect", 15*time.Second, client.IsConnected)
	waitForRequest(t, srv, tg.UpdatesGetStateTypeID, false)
}

// The update-silence watchdog (docs: "no updates for 15 minutes or longer"
// must trigger gap recovery) re-runs the subscription restore when no update
// has arrived for the configured window, and an incoming update resets the
// timer.
func TestUpdateSilenceWatchdogResubscribes(t *testing.T) {
	client, srv := newTestClient(1, "hash", Config{
		ReconnectEnabled:     true,
		UpdateSilenceTimeout: 300 * time.Millisecond,
	})
	defer srv.Close()
	defer client.Close()

	hookFired := make(chan struct{}, 4)
	client.OnReconnect(func(*Client) { hookFired <- struct{}{} })

	connectAndIdentify(t, client, srv)

	// No updates arrive: the watchdog must fire within a couple of windows.
	waitForRequest(t, srv, tg.UpdatesGetStateTypeID, false)
	select {
	case <-hookFired:
	case <-time.After(5 * time.Second):
		t.Fatal("reconnect hook did not fire after update silence")
	}

	// An incoming update resets the silence timer: no further resubscribe for
	// at least a full window after it.
	srv.clearRecorded()
	if err := srv.push(&tg.UpdateShort{Update: &tg.UpdateDeleteMessages{Messages: []int32{1}, PTS: 1, PTSCount: 1}, Date: int32(time.Now().Unix())}); err != nil {
		t.Fatalf("push update: %v", err)
	}
	time.Sleep(600 * time.Millisecond)
	for _, r := range srv.recorded() {
		if r.constructor == tg.UpdatesGetStateTypeID {
			t.Fatalf("watchdog re-fired %v after an update reset the timer", 600*time.Millisecond)
		}
	}
}
