package transport

import (
	"bytes"
	"net"
)

// Transport defines the interface for MTProto transport-level message framing.
// Implementations handle serialization of outgoing payloads and deserialization
// of incoming payloads according to a specific wire format (full, abridged,
// intermediate, etc.).
type Transport interface {
	// Send writes the contents of buf to the underlying connection using the
	// transport's framing format. The caller must not modify buf while Send is
	// in progress.
	Send(buf *bytes.Buffer) error

	// Recv reads the next framed message from the underlying connection and
	// returns the decrypted payload bytes. It blocks until a complete message
	// is available or an error occurs.
	Recv() ([]byte, error)
}

// Conn is an alias for net.Conn used throughout the transport package to
// represent a raw network connection.
type Conn interface {
	net.Conn
}
