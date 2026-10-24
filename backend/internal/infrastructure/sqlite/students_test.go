package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	return db
}

var registered = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

func save(t *testing.T, repo *StudentRepository, students ...student.Student) {
	t.Helper()
	for _, s := range students {
		if err := repo.Save(context.Background(), s); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestSaveUpsertsAndKeepsRegistration(t *testing.T) {
	ctx := context.Background()
	repo := NewStudentRepository(openTestDB(t))

	save(t, repo,
		student.Student{ID: 1, Login: "jdoe", DisplayName: "John", RegisteredAt: registered},
		// A later profile refresh without RegisteredAt must not unregister.
		student.Student{ID: 1, Login: "jdoe", DisplayName: "John Doe", ImageURL: "http://img", Alumni: true},
	)

	s, err := repo.FindByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if s.DisplayName != "John Doe" || s.ImageURL != "http://img" || !s.Alumni || s.Staff {
		t.Errorf("profile = %+v", s)
	}
	if !s.RegisteredAt.Equal(registered) {
		t.Errorf("registered at %v, want %v", s.RegisteredAt, registered)
	}
}

func TestReplaceProjectsAndFind(t *testing.T) {
	ctx := context.Background()
	repo := NewStudentRepository(openTestDB(t))
	save(t, repo, student.Student{ID: 1, Login: "alice", DisplayName: "Alice", RegisteredAt: registered})

	marked := time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)
	synced := time.Date(2026, 10, 24, 12, 0, 0, 123456789, time.UTC)
	if err := repo.ReplaceProjects(ctx, 1, []student.CompletedProject{
		{ProjectID: 1314, Slug: "42cursus-libft", FinalMark: 125, MarkedAt: marked},
		{ProjectID: 1327, FinalMark: 100},
	}, synced.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	// A second sync replaces the first one.
	if err := repo.ReplaceProjects(ctx, 1, []student.CompletedProject{
		{ProjectID: 1314, Slug: "42cursus-libft", FinalMark: 125, MarkedAt: marked},
	}, synced); err != nil {
		t.Fatal(err)
	}

	s, err := repo.FindByLogin(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if !s.ProjectsSyncedAt.Equal(synced) {
		t.Errorf("synced at %v, want %v", s.ProjectsSyncedAt, synced)
	}
	if p := s.Projects; len(p) != 1 || p[0].ProjectID != 1314 || p[0].Slug != "42cursus-libft" ||
		p[0].FinalMark != 125 || !p[0].MarkedAt.Equal(marked) {
		t.Errorf("projects = %+v", p)
	}

	if _, err := repo.FindByLogin(ctx, "nobody"); !errors.Is(err, student.ErrNotFound) {
		t.Errorf("FindByLogin(nobody) err = %v, want ErrNotFound", err)
	}
	if err := repo.ReplaceProjects(ctx, 99, nil, synced); !errors.Is(err, student.ErrNotFound) {
		t.Errorf("ReplaceProjects(99) err = %v, want ErrNotFound", err)
	}
}

func TestListsOnlyIncludeRegistered(t *testing.T) {
	ctx := context.Background()
	repo := NewStudentRepository(openTestDB(t))
	save(t, repo,
		student.Student{ID: 1, Login: "alice", DisplayName: "Alice", RegisteredAt: registered},
		student.Student{ID: 2, Login: "bob", DisplayName: "Bob", RegisteredAt: registered},
		student.Student{ID: 3, Login: "viewer", DisplayName: "Only Viewed"},
	)
	now := time.Now()
	for _, id := range []student.ID{1, 3} {
		if err := repo.ReplaceProjects(ctx, id, []student.CompletedProject{{ProjectID: 1314}}, now); err != nil {
			t.Fatal(err)
		}
	}

	all, err := repo.ListRegistered(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].Login != "alice" || all[1].Login != "bob" {
		t.Errorf("ListRegistered = %+v", all)
	}
	if all[0].ProjectsSyncedAt.IsZero() || !all[1].ProjectsSyncedAt.IsZero() {
		t.Errorf("synced times = %v, %v", all[0].ProjectsSyncedAt, all[1].ProjectsSyncedAt)
	}

	synced, err := repo.ListSynced(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(synced) != 1 || synced[0].Login != "alice" || len(synced[0].Projects) != 1 {
		t.Errorf("ListSynced = %+v", synced)
	}
}

func TestSearch(t *testing.T) {
	ctx := context.Background()
	repo := NewStudentRepository(openTestDB(t))
	save(t, repo,
		student.Student{ID: 1, Login: "anna", DisplayName: "Anna Smith", RegisteredAt: registered},
		student.Student{ID: 2, Login: "bob", DisplayName: "Bob Annan", RegisteredAt: registered},
		student.Student{ID: 3, Login: "carl", DisplayName: "Carl", RegisteredAt: registered},
		student.Student{ID: 4, Login: "annie", DisplayName: "Viewed Only"},
		student.Student{ID: 5, Login: "x_y", DisplayName: "Under Score", RegisteredAt: registered},
	)

	tests := []struct {
		query  string
		limit  int
		offset int
		want   []string
		total  int
	}{
		{"ann", 10, 0, []string{"anna", "bob"}, 2}, // login or display name, case-insensitive
		{"ANN", 1, 1, []string{"bob"}, 2},          // paging
		{"_", 10, 0, []string{"x_y"}, 1},           // LIKE wildcards are literal
		{"zzz", 10, 0, nil, 0},
	}
	for _, tt := range tests {
		got, total, err := repo.Search(ctx, tt.query, tt.limit, tt.offset)
		if err != nil {
			t.Fatal(err)
		}
		var logins []string
		for _, s := range got {
			logins = append(logins, s.Login)
		}
		if total != tt.total || len(logins) != len(tt.want) || (len(logins) > 0 && logins[0] != tt.want[0]) ||
			(len(logins) > 1 && logins[1] != tt.want[1]) {
			t.Errorf("Search(%q, %d, %d) = %v (total %d), want %v (total %d)",
				tt.query, tt.limit, tt.offset, logins, total, tt.want, tt.total)
		}
	}
}
