package types

// PhoneNumberCodeType indicates the delivery method for a phone number verification code.
type PhoneNumberCodeType string

// Phone number code delivery method constants.
const (
	PhoneNumberCodeTypeFlashCall  PhoneNumberCodeType = "flash_call"
	PhoneNumberCodeTypeSms        PhoneNumberCodeType = "sms"
	PhoneNumberCodeTypeFragment   PhoneNumberCodeType = "fragment"
	PhoneNumberCodeTypeMissedCall PhoneNumberCodeType = "missed_call"
)

// String returns the string representation of the PhoneNumberCodeType.
func (p PhoneNumberCodeType) String() string { return string(p) }
