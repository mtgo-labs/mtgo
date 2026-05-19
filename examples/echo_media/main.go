package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/fileid"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "echo_media",
		SavePeers:   true,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	client.OnMessage(func(client *telegram.Client, msg *types.Message) {
		if msg.Outgoing {
			return
		}

		if fileID := extractFileID(msg); fileID != "" {
			client.Log.Infof("echo media: %s", fileID)
			f := telegram.FileID(fileID)
			if _, err := client.SendDocument(context.Background(), msg.Chat.ID, f, ""); err != nil {
				client.Log.Errorf("echo media: %v", err)
			}
			return
		}

		fileID := strings.TrimSpace(msg.Text)
		if fileID == "" {
			return
		}

		if strings.HasPrefix(fileID, "http://") || strings.HasPrefix(fileID, "https://") {
			client.Log.Infof("echo url: %s", fileID)
			if _, err := client.SendDocument(context.Background(), msg.Chat.ID, telegram.URL(fileID), ""); err != nil {
				client.Log.Errorf("echo url: %v", err)
			}
			return
		}

		client.Log.Infof("echo file_id: %s", fileID)

		decoded, err := fileid.Decode(fileID)
		if err != nil {
			_, _ = msg.Reply(fmt.Sprintf("bad file_id: %v", err))
			return
		}

		f := telegram.FileID(fileID)

		ctx := context.Background()
		if decoded.Type.IsPhoto() {
			_, err = client.SendPhoto(ctx, msg.Chat.ID, f, "")
		} else {
			_, err = client.SendDocument(ctx, msg.Chat.ID, f, "")
		}
		if err != nil {
			_, _ = msg.Reply(fmt.Sprintf("error: %v", err))
		}
	}, telegram.All)

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	bot, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}

	fmt.Printf("@%s — send a file_id, get the media back\n", bot.Username)
	client.Idle()
}

func extractFileID(msg *types.Message) string {
	switch {
	case msg.Photo != nil:
		return msg.Photo.FileID
	case msg.Animation != nil:
		return msg.Animation.FileID
	case msg.Sticker != nil:
		return msg.Sticker.FileID
	case msg.Video != nil:
		return msg.Video.FileID
	case msg.Audio != nil:
		return msg.Audio.FileID
	case msg.Voice != nil:
		return msg.Voice.FileID
	case msg.VideoNote != nil:
		return msg.VideoNote.FileID
	case msg.Document != nil:
		return msg.Document.FileID
	default:
		return ""
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
