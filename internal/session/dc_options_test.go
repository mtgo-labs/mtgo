package session

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestDCOptionPoolResetHealth(t *testing.T) {
	pool := NewDCOptionPool(2, time.Hour)
	option := DataCenter{ID: 2}
	pool.AddOption(option)
	pool.RecordFailure(option)
	if _, err := pool.Candidates(1); err != ErrAllEndpointsFailing {
		t.Fatalf("Candidates before reset = %v, want ErrAllEndpointsFailing", err)
	}

	pool.ResetHealth()
	candidates, err := pool.Candidates(1)
	if err != nil {
		t.Fatalf("Candidates after reset: %v", err)
	}
	if len(candidates) != 1 || candidates[0] != option {
		t.Fatalf("candidates after reset = %+v", candidates)
	}
}

// familyUnreachableError builds an error shaped like a real failed dial on
// an unroutable family: the kernel rejects it instantly and net wraps it in
// OpError/SyscallError, then the dialer wraps it once more.
func familyUnreachableError(t *testing.T) error {
	t.Helper()
	return fmt.Errorf("dial [2001:b28:f23d:f001::a]:443: %w", &net.OpError{
		Op:   "dial",
		Net:  "tcp6",
		Addr: &net.TCPAddr{IP: net.ParseIP("2001:b28:f23d:f001::a"), Port: 443},
		Err:  os.NewSyscallError("connect", syscall.ENETUNREACH),
	})
}

func TestIsFamilyUnreachableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"net-unreachable", familyUnreachableError(t), true},
		{"host-unreachable", os.NewSyscallError("connect", syscall.EHOSTUNREACH), true},
		{"family-unsupported", os.NewSyscallError("socket", syscall.EAFNOSUPPORT), true},
		{"refused", os.NewSyscallError("connect", syscall.ECONNREFUSED), false},
		{"handshake", errors.New("transport handshake: server error code: 444"), false},
		{"nil-error", nil, false},
	}
	for _, tt := range tests {
		if got := isFamilyUnreachableError(tt.err); got != tt.want {
			t.Errorf("%s: isFamilyUnreachableError = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestDCOptionPoolFamilySuppression(t *testing.T) {
	v4 := DataCenter{ID: 1}
	v6 := DataCenter{ID: 1, IPv6: true, IPAddress: "2001:b28:f23d:f001::a"}

	// A cool-down of 0 keeps failed endpoints retryable so the family
	// filter is the only thing excluding candidates; the last-resort case
	// below opts into a real cool-down.
	newPool := func(coolDown time.Duration) *DCOptionPool {
		pool := NewDCOptionPool(1, coolDown)
		pool.AddOption(v4)
		pool.AddOption(v6)
		return pool
	}

	// A routing-level dial failure on the IPv6 endpoint suppresses the whole
	// IPv6 family while IPv4 still has candidates.
	pool := newPool(0)
	pool.RecordDialFailure(v6, familyUnreachableError(t))
	got, err := pool.CandidatesForDC(1, 0)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 1 || got[0] != v4 {
		t.Fatalf("candidates = %v, want only %v", got, v4)
	}

	// Endpoint-level failures (connection refused, handshake errors) do not
	// suppress the family.
	pool = newPool(0)
	pool.RecordDialFailure(v4, os.NewSyscallError("connect", syscall.ECONNREFUSED))
	got, err = pool.CandidatesForDC(1, 0)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("candidates = %v, want both families", got)
	}

	// Last resort: the IPv4 endpoint is in failure cool-down and the only
	// eligible endpoint left belongs to the suppressed family — it is still
	// returned rather than nothing.
	pool = newPool(time.Hour)
	v6b := DataCenter{ID: 1, IPv6: true, IPAddress: "2001:b28:f23d:f001::e"}
	pool.AddOption(v6b)
	pool.RecordDialFailure(v6, familyUnreachableError(t))
	pool.RecordFailure(v4)
	got, err = pool.CandidatesForDC(1, 0)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 1 || got[0] != v6b {
		t.Fatalf("candidates = %v, want last-resort %v", got, v6b)
	}

	// After the probe interval passes, the suppressed family is retried.
	pool = newPool(0)
	pool.RecordDialFailure(v6, familyUnreachableError(t))
	pool.mu.Lock()
	pool.familyUnreachable[true] = time.Now().Add(-pool.familyProbeInterval - time.Second)
	pool.mu.Unlock()
	got, err = pool.CandidatesForDC(1, 0)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("candidates = %v, want both families after probe interval", got)
	}

	// A network change clears family suppression along with endpoint health.
	pool = newPool(0)
	pool.RecordDialFailure(v6, familyUnreachableError(t))
	pool.ResetHealth()
	got, err = pool.CandidatesForDC(1, 0)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("candidates = %v, want both families after ResetHealth", got)
	}
}
