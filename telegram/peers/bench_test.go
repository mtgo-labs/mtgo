package peers

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func benchManager() *Manager {
	return NewManager(Deps{
		CacheSize: func() int { return 5000 },
		SavePeers: func() bool { return false },
	})
}

func BenchmarkCache(b *testing.B) {
	m := benchManager()
	peer := &tg.InputPeerUser{UserID: 1, AccessHash: 2}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Cache(int64(i%1000), peer)
	}
}

func BenchmarkCachedHit(b *testing.B) {
	m := benchManager()
	m.Cache(42, &tg.InputPeerUser{UserID: 42, AccessHash: 2})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := m.Cached(42); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCachedMarkedChannelID(b *testing.B) {
	m := benchManager()
	m.Cache(1234, &tg.InputPeerChannel{ChannelID: 1234, AccessHash: 2})
	marked := int64(-1000000001234)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := m.Cached(marked); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheUsername(b *testing.B) {
	m := benchManager()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.CacheUsername("user", 42)
	}
}

func BenchmarkIngestUsers(b *testing.B) {
	m := benchManager()
	users := make([]tg.UserClass, 32)
	for i := range users {
		users[i] = &tg.User{ID: int64(i), AccessHash: 7, Username: "u", Phone: "7900"}
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.IngestUsers(users)
	}
}
