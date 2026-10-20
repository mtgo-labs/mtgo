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

	valid := ((now + 59) << 32) | 1
	if !v.Check(valid) {
		t.Error("msg_id within 60s future window should be accepted")
	}

	tooFar := ((now + 61) << 32) | 1
	if v.Check(tooFar) {
		t.Error("msg_id > 60s in future should be rejected")
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

// TestMsgIDValidatorSkewedLocalClock is the regression test for the field
// report where a client went deaf with healthy connections: the validator
// compared server-stamped msg_ids against the raw local clock, so a local
// clock >60s behind Telegram time rejected every inbound frame. The session
// now wires serverTS to the server-time-corrected clock; with a corrected
// reference, frames stay valid even under local clock skew.
func TestMsgIDValidatorSkewedLocalClock(t *testing.T) {
	serverNow := time.Now().Unix()
	// Local clock 10 minutes behind the server.
	v := newMsgIDValidator(func() int64 { return serverNow - 600 })

	fresh := ((serverNow - 5) << 32) | 1
	if v.Check(fresh) {
		t.Error("msg_id fresh in server time accepted against a skewed reference: validator is not using server-corrected time")
	}

	// With the CORRECTED reference (as wired in NewSession), the same frame
	// must validate.
	corrected := newMsgIDValidator(func() int64 { return serverNow })
	if !corrected.Check(fresh) {
		t.Error("fresh msg_id must validate against the server-corrected clock")
	}

	// Reasons must be observable for diagnostics.
	if ok, reason := corrected.CheckWithReason(((serverNow - 4000) << 32) | 1); ok || reason == "" {
		t.Errorf("stale msg_id must reject with a reason, got ok=%v reason=%q", ok, reason)
	}
	if ok, reason := corrected.CheckWithReason((serverNow << 32) | 2); ok || reason == "" {
		t.Errorf("even msg_id must reject with a reason, got ok=%v reason=%q", ok, reason)
	}
	if ok, reason := corrected.CheckWithReason(((serverNow + 1) << 32) | 1); !ok || reason != "" {
		t.Errorf("valid msg_id must accept with empty reason, got ok=%v reason=%q", ok, reason)
	}
}
