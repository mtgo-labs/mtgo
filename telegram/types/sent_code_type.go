package types

// SentCodeType indicates the delivery method used to send a Telegram
// authentication code.
type SentCodeType string

// SentCodeType constants enumerate the possible verification code delivery
// methods.
const (
	SentCodeTypeApp                SentCodeType = "app"
	SentCodeTypeCall               SentCodeType = "call"
	SentCodeTypeFlashCall          SentCodeType = "flash_call"
	SentCodeTypeMissedCall         SentCodeType = "missed_call"
	SentCodeTypeSMS                SentCodeType = "sms"
	SentCodeTypeFragmentSMS        SentCodeType = "fragment_sms"
	SentCodeTypeEmailCode          SentCodeType = "email_code"
	SentCodeTypeFirebaseSMS        SentCodeType = "firebase_sms"
	SentCodeTypeSetupEmailRequired SentCodeType = "setup_email_required"
	SentCodeTypeSmsPhrase          SentCodeType = "sms_phrase"
	SentCodeTypeSmsWord            SentCodeType = "sms_word"
)

// String returns the string representation of the SentCodeType.
func (s SentCodeType) String() string { return string(s) }
