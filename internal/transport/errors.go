package transport

import (
	"errors"
	"fmt"
)

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

// TransportError represents a server-side transport error code sent as a
// 4-byte signed little-endian negative integer. The absolute value is the
// HTTP-like error code.
//
// See https://core.telegram.org/mtproto/mtproto-transports#transport-errors
type TransportError struct {
	Code int32 // negative error code (e.g., -404, -429, -444)
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("transport: server error %d", e.Code)
}

// Common transport error codes returned by the server.
const (
	// ErrCodeAuthKeyNotFound (-404): the auth key ID is unknown to this DC.
	// The client must recreate the auth key.
	ErrCodeAuthKeyNotFound int32 = -404
	// ErrCodeFlood (-429): too many transport connections or service message
	// limits reached. The client must back off before reconnecting.
	ErrCodeFlood int32 = -429
	// ErrCodeInvalidDC (-444): an invalid DC ID was specified during auth key
	// creation or MTProxy connection.
	ErrCodeInvalidDC int32 = -444
)

// IsTransportError reports whether err is a *TransportError with the given code.
// If no codes are provided, reports whether err is any *TransportError.
func IsTransportError(err error, codes ...int32) bool {
	var te *TransportError
	if !errors.As(err, &te) {
		return false
	}
	if len(codes) == 0 {
		return true
	}
	for _, c := range codes {
		if te.Code == c {
			return true
		}
	}
	return false
}

// DetectTransportError checks whether a 4-byte payload is a transport error
// code (negative signed int32). Returns nil if the payload is not a transport
// error.
//
// Transport error codes are small negative numbers (e.g., -404, -429, -444).
// Values with bit 31 set but a large absolute value (code < -1000) are quick
// ACK tokens, not transport errors.
//
// See https://core.telegram.org/mtproto/mtproto-transports#transport-errors
func DetectTransportError(payload []byte) *TransportError {
	if len(payload) != 4 {
		return nil
	}
	code := int32(uint32(payload[0]) | uint32(payload[1])<<8 | uint32(payload[2])<<16 | uint32(payload[3])<<24)
	if code >= 0 || code < -1000 {
		return nil
	}
	return &TransportError{Code: code}
}

// IsQuickAckToken reports whether a payload is a quick ACK token from the
// server. Quick ACK tokens are 4-byte values with bit 31 set (≥ 0x80000000
// unsigned), which the server sends to acknowledge receipt of a payload.
//
// They can be distinguished from transport errors because transport error
// codes are small negative numbers (≥ -1000), while quick ACK tokens have
// large absolute values (< -1000 as signed int32).
//
// See https://core.telegram.org/mtproto/mtproto-transports#quick-ack
func IsQuickAckToken(payload []byte) bool {
	if len(payload) != 4 {
		return false
	}
	val := uint32(payload[0]) | uint32(payload[1])<<8 | uint32(payload[2])<<16 | uint32(payload[3])<<24
	return val >= 0x80000000
}
