package types

// UpgradedGiftOrigin indicates how a user acquired an upgraded (unique)
// gift. Use this to display provenance information in the gift detail view.
type UpgradedGiftOrigin string

const (
	// UpgradedGiftOriginUpgrade indicates the user upgraded a regular star gift
	// to a unique version.
	UpgradedGiftOriginUpgrade UpgradedGiftOrigin = "upgrade"
	// UpgradedGiftOriginTransfer indicates the user received the upgraded gift
	// via a direct transfer from another user.
	UpgradedGiftOriginTransfer UpgradedGiftOrigin = "transfer"
	// UpgradedGiftOriginResale indicates the user purchased the upgraded gift
	// from the gift resale marketplace.
	UpgradedGiftOriginResale UpgradedGiftOrigin = "resale"
	// UpgradedGiftOriginBlockchain indicates the gift was minted or transferred
	// via the TON blockchain.
	UpgradedGiftOriginBlockchain UpgradedGiftOrigin = "blockchain"
	// UpgradedGiftOriginGiftedUpgrade indicates the user received the upgraded
	// gift directly as a gift from another user.
	UpgradedGiftOriginGiftedUpgrade UpgradedGiftOrigin = "gifted_upgrade"
	// UpgradedGiftOriginOffer indicates the user acquired the gift through a
	// purchase offer acceptance.
	UpgradedGiftOriginOffer UpgradedGiftOrigin = "offer"
	// UpgradedGiftOriginCraft indicates the user crafted the upgraded gift by
	// combining other gifts.
	UpgradedGiftOriginCraft UpgradedGiftOrigin = "craft"
)

// String returns the string representation of the UpgradedGiftOrigin.
func (u UpgradedGiftOrigin) String() string { return string(u) }
