package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// import_session demonstrates using a session string from any format
// (Telethon, Pyrogram, GramJS, mtcute) with auto-detection.
//
// Required environment variables:
//   - API_ID:     your Telegram API ID
//   - API_HASH:   your Telegram API hash
//   - SESSION:    the string session to import
//
// Usage:
//
//	# From a Telethon string:
//	SESSION="1abc..." go run .
//
//	# From a Pyrogram string (auto-converted):
//	SESSION="base64..." go run .
func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	sessionStr := mustEnv("SESSION")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		SessionString: sessionStr,
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

	fmt.Println("=== Connected As ===")
	fmt.Printf("  ID:       %d\n", me.ID)
	fmt.Printf("  Name:     %s\n", me.FirstName)
	if me.Username != "" {
		fmt.Printf("  Username: @%s\n", me.Username)
	}

	client.OnMessage(func(client *telegram.Client, msg *types.Message) {
		if msg.Text == "/ping" {
			msg.Reply("pong")
		}
	}, telegram.Private)

	fmt.Println("import_session is running. Press Ctrl+C to stop.")
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
