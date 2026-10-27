package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// The server returns rpc_error -500 "No workers running" when the server-side
// connection has lost its worker pool. Every RPC on that connection keeps
// failing until the connection is re-established, so in-place retries are
// futile: production logs show 18+ consecutive failures over 62 seconds until
// the TCP connection finally died on its read deadline.
//
// These tests pin the recovery contract: after noWorkersRecycleThreshold
// consecutive errors the session must recycle itself (fail-fast, like the
// write circuit breaker) so the client reconnects and replays the RPC.

func TestInvokeNoWorkersRunningRecyclesPoisonedConnection(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	done := make(chan error, 1)
	go func() {
		_, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: 1}, 3, 5*time.Second)
		done <- err
	}()

	// Answer each attempt with -500 "No workers running". The second
	// consecutive error must trip the recycle, so only two responses are
	// fed before the caller returns.
	for range noWorkersRecycleThreshold {
		sent := <-mt.sendCh
		msg := unpackIncoming(sent, s)
		if msg == nil {
			t.Fatal("outgoing message did not decode")
		}
		mt.recvCh <- makeRPCErrorResponse(s, msg.MsgID, -500, "No workers running")
	}

	select {
	case err := <-done:
		if !errors.Is(err, ErrSessionClosed) {
			t.Fatalf("Invoke error = %v, want ErrSessionClosed after poisoned-connection recycle", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Invoke did not return after no-workers recycle")
	}

	source, _, cause := s.ShutdownCause()
	if source != "noWorkers" || cause == nil {
		t.Fatalf("ShutdownCause() = (%q, %v), want source noWorkers", source, cause)
	}

	// The poisoned connection must not carry a third in-place attempt.
	select {
	case <-mt.sendCh:
		t.Fatal("Invoke sent another RPC on a poisoned connection")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestInvokeNoWorkersRunningSingleErrorDoesNotRecycle(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	done := make(chan error, 1)
	go func() {
		// retries=1: a single attempt so the error surfaces to the caller
		// instead of driving the streak to the threshold in place.
		_, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: 2}, 1, 5*time.Second)
		done <- err
	}()

	sent := <-mt.sendCh
	msg := unpackIncoming(sent, s)
	if msg == nil {
		t.Fatal("outgoing message did not decode")
	}
	mt.recvCh <- makeRPCErrorResponse(s, msg.MsgID, -500, "No workers running")

	select {
	case err := <-done:
		if errors.Is(err, ErrSessionClosed) {
			t.Fatalf("Invoke error = %v, single no-workers error must not recycle the session", err)
		}
		if !isNoWorkersRunningError(rpcErrorOf(t, err)) {
			t.Fatalf("Invoke error = %v, want the surfaced no-workers RPC error", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Invoke did not return")
	}

	if _, _, cause := s.ShutdownCause(); cause != nil {
		t.Fatalf("ShutdownCause() = %v, want none after a single no-workers error", cause)
	}
}

func TestInvokeNoWorkersStreakResetsOnSuccess(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	// startInvoke launches Invoke in the background; the caller reads the
	// outgoing RPC from sendCh and feeds a response before waiting on done.
	startInvoke := func(pingID int64, retries int) <-chan error {
		done := make(chan error, 1)
		go func() {
			_, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: pingID}, retries, 5*time.Second)
			done <- err
		}()
		return done
	}
	respondNoWorkers := func() {
		t.Helper()
		sent := <-mt.sendCh
		msg := unpackIncoming(sent, s)
		if msg == nil {
			t.Fatal("outgoing message did not decode")
		}
		mt.recvCh <- makeRPCErrorResponse(s, msg.MsgID, -500, "No workers running")
	}

	// First call: one no-workers error (streak=1, below threshold).
	first := startInvoke(1, 1)
	respondNoWorkers()
	if err := <-first; errors.Is(err, ErrSessionClosed) {
		t.Fatalf("first call recycled the session: %v", err)
	}

	// Successful call resets the streak.
	second := startInvoke(2, 1)
	sent := <-mt.sendCh
	msg := unpackIncoming(sent, s)
	if msg == nil {
		t.Fatal("outgoing message did not decode")
	}
	mt.recvCh <- makeSuccessResponse(s, msg.MsgID, 2)
	if err := <-second; err != nil {
		t.Fatalf("success call error = %v", err)
	}

	// After the reset, the streak must climb from zero again: the first
	// no-workers error of a fresh call stays below the threshold.
	third := startInvoke(3, 1)
	respondNoWorkers()
	if err := <-third; errors.Is(err, ErrSessionClosed) {
		t.Fatalf("post-reset single error recycled the session: %v", err)
	}
	if _, _, cause := s.ShutdownCause(); cause != nil {
		t.Fatalf("ShutdownCause() = %v, want none after reset + single error", cause)
	}
}

func rpcErrorOf(t *testing.T, err error) *tgerr.Error {
	t.Helper()
	var e *tgerr.Error
	if errors.As(err, &e) {
		return e
	}
	t.Fatalf("error %v does not wrap an RPC error", err)
	return nil
}
