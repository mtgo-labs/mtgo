# Phase 4 — Bots, Business, Premium

Status: complete

## Objective

Complete the bot-owner surface (per-user menu button, default admin rights,
business-bot messaging permission, mini-app methods, bot discovery), the
business domain (profile settings and chat links), and normalize the boost
listings.

## What changed

### Commit `224de54` — bots

New `telegram/bots_extended.go` on `BotsMethods`:

- `SetBotUserMenuButton` / `GetBotUserMenuButton` — the menu button a specific
  user sees (the account-level chat menu button pair already existed).
- `SetBotDefaultAdminRights(forChannels, rights)` — one entry point over the
  group and broadcast default-rights variants.
- `CanMessageBot` / `AllowBotToMessage` — business-bot messaging permission.
- `InvokeWebViewCustomMethod(botID, method, paramsJSON)` — mini-app custom
  method; returns the raw JSON reply.
- `GetPopularAppBots(offset, limit)` — paginated mini-app bot search returning
  parsed users plus the next offset.
- `ReorderBotUsernames` / `ToggleBotUsername` — owned-bot username management,
  symmetric with the user-domain ops.
- `GetBotRecommendations` — bots recommended alongside a given bot.

Pre-existing coverage kept: commands (set/get/reset), bot info (name,
description, short description), account-level menu button, games, callback
answers, inline queries and guest answers.

### Commit `f1c089f` — business

`BusinessMethods` grows from one method to the full profile surface:

- Setters: `SetBusinessWorkHours` (nil clears), `SetBusinessLocation`
  (address and/or coordinates; both empty clears), `SetBusinessGreetingMessage`,
  `SetBusinessAwayMessage`, `SetBusinessIntro` (nil disables/resets).
- Chat links: `GetBusinessChatLinks`, `CreateBusinessChatLink`,
  `EditBusinessChatLink`, `DeleteBusinessChatLink`, `ResolveBusinessChatLink`.

Read side intentionally not duplicated: business state is returned with full
user info.

### Commit `9700b7a` — premium (breaking)

- `GetBoosts` renamed `GetMyBoosts` (it lists the logged-in user's own boosts);
  `Context` wrapper and tests migrated.
- New `GetChatBoosts(ctx, chatID, offset, limit, giftsOnly)` returning a
  paginated `ChatBoosts` page (count, boosts, next offset, users) and
  `GetUserBoosts(ctx, chatID, userID)` for one user's boosts on a chat.

Generic wrappers regenerated (164).

## Tests executed

| Gate | Result |
|---|---|
| `go build ./...` / `go vet ./...` | pass |
| `golangci-lint run` (via `make all`) | pass |
| `go test ./...` (full module, 0 failures) | pass |
| `TestNoPromotedMethodCollisions` | pass |
| `go mod tidy` (diff clean) | pass |

13 new constructor-ID-keyed tests (`bots_extended_test.go`,
`business_extended_test.go`, premium additions), including the CRUD path of
business chat links and both default-admin-rights variants.

## Remaining gaps

- Compiler-gated: `bots.getAdminedBots` returns a bare `Vector<User>` which the
  generator types as `TLObject` (same limitation as
  `chatlists.getLeaveChatlistSuggestions`); a typed `GetOwnedBots` needs
  generator support for bare vector results.
- Deferred (schema-supported but niche bot-owner revenue/admin surface):
  preview-media family (`bots.*PreviewMedia*`), star-ref program
  (`bots.updateStarRefProgram`), custom verification
  (`bots.setCustomVerification`), user emoji-status permission family, webhook
  JSON bridge (`bots.answerWebhookJSONQuery`, `bots.sendCustomRequest`).
  These stay available through the raw RPC surface; promote on demand.
- Premium feature listing (`premium.getPremiumFeatures`) is absent from the
  pinned TL layer.

## Commits

- `224de54` feat(bots): per-user menu button, default admin rights, business-bot messaging, mini-app surface
- `f1c089f` feat(business): business profile settings and chat links
- `9700b7a` refactor(premium)!: rename GetBoosts to GetMyBoosts, add chat and user boost listings

All on `rewrite`; branch verified before every commit; no tags; no downstream
updates.
