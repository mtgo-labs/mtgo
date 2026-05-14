package telegram

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/storage"
)

type updateManagerConfig struct {
	QueueSize       int
	DurableQueue    bool
	MaxHandlerRetry int
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

	mu         sync.Mutex
	state      updateState
	channels   map[int64]channelState
	recovering bool

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
	m.mu.Lock()
	defer m.mu.Unlock()
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
	_, _, rawUpdates := m.client.flattenUpdates(updates)
	for _, raw := range rawUpdates {
		if err := m.applyUpdate(ctx, raw, updates); err != nil {
			m.client.Log.Warnf("apply update: %v", err)
		}
	}
}

func (m *updateManager) applyUpdate(ctx context.Context, raw tg.UpdateClass, container tg.UpdatesClass) error {
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
		return m.applyChannelUpdate(ctx, raw, meta, container)
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
		if m.rpc != nil {
			if recErr := m.RecoverAccount(ctx, m.rpc); recErr != nil {
				m.client.Log.Warnf("gap recovery failed: %v", recErr)
			}
			retryKind := m.classifyUpdate(meta)
			if retryKind == duplicateUpdate {
				m.mu.Lock()
				m.health.DuplicateCount++
				m.mu.Unlock()
				return nil
			}
			if retryKind == noGap {
				return m.deliverUpdate(container, raw, meta)
			}
		}
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

	return m.deliverUpdate(container, raw, meta)
}

func (m *updateManager) applyChannelUpdate(ctx context.Context, raw tg.UpdateClass, meta updateMeta, container tg.UpdatesClass) error {
	m.mu.Lock()
	ch, ok := m.channels[meta.ChannelID]
	m.mu.Unlock()

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
			m.mu.Lock()
			ch2 := m.channels[meta.ChannelID]
			m.mu.Unlock()
			retryKind := classifyChannelUpdate(ch2, meta)
			if retryKind == duplicateUpdate {
				m.mu.Lock()
				m.health.DuplicateCount++
				m.mu.Unlock()
				return nil
			}
			if retryKind == noGap {
				return m.deliverUpdate(container, raw, meta)
			}
		}
		return nil
	case noGap:
	}

	return m.deliverUpdate(container, raw, meta)
}

func (m *updateManager) classifyUpdate(meta updateMeta) gapKind {
	m.mu.Lock()
	defer m.mu.Unlock()
	return classifyAccountUpdate(m.state, meta)
}

func (m *updateManager) deliverUpdate(container tg.UpdatesClass, raw tg.UpdateClass, meta updateMeta) error {
	if m.cfg.DurableQueue && meta.Key != "" {
		record := &storage.DurableUpdate{
			SessionID: m.sessionID,
			ID:        meta.Key,
			Payload:   []byte(meta.Key),
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		}
		if err := m.store.EnqueueDurableUpdate(record); err != nil {
			m.client.Log.Warnf("durable queue write: %v", err)
		}
	}

	parsedUsers, parsedChats, _ := m.client.flattenUpdates(container)
	userMap := buildUserMap(parsedUsers)
	chatMap := buildChatMap(parsedChats)
	pm := buildPeerMapFromClasses(parsedUsers, parsedChats)
	upd := m.client.toUpdate(raw, userMap, chatMap, pm)

	var lastErr error
	for attempt := 0; attempt <= m.cfg.MaxHandlerRetry; attempt++ {
		err := m.client.handlerDispatcher.DispatchSafe(m.client, upd)
		if err == nil {
			upd.reset()
			updatePool.Put(upd)
			if m.cfg.DurableQueue && meta.Key != "" {
				_ = m.store.DeleteDurableUpdate(m.sessionID, meta.Key)
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
		_ = m.store.MarkDurableUpdateFailed(m.sessionID, meta.Key, m.cfg.MaxHandlerRetry+1, lastErr.Error())
	}

	return fmt.Errorf("%w: %v", ErrUpdateHandlerFailed, lastErr)
}

func (m *updateManager) replayDurableQueue() error {
	items, err := m.store.LoadDurableUpdates(m.sessionID, 100)
	if err != nil {
		return err
	}
	for _, item := range items {
		_ = m.store.DeleteDurableUpdate(m.sessionID, item.ID)
	}
	return nil
}

func (m *updateManager) advanceState(meta updateMeta) {
	m.mu.Lock()
	defer m.mu.Unlock()

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
		_ = m.store.SaveChannelUpdateState(&storage.ChannelUpdateState{
			SessionID: m.sessionID,
			ChannelID: meta.ChannelID,
			Pts:       meta.ChannelPts,
		})
	}

	_ = m.store.SaveUpdateState(&storage.UpdateState{
		SessionID: m.sessionID,
		Pts:       m.state.Pts,
		Qts:       m.state.Qts,
		Date:      m.state.Date,
		Seq:       m.state.Seq,
	})
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
