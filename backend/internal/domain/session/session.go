// Package session holds sign-in sessions. The browser keeps a random token in
// an httpOnly cookie; only its SHA-256 hash is stored.
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

// ErrNotFound means no session matches the token.
var ErrNotFound = errors.New("session not found")

// Session links a token to a signed-in student.
type Session struct {
	TokenHash string
	StudentID student.ID
	ExpiresAt time.Time
}

// New creates a session for id lasting ttl and returns it with its token.
func New(id student.ID, ttl time.Duration, now time.Time) (token string, s Session, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", Session{}, err
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, Session{TokenHash: HashToken(token), StudentID: id, ExpiresAt: now.Add(ttl)}, nil
}

// HashToken returns the stored form of a session token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Expired reports whether the session is no longer valid at now.
func (s Session) Expired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}

// Repository persists sessions.
type Repository interface {
	Create(ctx context.Context, s Session) error
	// Find returns the session with tokenHash, or ErrNotFound.
	Find(ctx context.Context, tokenHash string) (Session, error)
	Delete(ctx context.Context, tokenHash string) error
	// DeleteExpired removes sessions expired at now and returns how many.
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}
