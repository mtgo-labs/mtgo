package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

type GiftAttribute struct {
	Type         GiftAttributeType
	Name         string
	DocumentID   int64
	BackdropID   int32
	CenterColor  int32
	EdgeColor    int32
	PatternColor int32
	TextColor    int32
	Rarity       GiftAttributeRarity
	Crafted      bool
}

type GiftAttributeRarity struct {
	Type     GiftAttributeRarityType
	Permille int32
}

type GiftAttributeRarityType string

const (
	RarityPerMille  GiftAttributeRarityType = "permille"
	RarityUncommon  GiftAttributeRarityType = "uncommon"
	RarityRare      GiftAttributeRarityType = "rare"
	RarityEpic      GiftAttributeRarityType = "epic"
	RarityLegendary GiftAttributeRarityType = "legendary"
)

func ParseGiftAttribute(raw tg.StarGiftAttributeClass) *GiftAttribute {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.StarGiftAttributeModel:
		a := &GiftAttribute{
			Type:    GiftAttributeTypeModel,
			Name:    v.Name,
			Crafted: v.Crafted,
		}
		if v.Rarity != nil {
			a.Rarity = parseRarity(v.Rarity)
		}
		return a
	case *tg.StarGiftAttributeBackdrop:
		return &GiftAttribute{
			Type:         GiftAttributeTypeBackdrop,
			Name:         v.Name,
			BackdropID:   v.BackdropID,
			CenterColor:  v.CenterColor,
			EdgeColor:    v.EdgeColor,
			PatternColor: v.PatternColor,
			TextColor:    v.TextColor,
			Rarity:       parseRarity(v.Rarity),
		}
	case *tg.StarGiftAttributePattern:
		return &GiftAttribute{
			Type:   GiftAttributeTypeSymbol,
			Name:   v.Name,
			Rarity: parseRarity(v.Rarity),
		}
	case *tg.StarGiftAttributeOriginalDetails:
		return &GiftAttribute{
			Type: GiftAttributeTypeOriginalDetails,
		}
	}
	return nil
}

func parseRarity(raw tg.StarGiftAttributeRarityClass) GiftAttributeRarity {
	if raw == nil {
		return GiftAttributeRarity{}
	}
	switch v := raw.(type) {
	case *tg.StarGiftAttributeRarity:
		return GiftAttributeRarity{Type: RarityPerMille, Permille: v.Permille}
	case *tg.StarGiftAttributeRarityUncommon:
		return GiftAttributeRarity{Type: RarityUncommon}
	case *tg.StarGiftAttributeRarityRare:
		return GiftAttributeRarity{Type: RarityRare}
	case *tg.StarGiftAttributeRarityEpic:
		return GiftAttributeRarity{Type: RarityEpic}
	case *tg.StarGiftAttributeRarityLegendary:
		return GiftAttributeRarity{Type: RarityLegendary}
	}
	return GiftAttributeRarity{}
}

type UpgradedGiftAttributeID struct {
	Type       string
	DocumentID int64
	BackdropID int32
}

type UpgradedGiftPurchaseOfferRejected struct{}

type GiftCollection struct {
	ID         int32
	Title      string
	GiftsCount int32
	Hash       int64
}

func ParseGiftCollection(raw *tg.StarGiftCollection) *GiftCollection {
	if raw == nil {
		return nil
	}
	return &GiftCollection{
		ID:         raw.CollectionID,
		Title:      raw.Title,
		GiftsCount: raw.GiftsCount,
		Hash:       raw.Hash,
	}
}

type GiftPurchaseLimit struct {
	PerUserTotal   int32
	PerUserRemains int32
}

type GiftResaleParameters struct {
	MinStars int64
	Slug     string
}

type GiftResalePrice struct {
	StarsAmount int64
	TonAmount   int64
}

type GiftUpgradePreview struct {
	SampleAttributes []GiftAttribute
	Prices           []GiftUpgradePrice
}

func ParseGiftUpgradePreview(raw *tg.PaymentsStarGiftUpgradePreview) *GiftUpgradePreview {
	if raw == nil {
		return nil
	}
	p := &GiftUpgradePreview{}
	for _, a := range raw.SampleAttributes {
		if attr := ParseGiftAttribute(a); attr != nil {
			p.SampleAttributes = append(p.SampleAttributes, *attr)
		}
	}
	for _, price := range raw.Prices {
		p.Prices = append(p.Prices, GiftUpgradePrice{
			Date:         price.Date,
			UpgradeStars: price.UpgradeStars,
		})
	}
	return p
}

type GiftUpgradePrice struct {
	Date         int32
	UpgradeStars int64
}

type GiftUpgradeVariants struct {
	Count int32
}

type CheckedGiftCode struct {
	ViaGiveaway   bool
	FromID        int64
	GiveawayMsgID int32
	ToID          int64
	Date          int32
	Days          int32
	UsedDate      int32
}

func ParseCheckedGiftCode(raw *tg.PaymentsCheckedGiftCode) *CheckedGiftCode {
	if raw == nil {
		return nil
	}
	c := &CheckedGiftCode{
		ViaGiveaway: raw.ViaGiveaway,
		Date:        raw.Date,
		Days:        raw.Days,
	}
	if raw.GiveawayMsgID != 0 {
		c.GiveawayMsgID = raw.GiveawayMsgID
	}
	if raw.ToID != 0 {
		c.ToID = raw.ToID
	}
	if raw.UsedDate != 0 {
		c.UsedDate = raw.UsedDate
	}
	return c
}

type PremiumGiftCode struct {
	Code   string
	Months int32
	Used   bool
}

type CraftGiftResult struct {
	Success bool
	Gift    *Gift
}

type CraftGiftResultSuccess struct {
	Gift *Gift
}

type CraftGiftResultFail struct{}

type GiftedPremium struct {
	GifterID  int64
	Currency  string
	Amount    int64
	Months    int32
	Anonymous bool
}

type GiftedStars struct {
	GifterID  int64
	Stars     int64
	Anonymous bool
}

type GiftedTon struct {
	GifterID  int64
	Amount    int64
	Anonymous bool
}

type AuctionBid struct {
	PeerID int64
	Amount int64
	Date   int32
	Round  int32
}

type AuctionStateActive struct {
	Version      int32
	StartDate    int32
	EndDate      int32
	MinBidAmount int64
	GiftsLeft    int32
	CurrentRound int32
	TotalRounds  int32
	Rounds       []AuctionRound
}

type AuctionStateFinished struct {
	StartDate    int32
	EndDate      int32
	AveragePrice int64
}
