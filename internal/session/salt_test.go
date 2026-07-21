package session

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

func fakeNow(t time.Time) nowFunc {
	return func() time.Time { return t }
}

func TestSaltManagerStoreSimple(t *testing.T) {
	now := time.Unix(1000, 0)
	m := newSaltManager(fakeNow(now))

	m.StoreSimple(0xAAA)

	if m.Load() != 0xAAA {
		t.Fatalf("Load() = %x, want %x", m.Load(), 0xAAA)
	}
	if m.IsExpired() {
		t.Fatal("IsExpired() = true, want false")
	}
	if !m.HasValidSalt() {
		t.Fatal("HasValidSalt() = false, want true")
	}
}

func TestSaltManagerStoreFromFutureSalts_SelectsCurrent(t *testing.T) {
	now := time.Unix(1000, 0)
	m := newSaltManager(fakeNow(now))

	entries := []saltEntry{
		{validSince: 990, validUntil: 1010, salt: 0xAAA},
		{validSince: 1010, validUntil: 1030, salt: 0xBBB},
		{validSince: 1030, validUntil: 1050, salt: 0xCCC},
	}
	m.StoreFromFutureSalts(entries)

	if m.Load() != 0xAAA {
		t.Fatalf("Load() = %x, want 0xAAA (currently valid)", m.Load())
	}
	if m.IsExpired() {
		t.Fatal("IsExpired() = true, want false")
	}
}

func TestSaltManagerExpiredSaltNotSelectedAfterPromotion(t *testing.T) {
	baseTime := time.Unix(1000, 0)
	now := fakeNow(baseTime)
	m := newSaltManager(now)

	entries := []saltEntry{
		{validSince: 990, validUntil: 1010, salt: 0xAAA},
		{validSince: 1010, validUntil: 1030, salt: 0xBBB},
	}
	m.StoreFromFutureSalts(entries)

	if m.Load() != 0xAAA {
		t.Fatalf("Load() = %x, want 0xAAA", m.Load())
	}

	now = fakeNow(baseTime.Add(15 * time.Second))
	m.now = now

	if m.IsExpired() {
		t.Fatal("IsExpired() = true after promotion to BBB")
	}

	salt := m.Load()
	if salt != 0xBBB {
		t.Fatalf("Load() = %x, want 0xBBB (auto-promoted)", salt)
	}
}

func TestSaltManagerNextRefreshAt75Percent(t *testing.T) {
	baseTime := time.Unix(1000, 0)
	now := fakeNow(baseTime)
	m := newSaltManager(now)
	m.refreshRatio = 0.75
	m.refreshMin = 0

	entries := []saltEntry{
		{validSince: 1000, validUntil: 1100, salt: 0xAAA},
	}
	m.StoreFromFutureSalts(entries)

	refreshIn := m.NextRefreshIn()
	expected := 75 * time.Second
	if refreshIn != expected {
		t.Fatalf("NextRefreshIn() = %v, want %v", refreshIn, expected)
	}

	now = fakeNow(baseTime.Add(50 * time.Second))
	m.now = now
	refreshIn = m.NextRefreshIn()
	expected = 25 * time.Second
	if refreshIn != expected {
		t.Fatalf("NextRefreshIn() = %v, want %v", refreshIn, expected)
	}

	now = fakeNow(baseTime.Add(75 * time.Second))
	m.now = now
	refreshIn = m.NextRefreshIn()
	if refreshIn != 0 {
		t.Fatalf("NextRefreshIn() = %v, want 0 (time to refresh)", refreshIn)
	}
}

func TestSaltManagerNextRefreshMinLead(t *testing.T) {
	baseTime := time.Unix(1000, 0)
	now := fakeNow(baseTime)
	m := newSaltManager(now)
	m.refreshRatio = 0.75
	m.refreshMin = 30 * time.Second

	entries := []saltEntry{
		{validSince: 1000, validUntil: 1020, salt: 0xAAA},
	}
	m.StoreFromFutureSalts(entries)

	refreshIn := m.NextRefreshIn()
	if refreshIn != 30*time.Second {
		t.Fatalf("NextRefreshIn() = %v, want 30s (min lead time)", refreshIn)
	}
}

func TestSaltManagerSaltCloseToExpiryTriggersRefreshSoon(t *testing.T) {
	baseTime := time.Unix(1000, 0)
	now := fakeNow(baseTime.Add(9 * time.Second))
	m := newSaltManager(now)
	m.refreshRatio = 0.75
	m.refreshMin = 5 * time.Second

	entries := []saltEntry{
		{validSince: 1000, validUntil: 1010, salt: 0xAAA},
	}
	m.StoreFromFutureSalts(entries)

	refreshIn := m.NextRefreshIn()
	if refreshIn > 5*time.Second {
		t.Fatalf("NextRefreshIn() = %v, should be <= 5s (near expiry)", refreshIn)
	}
}

func TestSaltManagerNoGoroutineLeak(t *testing.T) {
	m := newSaltManager(time.Now)
	m.StoreSimple(0xAAA)
	m.StoreFromFutureSalts([]saltEntry{
		{validSince: 1000, validUntil: 2000, salt: 0xBBB},
	})
	m.Load()
	m.IsExpired()
	m.HasValidSalt()
	m.NextRefreshIn()
}

func TestSaltManagerWaitForValidCancellation(t *testing.T) {
	for range 100 {
		m := newSaltManager(time.Now)
		m.mu.Lock()
		m.salt = 1
		m.validSince = time.Now().Add(-2 * time.Hour).Unix()
		m.validUntil = time.Now().Add(-time.Hour).Unix()
		m.mu.Unlock()

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan bool, 1)
		go func() { done <- m.WaitForValid(ctx) }()
		cancel()

		select {
		case valid := <-done:
			if valid {
				t.Fatal("WaitForValid returned true for an expired salt")
			}
		case <-time.After(time.Second):
			t.Fatal("WaitForValid missed context cancellation")
		}
	}
}

func TestSaltManagerWaitForValidWakesOnStore(t *testing.T) {
	m := newSaltManager(time.Now)
	m.mu.Lock()
	m.salt = 1
	m.validSince = time.Now().Add(-2 * time.Hour).Unix()
	m.validUntil = time.Now().Add(-time.Hour).Unix()
	m.mu.Unlock()

	done := make(chan bool, 1)
	go func() { done <- m.WaitForValid(context.Background()) }()
	m.StoreSimple(2)

	select {
	case valid := <-done:
		if !valid {
			t.Fatal("WaitForValid returned false after a valid salt was stored")
		}
	case <-time.After(time.Second):
		t.Fatal("WaitForValid did not wake after StoreSimple")
	}
}

func TestSaltManagerStoreFromFutureSaltsEmpty(t *testing.T) {
	m := newSaltManager(time.Now)
	m.StoreFromFutureSalts(nil)
	m.StoreFromFutureSalts([]saltEntry{})
	if m.Load() != 0 {
		t.Fatal("empty StoreFromFutureSalts should not set salt")
	}
}

func TestSaltManagerBadServerSaltOverrides(t *testing.T) {
	now := time.Unix(1000, 0)
	m := newSaltManager(fakeNow(now))

	m.StoreFromFutureSalts([]saltEntry{
		{validSince: 990, validUntil: 1010, salt: 0xAAA},
	})

	m.StoreSimple(0xFFF)

	if m.Load() != 0xFFF {
		t.Fatalf("Load() = %x, want 0xFFF after BadServerSalt", m.Load())
	}
}

func TestSaltManagerSetRefreshRatio(t *testing.T) {
	m := newSaltManager(time.Now)
	m.SetRefreshRatio(0.5)
	if m.refreshRatio != 0.5 {
		t.Fatal("refreshRatio not updated")
	}
	m.SetRefreshRatio(-1)
	if m.refreshRatio != defaultSaltRefreshRatio {
		t.Fatal("negative ratio should reset to default")
	}
	m.SetRefreshRatio(2.0)
	if m.refreshRatio != defaultSaltRefreshRatio {
		t.Fatal("ratio > 1 should reset to default")
	}
}

func TestSaltManagerSetRefreshMin(t *testing.T) {
	m := newSaltManager(time.Now)
	m.SetRefreshMin(10 * time.Second)
	if m.refreshMin != 10*time.Second {
		t.Fatal("refreshMin not updated")
	}
	m.SetRefreshMin(-1)
	if m.refreshMin != 0 {
		t.Fatal("negative refreshMin should be 0")
	}
}

func TestSaltManagerHasValidSaltZero(t *testing.T) {
	m := newSaltManager(time.Now)
	if m.HasValidSalt() {
		t.Fatal("empty manager should not have valid salt")
	}
}

func TestSaltManagerIsExpiredNoValidity(t *testing.T) {
	m := newSaltManager(time.Now)
	m.StoreSimple(0xAAA)
	if m.IsExpired() {
		t.Fatal("StoreSimple salt should not be expired immediately")
	}
}

func TestSaltEntriesFromFuture(t *testing.T) {
	futures := []*tg.FutureSalt{
		{ValidSince: 100, ValidUntil: 200, Salt: 0xAAA},
		{ValidSince: 200, ValidUntil: 300, Salt: 0xBBB},
	}
	entries := saltEntriesFromFuture(futures)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].validSince != 100 || entries[0].salt != 0xAAA {
		t.Fatalf("entry[0] = %+v, want since=100 salt=0xAAA", entries[0])
	}
	if entries[1].validSince != 200 || entries[1].salt != 0xBBB {
		t.Fatalf("entry[1] = %+v, want since=200 salt=0xBBB", entries[1])
	}
}

func TestSaltManagerMultipleSaltsPicksBestCurrent(t *testing.T) {
	now := time.Unix(1000, 0)
	m := newSaltManager(fakeNow(now))

	entries := []saltEntry{
		{validSince: 980, validUntil: 1005, salt: 0xAAA},
		{validSince: 990, validUntil: 1020, salt: 0xBBB},
		{validSince: 1010, validUntil: 1040, salt: 0xCCC},
	}
	m.StoreFromFutureSalts(entries)

	if m.Load() != 0xBBB {
		t.Fatalf("Load() = %x, want 0xBBB (longest current validity)", m.Load())
	}
}

func TestSaltManagerPromotesFromPendingWhenExpired(t *testing.T) {
	baseTime := time.Unix(1000, 0)
	m := newSaltManager(fakeNow(baseTime))

	entries := []saltEntry{
		{validSince: 990, validUntil: 1005, salt: 0xAAA},
		{validSince: 1005, validUntil: 1020, salt: 0xBBB},
		{validSince: 1020, validUntil: 1040, salt: 0xCCC},
	}
	m.StoreFromFutureSalts(entries)

	if m.Load() != 0xAAA {
		t.Fatalf("initial Load() = %x, want 0xAAA", m.Load())
	}

	m.now = fakeNow(baseTime.Add(6 * time.Second))

	salt := m.Load()
	if salt != 0xBBB {
		t.Fatalf("promoted Load() = %x, want 0xBBB", salt)
	}

	m.now = fakeNow(baseTime.Add(21 * time.Second))

	salt = m.Load()
	if salt != 0xCCC {
		t.Fatalf("second promotion Load() = %x, want 0xCCC", salt)
	}
}

func TestSaltManagerFutureSaltsResponseUpdatesBothCurrentAndPending(t *testing.T) {
	now := time.Unix(1000, 0)
	m := newSaltManager(fakeNow(now))

	m.StoreSimple(0x111)

	entries := []saltEntry{
		{validSince: 990, validUntil: 1010, salt: 0xAAA},
		{validSince: 1010, validUntil: 1030, salt: 0xBBB},
	}
	m.StoreFromFutureSalts(entries)

	if m.Load() != 0xAAA {
		t.Fatalf("Load() = %x, want 0xAAA", m.Load())
	}

	m.now = fakeNow(time.Unix(1015, 0))
	if m.IsExpired() {
		t.Fatal("should not be expired, BBB is valid")
	}
	if m.Load() != 0xBBB {
		t.Fatalf("promoted Load() = %x, want 0xBBB", m.Load())
	}
}

func TestSessionSaltLoopTriggersPreFetch(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.saltMgr = newSaltManager(func() time.Time {
		return time.Unix(1000, 0)
	})
	s.saltMgr.StoreFromFutureSalts([]saltEntry{
		{validSince: 990, validUntil: 1030, salt: 0xAAA},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.ackCh = make(chan int64, 1024)
	s.pingCbs = make(map[int64]chan struct{})
	s.done = make(chan struct{})
	s.sm.forceSetState(StateActive)

	go func() { _ = s.saltLoop(ctx) }()

	select {
	case <-mt.sendCh:
	case <-time.After(20 * time.Second):
		t.Fatal("saltLoop did not send GetFutureSaltsRequest after initial wait")
	}

	cancel()
	close(s.done)
}

func TestSessionBadServerSaltStillWorks(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	pingID := time.Now().UnixNano()
	invokeDone := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	go func() {
		obj, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: pingID}, 2, 5*time.Second)
		invokeDone <- struct {
			obj tg.TLObject
			err error
		}{obj: obj, err: err}
	}()

	firstSent := <-mt.sendCh
	firstMsg := unpackIncoming(firstSent, s)
	if firstMsg == nil {
		t.Fatal("first sent message did not decode")
	}

	newSalt := int64(0xDEADBEEF)
	mt.recvCh <- makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.BadServerSalt{
		BadMsgID:      firstMsg.MsgID,
		BadMsgSeqno:   int32(firstMsg.SeqNo),
		ErrorCode:     48,
		NewServerSalt: newSalt,
	})

	secondSent := <-mt.sendCh
	secondMsg := unpackIncoming(secondSent, s)
	if secondMsg == nil {
		t.Fatal("second sent message did not decode")
	}
	if secondMsg.MsgID == firstMsg.MsgID {
		t.Fatal("retry reused msg_id")
	}
	if s.saltMgr.Load() != newSalt {
		t.Fatalf("salt = %x, want %x", s.saltMgr.Load(), newSalt)
	}

	mt.recvCh <- makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.Pong{
		MsgID:  secondMsg.MsgID,
		PingID: pingID,
	})

	select {
	case got := <-invokeDone:
		if got.err != nil {
			t.Fatalf("Invoke() error: %v", got.err)
		}
		if _, ok := got.obj.(*tg.Pong); !ok {
			t.Fatalf("Invoke() = %T, want *tg.Pong", got.obj)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Invoke() timed out")
	}
}

func TestSessionStoreFutureSaltsWithValidity(t *testing.T) {
	s := newSessionWithAuthKey(t)

	now := time.Unix(1000, 0)
	s.saltMgr = newSaltManager(func() time.Time { return now })

	fs := &tg.FutureSalts{
		Now: 1000,
		Salts: []*tg.FutureSalt{
			{ValidSince: 990, ValidUntil: 1010, Salt: 0xAAA},
			{ValidSince: 1010, ValidUntil: 1030, Salt: 0xBBB},
		},
	}
	s.storeFutureSalts(fs)

	if s.saltMgr.Load() != 0xAAA {
		t.Fatalf("salt = %x, want 0xAAA", s.saltMgr.Load())
	}

	now = time.Unix(1015, 0)
	if s.saltMgr.Load() != 0xBBB {
		t.Fatalf("salt after promotion = %x, want 0xBBB", s.saltMgr.Load())
	}
}

func TestSessionHandleRawFutureSaltsWithValidity(t *testing.T) {
	s := newSessionWithAuthKey(t)

	now := time.Unix(1000, 0)
	s.saltMgr = newSaltManager(func() time.Time { return now })

	var body []byte
	var buf bytes.Buffer
	tg.WriteInt(&buf, tg.FutureSaltsTypeID)
	tg.WriteLong(&buf, 0)
	tg.WriteInt(&buf, 1000)
	tg.WriteInt(&buf, tg.VectorTypeID)
	tg.WriteInt(&buf, 2)
	encodeFutureSaltBuf(&buf, 990, 1010, 0xAAA)
	encodeFutureSaltBuf(&buf, 1010, 1030, 0xBBB)
	body = buf.Bytes()

	result := s.handleRawFutureSalts(body)
	if !result {
		t.Fatal("handleRawFutureSalts returned false")
	}
	if s.saltMgr.Load() != 0xAAA {
		t.Fatalf("salt = %x, want 0xAAA", s.saltMgr.Load())
	}
}

func encodeFutureSaltBuf(b *bytes.Buffer, validSince, validUntil int32, salt int64) {
	tg.WriteInt(b, tg.FutureSaltTypeID)
	tg.WriteInt(b, uint32(validSince))
	tg.WriteInt(b, uint32(validUntil))
	tg.WriteLong(b, salt)
}
