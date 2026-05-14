package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	tg "github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")
	chatID := mustAtoi(mustEnv("CHAT_ID"))

	client, err := tg.NewClient(int32(mustAtoi(apiID)), apiHash, &tg.Config{
		BotToken:    botToken,
		SessionName: "send_media_bot",
		SavePeers:   true,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Disconnect()

	var msg *types.Message

	// ── SendPhoto ────────────────────────────────────────────

	// from file path
	msg, err = client.SendPhoto(ctx, chatID,
		tg.Path("photo.jpg"),
		"photo from disk",
		&params.SendPhoto{ReplyToMessageID: 1},
	)
	printResult("SendPhoto (path)", msg, err)

	// from URL
	msg, err = client.SendPhoto(ctx, chatID,
		tg.URL("https://example.com/photo.jpg"),
		"photo from URL",
	)
	printResult("SendPhoto (url)", msg, err)

	// from file_id (already uploaded)
	msg, err = client.SendPhoto(ctx, chatID,
		tg.FileID("AgACAgIAAxkBAAI..."),
		"photo by file_id",
	)
	printResult("SendPhoto (file_id)", msg, err)

	// from bytes
	msg, err = client.SendPhoto(ctx, chatID,
		tg.FromBytes([]byte("...png bytes..."), "pic.png"),
		"photo from bytes",
	)
	printResult("SendPhoto (bytes)", msg, err)

	// ── SendVideo ────────────────────────────────────────────

	msg, err = client.SendVideo(ctx, chatID,
		tg.Path("clip.mp4"),
		"video from disk",
		&params.SendVideo{
			Duration: 12.5,
			Width:    1280,
			Height:   720,
			FileName: "demo.mp4",
		},
	)
	printResult("SendVideo (path)", msg, err)

	msg, err = client.SendVideo(ctx, chatID,
		tg.URL("https://example.com/clip.mp4"),
		"video from URL",
		&params.SendVideo{SupportsStreaming: true},
	)
	printResult("SendVideo (url)", msg, err)

	msg, err = client.SendVideo(ctx, chatID,
		tg.FromIDs(123456, 789, nil),
		"video by file_id",
	)
	printResult("SendVideo (file_id)", msg, err)

	// ── SendAudio ────────────────────────────────────────────

	msg, err = client.SendAudio(ctx, chatID,
		tg.Path("song.mp3"),
		"here is a song",
		&params.SendAudio{
			Duration:  245,
			Performer: "Artist",
			Title:     "Track Name",
		},
	)
	printResult("SendAudio (path)", msg, err)

	msg, err = client.SendAudio(ctx, chatID,
		tg.URL("https://example.com/track.mp3"),
		"audio from URL",
	)
	printResult("SendAudio (url)", msg, err)

	// ── SendDocument ─────────────────────────────────────────

	msg, err = client.SendDocument(ctx, chatID,
		tg.Path("report.pdf"),
		"here is the report",
		&params.SendDocument{
			MimeType: "application/pdf",
		},
	)
	printResult("SendDocument (path)", msg, err)

	msg, err = client.SendDocument(ctx, chatID,
		tg.URL("https://example.com/file.zip"),
		"document from URL",
	)
	printResult("SendDocument (url)", msg, err)

	// ── SendAnimation ────────────────────────────────────────

	msg, err = client.SendAnimation(ctx, chatID,
		tg.Path("meme.gif"),
		"funny gif",
	)
	printResult("SendAnimation (path)", msg, err)

	msg, err = client.SendAnimation(ctx, chatID,
		tg.URL("https://example.com/cat.gif"),
		"gif from URL",
		&params.SendAnimation{FileName: "cat.gif"},
	)
	printResult("SendAnimation (url)", msg, err)

	// ── SendVoice ────────────────────────────────────────────

	msg, err = client.SendVoice(ctx, chatID,
		tg.Path("voice.ogg"),
		"voice message",
		&params.SendVoice{Duration: 15},
	)
	printResult("SendVoice (path)", msg, err)

	msg, err = client.SendVoice(ctx, chatID,
		tg.FromBytes([]byte("...ogg bytes..."), "note.ogg"),
		"voice from bytes",
	)
	printResult("SendVoice (bytes)", msg, err)

	// ── SendVideoNote ────────────────────────────────────────
	// round video notes — no caption

	msg, err = client.SendVideoNote(ctx, chatID,
		tg.Path("round.mp4"),
		&params.SendVideoNote{Duration: 6.0},
	)
	printResult("SendVideoNote (path)", msg, err)

	msg, err = client.SendVideoNote(ctx, chatID,
		tg.FromIDs(111, 222, nil),
	)
	printResult("SendVideoNote (file_id)", msg, err)

	// ── SendSticker ──────────────────────────────────────────
	// stickers — no caption

	msg, err = client.SendSticker(ctx, chatID,
		tg.Path("sticker.webp"),
	)
	printResult("SendSticker (path)", msg, err)

	msg, err = client.SendSticker(ctx, chatID,
		tg.FileID("CAACAgIAAxkBAAI..."),
		&params.SendSticker{ReplyToMessageID: 5},
	)
	printResult("SendSticker (file_id)", msg, err)

	msg, err = client.SendSticker(ctx, chatID,
		tg.URL("https://example.com/sticker.webp"),
	)
	printResult("SendSticker (url)", msg, err)

	fmt.Println("\nAll send media examples completed.")
}

func printResult(label string, msg *types.Message, err error) {
	if err != nil {
		fmt.Printf("[FAIL] %s: %v\n", label, err)
		return
	}
	fmt.Printf("[ OK ] %s: msg_id=%d\n", label, msg.ID)
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("env %s is required", key)
	}
	return v
}

func mustAtoi(s string) int64 {
	s = strings.TrimSpace(s)
	var n int64
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		log.Fatalf("invalid integer %q: %v", s, err)
	}
	return n
}
