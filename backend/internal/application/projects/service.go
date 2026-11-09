// Package projects searches the 42cursus project catalogue and tells which
// bachelor modules each project counts for. The catalogue is fetched from 42
// once and kept in memory, so searching costs no API calls.
package projects

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/upstream"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
)

// catalogTTL is how long the fetched catalogue is used before it is
// fetched again; projects rarely change.
const catalogTTL = 24 * time.Hour

// fetchTimeout bounds a catalogue fetch, which is shared by concurrent
// searches and so does not use any single caller's context. 42 serves the
// catalogue slowly: about 500 projects take 40s or more.
const fetchTimeout = 3 * time.Minute

// searchWait is how long a search waits for a catalogue fetch in progress
// before answering ErrLoading (the fetch goes on in the background).
const searchWait = 10 * time.Second

// ErrLoading means the catalogue is still being fetched from 42.
var ErrLoading = errors.New("the project list is still loading from 42, try again in a few seconds")

// Project is a 42cursus project.
type Project struct {
	ID   curriculum.ProjectID
	Name string
	Slug string
	Exam bool
	// Child: a sub-project of another project (e.g. an apprenticeship step).
	Child bool
}

// Catalog fetches the 42cursus projects.
type Catalog interface {
	CursusProjects(ctx context.Context, accessToken string) ([]Project, error)
}

// AppTokenSource provides a 42 application token.
type AppTokenSource interface {
	AppToken(ctx context.Context) (string, error)
}

// Result is a matching project and the bachelor modules it counts for (none
// if it is outside the bachelor).
type Result struct {
	Project Project
	Modules []curriculum.Module
}

// Service runs the project search.
type Service struct {
	catalog    Catalog
	tokens     AppTokenSource
	modules    map[curriculum.ProjectID][]curriculum.Module
	log        *slog.Logger
	now        func() time.Time
	searchWait time.Duration

	mu       sync.Mutex
	projects []Project
	loadedAt time.Time
	loads    singleflight.Group
}

func NewService(catalog Catalog, tokens AppTokenSource, c curriculum.Curriculum, log *slog.Logger) *Service {
	modules := map[curriculum.ProjectID][]curriculum.Module{}
	for _, m := range c.Modules() {
		for _, id := range m.ProjectIDs {
			modules[id] = append(modules[id], m)
		}
	}
	return &Service{catalog: catalog, tokens: tokens, modules: modules, log: log, now: time.Now,
		searchWait: searchWait}
}

// Warm starts fetching the catalogue in the background, e.g. at startup, so
// the first search does not wait for it.
func (s *Service) Warm() {
	s.load()
}

// Search returns up to limit projects whose name or slug contains query
// (case-insensitive), best matches first: exact name, name prefix, word
// prefix, then anywhere. Exams and sub-projects outside the bachelor are
// left out.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, nil
	}
	all, err := s.catalogue(ctx)
	if err != nil {
		return nil, err
	}

	type match struct {
		Result
		rank int
	}
	var matches []match
	for _, p := range all {
		mods := s.modules[p.ID]
		if p.Exam || (p.Child && len(mods) == 0) {
			continue
		}
		if rank, ok := matchRank(q, p); ok {
			matches = append(matches, match{Result{Project: p, Modules: mods}, rank})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].rank != matches[j].rank {
			return matches[i].rank < matches[j].rank
		}
		return strings.ToLower(matches[i].Project.Name) < strings.ToLower(matches[j].Project.Name)
	})

	results := make([]Result, 0, min(limit, len(matches)))
	for _, m := range matches[:min(limit, len(matches))] {
		results = append(results, m.Result)
	}
	return results, nil
}

// matchRank scores how well q matches p; lower is better.
func matchRank(q string, p Project) (int, bool) {
	name := strings.ToLower(p.Name)
	switch {
	case name == q:
		return 0, true
	case strings.HasPrefix(name, q):
		return 1, true
	case strings.Contains(" "+strings.NewReplacer("_", " ", "-", " ").Replace(name), " "+q):
		return 2, true
	case strings.Contains(name, q), strings.Contains(strings.ToLower(p.Slug), q):
		return 3, true
	}
	return 0, false
}

// catalogue returns the cached projects. When they are older than
// catalogTTL, the old list is returned while a new one is fetched in the
// background. Without any list it waits up to searchWait for the fetch, then
// returns ErrLoading.
func (s *Service) catalogue(ctx context.Context) ([]Project, error) {
	s.mu.Lock()
	projects, loadedAt := s.projects, s.loadedAt
	s.mu.Unlock()
	if projects != nil {
		if s.now().Sub(loadedAt) >= catalogTTL {
			s.load()
		}
		return projects, nil
	}

	ch := s.load()
	wait := time.NewTimer(s.searchWait)
	defer wait.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-wait.C:
		return nil, ErrLoading
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		return res.Val.([]Project), nil
	}
}

// load fetches the catalogue unless a fetch is already running; concurrent
// callers share one fetch.
func (s *Service) load() <-chan singleflight.Result {
	return s.loads.DoChan("catalogue", func() (any, error) {
		ctx, cancel := context.WithTimeout(upstream.Background(context.Background()), fetchTimeout)
		defer cancel()
		start := s.now()
		token, err := s.tokens.AppToken(ctx)
		if err != nil {
			s.log.Warn("fetching the project catalogue failed", "err", err)
			return nil, err
		}
		fetched, err := s.catalog.CursusProjects(ctx, token)
		if err != nil {
			s.log.Warn("fetching the project catalogue failed", "err", err)
			return nil, err
		}
		s.mu.Lock()
		s.projects, s.loadedAt = fetched, s.now()
		s.mu.Unlock()
		s.log.Info("project catalogue fetched", "projects", len(fetched), "duration", s.now().Sub(start).String())
		return fetched, nil
	})
}
