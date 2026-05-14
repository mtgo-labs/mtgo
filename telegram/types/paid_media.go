package types

import "github.com/mtgo-labs/mtgo/tg"

type PaidMediaInfo struct {
	StarsAmount int64
	MediaCount  int32
}

type PaidReactor struct {
	UserID int64
	Stars  int64
}

type MyBoost struct {
	ID       int64
	Date     int32
	Expires  int32
	ChatID   int64
	Slot     int32
	Cooldown int32
}

func ParseMyBoost(raw *tg.MyBoost) *MyBoost {
	if raw == nil {
		return nil
	}
	b := &MyBoost{
		Date:    raw.Date,
		Expires: raw.Expires,
		Slot:    raw.Slot,
	}
	if raw.Peer != nil {
		b.ChatID = peerToChatID(raw.Peer)
	}
	if raw.CooldownUntilDate != 0 {
		b.Cooldown = raw.CooldownUntilDate
	}
	return b
}

func peerToChatID(peer tg.PeerClass) int64 {
	if peer == nil {
		return 0
	}
	switch p := peer.(type) {
	case *tg.PeerChat:
		return -p.ChatID
	case *tg.PeerChannel:
		return -p.ChannelID
	}
	return 0
}
