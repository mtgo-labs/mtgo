package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/internal/storage/sqlite"
)

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "storage_bot.session",
		SavePeers:   true,
		Storage:     sqlite.New(),
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		ctx.Reply(
			"<b>Storage Bot</b>\n\n" +
				"Commands:\n" +
				"• /note &lt;text&gt; — save a note\n" +
				"• /notes — list your notes\n" +
				"• /peers — show cached peers\n" +
				"• /clear — delete your notes",
		)
	}, telegram.Command("start"))

	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		text := msg.Text
		if len(text) <= 5 {
			ctx.Reply("Usage: /note <text>")
			return
		}
		body := text[6:]
		conv := &storage.Conversation{
			ChatID:    msg.ChatID,
			UserID:    msg.FromID,
			Name:      "note:" + body,
			Step:      0,
			UpdatedAt: time.Now().Unix(),
		}
		if err := client.SaveConversation(conv); err != nil {
			ctx.Reply(fmt.Sprintf("save error: %v", err))
			return
		}
		ctx.Reply("Note saved!")
	}, telegram.Command("note"))

	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		conv, err := client.LoadConversation(msg.ChatID, msg.FromID)
		if err != nil {
			ctx.Reply(fmt.Sprintf("error: %v", err))
			return
		}
		if conv == nil {
			ctx.Reply("No notes found. Use /note <text> to save one.")
			return
		}
		ctx.Reply(fmt.Sprintf("Your note: %s", conv.Name))
	}, telegram.Command("notes"))

	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		if err := client.DeleteConversation(msg.ChatID, msg.FromID); err != nil {
			ctx.Reply(fmt.Sprintf("error: %v", err))
			return
		}
		ctx.Reply("Notes cleared!")
	}, telegram.Command("clear"))

	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		peers, err := client.LoadPeers()
		if err != nil {
			ctx.Reply(fmt.Sprintf("error: %v", err))
			return
		}
		if len(peers) == 0 {
			ctx.Reply("No cached peers yet.")
			return
		}
		var text strings.Builder
		fmt.Fprintf(&text, "Cached peers (%d):\n", len(peers))
		for i, p := range peers {
			if i >= 20 {
				text.WriteString("...")
				break
			}
			name := p.FirstName
			if p.Username != "" {
				name += " @" + p.Username
			}
			fmt.Fprintf(&text, "• %s (id=%d, %s)\n", name, p.ID, p.Type)
		}
		ctx.Reply(text.String())
	}, telegram.Command("peers"))

	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		if msg.FromID == 0 {
			return
		}
		peer := &storage.Peer{
			ID:          msg.FromID,
			Type:        storage.PeerTypeUser,
			LastUpdated: time.Now().Unix(),
		}
		client.SavePeer(peer)
	})

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	shutdownCtx, stopNotify := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopNotify()
	go func() {
		<-shutdownCtx.Done()
		client.Stop()
	}()

	bot, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}

	fmt.Println("=== Storage Bot ===")
	fmt.Printf("  Bot: %s (@%s)\n", bot.FirstName, bot.Username)
	fmt.Println("  Commands: /start /note /notes /peers /clear")
	fmt.Println("─────────────────────")
	fmt.Println("storage bot running, press Ctrl+C to stop")

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
