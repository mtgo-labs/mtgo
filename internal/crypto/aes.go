package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"sync"
)

var igeBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0, 4096)
		return &buf
	},
}

func getAESBuf(size int) []byte {
	bp := igeBufPool.Get().(*[]byte)
	buf := *bp
	if cap(buf) < size {
		buf = make([]byte, size)
	} else {
		buf = buf[:size]
	}
	*bp = buf
	return buf
}

// ReleaseAESBuf returns a buffer obtained from IGEEncrypt or IGEDecrypt
// back to the pool. Call this after the buffer is no longer needed.
func ReleaseAESBuf(buf []byte) {
	if buf == nil {
		return
	}
	igeBufPool.Put(&buf)
}

func xorInPlace(dst, a, b []byte) {
	for i := range a {
		dst[i] = a[i] ^ b[i]
	}
}

// TODO: newAESBlock creates a new cipher.Block on every call. Since the auth
// key is the same for every message in a session and cipher.Block is immutable
// and goroutine-safe, callers should cache the block at the session level
// (e.g. in Session.SetAuthKey) and pass it through to IGEEncrypt/IGEDecrypt
// instead of passing the raw key.
func newAESBlock(key []byte) cipher.Block {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic("crypto/aes: " + err.Error())
	}
	return block
}

// IGEEncrypt encrypts data using AES-256 in Infinite Garble Extension (IGE)
// mode. The data length must be a multiple of 16. The key must be 32 bytes and
// iv must be 32 bytes (split into two 16-byte halves for IGE chaining).
// Returns the encrypted ciphertext. The caller must call ReleaseAESBuf when
// the returned buffer is no longer needed.
//
// See https://core.telegram.org/mtproto/description#encrypted-message.
func IGEEncrypt(data, key, iv []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	if len(data)%16 != 0 {
		panic("crypto/aes: IGE data length must be multiple of 16")
	}
	block := newAESBlock(key)
	var iv1Buf, iv2Buf [16]byte
	iv1 := iv1Buf[:]
	iv2 := iv2Buf[:]
	copy(iv1, iv[:16])
	copy(iv2, iv[16:])

	result := getAESBuf(len(data))
	var xored, encrypted [16]byte
	for i := 0; i < len(data); i += 16 {
		chunk := data[i : i+16]
		xorInPlace(xored[:], chunk, iv1)
		block.Encrypt(encrypted[:], xored[:])
		xorInPlace(result[i:i+16], encrypted[:], iv2)
		iv1 = result[i : i+16]
		iv2 = chunk
	}
	return result
}

// IGEDecrypt decrypts data using AES-256 in Infinite Garble Extension (IGE)
// mode. The data length must be a multiple of 16. The key must be 32 bytes and
// iv must be 32 bytes (split into two 16-byte halves for IGE chaining).
// Returns the decrypted plaintext. The caller must call ReleaseAESBuf when
// the returned buffer is no longer needed.
//
// See https://core.telegram.org/mtproto/description#encrypted-message.
func IGEDecrypt(data, key, iv []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	if len(data)%16 != 0 {
		panic("crypto/aes: IGE data length must be multiple of 16")
	}
	block := newAESBlock(key)
	var iv1Buf, iv2Buf [16]byte
	iv1 := iv1Buf[:]
	iv2 := iv2Buf[:]
	copy(iv1, iv[:16])
	copy(iv2, iv[16:])

	result := getAESBuf(len(data))
	var xored, decrypted [16]byte
	for i := 0; i < len(data); i += 16 {
		chunk := data[i : i+16]
		xorInPlace(xored[:], chunk, iv2)
		block.Decrypt(decrypted[:], xored[:])
		xorInPlace(result[i:i+16], decrypted[:], iv1)
		iv1 = chunk
		iv2 = result[i : i+16]
	}
	return result
}

func incrementIV(iv []byte) {
	for i := 15; i >= 0; i-- {
		iv[i]++
		if iv[i] != 0 {
			break
		}
	}
}

// CTREncrypt encrypts data using AES-256 in counter (CTR) mode with a 16-byte
// IV that is incremented as a big-endian integer. Returns the encrypted ciphertext.
func CTREncrypt(data, key, iv []byte) []byte {
	return ctrCrypt(data, key, iv)
}

// CTRDecrypt decrypts data using AES-256 in counter (CTR) mode with a 16-byte
// IV that is incremented as a big-endian integer. Returns the decrypted plaintext.
func CTRDecrypt(data, key, iv []byte) []byte {
	return ctrCrypt(data, key, iv)
}

func ctrCrypt(data, key, iv []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	block := newAESBlock(key)

	ivCopy := make([]byte, 16)
	copy(ivCopy, iv)

	var keystream [16]byte
	block.Encrypt(keystream[:], ivCopy)

	out := make([]byte, len(data))
	pos := 0
	for i := 0; i < len(data); i++ {
		out[i] = data[i] ^ keystream[pos]
		pos++
		if pos >= 16 {
			pos = 0
			incrementIV(ivCopy)
			block.Encrypt(keystream[:], ivCopy)
		}
	}
	return out
}

// CTRCipher implements a stateful AES-256 CTR-mode stream cipher that can
// process data incrementally across multiple calls while maintaining the
// keystream position and IV counter.
type CTRCipher struct {
	block     cipher.Block
	iv        []byte
	keystream [16]byte
	pos       int
}

// NewCTRCipher creates a new CTRCipher initialized with the given 32-byte key
// and 16-byte IV. The cipher is ready to process data immediately.
func NewCTRCipher(key, iv []byte) *CTRCipher {
	block := newAESBlock(key)
	ivCopy := make([]byte, 16)
	copy(ivCopy, iv)
	c := &CTRCipher{
		block: block,
		iv:    ivCopy,
	}
	block.Encrypt(c.keystream[:], c.iv)
	return c
}

// Process XORs data with the CTR keystream, advancing the IV counter and
// keystream position as needed. It can be called multiple times with
// successive chunks of data; the cipher state is preserved between calls.
// Returns the resulting ciphertext or plaintext (CTR is symmetric).
func (c *CTRCipher) Process(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	out := make([]byte, len(data))
	c.ProcessTo(data, out)
	return out
}

// ProcessTo XORs data with the CTR keystream and writes the result into out.
// out must have length >= len(data). The cipher state is preserved between calls.
func (c *CTRCipher) ProcessTo(data []byte, out []byte) {
	for i := 0; i < len(data); i++ {
		out[i] = data[i] ^ c.keystream[c.pos]
		c.pos++
		if c.pos >= 16 {
			c.pos = 0
			incrementIV(c.iv)
			c.block.Encrypt(c.keystream[:], c.iv)
		}
	}
}
