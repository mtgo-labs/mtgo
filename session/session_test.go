package session

import (
	"crypto/rand"
	"encoding/base64"
	"net"
	"testing"
)

func makeTestAuthKey() []byte {
	key := make([]byte, 256)
	_, _ = rand.Read(key)
	return key
}

func makeTestData() *SessionData {
	return &SessionData{
		DCID:          2,
		ServerAddress: "149.154.167.50",
		Port:          443,
		AuthKey:       makeTestAuthKey(),
		AppID:         12345,
		TestMode:      false,
		UserID:        987654321,
		IsBot:         false,
	}
}

// --- Telethon ---

func TestTelethonRoundTrip(t *testing.T) {
	orig := makeTestData()
	encoded, err := EncodeTelethon(orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if encoded == "" {
		t.Fatal("empty encoded string")
	}
	if encoded[0] != '2' {
		t.Fatalf("expected '2' prefix (has api_id), got %q", encoded[0])
	}

	decoded, err := DecodeTelethon(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertEqualSession(t, orig, decoded)
	if decoded.AppID != orig.AppID {
		t.Fatalf("AppID: %d vs %d", decoded.AppID, orig.AppID)
	}
}

func TestTelethonIPv6RoundTrip(t *testing.T) {
	orig := &SessionData{
		DCID:          1,
		ServerAddress: "2001:67c:4e8:f002::e",
		Port:          443,
		AuthKey:       makeTestAuthKey(),
	}
	encoded, err := EncodeTelethon(orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeTelethon(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	// IPv6 addresses may normalize; compare parsed IPs.
	origIP := net.ParseIP(orig.ServerAddress)
	decIP := net.ParseIP(decoded.ServerAddress)
	if !origIP.Equal(decIP) {
		t.Fatalf("IP mismatch: %s vs %s", orig.ServerAddress, decoded.ServerAddress)
	}
	if orig.DCID != decoded.DCID || orig.Port != decoded.Port {
		t.Fatalf("DC/port mismatch")
	}
	assertAuthKeyEqual(t, orig.AuthKey, decoded.AuthKey)
}

func TestTelethonRoundTripNoAPIID(t *testing.T) {
	orig := &SessionData{
		DCID:          2,
		ServerAddress: "149.154.167.50",
		Port:          443,
		AuthKey:       makeTestAuthKey(),
		AppID:         0,
	}
	encoded, err := EncodeTelethon(orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if encoded[0] != '1' {
		t.Fatalf("expected '1' prefix (no api_id), got %q", encoded[0])
	}
	decoded, err := DecodeTelethon(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertEqualSession(t, orig, decoded)
}

func TestTelethonV2Detect(t *testing.T) {
	orig := makeTestData()
	encoded, err := EncodeTelethon(orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	f := DetectFormat(encoded)
	if f != FormatTelethon {
		t.Fatalf("expected FormatTelethon, got %s", f)
	}
}

// --- Pyrogram ---

func TestPyrogramRoundTrip(t *testing.T) {
	orig := makeTestData()
	encoded, err := EncodePyrogram(orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if encoded == "" {
		t.Fatal("empty encoded string")
	}
	decoded, err := DecodePyrogram(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if orig.DCID != decoded.DCID {
		t.Fatalf("DCID mismatch: %d vs %d", orig.DCID, decoded.DCID)
	}
	if orig.AppID != decoded.AppID {
		t.Fatalf("AppID mismatch: %d vs %d", orig.AppID, decoded.AppID)
	}
	if orig.TestMode != decoded.TestMode {
		t.Fatalf("TestMode mismatch")
	}
	if orig.UserID != decoded.UserID {
		t.Fatalf("UserID mismatch: %d vs %d", orig.UserID, decoded.UserID)
	}
	if orig.IsBot != decoded.IsBot {
		t.Fatalf("IsBot mismatch")
	}
	assertAuthKeyEqual(t, orig.AuthKey, decoded.AuthKey)
}

// --- GramJS ---

func TestGramjsRoundTrip(t *testing.T) {
	orig := makeTestData()
	encoded, err := EncodeGramjs(orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if encoded == "" {
		t.Fatal("empty encoded string")
	}
	if encoded[0] != '1' {
		t.Fatalf("expected '1' prefix, got %q", encoded[0])
	}

	decoded, err := DecodeGramjs(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertEqualSession(t, orig, decoded)
}

// --- mtcute ---

func TestMtcuteRoundTrip(t *testing.T) {
	orig := makeTestData()
	encoded, err := EncodeMtcute(orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if encoded == "" {
		t.Fatal("empty encoded string")
	}

	decoded, err := DecodeMtcute(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if orig.DCID != decoded.DCID {
		t.Fatalf("DCID mismatch: %d vs %d", orig.DCID, decoded.DCID)
	}
	if orig.ServerAddress != decoded.ServerAddress {
		t.Fatalf("ServerAddress mismatch: %s vs %s", orig.ServerAddress, decoded.ServerAddress)
	}
	if orig.Port != decoded.Port {
		t.Fatalf("Port mismatch: %d vs %d", orig.Port, decoded.Port)
	}
	assertAuthKeyEqual(t, orig.AuthKey, decoded.AuthKey)
}

func TestMtcuteRoundTripNoSelf(t *testing.T) {
	orig := &SessionData{
		DCID:          2,
		ServerAddress: "149.154.167.50",
		Port:          443,
		AuthKey:       makeTestAuthKey(),
		UserID:        0,
	}
	encoded, err := EncodeMtcute(orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeMtcute(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.DCID != 2 {
		t.Fatalf("DCID mismatch")
	}
	assertAuthKeyEqual(t, orig.AuthKey, decoded.AuthKey)
}

func TestMtcuteRoundTripBot(t *testing.T) {
	orig := &SessionData{
		DCID:          2,
		ServerAddress: "149.154.167.50",
		Port:          443,
		AuthKey:       makeTestAuthKey(),
		UserID:        123456789,
		IsBot:         true,
	}
	encoded, err := EncodeMtcute(orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeMtcute(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !decoded.IsBot {
		t.Fatal("expected IsBot=true")
	}
	if decoded.UserID != 123456789 {
		t.Fatalf("UserID mismatch: %d", decoded.UserID)
	}
}

func TestMtcuteDecodeRealSession(t *testing.T) {
	// Real mtcute test vector.
	sessionStr := "AwQAAAAXAgIADjE0OS4xNTQuMTY3LjUwALsBAAAXAgICDzE0OS4xNTQuMTY3LjIyMrsBAAD-AAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

	decoded, err := DecodeMtcute(sessionStr)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.DCID != 2 {
		t.Fatalf("DCID = %d, want 2", decoded.DCID)
	}
	if decoded.ServerAddress != "149.154.167.50" {
		t.Fatalf("ServerAddress = %q", decoded.ServerAddress)
	}
}

// --- Auto-detect ---

func TestAutoDetectTelethon(t *testing.T) {
	orig := makeTestData()
	s, _ := EncodeTelethon(orig)
	data, format, err := Decode(s)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if format != FormatTelethon {
		t.Fatalf("detected %s, want telethon", format)
	}
	if data.DCID != orig.DCID {
		t.Fatal("DCID mismatch")
	}
}

func TestAutoDetectGramjs(t *testing.T) {
	orig := makeTestData()
	s, _ := EncodeGramjs(orig)
	data, format, err := Decode(s)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if format != FormatGramJS {
		t.Fatalf("detected %s, want gramjs", format)
	}
	if data.DCID != orig.DCID {
		t.Fatal("DCID mismatch")
	}
}

func TestAutoDetectPyrogram(t *testing.T) {
	orig := makeTestData()
	s, _ := EncodePyrogram(orig)
	data, format, err := Decode(s)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if format != FormatPyrogram {
		t.Fatalf("detected %s, want pyrogram", format)
	}
	if data.DCID != orig.DCID {
		t.Fatal("DCID mismatch")
	}
}

func TestAutoDetectMtcute(t *testing.T) {
	orig := makeTestData()
	s, _ := EncodeMtcute(orig)
	data, format, err := Decode(s)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if format != FormatMtcute {
		t.Fatalf("detected %s, want mtcute", format)
	}
	if data.DCID != orig.DCID {
		t.Fatal("DCID mismatch")
	}
}

// --- Invalid inputs ---

func TestDecodeEmpty(t *testing.T) {
	_, _, err := Decode("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestDecodeInvalidBase64(t *testing.T) {
	_, _, err := Decode("!!!not-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDetectFormatEmpty(t *testing.T) {
	if f := DetectFormat(""); f != FormatUnknown {
		t.Fatalf("expected unknown, got %s", f)
	}
}

// TestDecodeGotgExtendedMalformedNoPanic ensures a malformed gotg_extended
// payload whose null-padding run advances the cursor so that fewer than 256
// auth-key bytes remain returns an error instead of panicking on an
// out-of-range slice. Regression for the unbounded auth-key copy.
func TestDecodeGotgExtendedMalformedNoPanic(t *testing.T) {
	// 283-byte payload: dc low byte <=5, version, ip_len=0, then all-zero
	// padding so the null-skip loop walks the cursor to near the end, leaving
	// < 256 bytes for the auth key.
	payload := make([]byte, 283)
	payload[0] = 1 // dc id low byte (<=5 so DetectFormat picks gotg_extended)
	payload[281] = 0x01
	payload[282] = 0x02 // non-zero bytes stop the null loop and serve as the port

	s := base64.URLEncoding.EncodeToString(payload)

	if f := DetectFormat(s); f != FormatGotgExtended {
		t.Fatalf("expected gotg_extended, got %s", f)
	}

	// Must not panic; must return an error.
	if _, _, err := Decode(s); err == nil {
		t.Fatal("expected error for truncated gotg_extended auth key, got nil")
	}
	if _, err := DecodeGotgExtended(s); err == nil {
		t.Fatal("expected error from DecodeGotgExtended, got nil")
	}
}

// --- helpers ---

func assertEqualSession(t *testing.T, a, b *SessionData) {
	t.Helper()
	if a.DCID != b.DCID {
		t.Fatalf("DCID: %d vs %d", a.DCID, b.DCID)
	}
	if a.ServerAddress != b.ServerAddress {
		t.Fatalf("ServerAddress: %s vs %s", a.ServerAddress, b.ServerAddress)
	}
	if a.Port != b.Port {
		t.Fatalf("Port: %d vs %d", a.Port, b.Port)
	}
	assertAuthKeyEqual(t, a.AuthKey, b.AuthKey)
}

func assertAuthKeyEqual(t *testing.T, a, b []byte) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("auth key length: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("auth key mismatch at byte %d", i)
		}
	}
}
