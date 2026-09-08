// mtgo1_getme loads a native mtgo MTGO1 session string and calls GetMe to
// prove the session is live.
//
// Unlike the other session formats, MTGO1 is self-contained: the payload
// carries the API ID and API hash alongside the auth key, so no separate
// API_ID/API_HASH environment variables are needed.
//
// Required environment variables:
//   - SESSION: the MTGO1 session string (MTGO1.<payload>)
//
// Usage:
//
//	SESSION="MTGO1.AQ..." go run ./examples/mtgo1_getme
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mtgo-labs/mtgo/telegram"
)

func main() {
	sessionStr := os.Getenv("SESSION")
	if sessionStr == "" {
		log.Fatal("environment variable SESSION is required")
	}

	client, err := telegram.NewClient(0, "", &telegram.Config{
		SessionString: sessionStr,
		InMemory:      true,
		NoUpdates:     true,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	me, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}

	fmt.Println("=== Connected As ===")
	fmt.Printf("  ID:       %d\n", me.ID)
	fmt.Printf("  Name:     %s\n", me.FirstName)
	if me.Username != "" {
		fmt.Printf("  Username: @%s\n", me.Username)
	}
	fmt.Printf("  IsBot:    %v\n", me.IsBot)
}
