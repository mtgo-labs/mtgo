package types

// StickerType categorizes the kind of sticker in a sticker set.
type StickerType string

// StickerType constants enumerate the possible sticker categories.
const (
	StickerTypeRegular     StickerType = "regular"
	StickerTypeMask        StickerType = "mask"
	StickerTypeCustomEmoji StickerType = "custom_emoji"
)

// String returns the string representation of the StickerType.
func (s StickerType) String() string { return string(s) }
