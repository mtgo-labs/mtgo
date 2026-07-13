package generic

import "errors"

// ErrNoMessageUpdates is returned by [AsMessage] when the [tg.UpdatesClass]
// response contains no message-bearing update.
var ErrNoMessageUpdates = errors.New("no message updates in response")
