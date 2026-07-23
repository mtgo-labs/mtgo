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

// uploadRPCs returns RPC clients for parallel upload. When UploadPoolSize > 1,
// it lazily creates a pool of independent sessions on the home DC that share
// the main session's permanent auth key (no DH exchange, no PFS — matching
// mtcute's media-connection design). Each session has its own TCP connection,
// so uploads survive individual connection deaths. Falls back to the main
// session when the pool is disabled or unavailable.
func (c *Client) uploadRPCs(ctx context.Context, fileSize int64) ([]*tg.RPCClient, error) {
	poolSize := c.config().UploadPoolSize
	if poolSize <= 1 {
		return []*tg.RPCClient{c.Raw()}, nil
	}

	c.uploadPoolMu.Lock()
	if c.uploadPool == nil {
		sz := uploadPoolSize(fileSize, poolSize)
		c.uploadPool = newUploadSessionPool(c, sz)
	}
	pool := c.uploadPool
	c.uploadPoolMu.Unlock()

	if err := pool.ensureCreated(ctx); err != nil {
		return []*tg.RPCClient{c.Raw()}, nil
	}

	// Return the pool invoker as a single RPC client. It handles round-robin
	// across pool entries, automatic dead-session replacement, and fallback
	// to the main session — so workers don't need retry logic.
	return []*tg.RPCClient{pool.rpcClient()}, nil
}
