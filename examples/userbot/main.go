package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// userbot demonstrates a full userbot login flow using phone number + code.
//
// It supports three scenarios:
//   1. Phone + code login
//   2. Phone + code + 2FA password
//   3. Reusing a saved session string (no re-login needed)
//
// Required environment variables:
//   - API_ID:   your Telegram API ID
//   - API_HASH: your Telegram API hash
//
// Optional environment variables:
//   - SESSION:  existing session string (skips login if set)
//
// Usage:
//
//	# First run (interactive login):
//	API_ID=12345 API_HASH=abc go run .
//
//	# Subsequent runs with saved session:
//	API_ID=12345 API_HASH=abc SESSION="saved_session_string" go run .
func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	sessionStr := os.Getenv("SESSION")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		SessionName:       "userbot",
		SessionString:     sessionStr,
		InMemory:          sessionStr != "",
		SavePeers:         true,
		DispatchQueueSize: 512,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	client.OnMessage(func(client *telegram.Client, msg *types.Message) {
		if msg.Text == "" {
			return
		}

		switch {
		case strings.HasPrefix(msg.Text, "/ping"):
			msg.Reply("pong")
		case strings.HasPrefix(msg.Text, "/me"):
			me := client.Me()
			if me != nil {
				msg.Reply(fmt.Sprintf("ID: %d\nName: %s\nUsername: @%s", me.ID, me.FirstName, me.Username))
			}
		case strings.HasPrefix(msg.Text, "/session"):
			s, err := client.ExportSessionString()
			if err != nil {
				msg.Reply(fmt.Sprintf("error: %v", err))
				return
			}
			msg.Reply("Session string saved to logs (not sent here for security)")
			log.Printf("SESSION=%s", s)
		}
	}, telegram.Private)

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	me, err := client.GetMe(context.Background())
	if err == nil && me != nil {
		fmt.Println("=== Already Authorized ===")
		printUser(me)
		fmt.Println("userbot is running. Press Ctrl+C to stop.")
		client.Idle()
		return
	}

	fmt.Println("=== Userbot Login ===")
	me, err = loginUser(client)
	if err != nil {
		log.Fatalf("login: %v", err)
	}

	printUser(me)

	sessionString, err := client.ExportSessionString()
	if err != nil {
		log.Printf("warning: could not export session string: %v", err)
	} else {
		fmt.Println("\nSave this session string for future runs:")
		fmt.Printf("  SESSION=%q\n", sessionString)
	}

	fmt.Println("\nuserbot is running. Press Ctrl+C to stop.")
	client.Idle()
}

func loginUser(client *telegram.Client) (*types.User, error) {
	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Phone number (e.g. +1234567890): ")
	phone, _ := reader.ReadString('\n')
	phone = strings.TrimSpace(phone)

	codeResult, err := client.SendCode(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("send code: %w", err)
	}
	fmt.Println("Verification code sent. Check your Telegram app.")

	fmt.Print("Enter code: ")
	code, _ := reader.ReadString('\n')
	code = strings.TrimSpace(code)

	user, err := client.SignIn(ctx, phone, codeResult.PhoneCodeHash, code)
	if err == nil {
		return user, nil
	}

	if err.Error() == telegram.Err2FARequired.Error() {
		fmt.Print("2FA password: ")
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)

		return client.CheckPassword(ctx, password)
	}

	if err.Error() == telegram.ErrSignUpRequired.Error() {
		fmt.Print("First name: ")
		firstName, _ := reader.ReadString('\n')
		firstName = strings.TrimSpace(firstName)

		fmt.Print("Last name (optional): ")
		lastName, _ := reader.ReadString('\n')
		lastName = strings.TrimSpace(lastName)

		return client.SignUp(ctx, phone, codeResult.PhoneCodeHash, firstName, lastName)
	}

	return nil, err
}

func printUser(u *types.User) {
	fmt.Printf("  ID:       %d\n", u.ID)
	fmt.Printf("  Name:     %s", u.FirstName)
	if u.LastName != "" {
		fmt.Printf(" %s", u.LastName)
	}
	fmt.Println()
	if u.Username != "" {
		fmt.Printf("  Username: @%s\n", u.Username)
	}
	fmt.Printf("  Bot:      %v\n", u.IsBot)
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
