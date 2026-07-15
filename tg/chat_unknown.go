package tg

import (
	"bytes"
	"fmt"
)

// ChatUnknownTypeID is the constructor ID for a Chat variant observed in
// production (2026-07-15) but not yet documented in any public TL schema.
// The type name and fields are unknown; this stub captures the constructor
// as a bare Chat type. Update when upstream schema is published.
const ChatUnknownTypeID = 0x8f97c628

// ChatUnknown is a stub Chat type for an undocumented constructor. It
// satisfies ChatClass so that account.getAutoSaveSettings (and other RPCs
// that return Vector<Chat>) can decode successfully when the server sends
// this variant.
type ChatUnknown struct {
}

func (*ChatUnknown) isChat() {}

func (c *ChatUnknown) ConstructorID() uint32 { return ChatUnknownTypeID }

func (c *ChatUnknown) Encode(b *bytes.Buffer) error {
	WriteInt(b, ChatUnknownTypeID)
	return nil
}

func DecodeChatUnknown(r *Reader) (*ChatUnknown, error) {
	v := &ChatUnknown{}
	return v, nil
}

func (c *ChatUnknown) String() string {
	return fmt.Sprintf("chatUnknown#%08x", ChatUnknownTypeID)
}

func init() {
	Registry[ChatUnknownTypeID] = func(r *Reader) (TLObject, error) {
		return DecodeChatUnknown(r)
	}
}
