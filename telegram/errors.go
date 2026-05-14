package telegram

import (
	"errors"
	"fmt"
)

var (
	ErrNotConnected      = errors.New("client: not connected")
	ErrAlreadyConnected  = errors.New("client: already connected")
	ErrPeerNotFound      = errors.New("client: peer not found")
	ErrClientClosed      = errors.New("client: closed")
	ErrReconnectFailed   = errors.New("client: reconnect failed")
	ErrHealthTimeout     = errors.New("client: health check timeout")
	ErrMigrationFailed   = errors.New("client: DC migration failed")
	ErrMigrationUnsafe   = errors.New("client: DC migration unsafe for non-idempotent request")
	ErrMigrationUnknown  = errors.New("client: DC migration to unknown DC")
)

// ReconnectError indicates that reconnection attempts were exhausted.
//
// Example:
//
//	err := client.Connect(ctx)
//	if err != nil {
//		var reconnErr *telegram.ReconnectError
//		if errors.As(err, &reconnErr) {
//			fmt.Printf("gave up after %d attempts: %v\n", reconnErr.Attempts, reconnErr.Err)
//		}
//	}
type ReconnectError struct {
	Attempts int
	Err      error
}

func (e *ReconnectError) Error() string {
	return fmt.Sprintf("client: reconnect failed after %d attempts: %v", e.Attempts, e.Err)
}

func (e *ReconnectError) Unwrap() error { return e.Err }

// MigrationError indicates a failure to migrate the connection to a different DC.
//
// Example:
//
//	_, err := client.SendMessage(ctx, peer, "hello")
//	if err != nil {
//		var migErr *telegram.MigrationError
//		if errors.As(err, &migErr) {
//			fmt.Printf("migration to DC %d failed: %v\n", migErr.TargetDC, migErr.Err)
//		}
//	}
type MigrationError struct {
	TargetDC int
	Err      error
}

func (e *MigrationError) Error() string {
	return fmt.Sprintf("client: DC migration to DC %d failed: %v", e.TargetDC, e.Err)
}

func (e *MigrationError) Unwrap() error { return e.Err }

// UnsafeMigrationError indicates a non-idempotent request was interrupted by a DC migration.
// The request is not automatically retried because it may have already been applied.
//
// Example:
//
//	_, err := client.ForwardMessages(ctx, peer, msgIDs)
//	if err != nil {
//		var unsafeErr *telegram.UnsafeMigrationError
//		if errors.As(err, &unsafeErr) {
//			fmt.Printf("non-idempotent %q interrupted by migration to DC %d\n",
//				unsafeErr.Method, unsafeErr.TargetDC)
//		}
//	}
type UnsafeMigrationError struct {
	TargetDC int
	Method   string
}

func (e *UnsafeMigrationError) Error() string {
	return fmt.Sprintf("client: refusing to retry non-idempotent %q after DC migration to DC %d", e.Method, e.TargetDC)
}
