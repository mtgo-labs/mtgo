package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// export_session connects to Telegram and prints the session string
// that can be used with session.TelethonSession() or Config.SessionString.
//
// Required environment variables:
//   - API_ID:     your Telegram API ID
//   - API_HASH:   your Telegram API hash
//   - BOT_TOKEN:  your bot token

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "export_session",
		SavePeers:   true,
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

	fmt.Println("=== Bot Info ===")
	fmt.Printf("  ID:       %d\n", me.ID)
	fmt.Printf("  Name:     %s\n", me.FirstName)
	if me.Username != "" {
		fmt.Printf("  Username: @%s\n", me.Username)
	}

	sessionStr, err := client.ExportSessionString()
	if err != nil {
		log.Fatalf("export: %v", err)
	}

	fmt.Println()
	fmt.Println("=== Session String ===")
	fmt.Println(sessionStr)
	fmt.Println()
	fmt.Println("Use with:")
	fmt.Println("  Config.SessionString: sessionStr")

	client.OnMessage(func(client *telegram.Client, msg *types.Message) {
		if msg.Text == "/exportsession" {
			s, _ := client.ExportSessionString()
			msg.Reply("Session string:\n" + s)
		}
	}, telegram.Private)

	fmt.Println("export_session is running. Press Ctrl+C to stop.")
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
