package session

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type SessionState uint8

const (
	StateIdle SessionState = iota
	StateConnecting
	StateActive
	StateDraining
	StateClosed
)

func (s SessionState) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateConnecting:
		return "Connecting"
	case StateActive:
		return "Active"
	case StateDraining:
		return "Draining"
	case StateClosed:
		return "Closed"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

var allowedTransitions = map[SessionState][]SessionState{
	StateIdle:       {StateConnecting},
	StateConnecting: {StateActive, StateDraining, StateClosed},
	StateActive:     {StateDraining, StateClosed},
	StateDraining:   {StateConnecting, StateClosed},
	StateClosed:     {},
}

type stateMachine struct {
	mu    sync.Mutex
	state atomic.Uint32
}

func newStateMachine() *stateMachine {
	sm := &stateMachine{}
	sm.state.Store(uint32(StateIdle))
	return sm
}

func (sm *stateMachine) State() SessionState {
	return SessionState(sm.state.Load())
}

func (sm *stateMachine) transition(from, to SessionState) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if SessionState(sm.state.Load()) != from {
		return false
	}
	return sm.doTransition(to)
}

func (sm *stateMachine) transitionTo(to SessionState) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.doTransition(to)
}

func (sm *stateMachine) doTransition(to SessionState) bool {
	cur := SessionState(sm.state.Load())
	allowed, ok := allowedTransitions[cur]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			sm.state.Store(uint32(to))
			return true
		}
	}
	return false
}

func (sm *stateMachine) canWrite() bool {
	s := SessionState(sm.state.Load())
	return s == StateActive || s == StateConnecting
}

func (sm *stateMachine) canConnect() bool {
	return SessionState(sm.state.Load()) == StateIdle
}

func (sm *stateMachine) canReconnect() bool {
	return SessionState(sm.state.Load()) == StateDraining
}

func (sm *stateMachine) canClose() bool {
	return SessionState(sm.state.Load()) != StateClosed
}

func (sm *stateMachine) isActive() bool {
	return SessionState(sm.state.Load()) == StateActive
}

func (sm *stateMachine) isClosed() bool {
	return SessionState(sm.state.Load()) == StateClosed
}

func (sm *stateMachine) forceSetState(state SessionState) {
	sm.mu.Lock()
	sm.state.Store(uint32(state))
	sm.mu.Unlock()
}
