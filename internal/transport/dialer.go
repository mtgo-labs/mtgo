package transport

import (
	"net"
	"time"
)

// Dialer abstracts the creation of network connections with a timeout,
// allowing different dialer implementations (standard net, proxy, etc.)
// to be swapped in.
type Dialer interface {
	// Dial establishes a new connection to the specified address using the
	// given network type (e.g. "tcp") with the provided timeout.
	Dial(network, address string, timeout time.Duration) (net.Conn, error)
}

// NetDialer implements [Dialer] using the standard library's
// [net.DialTimeout].
type NetDialer struct {
	LocalAddr string
}

func (d *NetDialer) Dial(network, address string, timeout time.Duration) (net.Conn, error) {
	if d.LocalAddr != "" {
		laddr, err := net.ResolveTCPAddr(network, d.LocalAddr)
		if err != nil {
			return nil, err
		}
		dialer := net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second, LocalAddr: laddr}
		return dialer.Dial(network, address)
	}
	dialer := net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}
	return dialer.Dial(network, address)
}
