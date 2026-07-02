package tgerr

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Error represents a Telegram RPC error returned by the API, including the
// numeric error code, the original message string, the extracted error type,
// and an optional numeric argument parsed from the message.
type Error struct {
	// Code is the numeric RPC error code (e.g. 400, 401, 420, 500).
	Code int
	// Message is the original error string returned by Telegram (e.g. "FLOOD_WAIT_60").
	Message string
	// Type is the error type extracted from Message with numeric suffixes removed (e.g. "FLOOD_WAIT").
	Type string
	// Argument is the numeric argument parsed from the error message, such as the
	// wait duration in seconds for FLOOD_WAIT errors.
	Argument int
}

// ErrorInfo is implemented by all mtgo error types that carry structured
// diagnostic metadata. It provides a uniform way to extract code, message,
// type, and argument regardless of the concrete error type.
//
// Methods use the Error prefix to avoid clashing with struct field names
// (e.g. Error.Code is a field; ErrorCode() is the interface method).
type ErrorInfo interface {
	error

	// ErrorCode returns the error's numeric code. For RPC errors this is
	// the Telegram error code (e.g. 400, 420, 500); for transport errors
	// it is the negative transport code (e.g. -404). Returns 0 for errors
	// that have no numeric code.
	ErrorCode() int

	// ErrorMessage returns a human-readable description of the error.
	ErrorMessage() string

	// ErrorType returns the error's type name (e.g. "FLOOD_WAIT",
	// "TRANSPORT", "MIGRATION"). Returns "" for errors without a type.
	ErrorType() string

	// ErrorArg returns the error's numeric argument (e.g. the wait time in
	// seconds for FLOOD_WAIT_60). Returns 0 if the error has no argument.
	ErrorArg() int
}

// ErrorCode returns the numeric RPC error code.
func (e *Error) ErrorCode() int { return e.Code }

// ErrorMessage returns the original error string from Telegram.
func (e *Error) ErrorMessage() string { return e.Message }

// ErrorType returns the parsed error type (message with numeric suffix removed).
func (e *Error) ErrorType() string { return e.Type }

// ErrorArg returns the numeric argument parsed from the error message.
func (e *Error) ErrorArg() int { return e.Argument }

// New creates a new Error with the given RPC error code and message string.
// It extracts the error type and numeric argument from msg automatically.
func New(code int, msg string) *Error {
	e := &Error{
		Code:    code,
		Message: msg,
	}
	e.extractArgument()
	return e
}

// Error implements the error interface. It returns a human-readable string
// representation of the RPC error including the code, type, and argument.
func (e *Error) Error() string {
	if e.Type != e.Message {
		return fmt.Sprintf("rpc error code %d: %s (%d)", e.Code, e.Type, e.Argument)
	}
	return fmt.Sprintf("rpc error code %d: %s", e.Code, e.Message)
}

// IsType reports whether the error's Type matches t. It is safe to call on a
// nil Error.
func (e *Error) IsType(t string) bool {
	if e == nil {
		return false
	}
	return e.Type == t
}

// IsCode reports whether the error's Code matches code. It is safe to call on a
// nil Error.
func (e *Error) IsCode(code int) bool {
	if e == nil {
		return false
	}
	return e.Code == code
}

// IsOneOf reports whether the error's Type matches any of the provided error
// type strings. It is safe to call on a nil Error.
func (e *Error) IsOneOf(tt ...string) bool {
	if e == nil {
		return false
	}
	for _, t := range tt {
		if e.IsType(t) {
			return true
		}
	}
	return false
}

// IsCodeOneOf reports whether the error's Code matches any of the provided
// numeric codes. It is safe to call on a nil Error.
func (e *Error) IsCodeOneOf(codes ...int) bool {
	if e == nil {
		return false
	}
	for _, code := range codes {
		if e.IsCode(code) {
			return true
		}
	}
	return false
}

// IsTransient reports whether the error represents a temporary server-side
// condition that may resolve on retry (codes 420 and 500). Returns false
// for permanent errors such as bad request (400), unauthorized (401),
// forbidden (403), or migration redirects (303). It is safe to call on a
// nil Error.
func (e *Error) IsTransient() bool {
	if e == nil {
		return false
	}
	return e.Code == 420 || e.Code == 500
}

func (e *Error) extractArgument() {
	if e.Message == "" {
		return
	}
	e.Type = e.Message
	parts := strings.Split(e.Message, "_")
	if len(parts) < 2 {
		return
	}
	var typeParts []string
	for _, part := range parts {
		isDigit := true
		for _, r := range part {
			if !unicode.IsDigit(r) {
				isDigit = false
				break
			}
		}
		if isDigit {
			argument, err := strconv.Atoi(part)
			if err != nil {
				return
			}
			e.Argument = argument
		} else {
			typeParts = append(typeParts, part)
		}
	}
	e.Type = strings.Join(typeParts, "_")
}

// AsType uses errors.As to extract an *Error from err and reports whether its
// Type matches t. It returns the matched Error and true on success, or nil and
// false otherwise.
func AsType(err error, t string) (rpcErr *Error, ok bool) {
	if errors.As(err, &rpcErr) && rpcErr.Type == t {
		return rpcErr, true
	}
	return nil, false
}

// As uses errors.As to extract an *Error from err. It returns the matched Error
// and true on success, or nil and false otherwise.
func As(err error) (rpcErr *Error, ok bool) {
	if errors.As(err, &rpcErr) {
		return rpcErr, true
	}
	return nil, false
}

// Is reports whether err wraps an *Error whose Type matches any of the provided
// error type strings.
func Is(err error, tt ...string) bool {
	if rpcErr, ok := As(err); ok {
		return rpcErr.IsOneOf(tt...)
	}
	return false
}

// IsCode reports whether err wraps an *Error whose Code matches any of the
// provided numeric codes.
func IsCode(err error, code ...int) bool {
	if rpcErr, ok := As(err); ok {
		return rpcErr.IsCodeOneOf(code...)
	}
	return false
}

// IsTransient reports whether err wraps an *Error with a transient error code
// (420 or 500), indicating the operation may succeed on retry.
func IsTransient(err error) bool {
	if rpcErr, ok := As(err); ok {
		return rpcErr.IsTransient()
	}
	return false
}

// IsRetryable reports whether err wraps any error type that represents a
// transient condition worth retrying. Unlike IsTransient (which only checks
// RPC codes 420/500), this covers all error types in the mtgo ecosystem:
//
//   - RPC flood wait (code 420) and internal server errors (code 500)
//   - Reconnection failures
//
// Migration errors (code 303) are excluded because they require DC switching,
// not a blind retry.
func IsRetryable(err error) bool {
	var info ErrorInfo
	if !errors.As(err, &info) {
		return false
	}
	code := info.ErrorCode()
	if code == 420 || code == 500 {
		return true
	}
	switch info.ErrorType() {
	case "RECONNECT":
		return true
	}
	return false
}
