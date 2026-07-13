package session

import (
	"testing"
	"time"
)

// --- Container tracker ---

func BenchmarkContainerTrackerTrackContainer(b *testing.B) {
	ct := NewContainerTracker()
	childIDs := []int64{1, 2, 3, 4, 5}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ct.TrackContainer(int64(i), childIDs)
	}
}

func BenchmarkContainerTrackerAckChild(b *testing.B) {
	ct := NewContainerTracker()
	childIDs := []int64{1, 2, 3, 4, 5}
	ct.TrackContainer(100, childIDs)
	b.ReportAllocs()
	for b.Loop() {
		ct.AckChild(1)
	}
}

// --- Pending manager ---

func BenchmarkPendingRegisterResolve(b *testing.B) {
	pm := NewPendingManager()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h, _ := pm.Register(int64(i), false)
		pm.Resolve(int64(i), nil)
		_ = h
	}
}

func BenchmarkPendingRegisterCancel(b *testing.B) {
	pm := NewPendingManager()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pm.Register(int64(i), false)
		pm.Cancel(int64(i))
	}
}

// --- Salt manager ---

func BenchmarkSaltManagerLoad(b *testing.B) {
	sm := newSaltManager(time.Now)
	sm.StoreSimple(0xABCDEF)
	b.ReportAllocs()
	for b.Loop() {
		sm.Load()
	}
}

// --- Msg ID generator ---

func BenchmarkMsgFactoryAllocateMsgID(b *testing.B) {
	mf := NewMsgFactory(time.Now())
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mf.AllocateMsgID()
	}
}

func BenchmarkMsgFactoryAllocateSeqNo(b *testing.B) {
	mf := NewMsgFactory(time.Now())
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mf.AllocateSeqNo(true)
	}
}

// --- Msg ID validator ---

func BenchmarkMsgIDValidatorCheck(b *testing.B) {
	v := newMsgIDValidator(func() int64 { return 1700000000 })
	validID := int64(1700000000) << 32
	v.Check(validID)
	b.ReportAllocs()
	for b.Loop() {
		v.Check(validID)
	}
}

// --- DC option pool ---

func BenchmarkDCOptionPoolFindBest(b *testing.B) {
	pool := NewDCOptionPool(2, 5*time.Minute)
	pool.AddOption(DataCenter{ID: 2})
	pool.AddOption(DataCenter{ID: 2, IPv6: true})
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = pool.FindBest()
	}
}
