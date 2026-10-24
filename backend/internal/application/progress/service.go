// Package progress syncs students' 42 projects and serves the leaderboard.
package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/leaderboard"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

// ProjectSource fetches a student's validated projects from 42.
type ProjectSource interface {
	CompletedProjects(ctx context.Context, accessToken string, id student.ID) ([]student.CompletedProject, error)
}

// Service runs the progress use cases.
type Service struct {
	source     ProjectSource
	students   student.Repository
	curriculum curriculum.Curriculum
	now        func() time.Time
}

func NewService(source ProjectSource, students student.Repository, c curriculum.Curriculum) *Service {
	return &Service{source: source, students: students, curriculum: c, now: time.Now}
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
