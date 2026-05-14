package transport

import "net"

// TCPIntermediateO combines [TCPIntermediate] framing with [TCPObfuscated]
// encryption in a single type. It is a convenience wrapper for the common
// obfuscated-intermediate transport mode used by Telegram clients.
type TCPIntermediateO struct {
	*TCPObfuscated
}

// NewTCPIntermediateO returns a new obfuscated intermediate transport wrapping conn.
func NewTCPIntermediateO(conn net.Conn) *TCPIntermediateO {
	inner := NewTCPIntermediate(conn)
	obf := NewTCPObfuscated(inner, 0xEE)
	return &TCPIntermediateO{TCPObfuscated: obf}
}
