package tgerr

import "testing"

func TestSecurityCheckMismatch(t *testing.T) {
	err := &SecurityCheckMismatch{Name: "test_check"}
	if err.Error() != "security check failed: test_check" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}
