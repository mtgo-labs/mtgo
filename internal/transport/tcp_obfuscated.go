package transport

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/mtgo-labs/mtgo/internal/crypto"
)

var forbiddenFirstInts = []uint32{
	0xdddddddd,
	0xeeeeeeee,
	0x44414548,
	0x54534f50,
	0x20544547,
	0x4954504f,
	0x02010316,
}

func isForbiddenNonce(nonce []byte) bool {
	if nonce[0] == 0xef {
		return true
	}
	firstInt := binary.LittleEndian.Uint32(nonce[0:4])
	for _, v := range forbiddenFirstInts {
		if firstInt == v {
			return true
		}
	}
	secondInt := binary.LittleEndian.Uint32(nonce[4:8])
	if secondInt == 0 {
		return true
	}
	return false
}

// TCPObfuscated wraps an inner Transport with AES-CTR encryption to hide
// the transport fingerprint from middleboxes. It negotiates encryption
// keys via a random 64-byte nonce exchanged during Connect and then
// encrypts/decrypts all subsequent traffic transparently.
type TCPObfuscated struct {
	inner   Transport
	conn    net.Conn
	enc     *crypto.CTRCipher
	dec     *crypto.CTRCipher
	marker  byte
	nonce   []byte
	reverse bool
	readBuf []byte
}

// NewTCPObfuscated returns a new obfuscated transport wrapping inner. The
// marker byte identifies the inner transport type and is embedded in the
// nonce (e.g. 0xEF for abridged, 0xEE for intermediate).
func NewTCPObfuscated(inner Transport, marker byte) *TCPObfuscated {
	return &TCPObfuscated{
		inner:  inner,
		marker: marker,
	}
}

// NewTCPObfuscatedWithNonce returns a new obfuscated transport using a
// pre-established nonce. When reverse is true the encrypt/decrypt keys are
// swapped so that this side acts as the responder (party B) in the
// obfuscated handshake.
func NewTCPObfuscatedWithNonce(inner Transport, marker byte, nonce []byte, reverse bool) *TCPObfuscated {
	return &TCPObfuscated{
		inner:   inner,
		marker:  marker,
		nonce:   nonce,
		reverse: reverse,
	}
}

// Connect performs the obfuscated transport handshake. It generates (or uses
// the provided) 64-byte nonce, derives AES-CTR encrypt/decrypt keys, and
// writes the encrypted nonce to the peer. For reverse connections (responder
// side) it skips writing the nonce and only sets up the cipher state.
func (t *TCPObfuscated) Connect() error {
	if t.reverse {
		return t.connectReverse()
	}

	var nonce []byte
	if t.nonce != nil {
		nonce = make([]byte, 64)
		copy(nonce, t.nonce)
	} else {
		nonce = make([]byte, 64)
		for {
			rand.Read(nonce)
			if !isForbiddenNonce(nonce) {
				break
			}
		}
	}

	nonce[56] = t.marker
	nonce[57] = t.marker
	nonce[58] = t.marker
	nonce[59] = t.marker

	var encKey [32]byte
	var encIV [16]byte
	var decKey [32]byte
	var decIV [16]byte
	copy(encKey[:], nonce[8:40])
	copy(encIV[:], nonce[40:56])

	var reversed [48]byte
	for i := 0; i < 48; i++ {
		reversed[i] = nonce[55-i]
	}
	copy(decKey[:], reversed[0:32])
	copy(decIV[:], reversed[32:48])

	var err error
	t.enc, err = crypto.NewCTRCipher(encKey[:], encIV[:])
	if err != nil {
		return fmt.Errorf("tcp_obfuscated: create enc cipher: %w", err)
	}
	t.dec, err = crypto.NewCTRCipher(decKey[:], decIV[:])
	if err != nil {
		return fmt.Errorf("tcp_obfuscated: create dec cipher: %w", err)
	}

	encrypted := t.enc.Process(nonce)
	copy(nonce[56:64], encrypted[56:64])

	t.nonce = nonce

	t.conn = t.getInnerConn()

	if _, err := t.conn.Write(nonce); err != nil {
		return fmt.Errorf("tcp_obfuscated: write nonce: %w", err)
	}

	return nil
}

func (t *TCPObfuscated) connectReverse() error {
	nonce := t.nonce

	var reversed [48]byte
	for i := 0; i < 48; i++ {
		reversed[i] = nonce[55-i]
	}
	var encKey, decKey [32]byte
	var encIV, decIV [16]byte
	copy(encKey[:], reversed[0:32])
	copy(encIV[:], reversed[32:48])

	copy(decKey[:], nonce[8:40])
	copy(decIV[:], nonce[40:56])

	var err error
	t.enc, err = crypto.NewCTRCipher(encKey[:], encIV[:])
	if err != nil {
		return fmt.Errorf("tcp_obfuscated: create enc cipher: %w", err)
	}
	t.dec, err = crypto.NewCTRCipher(decKey[:], decIV[:])
	if err != nil {
		return fmt.Errorf("tcp_obfuscated: create dec cipher: %w", err)
	}

	t.dec.Process(make([]byte, 64))

	t.conn = t.getInnerConn()
	return nil
}

func (t *TCPObfuscated) getInnerConn() net.Conn {
	switch inner := t.inner.(type) {
	case *TCPIntermediate:
		return inner.conn
	case *TCPAbridged:
		return inner.conn
	case *TCPFull:
		return inner.conn
	}
	return nil
}

// Send encrypts buf with the AES-CTR cipher and writes it to the connection
// using the inner transport's framing format (abridged or intermediate).
// Returns an error if the inner transport type is unsupported.
func (t *TCPObfuscated) Send(buf *bytes.Buffer) error {
	data := buf.Bytes()

	switch inner := t.inner.(type) {
	case *TCPIntermediate:
		var header [4]byte
		binary.LittleEndian.PutUint32(header[:], uint32(len(data)))
		encHeader := t.enc.Process(header[:])
		encData := t.enc.Process(data)
		if _, err := t.conn.Write(encHeader); err != nil {
			return fmt.Errorf("tcp_obfuscated: send: %w", err)
		}
		if _, err := t.conn.Write(encData); err != nil {
			return fmt.Errorf("tcp_obfuscated: send: %w", err)
		}
	case *TCPAbridged:
		length := len(data) / 4
		if length <= 126 {
			h := [1]byte{byte(length)}
			encHeader := t.enc.Process(h[:])
			encData := t.enc.Process(data)
			if _, err := t.conn.Write(encHeader); err != nil {
				return fmt.Errorf("tcp_obfuscated: send: %w", err)
			}
			if _, err := t.conn.Write(encData); err != nil {
				return fmt.Errorf("tcp_obfuscated: send: %w", err)
			}
		} else {
			var header [4]byte
			header[0] = 0x7f
			header[1] = byte(length)
			header[2] = byte(length >> 8)
			header[3] = byte(length >> 16)
			encHeader := t.enc.Process(header[:])
			encData := t.enc.Process(data)
			if _, err := t.conn.Write(encHeader); err != nil {
				return fmt.Errorf("tcp_obfuscated: send: %w", err)
			}
			if _, err := t.conn.Write(encData); err != nil {
				return fmt.Errorf("tcp_obfuscated: send: %w", err)
			}
		}
	default:
		_ = inner
		return ErrUnsupportedTransport
	}
	return nil
}

// Recv reads the next framed message from the connection, decrypts it with
// the AES-CTR cipher, and returns the plaintext payload. Returns an error if
// the inner transport type is unsupported.
func (t *TCPObfuscated) Recv() ([]byte, error) {
	switch t.inner.(type) {
	case *TCPIntermediate:
		var lenBytes [4]byte
		if _, err := io.ReadFull(t.conn, lenBytes[:]); err != nil {
			return nil, err
		}
		decLen := t.dec.Process(lenBytes[:])
		length := binary.LittleEndian.Uint32(decLen)

		if length > uint32(MaxPayloadLen) {
			return nil, ErrPayloadTooLarge
		}
		if cap(t.readBuf) < int(length) {
			t.readBuf = make([]byte, length)
		}
		data := t.readBuf[:length]
		if _, err := io.ReadFull(t.conn, data); err != nil {
			return nil, err
		}
		return t.dec.Process(data), nil

	case *TCPAbridged:
		var lenByte [1]byte
		if _, err := io.ReadFull(t.conn, lenByte[:]); err != nil {
			return nil, err
		}
		decLen := t.dec.Process(lenByte[:])

		var length int
		if decLen[0] == 0x7f {
			var extLen [3]byte
			if _, err := io.ReadFull(t.conn, extLen[:]); err != nil {
				return nil, err
			}
			decExt := t.dec.Process(extLen[:])
			length = int(decExt[0]) | int(decExt[1])<<8 | int(decExt[2])<<16
		} else {
			length = int(decLen[0])
		}

		length *= 4
		if length > MaxPayloadLen {
			return nil, ErrPayloadTooLarge
		}
		if cap(t.readBuf) < length {
			t.readBuf = make([]byte, length)
		}
		data := t.readBuf[:length]
		if _, err := io.ReadFull(t.conn, data); err != nil {
			return nil, err
		}
		return t.dec.Process(data), nil

	default:
		return nil, ErrUnsupportedTransport
	}
}
