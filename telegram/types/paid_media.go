package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// PaidMediaInfo describes paid media attached to a message, including the star
// price and the unlocked media items.
//
// Example:
//
//	info := media.(*types.PaidMediaInfo)
//	fmt.Printf("Paid media: %d stars, %d items\n", info.StarsAmount, info.MediaCount)
type PaidMediaInfo struct {
	StarsAmount int64
	MediaCount  int32
	Media       []Media
}

// PaidReactor represents a user who reacted with a paid star reaction.
type PaidReactor struct {
	Sender      *Chat
	StarCount   int64
	IsTop       bool
	IsMe        bool
	IsAnonymous bool
}

// MyBoost represents a single boost slot applied by the current user to a chat.
//
// Example:
//
//	b := types.ParseMyBoost(rawBoost, peerMap)
//	fmt.Printf("Boost slot %d on %s, expires %s\n", b.Slot, b.Chat.Title, b.ExpireDate)
type MyBoost struct {
	Slot              int32
	Date              time.Time
	ExpireDate        time.Time
	Chat              *Chat
	CooldownUntilDate time.Time
}

// ParseMyBoost converts a TL MyBoost into a MyBoost, resolving the target chat.
// Returns nil if raw is nil.
//
// Example:
//
//	boost := types.ParseMyBoost(rawMyBoost, peerMap)
//	if boost != nil {
//	    fmt.Println("Boosted:", boost.Chat.Title)
//	}
func ParseMyBoost(raw *tg.MyBoost, pm *PeerMap) *MyBoost {
	if raw == nil {
		return nil
	}
	b := &MyBoost{
		Slot: raw.Slot,
		Date: time.Unix(int64(raw.Date), 0),
	}
	if raw.Expires != 0 {
		b.ExpireDate = time.Unix(int64(raw.Expires), 0)
	}
	if raw.CooldownUntilDate != 0 {
		b.CooldownUntilDate = time.Unix(int64(raw.CooldownUntilDate), 0)
	}
	if raw.Peer != nil {
		b.Chat = ParseChatFromPeer(raw.Peer, pm)
	}
	return b
}
