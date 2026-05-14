package tgerr

import "testing"

func TestSecurityCheckMismatch(t *testing.T) {
	err := &SecurityCheckMismatch{Name: "test_check"}
	if err.Error() != "security check failed: test_check" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestCheck(t *testing.T) {
	Check(true, "should_pass")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		serr, ok := r.(*SecurityCheckMismatch)
		if !ok {
			t.Fatalf("expected *SecurityCheckMismatch, got %T", r)
		}
		if serr.Name != "failed_check" {
			t.Fatalf("unexpected name: %s", serr.Name)
		}
	}()
	Check(false, "failed_check")
}
