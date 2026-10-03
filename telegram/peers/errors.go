package peers

import "errors"

// ErrNotFound is the sentinel for unresolvable peers: unknown IDs, usernames
// that do not exist, or cascades that exhausted every strategy. Wrap it with
// %w when adding context. Transient RPC failures must NOT wrap it.
//
// telegram.ErrPeerNotFound is an alias of this value.
var ErrNotFound = errors.New("peers: not found")
