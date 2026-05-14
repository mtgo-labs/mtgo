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
