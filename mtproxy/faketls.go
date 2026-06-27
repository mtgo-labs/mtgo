package mtproxy

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
)

const (
	tlsRecordHeaderLen  = 5
	tlsHandshakeType    = 0x16
	tlsAppDataType      = 0x17
	tlsChangeCipherSpec = 0x14
	tlsVersion10        = 0x0301
	tlsVersion12        = 0x0303
	tlsVersion13        = 0x0304

	maxTLSRecordPayload = 16384
)

// Chrome 131 cipher suites (without GREASE — injected separately).
var chromeCipherSuites = []uint16{
	0x1301, // TLS_AES_128_GCM_SHA256
	0x1302, // TLS_AES_256_GCM_SHA384
	0x1303, // TLS_CHACHA20_POLY1305_SHA256
	0xC02B, // TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
	0xC02F, // TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
	0xC02C, // TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
	0xC030, // TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
	0xCCA9, // TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
	0xCCA8, // TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
	0xC013, // TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA
	0xC014, // TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA
	0x009C, // TLS_RSA_WITH_AES_128_GCM_SHA256
	0x009D, // TLS_RSA_WITH_AES_256_GCM_SHA384
	0x002F, // TLS_RSA_WITH_AES_128_CBC_SHA
	0x0035, // TLS_RSA_WITH_AES_256_CBC_SHA
}

// Chrome signature algorithms (preference order).
var chromeSigAlgs = []uint16{
	0x0403, // ecdsa_secp256r1_sha256
	0x0804, // rsa_pss_rsae_sha256
	0x0404, // ecdsa_secp384r1_sha384
	0x0805, // rsa_pss_rsae_sha384
	0x0503, // ecdsa_secp521r1_sha512
	0x0806, // rsa_pss_rsae_sha512
	0x0807, // ed25519
	0x0809, // rsa_pss_pss_sha256
	0x080A, // rsa_pss_pss_sha384
	0x080B, // rsa_pss_pss_sha512
	0x0601, // rsa_pkcs1_sha512
	0x0501, // rsa_pkcs1_sha384
	0x0E07, // ed448
	0x0603, // rsa_pkcs1_sha256
	0x0401, // ecdsa_sha1
	0x0501, // (dup-safe, Chrome sends this list)
	0x0201, // rsa_pkcs1_sha1
}

// Chrome certificate signature algorithms (subset).
var chromeCertSigAlgs = []uint16{
	0x0403, 0x0804, 0x0404, 0x0805, 0x0503, 0x0806, 0x0601, 0x0501, 0x0603,
}

// Chrome supported groups (named curves).
var chromeSupportedGroups = []uint16{
	0x001D, // x25519
	0x0017, // secp256r1
	0x0018, // secp384r1
}

// greaseValues are the reserved GREASE code points used to prevent
// middlebox ossification. Chrome picks one at random per category.
var greaseValues = []uint16{
	0x0A0A, 0x1A1A, 0x2A2A, 0x3A3A, 0x4A4A, 0x5A5A,
	0x6A6A, 0x7A7A, 0x8A8A, 0x9A9A, 0xAAAA, 0xBABA,
	0xCACA, 0xDADA, 0xEAEA,
}

func randomGrease() uint16 {
	var b [2]byte
	rand.Read(b[:])
	return greaseValues[int(b[0])%len(greaseValues)]
}

// buildClientHello constructs a Chrome 131-like TLS ClientHello for the
// FakeTLS transport. The MTProxy digest is injected into the ClientRandom
// field so the proxy can authenticate the connection. The structure —
// cipher suites, extensions, GREASE values — matches what Chrome sends so
// DPI systems cannot distinguish the ClientHello from a real browser.
func buildClientHello(secret []byte, domain string) ([]byte, []byte, error) {
	// Pick distinct GREASE values per category (Chrome behaviour).
	greaseCipher := randomGrease()
	greaseExt := randomGrease()
	greaseGroup := randomGrease()
	greaseVersion := randomGrease()
	greaseKeyShare := randomGrease()

	// Build the ClientHello body (everything after the 4-byte handshake header).
	var body bytes.Buffer

	// client_version: TLS 1.2 (legacy field; real version is in supported_versions).
	body.Write([]byte{0x03, 0x03})

	// random: 32 bytes zeroed — will be overwritten with the MTProxy digest.
	randomOffset := body.Len()
	body.Write(make([]byte, 32))

	// session_id: 32 random bytes (Chrome uses 32).
	sessionID := make([]byte, 32)
	rand.Read(sessionID)
	body.WriteByte(0x20) // length
	body.Write(sessionID)

	// cipher_suites: GREASE + Chrome list + GREASE.
	var ciphers bytes.Buffer
	writeUint16(&ciphers, greaseCipher)
	for _, cs := range chromeCipherSuites {
		writeUint16(&ciphers, cs)
	}
	writeUint16(&body, uint16(ciphers.Len()))
	body.Write(ciphers.Bytes())

	// compression_methods: null only.
	body.WriteByte(0x01) // length
	body.WriteByte(0x00) // null

	// ---- Extensions ----
	var exts bytes.Buffer

	// GREASE extension (first, matching Chrome).
	writeUint16(&exts, greaseExt)
	writeUint16(&exts, 0) // zero-length

	// server_name (SNI).
	domainBytes := []byte(domain)
	var sniInner bytes.Buffer
	writeUint16(&sniInner, 0x0001) // host_name type
	writeUint16(&sniInner, uint16(len(domainBytes)))
	sniInner.Write(domainBytes)
	var sniList bytes.Buffer
	writeUint16(&sniList, uint16(sniInner.Len()))
	sniList.Write(sniInner.Bytes())
	writeExtension(&exts, 0x0000, sniList.Bytes())

	// extended_master_secret (empty).
	writeExtension(&exts, 0x0017, nil)

	// renegotiation_info (1 byte: 0x00).
	writeExtension(&exts, 0xFF01, []byte{0x00})

	// supported_groups: GREASE + x25519 + secp256r1 + secp384r1.
	var groups bytes.Buffer
	writeUint16(&groups, greaseGroup)
	for _, g := range chromeSupportedGroups {
		writeUint16(&groups, g)
	}
	var groupList bytes.Buffer
	writeUint16(&groupList, uint16(groups.Len()))
	groupList.Write(groups.Bytes())
	writeExtension(&exts, 0x000A, groupList.Bytes())

	// ec_point_formats: uncompressed.
	writeExtension(&exts, 0x000B, []byte{0x01, 0x00})

	// session_ticket (empty — no prior ticket).
	writeExtension(&exts, 0x0023, nil)

	// ALPN: h2, http/1.1.
	writeExtension(&exts, 0x0010, []byte{
		0x00, 0x06, // ALPN extension list length
		0x02, 'h', '2', // h2
		0x08, 'h', 't', 't', 'p', '/', '1', '.', '1', // http/1.1
	})

	// status_request (OCSP stapling): empty responder_id_list.
	writeExtension(&exts, 0x0005, []byte{0x01, 0x00, 0x00, 0x00, 0x00})

	// signature_algorithms_cert.
	var certSigAlgs bytes.Buffer
	writeUint16(&certSigAlgs, uint16(len(chromeCertSigAlgs)*2))
	for _, sa := range chromeCertSigAlgs {
		writeUint16(&certSigAlgs, sa)
	}
	writeExtension(&exts, 0x0032, certSigAlgs.Bytes())

	// signature_algorithms.
	var sigAlgs bytes.Buffer
	writeUint16(&sigAlgs, uint16(len(chromeSigAlgs)*2))
	for _, sa := range chromeSigAlgs {
		writeUint16(&sigAlgs, sa)
	}
	writeExtension(&exts, 0x000D, sigAlgs.Bytes())

	// key_share: GREASE + x25519 (32 random bytes).
	var ks bytes.Buffer
	// GREASE entry.
	writeUint16(&ks, greaseKeyShare)
	writeUint16(&ks, 1) // 1-byte share
	ks.WriteByte(0x00)
	// x25519 entry.
	writeUint16(&ks, 0x001D) // x25519
	writeUint16(&ks, 32)     // 32-byte share
	keyShare := make([]byte, 32)
	rand.Read(keyShare)
	ks.Write(keyShare)
	var ksList bytes.Buffer
	writeUint16(&ksList, uint16(ks.Len()))
	ksList.Write(ks.Bytes())
	writeExtension(&exts, 0x0033, ksList.Bytes())

	// PSK key exchange modes: psk_dhe_ke (1).
	writeExtension(&exts, 0x002D, []byte{0x01, 0x01})

	// supported_versions: GREASE + TLS 1.3 + TLS 1.2.
	var versions bytes.Buffer
	writeUint16(&versions, greaseVersion)
	writeUint16(&versions, tlsVersion13) // 0x0304
	writeUint16(&versions, tlsVersion12) // 0x0303
	var versionsList bytes.Buffer
	writeUint16(&versionsList, uint16(versions.Len()))
	versionsList.Write(versions.Bytes())
	writeExtension(&exts, 0x002B, versionsList.Bytes())

	// compressed_certificate: brotli (0x0002).
	writeExtension(&exts, 0x001B, []byte{
		0x00, 0x02, // algorithm list length
		0x00, 0x02, // brotli
	})

	// Trailing GREASE extension.
	writeUint16(&exts, greaseExt)
	writeUint16(&exts, 0)

	// Write extensions block into body.
	writeUint16(&body, uint16(exts.Len()))
	body.Write(exts.Bytes())

	// Assemble the full handshake message.
	handshakeLen := body.Len()
	var hello bytes.Buffer
	hello.Grow(tlsRecordHeaderLen + 4 + handshakeLen)

	// Record header.
	hello.WriteByte(tlsHandshakeType) // 0x16
	hello.WriteByte(0x03)             // TLS 1.0 (legacy record version)
	hello.WriteByte(0x01)
	writeUint16(&hello, uint16(4+handshakeLen)) // record payload length

	// Handshake header.
	hello.WriteByte(0x01) // ClientHello
	hello.WriteByte(byte(handshakeLen >> 16))
	hello.WriteByte(byte(handshakeLen >> 8))
	hello.WriteByte(byte(handshakeLen))

	// Handshake body.
	hello.Write(body.Bytes())

	out := hello.Bytes()

	// Inject MTProxy digest into ClientRandom.
	// The random field starts at: record header (5) + handshake header (4) +
	// client_version (2) = offset 11.
	mac := hmac.New(sha256.New, secret)
	mac.Write(out)
	digest := mac.Sum(nil)

	ts := uint32(time.Now().Unix())
	digest[28] ^= byte(ts)
	digest[29] ^= byte(ts >> 8)
	digest[30] ^= byte(ts >> 16)
	digest[31] ^= byte(ts >> 24)

	copy(out[randomOffset+tlsRecordHeaderLen+4:], digest)

	clientRandom := make([]byte, 32)
	copy(clientRandom, digest)

	return out, clientRandom, nil
}

func writeUint16(buf *bytes.Buffer, v uint16) {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], v)
	buf.Write(b[:])
}

func writeExtension(buf *bytes.Buffer, extType uint16, data []byte) {
	writeUint16(buf, extType)
	writeUint16(buf, uint16(len(data)))
	buf.Write(data)
}

func verifyServerHello(secret, clientRandom, response []byte) error {
	if len(response) < 74 {
		return fmt.Errorf("mtproxy: server hello too short (%d bytes)", len(response))
	}

	serverRandom := make([]byte, 32)
	copy(serverRandom, response[11:43])

	for i := 11; i < 43; i++ {
		response[i] = 0
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(clientRandom)
	mac.Write(response)
	expected := mac.Sum(nil)

	if !hmac.Equal(serverRandom, expected) {
		return ErrServerHelloFailed
	}

	return nil
}

type tlsConn struct {
	conn       net.Conn
	enc        *crypto.CTRCipher
	dec        *crypto.CTRCipher
	firstWrite bool
}

func fakeTLSHandshake(conn net.Conn, secret []byte, domain string) (*tlsConn, error) {
	hello, clientRandom, err := buildClientHello(secret, domain)
	if err != nil {
		return nil, fmt.Errorf("mtproxy: build client hello: %w", err)
	}

	if _, err := conn.Write(hello); err != nil {
		return nil, fmt.Errorf("mtproxy: write client hello: %w", err)
	}

	header := make([]byte, tlsRecordHeaderLen)
	if _, err := readFull(conn, header); err != nil {
		return nil, fmt.Errorf("mtproxy: read server hello header: %w", err)
	}
	if header[0] != tlsHandshakeType {
		return nil, fmt.Errorf("mtproxy: expected handshake record, got 0x%02x", header[0])
	}
	recordLen := int(binary.BigEndian.Uint16(header[3:5]))

	serverHello := make([]byte, recordLen)
	if _, err := readFull(conn, serverHello); err != nil {
		return nil, fmt.Errorf("mtproxy: read server hello: %w", err)
	}

	fullServer := make([]byte, len(header)+len(serverHello))
	copy(fullServer, header)
	copy(fullServer[tlsRecordHeaderLen:], serverHello)

	if err := verifyServerHello(secret, clientRandom, fullServer); err != nil {
		conn.Close()
		return nil, err
	}

	ccs := []byte{0x14, 0x03, 0x03, 0x00, 0x01, 0x01}
	if _, err := conn.Write(ccs); err != nil {
		return nil, fmt.Errorf("mtproxy: write change cipher spec: %w", err)
	}

	encKey := make([]byte, 32)
	encIV := make([]byte, 16)
	rand.Read(encKey)
	rand.Read(encIV)

	decKey := make([]byte, 32)
	decIV := make([]byte, 16)
	rand.Read(decKey)
	rand.Read(decIV)

	enc, err := crypto.NewCTRCipher(encKey, encIV)
	if err != nil {
		return nil, fmt.Errorf("mtproxy: create enc cipher: %w", err)
	}
	dec, err := crypto.NewCTRCipher(decKey, decIV)
	if err != nil {
		return nil, fmt.Errorf("mtproxy: create dec cipher: %w", err)
	}

	return &tlsConn{
		conn:       conn,
		enc:        enc,
		dec:        dec,
		firstWrite: true,
	}, nil
}

func (t *tlsConn) Read(p []byte) (int, error) {
	header := make([]byte, tlsRecordHeaderLen)
	if _, err := readFull(t.conn, header); err != nil {
		return 0, err
	}

	for header[0] == tlsChangeCipherSpec {
		recordLen := int(binary.BigEndian.Uint16(header[3:5]))
		discard := make([]byte, recordLen)
		if _, err := readFull(t.conn, discard); err != nil {
			return 0, err
		}
		if _, err := readFull(t.conn, header); err != nil {
			return 0, err
		}
	}

	if header[0] != tlsAppDataType {
		return 0, fmt.Errorf("mtproxy: unexpected TLS record type 0x%02x", header[0])
	}

	recordLen := int(binary.BigEndian.Uint16(header[3:5]))
	if recordLen > maxTLSRecordPayload {
		return 0, fmt.Errorf("mtproxy: TLS record too large (%d)", recordLen)
	}

	data := make([]byte, recordLen)
	if _, err := readFull(t.conn, data); err != nil {
		return 0, err
	}

	decrypted := t.dec.Process(data)
	return copy(p, decrypted), nil
}

func (t *tlsConn) Write(p []byte) (int, error) {
	var prefix []byte
	if t.firstWrite {
		prefix = []byte{0x14, 0x03, 0x03, 0x00, 0x01, 0x01}
		t.firstWrite = false
	}

	encrypted := t.enc.Process(p)

	buf := make([]byte, 0, len(prefix)+tlsRecordHeaderLen+len(encrypted))
	buf = append(buf, prefix...)
	record := make([]byte, tlsRecordHeaderLen)
	record[0] = tlsAppDataType
	record[1] = 0x03
	record[2] = 0x03
	binary.BigEndian.PutUint16(record[3:], uint16(len(encrypted)))
	buf = append(buf, record...)
	buf = append(buf, encrypted...)

	if _, err := t.conn.Write(buf); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (t *tlsConn) Close() error                       { return t.conn.Close() }
func (t *tlsConn) LocalAddr() net.Addr                { return t.conn.LocalAddr() }
func (t *tlsConn) RemoteAddr() net.Addr               { return t.conn.RemoteAddr() }
func (t *tlsConn) SetDeadline(d time.Time) error      { return t.conn.SetDeadline(d) }
func (t *tlsConn) SetReadDeadline(d time.Time) error  { return t.conn.SetReadDeadline(d) }
func (t *tlsConn) SetWriteDeadline(d time.Time) error { return t.conn.SetWriteDeadline(d) }

func readFull(conn net.Conn, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		nn, err := conn.Read(buf[n:])
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
