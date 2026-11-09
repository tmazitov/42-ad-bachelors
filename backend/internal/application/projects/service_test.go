package projects

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/upstream"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
)

type fakeCatalog struct {
	projects   []Project
	err        error
	calls      int
	background bool
}

func (f *fakeCatalog) CursusProjects(ctx context.Context, _ string) ([]Project, error) {
	f.calls++
	f.background = upstream.IsBackground(ctx)
	return f.projects, f.err
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
	svc := NewService(f, f, curriculum.ADBachelor())

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
	svc := NewService(f, f, curriculum.ADBachelor())
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

	// After a day it is refetched; if that fails, the old list is still used.
	now = now.Add(25 * time.Hour)
	f.err = errors.New("42 is down")
	got, err := svc.Search(context.Background(), "libft", 10)
	if err != nil || len(got) == 0 || f.calls != 2 {
		t.Errorf("stale fallback: %v, %v, calls = %d", names(got), err, f.calls)
	}
}

func TestSearchFailsWithoutCatalogue(t *testing.T) {
	f := &fakeCatalog{err: upstream.ErrBusy}
	svc := NewService(f, f, curriculum.ADBachelor())
	if _, err := svc.Search(context.Background(), "libft", 10); !errors.Is(err, upstream.ErrBusy) {
		t.Errorf("err = %v, want ErrBusy", err)
	}
}
