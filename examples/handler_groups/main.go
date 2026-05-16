package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/params"
)

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "handler_groups_bot",
		SavePeers:   true,
		ParseMode:   telegram.HTML,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}
	// ── Group -10: logging (runs first) ──────────────────────────────
	// Handlers in lower-numbered groups execute before higher-numbered ones.
	// Use this for cross-cutting concerns like logging or metrics.
	loggingHandler := telegram.NewMessageHandler(func(ctx *telegram.Context) {
		if ctx.Message != nil {
			log.Printf("[log] chat=%d text=%q", ctx.Message.ChatID, ctx.Message.Text)
		}
	})
	client.AddHandler(loggingHandler, -10)

	// ── Group 0: auth guard (default group) ──────────────────────────
	// Handlers without an explicit group default to group 0.
	// This guard stops propagation for unauthorized users, so no
	// higher-numbered groups (1, 2, …) will ever see those updates.
	adminID := int64(0)
	if envAdmin := os.Getenv("ADMIN_ID"); envAdmin != "" {
		adminID = mustAtoi64(envAdmin)
	}
	if adminID != 0 {
		authHandler := telegram.NewMessageHandler(func(ctx *telegram.Context) {
			if ctx.Message != nil && ctx.Message.FromID != adminID {
				ctx.Reply("⛔ Unauthorized")
				ctx.StopPropagation()
			}
		})
		client.AddHandler(authHandler, 0)
	}

	// ── Group 10: business logic (runs after logging & auth) ─────────
	startHandler := telegram.NewMessageHandler(func(ctx *telegram.Context) {
		ctx.Reply(
			"<b>Handler Groups Bot</b>\n\n"+
				"This bot demonstrates handler groups:\n"+
				"• Group -10: logging\n"+
				"• Group 0: auth guard\n"+
				"• Group 10: commands (this handler)\n\n"+
				"Handlers in lower groups run first.\n"+
				"Calling ctx.StopPropagation() stops all later groups.\n\n"+
				"Commands: /start /ping",
			&params.SendMessage{ParseMode: params.ParseModeHTML},
		)
	}, telegram.Command("start"))
	client.AddHandler(startHandler, 10)

	pingHandler := telegram.NewMessageHandler(func(ctx *telegram.Context) {
		ctx.Reply("pong")
	}, telegram.Command("ping"))
	client.AddHandler(pingHandler, 10)

	// ── Group 20: fallback (runs last) ───────────────────────────────
	// Catches any message that wasn't handled by earlier groups.
	fallbackHandler := telegram.NewMessageHandler(func(ctx *telegram.Context) {
		if ctx.Message != nil && ctx.Message.Text != "" {
			ctx.Reply("Unknown command. Try /start")
		}
	})
	client.AddHandler(fallbackHandler, 20)

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	bot, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}
	fmt.Printf("@%s is running — press Ctrl+C to stop\n", bot.Username)

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

func mustAtoi64(s string) int64 {
	var n int64
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		log.Fatalf("invalid integer %q: %v", s, err)
	}
	return n
}
