package types

import (
	"testing"
	"time"

	tl "github.com/mtgo-labs/mtgo/tg"
)

func TestParseUser_Nil(t *testing.T) {
	if ParseUser(nil) != nil {
		t.Error("ParseUser(nil) should return nil")
	}
}

func TestParseUser_BasicFields(t *testing.T) {
	raw := &tl.User{
		ID:        12345,
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
		Phone:     "+1234567890",
		Bot:       false,
		Verified:  true,
		Premium:   true,
	}
	u := ParseUser(raw)
	if u == nil {
		t.Fatal("ParseUser returned nil")
	}
	if u.ID != 12345 {
		t.Errorf("ID = %d, want 12345", u.ID)
	}
	if u.FirstName != "John" {
		t.Errorf("FirstName = %q, want %q", u.FirstName, "John")
	}
	if u.LastName != "Doe" {
		t.Errorf("LastName = %q, want %q", u.LastName, "Doe")
	}
	if u.Username != "johndoe" {
		t.Errorf("Username = %q, want %q", u.Username, "johndoe")
	}
	if u.Phone != "+1234567890" {
		t.Errorf("Phone = %q, want %q", u.Phone, "+1234567890")
	}
	if u.IsBot {
		t.Error("IsBot should be false")
	}
	if !u.IsVerified {
		t.Error("IsVerified should be true")
	}
	if !u.IsPremium {
		t.Error("IsPremium should be true")
	}
}

func TestParseUser_BotUser(t *testing.T) {
	raw := &tl.User{ID: 987654321, FirstName: "TestBot", Bot: true}
	u := ParseUser(raw)
	if u == nil {
		t.Fatal("ParseUser returned nil")
	}
	if !u.IsBot {
		t.Error("IsBot should be true")
	}
}

func TestParseUser_StatusOnline(t *testing.T) {
	raw := &tl.User{
		ID:     1,
		Status: &tl.UserStatusOnline{Expires: int32(time.Now().Add(1 * time.Hour).Unix())},
	}
	u := ParseUser(raw)
	if u.Status != UserStatusOnline {
		t.Errorf("Status = %q, want %q", u.Status, UserStatusOnline)
	}
}

func TestParseUser_StatusOffline(t *testing.T) {
	raw := &tl.User{
		ID:     1,
		Status: &tl.UserStatusOffline{WasOnline: int32(time.Now().Add(-1 * time.Hour).Unix())},
	}
	u := ParseUser(raw)
	if u.Status != UserStatusRecently {
		t.Errorf("Status = %q, want %q", u.Status, UserStatusRecently)
	}
}

func TestParseUser_Empty(t *testing.T) {
	u := ParseUser(&tl.UserEmpty{ID: 42})
	if u == nil {
		t.Fatal("ParseUser returned nil for UserEmpty")
	}
	if u.ID != 42 {
		t.Errorf("ID = %d, want 42", u.ID)
	}
}

func TestParseUser_Deleted(t *testing.T) {
	raw := &tl.User{ID: 1, Deleted: true}
	u := ParseUser(raw)
	if !u.IsDeleted {
		t.Error("IsDeleted should be true")
	}
}

func TestUser_String_Username(t *testing.T) {
	u := &User{ID: 1, Username: "test"}
	if u.String() != "test" {
		t.Errorf("String() = %q, want %q", u.String(), "test")
	}
}

func TestUser_String_FirstLastName(t *testing.T) {
	u := &User{ID: 1, FirstName: "John", LastName: "Doe"}
	if u.String() != "John Doe" {
		t.Errorf("String() = %q, want %q", u.String(), "John Doe")
	}
}

func TestUser_String_Empty(t *testing.T) {
	u := &User{ID: 12345}
	if u.String() != "user_12345" {
		t.Errorf("String() = %q, want %q", u.String(), "user_12345")
	}
}

func TestUser_MentionName(t *testing.T) {
	u := &User{ID: 1, Username: "test"}
	if u.MentionName() != "@test" {
		t.Errorf("MentionName() = %q, want %q", u.MentionName(), "@test")
	}
}
