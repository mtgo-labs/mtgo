# I/O & Concurrency Findings

## Critical

### 1. ws.go bufio.Reader discarded after HTTP upgrade — DATA CORRUPTION

**File:** `internal/transport/ws.go:268`
**Severity:** high (correctness bug)

`wsDial` creates `bufio.NewReader(conn)` for reading the HTTP upgrade response. Then `newWSConn(conn)` creates a **separate** `bufio.NewReaderSize(conn, 4096)`. Any bytes the first reader buffered beyond the HTTP response are lost, causing silent data corruption or hangs on the first WebSocket frame read.

**Fix:** Pass the first `bufio.Reader` into `newWSConn` instead of creating a fresh one, or wrap the connection in a shared `bufio.ReadWriter`.

---

### 2. 1 MiB scratch buffer per read call

**File:** `internal/transport/tcp_intermediate_noheader.go:63`
**Severity:** high

`fill()` allocates `make([]byte, 1<<20)` on every call, even for reading 1 byte to complete a 4-byte header. Called in a loop until enough bytes are buffered.

Additionally, `t.buf` grows by appending every read and is never trimmed or capped. Over a connection's lifetime, a single large message means the buffer stays at that capacity forever.

**Fix:** Use `io.ReadFull` for known-length reads, or keep a persistent read buffer on the struct.

---

### 3. Goroutine leak in dispatchUpdate

**File:** `internal/session/session.go:626-639`
**Severity:** high

When the session is stopped while goroutines are waiting on the `updateSem` semaphore (64 slots), goroutines already blocked on the channel send are stuck forever. The `cancel` channel close only helps future calls.

**Fix:** Make the semaphore acquire selectable against the cancel channel:
```go
select {
case <-s.cancel:
    return
case s.updateSem <- struct{}{}:
    defer func() { <-s.updateSem }()
}
```

---

## High-Severity Goroutine Lifecycle Issues

### 4. globalHousekeeper runs forever

**File:** `internal/session/session.go:107-127`
**Severity:** high

The `run()` goroutine in `globalHousekeeper` ticks indefinitely with no exit condition, even when no sessions are registered.

**Fix:** Add a `done` channel or `context.Context` to allow shutdown when all sessions are unregistered.

---

### 5. receiveLoop can block up to 2 minutes after writer fails

**File:** `internal/session/session.go:859`
**Severity:** high

If the `writer()` goroutine exits on error but `receiveLoop` is still blocked in `Recv()`, the session is in a half-dead state. The `readDeadline` (line 1323) eventually fires, but there's a window of up to `pingInterval * 2` (2 minutes) where the receive goroutine is stuck.

**Fix:** Close the transport from `Stop()` to unblock the read, or set an aggressive read deadline from the writer error path.

---

### 6. No WaitGroup for background goroutines

**File:** `internal/session/session.go:857, 859, 1032`
**Severity:** medium

The session starts writer, receiveLoop, and dispatch worker goroutines but doesn't track them with a `sync.WaitGroup`. `Stop()` returns immediately without waiting for goroutines to finish.

**Fix:** Add a `sync.WaitGroup` and expose a wait mechanism for callers.

---

## Medium-Severity Issues

### 7. TLS dial without timeout

**File:** `internal/transport/ws.go:113`
**Severity:** medium

`tls.DialWithDialer` uses `&net.Dialer{}` with no timeout and doesn't respect the passed-in `ctx` for cancellation.

**Fix:** Set `net.Dialer{Timeout: ...}` derived from ctx deadline.

---

### 8. defer inside loop — buffer accumulation

**File:** `internal/crypto/rsa.go:69`
**Severity:** medium

`rsapad` has `defer ReleaseAESBuf(aesEncrypted)` inside a `for` loop. Defers run at function exit, not loop iteration exit. All intermediate buffers accumulate until `rsapad` returns.

**Fix:** Call `ReleaseAESBuf(aesEncrypted)` explicitly at the end of each iteration.

---

### 9. Retry loops use time.Sleep without context

**File:** `internal/session/session.go:714, 750`
**Severity:** medium

`Invoke` and `InvokeRaw` use `time.Sleep(backoff)` between retries, blocking without respecting `ctx`.

**Fix:**
```go
select {
case <-ctx.Done():
    return ctx.Err()
case <-time.After(backoff):
}
```

---

### 10. deliverResult timer allocation

**File:** `internal/session/session.go:354-359`
**Severity:** medium

`deliverResult` uses `time.After(5 * time.Second)` as a fallback, allocating a new timer when the channel is momentarily full. The inner select with timer may be redundant given the outer `default` case.

**Fix:** Remove the inner select; use the `default` case for immediate drop. Or use a pooled timer.

---

### 11. sendJob.done channel can block writer

**File:** `internal/session/session.go:1001`
**Severity:** medium

`job.done <- err` is a blocking send on a size-1 channel. If the caller goroutine is slow to read, the writer blocks on the caller's receive readiness.

**Fix:** Non-blocking send with fallback:
```go
select {
case job.done <- err:
default:
}
```

---

### 12. Drain loop may leave orphaned sendJob waiters

**File:** `internal/session/session.go:1010-1021`
**Severity:** medium

After a write error, jobs are drained from `sendCh`. But jobs enqueued *after* the drain loop completes but before the writer returns are never processed, leaving callers blocked on `job.done` forever.

**Fix:** Close `sendCh` before draining to prevent new enqueues.

---

### 13. writeControl error silently discarded

**File:** `internal/transport/ws.go:85`
**Severity:** medium

When a ping frame arrives, `c.writeControl(wsOpPong, payload)` is called from within `Read()`, but its error return is silently discarded. Additionally, `writeControl` acquires `c.mu`, causing read-path contention on the write mutex.

**Fix:** Log or propagate the error; consider making pong writes non-blocking.

---

## Additional High-Severity Issues (discovered by cross-review)

### 14. Unmasked client-to-server WebSocket control frames

**File:** `internal/transport/ws.go:85, 215-224`
**Severity:** high (protocol compliance bug)

`writeControl` sends pong responses without masking. RFC 6455 section 5.3 requires all client-to-server frames to be masked. Strict WebSocket servers will reject unmasked frames and close the connection.

**Fix:** Apply the same masking logic from `writeFrame` to `writeControl`.

---

### 15. Timer leaks in Send/SendRaw

**File:** `internal/session/session.go:482-545, 646-698`
**Severity:** medium

`Send` and `SendRaw` create two `time.NewTimer` instances each (write deadline + response deadline). When a timer fires (timeout path), `Stop()` is never called. Timers that fire but aren't stopped leak until GC collects them. Under sustained timeouts, this accumulates unreleased timer objects.

**Fix:** Always call `timer.Stop()` in all code paths, including the timeout case:
```go
case <-writeTimer.C:
    writeTimer.Stop() // add this
    return ..., ErrTimeout
```

---

### 16. Missing ctx.Done() in SendRaw first select

**File:** `internal/session/session.go:646-671`
**Severity:** medium

`Send` checks `ctx.Done()` when sending to `sendCh` (line 509), but `SendRaw` does not check context cancellation in its first select block (lines 664-671). If the context is cancelled while waiting to enqueue, the goroutine blocks until the write timeout fires.

**Fix:** Add `case <-ctx.Done():` to the first select block in `SendRaw`, matching the `Send` pattern.

---

### 17. No bufio.Reader for tcp_intermediate_noheader

**File:** `internal/transport/tcp_intermediate_noheader.go:62-69`
**Severity:** medium

`fill()` calls `t.conn.Read(tmp)` directly — one syscall per `fill()`. When multiple small frames arrive, each frame requires a separate syscall to read the 4-byte length prefix, then another for the body. Other transport implementations use buffered reads.

**Fix:** Wrap `t.conn` in a `bufio.Reader` to batch small reads.

---

### 18. t.buf unbounded growth in TCPIntermediateNoHeader

**File:** `internal/transport/tcp_intermediate_noheader.go:37-59`
**Severity:** medium

`t.buf` grows via `append` and is never shrunk. Under sustained traffic, a single large message permanently inflates the buffer. After consuming data, `t.buf = t.buf[needed:]` reslices but retains the full backing array.

**Fix:** Compact after draining:
```go
copy(t.buf, t.buf[needed:])
t.buf = t.buf[:len(t.buf)-needed]
```

---

## Low-Severity Issues

- `session/session.go:469` — `SetTransport` not thread-safe; `s.transport` assigned without synchronization
- `session/session.go:213` — `cancel` channel replaced on reconnect without closing old one; old goroutines never signaled
- `transport/tcp_obfuscated.go:162-171` — `getInnerConn` returns nil for unsupported types, causing nil pointer panic
- `crypto/secret.go:174` — `SecretDecrypt` uses `bytes.Equal` for msgKey check (not constant-time), unlike `mtproto.go` which uses `subtle.ConstantTimeCompare`
- `storage/storage.go:516-519` — `adapterWrapper.sess` accessed without synchronization; concurrent callers may race
- `transport/dialer.go:23` — No TCP keepalive configured on `net.Dialer`; long-lived connections can't detect dead peers
- `session/session.go:1348` — Unnecessary `SetReadDeadline(time.Time{})` syscall on every successful packet; deadline resets at loop top anyway
- `session/session.go:1306` — `time.NewTimer(100ms)` created per retry iteration in receiveLoop instead of reusing with `Reset()`
- `session/session.go:547` — `sendRPCDrop` calls `crypto.Pack` before checking `s.cancel`; wastes encryption when session is shutting down
- `session/auth.go:373` — Dead code: `return nil, ErrDHGenRetry` unreachable after the retry loop
- `transport/ws_stdlib.go:250` — Custom `readFull` instead of `io.ReadFull`; may miss final partial read with simultaneous EOF
