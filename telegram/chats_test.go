package telegram

import (
	"testing"
)

func TestParseJoinLink(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantUser string
		wantHash string
		wantErr  bool
	}{
		{
			name:     "username",
			input:    "mtgo_labs",
			wantUser: "mtgo_labs",
		},
		{
			name:     "username with @",
			input:    "@mtgo_labs",
			wantUser: "mtgo_labs",
		},
		{
			name:     "invite hash with +",
			input:    "+tPknWlMV9eU3NThk",
			wantHash: "tPknWlMV9eU3NThk",
		},
		{
			name:     "invite hash long no prefix",
			input:    "tPknWlMV9eU3NThkXXXXX",
			wantHash: "tPknWlMV9eU3NThkXXXXX",
		},
		{
			name:     "URL t.me username",
			input:    "https://t.me/mtgo_labs",
			wantUser: "mtgo_labs",
		},
		{
			name:     "URL t.me invite hash",
			input:    "https://t.me/+tPknWlMV9eU3NThk",
			wantHash: "tPknWlMV9eU3NThk",
		},
		{
			name:     "URL telegram.me username",
			input:    "https://telegram.me/mtgo_labs",
			wantUser: "mtgo_labs",
		},
		{
			name:     "URL telegram.dog username",
			input:    "https://telegram.dog/channelname",
			wantUser: "channelname",
		},
		{
			name:     "URL without scheme",
			input:    "t.me/mtgo_labs",
			wantUser: "mtgo_labs",
		},
		{
			name:     "URL with trailing slash",
			input:    "https://t.me/mtgo_labs/",
			wantUser: "mtgo_labs",
		},
		{
			name:     "URL invite with query params",
			input:    "https://t.me/+tPknWlMV9eU3NThk?single",
			wantHash: "tPknWlMV9eU3NThk",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
		{
			name:     "hash with dashes",
			input:    "abc-def_123",
			wantHash: "abc-def_123",
		},
		{
			name:     "short username",
			input:    "ab",
			wantUser: "ab",
		},
		{
			name:     "URL t.me with @ in path",
			input:    "https://t.me/@mtgo_labs",
			wantUser: "mtgo_labs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseJoinLink(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseJoinLink(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseJoinLink(%q) unexpected error: %v", tt.input, err)
			}
			if got.username != tt.wantUser {
				t.Errorf("username = %q, want %q", got.username, tt.wantUser)
			}
			if got.inviteHash != tt.wantHash {
				t.Errorf("inviteHash = %q, want %q", got.inviteHash, tt.wantHash)
			}
		})
	}
}
