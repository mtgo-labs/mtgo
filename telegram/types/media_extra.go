package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// LivePhoto represents a live photo (short video attached to a photo).
type LivePhoto struct {
	ID         int64
	AccessHash int64
	Date       int32
	Sizes      []PhotoSize
	VideoSizes []VideoSize
}

// VideoSize represents a video size variant.
type VideoSize struct {
	Type   string
	Width  int32
	Height int32
	Size   int32
}

// Thumbnail represents a photo or file thumbnail (wraps PhotoSize).
type Thumbnail struct {
	Type   string
	Width  int32
	Height int32
	Size   int32
}

func ParseThumbnail(raw tg.PhotoSizeClass) *Thumbnail {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.PhotoSize:
		return &Thumbnail{Type: v.Type, Width: v.W, Height: v.H, Size: v.Size}
	case *tg.PhotoSizeProgressive:
		sz := int32(0)
		if len(v.Sizes) > 0 {
			sz = v.Sizes[len(v.Sizes)-1]
		}
		return &Thumbnail{Type: v.Type, Width: v.W, Height: v.H, Size: sz}
	case *tg.PhotoCachedSize:
		return &Thumbnail{Type: v.Type, Width: v.W, Height: v.H}
	}
	return nil
}

// StrippedThumbnail represents a stripped inline thumbnail.
type StrippedThumbnail struct {
	Type   string
	Inline []byte
}

func ParseStrippedThumbnail(raw *tg.PhotoStrippedSize) *StrippedThumbnail {
	if raw == nil {
		return nil
	}
	return &StrippedThumbnail{Type: raw.Type, Inline: raw.Bytes}
}

// AvailableEffect describes an available message effect.
type AvailableEffect struct {
	ID                int64
	Emoticon          string
	StaticIconID      int64
	EffectStickerID   int64
	EffectAnimationID int64
	PremiumRequired   bool
}

func ParseAvailableEffect(raw *tg.AvailableEffect) *AvailableEffect {
	if raw == nil {
		return nil
	}
	e := &AvailableEffect{
		ID:              raw.ID,
		Emoticon:        raw.Emoticon,
		EffectStickerID: raw.EffectStickerID,
		PremiumRequired: raw.PremiumRequired,
	}
	if raw.StaticIconID != 0 {
		e.StaticIconID = raw.StaticIconID
	}
	if raw.EffectAnimationID != 0 {
		e.EffectAnimationID = raw.EffectAnimationID
	}
	return e
}

// FormattedText contains text with formatting entities.
type FormattedText struct {
	Text     string
	Entities []*MessageEntity
}

// MessageOrigin describes the origin of a forwarded/replied-to message.
type MessageOrigin struct {
	Type        MessageOriginType
	Date        int32
	UserID      int64
	ChatID      int64
	ChannelPost int32
	Author      string
	Imported    bool
}

// ExternalReplyInfo contains info about a message being replied to from another chat.
type ExternalReplyInfo struct {
	ReplyToMsgID  int32
	ReplyToPeerID int64
	ForumTopic    bool
	Quote         bool
	QuoteText     string
}

func ParseExternalReplyInfo(raw *tg.MessageReplyHeader) *ExternalReplyInfo {
	if raw == nil {
		return nil
	}
	info := &ExternalReplyInfo{
		ForumTopic: raw.ForumTopic,
		Quote:      raw.Quote,
	}
	if raw.ReplyToMsgID != 0 {
		info.ReplyToMsgID = raw.ReplyToMsgID
	}
	if raw.ReplyToPeerID != nil {
		switch p := raw.ReplyToPeerID.(type) {
		case *tg.PeerUser:
			info.ReplyToPeerID = p.UserID
		case *tg.PeerChat:
			info.ReplyToPeerID = p.ChatID
		case *tg.PeerChannel:
			info.ReplyToPeerID = p.ChannelID
		}
	}
	if raw.QuoteText != "" {
		info.QuoteText = raw.QuoteText
	}
	return info
}

// StarAmount describes a possibly non-integer amount of Telegram Stars.
type StarAmount struct {
	Amount int64
	Nanos  int32
}

func ParseStarAmount(raw tg.StarsAmountClass) *StarAmount {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.StarsAmount:
		return &StarAmount{Amount: v.Amount, Nanos: v.Nanos}
	}
	return nil
}

// StoryView contains information about a story view.
type StoryView struct {
	UserID               int64
	Date                 int32
	Blocked              bool
	BlockedMyStoriesFrom bool
	ReactionEmoji        string
}

func ParseStoryView(raw tg.StoryViewClass) *StoryView {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.StoryView:
		sv := &StoryView{
			UserID:               v.UserID,
			Date:                 v.Date,
			Blocked:              v.Blocked,
			BlockedMyStoriesFrom: v.BlockedMyStoriesFrom,
		}
		if v.Reaction != nil {
			if r, ok := v.Reaction.(*tg.ReactionEmoji); ok {
				sv.ReactionEmoji = r.Emoticon
			}
		}
		return sv
	}
	return nil
}

// ChatBoost contains info about boosts applied by a user.
type ChatBoost struct {
	ID         string
	UserID     int64
	Date       int32
	Expires    int32
	Multiplier int32
	Gift       bool
	Giveaway   bool
	Unclaimed  bool
}

// BoostsStatus contains info about boost status of a chat.
type BoostsStatus struct {
	Level                     int32
	CurrentLevelBoosts        int32
	Boosts                    int32
	NextLevelBoosts           int32
	GiftBoosts                int32
	MyBoost                   bool
	BoostURL                  string
	MyBoostSlots              []int32
	PremiumAudiencePercentage float64
}

func ParseBoostsStatus(raw *tg.PremiumBoostsStatus) *BoostsStatus {
	if raw == nil {
		return nil
	}
	bs := &BoostsStatus{
		Level:              raw.Level,
		CurrentLevelBoosts: raw.CurrentLevelBoosts,
		Boosts:             raw.Boosts,
		MyBoost:            raw.MyBoost,
		BoostURL:           raw.BoostURL,
		MyBoostSlots:       raw.MyBoostSlots,
	}
	if raw.NextLevelBoosts != 0 {
		bs.NextLevelBoosts = raw.NextLevelBoosts
	}
	if raw.GiftBoosts != 0 {
		bs.GiftBoosts = raw.GiftBoosts
	}
	if raw.PremiumAudience != nil {
		bs.PremiumAudiencePercentage = raw.PremiumAudience.Part
	}
	return bs
}

// ChatReactions describes available reactions in a chat.
type ChatReactions struct {
	AllowAll    bool
	AllowCustom bool
	Reactions   []string
}

func ParseChatReactions(raw tg.ChatReactionsClass) *ChatReactions {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.ChatReactionsAll:
		return &ChatReactions{AllowAll: true, AllowCustom: v.AllowCustom}
	case *tg.ChatReactionsNone:
		return &ChatReactions{}
	case *tg.ChatReactionsSome:
		var reactions []string
		for _, r := range v.Reactions {
			if e, ok := r.(*tg.ReactionEmoji); ok {
				reactions = append(reactions, e.Emoticon)
			}
		}
		return &ChatReactions{Reactions: reactions}
	}
	return nil
}

// GroupCallMember contains information about a group call participant.
type GroupCallMember struct {
	PeerID          int64
	Date            int32
	ActiveDate      int32
	Source          int32
	Volume          int32
	About           string
	Muted           bool
	MutedByYou      bool
	Left            bool
	CanSelfUnmute   bool
	JustJoined      bool
	VideoJoined     bool
	Self            bool
	RaiseHandRating int64
}

func ParseGroupCallMember(raw *tg.GroupCallParticipant) *GroupCallMember {
	if raw == nil {
		return nil
	}
	m := &GroupCallMember{
		Date:          raw.Date,
		Source:        raw.Source,
		Muted:         raw.Muted,
		MutedByYou:    raw.MutedByYou,
		Left:          raw.Left,
		CanSelfUnmute: raw.CanSelfUnmute,
		JustJoined:    raw.JustJoined,
		VideoJoined:   raw.VideoJoined,
		Self:          raw.Self,
	}
	if raw.Peer != nil {
		switch p := raw.Peer.(type) {
		case *tg.PeerUser:
			m.PeerID = p.UserID
		case *tg.PeerChat:
			m.PeerID = p.ChatID
		case *tg.PeerChannel:
			m.PeerID = p.ChannelID
		}
	}
	if raw.ActiveDate != 0 {
		m.ActiveDate = raw.ActiveDate
	}
	if raw.Volume != 0 {
		m.Volume = raw.Volume
	}
	if raw.About != "" {
		m.About = raw.About
	}
	if raw.RaiseHandRating != 0 {
		m.RaiseHandRating = raw.RaiseHandRating
	}
	return m
}

// ChatBackground describes a background set for a specific chat.
type ChatBackground struct {
	ID             int64
	WallpaperDocID int64
}

// ChatTheme describes a chat theme.
type ChatTheme struct {
	Emoticon string
}

func ParseChatTheme(raw *tg.ChatTheme) *ChatTheme {
	if raw == nil {
		return nil
	}
	return &ChatTheme{
		Emoticon: raw.Emoticon,
	}
}
