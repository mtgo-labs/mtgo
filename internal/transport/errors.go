package transport

import "errors"

// MaxPayloadLen is the maximum allowed payload size for received transport
// packets. Packets exceeding this size are rejected to prevent OOM attacks.
const MaxPayloadLen = 16 * 1024 * 1024

// Transport-level protocol errors.
//
// These errors indicate data integrity or protocol compatibility issues at
// the transport layer (TCP, obfuscated TCP, etc.).
var (
	// ErrPayloadTooLarge is returned when attempting to send or receive a
	// message that exceeds the maximum payload size for the configured
	// transport.
	ErrPayloadTooLarge = errors.New("transport: payload exceeds maximum size")
	// ErrCRC32Mismatch is returned when the CRC32 checksum of a received
	// TCP packet does not match the expected value, indicating data
	// corruption in transit.
	ErrCRC32Mismatch = errors.New("tcp_full: crc32 checksum mismatch")
	// ErrUnsupportedTransport is returned when the obfuscated transport
	// handshake detects an inner transport type that is not supported.
	ErrUnsupportedTransport = errors.New("tcp_obfuscated: unsupported inner transport type")
)
