package telegram

import "testing"

func TestWSDCAddress(t *testing.T) {
	tests := []struct {
		name     string
		dcID     int
		testMode bool
		tls      bool
		want     string
	}{
		{
			name: "plain",
			dcID: 4,
			want: "ws://vesta.web.telegram.org:80/apiws",
		},
		{
			name: "tls",
			dcID: 4,
			tls:  true,
			want: "wss://vesta.web.telegram.org:443/apiws",
		},
		{
			name:     "test mode",
			dcID:     2,
			testMode: true,
			tls:      true,
			want:     "wss://venus.web.telegram.org:443/apiws_test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wsDCAddress(tt.dcID, tt.testMode, tt.tls)
			if got != tt.want {
				t.Fatalf("wsDCAddress(%d, %v, %v) = %q, want %q", tt.dcID, tt.testMode, tt.tls, got, tt.want)
			}
		})
	}
}
