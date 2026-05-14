package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/types"
)

const testBotToken = "123456:ABC-DEF"

func buildInitData(t *testing.T, botToken string, authDate int64, extra map[string]string, tamperHash bool) string {
	t.Helper()

	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))
	sk := secretKey.Sum(nil)

	params := map[string]string{
		"query_id":      "AAHdF4e9q",
		"auth_date":     strconv.FormatInt(authDate, 10),
		"chat_instance": "-409651665",
		"chat_type":     "private",
	}
	params["user"] = `{"id":279058397,"first_name":"John","last_name":"Doe","username":"johndoe","language_code":"en"}`
	maps.Copy(params, extra)

	delete(params, "hash")

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+params[k])
	}
	dataCheckString := strings.Join(lines, "\n")

	mac := hmac.New(sha256.New, sk)
	mac.Write([]byte(dataCheckString))
	hash := hex.EncodeToString(mac.Sum(nil))
	if tamperHash {
		hash = "aaaa" + hash[4:]
	}

	params["hash"] = hash

	vals := make(url.Values, len(params))
	for k, v := range params {
		vals.Set(k, v)
	}
	return vals.Encode()
}

func TestCreateWebAppSecretKey(t *testing.T) {
	key := CreateWebAppSecretKey(testBotToken)
	if len(key) != sha256.Size {
		t.Fatalf("expected %d bytes, got %d", sha256.Size, len(key))
	}

	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(testBotToken))
	expected := mac.Sum(nil)
	if !hmac.Equal(key, expected) {
		t.Error("secret key does not match expected HMAC-SHA256")
	}
}

func TestCreateWebAppSecretKey_Deterministic(t *testing.T) {
	k1 := CreateWebAppSecretKey("123:ABC")
	k2 := CreateWebAppSecretKey("123:ABC")
	if !hmac.Equal(k1, k2) {
		t.Error("same token should produce same key")
	}

	k3 := CreateWebAppSecretKey("456:XYZ")
	if hmac.Equal(k1, k3) {
		t.Error("different tokens should produce different keys")
	}
}

func TestParseWebAppData_Valid(t *testing.T) {
	authDate := time.Now().Add(-10 * time.Second).Unix()
	initData := buildInitData(t, testBotToken, authDate, nil, false)

	secretKey := CreateWebAppSecretKey(testBotToken)
	data, err := ParseWebAppData(secretKey, initData, 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.QueryID != "AAHdF4e9q" {
		t.Errorf("QueryID = %q, want %q", data.QueryID, "AAHdF4e9q")
	}
	if data.User.ID != 279058397 {
		t.Errorf("User.ID = %d, want %d", data.User.ID, 279058397)
	}
	if data.User.FirstName != "John" {
		t.Errorf("User.FirstName = %q, want %q", data.User.FirstName, "John")
	}
	if data.User.LastName != "Doe" {
		t.Errorf("User.LastName = %q, want %q", data.User.LastName, "Doe")
	}
	if data.User.Username != "johndoe" {
		t.Errorf("User.Username = %q, want %q", data.User.Username, "johndoe")
	}
	if data.User.LanguageCode != "en" {
		t.Errorf("User.LanguageCode = %q, want %q", data.User.LanguageCode, "en")
	}
	if data.ChatType != "private" {
		t.Errorf("ChatType = %q, want %q", data.ChatType, "private")
	}
	if data.ChatInstance != "-409651665" {
		t.Errorf("ChatInstance = %q, want %q", data.ChatInstance, "-409651665")
	}
	if data.AuthDate.Unix() != authDate {
		t.Errorf("AuthDate.Unix() = %d, want %d", data.AuthDate.Unix(), authDate)
	}
	if data.Hash == "" {
		t.Error("Hash should not be empty")
	}
}

func TestParseWebAppData_WithStartParam(t *testing.T) {
	authDate := time.Now().Add(-5 * time.Second).Unix()
	initData := buildInitData(t, testBotToken, authDate, map[string]string{
		"start_param": "debug",
	}, false)

	secretKey := CreateWebAppSecretKey(testBotToken)
	data, err := ParseWebAppData(secretKey, initData, 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.StartParam != "debug" {
		t.Errorf("StartParam = %q, want %q", data.StartParam, "debug")
	}
}

func TestParseWebAppData_Outdated(t *testing.T) {
	authDate := time.Now().Add(-120 * time.Second).Unix()
	initData := buildInitData(t, testBotToken, authDate, nil, false)

	secretKey := CreateWebAppSecretKey(testBotToken)
	_, err := ParseWebAppData(secretKey, initData, 60*time.Second)
	if err == nil {
		t.Fatal("expected error for outdated data")
	}
	if !isWebAppError(err, types.ErrWebAppDataOutdated) {
		t.Errorf("expected ErrWebAppDataOutdated, got %v", err)
	}
}

func TestParseWebAppData_NoMaxAge(t *testing.T) {
	authDate := time.Now().Add(-9999 * time.Second).Unix()
	initData := buildInitData(t, testBotToken, authDate, nil, false)

	secretKey := CreateWebAppSecretKey(testBotToken)
	data, err := ParseWebAppData(secretKey, initData, 0)
	if err != nil {
		t.Fatalf("unexpected error with maxAge=0: %v", err)
	}
	if data.AuthDate.Unix() != authDate {
		t.Errorf("AuthDate = %d, want %d", data.AuthDate.Unix(), authDate)
	}
}

func TestParseWebAppData_HashMismatch(t *testing.T) {
	authDate := time.Now().Add(-5 * time.Second).Unix()
	initData := buildInitData(t, testBotToken, authDate, nil, true)

	secretKey := CreateWebAppSecretKey(testBotToken)
	_, err := ParseWebAppData(secretKey, initData, 60*time.Second)
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}
	if !isWebAppError(err, types.ErrWebAppDataMismatch) {
		t.Errorf("expected ErrWebAppDataMismatch, got %v", err)
	}
}

func TestParseWebAppData_WrongBotToken(t *testing.T) {
	authDate := time.Now().Add(-5 * time.Second).Unix()
	initData := buildInitData(t, testBotToken, authDate, nil, false)

	secretKey := CreateWebAppSecretKey("999:WRONG")
	_, err := ParseWebAppData(secretKey, initData, 60*time.Second)
	if err == nil {
		t.Fatal("expected error with wrong bot token")
	}
	if !isWebAppError(err, types.ErrWebAppDataMismatch) {
		t.Errorf("expected ErrWebAppDataMismatch, got %v", err)
	}
}

func TestParseWebAppData_MissingHash(t *testing.T) {
	initData := "query_id=AAHdF4e9q&auth_date=1234567890&user=%7B%22id%22%3A1%7D"
	secretKey := CreateWebAppSecretKey(testBotToken)
	_, err := ParseWebAppData(secretKey, initData, 60*time.Second)
	if err == nil {
		t.Fatal("expected error for missing hash")
	}
	if !isWebAppError(err, types.ErrWebAppDataInvalid) {
		t.Errorf("expected ErrWebAppDataInvalid, got %v", err)
	}
}

func TestParseWebAppData_MissingAuthDate(t *testing.T) {
	initData := "query_id=AAHdF4e9q&hash=abc123&user=%7B%22id%22%3A1%7D"
	secretKey := CreateWebAppSecretKey(testBotToken)
	_, err := ParseWebAppData(secretKey, initData, 60*time.Second)
	if err == nil {
		t.Fatal("expected error for missing auth_date")
	}
	if !isWebAppError(err, types.ErrWebAppDataInvalid) {
		t.Errorf("expected ErrWebAppDataInvalid, got %v", err)
	}
}

func TestParseWebAppData_EmptyInitData(t *testing.T) {
	secretKey := CreateWebAppSecretKey(testBotToken)
	_, err := ParseWebAppData(secretKey, "", 60*time.Second)
	if err == nil {
		t.Fatal("expected error for empty init data")
	}
	if !isWebAppError(err, types.ErrWebAppDataInvalid) {
		t.Errorf("expected ErrWebAppDataInvalid, got %v", err)
	}
}

func TestParseWebAppData_EmptySecretKey(t *testing.T) {
	_, err := ParseWebAppData(nil, "some=data", 60*time.Second)
	if err == nil {
		t.Fatal("expected error for empty secret key")
	}
	if !isWebAppError(err, types.ErrWebAppDataInvalid) {
		t.Errorf("expected ErrWebAppDataInvalid, got %v", err)
	}
}

func TestValidateWebAppData(t *testing.T) {
	authDate := time.Now().Add(-5 * time.Second).Unix()
	initData := buildInitData(t, testBotToken, authDate, map[string]string{
		"start_param": "test",
	}, false)

	data, err := ValidateWebAppData(testBotToken, initData, 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.QueryID != "AAHdF4e9q" {
		t.Errorf("QueryID = %q, want %q", data.QueryID, "AAHdF4e9q")
	}
	if data.StartParam != "test" {
		t.Errorf("StartParam = %q, want %q", data.StartParam, "test")
	}
	if data.User.ID != 279058397 {
		t.Errorf("User.ID = %d, want %d", data.User.ID, 279058397)
	}
}

func TestWebAppInitData_RawPreserved(t *testing.T) {
	authDate := time.Now().Add(-5 * time.Second).Unix()
	initData := buildInitData(t, testBotToken, authDate, nil, false)

	secretKey := CreateWebAppSecretKey(testBotToken)
	data, err := ParseWebAppData(secretKey, initData, 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := data.Raw["query_id"]; !ok {
		t.Error("Raw map missing query_id")
	}
	if _, ok := data.Raw["user"]; !ok {
		t.Error("Raw map missing user")
	}
	if _, ok := data.Raw["hash"]; !ok {
		t.Error("Raw map should include hash")
	}
	if data.Raw["chat_type"] != "private" {
		t.Errorf("Raw[chat_type] = %q, want %q", data.Raw["chat_type"], "private")
	}
}

func TestWebAppInitData_IsOutdated(t *testing.T) {
	data := &types.WebAppInitData{
		AuthDate: time.Now().Add(-120 * time.Second),
	}
	if !data.IsOutdated(60 * time.Second) {
		t.Error("expected data to be outdated")
	}

	data.AuthDate = time.Now().Add(-10 * time.Second)
	if data.IsOutdated(60 * time.Second) {
		t.Error("expected data to be fresh")
	}
}

func TestWebAppInitData_PremiumUser(t *testing.T) {
	authDate := time.Now().Add(-5 * time.Second).Unix()
	userJSON := `{"id":123456,"first_name":"Premium","last_name":"User","username":"premium_user","language_code":"en","is_premium":true,"photo_url":"https://example.com/photo.jpg"}`
	secretKey := CreateWebAppSecretKey(testBotToken)

	params := map[string]string{
		"query_id":      "TEST",
		"auth_date":     fmt.Sprintf("%d", authDate),
		"chat_instance": "123",
		"chat_type":     "private",
		"user":          userJSON,
	}
	initData := buildSignedInitData(t, secretKey, params)

	data, err := ParseWebAppData(secretKey, initData, 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !data.User.IsPremium {
		t.Error("expected IsPremium to be true")
	}
	if data.User.PhotoURL != "https://example.com/photo.jpg" {
		t.Errorf("PhotoURL = %q, want %q", data.User.PhotoURL, "https://example.com/photo.jpg")
	}
}

func buildSignedInitData(t *testing.T, secretKey []byte, params map[string]string) string {
	t.Helper()

	delete(params, "hash")

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+params[k])
	}
	dataCheckString := strings.Join(lines, "\n")

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString))
	hash := hex.EncodeToString(mac.Sum(nil))

	params["hash"] = hash

	vals := make(url.Values, len(params))
	for k, v := range params {
		vals.Set(k, v)
	}
	return vals.Encode()
}

func isWebAppError(err error, target error) bool {
	return strings.Contains(err.Error(), target.Error())
}
