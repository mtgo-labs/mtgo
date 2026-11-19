package session

import (
	"encoding/binary"
	"testing"

	"github.com/mtgo-labs/mtgo/internal/transport"
)

type errorCodeTransport struct {
	payload []byte
}

func (t *errorCodeTransport) Send([]byte) error { return nil }

func (t *errorCodeTransport) Recv() ([]byte, error) { return t.payload, nil }

// TestAuthServerErrorCodeTyped verifies that a 4-byte server error received
// during the unencrypted handshake surfaces as a typed transport error, so
// callers can detect -444 (endpoint does not host key generation for the
// declared DC; issue #110) with transport.IsTransportError.
func TestAuthServerErrorCodeTyped(t *testing.T) {
	code := int32(transport.ErrCodeInvalidDC)
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, uint32(code))

	auth := &Auth{DC: 2}
	_, err := auth.Create(&errorCodeTransport{payload: payload})
	if err == nil {
		t.Fatal("Create error = nil, want transport error")
	}
	if !transport.IsTransportError(err, transport.ErrCodeInvalidDC) {
		t.Fatalf("Create error = %v, want -444 transport error", err)
	}
}
