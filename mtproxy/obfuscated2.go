package mtproxy

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
)

const obfuscatedHeaderLen = 64

type obfuscatedConn struct {
	conn net.Conn
	enc  *crypto.CTRCipher
	dec  *crypto.CTRCipher
}

func obfuscated2Handshake(conn net.Conn, secret []byte, dcID int, codec byte) (*obfuscatedConn, error) {
	header := make([]byte, obfuscatedHeaderLen)

	for {
		rand.Read(header)
		if header[0] == 0xEF {
			continue
		}
		v := binary.BigEndian.Uint32(header[:4])
		if v == 0x44415441 || v == 0x504F5354 || v == 0x47455420 ||
			v == 0x4F505449 || v == 0xDDDDDDDD || v == 0xEEEEEEEE {
			continue
		}
		if binary.LittleEndian.Uint32(header[4:8]) == 0x00000000 {
			continue
		}
		break
	}

	encKeyInput := make([]byte, 32+16)
	copy(encKeyInput, header[8:40])
	copy(encKeyInput[32:], secret)
	encKeyHash := sha256.Sum256(encKeyInput)
	encKey := encKeyHash[:]
	encIV := make([]byte, 16)
	copy(encIV, header[40:56])

	reversed := make([]byte, 48)
	for i := 0; i < 48; i++ {
		reversed[i] = header[55-i]
	}
	decKeyInput := make([]byte, 32+16)
	copy(decKeyInput, reversed[:32])
	copy(decKeyInput[32:], secret)
	decKeyHash := sha256.Sum256(decKeyInput)
	decKey := decKeyHash[:]
	decIV := make([]byte, 16)
	copy(decIV, reversed[32:48])

	enc, err := crypto.NewCTRCipher(encKey, encIV)
	if err != nil {
		return nil, fmt.Errorf("mtproxy: create enc cipher: %w", err)
	}
	dec, err := crypto.NewCTRCipher(decKey, decIV)
	if err != nil {
		return nil, fmt.Errorf("mtproxy: create dec cipher: %w", err)
	}

	header[56] = codec
	header[57] = codec
	header[58] = codec
	header[59] = codec
	binary.LittleEndian.PutUint16(header[60:62], uint16(int16(dcID)))

	encrypted := enc.Process(header[56:64])
	copy(header[56:64], encrypted)

	if _, err := conn.Write(header); err != nil {
		return nil, fmt.Errorf("mtproxy: write obfuscated header: %w", err)
	}

	return &obfuscatedConn{conn: conn, enc: enc, dec: dec}, nil
}

func (o *obfuscatedConn) Read(p []byte) (int, error) {
	buf := make([]byte, len(p))
	n, err := io.ReadFull(o.conn, buf)
	if err != nil {
		return n, err
	}
	decrypted := o.dec.Process(buf[:n])
	copy(p, decrypted)
	return n, nil
}

func (o *obfuscatedConn) Write(p []byte) (int, error) {
	encrypted := o.enc.Process(p)
	if _, err := o.conn.Write(encrypted); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (o *obfuscatedConn) Close() error                       { return o.conn.Close() }
func (o *obfuscatedConn) LocalAddr() net.Addr                { return o.conn.LocalAddr() }
func (o *obfuscatedConn) RemoteAddr() net.Addr               { return o.conn.RemoteAddr() }
func (o *obfuscatedConn) SetDeadline(d time.Time) error      { return o.conn.SetDeadline(d) }
func (o *obfuscatedConn) SetReadDeadline(d time.Time) error  { return o.conn.SetReadDeadline(d) }
func (o *obfuscatedConn) SetWriteDeadline(d time.Time) error { return o.conn.SetWriteDeadline(d) }

// Dial connects to an MTProxy server and returns a net.Conn ready for
// MTProto traffic. The secret is parsed from hex and the connection is
// established with the appropriate obfuscation (dd: obfuscated2 only,
// ee: fake TLS + obfuscated2, simple: obfuscated2 only).
func Dial(addr string, secretHex string, dcID int, timeout time.Duration) (net.Conn, error) {
	secret, err := ParseSecret(secretHex)
	if err != nil {
		return nil, err
	}
	return DialSecret(addr, secret, dcID, timeout)
}

// DialSecret connects to an MTProxy server using a parsed Secret.
func DialSecret(addr string, secret Secret, dcID int, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("mtproxy: dial %s: %w", addr, err)
	}

	return Handshake(conn, secret, dcID)
}

// Handshake performs the MTProxy handshake on an already-connected TCP conn.
func Handshake(conn net.Conn, secret Secret, dcID int) (net.Conn, error) {
	underlying := conn

	if secret.NeedsFakeTLS() {
		domain := secret.Domain
		if domain == "" {
			domain = "www.google.com"
		}
		tlsConn, err := fakeTLSHandshake(conn, secret.Secret, domain)
		if err != nil {
			conn.Close()
			return nil, err
		}
		underlying = tlsConn
	}

	obfs, err := obfuscated2Handshake(underlying, secret.Secret, dcID, secret.Codec())
	if err != nil {
		if underlying != conn {
			underlying.Close()
		}
		conn.Close()
		return nil, err
	}

	return obfs, nil
}
