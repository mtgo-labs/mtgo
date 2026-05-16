package types

// MaskPointType enumerates the facial points where a mask sticker can be placed.
type MaskPointType string

// Mask point type constants representing the facial positions for mask sticker placement.
const (
	MaskPointForehead MaskPointType = "forehead"
	MaskPointEyes     MaskPointType = "eyes"
	MaskPointMouth    MaskPointType = "mouth"
	MaskPointChin     MaskPointType = "chin"
)

// String returns the string representation of the MaskPointType.
func (m MaskPointType) String() string { return string(m) }
