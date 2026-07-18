package otel

import (
	"context"
	"errors"
	"testing"
	"time"

	mtgo "github.com/mtgo-labs/mtgo/telegram"
)

func TestGlobalProviderAdapter(t *testing.T) {
	observer, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now().Add(-time.Millisecond)
	observer.ObserveRPC(context.Background(), mtgo.RPCObservation{
		Method: "HelpGetConfig", DCID: 2, Attempt: 1,
		StartedAt: started, EndedAt: time.Now(), ErrorClass: "ok",
	})
	observer.ObserveConnection(context.Background(), mtgo.ConnectionObservation{
		Kind: "dial.failure", Endpoint: "127.0.0.1:443",
		StartedAt: started, EndedAt: time.Now(), Error: errors.New("test"),
	})
}
