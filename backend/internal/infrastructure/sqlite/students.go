package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

const studentColumns = `id, login, display_name, COALESCE(image_url, ''), is_staff, is_alumni,
	COALESCE(stream, ''), registered_at, projects_synced_at`

func (r *StudentRepository) Save(ctx context.Context, s student.Student) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO students (id, login, display_name, image_url, is_staff, is_alumni, registered_at)
		VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			login         = excluded.login,
			display_name  = excluded.display_name,
			image_url     = excluded.image_url,
			is_staff      = excluded.is_staff,
			is_alumni     = excluded.is_alumni,
			registered_at = COALESCE(students.registered_at, excluded.registered_at),
			updated_at    = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`,
		s.ID, s.Login, s.DisplayName, s.ImageURL, s.Staff, s.Alumni, formatNullTime(s.RegisteredAt))
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
		return fmt.Errorf("student %d: %w", id, student.ErrNotFound)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM student_projects WHERE student_id = ?`, id); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO student_projects (student_id, project_id, slug, final_mark, marked_at)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range projects {
		if _, err := stmt.ExecContext(ctx, id, p.ProjectID, p.Slug, p.FinalMark, formatNullTime(p.MarkedAt)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *StudentRepository) SetStream(ctx context.Context, id student.ID, stream string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE students SET stream = NULLIF(?, ''), updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ?`, stream, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("student %d: %w", id, student.ErrNotFound)
	}
	return nil
}

func (r *StudentRepository) SetModuleMark(ctx context.Context, id student.ID, code string, completed bool) error {
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM students WHERE id = ?)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("student %d: %w", id, student.ErrNotFound)
	}
	var err error
	if completed {
		_, err = r.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO student_marked_modules (student_id, module_code) VALUES (?, ?)`, id, code)
	} else {
		_, err = r.db.ExecContext(ctx, `
			DELETE FROM student_marked_modules WHERE student_id = ? AND module_code = ?`, id, code)
	}
	return err
}

func (r *StudentRepository) FindByID(ctx context.Context, id student.ID) (student.Student, error) {
	return r.findOne(ctx, `WHERE id = ?`, id)
}

func (r *StudentRepository) FindByLogin(ctx context.Context, login string) (student.Student, error) {
	return r.findOne(ctx, `WHERE login = ?`, login)
}

func (r *StudentRepository) findOne(ctx context.Context, where string, arg any) (student.Student, error) {
	s, err := scanStudent(r.db.QueryRowContext(ctx, `SELECT `+studentColumns+` FROM students `+where, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return student.Student{}, student.ErrNotFound
	}
	if err != nil {
		return student.Student{}, err
	}

	byID, err := r.loadProjects(ctx, `WHERE student_id = ?`, s.ID)
	if err != nil {
		return student.Student{}, err
	}
	s.Projects = byID[s.ID]

	marks, err := r.loadMarks(ctx, `WHERE student_id = ?`, s.ID)
	if err != nil {
		return student.Student{}, err
	}
	s.MarkedModules = marks[s.ID]
	return s, nil
}

func (r *StudentRepository) Search(ctx context.Context, query string, limit, offset int) ([]student.Student, int, error) {
	pattern := "%" + escapeLike(query) + "%"
	const where = `WHERE registered_at IS NOT NULL
		AND (login LIKE ? ESCAPE '\' OR display_name LIKE ? ESCAPE '\')`

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM students `+where, pattern, pattern).Scan(&total); err != nil {
		return nil, 0, err
	}

	students, err := r.query(ctx, `SELECT `+studentColumns+` FROM students `+where+`
		ORDER BY login LIMIT ? OFFSET ?`, pattern, pattern, limit, offset)
	return students, total, err
}

func (r *StudentRepository) ListRegistered(ctx context.Context) ([]student.Student, error) {
	return r.query(ctx, `SELECT `+studentColumns+` FROM students
		WHERE registered_at IS NOT NULL ORDER BY id`)
}

func (r *StudentRepository) ListSynced(ctx context.Context) ([]student.Student, error) {
	students, err := r.query(ctx, `SELECT `+studentColumns+` FROM students
		WHERE registered_at IS NOT NULL AND projects_synced_at IS NOT NULL`)
	if err != nil {
		return nil, err
	}

	byID, err := r.loadProjects(ctx, ``)
	if err != nil {
		return nil, err
	}
	marks, err := r.loadMarks(ctx, ``)
	if err != nil {
		return nil, err
	}
	for i := range students {
		students[i].Projects = byID[students[i].ID]
		students[i].MarkedModules = marks[students[i].ID]
	}
	return students, nil
}

func (r *StudentRepository) loadMarks(ctx context.Context, where string, args ...any) (map[student.ID][]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT student_id, module_code FROM student_marked_modules `+where+` ORDER BY module_code`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := map[student.ID][]string{}
	for rows.Next() {
		var sid student.ID
		var code string
		if err := rows.Scan(&sid, &code); err != nil {
			return nil, err
		}
		byID[sid] = append(byID[sid], code)
	}
	return byID, rows.Err()
}

func (r *StudentRepository) query(ctx context.Context, q string, args ...any) ([]student.Student, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var students []student.Student
	for rows.Next() {
		s, err := scanStudent(rows)
		if err != nil {
			return nil, err
		}
		students = append(students, s)
	}
	return students, rows.Err()
}

func (r *StudentRepository) loadProjects(ctx context.Context, where string, args ...any) (map[student.ID][]student.CompletedProject, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT student_id, project_id, slug, final_mark, marked_at
		FROM student_projects `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := map[student.ID][]student.CompletedProject{}
	for rows.Next() {
		var sid student.ID
		var p student.CompletedProject
		var markedAt sql.NullString
		if err := rows.Scan(&sid, &p.ProjectID, &p.Slug, &p.FinalMark, &markedAt); err != nil {
			return nil, err
		}
		if p.MarkedAt, err = parseNullTime(markedAt); err != nil {
			return nil, fmt.Errorf("student %d project %d marked_at: %w", sid, p.ProjectID, err)
		}
		byID[sid] = append(byID[sid], p)
	}
	return byID, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanStudent(row scanner) (student.Student, error) {
	var s student.Student
	var registeredAt, syncedAt sql.NullString
	if err := row.Scan(&s.ID, &s.Login, &s.DisplayName, &s.ImageURL, &s.Staff, &s.Alumni, &s.Stream, &registeredAt, &syncedAt); err != nil {
		return student.Student{}, err
	}
	var err error
	if s.RegisteredAt, err = parseNullTime(registeredAt); err != nil {
		return student.Student{}, fmt.Errorf("student %d registered_at: %w", s.ID, err)
	}
	if s.ProjectsSyncedAt, err = parseNullTime(syncedAt); err != nil {
		return student.Student{}, fmt.Errorf("student %d projects_synced_at: %w", s.ID, err)
	}
	return s, nil
}

func formatNullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(timeLayout)
}

func parseNullTime(v sql.NullString) (time.Time, error) {
	if !v.Valid {
		return time.Time{}, nil
	}
	return time.Parse(timeLayout, v.String)
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
