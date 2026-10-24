package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/session"
)

// SessionRepository implements session.Repository.
type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

var _ session.Repository = (*SessionRepository)(nil)

func (r *SessionRepository) Create(ctx context.Context, s session.Session) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, student_id, expires_at) VALUES (?, ?, ?)`,
		s.TokenHash, s.StudentID, s.ExpiresAt.Unix())
	return err
}

func (r *SessionRepository) Find(ctx context.Context, tokenHash string) (session.Session, error) {
	s := session.Session{TokenHash: tokenHash}
	var expires int64
	err := r.db.QueryRowContext(ctx, `SELECT student_id, expires_at FROM sessions WHERE token_hash = ?`, tokenHash).
		Scan(&s.StudentID, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return session.Session{}, session.ErrNotFound
	}
	if err != nil {
		return session.Session{}, err
	}
	s.ExpiresAt = time.Unix(expires, 0).UTC()
	return s, nil
}

func (r *SessionRepository) Delete(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (r *SessionRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, now.Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
