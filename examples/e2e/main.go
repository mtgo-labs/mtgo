package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg/e2e"
)

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	phoneNumber := mustEnv("PHONE_NUMBER")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		SessionName: "e2e_example",
		PhoneNumber: phoneNumber,
	})
	if err != nil {
		log.Fatalf("create client: %v", err)
	}
	defer client.Disconnect()

	client.OnSecretChatRequest(func(chat *telegram.SecretChat) bool {
		fmt.Printf("Incoming secret chat request from user %d\n", chat.AdminID)
		fmt.Printf("Key visualization: %v\n", chat.Visualization())
		fmt.Print("Accept? [y/N]: ")

		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			_ = client.DiscardSecretChat(context.Background(), chat.ID, false)
			return false
		}

		_, err := client.AcceptSecretChat(context.Background(), chat.ID)
		if err != nil {
			log.Printf("accept secret chat: %v", err)
			return false
		}
		fmt.Println("Secret chat accepted!")
		return true
	})

	client.OnSecretMessage(func(chat *telegram.SecretChat, layer *e2e.DecryptedMessageLayer) {
		switch msg := layer.Message.(type) {
		case *e2e.DecryptedMessageTL:
			fmt.Printf("[secret:%d] %s\n", chat.ID, msg.Message)
		case *e2e.DecryptedMessageService:
			fmt.Printf("[secret:%d] (service message)\n", chat.ID)
		default:
			fmt.Printf("[secret:%d] (unknown message type %T)\n", chat.ID, msg)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = ctx
	if err := client.Start(); err != nil {
		log.Fatalf("start: %v", err)
	}

	fmt.Println("E2E example running. Press Ctrl+C to exit.")
	fmt.Println("Use RequestSecretChat to initiate a secret chat with another user.")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Shutting down...")
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
	fmt.Sscanf(s, "%d", &n)
	return n
}
