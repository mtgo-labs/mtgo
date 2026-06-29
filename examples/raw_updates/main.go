package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg"
)

// raw_updates demonstrates every style of typed raw update handling in mtgo.
//
// Typed callbacks auto-filter by update type — no manual type-switching.
// The callback signature determines which update type triggers the handler.
//
// Usage:
//
//	API_ID=12345 API_HASH=abc BOT_TOKEN=123:ABC go run .
func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken: botToken,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	// ═══════════════════════════════════════════════════════════════════
	// Style 1: Typed callback — update only
	// Fires ONLY for *tg.UpdateUserTyping. Other update types are ignored.
	// ═══════════════════════════════════════════════════════════════════
	client.OnRawUpdate(func(upd *tg.UpdateUserTyping) {
		log.Printf("[typing] user %d: %T", upd.UserID, upd.Action)
	})

	// ═══════════════════════════════════════════════════════════════════
	// Style 2: Typed callback with context
	// Same auto-filtering, plus access to ctx for replies, client, etc.
	// ═══════════════════════════════════════════════════════════════════
	client.OnRawUpdate(func(ctx *telegram.Context, upd *tg.UpdateDeleteMessages) {
		log.Printf("[deleted] %d messages removed from chat", len(upd.Messages))
	})

	// ═══════════════════════════════════════════════════════════════════
	// Style 3: Typed callback with client
	// Direct client access for API calls without ctx.
	// ═══════════════════════════════════════════════════════════════════
	client.OnRawUpdate(func(c *telegram.Client, upd *tg.UpdateReadHistoryOutbox) {
		log.Printf("[read] outgoing messages read up to %d", upd.MaxID)
	})

	// ═══════════════════════════════════════════════════════════════════
	// Style 4: Catch-all with UpdateType filter
	// Uses the traditional func(*Context) form but filtered by type.
	// Useful when you need the full Context (Users, Chats maps, etc.)
	// along with the typed update.
	// ═══════════════════════════════════════════════════════════════════
	client.OnRawUpdate(func(ctx *telegram.Context) {
		upd := ctx.Update.Raw.(*tg.UpdateMessagePoll)
		log.Printf("[poll] id=%d voters=%d", upd.PollID, upd.Results.TotalVoters)
	}, telegram.UpdateType[*tg.UpdateMessagePoll]())

	// ═══════════════════════════════════════════════════════════════════
	// Style 5: Multiple update types via catch-all + type switch
	// For cases where you want to handle several types in one handler.
	// ═══════════════════════════════════════════════════════════════════
	client.OnRawUpdate(func(ctx *telegram.Context) {
		switch upd := ctx.Update.Raw.(type) {
		case *tg.UpdateUserStatus:
			log.Printf("[status] user %d -> %T", upd.UserID, upd.Status)
		case *tg.UpdateChannelUserTyping:
			log.Printf("[typing] channel %d: %T", upd.ChannelID, upd.Action)
		case *tg.UpdateChatUserTyping:
			log.Printf("[typing] chat %d: %T", upd.ChatID, upd.Action)
		}
	})

	// ═══════════════════════════════════════════════════════════════════
	// Style 6: Typed callback for phone call signaling (userbot feature)
	// Demonstrates handling a niche update type cleanly.
	// ═══════════════════════════════════════════════════════════════════
	client.OnRawUpdate(func(upd *tg.UpdatePhoneCallSignalingData) {
		log.Printf("[phone] call %d signaling: %d bytes", upd.PhoneCallID, len(upd.Data))
	})

	// ═══════════════════════════════════════════════════════════════════
	// Style 7: Combining typed handler with additional filters
	// The typed form still accepts optional Filter arguments.
	// ═══════════════════════════════════════════════════════════════════
	// Example: only log typing from a specific user (custom filter)
	userFilter := func(ctx *telegram.Context) bool {
		upd, ok := ctx.Update.Raw.(*tg.UpdateUserTyping)
		if !ok {
			return false
		}
		return upd.UserID == 12345678 // replace with target user ID
	}
	client.OnRawUpdate(func(ctx *telegram.Context) {
		upd := ctx.Update.Raw.(*tg.UpdateUserTyping)
		log.Printf("[filtered typing] user %d: %T", upd.UserID, upd.Action)
	}, userFilter)

	// ═══════════════════════════════════════════════════════════════════
	// Style 8: Pure catch-all (logs everything)
	// The original OnRawUpdate behavior — fires for ALL updates.
	// ═══════════════════════════════════════════════════════════════════
	client.OnRawUpdate(func(ctx *telegram.Context) {
		// Uncomment to see every raw update:
		// log.Printf("[raw] %T", ctx.Update.Raw)
	})

	// ── Connect ──────────────────────────────────────────────────────

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	me, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}

	fmt.Println("=== Raw Updates Demo ===")
	fmt.Printf("  Bot:      @%s\n", me.Username)
	fmt.Println("  Handlers: 8 registered (typed, filtered, catch-all)")
	fmt.Println("────────────────────────────")
	fmt.Println("listening for raw updates, press Ctrl+C to stop")

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
