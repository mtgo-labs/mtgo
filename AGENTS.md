# AGENTS.md

## Agent Tools — Use First

This repository uses CodeGraph locally. Run `codegraph init -i` after cloning the project. Do not commit `.codegraph/` (it's in `.gitignore`).

**Always prefer CodeGraph MCP over grep/Read for code exploration.** It is the pre-built semantic index — re-deriving its answers with grep + Read wastes tokens and time.

**CodeGraph tool selection:**

| Tool | Use for |
|------|---------|
| `codegraph_context` | First call for any task/feature/bug — composes search + node + callers + callees in one shot |
| `codegraph_trace` | "How does X reach Y?" — full call path with inline source at each hop |
| `codegraph_explore` | Survey several related symbols' source in one budget-capped call |
| `codegraph_search` | Find a symbol by name |
| `codegraph_callers` / `codegraph_callees` | Walk call flow one hop |
| `codegraph_impact` | Check what's affected before editing |
| `codegraph_node` | Single symbol source/signature |

Source returned by CodeGraph is verbatim live file content — treat it as already Read. Only use Read/Grep to confirm a detail CodeGraph didn't cover.

**AgentMemory MCP** is available for persisting decisions, patterns, and session context across conversations.

- **At session start:** `memory_recall` to check for prior decisions, conventions, or long-running task context before re-deriving them.
- **After decisions/fixes:** `memory_save` to store durable facts, file paths, commands, and rationale.
- **Lessons learned:** `memory_lesson_save` for what worked, what to avoid, and when to use a specific approach.
- Do not store secrets or noisy transcripts.

## Build & Verify

```bash
go build ./...          # compile everything
go vet ./...            # static analysis
go test ./...           # run all tests
golangci-lint run       # lint (config in .golangci.yml)
```

Run a single package or test:

```bash
go test ./internal/session/...
go test -run TestSessionConnect ./internal/session/
go test -bench=BenchmarkSessionRPCResult -benchmem ./internal/session/
```

## Monorepo Layout

This repo is part of the `mtgo-labs` workspace.

```
mtgo-labs/
├── mtgo/               ← this repo (module: github.com/mtgo-labs/mtgo)
├── session-converter/  ← session string converter library
├── session-generator/  ← tgconv CLI for session conversion + generation
├── storage/            ← storage interfaces
└── storage/sqlite/     ← SQLite adapter
```

Key packages:

| Path | Role | Editable? |
|------|------|-----------|
| `tg/` | Generated TL types (layer 228) | **No** — use codegen |
| `tgerr/` | Generated error types | **No** — use codegen |
| `compiler/` | TL compiler and templates | Yes |
| `internal/session/` | MTProto session, state machine, auth, pending RPCs | Yes |
| `internal/crypto/` | MTProto crypto (AES-IGE, RSA, DH, SRP) | Yes |
| `internal/transport/` | TCP transports (abridged, intermediate, full, obfuscated, WS) | Yes |
| `internal/storage/` | Storage adapter wrapper | Yes |
| `telegram/` | High-level client API, handlers, filters, middleware, plugins | Yes |
| `mtproxy/` | MTProxy obfuscated2/fake-TLS transport | Yes |
| `cmd/tlgen/` | TL schema code generator | Yes |
| `cmd/errgen/` | Error type generator | Yes |

Session string conversion (Telethon, Pyrogram, GramJS, mtcute, MTKruto, gogram,
gotgproto) is provided by the external
[session-converter](https://github.com/mtgo-labs/session-converter) package.
The CLI tool for session conversion and generation lives in
[session-generator](https://github.com/mtgo-labs/session-generator).

Never edit `*_gen.go` files directly. If a TL type is missing or wrong, fix the schema/compiler and regenerate.

## Conventions

- **Go 1.26+** (see `go.mod`)
- **No CGO** — SQLite via `modernc.org/sqlite`
- **Commit style:** Conventional Commits with scope: `feat(telegram):`, `fix(session):`, `chore(tg):`, etc.
- **Branch prefixes:** `feat/`, `fix/`, `refactor/`, `docs/`, `test/`, `chore/`
- **errcheck disabled** in golangci-lint — intentional project choice
- **Goroutine leak detection:** `internal/session/` uses `goleak.VerifyTestMain` in `goleak_test.go`

## Architecture Notes

- **Engineering priority order**: (1) security and correctness, (2) maximum measured
  performance, (3) lowest real-world latency, (4) minimal raw MTProto overhead.
  Every hot-path abstraction must justify copies, allocations, queues, locks,
  goroutines, and interface boundaries with correctness or benchmark evidence.

- `telegram.Client` is the main entry point. It owns `session.Session` instances (one per DC).
- `internal/session.Session` manages the MTProto connection: encrypted message packing/unpacking, RPC lifecycle, keep-alive pings, salt management, and state machine (Idle → Connecting → Active → Draining → Closed).
- `tg.TLObject` is the base interface for all generated TL types. Hand-written files in `tg/`: `tl.go`, `reader.go`, `primitives.go`, `fields.go`, `gzip.go`, `msg_container.go`, `invoker.go`, `layer.go`.
- Handler dispatch: `telegram.Dispatcher` → handler groups sorted by priority → `handler.Check(update)` → `handler.Handle(ctx)`.
- Middleware chains: invoker-level (wraps RPC calls) and handler-level (update dispatch).
- `internal/session.PendingManager` tracks outstanding RPC calls with `CallHandle` (future-like: `Done() <-chan struct{}`, `Result()`).
- **RPC Retry on Reconnect**: On transport disconnect, pending RPCs carry explicit delivery state. Read-only/replay-safe methods wait on the event-driven reconnect signal and retry up to `MaxRPCReconnectRetries`; delivery-uncertain mutations return `RPCDeliveryError` instead of risking duplicate execution. `stateCheckLoop` proactively uses `msgs_state_req` and handles `msgs_state_info`/`msg_resend_req` for overdue decoded and raw calls.
- **DC Endpoint Health**: `DCOptionPool` tracks per-endpoint health (Ok/Error/Untested) with timestamps. Scoring: Ok (most recent) > Untested > Error with cool-down. Ported from TDLib `DcOptionsSet::find_connection`.
- **Connection Pool**: `ConnectionPool` caches warm connections for 10s TTL to avoid redundant TCP handshakes. Entries consumed on first use. Ported from TDLib `ConnectionCreator::ready_connections`.
- **Multi-Session Routing**: `SessionRouter` routes queries to session slots by method name: `upload.*` → SlotUpload, `upload.getFile/getWebFile` → SlotDownload, else → SlotMain. Idle upload/download slots auto-close after 5min. Ported from TDLib `NetQueryDispatcher`.
- **Multi-DC Auth**: `DcAuthManager` tracks non-main DC auth state and supports auth export/import state transitions for transparent DC session authorization. Ported from TDLib `DcAuthManager`.
- **Container ACK Tracking**: `ContainerTracker` maps container message IDs to child message IDs and tracks child/container ACK cleanup. Ported from TDLib `Session` query container tracking.
- **Per-DC Backoff**: `PerDCBackoff` keeps reconnect delays independent per DC, preventing one failing DC from delaying unrelated DCs. Ported from TDLib `ConnectionCreator::ClientInfo::Backoff`.
- **Flood Wait Handling**: `FloodWaitQueue` records delayed FLOOD_WAIT queries while `Session.Invoke` parses `FLOOD_WAIT_X`, waits, and retries without surfacing the first flood error to callers. Ported from TDLib `NetQueryDelayer`.
- **PFS (Perfect Forward Secrecy)**: `TempKeyManager` generates a separate temp auth key for every main, upload, download, and CDN session, binds each key to its own session ID via `auth.bindTempAuthKey`, rotates at 75% lifetime, and fails closed when enabled. Ported from TDLib `Session::auth_loop`.
- **Outbound Container Packing**: `OutboundBatcher` coalesces concurrent RPCs into MTProto `msg_container#73f1f8dc` via adaptive flushing (immediate on idle/lone, batch when N>1 queued). Per-priority FIFOs (High/Low). Opt-in via `Config.OutboundBatchEnabled`. Ported from TDLib `net/Session.h` outbound container packing.
- **Cryptographic Trust**: `RSAKeySet` wraps bundled canonical Telegram RSA keys as the immutable trust root; `PublicRsaKeyWatchdog` fetches and verifies rotated keys against the trust set (fail-closed). `ErrKeyVerificationFailed` typed error for MITM detection. Opt-in via `Config.RSAKeyRotationInterval`. Ported from TDLib `net/PublicRsaKeyWatchdog.h`.
- **Overload Control**: `OverloadController` gates RPC admission by priority — low-priority fast-fails at capacity (`ErrOverload`), high-priority gets bounded deferred admission. `LoadSnapshot` aggregates queue depths, in-flight counts, throttle level for observability. Opt-in via `Config.MaxInFlightRPCs`. Ported from TDLib `net/NetQueryDispatcher.h`.
- **MTProto Client Focus**: mtgo is an MTProto client library, not a full Telegram client. Features MUST improve the protocol layer (connection, auth, encryption, RPC lifecycle). Application-layer features (UI, stickers, stories, payments) are out of scope. See Constitution Principle VII.

## Testing Gotchas

- `internal/session/` has goroutine leak detection via `goleak` — any new goroutine that doesn't clean up will fail tests.
- Tests use `mockTransport` (in `session_test.go`) with `sendCh`/`recvCh` channels for simulating transport behavior.
- `startTestWorkers` bypasses the full lifecycle to test Send/Read/ACK loops in isolation.
- `forceSetState` on the state machine is for test use only — do not use in production code.

<!-- SPECKIT START -->
## Active Feature Plan

**Feature**: `003-production-hardening`
**Plan**: `specs/003-production-hardening/plan.md`
**Spec**: `specs/003-production-hardening/spec.md`
**Tasks**: `specs/003-production-hardening/tasks.md`
**Status**: Implementation complete — all 38 tasks done, all tests pass

Artifacts:
- `specs/003-production-hardening/plan.md` — Implementation plan with research, constitution check
- `specs/003-production-hardening/research.md` — 5 design decisions grounded in TDLib + codebase
- `specs/003-production-hardening/data-model.md` — 7 entities with state transitions
- `specs/003-production-hardening/contracts/` — Interface contracts (outbound-batcher, crypto-trust, overload-control)
- `specs/003-production-hardening/quickstart.md` — 10 validation scenarios
- `specs/003-production-hardening/tasks.md` — 38 tasks across 4 user stories + setup + foundational + polish

Previous features (completed):
- `specs/002-mtproto-stability/` — MTProto protocol stability (31 tasks, all complete)
- `specs/001-connection-overhaul/` — Connection architecture overhaul (37 tasks, all complete)
<!-- SPECKIT END -->
