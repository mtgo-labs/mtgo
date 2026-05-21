package session

import (
	"sync"
	"time"
)

const (
	msgIDReplayCapacity = 256
	msgIDFutureWindow   = 30 * time.Second
	msgIDPastWindow     = 300 * time.Second
)

type msgIDValidator struct {
	mu       sync.Mutex
	ids      []int64
	idSet    map[int64]struct{}
	serverTS func() int64
}

func newMsgIDValidator(serverTS func() int64) *msgIDValidator {
	return &msgIDValidator{
		ids:      make([]int64, 0, msgIDReplayCapacity),
		idSet:    make(map[int64]struct{}, msgIDReplayCapacity),
		serverTS: serverTS,
	}
}

func (v *msgIDValidator) Check(msgID int64) bool {
	if msgID%2 != 1 {
		return false
	}

	now := v.serverTS()
	msgTime := msgID >> 32
	if msgTime > now+30 {
		return false
	}
	if msgTime < now-300 {
		return false
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	if _, exists := v.idSet[msgID]; exists {
		return false
	}

	v.ids = append(v.ids, msgID)
	v.idSet[msgID] = struct{}{}
	if len(v.ids) > msgIDReplayCapacity {
		evicted := v.ids[:len(v.ids)-msgIDReplayCapacity]
		for _, id := range evicted {
			delete(v.idSet, id)
		}
		v.ids = v.ids[len(v.ids)-msgIDReplayCapacity:]
	}

	return true
}

func (v *msgIDValidator) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.ids = v.ids[:0]
	for k := range v.idSet {
		delete(v.idSet, k)
	}
}
