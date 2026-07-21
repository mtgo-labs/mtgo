package telegram

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestOverloadController_DisabledAlwaysAdmits(t *testing.T) {
	oc := NewOverloadController(OverloadConfig{
		Ceilings: ResourceCeilings{MaxInFlightRPCs: 0},
	})
	if oc.Enabled() {
		t.Fatal("MaxInFlightRPCs=0 should be disabled")
	}
	release, err := oc.Admit(context.Background(), overloadPriorityHigh)
	if err != nil {
		t.Fatalf("disabled Admit: %v", err)
	}
	release()
}

func TestOverload_LowPriorityFastFail(t *testing.T) {
	oc := NewOverloadController(OverloadConfig{
		Ceilings:          ResourceCeilings{MaxInFlightRPCs: 2},
		AdmissionDeadline: 100 * time.Millisecond,
	})
	// Fill both slots with high-priority (which blocks).
	r1, _ := oc.Admit(context.Background(), overloadPriorityHigh)
	r2, _ := oc.Admit(context.Background(), overloadPriorityHigh)

	// Third call with Low priority should fast-fail immediately.
	start := time.Now()
	_, err := oc.Admit(context.Background(), overloadPriorityLow)
	elapsed := time.Since(start)
	if !errors.Is(err, ErrOverload) {
		t.Fatalf("expected ErrOverload, got %v", err)
	}
	if elapsed > 50*time.Millisecond {
		t.Fatalf("Low fast-fail took %v, expected immediate", elapsed)
	}

	r1()
	r2()
}

func TestInvokeWithRawResultHonorsOverloadControl(t *testing.T) {
	c, err := NewClient(12345, "hash", &Config{
		InMemory:        true,
		MaxInFlightRPCs: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	c.state.SetConnecting(2)
	c.state.SetConnected()

	release, err := c.overloadController.Admit(context.Background(), overloadPriorityHigh)
	if err != nil {
		t.Fatalf("occupy overload slot: %v", err)
	}
	defer release()

	_, err = c.InvokeWithRawResult(context.Background(), &tg.UploadSaveFilePartRequest{})
	if !errors.Is(err, ErrOverload) {
		t.Fatalf("InvokeWithRawResult error = %v, want ErrOverload", err)
	}
}

func TestOverload_HighPriorityDeferredAdmitted(t *testing.T) {
	oc := NewOverloadController(OverloadConfig{
		Ceilings:          ResourceCeilings{MaxInFlightRPCs: 1},
		AdmissionDeadline: 500 * time.Millisecond,
	})
	// Fill the single slot.
	r1, _ := oc.Admit(context.Background(), overloadPriorityHigh)

	// High-priority should wait, then be admitted when slot frees.
	done := make(chan error, 1)
	go func() {
		_, err := oc.Admit(context.Background(), overloadPriorityHigh)
		done <- err
	}()

	time.Sleep(50 * time.Millisecond) // let the goroutine start waiting
	r1()                              // free the slot

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("deferred admit should succeed, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("deferred admit timed out")
	}
}

func TestOverload_HighPriorityDeadlineExceeded(t *testing.T) {
	oc := NewOverloadController(OverloadConfig{
		Ceilings:          ResourceCeilings{MaxInFlightRPCs: 1},
		AdmissionDeadline: 50 * time.Millisecond,
	})
	// Fill the single slot and never release.
	r1, _ := oc.Admit(context.Background(), overloadPriorityHigh)
	defer r1()

	start := time.Now()
	_, err := oc.Admit(context.Background(), overloadPriorityHigh)
	elapsed := time.Since(start)

	if !errors.Is(err, ErrOverload) {
		t.Fatalf("expected ErrOverload, got %v", err)
	}
	if elapsed < 40*time.Millisecond {
		t.Fatalf("deadline too short: %v (expected ~50ms)", elapsed)
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("deadline too long: %v", elapsed)
	}

	// Verify the error has the right reason.
	var oe *OverloadError
	if errors.As(err, &oe) {
		if oe.Info.Reason != "admission_deadline_exceeded" {
			t.Errorf("Reason = %q, want 'admission_deadline_exceeded'", oe.Info.Reason)
		}
	}
}

func TestOverload_ReleaseRestoresSlot(t *testing.T) {
	oc := NewOverloadController(OverloadConfig{
		Ceilings:          ResourceCeilings{MaxInFlightRPCs: 1},
		AdmissionDeadline: 100 * time.Millisecond,
	})
	r1, err := oc.Admit(context.Background(), overloadPriorityHigh)
	if err != nil {
		t.Fatalf("first admit: %v", err)
	}
	r1()

	// Slot should be available again.
	r2, err := oc.Admit(context.Background(), overloadPriorityLow)
	if err != nil {
		t.Fatalf("second admit after release: %v", err)
	}
	r2()

	snap := oc.Snapshot()
	if snap.InFlight != 0 {
		t.Errorf("InFlight = %d after all released, want 0", snap.InFlight)
	}
}

func TestOverload_DegradationLadder(t *testing.T) {
	oc := NewOverloadController(OverloadConfig{
		Ceilings:          ResourceCeilings{MaxInFlightRPCs: 4},
		AdmissionDeadline: 50 * time.Millisecond,
	})
	if tl := oc.ThrottleLevel(); tl != ThrottleNormal {
		t.Fatalf("empty: ThrottleLevel = %d, want Normal", tl)
	}

	// Fill 3 of 4 slots (75% = threshold for Shedding).
	r1, _ := oc.Admit(context.Background(), overloadPriorityHigh)
	r2, _ := oc.Admit(context.Background(), overloadPriorityHigh)
	r3, _ := oc.Admit(context.Background(), overloadPriorityHigh)

	if tl := oc.ThrottleLevel(); tl < ThrottleShedding {
		t.Errorf("75%% full: ThrottleLevel = %d, want >= Shedding", tl)
	}

	// Fill the last slot → at capacity → ThrottlingLow.
	r4, _ := oc.Admit(context.Background(), overloadPriorityHigh)
	if tl := oc.ThrottleLevel(); tl != ThrottleThrottlingLow {
		t.Errorf("100%% full: ThrottleLevel = %d, want ThrottlingLow", tl)
	}

	r1()
	r2()
	r3()
	r4()
}

func TestOverload_SnapshotReflectsState(t *testing.T) {
	oc := NewOverloadController(OverloadConfig{
		Ceilings:          ResourceCeilings{MaxInFlightRPCs: 5},
		AdmissionDeadline: 100 * time.Millisecond,
	})
	r1, _ := oc.Admit(context.Background(), overloadPriorityHigh)
	r2, _ := oc.Admit(context.Background(), overloadPriorityHigh)

	snap := oc.Snapshot()
	if snap.InFlight != 2 {
		t.Errorf("InFlight = %d, want 2", snap.InFlight)
	}
	if snap.TotalAdmitted != 2 {
		t.Errorf("TotalAdmitted = %d, want 2", snap.TotalAdmitted)
	}

	r1()
	r2()

	snap = oc.Snapshot()
	if snap.InFlight != 0 {
		t.Errorf("InFlight after release = %d, want 0", snap.InFlight)
	}
}

func TestOverload_ConcurrentAdmitRelease(t *testing.T) {
	oc := NewOverloadController(OverloadConfig{
		Ceilings:          ResourceCeilings{MaxInFlightRPCs: 3},
		AdmissionDeadline: 200 * time.Millisecond,
	})
	var wg sync.WaitGroup
	const goroutines = 20
	const perG = 10
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				release, err := oc.Admit(context.Background(), overloadPriorityHigh)
				if err == nil {
					time.Sleep(time.Millisecond)
					release()
				}
			}
		}()
	}
	wg.Wait()

	snap := oc.Snapshot()
	if snap.InFlight != 0 {
		t.Errorf("InFlight after all done = %d, want 0", snap.InFlight)
	}
	if snap.TotalAdmitted == 0 {
		t.Error("TotalAdmitted = 0, expected some admits")
	}
}
