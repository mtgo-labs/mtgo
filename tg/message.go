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
	msg := &MTProtoMessage{
		MsgID: msgID,
		SeqNo: seqNo,
		Body:  obj,
	}
	return msg, nil
}

func init() {
	Registry[MTProtoMessageID] = func(r *Reader) (TLObject, error) {
		return DecodeMTProtoMessage(r)
	}
}
