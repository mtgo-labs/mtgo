package types

type auctionStateMarker interface{ auctionState() }

// GiftAuctionState represents the current auction state of a gift on the
// resale market, containing the gift and its active or finished state.
type GiftAuctionState struct {
	Gift  *Gift
	State AuctionStateVariant
	Raw   any
}

// AuctionStateVariant is the interface implemented by auction state types such
// as AuctionStateActive and AuctionStateFinished.
type AuctionStateVariant interface{ auctionStateMarker }

// GiftResalePriceStar represents a gift resale price in Telegram Stars.
type GiftResalePriceStar struct {
	StarCount int64
}

// GiftResalePriceTon represents a gift resale price denominated in TON cents.
type GiftResalePriceTon struct {
	ToncoinCentCount int64
}

// UpgradedGiftAttributeIDModel identifies a model attribute of an upgraded
// gift by its sticker ID.
type UpgradedGiftAttributeIDModel struct {
	StickerID int64
}

// UpgradedGiftAttributeIDSymbol identifies a symbol attribute of an upgraded gift.
type UpgradedGiftAttributeIDSymbol struct {
	StickerID int64
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
