package progress

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/upstream"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

type fakeSource struct {
	failFor map[student.ID]bool
	failErr error // returned for failFor; default "42 says no"
	tokens  []string
}

func (f *fakeSource) CompletedProjects(_ context.Context, token string, id student.ID) ([]student.CompletedProject, error) {
	f.tokens = append(f.tokens, token)
	if f.failFor[id] {
		if f.failErr != nil {
			return nil, f.failErr
		}
		return nil, errors.New("42 says no")
	}
	return []student.CompletedProject{{ProjectID: 1314}}, nil
}

type fakeTokens struct{ err error }

func (f fakeTokens) AppToken(context.Context) (string, error) { return "app", f.err }

type memRepo struct {
	student.Repository
	students []student.Student
	replaced []student.ID
}

func (m *memRepo) ReplaceProjects(_ context.Context, id student.ID, _ []student.CompletedProject, _ time.Time) error {
	m.replaced = append(m.replaced, id)
	return nil
}
func (m *memRepo) ListRegistered(context.Context) ([]student.Student, error) { return m.students, nil }

func newRepo() *memRepo {
	return &memRepo{students: []student.Student{
		{ID: 1, Login: "alice"}, {ID: 2, Login: "bob"}, {ID: 3, Login: "carol"},
	}}
}

func TestSyncAllContinuesPastFailures(t *testing.T) {
	repo := newRepo()
	src := &fakeSource{failFor: map[student.ID]bool{2: true}}
	svc := NewService(src, fakeTokens{}, repo, curriculum.ADBachelor())

	report, err := svc.SyncAll(context.Background(), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if report.Students != 3 || report.Synced != 2 || len(report.Failures) != 1 || report.Failures[0].Login != "bob" {
		t.Errorf("report = %+v", report)
	}
	if len(repo.replaced) != 2 || repo.replaced[0] != 1 || repo.replaced[1] != 3 {
		t.Errorf("replaced = %v, want [1 3]", repo.replaced)
	}
	for _, tok := range src.tokens {
		if tok != "app" {
			t.Errorf("synced with token %q, want the app token", tok)
		}
	}
}

func TestSyncAllStopsWithoutAppToken(t *testing.T) {
	repo := newRepo()
	svc := NewService(&fakeSource{}, fakeTokens{err: errors.New("bad secret")}, repo, curriculum.ADBachelor())

	if _, err := svc.SyncAll(context.Background(), 24*time.Hour); err == nil {
		t.Fatal("want error when app token is unavailable")
	}
	if len(repo.replaced) != 0 {
		t.Errorf("replaced = %v, want none", repo.replaced)
	}
}

func TestSyncAllStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := NewService(&fakeSource{}, fakeTokens{}, newRepo(), curriculum.ADBachelor())

	report, err := svc.SyncAll(ctx, 24*time.Hour)
	if !errors.Is(err, context.Canceled) || report.Synced != 0 {
		t.Errorf("report = %+v, err = %v", report, err)
	}
}

func TestSyncAllSkipsRecentlySynced(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	repo := &memRepo{students: []student.Student{
		{ID: 1, Login: "fresh", ProjectsSyncedAt: now.Add(-2 * time.Hour)},
		{ID: 2, Login: "stale", ProjectsSyncedAt: now.Add(-25 * time.Hour)},
		{ID: 3, Login: "never"},
	}}
	svc := NewService(&fakeSource{}, fakeTokens{}, repo, curriculum.ADBachelor())
	svc.now = func() time.Time { return now }

	report, err := svc.SyncAll(context.Background(), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if report.Students != 3 || report.Skipped != 1 || report.Synced != 2 {
		t.Errorf("report = %+v", report)
	}
	if len(repo.replaced) != 2 || repo.replaced[0] != 2 || repo.replaced[1] != 3 {
		t.Errorf("replaced = %v, want [2 3]", repo.replaced)
	}
}

func TestSyncAllStopsWhen42IsBusy(t *testing.T) {
	repo := newRepo()
	src := &fakeSource{failFor: map[student.ID]bool{2: true}, failErr: upstream.ErrBusy}
	svc := NewService(src, fakeTokens{}, repo, curriculum.ADBachelor())

	report, err := svc.SyncAll(context.Background(), 24*time.Hour)
	if !errors.Is(err, upstream.ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
	// alice synced, bob hit the busy budget, carol was not attempted.
	if report.Synced != 1 || len(src.tokens) != 2 {
		t.Errorf("report = %+v, 42 calls = %d", report, len(src.tokens))
	}
}
