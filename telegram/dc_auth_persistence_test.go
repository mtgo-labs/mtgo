package telegram

import "testing"

func TestDCAuthKeyPersistenceRoundTrip(t *testing.T) {
	c, _ := NewClient(1, "test", nil)
	c.storage = NewMemoryStorage()

	if got := c.loadPersistedDCAuthKey(4); got != nil {
		t.Fatalf("empty store returned key: %+v", got)
	}

	key := []byte("dc4-permanent-auth-key-256-bytes-padded-0000000000000000")
	salt := int64(0xC0FFEE)
	c.persistDCAuthKey(4, key, salt)

	got := c.loadPersistedDCAuthKey(4)
	if got == nil {
		t.Fatal("persisted key not loaded")
	}
	if string(got.key) != string(key) || got.salt != salt {
		t.Fatalf("loaded = %+v, want key %q salt %d", got, key, salt)
	}

	// Tombstoning must make the loader treat the entry as absent.
	c.forgetPersistedDCAuthKey(4)
	if got := c.loadPersistedDCAuthKey(4); got != nil {
		t.Fatalf("tombstoned key still loaded: %+v", got)
	}
}

func TestDCAuthKeyReuseFromPersistedState(t *testing.T) {
	c, _ := NewClient(1, "test", nil)
	c.storage = NewMemoryStorage()

	key := []byte("persisted-dc5-key-0000000000000000000000000000000000")
	c.persistDCAuthKey(5, key, 77)

	// The in-memory map is empty (fresh client); the persisted key is
	// adopted as the first authorized key for the DC.
	info := c.dcSessions.dcAuthKey(5)
	if info != nil {
		t.Fatalf("in-memory key present before adoption: %+v", info)
	}
	adopted := c.loadPersistedDCAuthKey(5)
	if adopted == nil || string(adopted.key) != string(key) {
		t.Fatalf("adoption failed: %+v", adopted)
	}

	// A revocation drops both the in-memory key and the persisted one.
	c.dcSessions.setDCAuthKey(5, key, 77, 0)
	c.forgetPersistedDCAuthKey(5)
	c.dcSessions.forgetDCAuthKey(5)
	if c.dcSessions.dcAuthKey(5) != nil {
		t.Fatal("in-memory key not forgotten")
	}
	if c.loadPersistedDCAuthKey(5) != nil {
		t.Fatal("persisted key not forgotten")
	}
}

// The reuse branch must not adopt a persisted key for a different DC.
func TestDCAuthKeyLoadIsPerDC(t *testing.T) {
	c, _ := NewClient(1, "test", nil)
	c.storage = NewMemoryStorage()

	c.persistDCAuthKey(2, []byte("dc2-key"), 1)
	if got := c.loadPersistedDCAuthKey(3); got != nil {
		t.Fatalf("DC 3 adopted DC 2's key: %+v", got)
	}
}
