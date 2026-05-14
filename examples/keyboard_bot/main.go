package main

import (
	"context"
	"fmt"
	"log"
	"os"

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
		SessionName: "keyboard_bot",
		SavePeers:   true,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	// /start — welcome with inline URL button.
	client.OnMessage(func(ctx *tg.Context, msg *types.Message) {
		text := "Here is some bot commands:\n\n"
		text += "- /keyboard - show keyboard\n"
		text += "- /inline - show inline keyboard\n"
		text += "- /remove - remove keyboard\n"
		text += "- /force - force reply"

		_, _ = ctx.Reply(text, &params.SendMessage{
			ReplyMarkup: tg.Keyboard().
				URL("GitHub", "https://github.com/mtgo-labs/mtgo").
				Build(),
		})
	}, tg.Command("start"))

	// /inline — inline keyboard with callback buttons.
	client.OnMessage(func(ctx *tg.Context, msg *types.Message) {
		_, _ = ctx.Reply("This is an inline keyboard", &params.SendMessage{
			ReplyMarkup: tg.Keyboard().
				Callback("OwO", "OwO").
				Callback("UwU", "UwU").
				Build(),
		})
	}, tg.Command("inline"))

	// /keyboard — reply keyboard.
	client.OnMessage(func(ctx *tg.Context, msg *types.Message) {
		_, _ = ctx.Reply("This is a keyboard", &params.SendMessage{
			ReplyMarkup: tg.Keyboard().
				Text("OwO").
				Text("UwU").
				BuildReply(tg.ReplyOpts{Resize: true, OneTime: true}),
		})
	}, tg.Command("keyboard"))

	// /remove — remove reply keyboard.
	client.OnMessage(func(ctx *tg.Context, msg *types.Message) {
		_, _ = ctx.Reply("Keyboards removed", &params.SendMessage{
			ReplyMarkup: tg.RemoveKeyboard(),
		})
	}, tg.Command("remove"))

	// /force — force reply.
	client.OnMessage(func(ctx *tg.Context, msg *types.Message) {
		_, _ = ctx.Reply("This is a force reply", &params.SendMessage{
			ReplyMarkup: tg.ForceReplyMarkup(),
		})
	}, tg.Command("force"))

	// Echo any other text.
	client.OnMessage(func(ctx *tg.Context, msg *types.Message) {
		_, _ = ctx.Reply(fmt.Sprintf("You said %q", msg.Text))
	})

	// Handle inline keyboard button presses.
	client.OnCallbackQuery(func(ctx *tg.Context) {
		data := string(ctx.CallbackQuery.Data)
		_, _ = ctx.CallbackEditText(
			fmt.Sprintf("You pressed %s", data),
			&params.EditMessage{
				ReplyMarkup: tg.Keyboard().
					URL("GitHub", "https://github.com/mtgo-labs/mtgo").
					Build(),
			},
		)
		_ = ctx.Answer("", false)
	})

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	bot, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}

	fmt.Println("=== Keyboard Bot ===")
	fmt.Printf("  ID:   %d\n", bot.ID)
	fmt.Printf("  Name: %s\n", bot.FirstName)
	fmt.Println("────────────────────")
	fmt.Println("keyboard bot running, press Ctrl+C to stop")

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
