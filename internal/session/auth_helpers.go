package session

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"time"
)

func randomInt128() ([16]byte, error) {
	var buf [16]byte
	_, err := io.ReadFull(cryptorand.Reader, buf[:])
	return buf, err
}

func randomInt256() ([32]byte, error) {
	var buf [32]byte
	_, err := io.ReadFull(cryptorand.Reader, buf[:])
	return buf, err
}

func sha1Hash(data []byte) []byte {
	h := sha1.Sum(data)
	return h[:]
}

func xorBytes(a, b []byte) []byte {
	n := min(len(b), len(a))
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = a[i] ^ b[i]
	}
	return out
}

func computeKeyAndIV(newNonce, serverNonce []byte) (key, iv []byte) {
	hash1 := sha1Hash(append(newNonce, serverNonce...))
	hash2 := sha1Hash(append(serverNonce, newNonce...))
	hash3 := sha1Hash(append(newNonce, newNonce...))

	key = make([]byte, 32)
	copy(key[0:20], hash1)
	copy(key[20:32], hash2[0:12])

	iv = make([]byte, 32)
	copy(iv[0:8], hash2[12:20])
	copy(iv[8:28], hash3)
	copy(iv[28:32], newNonce[0:4])

	return key, iv
}

func computeNewNonceHash1(newNonce []byte, authKey []byte) []byte {
	authKeyHash := sha1Hash(authKey)
	buf := make([]byte, len(newNonce)+1+8)
	copy(buf, newNonce)
	buf[len(newNonce)] = 1
	copy(buf[len(newNonce)+1:], authKeyHash[:8])
	h := sha1Hash(buf)
	return h[4:20]
}

func unwrapDataWithHash(dataWithHash []byte) ([]byte, error) {
	if len(dataWithHash) < sha1.Size {
		return nil, fmt.Errorf("data with hash too short: %d", len(dataWithHash))
	}

	hash := dataWithHash[:sha1.Size]
	for padding := 0; padding < 16; padding++ {
		end := len(dataWithHash) - padding
		if end < sha1.Size {
			break
		}
		data := dataWithHash[sha1.Size:end]
		if bytes.Equal(sha1Hash(data), hash) {
			return data, nil
		}
	}

	return nil, fmt.Errorf("SHA1 hash mismatch")
}

func computeAuthKey(gA, b, dhPrime *big.Int) []byte {
	k := new(big.Int).Exp(gA, b, dhPrime)
	return k.Bytes()
}

func computeServerSalt(newNonce, serverNonce []byte) int64 {
	xored := xorBytes(newNonce[:8], serverNonce[:8])
	return int64(binary.LittleEndian.Uint64(xored))
}

func packUnencrypted(w io.Writer, payload []byte) error {
	now := time.Now()
	unixSec := now.Unix()
	nanoFrac := now.Nanosecond() &^ 3
	msgID := (unixSec << 32) | int64(nanoFrac)
	header := make([]byte, 8+8+4)
	binary.LittleEndian.PutUint64(header[0:8], 0)
	binary.LittleEndian.PutUint64(header[8:16], uint64(msgID))
	binary.LittleEndian.PutUint32(header[16:20], uint32(len(payload)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func unpackUnencrypted(data []byte) ([]byte, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("unencrypted packet too short: %d", len(data))
	}
	authKeyID := binary.LittleEndian.Uint64(data[0:8])
	if authKeyID != 0 {
		return nil, fmt.Errorf("expected auth_key_id=0, got %d", authKeyID)
	}
	msgLen := binary.LittleEndian.Uint32(data[16:20])
	if int(20+msgLen) > len(data) {
		return nil, fmt.Errorf("message length %d exceeds packet size %d", msgLen, len(data)-20)
	}
	return data[20 : 20+msgLen], nil
}
