package crypto

import (
	"testing"
)

func TestDecompose(t *testing.T) {
	tests := []struct {
		pq    int64
		wantP int64
		wantQ int64
	}{
		{6, 2, 3},
		{15, 3, 5},
		{35, 5, 7},
		{77, 7, 11},
		{143, 11, 13},
		{221, 13, 17},
		{1000001, 101, 9901},
	}

	for _, tt := range tests {
		p := Decompose(tt.pq)
		if p != tt.wantP {
			t.Fatalf("Decompose(%d) = %d, want %d", tt.pq, p, tt.wantP)
		}
	}
}

func TestDecomposeEven(t *testing.T) {
	if Decompose(4) != 2 {
		t.Fatalf("Decompose(4) should return 2")
	}
	if Decompose(100) != 2 {
		t.Fatalf("Decompose(100) should return 2")
	}
}

func TestDecomposeLarge(t *testing.T) {
	p, q := int64(104729), int64(104743)
	pq := p * q
	factor := Decompose(pq)
	if factor != p {
		t.Fatalf("Decompose(%d) = %d, want %d", pq, factor, p)
	}
}

func TestCurrentDHPrime(t *testing.T) {
	if CurrentDHPrime.Sign() <= 0 {
		t.Fatal("CurrentDHPrime must be positive")
	}
	if CurrentDHPrime.BitLen() != 2048 {
		t.Fatalf("CurrentDHPrime bit length: got %d, want 2048", CurrentDHPrime.BitLen())
	}
}

func TestCurrentDHPrimeIsPrime(t *testing.T) {
	if !CurrentDHPrime.ProbablyPrime(20) {
		t.Fatal("CurrentDHPrime is not prime")
	}
}
