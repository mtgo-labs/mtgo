package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// userbot demonstrates a userbot with automatic phone login.
//
// On first run (no saved session), Connect() detects the session is not
// authorized and automatically prompts for:
//
//  1. Verification code (sent via SMS/Telegram)
//  2. 2FA password (if enabled)
//
// The session is saved to disk, so subsequent runs reuse it without prompts.
//
// Usage:
//
//	# First run (interactive login):
//	API_ID=12345 API_HASH=abc PHONE="+1234567890" go run .
//
//	# Subsequent runs reuse the saved session automatically:
//	API_ID=12345 API_HASH=abc PHONE="+1234567890" go run .
func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	phone := mustEnv("PHONE")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		PhoneNumber:       phone,
		SessionName:       "userbot",
		SavePeers:         true,
		DispatchQueueSize: 512,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	client.OnMessage(func(client *telegram.Client, msg *types.Message) {
		if msg.Text == "" {
			return
		}

		switch msg.Text {
		case "/ping":
			msg.Reply("pong")
		case "/me":
			me := client.Me()
			if me != nil {
				msg.Reply(fmt.Sprintf("ID: %d\nName: %s\nUsername: @%s", me.ID, me.FirstName, me.Username))
			}
		case "/session":
			s, err := client.ExportSessionString()
			if err != nil {
				msg.Reply(fmt.Sprintf("error: %v", err))
				return
			}
			msg.Reply("Session string saved to logs (not sent here for security)")
			log.Printf("SESSION=%s", s)
		}
	}, telegram.Private)

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

	fmt.Println("=== Userbot ===")
	printUser(me)

	sessionString, err := client.ExportSessionString()
	if err != nil {
		if !errors.Is(err, telegram.ErrAPIIDRequired) {
			log.Printf("warning: could not export session: %v", err)
		}
	} else {
		fmt.Println("\nSave this for future runs without re-login:")
		fmt.Printf("  SESSION=%q\n", sessionString)
	}

	fmt.Println("\nuserbot is running. Press Ctrl+C to stop.")
	client.Idle()
}

func printUser(u *types.User) {
	fmt.Printf("  ID:       %d\n", u.ID)
	fmt.Printf("  Name:     %s", u.FirstName)
	if u.LastName != "" {
		fmt.Printf(" %s", u.LastName)
	}
	fmt.Println()
	if u.Username != "" {
		fmt.Printf("  Username: @%s\n", u.Username)
	}
	fmt.Printf("  Bot:      %v\n", u.IsBot)
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
