// Package auth implements sign-in with 42 OAuth. The backend keeps the client
// secret; the browser receives and stores the 42 tokens.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

// ErrInvalidGrant means the authorization code or refresh token was rejected
// by 42 (invalid, expired or already used).
var ErrInvalidGrant = errors.New("invalid grant")

// Token is a 42 OAuth token pair.
type Token struct {
	AccessToken  string
	TokenType    string
	RefreshToken string
	Scope        string
	ExpiresIn    int   // seconds
	CreatedAt    int64 // unix seconds
}

// Profile is the signed-in 42 user.
type Profile struct {
	ID          int
	Login       string
	DisplayName string
	ImageURL    string
}

// Provider is the 42 OAuth server.
type Provider interface {
	ExchangeCode(ctx context.Context, code string) (Token, error)
	RefreshToken(ctx context.Context, refreshToken string) (Token, error)
	Profile(ctx context.Context, accessToken string) (Profile, error)
}

// ProgressSyncer refreshes a student's stored projects.
type ProgressSyncer interface {
	SyncStudent(ctx context.Context, accessToken string, id student.ID) error
}

// Service runs the sign-in use cases.
type Service struct {
	provider Provider
	students student.Repository
	progress ProgressSyncer
	log      *slog.Logger
}

func NewService(provider Provider, students student.Repository, progress ProgressSyncer, log *slog.Logger) *Service {
	return &Service{provider: provider, students: students, progress: progress, log: log}
}

// Login exchanges an authorization code for tokens, registers the student and
// syncs their projects. A failed sync is logged but does not fail the login.
func (s *Service) Login(ctx context.Context, code string) (Token, error) {
	token, err := s.provider.ExchangeCode(ctx, code)
	if err != nil {
		return Token{}, err
	}

	profile, err := s.provider.Profile(ctx, token.AccessToken)
	if err != nil {
		return Token{}, fmt.Errorf("fetch profile: %w", err)
	}

	st, err := student.New(student.ID(profile.ID), profile.Login, profile.DisplayName, profile.ImageURL)
	if err != nil {
		return Token{}, err
	}
	if err := s.students.Save(ctx, st); err != nil {
		return Token{}, fmt.Errorf("save student: %w", err)
	}

	if err := s.progress.SyncStudent(ctx, token.AccessToken, st.ID); err != nil {
		s.log.WarnContext(ctx, "progress sync on login failed", "login", st.Login, "err", err)
	}
	return token, nil
}

// Refresh exchanges a refresh token for a new token pair.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Token, error) {
	return s.provider.RefreshToken(ctx, refreshToken)
}
