package peers

import (
	"sync"

	"github.com/mtgo-labs/mtgo/tg"
)

// anchor is a message-anchored reference for a peer whose access hash is
// unknown: the server can address the peer through the message it appeared
// in, without any hash.
type anchor struct {
	chatID int64 // chat where the message lives (raw peer ID)
	msgID  int32
}

// anchorStore keeps per-peer message anchors, bounded FIFO like the other
// cache indexes. Bots and accounts both accept inputPeer*FromMessage.
type anchorStore struct {
	mu     sync.Mutex
	byPeer map[int64]anchor
	order  []int64
	limit  int
}

func newAnchorStore(limit int) *anchorStore {
	return &anchorStore{byPeer: make(map[int64]anchor), limit: limit}
}

// CacheAnchor records that peerID appeared in msgID inside chatID. Later
// anchor resolutions for the same peer overwrite earlier ones (fresher
// message, same validity).
func (a *anchorStore) CacheAnchor(peerID, chatID int64, msgID int32) {
	if peerID == 0 || chatID == 0 || msgID == 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.byPeer[peerID]; !exists {
		a.order = append(a.order, peerID)
	}
	a.byPeer[peerID] = anchor{chatID: chatID, msgID: msgID}
	for a.limit > 0 && len(a.byPeer) > a.limit && len(a.order) > 0 {
		oldest := a.order[0]
		delete(a.byPeer, oldest)
		copy(a.order, a.order[1:])
		a.order[len(a.order)-1] = 0
		a.order = a.order[:len(a.order)-1]
	}
}

// get returns the freshest anchor for a peer.
func (a *anchorStore) get(peerID int64) (anchor, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	an, ok := a.byPeer[peerID]
	return an, ok
}

// anchorLimit bounds the anchor store independently of the peer cache.
const anchorLimit = 1000

// CacheAnchor records a message anchor for a peer lacking a usable access
// hash, so later numeric resolution can address it via
// inputPeerUserFromMessage / inputPeerChannelFromMessage.
func (m *Manager) CacheAnchor(peerID, chatID int64, msgID int32) {
	m.anchors.Lock()
	defer m.anchors.Unlock()
	if m.anchors.store == nil {
		m.anchors.store = newAnchorStore(anchorLimit)
	}
	m.anchors.store.CacheAnchor(peerID, chatID, msgID)
}

// anchorInputPeer builds a message-anchored input peer for id when an anchor
// exists and the containing chat is itself resolvable.
func (m *Manager) anchorInputPeer(id int64) (tg.InputPeerClass, bool) {
	m.anchors.Lock()
	store := m.anchors.store
	m.anchors.Unlock()
	if store == nil {
		return nil, false
	}
	an, ok := store.get(id)
	if !ok {
		return nil, false
	}
	chatPeer, err := m.Cached(an.chatID)
	if err != nil {
		return nil, false
	}
	switch {
	case id > 0:
		return &tg.InputPeerUserFromMessage{
			Peer:   chatPeer,
			MsgID:  an.msgID,
			UserID: id,
		}, true
	default:
		return &tg.InputPeerChannelFromMessage{
			Peer:      chatPeer,
			ChannelID: id,
			MsgID:     an.msgID,
		}, true
	}
}

// anchorsField is embedded in Manager; a small struct keeps the lazy store
// behind one mutex instead of enlarging the main cache critical section.
type anchorsField struct {
	sync.Mutex
	store *anchorStore
}
