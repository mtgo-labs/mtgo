package mtgo

import (
	"runtime"
	"strings"
	"testing"
)

// TestToolchainNotGo1268 fails the test suite when it runs under the go1.26.8
// toolchain.
//
// go1.26.8 has a GC regression: mark-phase stack scanning of goroutines with
// cgo callback frames dies ("found pointer to free object" / SIGSEGV in
// runtime.scanstack), crashing any process hosting a cgo library that calls
// back into Go. go1.26.7 is the last known-good patch release. This guard
// keeps builds from silently floating onto the broken toolchain until the fix
// ships upstream. Full analysis, traces, and mitigation:
// issues/go1.26.8-gc-crash.md
func TestToolchainNotGo1268(t *testing.T) {
	// runtime.Version() is e.g. "go1.26.7", "go1.26.8 X:cgocheck2", or
	// "devel ...". Trim any experiment suffix, then match the exact broken
	// release — never 1.26.7, 1.26.8rc1 (untested, not observed broken), or
	// later patches.
	v := strings.SplitN(runtime.Version(), " ", 2)[0]
	if v != "go1.26.8" {
		return
	}

	t.Fatalf("running under go1.26.8, which crashes in the GC with cgo callback threads\n" +
		"(see issues/go1.26.8-gc-crash.md). Use the last known-good toolchain instead:\n" +
		"  GOTOOLCHAIN=go1.26.7 go test ./...\n" +
		"or just run `make test`, which pins the toolchain. Lift this guard only\n" +
		"after the upstream fix ships and a cgocheck2 + clobberfree soak passes.")
}
