package session

import "testing"

func TestSeqNoContentRelated(t *testing.T) {
	gen := NewSeqNoGenerator()
	first := gen.Next(true)
	second := gen.Next(true)
	if first != 1 {
		t.Errorf("first content seq_no = %d, want 1", first)
	}
	if second != 3 {
		t.Errorf("second content seq_no = %d, want 3", second)
	}
}

func TestSeqNoAck(t *testing.T) {
	gen := NewSeqNoGenerator()
	_ = gen.Next(true)
	ack := gen.Next(false)
	if ack != 2 {
		t.Errorf("ack seq_no = %d, want 2", ack)
	}
	next := gen.Next(true)
	if next != 3 {
		t.Errorf("next content seq_no = %d, want 3", next)
	}
}

func TestSeqNoReset(t *testing.T) {
	gen := NewSeqNoGenerator()
	gen.Next(true)
	gen.Next(true)
	gen.Reset()
	val := gen.Next(true)
	if val != 1 {
		t.Errorf("after reset, first seq_no = %d, want 1", val)
	}
}
