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
package peers
