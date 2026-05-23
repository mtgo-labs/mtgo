package session

import (
	"fmt"
	"sync"
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
	state SessionState
}

func newStateMachine() *stateMachine {
	return &stateMachine{state: StateIdle}
}

func (sm *stateMachine) State() SessionState {
	sm.mu.Lock()
	s := sm.state
	sm.mu.Unlock()
	return s
}

func (sm *stateMachine) transition(from, to SessionState) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.state != from {
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
	allowed, ok := allowedTransitions[sm.state]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			sm.state = to
			return true
		}
	}
	return false
}

func (sm *stateMachine) canWrite() bool {
	sm.mu.Lock()
	s := sm.state
	sm.mu.Unlock()
	return s == StateActive || s == StateConnecting
}

func (sm *stateMachine) canConnect() bool {
	sm.mu.Lock()
	s := sm.state
	sm.mu.Unlock()
	return s == StateIdle
}

func (sm *stateMachine) canReconnect() bool {
	sm.mu.Lock()
	s := sm.state
	sm.mu.Unlock()
	return s == StateDraining
}

func (sm *stateMachine) canClose() bool {
	sm.mu.Lock()
	s := sm.state
	sm.mu.Unlock()
	return s != StateClosed
}

func (sm *stateMachine) isActive() bool {
	sm.mu.Lock()
	s := sm.state
	sm.mu.Unlock()
	return s == StateActive
}

func (sm *stateMachine) isClosed() bool {
	sm.mu.Lock()
	s := sm.state
	sm.mu.Unlock()
	return s == StateClosed
}

func (sm *stateMachine) forceSetState(state SessionState) {
	sm.mu.Lock()
	sm.state = state
	sm.mu.Unlock()
}
