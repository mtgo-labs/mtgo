package types

// GiftAttributeType enumerates the types of attributes a Telegram gift can have.
// Each attribute contributes to the gift's visual appearance and rarity.
type GiftAttributeType string

const (
	// GiftAttributeTypePattern is a decorative pattern overlay on the gift.
	GiftAttributeTypePattern GiftAttributeType = "pattern"
	// GiftAttributeTypeBackdrop is a background color or gradient for the gift.
	GiftAttributeTypeBackdrop GiftAttributeType = "backdrop"
	// GiftAttributeTypeCentered centers the gift's visual element within its
	// display area.
	GiftAttributeTypeCentered GiftAttributeType = "centered"
	// GiftAttributeTypeModel is a 3D model or sticker displayed as the gift's
	// primary visual.
	GiftAttributeTypeModel GiftAttributeType = "model"
	// GiftAttributeTypeCustom is a user-defined or special attribute not covered
	// by the standard categories.
	GiftAttributeTypeCustom GiftAttributeType = "custom"
	// GiftAttributeTypeSymbol is a decorative symbol/pattern attribute.
	GiftAttributeTypeSymbol GiftAttributeType = "symbol"
	// GiftAttributeTypeCounter is a gift attribute representing a counter.
	GiftAttributeTypeCounter GiftAttributeType = "counter"
	// GiftAttributeTypeOriginalDetails is a gift attribute holding original gift details.
	GiftAttributeTypeOriginalDetails GiftAttributeType = "original_details"
)

// String returns the string representation of the GiftAttributeType.
func (g GiftAttributeType) String() string { return string(g) }
