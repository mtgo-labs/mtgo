package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mtgo-labs/mtgo/telegram"
)

var client *telegram.Client

func main() {
	apiID := mustEnv("API_ID")
	apiHash := mustEnv("API_HASH")
	botToken := mustEnv("BOT_TOKEN")
	port := envOrDefault("PORT", "8080")

	var err error
	client, err = telegram.NewClient(mustAtoi(apiID), apiHash, &telegram.Config{
		BotToken:    botToken,
		SessionName: "webapp_bot",
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	if err := client.Connect(0); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Stop()

	bot, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get me: %v", err)
	}
	fmt.Printf("bot @%s connected\n", bot.Username)

	secretKey := telegram.CreateWebAppSecretKey(botToken)

	http.HandleFunc("/validate", func(w http.ResponseWriter, r *http.Request) {
		initData := r.URL.Query().Get("init_data")
		if initData == "" {
			http.Error(w, "missing init_data parameter", http.StatusBadRequest)
			return
		}

		data, err := telegram.ParseWebAppData(secretKey, initData, 5*time.Minute)
		if err != nil {
			http.Error(w, fmt.Sprintf("validation failed: %v", err), http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_id":     data.User.ID,
			"first_name":  data.User.FirstName,
			"username":    data.User.Username,
			"auth_date":   data.AuthDate,
			"query_id":    data.QueryID,
			"chat_type":   data.ChatType,
			"start_param": data.StartParam,
		})
	})

	http.HandleFunc("/validate-simple", func(w http.ResponseWriter, r *http.Request) {
		initData := r.URL.Query().Get("init_data")
		if initData == "" {
			http.Error(w, "missing init_data parameter", http.StatusBadRequest)
			return
		}

		data, err := telegram.ValidateWebAppData(botToken, initData, 5*time.Minute)
		if err != nil {
			http.Error(w, fmt.Sprintf("validation failed: %v", err), http.StatusUnauthorized)
			return
		}

		fmt.Fprintf(w, "authenticated user %d (@%s)", data.User.ID, data.User.Username)
	})

	fmt.Printf("webapp validation server listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("environment variable %s is required", key)
	}
	return v
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustAtoi(s string) int32 {
	var n int32
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		log.Fatalf("invalid integer %q: %v", s, err)
	}
	return n
}
