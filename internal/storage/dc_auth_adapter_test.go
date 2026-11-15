package storage

import "testing"

func TestAdapterWrapperDCAuthPassthrough(t *testing.T) {
	var w Storage = NewAdapter(newMemoryAdapter())
	ds, ok := w.(DCAuthStore)
	if !ok {
		t.Fatal("adapter wrapper does not expose the DC-auth extension")
	}
	entry := DCAuthEntry{DCID: 4, AuthKey: []byte("k"), ServerSalt: 9}
	if err := ds.SaveDCAuth(entry); err != nil {
		t.Fatalf("SaveDCAuth: %v", err)
	}
	got, err := ds.LoadDCAuth(4)
	if err != nil {
		t.Fatalf("LoadDCAuth: %v", err)
	}
	if got.DCID != 4 || string(got.AuthKey) != "k" || got.ServerSalt != 9 {
		t.Fatalf("roundtrip = %+v", got)
	}
}
