package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// Giveaway represents a Telegram Premium or stars giveaway launched in one or more channels.
//
// Example:
//
//	gw := types.ParseGiveaway(rawMedia, peerMap)
//	fmt.Printf("Giveaway: %d prizes until %s\n", gw.Quantity, gw.UntilDate)
type Giveaway struct {
	Chats              []*Chat
	Quantity           int32
	Months             int32
	UntilDate          time.Time
	Description        string
	OnlyNewSubscribers bool
	OnlyForCountries   []string
	WinnersAreVisible  bool
	Stars              int64
}

// ParseGiveaway converts a TL MessageMediaGiveaway into a Giveaway.
// Returns nil if raw is nil.
//
// Example:
//
//	gw := types.ParseGiveaway(rawGiveaway, peerMap)
//	if gw != nil {
//	    fmt.Printf("Giveaway for %d months in %d chats\n", gw.Months, len(gw.Chats))
//	}
func ParseGiveaway(raw *tg.MessageMediaGiveaway, pm *PeerMap) *Giveaway {
	if raw == nil {
		return nil
	}
	out := &Giveaway{
		Quantity:           raw.Quantity,
		UntilDate:          time.Unix(int64(raw.UntilDate), 0),
		OnlyNewSubscribers: raw.OnlyNewSubscribers,
		WinnersAreVisible:  raw.WinnersAreVisible,
	}
	for _, chID := range raw.Channels {
		out.Chats = append(out.Chats, ParseChatFromPeer(&tg.PeerChannel{ChannelID: chID}, pm))
	}
	if len(raw.CountriesIso2) > 0 {
		out.OnlyForCountries = raw.CountriesIso2
	}
	if raw.PrizeDescription != "" {
		out.Description = raw.PrizeDescription
	}
	if raw.Months != 0 {
		out.Months = raw.Months
	}
	if raw.Stars != 0 {
		out.Stars = raw.Stars
	}
	return out
}

// GiveawayWinners represents the results of a completed giveaway, listing the
// winners and prize details.
//
// Example:
//
//	winners := types.ParseGiveawayWinners(rawResults, peerMap)
//	for _, w := range winners.Winners {
//	    fmt.Printf("Winner: %s\n", w.FirstName)
//	}
type GiveawayWinners struct {
	Chat                          *Chat
	GiveawayMessageID             int32
	WinnersSelectionDate          time.Time
	Quantity                      int32
	WinnerCount                   int32
	UnclaimedPrizeCount           int32
	Winners                       []*User
	GiveawayMessage               *Message
	AdditionalChatCount           int32
	PrizeStarCount                int64
	PremiumSubscriptionMonthCount int32
	OnlyNewMembers                bool
	WasRefunded                   bool
	PrizeDescription              string
}

// ParseGiveawayWinners converts a TL MessageMediaGiveawayResults into a GiveawayWinners.
// Returns nil if raw is nil.
//
// Example:
//
//	results := types.ParseGiveawayWinners(rawResults, peerMap)
//	fmt.Printf("Winners: %d, unclaimed: %d\n", results.WinnerCount, results.UnclaimedPrizeCount)
func ParseGiveawayWinners(raw *tg.MessageMediaGiveawayResults, pm *PeerMap) *GiveawayWinners {
	if raw == nil {
		return nil
	}
	out := &GiveawayWinners{
		Chat:                 ParseChatFromPeer(&tg.PeerChannel{ChannelID: raw.ChannelID}, pm),
		GiveawayMessageID:    raw.LaunchMsgID,
		WinnersSelectionDate: time.Unix(int64(raw.UntilDate), 0),
		WinnerCount:          raw.WinnersCount,
		UnclaimedPrizeCount:  raw.UnclaimedCount,
		OnlyNewMembers:       raw.OnlyNewSubscribers,
		WasRefunded:          raw.Refunded,
	}
	if len(raw.Winners) > 0 && pm != nil {
		for _, uid := range raw.Winners {
			out.Winners = append(out.Winners, getUserFromPM(pm, uid))
		}
	}
	if raw.AdditionalPeersCount != 0 {
		out.AdditionalChatCount = raw.AdditionalPeersCount
	}
	if raw.Stars != 0 {
		out.PrizeStarCount = raw.Stars
	}
	if raw.Months != 0 {
		out.PremiumSubscriptionMonthCount = raw.Months
	}
	if raw.PrizeDescription != "" {
		out.PrizeDescription = raw.PrizeDescription
	}
	return out
}

// GiveawayCreated represents the event of a giveaway being launched.
type GiveawayCreated struct {
	PrizeStarCount int64
}

// ParseGiveawayCreated converts a TL MessageActionGiveawayLaunch into a GiveawayCreated.
// Returns nil if raw is nil.
//
// Example:
//
//	created := types.ParseGiveawayCreated(rawLaunch)
//	fmt.Printf("Stars prize: %d\n", created.PrizeStarCount)
func ParseGiveawayCreated(raw *tg.MessageActionGiveawayLaunch) *GiveawayCreated {
	if raw == nil {
		return nil
	}
	return &GiveawayCreated{
		PrizeStarCount: raw.Stars,
	}
}

// GiveawayCompleted represents the event of a giveaway completing with winner and
// unclaimed prize counts.
type GiveawayCompleted struct {
	WinnerCount         int32
	UnclaimedPrizeCount int32
	GiveawayMessageID   int32
	GiveawayMessage     *Message
	IsStarGiveaway      bool
}

// GiveawayPrizeStars represents a star prize awarded in a giveaway, including the
// transaction ID and the boosted chat.
type GiveawayPrizeStars struct {
	StarCount         int64
	TransactionID     string
	BoostedChat       *Chat
	GiveawayMessageID int32
	GiveawayMessage   *Message
	IsUnclaimed       bool
	Sticker           *Sticker
}
