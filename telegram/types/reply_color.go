package types

// ReplyColor represents the color scheme used for a reply background.
// Use this to customize or identify the visual treatment of quoted reply
// blocks in messages.
type ReplyColor string

const (
	// ReplyColorPrimary uses the primary theme color for the reply background.
	ReplyColorPrimary ReplyColor = "primary"
	// ReplyColorSecondary uses the secondary theme color for the reply background.
	ReplyColorSecondary ReplyColor = "secondary"
	// ReplyColorAccent uses the accent theme color for the reply background.
	ReplyColorAccent ReplyColor = "accent"
	// ReplyColorNone indicates no background color is applied to the reply.
	ReplyColorNone ReplyColor = "none"
)

// String returns the string representation of the ReplyColor.
func (r ReplyColor) String() string { return string(r) }
