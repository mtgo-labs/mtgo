package telegram

import (
	"time"
)

var (
	// TestDCs maps Telegram test data-center IDs to their IP addresses.
	TestDCs = map[int]string{
		1: "149.154.175.10",
		2: "149.154.167.40",
		3: "149.154.175.117",
	}

	// ProdDCs maps Telegram production data-center IDs to their IP addresses.
	ProdDCs = map[int]string{
		1:   "149.154.175.53",
		2:   "149.154.167.51",
		3:   "149.154.175.100",
		4:   "149.154.167.91",
		5:   "91.108.56.130",
		203: "91.105.192.100",
	}

	// DefaultTestPort is the default TCP port for test-mode connections.
	DefaultTestPort = 80
	// DefaultProdPort is the default TCP port for production connections.
	DefaultProdPort = 443
)

// ServerTime returns the current Unix timestamp adjusted by the given offset in seconds.
func ServerTime(offset int) int32 {
	return int32(time.Now().Unix() + int64(offset))
}

// ResolveDCAddress returns the IP address for the given data-center ID. When testMode is
// true it looks up TestDCs, otherwise ProdDCs. Returns an empty string if the DC ID is
// unknown.
func ResolveDCAddress(dcID int, testMode bool) string {
	if testMode {
		if addr, ok := TestDCs[dcID]; ok {
			return addr
		}
		return ""
	}
	if addr, ok := ProdDCs[dcID]; ok {
		return addr
	}
	return ""
}

// DefaultDCPort returns the default TCP port for the given mode: DefaultTestPort when
// testMode is true, otherwise DefaultProdPort.
func DefaultDCPort(testMode bool) int {
	if testMode {
		return DefaultTestPort
	}
	return DefaultProdPort
}
