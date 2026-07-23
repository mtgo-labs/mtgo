package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
)

// uploadSessionPool manages N independent MTProto sessions on the home DC for
// parallel file uploads. Each session shares the main session's permanent auth
// key (no DH exchange) but has its own TCP connection and session ID. PFS is
// intentionally disabled, matching mtcute's design: "we do not set temp auth
// keys for media connections, as they are ephemeral and dc-bound."
type uploadSessionPool struct {
	mu      sync.RWMutex
	entries []*dcSessionEntry
	next    atomic.Uint64
	client  *Client
	size    int
}

func newUploadSessionPool(client *Client, size int) *uploadSessionPool {
	return &uploadSessionPool{
		client: client,
		size:   size,
	}
}

// ensureCreated lazily creates the pool sessions. It is safe to call
// concurrently; the first caller creates the pool, subsequent callers reuse it.
func (p *uploadSessionPool) ensureCreated(ctx context.Context) error {
	p.mu.RLock()
	if len(p.entries) >= p.size {
		p.mu.RUnlock()
		return nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	if len(p.entries) >= p.size {
		p.mu.Unlock()
		return nil
	}
	needed := p.size - len(p.entries)
	p.mu.Unlock()

	// Create sessions in parallel. Use a channel to collect entries without
	// holding p.mu during creation (avoids lock-ordering deadlock).
	errCh := make(chan error, needed)
	entryCh := make(chan *dcSessionEntry, needed)
	var wg sync.WaitGroup

	for range needed {
		wg.Add(1)
		go func() {
			defer wg.Done()
			entry, err := p.client.createUploadSession(ctx)
			if err != nil {
				errCh <- err
				return
			}
			entryCh <- entry
			errCh <- nil
		}()
	}
	wg.Wait()
	close(errCh)
	close(entryCh)

	var firstErr error
	for err := range errCh {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		// Close any successfully created entries on failure.
		for entry := range entryCh {
			entry.close()
		}
		return fmt.Errorf("upload pool: %w", firstErr)
	}

	p.mu.Lock()
	for entry := range entryCh {
		p.entries = append(p.entries, entry)
	}
	p.mu.Unlock()
	return nil
}

// rpcClients returns one RPC client per pool entry.
func (p *uploadSessionPool) rpcClients() []*tg.RPCClient {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.entries) == 0 {
		return nil
	}
	rpcs := make([]*tg.RPCClient, len(p.entries))
	for i, e := range p.entries {
		rpcs[i] = e.rpc
	}
	return rpcs
}

// close terminates all pool sessions.
func (p *uploadSessionPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.entries {
		e.close()
	}
	p.entries = nil
}

// entry returns the pool entry at idx, or nil if out of range.
func (p *uploadSessionPool) entry(idx int) *dcSessionEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if idx < 0 || idx >= len(p.entries) {
		return nil
	}
	return p.entries[idx]
}

// isAlive reports whether the entry has a live session.
func (e *dcSessionEntry) isAlive() bool {
	return e != nil && !e.retired.Load() && e.sess != nil && e.sess.IsConnected()
}

// tryReplace replaces a dead entry at idx with a fresh shared-key session.
// Non-blocking: if creation fails, the old (dead) entry remains.
func (p *uploadSessionPool) tryReplace(ctx context.Context, idx int) {
	p.mu.Lock()
	if idx >= len(p.entries) {
		p.mu.Unlock()
		return
	}
	current := p.entries[idx]
	if current.isAlive() {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	// Create replacement outside lock to avoid blocking other readers.
	newEntry, err := p.client.createUploadSession(ctx)
	if err != nil {
		return
	}
	p.mu.Lock()
	if idx < len(p.entries) && !p.entries[idx].isAlive() {
		old := p.entries[idx]
		p.entries[idx] = newEntry
		p.mu.Unlock()
		if old != nil {
			old.close()
		}
	} else {
		p.mu.Unlock()
		newEntry.close()
	}
}

// uploadPoolInvoker implements tg.Invoker with round-robin across pool entries,
// automatic dead-session replacement, and fallback to the main session.
type uploadPoolInvoker struct {
	pool   *uploadSessionPool
	client *Client
	next   atomic.Uint64
}

func (u *uploadPoolInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	size := u.pool.size
	for range size {
		idx := int(u.next.Add(1)-1) % size
		entry := u.pool.entry(idx)
		if !entry.isAlive() {
			u.pool.tryReplace(ctx, idx)
			entry = u.pool.entry(idx)
		}
		if !entry.isAlive() {
			continue
		}
		result, err := entry.rpc.Invoke(ctx, input, decode)
		if err == nil {
			return result, nil
		}
		if isTransferSessionDeadErr(err) {
			entry.retire()
		}
	}
	return u.client.Raw().Invoke(ctx, input, decode)
}

func (u *uploadPoolInvoker) RPCInvokeRaw(ctx context.Context, input tg.TLObject) ([]byte, error) {
	size := u.pool.size
	for range size {
		idx := int(u.next.Add(1)-1) % size
		entry := u.pool.entry(idx)
		if !entry.isAlive() {
			u.pool.tryReplace(ctx, idx)
			entry = u.pool.entry(idx)
		}
		if !entry.isAlive() {
			continue
		}
		data, err := entry.rpc.InvokeWithRawResult(ctx, input)
		if err == nil {
			return data, nil
		}
		if isTransferSessionDeadErr(err) {
			entry.retire()
		}
	}
	return u.client.Raw().InvokeWithRawResult(ctx, input)
}

// rpcClient returns a single RPC client that wraps the pool invoker,
// providing automatic round-robin and dead-session replacement.
func (p *uploadSessionPool) rpcClient() *tg.RPCClient {
	return tg.NewRPCClient(&uploadPoolInvoker{pool: p, client: p.client})
}

// createUploadSession creates a single upload session on the home DC that
// shares the main session's permanent auth key. No DH exchange, no PFS,
// no auth export — the key is copied directly from the main session.
func (c *Client) createUploadSession(ctx context.Context) (*dcSessionEntry, error) {
	homeDC := c.homeDC()
	if homeDC == 0 {
		return nil, ErrNotConnected
	}

	c.mu.RLock()
	mainSess := c.session
	c.mu.RUnlock()
	if mainSess == nil {
		return nil, ErrNotConnected
	}

	authKey := mainSess.AuthKey()
	if len(authKey) == 0 {
		return nil, errors.New("upload pool: main session has no auth key")
	}
	serverSalt := mainSess.ServerSalt()

	dc := session.DataCenter{
		ID:       homeDC,
		TestMode: c.config().TestMode,
		IPv6:     c.config().IPv6,
	}

	// Dial directly to the home DC, bypassing the DCOptionPool to avoid
	// IPv6 endpoint issues and duplicate-pool churn from parallel creation.
	sessionTp, err := c.dialTCPTransportContext(ctx, dc, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("upload pool: transport DC %d: %w", homeDC, err)
	}

	// Create storage with the main session's auth key so the new session
	// can encrypt/decrypt without a fresh DH exchange.
	dcStorage := NewMemoryStorage()
	if err := dcStorage.SetAuthKey(authKey); err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("upload pool: set auth key: %w", err)
	}

	cfg := c.config()
	sess, err := session.NewSession(dc, dcStorage, cfg.Device.DeviceModel,
		cfg.Device.AppVersion, cfg.Device.SystemLangCode, cfg.Device.LangCode)
	if err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("upload pool: create session: %w", err)
	}

	// Wire logger. Do NOT wire PFSInitConnection or reconnect hooks —
	// upload sessions are ephemeral and dc-bound (mtcute pattern).
	if c.Log != nil {
		sess.SetLogger(c.Log)
	}
	sess.SetUpdateHandler(func(obj tg.TLObject) {})
	sess.SetServerSalt(serverSalt)
	sess.SetServerTime(time.Now())

	// Connect with the pre-set auth key. No DH exchange, no PFS.
	if err := sess.Connect(sessionTp, 15*time.Second); err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("upload pool: connect DC %d: %w", homeDC, err)
	}

	c.Log.Debugf("upload session established on DC %d", homeDC)
	entry := newDCSessionEntry(sess, sessionTp, c)
	return entry, nil
}

// uploadPoolSize returns the number of upload sessions to create. It scales
// with file size, matching gogram's countWorkers but more conservative to
// avoid wasting resources: 4 for most files, up to 8 for very large ones.
func uploadPoolSize(fileSize int64, cfgSize int) int {
	if cfgSize > 0 {
		return clampTransferWorkers(cfgSize)
	}
	parts := fileSize / int64(uploadPartSize)
	switch {
	case parts <= 4:
		return 1
	case parts <= 100:
		return 2
	case parts <= 1000:
		return 4
	case parts <= 4000:
		return 6
	default:
		return 8
	}
}
