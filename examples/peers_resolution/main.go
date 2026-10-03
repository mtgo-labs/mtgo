// Command peers_resolution demonstrates mtgo's peer resolution: the frozen
// Client API (unchanged signatures, now backed by the telegram/peers engine)
// and the richer peers.Manager surface reachable via Client.Peers().
//
// Requirements:
//
//	API_ID, API_HASH — from https://my.telegram.org
//	SESSION          — optional session string; logs in interactively otherwise
//
// Run:
//
//	API_ID=... API_HASH=... go run ./examples/peers_resolution
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/peers"
	"github.com/mtgo-labs/mtgo/tg"
)

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")

	client, err := telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		SessionString: os.Getenv("SESSION"),
		SavePeers:     true, // persist access hashes across restarts
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := client.Connect(30 * time.Second); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Disconnect()

	// ─────────────────────────────────────────────────────────────────────
	// 1. The classic API — unchanged signatures, now with a full cascade:
	//    memory cache → persistent store → RPC fallbacks. These all work on
	//    a cold cache, exactly like Telethon's get_input_entity.
	// ─────────────────────────────────────────────────────────────────────
	for _, input := range []any{
		"@durov",                             // username
		int64(204769871),                     // numeric user ID
		telegram.ChatRefFrom("durov"),        // ref built from a string
		telegram.ChatRefFrom("+15551234567"), // phone (needs contact visibility)
		telegram.ChatPeer(&tg.InputPeerSelf{}),
	} {
		peer, err := client.ResolvePeer(ctx, input)
		if err != nil {
			fmt.Printf("resolve %-24v -> error: %v\n", input, err)
			continue
		}
		fmt.Printf("resolve %-24v -> %T\n", input, peer)
	}

	// Link parsing is uniform: usernames, numeric IDs, t.me/c/<id> private
	// channel posts, and tg:// deep links all become usable refs. Invite
	// links (t.me/+hash) deliberately do NOT resolve — join them instead.
	ref := telegram.ChatRefFrom("https://t.me/c/123456/42")
	fmt.Printf("t.me/c/… link -> %T with ID\n", ref)

	// ─────────────────────────────────────────────────────────────────────
	// 2. Honest errors. Transient failures (flood wait, network, auth) no
	//    longer masquerade as "not found" — only genuine misses do.
	// ─────────────────────────────────────────────────────────────────────
	_, err = client.ResolveUsername(ctx, "this-username-does-not-exist")
	var notFound *peers.NotFoundError
	switch {
	case errors.As(err, &notFound):
		fmt.Printf("not found:   ref=%q (server rejected it)\n", notFound.Ref)
	case errors.Is(err, telegram.ErrPeerNotFound):
		fmt.Printf("not found:   %v\n", err)
	case err != nil:
		fmt.Printf("transient:   %v (retry later — NOT a miss)\n", err)
	}

	// ─────────────────────────────────────────────────────────────────────
	// 3. The engine itself, via Client.Peers(): typed kind resolution.
	// ─────────────────────────────────────────────────────────────────────
	mgr := client.Peers()

	user, err := mgr.InputUser(ctx, peers.Ref{Username: "durov"})
	if err != nil {
		var notUser peers.NotUserError
		if errors.As(err, &notUser) {
			log.Fatalf("resolved, but it's a %T, not a user", notUser.Peer)
		}
		log.Fatalf("InputUser: %v", err)
	}
	fmt.Printf("InputUser:   %+v\n", user)

	channel, err := mgr.InputChannel(ctx, peers.Ref{ID: 1277341946}) // telegram's channel ID
	var notChannel peers.NotChannelError
	switch {
	case err == nil:
		fmt.Printf("InputChannel: %+v\n", channel)
	case errors.As(err, &notChannel):
		fmt.Printf("InputChannel: resolved, but not a channel: %T\n", notChannel.Peer)
	default:
		fmt.Printf("InputChannel: %v\n", err)
	}

	// ─────────────────────────────────────────────────────────────────────
	// 4. Feeding and invalidating the cache manually. Every entity the
	//    client observes (updates, dialogs, resolve responses) is ingested
	//    automatically; you only need Ingest for entities from manual RPCs.
	// ─────────────────────────────────────────────────────────────────────
	res, err := mgr.ResolveUsernameFull(ctx, "telegram")
	if err != nil {
		log.Fatalf("ResolveUsernameFull: %v", err)
	}
	for _, ch := range res.Chats { // full server response, entities included
		if c, ok := ch.(*tg.Channel); ok {
			fmt.Printf("full resolve: @telegram -> channel %d (%s)\n", c.ID, c.Title)
		}
	}

	if username := mgr.LookupUsername(204769871); username != "" {
		fmt.Printf("cached username for 204769871: @%s\n", username)
	}

	// Invalidate is what the stale-hash middleware does automatically on
	// PEER_ID_INVALID; call it yourself when you know a hash went stale.
	// mgr.Invalidate(204769871)

	fmt.Println("done")
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
