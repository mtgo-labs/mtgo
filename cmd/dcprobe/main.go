// Command dcprobe verifies MTProto DH key-generation availability per
// Telegram DC endpoint.
//
// It reproduces and verifies issue #110: the static media-only endpoints
// (prodMediaIPv4) answer req_pq_multi but reject req_DH_params with
// transport error -444, while the DC's main endpoint accepts the identical
// handshake. Auxiliary sessions (inline edits on foreign-DC inline message
// ids, file transfers through the DC pools) fail to establish until mtgo
// falls back to the main endpoint.
//
// For every requested DC the probe performs a fresh, complete DH handshake
// against the main endpoint and — when one is known — the media endpoint,
// in three transport variants:
//
//	plain   abridged transport, no obfuscation
//	obf     obfuscated2 abridged transport (mtgo AlwaysObfuscate behavior)
//	obf-dc  obfuscated2 with the DC id embedded in the nonce (MTProxy style)
//
// The -config flag additionally performs an unauthenticated help.getConfig
// over the first successful main-endpoint handshake and prints the live
// dcOption table, which is the authoritative source for media endpoints.
//
// Usage:
//
//	dcprobe              probe all production DCs
//	dcprobe -dc 2        probe a single DC
//	dcprobe -test        probe test DCs
//	dcprobe -config      also dump live help.getConfig dcOptions
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"net"
	"sort"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/internal/transport"
	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg"
)

var (
	dcFlag    = flag.Int("dc", 0, "probe only this DC (0 = all production DCs)")
	testFlag  = flag.Bool("test", false, "probe test DCs instead of production")
	configFlg = flag.Bool("config", false, "fetch help.getConfig on first success and print live dcOptions")
	timeout   = flag.Duration("timeout", 10*time.Second, "per-handshake timeout")
)

const abridgedMarker = 0xEF

func main() {
	flag.Parse()

	dcIDs := []int{1, 2, 3, 4, 5}
	if *dcFlag > 0 {
		dcIDs = []int{*dcFlag}
	}

	handshakes := 0
	for _, dcID := range dcIDs {
		base := session.DataCenter{ID: dcID, TestMode: *testFlag}
		endpoints := []endpointSpec{
			{name: fmt.Sprintf("DC%d-main", dcID), dc: base},
		}
		if transfer := base.TransferDataCenter(); transfer.Address() != "" && transfer.Address() != base.Address() {
			endpoints = append(endpoints, endpointSpec{name: fmt.Sprintf("DC%d-media", dcID), dc: transfer})
		}
		for _, ep := range endpoints {
			for _, variant := range []string{"plain", "obf", "obf-dc"} {
				handshakes++
				report(dcID, ep, variant, probe(dcID, ep, variant))
			}
		}
	}

	if *configFlg {
		dumpLiveConfig(dcIDs)
	}

	fmt.Printf("\n%d handshakes completed\n", handshakes)
}

type endpointSpec struct {
	name string
	dc   session.DataCenter
}

// probe performs one complete DH key exchange against the endpoint using
// the requested transport variant.
func probe(dcID int, ep endpointSpec, variant string) error {
	addr := net.JoinHostPort(ep.dc.Address(), fmt.Sprint(ep.dc.Port()))
	conn, err := net.DialTimeout("tcp", addr, *timeout)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(*timeout))

	tp, err := newVariantTransport(conn, variant, dcID)
	if err != nil {
		return err
	}
	if err := tp.Connect(); err != nil {
		return fmt.Errorf("transport connect: %w", err)
	}

	auth := &session.Auth{DC: dcID, TestMode: *testFlag}
	result, err := auth.Create(&probeAuthTransport{tp: tp})
	if err != nil {
		return err
	}
	return success{serverTime: result.ServerTime}
}

// success reports an established key alongside the server's reported time.
type success struct{ serverTime int32 }

func (s success) Error() string { return "" }

func report(dcID int, ep endpointSpec, variant string, err error) {
	fmt.Printf("%-9s  %-8s -> ", ep.name, variant)
	var ok success
	if errors.As(err, &ok) {
		fmt.Printf("OK (key established, server_time=%d)\n", ok.serverTime)
		return
	}
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		return
	}
	fmt.Println("OK")
}

// connectableTransport is a transport that still needs its handshake
// (obfuscated2 nonce or plain protocol marker) performed via Connect.
type connectableTransport interface {
	transport.Transport
	Connect() error
}

// newVariantTransport wraps conn for the requested probe variant.
func newVariantTransport(conn net.Conn, variant string, dcID int) (connectableTransport, error) {
	switch variant {
	case "plain":
		return transport.NewTCPAbridged(conn), nil
	case "obf":
		return transport.NewTCPObfuscated(transport.NewTCPAbridged(conn), abridgedMarker), nil
	case "obf-dc":
		nonce := make([]byte, 64)
		if _, err := rand.Read(nonce); err != nil {
			return nil, err
		}
		// MTProxy-style DC routing tag: bytes 60-62 of the obfuscated2
		// nonce, little-endian (see mtproxy.obfuscated2Handshake).
		binary.LittleEndian.PutUint16(nonce[60:62], uint16(int16(dcID)))
		return transport.NewTCPObfuscatedWithNonce(transport.NewTCPAbridged(conn), abridgedMarker, nonce, false), nil
	default:
		return nil, fmt.Errorf("unknown variant %q", variant)
	}
}

// dumpLiveConfig establishes one unauthenticated session on the first DC
// whose main endpoint completes a handshake and prints the live dcOption
// table from help.getConfig.
func dumpLiveConfig(dcIDs []int) {
	fmt.Println("\n-- live help.getConfig --")
	for _, dcID := range dcIDs {
		base := session.DataCenter{ID: dcID, TestMode: *testFlag}
		if base.Address() == "" {
			continue
		}
		cfg, err := fetchConfig(base, dcID)
		if err != nil {
			fmt.Printf("DC%d: %v\n", dcID, err)
			continue
		}
		printDCOptions(dcID, cfg)
		return
	}
	fmt.Println("no DC completed a handshake for help.getConfig")
}

func fetchConfig(base session.DataCenter, dcID int) (*tg.Config, error) {
	addr := net.JoinHostPort(base.Address(), fmt.Sprint(base.Port()))
	conn, err := net.DialTimeout("tcp", addr, *timeout)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	tp := transport.NewTCPObfuscated(transport.NewTCPAbridged(conn), abridgedMarker)
	if err := tp.Connect(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("transport connect: %w", err)
	}

	auth := &session.Auth{DC: dcID, TestMode: *testFlag}
	result, err := auth.Create(&probeAuthTransport{tp: tp})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("DH exchange: %w", err)
	}

	st := telegram.NewMemoryStorage()
	if err := st.SetAuthKey(result.AuthKey); err != nil {
		conn.Close()
		return nil, err
	}
	sess, err := session.NewSession(base, st, "dcprobe", "1.0", "en", "en")
	if err != nil {
		conn.Close()
		return nil, err
	}
	sess.SetServerSalt(result.ServerSalt)
	sess.SetServerTime(time.Unix(int64(result.ServerTime), 0))
	if err := sess.Connect(&probeSessionTransport{tp: tp, conn: conn}, *timeout); err != nil {
		conn.Close()
		return nil, fmt.Errorf("session connect: %w", err)
	}
	defer sess.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	// Modern servers require the invokeWithLayer(initConnection(...)) wrapper
	// even for unauthenticated queries.
	query := &tg.InvokeWithLayerRequest{
		Layer: tg.Layer,
		Query: &tg.InitConnectionRequest{
			APIID:          6, // arbitrary id: the query is unauthenticated
			DeviceModel:    "dcprobe",
			SystemVersion:  "linux",
			AppVersion:     "1.0",
			SystemLangCode: "en",
			LangCode:       "en",
			Query:          &tg.HelpGetConfigRequest{},
		},
	}
	resp, err := sess.Invoke(ctx, query, 1, *timeout)
	if err != nil {
		return nil, fmt.Errorf("help.getConfig: %w", err)
	}
	cfg, ok := resp.(*tg.Config)
	if !ok {
		return nil, fmt.Errorf("help.getConfig: unexpected response %T", resp)
	}
	return cfg, nil
}

func printDCOptions(dcID int, cfg *tg.Config) {
	options := append([]*tg.DCOption(nil), cfg.DCOptions...)
	sort.Slice(options, func(i, j int) bool {
		if options[i].ID != options[j].ID {
			return options[i].ID < options[j].ID
		}
		return options[i].IpAddress < options[j].IpAddress
	})
	fmt.Printf("DC%d answered (this_dc=%d, %d dcOptions):\n", dcID, cfg.ThisDC, len(options))
	for _, opt := range options {
		flags := ""
		if opt.MediaOnly {
			flags += " media_only"
		}
		if opt.TcpoOnly {
			flags += " tcpo_only"
		}
		if opt.Static {
			flags += " static"
		}
		if opt.IPv6 {
			flags += " ipv6"
		}
		if opt.CDN {
			flags += " cdn"
		}
		fmt.Printf("  dc%-2d %s:%-5d%s\n", opt.ID, opt.IpAddress, opt.Port, flags)
	}
}

// probeAuthTransport adapts a transport.Transport to session's authTransport.
type probeAuthTransport struct {
	tp transport.Transport
}

func (t *probeAuthTransport) Send(data []byte) error {
	return t.tp.Send(bytes.NewBuffer(data))
}

func (t *probeAuthTransport) Recv() ([]byte, error) { return t.tp.Recv() }

// probeSessionTransport adapts a transport.Transport plus its net.Conn to
// session.Transport.
type probeSessionTransport struct {
	tp   transport.Transport
	conn net.Conn
}

func (t *probeSessionTransport) Send(data []byte) error { return t.tp.Send(bytes.NewBuffer(data)) }
func (t *probeSessionTransport) Recv() ([]byte, error)  { return t.tp.Recv() }
func (t *probeSessionTransport) Close() error           { return t.conn.Close() }
func (t *probeSessionTransport) IsConnected() bool      { return t.conn.RemoteAddr() != nil }
func (t *probeSessionTransport) SetWriteDeadline(tm time.Time) error {
	return t.conn.SetWriteDeadline(tm)
}

func (t *probeSessionTransport) SetReadDeadline(tm time.Time) error {
	return t.conn.SetReadDeadline(tm)
}
