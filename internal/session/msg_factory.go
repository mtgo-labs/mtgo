package session

import (
	"time"
)

// MsgFactory combines a MsgIDGenerator and a SeqNoGenerator to provide a
// single entry point for allocating message identifiers and sequence numbers.
type MsgFactory struct {
	msgIDGen *MsgIDGenerator
	seqNoGen *SeqNoGenerator
}

// NewMsgFactory returns a MsgFactory initialized with the given server time.
func NewMsgFactory(serverTime time.Time) *MsgFactory {
	return &MsgFactory{
		msgIDGen: NewMsgIDGenerator(serverTime),
		seqNoGen: NewSeqNoGenerator(),
	}
}

// AllocateMsgID returns the next unique MTProto message ID.
func (f *MsgFactory) AllocateMsgID() int64 {
	return f.msgIDGen.Next()
}

// AllocateSeqNo returns the next sequence number. When contentRelated is true
// the counter is advanced; otherwise the current value is returned.
func (f *MsgFactory) AllocateSeqNo(contentRelated bool) int32 {
	return f.seqNoGen.Next(contentRelated)
}

// UpdateServerTime forwards the updated server time to the underlying message
// ID generator.
func (f *MsgFactory) UpdateServerTime(t time.Time) {
	f.msgIDGen.UpdateServerTime(t)
}

// ForceResetServerTime forwards a forced (possibly backward) server-time reset
// to the underlying message ID generator. See MsgIDGenerator.ForceResetServerTime.
func (f *MsgFactory) ForceResetServerTime(t time.Time) {
	f.msgIDGen.ForceResetServerTime(t)
}
