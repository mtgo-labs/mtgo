package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// raw_invoke demonstrates using the raw RPC client (client.Raw()) to call
// multiple Telegram MTProto methods directly without high-level wrappers.
//
// Usage:
//
//	# With session string (no api_id/api_hash needed):
//	SESSION_STRING="..." go run examples/raw_invoke/main.go
//
//	# With bot token:
//	API_ID=... API_HASH=... BOT_TOKEN="..." go run examples/raw_invoke/main.go
func main() {
	client, err := createClient()
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	ctx := context.Background()
	rpc := client.Raw()

	fmt.Println("=== Raw Invoke Demo ===")
	fmt.Println()

	// 1. UsersGetFullUser — fetch own user profile
	me, err := fetchSelf(ctx, rpc)
	if err != nil {
		log.Fatalf("fetchSelf: %v", err)
	}
	fmt.Printf("[UsersGetFullUser]   ID=%-12d Name=%s  @%s\n", me.ID, me.FirstName, me.Username)

	// 2. MessagesGetDialogs — count dialogs (user accounts only, bots get BOT_METHOD_INVALID)
	count, err := fetchDialogCount(ctx, rpc)
	if err != nil {
		fmt.Printf("[MessagesGetDialogs] skipped: %v\n", err)
	} else {
		fmt.Printf("[MessagesGetDialogs] %d dialogs\n", count)
	}

	// 3. HelpGetConfig — fetch server config
	config, err := rpc.HelpGetConfig(ctx)
	if err != nil {
		log.Printf("[HelpGetConfig] error: %v", err)
	} else {
		fmt.Printf("[HelpGetConfig]      DC=%d  date=%d\n", config.ThisDC, config.Date)
	}

	// 4. UpdatesGetState — fetch update state
	state, err := rpc.UpdatesGetState(ctx)
	if err != nil {
		log.Printf("[UpdatesGetState] error: %v", err)
	} else {
		fmt.Printf("[UpdatesGetState]    pts=%d  qts=%d  date=%d\n", state.PTS, state.Qts, state.Date)
	}

	// 5. MessagesSendMessage — send a raw message
	sent, err := rpc.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:     &tg.InputPeerSelf{},
		Message:  "Hello from raw invoke!",
		RandomID: client.RandomID(),
	})
	if err != nil {
		log.Printf("[MessagesSendMessage] error: %v", err)
	} else {
		fmt.Printf("[MessagesSendMessage] sent: %T\n", sent)
	}

	fmt.Println()
	fmt.Println("Done. Press Ctrl+C to stop.")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}

func createClient() (*telegram.Client, error) {
	if s := os.Getenv("SESSION_STRING"); s != "" {
		return telegram.NewClient(0, "", &telegram.Config{
			SessionString: s,
			InMemory:      true,
		})
	}

	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := os.Getenv("BOT_TOKEN")
	phone := os.Getenv("PHONE_NUMBER")

	cfg := &telegram.Config{
		SessionName: "raw_invoke",
		SavePeers:   true,
	}
	if botToken != "" {
		cfg.BotToken = botToken
	} else if phone != "" {
		cfg.PhoneNumber = phone
		cfg.PhoneCode = os.Getenv("PHONE_CODE")
		cfg.Password = os.Getenv("PASSWORD")
	}

	return telegram.NewClient(mustAtoi(apiID), apiHash, cfg)
}

func fetchSelf(ctx context.Context, rpc *tg.RPCClient) (*types.User, error) {
	result, err := rpc.UsersGetFullUser(ctx, &tg.UsersGetFullUserRequest{
		ID: &tg.InputUserSelf{},
	})
	if err != nil {
		return nil, fmt.Errorf("UsersGetFullUser: %w", err)
	}

	uf, ok := result.(*tg.UsersUserFull)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", result)
	}
	if len(uf.Users) == 0 {
		return nil, fmt.Errorf("no users in response")
	}
	return types.ParseUser(uf.Users[0]), nil
}

func fetchDialogCount(ctx context.Context, rpc *tg.RPCClient) (int, error) {
	result, err := rpc.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetDate: 0,
		OffsetID:   0,
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      1,
	})
	if err != nil {
		return 0, fmt.Errorf("MessagesGetDialogs: %w", err)
	}

	switch dialogs := result.(type) {
	case *tg.MessagesDialogs:
		return len(dialogs.Dialogs), nil
	case *tg.MessagesDialogsSlice:
		return int(dialogs.Count), nil
	case *tg.MessagesDialogsNotModified:
		return -1, nil
	default:
		return 0, fmt.Errorf("unexpected type %T", result)
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
