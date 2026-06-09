package telegram

import (
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
)

// SecretChatState represents the current state of an end-to-end encrypted secret chat.
//
// Possible states:
//   - SecretChatStateWaiting – DH key exchange is in progress.
//   - SecretChatStateReady – keys are exchanged; the chat is ready for messages.
//   - SecretChatStateDiscarded – the chat was terminated by either party.
//
// Example:
//
//	chat, _ := client.GetSecretChat(chatID)
//	if chat.GetState() == telegram.SecretChatStateReady {
//		fmt.Println("secret chat is ready for encrypted messaging")
//	}
type SecretChatState int

const (
	SecretChatStateWaiting SecretChatState = iota
	SecretChatStateReady
	SecretChatStateDiscarded
)

// SecretChat holds the state and cryptographic material for an encrypted chat session.
// It tracks the DH key exchange, sequence numbers, and the negotiated auth key.
//
// Example:
//
//	chat, ok := client.GetSecretChat(chatID)
//	if !ok {
//		log.Fatal("secret chat not found")
//	}
//	vis := chat.Visualization()
//	fmt.Printf("chat %d state=%d key_vis=%v\n", chat.ID, chat.GetState(), vis)
type SecretChat struct {
	mu sync.RWMutex

	ID         int32
	AccessHash int64
	AdminID    int64
	PartID     int64
	State      SecretChatState
	Outgoing   bool
	Layer      int32

	DHPrime  *big.Int
	G        int32
	MySecret *big.Int
	GA       *big.Int
	GB       []byte

	AuthKey         []byte
	KeyFingerprint  int64
	DHConfigVersion int32

	InSeqNo  int32
	OutSeqNo int32

	RandomID int32
}

func (sc *SecretChat) SetState(s SecretChatState) {
	sc.mu.Lock()
	sc.State = s
	sc.mu.Unlock()
}

func (sc *SecretChat) GetState() SecretChatState {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.State
}

// seqParity returns the parity bit x used to wire-encode this side's outgoing
// sequence numbers, per https://core.telegram.org/api/end-to-end/seq_no:
// x = 0 for the chat creator, x = 1 for the joining party. Outgoing is set once
// at chat construction and is immutable afterwards.
func (sc *SecretChat) seqParity() int32 {
	if sc.Outgoing {
		return 0
	}
	return 1
}

// NextOutSeqNo advances the outgoing-message counter and returns the
// wire-encoded out_seq_no (2*count + x) to embed in the message being sent.
func (sc *SecretChat) NextOutSeqNo() int32 {
	count := atomic.AddInt32(&sc.OutSeqNo, 1) - 1
	return count*2 + sc.seqParity()
}

// CurrentInSeqNo returns the wire-encoded in_seq_no (2*count + (1-x)) to embed
// in an outgoing message, reflecting the number of messages received so far.
// It reads the counter atomically and does not advance it.
func (sc *SecretChat) CurrentInSeqNo() int32 {
	count := atomic.LoadInt32(&sc.InSeqNo)
	return count*2 + (1 - sc.seqParity())
}

// NextInSeqNo advances the received-message counter after a message is
// successfully decrypted and returns the previous raw count.
func (sc *SecretChat) NextInSeqNo() int32 {
	return atomic.AddInt32(&sc.InSeqNo, 1) - 1
}

// ExpectedInboundParity returns the parity an inbound message's out_seq_no must
// have: the complement of this side's outgoing parity. A received message whose
// out_seq_no parity differs indicates a protocol violation (e.g. message
// mirroring) and the chat should be treated as compromised.
func (sc *SecretChat) ExpectedInboundParity() int32 {
	return 1 - sc.seqParity()
}

func (sc *SecretChat) InputPeer() *tg.InputEncryptedChat {
	return &tg.InputEncryptedChat{
		ChatID:     sc.ID,
		AccessHash: sc.AccessHash,
	}
}

func (sc *SecretChat) Visualization() []string {
	if sc.AuthKey == nil {
		return nil
	}
	return crypto.KeyVisualization(sc.AuthKey)
}

// SecretChatManager manages a collection of active secret chats by their IDs.
// It provides thread-safe insertion, lookup, removal, and iteration over chats.
//
// Example:
//
//	mgr := telegram.NewSecretChatManager()
//	mgr.Put(chat)
//	if c, ok := mgr.Get(chat.ID); ok {
//		fmt.Println("found chat", c.ID)
//	}
//	mgr.Each(func(sc *telegram.SecretChat) {
//		fmt.Println("active chat:", sc.ID)
//	})
type SecretChatManager struct {
	mu    sync.RWMutex
	chats map[int32]*SecretChat
}

// NewSecretChatManager creates and returns a new SecretChatManager for tracking
// active secret chats.
//
// Example:
//
//	mgr := telegram.NewSecretChatManager()
//	mgr.Put(&telegram.SecretChat{ID: 42, State: telegram.SecretChatStateReady})
//	chat, ok := mgr.Get(42)
//	fmt.Println(chat.ID, ok)
//	// Output: 42 true
func NewSecretChatManager() *SecretChatManager {
	return &SecretChatManager{
		chats: make(map[int32]*SecretChat),
	}
}

func (m *SecretChatManager) Put(chat *SecretChat) {
	m.mu.Lock()
	m.chats[chat.ID] = chat
	m.mu.Unlock()
}

func (m *SecretChatManager) Get(id int32) (*SecretChat, bool) {
	m.mu.RLock()
	c, ok := m.chats[id]
	m.mu.RUnlock()
	return c, ok
}

func (m *SecretChatManager) Remove(id int32) {
	m.mu.Lock()
	delete(m.chats, id)
	m.mu.Unlock()
}

func (m *SecretChatManager) Each(fn func(*SecretChat)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.chats {
		fn(c)
	}
}
