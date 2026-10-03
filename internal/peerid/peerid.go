// Package peerid is the single source of truth for mtgo's marked chat-ID
// convention.
//
// Telegram assigns raw IDs in three overlapping spaces: users, basic groups,
// and channels/supergroups all use positive int64 values. To make IDs
// self-describing in logs, storage keys, and Bot-API-style surfaces, mtgo marks
// them:
//
//   - users:         +id (unchanged)
//   - basic groups:  -id
//   - channels:      ChannelPrefix - id
//
// All conversions between raw and marked forms, and from tg.PeerClass values,
// MUST go through this package. Re-deriving the arithmetic at call sites has
// historically produced five divergent copies.
package peerid

import "github.com/mtgo-labs/mtgo/tg"

// ChannelPrefix is the offset applied to raw channel/supergroup IDs to produce
// their marked chat ID: MarkChannel(id) == ChannelPrefix - id.
const ChannelPrefix int64 = -1_000_000_000_000

// MarkChannel returns the marked chat ID of a raw channel/supergroup ID.
func MarkChannel(channelID int64) int64 {
	return ChannelPrefix - channelID
}

// MarkChat returns the marked chat ID of a raw basic-group ID.
func MarkChat(chatID int64) int64 {
	return -chatID
}

// UnmarkChannel returns the raw channel ID behind a marked chat ID. The second
// return value reports whether id is in the marked channel range at all.
func UnmarkChannel(id int64) (int64, bool) {
	if IsMarkedChannel(id) {
		return ChannelPrefix - id, true
	}
	return 0, false
}

// IsMarkedChannel reports whether id is in the marked channel/supergroup range.
func IsMarkedChannel(id int64) bool {
	return id <= ChannelPrefix
}

// MarkFromPeer returns the marked chat ID of a tg.PeerClass value: positive
// for users, negated for basic groups, channel-prefixed for channels. It
// returns 0 for nil or unrecognized peers.
func MarkFromPeer(p tg.PeerClass) int64 {
	switch v := p.(type) {
	case *tg.PeerUser:
		return v.UserID
	case *tg.PeerChat:
		return MarkChat(v.ChatID)
	case *tg.PeerChannel:
		return MarkChannel(v.ChannelID)
	default:
		return 0
	}
}

// RawFromPeer returns the raw positive ID carried by a tg.PeerClass value,
// regardless of peer kind. The second return value reports whether the peer
// kind was recognized.
func RawFromPeer(p tg.PeerClass) (int64, bool) {
	switch v := p.(type) {
	case *tg.PeerUser:
		return v.UserID, true
	case *tg.PeerChat:
		return v.ChatID, true
	case *tg.PeerChannel:
		return v.ChannelID, true
	default:
		return 0, false
	}
}
