# Phase 6 — Payments (Stars, Gifts) and Wallet

Status: complete

## Objective

Complete the payments surface — gift collections, auctions, the Stars
economy, gift codes and giveaways, and the payment-form tail — and land the
TON wallet domain.

## What changed

### Commit `0a833ba` — gift collections, auctions, send checks

`telegram/gift_collections.go` on `PaymentMethods`:

- Collections: `CreateGiftCollection`, `UpdateGiftCollection` (partial edits:
  rename, remove/add gifts, reorder), `ReorderGiftCollections`,
  `DeleteGiftCollection`, `GetGiftCollections` (hash-gated, NotModified → nil).
- Auctions: `GetGiftAuctionState` (version-delta polling, returns the auction
  state / finished / NotModified class), `GetGiftAuctionAcquiredGifts`,
  `GetActiveGiftAuctions` (NotModified → nil).
- `CheckCanSendGift` — the server's refusal reason becomes a Go error;
  `SetGiftChatNotifications`.

### Commit `79e97fb` — stars completion and payment-form tail

`telegram/stars.go`:

- Options: `GetStarsTopupOptions`, `GetStarsGiveawayOptions`, and
  `GetPremiumGiftCodeOptions` — all bare `Vector<...>` results, unwrapped via
  a typed generic over `tg.GenericVector` (the vector constructor is
  registered in the TL registry, so these decode fine; earlier
  "compiler-gated" worries were unfounded).
- Subscriptions: `GetStarsSubscriptions` (typed page: items, next offset,
  missing balance), `ChangeStarsSubscription`,
  `FulfillStarsSubscription`, `CancelStarsSubscriptionByBot` (with restore).
- Transactions and revenue: `GetStarsTransactionsByID`,
  `GetStarsRevenueStats`, `WithdrawStarsRevenue` (cloud-password SRP computed
  when provided), `GetStarsRevenueAdsAccountURL`.
- Gift codes and giveaways: `CheckGiftCode`, `ApplyGiftCode`, `GetGiveawayInfo`,
  `LaunchPrepaidGiveaway`.
- Payment-form tail: `GetPaymentReceipt`, `ValidateRequestedInfo` (shipping
  options), `GetSavedPaymentInfo`/`ClearSavedPaymentInfo`,
  `CanPurchaseFromStore`, `AssignAppStoreTransaction` (bytes receipt) and
  `AssignPlayMarketTransaction` (JSON receipt).

### Commit `a21146c` — wallet (TON)

`telegram/wallet.go` gives `WalletMethods` (previously an empty group) its
first operations:

- `GetWalletState` (empty wallet → nil), `GetExistingWalletBalance`.
- `GetWalletUserAddresses` — resolve by user IDs or by addresses, optional
  cache bypass.
- Transactions: `GetWalletTransactions` (typed page with balance and next
  offset; direction filters; default limit 25), `GetWalletTransactionsByIDs`,
  `GetWalletTransactionsByMessageHash`.
- `SendWalletTransfer` — signed TON transfer, normal or gasless payload with
  an exactly-one-payload guard.
- `GetWalletNFTs` — typed page.

Generic wrappers regenerated (186).

## Tests executed

| Gate | Result |
|---|---|
| `go build ./...` / `go vet ./...` | pass |
| `golangci-lint run` (via `make all`) | pass |
| `go test ./...` (full module, 0 failures) | pass |
| `TestNoPromotedMethodCollisions` | pass |
| `go mod tidy` (diff clean) | pass |

24 new constructor-ID-keyed tests across the three files, including bare-
vector unwrapping, NotModified paths, partial-collection edits, the
withdrawal SRP-omission path, wallet empty-state, and the transfer payload
guard.

## Remaining gaps

- Deferred wallet internals on the raw RPC layer: secret-phrase
  export/backup management (security-sensitive), the Ton Connect session
  family, gasless info, proof challenge, card stickers.
- Star-ref (affiliate) suite (`payments.*StarRef*`, 5 methods) deferred with
  the Phase 4 bot-revenue surface; now easily promoted using the same
  GenericVector unwrap where applicable.
- `payments.getBankCardData` left unwrapped (niche bank-card preview).
- Follow-up enabled by this phase: `bots.getAdminedBots` and
  `chatlists.getLeaveChatlistSuggestions` (bare vectors previously thought
  compiler-gated) can now be wrapped the same way — queued for Phase 7.

## Commits

- `0a833ba` feat(payments): gift collections, auctions, and send-gift checks
- `79e97fb` feat(payments): stars completion — topup/giveaway options, subscriptions, revenue, gift codes, payment-form tail
- `a21146c` feat(wallet): TON wallet domain

All on `rewrite`; branch verified before every commit; no tags; no downstream
updates.
