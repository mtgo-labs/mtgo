package crypto

import (
	"math/big"
	"testing"
)

func TestServerKeyCount(t *testing.T) {
	if len(ServerPublicKeys) < 9 {
		t.Fatalf("expected >= 9 server public keys, got %d", len(ServerPublicKeys))
	}
}

func TestServerKeyFingerprints(t *testing.T) {
	fingerprints := []int64{
		-4344800451088585951,
		847625836280919973,
		1562291298945373506,
		-5859577972006586033,
		6491968696586960280,
		-7395192255793472640,
		2685959930972952888,
		-3997872768018684475,
		-4960899639492471258,
	}
	for _, fp := range fingerprints {
		key, ok := ServerPublicKeys[fp]
		if !ok {
			t.Fatalf("missing fingerprint %d", fp)
		}
		if key.N == nil || key.E == nil {
			t.Fatalf("key for fingerprint %d has nil N or E", fp)
		}
		if key.N.Sign() <= 0 {
			t.Fatalf("key.N for fingerprint %d is not positive", fp)
		}
	}
}

func TestServerKeyModulusSize(t *testing.T) {
	for fp, key := range ServerPublicKeys {
		if key.N.BitLen() != 2048 {
			t.Fatalf("key %d: expected 2048-bit modulus, got %d", fp, key.N.BitLen())
		}
		if key.E.Cmp(big.NewInt(65537)) != 0 {
			t.Fatalf("key %d: expected exponent 65537, got %s", fp, key.E)
		}
	}
}

func TestGetServerKey(t *testing.T) {
	_, ok := GetServerKey(-4344800451088585951)
	if !ok {
		t.Fatal("expected to find key for fingerprint -4344800451088585951")
	}

	_, ok = GetServerKey(999999999999)
	if ok {
		t.Fatal("expected no key for unknown fingerprint")
	}
}
