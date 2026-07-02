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

// ErrorCode returns 0; security check mismatches have no numeric code.
func (e *SecurityCheckMismatch) ErrorCode() int { return 0 }

// ErrorMessage returns the name of the failed security check.
func (e *SecurityCheckMismatch) ErrorMessage() string {
	return "security check failed: " + e.Name
}

// ErrorType returns "SECURITY_CHECK_MISMATCH".
func (e *SecurityCheckMismatch) ErrorType() string { return "SECURITY_CHECK_MISMATCH" }

// ErrorArg returns 0; security check mismatches have no numeric argument.
func (e *SecurityCheckMismatch) ErrorArg() int { return 0 }
