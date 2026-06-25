package telegram

import (
	"strconv"
	"strings"

	"github.com/mtgo-labs/mtgo/tg"
)

func extractUpdateMeta(update tg.UpdateClass) updateMeta {
	meta := updateMeta{Key: "type:" + formatHex32(update.ConstructorID())}
	switch u := update.(type) {
	case *tg.UpdateNewMessage:
		meta.Pts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = buildKey("new-message:", formatHex32(u.ConstructorID()), ":", strconv.FormatInt(int64(u.PTS), 10), ":", strconv.FormatInt(int64(messageID(u.Message)), 10))
	case *tg.UpdateDeleteMessages:
		meta.Pts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = buildKey("delete-messages:", strconv.FormatInt(int64(u.PTS), 10), ":", int32ListKey(u.Messages))
	case *tg.UpdateEditMessage:
		meta.Pts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = buildKey("edit-message:", strconv.FormatInt(int64(u.PTS), 10), ":", strconv.FormatInt(int64(messageID(u.Message)), 10))
	case *tg.UpdateNewChannelMessage:
		meta.IsChannel = true
		meta.ChannelID = channelIDFromMessage(u.Message)
		meta.ChannelPts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = buildKey("channel-new:", strconv.FormatInt(meta.ChannelID, 10), ":", strconv.FormatInt(int64(u.PTS), 10), ":", strconv.FormatInt(int64(messageID(u.Message)), 10))
	case *tg.UpdateEditChannelMessage:
		meta.IsChannel = true
		meta.ChannelID = channelIDFromMessage(u.Message)
		meta.ChannelPts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = buildKey("channel-edit:", strconv.FormatInt(meta.ChannelID, 10), ":", strconv.FormatInt(int64(u.PTS), 10), ":", strconv.FormatInt(int64(messageID(u.Message)), 10))
	case *tg.UpdateDeleteChannelMessages:
		meta.IsChannel = true
		meta.ChannelID = u.ChannelID
		meta.ChannelPts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = buildKey("channel-delete:", strconv.FormatInt(u.ChannelID, 10), ":", strconv.FormatInt(int64(u.PTS), 10), ":", int32ListKey(u.Messages))
	case *tg.UpdateChannelWebPage:
		meta.IsChannel = true
		meta.ChannelID = u.ChannelID
		meta.ChannelPts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = buildKey("channel-webpage:", strconv.FormatInt(u.ChannelID, 10), ":", strconv.FormatInt(int64(u.PTS), 10))
	case *tg.UpdatePinnedChannelMessages:
		meta.IsChannel = true
		meta.ChannelID = u.ChannelID
		meta.ChannelPts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = buildKey("channel-pinned:", strconv.FormatInt(u.ChannelID, 10), ":", strconv.FormatInt(int64(u.PTS), 10), ":", int32ListKey(u.Messages))
	case *tg.UpdateChannelTooLong:
		meta.IsChannel = true
		meta.ChannelID = u.ChannelID
		if u.PTS != 0 {
			meta.ChannelPts = u.PTS
		}
		meta.Key = buildKey("channel-too-long:", strconv.FormatInt(u.ChannelID, 10))
	case *tg.UpdateWebPage:
		meta.Pts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = buildKey("webpage:", strconv.FormatInt(int64(u.PTS), 10))
	case *tg.UpdatePinnedMessages:
		meta.Pts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = buildKey("pinned:", strconv.FormatInt(int64(u.PTS), 10), ":", int32ListKey(u.Messages))
	case *tg.UpdateNewEncryptedMessage:
		setQts(&meta, "encrypted:", u.Qts)
	// qts-bearing updates: these share Telegram's single monotonic qts sequence
	// (alongside encrypted messages) and must flow through qts gap-recovery +
	// dedup — otherwise a missed qts is never fetched via getDifference, and
	// worse, without a unique key they collapse onto the default "type:<id>" key
	// and get dedup-dropped (only the first of each type survives). Mirrors
	// TDLib, which tracks qts for all of these.
	case *tg.UpdateMessagePollVote:
		setQts(&meta, "poll-vote:", u.Qts)
	case *tg.UpdateChatParticipant:
		setQts(&meta, "chat-participant:", u.Qts)
	case *tg.UpdateChannelParticipant:
		setQts(&meta, "channel-participant:", u.Qts)
	case *tg.UpdateBotStopped:
		setQts(&meta, "bot-stopped:", u.Qts)
	case *tg.UpdateBotChatInviteRequester:
		setQts(&meta, "bot-invite-request:", u.Qts)
	case *tg.UpdateBotChatBoost:
		setQts(&meta, "bot-boost:", u.Qts)
	case *tg.UpdateBotMessageReaction:
		setQts(&meta, "bot-reaction:", u.Qts)
	case *tg.UpdateBotMessageReactions:
		setQts(&meta, "bot-reactions:", u.Qts)
	case *tg.UpdateBotBusinessConnect:
		setQts(&meta, "bot-business:", u.Qts)
	case *tg.UpdateBotNewBusinessMessage:
		setQts(&meta, "bot-business-new:", u.Qts)
	case *tg.UpdateBotEditBusinessMessage:
		setQts(&meta, "bot-business-edit:", u.Qts)
	case *tg.UpdateBotDeleteBusinessMessage:
		setQts(&meta, "bot-business-delete:", u.Qts)
	case *tg.UpdateBotPurchasedPaidMedia:
		setQts(&meta, "bot-paid-media:", u.Qts)
	case *tg.UpdateManagedBot:
		setQts(&meta, "managed-bot:", u.Qts)
	case *tg.UpdateBotGuestChatQuery:
		setQts(&meta, "bot-guest-chat:", u.Qts)
	// Bot query updates carry no qts (delivered via the common stream), so they
	// can't participate in qts gap-recovery — but they still need a unique dedup
	// key (their query_id) instead of the default type-id key, which would
	// otherwise collapse every distinct query into one and dedup-drop the rest.
	case *tg.UpdateBotCallbackQuery:
		meta.Key = buildKey("bot-callback:", strconv.FormatInt(u.QueryID, 10))
	case *tg.UpdateBotShippingQuery:
		meta.Key = buildKey("bot-shipping:", strconv.FormatInt(u.QueryID, 10))
	case *tg.UpdateBotPrecheckoutQuery:
		meta.Key = buildKey("bot-precheckout:", strconv.FormatInt(u.QueryID, 10))
	}
	return meta
}

// setQts records a qts-bearing update's monotonic qts (driving qts gap-recovery
// and dedup via classifyAccountUpdate) and a qts-unique dedup key.
func setQts(meta *updateMeta, prefix string, qts int32) {
	meta.Qts = qts
	meta.Key = buildKey(prefix, strconv.FormatInt(int64(qts), 10))
}

func messageID(msg tg.MessageClass) int32 {
	if msg == nil {
		return 0
	}
	switch v := msg.(type) {
	case *tg.Message:
		return v.ID
	case *tg.MessageEmpty:
		return v.ID
	case *tg.MessageService:
		return v.ID
	default:
		return 0
	}
}

func channelIDFromMessage(msg tg.MessageClass) int64 {
	if msg == nil {
		return 0
	}
	var peer tg.PeerClass
	switch v := msg.(type) {
	case *tg.Message:
		peer = v.PeerID
	case *tg.MessageEmpty:
		peer = v.PeerID
	case *tg.MessageService:
		peer = v.PeerID
	default:
		return 0
	}
	if peer == nil {
		return 0
	}
	if ch, ok := peer.(*tg.PeerChannel); ok {
		return ch.ChannelID
	}
	return 0
}

func int32ListKey(values []int32) string {
	if len(values) == 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(len(values) * 6)
	for i, v := range values {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatInt(int64(v), 10))
	}
	return b.String()
}

func buildKey(parts ...string) string {
	var n int
	for _, p := range parts {
		n += len(p)
	}
	var b strings.Builder
	b.Grow(n)
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}

func formatHex32(v uint32) string {
	const hexDigits = "0123456789abcdef"
	var buf [8]byte
	for i := 7; i >= 0; i-- {
		buf[i] = hexDigits[v&0xf]
		v >>= 4
	}
	return string(buf[:])
}
