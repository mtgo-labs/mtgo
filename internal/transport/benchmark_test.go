package transport

import (
	"crypto/rand"
	"encoding/binary"
	"testing"
)

// --- Transport framing: length encoding benchmarks (no network I/O) ---

func BenchmarkAbridgedEncodeLength(b *testing.B) {
	data := make([]byte, 256)
	rand.Read(data)
	b.ReportAllocs()
	for b.Loop() {
		length := len(data) / 4
		if length <= 126 {
			_ = []byte{byte(length)}
		} else {
			var header [4]byte
			header[0] = 0x7f
			header[1] = byte(length)
			header[2] = byte(length >> 8)
			header[3] = byte(length >> 16)
			_ = header
		}
	}
}

func BenchmarkIntermediateEncodeLength(b *testing.B) {
	data := make([]byte, 256)
	rand.Read(data)
	b.ReportAllocs()
	for b.Loop() {
		var header [4]byte
		binary.LittleEndian.PutUint32(header[:], uint32(len(data)))
		_ = header
	}
}

// --- Transport error detection ---

func BenchmarkDetectTransportError4Byte(b *testing.B) {
	data := make([]byte, 4)
	data[3] = 0xFF // negative = transport error
	b.ReportAllocs()
	for b.Loop() {
		DetectTransportError(data)
	}
}

func BenchmarkDetectTransportErrorLarge(b *testing.B) {
	data := make([]byte, 1024)
	rand.Read(data)
	b.ReportAllocs()
	for b.Loop() {
		DetectTransportError(data)
	}
}

func BenchmarkIsQuickAckToken(b *testing.B) {
	data := make([]byte, 4)
	data[3] = 0x80 // bit 31 set
	b.ReportAllocs()
	for b.Loop() {
		IsQuickAckToken(data)
	}
}

// --- Padded intermediate payload trimming ---

func BenchmarkTrimPaddedIntermediate(b *testing.B) {
	data := make([]byte, 256)
	rand.Read(data)
	b.ReportAllocs()
	for b.Loop() {
		trimPaddedIntermediatePayload(data)
	}
}

// --- Obfuscated nonce forbidden check ---

func BenchmarkIsForbiddenNonce(b *testing.B) {
	nonce := make([]byte, 64)
	rand.Read(nonce)
	b.ReportAllocs()
	for b.Loop() {
		isForbiddenNonce(nonce)
	}
}
