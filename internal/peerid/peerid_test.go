package peerid

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestMarkChannel(t *testing.T) {
	tests := []struct {
		name string
		in   int64
		want int64
	}{
		{"ordinary channel", 1234, -1000000001234},
		{"small channel", 1, -1000000000001},
		{"zero channel", 0, -1000000000000},
		{"large channel", 1844674407305057791, -1844675407305057791},
	}
	for _, tt := range tests {
		if got := MarkChannel(tt.in); got != tt.want {
			t.Errorf("%s: MarkChannel(%d) = %d, want %d", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestMarkChat(t *testing.T) {
	if got := MarkChat(500); got != -500 {
		t.Errorf("MarkChat(500) = %d, want -500", got)
	}
}

func TestUnmarkChannel(t *testing.T) {
	tests := []struct {
		name   string
		in     int64
		wantID int64
		wantOK bool
	}{
		{"marked channel", -1000000001234, 1234, true},
		{"prefix edge", -1000000000000, 0, true},
		{"basic chat not marked", -500, 0, false},
		{"user not marked", 7, 0, false},
		{"zero not marked", 0, 0, false},
	}
	for _, tt := range tests {
		gotID, gotOK := UnmarkChannel(tt.in)
		if gotID != tt.wantID || gotOK != tt.wantOK {
			t.Errorf("%s: UnmarkChannel(%d) = (%d, %v), want (%d, %v)",
				tt.name, tt.in, gotID, gotOK, tt.wantID, tt.wantOK)
		}
	}
}

func TestIsMarkedChannel(t *testing.T) {
	if !IsMarkedChannel(-1000000001234) {
		t.Error("IsMarkedChannel(-1000000001234) = false, want true")
	}
	if IsMarkedChannel(-1000000000000 + 1) {
		t.Error("IsMarkedChannel just above prefix = true, want false")
	}
	if IsMarkedChannel(-500) {
		t.Error("IsMarkedChannel(-500) = true, want false")
	}
}

func TestMarkFromPeer(t *testing.T) {
	tests := []struct {
		name string
		in   tg.PeerClass
		want int64
	}{
		{"user", &tg.PeerUser{UserID: 42}, 42},
		{"basic chat", &tg.PeerChat{ChatID: 9}, -9},
		{"channel", &tg.PeerChannel{ChannelID: 1234}, -1000000001234},
		{"nil", nil, 0},
	}
	for _, tt := range tests {
		if got := MarkFromPeer(tt.in); got != tt.want {
			t.Errorf("%s: MarkFromPeer(%v) = %d, want %d", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestRawFromPeer(t *testing.T) {
	tests := []struct {
		name   string
		in     tg.PeerClass
		wantID int64
		wantOK bool
	}{
		{"user", &tg.PeerUser{UserID: 42}, 42, true},
		{"basic chat", &tg.PeerChat{ChatID: 9}, 9, true},
		{"channel", &tg.PeerChannel{ChannelID: 1234}, 1234, true},
		{"nil", nil, 0, false},
	}
	for _, tt := range tests {
		gotID, gotOK := RawFromPeer(tt.in)
		if gotID != tt.wantID || gotOK != tt.wantOK {
			t.Errorf("%s: RawFromPeer(%v) = (%d, %v), want (%d, %v)",
				tt.name, tt.in, gotID, gotOK, tt.wantID, tt.wantOK)
		}
	}
}
