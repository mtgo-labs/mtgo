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
		SessionName: "keyboard_bot",
		SavePeers:   true,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	// /start — welcome with inline URL button.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		text := "Here is some bot commands:\n\n"
		text += "- /keyboard - show keyboard\n"
		text += "- /inline - show inline keyboard\n"
		text += "- /entities - show formatted text\n"
		text += "- /remove - remove keyboard\n"
		text += "- /force - force reply"

		_, _ = ctx.Reply(text, &params.SendMessage{
			ReplyMarkup: telegram.Keyboard().
				URL("GitHub", "https://github.com/mtgo-labs/mtgo").
				Build(),
		})
	}, telegram.Command("start"))

	// /inline — inline keyboard with callback buttons.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		_, _ = ctx.Reply("This is an inline keyboard", &params.SendMessage{
			ReplyMarkup: telegram.Keyboard().
				Callback("OwO", "OwO").
				Callback("UwU", "UwU").
				Next().
				URL("Docs", "https://example.com").
				Build(),
		})
	}, telegram.Command("inline"))

	// /keyboard — reply keyboard.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		_, _ = ctx.Reply("This is a keyboard", &params.SendMessage{
			ReplyMarkup: telegram.Keyboard().
				Text("OwO").
				Text("UwU").
				BuildReply(telegram.ReplyOpts{Resize: true, OneTime: true}),
		})
	}, telegram.Command("keyboard"))

	// /entities — demonstrate formatted text using params.Entities.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		text := "Hello World"
		_, _ = ctx.Reply(text, &params.SendMessage{
			Entities: params.Entities(
				params.Bold(0, 5),
				params.Italic(6, 5),
			),
		})
	}, telegram.Command("entities"))

	// /formatted — more advanced entity formatting example.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		text := "Bold Italic Code Pre Underline Strike Spoiler"
		_, _ = ctx.Reply(text, &params.SendMessage{
			Entities: params.Entities(
				params.Bold(0, 4),
				params.Italic(5, 6),
				params.Code(12, 4),
				params.Pre(17, 3, "go"),
				params.Underline(21, 9),
				params.Strikethrough(31, 6),
				params.Spoiler(38, 7),
			),
		})
	}, telegram.Command("formatted"))

	// /remove — remove reply keyboard.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		_, _ = ctx.Reply("Keyboards removed", &params.SendMessage{
			ReplyMarkup: telegram.RemoveKeyboard(),
		})
	}, telegram.Command("remove"))

	// /force — force reply.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		_, _ = ctx.Reply("This is a force reply", &params.SendMessage{
			ReplyMarkup: telegram.ForceReplyMarkup(),
		})
	}, telegram.Command("force"))

	// Echo any other text.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		_, _ = ctx.Reply(fmt.Sprintf("You said %q", msg.Text))
	})

	// Handle inline keyboard button presses.
	client.OnCallbackQuery(func(ctx *telegram.Context) {
		data := string(ctx.CallbackQuery.Data)
		_, _ = ctx.CallbackEditText(
			fmt.Sprintf("You pressed %s", data),
			&params.EditMessage{
				ReplyMarkup: telegram.Keyboard().
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
