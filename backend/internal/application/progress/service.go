// Package progress syncs students' 42 projects and serves the leaderboard.
package progress

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

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

// boardTTL is how long a computed leaderboard is reused. Ranking reads every
// synced student with their projects, so it is not redone per request.
const boardTTL = 2 * time.Minute

// Service runs the progress use cases.
type Service struct {
	source     ProjectSource
	tokens     AppTokenSource
	students   student.Repository
	curriculum curriculum.Curriculum
	now        func() time.Time

	boardMu     sync.Mutex
	board       *Board
	boardBuilt  time.Time
	boardBuilds singleflight.Group
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
	// Leave the part of the 42 budget kept for sign-ins alone.
	ctx = upstream.Background(ctx)
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
	Students  int     // number of ranked students
	Average   float64 // average percentage
}

// Leaderboard ranks all synced students. The result is cached for boardTTL
// (2 minutes), so changes show up within that time.
func (s *Service) Leaderboard(ctx context.Context) (Board, error) {
	s.boardMu.Lock()
	cached, built := s.board, s.boardBuilt
	s.boardMu.Unlock()
	if cached != nil && s.now().Sub(built) < boardTTL {
		return *cached, nil
	}

	res, err, _ := s.boardBuilds.Do("board", func() (any, error) {
		students, err := s.students.ListSynced(ctx)
		if err != nil {
			return nil, err
		}
		board := &Board{Students: len(students)}
		var sum float64
		for _, st := range students {
			if st.ProjectsSyncedAt.After(board.UpdatedAt) {
				board.UpdatedAt = st.ProjectsSyncedAt
			}
		}
		board.Entries = leaderboard.Rank(s.curriculum, students)
		for _, e := range board.Entries {
			sum += e.Progress.Percentage
		}
		if len(board.Entries) > 0 {
			board.Average = sum / float64(len(board.Entries))
		}

		s.boardMu.Lock()
		s.board, s.boardBuilt = board, s.now()
		s.boardMu.Unlock()
		return board, nil
	})
	if err != nil {
		return Board{}, err
	}
	return *res.(*Board), nil
}

// BoardPage is one page of the leaderboard, optionally filtered by a search.
type BoardPage struct {
	UpdatedAt time.Time
	Students  int     // ranked students in total (not only matches)
	Average   float64 // average percentage of all ranked students
	Entries   []leaderboard.Entry
	Matches   int // entries matching the search (all when no search)
	Page      int // 1-based, clamped to the last page
	PageSize  int
	Me        *MyRank // the viewer's place; nil if they are not ranked
}

// MyRank is the viewer's place on the leaderboard.
type MyRank struct {
	Entry leaderboard.Entry
	// Page is the page holding the viewer in the unfiltered board.
	Page int
	// Percentile is the share (0-100) of other students ranked below them;
	// -1 when they are the only one.
	Percentile int
}

// LeaderboardPage returns page (1-based) of the leaderboard, keeping only
// students whose login or display name contains query (case-insensitive),
// and the place of the student viewer.
func (s *Service) LeaderboardPage(ctx context.Context, query string, page, pageSize int, viewer student.ID) (BoardPage, error) {
	board, err := s.Leaderboard(ctx)
	if err != nil {
		return BoardPage{}, err
	}

	matches := board.Entries
	if q := strings.ToLower(strings.TrimSpace(query)); q != "" {
		matches = nil
		for _, e := range board.Entries {
			if strings.Contains(strings.ToLower(e.Student.Login), q) ||
				strings.Contains(strings.ToLower(e.Student.DisplayName), q) {
				matches = append(matches, e)
			}
		}
	}

	pages := max(1, (len(matches)+pageSize-1)/pageSize)
	page = min(max(page, 1), pages)
	start := (page - 1) * pageSize
	end := min(start+pageSize, len(matches))

	out := BoardPage{
		UpdatedAt: board.UpdatedAt,
		Students:  board.Students,
		Average:   board.Average,
		Entries:   matches[start:end],
		Matches:   len(matches),
		Page:      page,
		PageSize:  pageSize,
	}
	for i, e := range board.Entries {
		if e.Student.ID != viewer {
			continue
		}
		me := &MyRank{Entry: e, Page: i/pageSize + 1, Percentile: -1}
		if n := len(board.Entries); n > 1 {
			below := 0
			for _, other := range board.Entries {
				if other.Progress.Total < e.Progress.Total {
					below++
				}
			}
			me.Percentile = int(float64(below)/float64(n-1)*100 + 0.5)
		}
		out.Me = me
		break
	}
	return out, nil
}
