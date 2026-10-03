# Peer Resolution in gotd/td and gogram — Survey

Sources: shallow clones of github.com/gotd/td and github.com/amarnathcjd/gogram (2026-10-03). Paths relative to each repo root.

## gotd/td

- **Entry points.** `Manager.Resolve(string)` dispatches deeplink / phone / username (`telegram/peers/resolve.go:37`). Username → `contacts.resolveUsername` behind a `singleflight.Group` dedup (`resolve.go`, `ResolveDomain`). Phone → persisted phone index, else `contacts.getContacts` refresh (`ResolvePhone`, `resolve.go`; hash math in `apply.go:200+`). By ID: `ResolveUserID`/`ResolveChannelID` read the stored access hash then call `users.getUsers`/`channels.getChannels` (`telegram/peers/id.go:24,55`); `ResolveTDLibID` accepts marked TDLib IDs (`id.go:10`); `ResolvePeer(tg.PeerClass)` converts output peers (`peers.go`).
- **Cache layers.** Two-tier: `Storage` (Key{prefix,id}→`Value{AccessHash}` + phone index + contacts hash; `telegram/peers/storage.go:10-35`) and a richer `Cache` for whole `tg.User/Chat/Channel` (+Fulls) (`storage.go:44-60`). Defaults are in-memory `InmemoryStorage` + `NoopCache` (`telegram/peers/options.go:13-22`); persistence is delegated to external backends. Manager is opt-in and "completely experimental" (`peers.go:9-10`, `options.go:26`).
- **Write-through.** `Manager.Apply`/`applyEntities` ingests RPC results and resolves (`telegram/peers/apply.go:130-137`). Separately, the updates manager feeds channel/user hashes to its own `ChannelAccessHasher`/`UserAccessHasher`, skipping min/zero hashes (`telegram/updates/access_hash_feeder.go:5,49`; interfaces in `telegram/updates/storage.go:36-48`) — two parallel hash stores, not one cascade.
- **Min entities.** Min users are dropped from persistence (`apply.go:22-24`); min channels are never persisted — neither overwriting a real hash nor inserting a bogus one (`apply.go:88-95`, TODO: promotion hook).
- **Invalidation.** Passive. `USER_ID_INVALID`/`CHANNEL_INVALID` on a hashless ID become `PeerNotFoundError` (`id.go:37-43,60-66`); one targeted retry exists in message sending: join-on-`PEER_ID_INVALID` (`telegram/message/join.go:85`). No invalidate+retry middleware.
- **Errors.** Typed `PhoneNotFoundError`, `PeerNotFoundError` (`telegram/peers/errors.go:8,17`); transient-vs-missing distinguished via `tgerr.Is` (`tgerr/error.go:139`).
- **ID conventions.** Raw MTProto IDs at the surface; marked IDs only via `constant.TDLibPeerID` (`constant/tdlib_ids.go:26-56`).

## gogram

- **Entry points.** `Client.ResolvePeer(any)` = `GetSendablePeer`, a giant type switch accepting peers, objects, ints, strings (`"me"`, numeric, username) (`telegram/helpers.go:337-455,457`). Numeric IDs: `GetInputPeer` — `-100` prefix → channel, negative → chat, positive → user then channel fallback (`telegram/cache.go:667-729`; `trimSuffixHundred` `cache.go:1196`). Username: `Cache.LookupUsername` first (`cache.go:636`), else `ResolveUsername` → `contacts.resolveUsername` with cache write-through (`telegram/helpers.go:1337-1374`).
- **Cache layers.** Single `CACHE`: full entity maps, `usernameMap`, persisted `InputPeers{InputUsers/InputChannels: id→accessHash}` (`cache.go:116-164`), flushed to JSON `cache<session>.db` with debounced writes (`telegram/client.go:201-203`; `cache.go:463-535`), LRU + size cap. Updates write through pervasively (`telegram/updates.go:3123,3590,3942` …). No phone index; no dialogs-based fallback (verified absent).
- **Min entities.** Side maps `minUsers`/`minChannels` keep min hashes separate; never overwrite a real hash, deleted once a full entity arrives (`cache.go:945-999`).
- **Invalidation.** None on `PEER_ID_INVALID`. Not-found resolution retries `getUsers`/`getChannels` with `access_hash=0` (`cache.go:757-779,812-833`).
- **Errors.** Untyped `fmt.Errorf` strings (`cache.go:690,722`); `PEER_ID_INVALID` is only a description map entry (`errors.go:459`); flood handled via regex `MatchError`/`GetFloodWait` (`telegram/utils.go:500,515`; `telegram/const.go:31-33`).
- **ID conventions.** Marked `-100` IDs accepted at the surface; raw positive IDs stored internally.

## Comparison

gotd's split Storage/Cache with typed not-found errors and strict min-exclusion is cleaner, but its opt-in Manager plus a second updates-side hash store duplicates state. gogram's one persistent cache with write-through on every update path is operationally simpler (and persists usernames), at the cost of stringly-typed errors and no phone/username invalidation. mtgo's "single Manager cascade + ingest + invalidate middleware" subsumes both: gotd-grade typing/min rules with gogram-grade single-store write-through.
