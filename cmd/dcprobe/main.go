// dcprobe — throwaway diagnostic: probes fresh DH handshakes against
// Telegram DC endpoints over different transports to pin down which
// combination the server accepts.
//
// Variants per endpoint:
//   - plain      — raw abridged transport, no obfuscation
//   - obf-nodc   — obfuscated2, no dc_id encoded (mtgo behavior)
//   - obf-dc     — obfuscated2 with dc_id at nonce[60:62] (tdesktop/gotd)
//
// Result: Auth.Create either completes (routing + handshake OK) or fails
// (e.g. "server error code: 444" = wrong DC).
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/internal/transport"
)

type authAdapter struct {
	tp transport.Transport
}

func (a *authAdapter) Send(data []byte) error {
	return a.tp.Send(bytes.NewBuffer(data))
}

func (a *authAdapter) Recv() ([]byte, error) {
	return a.tp.Recv()
}

func main() {
	targets := []struct {
		name string
		addr string
		dc   int
	}{
		{"DC4-media", "149.154.165.96:443", 4},
		{"DC4-main", "149.154.167.91:443", 4},
		{"DC2-media", "149.154.167.223:443", 2},
		{"DC1-main", "149.154.175.53:443", 1},
	}
	for _, t := range targets {
		for _, variant := range []string{"plain", "obf-nodc", "obf-dc"} {
			res := probe(t.addr, t.dc, variant)
			fmt.Printf("%-10s %-9s -> %s\n", t.name, variant, res)
		}
	}
	os.Exit(0)
}

func probe(addr string, dc int, variant string) string {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return "dial: " + err.Error()
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))

	var tp transport.Transport = transport.NewTCPAbridged(conn)
	if variant != "plain" {
		nonce := make([]byte, 64)
		if _, err := rand.Read(nonce); err != nil {
			return "rand: " + err.Error()
		}
		if variant == "obf-dc" {
			binary.LittleEndian.PutUint16(nonce[60:62], uint16(dc))
		}
		obf := transport.NewTCPObfuscatedWithNonce(transport.NewTCPAbridged(conn), 0xEF, nonce, false)
		if err := obf.Connect(); err != nil {
			return "obf connect: " + err.Error()
		}
		tp = obf
	}

	auth := &session.Auth{DC: dc, TestMode: false}
	result, err := auth.Create(&authAdapter{tp: tp})
	if err != nil {
		return "FAIL: " + err.Error()
	}
	return fmt.Sprintf("OK (key established, time=%d)", result.ServerTime)
}
