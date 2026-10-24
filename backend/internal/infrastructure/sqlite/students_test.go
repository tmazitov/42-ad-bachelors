package sqlite

import (
	"context"
	"database/sql"
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

func TestStudentSaveUpserts(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	repo := NewStudentRepository(db)

	if err := repo.Save(ctx, student.Student{ID: 1, Login: "jdoe", DisplayName: "John"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, student.Student{ID: 1, Login: "jdoe", DisplayName: "John Doe", ImageURL: "http://img"}); err != nil {
		t.Fatal(err)
	}

	var count int
	var name string
	var image sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*), MAX(display_name), MAX(image_url) FROM students`).Scan(&count, &name, &image); err != nil {
		t.Fatal(err)
	}
	if count != 1 || name != "John Doe" || image.String != "http://img" {
		t.Errorf("count=%d name=%q image=%q", count, name, image.String)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestReplaceProjectsAndListSynced(t *testing.T) {
	ctx := context.Background()
	repo := NewStudentRepository(openTestDB(t))

	for _, s := range []student.Student{{ID: 1, Login: "alice", DisplayName: "Alice"}, {ID: 2, Login: "bob", DisplayName: "Bob"}} {
		if err := repo.Save(ctx, s); err != nil {
			t.Fatal(err)
		}
	}

	// bob is registered but never synced: not listed.
	list, err := repo.ListSynced(ctx)
	if err != nil || len(list) != 0 {
		t.Fatalf("ListSynced before sync = %+v, %v", list, err)
	}

	marked := time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)
	synced := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	if err := repo.ReplaceProjects(ctx, 1, []student.CompletedProject{
		{ProjectID: 1314, FinalMark: 125, MarkedAt: marked},
		{ProjectID: 1327, FinalMark: 100},
	}, synced.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	// A second sync replaces the first one.
	if err := repo.ReplaceProjects(ctx, 1, []student.CompletedProject{
		{ProjectID: 1314, FinalMark: 125, MarkedAt: marked},
	}, synced); err != nil {
		t.Fatal(err)
	}

	list, err = repo.ListSynced(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Login != "alice" || !list[0].ProjectsSyncedAt.Equal(synced) {
		t.Fatalf("list = %+v", list)
	}
	if p := list[0].Projects; len(p) != 1 || p[0].ProjectID != 1314 || p[0].FinalMark != 125 || !p[0].MarkedAt.Equal(marked) {
		t.Errorf("projects = %+v", p)
	}
}

func TestReplaceProjectsUnknownStudent(t *testing.T) {
	repo := NewStudentRepository(openTestDB(t))
	if err := repo.ReplaceProjects(context.Background(), 99, nil, time.Now()); err == nil {
		t.Fatal("want error for unknown student")
	}
}

func TestListReturnsAllRegistered(t *testing.T) {
	ctx := context.Background()
	repo := NewStudentRepository(openTestDB(t))
	for _, s := range []student.Student{{ID: 2, Login: "bob", DisplayName: "Bob"}, {ID: 1, Login: "alice", DisplayName: "Alice", ImageURL: "http://img"}} {
		if err := repo.Save(ctx, s); err != nil {
			t.Fatal(err)
		}
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Login != "alice" || list[0].ImageURL != "http://img" || list[1].Login != "bob" {
		t.Errorf("list = %+v", list)
	}
}
