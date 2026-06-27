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

type dcSessions struct {
	mu        sync.Mutex
	entries   map[int]*dcSessionEntry
	initLocks map[int]*sync.Mutex
}

func newDCSessions() *dcSessions {
	return &dcSessions{
		entries:   make(map[int]*dcSessionEntry),
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

func (d *dcSessions) cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, e := range d.entries {
		e.sess.Stop()
		if e.closer != nil {
			e.closer.Close()
		}
	}
	d.entries = make(map[int]*dcSessionEntry)
}

func (c *Client) dcRPC(ctx context.Context, dcID int) (*tg.RPCClient, error) {
	if dcID <= 0 {
		return c.Raw(), nil
	}

	c.mu.RLock()
	homeDC := c.state.DC()
	c.mu.RUnlock()
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
	c.mu.RUnlock()

	dcStorage := NewMemoryStorage()

	sess, err := session.NewSession(dc, dcStorage, cfg.Device.DeviceModel, cfg.Device.AppVersion, cfg.Device.SystemLangCode, cfg.Device.LangCode)
	if err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("download: create session DC %d: %w", dcID, err)
	}
	configureSessionDispatch(sess, cfg, log)

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

	sess.SetUpdateHandler(func(obj tg.TLObject) {})

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
