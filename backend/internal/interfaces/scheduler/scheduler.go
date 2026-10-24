// Package scheduler runs background jobs on a fixed interval. Like the HTTP
// layer, it only triggers application use cases.
package scheduler

import (
	"context"
	"time"
)

// Every runs job immediately and then every interval until ctx is cancelled.
// Runs never overlap: if a run takes longer than interval, the next one starts
// right after it. Every returns once the current run has finished.
func Every(ctx context.Context, interval time.Duration, job func(context.Context)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		job(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
