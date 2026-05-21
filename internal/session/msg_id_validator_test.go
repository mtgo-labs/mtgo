package session

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestMsgIDValidatorParity(t *testing.T) {
	v := newMsgIDValidator(func() int64 { return time.Now().Unix() })
	serverMsgID := (time.Now().Unix() << 32) | 1
	if !v.Check(serverMsgID) {
		t.Error("odd msg_id should be accepted")
	}
	clientMsgID := (time.Now().Unix() << 32)
	if v.Check(clientMsgID) {
		t.Error("even msg_id should be rejected")
	}
}

func TestMsgIDValidatorReplay(t *testing.T) {
	v := newMsgIDValidator(func() int64 { return time.Now().Unix() })
	msgID := (time.Now().Unix() << 32) | 1
	if !v.Check(msgID) {
		t.Error("first occurrence should be accepted")
	}
	if v.Check(msgID) {
		t.Error("duplicate msg_id should be rejected")
	}
}

func TestMsgIDValidatorFutureWindow(t *testing.T) {
	now := time.Now().Unix()
	v := newMsgIDValidator(func() int64 { return now })

	valid := ((now + 29) << 32) | 1
	if !v.Check(valid) {
		t.Error("msg_id within 30s future window should be accepted")
	}

	tooFar := ((now + 31) << 32) | 1
	if v.Check(tooFar) {
		t.Error("msg_id > 30s in future should be rejected")
	}
}

func TestMsgIDValidatorPastWindow(t *testing.T) {
	now := time.Now().Unix()
	v := newMsgIDValidator(func() int64 { return now })

	valid := ((now - 299) << 32) | 1
	if !v.Check(valid) {
		t.Error("msg_id within 300s past window should be accepted")
	}

	tooOld := ((now - 301) << 32) | 1
	if v.Check(tooOld) {
		t.Error("msg_id > 300s in past should be rejected")
	}
}

func TestMsgIDValidatorCapacity(t *testing.T) {
	now := time.Now().Unix()
	var called atomic.Int64
	v := newMsgIDValidator(func() int64 { return called.Load() })
	called.Store(now)

	for i := 0; i < msgIDReplayCapacity+10; i++ {
		msgID := ((now - int64(i)) << 32) | 1
		v.Check(msgID)
	}

	old := ((now - 400) << 32) | 1
	if v.Check(old) {
		t.Error("very old msg_id should still be rejected by time window")
	}
}

func TestMsgIDValidatorReset(t *testing.T) {
	v := newMsgIDValidator(func() int64 { return time.Now().Unix() })
	msgID := (time.Now().Unix() << 32) | 1
	v.Check(msgID)

	v.Reset()
	if !v.Check(msgID) {
		t.Error("msg_id should be accepted after reset")
	}
}
