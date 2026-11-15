# Phase 7 — Utilities, Types, Params Normalization, Bound-Set Completion

Status: complete

## Objective

Finish the implementation surface: canonical media type names, per-family
send option structs, the previously compiler-gated bare-vector operations,
the application-config lookup, and the remaining Message bound placeholders.

## What changed

### Commit `346c6a1` — bare-vector unlocks

- `BotsMethods.GetOwnedBots` — the logged-in user's created bots, parsed from
  the registered `GenericVector` bare-vector result.
- `FolderMethods.GetLeaveFolderSuggestions` — chats the server suggests
  leaving alongside a shareable folder, returned as marked chat IDs via
  `peerid.MarkFromPeer`.
- Clears the compiler-gated notes recorded in phases 2 and 4.

### Commit `fbc63ff` — canonical document name (breaking)

- `types.DocumentMedia` renamed `types.Document`; a deprecated
  `DocumentMedia = Document` alias keeps external consumers compiling.
  All internal references, tests, and the download example migrated (61
  sites).
- Naming decision: the remaining message-media wrappers keep their uniform
  `XxxMedia` suffix — `Photo`, `Story`, `Invoice`, and `Giveaway` entity
  names are already taken by richer entity types, so de-suffixing only some
  wrappers would split the family.

### Commit `2ecb91d` — per-family send options (breaking)

- `SendContact`, `SendLocation`, `SendVenue`, `SendPoll`, and `SendDice` now
  take their own variadic option structs (like the photo/video families)
  instead of a flattened `*params.SendMessage`; the `ToSendMsg` converters
  already in `params` drive the mapping.
- Unlocks previously dead capability, each covered by a new test: contact
  vCards; live locations (period, heading, proximity alert →
  `InputMediaGeoLive`); venue provider/Venue-ID/Venue-type with the
  Foursquare mapping; poll flags (quiz, public voters, close date/period,
  explanation); dice options beyond the emoticon.
- `MessageBinder` members and the bound `Reply*/Answer*` pairs follow the new
  signatures and pass the per-family struct directly.
- Documented limitation: `SendPoll.CorrectAnswers` is not mapped — the pinned
  TL generator encodes the wire field as `Vector<int>` instead of
  `Vector<bytes>`; quiz answer keys need a compiler fix first.

### Commit `89f0473` — Message bound-set completion

The remaining placeholder bound methods are implemented against domain
operations (`MessageBinder` grows five members):

- `EditLiveLocation` / `StopLiveLocation`.
- `Click(index)` — presses an inline callback button via
  `RequestCallbackAnswer`; flattened-index lookup over the parsed reply
  markup, non-callback buttons rejected with clear errors.
- `ReplyInvoice` / `AnswerInvoice` (take an `InputMediaInvoice`) and
  `ReplyPaidMedia` / `AnswerPaidMedia` (take an `InputMediaPaidMedia`), both
  with reply presets.
- `AcceptGiftPurchaseOffer` / `RejectGiftPurchaseOffer` → `ResolveStarGiftOffer`.
- `Pay` remains an explicit placeholder (interactive checkout flow); the
  stale not-implemented comments on `Summarize` corrected.

### Commit `c6a1825` — application configuration

- `UtilityMethods.GetAppConfig(hash)` — server-side app configuration,
  NotModified → nil.

## Tests executed

| Gate | Result |
|---|---|
| `go build ./...` / `go vet ./...` | pass |
| `golangci-lint run` (via `make all`) | pass |
| `go test ./...` (full module, 0 failures) | pass |
| `TestNoPromotedMethodCollisions` | pass |
| `go mod tidy` (diff clean) | pass |

New tests: 4 capability tests for the send families, 6 bound-method tests
(live location, click navigation including rejections, invoice/paid-media
reply presets, gift-offer accept/reject polarity), 3 for the bare-vector
unlocks and app config. Generic wrappers regenerated (186).

Bench gate not re-run: the rename is type-level (pointer identity), and the
send-family rewiring touches cold paths; Phase 0's comparison remains the
structural reference.

## Remaining gaps

- `SendPoll.CorrectAnswers` (generator `Vector<int>` mismatch — compiler fix
  required).
- `Message.Pay` (interactive payment-form checkout).
- `messages.getAttachedStickers` still unwrapped (needs a media-message input
  type; tracked in Phase 5).
- Wallet internals, Ton Connect, star-ref suite, story albums (Phase 5/6
  deferrals unchanged).

## Commits

- `346c6a1` feat(bots,folders): owned-bots listing and leave-folder suggestions
- `fbc63ff` refactor(types)!: rename DocumentMedia to Document
- `2ecb91d` refactor(messages)!: per-family option structs for contact, location, venue, poll, and dice sends
- `89f0473` feat(types): complete the Message bound set
- `c6a1825` feat(utilities): application configuration lookup

All on `rewrite`; branch verified before every commit; no tags; no downstream
updates.
