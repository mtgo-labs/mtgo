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

func TestPrepareAPIQuery(t *testing.T) {
	cfg := Config{APIID: 12345}
	query := &tg.PingRequest{PingID: 42}

	prepared, initializesAPI := prepareAPIQuery(cfg, false, query)
	if !initializesAPI {
		t.Fatal("first API query should initialize the connection")
	}
	wrapped, ok := prepared.(*tg.InvokeWithLayerRequest)
	if !ok {
		t.Fatalf("first API query = %T, want *tg.InvokeWithLayerRequest", prepared)
	}
	init, ok := wrapped.Query.(*tg.InitConnectionRequest)
	if !ok || init.Query != query {
		t.Fatalf("first API query payload = %T, want initConnection containing original query", wrapped.Query)
	}

	prepared, initializesAPI = prepareAPIQuery(cfg, true, query)
	if initializesAPI {
		t.Fatal("initialized connection should not be initialized again")
	}
	if prepared != query {
		t.Fatalf("initialized query = %T, want original query", prepared)
	}

	explicit := &tg.InvokeWithLayerRequest{Layer: tg.Layer, Query: query}
	prepared, initializesAPI = prepareAPIQuery(cfg, false, explicit)
	if initializesAPI || prepared != explicit {
		t.Fatal("explicit layer wrapper should pass through unchanged")
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
