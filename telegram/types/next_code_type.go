package types

// NextCodeType indicates how the next authentication code will be delivered
// during the Telegram login flow.
type NextCodeType string

// Next code delivery method constants.
const (
	NextCodeTypeSMS      NextCodeType = "sms"
	NextCodeTypeCall     NextCodeType = "call"
	NextCodeTypeFlash    NextCodeType = "flash"
	NextCodeTypeMissed   NextCodeType = "missed"
	NextCodeTypeFragment NextCodeType = "fragment"
	NextCodeTypeFirebase NextCodeType = "firebase"
)

// String returns the string representation of the NextCodeType.
func (n NextCodeType) String() string { return string(n) }
