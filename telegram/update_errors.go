package telegram

import "errors"

var (
	ErrUpdateGapDetected      = errors.New("telegram: update gap detected")
	ErrDifferenceRecovery     = errors.New("telegram: difference recovery failed")
	ErrChannelDifference      = errors.New("telegram: channel difference failed")
	ErrUpdateStateUnavailable = errors.New("telegram: update state unavailable")
	ErrUpdateQueueFull        = errors.New("telegram: update queue full")
	ErrUpdateHandlerFailed    = errors.New("telegram: update handler failed")
	ErrUpdateManagerClosed    = errors.New("telegram: update manager closed")
)
