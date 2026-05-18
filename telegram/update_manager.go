package telegram

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/internal/storage"
)

type updateManagerConfig struct {
	QueueSize       int
	DurableQueue    bool
	MaxHandlerRetry int
	// GapBuffer is the duration to defer getDifference calls after detecting
	// a PTS gap. Defaults to 500ms. Set to 0 to disable buffering.
	GapBuffer time.Duration
}

// UpdateHealth holds diagnostic metrics about the update processing pipeline,
// including state counters, gap detection, and error tracking.
//
// Example:
//
//	health := client.UpdateHealth()
//	fmt.Printf("pts=%d qts=%d seq=%d pending=%d\n",
//		health.Pts, health.Qts, health.Seq, health.Pending)
//	if health.LastGap.After(time.Now().Add(-5 * time.Minute)) {
//		fmt.Println("gap detected recently")
//	}
//	if health.LastUpdateError != nil {
//		fmt.Println("last handler error:", health.LastUpdateError)
//	}
type UpdateHealth struct {
	Pts             int32
	Qts             int32
	Date            int32
	Seq             int32
	Pending         int
	LastRecovery    time.Time
	LastGap         time.Time
	RecoveryCount   int64
	DuplicateCount  int64
	LastUpdateError error
}

type updateManager struct {
	client *Client
	store  storage.UpdateStateStore
	cfg    updateManagerConfig
	rpc    differenceRPC

	sessionID string
	queue     chan tg.UpdatesClass
	done      chan struct{}
	cancel    context.CancelFunc

	// mu protects state, channels, health, and recoveryTimer.
	mu sync.RWMutex
	state      updateState
	channels   map[int64]channelState
	recovering bool
	// recoveryTimer buffers gap detection for 500ms to avoid triggering
	// expensive getDifference calls for gaps that self-resolve via the
	// next arriving update. When non-nil, a deferred recovery is pending.
	recoveryTimer *time.Timer

	health UpdateHealth
}

func newUpdateManager(c *Client, st storage.Storage, cfg updateManagerConfig) *updateManager {
	var store storage.UpdateStateStore
	if s, ok := st.(storage.UpdateStateStore); ok {
		store = s
	}
	var sessionID string
	if sid, err := st.SessionID(); err == nil {
		sessionID = sid
	}
	return &updateManager{
		client:    c,
		store:     store,
		cfg:       cfg,
		sessionID: sessionID,
		queue:     make(chan tg.UpdatesClass, cfg.QueueSize),
		channels:  make(map[int64]channelState),
		done:      make(chan struct{}),
	}
}

func (m *updateManager) Start(ctx context.Context) error {
	if m.store == nil {
		return ErrUpdateStateUnavailable
	}

	saved, err := m.store.LoadUpdateState(m.sessionID)
	if err != nil {
		return err
	}
	if saved != nil {
		m.state = updateState{
			Pts:  saved.Pts,
			Qts:  saved.Qts,
			Date: saved.Date,
			Seq:  saved.Seq,
		}
	}

	chStates, err := m.store.LoadAllChannelUpdateStates(m.sessionID)
	if err != nil {
		return err
	}
	for _, cs := range chStates {
		m.channels[cs.ChannelID] = channelState{
			ChannelID: cs.ChannelID,
			Pts:       cs.Pts,
		}
	}

	childCtx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	go m.run(childCtx)
	return nil
}

func (m *updateManager) Stop(ctx context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Lock()
	if m.recoveryTimer != nil {
		m.recoveryTimer.Stop()
		m.recoveryTimer = nil
	}
	m.mu.Unlock()
	select {
	case <-m.done:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (m *updateManager) EnqueueLive(updates tg.UpdatesClass) error {
	select {
	case m.queue <- updates:
		return nil
	default:
		return ErrUpdateQueueFull
	}
}

func (m *updateManager) SetRPC(rpc differenceRPC) {
	m.rpc = rpc
}

func (m *updateManager) OnReconnect(ctx context.Context, rpc differenceRPC) error {
	if err := m.RecoverAccount(ctx, rpc); err != nil {
		return err
	}
	m.mu.Lock()
	channels := make(map[int64]channelState, len(m.channels))
	maps.Copy(channels, m.channels)
	m.mu.Unlock()
	for id := range channels {
		input := &tg.InputChannel{ChannelID: id}
		if err := m.RecoverChannel(ctx, rpc, id, input); err != nil {
			m.client.Log.Warnf("channel sweep recovery for %d: %v", id, err)
		}
	}
	return nil
}

func (m *updateManager) Health() UpdateHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h := m.health
	h.Pts = m.state.Pts
	h.Qts = m.state.Qts
	h.Date = m.state.Date
	h.Seq = m.state.Seq
	h.Pending = len(m.queue)
	return h
}

func (m *updateManager) run(ctx context.Context) {
	defer close(m.done)
	defer func() {
		if r := recover(); r != nil {
			m.client.Log.Errorf("update manager panic: %v", r)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case updates := <-m.queue:
			m.processUpdates(ctx, updates)
		}
	}
}

func (m *updateManager) processUpdates(ctx context.Context, updates tg.UpdatesClass) {
	parsedUsers, parsedChats, rawUpdates := m.client.flattenUpdates(updates)
	userMap := buildUserMap(parsedUsers)
	chatMap := buildChatMap(parsedChats)
	pm := buildPeerMapFromClasses(parsedUsers, parsedChats)
	for _, raw := range rawUpdates {
		if err := m.applyUpdate(ctx, raw, updates, userMap, chatMap, pm); err != nil {
			m.client.Log.Warnf("apply update: %v", err)
		}
	}
}

func (m *updateManager) applyUpdate(ctx context.Context, raw tg.UpdateClass, container tg.UpdatesClass, userMap map[int64]*types.User, chatMap map[int64]*types.Chat, pm *types.PeerMap) error {
	meta := extractUpdateMeta(raw)

	if _, ok := raw.(*tg.UpdateChannelTooLong); ok {
		if m.rpc != nil {
			input := &tg.InputChannel{ChannelID: meta.ChannelID}
			if recErr := m.RecoverChannel(ctx, m.rpc, meta.ChannelID, input); recErr != nil {
				m.client.Log.Warnf("channel too long recovery failed: %v", recErr)
			}
		}
		return nil
	}

	if meta.IsChannel {
		return m.applyChannelUpdate(ctx, raw, meta, container, userMap, chatMap, pm)
	}

	kind := m.classifyUpdate(meta)
	switch kind {
	case duplicateUpdate:
		m.mu.Lock()
		m.health.DuplicateCount++
		m.mu.Unlock()
		return nil
	case accountPtsGap, accountSeqGap, accountQtsGap:
		m.mu.Lock()
		m.health.LastGap = time.Now()
		m.mu.Unlock()
		m.bufferGapRecovery(ctx, container, raw, meta, userMap, chatMap, pm)
		return nil
	case noGap:
	}

	if meta.Key != "" {
		if inserted, err := m.store.SaveUpdateDedupKey(m.sessionID, meta.Key); err == nil && !inserted {
			m.mu.Lock()
			m.health.DuplicateCount++
			m.mu.Unlock()
			return nil
		}
	}

	return m.deliverUpdate(container, raw, meta, userMap, chatMap, pm)
}

func (m *updateManager) applyChannelUpdate(ctx context.Context, raw tg.UpdateClass, meta updateMeta, container tg.UpdatesClass, userMap map[int64]*types.User, chatMap map[int64]*types.Chat, pm *types.PeerMap) error {
	m.mu.RLock()
	ch, ok := m.channels[meta.ChannelID]
	m.mu.RUnlock()

	if !ok {
		ch = channelState{ChannelID: meta.ChannelID}
	}

	kind := classifyChannelUpdate(ch, meta)
	switch kind {
	case duplicateUpdate:
		m.mu.Lock()
		m.health.DuplicateCount++
		m.mu.Unlock()
		return nil
	case channelPtsGap:
		m.mu.Lock()
		m.health.LastGap = time.Now()
		m.mu.Unlock()
		if m.rpc != nil {
			input := &tg.InputChannel{ChannelID: meta.ChannelID}
			if recErr := m.RecoverChannel(ctx, m.rpc, meta.ChannelID, input); recErr != nil {
				m.client.Log.Warnf("channel gap recovery failed: %v", recErr)
			}
			m.mu.RLock()
			ch2 := m.channels[meta.ChannelID]
			m.mu.RUnlock()
			retryKind := classifyChannelUpdate(ch2, meta)
			if retryKind == duplicateUpdate {
				m.mu.Lock()
				m.health.DuplicateCount++
				m.mu.Unlock()
				return nil
			}
			if retryKind == noGap {
				return m.deliverUpdate(container, raw, meta, userMap, chatMap, pm)
			}
		}
		return nil
	case noGap:
	}

	return m.deliverUpdate(container, raw, meta, userMap, chatMap, pm)
}

func (m *updateManager) classifyUpdate(meta updateMeta) gapKind {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return classifyAccountUpdate(m.state, meta)
}

func (m *updateManager) deliverUpdate(container tg.UpdatesClass, raw tg.UpdateClass, meta updateMeta, userMap map[int64]*types.User, chatMap map[int64]*types.Chat, pm *types.PeerMap) error {
	if m.cfg.DurableQueue && meta.Key != "" {
		nowUnix := time.Now().Unix()
		record := &storage.DurableUpdate{
			SessionID: m.sessionID,
			ID:        meta.Key,
			Payload:   []byte(meta.Key),
			CreatedAt: nowUnix,
			UpdatedAt: nowUnix,
		}
		if err := m.store.EnqueueDurableUpdate(record); err != nil {
			m.client.Log.Warnf("durable queue write: %v", err)
		}
	}

	upd := m.client.toUpdate(raw, userMap, chatMap, pm)

	var lastErr error
	for attempt := 0; attempt <= m.cfg.MaxHandlerRetry; attempt++ {
		err := m.client.handlerDispatcher.DispatchSafe(m.client, upd)
		if err == nil {
			upd.reset()
			updatePool.Put(upd)
			if m.cfg.DurableQueue && meta.Key != "" {
				if err := m.store.DeleteDurableUpdate(m.sessionID, meta.Key); err != nil {
					m.client.Log.Warnf("delete durable update: %v", err)
				}
			}
			m.advanceState(meta)
			return nil
		}
		lastErr = err
	}

	upd.reset()
	updatePool.Put(upd)

	m.mu.Lock()
	m.health.LastUpdateError = lastErr
	m.mu.Unlock()

	if m.cfg.DurableQueue && meta.Key != "" {
		if err := m.store.MarkDurableUpdateFailed(m.sessionID, meta.Key, m.cfg.MaxHandlerRetry+1, lastErr.Error()); err != nil {
			m.client.Log.Warnf("mark durable update failed: %v", err)
		}
	}

	return fmt.Errorf("%w: %v", ErrUpdateHandlerFailed, lastErr)
}

func (m *updateManager) replayDurableQueue() error {
	items, err := m.store.LoadDurableUpdates(m.sessionID, 100)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := m.store.DeleteDurableUpdate(m.sessionID, item.ID); err != nil {
			m.client.Log.Warnf("delete durable update replay: %v", err)
		}
	}
	return nil
}

func (m *updateManager) advanceState(meta updateMeta) {
	m.mu.Lock()

	if meta.Pts > 0 {
		m.state.Pts = meta.Pts
	}
	if meta.Qts > 0 {
		m.state.Qts = meta.Qts
	}
	if meta.Seq > 0 {
		m.state.Seq = meta.Seq
	}
	if meta.Date > 0 {
		m.state.Date = meta.Date
	}
	if meta.IsChannel && meta.ChannelPts > 0 {
		ch, ok := m.channels[meta.ChannelID]
		if !ok {
			ch = channelState{ChannelID: meta.ChannelID}
		}
		ch.Pts = meta.ChannelPts
		m.channels[meta.ChannelID] = ch
	}

	// Snapshot state under lock, then save to storage without holding the lock
	// to avoid blocking callers while waiting on I/O.
	snapshot := storage.UpdateState{
		SessionID: m.sessionID,
		Pts:       m.state.Pts,
		Qts:       m.state.Qts,
		Date:      m.state.Date,
		Seq:       m.state.Seq,
	}
	channelStateToSave := meta.IsChannel && meta.ChannelPts > 0
	var chSnapshot storage.ChannelUpdateState
	if channelStateToSave {
		chSnapshot = storage.ChannelUpdateState{
			SessionID: m.sessionID,
			ChannelID: meta.ChannelID,
			Pts:       meta.ChannelPts,
		}
	}
	m.mu.Unlock()

	if channelStateToSave {
		if err := m.store.SaveChannelUpdateState(&chSnapshot); err != nil {
			m.client.Log.Warnf("save channel update state: %v", err)
		}
	}
	if err := m.store.SaveUpdateState(&snapshot); err != nil {
		m.client.Log.Warnf("save update state: %v", err)
	}
}

func classifyAccountUpdate(state updateState, meta updateMeta) gapKind {
	if meta.Pts > 0 {
		expected := state.Pts + meta.PtsCount
		switch {
		case meta.Pts == expected:
			return noGap
		case meta.Pts <= state.Pts:
			return duplicateUpdate
		default:
			return accountPtsGap
		}
	}
	if meta.Qts > 0 {
		if meta.Qts <= state.Qts {
			return duplicateUpdate
		}
		if meta.Qts > state.Qts+1 {
			return accountQtsGap
		}
	}
	if meta.Seq > 0 {
		if meta.Seq <= state.Seq {
			return duplicateUpdate
		}
		if meta.Seq > state.Seq+1 {
			return accountSeqGap
		}
	}
	return noGap
}

func classifyChannelUpdate(state channelState, meta updateMeta) gapKind {
	expected := state.Pts + meta.PtsCount
	switch {
	case meta.ChannelPts == expected:
		return noGap
	case meta.ChannelPts <= state.Pts:
		return duplicateUpdate
	default:
		return channelPtsGap
	}
}

func buildPeerMapFromClasses(users []tg.UserClass, chats []tg.ChatClass) *types.PeerMap {
	return types.NewPeerMapFromClasses(users, chats)
}

// bufferGapRecovery defers account gap recovery by 500ms. If the gap is
// filled by the next arriving update before the timer fires, the expensive
// getDifference call is skipped. If the timer fires and the gap persists,
// RecoverAccount is triggered.
func (m *updateManager) bufferGapRecovery(ctx context.Context, container tg.UpdatesClass, raw tg.UpdateClass, meta updateMeta, userMap map[int64]*types.User, chatMap map[int64]*types.Chat, pm *types.PeerMap) {
	m.mu.Lock()
	if m.recoveryTimer != nil {
		m.mu.Unlock()
		return
	}
	if m.cfg.GapBuffer <= 0 {
		m.mu.Unlock()
		m.doGapRecovery(ctx, container, raw, meta, userMap, chatMap, pm)
		return
	}
	m.recoveryTimer = time.AfterFunc(m.cfg.GapBuffer, func() {
		m.mu.Lock()
		m.recoveryTimer = nil
		m.mu.Unlock()
		m.doGapRecovery(ctx, container, raw, meta, userMap, chatMap, pm)
	})
	m.mu.Unlock()
}

func (m *updateManager) doGapRecovery(ctx context.Context, container tg.UpdatesClass, raw tg.UpdateClass, meta updateMeta, userMap map[int64]*types.User, chatMap map[int64]*types.Chat, pm *types.PeerMap) {
	kind := m.classifyUpdate(meta)
	if kind == noGap || kind == duplicateUpdate {
		return
	}
	if m.rpc != nil {
		if recErr := m.RecoverAccount(ctx, m.rpc); recErr != nil {
			m.client.Log.Warnf("gap recovery failed: %v", recErr)
		}
	}
	retryKind := m.classifyUpdate(meta)
	if retryKind == duplicateUpdate {
		m.mu.Lock()
		m.health.DuplicateCount++
		m.mu.Unlock()
	} else if retryKind == noGap {
		_ = m.deliverUpdate(container, raw, meta, userMap, chatMap, pm)
	}
}
