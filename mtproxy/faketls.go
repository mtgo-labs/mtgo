package mtproxy

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
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
	clientHelloLen      = 517

	maxTLSRecordPayload = 16384
)

var defaultCipherSuites = []byte{
	0x13, 0x01, 0x13, 0x02, 0x13, 0x03, 0xC0, 0x2B,
	0xC0, 0x2F, 0xC0, 0x2C, 0xC0, 0x30, 0xCC, 0xA9,
	0xCC, 0xA8, 0xC0, 0x13, 0xC0, 0x14, 0x00, 0x9C,
	0x00, 0x9D, 0x00, 0x2F, 0x00, 0x35, 0x01, 0x00,
}

var curve25519P = new(big.Int).Sub(
	new(big.Int).Exp(big.NewInt(2), big.NewInt(255), nil),
	big.NewInt(19),
)

func buildClientHello(secret []byte, domain string) ([]byte, []byte, error) {
	buf := make([]byte, clientHelloLen)
	pos := 0

	putUint16 := func(v uint16) { binary.BigEndian.PutUint16(buf[pos:], v); pos += 2 }
	putUint24 := func(v uint32) { buf[pos] = byte(v >> 16); buf[pos+1] = byte(v >> 8); buf[pos+2] = byte(v); pos += 3 }
	putByte := func(b byte) { buf[pos] = b; pos++ }

	putByte(tlsHandshakeType)
	putUint16(tlsVersion10)
	putUint16(512)

	putByte(0x01)
	putUint24(508)

	putUint16(tlsVersion12)

	pos += 32

	putByte(0x20)
	rand.Read(buf[pos : pos+32])
	pos += 32

	putUint16(32)
	copy(buf[pos:], defaultCipherSuites)
	pos += 32

	grease := make([]byte, 2)
	rand.Read(grease)
	grease[0] = (grease[0] & 0xF0) | 0x0A
	grease[1] = (grease[1] & 0xF0) | 0x0A
	if grease[1] == grease[0] {
		grease[1] ^= 0x10
	}
	copy(buf[pos:], grease)
	pos += 2

	putByte(0x01)
	putByte(0x00)
	putByte(0x00)
	putByte(0x00)

	extStart := pos
	putUint16(0)
	innerStart := pos
	putUint16(0)
	putByte(0x00)
	sniStart := pos
	putUint16(0)
	domainBytes := []byte(domain)
	copy(buf[pos:], domainBytes)
	pos += len(domainBytes)
	binary.BigEndian.PutUint16(buf[sniStart:], uint16(pos-sniStart-2))
	binary.BigEndian.PutUint16(buf[innerStart:], uint16(pos-innerStart-2))
	binary.BigEndian.PutUint16(buf[extStart:], uint16(pos-extStart-2))

	remaining := clientHelloLen - pos
	if remaining > 4 {
		putByte(0x00)
		putByte(0x17)
		binary.BigEndian.PutUint16(buf[pos:], uint16(remaining-4))
		pos += 2
		key := generateFakeKeyShare()
		copy(buf[pos:], key)
		pos += len(key)
	}

	if pos < clientHelloLen {
		padding := make([]byte, clientHelloLen-pos)
		rand.Read(padding)
		copy(buf[pos:], padding)
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(buf)
	digest := mac.Sum(nil)

	ts := uint32(time.Now().Unix())
	digest[28] ^= byte(ts)
	digest[29] ^= byte(ts >> 8)
	digest[30] ^= byte(ts >> 16)
	digest[31] ^= byte(ts >> 24)

	copy(buf[11:43], digest)

	clientRandom := make([]byte, 32)
	copy(clientRandom, digest)

	return buf, clientRandom, nil
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

func generateFakeKeyShare() []byte {
	for {
		key := make([]byte, 32)
		rand.Read(key)
		key[31] &= 127

		x := new(big.Int).SetBytes(key)
		y2 := curve25519Y2(x)

		if isQuadraticResidue(y2, curve25519P) {
			for i := 0; i < 3; i++ {
				x = curve25519DoubleX(x)
			}
			result := make([]byte, 32)
			xBytes := x.Bytes()
			copy(result[32-len(xBytes):], xBytes)
			return result
		}
	}
}

func curve25519Y2(x *big.Int) *big.Int {
	y := new(big.Int).Set(x)
	y.Mul(y, x)
	y.Mod(y, curve25519P)

	a := big.NewInt(486662)
	y.Add(y, a)
	y.Mul(y, x)
	y.Mod(y, curve25519P)

	y.Add(y, big.NewInt(1))
	y.Mul(y, x)
	y.Mod(y, curve25519P)

	return y
}

func isQuadraticResidue(a, p *big.Int) bool {
	exp := new(big.Int).Sub(p, big.NewInt(1))
	exp.Rsh(exp, 1)
	r := new(big.Int).Exp(a, exp, p)
	return r.Cmp(big.NewInt(1)) == 0
}

func curve25519DoubleX(x *big.Int) *big.Int {
	x2 := new(big.Int).Mul(x, x)
	x2.Mod(x2, curve25519P)

	num := new(big.Int).Sub(x2, big.NewInt(1))
	num.Mul(num, num)
	num.Mod(num, curve25519P)

	y2 := curve25519Y2(x)
	den := new(big.Int).Mul(y2, big.NewInt(4))
	den.Mod(den, curve25519P)

	denInv := new(big.Int).ModInverse(den, curve25519P)
	if denInv == nil {
		return big.NewInt(0)
	}

	result := new(big.Int).Mul(num, denInv)
	result.Mod(result, curve25519P)
	return result
}
