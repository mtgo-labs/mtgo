package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/types"
)

var webAppSecretKey = []byte("WebAppData")

// CreateWebAppSecretKey derives the HMAC-SHA256 secret key used to verify WebApp init data.
// The key is computed as HMAC-SHA256("WebAppData", botToken) per the Telegram WebApp spec.
// Use this when you need to validate init data across multiple requests without re-deriving
// the key each time.
//
// Example:
//
//	secretKey := telegram.CreateWebAppSecretKey("123456:ABC-DEF")
//	data, err := telegram.ParseWebAppData(secretKey, initData, 5*time.Minute)
func CreateWebAppSecretKey(botToken string) []byte {
	mac := hmac.New(sha256.New, webAppSecretKey)
	mac.Write([]byte(botToken))
	return mac.Sum(nil)
}

// ParseWebAppData validates and parses Telegram WebApp init data using the provided secret key.
// It verifies the HMAC-SHA256 hash, checks the auth_date freshness against maxAge, and
// deserializes the user object from the embedded JSON.
//
// Returns ErrWebAppDataOutdated if maxAge > 0 and the data is older than maxAge.
// Returns ErrWebAppDataMismatch if the computed hash does not match the provided hash.
// Returns ErrWebAppDataInvalid if the data is malformed or missing required fields.
//
// Example:
//
//	secretKey := telegram.CreateWebAppSecretKey(botToken)
//	data, err := telegram.ParseWebAppData(secretKey, initData, 5*time.Minute)
//	if err != nil {
//	    http.Error(w, "invalid webapp data", http.StatusUnauthorized)
//	    return
//	}
//	fmt.Printf("User ID: %d", data.User.ID)
func ParseWebAppData(secretKey []byte, initData string, maxAge time.Duration) (*types.WebAppInitData, error) {
	if len(secretKey) == 0 {
		return nil, fmt.Errorf("%w: empty secret key", types.ErrWebAppDataInvalid)
	}
	if initData == "" {
		return nil, fmt.Errorf("%w: empty init data", types.ErrWebAppDataInvalid)
	}

	vals, err := url.ParseQuery(initData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrWebAppDataInvalid, err)
	}

	raw := make(map[string]string, len(vals))
	for k, v := range vals {
		raw[k] = v[0]
	}

	hash, ok := raw["hash"]
	if !ok || hash == "" {
		return nil, fmt.Errorf("%w: missing hash", types.ErrWebAppDataInvalid)
	}
	authDateStr, ok := raw["auth_date"]
	if !ok || authDateStr == "" {
		return nil, fmt.Errorf("%w: missing auth_date", types.ErrWebAppDataInvalid)
	}

	authDateUnix, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid auth_date: %v", types.ErrWebAppDataInvalid, err)
	}
	authDate := time.Unix(authDateUnix, 0)

	if maxAge > 0 && time.Since(authDate) > maxAge {
		return nil, fmt.Errorf("%w: auth_date %d", types.ErrWebAppDataOutdated, authDateUnix)
	}

	delete(raw, "hash")

	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+raw[k])
	}
	dataCheckString := strings.Join(lines, "\n")

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedHash), []byte(hash)) {
		return nil, types.ErrWebAppDataMismatch
	}

	raw["hash"] = hash

	result := &types.WebAppInitData{
		Hash:         hash,
		AuthDate:     authDate,
		ChatInstance: raw["chat_instance"],
		ChatType:     raw["chat_type"],
		StartParam:   raw["start_param"],
		QueryID:      raw["query_id"],
		Raw:          raw,
	}

	if userJSON, ok := raw["user"]; ok && userJSON != "" {
		if err := json.Unmarshal([]byte(userJSON), &result.User); err != nil {
			return nil, fmt.Errorf("%w: invalid user JSON: %v", types.ErrWebAppDataInvalid, err)
		}
	}

	return result, nil
}

// ValidateWebAppData validates and parses Telegram WebApp init data using a bot token.
// It derives the secret key from the token and delegates to ParseWebAppData.
// This is a convenience wrapper for one-shot validation when you don't need to
// cache the derived secret key.
//
// Example:
//
//	data, err := telegram.ValidateWebAppData(botToken, initData, 5*time.Minute)
//	if err != nil {
//	    http.Error(w, "invalid webapp data", http.StatusUnauthorized)
//	    return
//	}
//	fmt.Printf("User ID: %d", data.User.ID)
func ValidateWebAppData(botToken string, initData string, maxAge time.Duration) (*types.WebAppInitData, error) {
	secretKey := CreateWebAppSecretKey(botToken)
	return ParseWebAppData(secretKey, initData, maxAge)
}
