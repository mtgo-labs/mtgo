package telegram

import (
	"context"
	"errors"
	"math/big"
	"testing"

	mtcrypto "github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestEnableCloudPassword_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.EnableCloudPassword(context.Background(), "pw", "hint")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestChangeCloudPassword_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.ChangeCloudPassword(context.Background(), "old", "new", "hint")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestRemoveCloudPassword_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.RemoveCloudPassword(context.Background(), "pw")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestComputeCheckPasswordSRPDoesNotExtendSalt1(t *testing.T) {
	backing := make([]byte, 64)
	for i := range backing {
		backing[i] = byte(i + 1)
	}
	salt1 := backing[:16]
	tail := append([]byte(nil), backing[16:]...)

	// A valid in-range server B = g^k mod p.
	validB := new(big.Int).Exp(big.NewInt(3), big.NewInt(12345), mtcrypto.CurrentDHPrime)
	srpB := make([]byte, 256)
	bBytes := validB.Bytes()
	copy(srpB[256-len(bBytes):], bBytes)

	_, err := computeCheckPasswordSRP(&tg.AccountPassword{
		CurrentAlgo: &tg.PasswordKdfAlgoSha256sha256pbkdf2hmacsha512iter100000sha256modPow{
			Salt1: salt1,
			Salt2: []byte("server-salt-2"),
			G:     3,
			P:     mtcrypto.CurrentDHPrime.Bytes(),
		},
		SRPB:  srpB,
		SRPID: 1,
	}, "password")
	if err != nil {
		t.Fatalf("computeCheckPasswordSRP failed: %v", err)
	}

	for i, want := range tail {
		if got := backing[16+i]; got != want {
			t.Fatalf("salt1 backing data mutated at offset %d: got %d, want %d", 16+i, got, want)
		}
	}
}
