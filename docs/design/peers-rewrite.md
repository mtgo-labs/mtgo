# Peer Subsystem Rewrite — Design & Plan

Status: proposed
Decisions: exported `telegram/peers` package · full resolution cascade at every call site
Constraint: **no breaking changes to the existing public API** (additive only)

## 1. Problem

Peer resolution in mtgo is fragmented across 8+ files with three capability tiers,
inconsistent errors, and duplicated logic. Full audit findings (file:line refs from
the exploration pass):

1. **Three resolution tiers.** `Client.ResolvePeer` (full fallback), cache-only
   `resolvePeer(r, id)` (~100 call sites), and `ChatRef/UserRef.resolve` (no RPC
   fallback for numeric IDs). `GetChatHistory(ctx, 12345, …)` fails on cold cache
   while `SendMessage` would succeed — Telethon/TDLib route everything through one
   cascade.
2. **Errors mask failures.** `ResolveUsername`/`ResolvePhone` wrap *every* RPC
   error in `ErrPeerNotFound` (double-`%w`), so FLOOD_WAIT/auth/network errors
   masquerade as "peer not found". 11 distinct wrap formats + ad-hoc strings;
   `PeerToInputPeer` errors escape the sentinel contract entirely.
3. **Engine lives in the god-file.** ~800 lines of resolution/cache logic in
   `client.go` (L122–127, 3561–3800, 4182–4470); fragments in `resolve_peer.go`,
   `peer_store.go`, `helpers.go`.
4. **Duplicated logic.** 5+ copies of PeerClass→bot-API-ID marking
   (`types/peers.go`, `types/message.go`, `generic/resolve.go`, …); `mergePeer`
   duplicates the external `storage.MergePeer`; two divergent t.me link parsers
   (`ChatRefFrom` vs `parseJoinLink`) that disagree on `+hash` links.
5. **Cache gaps.** No phone→ID index (`ResolvePhone` always RPCs;
   `storage.Peer.PhoneNumber` written but never read); RPC-response entities never
   cached; `joinByUsername` bypasses cache/coalescer; no TTL/invalidation —
   `PEER_ID_INVALID` surfaces raw instead of invalidate+re-resolve.
6. **Correctness hazards.** Unknown zero-hash user resolves to `InputPeerSelf`
   (`PeerToInputPeer`); `UserRef` is exported but unresolvable; O(N) scans in hot
   paths (`reverseUsernameCache`, `lookupUsername`, legacy store gets).
7. **Naming collisions.** `Client.ResolvePeer(ctx, any)` vs `Context.ResolvePeer(id) any`
   vs unexported `resolvePeer`; constructor asymmetry (`Username`→ChatRef vs
   `UserUsername`→UserRef, `ChatPeer` vs `UserInput`).

## 2. Target architecture

One deep module — `telegram/peers.Manager` — behind the frozen `telegram` API as a
facade. Shape follows gotd `telegram/peers` / mtcute `PeerManager` / TDLib
`ContactsManager`.

### 2.1 File layout

```
telegram/peers/            ← new exported package (the rewrite)
├── doc.go                 package contract: ID conventions, cascade order, error semantics
├── manager.go             Manager type, Options, construction; owns cache + store + coalescer
├── cascade.go             the ONE resolution cascade + single-flight keys
├── rpc.go                 RPC strategies: username, phone, numeric-bot, numeric-account, dialog preload
├── cache.go               memory cache: ID/username/phone indexes, FIFO eviction, min-hash guard
├── ingest.go              entity ingest from updates AND RPC responses (min policy, backfill)
├── storage.go             Store adapters (storage.PeerStore + legacy PeerCache); uses external MergePeer
├── invalidation.go        PEER_ID_INVALID / CHANNEL_INVALID → drop + re-resolve once
└── errors.go              typed error contract

telegram/
├── peer.go                NEW  — Client.peers() accessor; frozen-API delegates; PeerResolver adapter
├── peer_ref.go            KEPT — ChatRef/UserRef (frozen), resolve() delegates to Manager
├── peer_link.go           NEW  — single link parser: t.me, +hash invites, t.me/c/<id>, tg://
├── resolve_peer.go        DELETED — absorbed by peers/ + peer.go
├── peer_store.go          DELETED — absorbed by peers/storage.go
└── client.go              loses ~800 lines; keeps frozen method signatures as 1-line delegates

internal/peerid/           NEW leaf package (imports only tg)
└── peerid.go              MarkChannelID / UnmarkChannelID / Kind(id) / Canonical(id, peer)
                           — single source of truth for the sign convention, adopted at all
                           5+ duplicate sites (types, generic, message.go, …)
```

### 2.2 Manager surface (small interface, deep implementation)

```go
package peers

type Manager struct{ /* cache, store, invoker, cfg */ }

func New(invoker tg.Invoker, opts ...Option) *Manager
// Options: WithStore(peers.Store), WithCacheSize(int), WithSavePeers(bool), WithBotFlag(...)

// Resolution — the cascade: memory cache → store → RPC fallback (username/phone/
// numeric bot/account paths, dialog preload) → bot hash completion
func (m *Manager) InputPeer(ctx context.Context, r Ref) (tg.InputPeerClass, error)
func (m *Manager) InputUser(ctx context.Context, r Ref) (tg.InputUserClass, error)       // NotUserError
func (m *Manager) InputChannel(ctx context.Context, r Ref) (tg.InputChannelClass, error) // NotChannelError
func (m *Manager) EnsureUsable(ctx context.Context, p tg.InputPeerClass) (tg.InputPeerClass, error)

// Cache & ingest — no RPC
func (m *Manager) Cached(id int64) (tg.InputPeerClass, bool)
func (m *Manager) Cache(id int64, p tg.InputPeerClass)
func (m *Manager) IngestUsers(users []tg.UserClass)
func (m *Manager) IngestChats(chats []tg.ChatClass)
func (m *Manager) Invalidate(id int64)

// Ref — the value type behind ChatRef/UserRef (peers must not import telegram)
type Ref struct {
    ID       int64
    Username string
    Phone    string
    Peer     tg.InputPeerClass
}
```

Dependencies are accepted, not created: `tg.Invoker` (satisfied by `Client.Raw()`),
a `Store`, config knobs — so the Manager is unit-testable without a `*Client`.

### 2.3 Error contract

| Error | Meaning | Facade behavior |
|---|---|---|
| `peers.ErrNotFound` + `NotFoundError{Ref, Cause}` | genuine unknown: `USERNAME_NOT_OCCUPIED`, cascade exhausted | satisfies `errors.Is(err, telegram.ErrPeerNotFound)` |
| `peers.ErrInvalid` | stale access hash, unresolvable after invalidate+retry | wraps to `ErrPeerNotFound` |
| `peers.NotUserError` / `NotChannelError` | kind mismatch (replaces ad-hoc `"peer %T is not a …"` strings) | additive, new |
| transient RPC errors | flood / network / auth — **pass through unwrapped** | unchanged shape |

### 2.4 Frozen public API → facade mapping

| Frozen symbol | Implementation after rewrite |
|---|---|
| `Client.ResolvePeer(ctx, any)` | switch → `manager.InputPeer` (+ additive `UserRef` case) |
| `Client.ResolveUsername` / `ResolvePhone` | `manager.InputPeer(peers.Ref{Username/Phone: …})` |
| `Client.ResolvePeerCache(id)` | `manager.Cached` → wrap miss as `ErrPeerNotFound` |
| `Client.CachePeer(id, peer)` | `manager.Cache` |
| `Client.SavePeer/GetPeer/GetPeerByUsername/LoadPeers/DeletePeer` | unchanged storage facade |
| `PeerResolver` interface (helpers.go) | adapter over Manager; implemented by Client as today |
| `ChatRef`/`UserRef` + all 9 constructors | unchanged; `resolve()` delegates to Manager |
| `PeerToInputPeer` | reimplemented on shared user/chat maps; errors join the sentinel contract |
| `types.PeerMap`, `NewPeerMap*`, `GetPeerID` | unchanged signatures; internals use `internal/peerid` |
| `generic.PeerInput`, `generic.Caller`, generated wrappers | unchanged; still funnel to `Client.ResolvePeer` |
| `Context.ResolvePeer(id) any` | frozen; deprecated in docs; additive typed `Context.Peer(id)` |

## 3. Intentional behavior changes

1. Full cascade at all ~100 former cache-only call sites (Telethon
   `get_input_entity` semantics — cold cache now triggers RPC, not failure).
2. Transient RPC errors no longer masquerade as `ErrPeerNotFound`; `errors.Is`
   still matches for genuine not-found.
3. Unknown zero-hash user no longer resolves to `InputPeerSelf` (correctness fix).
4. Phone resolution cached: in-memory phone index + `storage.Peer.PhoneNumber` read-back.
5. Entities from any RPC response ingested into the cache (today only updates and
   `resolveUsername/Phone` responses).
6. `PEER_ID_INVALID` / `CHANNEL_INVALID` → invalidate + re-resolve once.
7. One link parser handling `t.me`, `telegram.me/dog`, `+hash` invites,
   `t.me/c/<id>/<msg>`, `tg://`; `joinByUsername` rejoins the cascade.
8. `UserRef` resolvable through `Client.ResolvePeer` (additive switch case).
9. `ChatRef` not-found error includes the offending ID/username.

## 4. Phasing

Gate for every phase: `go build ./... && go vet ./... && golangci-lint run && go test ./...`
(plus benchmarks on cache hot paths per the engineering priority order).

- **P0 — `internal/peerid`** (pure refactor): extract marking helpers, adopt at all
  duplicate sites, delete copies. No behavior change.
- **P1 — package skeleton with today's semantics**: build `telegram/peers`
  (manager, cascade, cache, storage, ingest) reproducing current behavior; wire
  `Client` facade delegates; delete `resolve_peer.go`/`peer_store.go`; slim
  `client.go`. Moves only — behavior identical.
- **P2 — unification**: swap ~100 `resolvePeer` call sites to the cascade; phone
  index; RPC-response ingest; `peer_link.go`; `joinByUsername` reuse; `UserRef`
  dispatch; fix O(N) scans (reverse index, dialog fallback) and
  `SendMessage`'s broken `InputPeerUserFromMessage` fallback.
- **P3 — error contract & hardening**: typed errors + facade mapping; stop error
  masking; `InputPeerSelf` misattribution fix; PEER_ID_INVALID invalidation;
  doc/config drift (`API_REFERENCE.md`, `PeerCacheSize`/`SavePeers` comments);
  deprecate `Context.ResolvePeer` in docs, add `Context.Peer`.

## 5. Verification

- Existing tests must pass unmodified through P1 (pure move).
- New unit tests per peers/ file: cache (eviction, min-guard, phone index),
  cascade order (mock invoker), ingest (min entities), invalidation, link parser
  (table-driven), error mapping.
- `mockTransport`/`mockPeerResolver` patterns continue to work — `PeerResolver`
  adapter keeps `testResolver` override honored.
- goleak in internal/session unaffected; add goroutine-leak check for coalescer
  waiters if needed.
- Bench: `CachePeer`/`ResolvePeerCache`/eviction before/after (allocation +
  latency), since these sit on every send path.
