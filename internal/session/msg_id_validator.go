package session

import (
	"fmt"
	"sync"
	"time"
)

const (
	msgIDReplayCapacity = 256
	// msgIDFutureWindow tolerates server-time jitter in the validator's
	// clock reference. 60s keeps genuine frames under load (server batches,
	// minor offset drift) from being classified as "from the future".
	msgIDFutureWindow = 60 * time.Second
	msgIDPastWindow   = 300 * time.Second
)

type msgIDValidator struct {
	mu       sync.Mutex
	ids      []int64
	idSet    map[int64]struct{}
	serverTS func() int64
}

func newMsgIDValidator(serverTS func() int64) *msgIDValidator {
	return &msgIDValidator{
		serverTS: serverTS,
	}
}

func (v *msgIDValidator) Check(msgID int64) bool {
	ok, _ := v.CheckWithReason(msgID)
	return ok
}

// CheckWithReason is Check with a human-readable rejection reason for
// logging. Rejecting every inbound frame silently (e.g. because the clock
// reference drifted from server time) makes a client go deaf with healthy
// connections and no diagnostics — the reason must be observable.
func (v *msgIDValidator) CheckWithReason(msgID int64) (bool, string) {
	if msgID%2 != 1 {
		return false, "even msg_id (not server-originated)"
	}

	now := v.serverTS()
	msgTime := msgID >> 32
	if msgTime > now+int64(msgIDFutureWindow/time.Second) {
		return false, fmt.Sprintf("msg_id timestamp %d is in the future vs reference %d (reference not synced to server time?)", msgTime, now)
	}
	if msgTime < now-int64(msgIDPastWindow/time.Second) {
		return false, fmt.Sprintf("msg_id timestamp %d is older than the %s replay window vs reference %d (reference ahead of server time?)", msgTime, msgIDPastWindow, now)
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	if v.idSet == nil {
		v.ids = make([]int64, 0, msgIDReplayCapacity)
		v.idSet = make(map[int64]struct{}, msgIDReplayCapacity)
	}

	if _, exists := v.idSet[msgID]; exists {
		return false, "replayed msg_id"
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

	return true, ""
}

func (v *msgIDValidator) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.ids = v.ids[:0]
	for k := range v.idSet {
		delete(v.idSet, k)
	}
}
