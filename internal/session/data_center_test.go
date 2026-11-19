package session

import "testing"

func TestDataCenterProductionAddresses(t *testing.T) {
	dc := DataCenter{ID: 2, TestMode: false}
	addr := dc.Address()
	port := dc.Port()
	if addr != "149.154.167.51" {
		t.Errorf("DC2 prod address = %q, want %q", addr, "149.154.167.51")
	}
	if port != 443 {
		t.Errorf("DC2 prod port = %d, want %d", port, 443)
	}
}

func TestDataCenterTestAddresses(t *testing.T) {
	dc := DataCenter{ID: 1, TestMode: true}
	addr := dc.Address()
	port := dc.Port()
	if addr != "149.154.175.10" {
		t.Errorf("DC1 test address = %q, want %q", addr, "149.154.175.10")
	}
	if port != 443 {
		t.Errorf("DC1 test port = %d, want %d", port, 443)
	}
}

func TestDataCenterIPv6(t *testing.T) {
	dc := DataCenter{ID: 2, TestMode: false, IPv6: true}
	addr := dc.Address()
	if addr != "2001:b28:f23d:f001::e" {
		t.Errorf("DC2 prod IPv6 = %q, want %q", addr, "2001:b28:f23d:f001::e")
	}
}

func TestDataCenterUnknownDC(t *testing.T) {
	dc := DataCenter{ID: 99, TestMode: false}
	addr := dc.Address()
	if addr != "" {
		t.Errorf("unknown DC address = %q, want empty", addr)
	}
}

func TestDataCenterString(t *testing.T) {
	dc := DataCenter{ID: 1, TestMode: false}
	if dc.String() != "DC1" {
		t.Errorf("String() = %q, want %q", dc.String(), "DC1")
	}
}

func TestDataCenterTransferPrefersMediaEndpoint(t *testing.T) {
	dc := DataCenter{ID: 2}
	transfer := dc.TransferDataCenter()
	if transfer.Address() != "149.154.167.222" {
		t.Errorf("DC2 transfer address = %q, want media %q", transfer.Address(), "149.154.167.222")
	}
	// DCs without a known media endpoint transfer over the main address.
	plain := DataCenter{ID: 1}.TransferDataCenter()
	if plain.Address() != "149.154.175.53" {
		t.Errorf("DC1 transfer address = %q, want main %q", plain.Address(), "149.154.175.53")
	}
}

// TestDataCenterTransferKeepsRuntimeAddress verifies that an address learned
// from a live help.getConfig is never overridden by the static media
// snapshot (issue #110: the static media IPs went stale server-side).
func TestDataCenterTransferKeepsRuntimeAddress(t *testing.T) {
	dc := DataCenter{ID: 2, IPAddress: "203.0.113.7", PortValue: 443}
	transfer := dc.TransferDataCenter()
	if transfer.Address() != "203.0.113.7" {
		t.Errorf("transfer address = %q, want runtime address %q", transfer.Address(), "203.0.113.7")
	}
}
