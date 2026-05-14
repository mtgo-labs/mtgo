package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

type VerificationStatus struct {
	BotID       int64
	Icon        int64
	Description string
}

func ParseVerificationStatus(raw *tg.BotVerification) *VerificationStatus {
	if raw == nil {
		return nil
	}
	return &VerificationStatus{
		BotID:       raw.BotID,
		Icon:        raw.Icon,
		Description: raw.Description,
	}
}

type AcceptedGiftTypes struct {
	UnlimitedStarGifts    bool
	LimitedStarGifts      bool
	UniqueStarGifts       bool
	PremiumGifts          bool
	StarGiftsFromChannels bool
}

func ParseAcceptedGiftTypes(raw *tg.DisallowedGiftsSettings) *AcceptedGiftTypes {
	if raw == nil {
		return nil
	}
	return &AcceptedGiftTypes{
		UnlimitedStarGifts:    !raw.DisallowUnlimitedStargifts,
		LimitedStarGifts:      !raw.DisallowLimitedStargifts,
		UniqueStarGifts:       !raw.DisallowUniqueStargifts,
		PremiumGifts:          !raw.DisallowPremiumGifts,
		StarGiftsFromChannels: !raw.DisallowStargiftsFromChannels,
	}
}

type ChatAdminWithInviteLinks struct {
	AdminID             int64
	InvitesCount        int32
	RevokedInvitesCount int32
}

func ParseChatAdminWithInviteLinks(raw *tg.ChatAdminWithInvites) *ChatAdminWithInviteLinks {
	if raw == nil {
		return nil
	}
	return &ChatAdminWithInviteLinks{
		AdminID:             raw.AdminID,
		InvitesCount:        raw.InvitesCount,
		RevokedInvitesCount: raw.RevokedInvitesCount,
	}
}

type FailedToAddMember struct {
	UserID               int64
	PremiumWouldAllow     bool
	PremiumRequiredForPM bool
}

func ParseFailedToAddMember(raw *tg.MissingInvitee) *FailedToAddMember {
	if raw == nil {
		return nil
	}
	return &FailedToAddMember{
		UserID:               raw.UserID,
		PremiumWouldAllow:     raw.PremiumWouldAllowInvite,
		PremiumRequiredForPM: raw.PremiumRequiredForPm,
	}
}

type FoundContacts struct {
	MyResults []int64
	Results   []int64
	Chats     []*Chat
	Users     []*User
}

type UserRating struct {
	PeerID int64
	Rating float64
}

type HistoryCleared struct{}

type ChatFolderInviteLinkInfo struct {
	FilterID     int32
	Title        string
	Emoticon     string
	Peers        []*PeerInfo
	Chats        []*Chat
	Users        []*User
	AlreadyPeers []*PeerInfo
	MissingPeers []*PeerInfo
}

type ChatJoiner struct {
	UserID      int64
	Date        int32
	About       string
	ViaChatlist bool
}
