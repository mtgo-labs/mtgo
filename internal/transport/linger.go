package transport

import "net"

// SetTCPLinger sets SO_LINGER on the underlying *net.TCPConn, unwrapping
// through common wrappers like *tls.Conn. secs=0 forces RST on close (abortive
// close), avoiding TIME_WAIT that can cause the server to see two connections
// with the same auth key and return AUTH_KEY_DUPLICATED (406).
//
// If the conn is not a TCP connection or the linger option is not supported,
// this is a no-op.
func SetTCPLinger(conn net.Conn, secs int) error {
	for {
		switch c := conn.(type) {
		case *net.TCPConn:
			return c.SetLinger(secs)
		case interface{ NetConn() net.Conn }:
			conn = c.NetConn()
		default:
			return nil
		}
	}
}
