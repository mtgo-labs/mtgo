package peers

import (
	"errors"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// ErrNotFound is the sentinel for unresolvable peers: unknown IDs, usernames
// that do not exist, or cascades that exhausted every strategy. Wrap it with
// %w when adding context. Transient RPC failures (flood wait, network, auth)
// must NOT wrap it.
//
// telegram.ErrPeerNotFound is an alias of this value.
var ErrNotFound = errors.New("peers: not found")

// NotUserError is returned when a resolved peer cannot be used as an input
// user because it is of a different kind.
type NotUserError struct {
	Peer tg.InputPeerClass
}

func (e NotUserError) Error() string {
	return fmt.Sprintf("peers: input peer %T is not a user", e.Peer)
}

// NotChannelError is returned when a resolved peer cannot be used as an
// input channel because it is of a different kind.
type NotChannelError struct {
	Peer tg.InputPeerClass
}

func (e NotChannelError) Error() string {
	return fmt.Sprintf("peers: input peer %T is not a channel", e.Peer)
}

// notFoundRPCErrors lists server errors meaning the referenced peer does not
// exist or is inaccessible — as opposed to transport, flood, or auth trouble
// that callers may want to retry.
var notFoundRPCErrors = []string{
	"USERNAME_NOT_OCCUPIED",
	"USERNAME_INVALID",
	"PHONE_NUMBER_INVALID",
	"PHONE_NUMBER_UNPRIVACY",
	"PHONE_NUMBER_BANNED",
	"PEER_ID_INVALID",
	"USER_ID_INVALID",
	"CHANNEL_INVALID",
	"CHANNEL_PRIVATE",
}

// isNotFoundRPC classifies an RPC error from a resolution call: true means
// the peer genuinely cannot be resolved (wrap with [ErrNotFound]); false
// means the error is transient or unrelated and must pass through unmasked.
func isNotFoundRPC(err error) bool {
	return tgerr.Is(err, notFoundRPCErrors...)
}
