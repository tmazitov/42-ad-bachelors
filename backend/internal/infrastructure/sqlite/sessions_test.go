package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/session"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

func TestSessions(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	save(t, NewStudentRepository(db), student.Student{ID: 1, Login: "alice", DisplayName: "Alice", RegisteredAt: registered})
	repo := NewSessionRepository(db)

	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	live := session.Session{TokenHash: "live", StudentID: 1, ExpiresAt: now.Add(time.Hour)}
	old := session.Session{TokenHash: "old", StudentID: 1, ExpiresAt: now.Add(-time.Hour)}
	for _, s := range []session.Session{live, old} {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatal(err)
		}
	}

	got, err := repo.Find(ctx, "live")
	if err != nil {
		t.Fatal(err)
	}
	if got.StudentID != 1 || !got.ExpiresAt.Equal(live.ExpiresAt) {
		t.Errorf("Find = %+v", got)
	}

	if n, err := repo.DeleteExpired(ctx, now); err != nil || n != 1 {
		t.Errorf("DeleteExpired = %d, %v; want 1", n, err)
	}
	if _, err := repo.Find(ctx, "old"); !errors.Is(err, session.ErrNotFound) {
		t.Errorf("expired session still found: %v", err)
	}

	if err := repo.Delete(ctx, "live"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Find(ctx, "live"); !errors.Is(err, session.ErrNotFound) {
		t.Errorf("deleted session still found: %v", err)
	}
}
