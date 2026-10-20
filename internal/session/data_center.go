package session

import "fmt"

// DataCenter represents a Telegram data center identified by its numeric ID,
// with optional test mode and IPv6 flags.
type DataCenter struct {
	// ID is the numeric data center identifier (1–5).
	ID int
	// TestMode indicates whether to use test server addresses.
	TestMode bool
	// IPv6 indicates whether to use IPv6 addresses.
	IPv6 bool
	// IPAddress overrides the built-in Telegram address table when populated
	// from help.getConfig.
	IPAddress string
	// PortValue overrides the default Telegram port when populated from
	// help.getConfig.
	PortValue int
}

// Address returns the IP address (IPv4 or IPv6) of this data center based on
// its ID, test mode, and IPv6 flags.
func (dc DataCenter) Address() string {
	if dc.IPAddress != "" {
		return dc.IPAddress
	}
	if dc.IPv6 {
		return dcIPv6Address(dc.ID, dc.TestMode)
	}
	return dcIPv4Address(dc.ID, dc.TestMode)
}

// Port returns the port number for connecting to this data center.
func (dc DataCenter) Port() int {
	if dc.PortValue > 0 {
		return dc.PortValue
	}
	return 443
}

// MediaAddress returns the media-only transfer endpoint for the DC, if one
// is known. Empty string means no dedicated media endpoint exists and the
// main address should be used.
func (dc DataCenter) MediaAddress() string {
	if dc.TestMode {
		return testMediaIPv4[dc.ID]
	}
	return prodMediaIPv4[dc.ID]
}

// TransferDataCenter returns the DataCenter to use for file transfers: the
// media-only endpoint when known, otherwise the main endpoint.
func (dc DataCenter) TransferDataCenter() DataCenter {
	if addr := dc.MediaAddress(); addr != "" {
		t := dc
		t.IPAddress = addr
		return t
	}
	return dc
}

// String returns a human-readable label for this data center, e.g. "DC2".
func (dc DataCenter) String() string {
	if dc.IPAddress != "" || dc.PortValue > 0 {
		return fmt.Sprintf("DC%d(%s:%d)", dc.ID, dc.Address(), dc.Port())
	}
	return fmt.Sprintf("DC%d", dc.ID)
}

var (
	prodIPv4 = map[int]string{
		1: "149.154.175.53",
		2: "149.154.167.51",
		3: "149.154.175.100",
		4: "149.154.167.91",
		5: "149.154.171.5",
	}
	testIPv4 = map[int]string{
		1: "149.154.175.10",
		2: "149.154.167.40",
		3: "149.154.175.117",
		4: "149.154.167.91",
		5: "149.154.171.5",
	}
	prodIPv6 = map[int]string{
		1: "2001:b28:f23d:f001::a",
		2: "2001:b28:f23d:f001::e",
		3: "2001:b28:f23d:f001::d",
		4: "2001:b28:f23d:f001::14",
		5: "2001:b28:f23d:f001::5",
	}
	prodMediaIPv4 = map[int]string{
		// Media-only endpoints from help.getConfig (production). Telegram
		// exposes dedicated file-transfer IPs only for some DCs; transfers
		// on the main API endpoints get throttled and culled.
		2: "149.154.167.223",
		4: "149.154.165.96",
	}
	testMediaIPv4 = map[int]string{}
	testIPv6      = map[int]string{
		1: "2001:b28:f23f:f001::a",
		2: "2001:67c:0468:f001::a",
		3: "2001:b28:f23f:f001::d",
		4: "2001:67c:0468:f001::a",
		5: "2001:67c:0468:f001::a",
	}
)

func dcIPv4Address(id int, testMode bool) string {
	m := prodIPv4
	if testMode {
		m = testIPv4
	}
	return m[id]
}

func dcIPv6Address(id int, testMode bool) string {
	m := prodIPv6
	if testMode {
		m = testIPv6
	}
	return m[id]
}
