//go:build linux || darwin

package telegram

import "github.com/mtgo-labs/mtgo/internal/transport"

func newNetPollDialer() transport.Dialer {
	return &transport.NetPollDialer{}
}
