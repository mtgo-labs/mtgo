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
	"github.com/mtgo-labs/mtgo/tgerr"
)

var errDCBecameHome = errors.New("telegram: requested DC became the main DC")

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

type dcSessionCreation struct {
	mu            sync.Mutex
	cancel        context.CancelFunc
	closer        ioCloser
	done          chan struct{}
	doneOnce      sync.Once
	stopped       bool
	waitForFinish bool
}

func (c *dcSessionCreation) setCloser(closer ioCloser) bool {
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		if closer != nil {
			_ = closer.Close()
		}
		return false
	}
	c.closer = closer
	c.mu.Unlock()
	return true
}

func (c *dcSessionCreation) stop() <-chan struct{} {
	c.mu.Lock()
	if c.stopped {
		waitForFinish := c.waitForFinish
		done := c.done
		c.mu.Unlock()
		if waitForFinish {
			return done
		}
		return nil
	}
	c.stopped = true
	c.waitForFinish = c.closer == nil
	cancel := c.cancel
	closer := c.closer
	waitForFinish := c.waitForFinish
	done := c.done
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if closer != nil {
		_ = closer.Close()
	}
	if waitForFinish {
		return done
	}
	return nil
}

func (c *dcSessionCreation) finish() {
	c.doneOnce.Do(func() { close(c.done) })
}

type dcSessions struct {
	mu         sync.Mutex
	entries    map[int]*dcSessionEntry
	pools      map[int]*dcSessionPool
	initLocks  map[int]*sync.Mutex
	creations  map[*dcSessionCreation]uint64
	generation uint64
}

func newDCSessions() *dcSessions {
	return &dcSessions{
		entries:   make(map[int]*dcSessionEntry),
		pools:     make(map[int]*dcSessionPool),
		initLocks: make(map[int]*sync.Mutex),
		creations: make(map[*dcSessionCreation]uint64),
	}
}

func (d *dcSessions) beginCreation(parent context.Context, generation uint64) (*dcSessionCreation, context.Context, bool) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	creation := &dcSessionCreation{cancel: cancel, done: make(chan struct{})}
	d.mu.Lock()
	if d.generation != generation {
		d.mu.Unlock()
		cancel()
		return nil, nil, false
	}
	d.creations[creation] = generation
	d.mu.Unlock()
	return creation, ctx, true
}

func (d *dcSessions) finishCreation(creation *dcSessionCreation) {
	if creation == nil {
		return
	}
	d.mu.Lock()
	delete(d.creations, creation)
	d.mu.Unlock()
	creation.finish()
}

func stopDCSessionCreations(creations []*dcSessionCreation) []<-chan struct{} {
	waits := make([]<-chan struct{}, 0, len(creations))
	for _, creation := range creations {
		if done := creation.stop(); done != nil {
			waits = append(waits, done)
		}
	}
	return waits
}

func waitDCSessionCreations(waits []<-chan struct{}) {
	for _, done := range waits {
		<-done
	}
}

func (d *dcSessions) invalidateCreationsLocked(generation uint64) []*dcSessionCreation {
	creations := make([]*dcSessionCreation, 0, len(d.creations))
	for creation, candidateGeneration := range d.creations {
		if candidateGeneration <= generation {
			creations = append(creations, creation)
		}
	}
	return creations
}

func (d *dcSessions) getInitLock(dcID int) (*sync.Mutex, uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	mu, ok := d.initLocks[dcID]
	if !ok {
		mu = &sync.Mutex{}
		d.initLocks[dcID] = mu
	}
	return mu, d.generation
}

func (d *dcSessions) isGeneration(generation uint64) bool {
	d.mu.Lock()
	current := d.generation == generation
	d.mu.Unlock()
	return current
}

func (d *dcSessions) get(dcID int) (*dcSessionEntry, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	e, ok := d.entries[dcID]
	return e, ok
}

func (d *dcSessions) putIfGeneration(dcID int, e *dcSessionEntry, generation uint64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.generation != generation {
		return false
	}
	d.entries[dcID] = e
	// The per-DC init lock is only needed while the entry is being created;
	// once the entry exists every future caller returns from get() without
	// touching initLocks, so drop it to keep initLocks bounded. Safe because
	// entries are never removed: any later getInitLock(dcID) caller would first
	// hit the entry in get() and never reach getInitLock.
	delete(d.initLocks, dcID)
	return true
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

func (d *dcSessions) putPoolIfGeneration(dcID int, p *dcSessionPool, generation uint64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.generation != generation {
		return false
	}
	d.pools[dcID] = p
	delete(d.initLocks, dcID)
	return true
}

func (d *dcSessions) updatePoolIfGeneration(
	dcID int,
	pool *dcSessionPool,
	entries []*dcSessionEntry,
	generation uint64,
) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.generation != generation || d.pools[dcID] != pool {
		return false
	}
	pool.mu.Lock()
	pool.entries = entries
	pool.mu.Unlock()
	return true
}

func (d *dcSessions) replacePoolEntryIfGeneration(
	dcID int,
	pool *dcSessionPool,
	idx int,
	expected *dcSessionEntry,
	replacement *dcSessionEntry,
	generation uint64,
) (old *dcSessionEntry, current *dcSessionEntry, replaced bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.generation != generation || d.pools[dcID] != pool {
		return nil, nil, false
	}
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if len(pool.entries) == 0 {
		return nil, nil, false
	}
	idx %= len(pool.entries)
	current = pool.entries[idx]
	if expected != nil && current != expected {
		return nil, current, false
	}
	pool.entries[idx] = replacement
	return current, replacement, true
}

func (d *dcSessions) remove(dcID int) {
	d.mu.Lock()
	oldGeneration := d.generation
	d.generation++
	creations := d.invalidateCreationsLocked(oldGeneration)
	e := d.entries[dcID]
	delete(d.entries, dcID)
	p := d.pools[dcID]
	delete(d.pools, dcID)
	delete(d.initLocks, dcID)
	d.mu.Unlock()
	waits := stopDCSessionCreations(creations)

	if e != nil {
		e.close()
	}
	if p != nil {
		for _, e := range p.snapshot(0) {
			e.close()
		}
	}
	waitDCSessionCreations(waits)
}

func (d *dcSessions) removeEntryIfCurrent(dcID int, expected *dcSessionEntry) bool {
	d.mu.Lock()
	if d.entries[dcID] != expected {
		d.mu.Unlock()
		return false
	}
	delete(d.entries, dcID)
	delete(d.initLocks, dcID)
	d.mu.Unlock()
	return true
}

func (d *dcSessions) cleanup(waitForDial ...bool) {
	shouldWait := true
	if len(waitForDial) > 0 {
		shouldWait = waitForDial[0]
	}
	d.mu.Lock()
	oldGeneration := d.generation
	d.generation++
	creations := d.invalidateCreationsLocked(oldGeneration)
	entries := d.entries
	pools := d.pools
	d.entries = make(map[int]*dcSessionEntry)
	d.pools = make(map[int]*dcSessionPool)
	d.initLocks = make(map[int]*sync.Mutex)
	d.mu.Unlock()
	waits := stopDCSessionCreations(creations)

	for _, e := range entries {
		e.close()
	}
	for _, p := range pools {
		for _, e := range p.snapshot(0) {
			e.close()
		}
	}
	if shouldWait {
		waitDCSessionCreations(waits)
	}
}

func (c *Client) createDCSessionCandidate(
	ctx context.Context,
	dcID int,
	generation uint64,
) (*dcSessionEntry, func(), error) {
	mainSess, ok := c.activeDCSessionOwner(generation)
	if !ok {
		return nil, nil, ErrNotConnected
	}
	creation, creationCtx, ok := c.dcSessions.beginCreation(ctx, generation)
	if !ok {
		return nil, nil, ErrNotConnected
	}
	release := func() { c.dcSessions.finishCreation(creation) }
	entry, err := c.createDCSession(creationCtx, dcID, generation, mainSess, creation)
	if err != nil {
		release()
		return nil, nil, err
	}
	return entry, release, nil
}

func (c *Client) activeDCSessionOwner(generation uint64) (*session.Session, bool) {
	if c == nil || c.dcSessions == nil || c.explicitLogout.Load() || c.authLossError() != nil || !c.state.IsConnected() {
		return nil, false
	}
	c.mu.RLock()
	mainSess := c.session
	c.mu.RUnlock()
	if !c.ownsDCSessionOwner(mainSess, generation) {
		return nil, false
	}
	return mainSess, true
}

func (c *Client) ownsDCSessionOwner(mainSess *session.Session, generation uint64) bool {
	if mainSess == nil || c == nil || c.dcSessions == nil ||
		c.explicitLogout.Load() || c.authLossError() != nil ||
		!c.state.IsConnected() || !c.dcSessions.isGeneration(generation) {
		return false
	}
	c.mu.RLock()
	owned := c.session == mainSess
	c.mu.RUnlock()
	return owned && c.state.IsConnected() && !c.explicitLogout.Load() &&
		c.authLossError() == nil && c.dcSessions.isGeneration(generation)
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
			if errors.Is(err, errDCBecameHome) {
				return c.Raw(), nil
			}
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
	initMu, generation := c.dcSessions.getInitLock(dcID)
	initMu.Lock()
	defer initMu.Unlock()

	if entry, ok := c.dcSessions.get(dcID); ok {
		return entry.rpc, nil
	}

	entry, release, err := c.createDCSessionCandidate(ctx, dcID, generation)
	if err != nil {
		if errors.Is(err, errDCBecameHome) {
			return c.Raw(), nil
		}
		return nil, err
	}
	defer release()

	if !c.dcSessions.putIfGeneration(dcID, entry, generation) {
		entry.close()
		if current, ok := c.dcSessions.get(dcID); ok {
			return current.rpc, nil
		}
		return nil, ErrNotConnected
	}
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
		if errors.Is(err, errDCBecameHome) {
			mainRPC := c.Raw()
			rpcs := make([]*tg.RPCClient, size)
			for i := range rpcs {
				rpcs[i] = mainRPC
			}
			return rpcs, nil
		}
		return nil, err
	}
	return pool.rpcClients(size), nil
}

func (c *Client) ensureDCRPCPool(ctx context.Context, dcID int, size int) (*dcSessionPool, error) {
	size = min(max(size, 1), 16)
	if pool, ok := c.dcSessions.getPool(dcID, size); ok {
		return pool, nil
	}

	initMu, generation := c.dcSessions.getInitLock(dcID)
	initMu.Lock()
	defer initMu.Unlock()
	if pool, ok := c.dcSessions.getPool(dcID, size); ok {
		return pool, nil
	}
	if !c.dcSessions.isGeneration(generation) {
		return nil, ErrNotConnected
	}

	pool, _ := c.dcSessions.getPool(dcID, 1)
	entries := make([]*dcSessionEntry, 0, size)
	if pool != nil {
		entries = append(entries, pool.snapshot(0)...)
	}
	created := make([]*dcSessionEntry, 0, size-len(entries))
	releases := make([]func(), 0, size-len(entries))
	defer func() {
		for _, release := range releases {
			release()
		}
	}()
	for len(entries) < size {
		entry, release, err := c.createDCSessionCandidate(ctx, dcID, generation)
		if err != nil {
			for _, newEntry := range created {
				newEntry.close()
			}
			return nil, err
		}
		entries = append(entries, entry)
		created = append(created, entry)
		releases = append(releases, release)
	}

	if pool == nil {
		pool = &dcSessionPool{entries: entries}
		pool.rpc = tg.NewRPCClient(&dcPoolInvoker{pool: pool, client: c, dcID: dcID})
		if !c.dcSessions.putPoolIfGeneration(dcID, pool, generation) {
			for _, newEntry := range created {
				newEntry.close()
			}
			if current, ok := c.dcSessions.getPool(dcID, size); ok {
				return current, nil
			}
			return nil, ErrNotConnected
		}
	} else {
		if !c.dcSessions.updatePoolIfGeneration(dcID, pool, entries, generation) {
			for _, newEntry := range created {
				newEntry.close()
			}
			if current, ok := c.dcSessions.getPool(dcID, size); ok {
				return current, nil
			}
			return nil, ErrNotConnected
		}
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

	initMu, generation := c.dcSessions.getInitLock(dcID)
	initMu.Lock()
	defer initMu.Unlock()
	if !c.dcSessions.isGeneration(generation) {
		return nil, ErrNotConnected
	}
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

	entry, release, err := c.createDCSessionCandidate(ctx, dcID, generation)
	if err != nil {
		if errors.Is(err, errDCBecameHome) {
			return c.Raw(), nil
		}
		return nil, err
	}
	defer release()

	old, current, replaced := c.dcSessions.replacePoolEntryIfGeneration(dcID, pool, idx, expected, entry, generation)
	if !replaced {
		entry.close()
		if current != nil {
			return current.rpc, nil
		}
		if active, ok := c.dcSessions.getPool(dcID, 1); ok {
			entries := active.snapshot(0)
			if len(entries) != 0 {
				return entries[idx%len(entries)].rpc, nil
			}
		}
		return nil, ErrNotConnected
	}
	if old != nil {
		old.retire()
	}

	return entry.rpc, nil
}

func (c *Client) createDCSession(
	ctx context.Context,
	dcID int,
	generation uint64,
	mainSess *session.Session,
	creation *dcSessionCreation,
) (*dcSessionEntry, error) {
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

	sessionTp, err := c.dialTransportContext(ctx, dc, 15*time.Second, c.testDialer)
	if err != nil {
		return nil, fmt.Errorf("download: transport DC %d (%s:%d): %w", dcID, addr, port, err)
	}
	if !creation.setCloser(sessionTp) {
		return nil, ErrNotConnected
	}
	if !c.ownsDCSessionOwner(mainSess, generation) {
		sessionTp.Close()
		return nil, ErrNotConnected
	}

	cfg := c.config()
	c.mu.RLock()
	homeDC := c.state.DC()
	if c.session != mainSess {
		mainSess = nil
	} else if mainSess != nil {
		homeDC = mainSess.DC().ID
	}
	c.mu.RUnlock()
	if mainSess == nil {
		sessionTp.Close()
		return nil, ErrNotConnected
	}
	if dcID == homeDC {
		sessionTp.Close()
		return nil, errDCBecameHome
	}

	dcStorage := NewMemoryStorage()

	sess, err := session.NewSession(dc, dcStorage, cfg.Device.DeviceModel, cfg.Device.AppVersion, cfg.Device.SystemLangCode, cfg.Device.LangCode)
	if err != nil {
		sessionTp.Close()
		return nil, fmt.Errorf("download: create session DC %d: %w", dcID, err)
	}
	configureSessionDispatch(sess, c)
	configureSessionHealth(sess, cfg, c.connMetrics)
	sess.SetUpdateHandler(func(obj tg.TLObject) {})

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

	// Each auxiliary session owns a distinct permanent key, so authorization
	// must be exported/imported for every candidate. A per-DC "authorized"
	// cache would incorrectly skip import for pool entries 2..N and replacements.
	if !c.ownsDCSessionOwner(mainSess, generation) {
		sess.Stop()
		sessionTp.Close()
		return nil, ErrNotConnected
	}
	var exportResult *tg.AuthExportedAuthorization
	err = retryDCAuthorization(ctx, func() error {
		var exportErr error
		exportResult, exportErr = c.Raw().AuthExportAuthorization(ctx, &tg.AuthExportAuthorizationRequest{
			DCID: int32(dcID),
		})
		return exportErr
	})
	if err != nil {
		sess.Stop()
		sessionTp.Close()
		return nil, fmt.Errorf("download: export auth for DC %d: %w", dcID, err)
	}
	err = retryDCAuthorization(ctx, func() error {
		_, importErr := rpc.AuthImportAuthorization(ctx, &tg.AuthImportAuthorizationRequest{
			ID:    exportResult.ID,
			Bytes: exportResult.Bytes,
		})
		return importErr
	})
	if err != nil {
		sess.Stop()
		sessionTp.Close()
		return nil, fmt.Errorf("download: import auth on DC %d: %w", dcID, err)
	}

	c.Log.Infof("Auth transfer complete for DC %d", dcID)

	return entry, nil
}

func retryDCAuthorization(ctx context.Context, call func() error) error {
	return retryFloodWait(ctx, call)
}

type dcSessionInvoker struct {
	sess              *session.Session
	client            *Client
	entry             *dcSessionEntry
	suppressTelemetry bool
	apiInit           atomic.Bool
}

func (d *dcSessionInvoker) retireOnFailure(err error) {
	if d == nil || d.entry == nil || d.client == nil || !isDCConnectionFailure(err) {
		return
	}
	d.entry.retire()
	if d.entry.sess != nil {
		d.client.dcSessions.removeEntryIfCurrent(d.entry.sess.DC().ID, d.entry)
	}
}

func (d *dcSessionInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (result tg.TLObject, err error) {
	started := time.Now()
	defer func() {
		if d.client != nil && !d.suppressTelemetry {
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

	err = retryTransferFloodWait(ctx, func() error {
		var invokeErr error
		result, invokeErr = d.sess.Invoke(ctx, query, 2, timeout)
		if invokeErr != nil {
			return invokeErr
		}
		if transferFloodRetryEnabled(ctx) {
			rpcErr, ok := result.(*tg.RPCError)
			if !ok {
				return nil
			}
			parsed := tgerr.New(int(rpcErr.ErrorCode), rpcErr.ErrorMessage)
			if _, floodWait := tgerr.AsFloodWait(parsed); floodWait {
				return parsed
			}
		}
		return nil
	})
	if err != nil {
		d.retireOnFailure(err)
		return nil, err
	}

	if rpcErr, ok := result.(*tg.RPCError); ok {
		parsed := tgerr.New(int(rpcErr.ErrorCode), rpcErr.ErrorMessage)
		d.retireOnFailure(parsed)
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
		if d.client != nil && !d.suppressTelemetry {
			d.client.observeRPC(ctx, input, 1, started, err)
		}
	}()
	if d.entry != nil {
		started := d.entry.beginRequest()
		defer func() { d.entry.endRequest(started, err, d.client.config().EndpointCoolDown) }()
	}

	query, initializesAPI := prepareAPIQuery(d.client.config(), d.apiInit.Load(), input)

	err = retryTransferFloodWait(ctx, func() error {
		var invokeErr error
		data, invokeErr = d.sess.InvokeRaw(ctx, query, 2, 60*time.Second)
		return invokeErr
	})
	if err != nil {
		d.retireOnFailure(err)
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
