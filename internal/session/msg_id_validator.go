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
	serverTS func() int64
}

func newMsgIDValidator(serverTS func() int64) *msgIDValidator {
	return &msgIDValidator{
		ids:      make([]int64, 0, msgIDReplayCapacity),
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

	for _, id := range v.ids {
		if id == msgID {
			return false
		}
	}

	for i := range v.ids {
		if v.ids[i] < msgID {
			continue
		}
		if v.ids[i] > msgID {
			break
		}
	}

	v.ids = append(v.ids, msgID)
	if len(v.ids) > msgIDReplayCapacity {
		v.ids = v.ids[len(v.ids)-msgIDReplayCapacity:]
	}

	return true
}

func (v *msgIDValidator) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.ids = v.ids[:0]
}
