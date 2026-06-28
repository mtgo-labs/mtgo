package telegram

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type dcSessionEntry struct {
	sess   *session.Session
	closer ioCloser
	rpc    *tg.RPCClient
}

type dcSessionPool struct {
	entries []*dcSessionEntry
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
	defer d.mu.Unlock()
	p, ok := d.pools[dcID]
	if !ok || p == nil || len(p.entries) < size {
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
		if e.sess != nil {
			e.sess.Stop()
		}
		if e.closer != nil {
			e.closer.Close()
		}
	}
	if p != nil {
		for _, e := range p.entries {
			if e.sess != nil {
				e.sess.Stop()
			}
			if e.closer != nil {
				e.closer.Close()
			}
		}
	}
}

func (d *dcSessions) cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, e := range d.entries {
		e.sess.Stop()
		if e.closer != nil {
			e.closer.Close()
		}
	}
	for _, p := range d.pools {
		for _, e := range p.entries {
			e.sess.Stop()
			if e.closer != nil {
				e.closer.Close()
			}
		}
	}
	d.entries = make(map[int]*dcSessionEntry)
	d.pools = make(map[int]*dcSessionPool)
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

	if pool, ok := c.dcSessions.getPool(dcID, size); ok {
		rpcs := make([]*tg.RPCClient, size)
		for i := range rpcs {
			rpcs[i] = pool.entries[i].rpc
		}
		return rpcs, nil
	}

	initMu := c.dcSessions.getInitLock(dcID)
	initMu.Lock()
	defer initMu.Unlock()

	if pool, ok := c.dcSessions.getPool(dcID, size); ok {
		rpcs := make([]*tg.RPCClient, size)
		for i := range rpcs {
			rpcs[i] = pool.entries[i].rpc
		}
		return rpcs, nil
	}

	entries := make([]*dcSessionEntry, 0, size)
	for i := 0; i < size; i++ {
		entry, err := c.createDCSession(ctx, dcID)
		if err != nil {
			for _, e := range entries {
				if e.sess != nil {
					e.sess.Stop()
				}
				if e.closer != nil {
					e.closer.Close()
				}
			}
			return nil, err
		}
		entries = append(entries, entry)
	}

	c.dcSessions.putPool(dcID, &dcSessionPool{entries: entries})

	rpcs := make([]*tg.RPCClient, len(entries))
	for i, e := range entries {
		rpcs[i] = e.rpc
	}
	return rpcs, nil
}

func (c *Client) replaceDCRPCPoolEntry(ctx context.Context, dcID int, size int, idx int) (*tg.RPCClient, error) {
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

	initMu := c.dcSessions.getInitLock(dcID)
	initMu.Lock()
	defer initMu.Unlock()

	pool, ok := c.dcSessions.getPool(dcID, size)
	if !ok {
		entries := make([]*dcSessionEntry, 0, size)
		for i := 0; i < size; i++ {
			entry, err := c.createDCSession(ctx, dcID)
			if err != nil {
				for _, e := range entries {
					if e.sess != nil {
						e.sess.Stop()
					}
					if e.closer != nil {
						e.closer.Close()
					}
				}
				return nil, err
			}
			entries = append(entries, entry)
		}
		c.dcSessions.putPool(dcID, &dcSessionPool{entries: entries})
		return entries[idx%len(entries)].rpc, nil
	}

	entry, err := c.createDCSession(ctx, dcID)
	if err != nil {
		return nil, err
	}

	idx %= len(pool.entries)
	c.dcSessions.mu.Lock()
	old := pool.entries[idx]
	pool.entries[idx] = entry
	c.dcSessions.mu.Unlock()

	if old != nil {
		if old.sess != nil {
			old.sess.Stop()
		}
		if old.closer != nil {
			old.closer.Close()
		}
	}

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
		return nil, fmt.Errorf("download: unknown dc_id: %d", dcID)
	}
	port := dc.Port()

	d := c.dialer
	if c.testDialer != nil {
		d = c.testDialer
	}

	conn, err := d.Dial("tcp", fmt.Sprintf("%s:%d", addr, port), 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("download: dial DC %d (%s:%d): %w", dcID, addr, port, err)
	}

	tp, err := c.createTransport(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("download: transport DC %d: %w", dcID, err)
	}
	if err := tp.Connect(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("download: transport handshake DC %d: %w", dcID, err)
	}

	sessionTp := newSessionTransport(tp, conn)

	c.mu.RLock()
	cfg := c.cfg
	log := c.Log
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
	configureSessionDispatch(sess, cfg, log)
	sess.SetUpdateHandler(func(obj tg.TLObject) {})

	if mainSess != nil && dcID == homeDC {
		authKey := mainSess.AuthKey()
		if len(authKey) == 0 {
			sessionTp.Close()
			return nil, fmt.Errorf("download: no auth key for same-DC sender %d", dcID)
		}
		sess.SetAuthKey(authKey)
		sess.SetServerSalt(mainSess.ServerSalt())
		if err := sess.Connect(sessionTp, 15*time.Second); err != nil {
			sessionTp.Close()
			return nil, fmt.Errorf("download: start same-DC session DC %d: %w", dcID, err)
		}
		c.Log.Infof("Same-DC session established for DC %d", dcID)
		return &dcSessionEntry{
			sess:   sess,
			closer: sessionTp,
			rpc:    tg.NewRPCClient(&dcSessionInvoker{sess: sess, client: c}),
		}, nil
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

	if err := sess.Connect(sessionTp, 15*time.Second); err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("download: start session DC %d: %w", dcID, err)
	}

	c.Log.Infof("DC session established for DC %d", dcID)

	invoker := &dcSessionInvoker{sess: sess, client: c}
	rpc := tg.NewRPCClient(invoker)

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

	return &dcSessionEntry{
		sess:   sess,
		closer: sessionTp,
		rpc:    rpc,
	}, nil
}

type dcSessionInvoker struct {
	sess    *session.Session
	client  *Client
	apiInit bool
}

func (d *dcSessionInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	deadline, ok := ctx.Deadline()
	timeout := time.Duration(0)
	if ok {
		timeout = time.Until(deadline)
		if timeout < 0 {
			timeout = 0
		}
	} else {
		timeout = 60 * time.Second
	}

	query := input
	if !d.apiInit && needsInitConnection(input) {
		query = wrapInitConnection(d.client.config(), input)
	}

	result, err := d.sess.Invoke(ctx, query, 2, timeout)
	if err != nil {
		return nil, err
	}

	if rpcErr, ok := result.(*tg.RPCError); ok {
		parsed := tgerr.New(int(rpcErr.ErrorCode), rpcErr.ErrorMessage)
		return nil, parsed
	}

	if !d.apiInit && needsInitConnection(input) {
		d.apiInit = true
	}

	return result, nil
}

func (d *dcSessionInvoker) RPCInvokeRaw(ctx context.Context, input tg.TLObject) ([]byte, error) {
	query := input
	if !d.apiInit && needsInitConnection(input) {
		query = wrapInitConnection(d.client.config(), input)
	}

	data, err := d.sess.InvokeRaw(ctx, query, 2, 60*time.Second)
	if err != nil {
		return nil, err
	}

	if !d.apiInit && needsInitConnection(input) {
		d.apiInit = true
	}

	return data, nil
}

type ioCloser interface {
	Close() error
}
