package types

import "github.com/mtgo-labs/mtgo/tg"

// Giveaway represents a Telegram giveaway event for distributing prizes.
type Giveaway struct {
	// Channels is the list of channel IDs participating in the giveaway.
	Channels []int64
	// CountriesISO2 is the list of two-letter country codes eligible for the giveaway.
	CountriesISO2 []string
	// PrizeDescription is an optional description of the giveaway prize.
	PrizeDescription string
	// Quantity is the number of winners in the giveaway.
	Quantity int32
	// Months is the number of months of Telegram Premium awarded (if applicable).
	Months int32
	// Stars is the number of Telegram Stars awarded (if applicable).
	Stars int64
	// UntilDate is the Unix timestamp when the giveaway ends.
	UntilDate int32
	// OnlyNewSubscribers indicates whether only new subscribers can participate.
	OnlyNewSubscribers bool
	// WinnersAreVisible indicates whether the winners are publicly visible.
	WinnersAreVisible bool
}

// GiveawayWinners represents the results of a Telegram giveaway.
type GiveawayWinners struct {
	// ChannelID is the ID of the channel that hosted the giveaway.
	ChannelID int64
	// AdditionalPeersCount is the number of additional peers in the giveaway.
	AdditionalPeersCount int32
	// LaunchMsgID is the message ID of the giveaway launch message.
	LaunchMsgID int32
	// Winners is the list of user IDs who won the giveaway.
	Winners []int64
	// WinnersCount is the total number of winners.
	WinnersCount int32
	// UnclaimedCount is the number of unclaimed prizes.
	UnclaimedCount int32
	// Months is the number of months of Telegram Premium awarded (if applicable).
	Months int32
	// Stars is the number of Telegram Stars awarded (if applicable).
	Stars int64
	// PrizeDescription is an optional description of the giveaway prize.
	PrizeDescription string
	// UntilDate is the Unix timestamp when the giveaway ends.
	UntilDate int32
	// OnlyNewSubscribers indicates whether only new subscribers could participate.
	OnlyNewSubscribers bool
	// Refunded indicates whether the giveaway was refunded.
	Refunded bool
}

// ParseGiveaway converts a TL MessageMediaGiveaway into a Giveaway.
func ParseGiveaway(raw *tg.MessageMediaGiveaway) *Giveaway {
	if raw == nil {
		return nil
	}
	out := &Giveaway{
		Channels:           raw.Channels,
		CountriesISO2:      raw.CountriesIso2,
		Quantity:           raw.Quantity,
		UntilDate:          raw.UntilDate,
		OnlyNewSubscribers: raw.OnlyNewSubscribers,
		WinnersAreVisible:  raw.WinnersAreVisible,
	}
	if raw.PrizeDescription != "" {
		out.PrizeDescription = raw.PrizeDescription
	}
	if raw.Months != 0 {
		out.Months = raw.Months
	}
	if raw.Stars != 0 {
		out.Stars = raw.Stars
	}
	return out
}

// ParseGiveawayWinners converts a TL MessageMediaGiveawayResults into GiveawayWinners.
func ParseGiveawayWinners(raw *tg.MessageMediaGiveawayResults) *GiveawayWinners {
	if raw == nil {
		return nil
	}
	out := &GiveawayWinners{
		ChannelID:          raw.ChannelID,
		LaunchMsgID:        raw.LaunchMsgID,
		WinnersCount:       raw.WinnersCount,
		UnclaimedCount:     raw.UnclaimedCount,
		Winners:            raw.Winners,
		UntilDate:          raw.UntilDate,
		OnlyNewSubscribers: raw.OnlyNewSubscribers,
		Refunded:           raw.Refunded,
	}
	if raw.AdditionalPeersCount != 0 {
		out.AdditionalPeersCount = raw.AdditionalPeersCount
	}
	if raw.Months != 0 {
		out.Months = raw.Months
	}
	if raw.Stars != 0 {
		out.Stars = raw.Stars
	}
	if raw.PrizeDescription != "" {
		out.PrizeDescription = raw.PrizeDescription
	}
	return out
}
