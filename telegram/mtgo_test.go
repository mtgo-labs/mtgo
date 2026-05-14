package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestLayer(t *testing.T) {
	if tg.Layer != 225 {
		t.Fatalf("expected layer 225, got %d", tg.Layer)
	}
}

func TestRegistryNotEmpty(t *testing.T) {
	if len(tg.Registry) < 100 {
		t.Fatalf("expected at least 100 registered types, got %d", len(tg.Registry))
	}
}

func TestCoreTypesRegistered(t *testing.T) {
	required := []uint32{
		0x5BB8E511,
		0x73F1F8DC,
		0x3072CFA1,
		0xAE500895,
		0x0949D9DC,
		0xBC799737,
		0x997275B5,
		0x1CB5C415,
		0x62D6B459,
		0xF35C6D01,
		0x2144CA19,
		0x347773C5,
	}
	for _, id := range required {
		if _, ok := tg.Registry[id]; !ok {
			t.Fatalf("missing registry entry for 0x%08X", id)
		}
	}
}
