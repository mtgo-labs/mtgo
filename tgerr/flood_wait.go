package tgerr

import (
	"context"
	"time"
)

// FloodWaitErrors lists the error type strings that indicate a flood wait
// condition (ErrFloodWait and ErrFloodPremiumWait).
var FloodWaitErrors = []string{ErrFloodWait, ErrFloodPremiumWait}

// AsFloodWait checks whether err wraps a flood-wait Error. If so, it returns
// the required wait duration derived from the error's Argument field and true.
// Otherwise it returns zero and false.
func AsFloodWait(err error) (d time.Duration, ok bool) {
	for _, e := range FloodWaitErrors {
		if rpcErr, ok := AsType(err, e); ok {
			return time.Second * time.Duration(rpcErr.Argument), true
		}
	}
	return 0, false
}

// FloodWait blocks until the flood wait duration specified by err has elapsed
// (plus a 1-second buffer), respecting cancellation via ctx. It returns true and
// the original error after waiting, or false and the original error if err is
// not a flood wait error. If ctx is cancelled while waiting, it returns false
// and ctx.Err().
func FloodWait(ctx context.Context, err error) (bool, error) {
	if d, ok := AsFloodWait(err); ok {
		timer := time.NewTimer(d + 1*time.Second)
		defer timer.Stop()
		select {
		case <-timer.C:
			return true, err
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
	return false, err
}
