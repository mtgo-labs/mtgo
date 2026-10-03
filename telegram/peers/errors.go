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

// ErrInvalid marks a peer whose cached access hash was rejected by the
// server and remained rejected after invalidation and one replay — the peer
// exists but this client can no longer address it without a fresh entity.
var ErrInvalid = errors.New("peers: invalid")

// NotFoundError is the typed form of [ErrNotFound]: it carries a short
// description of the reference that failed and the underlying cause (when
// the failure came from an RPC). errors.Is(err, ErrNotFound) and
// errors.Is(err, ErrInvalid) keep working through it.
type NotFoundError struct {
	// Ref describes what was being resolved, e.g. "@durov" or "user 42".
	Ref string
	// Cause is the underlying error, if any.
	Cause error
}

func (e *NotFoundError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("peers: %s not found", e.Ref)
	}
	return fmt.Sprintf("peers: %s not found: %v", e.Ref, e.Cause)
}

// Unwrap exposes the cause for errors.Is/As inspection of the underlying
// RPC error.
func (e *NotFoundError) Unwrap() error { return e.Cause }

// Is makes the error satisfy errors.Is(err, ErrNotFound).
func (e *NotFoundError) Is(target error) bool { return target == ErrNotFound }

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
