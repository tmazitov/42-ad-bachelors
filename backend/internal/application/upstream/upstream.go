// Package upstream holds errors shared by the use cases that call the 42 API.
package upstream

import (
	"context"
	"errors"
)

// ErrBusy means the 42 API budget is used up for now (rate limit or 429):
// answering would mean waiting too long, so the call failed fast instead.
var ErrBusy = errors.New("42 API is busy, try again in a minute")

type backgroundKey struct{}

// Background marks ctx as background work (profile refreshes, lookups, the
// sync). The 42 client keeps part of its hourly budget for sign-ins, so
// background calls fail with ErrBusy before the budget is fully used.
func Background(ctx context.Context) context.Context {
	return context.WithValue(ctx, backgroundKey{}, true)
}

// IsBackground reports whether ctx was marked with Background.
func IsBackground(ctx context.Context) bool {
	v, _ := ctx.Value(backgroundKey{}).(bool)
	return v
}
