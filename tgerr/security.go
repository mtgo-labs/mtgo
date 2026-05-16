package tgerr

import "fmt"

// SecurityCheckMismatch is an error returned when a client-side security check
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
