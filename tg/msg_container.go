package tg

import (
	"bytes"
)

const MsgContainerID = 0x73F1F8DC

type MsgContainer struct {
	Messages []*MTProtoMessage
}

func (c *MsgContainer) ConstructorID() uint32 { return MsgContainerID }

func (c *MsgContainer) Encode(b *bytes.Buffer) error {
	WriteInt(b, uint32(len(c.Messages)))
	for _, msg := range c.Messages {
		if err := msg.Encode(b); err != nil {
			return err
		}
	}
	return nil
}

func DecodeMsgContainer(r *Reader) (*MsgContainer, error) {
	count, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	if err := CheckVectorCount(count); err != nil {
		return nil, err
	}
	c := &MsgContainer{Messages: make([]*MTProtoMessage, count)}
	for i := uint32(0); i < count; i++ {
		msg, err := DecodeMTProtoMessage(r)
		if err != nil {
			return nil, err
		}
		c.Messages[i] = msg
	}
	return c, nil
}

func init() {
	Registry[MsgContainerID] = func(r *Reader) (TLObject, error) {
		return DecodeMsgContainer(r)
	}
}
