# Phase 1 — Messages Domain

Status: complete

## Objective

Bring the messages domain to its canonical operation surface: rename misnamed
operations, fill schema-supported capability gaps, and migrate every bound
method (except the folder bridge) from the transitional `Bound*` layer to
direct calls on domain group operations expressed through minimal binder
interfaces.

## What changed

### Commit `5936a3b` — canonical renames

- `ReadHistory` → `ReadChatHistory` (message-domain semantics).
- `GetMessagesViews` → `ViewMessages`.
- Generic wrappers regenerated to match.

### Commit `dd6d23b` — capability gap fill (schema-supported)

New `MessageMethods` operations, each covered by constructor-ID keyed mock
tests (`telegram/messages_misc_test.go`):

- `EditLiveLocation`, `StopLiveLocation` (live-location lifecycle)
- `SummarizeMessage` (server-side message summarization)
- `SendScreenshotNotification`
- `GetScheduledMessages`
- `TranslateText`, `TranslateMessageText`
- `SearchPosts` (+ `SearchPostsResult` cursor type)
- `StartBot` (deep-link start)
- `AddChecklistTasks`, `MarkChecklistTasksAsDone` (todo-list/checklist ops)
- `SendLivePhoto`
- New `PaymentMethods.ResolveStarGiftOffer` (accept/decline gift offers)

### Commit `3d1e384` — bound methods call domain operations directly

- All 139 bound `*Context` methods on Message, Chat, User, Story, Gift,
  CallbackQuery, InlineQuery, ChatJoinRequest, PreCheckoutQuery,
  ShippingQuery, and ActiveSession were migrated from `Bound*` bridge calls
  to direct group-operation calls; reply targets are expressed via the
  option structs (`&params.SendMessage{ReplyToMessageID: ...}`), per the
  agreed style. No merge-helper functions in the public API.
- The twelve binder interfaces were regenerated from actual call-site usage
  (`MessageBinder` 38 members, `ChatBinder` 28, `CallbackQueryBinder` 12,
  `StoryBinder` 10, `GiftBinder` 5, `UserBinder` 5, and single-member
  query/session interfaces) — interface segregation now reflects real
  dependencies, not estimates.
- 163 of 170 `Bound*` bridge implementations deleted along with their
  dedicated test file; coverage continues through the type-level tests and
  operation tests. Folder's seven bridge members remain until folder
  operations land (Phase 2).
- New `StoryMethods` operations `ReadStories`, `EditStoryPrivacy`,
  `SendStoryReaction` replace raw-RPC bound bodies.
- `Message.Summarize` now performs the real summarize operation (was a
  placeholder); `Message.View` now increments views via `ViewMessages`
  (previously misrouted to read-history; noted behavior fix).
- Stubs that were placeholders before the phase (`Click`, `Pay`,
  `ReplyPaidMedia`) remain explicit `errNotImplemented` placeholders.
- The types test mock was regenerated against the new interfaces with
  recording for the observed flows.

## Tests executed

| Gate | Result |
|---|---|
| `go build ./...` | pass |
| `go vet ./...` | pass |
| `golangci-lint run` (via `make all`) | pass |
| `go test ./...` (8 packages in telegram tree, full module) | pass |
| `TestNoPromotedMethodCollisions` | pass |
| `go mod tidy` | clean |

## Remaining gaps

- Schema-gated (constructor absent from TL layer 232; requires a schema
  update via `cmd/tlgen` before implementation): direct-messages chat topics
  family, `GetUserPersonalChatMessages`, AI text-fix/compose methods.
- `Folder` bound methods still bind to the seven `Bound*` bridge members;
  folder CRUD operations land in Phase 2 and the bridge is deleted then.
- `Click` and `Pay` message bound methods are explicit placeholders pending
  their interactive flows.
- Message send families (`SendContact`, `SendLocation`, `SendVenue`,
  `SendPoll`) still take a flattened `*params.SendMessage`; unifying with the
  per-family option structs is deferred to the params normalization slice of
  Phase 7.

## Commits

- `5936a3b` refactor(telegram)!: rename ReadHistory to ReadChatHistory and GetMessagesViews to ViewMessages
- `dd6d23b` feat(messages): live location, summarize, translate, scheduled, posts search, checklists, start-bot, gift offers
- `3d1e384` refactor(types,telegram)!: bound methods call domain operations directly

All on `rewrite`; branch verified before each commit; no tags; no downstream
updates.
