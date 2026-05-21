package telegram

import (
	"context"
	"time"
)

var (
	prodWSDomains = map[int]string{
		1: "pluto.web.telegram.org",
		2: "venus.web.telegram.org",
		3: "aurora.web.telegram.org",
		4: "vesta.web.telegram.org",
		5: "flora.web.telegram.org",
	}
	testWSDomains = map[int]string{
		1: "pluto.web.telegram.org",
		2: "venus.web.telegram.org",
		3: "aurora.web.telegram.org",
		4: "vesta.web.telegram.org",
		5: "flora.web.telegram.org",
	}
)

func wsDCAddress(dcID int, testMode bool, tls bool) string {
	m := prodWSDomains
	if testMode {
		m = testWSDomains
	}
	host := m[dcID]
	path := "/apiws"
	if testMode {
		path = "/apiws_test"
	}
	if tls {
		return "wss://" + host + ":443" + path
	}
	return "ws://" + host + ":80" + path
}

func useWebSocket(cfg Config) bool {
	return cfg.WebSocket
}

func dialerCtx(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
