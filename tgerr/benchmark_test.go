package tgerr

import (
	"errors"
	"testing"
)

// BenchmarkIs checks the performance of error string matching, which runs on
// every RPC response.
func BenchmarkIs(b *testing.B) {
	err := New(420, "FLOOD_WAIT_30")
	b.ReportAllocs()
	for b.Loop() {
		Is(err, "FLOOD_WAIT_30")
	}
}

func BenchmarkIsNotMatch(b *testing.B) {
	err := New(420, "FLOOD_WAIT_30")
	b.ReportAllocs()
	for b.Loop() {
		Is(err, "SESSION_PASSWORD_NEEDED")
	}
}

func BenchmarkIsNil(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		Is(nil, "FLOOD_WAIT_30")
	}
}

// BenchmarkIsVariadic benchmarks Is with multiple candidates.
func BenchmarkIsVariadic(b *testing.B) {
	err := New(420, "FLOOD_WAIT_30")
	b.ReportAllocs()
	for b.Loop() {
		Is(err, "SESSION_PASSWORD_NEEDED", "PHONE_NUMBER_BANNED", "FLOOD_WAIT_30")
	}
}

// BenchmarkNew benchmarks error construction (runs on every RPC error response).
func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		New(420, "FLOOD_WAIT_60")
	}
}

// BenchmarkAs checks errors.As performance for typed error extraction.
func BenchmarkAs(b *testing.B) {
	err := New(400, "ENCRYPTED_MESSAGE_INVALID")
	wrapped := errors.New("wrapper")
	combined := errors.Join(wrapped, err)
	b.ReportAllocs()
	for b.Loop() {
		var target *Error
		errors.As(combined, &target)
	}
}
