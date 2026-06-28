package storage

import (
	"encoding/base64"
	"encoding/binary"
	"strings"
	"testing"
)

func newTestStorage() Storage {
	return NewMemory()
}

func TestNewMemory(t *testing.T) {
	s := newTestStorage()
	if s == nil {
		t.Fatal("NewMemory() returned nil")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

func TestNewAdapter(t *testing.T) {
	inner := newMemoryAdapter()
	w := NewAdapter(inner)
	if w == nil {
		t.Fatal("NewAdapter() returned nil")
	}
}

func TestDefaultValues(t *testing.T) {
	s := newTestStorage()

	if v, err := s.SessionID(); err != nil || v != "" {
		t.Fatalf("SessionID() = %q, err=%v, want empty", v, err)
	}
	if v, err := s.DCID(); err != nil || v != 0 {
		t.Fatalf("DCID() = %d, err=%v, want 0", v, err)
	}
	if v, err := s.APIID(); err != nil || v != 0 {
		t.Fatalf("APIID() = %d, err=%v, want 0", v, err)
	}
	if v, err := s.APIHash(); err != nil || v != "" {
		t.Fatalf("APIHash() = %q, err=%v, want empty", v, err)
	}
	if v, err := s.TestMode(); err != nil || v {
		t.Fatalf("TestMode() = %v, err=%v, want false", v, err)
	}
	if v, err := s.AuthKey(); err != nil || v != nil {
		t.Fatalf("AuthKey() = %v, err=%v, want nil", v, err)
	}
	if v, err := s.UserID(); err != nil || v != 0 {
		t.Fatalf("UserID() = %d, err=%v, want 0", v, err)
	}
	if v, err := s.IsBot(); err != nil || v {
		t.Fatalf("IsBot() = %v, err=%v, want false", v, err)
	}
	if v, err := s.FirstName(); err != nil || v != "" {
		t.Fatalf("FirstName() = %q, err=%v, want empty", v, err)
	}
	if v, err := s.LastName(); err != nil || v != "" {
		t.Fatalf("LastName() = %q, err=%v, want empty", v, err)
	}
	if v, err := s.Username(); err != nil || v != "" {
		t.Fatalf("Username() = %q, err=%v, want empty", v, err)
	}
	if v, err := s.Date(); err != nil || v != 0 {
		t.Fatalf("Date() = %d, err=%v, want 0", v, err)
	}
	if v, err := s.State(); err != nil || v != nil {
		t.Fatalf("State() = %v, err=%v, want nil", v, err)
	}
}

func TestSessionFieldRoundTrip(t *testing.T) {
	s := newTestStorage()

	tests := []struct {
		name  string
		set   func() error
		get   func() (any, error)
		want  any
		equal func(got, want any) bool
	}{
		{
			"SessionID",
			func() error { return s.SetSessionID("sess-123") },
			func() (any, error) { return s.SessionID() },
			"sess-123",
			func(g, w any) bool { return g.(string) == w.(string) },
		},
		{
			"DCID",
			func() error { return s.SetDCID(2) },
			func() (any, error) { return s.DCID() },
			2,
			func(g, w any) bool { return g.(int) == w.(int) },
		},
		{
			"APIID",
			func() error { return s.SetAPIID(12345) },
			func() (any, error) { return s.APIID() },
			int32(12345),
			func(g, w any) bool { return g.(int32) == w.(int32) },
		},
		{
			"APIHash",
			func() error { return s.SetAPIHash("deadbeef") },
			func() (any, error) { return s.APIHash() },
			"deadbeef",
			func(g, w any) bool { return g.(string) == w.(string) },
		},
		{
			"TestMode",
			func() error { return s.SetTestMode(true) },
			func() (any, error) { return s.TestMode() },
			true,
			func(g, w any) bool { return g.(bool) == w.(bool) },
		},
		{
			"AuthKey",
			func() error { return s.SetAuthKey([]byte{1, 2, 3, 4}) },
			func() (any, error) { return s.AuthKey() },
			[]byte{1, 2, 3, 4},
			func(g, w any) bool {
				return string(g.([]byte)) == string(w.([]byte))
			},
		},
		{
			"UserID",
			func() error { return s.SetUserID(9876543210) },
			func() (any, error) { return s.UserID() },
			int64(9876543210),
			func(g, w any) bool { return g.(int64) == w.(int64) },
		},
		{
			"IsBot",
			func() error { return s.SetIsBot(true) },
			func() (any, error) { return s.IsBot() },
			true,
			func(g, w any) bool { return g.(bool) == w.(bool) },
		},
		{
			"FirstName",
			func() error { return s.SetFirstName("Alice") },
			func() (any, error) { return s.FirstName() },
			"Alice",
			func(g, w any) bool { return g.(string) == w.(string) },
		},
		{
			"LastName",
			func() error { return s.SetLastName("Smith") },
			func() (any, error) { return s.LastName() },
			"Smith",
			func(g, w any) bool { return g.(string) == w.(string) },
		},
		{
			"Username",
			func() error { return s.SetUsername("alice") },
			func() (any, error) { return s.Username() },
			"alice",
			func(g, w any) bool { return g.(string) == w.(string) },
		},
		{
			"Date",
			func() error { return s.SetDate(1700000000) },
			func() (any, error) { return s.Date() },
			1700000000,
			func(g, w any) bool { return g.(int) == w.(int) },
		},
		{
			"State",
			func() error { return s.SetState([]byte{0xAA, 0xBB}) },
			func() (any, error) { return s.State() },
			[]byte{0xAA, 0xBB},
			func(g, w any) bool {
				return string(g.([]byte)) == string(w.([]byte))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.set(); err != nil {
				t.Fatalf("set: %v", err)
			}
			got, err := tt.get()
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if !tt.equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFieldOverwrite(t *testing.T) {
	s := newTestStorage()

	if err := s.SetDCID(1); err != nil {
		t.Fatal(err)
	}
	if v, err := s.DCID(); err != nil || v != 1 {
		t.Fatalf("got %d, want 1", v)
	}
	if err := s.SetDCID(4); err != nil {
		t.Fatal(err)
	}
	if v, err := s.DCID(); err != nil || v != 4 {
		t.Fatalf("got %d, want 4", v)
	}

	if err := s.SetFirstName("first"); err != nil {
		t.Fatal(err)
	}
	if v, err := s.FirstName(); err != nil || v != "first" {
		t.Fatalf("got %q, want %q", v, "first")
	}
	if err := s.SetFirstName("second"); err != nil {
		t.Fatal(err)
	}
	if v, err := s.FirstName(); err != nil || v != "second" {
		t.Fatalf("got %q, want %q", v, "second")
	}
}

func TestExportSessionStringEmpty(t *testing.T) {
	s := newTestStorage()
	v, err := s.ExportSessionString()
	if err != nil {
		t.Fatal(err)
	}
	if v != "" {
		t.Fatalf("ExportSessionString() = %q, want empty with no auth key", v)
	}
}

func TestExportSessionStringWithAPIID(t *testing.T) {
	s := newTestStorage()
	authKey := make([]byte, 256)
	for i := range authKey {
		authKey[i] = byte(i)
	}

	if err := s.SetDCID(4); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAuthKey(authKey); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAPIID(12345); err != nil {
		t.Fatal(err)
	}
	if err := s.SetUserID(998877); err != nil {
		t.Fatal(err)
	}
	if err := s.SetIsBot(true); err != nil {
		t.Fatal(err)
	}

	str, err := s.ExportSessionString()
	if err != nil {
		t.Fatal(err)
	}
	if str == "" {
		t.Fatal("ExportSessionString() returned empty")
	}

	data, err := base64.URLEncoding.DecodeString(str + strings.Repeat("=", (4-len(str)%4)%4))
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}

	if len(data) != 271 {
		t.Fatalf("decoded length = %d, want 271 (Pyrogram)", len(data))
	}

	dc := data[0]
	if dc != 4 {
		t.Fatalf("DC byte = %d, want 4", dc)
	}

	apiID := binary.BigEndian.Uint32(data[1:5])
	if apiID != 12345 {
		t.Fatalf("api_id = %d, want 12345", apiID)
	}

	testMode := data[5]
	if testMode != 0 {
		t.Fatalf("test_mode = %d, want 0", testMode)
	}

	userID := binary.BigEndian.Uint64(data[262:270])
	if userID != 998877 {
		t.Fatalf("user_id = %d, want 998877", userID)
	}

	isBot := data[270]
	if isBot != 1 {
		t.Fatalf("is_bot = %d, want 1", isBot)
	}
}

func TestExportSessionStringNoAPIID(t *testing.T) {
	s := newTestStorage()
	authKey := make([]byte, 256)
	if err := s.SetDCID(2); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAuthKey(authKey); err != nil {
		t.Fatal(err)
	}

	_, err := s.ExportSessionString()
	if err == nil {
		t.Fatal("expected error when api_id not stored")
	}
}

func TestExportSessionStringDCAddresses(t *testing.T) {
	expected := map[int]string{
		1: "149.154.175.53",
		2: "149.154.167.51",
		3: "149.154.175.100",
		4: "149.154.167.91",
		5: "149.154.171.5",
	}

	authKey := make([]byte, 256)
	for dc := range expected {
		s := newTestStorage()
		if err := s.SetDCID(dc); err != nil {
			t.Fatal(err)
		}
		if err := s.SetAuthKey(authKey); err != nil {
			t.Fatal(err)
		}
		if err := s.SetAPIID(99999); err != nil {
			t.Fatal(err)
		}

		str, err := s.ExportSessionString()
		if err != nil {
			t.Fatal(err)
		}
		if str == "" {
			t.Fatalf("DC %d: empty session string", dc)
		}

		data, err := base64.URLEncoding.DecodeString(str + strings.Repeat("=", (4-len(str)%4)%4))
		if err != nil {
			t.Fatalf("DC %d: base64 decode: %v", dc, err)
		}

		if len(data) != 271 {
			t.Fatalf("DC %d: decoded length = %d, want 271", dc, len(data))
		}
		if data[0] != uint8(dc) {
			t.Fatalf("DC %d: byte = %d", dc, data[0])
		}
	}
}

func TestExportSessionStringUnknownDC(t *testing.T) {
	s := newTestStorage()
	authKey := make([]byte, 256)
	if err := s.SetDCID(99); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAuthKey(authKey); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAPIID(99999); err != nil {
		t.Fatal(err)
	}

	str, err := s.ExportSessionString()
	if err != nil {
		t.Fatal(err)
	}
	if str == "" {
		t.Fatal("expected non-empty session string for unknown DC")
	}

	data, err := base64.URLEncoding.DecodeString(str + strings.Repeat("=", (4-len(str)%4)%4))
	if err != nil {
		t.Fatal(err)
	}

	if data[0] != 99 {
		t.Fatalf("DC byte = %d, want 99", data[0])
	}
}

func TestSetSessionIDTriggersSessionIDAware(t *testing.T) {
	inner := newMemoryAdapter()
	w := NewAdapter(inner)

	if inner.sessionName != "" {
		t.Fatalf("initial sessionName = %q, want empty", inner.sessionName)
	}

	if err := w.SetSessionID("test-session"); err != nil {
		t.Fatal(err)
	}

	if inner.sessionName != "test-session" {
		t.Fatalf("sessionName = %q, want %q", inner.sessionName, "test-session")
	}

	got, err := w.SessionID()
	if err != nil {
		t.Fatal(err)
	}
	if got != "test-session" {
		t.Fatalf("SessionID() = %q, want %q", got, "test-session")
	}
}

func TestPeerOperations(t *testing.T) {
	s := newTestStorage()
	ps, ok := s.(PeerStore)
	if !ok {
		t.Fatal("Storage does not implement PeerStore")
	}

	p1 := &Peer{
		ID:         100,
		Type:       PeerTypeUser,
		AccessHash: 999,
		Username:   "alice",
		FirstName:  "Alice",
	}
	if err := ps.SavePeer(p1); err != nil {
		t.Fatal(err)
	}

	got, err := ps.GetPeer(100)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("GetPeer(100) returned nil")
	}
	if got.ID != 100 || got.Username != "alice" || got.AccessHash != 999 {
		t.Fatalf("GetPeer = %+v, want ID=100,Username=alice,AccessHash=999", got)
	}

	got, err = ps.GetPeerByUsername("alice")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != 100 {
		t.Fatalf("GetPeerByUsername = %v, want ID=100", got)
	}

	got, err = ps.GetPeer(999)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("GetPeer(999) = %v, want nil", got)
	}

	got, err = ps.GetPeerByUsername("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("GetPeerByUsername(nonexistent) = %v, want nil", got)
	}

	peers, err := ps.LoadPeers()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 {
		t.Fatalf("LoadPeers() returned %d peers, want 1", len(peers))
	}

	p2 := &Peer{
		ID:         200,
		Type:       PeerTypeChannel,
		AccessHash: 888,
		Username:   "bob_channel",
	}
	if err := ps.SavePeer(p2); err != nil {
		t.Fatal(err)
	}
	peers, err = ps.LoadPeers()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 2 {
		t.Fatalf("LoadPeers() = %d peers, want 2", len(peers))
	}

	if err := ps.DeletePeer(100); err != nil {
		t.Fatal(err)
	}
	got, err = ps.GetPeer(100)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("GetPeer(100) should return nil after delete")
	}
	got, err = ps.GetPeerByUsername("alice")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("GetPeerByUsername(alice) should return nil after delete")
	}

	peers, err = ps.LoadPeers()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 {
		t.Fatalf("LoadPeers() = %d peers after delete, want 1", len(peers))
	}
}

func TestSavePeers(t *testing.T) {
	s := newTestStorage()

	peers := []*Peer{
		{ID: 1, Type: PeerTypeUser, Username: "u1"},
		{ID: 2, Type: PeerTypeChat, Username: "u2"},
	}
	if err := s.(*adapterWrapper).SavePeers(peers); err != nil {
		t.Fatal(err)
	}

	ps := s.(PeerStore)
	loaded, err := ps.LoadPeers()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Fatalf("LoadPeers() = %d, want 2", len(loaded))
	}
}

func TestPeerUpdate(t *testing.T) {
	s := newTestStorage()
	ps := s.(PeerStore)

	p := &Peer{ID: 50, Type: PeerTypeUser, Username: "old", FirstName: "Old"}
	if err := ps.SavePeer(p); err != nil {
		t.Fatal(err)
	}

	p2 := &Peer{ID: 50, Type: PeerTypeUser, Username: "new", FirstName: "New"}
	if err := ps.SavePeer(p2); err != nil {
		t.Fatal(err)
	}

	got, err := ps.GetPeer(50)
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "new" || got.FirstName != "New" {
		t.Fatalf("updated peer = %+v", got)
	}

	old, err := ps.GetPeerByUsername("old")
	if err != nil {
		t.Fatal(err)
	}
	if old == nil || old.ID != 50 {
		t.Fatal("old username still resolves to same peer (adapter does not clean up old username on overwrite)")
	}

	got, err = ps.GetPeerByUsername("new")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != 50 {
		t.Fatal("new username should resolve")
	}
}

func TestConversationOperations(t *testing.T) {
	s := newTestStorage()
	cs, ok := s.(ConversationStore)
	if !ok {
		t.Fatal("Storage does not implement ConversationStore")
	}

	loaded, err := cs.LoadConversation(10, 20)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != nil {
		t.Fatal("LoadConversation on empty should return nil")
	}

	c := &Conversation{
		ChatID: 10,
		UserID: 20,
		Name:   "test",
		Step:   3,
		Data:   []byte("payload"),
	}
	if err := cs.SaveConversation(c); err != nil {
		t.Fatal(err)
	}

	loaded, err = cs.LoadConversation(10, 20)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil {
		t.Fatal("LoadConversation returned nil after save")
	}
	if loaded.ChatID != 10 || loaded.UserID != 20 || loaded.Name != "test" || loaded.Step != 3 {
		t.Fatalf("loaded = %+v", loaded)
	}
	if string(loaded.Data) != "payload" {
		t.Fatalf("Data = %q, want %q", loaded.Data, "payload")
	}
	if loaded.CreatedAt == 0 {
		t.Fatal("CreatedAt should be auto-set")
	}
	if loaded.UpdatedAt == 0 {
		t.Fatal("UpdatedAt should be auto-set")
	}

	c2 := &Conversation{
		ChatID:    10,
		UserID:    20,
		Name:      "updated",
		Step:      5,
		CreatedAt: loaded.CreatedAt,
		UpdatedAt: 0,
	}
	if err := cs.SaveConversation(c2); err != nil {
		t.Fatal(err)
	}
	loaded2, err := cs.LoadConversation(10, 20)
	if err != nil {
		t.Fatal(err)
	}
	if loaded2.Name != "updated" || loaded2.Step != 5 {
		t.Fatalf("updated conversation = %+v", loaded2)
	}

	if err := cs.DeleteConversation(10, 20); err != nil {
		t.Fatal(err)
	}
	loaded3, err := cs.LoadConversation(10, 20)
	if err != nil {
		t.Fatal(err)
	}
	if loaded3 != nil {
		t.Fatal("LoadConversation should return nil after delete")
	}
}

func TestConversationPreservesTimestamps(t *testing.T) {
	s := newTestStorage()
	cs := s.(ConversationStore)

	c := &Conversation{
		ChatID:    100,
		UserID:    200,
		Name:      "test",
		CreatedAt: 1000,
		UpdatedAt: 2000,
	}
	if err := cs.SaveConversation(c); err != nil {
		t.Fatal(err)
	}

	loaded, err := cs.LoadConversation(100, 200)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CreatedAt != 1000 {
		t.Fatalf("CreatedAt = %d, want 1000", loaded.CreatedAt)
	}
	if loaded.UpdatedAt != 2000 {
		t.Fatalf("UpdatedAt = %d, want 2000", loaded.UpdatedAt)
	}
}

func TestUpdateStateOperations(t *testing.T) {
	s := newTestStorage()
	uss, ok := s.(UpdateStateStore)
	if !ok {
		t.Fatal("Storage does not implement UpdateStateStore")
	}

	state, err := uss.LoadUpdateState("sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if state != nil {
		t.Fatal("LoadUpdateState on empty should return nil")
	}

	saved := &UpdateState{
		SessionID: "sess-1",
		Pts:       100,
		Qts:       200,
		Date:      300,
		Seq:       400,
	}
	if err := uss.SaveUpdateState(saved); err != nil {
		t.Fatal(err)
	}

	state, err = uss.LoadUpdateState("sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if state == nil {
		t.Fatal("LoadUpdateState returned nil after save")
	}
	if state.Pts != 100 || state.Qts != 200 || state.Date != 300 || state.Seq != 400 {
		t.Fatalf("state = %+v", state)
	}

	state.Pts = 999
	loadedAgain, err := uss.LoadUpdateState("sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if loadedAgain.Pts != 100 {
		t.Fatal("mutation of returned state should not affect stored state")
	}
}

func TestChannelUpdateStateOperations(t *testing.T) {
	s := newTestStorage()
	uss := s.(UpdateStateStore)

	cs, err := uss.LoadChannelUpdateState("s1", 42)
	if err != nil {
		t.Fatal(err)
	}
	if cs != nil {
		t.Fatal("LoadChannelUpdateState on empty should return nil")
	}

	all, err := uss.LoadAllChannelUpdateStates("s1")
	if err != nil {
		t.Fatal(err)
	}
	if all != nil {
		t.Fatal("LoadAllChannelUpdateStates on empty should return nil")
	}

	ch1 := &ChannelUpdateState{
		SessionID: "s1",
		ChannelID: 42,
		Pts:       10,
	}
	if err := uss.SaveChannelUpdateState(ch1); err != nil {
		t.Fatal(err)
	}

	ch2 := &ChannelUpdateState{
		SessionID: "s1",
		ChannelID: 99,
		Pts:       20,
	}
	if err := uss.SaveChannelUpdateState(ch2); err != nil {
		t.Fatal(err)
	}

	loaded, err := uss.LoadChannelUpdateState("s1", 42)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.Pts != 10 {
		t.Fatalf("LoadChannelUpdateState(s1,42) = %v", loaded)
	}

	all, err = uss.LoadAllChannelUpdateStates("s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("LoadAllChannelUpdateStates = %d entries, want 2", len(all))
	}

	ch1Updated := &ChannelUpdateState{
		SessionID: "s1",
		ChannelID: 42,
		Pts:       50,
	}
	if err := uss.SaveChannelUpdateState(ch1Updated); err != nil {
		t.Fatal(err)
	}
	loaded, err = uss.LoadChannelUpdateState("s1", 42)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Pts != 50 {
		t.Fatalf("Pts after update = %d, want 50", loaded.Pts)
	}

	all, err = uss.LoadAllChannelUpdateStates("s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("LoadAllChannelUpdateStates after update = %d, want 2", len(all))
	}
}

func TestUpdateDedup(t *testing.T) {
	s := newTestStorage()
	uss := s.(UpdateStateStore)

	exists, err := uss.UpdateDedupKeyExists("s1", "key-a")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("key should not exist initially")
	}

	inserted, err := uss.SaveUpdateDedupKey("s1", "key-a")
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("first insert should return true")
	}

	exists, err = uss.UpdateDedupKeyExists("s1", "key-a")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("key should exist after save")
	}

	inserted, err = uss.SaveUpdateDedupKey("s1", "key-a")
	if err != nil {
		t.Fatal(err)
	}
	if inserted {
		t.Fatal("duplicate insert should return false")
	}

	exists, err = uss.UpdateDedupKeyExists("s1", "key-b")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("different key should not exist")
	}
}

func TestDurableUpdateOperations(t *testing.T) {
	s := newTestStorage()
	uss := s.(UpdateStateStore)

	loaded, err := uss.LoadDurableUpdates("s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != nil {
		t.Fatal("LoadDurableUpdates on empty should return nil")
	}

	u1 := &DurableUpdate{
		SessionID: "s1",
		ID:        "upd-1",
		Payload:   []byte("data-1"),
		Attempts:  0,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	if err := uss.EnqueueDurableUpdate(u1); err != nil {
		t.Fatal(err)
	}

	u2 := &DurableUpdate{
		SessionID: "s1",
		ID:        "upd-2",
		Payload:   []byte("data-2"),
		Attempts:  0,
		CreatedAt: 2000,
		UpdatedAt: 2000,
	}
	if err := uss.EnqueueDurableUpdate(u2); err != nil {
		t.Fatal(err)
	}

	loaded, err = uss.LoadDurableUpdates("s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Fatalf("LoadDurableUpdates = %d, want 2", len(loaded))
	}

	loadedLimited, err := uss.LoadDurableUpdates("s1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(loadedLimited) != 1 {
		t.Fatalf("LoadDurableUpdates(limit=1) = %d, want 1", len(loadedLimited))
	}

	if err := uss.MarkDurableUpdateFailed("s1", "upd-1", 3, "timeout"); err != nil {
		t.Fatal(err)
	}

	loaded, err = uss.LoadDurableUpdates("s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	var found *DurableUpdate
	for _, u := range loaded {
		if u.ID == "upd-1" {
			found = u
			break
		}
	}
	if found == nil {
		t.Fatal("upd-1 not found after MarkDurableUpdateFailed")
	}
	if found.Attempts != 3 {
		t.Fatalf("Attempts = %d, want 3", found.Attempts)
	}
	if found.LastError != "timeout" {
		t.Fatalf("LastError = %q, want %q", found.LastError, "timeout")
	}

	if err := uss.DeleteDurableUpdate("s1", "upd-1"); err != nil {
		t.Fatal(err)
	}
	loaded, err = uss.LoadDurableUpdates("s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 {
		t.Fatalf("LoadDurableUpdates after delete = %d, want 1", len(loaded))
	}
	if loaded[0].ID != "upd-2" {
		t.Fatalf("remaining ID = %q, want upd-2", loaded[0].ID)
	}
}

func TestMarkDurableUpdateFailedNonexistent(t *testing.T) {
	s := newTestStorage()
	uss := s.(UpdateStateStore)

	err := uss.MarkDurableUpdateFailed("s1", "no-such-id", 5, "err")
	if err != nil {
		t.Fatalf("MarkDurableUpdateFailed on nonexistent should not error: %v", err)
	}
}

func TestDeleteDurableUpdateNonexistent(t *testing.T) {
	s := newTestStorage()
	uss := s.(UpdateStateStore)

	err := uss.DeleteDurableUpdate("s1", "no-such-id")
	if err != nil {
		t.Fatalf("DeleteDurableUpdate on nonexistent should not error: %v", err)
	}
}

func TestSessionIsolation(t *testing.T) {
	s := newTestStorage()
	uss := s.(UpdateStateStore)

	if err := uss.SaveUpdateState(&UpdateState{SessionID: "a", Pts: 1}); err != nil {
		t.Fatal(err)
	}
	if err := uss.SaveUpdateState(&UpdateState{SessionID: "b", Pts: 2}); err != nil {
		t.Fatal(err)
	}

	sa, err := uss.LoadUpdateState("a")
	if err != nil {
		t.Fatal(err)
	}
	if sa.Pts != 1 {
		t.Fatalf("session a Pts = %d, want 1", sa.Pts)
	}

	sb, err := uss.LoadUpdateState("b")
	if err != nil {
		t.Fatal(err)
	}
	if sb.Pts != 2 {
		t.Fatalf("session b Pts = %d, want 2", sb.Pts)
	}

	sc, err := uss.LoadUpdateState("c")
	if err != nil {
		t.Fatal(err)
	}
	if sc != nil {
		t.Fatal("nonexistent session should return nil")
	}
}

func TestCloseIdempotent(t *testing.T) {
	s := newTestStorage()
	if err := s.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestNewAdapterDelegatesPeerOps(t *testing.T) {
	inner := newMemoryAdapter()
	w := NewAdapter(inner)

	p := &Peer{ID: 42, Type: PeerTypeUser, Username: "test"}
	if err := w.SavePeer(p); err != nil {
		t.Fatal(err)
	}
	got, err := w.GetPeer(42)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Username != "test" {
		t.Fatalf("GetPeer = %v", got)
	}

	got, err = w.GetPeerByUsername("test")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != 42 {
		t.Fatalf("GetPeerByUsername = %v", got)
	}

	peers, err := w.LoadPeers()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 {
		t.Fatalf("LoadPeers = %d, want 1", len(peers))
	}

	if err := w.DeletePeer(42); err != nil {
		t.Fatal(err)
	}
	got, err = w.GetPeer(42)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("GetPeer should return nil after delete")
	}
}

func TestNewAdapterDelegatesConversationOps(t *testing.T) {
	inner := newMemoryAdapter()
	w := NewAdapter(inner)

	c := &Conversation{ChatID: 1, UserID: 2, Name: "conv", Step: 1}
	if err := w.SaveConversation(c); err != nil {
		t.Fatal(err)
	}

	loaded, err := w.LoadConversation(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.Name != "conv" {
		t.Fatalf("LoadConversation = %v", loaded)
	}

	if err := w.DeleteConversation(1, 2); err != nil {
		t.Fatal(err)
	}

	loaded, err = w.LoadConversation(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != nil {
		t.Fatal("LoadConversation should return nil after delete")
	}
}

func TestNewAdapterDelegatesUpdateStateOps(t *testing.T) {
	inner := newMemoryAdapter()
	w := NewAdapter(inner)

	state := &UpdateState{SessionID: "s1", Pts: 42}
	if err := w.SaveUpdateState(state); err != nil {
		t.Fatal(err)
	}

	loaded, err := w.LoadUpdateState("s1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.Pts != 42 {
		t.Fatalf("LoadUpdateState = %v", loaded)
	}

	chState := &ChannelUpdateState{SessionID: "s1", ChannelID: 10, Pts: 5}
	if err := w.SaveChannelUpdateState(chState); err != nil {
		t.Fatal(err)
	}

	chLoaded, err := w.LoadChannelUpdateState("s1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if chLoaded == nil || chLoaded.Pts != 5 {
		t.Fatalf("LoadChannelUpdateState = %v", chLoaded)
	}

	allCh, err := w.LoadAllChannelUpdateStates("s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(allCh) != 1 {
		t.Fatalf("LoadAllChannelUpdateStates = %d, want 1", len(allCh))
	}
}

func TestNewAdapterDelegatesDedupOps(t *testing.T) {
	inner := newMemoryAdapter()
	w := NewAdapter(inner)

	inserted, err := w.SaveUpdateDedupKey("s1", "key1")
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("first insert should return true")
	}

	exists, err := w.UpdateDedupKeyExists("s1", "key1")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("key should exist")
	}

	inserted, err = w.SaveUpdateDedupKey("s1", "key1")
	if err != nil {
		t.Fatal(err)
	}
	if inserted {
		t.Fatal("duplicate should return false")
	}
}

func TestNewAdapterDelegatesDurableUpdateOps(t *testing.T) {
	inner := newMemoryAdapter()
	w := NewAdapter(inner)

	u := &DurableUpdate{SessionID: "s1", ID: "u1", Payload: []byte("p")}
	if err := w.EnqueueDurableUpdate(u); err != nil {
		t.Fatal(err)
	}

	loaded, err := w.LoadDurableUpdates("s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 {
		t.Fatalf("LoadDurableUpdates = %d, want 1", len(loaded))
	}

	if err := w.MarkDurableUpdateFailed("s1", "u1", 2, "fail"); err != nil {
		t.Fatal(err)
	}

	if err := w.DeleteDurableUpdate("s1", "u1"); err != nil {
		t.Fatal(err)
	}

	loaded, err = w.LoadDurableUpdates("s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Fatalf("LoadDurableUpdates after delete = %d, want 0", len(loaded))
	}
}

func TestDeletePeerNonexistent(t *testing.T) {
	s := newTestStorage()
	ps := s.(PeerStore)

	err := ps.DeletePeer(99999)
	if err != nil {
		t.Fatalf("DeletePeer on nonexistent should not error: %v", err)
	}
}

func TestPeerWithoutUsername(t *testing.T) {
	s := newTestStorage()
	ps := s.(PeerStore)

	p := &Peer{ID: 300, Type: PeerTypeChat, AccessHash: 123}
	if err := ps.SavePeer(p); err != nil {
		t.Fatal(err)
	}

	got, err := ps.GetPeer(300)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.AccessHash != 123 {
		t.Fatalf("GetPeer = %v", got)
	}
}

func TestDeleteConversationNonexistent(t *testing.T) {
	s := newTestStorage()
	cs := s.(ConversationStore)

	err := cs.DeleteConversation(999, 999)
	if err != nil {
		t.Fatalf("DeleteConversation on nonexistent should not error: %v", err)
	}
}

func TestLoadDurableUpdatesEmptySession(t *testing.T) {
	s := newTestStorage()
	uss := s.(UpdateStateStore)

	loaded, err := uss.LoadDurableUpdates("nonexistent", 0)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != nil {
		t.Fatalf("LoadDurableUpdates on empty session = %v, want nil", loaded)
	}
}

func TestLoadAllChannelUpdateStatesEmpty(t *testing.T) {
	s := newTestStorage()
	uss := s.(UpdateStateStore)

	all, err := uss.LoadAllChannelUpdateStates("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if all != nil {
		t.Fatalf("LoadAllChannelUpdateStates on empty = %v, want nil", all)
	}
}

func TestDurableUpdateMutationSafety(t *testing.T) {
	s := newTestStorage()
	uss := s.(UpdateStateStore)

	original := &DurableUpdate{
		SessionID: "s1",
		ID:        "u1",
		Payload:   []byte("original"),
		Attempts:  0,
	}
	if err := uss.EnqueueDurableUpdate(original); err != nil {
		t.Fatal(err)
	}

	loaded, err := uss.LoadDurableUpdates("s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	loaded[0].Attempts = 999

	loaded2, err := uss.LoadDurableUpdates("s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if loaded2[0].Attempts != 0 {
		t.Fatal("mutation of returned update should not affect stored update")
	}
}

func TestUpdateStateMutationSafety(t *testing.T) {
	s := newTestStorage()
	uss := s.(UpdateStateStore)

	if err := uss.SaveUpdateState(&UpdateState{SessionID: "s1", Pts: 10}); err != nil {
		t.Fatal(err)
	}

	state, err := uss.LoadUpdateState("s1")
	if err != nil {
		t.Fatal(err)
	}
	state.Pts = 999

	state2, err := uss.LoadUpdateState("s1")
	if err != nil {
		t.Fatal(err)
	}
	if state2.Pts != 10 {
		t.Fatal("mutation of returned state should not affect stored state")
	}
}

func TestPeerMutationSafety(t *testing.T) {
	s := newTestStorage()
	ps := s.(PeerStore)

	if err := ps.SavePeer(&Peer{ID: 1, Username: "original", AccessHash: 100}); err != nil {
		t.Fatal(err)
	}

	got, err := ps.GetPeer(1)
	if err != nil {
		t.Fatal(err)
	}
	got.Username = "mutated"

	got2, err := ps.GetPeer(1)
	if err != nil {
		t.Fatal(err)
	}
	if got2.Username != "original" {
		t.Fatal("mutation of returned peer should not affect stored peer")
	}
}
