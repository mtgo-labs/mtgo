//go:build !linux && !darwin

package telegram

import (
	"fmt"
	"runtime"

	"github.com/mtgo-labs/mtgo/internal/transport"
)

func newNetPollDialer() transport.Dialer {
	panic(fmt.Sprintf("netpoll is not supported on %s", runtime.GOOS))
}
