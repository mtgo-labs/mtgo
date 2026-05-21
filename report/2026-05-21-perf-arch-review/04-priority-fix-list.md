# Priority Fix List

Ordered by estimated impact (allocations per message × ease of fix).

## Tier 1: Stack-allocate crypto temporaries (5-8 allocs eliminated per MTProto message)

Trivial changes, all follow the same pattern: `make([]byte, N)` → `var buf [N]byte`.

| # | File | What | Lines |
|---|------|------|-------|
| 1 | `crypto/aes.go` | IGE IV: `var iv1, iv2 [16]byte` | 66-67, 99-100 |
| 2 | `crypto/aes.go` | CTR keystream: `var keystream [16]byte` | 150 |
| 3 | `crypto/mtproto.go` | Padding: `var padding [28]byte` | 81, 130 |
| 4 | `crypto/secret.go` | msgKeyLargeInput: stack-alloc or pool | 138, 170 |
| 5 | `crypto/secret.go` | secretKDF: `var tmpA, tmpB [52]byte` + direct indexing | 194-213 |

**Total eliminated:** ~8 heap allocations per message on the encrypt/decrypt path.

---

## Tier 2: Correctness + high-ROI transport fixes

| # | File | What | Severity | Why |
|---|------|------|----------|-----|
| 6 | `transport/ws.go:268` | Pass bufio.Reader into newWSConn | **correctness** | Buffered bytes lost = data corruption |
| 7 | `transport/tcp_intermediate_noheader.go:63` | Replace 1 MiB per-read alloc with persistent buffer | high | 1 MiB per read syscall |
| 8 | `crypto/rsa.go:69` | Move `defer ReleaseAESBuf` out of loop | medium | Buffer accumulation until function exit |
| 9 | `transport/ws.go:85, 215-224` | Mask client-to-server control frames in writeControl | **correctness** | RFC 6455 violation, servers reject unmasked frames |
| 10 | `session/session.go:482-545` | Stop timers in all code paths (Send/SendRaw) | medium | Timer objects leak when timeout fires |
| 11 | `session/session.go:646-671` | Add ctx.Done() to SendRaw first select | medium | Goroutine blocks on cancelled context until timeout |

---

## Tier 3: Transport per-packet allocations

| # | File | What |
|---|------|------|
| 12 | `transport/tcp_obfuscated.go:182-201` | Stack-alloc header, avoid append-copy of payload |
| 13 | `transport/ws.go:130,165` | Stack-alloc frame header, reuse payload buffer |
| 14 | `transport/tcp_full.go:36` | Reuse struct-level send buffer |
| 15 | `transport/tcp_padded_intermediate.go:34,42,52` | Stack-alloc padding + lenBytes, reuse send buffer |
| 16 | `crypto/aes.go:196` | `CTRCipher.Process` — accept output buffer param |
| 17 | `transport/ws_stdlib.go:148,154` | In-place encrypt/decrypt for obfuscated conn |
| 18 | `transport/tcp_intermediate_noheader.go` | Wrap conn in bufio.Reader for batched small reads |
| 19 | `transport/tcp_intermediate_noheader.go:37-59` | Compact t.buf after draining, cap growth |

---

## Tier 4: Algorithmic + caching

| # | File | What |
|---|------|------|
| 20 | `session/msg_id_validator.go:44-57` | `map[int64]struct{}` for O(1) dedup, remove dead loop |
| 21 | `crypto/aes.go:43` | Cache `cipher.Block` at session level |
| 22 | `crypto/srp.go:49` | Document caller-caching for PBKDF2 |
| 23 | `crypto/srp.go:76,119` | Cache `p.Bytes()` locally |
| 24 | `crypto/prime.go:39` | Hoist `smallPrimes` to package level |
| 25 | `crypto/secret.go:119-126` | Review suspicious padding calculation (constant 1024 init) |

---

## Tier 5: Goroutine lifecycle + context awareness

| # | File | What |
|---|------|------|
| 26 | `session/session.go:626-639` | dispatchUpdate goroutine leak — select against cancel |
| 27 | `session/session.go:107-127` | globalHousekeeper shutdown mechanism |
| 28 | `session/session.go:857-859` | WaitGroup for background goroutines |
| 29 | `session/session.go:714,750` | Context-aware retry backoff |
| 30 | `session/session.go:859` | Unblock receiveLoop on writer failure |
| 31 | `transport/ws.go:113` | TLS dial timeout + context |
| 32 | `transport/dialer.go:23` | Add TCP keepalive (30s) to net.Dialer |

---

## Tier 6: Minor cleanups

- `session/session.go:354-359` — Remove redundant inner select in deliverResult
- `session/session.go:469` — Thread-safe SetTransport
- `session/session.go:1001,1010-1021` — Non-blocking sendJob.done, proper drain
- `session/session.go:213` — Close old cancel channel on reconnect
- `crypto/secret.go:174` — Constant-time comparison for msgKey
- `storage/storage.go:516-519` — Mutex for adapterWrapper.sess
- `transport/ws_stdlib.go:214` — Avoid string([]byte) in nonce check
- `session/session.go:1348` — Remove unnecessary `SetReadDeadline(time.Time{})` syscall per packet
- `session/session.go:1306` — Reuse timer with `Reset()` instead of `time.NewTimer` per retry
- `session/session.go:547` — Check `s.cancel` before `crypto.Pack` in sendRPCDrop
- `session/auth.go:373` — Remove dead code (`return nil, ErrDHGenRetry`)
- `transport/ws_stdlib.go:250` — Replace custom `readFull` with `io.ReadFull`
- Field alignment — Reorder struct fields in Session, sendJob, msgIDValidator, SRPResult, transport structs, storage structs (see 01-allocations.md)

---

## Estimated Impact

If Tier 1 + Tier 3 are fully addressed:
- **~10-15 fewer heap allocations per message** on the encrypt/send path
- **~8-12 fewer heap allocations per message** on the recv/decrypt path
- For a session processing 100 msgs/sec: ~2000 fewer GC-eligible objects/sec
- GC pressure reduction is the primary win; CPU improvement is secondary

Tier 2 item 6 (ws.go bufio fix) is a **correctness fix**, not just performance — should be prioritized regardless.
