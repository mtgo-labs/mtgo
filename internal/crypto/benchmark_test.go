package crypto

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

var (
	benchAuthKey   [256]byte
	benchAuthKeyID [8]byte
	benchSessionID [8]byte
	benchSalt      int64 = 0x12345678
	benchMsgKey    [16]byte
	benchPlaintext []byte
	benchEncrypted []byte
)

func init() {
	rand.Read(benchAuthKey[:])
	rand.Read(benchAuthKeyID[:])
	rand.Read(benchSessionID[:])
	rand.Read(benchMsgKey[:])

	// Simulate a realistic message body (a Pong response, ~40 bytes).
	benchPlaintext = bytes.Repeat([]byte{0xAB}, 64)
}

// --- MTProto v2 Pack/Unpack (the most critical hot path) ---

func BenchmarkCryptoPack(b *testing.B) {
	msg := &tg.MTProtoMessage{
		MsgID: 123456,
		SeqNo: 1,
		Body:  &tg.Pong{MsgID: 123456, PingID: 42},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := Pack(msg, benchSalt, benchSessionID[:], benchAuthKey[:], benchAuthKeyID[:])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCryptoUnpack(b *testing.B) {
	// Pack always uses x=0 (outgoing), Unpack uses x=8 (incoming).
	// To benchmark Unpack, we need data encrypted with x=8, which
	// requires server-simulated data. Covered indirectly by Pack +
	// IGEDecrypt benchmarks.
	b.Skip("Unpack requires x=8 encrypted data (server-simulated)")
}

// --- KDF (called on every Pack and Unpack) ---

func BenchmarkKDF(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		KDF(benchAuthKey[:], benchMsgKey[:], true)
	}
}

// --- AES-IGE (called on every Pack and Unpack) ---

func BenchmarkIGEEncrypt(b *testing.B) {
	data := make([]byte, 256) // typical message size
	rand.Read(data)
	key := benchAuthKey[:32]
	iv := benchAuthKey[32:64]
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := IGEEncrypt(data, key, iv)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIGEDecrypt(b *testing.B) {
	data := make([]byte, 256)
	rand.Read(data)
	key := benchAuthKey[:32]
	iv := benchAuthKey[32:64]
	enc, _ := IGEEncrypt(data, key, iv)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := IGEDecrypt(enc, key, iv)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- E2E Secret Chat encrypt/decrypt ---

func BenchmarkSecretEncrypt(b *testing.B) {
	key := benchAuthKey[:]
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := SecretEncrypt(benchPlaintext, key, true)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSecretDecrypt(b *testing.B) {
	key := benchAuthKey[:]
	// Encrypt with outgoing=true (x=0); decrypt with outgoing=true (x=0)
	// to simulate the recipient decrypting an originator's message.
	enc, err := SecretEncrypt(benchPlaintext, key, true)
	if err != nil {
		b.Fatal(err)
	}
	benchEncrypted = enc
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := SecretDecrypt(benchEncrypted, key, true)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSecretKDF(b *testing.B) {
	key := benchAuthKey[:]
	b.ReportAllocs()
	for b.Loop() {
		secretKDF(key, benchMsgKey[:], 0)
	}
}

// --- PFS binding message KDF is in internal/session, skip here ---

// --- DH validation (called once per secret chat creation) ---

func BenchmarkValidateGA(b *testing.B) {
	ga := makeTestGA(currentDHPrime)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		ValidateGA(ga, currentDHPrime)
	}
}

func BenchmarkValidateDHPrime(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		ValidateDHPrime(currentDHPrime)
	}
}

// --- Key fingerprint and visualization (called once per key exchange) ---

func BenchmarkKeyFingerprint(b *testing.B) {
	key := benchAuthKey[:]
	b.ReportAllocs()
	for b.Loop() {
		KeyFingerprint(key)
	}
}

func BenchmarkKeyVisualization(b *testing.B) {
	key := benchAuthKey[:]
	b.ReportAllocs()
	for b.Loop() {
		KeyVisualization(key)
	}
}

func makeTestGA(prime *big.Int) *big.Int {
	// p/2 is safely within the valid range [2^1984, p - 2^1984].
	return new(big.Int).Rsh(prime, 1)
}
