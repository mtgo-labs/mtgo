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
	outputDir := os.Getenv("DOWNLOAD_DIR")
	if outputDir == "" {
		outputDir = "./downloads"
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "download_bot",
		SavePeers:   true,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	client.OnMessage(func(client *telegram.Client, msg *types.Message) {
		if msg == nil || msg.Media == nil {
			return
		}

		fileName := resolveFileName(msg)
		destPath := filepath.Join(outputDir, fileName)

		switch m := msg.Media.(type) {
		case *types.PhotoMedia:
			fmt.Printf("Downloading photo (msg_id=%d)...\n", msg.ID)

			data, err := client.DownloadMedia(context.Background(), m, "", &params.Download{
				Progress: func(info params.ProgressInfo) {
					fmt.Printf("  progress: %d / %d bytes\n", info.DownloadedBytes, info.TotalBytes)
				},
			})
			if err != nil {
				log.Printf("download photo: %v", err)
				return
			}

			if err := os.WriteFile(destPath+".jpg", data, 0o644); err != nil {
				log.Printf("save photo: %v", err)
				return
			}
			fmt.Printf("  saved: %s (%d bytes)\n", destPath+".jpg", len(data))

		case *types.DocumentMedia:
			fmt.Printf("Downloading document %q (msg_id=%d, %d bytes)...\n", m.FileName, msg.ID, m.FileSize)

			err := client.DownloadMediaToFile(context.Background(), m, "", destPath, m.FileSize, &params.Download{
				Progress: func(info params.ProgressInfo) {
					fmt.Printf("  progress: %d / %d bytes\n", info.DownloadedBytes, info.TotalBytes)
				},
			})
			if err != nil {
				log.Printf("download document: %v", err)
				return
			}
			fmt.Printf("  saved: %s\n", destPath)
		}
	}, telegram.Media)

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	fmt.Println("download bot is running — send a file to download it")
	fmt.Println("press Ctrl+C to stop")

	client.Idle()
}

func resolveFileName(msg *types.Message) string {
	switch m := msg.Media.(type) {
	case *types.DocumentMedia:
		if m.FileName != "" {
			return m.FileName
		}
		ext := guessExt(m.MimeType)
		return fmt.Sprintf("file_%d%s", msg.ID, ext)
	case *types.PhotoMedia:
		return fmt.Sprintf("photo_%d", msg.ID)
	default:
		return fmt.Sprintf("media_%d", msg.ID)
	}
}

func guessExt(mime string) string {
	switch mime {
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "audio/mpeg":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "audio/opus":
		return ".opus"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	default:
		if strings.HasPrefix(mime, "video/") {
			return ".mp4"
		}
		if strings.HasPrefix(mime, "audio/") {
			return ".bin"
		}
		return ".bin"
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
