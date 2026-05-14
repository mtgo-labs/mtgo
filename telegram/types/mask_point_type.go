package types

// MaskPointType enumerates the facial points where a mask sticker can be placed.
type MaskPointType string

// Mask point type constants representing the facial positions for mask sticker placement.
const (
	MaskPointTypeForehead MaskPointType = "forehead"
	MaskPointTypeEyes     MaskPointType = "eyes"
	MaskPointTypeMouth    MaskPointType = "mouth"
	MaskPointTypeChin     MaskPointType = "chin"
)

// String returns the string representation of the MaskPointType.
func (m MaskPointType) String() string { return string(m) }
