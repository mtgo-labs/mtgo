package tg

import (
	"bytes"
	"io"
)

// MsgContainerID is the TL constructor ID for the msg_container type.
const MsgContainerID = 0x73F1F8DC

// MsgContainer represents a TL message container holding multiple messages.
type MsgContainer struct {
	// Messages is the slice of messages in this container.
	Messages []*MTProtoMessage
}

// ConstructorID returns the TL constructor ID for MsgContainer.
func (c *MsgContainer) ConstructorID() uint32 { return MsgContainerID }

// Encode writes the container and all its messages to b in TL binary format.
func (c *MsgContainer) Encode(b *bytes.Buffer) error {
	WriteInt(b, uint32(len(c.Messages)))
	for _, msg := range c.Messages {
		if err := msg.Encode(b); err != nil {
			return err
		}
	}
	return nil
}

// DecodeMsgContainer reads a MsgContainer from r, decoding each contained message.
func DecodeMsgContainer(r io.Reader) (*MsgContainer, error) {
	count := ReadInt(r)
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
	Registry[MsgContainerID] = func(r io.Reader) (TLObject, error) {
		return DecodeMsgContainer(r)
	}
}
