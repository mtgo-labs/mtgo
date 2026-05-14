package types

// GiftAuctionState represents the current state of a gift auction.
type GiftAuctionState struct {
	Active    bool
	Finished  bool
	Version   int32
	StartDate int32
	EndDate   int32
	MinBid    int64
	GiftsLeft int32
}

// GiftResalePriceStar represents a gift resale price in Telegram Stars.
type GiftResalePriceStar struct {
	Stars int64
}

// GiftResalePriceTon represents a gift resale price in TON.
type GiftResalePriceTon struct {
	TonAmount int64
}

// UpgradedGiftAttributeIDModel identifies a model attribute of an upgraded gift.
type UpgradedGiftAttributeIDModel struct {
	DocumentID int64
}

// UpgradedGiftAttributeIDSymbol identifies a symbol attribute of an upgraded gift.
type UpgradedGiftAttributeIDSymbol struct {
	DocumentID int64
}

// UpgradedGiftAttributeIDBackdrop identifies a backdrop attribute of an upgraded gift.
type UpgradedGiftAttributeIDBackdrop struct {
	BackdropID int32
}

// UpgradedGiftAttributeRarityPerMille represents gift rarity in per-mille.
type UpgradedGiftAttributeRarityPerMille struct {
	Permille int32
}

// UpgradedGiftAttributeRarityUncommon represents an uncommon rarity level.
type UpgradedGiftAttributeRarityUncommon struct{}

// UpgradedGiftAttributeRarityRare represents a rare rarity level.
type UpgradedGiftAttributeRarityRare struct{}

// UpgradedGiftAttributeRarityEpic represents an epic rarity level.
type UpgradedGiftAttributeRarityEpic struct{}

// UpgradedGiftAttributeRarityLegendary represents a legendary rarity level.
type UpgradedGiftAttributeRarityLegendary struct{}
