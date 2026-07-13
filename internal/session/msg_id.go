package session

import (
	"sync/atomic"
	"time"
)

// MsgIDGenerator produces unique, monotonically increasing MTProto message IDs.
//
// The timestamp portion is derived from the live wall clock plus a server-time
// offset (server_time - local_time), recomputed on every allocation so message
// IDs advance with real time and never go stale between corrections. The offset
// — a slowly changing skew, not an absolute timestamp — is what is stored and
// corrected. This mirrors mtcute, tdlib, gogram, MadelineProto and Telethon.
//
// A frozen absolute timestamp (the previous design) drifts into the past within
// minutes and forces the server to reject sends with bad_msg_notification code
// 16 (msg_id too low), producing a correct-then-immediately-stale loop.
//
// Message IDs are structured as (server_time << 32) with the low two bits held
// at 0b00, the client-side parity required by MTProto.
type MsgIDGenerator struct {
	// timeOffset is server_time - local_time, in seconds. server_time is
	// reconstructed as time.Now().Unix() + timeOffset on every allocation.
	timeOffset atomic.Int64
	// lastID is a strict monotonicity floor, guaranteeing uniqueness when several
	// IDs are allocated within the same wall-clock second or across a backward
	// clock jump (NTP step, offset update).
	lastID atomic.Int64
}

// NewMsgIDGenerator returns a generator seeded with the given server time,
// recorded as an offset from the local clock. A time.Now() seed yields a zero
// offset, which is correct when the local clock is in sync.
func NewMsgIDGenerator(serverTime time.Time) *MsgIDGenerator {
	g := &MsgIDGenerator{}
	g.timeOffset.Store(serverTime.Unix() - time.Now().Unix())
	return g
}

// UpdateServerTime records the server's current time as an offset from the
// local clock. Because every allocation recomputes the timestamp from the live
// clock, one correction fixes all subsequent sends — there is no per-burst
// correction loop. The monotonicity floor is reset so a corrected (e.g. lower,
// code-17) offset is not pinned to stale, too-high IDs; this mirrors mtcute,
// which zeroes _lastMessageId on every time-offset update.
func (g *MsgIDGenerator) UpdateServerTime(serverT time.Time) {
	g.timeOffset.Store(serverT.Unix() - time.Now().Unix())
	g.lastID.Store(0)
}

// AdvanceOffset monotonically advances the server-time offset if serverT implies
// a larger offset than currently stored; a serverT that would lower the offset
// is ignored. This is the continuous, passive counterpart to UpdateServerTime:
// the high 32 bits of every server-originated msg_id encode the server's unix
// time, so calling this on each inbound message keeps the offset accurate for
// the life of the session without waiting for a bad_msg_notification (the
// pattern tdlib uses in its check_packet path).
//
// It does not reset the monotonicity floor: because the offset only ever
// increases, subsequent IDs can only move forward, so lastID stays consistent.
func (g *MsgIDGenerator) AdvanceOffset(serverT time.Time) {
	newOffset := serverT.Unix() - time.Now().Unix()
	for {
		cur := g.timeOffset.Load()
		if newOffset <= cur {
			return
		}
		if g.timeOffset.CompareAndSwap(cur, newOffset) {
			return
		}
	}
}

// Next returns the next unique, monotonically increasing message ID. The
// timestamp portion is recomputed from the live wall clock on every call.
func (g *MsgIDGenerator) Next() int64 {
	serverSecond := time.Now().Unix() + g.timeOffset.Load()
	candidate := serverSecond << 32
	for {
		last := g.lastID.Load()
		if candidate <= last {
			// Same clock second or backward jump: bump past the last ID,
			// preserving the client-side divisibility-by-4 invariant.
			candidate = last + 4
		}
		if g.lastID.CompareAndSwap(last, candidate) {
			return candidate
		}
	}
}
