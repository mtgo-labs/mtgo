package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

type mockPeerResolver struct {
	peers     map[int64]tg.InputPeerClass
	usernames map[string]tg.InputPeerClass
	err       error
}

func (m *mockPeerResolver) ResolvePeerCache(id int64) (tg.InputPeerClass, error) {
	if m.err != nil {
		return nil, m.err
	}
	if p, ok := m.peers[id]; ok {
		return p, nil
	}
	return nil, ErrPeerNotFound
}

func (m *mockPeerResolver) ResolveUsername(_ context.Context, username string) (tg.InputPeerClass, error) {
	if m.err != nil {
		return nil, m.err
	}
	if p, ok := m.usernames[username]; ok {
		return p, nil
	}
	return nil, ErrPeerNotFound
}

func (m *mockPeerResolver) ResolvePhone(_ context.Context, phone string) (tg.InputPeerClass, error) {
	return nil, ErrPeerNotFound
}

// testClient wraps a PeerResolver in a Client whose test override makes the
// resolution helpers use it with cache-only semantics.
func testClient(r PeerResolver) *Client {
	return &Client{testResolver: r}
}

func TestResolvePeer_Self(t *testing.T) {
	r := &mockPeerResolver{}
	peer, err := resolvePeer(t.Context(), testClient(r), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := peer.(*tg.InputPeerSelf); !ok {
		t.Fatalf("expected InputPeerSelf, got %T", peer)
	}
}

func TestResolvePeer_Cached(t *testing.T) {
	r := &mockPeerResolver{
		peers: map[int64]tg.InputPeerClass{
			123: &tg.InputPeerUser{UserID: 123, AccessHash: 456},
		},
	}
	peer, err := resolvePeer(t.Context(), testClient(r), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p, ok := peer.(*tg.InputPeerUser)
	if !ok {
		t.Fatalf("expected InputPeerUser, got %T", peer)
	}
	if p.UserID != 123 {
		t.Errorf("expected UserID 123, got %d", p.UserID)
	}
}

func TestResolvePeer_NotFound(t *testing.T) {
	r := &mockPeerResolver{}
	_, err := resolvePeer(t.Context(), testClient(r), 999)
	if err == nil {
		t.Fatal("expected error for unresolvable peer")
	}
}

func TestResolveUserID_Self(t *testing.T) {
	r := &mockPeerResolver{}
	user, err := resolveUserID(t.Context(), testClient(r), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := user.(*tg.InputUserSelf); !ok {
		t.Fatalf("expected InputUserSelf, got %T", user)
	}
}

func TestResolveUserID_FromPeer(t *testing.T) {
	r := &mockPeerResolver{
		peers: map[int64]tg.InputPeerClass{
			77: &tg.InputPeerUser{UserID: 77, AccessHash: 88},
		},
	}
	user, err := resolveUserID(t.Context(), testClient(r), 77)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	u, ok := user.(*tg.InputUser)
	if !ok {
		t.Fatalf("expected InputUserTL, got %T", user)
	}
	if u.UserID != 77 || u.AccessHash != 88 {
		t.Errorf("expected UserID=77 AccessHash=88, got %d/%d", u.UserID, u.AccessHash)
	}
}

func TestResolveUserID_NotUser(t *testing.T) {
	r := &mockPeerResolver{
		peers: map[int64]tg.InputPeerClass{
			100: &tg.InputPeerChat{ChatID: 100},
		},
	}
	_, err := resolveUserID(t.Context(), testClient(r), 100)
	if err == nil {
		t.Fatal("expected error when peer is not a user")
	}
}

func TestResolveChannelID(t *testing.T) {
	r := &mockPeerResolver{
		peers: map[int64]tg.InputPeerClass{
			5: &tg.InputPeerChannel{ChannelID: 5, AccessHash: 6},
		},
	}
	ch, err := resolveChannelID(t.Context(), testClient(r), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c, ok := ch.(*tg.InputChannel)
	if !ok {
		t.Fatalf("expected InputChannelTL, got %T", ch)
	}
	if c.ChannelID != 5 || c.AccessHash != 6 {
		t.Errorf("expected ChannelID=5 AccessHash=6, got %d/%d", c.ChannelID, c.AccessHash)
	}
}

func TestResolveChannelID_NotChannel(t *testing.T) {
	r := &mockPeerResolver{
		peers: map[int64]tg.InputPeerClass{
			100: &tg.InputPeerUser{UserID: 100, AccessHash: 200},
		},
	}
	_, err := resolveChannelID(t.Context(), testClient(r), 100)
	if err == nil {
		t.Fatal("expected error when peer is not a channel")
	}
}

func TestClientPeerResolver_Override(t *testing.T) {
	inner := &mockPeerResolver{
		peers: map[int64]tg.InputPeerClass{
			42: &tg.InputPeerUser{UserID: 42, AccessHash: 100},
		},
	}
	c := &Client{testResolver: inner}
	resolver := c.clientPeerResolver()
	peer, err := resolvePeer(t.Context(), testClient(resolver), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p, ok := peer.(*tg.InputPeerUser)
	if !ok {
		t.Fatalf("expected InputPeerUser, got %T", peer)
	}
	if p.UserID != 42 {
		t.Errorf("UserID = %d, want 42", p.UserID)
	}
}

func TestClientPeerResolver_FallbackToSelf(t *testing.T) {
	c := &Client{}
	c.peersManager().Cache(10, &tg.InputPeerChat{ChatID: 10})
	resolver := c.clientPeerResolver()
	peer, err := resolvePeer(t.Context(), testClient(resolver), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := peer.(*tg.InputPeerChat); !ok {
		t.Fatalf("expected InputPeerChat, got %T", peer)
	}
}

func TestChatRefFrom_NumericString(t *testing.T) {
	r := &mockPeerResolver{
		peers: map[int64]tg.InputPeerClass{
			42: &tg.InputPeerUser{UserID: 42, AccessHash: 99},
		},
	}
	peer, err := ChatRefFrom("42").resolve(context.Background(), r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p, ok := peer.(*tg.InputPeerUser)
	if !ok {
		t.Fatalf("expected InputPeerUser, got %T", peer)
	}
	if p.UserID != 42 {
		t.Errorf("UserID = %d, want 42", p.UserID)
	}
}

func TestChatRefFrom_Username(t *testing.T) {
	r := &mockPeerResolver{
		usernames: map[string]tg.InputPeerClass{
			"durov": &tg.InputPeerUser{UserID: 1, AccessHash: 2},
		},
	}
	peer, err := ChatRefFrom("durov").resolve(context.Background(), r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := peer.(*tg.InputPeerUser); !ok {
		t.Fatalf("expected InputPeerUser, got %T", peer)
	}
}

func TestChatRefFrom_AtUsername(t *testing.T) {
	r := &mockPeerResolver{
		usernames: map[string]tg.InputPeerClass{
			"test": &tg.InputPeerUser{UserID: 1, AccessHash: 2},
		},
	}
	peer, err := ChatRefFrom("@test").resolve(context.Background(), r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := peer.(*tg.InputPeerUser); !ok {
		t.Fatalf("expected InputPeerUser, got %T", peer)
	}
}

func TestChatRefFrom_URL(t *testing.T) {
	r := &mockPeerResolver{
		usernames: map[string]tg.InputPeerClass{
			"telegram": &tg.InputPeerChannel{ChannelID: 1, AccessHash: 2},
		},
	}
	for _, url := range []string{
		"https://t.me/telegram",
		"http://t.me/telegram",
		"t.me/telegram",
		"https://telegram.me/telegram",
		"telegram.dog/telegram",
	} {
		peer, err := ChatRefFrom(url).resolve(context.Background(), r)
		if err != nil {
			t.Fatalf("ChatRefFrom(%q): %v", url, err)
		}
		if _, ok := peer.(*tg.InputPeerChannel); !ok {
			t.Fatalf("ChatRefFrom(%q): expected InputPeerChannel, got %T", url, peer)
		}
	}
}

func TestChatRefFrom_Phone(t *testing.T) {
	ref := ChatRefFrom("+1234567890")
	if ref.phone == "" {
		t.Fatal("expected phone to be set for +prefixed string")
	}
}

func TestChatPhone(t *testing.T) {
	r := &mockPeerResolver{
		peers: map[int64]tg.InputPeerClass{},
	}
	_, err := ChatPhone("+1234567890").resolve(context.Background(), r)
	if err == nil {
		t.Fatal("expected error from mock without phone support")
	}
}

func TestParseLink(t *testing.T) {
	tests := []struct {
		input string
		want  link
	}{
		{"https://t.me/username", link{kind: linkUsername, username: "username"}},
		{"http://t.me/username", link{kind: linkUsername, username: "username"}},
		{"t.me/username", link{kind: linkUsername, username: "username"}},
		{"https://telegram.me/username", link{kind: linkUsername, username: "username"}},
		{"telegram.dog/username", link{kind: linkUsername, username: "username"}},
		{"https://t.me/username/something", link{kind: linkUsername, username: "username"}},
		{"https://t.me/username?param=1", link{kind: linkUsername, username: "username"}},
		{"https://t.me/@username", link{kind: linkUsername, username: "username"}},
		{"https://t.me/+AbCdEf123", link{kind: linkInviteHash, hash: "AbCdEf123"}},
		{"https://t.me/joinchat/AbCdEf123", link{kind: linkInviteHash, hash: "AbCdEf123"}},
		{"tg://join?invite=AbCdEf123", link{kind: linkInviteHash, hash: "AbCdEf123"}},
		{"tg://resolve?domain=username", link{kind: linkUsername, username: "username"}},
		{"https://t.me/c/123456/42", link{kind: linkID, id: 123456}},
		{"+1234567890", link{kind: linkPhone, phone: "1234567890"}},
		{"001234567890", link{kind: linkPhone, phone: "1234567890"}},
		{"-1000000001234", link{kind: linkID, id: -1000000001234}},
		{"@durov", link{kind: linkUsername, username: "durov"}},
		{"durov", link{kind: linkUsername, username: "durov"}},
		{"https://example.com/username", link{kind: linkInvalid}},
	}
	for _, tt := range tests {
		got := parseLink(tt.input)
		if got != tt.want {
			t.Errorf("parseLink(%q) = %+v, want %+v", tt.input, got, tt.want)
		}
	}
}

func TestResolvePeer_ByPhone(t *testing.T) {
	r := &mockPeerResolver{
		peers: map[int64]tg.InputPeerClass{},
	}
	_, err := ChatPhone("+1234567890").resolve(context.Background(), r)
	if err == nil {
		t.Fatal("expected error from mock without phone support")
	}
}

func TestUserRef_Phone(t *testing.T) {
	r := &mockPeerResolver{}
	_, err := UserPhone("+1234567890").resolve(context.Background(), r)
	if err == nil {
		t.Fatal("expected error from mock without phone support")
	}
}
