package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "echo_bot",
		SavePeers:   true,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	client.OnMessage(func(client *telegram.Client, msg *types.Message) {
		if msg == nil || msg.Text == "" {
			return
		}

		client.Log.Info(msg.Text)

		_, err := msg.Reply(msg.Text)
		if err != nil {
			log.Printf("reply error: %v", err)
		}
	}, telegram.Private)

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	bot, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}

	fmt.Println("=== Bot Info ===")
	fmt.Printf("  ID:       %d\n", bot.ID)
	fmt.Printf("  Name:     %s\n", bot.FirstName)
	if bot.LastName != "" {
		fmt.Printf("  Last:     %s\n", bot.LastName)
	}
	if bot.Username != "" {
		fmt.Printf("  Username: @%s\n", bot.Username)
	}
	fmt.Printf("  Bot:      %v\n", bot.IsBot)
	fmt.Println("─────────────────────")
	fmt.Println("echo bot is running, press Ctrl+C to stop")

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
