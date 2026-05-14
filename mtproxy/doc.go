// Package mtproxy implements MTProxy client support for connecting to
// Telegram through intermediate proxy servers.
//
// MTProxy allows bypassing network restrictions by tunneling MTProto traffic
// through a proxy that mimics normal TLS or uses obfuscated transport.
//
// Three secret types are supported:
//
//   - Simple (32 hex chars): raw 16-byte secret, uses obfuscated2 transport.
//   - dd-prefixed (34 hex chars): tag byte + 16-byte secret, uses obfuscated2
//     with PaddedIntermediate codec (most common).
//   - ee-prefixed (36+ hex chars): tag byte + 16-byte secret + domain name,
//     uses fake TLS handshake followed by obfuscated2.
//
// Basic usage:
//
//	conn, err := mtproxy.Dial(
//	    "proxy.example.com:443",
//	    "dd05fb7acb549be047a7c585116581418",
//	    2,     // DC ID
//	    30 * time.Second,
//	)
//
// Or via the client config:
//
//	client, _ := telegram.NewClient(apiID, apiHash, &telegram.Config{
//	    MTProxy: &telegram.MTProxyConfig{
//	        Addr:   "proxy.example.com:443",
//	        Secret: "dd05fb7acb549be047a7c585116581418",
//	    },
//	})
package mtproxy
