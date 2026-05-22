package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")

	logger := telegram.NewLogger("mybot")
	logger.SetLevel(telegram.TraceLevel)
	logger.NoColor(false)

	tmp, err := os.CreateTemp("", "mybot-*.log")
	if err != nil {
		log.Fatalf("temp file: %v", err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	if err := logger.SetFile(tmp.Name(), 10*1024*1024); err != nil {
		log.Fatalf("log file: %v", err)
	}
	defer logger.Close()

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "logger_bot",
		Log: telegram.LogConfig{
			Logger: logger,
		},
	})
	if err != nil {
		logger.Fatal("new client failed")
	}

	client.OnMessage(func(client *telegram.Client, msg *types.Message) {
		if msg == nil || msg.Text == "" {
			return
		}

		client.Log.Infof("incoming message from %d: %s", msg.FromID, msg.Text)

		_, err := msg.Reply(msg.Text)
		if err != nil {
			client.Log.ErrorWithCause(err, "reply failed")
		}
	}, telegram.Private)

	if err := client.Connect(0); err != nil {
		logger.FatalWithCause(err, "connect failed")
	}
	defer client.Stop()

	bot, err := client.GetMe(context.Background())
	if err != nil {
		client.Log.FatalWithCause(err, "get me failed")
	}

	fmt.Printf("bot @%s running, logs: %s\n", bot.Username, tmp.Name())
	fmt.Println("press Ctrl+C to stop")

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
