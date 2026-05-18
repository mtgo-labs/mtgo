package tg

import (
	"bytes"
	"fmt"
)

const MTProtoMessageID = 0x5BB8E511

type MTProtoMessage struct {
	MsgID int64
	SeqNo uint32
	Body  TLObject
}

func (m *MTProtoMessage) ConstructorID() uint32 { return MTProtoMessageID }

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

func DecodeMTProtoMessage(r *Reader) (*MTProtoMessage, error) {
	msgID, err := r.ReadInt64()
	if err != nil {
		return nil, err
	}
	seqNo, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	length, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	if length > 1<<20 {
		return nil, fmt.Errorf("message body too large: %d bytes", length)
	}
	bodyData, err := r.ReadRawBytes(int(length))
	if err != nil {
		return nil, err
	}
	bodyReader := NewReader(bodyData)
	defer ReleaseReader(bodyReader)
	obj, err := ReadTLObject(bodyReader)
	if err != nil {
		return nil, err
	}
	return &MTProtoMessage{MsgID: msgID, SeqNo: seqNo, Body: obj}, nil
}

// MTProtoMessageRaw is like MTProtoMessage but holds the raw body bytes
// instead of a decoded TLObject, avoiding the expensive TL deserialization.
type MTProtoMessageRaw struct {
	MsgID   int64
	SeqNo   uint32
	BodyRaw []byte // raw TL bytes for the body (includes constructor ID prefix)
}

// DecodeMTProtoMessageRaw decodes only the MTProto envelope (msgID, seqNo,
// body length) and returns the raw body bytes without TL deserialization.
// Use this when the caller only needs raw bytes, e.g. for diff-based polling
// or InvokeWithRawByte where the full decode is unnecessary.
func DecodeMTProtoMessageRaw(r *Reader) (*MTProtoMessageRaw, error) {
	msgID, err := r.ReadInt64()
	if err != nil {
		return nil, err
	}
	seqNo, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	length, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	if length > 1<<20 {
		return nil, fmt.Errorf("message body too large: %d bytes", length)
	}
	bodyData, err := r.ReadRawBytes(int(length))
	if err != nil {
		return nil, err
	}
	return &MTProtoMessageRaw{MsgID: msgID, SeqNo: seqNo, BodyRaw: bodyData}, nil
}

func init() {
	Registry[MTProtoMessageID] = func(r *Reader) (TLObject, error) {
		return DecodeMTProtoMessage(r)
	}
}
