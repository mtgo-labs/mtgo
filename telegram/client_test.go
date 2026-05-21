package telegram

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/internal/transport"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestNewClientDefaults(t *testing.T) {
	c, _ := NewClient(12345, "deadbeef", nil)

	if c.cfg.APIID != 12345 {
		t.Errorf("APIID = %d, want 12345", c.cfg.APIID)
	}
	if c.cfg.APIHash != "deadbeef" {
		t.Errorf("APIHash = %q, want %q", c.cfg.APIHash, "deadbeef")
	}
	if c.cfg.SleepThreshold != 10*time.Second {
		t.Errorf("SleepThreshold = %v, want 10s", c.cfg.SleepThreshold)
	}
	if c.cfg.MaxConcurrentTrans != 1 {
		t.Errorf("MaxConcurrentTrans = %d, want 1", c.cfg.MaxConcurrentTrans)
	}
	if c.cfg.DispatchWorkers != 0 {
		t.Errorf("DispatchWorkers = %d, want 0", c.cfg.DispatchWorkers)
	}
	if c.cfg.DispatchQueueSize != defaultDispatchQueueSize {
		t.Errorf("DispatchQueueSize = %d, want %d", c.cfg.DispatchQueueSize, defaultDispatchQueueSize)
	}
	if c.cfg.MaxMessageCacheSize != 1000 {
		t.Errorf("MaxMessageCacheSize = %d, want 1000", c.cfg.MaxMessageCacheSize)
	}
	if c.cfg.Device.LangCode != "en" {
		t.Errorf("LangCode = %q, want %q", c.cfg.Device.LangCode, "en")
	}
	if c.cfg.Device.SystemLangCode != "en" {
		t.Errorf("SystemLangCode = %q, want %q", c.cfg.Device.SystemLangCode, "en")
	}
	if !c.cfg.SkipUpdates {
		t.Error("SkipUpdates = false, want true")
	}
	if !c.cfg.FetchReplies {
		t.Error("FetchReplies = false, want true")
	}
	if !c.cfg.FetchTopics {
		t.Error("FetchTopics = false, want true")
	}
	if !c.cfg.FetchStories {
		t.Error("FetchStories = false, want true")
	}
	if !c.cfg.FetchStickers {
		t.Error("FetchStickers = false, want true")
	}
	if c.cfg.Device.ClientPlatform != types.ClientPlatformAndroid {
		t.Errorf("ClientPlatform = %q, want %q", c.cfg.Device.ClientPlatform, types.ClientPlatformAndroid)
	}
	if c.cfg.TransportMode != TransportModeAbridged {
		t.Errorf("TransportMode = %q, want %q", c.cfg.TransportMode, TransportModeAbridged)
	}
}

func TestNewClientWithOptions(t *testing.T) {
	c, _ := NewClient(111, "hash", &Config{
		SessionName:         "test-session",
		BotToken:            "123456:ABC",
		PhoneNumber:         "+1234567890",
		PhoneCode:           "12345",
		Password:            "mypass",
		WorkDir:             "/tmp/mtgo",
		InMemory:            true,
		DC:                  4,
		TestMode:            true,
		IPv6:                true,
		NoUpdates:           true,
		SleepThreshold:      5 * time.Second,
		MaxConcurrentTrans:  4,
		DispatchWorkers:     8,
		DispatchQueueSize:   512,
		MaxMessageCacheSize: 500,
		ParseMode:           HTML,
		HidePassword:        true,
		ClientPlatform:      types.ClientPlatformIOS,
		DeviceModel:         "iPhone 15",
		SystemVersion:       "iOS 17",
		LangCode:            "ru",
		LangPack:            "android",
		SystemLangCode:      "ru",
		TZOffset:            3,
		TransportMode:       TransportModeFull,
	})

	cfg := c.Config()
	if cfg.SessionName != "test-session" {
		t.Errorf("SessionName = %q", cfg.SessionName)
	}
	if cfg.BotToken != "123456:ABC" {
		t.Errorf("BotToken = %q", cfg.BotToken)
	}
	if cfg.PhoneNumber != "+1234567890" {
		t.Errorf("PhoneNumber = %q", cfg.PhoneNumber)
	}
	if cfg.PhoneCode != "12345" {
		t.Errorf("PhoneCode = %q", cfg.PhoneCode)
	}
	if cfg.Password != "mypass" {
		t.Errorf("Password = %q", cfg.Password)
	}
	if cfg.WorkDir != "/tmp/mtgo" {
		t.Errorf("WorkDir = %q", cfg.WorkDir)
	}
	if !cfg.InMemory {
		t.Error("InMemory = false")
	}
	if cfg.DC != 4 {
		t.Errorf("DC = %d", cfg.DC)
	}
	if !cfg.TestMode {
		t.Error("TestMode = false")
	}
	if !cfg.IPv6 {
		t.Error("IPv6 = false")
	}
	if !cfg.NoUpdates {
		t.Error("NoUpdates = false")
	}
	if cfg.SleepThreshold != 5*time.Second {
		t.Errorf("SleepThreshold = %v", cfg.SleepThreshold)
	}
	if cfg.MaxConcurrentTrans != 4 {
		t.Errorf("MaxConcurrentTrans = %d", cfg.MaxConcurrentTrans)
	}
	if cfg.DispatchWorkers != 8 {
		t.Errorf("DispatchWorkers = %d", cfg.DispatchWorkers)
	}
	if cfg.DispatchQueueSize != 512 {
		t.Errorf("DispatchQueueSize = %d", cfg.DispatchQueueSize)
	}
	if cfg.MaxMessageCacheSize != 500 {
		t.Errorf("MaxMessageCacheSize = %d", cfg.MaxMessageCacheSize)
	}
	if cfg.ParseMode != HTML {
		t.Errorf("ParseMode = %s", cfg.ParseMode)
	}
	if !cfg.HidePassword {
		t.Error("HidePassword = false")
	}
	if cfg.Device.ClientPlatform != types.ClientPlatformIOS {
		t.Errorf("ClientPlatform = %q", cfg.Device.ClientPlatform)
	}
	if cfg.Device.DeviceModel != "iPhone 15" {
		t.Errorf("DeviceModel = %q", cfg.Device.DeviceModel)
	}
	if cfg.Device.SystemVersion != "iOS 17" {
		t.Errorf("SystemVersion = %q", cfg.Device.SystemVersion)
	}
	if cfg.Device.LangCode != "ru" {
		t.Errorf("LangCode = %q", cfg.Device.LangCode)
	}
	if cfg.Device.LangPack != "android" {
		t.Errorf("LangPack = %q", cfg.Device.LangPack)
	}
	if cfg.Device.SystemLangCode != "ru" {
		t.Errorf("SystemLangCode = %q", cfg.Device.SystemLangCode)
	}
	if cfg.Device.TZOffset != 3 {
		t.Errorf("TZOffset = %d", cfg.Device.TZOffset)
	}
	if cfg.TransportMode != TransportModeFull {
		t.Errorf("TransportMode = %q", cfg.TransportMode)
	}
}

func TestNewClientRejectsInvalidTransportMode(t *testing.T) {
	_, err := NewClient(111, "hash", &Config{TransportMode: "invalid"})
	if err == nil {
		t.Fatal("expected invalid transport mode error")
	}
}

func TestWithProxy(t *testing.T) {
	c, _ := NewClient(1, "h", &Config{Proxy: &Proxy{Addr: "1.2.3.4:1080", Username: "user", Password: "pass"}})
	cfg := c.Config()
	if cfg.Proxy == nil {
		t.Fatal("Proxy is nil")
	}
	if cfg.Proxy.Addr != "1.2.3.4:1080" {
		t.Errorf("Proxy.Addr = %q", cfg.Proxy.Addr)
	}
	if cfg.Proxy.Username != "user" {
		t.Errorf("Proxy.Username = %q", cfg.Proxy.Username)
	}
	if cfg.Proxy.Password != "pass" {
		t.Errorf("Proxy.Password = %q", cfg.Proxy.Password)
	}

	c2, _ := NewClient(1, "h", nil)
	if c2.Config().Proxy != nil {
		t.Error("expected nil Proxy by default")
	}
}

func TestGuessMIMEType(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"photo.JPG", "image/jpeg"},
		{"image.png", "image/png"},
		{"anim.gif", "image/gif"},
		{"pic.bmp", "image/bmp"},
		{"pic.webp", "image/webp"},
		{"icon.svg", "image/svg+xml"},
		{"clip.mp4", "video/mp4"},
		{"clip.mov", "video/quicktime"},
		{"vid.avi", "video/x-msvideo"},
		{"vid.mkv", "video/x-matroska"},
		{"vid.webm", "video/webm"},
		{"song.mp3", "audio/mpeg"},
		{"song.flac", "audio/flac"},
		{"audio.wav", "audio/wav"},
		{"audio.ogg", "audio/ogg"},
		{"audio.opus", "audio/opus"},
		{"doc.pdf", "application/pdf"},
		{"file.zip", "application/zip"},
		{"data.json", "application/json"},
		{"page.html", "text/html"},
		{"unknown.xyz", "application/octet-stream"},
	}
	for _, tt := range tests {
		got := GuessMIMEType(tt.filename)
		if got != tt.want {
			t.Errorf("GuessMIMEType(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}

func TestGuessExtension(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"image/jpeg", ".jpeg"},
		{"image/png", ".png"},
		{"video/mp4", ".mp4"},
		{"audio/mpeg", ".mp3"},
		{"application/pdf", ".pdf"},
		{"application/zip", ".zip"},
		{"application/json", ".json"},
		{"text/html", ".html"},
		{"application/octet-stream", ""},
	}
	for _, tt := range tests {
		got := GuessExtension(tt.mime)
		if got != tt.want {
			t.Errorf("GuessExtension(%q) = %q, want %q", tt.mime, got, tt.want)
		}
	}
}

func TestDCAddresses(t *testing.T) {
	if len(TestDCs) == 0 {
		t.Fatal("TestDCs should not be empty")
	}
	if len(ProdDCs) == 0 {
		t.Fatal("ProdDCs should not be empty")
	}
	if TestDCs[1] != "149.154.175.10" {
		t.Errorf("TestDCs[1] = %q, want 149.154.175.10", TestDCs[1])
	}
	if ProdDCs[1] != "149.154.175.53" {
		t.Errorf("ProdDCs[1] = %q, want 149.154.175.53", ProdDCs[1])
	}
	if ProdDCs[5] != "91.108.56.130" {
		t.Errorf("ProdDCs[5] = %q, want 91.108.56.130", ProdDCs[5])
	}
}

func TestDefaultDCPort(t *testing.T) {
	if DefaultDCPort(true) != 80 {
		t.Errorf("DefaultDCPort(true) = %d, want 80", DefaultDCPort(true))
	}
	if DefaultDCPort(false) != 443 {
		t.Errorf("DefaultDCPort(false) = %d, want 443", DefaultDCPort(false))
	}
}

func TestResolveDCAddress(t *testing.T) {
	if addr := ResolveDCAddress(2, false); addr != "149.154.167.51" {
		t.Errorf("ResolveDCAddress(2, false) = %q", addr)
	}
	if addr := ResolveDCAddress(2, true); addr != "149.154.167.40" {
		t.Errorf("ResolveDCAddress(2, true) = %q", addr)
	}
	if addr := ResolveDCAddress(999, false); addr != "" {
		t.Errorf("ResolveDCAddress(999, false) = %q, want empty", addr)
	}
}

func TestConnectionStateTransitions(t *testing.T) {
	cs := newConnectionState()

	if cs.isConnected() {
		t.Error("new state should not be connected")
	}
	if err := cs.requireConnected(); !errors.Is(err, ErrNotConnected) {
		t.Errorf("requireConnected() = %v, want ErrNotConnected", err)
	}

	cs.setConnected(true)
	if !cs.isConnected() {
		t.Error("should be connected after setConnected(true)")
	}
	if err := cs.requireConnected(); err != nil {
		t.Errorf("requireConnected() = %v, want nil", err)
	}

	cs.setConnected(false)
	if cs.isConnected() {
		t.Error("should not be connected after setConnected(false)")
	}
}

func TestClientInitialState(t *testing.T) {
	c, _ := NewClient(1, "hash", nil)

	if c.IsConnected() {
		t.Error("new client should not be connected")
	}
	if c.Me() != nil {
		t.Error("new client Me() should be nil")
	}
	if c.Session() != nil {
		t.Error("new client Session() should be nil")
	}
	if c.Storage() != nil {
		t.Error("new client Storage() should be nil")
	}
	if len(c.sessions) != 0 {
		t.Error("new client sessions should be empty")
	}
}

func TestConnectAlreadyConnected(t *testing.T) {
	c, _ := NewClient(12345, "hash", &Config{InMemory: true, SessionName: "test"})
	c.state.setConnected(true)

	err := c.Connect(5 * time.Second)
	if !errors.Is(err, ErrAlreadyConnected) {
		t.Errorf("Connect() = %v, want ErrAlreadyConnected", err)
	}
}

func TestDisconnectNotConnected(t *testing.T) {
	c, _ := NewClient(12345, "hash", nil)

	err := c.Disconnect()
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("Disconnect() = %v, want ErrNotConnected", err)
	}
}

func TestDisconnectTwice(t *testing.T) {
	c, srv := newTestClient(12345, "hash", Config{NoUpdates: true})
	defer srv.Close()

	if err := c.Connect(5 * time.Second); err != nil {
		t.Fatalf("first Connect() = %v", err)
	}
	if err := c.Disconnect(); err != nil {
		t.Fatalf("first Disconnect() = %v", err)
	}

	err2 := c.Disconnect()
	if !errors.Is(err2, ErrNotConnected) {
		t.Errorf("second Disconnect() = %v, want ErrNotConnected", err2)
	}
}

func TestInvokeNotConnected(t *testing.T) {
	c, _ := NewClient(12345, "hash", nil)

	_, err := c.Invoke(context.Background(), nil, 1, 5*time.Second)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("Invoke() = %v, want ErrNotConnected", err)
	}
}

func TestInvokeRawNotConnected(t *testing.T) {
	c, _ := NewClient(12345, "hash", nil)

	_, err := c.InvokeRaw(context.Background(), nil, 1, 5*time.Second)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("InvokeRaw() = %v, want ErrNotConnected", err)
	}
}

func TestInvokeWithRawResultNotConnected(t *testing.T) {
	c, _ := NewClient(12345, "hash", nil)

	_, err := c.InvokeWithRawResult(context.Background(), nil)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("InvokeWithRawResult() = %v, want ErrNotConnected", err)
	}
}

func TestInvokeWithRawResult(t *testing.T) {
	c, _ := NewClient(12345, "hash", &Config{NoUpdates: true})

	_, err := c.InvokeWithRawResult(context.Background(), &tg.PingRequest{})
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("InvokeWithRawResult() = %v, want ErrNotConnected", err)
	}
}

func setupTestServerForClient(c *Client, st storage.Storage) *testServer {
	srv, err := newTestServer(nil)
	if err != nil {
		panic(err)
	}
	c.cfg.TransportMode = TransportModeIntermediate
	authKey, _ := st.AuthKey()
	srv.authKey = authKey
	dialer := &testServerDialer{addr: srv.Addr()}
	c.setTestDialer(transport.Dialer(dialer))
	return srv
}

func TestConnectWithInMemoryStorage(t *testing.T) {
	c, _ := NewClient(12345, "hash", &Config{InMemory: true, SessionName: "test-inmem", NoUpdates: true})
	st := NewMemoryStorage()
	_ = st.SetAuthKey(make([]byte, 256))
	c.setTestStorage(st)

	sess, err := session.NewSession(session.DataCenter{ID: 2}, st, "Test", "0.1", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() = %v", err)
	}
	c.setTestSession(sess)

	srv := setupTestServerForClient(c, st)
	defer srv.Close()

	if err := c.Connect(5 * time.Second); err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	if !c.IsConnected() {
		t.Error("expected connected")
	}
	if c.Storage() == nil {
		t.Error("expected non-nil storage")
	}
	if c.Session() == nil {
		t.Error("expected non-nil session")
	}

	c.Disconnect()
}

func TestHandleUpdatesDroppedWhenNoDispatcher(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)
	c.HandleUpdates(&tg.UpdatesTooLong{})
}

func TestHandleUpdatesDroppedWhenNoUpdates(t *testing.T) {
	c, _ := NewClient(1, "h", &Config{NoUpdates: true})
	disp := &mockDispatcher{}
	c.SetDispatcher(disp)
	c.state.setConnected(true)

	c.HandleUpdates(&tg.UpdatesTooLong{})
	if len(disp.Packets()) != 0 {
		t.Error("no_updates=true should drop all updates")
	}
}

func TestHandleUpdatesDroppedWhenNotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	disp := &mockDispatcher{}
	c.SetDispatcher(disp)

	c.HandleUpdates(&tg.UpdatesTooLong{})
	if len(disp.Packets()) != 0 {
		t.Error("updates should be dropped when not connected")
	}
}

func TestHandleUpdatesEnqueues(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	disp := &mockDispatcher{}
	c.SetDispatcher(disp)
	c.state.setConnected(true)

	c.HandleUpdates(&tg.UpdatesTooLong{})
	packets := disp.Packets()
	if len(packets) != 1 {
		t.Fatalf("expected 1 packet, got %d", len(packets))
	}
	if _, ok := packets[0].Update.(*tg.UpdatesTooLong); !ok {
		t.Error("packet should contain the update")
	}
}

func TestHandleUpdatesBindsMessageMethods(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)

	var captured *types.Message
	c.OnMessage(func(_ *Client, msg *types.Message) {
		captured = msg
	})

	c.HandleUpdates(&tg.UpdateShortMessage{
		ID:      1,
		UserID:  2,
		Message: "hello",
		Date:    1,
	})
	if captured == nil {
		t.Fatal("expected message")
	}

	c.state.setConnected(false)
	_, err := captured.Reply("hello")
	if errors.Is(err, types.ErrNoBinder) {
		t.Fatalf("Reply() = %v, want message bound to client", err)
	}
}

func TestResolvePeerNotConnected(t *testing.T) {
	c, _ := NewClient(12345, "hash", nil)
	_, err := c.ResolvePeer(context.Background(), ChatID(0))
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("ResolvePeer() = %v, want ErrNotConnected", err)
	}
}

func TestResolvePeerSelf(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)

	peer, err := c.ResolvePeer(context.Background(), ChatID(0))
	if err != nil {
		t.Fatalf("ResolvePeer(self) = %v", err)
	}
	if _, ok := peer.(*tg.InputPeerSelf); !ok {
		t.Errorf("ResolvePeer(self) = %T, want InputPeerSelf", peer)
	}
}

func TestResolvePeerEmpty(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)

	peer, err := c.ResolvePeer(context.Background(), ChatPeer(&tg.InputPeerEmpty{}))
	if err != nil {
		t.Fatalf("ResolvePeer(empty) = %v", err)
	}
	if _, ok := peer.(*tg.InputPeerEmpty); !ok {
		t.Errorf("ResolvePeer(empty) = %T, want InputPeerEmpty", peer)
	}
}

func TestExportSessionStringNotConnected(t *testing.T) {
	c, _ := NewClient(12345, "hash", nil)
	_, err := c.ExportSessionString()
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("ExportSessionString() = %v, want ErrNotConnected", err)
	}
}

func TestExportSessionStringWithStorage(t *testing.T) {
	c, _ := NewClient(12345, "hash", &Config{InMemory: true, SessionName: "test-export", NoUpdates: true})
	st := NewMemoryStorage()
	_ = st.SetAuthKey(make([]byte, 256))
	_ = st.SetDCID(2)
	c.setTestStorage(st)

	sess, err := session.NewSession(session.DataCenter{ID: 2}, st, "Test", "0.1", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() = %v", err)
	}
	c.setTestSession(sess)

	srv := setupTestServerForClient(c, st)
	defer srv.Close()

	if err := c.Connect(5 * time.Second); err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	defer c.Disconnect()

	s, err := c.ExportSessionString()
	if err != nil {
		t.Errorf("ExportSessionString() = %v", err)
	}
	if s == "" {
		t.Error("ExportSessionString() returned empty string")
	}
}

func TestGetMeNotConnected(t *testing.T) {
	c, _ := NewClient(12345, "hash", nil)
	_, err := c.GetMe(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("GetMe() = %v, want ErrNotConnected", err)
	}
}

func TestConfigAccessors(t *testing.T) {
	c, _ := NewClient(12345, "abcdef", &Config{
		SessionName: "mysession",
		BotToken:    "123:TOKEN",
		DC:          4,
		TestMode:    true,
		NoUpdates:   true,
	})

	if c.APIID() != 12345 {
		t.Errorf("APIID() = %d, want 12345", c.APIID())
	}
	if c.APIHash() != "abcdef" {
		t.Errorf("APIHash() = %q", c.APIHash())
	}
	if c.SessionName() != "mysession" {
		t.Errorf("SessionName() = %q", c.SessionName())
	}
	if c.BotToken() != "123:TOKEN" {
		t.Errorf("BotToken() = %q", c.BotToken())
	}
	if c.DC() != 4 {
		t.Errorf("DC() = %d, want 4", c.DC())
	}
	if !c.TestMode() {
		t.Error("TestMode() should be true")
	}
	if !c.NoUpdates() {
		t.Error("NoUpdates() should be true")
	}
	if !c.IsBot() {
		t.Error("IsBot() should be true with BotToken set")
	}

	c.SetBotToken("new-token")
	if c.BotToken() != "new-token" {
		t.Errorf("BotToken() = %q after SetBotToken", c.BotToken())
	}

	c2, _ := NewClient(1, "h", nil)
	if c2.IsBot() {
		t.Error("IsBot() = true with no BotToken")
	}
}

func TestWithConfig(t *testing.T) {
	c, _ := NewClient(111, "hash", &Config{
		SessionName:    "struct-session",
		BotToken:       "999:TOKEN",
		DC:             3,
		TestMode:       true,
		LangCode:       "de",
		DeviceModel:    "Pixel 8",
		ClientPlatform: types.ClientPlatformAndroid,
	})

	if c.SessionName() != "struct-session" {
		t.Errorf("SessionName = %q", c.SessionName())
	}
	if c.BotToken() != "999:TOKEN" {
		t.Errorf("BotToken = %q", c.BotToken())
	}
	if c.DC() != 3 {
		t.Errorf("DC = %d, want 3", c.DC())
	}
	if !c.TestMode() {
		t.Error("TestMode should be true")
	}
	if c.APIID() != 111 {
		t.Errorf("APIID = %d, want 111", c.APIID())
	}
}

func TestWithConfigMixedWithOption(t *testing.T) {
	c, _ := NewClient(1, "h", &Config{
		SessionName: "from-struct",
		BotToken:    "override-token",
	})

	if c.SessionName() != "from-struct" {
		t.Errorf("SessionName = %q", c.SessionName())
	}
	if c.BotToken() != "override-token" {
		t.Errorf("BotToken = %q", c.BotToken())
	}
}

func TestInitialDCID(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	st := NewMemoryStorage()

	if got := c.initialDCID(st); got != 2 {
		t.Fatalf("initialDCID empty storage = %d, want 2", got)
	}

	if err := st.SetDCID(4); err != nil {
		t.Fatalf("SetDCID error: %v", err)
	}
	if got := c.initialDCID(st); got != 4 {
		t.Fatalf("initialDCID stored = %d, want 4", got)
	}

	c, _ = NewClient(1, "h", &Config{DC: 3})
	if got := c.initialDCID(st); got != 3 {
		t.Fatalf("initialDCID configured = %d, want 3", got)
	}
}

func TestLogOutNotConnected(t *testing.T) {
	c, _ := NewClient(12345, "hash", nil)
	err := c.LogOut()
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("LogOut() = %v, want ErrNotConnected", err)
	}
}

func TestSetMe(t *testing.T) {
	c, _ := NewClient(12345, "hash", nil)

	if c.Me() != nil {
		t.Error("Me() should be nil initially")
	}

	c.SetMe(&types.User{ID: 42, FirstName: "test-user"})
	if c.Me() == nil || c.Me().FirstName != "test-user" {
		t.Errorf("Me() = %v, want test-user", c.Me())
	}
}

func TestClientServerTime(t *testing.T) {
	c, _ := NewClient(1, "h", &Config{TZOffset: 3})
	got := c.ServerTime()
	now := int32(time.Now().Unix()) + 3
	if got < now-2 || got > now+2 {
		t.Errorf("ServerTime() = %d, want approximately %d", got, now)
	}
}

func TestGetSessionReturnsMainForCurrentDC(t *testing.T) {
	c, _ := NewClient(1, "h", &Config{InMemory: true, SessionName: "test", NoUpdates: true})
	st := NewMemoryStorage()
	_ = st.SetAuthKey(make([]byte, 256))
	_ = st.SetDCID(2)
	c.setTestStorage(st)

	mainSess, err := session.NewSession(session.DataCenter{ID: 2}, st, "Test", "0.1", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() = %v", err)
	}
	c.setTestSession(mainSess)

	srv := setupTestServerForClient(c, st)
	defer srv.Close()

	if err := c.Connect(5 * time.Second); err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	defer c.Disconnect()

	sess, err := c.GetSession(context.Background(), 2, false, false)
	if err != nil {
		t.Fatalf("GetSession(2) = %v", err)
	}
	if sess != mainSess {
		t.Error("GetSession should return main session for current DC")
	}
}

func TestGetSessionCachesOtherDCSessions(t *testing.T) {
	c, _ := NewClient(1, "h", &Config{InMemory: true, SessionName: "test", NoUpdates: true})
	st := NewMemoryStorage()
	_ = st.SetAuthKey(make([]byte, 256))
	_ = st.SetDCID(2)
	c.setTestStorage(st)

	callCount := 0
	c.setTestSessionFactory(func(ctx context.Context, dcID int, addr string, port int, authKey []byte) (*session.Session, error) {
		callCount++
		return &session.Session{}, nil
	})

	mainSess, err := session.NewSession(session.DataCenter{ID: 2}, st, "Test", "0.1", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() = %v", err)
	}
	c.setTestSession(mainSess)

	srv := setupTestServerForClient(c, st)
	defer srv.Close()

	if err := c.Connect(5 * time.Second); err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	defer c.Disconnect()

	sess1, err := c.GetSession(context.Background(), 4, false, false)
	if err != nil {
		t.Fatalf("GetSession(4) = %v", err)
	}
	sess2, err := c.GetSession(context.Background(), 4, false, false)
	if err != nil {
		t.Fatalf("GetSession(4) second = %v", err)
	}

	if sess1 != sess2 {
		t.Error("second call should return cached session")
	}
	if callCount != 1 {
		t.Errorf("session factory called %d times, want 1", callCount)
	}
}

func TestDisconnectStopsAllSessions(t *testing.T) {
	c, _ := NewClient(1, "h", &Config{InMemory: true, SessionName: "test", NoUpdates: true})
	st := NewMemoryStorage()
	_ = st.SetAuthKey(make([]byte, 256))
	_ = st.SetDCID(2)
	c.setTestStorage(st)

	mainSess, err := session.NewSession(session.DataCenter{ID: 2}, st, "Test", "0.1", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() = %v", err)
	}
	c.setTestSession(mainSess)

	c.setTestSessionFactory(func(ctx context.Context, dcID int, addr string, port int, authKey []byte) (*session.Session, error) {
		return &session.Session{}, nil
	})

	srv := setupTestServerForClient(c, st)
	defer srv.Close()

	if err := c.Connect(5 * time.Second); err != nil {
		t.Fatalf("Connect() = %v", err)
	}

	c.GetSession(context.Background(), 4, false, false)
	c.GetSession(context.Background(), 5, true, false)

	if err := c.Disconnect(); err != nil {
		t.Fatalf("Disconnect() = %v", err)
	}

	c.sessionsMu.Lock()
	count := len(c.sessions)
	c.sessionsMu.Unlock()

	if count != 0 {
		t.Errorf("sessions map should be empty after disconnect, has %d", count)
	}
}

type mockDispatcher struct {
	packets []UpdatePacket
	mu      sync.Mutex
}

func (m *mockDispatcher) Start(workers int) error        { return nil }
func (m *mockDispatcher) Stop() error                    { return nil }
func (m *mockDispatcher) AddHandler(_ Handler, _ int)    {}
func (m *mockDispatcher) RemoveHandler(_ Handler, _ int) {}
func (m *mockDispatcher) Enqueue(pkt UpdatePacket) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.packets = append(m.packets, pkt)
	return nil
}

func (m *mockDispatcher) Packets() []UpdatePacket {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.packets
}

func TestFullLifecycle(t *testing.T) {
	c, _ := NewClient(42, "testhash", &Config{InMemory: true, SessionName: "lifecycle", NoUpdates: true})
	st := NewMemoryStorage()
	_ = st.SetAuthKey(make([]byte, 256))
	_ = st.SetDCID(2)
	c.setTestStorage(st)

	sess, err := session.NewSession(session.DataCenter{ID: 2}, st, "Test", "0.1", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() = %v", err)
	}
	c.setTestSession(sess)

	srv := setupTestServerForClient(c, st)
	defer srv.Close()

	disp := &mockDispatcher{}
	c.SetDispatcher(disp)

	if c.IsConnected() {
		t.Fatal("should not be connected before Connect")
	}

	if err := c.Connect(5 * time.Second); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	c.cfg.NoUpdates = false

	if !c.IsConnected() {
		t.Fatal("should be connected")
	}

	c.HandleUpdates(&tg.UpdatesTooLong{})
	if len(disp.Packets()) != 1 {
		t.Errorf("expected 1 dispatched packet, got %d", len(disp.Packets()))
	}

	c.SetMe(&types.User{ID: 42, FirstName: "user-42"})
	if c.Me() == nil || c.Me().FirstName != "user-42" {
		t.Error("cached me should be set")
	}

	s, err := c.ExportSessionString()
	if err != nil {
		t.Errorf("ExportSessionString: %v", err)
	}
	if s == "" {
		t.Error("session string should not be empty")
	}

	if err := c.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if c.IsConnected() {
		t.Fatal("should not be connected after Disconnect")
	}
	if c.Me() != nil {
		t.Error("me should be nil after disconnect")
	}
}

func TestAllMethodsGuardNotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)

	_, err := c.Invoke(context.Background(), nil, 1, 5*time.Second)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("Invoke: %v", err)
	}

	_, err = c.InvokeRaw(context.Background(), nil, 1, 5*time.Second)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("InvokeRaw: %v", err)
	}

	_, err = c.ResolvePeer(context.Background(), ChatID(0))
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("ResolvePeer: %v", err)
	}

	_, err = c.GetMe(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("GetMe: %v", err)
	}

	_, err = c.ExportSessionString()
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("ExportSessionString: %v", err)
	}

	err = c.LogOut()
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("LogOut: %v", err)
	}

	_, err = c.GetSession(context.Background(), 2, false, false)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("GetSession: %v", err)
	}

	err = c.EnableCloudPassword(context.Background(), "pw", "hint")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("EnableCloudPassword: %v", err)
	}

	err = c.ChangeCloudPassword(context.Background(), "old", "new", "hint")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("ChangeCloudPassword: %v", err)
	}

	err = c.RemoveCloudPassword(context.Background(), "pw")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("RemoveCloudPassword: %v", err)
	}

	_, err = c.GetQRCodeLoginToken(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("GetQRCodeLoginToken: %v", err)
	}

	_, err = c.CheckQRCodeLoginToken(context.Background(), []byte("tok"))
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("CheckQRCodeLoginToken: %v", err)
	}

	err = c.SetPrivacy(context.Background(), &tg.InputPrivacyKeyStatusTimestamp{}, nil)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("SetPrivacy: %v", err)
	}

	_, err = c.GetPrivacy(context.Background(), &tg.InputPrivacyKeyStatusTimestamp{})
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("GetPrivacy: %v", err)
	}

	err = c.SetGlobalPrivacySettings(context.Background(), &tg.GlobalPrivacySettings{})
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("SetGlobalPrivacySettings: %v", err)
	}

	_, err = c.GetGlobalPrivacySettings(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("GetGlobalPrivacySettings: %v", err)
	}

	err = c.SetAccountTTL(context.Background(), 180)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("SetAccountTTL: %v", err)
	}

	_, err = c.GetAccountTTL(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("GetAccountTTL: %v", err)
	}

	err = c.SetProfilePhoto(context.Background(), &tg.InputFile{})
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("SetProfilePhoto: %v", err)
	}

	err = c.DeleteProfilePhoto(context.Background(), 123)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("DeleteProfilePhoto: %v", err)
	}

	err = c.SetUsername(context.Background(), "user")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("SetUsername: %v", err)
	}

	err = c.SetBio(context.Background(), "bio")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("SetBio: %v", err)
	}

	_, err = c.GetProfilePhotos(context.Background(), 0, nil)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("GetProfilePhotos: %v", err)
	}
}

func TestClientUpdateHealthNoManager(t *testing.T) {
	c, _ := NewClient(1, "hash", nil)
	h := c.UpdateHealth()
	if h.Pts != 0 || h.Pending != 0 {
		t.Fatalf("health = %+v", h)
	}
}

type closeTrackingStorage struct {
	*MemoryStorage
	closed bool
}

func (s *closeTrackingStorage) Close() error {
	s.closed = true
	return nil
}

func TestCleanupSessionsCanKeepStorageForMigration(t *testing.T) {
	c, _ := NewClient(1, "hash", nil)
	st := &closeTrackingStorage{MemoryStorage: NewMemoryStorage()}
	c.storage = st

	c.cleanupSessions(false)
	if st.closed {
		t.Fatal("storage should stay open for migration retry")
	}
	if c.storage != st {
		t.Fatal("storage should remain attached for migration retry")
	}

	c.cleanupSessions()
	if !st.closed {
		t.Fatal("storage should close during regular cleanup")
	}
	if c.storage != nil {
		t.Fatal("storage should be released during regular cleanup")
	}
}
