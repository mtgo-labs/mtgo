package session

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// TempKeyManager manages PFS temporary auth key lifecycle.
// Ported from td/td/telegram/net/Session.cpp:1488-1498 (auth_loop TmpAuthKey).
type TempKeyManager struct {
	dcID      int
	testMode  bool
	permKey   []byte    // permanent auth key
	tempKey   []byte    // current temp auth key
	tempKeyID int64     // SHA1-based temp key ID
	expiresAt time.Time // when the temp key expires
	issuedAt  time.Time // when the current temp key was generated
	// bindExpiresAt is the server-relative expiry sent to auth.bindTempAuthKey.
	// expiresAt remains local so rotation scheduling is unaffected by clock skew.
	bindExpiresAt int32
	bound         bool      // whether auth.bindTempAuthKey succeeded
	enabled       bool      // PFS mode flag
	createdAt     time.Time // when this manager (and perm key) was initialized
	needInit      bool      // caller must call initConnection after bind
	storage       storage.Storage
	mu            sync.Mutex
	bindMu        sync.Mutex
	bindEpoch     uint64
}

// NewTempKeyManager creates a new temp key manager.
func NewTempKeyManager(dcID int, testMode bool, permKey []byte, enabled bool, st storage.Storage, permKeyCreatedAt ...time.Time) *TempKeyManager {
	var createdAt time.Time
	if len(permKeyCreatedAt) > 0 {
		createdAt = permKeyCreatedAt[0]
	}
	return &TempKeyManager{
		dcID:      dcID,
		testMode:  testMode,
		permKey:   bytes.Clone(permKey),
		enabled:   enabled,
		createdAt: createdAt,
		storage:   st,
	}
}

// IsEnabled reports whether PFS mode is active.
func (m *TempKeyManager) IsEnabled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.enabled
}

// IsBound reports whether the temp key has been successfully bound to the
// permanent key via auth.bindTempAuthKey.
func (m *TempKeyManager) IsBound() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bound
}

// PermKey returns the permanent auth key. Used for fallback when bind fails.
func (m *TempKeyManager) PermKey() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return bytes.Clone(m.permKey)
}

// NeedsInitConnection reports whether the caller must call initConnection
// (wrapping help.getConfig) after a successful temp key binding. The flag is
// set by Bind and cleared by MarkInitConnectionDone.
//
// The PFS spec requires rewriting client info after each binding — see
// https://core.telegram.org/api/pfs.
func (m *TempKeyManager) NeedsInitConnection() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.needInit
}

// MarkInitConnectionDone records that client connection metadata was
// successfully rewritten after binding.
func (m *TempKeyManager) MarkInitConnectionDone() {
	m.mu.Lock()
	m.needInit = false
	m.mu.Unlock()
}

// BindEpoch identifies the latest successful binding. It lets concurrent
// AUTH_KEY_PERM_EMPTY recoveries share one bind instead of racing duplicates.
func (m *TempKeyManager) BindEpoch() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bindEpoch
}

// GetKey returns the current temp key and key ID. If PFS is disabled or no
// temp key exists, returns the permanent key.
func (m *TempKeyManager) GetKey() ([]byte, int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.enabled || len(m.tempKey) == 0 {
		return bytes.Clone(m.permKey), computeAuthKeyIDInt64(m.permKey)
	}
	return bytes.Clone(m.tempKey), m.tempKeyID
}

// NeedsRotation reports whether the temp key is approaching expiry and needs rotation.
func (m *TempKeyManager) NeedsRotation() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.enabled || len(m.tempKey) == 0 {
		return false
	}
	return m.rotationDueInLocked() <= 0
}

func (m *TempKeyManager) rotationDueIn() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.enabled || len(m.tempKey) == 0 {
		return 0
	}
	return m.rotationDueInLocked()
}

func (m *TempKeyManager) rotationDueInLocked() time.Duration {
	margin := 30 * time.Second
	if lifetime := m.expiresAt.Sub(m.issuedAt); lifetime > 0 {
		margin = lifetime / 4
		if margin < 30*time.Second {
			margin = 30 * time.Second
		}
	}
	return time.Until(m.expiresAt.Add(-margin))
}

// FallbackToPermKey disables PFS for this session (e.g., after bind failure).
func (m *TempKeyManager) FallbackToPermKey() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
	m.tempKey = nil
	m.tempKeyID = 0
	m.bindExpiresAt = 0
	m.bound = false
}

// Generate performs DH exchange to generate a new temp key for PFS.
// Uses p_q_inner_data_temp_dc so the server treats the key as temporary.
// Ported from td/td/telegram/net/Session.cpp:1488-1498 (create_gen_auth_key_actor).
func (m *TempKeyManager) Generate(transport Transport) error {
	auth := &Auth{
		DC:       m.dcID,
		TestMode: m.testMode,
	}

	// Request a temp key with 24h expiry, matching MadelineProto's PFS_DURATION.
	expiresIn := int32(24 * 60 * 60) // 24 hours
	result, err := auth.CreateTemp(transport, expiresIn)
	if err != nil {
		return fmt.Errorf("temp key DH exchange: %w", err)
	}

	m.installGeneratedKey(result, expiresIn, time.Now())

	return nil
}

func (m *TempKeyManager) installGeneratedKey(result *AuthResult, expiresIn int32, localNow time.Time) {
	m.mu.Lock()
	m.tempKey = result.AuthKey
	m.tempKeyID = computeAuthKeyIDInt64(result.AuthKey)
	m.issuedAt = localNow
	m.expiresAt = localNow.Add(time.Duration(expiresIn) * time.Second)
	m.bindExpiresAt = result.ServerTime + expiresIn
	m.bound = false
	m.mu.Unlock()
}

// deriveMsgAESKeyIV computes the MTProto v1 AES key and IV from an auth key
// and message key. x is the offset into auth_key (0 for client→server,
// 8 for server→client). This is the same algorithm used in session/tdesktop.
func deriveMsgAESKeyIV(authKey []byte, msgKey [16]byte, x int) (key [32]byte, iv [32]byte) {
	var dataA, dataB, dataC, dataD [48]byte
	copy(dataA[:16], msgKey[:])
	copy(dataA[16:], authKey[x:x+32])
	copy(dataB[:16], authKey[x+32:x+48])
	copy(dataB[16:32], msgKey[:])
	copy(dataB[32:], authKey[x+48:x+64])
	copy(dataC[:32], authKey[x+64:x+96])
	copy(dataC[32:], msgKey[:])
	copy(dataD[:16], msgKey[:])
	copy(dataD[16:], authKey[x+96:x+128])

	sha1A := sha1.Sum(dataA[:])
	sha1B := sha1.Sum(dataB[:])
	sha1C := sha1.Sum(dataC[:])
	sha1D := sha1.Sum(dataD[:])

	copy(key[0:8], sha1A[0:8])
	copy(key[8:20], sha1B[8:20])
	copy(key[20:32], sha1C[4:16])

	copy(iv[0:12], sha1A[8:20])
	copy(iv[12:20], sha1B[0:8])
	copy(iv[20:24], sha1C[16:20])
	copy(iv[24:32], sha1D[0:8])
	return key, iv
}

// buildEncryptedBindMessage constructs the encrypted_message for
// auth.bindTempAuthKey, following the format described at
// https://core.telegram.org/method/auth.bindTempAuthKey#binding-message-contents
//
// The message contains a bind_auth_key_inner payload, wrapped in a standard
// MTProto message structure, encrypted with AES-IGE using a key derived from
// the permanent auth key.
func (m *TempKeyManager) buildEncryptedBindMessage(permKey, tempKey []byte, permKeyID, nonce, sessionID, outerMsgID int64, expiresAt int32) ([]byte, error) {
	// 1. Serialize bind_auth_key_inner.
	inner := &tg.BindAuthKeyInner{
		Nonce:         nonce,
		TempAuthKeyID: computeAuthKeyIDInt64(tempKey),
		PermAuthKeyID: permKeyID,
		TempSessionID: sessionID,
		ExpiresAt:     expiresAt,
	}
	var innerBuf bytes.Buffer
	if err := inner.Encode(&innerBuf); err != nil {
		return nil, fmt.Errorf("encode bind_auth_key_inner: %w", err)
	}
	innerBytes := innerBuf.Bytes()

	// 2. Build MTProto message: random(16) + msg_id(8) + seq_no(4) + length(4) + data
	var randPrefix [16]byte
	if _, err := rand.Read(randPrefix[:]); err != nil {
		return nil, fmt.Errorf("generate random prefix: %w", err)
	}
	msg := make([]byte, 0, 32+len(innerBytes))
	msg = append(msg, randPrefix[:]...)
	var buf8 [8]byte
	binary.LittleEndian.PutUint64(buf8[:], uint64(outerMsgID))
	msg = append(msg, buf8[:]...)
	var buf4 [4]byte
	binary.LittleEndian.PutUint32(buf4[:], 0) // seq_no = 0
	msg = append(msg, buf4[:]...)
	binary.LittleEndian.PutUint32(buf4[:], uint32(len(innerBytes)))
	msg = append(msg, buf4[:]...)
	msg = append(msg, innerBytes...)

	// 3. msg_key = last 16 bytes of SHA1(message)
	msgHash := sha1.Sum(msg)
	var msgKey [16]byte
	copy(msgKey[:], msgHash[4:20])

	// 4. Pad to 16-byte multiple with random bytes.
	padLen := (16 - len(msg)%16) % 16
	if padLen > 0 {
		pad := make([]byte, padLen)
		if _, err := rand.Read(pad); err != nil {
			return nil, fmt.Errorf("generate padding: %w", err)
		}
		msg = append(msg, pad...)
	}

	// 5. Derive AES key/IV from permanent auth key + msg_key (x=0 client→server).
	aesKey, aesIV := deriveMsgAESKeyIV(permKey, msgKey, 0)

	// 6. AES-IGE encrypt.
	encrypted, err := crypto.IGEEncrypt(msg, aesKey[:], aesIV[:])
	if err != nil {
		return nil, fmt.Errorf("encrypt binding message: %w", err)
	}
	defer crypto.ReleaseAESBuf(encrypted)

	// 7. Final: perm_auth_key_id(8) + msg_key(16) + encrypted_data.
	result := make([]byte, 0, 8+16+len(encrypted))
	binary.LittleEndian.PutUint64(buf8[:], uint64(permKeyID))
	result = append(result, buf8[:]...)
	result = append(result, msgKey[:]...)
	result = append(result, encrypted...)
	return result, nil
}

// ErrBindRequiresKeyRotation signals that auth.bindTempAuthKey returned
// ENCRYPTED_MESSAGE_INVALID and the permanent auth key is older than 60 seconds.
// Both the permanent and temporary keys must be dropped and recreated.
var ErrBindRequiresKeyRotation = fmt.Errorf("session: ENCRYPTED_MESSAGE_INVALID with stale perm key; both keys must be recreated")

// bindTempAuthKeyQuery is materialized only after Session.Invoke allocates the
// outer MTProto message ID. The PFS spec requires the encrypted inner message
// to carry that exact ID, so every retry must rebuild the nonce and payload.
type bindTempAuthKeyQuery struct {
	manager   *TempKeyManager
	permKey   []byte
	tempKey   []byte
	permKeyID int64
	sessionID int64
	expiresAt int32
}

func (*bindTempAuthKeyQuery) ConstructorID() uint32 { return tg.AuthBindTempAuthKeyTypeID }

func (*bindTempAuthKeyQuery) Encode(*bytes.Buffer) error {
	return fmt.Errorf("session: auth.bindTempAuthKey requires an outer message ID")
}

func (q *bindTempAuthKeyQuery) prepareForMessageID(msgID int64) (tg.TLObject, error) {
	var nonceBytes [8]byte
	if _, err := rand.Read(nonceBytes[:]); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	nonce := int64(binary.LittleEndian.Uint64(nonceBytes[:]))
	encMsg, err := q.manager.buildEncryptedBindMessage(
		q.permKey,
		q.tempKey,
		q.permKeyID,
		nonce,
		q.sessionID,
		msgID,
		q.expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("build bind message: %w", err)
	}
	return &tg.AuthBindTempAuthKeyRequest{
		PermAuthKeyID:    q.permKeyID,
		Nonce:            nonce,
		ExpiresAt:        q.expiresAt,
		EncryptedMessage: encMsg,
	}, nil
}

// Bind calls auth.bindTempAuthKey to bind the temp key to the permanent key.
// The encrypted_message is constructed per the MTProto PFS spec.
// Ported from td/td/telegram/net/Session.cpp:1556-1579 (need_send_bind_key).
//
// If the server returns ENCRYPTED_MESSAGE_INVALID and the permanent key was
// created more than 60 seconds ago, Bind returns ErrBindRequiresKeyRotation.
// The caller must then drop both keys, recreate them, and retry.
// See https://core.telegram.org/api/pfs for the full recovery procedure.
func (m *TempKeyManager) Bind(ctx context.Context, sessionID int64, invoke func(ctx context.Context, query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error)) error {
	m.bindMu.Lock()
	defer m.bindMu.Unlock()
	return m.bind(ctx, sessionID, invoke)
}

// Rebind binds only if no other recovery has already advanced observedEpoch.
// The bool reports whether this call performed the bind.
func (m *TempKeyManager) Rebind(
	ctx context.Context,
	sessionID int64,
	observedEpoch uint64,
	invoke func(context.Context, tg.TLObject, int, time.Duration) (tg.TLObject, error),
) (bool, error) {
	m.bindMu.Lock()
	defer m.bindMu.Unlock()
	m.mu.Lock()
	currentEpoch := m.bindEpoch
	m.mu.Unlock()
	if currentEpoch != observedEpoch {
		return false, nil
	}
	return true, m.bind(ctx, sessionID, invoke)
}

func (m *TempKeyManager) bind(ctx context.Context, sessionID int64, invoke func(ctx context.Context, query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error)) error {
	m.mu.Lock()
	tempKey := bytes.Clone(m.tempKey)
	permKey := bytes.Clone(m.permKey)
	expiresAt := m.bindExpiresAt
	if expiresAt == 0 {
		expiresAt = int32(m.expiresAt.Unix())
	}
	createdAt := m.createdAt
	m.mu.Unlock()

	if len(tempKey) == 0 {
		return fmt.Errorf("temp key not generated")
	}
	if len(permKey) < 256 {
		return fmt.Errorf("permanent key too short: %d bytes", len(permKey))
	}

	bindReq := &bindTempAuthKeyQuery{
		manager:   m,
		permKey:   permKey,
		tempKey:   tempKey,
		permKeyID: computeAuthKeyIDInt64(permKey),
		sessionID: sessionID,
		expiresAt: expiresAt,
	}

	result, err := invoke(ctx, bindReq, 3, 10*time.Second)
	if err != nil {
		m.mu.Lock()
		m.bound = false
		m.mu.Unlock()

		// Handle ENCRYPTED_MESSAGE_INVALID per the PFS spec.
		if tgerr.Is(err, tgerr.ErrEncryptedMessageInvalid) {
			if time.Since(createdAt) > 60*time.Second {
				return ErrBindRequiresKeyRotation
			}
			return fmt.Errorf("auth.bindTempAuthKey (ENCRYPTED_MESSAGE_INVALID, key <60s old, will retry): %w", err)
		}
		return fmt.Errorf("auth.bindTempAuthKey: %w", err)
	}
	bindSucceeded := false
	switch result := result.(type) {
	case *tg.BoolTrue:
		bindSucceeded = true
	case tg.TLBool:
		bindSucceeded = bool(result)
	}
	if !bindSucceeded {
		m.mu.Lock()
		m.bound = false
		m.mu.Unlock()
		return fmt.Errorf("auth.bindTempAuthKey: unexpected result %T", result)
	}

	m.mu.Lock()
	m.bound = true
	m.needInit = true // caller must initConnection after bind per PFS spec
	m.bindEpoch++
	m.mu.Unlock()
	return nil
}
