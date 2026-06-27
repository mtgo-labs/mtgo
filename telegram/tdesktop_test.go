package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/internal/transport"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

func TestDeviceTDesktopWindows(t *testing.T) {
	d := DeviceTDesktopWindows()
	if d.DeviceModel != "Desktop" {
		t.Errorf("DeviceModel = %q, want %q", d.DeviceModel, "Desktop")
	}
	if d.SystemVersion != "Windows 10" {
		t.Errorf("SystemVersion = %q, want %q", d.SystemVersion, "Windows 10")
	}
	if d.LangPack != "tdesktop" {
		t.Errorf("LangPack = %q, want %q", d.LangPack, "tdesktop")
	}
	if d.ClientPlatform != types.ClientPlatformDesktop {
		t.Errorf("ClientPlatform = %q, want %q", d.ClientPlatform, types.ClientPlatformDesktop)
	}
}

func TestObfuscatedMarkerForMode(t *testing.T) {
	tests := []struct {
		mode   string
		want   byte
		wantOK bool
	}{
		{TransportModeAbridged, 0xEF, true},
		{TransportModeIntermediate, 0xEE, true},
		{TransportModePaddedIntermediate, 0xEE, true},
		{TransportModeFull, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got, ok := obfuscatedMarkerForMode(tt.mode)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Fatalf("marker = 0x%02x, want 0x%02x", got, tt.want)
			}
		})
	}
}

func TestCreateTransportNoObfuscate(t *testing.T) {
	c, err := NewClient(1, "hash", &Config{NoUpdates: true})
	if err != nil {
		t.Fatal(err)
	}
	tp, err := c.createTransport(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tp.(*transport.TCPAbridged); !ok {
		t.Fatalf("expected *TCPAbridged, got %T", tp)
	}
}

func TestCreateTransportAlwaysObfuscate(t *testing.T) {
	c, err := NewClient(1, "hash", &Config{AlwaysObfuscate: true, NoUpdates: true})
	if err != nil {
		t.Fatal(err)
	}
	tp, err := c.createTransport(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tp.(*transport.TCPObfuscated); !ok {
		t.Fatalf("expected *TCPObfuscated, got %T", tp)
	}
}
