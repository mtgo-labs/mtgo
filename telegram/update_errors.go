package telegram

import "errors"

// Update processing errors.
//
// These errors are returned by the update manager when processing incoming
// Telegram updates (new messages, channel differences, etc.).
var (
	// ErrUpdateGapDetected is returned when a gap is detected in the update
	// sequence, meaning some updates may have been missed.
	ErrUpdateGapDetected = errors.New("telegram: update gap detected")
	// ErrDifferenceRecovery is returned when the update manager fails to
	// recover the difference between the local and remote update state.
	ErrDifferenceRecovery = errors.New("telegram: difference recovery failed")
	// ErrChannelDifference is returned when fetching channel difference
	// updates fails.
	ErrChannelDifference = errors.New("telegram: channel difference failed")
	// ErrUpdateStateUnavailable is returned when the update state cannot be
	// retrieved or does not exist.
	ErrUpdateStateUnavailable = errors.New("telegram: update state unavailable")
	// ErrUpdateQueueFull is returned when the update queue has reached its
	// maximum capacity and cannot accept more updates.
	ErrUpdateQueueFull = errors.New("telegram: update queue full")
	// ErrUpdateHandlerFailed is returned when a registered update handler
	// returns an error during dispatch.
	ErrUpdateHandlerFailed = errors.New("telegram: update handler failed")
	// ErrUpdateManagerClosed is returned when an operation is attempted on
	// an update manager that has been stopped.
	ErrUpdateManagerClosed = errors.New("telegram: update manager closed")
)
