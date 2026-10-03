# Peer Resolution Across MTProto Clients — Comparison Matrix

Grounded survey of how six client libraries resolve peers, from primary sources
(shallow clones, 2026-10-03). Detailed per-library reports:

- `peers-resolution-gotd-gogram.md` — gotd/td, gogram (Go)
- session artifact `peer-resolution-pyrogram-mtcute-mtkruto.md` — Pyrogram, mtcute, MTKruto

This matrix drove (and validates) the `feat/peers-rewrite` design; the
"mtgo" column is the implemented state on that branch.

## Matrix

| Dimension | gotd/td | gogram | Pyrogram | mtcute | MTKruto | mtgo (feat/peers-rewrite) |
|---|---|---|---|---|---|---|
| **Resolve entry** | opt-in `peers.Manager.Resolve/ResolvePhone/ResolveDomain/ResolveUserID` (`peers/resolve.go`, `id.go`) | `ResolvePeer(any)` type switch (`helpers.go:337`) | `resolve_peer` cascade (`resolve_peer.py:31`) | `resolvePeer` + min-dummy unwrap (`resolve-peer.ts:21`) | `#getInputPeerInner` (`6_client.ts:607`) | **one `peers.Manager` cascade behind frozen `Client.ResolvePeer` facade; every call site uses it** |
| **Cache tiers** | 2: hash-only `Storage` + full-entity `Cache` (separate updates-side hash store!) | 1: client-wide `CACHE`, JSON-persisted, LRU | 1: SQLite-only | 2: LRU(100) over SQLite repos | in-memory maps + KV, 5s commit | **2: in-memory FIFO over `peers.Store` (single engine, no duplicate state)** |
| **Stores username? phone?** | username ✅ phone ✅ (contacts-hash trick) | username ✅ phone ❌ | both ✅ | both ✅ (+usernames[]) | both ✅ | **both ✅ (phone index memory-only — store contract lacks GetPeerByPhone)** |
| **Write-through scope** | `Apply` on RPC results + separate updates feeder | every update path | updates + per-method `fetch_peers` | debounced updates ingest | updates **and every RPC result** | updates, dialogs, resolve, bot paths, search — general RPC-response ingest deferred |
| **Min entities** | never persisted; no promotion | side maps, never overwrite | ignored (min overwrites full — worst) | full per-field merge (best) | skipped + message-anchored refs | zero-hash never overwrites (`preserveAccessHash`); no field merge |
| **Username/phone TTL** | — | — | 8h | 24h | 24h | **initially none → adopted this survey: 24h default (`Config.PeerIndexTTL`)** |
| **PEER_ID_INVALID reaction** | passive; one join-retry | none (hash-0 retry) | none | none | none | **invalidate + replay once, both invoke paths (middleware)** |
| **Not-found error typing** | `PeerNotFoundError`, `PhoneNotFoundError` | untyped `fmt.Errorf` | generated `PeerIdInvalid` | `MtPeerNotFoundError` | untyped `InputError` | **`NotFoundError{Ref,Cause}` + `NotUserError`/`NotChannelError`; transient errors unmasked** |
| **ID surface** | raw (marked via `constant.TDLibPeerID`) | marked `-100` | marked | marked | marked | **marked via `internal/peerid`, both forms accepted** |
| **Single-flight dedup** | `singleflight.Group` | ✗ | ✗ | ✗ | ✗ | **coalescer (ctx-aware)** |

## Decisions

Adopted from this survey (implemented / wired on the branch):

1. **Username/phone index TTL (24h default)** — Pyrogram 8h, mtcute 24h,
   MTKruto 24h all age out stale username→ID mappings; mtgo cached forever
   until FIFO eviction. Now `Config.PeerIndexTTL` (0 → default 24h, negative →
   disabled), enforced on read with self-healing eviction.
2. Confirmed-keep: invalidate+replay middleware (no surveyed library has it),
   typed not-found errors (only gotd/mtcute comparable), coalescer (only gotd
   has single-flight), min-hash guard (only Pyrogram is worse).

Deferred (recorded as follow-ups, not blocking):

3. **General RPC-response ingest** — Pyrogram (`fetch_peers`) and MTKruto
   (persist every result) write through from *all* RPC responses. Worth an
   invoker-level response-ingest middleware; moderate invasiveness.
4. **Min-entity field merge** (mtcute `peers.ts:183-260`, per Telegram
   constructor rules) — upgrade from hash-only preservation to per-field merge.
5. **Message-anchored min references** (MTKruto `addMinPeerReference`) — store
   `inputPeer*FromMessage` anchors for min peers lacking hashes.
6. **Phone re-resolve fallback for min peers** (mtcute) — cascade tries cached
   username today; add cached phone.
