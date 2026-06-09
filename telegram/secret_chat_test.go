package telegram

import "testing"

// TestSecretChatSeqNoParity verifies the MTProto E2E seq_no encoding
// (https://core.telegram.org/api/end-to-end/seq_no): out_seq_no = 2*count + x
// and the embedded in_seq_no = 2*recv + (1-x), where x is 0 for the chat
// creator and 1 for the joiner.
func TestSecretChatSeqNoParity(t *testing.T) {
	t.Run("creator x=0", func(t *testing.T) {
		sc := &SecretChat{Outgoing: true}
		// Three outgoing messages: 0, 2, 4 (even).
		for i, want := range []int32{0, 2, 4} {
			if got := sc.NextOutSeqNo(); got != want {
				t.Fatalf("out msg %d: got %d, want %d", i, got, want)
			}
		}
		// in_seq_no embeds opposite parity (odd) and does not advance.
		if got := sc.CurrentInSeqNo(); got != 1 {
			t.Fatalf("in_seq_no with no received: got %d, want 1", got)
		}
		sc.NextInSeqNo()
		if got := sc.CurrentInSeqNo(); got != 3 {
			t.Fatalf("in_seq_no after one received: got %d, want 3", got)
		}
		if got := sc.ExpectedInboundParity(); got != 1 {
			t.Fatalf("creator expects inbound odd parity, got %d", got)
		}
	})

	t.Run("joiner x=1", func(t *testing.T) {
		sc := &SecretChat{Outgoing: false}
		// Three outgoing messages: 1, 3, 5 (odd).
		for i, want := range []int32{1, 3, 5} {
			if got := sc.NextOutSeqNo(); got != want {
				t.Fatalf("out msg %d: got %d, want %d", i, got, want)
			}
		}
		// in_seq_no embeds opposite parity (even).
		if got := sc.CurrentInSeqNo(); got != 0 {
			t.Fatalf("in_seq_no with no received: got %d, want 0", got)
		}
		if got := sc.ExpectedInboundParity(); got != 0 {
			t.Fatalf("joiner expects inbound even parity, got %d", got)
		}
	})
}
