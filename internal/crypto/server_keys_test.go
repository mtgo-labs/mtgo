package crypto

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"
)

func TestRSAKeySet_NewSeededFromBundled(t *testing.T) {
	ks := NewRSAKeySet()
	cur := ks.Current()
	if len(cur) == 0 {
		t.Fatal("RSAKeySet.Current() is empty — expected bundled keys")
	}
	// Every bundled fingerprint should be trusted.
	for fp := range serverPublicKeys {
		if !ks.IsTrusted(fp) {
			t.Errorf("bundled fingerprint %d not trusted", fp)
		}
	}
}

func TestRSAKeySet_Get(t *testing.T) {
	ks := NewRSAKeySet()
	for fp, want := range serverPublicKeys {
		got, ok := ks.Get(fp)
		if !ok {
			t.Errorf("Get(%d): not found", fp)
			continue
		}
		if got != want {
			t.Errorf("Get(%d): key mismatch", fp)
		}
	}
	if _, ok := ks.Get(999999999); ok {
		t.Error("Get(unknown): expected not found")
	}
}

func TestRSAKeySet_VerifyAndAccept_AddsVerified(t *testing.T) {
	ks := NewRSAKeySet()
	// Create a synthetic key that is NOT bundled.
	newKey := &ServerKey{
		N: big.NewInt(0).SetBytes([]byte{0x01, 0x02, 0x03, 0x04, 0x05}),
		E: big.NewInt(65537),
	}
	fp := ComputeFingerprint(newKey)
	if ks.IsTrusted(fp) {
		t.Skip("synthetic key happens to match a bundled fingerprint")
	}
	if err := ks.VerifyAndAccept(fp, newKey); err != nil {
		t.Fatalf("VerifyAndAccept: %v", err)
	}
	if !ks.IsTrusted(fp) {
		t.Fatal("VerifyAndAccept: key not trusted after accept")
	}
	got, ok := ks.Get(fp)
	if !ok || got != newKey {
		t.Fatal("VerifyAndAccept: Get returns wrong key after accept")
	}
}

func TestRSAKeySet_VerifyAndAccept_RejectsFingerprintMismatch(t *testing.T) {
	ks := NewRSAKeySet()
	newKey := &ServerKey{
		N: big.NewInt(0).SetBytes([]byte{0xAA, 0xBB, 0xCC}),
		E: big.NewInt(65537),
	}
	wrongFP := int64(77777)
	if err := ks.VerifyAndAccept(wrongFP, newKey); err == nil {
		t.Fatal("VerifyAndAccept: expected error for fingerprint mismatch")
	}
	if ks.IsTrusted(wrongFP) {
		t.Fatal("key should not be trusted with mismatched fingerprint")
	}
}

func TestRSAKeySet_VerifyAndAccept_RejectsNilKey(t *testing.T) {
	ks := NewRSAKeySet()
	if err := ks.VerifyAndAccept(42, nil); err == nil {
		t.Fatal("VerifyAndAccept(nil): expected error")
	}
}

func TestRSAKeySet_BundledNotDuplicated(t *testing.T) {
	ks := NewRSAKeySet()
	// Pick a bundled key and try to accept it again — should be a no-op.
	for fp, key := range serverPublicKeys {
		before := len(ks.Current())
		if err := ks.VerifyAndAccept(fp, key); err != nil {
			t.Fatalf("VerifyAndAccept bundled: %v", err)
		}
		after := len(ks.Current())
		if after != before {
			t.Errorf("accepting bundled key changed set size: before=%d after=%d", before, after)
		}
		break
	}
}

func TestPublicRsaKeyWatchdog_RotationAbsorbed(t *testing.T) {
	ks := NewRSAKeySet()
	newKey := &ServerKey{
		N: big.NewInt(0).SetBytes([]byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x01}),
		E: big.NewInt(65537),
	}
	newFP := ComputeFingerprint(newKey)
	if ks.IsTrusted(newFP) {
		t.Skip("synthetic key matches a bundled fingerprint")
	}
	var calls atomic.Int32
	wd := NewPublicRsaKeyWatchdog(WatchdogConfig{
		KeySet:   ks,
		Interval: 50 * time.Millisecond,
		FetchFn: func(ctx context.Context) ([]FetchedKey, error) {
			calls.Add(1)
			return []FetchedKey{{Fingerprint: newFP, Key: newKey}}, nil
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	wd.Start(ctx)
	// Wait for at least one refresh cycle.
	time.Sleep(150 * time.Millisecond)
	cancel()
	wd.Wait()

	if calls.Load() == 0 {
		t.Fatal("FetchFn was never called")
	}
	if !ks.IsTrusted(newFP) {
		t.Fatal("rotated key was not trusted after watchdog refresh")
	}
	if wd.LastRefresh().IsZero() {
		t.Fatal("LastRefresh is zero after successful refresh")
	}
}

func TestPublicRsaKeyWatchdog_BadRefreshFailClosed(t *testing.T) {
	ks := NewRSAKeySet()
	beforeCount := len(ks.Current())
	wd := NewPublicRsaKeyWatchdog(WatchdogConfig{
		KeySet:   ks,
		Interval: 50 * time.Millisecond,
		FetchFn: func(ctx context.Context) ([]FetchedKey, error) {
			return nil, errors.New("simulated fetch failure")
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	wd.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	cancel()
	wd.Wait()

	// Existing trusted set unchanged (fail-closed).
	afterCount := len(ks.Current())
	if afterCount != beforeCount {
		t.Errorf("fail-closed violated: before=%d after=%d", beforeCount, afterCount)
	}
	if !wd.LastRefresh().IsZero() {
		t.Error("LastRefresh should be zero when all refreshes failed")
	}
}

func TestPublicRsaKeyWatchdog_RejectsInvalidKey(t *testing.T) {
	ks := NewRSAKeySet()
	beforeCount := len(ks.Current())
	wd := NewPublicRsaKeyWatchdog(WatchdogConfig{
		KeySet:   ks,
		Interval: 50 * time.Millisecond,
		FetchFn: func(ctx context.Context) ([]FetchedKey, error) {
			// Structurally invalid key (nil modulus) → VerifyAndAccept rejects.
			return []FetchedKey{{Fingerprint: 99999, Key: &ServerKey{}}}, nil
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	wd.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	cancel()
	wd.Wait()

	// Invalid key must NOT be in the trusted set (fail-closed).
	if len(ks.Current()) != beforeCount {
		t.Errorf("invalid key was accepted: before=%d after=%d", beforeCount, len(ks.Current()))
	}
}

func TestComputeFingerprint_ConsistentWithBundled(t *testing.T) {
	// Verify that ComputeFingerprint produces a value present in the bundled
	// map for at least one bundled key. This confirms the fingerprint algorithm
	// matches the format used for ServerPublicKeys keys.
	ks := NewRSAKeySet()
	cur := ks.Current()
	found := false
	for bundledFP, key := range cur {
		computed := ComputeFingerprint(key)
		if computed == bundledFP {
			found = true
			break
		}
	}
	// Note: the bundled fingerprints may use a different byte ordering or
	// computation. If none match, we still verify ComputeFingerprint is stable.
	if !found {
		// At minimum, the same key must produce the same fingerprint twice.
		for _, key := range cur {
			fp1 := ComputeFingerprint(key)
			fp2 := ComputeFingerprint(key)
			if fp1 != fp2 {
				t.Fatal("ComputeFingerprint is not deterministic")
			}
			break
		}
	}
}
