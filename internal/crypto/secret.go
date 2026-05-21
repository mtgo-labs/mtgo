package crypto

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	mathrand "math/rand"
)

var (
	errEmptyPlaintext     = errors.New("crypto/secret: empty plaintext")
	errCiphertextTooShort = errors.New("crypto/secret: ciphertext too short")
	errMsgKeyMismatch     = errors.New("crypto/secret: msg_key mismatch")
	errDecryptedTooShort  = errors.New("crypto/secret: decrypted too short")
	errInvalidMsgLen      = errors.New("crypto/secret: invalid message length")
	errFileNotAligned     = errors.New("crypto/secret: encrypted file not aligned to 16")
)

const (
	SecretChatKeyLen     = 256
	SecretChatMinGA      = 2048 - 64
	SecretChatLayer      = 46
	SecretChatMaxPadding = 1024
	SecretChatMinPadding = 12
)

var one = big.NewInt(1)

func GenerateDHSecret(dhPrime *big.Int, bits int) (*big.Int, error) {
	byteLen := bits / 8
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("crypto/secret: generate random: %w", err)
	}
	n := new(big.Int).SetBytes(b)
	primeMinus1 := new(big.Int).Sub(dhPrime, one)
	n.Mod(n, primeMinus1)
	n.Add(n, one)
	return n, nil
}

func ComputeGA(g int32, a *big.Int, dhPrime *big.Int) *big.Int {
	return new(big.Int).Exp(big.NewInt(int64(g)), a, dhPrime)
}

func ComputeSharedKey(gaOrB, secret *big.Int, dhPrime *big.Int) []byte {
	key := new(big.Int).Exp(gaOrB, secret, dhPrime).Bytes()
	if len(key) < SecretChatKeyLen {
		padded := make([]byte, SecretChatKeyLen)
		copy(padded[SecretChatKeyLen-len(key):], key)
		key = padded
	}
	return key[:SecretChatKeyLen]
}

func KeyFingerprint(key []byte) int64 {
	h := sha1.Sum(key)
	return int64(binary.LittleEndian.Uint64(h[12:20]))
}

func KeyVisualization(key []byte) []string {
	h1 := sha1.Sum(key)
	h2 := sha256.Sum256(key)
	combined := append(h1[:16], h2[:20]...)
	emoji := []string{
		"\U0001F60E", "\U0001F60A", "\U0001F60D", "\U0001F618",
		"\U0001F61C", "\U0001F60B", "\U0001F61A", "\U0001F638",
		"\U0001F431", "\U0001F436", "\U0001F433", "\U0001F427",
		"\U0001F42C", "\U0001F981", "\U0001F40E", "\U0001F43B",
		"\U0001F40D", "\U0001F42D", "\U0001F437", "\U0001F430",
		"\U0001F43C", "\U0001F42A", "\U0001F43D", "\U0001F435",
		"\U0001F612", "\U0001F60F", "\U0001F614", "\U0001F61E",
		"\U0001F616", "\U0001F625", "\U0001F630", "\U0001F628",
		"\U0001F623", "\U0001F62D", "\U0001F602", "\U0001F603",
		"\U0001F604", "\U0001F601", "\U0001F606", "\U0001F60B",
		"\U0001F609", "\U0001F61B", "\U0001F61D", "\U0001F631",
		"\U0001F620", "\U0001F621", "\U0001F624", "\U0001F633",
		"\U0001F62C", "\U0001F615", "\U0001F2E0", "\U0001F4A9",
		"\U0001F44D", "\U0001F44E", "\u2764\uFE0F", "\U0001F495",
		"\U0001F4A1", "\u2B50", "\U0001F525", "\U0001F4A8",
		"\U0001F4A6", "\U0001F3B5", "\U0001F4BB", "\U0001F517",
	}
	result := make([]string, 4)
	for i := 0; i < 4; i++ {
		idx := binary.BigEndian.Uint32(combined[i*4:]) % uint32(len(emoji))
		result[i] = emoji[idx]
	}
	return result
}

func ValidateGA(ga *big.Int, dhPrime *big.Int) bool {
	if ga.Cmp(one) <= 0 {
		return false
	}
	primeMinus1 := new(big.Int).Sub(dhPrime, one)
	if ga.Cmp(primeMinus1) >= 0 {
		return false
	}
	lowerBound := new(big.Int).Sub(dhPrime, new(big.Int).Lsh(one, uint(SecretChatMinGA)))
	return ga.Cmp(lowerBound) >= 0
}

func SecretEncrypt(plaintext, key []byte, outgoing bool) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errEmptyPlaintext
	}

	var buf bytes.Buffer
	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(plaintext)))
	buf.Write(lenBytes)
	buf.Write(plaintext)

	paddingLen := SecretChatMinPadding + mathrand.Intn(SecretChatMaxPadding-SecretChatMinPadding)
	if rem := paddingLen % 16; rem != 0 {
		paddingLen += 16 - rem
	}
	padding := make([]byte, paddingLen)
	rand.Read(padding)
	buf.Write(padding)

	data := buf.Bytes()

	x := 0
	if !outgoing {
		x = 8
	}

	var stackBuf [4096]byte
	msgKeyLargeInput := stackBuf[:0]
	if 32+len(data) > len(stackBuf) {
		msgKeyLargeInput = make([]byte, 32+len(data))
	} else {
		msgKeyLargeInput = stackBuf[:32+len(data)]
	}
	copy(msgKeyLargeInput, key[x+88:x+120])
	copy(msgKeyLargeInput[32:], data)
	msgKeyLarge := sha256.Sum256(msgKeyLargeInput)
	msgKey := msgKeyLarge[8:24]

	aesKey, aesIV := secretKDF(key, msgKey, x)
	encrypted := IGEEncrypt(data, aesKey[:], aesIV[:])

	var result bytes.Buffer
	result.Write(msgKey)
	result.Write(encrypted)
	ReleaseAESBuf(encrypted)
	return result.Bytes(), nil
}

func SecretDecrypt(ciphertext, key []byte, outgoing bool) ([]byte, error) {
	if len(ciphertext) < 16+16 {
		return nil, errCiphertextTooShort
	}

	msgKey := ciphertext[:16]
	encrypted := ciphertext[16:]

	x := 0
	if !outgoing {
		x = 8
	}

	aesKey, aesIV := secretKDF(key, msgKey, x)
	decrypted := IGEDecrypt(encrypted, aesKey[:], aesIV[:])

	var stackBuf [4096]byte
	msgKeyLargeInput := stackBuf[:0]
	if 32+len(decrypted) > len(stackBuf) {
		msgKeyLargeInput = make([]byte, 32+len(decrypted))
	} else {
		msgKeyLargeInput = stackBuf[:32+len(decrypted)]
	}
	copy(msgKeyLargeInput, key[x+88:x+120])
	copy(msgKeyLargeInput[32:], decrypted)
	msgKeyCheck := sha256.Sum256(msgKeyLargeInput)
	if subtle.ConstantTimeCompare(msgKey, msgKeyCheck[8:24]) != 1 {
		return nil, errMsgKeyMismatch
	}

	if len(decrypted) < 4 {
		return nil, errDecryptedTooShort
	}
	msgLen := int(binary.LittleEndian.Uint32(decrypted[:4]))
	if msgLen < 0 || msgLen+4 > len(decrypted) {
		return nil, errInvalidMsgLen
	}
	paddingLen := len(decrypted) - 4 - msgLen
	if paddingLen < SecretChatMinPadding || paddingLen > SecretChatMaxPadding {
		return nil, fmt.Errorf("crypto/secret: invalid padding length %d", paddingLen)
	}

	return decrypted[4 : 4+msgLen], nil
}

func secretKDF(key, msgKey []byte, x int) (aesKey, aesIV [32]byte) {
	var tmpA [52]byte
	copy(tmpA[:], msgKey)
	copy(tmpA[len(msgKey):], key[x:x+36])
	sha256A := sha256.Sum256(tmpA[:])

	var tmpB [52]byte
	copy(tmpB[:], key[x+40:x+76])
	copy(tmpB[36:], msgKey)
	sha256B := sha256.Sum256(tmpB[:])

	copy(aesKey[0:8], sha256A[:8])
	copy(aesKey[8:24], sha256B[8:24])
	copy(aesKey[24:32], sha256A[24:32])

	copy(aesIV[0:8], sha256B[:8])
	copy(aesIV[8:24], sha256A[8:24])
	copy(aesIV[24:32], sha256B[24:32])

	return aesKey, aesIV
}

func EncryptFile(data, fileKey, fileIV []byte) []byte {
	return IGEEncrypt(padFile(data), fileKey, fileIV)
}

func DecryptFile(data, fileKey, fileIV []byte) ([]byte, error) {
	if len(data)%16 != 0 {
		return nil, errFileNotAligned
	}
	return IGEDecrypt(data, fileKey, fileIV), nil
}

func FileKeyFingerprint(key, iv []byte) int32 {
	h := md5.Sum(append(key, iv...))
	fp := uint32(h[0])<<24 | uint32(h[1])<<16 | uint32(h[2])<<8 | uint32(h[3])
	fp ^= uint32(h[4])<<24 | uint32(h[5])<<16 | uint32(h[6])<<8 | uint32(h[7])
	return int32(fp)
}

func GenerateFileKeyIV() (key, iv []byte, err error) {
	key = make([]byte, 32)
	iv = make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		return nil, nil, err
	}
	if _, err = rand.Read(iv); err != nil {
		return nil, nil, err
	}
	return key, iv, nil
}

func padFile(data []byte) []byte {
	padding := 16 - (len(data) % 16)
	if padding == 0 {
		padding = 16
	}
	padded := make([]byte, len(data)+padding)
	copy(padded, data)
	return padded
}
