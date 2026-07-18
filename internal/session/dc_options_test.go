package session

import (
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
