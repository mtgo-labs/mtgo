package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/telegram/peers"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// updateRecoveryIdle is the interval between idle recovery sweeps when no
// gap was detected. Gaps are usually caught by tracking (pts/seq/qts jumps)
// or by the reconnect hook; the sweep covers anything the trackers miss.
const updateRecoveryIdle = 15 * time.Minute

// updateRecoveryBackoffMax caps the exponential retry delay of a failing
// recovery pass.
const updateRecoveryBackoffMax = 60 * time.Second

// differencePageLimit is the per-request page size for getDifference and
// getChannelDifference; the server never returns more, larger values only
// waste bandwidth.
const differencePageLimit = 100

// updateRecovery recovers updates missed across gaps, reconnects, and
// restarts via updates.getDifference / updates.getChannelDifference, backed
// by the persistent UpdateStateStore. It is the gap-recovery engine the
// new-session and reconnect hooks feed: every observed update advances the
// tracked pts/seq/qts state; a jump past the expected value schedules a
// recovery pass that fetches and dispatches the missing updates through the
// normal HandleUpdates path (ingest, dedup, handlers).
type updateRecovery struct {
	client *Client

	mu       sync.Mutex
	started  bool
	hasState bool
	state    storage.UpdateState
	channels map[int64]int32 // channel ID → last applied pts
	pending  map[int64]bool  // channels with a detected gap
	backoff  time.Duration

	wake     chan struct{}
	stop     chan struct{}
	stopped  chan struct{}
	stopOnce sync.Once
}

func newUpdateRecovery(c *Client) *updateRecovery {
	return &updateRecovery{
		client:   c,
		channels: make(map[int64]int32),
		pending:  make(map[int64]bool),
		wake:     make(chan struct{}, 1),
		stop:     make(chan struct{}),
		stopped:  make(chan struct{}),
	}
}

// initUpdateRecovery wires the engine's hooks. Called once from NewClient.
func (c *Client) initUpdateRecovery() {
	c.updateRec = newUpdateRecovery(c)
	c.OnUpdateReceived(func(cl *Client, updates tg.UpdatesClass) {
		cl.updateRec.trackBatch(updates)
	})
	c.OnReconnect(func(cl *Client) {
		cl.updateRec.Schedule()
	})
}

// startUpdateRecovery begins the recovery loop after the connection is up.
// Idempotent; stopped by stopUpdateRecovery.
func (r *updateRecovery) start() {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return
	}
	r.started = true
	r.mu.Unlock()
	go r.loop()
}

func (r *updateRecovery) shutdown() {
	r.stopOnce.Do(func() {
		close(r.stop)
		r.mu.Lock()
		started := r.started
		r.mu.Unlock()
		if !started {
			close(r.stopped)
			return
		}
		<-r.stopped
	})
}

func (c *Client) stopUpdateRecovery() {
	if c.updateRec != nil {
		c.updateRec.shutdown()
	}
}

func (r *updateRecovery) loop() {
	defer close(r.stopped)
	idle := time.NewTicker(updateRecoveryIdle)
	defer idle.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-r.wake:
		case <-idle.C:
		}
		r.pass()
	}
}

// Schedule asks for a recovery pass as soon as possible. Non-blocking: safe
// from update hooks (which run on the receive goroutine).
func (r *updateRecovery) Schedule() {
	if !r.isRunning() {
		return
	}
	select {
	case <-r.stop:
		return
	default:
	}
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *updateRecovery) isRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.started
}

// seedFromServer adopts the server's updates.getState as the tracked state
// when nothing was persisted. When a persisted state exists it is kept: the
// recovery pass will fetch everything missed while the client was stopped.
func (r *updateRecovery) seedFromServer(st *tg.UpdatesState) {
	if st == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.hasState {
		return
	}
	r.state = storage.UpdateState{
		SessionID: r.stateKeyLocked(),
		Pts:       st.PTS,
		Qts:       st.Qts,
		Date:      st.Date,
		Seq:       st.Seq,
	}
	r.hasState = true
	if us := r.client.UpdateState(); us != nil {
		cp := r.state
		_ = us.SaveUpdateState(&cp)
	}
}

// loadPersisted restores the tracked state and channel points at session
// load. Called before any updates can arrive.
func (r *updateRecovery) loadPersisted() {
	us := r.client.UpdateState()
	if us == nil {
		return
	}
	key := r.client.config().SessionName
	r.mu.Lock()
	defer r.mu.Unlock()
	if st, err := us.LoadUpdateState(key); err == nil && st != nil {
		r.state = *st
		r.hasState = true
	}
	if chs, err := us.LoadAllChannelUpdateStates(key); err == nil {
		for _, ch := range chs {
			if ch != nil {
				r.channels[ch.ChannelID] = ch.Pts
			}
		}
	}
}

func (r *updateRecovery) stateKeyLocked() string {
	return r.client.config().SessionName
}

// pass runs one recovery attempt: a full getDifference sequence followed by
// the queued channel differences. On failure it retries with exponential
// backoff by rescheduling itself.
func (r *updateRecovery) pass() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := r.recoverAll(ctx); err != nil {
		if ctx.Err() != nil {
			return
		}
		r.mu.Lock()
		if r.backoff == 0 {
			r.backoff = time.Second
		} else if r.backoff < updateRecoveryBackoffMax {
			r.backoff *= 2
			if r.backoff > updateRecoveryBackoffMax {
				r.backoff = updateRecoveryBackoffMax
			}
		}
		wait := r.backoff
		r.mu.Unlock()
		r.client.Log.Debugf("update recovery: %v (retrying in %s)", err, wait)
		time.AfterFunc(wait, r.Schedule)
		return
	}
	r.mu.Lock()
	r.backoff = 0
	r.mu.Unlock()
}

func (r *updateRecovery) recoverAll(ctx context.Context) error {
	if !r.stateReady() {
		return nil
	}
	if err := r.recoverDifference(ctx); err != nil {
		return err
	}
	// Drain queued channels; recoverChannel removes entries as it works.
	for {
		channelID, ok := r.nextPendingChannel()
		if !ok {
			return nil
		}
		if err := r.recoverChannel(ctx, channelID); err != nil {
			return fmt.Errorf("channel %d: %w", channelID, err)
		}
	}
}

func (r *updateRecovery) stateReady() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.hasState
}

func (r *updateRecovery) nextPendingChannel() (int64, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id := range r.pending {
		delete(r.pending, id)
		return id, true
	}
	return 0, false
}

// recoverDifference applies updates.getDifference until the server reports
// the state caught up.
func (r *updateRecovery) recoverDifference(ctx context.Context) error {
	for page := 0; page < 1000; page++ {
		r.mu.Lock()
		st := r.state
		r.mu.Unlock()
		diff, err := r.client.Raw().UpdatesGetDifference(ctx, &tg.UpdatesGetDifferenceRequest{
			PTS:      st.Pts,
			PTSLimit: differencePageLimit,
			Date:     st.Date,
			Qts:      st.Qts,
			QtsLimit: differencePageLimit,
		})
		if err != nil {
			if handled, hErr := r.handleDifferenceError(ctx, err); handled {
				return hErr
			}
			return err
		}
		switch v := diff.(type) {
		case *tg.UpdatesDifferenceEmpty:
			r.applyState(v.Date, v.Seq)
			return nil
		case *tg.UpdatesDifferenceTooLong:
			// The history before the returned pts is gone; fast-forward.
			r.setPTS(v.PTS)
			return nil
		case *tg.UpdatesDifference:
			r.applyEntities(v.NewMessages, v.OtherUpdates, v.Users, v.Chats)
			if v.State != nil {
				r.adoptState(v.State)
			}
		case *tg.UpdatesDifferenceSlice:
			r.applyEntities(v.NewMessages, v.OtherUpdates, v.Users, v.Chats)
			if v.IntermediateState != nil {
				r.adoptState(v.IntermediateState)
			}
		default:
			return fmt.Errorf("unexpected difference type %T", diff)
		}
		// No progress guard: a server that keeps paging without moving the
		// state forward would loop forever.
		r.mu.Lock()
		unchanged := r.state.Pts == st.Pts && r.state.Date == st.Date && r.state.Qts == st.Qts
		r.mu.Unlock()
		if unchanged {
			return nil
		}
	}
	return errors.New("difference did not converge")
}

// handleDifferenceError classifies getDifference failures: FLOOD_WAIT and
// persistent-timestamp resets are recoverable in-pass (returns handled=true,
// nil error), everything else is returned for backoff.
func (r *updateRecovery) handleDifferenceError(ctx context.Context, err error) (bool, error) {
	if wait, ok := tgerr.AsFloodWait(err); ok {
		select {
		case <-time.After(wait):
			return true, nil
		case <-r.stop:
			return true, ctx.Err()
		}
	}
	if tgerr.Is(err, tgerr.ErrPersistentTimestampInvalid) ||
		tgerr.Is(err, tgerr.ErrPersistentTimestampOutdated) {
		if rsErr := r.reseedFromServer(ctx); rsErr != nil {
			return false, rsErr
		}
		return true, nil
	}
	return false, err
}

// reseedFromServer resets the tracked state from updates.getState after the
// server rejected our persistent timestamps.
func (r *updateRecovery) reseedFromServer(ctx context.Context) error {
	st, err := r.client.Raw().UpdatesGetState(ctx)
	if err != nil {
		return fmt.Errorf("reseed: %w", err)
	}
	r.adoptState(st)
	return nil
}

// recoverChannel applies updates.getChannelDifference for one channel until
// final. Unknown or inaccessible channels drop their tracking.
func (r *updateRecovery) recoverChannel(ctx context.Context, channelID int64) error {
	input, err := r.channelInput(ctx, channelID)
	if err != nil {
		r.dropChannel(channelID)
		return nil // not retryable: the channel cannot be addressed
	}
	for page := 0; page < 100; page++ {
		pts := r.channelPTS(channelID)
		diff, err := r.client.Raw().UpdatesGetChannelDifference(ctx, &tg.UpdatesGetChannelDifferenceRequest{
			Channel: input,
			Filter:  &tg.ChannelMessagesFilterEmpty{},
			PTS:     pts,
			Limit:   differencePageLimit,
		})
		if err != nil {
			if wait, ok := tgerr.AsFloodWait(err); ok {
				select {
				case <-time.After(wait):
					continue
				case <-r.stop:
					return ctx.Err()
				}
			}
			if tgerr.Is(err, tgerr.ErrPersistentTimestampInvalid) ||
				tgerr.Is(err, tgerr.ErrPersistentTimestampOutdated) {
				r.dropChannel(channelID)
				r.Schedule()
				return nil
			}
			if tgerr.Is(err, tgerr.ErrChannelPrivate) || tgerr.Is(err, tgerr.ErrChannelInvalid) {
				r.dropChannel(channelID)
				return nil
			}
			return err
		}
		switch v := diff.(type) {
		case *tg.UpdatesChannelDifferenceEmpty:
			r.setChannelPTS(channelID, v.PTS)
			return nil
		case *tg.UpdatesChannelDifference:
			r.applyEntities(v.NewMessages, v.OtherUpdates, v.Users, v.Chats)
			r.setChannelPTS(channelID, v.PTS)
			if v.Final {
				return nil
			}
		case *tg.UpdatesChannelDifferenceTooLong:
			// The channel's history before the timeout pts is unavailable.
			// Deliver what the server attached and fast-forward the pts.
			r.applyEntities(v.Messages, nil, v.Users, v.Chats)
			r.setChannelPTS(channelID, v.Timeout)
			r.client.Log.Debugf("update recovery: channel %d gap too long, fast-forwarded to pts %d", channelID, v.Timeout)
			return nil
		default:
			return fmt.Errorf("unexpected channel difference type %T", diff)
		}
	}
	return fmt.Errorf("channel %d difference did not converge", channelID)
}

// channelInput resolves a channel to an InputChannelClass using the peer
// cache first; a cache miss triggers the full resolution cascade once.
func (r *updateRecovery) channelInput(ctx context.Context, channelID int64) (tg.InputChannelClass, error) {
	m := r.client.peersManager()
	if peer, err := m.Cached(channelID); err == nil {
		if ch, ok := peer.(*tg.InputPeerChannel); ok {
			return &tg.InputChannel{ChannelID: ch.ChannelID, AccessHash: ch.AccessHash}, nil
		}
		return nil, fmt.Errorf("peer %d is not a channel", channelID)
	}
	return m.InputChannel(ctx, peers.Ref{ID: channelID})
}

// ---- state application ----

// applyEntities dispatches a difference page through the normal update path
// (ingest, dedup, handlers) by wrapping it in an updates container.
func (r *updateRecovery) applyEntities(messages []tg.MessageClass, otherUpdates []tg.UpdateClass, users []tg.UserClass, chats []tg.ChatClass) {
	if len(messages) == 0 && len(otherUpdates) == 0 {
		// Entities alone still feed the peer cache.
		if len(users) > 0 || len(chats) > 0 {
			r.client.peersManager().Ingest(users, chats)
		}
		return
	}
	updates := make([]tg.UpdateClass, 0, len(messages)+len(otherUpdates))
	updates = append(updates, otherUpdates...)
	for _, msg := range messages {
		updates = append(updates, messageAsUpdate(msg))
	}
	r.client.HandleUpdates(&tg.Updates{
		Updates: updates,
		Users:   users,
		Chats:   chats,
	})
}

func messageAsUpdate(msg tg.MessageClass) tg.UpdateClass {
	if m, ok := msg.(*tg.Message); ok {
		if _, isChannel := m.PeerID.(*tg.PeerChannel); isChannel {
			return &tg.UpdateNewChannelMessage{Message: m}
		}
	}
	return &tg.UpdateNewMessage{Message: msg}
}

func (r *updateRecovery) adoptState(st *tg.UpdatesState) {
	r.mu.Lock()
	r.state.Pts = st.PTS
	r.state.Qts = st.Qts
	r.state.Date = st.Date
	r.state.Seq = st.Seq
	r.state.SessionID = r.stateKeyLocked()
	r.hasState = true
	r.mu.Unlock()
	r.persist()
}

func (r *updateRecovery) applyState(date, seq int32) {
	r.mu.Lock()
	changed := r.state.Date != date || r.state.Seq != seq
	r.state.Date = date
	r.state.Seq = seq
	r.mu.Unlock()
	if changed {
		r.persist()
	}
}

func (r *updateRecovery) setPTS(pts int32) {
	r.mu.Lock()
	changed := r.state.Pts != pts
	r.state.Pts = pts
	r.mu.Unlock()
	if changed {
		r.persist()
	}
}

func (r *updateRecovery) channelPTS(channelID int64) int32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.channels[channelID]
}

func (r *updateRecovery) setChannelPTS(channelID int64, pts int32) {
	r.mu.Lock()
	r.channels[channelID] = pts
	r.mu.Unlock()
	if us := r.client.UpdateState(); us != nil {
		_ = us.SaveChannelUpdateState(&storage.ChannelUpdateState{
			SessionID: r.client.config().SessionName,
			ChannelID: channelID,
			Pts:       pts,
		})
	}
}

func (r *updateRecovery) dropChannel(channelID int64) {
	r.mu.Lock()
	delete(r.channels, channelID)
	delete(r.pending, channelID)
	r.mu.Unlock()
	if us := r.client.UpdateState(); us != nil {
		// Backends without a delete API simply leave the row; it is
		// re-created if the channel is tracked again.
		_ = us.SaveChannelUpdateState(&storage.ChannelUpdateState{
			SessionID: r.client.config().SessionName,
			ChannelID: channelID,
			Pts:       0,
		})
	}
}

func (r *updateRecovery) persist() {
	us := r.client.UpdateState()
	if us == nil {
		return
	}
	r.mu.Lock()
	cp := r.state
	cp.SessionID = r.stateKeyLocked()
	r.mu.Unlock()
	_ = us.SaveUpdateState(&cp)
}

// ---- live tracking ----

// trackAffectedResponses returns an invoker middleware that applies the pts
// carried by mutation responses (messages.affectedMessages,
// messages.affectedFoundMessages). Deletes and edits advance the persistent
// timestamp state exactly like the updates echoing them would, so no
// getDifference round-trip is spent rediscovering the client's own writes.
func (r *updateRecovery) trackAffectedResponses() func(next tg.Invoker) tg.Invoker {
	return func(next tg.Invoker) tg.Invoker {
		return tg.InvokerFunc(func(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
			res, err := next.RPCInvoke(ctx, input, decode)
			if err == nil {
				switch v := res.(type) {
				case *tg.MessagesAffectedMessages:
					if r.trackPTS(v.PTS, v.PTSCount) {
						r.Schedule()
					}
				case *tg.MessagesAffectedFoundMessages:
					if r.trackPTS(v.PTS, v.PTSCount) {
						r.Schedule()
					}
				}
			}
			return res, err
		})
	}
}

// trackBatch advances the tracked state from an incoming update batch and
// schedules recovery when a gap is detected. Runs synchronously on the
// receive goroutine: only map updates and a non-blocking wake.
func (r *updateRecovery) trackBatch(updates tg.UpdatesClass) {
	if updates == nil {
		return
	}
	needRecovery := false
	switch v := updates.(type) {
	case *tg.Updates:
		needRecovery = r.trackSeq(v.Seq, v.Date) || needRecovery
		needRecovery = r.trackUpdateList(v.Updates) || needRecovery
	case *tg.UpdatesCombined:
		needRecovery = r.trackSeq(v.Seq, v.Date) || needRecovery
		needRecovery = r.trackUpdateList(v.Updates) || needRecovery
	case *tg.UpdateShort:
		needRecovery = r.trackUpdate(v.Update)
	case *tg.UpdateShortMessage:
		needRecovery = r.trackPTS(v.PTS, v.PTSCount)
		if v.Date > 0 {
			r.applyState(v.Date, 0)
		}
	case *tg.UpdateShortChatMessage:
		needRecovery = r.trackPTS(v.PTS, v.PTSCount)
		if v.Date > 0 {
			r.applyState(v.Date, 0)
		}
	case *tg.UpdateShortSentMessage:
		needRecovery = r.trackPTS(v.PTS, v.PTSCount)
	case *tg.UpdatesTooLong:
		needRecovery = true
	}
	if needRecovery {
		r.Schedule()
	}
}

func (r *updateRecovery) trackUpdateList(list []tg.UpdateClass) bool {
	need := false
	for _, upd := range list {
		if r.trackUpdate(upd) {
			need = true
		}
	}
	return need
}

// trackUpdate applies one update's persistent-timestamp effects. Returns
// true when a gap was detected.
func (r *updateRecovery) trackUpdate(upd tg.UpdateClass) bool {
	switch v := upd.(type) {
	case *tg.UpdateNewMessage:
		return r.trackPTS(v.PTS, v.PTSCount)
	case *tg.UpdateEditMessage:
		return r.trackPTS(v.PTS, v.PTSCount)
	case *tg.UpdateReadMessagesContents:
		return r.trackPTS(v.PTS, v.PTSCount)
	case *tg.UpdateDeleteMessages:
		return r.trackPTS(v.PTS, v.PTSCount)
	case *tg.UpdateReadHistoryInbox:
		return r.trackPTS(v.PTS, v.PTSCount)
	case *tg.UpdateReadHistoryOutbox:
		return r.trackPTS(v.PTS, v.PTSCount)
	case *tg.UpdateWebPage:
		return r.trackPTS(v.PTS, v.PTSCount)
	case *tg.UpdateFolderPeers:
		return r.trackPTS(v.PTS, v.PTSCount)
	case *tg.UpdatePTSChanged:
		// The server reset pts state; resynchronize from scratch.
		return true
	case *tg.UpdateNewEncryptedMessage:
		return r.trackQTS(v.Qts)
	case *tg.UpdateNewChannelMessage:
		if ch, ok := channelOfMessage(v.Message); ok {
			return r.trackChannelPTS(ch, v.PTS, v.PTSCount)
		}
		return false
	case *tg.UpdateEditChannelMessage:
		if ch, ok := channelOfMessage(v.Message); ok {
			return r.trackChannelPTS(ch, v.PTS, v.PTSCount)
		}
		return false
	case *tg.UpdateDeleteChannelMessages:
		return r.trackChannelPTS(v.ChannelID, v.PTS, v.PTSCount)
	case *tg.UpdateReadChannelInbox:
		return r.trackChannelPTS(v.ChannelID, v.PTS, 0)
	}
	return false
}

func channelOfMessage(msg tg.MessageClass) (int64, bool) {
	m, ok := msg.(*tg.Message)
	if !ok || m.PeerID == nil {
		return 0, false
	}
	if p, ok := m.PeerID.(*tg.PeerChannel); ok {
		return p.ChannelID, true
	}
	return 0, false
}

// trackPTS advances the common pts state. Returns true on a detected gap.
func (r *updateRecovery) trackPTS(pts, count int32) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.hasState {
		return false
	}
	switch {
	case count <= 0:
		if pts > r.state.Pts {
			r.state.Pts = pts
		}
		return false
	case pts == r.state.Pts+count:
		r.state.Pts = pts
		return false
	case pts > r.state.Pts+count:
		return true
	default:
		return false // old or duplicate
	}
}

// trackQTS advances the qts state one event at a time.
func (r *updateRecovery) trackQTS(qts int32) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.hasState {
		return false
	}
	switch {
	case qts == r.state.Qts+1:
		r.state.Qts = qts
		return false
	case qts > r.state.Qts+1:
		return true
	default:
		return false
	}
}

// trackSeq advances the seq/date state of an updates container.
func (r *updateRecovery) trackSeq(seq, date int32) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.hasState {
		return false
	}
	need := seq > r.state.Seq+1
	if date > r.state.Date {
		r.state.Date = date
	}
	if seq > r.state.Seq {
		r.state.Seq = seq
	}
	return need
}

// trackChannelPTS advances a channel's pts state; a gap queues the channel
// for getChannelDifference.
func (r *updateRecovery) trackChannelPTS(channelID int64, pts, count int32) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	known, tracked := r.channels[channelID]
	switch {
	case !tracked:
		r.channels[channelID] = pts
		return false
	case count <= 0:
		if pts > known {
			r.channels[channelID] = pts
		}
		return false
	case pts == known+count:
		r.channels[channelID] = pts
		return false
	case pts > known+count:
		r.pending[channelID] = true
		return true
	default:
		return false
	}
}
