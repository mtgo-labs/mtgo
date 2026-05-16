package crypto

import (
	"crypto/aes"
	"crypto/cipher"
)

func xorInPlace(dst, a, b []byte) {
	for i := range a {
		dst[i] = a[i] ^ b[i]
	}
}

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
// Returns the encrypted ciphertext.
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
	iv1 := make([]byte, 16)
	iv2 := make([]byte, 16)
	copy(iv1, iv[:16])
	copy(iv2, iv[16:])

	result := make([]byte, len(data))
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
// Returns the decrypted plaintext.
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
	iv1 := make([]byte, 16)
	iv2 := make([]byte, 16)
	copy(iv1, iv[:16])
	copy(iv2, iv[16:])

	result := make([]byte, len(data))
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

	keystream := make([]byte, 16)
	block.Encrypt(keystream, ivCopy)

	out := make([]byte, len(data))
	pos := 0
	for i := 0; i < len(data); i++ {
		out[i] = data[i] ^ keystream[pos]
		pos++
		if pos >= 16 {
			pos = 0
			incrementIV(ivCopy)
			block.Encrypt(keystream, ivCopy)
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
	for i := 0; i < len(data); i++ {
		out[i] = data[i] ^ c.keystream[c.pos]
		c.pos++
		if c.pos >= 16 {
			c.pos = 0
			incrementIV(c.iv)
			c.block.Encrypt(c.keystream[:], c.iv)
		}
	}
	return out
}
