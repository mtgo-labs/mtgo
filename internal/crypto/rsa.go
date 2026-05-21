package crypto

import (
	cryptorand "crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"math/big"
)

var (
	ErrUnknownFingerprint = errors.New("crypto/rsa: unknown server public key fingerprint")
	ErrDataTooLong        = errors.New("crypto/rsa: data exceeds 144 bytes")
)

const (
	rsaPadDataLimit    = 144
	dataWithPaddingLen = 192
	dataWithHashLen    = 224
	tempKeyLen         = 32
	rsaEncryptedLen    = 256
	rsaLegacyBlockLen  = 255
)

func reverseBytes(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

func rsapad(data []byte, n *big.Int) ([]byte, error) {
	if len(data) > rsaPadDataLimit {
		return nil, ErrDataTooLong
	}

	dataWithPadding := make([]byte, dataWithPaddingLen)
	copy(dataWithPadding, data)
	if _, err := cryptorand.Read(dataWithPadding[len(data):]); err != nil {
		return nil, err
	}

	dataPadReversed := make([]byte, dataWithPaddingLen)
	copy(dataPadReversed, dataWithPadding)
	reverseBytes(dataPadReversed)

	zeroIV := make([]byte, 32)

	var tempKey [tempKeyLen]byte
	var hashInput [tempKeyLen + dataWithPaddingLen]byte
	var dataWithHash [dataWithHashLen]byte
	var tempKeyXor [tempKeyLen]byte
	var result [rsaEncryptedLen]byte

	for {
		if _, err := cryptorand.Read(tempKey[:]); err != nil {
			return nil, err
		}

		copy(hashInput[:], tempKey[:])
		copy(hashInput[tempKeyLen:], dataWithPadding)
		hash := sha256.Sum256(hashInput[:])

		copy(dataWithHash[:], dataPadReversed)
		copy(dataWithHash[dataWithPaddingLen:], hash[:])

		aesEncrypted, err := IGEEncrypt(dataWithHash[:], tempKey[:], zeroIV)
			if err != nil {
				return nil, err
			}

		aesHash := sha256.Sum256(aesEncrypted)
		for i := 0; i < tempKeyLen; i++ {
			tempKeyXor[i] = tempKey[i] ^ aesHash[i]
		}

		copy(result[:], tempKeyXor[:])
		copy(result[tempKeyLen:], aesEncrypted)
		ReleaseAESBuf(aesEncrypted)

		resultInt := new(big.Int).SetBytes(result[:])
		if resultInt.Cmp(n) < 0 {
			return result[:], nil
		}
	}
}

func RSAEncrypt(data []byte, fingerprint int64) ([]byte, error) {
	key, ok := GetServerKey(fingerprint)
	if !ok {
		return nil, ErrUnknownFingerprint
	}

	padded, err := rsapad(data, key.N)
	if err != nil {
		return nil, err
	}

	m := new(big.Int).SetBytes(padded)
	c := new(big.Int).Exp(m, key.E, key.N)

	result := make([]byte, rsaEncryptedLen)
	cBytes := c.Bytes()
	copy(result[rsaEncryptedLen-len(cBytes):], cBytes)

	return result, nil
}

func RSAEncryptLegacy(data []byte, fingerprint int64) ([]byte, error) {
	key, ok := GetServerKey(fingerprint)
	if !ok {
		return nil, ErrUnknownFingerprint
	}

	hash := sha1.Sum(data)
	block := make([]byte, rsaLegacyBlockLen)
	copy(block, hash[:])
	copy(block[sha1.Size:], data)

	m := new(big.Int).SetBytes(block)
	c := new(big.Int).Exp(m, key.E, key.N)

	result := make([]byte, rsaEncryptedLen)
	c.FillBytes(result)
	return result, nil
}
