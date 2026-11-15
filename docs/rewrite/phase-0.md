# Phase 0 — Structural Scaffold

Status: complete

## Objective

Introduce the domain method-group architecture, minimal per-type binder
interfaces, and context-aware bound methods with zero behavior change: no RPC
behavior, error-handling, or return-value differences from `main`.

## What changed

### Commit `3d96c17` — compose client from domain method groups

- `telegram/groups.go` (new): 19 domain method groups, each holding a
  back-reference to `*Client`; `Client` embeds all of them so group methods
  are promoted onto the client (`client.SendMessage` ==
  `client.MessageMethods.SendMessage`).
- Receiver retarget of every client-facing operation to its domain group
  (messages, chats, users, contacts, account, auth, password, bots, business,
  premium, stories, payments, stickers, invite links, utilities, advanced,
  phone, wallet placeholder, secret chats). Method bodies are byte-identical;
  only receiver declarations changed (59 files).
- 170 helper-argument sites rewritten from bare `c` to `c.Client` with
  compiler-verified column precision (fixpoint over `go build` diagnostics).
- `telegram/collisions_test.go` (new): permanent AST-based gate failing on
  (a) a method name defined on two groups, (b) a group method shadowed by a
  direct `Client` method, (c) a group not embedded in `Client`.
- Test fixtures: bare `&Client{...}` constructions wrapped with
  `initClient(...)` so embedded groups are non-nil.

### Commit `1f42458` — minimal per-type binders, context-aware bound methods

- `telegram/types/binder.go` rewritten: twelve per-type interfaces sized to
  exactly the operations each type calls (`MessageBinder` 47 members,
  `ChatBinder` 28, `UserBinder` 5, `StoryBinder` 11, `GiftBinder` 5,
  `FolderBinder` 7, `CallbackQueryBinder` 10, `InlineQueryBinder` 1,
  `ChatJoinRequestBinder` 2, `PreCheckoutQueryBinder` 1,
  `ShippingQueryBinder` 1, `SessionBinder` 1). The monolithic `Binder`
  interface is gone. Old same-named interfaces in `gift.go`, `auth.go`,
  `dialog.go` were replaced by the minimal definitions.
- 146 bound methods across 12 types converted to the two-form grammar:
  convenience `Name(...)` delegating to `NameContext(ctx, ...)`. The context
  form is the single implementation and carries caller cancellation.
- All 170 `Bound*` implementations gained a leading `context.Context`
  parameter; they remain the interface members (interface segregation applied
  at the type boundary) until each domain phase replaces them with normalized
  high-level operations, at which point they are deleted.
- Option structs referenced across the package boundary moved to
  `telegram/params` with compatibility aliases in `telegram`:
  `EditInlineOpts`, `AnswerInlineQueryOption`, `SendInlineBotResultOption`,
  `SendDiceOption`, `GetPaymentFormOption`, `SendPaymentFormOption`.
- `telegram/binder_assert.go` (new): compile-time proof that `*Client`
  satisfies all twelve binder interfaces via promoted group methods.

### Commit `35523d6` — generator follows the groups

- `cmd/genhelpers` now discovers operations on group receivers (in addition
  to direct `Client` methods) and excludes transitional `Bound*` members from
  the generated surface. `telegram/generic/gen_helpers.go` regenerated: 140
  wrappers, byte-identical to the pre-change baseline.

### Hygiene commit (this report)

- Neutralized pre-existing external-project references in comments of files
  touched by the rewrite (client.go, messages.go, types/chat.go,
  types/message.go).

## Tests executed

| Gate | Result |
|---|---|
| `go build ./...` | pass |
| `go vet ./...` | pass |
| `golangci-lint run` (via `make all`) | pass |
| `go test ./...` (full module, incl. 917+ telegram tests) | pass |
| `TestNoPromotedMethodCollisions` | pass |
| `go mod tidy` (diff clean) | pass |
| `make bench-compare REF=main` (benchstat, B/op and allocs/op) | pass — geomean **+0.00%** on both metrics; every benchmark byte- and alloc-identical (all samples equal), confirming the restructure is allocation-neutral |

## Behavior preservation

- No method body logic changed in `3d96c17` (receiver-only diff).
- `1f42458` adds a leading ctx parameter to transitional `Bound*`
  implementations; their internal control flow is unchanged (ctx was already
  `context.Background()` internally, now supplied by the same convenience
  path). All pre-existing tests pass unmodified except for mechanical
  call-site updates (ctx threading) reviewed in the diff.
- Public API note: `SetBinder` on each type now takes the minimal interface
  instead of the monolith; `*telegram.Client` callers are unaffected
  (compile-time asserted).

## Remaining gaps (deliberate, tracked for later phases)

- The 170 `Bound*` methods remain as binder interface members; they are
  removed domain-by-domain in Phases 1–6 as normalized operations replace
  them (constraint: no deletion before consumers migrate and tests pass).
- `Folder` bound methods still bind to raw-RPC-based `Bound*` bodies; folder
  CRUD operations land in Phase 2.
- Message bound methods for live-location editing, extended media, and gift
  purchase offers still route through `Bound*` raw implementations; their
  high-level operations land in Phase 1/6.
- Handler-context (`*Context`) methods still call the old call shape; they
  work unchanged via promotion and are normalized per-domain in later phases.
- `docs/API_REFERENCE.md`, README, and examples still document the pre-change
  surface; refreshed in Phase 8.

## Commits

- `3d96c17` refactor(telegram): compose client from domain method groups
- `1f42458` refactor(types,telegram): minimal per-type binders with context-aware bound methods
- `35523d6` chore(genhelpers): scan domain method-group receivers
- (this commit) docs(rewrite): phase 0 report

All commits on `rewrite` (branch verified before every commit). No tags
created, no downstream repositories touched.
