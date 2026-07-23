package telegram

import (
	"context"
	"errors"
	"io"
	"net"
	"syscall"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
)

type transferRetryContextKey struct{}

// uploadPartTimeout is the per-part RPC timeout for uploads. Each upload part
// gets its own deadline instead of inheriting the full upload context deadline,
// so a single slow part can't consume the entire upload budget.
const uploadPartTimeout = 2 * time.Minute

func withTransferRetry(ctx context.Context) context.Context {
	return context.WithValue(ctx, transferRetryContextKey{}, true)
}

func isSessionClosedErr(err error) bool {
	return isSessionDeadErr(err) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE)
}

func isTransferSessionDeadErr(err error) bool {
	return isSessionClosedErr(err) ||
		errors.Is(err, session.ErrSendTimeout) ||
		isDCConnectionFailure(err)
}

// uploadRPC uses the main MTProto session. Session.Invoke supports concurrent
// RPCs; opening home-DC side sessions with the same permanent auth key can
// invalidate that key when the main session reconnects.
func (c *Client) uploadRPC() *tg.RPCClient {
	return c.Raw()
}
