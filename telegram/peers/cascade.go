package peers

import (
	"context"
	"fmt"
	"sync"

	"github.com/mtgo-labs/mtgo/internal/peerid"
	"github.com/mtgo-labs/mtgo/tg"
)

// InputPeer resolves a reference through the full cascade: pre-built peer,
// phone, username, then numeric ID with bot/account fallbacks. When the
// client is a bot, the result is additionally completed with a usable access
// hash.
func (m *Manager) InputPeer(ctx context.Context, r Ref) (tg.InputPeerClass, error) {
	var (
		peer tg.InputPeerClass
		err  error
	)
	switch {
	case r.Peer != nil:
		peer = r.Peer
	case r.Phone != "":
		peer, err = m.InputPeerByPhone(ctx, r.Phone)
	case r.Username != "":
		peer, err = m.InputPeerByUsername(ctx, r.Username)
	case r.ID == 0:
		peer = &tg.InputPeerSelf{}
	default:
		peer, err = m.inputPeerByID(ctx, r.ID)
	}
	if err != nil {
		return nil, err
	}
	if m.isBot() {
		return m.EnsureUsable(ctx, peer)
	}
	return peer, nil
}

// InputUser resolves a reference to an input user. It fails with a
// not-found error when the resolved peer is not a user.
func (m *Manager) InputUser(ctx context.Context, r Ref) (tg.InputUserClass, error) {
	peer, err := m.InputPeer(ctx, r)
	if err != nil {
		return nil, err
	}
	return InputPeerToUser(peer)
}

// InputChannel resolves a reference to an input channel. It fails with a
// not-found error when the resolved peer is not a channel.
func (m *Manager) InputChannel(ctx context.Context, r Ref) (tg.InputChannelClass, error) {
	peer, err := m.InputPeer(ctx, r)
	if err != nil {
		return nil, err
	}
	return InputPeerToChannel(peer)
}

// inputPeerByID resolves a numeric ID: cache first (when the cached entry
// carries a usable access hash), then the bot or account fallback chain.
func (m *Manager) inputPeerByID(ctx context.Context, id int64) (tg.InputPeerClass, error) {
	peer, err := m.cachedForUse(id)
	if err == nil {
		return peer, nil
	}
	if m.isBot() {
		return m.numericForBot(ctx, id)
	}
	return m.numericForAccount(ctx, id)
}

// cachedForUse returns the cached peer for id when it exists and carries a
// usable access hash.
func (m *Manager) cachedForUse(id int64) (tg.InputPeerClass, error) {
	if id == 0 {
		return &tg.InputPeerSelf{}, nil
	}
	peer, err := m.Cached(id)
	if err != nil {
		return nil, fmt.Errorf("could not resolve peer %d: %w", id, err)
	}
	if hasAccessHash(peer) {
		return peer, nil
	}
	return nil, ErrNotFound
}

// hasAccessHash returns false when peer is a channel or user with a zero
// access hash. Such peers (typically cached from min entities) are unusable
// for API calls like channels.getFullChannel and must be re-resolved.
func hasAccessHash(peer tg.InputPeerClass) bool {
	switch p := peer.(type) {
	case *tg.InputPeerChannel:
		return p.AccessHash != 0
	case *tg.InputPeerUser:
		return p.AccessHash != 0
	default:
		return true
	}
}

// EnsureUsable completes a peer with a usable access hash when the client is
// authorized as a bot: zero-hash users are fetched via users.getUsers and
// zero-hash channels via channels.getChannels. Peers that already carry a
// hash, and peers of other kinds, are returned unchanged.
func (m *Manager) EnsureUsable(ctx context.Context, peer tg.InputPeerClass) (tg.InputPeerClass, error) {
	switch p := peer.(type) {
	case *tg.InputPeerUser:
		if p.AccessHash != 0 {
			return peer, nil
		}
		return m.botUserAccessHash(ctx, p.UserID)
	case *tg.InputPeerChannel:
		if p.AccessHash != 0 {
			return peer, nil
		}
		return m.botChannelAccessHash(ctx, p.ChannelID)
	default:
		return peer, nil
	}
}

// coalescer prevents duplicate concurrent RPC calls for the same username or
// phone number. Multiple goroutines resolving the same peer share a single
// in-flight RPC call and all receive the same result.
type coalescer struct {
	mu       sync.Mutex
	inFlight map[string][]chan resolveResult
}

type resolveResult struct {
	peer tg.InputPeerClass
	err  error
}

func (r *coalescer) Do(key string, fn func() (tg.InputPeerClass, error)) (tg.InputPeerClass, error) {
	r.mu.Lock()
	if r.inFlight == nil {
		r.inFlight = make(map[string][]chan resolveResult)
	}
	if waiters, ok := r.inFlight[key]; ok {
		ch := make(chan resolveResult, 1)
		r.inFlight[key] = append(waiters, ch)
		r.mu.Unlock()
		res := <-ch
		return res.peer, res.err
	}
	r.inFlight[key] = nil
	r.mu.Unlock()

	peer, err := func() (peer tg.InputPeerClass, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				err = fmt.Errorf("resolve coalescer panic: %v", rec)
			}
		}()
		return fn()
	}()

	r.mu.Lock()
	waiters := r.inFlight[key]
	delete(r.inFlight, key)
	r.mu.Unlock()

	res := resolveResult{peer: peer, err: err}
	for _, ch := range waiters {
		ch <- res
	}
	return peer, err
}

func (m *Manager) coalesce(key string, fn func() (tg.InputPeerClass, error)) (tg.InputPeerClass, error) {
	return m.coalescer.Do(key, fn)
}

// inputPeerFromBareChatID converts a negated basic-group ID (without the
// channel prefix) into an input chat peer.
func inputPeerFromBareChatID(id int64) (tg.InputPeerClass, bool) {
	if id < 0 {
		if !peerid.IsMarkedChannel(id) {
			return &tg.InputPeerChat{ChatID: -id}, true
		}
	}
	return nil, false
}
