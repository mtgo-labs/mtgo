package telegram

import (
	"bytes"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/mtgo-labs/mtgo/internal/session"
)

type flakyAuthKeyStorage struct {
	*MemoryStorage
	err    error
	reads  atomic.Int32
	writes atomic.Int32
}

func (s *flakyAuthKeyStorage) AuthKey() ([]byte, error) {
	if s.reads.Add(1) > 1 {
		return nil, s.err
	}
	return s.MemoryStorage.AuthKey()
}

func (s *flakyAuthKeyStorage) SetAuthKey(key []byte) error {
	s.writes.Add(1)
	return s.MemoryStorage.SetAuthKey(key)
}

type authKeyReadErrorStorage struct {
	*MemoryStorage
	err error
}

func (s *authKeyReadErrorStorage) AuthKey() ([]byte, error) {
	return nil, s.err
}

type dcIDReadErrorStorage struct {
	*MemoryStorage
	err error
}

func (s *dcIDReadErrorStorage) DCID() (int, error) {
	return 0, s.err
}

type failingAuthTransport struct {
	err error
}

func (t *failingAuthTransport) Send(*bytes.Buffer) error { return t.err }
func (*failingAuthTransport) Recv() ([]byte, error)      { return nil, errors.New("unexpected receive") }

func TestPerformDHExchangeKeepsAuthKeyLoadedBySession(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 256)
	base := NewMemoryStorage()
	if err := base.SetAuthKey(key); err != nil {
		t.Fatalf("SetAuthKey: %v", err)
	}
	readErr := errors.New("second auth key read failed")
	st := &flakyAuthKeyStorage{MemoryStorage: base, err: readErr}
	dc := session.DataCenter{ID: 2}
	sess, err := session.NewSession(dc, st, "test", "test", "en", "en")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	c, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	tp := newSessionTransport(&failingAuthTransport{err: errors.New("unexpected DH exchange")}, nil)

	if err := c.performDHExchange(sess, st, dc, tp, false); err != nil {
		t.Fatalf("performDHExchange: %v", err)
	}
	if got := st.reads.Load(); got != 1 {
		t.Fatalf("AuthKey reads = %d, want 1", got)
	}
	if got := st.writes.Load(); got != 0 {
		t.Fatalf("SetAuthKey calls = %d, want 0", got)
	}
	if got := sess.AuthKey(); !bytes.Equal(got, key) {
		t.Fatal("session auth key changed")
	}
	stored, err := base.AuthKey()
	if err != nil {
		t.Fatalf("stored AuthKey: %v", err)
	}
	if !bytes.Equal(stored, key) {
		t.Fatal("stored auth key changed")
	}
}

func TestPerformDHExchangePropagatesAuthKeyReadError(t *testing.T) {
	base := NewMemoryStorage()
	dc := session.DataCenter{ID: 2}
	sess, err := session.NewSession(dc, base, "test", "test", "en", "en")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	readErr := errors.New("auth key read failed")
	st := &authKeyReadErrorStorage{MemoryStorage: base, err: readErr}
	c, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	tp := newSessionTransport(&failingAuthTransport{err: errors.New("unexpected DH exchange")}, nil)

	err = c.performDHExchange(sess, st, dc, tp, false)
	if !errors.Is(err, readErr) {
		t.Fatalf("performDHExchange error = %v, want %v", err, readErr)
	}
}

func TestInitSessionRejectsDCIDReadErrorWithStoredAuthKey(t *testing.T) {
	base := NewMemoryStorage()
	if err := base.SetAuthKey(bytes.Repeat([]byte{0x24}, 256)); err != nil {
		t.Fatalf("SetAuthKey: %v", err)
	}
	readErr := errors.New("dc_id read failed")
	st := &dcIDReadErrorStorage{MemoryStorage: base, err: readErr}
	c, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.initSession(st, nil)
	if !errors.Is(err, readErr) {
		t.Fatalf("initSession error = %v, want %v", err, readErr)
	}
}

func TestInitSessionAllowsDCIDReadErrorWithoutStoredAuthKey(t *testing.T) {
	base := NewMemoryStorage()
	st := &dcIDReadErrorStorage{MemoryStorage: base, err: errors.New("dc_id read failed")}
	c, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	sess, err := c.initSession(st, nil)
	if err != nil {
		t.Fatalf("initSession: %v", err)
	}
	if got := sess.DC().ID; got != 2 {
		t.Fatalf("session DC = %d, want 2", got)
	}
}

func TestInitSessionRejectsConfiguredDCMismatchWithStoredAuthKey(t *testing.T) {
	st := NewMemoryStorage()
	if err := st.SetDCID(4); err != nil {
		t.Fatalf("SetDCID: %v", err)
	}
	if err := st.SetAuthKey(bytes.Repeat([]byte{0x81}, 256)); err != nil {
		t.Fatalf("SetAuthKey: %v", err)
	}
	c, err := NewClient(1, "hash", &Config{DC: 3})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.initSession(st, nil); err == nil {
		t.Fatal("initSession succeeded with a stored DC4 key configured for DC3")
	}
	if got, err := st.DCID(); err != nil || got != 4 {
		t.Fatalf("stored DC after rejection = %d, %v; want 4, nil", got, err)
	}

	c.migratingDC.Store(true)
	sess, err := c.initSession(st, nil)
	if err != nil {
		t.Fatalf("initSession during migration: %v", err)
	}
	if got := sess.DC().ID; got != 3 {
		t.Fatalf("migration session DC = %d, want 3", got)
	}
}

func TestReconnectRejectsDCIDReadErrorWithStoredAuthKey(t *testing.T) {
	key := bytes.Repeat([]byte{0x35}, 256)
	base := NewMemoryStorage()
	if err := base.SetDCID(4); err != nil {
		t.Fatalf("SetDCID: %v", err)
	}
	if err := base.SetAuthKey(key); err != nil {
		t.Fatalf("SetAuthKey: %v", err)
	}
	readErr := errors.New("reconnect dc_id read failed")
	st := &dcIDReadErrorStorage{MemoryStorage: base, err: readErr}
	c, err := NewClient(1, "hash", &Config{ReconnectEnabled: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.storage = st
	if err := c.state.SetConnecting(4); err != nil {
		t.Fatalf("SetConnecting: %v", err)
	}
	c.state.SetConnected()
	c.state.SetReconnecting(errors.New("test disconnect"))
	release := make(chan struct{})
	close(release)
	dialer := &gatedFailDialer{entered: make(chan struct{}, 1), release: release}
	c.setTestDialer(dialer)

	err = c.reconnectOnce()
	if !errors.Is(err, readErr) {
		t.Fatalf("reconnectOnce error = %v, want %v", err, readErr)
	}
	if got := dialer.calls.Load(); got != 0 {
		t.Fatalf("dial calls = %d, want 0", got)
	}
	stored, err := base.AuthKey()
	if err != nil {
		t.Fatalf("AuthKey: %v", err)
	}
	if !bytes.Equal(stored, key) {
		t.Fatal("reconnect changed the stored auth key after dc_id read failure")
	}
}
