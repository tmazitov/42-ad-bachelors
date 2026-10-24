package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestEveryRunsImmediatelyAndRepeats(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var runs atomic.Int32
	done := make(chan struct{})

	go func() {
		Every(ctx, 10*time.Millisecond, func(context.Context) {
			if runs.Add(1) == 3 {
				cancel()
			}
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Every did not return after cancel")
	}
	if n := runs.Load(); n != 3 {
		t.Errorf("runs = %d, want 3", n)
	}
}
