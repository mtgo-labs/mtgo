# Phase 8 — Generated Coverage and Documentation

Status: complete (final phase)

## Objective

Replace hand-maintained inventories with a generator, refresh the public
documentation to the reconstructed surface, and close the rewrite.

## What changed

### Commit `ddd4911` — coverage generator and docs

- **`cmd/apicoverage`** (new): AST-scans the `telegram` package for domain
  group operations (with signatures and doc sentences), `telegram/types` for
  binder-backed bound method sets, and `telegram/params` for option structs;
  parses `compiler/source/api.tl` for the TL function inventory and records
  which functions the high-level layer actually invokes. Output is derived
  entirely from source and schema — nothing hand-maintained.
- **`docs/api-coverage.md`** (new, generated): 20 groups · 422 client
  operations · 13 bound types · 50 option structs · 316 of 867 TL functions
  (36%) invoked by high-level operations, with the deliberately unwrapped
  surface listed in full.
- **`docs/API_REFERENCE.md`** rewritten (3,638 → 640 lines): curated
  architecture and quick-start sections (verified against current names —
  `SendPhoneNumberCode`, `params.SendMessage`, `Connect(timeout)`) plus
  generated per-domain signature tables and bound/params inventories.
- **`make apicoverage`** target added; generator registered in AGENTS.md's
  never-hand-edit list.
- **README**: feature bullet now reflects the twenty-group surface and the
  generated inventory; project structure and code-generation sections updated.
- **AGENTS.md**: the stale "protocol-only, application layer out of scope"
  note replaced with the actual architecture (domain groups + collision test,
  group-first rule for new operations, constructor-ID mock tests,
  Name/NameContext bound pairs, generated inventory); layer note 230 → 233.

No example needed migration — the earlier phases kept `examples/` compiling,
verified again by the full build.

## Tests executed

| Gate | Result |
|---|---|
| `go build ./...` / `go vet ./...` | pass |
| `golangci-lint run` (via `make all`) | pass |
| `go test ./...` (full module, 0 failures) | pass |
| `TestNoPromotedMethodCollisions` | pass |
| `go mod tidy` (diff clean) | pass |
| `go run ./cmd/apicoverage` (deterministic emit) | pass |

## Reconstruction close-out

Across phases 0–8 the `rewrite` branch replaced the flat god-object
(543 `*Client` methods, a 170-member `Bound*` plumbing layer, triplicated
operations) with:

- 20 domain method groups, collision-gated, promoting onto `Client`;
- minimal per-type binder interfaces (13 types) with context-aware
  Name/NameContext bound pairs and zero `Bound*` surface remaining;
- the modern Telegram surface implemented: folders and shareable-folder
  invites, chat gaps, profile state, phone-code flows, cloud-password reset,
  bot-owner and business surfaces, boost listings, the sticker lifecycle,
  story feed/stealth/viewers, the Stars economy, gift collections and
  auctions, the TON wallet, message widgets (TL layer 233), and the
  per-family send option structs;
- documentation that is generated, not asserted.

Deliberately unwrapped (available via `client.Raw()`, listed in
api-coverage.md): 551 TL functions — channel admin plumbing, MTProto
internals, seldom-used read models, and the phase-gated items recorded in
earlier reports (`SendPoll.CorrectAnswers` generator bug, `Message.Pay`,
`messages.getAttachedStickers`, wallet internals and Ton Connect, star-ref
suite, story albums).

### Addendum — final verification fix

The closing `-count=1` verification caught two regressions the widget commit
introduced and a cached test run had masked: `Update.WidgetQuery` was not
carried into `Context` (now wired in `populateContext`, asserted by the
context-coverage gate), and the widget recording fields the mock binder's new
methods referenced were never applied (pattern matched pre-format alignment).
Fixed and re-verified with forced-uncached full-module runs — the lesson,
recorded for future phases: never trust a cached gate after test-file edits.

## Commits

- `ddd4911` feat(docs): generated API inventory and reference; docs refresh

All on `rewrite`; branch verified before every commit; no tags; no downstream
updates — per the standing constraints, which remain satisfied at close.
