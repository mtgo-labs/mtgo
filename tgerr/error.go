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

func (e *Error) extractArgument() {
	if e.Message == "" {
		return
	}
	e.Type = e.Message
	parts := strings.Split(e.Message, "_")
	if len(parts) < 2 {
		return
	}
	var nonDigit []string
Parts:
	for _, part := range parts {
		for _, r := range part {
			if unicode.IsDigit(r) {
				continue
			}
			nonDigit = append(nonDigit, part)
			continue Parts
		}
		argument, err := strconv.Atoi(part)
		if err != nil {
			return
		}
		e.Argument = argument
	}
	e.Type = strings.Join(nonDigit, "_")
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
