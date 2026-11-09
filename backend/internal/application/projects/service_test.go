package projects

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/upstream"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type fakeCatalog struct {
	mu         sync.Mutex
	projects   []Project
	err        error
	delay      time.Duration
	calls      int
	background bool
}

func (f *fakeCatalog) CursusProjects(ctx context.Context, _ string) ([]Project, error) {
	time.Sleep(f.delay)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.background = upstream.IsBackground(ctx)
	return f.projects, f.err
}

func (f *fakeCatalog) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeCatalog) AppToken(context.Context) (string, error) { return "app", nil }

var catalogue = []Project{
	{ID: 1314, Name: "Libft", Slug: "42cursus-libft"},
	{ID: 1327, Name: "get_next_line", Slug: "42cursus-get_next_line"},
	{ID: 1320, Name: "Exam Rank 02", Slug: "exam-rank-02", Exam: true},
	{ID: 9001, Name: "Libft Bonus Step", Slug: "libft-bonus-step", Child: true},
	{ID: 9002, Name: "My libft fan club", Slug: "fan-club"},
	{ID: 9003, Name: "Unrelated", Slug: "x-libft-y"},
	{ID: 1337, Name: "ft_transcendence", Slug: "ft_transcendence"},
}

func names(rs []Result) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Project.Name)
	}
	return out
}

func TestSearchRanksAndFilters(t *testing.T) {
	f := &fakeCatalog{projects: catalogue}
	svc := NewService(f, f, curriculum.ADBachelor(), quiet)

	got, err := svc.Search(context.Background(), " LIBFT ", 10)
	if err != nil {
		t.Fatal(err)
	}
	// exact name, then word prefix, then slug; the exam and the child are left out.
	want := []string{"Libft", "My libft fan club", "Unrelated"}
	if len(got) != len(want) {
		t.Fatalf("results = %v, want %v", names(got), want)
	}
	for i := range want {
		if got[i].Project.Name != want[i] {
			t.Errorf("result %d = %s, want %s", i, got[i].Project.Name, want[i])
		}
	}
	if len(got[0].Modules) != 1 || got[0].Modules[0].Code != "CCP 101" || got[0].Modules[0].Category != curriculum.Standard {
		t.Errorf("Libft modules = %+v, want CCP 101 (standard)", got[0].Modules)
	}
	if len(got[1].Modules) != 0 {
		t.Errorf("non-bachelor project has modules %+v", got[1].Modules)
	}

	if got, _ := svc.Search(context.Background(), "", 10); len(got) != 0 {
		t.Errorf("empty query = %v", names(got))
	}
	if got, _ := svc.Search(context.Background(), "t", 2); len(got) != 2 {
		t.Errorf("limit 2 = %v", names(got))
	}
}

func TestCatalogueIsFetchedOnceAndRefreshedDaily(t *testing.T) {
	f := &fakeCatalog{projects: catalogue}
	svc := NewService(f, f, curriculum.ADBachelor(), quiet)
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	for _, q := range []string{"libft", "get", "trans"} {
		if _, err := svc.Search(context.Background(), q, 10); err != nil {
			t.Fatal(err)
		}
	}
	if f.calls != 1 || !f.background {
		t.Errorf("catalogue fetched %d times (background %v), want once as background work", f.calls, f.background)
	}

	// After a day the old list is still served while it is refetched in
	// the background; a failed refetch keeps it.
	now = now.Add(25 * time.Hour)
	f.mu.Lock()
	f.err = errors.New("42 is down")
	f.mu.Unlock()
	got, err := svc.Search(context.Background(), "libft", 10)
	if err != nil || len(got) == 0 {
		t.Errorf("stale list: %v, %v", names(got), err)
	}
	deadline := time.Now().Add(time.Second)
	for f.callCount() != 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if f.callCount() != 2 {
		t.Errorf("calls = %d, want a background refetch", f.callCount())
	}
	if got, _ := svc.Search(context.Background(), "libft", 10); len(got) == 0 {
		t.Error("the old list was dropped after a failed refetch")
	}
}

func TestSlowCatalogueAnswersLoadingThenServes(t *testing.T) {
	f := &fakeCatalog{projects: catalogue, delay: 200 * time.Millisecond}
	svc := NewService(f, f, curriculum.ADBachelor(), quiet)
	svc.searchWait = 20 * time.Millisecond

	svc.Warm()
	start := time.Now()
	if _, err := svc.Search(context.Background(), "libft", 10); !errors.Is(err, ErrLoading) {
		t.Fatalf("err = %v, want ErrLoading while the catalogue loads", err)
	}
	if d := time.Since(start); d > 150*time.Millisecond {
		t.Errorf("waited %v, want about searchWait", d)
	}

	time.Sleep(300 * time.Millisecond)
	got, err := svc.Search(context.Background(), "libft", 10)
	if err != nil || len(got) == 0 || f.callCount() != 1 {
		t.Errorf("after loading: %v, %v, calls = %d (want one shared fetch)", names(got), err, f.callCount())
	}
}

func TestSearchFailsWithoutCatalogue(t *testing.T) {
	f := &fakeCatalog{err: upstream.ErrBusy}
	svc := NewService(f, f, curriculum.ADBachelor(), quiet)
	if _, err := svc.Search(context.Background(), "libft", 10); !errors.Is(err, upstream.ErrBusy) {
		t.Errorf("err = %v, want ErrBusy", err)
	}
}
