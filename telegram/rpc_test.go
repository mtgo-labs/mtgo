package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestWrapInitConnection(t *testing.T) {
	cfg := Config{
		APIID: 12345,
		Device: DeviceConfig{
			DeviceModel:    "test-device",
			SystemVersion:  "test-system",
			AppVersion:     "test-app",
			SystemLangCode: "en",
			LangPack:       "android",
			LangCode:       "en",
		},
	}
	query := &tg.PingRequest{PingID: 42}

	wrapped, ok := wrapInitConnection(cfg, query).(*tg.InvokeWithLayerRequest)
	if !ok {
		t.Fatalf("wrapInitConnection() = %T, want *tg.InvokeWithLayerRequest", wrapped)
	}
	if wrapped.Layer != tg.Layer {
		t.Errorf("Layer = %d, want %d", wrapped.Layer, tg.Layer)
	}

	init, ok := wrapped.Query.(*tg.InitConnectionRequest)
	if !ok {
		t.Fatalf("wrapped.Query = %T, want *tg.InitConnectionRequest", wrapped.Query)
	}
	if init.APIID != cfg.APIID {
		t.Errorf("ApiID = %d, want %d", init.APIID, cfg.APIID)
	}
	if init.DeviceModel != cfg.Device.DeviceModel {
		t.Errorf("DeviceModel = %q, want %q", init.DeviceModel, cfg.Device.DeviceModel)
	}
	if init.SystemVersion != cfg.Device.SystemVersion {
		t.Errorf("SystemVersion = %q, want %q", init.SystemVersion, cfg.Device.SystemVersion)
	}
	if init.AppVersion != cfg.Device.AppVersion {
		t.Errorf("AppVersion = %q, want %q", init.AppVersion, cfg.Device.AppVersion)
	}
	if init.SystemLangCode != cfg.Device.SystemLangCode {
		t.Errorf("SystemLangCode = %q, want %q", init.SystemLangCode, cfg.Device.SystemLangCode)
	}
	if init.LangPack != cfg.Device.LangPack {
		t.Errorf("LangPack = %q, want %q", init.LangPack, cfg.Device.LangPack)
	}
	if init.LangCode != cfg.Device.LangCode {
		t.Errorf("LangCode = %q, want %q", init.LangCode, cfg.Device.LangCode)
	}
	if init.Query != query {
		t.Errorf("Query = %p, want %p", init.Query, query)
	}
}

func TestNeedsInitConnection(t *testing.T) {
	if !needsInitConnection(&tg.PingRequest{}) {
		t.Fatal("PingRequest should need init connection")
	}
	if needsInitConnection(&tg.InvokeWithLayerRequest{}) {
		t.Fatal("InvokeWithLayerRequest should not need init connection")
	}
	if needsInitConnection(&tg.InitConnectionRequest{}) {
		t.Fatal("InitConnectionRequest should not need init connection")
	}
	if needsInitConnection(nil) {
		t.Fatal("nil should not need init connection")
	}
}
