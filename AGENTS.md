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
make all        # fmt + vet + lint + test — the pre-commit gate
make fmt        # gofmt -w .
make lint       # golangci-lint run (config in .golangci.yml)
make test       # go test ./...
make test-race  # go test -race ./...
```

Run a single package or test:

```bash
go test ./internal/session/...
go test -run TestSessionConnect ./internal/session/
go test -bench=BenchmarkSessionRPCResult -benchmem ./internal/session/
```

CI (`.github/workflows/ci.yml`) runs only: `go mod tidy` diff check, build, vet, test. It does **not** run lint or race — run those locally via `make`.

## Code Generation — Never Edit Generated Files

Three generators; never hand-edit their outputs (`*_gen.go`, `telegram/generic/gen_helpers.go`):

```bash
go run ./cmd/tlgen        # tg/ TL types from schema (layer pinned in tg/layer_gen.go)
go run ./cmd/errgen       # tgerr/ error types
go run ./cmd/genhelpers   # telegram/generic/gen_helpers.go (int|int64|string peer wrappers)
```

If a TL type is missing or wrong, fix the schema/compiler (`compiler/`, `cmd/`) and regenerate.

## Package Map

| Path | Role | Editable? |
|------|------|-----------|
| `tg/` | Generated TL types (`Layer = 230` in `tg/layer_gen.go`) | **No** — `cmd/tlgen` |
| `tgerr/` | Generated error types | **No** — `cmd/errgen` |
| `telegram/generic/gen_helpers.go` | Generated peer-ID wrapper methods | **No** — `cmd/genhelpers` |
| `compiler/` | TL compiler and templates | Yes |
| `internal/session/` | MTProto session: encryption, RPC lifecycle, state machine | Yes |
| `internal/crypto/` | AES-IGE, RSA, DH, SRP; server key trust (`server_keys.go`) | Yes |
| `internal/transport/` | Abridged, intermediate, full, obfuscated, WebSocket, HTTP | Yes |
| `internal/peerid/` | **Single source of truth for marked chat IDs** (+id users, `-id` basic groups, `-100id` channels) | Yes |
| `internal/storage/` | Storage adapter wrapper | Yes |
| `telegram/` | High-level client API; subpackages: `types`, `parser`, `peers`, `fileid`, `params`, `generic`, `otel` | Yes |
| `mtproxy/` | MTProxy obfuscated2/fake-TLS transport | Yes |
| `cmd/` | Code generators (`tlgen`, `errgen`, `genhelpers`) | Yes |

**Peer-ID rule:** all raw↔marked ID conversions MUST go through `internal/peerid`. Re-deriving the arithmetic at call sites has historically produced five divergent copies.

**Sibling repos are separate modules.** `mtgo-labs/session-converter` and `mtgo-labs/storage` live as sibling checkouts but there is no `go.work` — this module pins them by version in `go.mod`. Editing a sibling does not affect builds here until it's tagged and bumped. Session string conversion (Telethon, Pyrogram, GramJS, etc.) is provided by the external session-converter package.

## Conventions

- **Go 1.26+** (see `go.mod`); **no CGO** — SQLite via `modernc.org/sqlite`
- **Commit style:** Conventional Commits with scope: `feat(telegram):`, `fix(session):`, `chore(tg):`, etc.
- **Branch prefixes:** `feat/`, `fix/`, `refactor/`, `docs/`, `test/`, `chore/`
- **errcheck disabled** in golangci-lint — intentional project choice
- **MTProto client focus:** mtgo is a protocol-layer library. Features MUST improve connection, auth, encryption, or RPC lifecycle. Application-layer features (UI, stickers, stories, payments) are out of scope. See `ARCHITECTURE.md`.
- **Engineering priority order** (details in `ARCHITECTURE.md`): (1) security and correctness, (2) maximum measured performance, (3) lowest real-world latency, (4) minimal raw MTProto overhead. Every hot-path abstraction must justify copies, allocations, queues, locks, and goroutines with correctness or benchmark evidence.

## CI & Performance Gates

- **Benchmark gate:** every PR to `main` is compared against the committed `bench_baseline.txt` (`benchmark.yml`); ≥10% ns/op regressions are flagged in a PR comment. Baseline was captured on a Ryzen 5 7600X — judge regressions by **B/op and allocs/op**, not ns/op.
- **Before pushing perf-relevant changes:** `make bench-compare REF=main` (auto-installs benchstat; uses a sibling worktree).
- **Regenerate the baseline** after intentional perf changes: `make bench-baseline` on stable hardware, then commit `bench_baseline.txt`.
- **`go mod tidy` must be clean** — CI fails if `go.mod`/`go.sum` change after tidy.
- **Releases = tags.** Pushing a `v*` tag auto-bumps mtgo in 11 downstream repos and opens PRs there (`bump-deps.yml`). Tag only after CI is green.

## Architecture Notes

Flow: `telegram.Client` (entry point, owns one `session.Session` per DC) → `internal/session` (MTProto packing, RPC lifecycle, salt/ping management) → `internal/transport`. Updates flow back via `telegram.Dispatcher` → handler groups by priority → `handler.Check(update)` → `handler.Handle(ctx)`. Middleware exists at two levels: invoker-level (wraps RPC calls) and handler-level (update dispatch). State machine: Idle → Connecting → Active → Draining → Closed (`internal/session/state.go`).

| Mechanism | Where | Notes |
|-----------|-------|-------|
| Pending RPCs | `internal/session/pending.go` | `PendingManager` tracks `CallHandle` (future-like: `Done()`, `Result()`) |
| Reconnect RPC retry | `internal/session` + `telegram/config.go` | Replay-safe methods retry on reconnect up to `Config.MaxRPCReconnectRetries`; delivery-uncertain mutations return `DeliveryError` (`internal/session/errors.go`) instead of risking duplicate execution |
| State reconciliation | `internal/session/state_check.go` | `stateCheckLoop` sends `msgs_state_req`, handles `msgs_state_info`/`msg_resend_req` |
| Flood wait | `telegram/retry.go`, `tgerr.AsFloodWait` | Auto-wait and retry for `FLOOD_WAIT_X` |
| DC endpoint health | `internal/session/dc_options.go` | `DCOptionPool`: Ok > Untested > Error with cool-down (TDLib `DcOptionsSet`) |
| Warm connection cache | `internal/session/conn_pool.go` | `ConnectionPool`, short TTL, consumed on first use |
| Multi-DC auth | `internal/session/dc_auth.go` | `DcAuthManager`: auth export/import for non-main DCs |
| Container ACK tracking | `internal/session/container_tracker.go` | `ContainerTracker`: container ↔ child message-ID ACK cleanup |
| Per-DC backoff | `telegram/reconnect.go` | `PerDCBackoff`: reconnect delays independent per DC |
| PFS temp keys | `internal/session/pfs.go` | `TempKeyManager`: per-session temp auth keys, `auth.bindTempAuthKey`, rotation; fails closed when enabled |
| Outbound batching | `internal/session/outbound_batcher.go` | `OutboundBatcher` coalesces RPCs into `msg_container`; opt-in `Config.OutboundBatchEnabled` |
| RSA key trust | `internal/crypto/server_keys.go` | `RSAKeySet` (immutable trust root) + `PublicRsaKeyWatchdog` (fail-closed rotation via `Config.RSAKeyRotationInterval`); `ErrKeyVerificationFailed` = possible MITM |
| Overload control | `telegram/overload.go` | `OverloadController`: low-priority fast-fails `ErrOverload` at capacity; opt-in `Config.MaxInFlightRPCs`; `LoadSnapshot` in `telegram/introspection.go` |

## Testing Gotchas

- `internal/session/` has goroutine leak detection via `goleak.VerifyTestMain` (`goleak_test.go`) — any new goroutine that doesn't clean up will fail tests.
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

## Agent skills

### Issue tracker

Issues are tracked in GitHub Issues (github.com/mtgo-labs/mtgo) via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five-label vocabulary: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Read `docs/agents/domain.md` before domain work. `GLOSSARY.md` (repo root) and `docs/adr/` are created lazily by the domain-modeling skill — if absent, proceed silently.
