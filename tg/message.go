package tg

import (
	"bytes"
	"fmt"
	"io"
)

// MTProtoMessageID is the TL constructor ID for the MTProto message type.
const MTProtoMessageID = 0x5BB8E511

// MTProtoMessage represents a single MTProto message with its ID, sequence number,
// and serialized body.
type MTProtoMessage struct {
	MsgID int64
	SeqNo uint32
	Body  TLObject
}

// ConstructorID returns the TL constructor ID for MTProtoMessage.
func (m *MTProtoMessage) ConstructorID() uint32 { return MTProtoMessageID }

// Encode writes the message in TL binary format to b.
func (m *MTProtoMessage) Encode(b *bytes.Buffer) error {
	WriteLong(b, m.MsgID)
	WriteInt(b, m.SeqNo)
	var bodyBuf bytes.Buffer
	if err := EncodeTLObject(&bodyBuf, m.Body); err != nil {
		return err
	}
	bodyBytes := bodyBuf.Bytes()
	WriteInt(b, uint32(len(bodyBytes)))
	b.Write(bodyBytes)
	return nil
}

// DecodeMTProtoMessage reads a single MTProtoMessage from r, decoding its body as a TLObject.
func DecodeMTProtoMessage(r io.Reader) (*MTProtoMessage, error) {
	msg := &MTProtoMessage{
		MsgID: ReadLong(r),
		SeqNo: ReadInt(r),
	}
	length := ReadInt(r)
	if length > 1<<20 {
		return nil, fmt.Errorf("message body too large: %d bytes", length)
	}
	lr := io.LimitReader(r, int64(length))
	obj, err := ReadTLObject(lr)
	if err != nil {
		_, _ = io.Copy(io.Discard, lr)
		return nil, err
	}
	_, _ = io.Copy(io.Discard, lr)
	msg.Body = obj
	return msg, nil
}

func init() {
	Registry[MTProtoMessageID] = func(r io.Reader) (TLObject, error) {
		return DecodeMTProtoMessage(r)
	}
}
