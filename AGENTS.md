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

This repo is part of the `mtgo-labs` workspace. `go.mod` has `replace` directives pointing to sibling `../storage` and `../storage/sqlite` — both must exist locally for `go build` and `go test` to work.

```
mtgo-labs/
├── mtgo/           ← this repo (module: github.com/mtgo-labs/mtgo)
├── storage/        ← storage interfaces (required by replace)
└── storage/sqlite/ ← SQLite adapter (required by replace)
```

Key packages:

| Path | Role | Editable? |
|------|------|-----------|
| `tg/` | Generated TL types (layer 225) | **No** — use codegen |
| `tgerr/` | Generated error types | **No** — use codegen |
| `compiler/` | TL compiler and templates | Yes |
| `internal/session/` | MTProto session, state machine, auth, pending RPCs | Yes |
| `internal/crypto/` | MTProto crypto (AES-IGE, RSA, DH, SRP) | Yes |
| `internal/transport/` | TCP transports (abridged, intermediate, full, obfuscated, WS) | Yes |
| `internal/storage/` | Storage adapter wrapper | Yes |
| `telegram/` | High-level client API, handlers, filters, middleware, plugins | Yes |
| `session/` | Session string import/export (Telethon, Pyrogram, GramJS, mtcute) | Yes |
| `mtproxy/` | MTProxy obfuscated2/fake-TLS transport | Yes |
| `cmd/tlgen/` | TL schema code generator | Yes |
| `cmd/errgen/` | Error type generator | Yes |

## Code Generation

```bash
go run cmd/tlgen/main.go    # regenerates tg/*_gen.go files
go run cmd/errgen/main.go   # regenerates tgerr/errors_gen.go
```

Never edit `*_gen.go` files directly. If a TL type is missing or wrong, fix the schema/compiler and regenerate.

## Conventions

- **Go 1.26+** (see `go.mod`)
- **No CGO** — SQLite via `modernc.org/sqlite`
- **Commit style:** Conventional Commits with scope: `feat(telegram):`, `fix(session):`, `chore(tg):`, etc.
- **Branch prefixes:** `feat/`, `fix/`, `refactor/`, `docs/`, `test/`, `chore/`
- **errcheck disabled** in golangci-lint — intentional project choice
- **Goroutine leak detection:** `internal/session/` uses `goleak.VerifyTestMain` in `goleak_test.go`

## Architecture Notes

- `telegram.Client` is the main entry point. It owns `session.Session` instances (one per DC).
- `internal/session.Session` manages the MTProto connection: encrypted message packing/unpacking, RPC lifecycle, keep-alive pings, salt management, and state machine (Idle → Connecting → Active → Draining → Closed).
- `tg.TLObject` is the base interface for all generated TL types. Hand-written files in `tg/`: `tl.go`, `reader.go`, `primitives.go`, `fields.go`, `gzip.go`, `msg_container.go`, `invoker.go`, `layer.go`.
- Handler dispatch: `telegram.Dispatcher` → handler groups sorted by priority → `handler.Check(update)` → `handler.Handle(ctx)`.
- Middleware chains: invoker-level (wraps RPC calls) and handler-level (wraps update dispatch).
- `internal/session.PendingManager` tracks outstanding RPC calls with `CallHandle` (future-like: `Done() <-chan struct{}`, `Result()`).

## Testing Gotchas

- `internal/session/` has goroutine leak detection via `goleak` — any new goroutine that doesn't clean up will fail tests.
- Tests use `mockTransport` (in `session_test.go`) with `sendCh`/`recvCh` channels for simulating transport behavior.
- `startTestWorkers` bypasses the full lifecycle to test Send/Read/ACK loops in isolation.
- `forceSetState` on the state machine is for test use only — do not use in production code.
