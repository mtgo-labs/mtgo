package telegram

import (
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"testing"

	"github.com/mtgo-labs/mtgo/internal/session"
)

func TestIsSessionClosedErrUsesTypedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "session closed", err: session.ErrSessionClosed, want: true},
		{name: "wrapped draining", err: fmt.Errorf("invoke: %w", session.ErrDraining), want: true},
		{name: "network closed", err: net.ErrClosed, want: true},
		{name: "EOF", err: io.EOF, want: true},
		{name: "connection reset", err: syscall.ECONNRESET, want: true},
		{name: "broken pipe", err: syscall.EPIPE, want: true},
		{name: "string lookalike", err: errors.New("broken pipe"), want: false},
		{name: "unrelated", err: errors.New("rpc failed"), want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isSessionClosedErr(test.err); got != test.want {
				t.Fatalf("isSessionClosedErr(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}
