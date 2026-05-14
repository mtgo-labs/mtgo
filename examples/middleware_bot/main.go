package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mtgo-labs/middlewares/floodwait"
	"github.com/mtgo-labs/middlewares/ratelimit"
	tg "github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")

	client, err := tg.NewClient(mustAtoi(apiID), apiHash, &tg.Config{
		BotToken:    botToken,
		SessionName: "middleware_bot",
		SavePeers:   true,
		ParseMode:   tg.HTML,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	// ── Invoker middleware: rate limiting ──────────────────────────
	// Allow at most 20 RPC calls/sec with a burst of 5.
	// All outgoing Telegram API calls are throttled transparently.
	limiter := ratelimit.New(20, 5)
	client.UseInvokerMiddleware(limiter.Middleware())

	// ── Invoker middleware: flood-wait retry ───────────────────────
	// Automatically sleeps and retries when Telegram returns FLOOD_WAIT.
	waiter := floodwait.New()
	waiter.OnWait(func(d time.Duration) {
		log.Printf("flood wait: sleeping %v", d)
	})
	waiter.WithMaxWait(60 * time.Second)
	client.UseInvokerMiddleware(waiter.Middleware())

	// ── Handler middleware: logging (priority -10, runs first) ─────
	client.UseMiddleware(func(next tg.Handler) tg.Handler {
		return &tg.FuncHandler{Fn: func(ctx *tg.Context) {
			if ctx.Message != nil {
				from := ctx.Message.ChatID
				text := ctx.Message.Text
				log.Printf("[%d] %s", from, text)
			}
			next.Handle(ctx)
		}}
	}, -10)

	// ── Handler middleware: admin guard (priority 0, runs after logging) ─
	adminID := int64(0) // replace with your Telegram user ID
	if envAdmin := os.Getenv("ADMIN_ID"); envAdmin != "" {
		adminID = mustAtoi64(envAdmin)
	}
	if adminID != 0 {
		client.UseMiddleware(func(next tg.Handler) tg.Handler {
			return &tg.FuncHandler{Fn: func(ctx *tg.Context) {
				if ctx.Message != nil && ctx.Message.FromID != adminID {
					ctx.Reply("Unauthorized")
					ctx.Stopped = true
					return
				}
				next.Handle(ctx)
			}}
		})
	}

	// ── Handlers ───────────────────────────────────────────────────
	client.OnMessage(func(client *tg.Client, msg *types.Message) {
		if msg == nil || msg.Text == "" {
			return
		}

		switch msg.Text {
		case "/start":
			msg.Reply(
				"<b>Middleware Bot</b>\n\n"+
					"This bot demonstrates mtgo middleware:\n"+
					"• Rate limiting (invoker middleware)\n"+
					"• Flood-wait retry (invoker middleware)\n"+
					"• Request logging (handler middleware)\n"+
					"• Admin guard (handler middleware)\n\n"+
					"Commands: /start /ping /stats",
				&params.SendMessage{ParseMode: params.ParseModeHTML},
			)

		case "/ping":
			msg.Reply("pong")

		case "/stats":
			info := fmt.Sprintf(
				"Rate limiter: 20 tokens/sec, burst 5\nFlood-wait: max wait %v, max retries 5",
				60*time.Second,
			)
			msg.Reply(info)
		}
	}, tg.Private)

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
