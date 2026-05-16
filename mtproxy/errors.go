package mtproxy

import "errors"

// MTProxy errors.
//
// These errors are returned during MTProxy connection setup and the
// specialized handshake.
var (
	// ErrInvalidSecretLen is returned when the MTProxy secret is not 16 bytes
	// (standard), 17 bytes (with DD prefix for TLS), or 18+ bytes (with
	// domain for fake TLS).
	ErrInvalidSecretLen = errors.New("mtproxy: secret must be 16, 17, or 18+ bytes")
	// ErrServerHelloFailed is returned when the server's hello response
	// cannot be verified against the expected key derived from the secret.
	ErrServerHelloFailed = errors.New("mtproxy: server hello verification failed")
)
