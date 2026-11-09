// Package auth implements sign-in with 42 OAuth. The backend exchanges the
// code, registers the student and opens a session; the browser only ever
// holds the session token (in an httpOnly cookie), never a 42 token.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/session"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

var (
	// ErrInvalidGrant means the authorization code was rejected by 42
	// (invalid, expired or already used).
	ErrInvalidGrant = errors.New("invalid grant")
	// ErrUnauthenticated means the session token is missing, unknown or expired.
	ErrUnauthenticated = errors.New("not signed in")
)

// loginTimeout bounds a sign-in: the code exchange, the profile and the first
// project sync all call 42.
const loginTimeout = 20 * time.Second

// Token is a 42 OAuth token pair.
type Token struct {
	AccessToken  string
	TokenType    string
	RefreshToken string
	Scope        string
	ExpiresIn    int   // seconds
	CreatedAt    int64 // unix seconds
}

// Provider is the 42 OAuth server.
type Provider interface {
	AuthorizeURL(state string) string
	ExchangeCode(ctx context.Context, code string) (Token, error)
	// Profile returns the user who owns accessToken.
	Profile(ctx context.Context, accessToken string) (student.Student, error)
}

// ProgressSyncer refreshes a student's stored projects.
type ProgressSyncer interface {
	SyncStudent(ctx context.Context, accessToken string, id student.ID) error
}

// Service runs the sign-in use cases.
type Service struct {
	provider   Provider
	students   student.Repository
	sessions   session.Repository
	progress   ProgressSyncer
	sessionTTL time.Duration
	log        *slog.Logger
	now        func() time.Time
}

func NewService(provider Provider, students student.Repository, sessions session.Repository,
	progress ProgressSyncer, sessionTTL time.Duration, log *slog.Logger) *Service {
	return &Service{
		provider:   provider,
		students:   students,
		sessions:   sessions,
		progress:   progress,
		sessionTTL: sessionTTL,
		log:        log,
		now:        time.Now,
	}
}

// AuthorizeURL returns the 42 sign-in page URL carrying state.
func (s *Service) AuthorizeURL(state string) string {
	return s.provider.AuthorizeURL(state)
}

// Login exchanges an authorization code, registers the student, syncs their
// projects and opens a session. It returns the session token and its expiry.
// A failed sync is logged but does not fail the login.
func (s *Service) Login(ctx context.Context, code string) (token string, expires time.Time, err error) {
	ctx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()

	t, err := s.provider.ExchangeCode(ctx, code)
	if err != nil {
		return "", time.Time{}, err
	}

	st, err := s.provider.Profile(ctx, t.AccessToken)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("fetch profile: %w", err)
	}
	now := s.now().UTC()
	st.RegisteredAt = now
	if err := s.students.Save(ctx, st); err != nil {
		return "", time.Time{}, fmt.Errorf("save student: %w", err)
	}

	if err := s.progress.SyncStudent(ctx, t.AccessToken, st.ID); err != nil {
		s.log.WarnContext(ctx, "progress sync on login failed", "login", st.Login, "err", err)
	}

	token, sess, err := session.New(st.ID, s.sessionTTL, now)
	if err != nil {
		return "", time.Time{}, err
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return "", time.Time{}, fmt.Errorf("create session: %w", err)
	}
	return token, sess.ExpiresAt, nil
}

// Authenticate returns the student signed in with token.
func (s *Service) Authenticate(ctx context.Context, token string) (student.ID, error) {
	if token == "" {
		return 0, ErrUnauthenticated
	}
	sess, err := s.sessions.Find(ctx, session.HashToken(token))
	if errors.Is(err, session.ErrNotFound) {
		return 0, ErrUnauthenticated
	}
	if err != nil {
		return 0, err
	}
	if sess.Expired(s.now()) {
		_ = s.sessions.Delete(ctx, sess.TokenHash)
		return 0, ErrUnauthenticated
	}
	return sess.StudentID, nil
}

// Logout ends the session for token. Unknown tokens are ignored.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.sessions.Delete(ctx, session.HashToken(token))
}

// DeleteExpiredSessions removes expired sessions and returns how many.
func (s *Service) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	return s.sessions.DeleteExpired(ctx, s.now())
}
