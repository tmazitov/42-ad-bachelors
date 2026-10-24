package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

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
