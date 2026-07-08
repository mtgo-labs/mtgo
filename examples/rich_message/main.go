package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// htmlRichContent is the HTML body sent as a rich message. It exercises the
// block-level HTML elements Telegram renders: headings, divs, tables, lists,
// blockquotes, code blocks, and inline formatting.
const htmlRichContent = `
<h1>Rich Message Showcase</h1>
<p>This message was sent via <b>mtgo</b> using <code>messages.sendMessage</code>
with the <i>rich_message</i> flag.</p>

<h2>Inline Formatting</h2>
<div>
  <p>You can use <b>bold</b>, <i>italic</i>, <u>underline</u>,
  <s>strikethrough</s>, and <a href="https://github.com/mtgo-labs/mtgo">links</a>.</p>
</div>

<h2>Lists</h2>
<h3>Unordered</h3>
<ul>
  <li>First item</li>
  <li>Second item</li>
  <li>Third item</li>
</ul>
<h3>Ordered</h3>
<ol>
  <li>Step one</li>
  <li>Step two</li>
  <li>Step three</li>
</ol>

<h2>Blockquote</h2>
<blockquote>
  Simplicity is the soul of efficiency. — Austin Freeman
</blockquote>

<h2>Code Block</h2>
<pre><code>func main() {
    fmt.Println("Hello, rich message!")
}</code></pre>

<h2>Table</h2>
<div>
  <table border="1" rules="all" frame="box" style="border-collapse: collapse;">
    <thead>
      <tr>
        <th style="border: 1px solid #888;"></th>
        <th style="border: 1px solid #888;">mtgo</th>
        <th style="border: 1px solid #888;">TDLib</th>
        <th style="border: 1px solid #888;">Telethon</th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <th style="border: 1px solid #888;">HTML rich message</th>
        <td style="border: 1px solid #888;">Yes</td>
        <td style="border: 1px solid #888;">Yes</td>
        <td style="border: 1px solid #888;">No</td>
      </tr>
      <tr>
        <th style="border: 1px solid #888;">Markdown rich message</th>
        <td style="border: 1px solid #888;">Yes</td>
        <td style="border: 1px solid #888;">Yes</td>
        <td style="border: 1px solid #888;">No</td>
      </tr>
      <tr>
        <th style="border: 1px solid #888;">Tables</th>
        <td style="border: 1px solid #888;">Yes</td>
        <td style="border: 1px solid #888;">Yes</td>
        <td style="border: 1px solid #888;">No</td>
      </tr>
      <tr>
        <th style="border: 1px solid #888;">Layer</th>
        <td style="border: 1px solid #888;">225</td>
        <td style="border: 1px solid #888;">225</td>
        <td style="border: 1px solid #888;">225</td>
      </tr>
    </tbody>
  </table>
</div>

<hr>
<p><small>Sent from a bot via mtgo.</small></p>
`

// rich_message demonstrates Telegram's rich message feature (TL layer 225)
// using a bot token.
//
// The bot listens for commands in any chat it receives and replies with a rich
// message to that same chat — the destination chat_id is taken from the
// incoming message, so no CHAT_ID env var is needed.
//
// Commands:
//
//	/rich     — send an HTML rich message (headings, divs, tables, lists, …)
//	/richmd   — send a Markdown rich message
//	/getrich  — reply to a rich message to fetch its rendered content back
//
// Rich content is attached to messages.sendMessage via the optional
// rich_message flag (bit 23), which Encode sets automatically when the
// RichMessage field is non-nil. There is no high-level wrapper yet, so this
// example uses the raw RPC client (client.Raw()).
//
// Usage:
//
//	API_ID=... API_HASH=... BOT_TOKEN="..." go run examples/rich_message/main.go
func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "rich_message_bot",
		SavePeers:   true,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	// /rich — send an HTML rich message to the chat the command came from.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		sendRich(ctx, msg, &tg.InputRichMessageHTML{HTML: htmlRichContent}, "HTML")
	}, telegram.Command("rich"))

	// /richmd — send a Markdown rich message to the chat the command came from.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		sendRich(ctx, msg, &tg.InputRichMessageMarkdown{
			Markdown: "# Hello!\n\nThis is a **rich message** sent via _mtgo_.\n",
		}, "Markdown")
	}, telegram.Command("richmd"))

	// /getrich — reply to a rich message to fetch its rendered content back.
	client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
		fetchRich(ctx, msg)
	}, telegram.Command("getrich"))

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	bot, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}

	fmt.Println("=== Rich Message Bot ===")
	fmt.Printf("  ID:       %d\n", bot.ID)
	fmt.Printf("  Name:     %s\n", bot.FirstName)
	if bot.Username != "" {
		fmt.Printf("  Username: @%s\n", bot.Username)
	}
	fmt.Println("─────────────────────────")
	fmt.Println("Commands:")
	fmt.Println("  /rich     — HTML rich message")
	fmt.Println("  /richmd   — Markdown rich message")
	fmt.Println("  /getrich  — reply to a rich message to fetch it")
	fmt.Println("Bot is running, press Ctrl+C to stop")

	client.Idle()
}

// sendRich sends a rich message to the chat the command came from. The peer is
// resolved from the incoming message (already cached with its access hash), so
// no extra lookup is needed for the raw RPC call.
func sendRich(ctx *telegram.Context, msg *types.Message, rich tg.InputRichMessageClass, label string) {
	// chatID comes from the current chat.
	peer, err := ctx.Client.ResolvePeer(ctx.Ctx, msg.ChatID)
	if err != nil {
		log.Printf("[%s] resolve peer %d: %v", label, msg.ChatID, err)
		return
	}

	result, err := ctx.Client.Raw().MessagesSendMessage(ctx.Ctx, &tg.MessagesSendMessageRequest{
		Peer:        peer,
		Message:     "Rich message (" + label + ")",
		RandomID:    ctx.Client.RandomID(),
		RichMessage: rich,
	})
	if err != nil {
		log.Printf("[%s] send error: %v", label, err)
		return
	}
	log.Printf("[%s] sent to chat %d: %T", label, msg.ChatID, result)
}

// fetchRich fetches a rich message that the /getrich command is sent as a
// reply to. The target message ID comes from msg.ReplyToID.
func fetchRich(ctx *telegram.Context, msg *types.Message) {
	if msg.ReplyToID == 0 {
		_, _ = ctx.Reply("Reply to a rich message with /getrich to fetch it.")
		return
	}

	// messages.getRichMessage is user-account-only; bots get BOT_METHOD_INVALID.
	if ctx.Client.IsBot() {
		_, _ = ctx.Reply("Fetching rich messages (getRichMessage) is not available to bots — it requires a user session.")
		return
	}

	peer, err := ctx.Client.ResolvePeer(ctx.Ctx, msg.ChatID)
	if err != nil {
		log.Printf("[GetRich] resolve peer %d: %v", msg.ChatID, err)
		return
	}

	result, err := ctx.Client.Raw().MessagesGetRichMessage(ctx.Ctx, &tg.MessagesGetRichMessageRequest{
		Peer: peer,
		ID:   msg.ReplyToID,
	})
	if err != nil {
		log.Printf("[GetRich] error: %v", err)
		return
	}
	printFetched(msg.ReplyToID, result)
}

// printFetched summarizes a messages.getRichMessage result.
func printFetched(msgID int32, result tg.MessagesClass) {
	switch v := result.(type) {
	case *tg.MessagesMessages:
		fmt.Printf("[GetRich] id=%d messages=%d\n", msgID, len(v.Messages))
	case *tg.MessagesMessagesSlice:
		fmt.Printf("[GetRich] id=%d count=%d messages=%d\n", msgID, v.Count, len(v.Messages))
	case *tg.MessagesChannelMessages:
		fmt.Printf("[GetRich] id=%d messages=%d\n", msgID, len(v.Messages))
	case *tg.MessagesMessagesNotModified:
		fmt.Printf("[GetRich] id=%d not modified\n", msgID)
	default:
		fmt.Printf("[GetRich] id=%d type=%T\n", msgID, result)
	}
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
