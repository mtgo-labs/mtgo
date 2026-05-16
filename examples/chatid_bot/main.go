package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "chatid_bot",
		SavePeers:   true,
		ParseMode:   types.HTML,
		InMemory:    true,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	// /start
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		_, err := ctx.Reply(
			fmt.Sprintf("Your ID: <code>%d</code>\nPick a chat type below to get its ID.", msg.Sender.ID),
			&params.SendMessage{
				ReplyMarkup: telegram.Keyboard().
					RequestChannel("Channel", 1).
					RequestGroup("Group", 2).
					Next().
					RequestUser("User", 3, 1).
					RequestUser("Bot", 4, 1, telegram.PeerUserOpts{Bot: true}).
					BuildReply(telegram.ReplyOpts{
						Resize:      true,
						OneTime:     true,
						Placeholder: "Select chat type",
					}),
			},
		)
		if err != nil {
			log.Printf("/start reply error: %v", err)
		}
	}, telegram.Command("start"))

	// Handle service messages carrying shared peers.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		if msg.Service == nil || msg.Service.Type != types.ServiceActionRequestedPeer {
			return
		}
		rp := msg.Service.RequestedPeers
		if rp == nil {
			return
		}

		var label string
		var id int64

		switch rp.ButtonID {
		case 1: // Channel
			if len(rp.ChatIDs) > 0 {
				label = "Channel"
				id = rp.ChatIDs[0]
			}
		case 2: // Group
			if len(rp.ChatIDs) > 0 {
				label = "Group"
				id = rp.ChatIDs[0]
			}
		case 3: // User
			if len(rp.UserIDs) > 0 {
				label = "User"
				id = rp.UserIDs[0]
			}
		case 4: // Bot
			if len(rp.UserIDs) > 0 {
				label = "Bot"
				id = rp.UserIDs[0]
			}
		}

		if label != "" {
			_, err := ctx.Reply(fmt.Sprintf("%s ID: <code>%d</code>", label, id))
			if err != nil {
				log.Printf("peer reply error: %v", err)
			}
		}
	})

	// /id — detailed ID info.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		text := fmt.Sprintf("Chat ID: <code>%d</code>\nUser ID: <code>%d</code>", msg.ChatID, msg.FromID)
		_, err := ctx.Reply(text)
		if err != nil {
			log.Printf("/id reply error: %v", err)
		}
	}, telegram.Command("id"))

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	bot, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}

	fmt.Println("=== Chat ID Bot ===")
	fmt.Printf("  ID:   %d\n", bot.ID)
	fmt.Printf("  Name: %s\n", bot.FirstName)
	fmt.Println("───────────────────")
	fmt.Println("chat ID bot running, press Ctrl+C to stop")

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
