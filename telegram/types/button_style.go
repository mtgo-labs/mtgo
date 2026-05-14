package types

// ButtonStyle represents the visual style of an inline keyboard button.
type ButtonStyle string

// Button style variants.
const (
	// ButtonStyleDefault is the standard button appearance.
	ButtonStyleDefault ButtonStyle = "default"
	// ButtonStylePrimary highlights the button as the primary action.
	ButtonStylePrimary ButtonStyle = "primary"
	// ButtonStyleDanger styles the button for destructive actions.
	ButtonStyleDanger ButtonStyle = "danger"
	// ButtonStyleSuccess styles the button for confirmatory actions.
	ButtonStyleSuccess ButtonStyle = "success"
)

// String returns the string representation of the ButtonStyle.
func (b ButtonStyle) String() string { return string(b) }
