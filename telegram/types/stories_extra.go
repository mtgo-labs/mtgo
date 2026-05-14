package types

// StoriesStealthMode represents the stealth mode state for stories.
type StoriesStealthMode struct {
	ActiveUntil   int32
	CooldownUntil int32
}
