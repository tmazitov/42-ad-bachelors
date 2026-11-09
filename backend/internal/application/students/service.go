// Package students serves student profiles and progress from the database,
// refreshing them from 42 with the app token, so the browser never calls the
// 42 API itself:
//   - registered students (signed in to the app) are answered from the
//     database at once; data older than maxAge is refreshed in the background
//   - other students (only viewed) are refreshed before answering when their
//     data is older than visitorMaxAge
package students

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/cache"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/upstream"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

var (
	// ErrInvalidLogin means the login is not a valid 42 login.
	ErrInvalidLogin = errors.New("invalid login")
	// ErrUnknownStream means the stream key is not in the curriculum.
	ErrUnknownStream = errors.New("unknown stream")
	// ErrNotSelfReported means the module cannot be marked completed by the
	// student (unknown, or tracked through 42 only).
	ErrNotSelfReported = errors.New("module cannot be marked by the student")
)

// notFoundTTL is how long a login that 42 does not know is remembered, so
// repeated lookups of it don't cost API calls.
const (
	notFoundTTL        = time.Hour
	notFoundMaxEntries = 5000
)

// refreshTimeout bounds a 42 refresh, which is shared by concurrent callers
// and so does not use any single caller's context.
const refreshTimeout = 30 * time.Second

// ProfileSource looks up 42 users.
type ProfileSource interface {
	// UserByLogin returns the user, or student.ErrNotFound.
	UserByLogin(ctx context.Context, accessToken, login string) (student.Student, error)
}

// ProjectSyncer refreshes a student's stored projects.
type ProjectSyncer interface {
	SyncStudent(ctx context.Context, accessToken string, id student.ID) error
}

// AppTokenSource provides a 42 application token.
type AppTokenSource interface {
	AppToken(ctx context.Context) (string, error)
}

// View is a student with their progress.
type View struct {
	Student  student.Student
	Progress curriculum.Progress
}

// Service runs the student read use cases.
type Service struct {
	profiles   ProfileSource
	projects   ProjectSyncer
	tokens     AppTokenSource
	students   student.Repository
	curriculum curriculum.Curriculum
	// maxAge is how old a registered student's data may get before it is
	// refreshed in the background; visitorMaxAge is the same for students
	// who never signed in, refreshed before answering.
	maxAge        time.Duration
	visitorMaxAge time.Duration
	log           *slog.Logger
	now           func() time.Time

	refreshes  singleflight.Group
	background sync.WaitGroup
	// notFound holds logins 42 answered 404 for.
	notFound *cache.TTL[string, struct{}]
}

func NewService(profiles ProfileSource, projects ProjectSyncer, tokens AppTokenSource,
	students student.Repository, c curriculum.Curriculum, maxAge, visitorMaxAge time.Duration,
	log *slog.Logger) *Service {
	return &Service{
		profiles:      profiles,
		projects:      projects,
		tokens:        tokens,
		students:      students,
		curriculum:    c,
		maxAge:        maxAge,
		visitorMaxAge: visitorMaxAge,
		log:           log,
		now:           time.Now,
		notFound:      cache.NewTTL[string, struct{}](notFoundTTL, notFoundMaxEntries),
	}
}

// Wait blocks until background refreshes have finished, e.g. before the
// database is closed.
func (s *Service) Wait() {
	s.background.Wait()
}

// ByLogin returns any 42 student's profile and progress. Students not stored
// yet are fetched from 42 and stored (but not registered).
func (s *Service) ByLogin(ctx context.Context, login string) (View, error) {
	login = strings.ToLower(strings.TrimSpace(login))
	if !student.ValidLogin(login) {
		return View{}, ErrInvalidLogin
	}
	st, err := s.students.FindByLogin(ctx, login)
	if err != nil && !errors.Is(err, student.ErrNotFound) {
		return View{}, err
	}
	stored := err == nil
	if !stored {
		if _, unknown := s.notFound.Get(login); unknown {
			return View{}, student.ErrNotFound
		}
	}
	return s.fresh(ctx, login, st, stored)
}

// ByID returns a stored student's profile and progress, e.g. the signed-in one.
func (s *Service) ByID(ctx context.Context, id student.ID) (View, error) {
	st, err := s.students.FindByID(ctx, id)
	if err != nil {
		return View{}, err
	}
	return s.fresh(ctx, st.Login, st, true)
}

// SetStream stores the student's chosen stream and returns their updated view.
func (s *Service) SetStream(ctx context.Context, id student.ID, stream string) (View, error) {
	if !s.curriculum.HasStream(stream) {
		return View{}, ErrUnknownStream
	}
	if err := s.students.SetStream(ctx, id, stream); err != nil {
		return View{}, err
	}
	st, err := s.students.FindByID(ctx, id)
	if err != nil {
		return View{}, err
	}
	return s.view(st), nil
}

// MarkModule marks a self-reported module completed (or unmarks it) for the
// student and returns their updated view.
func (s *Service) MarkModule(ctx context.Context, id student.ID, code string, completed bool) (View, error) {
	if !s.curriculum.SelfReported(code) {
		return View{}, ErrNotSelfReported
	}
	if err := s.students.SetModuleMark(ctx, id, code, completed); err != nil {
		return View{}, err
	}
	st, err := s.students.FindByID(ctx, id)
	if err != nil {
		return View{}, err
	}
	return s.view(st), nil
}

// fresh returns st, refreshing it from 42 when it is stale (see the package
// comment). A registered student's stored data is returned at once and
// refreshed in the background. Others are refreshed before answering; if that
// fails but stored data exists, the stored data is returned.
func (s *Service) fresh(ctx context.Context, login string, st student.Student, stored bool) (View, error) {
	now := s.now()
	if stored && st.Registered() && !st.ProjectsSyncedAt.IsZero() {
		if st.NeedsSync(now, s.maxAge) {
			s.refreshInBackground(login)
		}
		return s.view(st), nil
	}
	if stored && !st.NeedsSync(now, s.visitorMaxAge) {
		return s.view(st), nil
	}

	refreshed, err := s.refresh(ctx, login)
	switch {
	case err == nil:
		return s.view(refreshed), nil
	case stored && !errors.Is(err, student.ErrNotFound) && ctx.Err() == nil:
		s.log.WarnContext(ctx, "serving stale student data", "login", login, "err", err)
		return s.view(st), nil
	default:
		return View{}, err
	}
}

// refreshInBackground starts a refresh that outlives the request; a failure
// is logged and the stored data stays.
func (s *Service) refreshInBackground(login string) {
	s.background.Go(func() {
		if _, err := s.refresh(context.Background(), login); err != nil {
			s.log.Warn("background refresh failed", "login", login, "err", err)
		}
	})
}

// refresh fetches the student's profile and projects from 42 and stores them.
// Concurrent refreshes of the same login share one 42 round trip. Its 42
// calls are background work (upstream.Background).
func (s *Service) refresh(ctx context.Context, login string) (student.Student, error) {
	ch := s.refreshes.DoChan(login, func() (any, error) {
		ctx, cancel := context.WithTimeout(upstream.Background(context.WithoutCancel(ctx)), refreshTimeout)
		defer cancel()

		token, err := s.tokens.AppToken(ctx)
		if err != nil {
			return nil, err
		}
		profile, err := s.profiles.UserByLogin(ctx, token, login)
		if errors.Is(err, student.ErrNotFound) {
			s.notFound.Set(login, struct{}{})
		}
		if err != nil {
			return nil, err
		}
		if err := s.students.Save(ctx, profile); err != nil {
			return nil, fmt.Errorf("save student: %w", err)
		}
		if err := s.projects.SyncStudent(ctx, token, profile.ID); err != nil {
			return nil, err
		}
		return s.students.FindByID(ctx, profile.ID)
	})

	select {
	case <-ctx.Done():
		return student.Student{}, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return student.Student{}, res.Err
		}
		return res.Val.(student.Student), nil
	}
}

func (s *Service) view(st student.Student) View {
	return View{Student: st, Progress: s.curriculum.Evaluate(st.Validated(), st.Marked(), st.Stream)}
}

// SearchResult is one page of matching registered students.
type SearchResult struct {
	Students []student.Student
	Total    int
}

// Search finds registered students by login or display name.
func (s *Service) Search(ctx context.Context, query string, page, pageSize int) (SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return SearchResult{}, nil
	}
	page = max(page, 1)
	list, total, err := s.students.Search(ctx, query, pageSize, (page-1)*pageSize)
	if err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Students: list, Total: total}, nil
}
