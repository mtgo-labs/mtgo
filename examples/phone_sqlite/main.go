package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/types"
	extstorage "github.com/mtgo-labs/storage"
	"github.com/mtgo-labs/storage/sqlite"
)

// phone_sqlite demonstrates a userbot with phone-number login backed by
// SQLite storage.
//
// On first run (no saved session), Connect() detects the session is not
// authorized and automatically prompts for:
//
//  1. Verification code (sent via SMS/Telegram)
//  2. 2FA password (if enabled)
//
// The session (auth key, DC, user info, cached peers) is persisted in a
// SQLite database, so subsequent runs reuse it without prompts.
//
// Usage:
//
//	API_ID=12345 API_HASH=abc PHONE="+1234567890" go run .
//
// To test DC migration (e.g. your phone belongs to DC 4), simply do not
// set Config.DC — the client auto-detects the correct DC on first login.
func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	phone := mustEnv("PHONE")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		PhoneNumber: phone,
		SessionName: "phone_sqlite.session",
		SavePeers:   true,
		Storage:     sqlite.New(),
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	// --- Handlers ---

	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		ctx.Reply(
			"<b>Phone SQLite Userbot</b>\n\n" +
				"Commands:\n" +
				"• /ping — pong\n" +
				"• /me — your account info\n" +
				"• /peers — cached peers in SQLite\n" +
				"• /session — export session string",
		)
	}, telegram.Command("start"))

	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		ctx.Reply("pong")
	}, telegram.Command("ping"))

	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		me := client.Me()
		if me == nil {
			ctx.Reply("Could not fetch account info.")
			return
		}
		ctx.Reply(fmt.Sprintf(
			"ID: %d\nName: %s\nUsername: @%s\nBot: %v",
			me.ID, me.FirstName, me.Username, me.IsBot,
		))
	}, telegram.Command("me"))

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
		s, err := client.ExportSessionString()
		if err != nil {
			if errors.Is(err, telegram.ErrAPIIDRequired) {
				ctx.Reply("Session export requires api_id.")
			} else {
				ctx.Reply(fmt.Sprintf("error: %v", err))
			}
			return
		}
		log.Printf("exported session string (%d chars)", len(s))
		ctx.Reply("Session string exported — check logs.")
	}, telegram.Command("session"))

	// Auto-save every incoming message sender to SQLite peer cache.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		if msg.FromID == 0 {
			return
		}
		_ = client.SavePeer(&extstorage.Peer{
			ID:          msg.FromID,
			Type:        extstorage.PeerTypeUser,
			LastUpdated: time.Now().Unix(),
		})
	}, telegram.Incoming)

	// --- Connect (triggers interactive phone login on first run) ---

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	me, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}

	fmt.Println("=== Phone SQLite Userbot ===")
	fmt.Printf("  ID:       %d\n", me.ID)
	fmt.Printf("  Name:     %s\n", me.FirstName)
	if me.Username != "" {
		fmt.Printf("  Username: @%s\n", me.Username)
	}
	fmt.Println("  Storage:  SQLite")
	fmt.Println("────────────────────────────")
	fmt.Println("userbot running, press Ctrl+C to stop")

	shutdownCtx, stopNotify := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopNotify()
	go func() {
		<-shutdownCtx.Done()
		client.Stop()
	}()

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
