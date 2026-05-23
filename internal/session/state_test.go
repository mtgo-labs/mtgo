package session

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestStateInitial(t *testing.T) {
	sm := newStateMachine()
	if sm.State() != StateIdle {
		t.Fatalf("initial state = %s, want Idle", sm.State())
	}
}

func TestStateTransitionsAllowed(t *testing.T) {
	tests := []struct {
		from, to SessionState
		want     bool
	}{
		{StateIdle, StateConnecting, true},
		{StateIdle, StateActive, false},
		{StateIdle, StateDraining, false},
		{StateIdle, StateClosed, false},
		{StateConnecting, StateActive, true},
		{StateConnecting, StateDraining, true},
		{StateConnecting, StateClosed, true},
		{StateConnecting, StateIdle, false},
		{StateActive, StateDraining, true},
		{StateActive, StateClosed, true},
		{StateActive, StateIdle, false},
		{StateActive, StateConnecting, false},
		{StateDraining, StateConnecting, true},
		{StateDraining, StateClosed, true},
		{StateDraining, StateActive, false},
		{StateDraining, StateIdle, false},
		{StateClosed, StateIdle, false},
		{StateClosed, StateActive, false},
		{StateClosed, StateConnecting, false},
		{StateClosed, StateDraining, false},
	}
	for _, tt := range tests {
		sm := newStateMachine()
		sm.forceSetState(tt.from)
		got := sm.transition(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("transition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
		if tt.want {
			if sm.State() != tt.to {
				t.Errorf("after transition(%s, %s), state = %s, want %s", tt.from, tt.to, sm.State(), tt.to)
			}
		} else {
			if sm.State() != tt.from {
				t.Errorf("after failed transition(%s, %s), state = %s, want %s", tt.from, tt.to, sm.State(), tt.from)
			}
		}
	}
}

func TestStateTransitionWrongFrom(t *testing.T) {
	sm := newStateMachine()
	if sm.transition(StateActive, StateClosed) {
		t.Fatal("transition(Active, Closed) from Idle should fail")
	}
	if sm.State() != StateIdle {
		t.Fatalf("state = %s, want Idle", sm.State())
	}
}

func TestStateClosedIsTerminal(t *testing.T) {
	sm := newStateMachine()
	sm.forceSetState(StateClosed)
	for _, target := range []SessionState{StateIdle, StateConnecting, StateActive, StateDraining} {
		if sm.transition(StateClosed, target) {
			t.Errorf("transition(Closed, %s) should fail", target)
		}
	}
}

func TestStateCanHelpers(t *testing.T) {
	sm := newStateMachine()

	if !sm.canConnect() {
		t.Error("canConnect() = false in Idle, want true")
	}
	if sm.canWrite() {
		t.Error("canWrite() = true in Idle, want false")
	}
	if sm.canReconnect() {
		t.Error("canReconnect() = true in Idle, want false")
	}
	if !sm.canClose() {
		t.Error("canClose() = false in Idle, want true")
	}

	sm.forceSetState(StateConnecting)
	if sm.canConnect() {
		t.Error("canConnect() = true in Connecting, want false")
	}
	if !sm.canWrite() {
		t.Error("canWrite() = false in Connecting, want true (initial ping)")
	}

	sm.forceSetState(StateActive)
	if sm.canConnect() {
		t.Error("canConnect() = true in Active, want false")
	}
	if !sm.canWrite() {
		t.Error("canWrite() = false in Active, want true")
	}

	sm.forceSetState(StateDraining)
	if sm.canWrite() {
		t.Error("canWrite() = true in Draining, want false")
	}
	if !sm.canReconnect() {
		t.Error("canReconnect() = false in Draining, want true")
	}

	sm.forceSetState(StateClosed)
	if sm.canClose() {
		t.Error("canClose() = true in Closed, want false")
	}
	if sm.canWrite() {
		t.Error("canWrite() = true in Closed, want false")
	}
}

func TestStateConcurrentTransitions(t *testing.T) {
	sm := newStateMachine()
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	successCount := 0
	var mu sync.Mutex
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if sm.transition(StateIdle, StateConnecting) {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful transition, got %d", successCount)
	}
	if sm.State() != StateConnecting {
		t.Fatalf("state = %s, want Connecting", sm.State())
	}
}

func TestStateTransitionTo(t *testing.T) {
	sm := newStateMachine()
	if !sm.transitionTo(StateConnecting) {
		t.Fatal("transitionTo(Connecting) from Idle should succeed")
	}
	if !sm.transitionTo(StateActive) {
		t.Fatal("transitionTo(Active) from Connecting should succeed")
	}
	if sm.transitionTo(StateIdle) {
		t.Fatal("transitionTo(Idle) from Active should fail")
	}
}

func TestSessionSendFailsInClosed(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.sm.forceSetState(StateClosed)

	_, err := s.Send(context.Background(), 1, 1, &tg.PingRequest{PingID: 1}, time.Second)
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Send() in Closed: err = %v, want ErrSessionClosed", err)
	}
}

func TestSessionSendFailsInDraining(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.sm.forceSetState(StateDraining)

	_, err := s.Send(context.Background(), 1, 1, &tg.PingRequest{PingID: 1}, time.Second)
	if !errors.Is(err, ErrDraining) {
		t.Fatalf("Send() in Draining: err = %v, want ErrDraining", err)
	}
}

func TestSessionSendFailsInIdle(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	_, err := s.Send(context.Background(), 1, 1, &tg.PingRequest{PingID: 1}, time.Second)
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("Send() in Idle: err = %v, want ErrNotConnected", err)
	}
}

func TestSessionSendRawFailsInClosed(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.sm.forceSetState(StateClosed)

	_, err := s.SendRaw(context.Background(), 1, 1, []byte{0}, time.Second)
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("SendRaw() in Closed: err = %v, want ErrSessionClosed", err)
	}
}

func TestSessionSendRawFailsInDraining(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.sm.forceSetState(StateDraining)

	_, err := s.SendRaw(context.Background(), 1, 1, []byte{0}, time.Second)
	if !errors.Is(err, ErrDraining) {
		t.Fatalf("SendRaw() in Draining: err = %v, want ErrDraining", err)
	}
}

func TestSessionDoubleConnectRejected(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.sm.forceSetState(StateActive)

	err := s.ConnectContext(context.Background(), mt)
	if err == nil {
		t.Fatal("ConnectContext() on Active session should fail")
	}
}

func TestSessionConnectOnClosedRejected(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.sm.forceSetState(StateClosed)

	err := s.ConnectContext(context.Background(), mt)
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("ConnectContext() on Closed: err = %v, want ErrSessionClosed", err)
	}
}

func TestSessionLifecycleIdleConnectingActiveClosed(t *testing.T) {
	s := newSessionWithAuthKey(t)
	if s.sm.State() != StateIdle {
		t.Fatalf("initial state = %s, want Idle", s.sm.State())
	}

	s.sm.transition(StateIdle, StateConnecting)
	if s.sm.State() != StateConnecting {
		t.Fatalf("state = %s, want Connecting", s.sm.State())
	}

	s.sm.transition(StateConnecting, StateActive)
	if s.sm.State() != StateActive {
		t.Fatalf("state = %s, want Active", s.sm.State())
	}

	if !s.IsConnected() {
		t.Error("IsConnected() = false in Active")
	}

	s.sm.transition(StateActive, StateClosed)
	if s.sm.State() != StateClosed {
		t.Fatalf("state = %s, want Closed", s.sm.State())
	}

	if s.IsConnected() {
		t.Error("IsConnected() = true in Closed")
	}
}

func TestSessionFailedConnectTransitionsToClosed(t *testing.T) {
	s := newSessionWithAuthKey(t)
	sm := s.sm

	sm.transition(StateIdle, StateConnecting)
	if sm.State() != StateConnecting {
		t.Fatalf("state = %s, want Connecting", sm.State())
	}

	sm.transition(StateConnecting, StateClosed)
	if sm.State() != StateClosed {
		t.Fatalf("state = %s, want Closed", sm.State())
	}
}

func TestSessionStopFromActive(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.pingInterval = 1 * time.Hour

	go func() {
		sentData, ok := <-mt.sendCh
		if !ok {
			return
		}
		msg := unpackIncoming(sentData, s)
		if msg == nil {
			return
		}
		ping, ok := msg.Body.(*tg.PingRequest)
		if !ok {
			return
		}
		respMsgID := makeServerMsgID()
		respSeqNo := s.msgFactory.AllocateSeqNo(false)
		pong := &tg.Pong{MsgID: msg.MsgID, PingID: ping.PingID}
		encrypted := makeEncryptedResponse(s, respMsgID, uint32(respSeqNo), pong)
		mt.recvCh <- encrypted
	}()

	err := s.Start(3 * time.Second)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if s.sm.State() != StateActive {
		t.Fatalf("state after Start = %s, want Active", s.sm.State())
	}

	s.Stop()
	time.Sleep(50 * time.Millisecond)
	if s.sm.State() != StateClosed {
		t.Fatalf("state after Stop = %s, want Closed", s.sm.State())
	}
}

func TestSessionConcurrentSendStop(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.pingInterval = 1 * time.Hour

	pingReady := make(chan struct{})
	go func() {
		sentData, ok := <-mt.sendCh
		if !ok {
			return
		}
		msg := unpackIncoming(sentData, s)
		if msg == nil {
			return
		}
		ping, ok := msg.Body.(*tg.PingRequest)
		if !ok {
			return
		}
		respMsgID := makeServerMsgID()
		respSeqNo := s.msgFactory.AllocateSeqNo(false)
		pong := &tg.Pong{MsgID: msg.MsgID, PingID: ping.PingID}
		encrypted := makeEncryptedResponse(s, respMsgID, uint32(respSeqNo), pong)
		mt.recvCh <- encrypted
		close(pingReady)
	}()

	err := s.Start(3 * time.Second)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	<-pingReady

	var wg sync.WaitGroup
	const n = 20
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			msgID := s.msgFactory.AllocateMsgID()
			seqNo := s.msgFactory.AllocateSeqNo(true)
			_, _ = s.Send(context.Background(), msgID, uint32(seqNo), &tg.PingRequest{PingID: time.Now().UnixNano()}, 100*time.Millisecond)
		}()
	}

	time.Sleep(10 * time.Millisecond)
	s.Stop()
	wg.Wait()
}
