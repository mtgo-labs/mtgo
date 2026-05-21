# Algorithmic Complexity & Caching Findings

## High Severity

### 1. O(n) linear scan for msgID deduplication

**File:** `internal/session/msg_id_validator.go:44`
**Severity:** high

`Check()` does a linear scan over `v.ids` (a `[]int64` with `msgIDReplayCapacity = 256`) for duplicate detection — O(n) per incoming message. A second loop at lines 50-57 is dead code (iterates to find a sorted position but never uses the result).

**Fix:** Replace with `map[int64]struct{}` for O(1) duplicate detection. Remove the dead code loop at lines 50-57.

---

## Medium Severity

### 2. AES key schedule recomputed per message

**File:** `internal/crypto/aes.go:43`
**Severity:** medium

`newAESBlock(key)` calls `aes.NewCipher(key)` on every call to `IGEEncrypt`, `IGEDecrypt`, `ctrCrypt`, and `NewCTRCipher`. For the session encryption path, the same 32-byte key is used for every message in a session. `cipher.Block` is safe for concurrent use and immutable once created.

**Fix:** Accept a `cipher.Block` parameter, or cache the block at the session/connection level.

---

### 3. PBKDF2 (100k iterations) with no caching

**File:** `internal/crypto/srp.go:49`
**Severity:** medium

`ComputePasswordHash` runs `pbkdf2.Key` with 100,000 iterations. This is the most expensive single CPU operation in the codebase. If the same password+salt combination is submitted multiple times (retry scenarios), the full cost is paid again.

**Fix:** Document that callers should cache the result. For concurrent scenarios, wrap in `singleflight`.

---

### 4. p.Bytes() allocated twice in SRP

**File:** `internal/crypto/srp.go:76, 119`
**Severity:** medium

`p.Bytes()` allocates a new byte slice from `big.Int` on every call. The SRP prime `p` is immutable within a session but `.Bytes()` is called in two separate places (`k` computation and `M1` computation).

**Fix:** Cache once: `pBytes := p.Bytes()` and reuse.

---

### 5. smallPrimes redeclared per call

**File:** `internal/crypto/prime.go:39`
**Severity:** medium

`smallPrimes` is redeclared as a local slice variable on every call to `Decpose`. The function is deterministic and the primes never change.

**Fix:** Hoist to package-level `var`.

---

### 6. auth_helpers computeKeyAndIV allocations

**File:** `internal/session/auth_helpers.go:41-43`
**Severity:** low

`computeKeyAndIV` uses `append(newNonce, serverNonce...)` inside `sha1Hash()`. Three calls = three allocations. Since `newNonce` and `serverNonce` are fixed-length (16 bytes each), a pre-sized `[32]byte` avoids heap allocation.

**Fix:** Use stack-allocated buffer like `KDF` in `mtproto.go`.

---

### 7. fmt.Sprintf("%T") in per-RPC path

**File:** `internal/session/session.go:817-826`
**Severity:** low

`typeName` uses `fmt.Sprintf("%T", v)` which relies on reflection. Called on every `Invoke` for error formatting.

**Fix:** Type switch with string constants for common RPC types, or cache results per ConstructorID.

---

## Additional Findings (discovered by cross-review)

### 8. Suspicious padding calculation in SecretEncrypt

**File:** `internal/crypto/secret.go:119-126`
**Severity:** medium

`paddingLen` is initialized to `SecretChatMinPadding + (SecretChatMaxPadding - SecretChatMinPadding)` (constant 1024), then conditionally recalculated based on alignment. The initial value of 1024 is only used if the modulo check passes, which appears to be almost never the intended behavior. Verify the intended padding logic.

**Fix:** Review and clarify the padding calculation. The constant 1024 initial value may be a bug.

---

## Potential Caching Opportunities

### AES Block Cipher (session-level)

The `cipher.Block` produced by `aes.NewCipher(key)` is immutable and goroutine-safe. The session already holds the auth key — add a cached `cipher.Block` field to avoid key schedule recomputation on every message.

### SRP Password Hash (authentication)

`ComputePasswordHash` is the most expensive computation in the codebase. Any retry path resubmitting the same password+salt pays the full 100k-iteration cost. If the library supports concurrent auth attempts (e.g., multiple DCs), `singleflight` would deduplicate them.

### Obfuscated Transport CTCipher

`CTRCipher` already maintains counter state, but `Process()` allocates a fresh output buffer each call. An in-place variant or output-buffer-accepting variant would eliminate the per-packet allocation for both read and write paths.
