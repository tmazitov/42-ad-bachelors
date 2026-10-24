package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

// Timestamps are stored as RFC 3339 UTC text.
const timeLayout = time.RFC3339Nano

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

func (r *StudentRepository) ReplaceProjects(ctx context.Context, id student.ID, projects []student.CompletedProject, syncedAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `UPDATE students SET projects_synced_at = ? WHERE id = ?`,
		syncedAt.UTC().Format(timeLayout), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("student %d not found", id)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM student_projects WHERE student_id = ?`, id); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO student_projects (student_id, project_id, final_mark, marked_at)
		VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range projects {
		if _, err := stmt.ExecContext(ctx, id, p.ProjectID, p.FinalMark, formatNullTime(p.MarkedAt)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *StudentRepository) List(ctx context.Context) ([]student.Student, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, login, display_name, COALESCE(image_url, '')
		FROM students
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var students []student.Student
	for rows.Next() {
		var s student.Student
		if err := rows.Scan(&s.ID, &s.Login, &s.DisplayName, &s.ImageURL); err != nil {
			return nil, err
		}
		students = append(students, s)
	}
	return students, rows.Err()
}

func (r *StudentRepository) ListSynced(ctx context.Context) ([]student.Student, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, login, display_name, COALESCE(image_url, ''), projects_synced_at
		FROM students
		WHERE projects_synced_at IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var students []student.Student
	index := map[student.ID]int{}
	for rows.Next() {
		var s student.Student
		var syncedAt string
		if err := rows.Scan(&s.ID, &s.Login, &s.DisplayName, &s.ImageURL, &syncedAt); err != nil {
			return nil, err
		}
		if s.ProjectsSyncedAt, err = time.Parse(timeLayout, syncedAt); err != nil {
			return nil, fmt.Errorf("student %d projects_synced_at: %w", s.ID, err)
		}
		index[s.ID] = len(students)
		students = append(students, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	prows, err := r.db.QueryContext(ctx, `SELECT student_id, project_id, final_mark, marked_at FROM student_projects`)
	if err != nil {
		return nil, err
	}
	defer prows.Close()

	for prows.Next() {
		var sid student.ID
		var p student.CompletedProject
		var markedAt sql.NullString
		if err := prows.Scan(&sid, &p.ProjectID, &p.FinalMark, &markedAt); err != nil {
			return nil, err
		}
		i, ok := index[sid]
		if !ok {
			continue
		}
		if markedAt.Valid {
			if p.MarkedAt, err = time.Parse(timeLayout, markedAt.String); err != nil {
				return nil, fmt.Errorf("student %d project %d marked_at: %w", sid, p.ProjectID, err)
			}
		}
		students[i].Projects = append(students[i].Projects, p)
	}
	return students, prows.Err()
}

func formatNullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(timeLayout)
}
