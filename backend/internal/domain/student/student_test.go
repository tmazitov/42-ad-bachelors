package student

import (
	"testing"
	"time"
)

func TestNeedsSync(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	tests := []struct {
		name     string
		syncedAt time.Time
		maxAge   time.Duration
		want     bool
	}{
		{"never synced", time.Time{}, day, true},
		{"synced an hour ago", now.Add(-time.Hour), day, false},
		{"synced exactly a day ago", now.Add(-day), day, true},
		{"synced two days ago", now.Add(-2 * day), day, true},
		{"zero max age syncs everyone", now, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Student{ProjectsSyncedAt: tt.syncedAt}
			if got := s.NeedsSync(now, tt.maxAge); got != tt.want {
				t.Errorf("NeedsSync = %v, want %v", got, tt.want)
			}
		})
	}
}
