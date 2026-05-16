package session

import "errors"

// Session decoder errors.
//
// These errors are returned when decoding (importing) session strings from
// other Telegram clients.
var (
	// ErrEmptySession is returned when attempting to decode an empty session
	// string.
	ErrEmptySession = errors.New("empty session string")
	// ErrUnknownFormat is returned when the session string does not match any
	// known format (Telethon, GramJS, mtCute, TData, etc.).
	ErrUnknownFormat = errors.New("unable to detect session format")
)
