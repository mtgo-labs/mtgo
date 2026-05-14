package session

import (
	"testing"
	"time"
)

func TestMsgFactoryMonotonicMsgIDs(t *testing.T) {
	gen := NewMsgFactory(time.Unix(1700000000, 0))
	m1 := gen.AllocateMsgID()
	m2 := gen.AllocateMsgID()
	if m2 <= m1 {
		t.Errorf("msg IDs not monotonic: %d -> %d", m1, m2)
	}
}

func TestMsgFactorySeqNoContent(t *testing.T) {
	gen := NewMsgFactory(time.Unix(1700000000, 0))
	s1 := gen.AllocateSeqNo(true)
	s2 := gen.AllocateSeqNo(true)
	if s2 <= s1 {
		t.Errorf("seq_no not monotonic: %d -> %d", s1, s2)
	}
}

func TestMsgFactorySeqNoAck(t *testing.T) {
	gen := NewMsgFactory(time.Unix(1700000000, 0))
	_ = gen.AllocateSeqNo(true)
	ack := gen.AllocateSeqNo(false)
	if ack != 2 {
		t.Errorf("ack seq_no=%d, want 2", ack)
	}
}

func TestMsgFactoryUpdateServerTime(t *testing.T) {
	gen := NewMsgFactory(time.Unix(1700000000, 0))
	gen.AllocateMsgID()
	newTime := time.Unix(1800000000, 0)
	gen.UpdateServerTime(newTime)
	id := gen.AllocateMsgID()
	expectedBase := int64(newTime.Unix()) << 32
	if id < expectedBase {
		t.Errorf("msg_id=%d less than expected base after update=%d", id, expectedBase)
	}
}
