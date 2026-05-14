package transport

import "net"

// TCPAbridgedO combines [TCPAbridged] framing with [TCPObfuscated]
// encryption in a single type. It is a convenience wrapper for the common
// obfuscated-abridged transport mode used by Telegram clients.
type TCPAbridgedO struct {
	*TCPObfuscated
}

// NewTCPAbridgedO returns a new obfuscated abridged transport wrapping conn.
func NewTCPAbridgedO(conn net.Conn) *TCPAbridgedO {
	inner := NewTCPAbridged(conn)
	obf := NewTCPObfuscated(inner, 0xEF)
	return &TCPAbridgedO{TCPObfuscated: obf}
}
