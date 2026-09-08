package telegram

import (
	"bytes"
	"strings"
	"testing"

	tgconv "github.com/mtgo-labs/session-converter"
)

func mtgoTestKey(b byte) []byte {
	key := make([]byte, 256)
	for i := range key {
		key[i] = b + byte(i)
	}
	return key
}

func mtgoTestSession() *tgconv.Session {
	return &tgconv.Session{
		DCID:        4,
		AuthKey:     mtgoTestKey(0x33),
		AppID:       22333936,
		UserID:      123456789,
		IsBot:       false,
		APIHash:     "89abcdef0123456789abcdef01234567",
		PhoneNumber: "+9996621234",
	}
}

// TestImportMTGOSessionString verifies that an MTGO1 string imports API ID,
// DC, auth key, user ID, API hash, and phone number into storage and config.
func TestImportMTGOSessionString(t *testing.T) {
	s := mtgoTestSession()
	str, err := tgconv.EncodeSession(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(str, "MTGO1.") {
		t.Fatalf("not an MTGO1 string: %q", str)
	}

	st := NewMemoryStorage()
	c, err := NewClient(0, "", &Config{SessionString: str})
	if err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	c.storage = st
	c.mu.Unlock()

	if err := c.importSessionString(st); err != nil {
		t.Fatal(err)
	}

	if apiID, err := st.APIID(); err != nil || apiID != s.AppID {
		t.Fatalf("api_id=%d err=%v", apiID, err)
	}
	if dcID, err := st.DCID(); err != nil || dcID != s.DCID {
		t.Fatalf("dc_id=%d err=%v", dcID, err)
	}
	if authKey, err := st.AuthKey(); err != nil || !bytes.Equal(authKey, s.AuthKey) {
		t.Fatalf("auth key mismatch: err=%v", err)
	}
	if userID, err := st.UserID(); err != nil || userID != s.UserID {
		t.Fatalf("user_id=%d err=%v", userID, err)
	}
	if cfg := c.config(); cfg.APIID != s.AppID || cfg.APIHash != s.APIHash || cfg.PhoneNumber != s.PhoneNumber {
		t.Fatalf("config=%+v", cfg)
	}
}

// TestImportMTGOSessionStringKeepsExplicitCredentials verifies that config
// values supplied by the caller are not overwritten by the session string.
func TestImportMTGOSessionStringKeepsExplicitCredentials(t *testing.T) {
	s := mtgoTestSession()
	str, err := tgconv.EncodeSession(s)
	if err != nil {
		t.Fatal(err)
	}

	st := NewMemoryStorage()
	c, err := NewClient(99999, "0123456789abcdef0123456789abcdef",
		&Config{SessionString: str, PhoneNumber: "+11111111111"})
	if err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	c.storage = st
	c.mu.Unlock()

	if err := c.importSessionString(st); err != nil {
		t.Fatal(err)
	}
	if cfg := c.config(); cfg.APIID != 99999 || cfg.APIHash != "0123456789abcdef0123456789abcdef" ||
		cfg.PhoneNumber != "+11111111111" {
		t.Fatalf("explicit credentials overwritten: %+v", cfg)
	}
}

// TestExportSessionString verifies that the client exports its current
// session in the native MTGO1 format with all fields intact.
func TestExportSessionString(t *testing.T) {
	st := NewMemoryStorage()
	s := mtgoTestSession()
	if err := st.SetAuthKey(s.AuthKey); err != nil {
		t.Fatal(err)
	}
	if err := st.SetAPIID(s.AppID); err != nil {
		t.Fatal(err)
	}
	if err := st.SetAPIHash(s.APIHash); err != nil {
		t.Fatal(err)
	}
	if err := st.SetDCID(s.DCID); err != nil {
		t.Fatal(err)
	}
	if err := st.SetUserID(s.UserID); err != nil {
		t.Fatal(err)
	}
	if err := st.SetIsBot(false); err != nil {
		t.Fatal(err)
	}

	c, err := NewClient(s.AppID, s.APIHash, &Config{PhoneNumber: s.PhoneNumber})
	if err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	c.storage = st
	c.mu.Unlock()

	str, err := c.ExportSessionString()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(str, "MTGO1.") {
		t.Fatalf("not an MTGO1 string: %q", str)
	}
	decoded, err := tgconv.DecodeSession(str)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.DCID != s.DCID || decoded.AppID != s.AppID || decoded.UserID != s.UserID ||
		decoded.IsBot != s.IsBot || decoded.TestMode != s.TestMode ||
		decoded.APIHash != s.APIHash || decoded.PhoneNumber != s.PhoneNumber {
		t.Fatalf("decoded=%+v", decoded)
	}
	if !bytes.Equal(decoded.AuthKey, s.AuthKey) {
		t.Fatal("auth key mismatch")
	}

	// The exported string must be importable again through the generic path.
	st2 := NewMemoryStorage()
	c2, err := NewClient(0, "", &Config{SessionString: str})
	if err != nil {
		t.Fatal(err)
	}
	c2.mu.Lock()
	c2.storage = st2
	c2.mu.Unlock()
	if err := c2.importSessionString(st2); err != nil {
		t.Fatal(err)
	}
	if authKey, err := st2.AuthKey(); err != nil || !bytes.Equal(authKey, s.AuthKey) {
		t.Fatalf("re-import auth key mismatch: err=%v", err)
	}
}

func TestExportSessionStringRequiresStorage(t *testing.T) {
	c, err := NewClient(12345, "0123456789abcdef0123456789abcdef", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ExportSessionString(); err != ErrNotConnected {
		t.Fatalf("err=%v, want ErrNotConnected", err)
	}
}

// TestExportSessionStringWithoutAuthKey verifies the pre-authorization
// contract: no auth key yields an empty string, not an error.
func TestExportSessionStringWithoutAuthKey(t *testing.T) {
	st := NewMemoryStorage()
	c, err := NewClient(12345, "0123456789abcdef0123456789abcdef", nil)
	if err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	c.storage = st
	c.mu.Unlock()
	str, err := c.ExportSessionString()
	if err != nil || str != "" {
		t.Fatalf("str=%q err=%v, want empty string and nil error", str, err)
	}
}

// TestExportSessionStringFallsBackToPyrogram verifies that a session lacking
// the API hash and user ID still exports — as a legacy Pyrogram string.
func TestExportSessionStringFallsBackToPyrogram(t *testing.T) {
	st := NewMemoryStorage()
	s := mtgoTestSession()
	if err := st.SetAuthKey(s.AuthKey); err != nil {
		t.Fatal(err)
	}
	if err := st.SetAPIID(s.AppID); err != nil {
		t.Fatal(err)
	}
	if err := st.SetDCID(s.DCID); err != nil {
		t.Fatal(err)
	}

	c, err := NewClient(s.AppID, s.APIHash, nil)
	if err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	c.storage = st
	c.mu.Unlock()

	str, err := c.ExportSessionString()
	if err != nil {
		t.Fatal(err)
	}
	if str == "" || strings.HasPrefix(str, "MTGO") {
		t.Fatalf("expected Pyrogram fallback, got %q", str)
	}
	decoded, format, err := tgconv.Decode(str)
	if err != nil {
		t.Fatal(err)
	}
	if format != tgconv.FormatPyrogram || decoded.DCID != s.DCID {
		t.Fatalf("decoded=%+v format=%s", decoded, format)
	}
}

// TestMemoryStorageExportSessionStringStillPyrogram guards the storage-level
// Pyrogram export against regressions from the MTGO1 client export change.
func TestMemoryStorageExportSessionStringStillPyrogram(t *testing.T) {
	st := NewMemoryStorage()
	s := mtgoTestSession()
	if err := st.SetAuthKey(s.AuthKey); err != nil {
		t.Fatal(err)
	}
	if err := st.SetAPIID(s.AppID); err != nil {
		t.Fatal(err)
	}
	if err := st.SetDCID(s.DCID); err != nil {
		t.Fatal(err)
	}
	if err := st.SetUserID(s.UserID); err != nil {
		t.Fatal(err)
	}
	if err := st.SetIsBot(false); err != nil {
		t.Fatal(err)
	}

	str, err := st.ExportSessionString()
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(str, "MTGO") {
		t.Fatalf("Pyrogram export must stay Pyrogram, got %q", str)
	}
	decoded, format, err := tgconv.Decode(str)
	if err != nil {
		t.Fatal(err)
	}
	if format != tgconv.FormatPyrogram || decoded.UserID != s.UserID || decoded.DCID != s.DCID {
		t.Fatalf("decoded=%+v format=%s", decoded, format)
	}
}
