package session

import (
	"strconv"
	"testing"
)

func TestSessionWorkersScaleTo1000IsolatedSessions(t *testing.T) {
	const sessionCount = 1_000
	sessions := make([]*Session, 0, sessionCount)
	cleanups := make([]func(), 0, sessionCount)
	ids := make(map[int64]struct{}, sessionCount)

	for i := 0; i < sessionCount; i++ {
		sess := newSessionWithAuthKey(t)
		if _, duplicate := ids[sess.SessionID()]; duplicate {
			t.Fatalf("duplicate session ID at index %d", i)
		}
		ids[sess.SessionID()] = struct{}{}

		transport := newMockTransport()
		sess.SetTransport(transport)
		cleanups = append(cleanups, startTestWorkers(sess, transport))
		sessions = append(sessions, sess)
	}

	const sharedMsgID = int64(42)
	for i, sess := range sessions {
		if _, err := sess.pending.Register(sharedMsgID, false); err != nil {
			t.Fatalf("session %d register: %v", i, err)
		}
	}
	for i, sess := range sessions {
		if !sess.pending.Has(sharedMsgID) {
			t.Fatalf("session %d lost isolated pending request", i)
		}
		sess.pending.Cancel(sharedMsgID)
	}

	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}
}

func TestMsgIDReplayStateAllocatesLazily(t *testing.T) {
	validator := newMsgIDValidator(func() int64 { return 1_700_000_000 })
	if validator.ids != nil || validator.idSet != nil {
		t.Fatal("replay state allocated before the first server message")
	}
	msgID := int64(1_700_000_000)<<32 | 1
	if !validator.Check(msgID) {
		t.Fatal("first valid message ID was rejected")
	}
	if validator.ids == nil || validator.idSet == nil {
		t.Fatal("replay state was not allocated for an authenticated message")
	}
	if validator.Check(msgID) {
		t.Fatal("replayed message ID was accepted")
	}
}

func BenchmarkSessionFleetConstruction(b *testing.B) {
	for _, size := range []int{1, 100, 1_000, 10_000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			b.ReportAllocs()
			b.ReportMetric(float64(size), "sessions/op")
			for b.Loop() {
				fleet := make([]*Session, size)
				for i := range fleet {
					fleet[i] = newSessionWithAuthKey(b)
				}
			}
		})
	}
}
