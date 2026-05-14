package telegram

import (
	"fmt"
	"strings"

	"github.com/mtgo-labs/mtgo/tg"
)

func extractUpdateMeta(update tg.UpdateClass) updateMeta {
	meta := updateMeta{Key: fmt.Sprintf("type:%08x", update.ConstructorID())}
	switch u := update.(type) {
	case *tg.UpdateNewMessage:
		meta.Pts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = fmt.Sprintf("new-message:%08x:%d:%d", u.ConstructorID(), u.PTS, messageID(u.Message))
	case *tg.UpdateDeleteMessages:
		meta.Pts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = fmt.Sprintf("delete-messages:%d:%s", u.PTS, int32ListKey(u.Messages))
	case *tg.UpdateEditMessage:
		meta.Pts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = fmt.Sprintf("edit-message:%d:%d", u.PTS, messageID(u.Message))
	case *tg.UpdateNewChannelMessage:
		meta.IsChannel = true
		meta.ChannelID = channelIDFromMessage(u.Message)
		meta.ChannelPts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = fmt.Sprintf("channel-new:%d:%d:%d", meta.ChannelID, u.PTS, messageID(u.Message))
	case *tg.UpdateEditChannelMessage:
		meta.IsChannel = true
		meta.ChannelID = channelIDFromMessage(u.Message)
		meta.ChannelPts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = fmt.Sprintf("channel-edit:%d:%d:%d", meta.ChannelID, u.PTS, messageID(u.Message))
	case *tg.UpdateDeleteChannelMessages:
		meta.IsChannel = true
		meta.ChannelID = u.ChannelID
		meta.ChannelPts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = fmt.Sprintf("channel-delete:%d:%d:%s", u.ChannelID, u.PTS, int32ListKey(u.Messages))
	case *tg.UpdateChannelWebPage:
		meta.IsChannel = true
		meta.ChannelID = u.ChannelID
		meta.ChannelPts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = fmt.Sprintf("channel-webpage:%d:%d", u.ChannelID, u.PTS)
	case *tg.UpdatePinnedChannelMessages:
		meta.IsChannel = true
		meta.ChannelID = u.ChannelID
		meta.ChannelPts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = fmt.Sprintf("channel-pinned:%d:%d:%s", u.ChannelID, u.PTS, int32ListKey(u.Messages))
	case *tg.UpdateChannelTooLong:
		meta.IsChannel = true
		meta.ChannelID = u.ChannelID
		if u.PTS != 0 {
			meta.ChannelPts = u.PTS
		}
		meta.Key = fmt.Sprintf("channel-too-long:%d", u.ChannelID)
	case *tg.UpdateWebPage:
		meta.Pts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = fmt.Sprintf("webpage:%d", u.PTS)
	case *tg.UpdatePinnedMessages:
		meta.Pts, meta.PtsCount = u.PTS, u.PTSCount
		meta.Key = fmt.Sprintf("pinned:%d:%s", u.PTS, int32ListKey(u.Messages))
	case *tg.UpdateNewEncryptedMessage:
		meta.Qts = u.Qts
		meta.Key = fmt.Sprintf("encrypted:%d", u.Qts)
	}
	return meta
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
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprint(value)
	}
	return strings.Join(parts, ",")
}
