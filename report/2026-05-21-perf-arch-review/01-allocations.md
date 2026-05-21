# Allocation & Memory Layout Findings

## Critical Hot-Path Allocations (per-message)

These run on every encrypted message or network packet.

### 1. IGE IV allocations

**File:** `internal/crypto/aes.go:66-67, 99-100`
**Severity:** high

`IGEEncrypt` and `IGEDecrypt` allocate `iv1 := make([]byte, 16)` and `iv2 := make([]byte, 16)` on every call. IGE encrypt/decrypt runs for every MTProto message. The rest of the IGE loop already uses stack-allocated `[16]byte` for `xored`/`encrypted` — these IV slices are the odd ones out.

**Fix:**
```go
var iv1, iv2 [16]byte
copy(iv1[:], iv[:16])
copy(iv2[:], iv[16:32])
```

---

### 2. CTR mode output + keystream allocations

**File:** `internal/crypto/aes.go:150, 196`
**Severity:** high

- `ctrCrypt` (standalone) allocates `out := make([]byte, len(data))` and `keystream := make([]byte, 16)` per call.
- `CTRCipher.Process` allocates `out := make([]byte, len(data))` per call. This is the hot path for obfuscated transport encryption — every packet.

The keystream is trivially stack-allocatable. The output buffer could accept a caller-provided slice.

**Fix:**
```go
// keystream — stack allocate
var keystream [16]byte

// Process — accept output buffer
func (c *CTRCipher) Process(data []byte, out []byte) []byte { ... }
```

---

### 3. MTProto padding allocation

**File:** `internal/crypto/mtproto.go:81, 130`
**Severity:** high

`Pack` and `PackRaw` allocate `padding := make([]byte, paddingLen)` (12-28 bytes) on every outgoing message.

**Fix:**
```go
var padding [28]byte // max padding is 12+16=28
rand.Read(padding[:paddingLen])
```

---

### 4. Secret chat msgKeyLargeInput allocation

**File:** `internal/crypto/secret.go:138, 170`
**Severity:** high

`SecretEncrypt` and `SecretDecrypt` allocate `msgKeyLargeInput := make([]byte, 32+len(data))` on every e2e message.

**Fix:** Use a stack-allocated staging buffer (MTProto packet limits bound the max size) or a pool, similar to how `KDF` in `mtproto.go` uses fixed-size arrays.

---

### 5. secretKDF heap allocations

**File:** `internal/crypto/secret.go:194, 199, 204-213`
**Severity:** high

`secretKDF` heap-allocates `tmpA` and `tmpB` (always 52 bytes) and builds `aesKey`/`aesIV` via 6 `append` calls. But `mtproto.go KDF` does the identical computation with `var tmpA [52]byte` + direct indexing.

**Fix:** Copy the `KDF` pattern:
```go
var tmpA, tmpB [52]byte
copy(tmpA[:], msgKey)
copy(tmpA[len(msgKey):], sha256_a[8:])

aesKey := make([]byte, 32)
copy(aesKey[0:8], sha256_a[8:8+8])
copy(aesKey[8:8+16], sha256_b[8:8+16])
copy(aesKey[24:32], sha256_a[24:32])
```

---

### 6. Obfuscated transport Send double-copy

**File:** `internal/transport/tcp_obfuscated.go:182-201`
**Severity:** high

`Send` allocates `header := make([]byte, 4)` then `append(header, data...)` copies the entire payload. Combined with `CTRCipher.Process` also allocating output, each send does **two full payload copies**.

The abridged branch (lines 191-201) has the same pattern with different header sizes.

**Fix:**
```go
var header [4]byte
binary.LittleEndian.PutUint32(header[:], uint32(len(data)))
encHeader := t.enc.Process(header[:])
encData := t.enc.Process(data)
_, err := t.conn.Write(append(encHeader, encData...))
```
Or better: encrypt in-place and write header+data separately.

---

### 7. WebSocket per-frame allocations

**File:** `internal/transport/ws.go:130, 165`
**Severity:** high

`readFrame` allocates `make([]byte, 2)` for header and `make([]byte, length)` for payload on every frame. `writeFrame` also allocates per frame.

**Fix:**
```go
// header — stack allocate
var header [2]byte
io.ReadFull(c.br, header[:])

// payload — reuse a read buffer on wsConn struct
```

---

## Medium-Severity Allocations

### 8. sync.Pool tuning for IGE buffers

**File:** `internal/crypto/aes.go:9-14`
**Severity:** medium

`igeBufPool` starts with 4096-byte buffers. If the first IGE operation is larger, the pool buffer is discarded and a new one allocated. The initial 4096 is arbitrary.

**Fix:** Consider initializing at 0 capacity since `getAESBuf` handles growth.

---

### 9. sha256.New per SRP call

**File:** `internal/crypto/srp.go:11`
**Severity:** medium

`sha256sum` allocates `sha256.New()` on every call. `ComputeSRP` calls it 6 times, `ComputePasswordHash` calls it 3 times.

**Fix:** Use `sync.Pool` for hash objects or restructure to reuse a single hasher with `Reset()`. Low priority since SRP is infrequent (2FA setup).

---

### 10. xorBytes byte-by-byte

**File:** `internal/crypto/srp.go:19-24`
**Severity:** medium

`xorBytes` loops byte-by-byte instead of using `subtle.XORBytes` (Go 1.20+) which is SIMD-optimized.

**Fix:** `subtle.XORBytes(out, a, b)`.

---

### 11. string([]byte) in nonce validation

**File:** `internal/transport/ws_stdlib.go:214`
**Severity:** medium

`invalidObfuscated2Nonce` does `switch string(nonce[:4])` which allocates a new string per call during nonce retry loops.

**Fix:** Use `binary.LittleEndian.Uint32` against known constants, like `tcp_obfuscated.go:28`.

---

### 12. Per-send packet allocations in transports

**Files:**
- `transport/tcp_full.go:36` — `make([]byte, 4+4+len(data)+4)` per send
- `transport/tcp_padded_intermediate.go:34,42` — padding + packet per send
- `transport/tcp_padded_intermediate.go:52` — `make([]byte, 4)` per recv (other transports use `var [4]byte`)

**Severity:** medium

**Fix:** Reuse struct-level send buffers across calls; use `var lenBytes [4]byte` for recv.

---

### 13. Double copy through CTCipher.Process

**File:** `internal/transport/ws_stdlib.go:148, 154-159`
**Severity:** medium

`obfsConn.Read` calls `c.dec.Process(buf[:n])` (allocates new slice), then `copy(p, plain)`. Same for Write. Every read/write allocates a temp buffer equal to the data size.

**Fix:** Add in-place encrypt/decrypt variant to `CTRCipher`, or accept output buffer param.

---

### 14. RPC result payload copy

**File:** `internal/session/session.go:1153-1161`
**Severity:** medium

`handleRawRPCResult` copies `payload` into a fresh `make([]byte, len(payload))` on every RPC response. Necessary because the source buffer is pooled, but the copy+alloc could be eliminated with pool release coordination.

**Fix:** Defer pool release until after the channel consumer finishes reading.

---

### 15. msg_id_validator backing array leak

**File:** `internal/session/msg_id_validator.go:59-61`
**Severity:** medium

When `len(v.ids) > msgIDReplayCapacity`, `v.ids = v.ids[len(v.ids)-msgIDReplayCapacity:]` reslices but retains the full backing array.

**Fix:** Copy-based trim:
```go
copy(v.ids, v.ids[len(v.ids)-msgIDReplayCapacity:])
v.ids = v.ids[:msgIDReplayCapacity]
```

---

### 16. Dead code in msgIDValidator.Check

**File:** `internal/session/msg_id_validator.go:50-57`
**Severity:** medium

A loop iterates over `v.ids` comparing elements but does nothing — appears to be leftover from an incomplete binary search or insertion sort.

**Fix:** Remove the dead loop.

---

## Low-Severity Allocations

- `transport/tcp_intermediate_noheader.go:63` — 1 MiB scratch buffer per read call (also in I/O report)
- `transport/ws_stdlib.go:179-192, 222-248` — One-time connection setup allocations (key/IV), low impact
- `crypto/rsa.go:42-46` — `rsapad` stack-allocatable temps (192+32 bytes), only during key exchange
- `session/auth_helpers.go:41-43` — `append(newNonce, serverNonce...)` may alias backing array
- `session/auth_helpers.go:26-29` — `sha1Hash` forces `[20]byte` to heap, 20-byte allocation per call
- `transport/tcp_obfuscated.go:236` — `Recv` allocates via `t.dec.Process(data)` return value

---

## Field Alignment Waste (discovered by cross-review)

Struct field ordering causes unnecessary padding. None are on hot paths, but they increase per-session/per-connection memory footprint at zero runtime cost to fix.

| File | Struct | Bytes wasted | Fix |
|------|--------|-------------|-----|
| `session/session.go` | `Session` | 120 | Group pointer/slice/map/chan fields together, then scalar fields |
| `session/session.go` | `sendJob` | 8 | Move `deadline time.Time` before `done chan error` |
| `session/msg_id_validator.go` | `msgIDValidator` | 24 | Move `mu sync.Mutex` after pointer fields |
| `crypto/srp.go` | `SRPResult` | 8 | Place slice fields first, then int64 |
| `transport/tcp_obfuscated.go` | `TCPObfuscated` | 8 | Group byte/bool fields after pointer fields |
| `transport/tcp_full.go` | `TCPFull` | 8 | Move `seqNo uint32` after `readBuf` |
| `transport/ws_stdlib.go` | `wsConn` | 32 | Group bool fields after pointer/interface fields |
| `storage/storage.go` | `Session` | 16 | Group string fields, then []byte, then int/bool |
| `storage/storage.go` | `Peer` | 40 | Move `Type PeerType` to end after all string fields |
| `storage/storage.go` | `DCAuthEntry` | 32 | Group []byte/string fields, then int/int64 |
| `storage/storage.go` | `Conversation` | 24 | Same pattern |
| `storage/storage.go` | `DurableUpdate` | 16 | Same pattern |
| `storage/storage.go` | `memoryAdapter` | 32 | Reorder pointer and non-pointer fields |

**Verification:** Run `fieldalignment ./internal/...` to see exact before/after sizes.
