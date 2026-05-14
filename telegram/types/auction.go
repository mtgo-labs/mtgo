package types

import "github.com/mtgo-labs/mtgo/tg"

// AuctionRound describes a single round within a gift auction.
type AuctionRound struct {
	// RoundNumber is the sequential number of the auction round.
	RoundNumber int32
	// StartDate is the Unix timestamp when the round starts.
	StartDate int32
	// EndDate is the Unix timestamp when the round ends.
	EndDate int32
}

// ParseAuctionRound converts a tg.StarGiftAuctionRoundClass into an AuctionRound.
// Returns nil if raw is nil.
func ParseAuctionRound(raw tg.StarGiftAuctionRoundClass) *AuctionRound {
	if raw == nil {
		return nil
	}
	switch r := raw.(type) {
	case *tg.StarGiftAuctionRound:
		return &AuctionRound{
			RoundNumber: r.Num,
			EndDate:     r.Duration,
		}
	case *tg.StarGiftAuctionRoundExtendable:
		return &AuctionRound{
			RoundNumber: r.Num,
			EndDate:     r.Duration,
		}
	}
	return nil
}

// AuctionState holds the current bidding state of a gift auction.
type AuctionState struct {
	// TopBidderID is the user ID of the current highest bidder, or zero.
	TopBidderID int64
	// TopBid is the current highest bid amount in Telegram Stars.
	TopBid int64
	// BidsCount is the total number of bids placed in the auction.
	BidsCount int32
	// Round is the current auction round number.
	Round int32
}

// ParseAuctionState converts a tg.StarGiftAuctionState into an AuctionState.
// Returns nil if raw is nil.
func ParseAuctionState(raw *tg.StarGiftAuctionState) *AuctionState {
	if raw == nil {
		return nil
	}
	s := &AuctionState{
		TopBid: raw.MinBidAmount,
		Round:  raw.CurrentRound,
	}
	if len(raw.TopBidders) > 0 {
		s.TopBidderID = raw.TopBidders[0]
	}
	return s
}

// GiftAuction represents a complete gift auction with its rounds and bidding info.
type GiftAuction struct {
	// GiftID is the unique identifier of the gift being auctioned.
	GiftID int64
	// Slug is the unique alphanumeric slug of the auction.
	Slug string
	// StartDate is the Unix timestamp when the auction starts.
	StartDate int32
	// EndDate is the Unix timestamp when the auction ends.
	EndDate int32
	// MinStars is the minimum bid amount in Telegram Stars.
	MinStars int64
	// TopBid is the current highest bid in Telegram Stars.
	TopBid int64
	// Bidders is the number of unique bidders.
	Bidders int32
	// Rounds contains the auction round details.
	Rounds []AuctionRound
}

// ParseGiftAuction converts a tg.StarGiftAuctionState into a GiftAuction.
// Returns nil if raw is nil.
func ParseGiftAuction(raw *tg.StarGiftAuctionState) *GiftAuction {
	if raw == nil {
		return nil
	}
	ga := &GiftAuction{
		StartDate: raw.StartDate,
		EndDate:   raw.EndDate,
		MinStars:  raw.MinBidAmount,
		TopBid:    raw.MinBidAmount,
		Bidders:   int32(len(raw.TopBidders)),
	}
	for _, r := range raw.Rounds {
		if parsed := ParseAuctionRound(r); parsed != nil {
			ga.Rounds = append(ga.Rounds, *parsed)
		}
	}
	return ga
}
