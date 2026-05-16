// mtproxy_bot demonstrates connecting a Telegram bot through an MTProxy server.
//
// Setup:
//
//	export API_ID=12345
//	export API_HASH=your_api_hash
//	export BOT_TOKEN=your_bot_token
//	export MTPROXY_ADDR=proxy.example.com:443
//	export MTPROXY_SECRET=dd00000000000000000000000000000000a
//
// The secret format determines the transport:
//
//   - dd-prefix (34 hex chars): obfuscated2 with PaddedIntermediate (most common)
//   - ee-prefix (36+ hex chars): fake TLS handshake + obfuscated2
//   - simple (32 hex chars): raw 16-byte secret, obfuscated2 with Intermediate
//
// Run:
//
//	go run ./examples/mtproxy_bot
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
	proxyAddr := mustEnv("MTPROXY_ADDR")
	proxySecret := mustEnv("MTPROXY_SECRET")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "mtproxy_bot",
		InMemory:    true,
		MTProxy: &telegram.MTProxyConfig{
			Addr:   proxyAddr,
			Secret: proxySecret,
		},
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
		log.Fatalf("connect via mtproxy: %v", err)
	}
	defer client.Stop()

	bot, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}

	fmt.Println("=== MTProxy Bot ===")
	fmt.Printf("  ID:       %d\n", bot.ID)
	fmt.Printf("  Name:     %s\n", bot.FirstName)
	if bot.Username != "" {
		fmt.Printf("  Username: @%s\n", bot.Username)
	}
	fmt.Printf("  Proxy:    %s\n", proxyAddr)
	fmt.Println("─────────────────────")
	fmt.Println("mtproxy bot running, press Ctrl+C to stop")

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
