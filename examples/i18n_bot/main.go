package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/plugins/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.yaml
var locales embed.FS

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "i18n_bot",
		SavePeers:   true,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	tr, err := i18n.NewTranslator(&i18n.Config{
		DefaultLang: language.English,
		Format:      i18n.FormatYAML,
		EmbedFS:     locales,
		LocaleDir:   "locales",
		GlobalContext: func(ctx *telegram.Context) map[string]any {
			name := ""
			if sender := ctx.Sender(); sender != nil {
				name = sender.FirstName
			}
			return map[string]any{
				"name": name,
			}
		},
	})
	if err != nil {
		log.Fatalf("i18n init: %v", err)
	}

	client.Use(tr)

	// {name} is auto-filled from GlobalContext — no need to pass it
	client.OnMessage(func(c *telegram.Client, msg *types.Message) {
		msg.Reply(msg.T("start"))
	}, telegram.Command("start"))

	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		msg.Reply(msg.T("help"))
	}, telegram.Command("help"))

	client.OnMessage(func(c *telegram.Client, msg *types.Message) {
		msg.Reply(msg.T("items", 5))
	}, telegram.Command("items"))

	client.OnMessage(func(ctx *telegram.Context, c *telegram.Client, msg *types.Message) {
		msg.Reply(msg.T("cart"))
	}, telegram.Command("cart"))

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	bot, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}

	fmt.Println("=== i18n Bot ===")
	fmt.Printf("  ID:       %d\n", bot.ID)
	fmt.Printf("  Name:     %s\n", bot.FirstName)
	fmt.Printf("  Locales:  %v\n", tr.Locales())
	fmt.Println("─────────────────────")
	fmt.Println("i18n bot running, press Ctrl+C to stop")

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
