package progress

import (
	"context"
	"errors"
	"fmt"
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

// boardRepo serves ListSynced and counts calls.
type boardRepo struct {
	student.Repository
	students []student.Student
	lists    int
}

func (r *boardRepo) ListSynced(context.Context) ([]student.Student, error) {
	r.lists++
	return r.students, nil
}

func boardStudents() []student.Student {
	var out []student.Student
	synced := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	// 25 students: s00 has no projects, s01 has Libft (4 credits), s02 Libft + get_next_line (7)...
	ids := []curriculum.ProjectID{1314, 1327, 1331, 1336, 1335}
	for i := range 25 {
		st := student.Student{ID: student.ID(i + 1), Login: fmt.Sprintf("s%02d", i), DisplayName: fmt.Sprintf("Student %d", i),
			ProjectsSyncedAt: synced.Add(time.Duration(i) * time.Minute)}
		for _, id := range ids[:min(i%6, len(ids))] {
			st.Projects = append(st.Projects, student.CompletedProject{ProjectID: id})
		}
		out = append(out, st)
	}
	return out
}

func TestLeaderboardIsCachedForTwoMinutes(t *testing.T) {
	repo := &boardRepo{students: boardStudents()}
	svc := NewService(&fakeSource{}, fakeTokens{}, repo, curriculum.ADBachelor())
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	for range 3 {
		if _, err := svc.Leaderboard(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	now = now.Add(119 * time.Second)
	svc.LeaderboardPage(context.Background(), "", 2, 10, 0)
	if repo.lists != 1 {
		t.Errorf("ListSynced called %d times within 2 minutes, want 1", repo.lists)
	}
	now = now.Add(2 * time.Second)
	svc.Leaderboard(context.Background())
	if repo.lists != 2 {
		t.Errorf("ListSynced called %d times after 2 minutes, want 2", repo.lists)
	}
}

func TestLeaderboardPage(t *testing.T) {
	repo := &boardRepo{students: boardStudents()}
	svc := NewService(&fakeSource{}, fakeTokens{}, repo, curriculum.ADBachelor())
	ctx := context.Background()

	p, err := svc.LeaderboardPage(ctx, "", 2, 10, 1) // viewer: s00, no credits, last
	if err != nil {
		t.Fatal(err)
	}
	if p.Students != 25 || p.Matches != 25 || p.Page != 2 || len(p.Entries) != 10 {
		t.Errorf("page 2: students %d, matches %d, page %d, entries %d", p.Students, p.Matches, p.Page, len(p.Entries))
	}
	if p.Entries[0].Rank > p.Entries[len(p.Entries)-1].Rank {
		t.Error("entries not in rank order")
	}
	if p.Me == nil || p.Me.Entry.Student.Login != "s00" || p.Me.Percentile != 0 {
		t.Fatalf("me = %+v", p.Me)
	}
	if p.Me.Page != 3 { // the five 0-credit students are last: rows 21-25
		t.Errorf("my page = %d, want 3", p.Me.Page)
	}
	if want := time.Date(2026, 11, 9, 10, 24, 0, 0, time.UTC); !p.UpdatedAt.Equal(want) {
		t.Errorf("updated at %v, want %v", p.UpdatedAt, want)
	}

	// Past the end: clamped to the last page.
	if p, _ := svc.LeaderboardPage(ctx, "", 99, 10, 0); p.Page != 3 || len(p.Entries) != 5 || p.Me != nil {
		t.Errorf("page 99 = page %d with %d entries, me %v", p.Page, len(p.Entries), p.Me)
	}

	// Search by login or display name; totals stay those of the whole board.
	p, _ = svc.LeaderboardPage(ctx, "STUDENT 1", 1, 10, 0)
	if p.Matches != 11 || p.Students != 25 || len(p.Entries) != 10 { // 1, 10-19
		t.Errorf("search: matches %d, students %d, entries %d", p.Matches, p.Students, len(p.Entries))
	}
	if p, _ := svc.LeaderboardPage(ctx, "nobody", 1, 10, 0); p.Matches != 0 || p.Page != 1 || len(p.Entries) != 0 {
		t.Errorf("no match: %+v", p)
	}

	// The top student is ahead of everyone with fewer credits.
	p, _ = svc.LeaderboardPage(ctx, "", 1, 10, 6) // s05: 5 projects, the most
	// 4 and 5 projects give the same credits here, so 8 students share rank
	// 1 and 17 of the 24 others are below: 71%.
	if p.Me == nil || p.Me.Entry.Rank != 1 || p.Me.Page != 1 || p.Me.Percentile != 71 {
		t.Errorf("s05 rank %d, page %d, percentile %d", p.Me.Entry.Rank, p.Me.Page, p.Me.Percentile)
	}
}
