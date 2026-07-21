package telegram

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type dcSessionEntry struct {
	sess   *session.Session
	closer ioCloser
	rpc    *tg.RPCClient
	stats  dcSessionStats

	retired   atomic.Bool
	closeOnce sync.Once
}

type dcSessionPool struct {
	mu      sync.RWMutex
	entries []*dcSessionEntry
	next    atomic.Uint64
	rpc     *tg.RPCClient
}

type dcSessions struct {
	mu        sync.Mutex
	entries   map[int]*dcSessionEntry
	pools     map[int]*dcSessionPool
	initLocks map[int]*sync.Mutex
}

func newDCSessions() *dcSessions {
	return &dcSessions{
		entries:   make(map[int]*dcSessionEntry),
		pools:     make(map[int]*dcSessionPool),
		initLocks: make(map[int]*sync.Mutex),
	}
}

func (d *dcSessions) getInitLock(dcID int) *sync.Mutex {
	d.mu.Lock()
	defer d.mu.Unlock()
	mu, ok := d.initLocks[dcID]
	if !ok {
		mu = &sync.Mutex{}
		d.initLocks[dcID] = mu
	}
	return mu
}

func (d *dcSessions) get(dcID int) (*dcSessionEntry, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	e, ok := d.entries[dcID]
	return e, ok
}

func (d *dcSessions) put(dcID int, e *dcSessionEntry) {
	d.mu.Lock()
	d.entries[dcID] = e
	// The per-DC init lock is only needed while the entry is being created;
	// once the entry exists every future caller returns from get() without
	// touching initLocks, so drop it to keep initLocks bounded. Safe because
	// entries are never removed: any later getInitLock(dcID) caller would first
	// hit the entry in get() and never reach getInitLock.
	delete(d.initLocks, dcID)
	d.mu.Unlock()
}

func (d *dcSessions) getPool(dcID int, size int) (*dcSessionPool, bool) {
	d.mu.Lock()
	p, ok := d.pools[dcID]
	d.mu.Unlock()
	if !ok || p == nil || p.len() < size {
		return nil, false
	}
	return p, true
}

func (d *dcSessions) putPool(dcID int, p *dcSessionPool) {
	d.mu.Lock()
	d.pools[dcID] = p
	delete(d.initLocks, dcID)
	d.mu.Unlock()
}

func (d *dcSessions) remove(dcID int) {
	d.mu.Lock()
	e := d.entries[dcID]
	delete(d.entries, dcID)
	p := d.pools[dcID]
	delete(d.pools, dcID)
	delete(d.initLocks, dcID)
	d.mu.Unlock()

	if e != nil {
		e.close()
	}
	if p != nil {
		for _, e := range p.snapshot(0) {
			e.close()
		}
	}
}

func (d *dcSessions) cleanup() {
	d.mu.Lock()
	entries := d.entries
	pools := d.pools
	d.entries = make(map[int]*dcSessionEntry)
	d.pools = make(map[int]*dcSessionPool)
	d.initLocks = make(map[int]*sync.Mutex)
	d.mu.Unlock()

	for _, e := range entries {
		e.close()
	}
	for _, p := range pools {
		for _, e := range p.snapshot(0) {
			e.close()
		}
	}
}

func (c *Client) homeDC() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.session != nil {
		return c.session.DC().ID
	}
	return c.state.DC()
}

func (c *Client) dcRPC(ctx context.Context, dcID int) (*tg.RPCClient, error) {
	if dcID <= 0 {
		return c.Raw(), nil
	}

	homeDC := c.homeDC()
	if dcID == homeDC || homeDC == 0 {
		return c.Raw(), nil
	}
	poolSize := min(max(c.config().DCPoolSize, 1), 16)
	if poolSize > 1 {
		pool, err := c.ensureDCRPCPool(ctx, dcID, poolSize)
		if err != nil {
			return nil, err
		}
		return pool.rpc, nil
	}
	if c.dcAuthManager != nil {
		c.dcAuthManager.UpdateMainDC(homeDC)
		if c.dcAuthManager.IsAuthorized(dcID) {
			if entry, ok := c.dcSessions.get(dcID); ok {
				return entry.rpc, nil
			}
		}
	}

	if entry, ok := c.dcSessions.get(dcID); ok {
		return entry.rpc, nil
	}

	// Use per-DC mutex to avoid serializing unrelated DC session creations.
	initMu := c.dcSessions.getInitLock(dcID)
	initMu.Lock()
	defer initMu.Unlock()

	if entry, ok := c.dcSessions.get(dcID); ok {
		return entry.rpc, nil
	}

	entry, err := c.createDCSession(ctx, dcID)
	if err != nil {
		return nil, err
	}

	c.dcSessions.put(dcID, entry)
	return entry.rpc, nil
}

func (c *Client) dcRPCPool(ctx context.Context, dcID int, size int) ([]*tg.RPCClient, error) {
	if size <= 1 || dcID <= 0 {
		rpc, err := c.dcRPC(ctx, dcID)
		if err != nil {
			return nil, err
		}
		return []*tg.RPCClient{rpc}, nil
	}

	homeDC := c.homeDC()
	// Same-DC: the main session multiplexes concurrent requests natively and
	// has robust reconnection logic. Return it for every worker instead of
	// creating fragile side sessions that share the auth key and cascade-fail
	// when one is replaced (killing sessions other workers still use).
	if dcID == homeDC || homeDC == 0 {
		mainRPC := c.Raw()
		rpcs := make([]*tg.RPCClient, size)
		for i := range rpcs {
			rpcs[i] = mainRPC
		}
		return rpcs, nil
	}

	pool, err := c.ensureDCRPCPool(ctx, dcID, min(size, 16))
	if err != nil {
		return nil, err
	}
	return pool.rpcClients(size), nil
}

func (c *Client) ensureDCRPCPool(ctx context.Context, dcID int, size int) (*dcSessionPool, error) {
	size = min(max(size, 1), 16)
	if pool, ok := c.dcSessions.getPool(dcID, size); ok {
		return pool, nil
	}

	initMu := c.dcSessions.getInitLock(dcID)
	initMu.Lock()
	defer initMu.Unlock()
	if pool, ok := c.dcSessions.getPool(dcID, size); ok {
		return pool, nil
	}

	pool, _ := c.dcSessions.getPool(dcID, 1)
	entries := make([]*dcSessionEntry, 0, size)
	if pool != nil {
		entries = append(entries, pool.snapshot(0)...)
	}
	created := make([]*dcSessionEntry, 0, size-len(entries))
	for len(entries) < size {
		entry, err := c.createDCSession(ctx, dcID)
		if err != nil {
			for _, newEntry := range created {
				newEntry.close()
			}
			return nil, err
		}
		entries = append(entries, entry)
		created = append(created, entry)
	}

	if pool == nil {
		pool = &dcSessionPool{entries: entries}
		pool.rpc = tg.NewRPCClient(&dcPoolInvoker{pool: pool, client: c, dcID: dcID})
		c.dcSessions.putPool(dcID, pool)
	} else {
		pool.mu.Lock()
		pool.entries = entries
		pool.mu.Unlock()
	}
	return pool, nil
}

func (c *Client) replaceDCRPCPoolEntry(ctx context.Context, dcID int, size int, idx int) (*tg.RPCClient, error) {
	return c.replaceDCRPCPoolEntryIfCurrent(ctx, dcID, size, idx, nil)
}

func (c *Client) replaceDCRPCPoolEntryIfCurrent(
	ctx context.Context,
	dcID int,
	size int,
	idx int,
	expected *dcSessionEntry,
) (*tg.RPCClient, error) {
	if size <= 1 || dcID <= 0 {
		if dcID > 0 {
			c.dcSessions.remove(dcID)
		}
		return c.dcRPC(ctx, dcID)
	}

	homeDC := c.homeDC()
	// Same-DC: cannot replace the main session; return it as-is.
	if dcID == homeDC || homeDC == 0 {
		return c.Raw(), nil
	}

	if _, ok := c.dcSessions.getPool(dcID, size); !ok {
		pool, err := c.ensureDCRPCPool(ctx, dcID, size)
		if err != nil {
			return nil, err
		}
		entries := pool.snapshot(size)
		return entries[idx%len(entries)].rpc, nil
	}

	initMu := c.dcSessions.getInitLock(dcID)
	initMu.Lock()
	defer initMu.Unlock()
	pool, ok := c.dcSessions.getPool(dcID, size)
	if !ok {
		return nil, ErrNotConnected
	}
	if expected != nil {
		current := pool.entry(idx)
		if current != expected {
			if current == nil {
				return nil, ErrNotConnected
			}
			return current.rpc, nil
		}
	}

	entry, err := c.createDCSession(ctx, dcID)
	if err != nil {
		return nil, err
	}

	old := pool.replace(idx, entry)
	old.retire()

	return entry.rpc, nil
}

func (c *Client) createDCSession(ctx context.Context, dcID int) (*dcSessionEntry, error) {
	dc := session.DataCenter{
		ID:       dcID,
		TestMode: c.config().TestMode,
		IPv6:     c.config().IPv6,
	}
	addr := dc.Address()
	if addr == "" {
		return nil, fmt.Errorf("download: %w: %d", ErrUnknownDC, dcID)
	}
	port := dc.Port()

	sessionTp, err := c.dialTransport(dc, 15*time.Second, c.testDialer)
	if err != nil {
		return nil, fmt.Errorf("download: transport DC %d (%s:%d): %w", dcID, addr, port, err)
	}

	cfg := c.config()
	c.mu.RLock()
	mainSess := c.session
	homeDC := c.state.DC()
	if mainSess != nil {
		homeDC = mainSess.DC().ID
	}
	c.mu.RUnlock()

	dcStorage := NewMemoryStorage()

	sess, err := session.NewSession(dc, dcStorage, cfg.Device.DeviceModel, cfg.Device.AppVersion, cfg.Device.SystemLangCode, cfg.Device.LangCode)
	if err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("download: create session DC %d: %w", dcID, err)
	}
	configureSessionDispatch(sess, c)
	configureSessionHealth(sess, cfg, c.connMetrics)
	sess.SetUpdateHandler(func(obj tg.TLObject) {})

	if mainSess != nil && dcID == homeDC {
		authKey := mainSess.AuthKey()
		if pfs := mainSess.PFS(); pfs != nil {
			authKey = pfs.PermKey()
		}
		if len(authKey) == 0 {
			sessionTp.Close()
			return nil, fmt.Errorf("download: no auth key for same-DC sender %d", dcID)
		}
		sess.SetAuthKey(authKey)
		sess.SetServerSalt(mainSess.ServerSalt())
		if err := c.prepareSessionPFS(sess, dcStorage, dc, sessionTp, authKey); err != nil {
			sessionTp.Close()
			return nil, fmt.Errorf("download: prepare PFS for DC %d: %w", dcID, err)
		}
		if err := sess.Connect(sessionTp, 15*time.Second); err != nil {
			sessionTp.Close()
			return nil, fmt.Errorf("download: start same-DC session DC %d: %w", dcID, err)
		}
		bindCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := c.bindSessionPFS(bindCtx, sess); err != nil {
			sess.Stop()
			sessionTp.Close()
			return nil, fmt.Errorf("download: bind PFS for DC %d: %w", dcID, err)
		}
		c.Log.Infof("Same-DC session established for DC %d", dcID)
		return newDCSessionEntry(sess, sessionTp, c), nil
	}

	auth := &session.Auth{
		DC:       dcID,
		TestMode: dc.TestMode,
	}
	if c.keySet != nil {
		auth.SetKeySet(c.keySet)
	}
	result, err := auth.Create(sessionTp)
	if err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("download: DH exchange DC %d: %w", dcID, err)
	}
	sess.SetAuthKey(result.AuthKey)
	sess.SetServerSalt(result.ServerSalt)
	sess.SetServerTime(time.Unix(int64(result.ServerTime), 0))
	if err := c.prepareSessionPFS(sess, dcStorage, dc, sessionTp, result.AuthKey); err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("download: prepare PFS for DC %d: %w", dcID, err)
	}

	if err := sess.Connect(sessionTp, 15*time.Second); err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("download: start session DC %d: %w", dcID, err)
	}
	bindCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := c.bindSessionPFS(bindCtx, sess); err != nil {
		sess.Stop()
		sessionTp.Close()
		return nil, fmt.Errorf("download: bind PFS for DC %d: %w", dcID, err)
	}

	c.Log.Infof("DC session established for DC %d", dcID)

	entry := newDCSessionEntry(sess, sessionTp, c)
	rpc := entry.rpc

	if c.dcAuthManager != nil {
		c.dcAuthManager.SetImporter(dcID, func(ctx context.Context, id int64, b []byte) error {
			_, err := rpc.AuthImportAuthorization(ctx, &tg.AuthImportAuthorizationRequest{
				ID:    id,
				Bytes: b,
			})
			return err
		})
		defer c.dcAuthManager.SetImporter(dcID, nil)
		if err := c.dcAuthManager.DCLoop(ctx, dcID); err != nil {
			sess.Stop()
			sessionTp.Close()
			return nil, fmt.Errorf("download: auth transfer for DC %d: %w", dcID, err)
		}
	} else {
		exportResult, err := c.Raw().AuthExportAuthorization(ctx, &tg.AuthExportAuthorizationRequest{
			DCID: int32(dcID),
		})
		if err != nil {
			sess.Stop()
			sessionTp.Close()
			return nil, fmt.Errorf("download: export auth for DC %d: %w", dcID, err)
		}
		_, err = rpc.AuthImportAuthorization(ctx, &tg.AuthImportAuthorizationRequest{
			ID:    exportResult.ID,
			Bytes: exportResult.Bytes,
		})
		if err != nil {
			sess.Stop()
			sessionTp.Close()
			return nil, fmt.Errorf("download: import auth on DC %d: %w", dcID, err)
		}
	}

	c.Log.Infof("Auth transfer complete for DC %d", dcID)

	return entry, nil
}

type dcSessionInvoker struct {
	sess    *session.Session
	client  *Client
	entry   *dcSessionEntry
	apiInit atomic.Bool
}

func (d *dcSessionInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (result tg.TLObject, err error) {
	started := time.Now()
	defer func() {
		if d.client != nil {
			d.client.observeRPC(ctx, input, 1, started, err)
		}
	}()
	if d.entry != nil {
		started := d.entry.beginRequest()
		defer func() { d.entry.endRequest(started, err, d.client.config().EndpointCoolDown) }()
	}

	deadline, ok := ctx.Deadline()
	timeout := time.Duration(0)
	if ok {
		timeout = max(time.Until(deadline), 0)
	} else {
		timeout = 60 * time.Second
	}

	query, initializesAPI := prepareAPIQuery(d.client.config(), d.apiInit.Load(), input)

	result, err = d.sess.Invoke(ctx, query, 2, timeout)
	if err != nil {
		return nil, err
	}

	if rpcErr, ok := result.(*tg.RPCError); ok {
		parsed := tgerr.New(int(rpcErr.ErrorCode), rpcErr.ErrorMessage)
		return nil, parsed
	}

	if initializesAPI {
		d.apiInit.Store(true)
	}

	return result, nil
}

func (d *dcSessionInvoker) RPCInvokeRaw(ctx context.Context, input tg.TLObject) (data []byte, err error) {
	started := time.Now()
	defer func() {
		if d.client != nil {
			d.client.observeRPC(ctx, input, 1, started, err)
		}
	}()
	if d.entry != nil {
		started := d.entry.beginRequest()
		defer func() { d.entry.endRequest(started, err, d.client.config().EndpointCoolDown) }()
	}

	query, initializesAPI := prepareAPIQuery(d.client.config(), d.apiInit.Load(), input)

	data, err = d.sess.InvokeRaw(ctx, query, 2, 60*time.Second)
	if err != nil {
		return nil, err
	}

	if initializesAPI {
		d.apiInit.Store(true)
	}

	return data, nil
}

type ioCloser interface {
	Close() error
}
