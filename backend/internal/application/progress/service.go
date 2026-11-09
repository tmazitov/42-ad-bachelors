// Package progress syncs students' 42 projects and serves the leaderboard.
package progress

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/upstream"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/leaderboard"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

// ProjectSource fetches a student's validated projects from 42.
type ProjectSource interface {
	CompletedProjects(ctx context.Context, accessToken string, id student.ID) ([]student.CompletedProject, error)
}

// AppTokenSource provides a 42 application token for server-side syncs.
type AppTokenSource interface {
	AppToken(ctx context.Context) (string, error)
}

// Service runs the progress use cases.
type Service struct {
	source     ProjectSource
	tokens     AppTokenSource
	students   student.Repository
	curriculum curriculum.Curriculum
	now        func() time.Time
}

func NewService(source ProjectSource, tokens AppTokenSource, students student.Repository, c curriculum.Curriculum) *Service {
	return &Service{source: source, tokens: tokens, students: students, curriculum: c, now: time.Now}
}

// SyncStudent fetches the student's validated projects with accessToken (the
// student's own token or an app token) and stores them.
func (s *Service) SyncStudent(ctx context.Context, accessToken string, id student.ID) error {
	projects, err := s.source.CompletedProjects(ctx, accessToken, id)
	if err != nil {
		return fmt.Errorf("fetch projects: %w", err)
	}
	if err := s.students.ReplaceProjects(ctx, id, projects, s.now().UTC()); err != nil {
		return fmt.Errorf("save projects: %w", err)
	}
	return nil
}

// SyncFailure is a student whose sync failed during SyncAll.
type SyncFailure struct {
	Login string
	Err   error
}

// SyncReport summarizes a SyncAll run.
type SyncReport struct {
	Students int // registered students
	Skipped  int // synced recently enough
	Synced   int
	Failures []SyncFailure
	Duration time.Duration
}

// SyncAll syncs, one at a time with the app token, every registered student
// whose projects were never synced or were synced at least maxAge ago (a login
// syncs the student too, so active students are skipped). A failing student
// is recorded in the report and does not stop the run; a cancelled context, an
// unavailable app token or a used-up 42 budget (upstream.ErrBusy) does.
func (s *Service) SyncAll(ctx context.Context, maxAge time.Duration) (report SyncReport, err error) {
	start := s.now()
	defer func() { report.Duration = s.now().Sub(start) }()

	students, err := s.students.ListRegistered(ctx)
	if err != nil {
		return report, fmt.Errorf("list students: %w", err)
	}
	report.Students = len(students)

	now := s.now()
	for _, st := range students {
		if !st.NeedsSync(now, maxAge) {
			report.Skipped++
			continue
		}
		if err := ctx.Err(); err != nil {
			return report, err
		}
		// Cached by the source; re-read so a long run survives token expiry.
		token, err := s.tokens.AppToken(ctx)
		if err != nil {
			return report, err
		}
		if err := s.SyncStudent(ctx, token, st.ID); err != nil {
			if ctx.Err() != nil {
				return report, ctx.Err()
			}
			if errors.Is(err, upstream.ErrBusy) {
				return report, err
			}
			report.Failures = append(report.Failures, SyncFailure{Login: st.Login, Err: err})
			continue
		}
		report.Synced++
	}
	return report, nil
}

// Board is the ranked leaderboard.
type Board struct {
	UpdatedAt time.Time // latest sync among all students; zero if empty
	Entries   []leaderboard.Entry
}

// Leaderboard ranks all synced students.
func (s *Service) Leaderboard(ctx context.Context) (Board, error) {
	students, err := s.students.ListSynced(ctx)
	if err != nil {
		return Board{}, err
	}

	var board Board
	for _, st := range students {
		if st.ProjectsSyncedAt.After(board.UpdatedAt) {
			board.UpdatedAt = st.ProjectsSyncedAt
		}
	}
	board.Entries = leaderboard.Rank(s.curriculum, students)
	return board, nil
}
