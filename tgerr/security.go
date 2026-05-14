package tgerr

import "fmt"

// SecurityCheckMismatch is a panic value used when a client-side security check
// (such as hash or authorization validation) fails. It carries the Name of the
// failed check.
type SecurityCheckMismatch struct {
	// Name identifies which security check failed.
	Name string
}

// Error implements the error interface, returning a message that includes the
// name of the failed security check.
func (e *SecurityCheckMismatch) Error() string {
	return fmt.Sprintf("security check failed: %s", e.Name)
}

// Check asserts that ok is true. If ok is false, it panics with a
// *SecurityCheckMismatch wrapping name. Recover the panic to obtain the error.
func Check(ok bool, name string) {
	if !ok {
		panic(&SecurityCheckMismatch{Name: name})
	}
}
