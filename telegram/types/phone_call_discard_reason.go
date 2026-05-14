package types

// PhoneCallDiscardReason describes why a phone call ended.
type PhoneCallDiscardReason string

// Phone call discard reason constants.
const (
	PhoneCallDiscardReasonMissed                  PhoneCallDiscardReason = "missed"
	PhoneCallDiscardReasonDeclined                PhoneCallDiscardReason = "declined"
	PhoneCallDiscardReasonDisconnected            PhoneCallDiscardReason = "disconnected"
	PhoneCallDiscardReasonHungUp                  PhoneCallDiscardReason = "hung_up"
	PhoneCallDiscardReasonUpgradeToConferenceCall PhoneCallDiscardReason = "upgrade_to_conference_call"
)

// String returns the string representation of the PhoneCallDiscardReason.
func (p PhoneCallDiscardReason) String() string { return string(p) }
