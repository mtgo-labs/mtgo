// Package peers is mtgo's peer resolution engine: the single module that turns
// any peer reference (numeric ID, username, phone number, t.me link, or a
// pre-built input peer) into a usable tg.InputPeerClass, backed by an
// in-memory cache and an optional persistent store.
//
// # ID conventions
//
// Marked chat IDs follow internal/peerid: users positive, basic groups
// negated, channels/supergroups prefixed with peerid.ChannelPrefix. All
// arithmetic goes through the peerid package.
//
// # Cascade
//
// Resolution follows one cascade, in order:
//
//  1. pre-built tg.InputPeerClass (passthrough)
//  2. phone number (contacts.resolvePhone, coalesced)
//  3. username (cache, then contacts.resolveUsername, coalesced)
//  4. numeric ID (cache, then bot/account-specific RPC fallbacks)
//
// For bots, resolved peers are additionally completed with a usable access
// hash via users.getUsers / channels.getChannels.
//
// # Errors
//
// Resolution failures that mean "this peer cannot be resolved" wrap
// [ErrNotFound]. Transient RPC errors (flood wait, network, auth) are
// returned unwrapped so callers can distinguish them from not-found.
//
// # Wiring
//
// telegram.Client constructs the Manager and exposes it through its frozen
// public API (ResolvePeer, ResolveUsername, ResolvePhone, ResolvePeerCache,
// CachePeer). The Manager reads its dependencies through the function fields
// of [Deps] at call time, so lazily-created storage and runtime configuration
// changes are honored without re-construction.
//
// # Provenance
//
// The design is grounded in a source-level survey of the other MTProto
// clients (see docs/design/peers-cross-lib.md in the repository):
//
//   - gotd/td: the opt-in peers.Manager shape and strict min-exclusion; its
//     singleflight dedup is the precedent for the coalescer.
//   - mtcute: one PeersService owning ingest, persistence and TTL; the 24h
//     username freshness window; typed not-found errors.
//   - Pyrogram: write-through caching of entities from updates and RPC
//     responses; the 8h username TTL.
//   - MTKruto: persisted username/phone indexes with 24h TTLs;
//     message-anchored references for min peers (a recorded follow-up here).
//   - gogram: the single persistent id→access-hash store with write-through.
//
// Behaviors not found in any of the five as of the survey: invalidate-and-
// replay on PEER_ID_INVALID ([Manager.InvalidateOnStaleHash]), context-aware
// request coalescing, and typed kind errors ([NotUserError]/[NotChannelError]).
package peers
