package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")
	chatID := mustAtoi64(mustEnv("CHAT_ID"))

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "upload_bot",
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

	// ── Upload + SendDocument ─────────────────────────────────

	msg, err := client.SendDocument(ctx, chatID,
		telegram.Path("report.pdf"),
		"uploaded document from disk",
		&params.SendDocument{
			MimeType: "application/pdf",
		},
	)
	printResult("SendDocument (path)", msg, err)

	// ── Upload + SendPhoto ────────────────────────────────────

	msg, err = client.SendPhoto(ctx, chatID,
		telegram.Path("photo.jpg"),
		"uploaded photo from disk",
	)
	printResult("SendPhoto (path)", msg, err)

	// ── Upload + SendVideo ────────────────────────────────────

	msg, err = client.SendVideo(ctx, chatID,
		telegram.Path("clip.mp4"),
		"uploaded video from disk",
		&params.SendVideo{
			Duration: 12.5,
			Width:    1280,
			Height:   720,
			FileName: "demo.mp4",
		},
	)
	printResult("SendVideo (path)", msg, err)

	// ── Upload + SendAudio ────────────────────────────────────

	msg, err = client.SendAudio(ctx, chatID,
		telegram.Path("song.mp3"),
		"uploaded audio from disk",
		&params.SendAudio{
			Duration:  245,
			Performer: "Artist",
			Title:     "Track Name",
		},
	)
	printResult("SendAudio (path)", msg, err)

	// ── Upload + SendAnimation ────────────────────────────────

	msg, err = client.SendAnimation(ctx, chatID,
		telegram.Path("meme.gif"),
		"uploaded gif from disk",
	)
	printResult("SendAnimation (path)", msg, err)

	// ── Upload from URL ───────────────────────────────────────

	msg, err = client.SendPhoto(ctx, chatID,
		telegram.URL("https://example.com/photo.jpg"),
		"photo from URL",
	)
	printResult("SendPhoto (url)", msg, err)

	// ── Upload from bytes ─────────────────────────────────────

	msg, err = client.SendDocument(ctx, chatID,
		telegram.FromBytes([]byte("hello world"), "hello.txt"),
		"document from bytes",
	)
	printResult("SendDocument (bytes)", msg, err)

	// ── Low-level UploadFile ──────────────────────────────────

	uploadPath := "large_file.zip"
	f2, err := os.Open(uploadPath)
	if err != nil {
		log.Fatalf("open file: %v", err)
	}
	defer f2.Close()

	info2, err := f2.Stat()
	if err != nil {
		log.Fatalf("stat file: %v", err)
	}

	result, err := client.UploadFile(ctx, f2, filepath.Base(uploadPath), info2.Size(), &telegram.UploadOptions{
		Workers: 4,
		Progress: func(info params.ProgressInfo) {
			fmt.Printf("  upload progress: %d / %d bytes\n", info.UploadedBytes, info.TotalBytes)
		},
	})
	if err != nil {
		log.Fatalf("upload file: %v", err)
	}
	fmt.Printf("[ OK ] UploadFile: name=%q size=%d big=%v\n", result.Name, result.Size, result.IsBig)

	fmt.Println("\nAll upload examples completed.")
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
		log.Fatalf("environment variable %s is required", key)
	}
	return v
}

func mustAtoi(s string) int32 {
	s = strings.TrimSpace(s)
	var n int32
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		log.Fatalf("invalid integer %q: %v", s, err)
	}
	return n
}

func mustAtoi64(s string) int64 {
	s = strings.TrimSpace(s)
	var n int64
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		log.Fatalf("invalid integer %q: %v", s, err)
	}
	return n
}
