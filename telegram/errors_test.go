package telegram

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mtgo-labs/mtgo/tgerr"
)

// Compile-time interface compliance checks.
var (
	_ tgerr.ErrorInfo = (*ReconnectError)(nil)
	_ tgerr.ErrorInfo = (*MigrationError)(nil)
	_ tgerr.ErrorInfo = (*UnsafeMigrationError)(nil)
)

func TestReconnectError_Getters(t *testing.T) {
	e := &ReconnectError{Attempts: 5, Err: fmt.Errorf("timeout")}
	if e.ErrorCode() != 0 {
		t.Errorf("ErrorCode() = %d, want 0", e.ErrorCode())
	}
	if e.ErrorType() != "RECONNECT" {
		t.Errorf("ErrorType() = %q, want %q", e.ErrorType(), "RECONNECT")
	}
	if e.ErrorArg() != 5 {
		t.Errorf("ErrorArg() = %d, want 5", e.ErrorArg())
	}
}

func TestMigrationError_Getters(t *testing.T) {
	e := &MigrationError{TargetDC: 4, Err: fmt.Errorf("network")}
	if e.ErrorCode() != 303 {
		t.Errorf("ErrorCode() = %d, want 303", e.ErrorCode())
	}
	if e.ErrorType() != "MIGRATION" {
		t.Errorf("ErrorType() = %q, want %q", e.ErrorType(), "MIGRATION")
	}
	if e.ErrorArg() != 4 {
		t.Errorf("ErrorArg() = %d, want 4", e.ErrorArg())
	}
}

func TestUnsafeMigrationError_Getters(t *testing.T) {
	e := &UnsafeMigrationError{TargetDC: 2, Method: "sendMessage"}
	if e.ErrorCode() != 303 {
		t.Errorf("ErrorCode() = %d, want 303", e.ErrorCode())
	}
	if e.ErrorType() != "UNSAFE_MIGRATION" {
		t.Errorf("ErrorType() = %q, want %q", e.ErrorType(), "UNSAFE_MIGRATION")
	}
	if e.ErrorArg() != 2 {
		t.Errorf("ErrorArg() = %d, want 2", e.ErrorArg())
	}
}

func TestErrors_AsErrorInfo(t *testing.T) {
	tests := []error{
		&ReconnectError{Attempts: 1},
		&MigrationError{TargetDC: 2},
		&UnsafeMigrationError{TargetDC: 3, Method: "forwardMessages"},
	}
	for i, err := range tests {
		var info tgerr.ErrorInfo
		if !errors.As(err, &info) {
			t.Errorf("test %d: errors.As failed for %T", i, err)
		}
	}
}
