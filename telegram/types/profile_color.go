package types

// ProfileColor represents the accent color applied to a user's profile.
// Premium users can choose a profile color to customize their profile
// appearance in the app.
type ProfileColor string

const (
	// ProfileColorBlue applies a blue accent color to the profile.
	ProfileColorBlue ProfileColor = "blue"
	// ProfileColorCyan applies a cyan accent color to the profile.
	ProfileColorCyan ProfileColor = "cyan"
	// ProfileColorGreen applies a green accent color to the profile.
	ProfileColorGreen ProfileColor = "green"
	// ProfileColorOrange applies an orange accent color to the profile.
	ProfileColorOrange ProfileColor = "orange"
	// ProfileColorPink applies a pink accent color to the profile.
	ProfileColorPink ProfileColor = "pink"
	// ProfileColorRed applies a red accent color to the profile.
	ProfileColorRed ProfileColor = "red"
	// ProfileColorViolet applies a violet accent color to the profile.
	ProfileColorViolet ProfileColor = "violet"
)

// String returns the string representation of the ProfileColor.
func (p ProfileColor) String() string { return string(p) }
