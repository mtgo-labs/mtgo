//go:build linux || darwin

package transport

import (
	"net"
	"time"

	"github.com/cloudwego/netpoll"
)

// NetPollDialer implements [Dialer] using the cloudwego/netpoll
// event-loop networking library for high-performance I/O on Linux and macOS.
type NetPollDialer struct{}

// Dial establishes a netpoll-based connection to address on the given network
// with the specified timeout.
func (d *NetPollDialer) Dial(network, address string, timeout time.Duration) (net.Conn, error) {
	conn, err := netpoll.DialConnection(network, address, timeout)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
