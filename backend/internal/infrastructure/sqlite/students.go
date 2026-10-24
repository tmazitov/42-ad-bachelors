package sqlite

import (
	"context"
	"database/sql"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

// StudentRepository implements student.Repository.
type StudentRepository struct {
	db *sql.DB
}

func NewStudentRepository(db *sql.DB) *StudentRepository {
	return &StudentRepository{db: db}
}

var _ student.Repository = (*StudentRepository)(nil)

func (r *StudentRepository) Save(ctx context.Context, s student.Student) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO students (id, login, display_name, image_url)
		VALUES (?, ?, ?, NULLIF(?, ''))
		ON CONFLICT (id) DO UPDATE SET
			login        = excluded.login,
			display_name = excluded.display_name,
			image_url    = excluded.image_url,
			updated_at   = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`,
		s.ID, s.Login, s.DisplayName, s.ImageURL)
	return err
}
