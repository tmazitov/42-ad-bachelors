package students

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/upstream"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

var now = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

// memRepo is an in-memory student.Repository.
type memRepo struct {
	student.Repository
	mu   sync.Mutex
	byID map[student.ID]student.Student
}

func newRepo(students ...student.Student) *memRepo {
	r := &memRepo{byID: map[student.ID]student.Student{}}
	for _, s := range students {
		r.byID[s.ID] = s
	}
	return r
}

func (r *memRepo) Save(_ context.Context, s student.Student) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.byID[s.ID]
	s.Projects, s.ProjectsSyncedAt = old.Projects, old.ProjectsSyncedAt
	if s.RegisteredAt.IsZero() {
		s.RegisteredAt = old.RegisteredAt
	}
	r.byID[s.ID] = s
	return nil
}

func (r *memRepo) ReplaceProjects(_ context.Context, id student.ID, p []student.CompletedProject, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.byID[id]
	s.Projects, s.ProjectsSyncedAt = p, at
	r.byID[id] = s
	return nil
}

func (r *memRepo) SetStream(_ context.Context, id student.ID, stream string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, ok := r.byID[id]
	if !ok {
		return student.ErrNotFound
	}
	st.Stream = stream
	r.byID[id] = st
	return nil
}

func (r *memRepo) SetModuleMark(_ context.Context, id student.ID, code string, completed bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, ok := r.byID[id]
	if !ok {
		return student.ErrNotFound
	}
	marks := []string{}
	for _, c := range st.MarkedModules {
		if c != code {
			marks = append(marks, c)
		}
	}
	if completed {
		marks = append(marks, code)
	}
	st.MarkedModules = marks
	r.byID[id] = st
	return nil
}

func (r *memRepo) FindByID(_ context.Context, id student.ID) (student.Student, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.byID[id]
	if !ok {
		return student.Student{}, student.ErrNotFound
	}
	return s, nil
}

func (r *memRepo) FindByLogin(_ context.Context, login string) (student.Student, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.byID {
		if s.Login == login {
			return s, nil
		}
	}
	return student.Student{}, student.ErrNotFound
}

// fake42 serves profiles and syncs projects into the repo.
type fake42 struct {
	repo    *memRepo
	users   map[string]student.Student
	fail    error
	delay   time.Duration
	lookups atomic.Int32
	// foreground counts lookups not marked upstream.Background.
	foreground atomic.Int32
	searches   atomic.Int32
}

func (f *fake42) AppToken(context.Context) (string, error) { return "app", nil }

func (f *fake42) UserByLogin(ctx context.Context, _, login string) (student.Student, error) {
	f.lookups.Add(1)
	if !upstream.IsBackground(ctx) {
		f.foreground.Add(1)
	}
	time.Sleep(f.delay)
	if f.fail != nil {
		return student.Student{}, f.fail
	}
	u, ok := f.users[login]
	if !ok {
		return student.Student{}, student.ErrNotFound
	}
	return u, nil
}

func (f *fake42) SearchCampusUsers(ctx context.Context, _, query string, limit int) ([]student.Student, error) {
	f.searches.Add(1)
	if !upstream.IsBackground(ctx) {
		f.foreground.Add(1)
	}
	if f.fail != nil {
		return nil, f.fail
	}
	var out []student.Student
	for _, login := range []string{"alice", "bob", "bobby", "3b3-x1", "carol"} {
		if u, ok := f.users[login]; ok && strings.Contains(login, query) && len(out) < limit {
			out = append(out, u)
		}
	}
	return out, nil
}

func (f *fake42) SyncStudent(ctx context.Context, _ string, id student.ID) error {
	return f.repo.ReplaceProjects(ctx, id, []student.CompletedProject{{ProjectID: 1314}}, now)
}

func newService(repo *memRepo, f *fake42) *Service {
	f.repo = repo
	svc := NewService(f, f, f, repo, curriculum.ADBachelor(), 2*time.Minute, 5*time.Minute,
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.now = func() time.Time { return now }
	return svc
}

var bob = student.Student{ID: 2, Login: "bob", DisplayName: "Bob"}

func TestByLoginServesFreshDataWithout42(t *testing.T) {
	fresh := student.Student{ID: 1, Login: "alice", DisplayName: "Alice", ProjectsSyncedAt: now.Add(-time.Minute),
		Projects: []student.CompletedProject{{ProjectID: 1314}}}
	f := &fake42{}
	svc := newService(newRepo(fresh), f)

	v, err := svc.ByLogin(context.Background(), "Alice")
	if err != nil {
		t.Fatal(err)
	}
	if v.Student.Login != "alice" || v.Progress.Total != 4 {
		t.Errorf("view = %+v", v)
	}
	if f.lookups.Load() != 0 {
		t.Errorf("42 called %d times for fresh data", f.lookups.Load())
	}
}

func TestByLoginRefreshesStaleAndUnknown(t *testing.T) {
	stale := student.Student{ID: 2, Login: "bob", DisplayName: "Old Bob", ProjectsSyncedAt: now.Add(-time.Hour)}
	f := &fake42{users: map[string]student.Student{"bob": bob, "carol": {ID: 3, Login: "carol", DisplayName: "Carol"}}}
	repo := newRepo(stale)
	svc := newService(repo, f)

	v, err := svc.ByLogin(context.Background(), "bob")
	if err != nil {
		t.Fatal(err)
	}
	if v.Student.DisplayName != "Bob" || !v.Student.ProjectsSyncedAt.Equal(now) || v.Progress.Total != 4 {
		t.Errorf("stale student not refreshed: %+v", v)
	}

	// Unknown locally: fetched, stored, but not registered.
	v, err = svc.ByLogin(context.Background(), "carol")
	if err != nil {
		t.Fatal(err)
	}
	if v.Student.ID != 3 || v.Student.Registered() {
		t.Errorf("carol = %+v", v.Student)
	}
	if _, err := repo.FindByLogin(context.Background(), "carol"); err != nil {
		t.Errorf("carol not stored: %v", err)
	}
}

func TestByLoginErrors(t *testing.T) {
	svc := newService(newRepo(), &fake42{users: map[string]student.Student{}})

	if _, err := svc.ByLogin(context.Background(), "../etc"); !errors.Is(err, ErrInvalidLogin) {
		t.Errorf("invalid login err = %v", err)
	}
	if _, err := svc.ByLogin(context.Background(), "ghost"); !errors.Is(err, student.ErrNotFound) {
		t.Errorf("unknown login err = %v, want ErrNotFound", err)
	}
}

func TestStaleDataServedWhen42Fails(t *testing.T) {
	stale := student.Student{ID: 2, Login: "bob", DisplayName: "Bob", ProjectsSyncedAt: now.Add(-time.Hour)}
	svc := newService(newRepo(stale), &fake42{fail: errors.New("42 is down")})

	v, err := svc.ByLogin(context.Background(), "bob")
	if err != nil || v.Student.Login != "bob" {
		t.Errorf("ByLogin = %+v, %v; want stale bob", v.Student, err)
	}

	empty := newService(newRepo(), &fake42{fail: errors.New("42 is down")})
	if _, err := empty.ByLogin(context.Background(), "bob"); err == nil {
		t.Error("want error when nothing is stored and 42 fails")
	}
}

func TestConcurrentRefreshesShareOneLookup(t *testing.T) {
	f := &fake42{users: map[string]student.Student{"bob": bob}, delay: 50 * time.Millisecond}
	svc := newService(newRepo(), f)

	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			if _, err := svc.ByLogin(context.Background(), "bob"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if n := f.lookups.Load(); n != 1 {
		t.Errorf("42 lookups = %d, want 1", n)
	}
}

func TestSetStream(t *testing.T) {
	alice := student.Student{ID: 1, Login: "alice", DisplayName: "Alice", ProjectsSyncedAt: now,
		Projects: []student.CompletedProject{{ProjectID: 1396}, {ProjectID: 1397}}} // camagru 2 (mobile-web), ft_ping 2 (network-security)
	svc := newService(newRepo(alice), &fake42{})

	if _, err := svc.SetStream(context.Background(), 1, "bogus"); !errors.Is(err, ErrUnknownStream) {
		t.Errorf("bogus stream err = %v, want ErrUnknownStream", err)
	}
	v, err := svc.SetStream(context.Background(), 1, "network-security")
	if err != nil {
		t.Fatal(err)
	}
	if v.Student.Stream != "network-security" || v.Progress.Stream != "network-security" || !v.Progress.StreamChosen {
		t.Errorf("view = %+v / %+v", v.Student.Stream, v.Progress)
	}
	if _, err := svc.SetStream(context.Background(), 99, "mobile-web"); !errors.Is(err, student.ErrNotFound) {
		t.Errorf("unknown student err = %v, want ErrNotFound", err)
	}
}

func TestMarkModule(t *testing.T) {
	alice := student.Student{ID: 1, Login: "alice", DisplayName: "Alice", ProjectsSyncedAt: now}
	svc := newService(newRepo(alice), &fake42{})
	ctx := context.Background()

	if _, err := svc.MarkModule(ctx, 1, "CCP 101", true); !errors.Is(err, ErrNotSelfReported) {
		t.Errorf("CCP 101 err = %v, want ErrNotSelfReported", err)
	}
	v, err := svc.MarkModule(ctx, 1, "GEN 201", true)
	if err != nil {
		t.Fatal(err)
	}
	if v.Progress.Credits[curriculum.General] != 2 {
		t.Errorf("general = %d, want 2", v.Progress.Credits[curriculum.General])
	}
	if v, _ = svc.MarkModule(ctx, 1, "GEN 201", false); v.Progress.Credits[curriculum.General] != 0 {
		t.Errorf("general after unmark = %d, want 0", v.Progress.Credits[curriculum.General])
	}
	if _, err := svc.MarkModule(ctx, 99, "GEN 101", true); !errors.Is(err, student.ErrNotFound) {
		t.Errorf("unknown student err = %v, want ErrNotFound", err)
	}
}

func TestRegisteredStaleIsServedAtOnceAndRefreshedInBackground(t *testing.T) {
	alice := student.Student{ID: 1, Login: "alice", DisplayName: "Old Alice", RegisteredAt: now.Add(-48 * time.Hour),
		ProjectsSyncedAt: now.Add(-3 * time.Minute)} // older than maxAge (2m in tests)
	f := &fake42{users: map[string]student.Student{"alice": {ID: 1, Login: "alice", DisplayName: "New Alice"}},
		delay: 200 * time.Millisecond}
	repo := newRepo(alice)
	svc := newService(repo, f)

	start := time.Now()
	v, err := svc.ByLogin(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if v.Student.DisplayName != "Old Alice" || time.Since(start) > 100*time.Millisecond {
		t.Errorf("got %q after %v, want the stored data at once", v.Student.DisplayName, time.Since(start))
	}

	svc.Wait()
	st, _ := repo.FindByID(context.Background(), 1)
	if st.DisplayName != "New Alice" || !st.ProjectsSyncedAt.Equal(now) {
		t.Errorf("after background refresh: %+v", st)
	}
	if f.lookups.Load() != 1 || f.foreground.Load() != 0 {
		t.Errorf("lookups = %d (%d not marked background), want 1 background", f.lookups.Load(), f.foreground.Load())
	}

	// Fresh again: no 42 call.
	if _, err := svc.ByID(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	svc.Wait()
	if f.lookups.Load() != 1 {
		t.Errorf("lookups = %d after a fresh read, want 1", f.lookups.Load())
	}
}

func TestRegisteredNeverSyncedIsRefreshedBeforeAnswering(t *testing.T) {
	alice := student.Student{ID: 1, Login: "alice", DisplayName: "Alice", RegisteredAt: now}
	f := &fake42{users: map[string]student.Student{"alice": {ID: 1, Login: "alice", DisplayName: "Alice"}}}
	svc := newService(newRepo(alice), f)

	v, err := svc.ByID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if v.Progress.Total != 4 || f.lookups.Load() != 1 {
		t.Errorf("total = %d, lookups = %d; want the synced projects", v.Progress.Total, f.lookups.Load())
	}
}

func TestVisitorMaxAge(t *testing.T) {
	for _, tt := range []struct {
		age     time.Duration
		lookups int32
	}{
		{4 * time.Minute, 0}, // within visitorMaxAge (5m in tests)
		{6 * time.Minute, 1},
	} {
		bob := student.Student{ID: 2, Login: "bob", DisplayName: "Bob", ProjectsSyncedAt: now.Add(-tt.age)}
		f := &fake42{users: map[string]student.Student{"bob": bob}}
		svc := newService(newRepo(bob), f)
		if _, err := svc.ByLogin(context.Background(), "bob"); err != nil {
			t.Fatal(err)
		}
		if f.lookups.Load() != tt.lookups {
			t.Errorf("visitor synced %v ago: lookups = %d, want %d", tt.age, f.lookups.Load(), tt.lookups)
		}
	}
}

func TestUnknownLoginsAreRemembered(t *testing.T) {
	f := &fake42{users: map[string]student.Student{}}
	svc := newService(newRepo(), f)

	for range 3 {
		if _, err := svc.ByLogin(context.Background(), "ghost"); !errors.Is(err, student.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	}
	if n := f.lookups.Load(); n != 1 {
		t.Errorf("42 lookups = %d, want 1 (later ones answered from the cache)", n)
	}

	// Other errors are not remembered.
	f.fail = errors.New("42 is down")
	for range 2 {
		if _, err := svc.ByLogin(context.Background(), "carol"); err == nil {
			t.Fatal("want error")
		}
	}
	if n := f.lookups.Load(); n != 3 {
		t.Errorf("42 lookups = %d, want 3 (failures are retried)", n)
	}
}

// searchRepo adds Search over registered students to memRepo.
type searchRepo struct{ *memRepo }

func (r searchRepo) Search(_ context.Context, query string, limit, offset int) ([]student.Student, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var all []student.Student
	for _, s := range r.byID {
		if s.Registered() && strings.Contains(s.Login, strings.ToLower(query)) {
			all = append(all, s)
		}
	}
	return all[min(offset, len(all)):min(offset+limit, len(all))], len(all), nil
}

func TestSearchAddsIntraStudents(t *testing.T) {
	registeredBob := student.Student{ID: 2, Login: "bob", DisplayName: "Bob", RegisteredAt: now, ProjectsSyncedAt: now}
	viewedBobby := student.Student{ID: 3, Login: "bobby", DisplayName: "Bobby"} // stored, not registered
	f := &fake42{users: map[string]student.Student{
		"bob": registeredBob, "bobby": viewedBobby,
		"3b3-x1": {ID: 9, Login: "3b3-x1"},
	}}
	repo := newRepo(registeredBob, viewedBobby)
	f.repo = repo
	svc := NewService(f, f, f, searchRepo{repo}, curriculum.ADBachelor(), time.Hour, 5*time.Minute,
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()

	res, err := svc.Search(ctx, "Bo", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Students) != 1 || res.Students[0].Login != "bob" {
		t.Errorf("app students = %+v", res.Students)
	}
	// bob is registered (listed above), 3b3-x1 is anonymized: only bobby.
	if len(res.Intra) != 1 || res.Intra[0].Login != "bobby" {
		t.Errorf("intra = %+v", res.Intra)
	}
	if f.foreground.Load() != 0 {
		t.Error("intra search was not marked as background work")
	}

	// Cached per query: the same search and page 2 cost nothing more.
	svc.Search(ctx, "bo", 1, 10)
	if res, _ := svc.Search(ctx, "bo", 2, 10); res.Intra != nil {
		t.Errorf("page 2 intra = %+v, want none", res.Intra)
	}
	if n := f.searches.Load(); n != 1 {
		t.Errorf("42 searches = %d, want 1", n)
	}

	// Too short, or not a login: no 42 call.
	svc.Search(ctx, "b", 1, 10)
	svc.Search(ctx, "Bob Smith", 1, 10)
	if n := f.searches.Load(); n != 1 {
		t.Errorf("42 searches = %d after short/non-login queries, want 1", n)
	}

	// 42 failing: app students are still returned.
	f.fail = upstream.ErrBusy
	res, err = svc.Search(ctx, "bob", 1, 10)
	if err != nil || len(res.Students) != 1 || len(res.Intra) != 0 {
		t.Errorf("with 42 down: %+v, %v", res, err)
	}
}
