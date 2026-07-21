package telegram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// sideSession holds a session that shares the main session's permanent auth
// key but runs on its own TCP connection with its own message ID space.
type sideSession struct {
	sess    *session.Session
	closer  ioCloser
	invoker tg.Invoker // *dcSessionInvoker with its own apiInit
	dead    atomic.Bool
}

// uploadPartTimeout is the per-part RPC timeout for uploads. Each upload part
// gets its own deadline instead of inheriting the full upload context deadline,
// so a single slow part can't consume the entire upload budget.
const uploadPartTimeout = 2 * time.Minute

// uploadPoolSize is the number of dedicated TCP connections for upload traffic.
// Multiple connections avoid single-connection write serialization when
// default upload workers compete for one transport.
const uploadPoolSize = defaultTransferWorkers

// uploadPoolInvoker round-robins RPC calls across multiple upload sessions.
// If a session dies, it is replaced and the current RPC is retried once.
type uploadPoolInvoker struct {
	client   *Client
	sessions []*sideSession
	idx      atomic.Uint64
}

func (p *uploadPoolInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	n := uint64(len(p.sessions))
	var lastErr error
	needsRepair := n == 0
	for i := uint64(0); i < n; i++ {
		idx := p.idx.Add(1) % n
		ss := p.sessions[idx]
		if sideSessionUnavailable(ss) {
			needsRepair = true
			continue
		}
		result, err := ss.invoker.RPCInvoke(ctx, input, decode)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if isTransferSessionDeadErr(err) {
			ss.dead.Store(true)
			needsRepair = true
			continue
		}
		return nil, err
	}
	if needsRepair && p.client != nil {
		pool, err := p.client.getUploadPool()
		if err == nil {
			return (&uploadPoolInvoker{sessions: pool}).RPCInvoke(ctx, input, decode)
		}
		if lastErr != nil {
			return nil, errors.Join(lastErr, fmt.Errorf("upload pool: repair: %w", err))
		}
		return nil, err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("upload pool: all sessions exhausted")
}

func (p *uploadPoolInvoker) RPCInvokeRaw(ctx context.Context, input tg.TLObject) ([]byte, error) {
	n := uint64(len(p.sessions))
	var lastErr error
	needsRepair := n == 0
	for i := uint64(0); i < n; i++ {
		idx := p.idx.Add(1) % n
		ss := p.sessions[idx]
		if sideSessionUnavailable(ss) {
			needsRepair = true
			continue
		}
		result, err := ss.invoker.RPCInvokeRaw(ctx, input)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if isTransferSessionDeadErr(err) {
			ss.dead.Store(true)
			needsRepair = true
			continue
		}
		return nil, err
	}
	if needsRepair && p.client != nil {
		pool, err := p.client.getUploadPool()
		if err == nil {
			return (&uploadPoolInvoker{sessions: pool}).RPCInvokeRaw(ctx, input)
		}
		if lastErr != nil {
			return nil, errors.Join(lastErr, fmt.Errorf("upload pool: repair: %w", err))
		}
		return nil, err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("upload pool: all sessions exhausted")
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
		tgerr.Is(err, tgerr.ErrAuthKeyUnregistered, tgerr.ErrAuthKeyPermEmpty)
}

func sideSessionUnavailable(ss *sideSession) bool {
	if ss == nil || ss.dead.Load() {
		return true
	}
	return ss.sess != nil && !ss.sess.IsConnected()
}

// uploadRPC returns an RPC client backed by a pool of dedicated upload
// sessions on separate TCP connections. This isolates upload traffic from API
// calls and updates, and parallelizes writes across multiple connections.
//
// If the pool cannot be created, it falls back to the main session (c.Raw()).
func (c *Client) uploadRPC() *tg.RPCClient {
	pool, err := c.getUploadPool()
	if err != nil || len(pool) == 0 {
		return c.Raw()
	}
	return tg.NewRPCClient(&uploadPoolInvoker{client: c, sessions: pool})
}

// getUploadPool returns the lazily-created upload session pool, creating it
// on first use. Thread-safe.
func (c *Client) getUploadPool() ([]*sideSession, error) {
	// Fast path: pool exists and no dead sessions.
	c.uploadSessionMu.Lock()
	pool := append([]*sideSession(nil), c.uploadPool...)
	c.uploadSessionMu.Unlock()
	if len(pool) > 0 && !hasDeadSession(pool) {
		return pool, nil
	}

	// Slow path: repair or create the pool.
	c.uploadSessionMu.Lock()
	defer c.uploadSessionMu.Unlock()

	if len(c.uploadPool) > 0 && !hasDeadSession(c.uploadPool) {
		return append([]*sideSession(nil), c.uploadPool...), nil
	}

	if len(c.uploadPool) > 0 {
		for i, ss := range c.uploadPool {
			if !sideSessionUnavailable(ss) {
				continue
			}
			if ss != nil {
				if ss.sess != nil {
					ss.sess.Stop()
				}
				if ss.closer != nil {
					_ = ss.closer.Close()
				}
			}
			replacement, err := c.createUploadSession()
			if err != nil {
				c.Log.Warnf("upload session repair failed: %v", err)
				c.uploadPool[i] = nil
				continue
			}
			c.uploadPool[i] = replacement
		}

		compact := c.uploadPool[:0]
		for _, ss := range c.uploadPool {
			if ss != nil {
				compact = append(compact, ss)
			}
		}
		c.uploadPool = compact
		if len(compact) > 0 {
			return append([]*sideSession(nil), compact...), nil
		}
	}

	sessions := c.createUploadSessions(uploadPoolSize)

	if len(sessions) == 0 {
		return nil, fmt.Errorf("upload pool: no sessions created")
	}

	c.uploadPool = sessions
	c.Log.Infof("upload pool ready: %d sessions on separate connections", len(sessions))
	return append([]*sideSession(nil), sessions...), nil
}

func (c *Client) createUploadSessions(size int) []*sideSession {
	type result struct {
		idx int
		ss  *sideSession
		err error
	}

	results := make(chan result, size)
	var wg sync.WaitGroup
	for i := 0; i < size; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ss, err := c.createUploadSession()
			results <- result{idx: idx, ss: ss, err: err}
		}(i)
	}
	wg.Wait()
	close(results)

	sessions := make([]*sideSession, size)
	created := 0
	for r := range results {
		if r.err != nil {
			c.Log.Warnf("upload session %d/%d failed: %v (continuing with %d sessions)", r.idx+1, size, r.err, created)
			continue
		}
		sessions[r.idx] = r.ss
		created++
	}

	compact := sessions[:0]
	for _, ss := range sessions {
		if ss != nil {
			compact = append(compact, ss)
		}
	}
	return compact
}

func hasDeadSession(pool []*sideSession) bool {
	for _, ss := range pool {
		if sideSessionUnavailable(ss) {
			return true
		}
	}
	return false
}

// createUploadSession opens a transport to the home DC and creates a session
// that shares the main session's auth key and server salt.
func (c *Client) createUploadSession() (*sideSession, error) {
	c.mu.RLock()
	cfg := c.cfg
	log := c.Log
	mainSess := c.session
	c.mu.RUnlock()

	if mainSess == nil {
		return nil, fmt.Errorf("upload session: main session not connected")
	}

	dc := mainSess.DC()
	if dc.Address() == "" {
		return nil, fmt.Errorf("upload session: unknown DC address")
	}

	sessionTp, err := c.dialTransport(dc, 15*time.Second, c.testDialer)
	if err != nil {
		return nil, fmt.Errorf("upload session: transport: %w", err)
	}

	uploadStorage := NewMemoryStorage()
	sess, err := session.NewSession(dc, uploadStorage,
		cfg.Device.DeviceModel, cfg.Device.AppVersion,
		cfg.Device.SystemLangCode, cfg.Device.LangCode)
	if err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("upload session: create session: %w", err)
	}
	configureSessionDispatch(sess, c)
	sess.SetUpdateHandler(func(obj tg.TLObject) {})

	authKey := mainSess.AuthKey()
	if pfs := mainSess.PFS(); pfs != nil {
		authKey = pfs.PermKey()
	}
	if len(authKey) == 0 {
		sessionTp.Close()
		return nil, fmt.Errorf("upload session: no auth key on main session")
	}
	sess.SetAuthKey(authKey)
	sess.SetServerSalt(mainSess.ServerSalt())
	configureSessionHealth(sess, c.config(), c.connMetrics)
	if err := c.prepareSessionPFS(sess, uploadStorage, dc, sessionTp, authKey); err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("upload session: prepare PFS: %w", err)
	}

	if err := sess.Connect(sessionTp, 15*time.Second); err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("upload session: connect: %w", err)
	}
	bindCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.bindSessionPFS(bindCtx, sess); err != nil {
		sess.Stop()
		sessionTp.Close()
		return nil, fmt.Errorf("upload session: bind PFS: %w", err)
	}

	log.Debugf("upload session established for DC %d", dc.ID)

	// Wrap with a resilientInvoker that marks the session dead on connection errors.
	invoker := &dcSessionInvoker{sess: sess, client: c}
	resilient := &resilientUploadInvoker{inner: invoker, session: sess}
	ss := &sideSession{
		sess:    sess,
		closer:  sessionTp,
		invoker: resilient,
	}
	resilient.ss = ss
	return ss, nil
}

// resilientUploadInvoker wraps dcSessionInvoker and marks the sideSession as
// dead when the underlying session closes, so the pool can recreate it.
type resilientUploadInvoker struct {
	inner   *dcSessionInvoker
	session *session.Session
	ss      *sideSession // set by caller after creation
}

func (r *resilientUploadInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	result, err := r.inner.RPCInvoke(ctx, input, decode)
	if err != nil && isTransferSessionDeadErr(err) {
		r.markDead()
	}
	return result, err
}

func (r *resilientUploadInvoker) RPCInvokeRaw(ctx context.Context, input tg.TLObject) ([]byte, error) {
	result, err := r.inner.RPCInvokeRaw(ctx, input)
	if err != nil && isTransferSessionDeadErr(err) {
		r.markDead()
	}
	return result, err
}

func (r *resilientUploadInvoker) markDead() {
	if r.ss != nil {
		r.ss.dead.Store(true)
	}
	if r.session != nil {
		r.session.Stop()
	}
}

// stopUploadSession tears down all upload sessions. Called during cleanup.
func (c *Client) stopUploadSession() {
	c.uploadSessionMu.Lock()
	pool := c.uploadPool
	c.uploadPool = nil
	c.uploadSessionMu.Unlock()

	for _, ss := range pool {
		if ss.sess != nil {
			ss.sess.Stop()
		}
		if ss.closer != nil {
			ss.closer.Close()
		}
	}
}
