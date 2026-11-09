// Package upstream holds errors shared by the use cases that call the 42 API.
package upstream

import "errors"

// ErrBusy means the 42 API budget is used up for now (rate limit or 429):
// answering would mean waiting too long, so the call failed fast instead.
var ErrBusy = errors.New("42 API is busy, try again in a minute")
