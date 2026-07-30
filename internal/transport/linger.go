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

// SetTCPNoDelay enables or disables Nagle's algorithm on the underlying
// *net.TCPConn, unwrapping through common wrappers like *tls.Conn. MTProto
// sends many small request/response packets; with Nagle's algorithm on (the
// Go default), each is delayed by up to ~200ms waiting for more data to
// coalesce. Disabling it eliminates that latency.
//
// If the conn is not a TCP connection or the option is not supported, this is
// a no-op.
func SetTCPNoDelay(conn net.Conn, enable bool) error {
	for {
		switch c := conn.(type) {
		case *net.TCPConn:
			return c.SetNoDelay(enable)
		case interface{ NetConn() net.Conn }:
			conn = c.NetConn()
		default:
			return nil
		}
	}
}
