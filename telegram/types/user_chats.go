package types

import (
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// VerificationStatus describes whether a user or chat is verified, flagged as
// scam or fake, and any bot verification icon.
type VerificationStatus struct {
	IsVerified                       bool
	IsScam                           bool
	IsFake                           bool
	BotVerificationIconCustomEmojiID string
}

// NewVerificationStatus creates a VerificationStatus from verification flags and
// an optional bot verification icon document ID.
//
// Example:
//
//	status := types.NewVerificationStatus(true, false, false, 12345)
//	fmt.Printf("Verified: %v\n", status.IsVerified)
func NewVerificationStatus(verified, scam, fake bool, botVerificationIcon int64) *VerificationStatus {
	v := &VerificationStatus{
		IsVerified: verified,
		IsScam:     scam,
		IsFake:     fake,
	}
	if botVerificationIcon != 0 {
		v.BotVerificationIconCustomEmojiID = fmt.Sprintf("%d", botVerificationIcon)
	}
	return v
}

// BotVerification represents a verification badge applied by a bot, including
// the bot user, custom emoji icon, and description.
type BotVerification struct {
	Bot           *User
	CustomEmojiID string
	Description   string
}

// ParseBotVerification converts a TL BotVerification into a BotVerification,
// resolving the bot user. Returns nil if raw is nil.
//
// Example:
//
//	bv := types.ParseBotVerification(raw, users)
//	fmt.Printf("Verified by: %s\n", bv.Bot.FirstName)
func ParseBotVerification(raw *tg.BotVerification, users map[int64]tg.UserClass) *BotVerification {
	if raw == nil {
		return nil
	}
	bv := &BotVerification{
		Bot:           getUser(users, raw.BotID),
		CustomEmojiID: fmt.Sprintf("%d", raw.Icon),
		Description:   raw.Description,
	}
	return bv
}

// AcceptedGiftTypes describes which categories of gifts a user accepts.
type AcceptedGiftTypes struct {
	UnlimitedGifts      bool
	LimitedGifts        bool
	UpgradedGifts       bool
	GiftsFromChannels   bool
	PremiumSubscription bool
}

// ParseAcceptedGiftTypes converts a TL DisallowedGiftsSettings into an AcceptedGiftTypes,
// inverting the disallowed flags. Returns nil if raw is nil.
//
// Example:
//
//	giftTypes := types.ParseAcceptedGiftTypes(rawSettings)
//	fmt.Printf("Accepts limited gifts: %v\n", giftTypes.LimitedGifts)
func ParseAcceptedGiftTypes(raw *tg.DisallowedGiftsSettings) *AcceptedGiftTypes {
	if raw == nil {
		return nil
	}
	return &AcceptedGiftTypes{
		UnlimitedGifts:      !raw.DisallowUnlimitedStargifts,
		LimitedGifts:        !raw.DisallowLimitedStargifts,
		UpgradedGifts:       !raw.DisallowUniqueStargifts,
		PremiumSubscription: !raw.DisallowPremiumGifts,
		GiftsFromChannels:   !raw.DisallowStargiftsFromChannels,
	}
}

// ChatAdminWithInviteLinks represents an admin together with their invite link
// statistics for a chat.
type ChatAdminWithInviteLinks struct {
	Admin                       *User
	ChatInviteLinksCount        int32
	RevokedChatInviteLinksCount int32
}

// ParseChatAdminWithInviteLinks converts a TL ChatAdminWithInvites into a
// ChatAdminWithInviteLinks. Returns nil if raw is nil.
//
// Example:
//
//	admin := types.ParseChatAdminWithInviteLinks(raw, users)
//	fmt.Printf("Admin %s created %d links\n", admin.Admin.FirstName, admin.ChatInviteLinksCount)
func ParseChatAdminWithInviteLinks(raw *tg.ChatAdminWithInvites, users map[int64]tg.UserClass) *ChatAdminWithInviteLinks {
	if raw == nil {
		return nil
	}
	return &ChatAdminWithInviteLinks{
		Admin:                       getUser(users, raw.AdminID),
		ChatInviteLinksCount:        raw.InvitesCount,
		RevokedChatInviteLinksCount: raw.RevokedInvitesCount,
	}
}

// FailedToAddMember represents a user who could not be added to a chat,
// with flags indicating whether Premium would have allowed the addition.
type FailedToAddMember struct {
	UserID                        int64
	PremiumWouldAllowInvite       bool
	PremiumRequiredToSendMessages bool
}

// ParseFailedToAddMember converts a TL MissingInvitee into a FailedToAddMember.
// Returns nil if raw is nil.
//
// Example:
//
//	failed := types.ParseFailedToAddMember(raw)
//	fmt.Printf("Failed to add user %d, premium would help: %v\n", failed.UserID, failed.PremiumWouldAllowInvite)
func ParseFailedToAddMember(raw *tg.MissingInvitee) *FailedToAddMember {
	if raw == nil {
		return nil
	}
	return &FailedToAddMember{
		UserID:                        raw.UserID,
		PremiumWouldAllowInvite:       raw.PremiumWouldAllowInvite,
		PremiumRequiredToSendMessages: raw.PremiumRequiredForPm,
	}
}

// FoundContacts holds the results of a user search, split into personal (my)
// and global results.
type FoundContacts struct {
	MyResults     []*Chat
	GlobalResults []*Chat
}

// UserRating represents a user's star rating with level progression info.
type UserRating struct {
	Level                 int32
	IsMaximumLevelReached bool
	Rating                int64
	CurrentLevelRating    int64
	NextLevelRating       int64
}

// ParseUserRating converts a TL StarsRating into a UserRating with level and
// star counts. Returns nil if raw is nil.
//
// Example:
//
//	rating := types.ParseUserRating(raw)
//	fmt.Printf("Level %d, %d/%d stars\n", rating.Level, rating.CurrentLevelRating, rating.NextLevelRating)
func ParseUserRating(raw *tg.StarsRating) *UserRating {
	if raw == nil {
		return nil
	}
	r := &UserRating{
		Level:              raw.Level,
		Rating:             raw.Stars,
		CurrentLevelRating: raw.CurrentLevelStars,
	}
	if raw.NextLevelStars != 0 {
		r.NextLevelRating = raw.NextLevelStars
	} else {
		r.IsMaximumLevelReached = true
	}
	return r
}

// HistoryCleared is a sentinel type indicating that chat history was cleared.
type HistoryCleared struct{}

// ChatFolderInfo holds the basic identity of a chat folder.
type ChatFolderInfo struct {
	ID       *int32
	Title    string
	Emoticon string
}

// ChatFolderInviteLinkInfo represents the result of resolving a chat folder
// invite link, containing the folder info and lists of missing and already-added chats.
type ChatFolderInviteLinkInfo struct {
	ChatFolderInfo *ChatFolderInfo
	MissingChats   []*Chat
	AddedChats     []*Chat
}

// ParseChatFolderInviteLinkInfo converts a TL ChatlistInviteClass into a
// ChatFolderInviteLinkInfo, resolving peer lists into Chat objects.
// Returns nil if raw is nil.
//
// Example:
//
//	info := types.ParseChatFolderInviteLinkInfo(rawInvite)
//	fmt.Printf("Folder: %s, missing: %d chats\n", info.ChatFolderInfo.Title, len(info.MissingChats))
func ParseChatFolderInviteLinkInfo(raw tg.ChatlistInviteClass) *ChatFolderInviteLinkInfo {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.ChatlistsChatlistInvite:
		pm := NewPeerMapFromClasses(v.Users, v.Chats)
		return &ChatFolderInviteLinkInfo{
			ChatFolderInfo: &ChatFolderInfo{
				Title:    textWithEntitiesText(v.Title),
				Emoticon: v.Emoticon,
			},
			MissingChats: parseChatsFromPeers(v.Peers, pm),
		}
	case *tg.ChatlistsChatlistInviteAlready:
		pm := NewPeerMapFromClasses(v.Users, v.Chats)
		filterID := v.FilterID
		return &ChatFolderInviteLinkInfo{
			ChatFolderInfo: &ChatFolderInfo{ID: &filterID},
			MissingChats:   parseChatsFromPeers(v.MissingPeers, pm),
			AddedChats:     parseChatsFromPeers(v.AlreadyPeers, pm),
		}
	default:
		return nil
	}
}

func parseChatsFromPeers(peers []tg.PeerClass, pm *PeerMap) []*Chat {
	if len(peers) == 0 {
		return nil
	}
	chats := make([]*Chat, 0, len(peers))
	for _, peer := range peers {
		if chat := ParseChatFromPeer(peer, pm); chat != nil {
			chats = append(chats, chat)
		}
	}
	return chats
}

func textWithEntitiesText(raw *tg.TextWithEntities) string {
	if raw == nil {
		return ""
	}
	return raw.Text
}

// ChatJoiner represents a user who joined a chat with their join date, bio,
// pending status, and approver information.
type ChatJoiner struct {
	User       *User
	Date       time.Time
	Bio        string
	Pending    bool
	ApprovedBy *User
}

// ParseChatJoiner converts a TL ChatParticipantClass into a ChatJoiner.
// Returns nil if raw is nil.
//
// Example:
//
//	joiner := types.ParseChatJoiner(rawParticipant, users)
//	fmt.Printf("Joined: %s at %s\n", joiner.User.FirstName, joiner.Date)
func ParseChatJoiner(raw tg.ChatParticipantClass, users map[int64]tg.UserClass) *ChatJoiner {
	if raw == nil {
		return nil
	}
	j := &ChatJoiner{}
	switch p := raw.(type) {
	case *tg.ChatParticipant:
		j.User = getUser(users, p.UserID)
		j.Date = time.Unix(int64(p.Date), 0)
		j.ApprovedBy = getUser(users, p.InviterID)
	case *tg.ChatParticipantAdmin:
		j.User = getUser(users, p.UserID)
		j.Date = time.Unix(int64(p.Date), 0)
		j.ApprovedBy = getUser(users, p.InviterID)
	default:
		return nil
	}
	return j
}

// ParseChannelJoiner converts a TL ChannelParticipantClass into a ChatJoiner.
// Returns nil if raw is nil.
//
// Example:
//
//	joiner := types.ParseChannelJoiner(rawParticipant, users)
//	if joiner.Pending {
//	    fmt.Printf("Join request pending from %s\n", joiner.User.FirstName)
//	}
func ParseChannelJoiner(raw tg.ChannelParticipantClass, users map[int64]tg.UserClass) *ChatJoiner {
	if raw == nil {
		return nil
	}
	j := &ChatJoiner{}
	switch p := raw.(type) {
	case *tg.ChannelParticipant:
		j.User = getUser(users, p.UserID)
		j.Date = time.Unix(int64(p.Date), 0)
	case *tg.ChannelParticipantSelf:
		j.User = getUser(users, p.UserID)
		j.Date = time.Unix(int64(p.Date), 0)
		j.Pending = p.ViaRequest
		j.ApprovedBy = getUser(users, p.InviterID)
	case *tg.ChannelParticipantAdmin:
		j.User = getUser(users, p.UserID)
		j.Date = time.Unix(int64(p.Date), 0)
		j.ApprovedBy = getUser(users, p.PromotedBy)
	default:
		return nil
	}
	return j
}
