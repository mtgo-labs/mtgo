package telegram

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/internal/transport"
	"github.com/mtgo-labs/mtgo/tg"
)

// issue110Dialer hands out scripted server connections for the auxiliary-DC
// handshake over the intermediate transport. Endpoints listed in rejectDH
// answer req_pq_multi with a valid ResPQ and then reject req_DH_params with
// transport error -444, exactly like the static media endpoints described in
// issue #110. Every other endpoint closes the connection after the ResPQ
// (step 8 fails with EOF), so the DH never completes against them.
type issue110Dialer struct {
	mu       sync.Mutex
	dialed   []string
	rejectDH map[string]bool
}

func (d *issue110Dialer) Dial(network, address string, timeout time.Duration) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address, timeout)
}

func (d *issue110Dialer) DialContext(_ context.Context, _, address string, _ time.Duration) (net.Conn, error) {
	d.mu.Lock()
	d.dialed = append(d.dialed, address)
	reject := d.rejectDH[address]
	d.mu.Unlock()

	clientConn, serverConn := net.Pipe()
	go serveIssue110Endpoint(serverConn, reject)
	return clientConn, nil
}

func (d *issue110Dialer) calls() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.dialed...)
}

// serveIssue110Endpoint plays a minimal unauthenticated Telegram frontend:
// intermediate transport framing, one valid ResPQ (echoed nonce, factorable
// PQ, trusted fingerprint), then either the 4-byte -444 transport error or a
// close.
func serveIssue110Endpoint(conn net.Conn, rejectDH bool) {
	defer conn.Close()
	// Intermediate transport Connect marker (0xEEEEEEEE).
	marker := make([]byte, 4)
	if _, err := io.ReadFull(conn, marker); err != nil {
		return
	}
	reqPQ, err := readIssue110Frame(conn)
	if err != nil || len(reqPQ) < 20 {
		return
	}
	// Unencrypted MTProto frame: auth_key_id[8] msg_id[8] len[4] body.
	body := reqPQ[20:]
	if len(body) < 20 || binary.LittleEndian.Uint32(body[:4]) != tg.ReqPQMultiTypeID {
		return
	}
	pq := make([]byte, 2)
	binary.BigEndian.PutUint16(pq, 251*257) // 64507 = 251 * 257, both in smallPrimes' reach
	resPQ := &tg.ResPQ{
		Nonce:                       [16]byte(body[4:20]),
		ServerNonce:                 [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		PQ:                          string(pq),
		ServerPublicKeyFingerprints: crypto.ServerKeyFingerprints()[:1],
	}
	var resPQBody bytes.Buffer
	if err := resPQ.Encode(&resPQBody); err != nil {
		return
	}
	writeIssue110Frame(conn, packIssue110Unencrypted(resPQBody.Bytes()))
	// Consume req_DH_params.
	if _, err := readIssue110Frame(conn); err != nil {
		return
	}
	if rejectDH {
		writeIssue110Frame(conn, []byte{0x44, 0xfe, 0xff, 0xff}) // int32 -444, little-endian
	}
}

// packIssue110Unencrypted frames a payload as an unencrypted MTProto message.
func packIssue110Unencrypted(payload []byte) []byte {
	frame := make([]byte, 20+len(payload))
	binary.LittleEndian.PutUint64(frame[8:16], uint64(time.Now().Unix())<<32|2)
	binary.LittleEndian.PutUint32(frame[16:20], uint32(len(payload)))
	copy(frame[20:], payload)
	return frame
}

func readIssue110Frame(conn net.Conn) ([]byte, error) {
	var lenBytes [4]byte
	if _, err := io.ReadFull(conn, lenBytes[:]); err != nil {
		return nil, err
	}
	length := int(binary.LittleEndian.Uint32(lenBytes[:]))
	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeIssue110Frame(conn net.Conn, payload []byte) {
	frame := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(frame[:4], uint32(len(payload)))
	copy(frame[4:], payload)
	_, _ = conn.Write(frame)
}

// TestDCSessionFallsBackToMainEndpointOn444 reproduces issue #110: the
// static media endpoint for DC 2 rejects req_DH_params with -444 while the
// main endpoint accepts the handshake. The media endpoint must be tried
// first, then abandoned for the main endpoint, and the rejection must be
// remembered so later sessions dial the main endpoint directly.
func TestDCSessionFallsBackToMainEndpointOn444(t *testing.T) {
	client, err := NewClient(1, "hash", &Config{TransportMode: TransportModeIntermediate, NoUpdates: true})
	if err != nil {
		t.Fatal(err)
	}
	setDCPoolClientConnected(t, client, 4) // bot home DC, like the reporter
	dialer := &issue110Dialer{rejectDH: map[string]bool{"149.154.167.222:443": true}}
	client.setTestDialer(dialer)
	_, generation := client.dcSessions.getInitLock(2)

	_, release, candErr := client.createDCSessionCandidate(context.Background(), 2, generation)
	if release != nil {
		release()
	}
	if candErr == nil {
		t.Fatal("candidate error = nil, want DH failure on both endpoints")
	}
	// The second attempt ends with EOF on the main endpoint, not another -444.
	if transport.IsTransportError(candErr, transport.ErrCodeInvalidDC) {
		t.Fatalf("candidate error = %v, want non-444 terminal error", candErr)
	}

	want := []string{
		"149.154.167.222:443", // media endpoint: ResPQ ok, req_DH -> -444
		"149.154.167.51:443",  // main endpoint: retried immediately
	}
	if got := dialer.calls(); !equalStrings(got, want) {
		t.Fatalf("dialed = %v, want %v", got, want)
	}
	if !client.dcSessions.isTransferKeygenRejected(2) {
		t.Fatal("media keygen rejection was not memoized for DC 2")
	}

	// A later session creation on the same DC must skip the dead endpoint.
	_, release2, candErr2 := client.createDCSessionCandidate(context.Background(), 2, generation)
	if release2 != nil {
		release2()
	}
	if candErr2 == nil {
		t.Fatal("second candidate error = nil, want EOF failure")
	}
	if got, want2 := dialer.calls(), []string{"149.154.167.222:443", "149.154.167.51:443", "149.154.167.51:443"}; !equalStrings(got, want2) {
		t.Fatalf("dialed after memo = %v, want %v", got, want2)
	}
}

// TestDCTransferCandidatesOrder covers endpoint selection for every
// production DC: media endpoint first when known and healthy, main endpoint
// always present, media endpoint dropped once rejected.
func TestDCTransferCandidatesOrder(t *testing.T) {
	client, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatal(err)
	}
	media := map[int]string{2: "149.154.167.222", 4: "149.154.165.96"}
	for dcID := 1; dcID <= 5; dcID++ {
		base := session.DataCenter{ID: dcID}
		candidates := client.dcTransferCandidates(dcID, base)
		if len(candidates) == 0 {
			t.Fatalf("DC %d: no candidates", dcID)
		}
		if got := candidates[len(candidates)-1].Address(); got != base.Address() {
			t.Fatalf("DC %d: last candidate = %q, want main %q", dcID, got, base.Address())
		}
		if addr, ok := media[dcID]; ok {
			if got := candidates[0].IPAddress; got != addr {
				t.Fatalf("DC %d: first candidate = %q, want media %q", dcID, got, addr)
			}
		} else if len(candidates) != 1 {
			t.Fatalf("DC %d: candidates = %d, want 1 (main only)", dcID, len(candidates))
		}

		client.dcSessions.markTransferKeygenRejected(dcID)
		candidates = client.dcTransferCandidates(dcID, base)
		if len(candidates) != 1 || candidates[0].IPAddress != "" {
			t.Fatalf("DC %d: after rejection candidates = %+v, want main only", dcID, candidates)
		}
	}
}

// TestDCSessionUnknownDC keeps the error contract for unknown DC ids.
func TestDCSessionUnknownDC(t *testing.T) {
	client, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatal(err)
	}
	candidates := client.dcTransferCandidates(99, session.DataCenter{ID: 99})
	if len(candidates) != 0 {
		t.Fatalf("candidates for unknown DC = %+v, want none", candidates)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
