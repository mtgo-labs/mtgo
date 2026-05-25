package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mtgo-labs/mtgo/telegram"
)

// phone_login demonstrates interactive phone number login.
//
// On first run (no saved session), Connect() detects that the session is
// not authorized and automatically prompts for:
//
//   1. Verification code (sent to the phone number via SMS/Telegram)
//   2. 2FA password (if the account has two-factor authentication enabled)
//
// On subsequent runs, the saved session is reused and no prompts appear.
//
// Usage:
//
//	API_ID=12345 API_HASH="abc" PHONE="+1234567890" go run examples/phone_login/main.go
//
// To override the default terminal prompts, set custom CodeFunc/PasswordFunc:
//
//	cfg := &telegram.Config{
//	    PhoneNumber: "+1234567890",
//	    SessionName: "my_session",
//	    CodeFunc: func(ctx context.Context, phone string) (string, error) {
//	        // Read code from your webhook, database, etc.
//	        return readFromWebhook(ctx)
//	    },
//	    PasswordFunc: func(ctx context.Context, hint string) (string, error) {
//	        return readFromUI(ctx)
//	    },
//	}
func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	phone := mustEnv("PHONE")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		PhoneNumber: phone,
		SessionName: "phone_login",
		SavePeers:   true,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	// Connect triggers the interactive login flow automatically when the
	// session is not yet authorized and PhoneNumber is set.
	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	me, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}

	fmt.Println("=== Phone Login ===")
	fmt.Printf("  ID:       %d\n", me.ID)
	fmt.Printf("  Name:     %s\n", me.FirstName)
	if me.Username != "" {
		fmt.Printf("  Username: @%s\n", me.Username)
	}
	fmt.Println("─────────────────────")
	fmt.Println("Logged in. Press Ctrl+C to stop.")
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
