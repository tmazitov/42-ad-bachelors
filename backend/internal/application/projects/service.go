// Package projects searches the 42cursus project catalogue and tells which
// bachelor modules each project counts for. The catalogue is fetched from 42
// once and kept in memory, so searching costs no API calls.
package projects

import (
	"context"
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

// fetchTimeout bounds a catalogue fetch (several pages), which is shared by
// concurrent searches and so does not use any single caller's context.
const fetchTimeout = 30 * time.Second

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
	catalog Catalog
	tokens  AppTokenSource
	modules map[curriculum.ProjectID][]curriculum.Module
	now     func() time.Time

	mu       sync.Mutex
	projects []Project
	loadedAt time.Time
	loads    singleflight.Group
}

func NewService(catalog Catalog, tokens AppTokenSource, c curriculum.Curriculum) *Service {
	modules := map[curriculum.ProjectID][]curriculum.Module{}
	for _, m := range c.Modules() {
		for _, id := range m.ProjectIDs {
			modules[id] = append(modules[id], m)
		}
	}
	return &Service{catalog: catalog, tokens: tokens, modules: modules, now: time.Now}
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

// catalogue returns the cached projects, fetching them when missing or older
// than catalogTTL. If a refetch fails, the old list is used.
func (s *Service) catalogue(ctx context.Context) ([]Project, error) {
	s.mu.Lock()
	projects, loadedAt := s.projects, s.loadedAt
	s.mu.Unlock()
	if projects != nil && s.now().Sub(loadedAt) < catalogTTL {
		return projects, nil
	}

	ch := s.loads.DoChan("catalogue", func() (any, error) {
		ctx, cancel := context.WithTimeout(upstream.Background(context.WithoutCancel(ctx)), fetchTimeout)
		defer cancel()
		token, err := s.tokens.AppToken(ctx)
		if err != nil {
			return nil, err
		}
		fetched, err := s.catalog.CursusProjects(ctx, token)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		s.projects, s.loadedAt = fetched, s.now()
		s.mu.Unlock()
		return fetched, nil
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			if projects != nil {
				return projects, nil // stale, but better than nothing
			}
			return nil, res.Err
		}
		return res.Val.([]Project), nil
	}
}
