package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mtgo-labs/mtgo/session/tdesktop"
	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// import_tdata demonstrates importing session data from a Telegram Desktop
// tdata directory.
//
// Required environment variables:
//   - API_ID:      your Telegram API ID
//   - API_HASH:    your Telegram API hash
//   - TDATA_PATH:  path to the tdata directory
//
// Optional:
//   - TDATA_PASSCODE: local passcode (leave empty if none was set)

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	tdataPath := mustEnv("TDATA_PATH")
	passcode := os.Getenv("TDATA_PASSCODE")

	accounts, err := tdesktop.Read(tdataPath, []byte(passcode))
	if err != nil {
		log.Fatalf("read tdata: %v", err)
	}
	if len(accounts) == 0 {
		log.Fatal("no accounts found in tdata")
	}

	acc := accounts[0]
	fmt.Printf("=== Account (UserID=%d, MainDC=%d) ===\n", acc.UserID, acc.MainDC)

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		SessionString: acc.String(),
		InMemory:      true,
		SavePeers:     true,
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

	fmt.Printf("  Connected as: %s (ID: %d)\n", me.FirstName, me.ID)
	if me.Username != "" {
		fmt.Printf("  Username: @%s\n", me.Username)
	}

	client.OnMessage(func(client *telegram.Client, msg *types.Message) {
		if msg.Text == "/ping" {
			msg.Reply("pong")
		}
	}, telegram.Private)

	fmt.Println("import_tdata is running. Press Ctrl+C to stop.")
	client.Idle()
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("environment variable %s is required", key)
	}
	return v
}

func mustAtoi(s string) int32 {
	var n int32
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		log.Fatalf("invalid integer %q: %v", s, err)
	}
	return n
}
