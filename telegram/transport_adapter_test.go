package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/internal/transport"
)

func TestNewTCPTransportModes(t *testing.T) {
	tests := []struct {
		mode string
		want any
	}{
		{TransportModeAbridged, (*transport.TCPAbridged)(nil)},
		{TransportModeIntermediate, (*transport.TCPIntermediate)(nil)},
		{TransportModePaddedIntermediate, (*transport.TCPPaddedIntermediate)(nil)},
		{TransportModeFull, (*transport.TCPFull)(nil)},
	}

	for _, tt := range tests {
		got, err := newTCPTransport(tt.mode, nil)
		if err != nil {
			t.Fatalf("newTCPTransport(%q) error: %v", tt.mode, err)
		}
		switch tt.want.(type) {
		case *transport.TCPAbridged:
			if _, ok := got.(*transport.TCPAbridged); !ok {
				t.Fatalf("newTCPTransport(%q) = %T", tt.mode, got)
			}
		case *transport.TCPIntermediate:
			if _, ok := got.(*transport.TCPIntermediate); !ok {
				t.Fatalf("newTCPTransport(%q) = %T", tt.mode, got)
			}
		case *transport.TCPPaddedIntermediate:
			if _, ok := got.(*transport.TCPPaddedIntermediate); !ok {
				t.Fatalf("newTCPTransport(%q) = %T", tt.mode, got)
			}
		case *transport.TCPFull:
			if _, ok := got.(*transport.TCPFull); !ok {
				t.Fatalf("newTCPTransport(%q) = %T", tt.mode, got)
			}
		}
	}
}
