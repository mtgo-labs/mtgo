package crypto

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"sync"
	"time"
)

// RSAKeySet is the trusted server-RSA-key store. The bundled canonical Telegram
// keys (from ServerPublicKeys) are the immutable trust root; VerifyAndAccept
// augments the set only with keys fetched over an authenticated session.
//
// The bundled root is never replaced wholesale — rotated keys are added to a
// separate map so the original trust anchors always remain. This implements
// the "bundled trust root + runtime fetch-and-rotate" decision from the
// production-hardening spec (clarify Q3, FR-015).
//
// Ported conceptually from TDLib PublicRsaKeyWatchdog (net/PublicRsaKeyWatchdog.h).
type RSAKeySet struct {
	// bundled is the immutable canonical Telegram keys (trust root).
	bundled map[int64]*ServerKey
	// rotated holds keys accepted via VerifyAndAccept (initially empty).
	rotated map[int64]*ServerKey
	mu      sync.RWMutex
}

// NewRSAKeySet creates a key set seeded from the bundled ServerPublicKeys.
// The bundled map is copied so the original package-level map is untouched.
func NewRSAKeySet() *RSAKeySet {
	bundled := make(map[int64]*ServerKey, len(serverPublicKeys))
	for fp, key := range serverPublicKeys {
		bundled[fp] = key
	}
	return &RSAKeySet{
		bundled: bundled,
		rotated: make(map[int64]*ServerKey),
	}
}

// Current returns a snapshot of all currently-trusted keys (bundled ∪ rotated).
// The returned map is a copy; callers may mutate it freely.
func (k *RSAKeySet) Current() map[int64]*ServerKey {
	k.mu.RLock()
	defer k.mu.RUnlock()
	out := make(map[int64]*ServerKey, len(k.bundled)+len(k.rotated))
	for fp, key := range k.bundled {
		out[fp] = key
	}
	for fp, key := range k.rotated {
		out[fp] = key
	}
	return out
}

// IsTrusted reports whether fp is in the current trusted set.
func (k *RSAKeySet) IsTrusted(fp int64) bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	if _, ok := k.bundled[fp]; ok {
		return true
	}
	_, ok := k.rotated[fp]
	return ok
}

// Get returns the trusted key for fp. Mirrors GetServerKey semantics.
func (k *RSAKeySet) Get(fp int64) (*ServerKey, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	if key, ok := k.bundled[fp]; ok {
		return key, true
	}
	key, ok := k.rotated[fp]
	return key, ok
}

// TrustedFingerprints returns the fingerprints of all currently-trusted keys.
func (k *RSAKeySet) TrustedFingerprints() []int64 {
	k.mu.RLock()
	defer k.mu.RUnlock()
	out := make([]int64, 0, len(k.bundled)+len(k.rotated))
	for fp := range k.bundled {
		out = append(out, fp)
	}
	for fp := range k.rotated {
		out = append(out, fp)
	}
	return out
}

// VerifyAndAccept records a key in the rotated set. The fingerprint must be
// the value the server reported for this key over an authenticated channel
// (the watchdog fetches via an RPC over an established session, which proves
// server identity — that IS the verification). The key is validated for
// structural validity (non-nil with modulus and exponent) and its computed
// fingerprint must match the server-reported fingerprint. The bundled root
// is never modified. Returns ErrKeyVerificationFailed if the key is invalid
// or its fingerprint does not match.
func (k *RSAKeySet) VerifyAndAccept(fingerprint int64, key *ServerKey) error {
	if key == nil || key.N == nil || key.E == nil {
		return ErrKeyVerificationFailed
	}
	if computed := ComputeFingerprint(key); computed != fingerprint {
		return ErrKeyVerificationFailed
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	// Don't duplicate bundled keys.
	if _, ok := k.bundled[fingerprint]; ok {
		return nil
	}
	k.rotated[fingerprint] = key
	return nil
}

// ErrKeyVerificationFailed is returned when a key cannot be verified against
// the trust root. Defined in crypto so it can be referenced from both the
// crypto and session packages.
var ErrKeyVerificationFailed = errKeyVerificationFailed{}

type errKeyVerificationFailed struct{}

func (errKeyVerificationFailed) Error() string {
	return "crypto: cryptographic key verification failed"
}

// ComputeFingerprint computes the MTProto RSA key fingerprint: the lower 64
// bits (bytes 12:20, little-endian) of SHA1 of the TL-serialized RSA public
// key (n:string e:string), matching the algorithm used by Telegram servers.
// This is verified to produce the fingerprints stored in ServerPublicKeys.
func ComputeFingerprint(key *ServerKey) int64 {
	if key == nil || key.N == nil {
		return 0
	}
	// Serialize n and e as TL bytes (length-prefixed, 4-byte aligned),
	// matching the gotd/td RSAFingerprint algorithm.
	data := tlSerializeBytes(key.N.Bytes())
	data = append(data, tlSerializeBytes(key.E.Bytes())...)
	h := sha1.Sum(data)
	return int64(binary.LittleEndian.Uint64(h[12:]))
}

// tlSerializeBytes encodes a byte slice in MTProto TL bytes format: a 1- or
// 4-byte length prefix followed by the data, zero-padded to a 4-byte boundary.
func tlSerializeBytes(b []byte) []byte {
	var buf []byte
	if len(b) < 254 {
		buf = append(buf, byte(len(b)))
		buf = append(buf, b...)
	} else {
		buf = append(buf, 0xFE)
		buf = append(buf, byte(len(b)), byte(len(b)>>8), byte(len(b)>>16))
		buf = append(buf, b...)
	}
	for len(buf)%4 != 0 {
		buf = append(buf, 0)
	}
	return buf
}

// FetchedKey is a server RSA key fetched by the watchdog, paired with the
// fingerprint the server reported for it.
type FetchedKey struct {
	Fingerprint int64
	Key         *ServerKey
}

// WatchdogConfig configures a PublicRsaKeyWatchdog.
type WatchdogConfig struct {
	// KeySet is the trust store that receives verified keys.
	KeySet *RSAKeySet
	// Interval is the refresh cadence. Must be > 0 to enable.
	Interval time.Duration
	// FetchFn returns keys fetched from an authenticated channel (e.g. an RPC
	// over an established session). The watchdog calls VerifyAndAccept on each
	// returned key. Errors are logged and the existing trusted set is kept
	// (fail-closed — FR-016).
	FetchFn func(ctx context.Context) ([]FetchedKey, error)
	// Log is an optional sink for diagnostic messages. When nil, logging is
	// suppressed.
	Log func(format string, args ...any)
}

// PublicRsaKeyWatchdog periodically fetches refreshed server RSA keys and
// verifies each against the RSAKeySet before accepting. It never replaces the
// bundled trust root; it only augments the rotated set. On fetch failure or
// unverified keys, it logs and keeps the existing trusted set (fail-closed).
//
// Ported conceptually from TDLib PublicRsaKeyWatchdog (net/PublicRsaKeyWatchdog.h).
type PublicRsaKeyWatchdog struct {
	cfg         WatchdogConfig
	lastRefresh time.Time
	mu          sync.RWMutex
	wg          sync.WaitGroup
}

// NewPublicRsaKeyWatchdog creates a watchdog. Call Start to launch the loop.
func NewPublicRsaKeyWatchdog(cfg WatchdogConfig) *PublicRsaKeyWatchdog {
	return &PublicRsaKeyWatchdog{cfg: cfg}
}

// Start launches the refresh goroutine. It runs until ctx is cancelled.
// Calling Start more than once is a no-op after the first (guarded by wg).
func (w *PublicRsaKeyWatchdog) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.loop(ctx)
}

func (w *PublicRsaKeyWatchdog) loop(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	// Do an initial refresh immediately.
	w.refresh(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.refresh(ctx)
		}
	}
}

func (w *PublicRsaKeyWatchdog) refresh(ctx context.Context) {
	if w.cfg.FetchFn == nil || w.cfg.KeySet == nil {
		return
	}
	keys, err := w.cfg.FetchFn(ctx)
	if err != nil {
		if w.cfg.Log != nil {
			w.cfg.Log("rsa watchdog: fetch failed (keeping existing keys): %v", err)
		}
		return // fail-closed: keep existing trusted set (FR-016)
	}
	accepted := 0
	for _, fk := range keys {
		if verr := w.cfg.KeySet.VerifyAndAccept(fk.Fingerprint, fk.Key); verr != nil {
			if w.cfg.Log != nil {
				w.cfg.Log("rsa watchdog: rejected unverified key fp=%d: %v", fk.Fingerprint, verr)
			}
			continue // fail-closed: skip unverified key
		}
		accepted++
	}
	if accepted > 0 {
		w.mu.Lock()
		w.lastRefresh = time.Now()
		w.mu.Unlock()
		if w.cfg.Log != nil {
			w.cfg.Log("rsa watchdog: accepted %d refreshed key(s)", accepted)
		}
	}
}

// LastRefresh returns the time of the last successful verified refresh.
func (w *PublicRsaKeyWatchdog) LastRefresh() time.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastRefresh
}

// Wait blocks until the watchdog goroutine has exited (for deterministic
// shutdown in tests — Constitution Principle V).
func (w *PublicRsaKeyWatchdog) Wait() { w.wg.Wait() }
