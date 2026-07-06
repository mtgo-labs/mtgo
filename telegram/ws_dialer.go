package telegram

import (
	"context"
	"net"

	"github.com/mtgo-labs/mtgo/internal/transport"
)

// RawWSDialer is a factory that returns a raw WebSocket bytestream (no MTProto
// framing applied) for the given address. The returned net.Conn must be a
// reliable, ordered, bidirectional byte stream — typically wrapping a host
// WebSocket implementation such as the browser's `new WebSocket(url)` API.
//
// The address is the fully-resolved WebSocket URL that mtgo would otherwise
// pass to transport.DialWebsocket (e.g. "wss://pluto.web.telegram.org/apiws").
type RawWSDialer func(ctx context.Context, addr string) (net.Conn, error)

// NewWSDialer wraps a RawWSDialer with mtgo's internal obfuscated2 MTProto
// framing layer, returning a function suitable for assignment to Config.WSDialer.
//
// This lets external code (notably a GOOS=js GOARCH=wasm build) inject a browser
// WebSocket as the transport without importing internal packages:
//
//	cfg.WebSocket = true
//	cfg.WebSocketTLS = true
//	cfg.WSDialer = telegram.NewWSDialer(browserRawWSFactory)
//
// browserRawWSFactory opens `new WebSocket(addr)` via syscall/js and adapts it
// to net.Conn. mtgo handles the obfuscation handshake and intermediate framing.
func NewWSDialer(raw RawWSDialer) func(ctx context.Context, addr string) (net.Conn, error) {
	if raw == nil {
		return nil
	}
	return func(ctx context.Context, addr string) (net.Conn, error) {
		conn, err := raw(ctx, addr)
		if err != nil {
			return nil, err
		}
		return transport.WrapObfuscatedWS(conn)
	}
}
