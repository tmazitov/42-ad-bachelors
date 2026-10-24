// Package student holds the Student aggregate: a 42 user known to the app.
package student

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
)

// ErrNotFound means no student with the given ID or login exists.
var ErrNotFound = errors.New("student not found")

// ID is the 42 intra user id.
type ID int

// CompletedProject is a 42 project the student has validated.
type CompletedProject struct {
	ProjectID curriculum.ProjectID
	Slug      string // intra slug, for links; may be empty
	FinalMark int
	MarkedAt  time.Time // zero if unknown
}

// Student is a 42 user stored by the app: either registered (signed in at
// least once) or only looked up by someone viewing their profile.
type Student struct {
	ID          ID
	Login       string
	DisplayName string
	ImageURL    string // may be empty
	Staff       bool
	Alumni      bool

	// RegisteredAt is when the student first signed in; zero for students
	// who were only viewed. Only registered students are ranked, searchable
	// and synced in the background.
	RegisteredAt time.Time

	// Projects is set when loaded by Find* or ListSynced. ProjectsSyncedAt
	// is zero until the first sync.
	Projects         []CompletedProject
	ProjectsSyncedAt time.Time
}

// New validates and builds a Student.
func New(id ID, login, displayName, imageURL string) (Student, error) {
	if id <= 0 {
		return Student{}, errors.New("student: id must be positive")
	}
	if !ValidLogin(login) {
		return Student{}, errors.New("student: invalid login")
	}
	if displayName == "" {
		displayName = login
	}
	return Student{ID: id, Login: login, DisplayName: displayName, ImageURL: imageURL}, nil
}

var loginPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

// ValidLogin reports whether s looks like a 42 login.
func ValidLogin(s string) bool {
	return loginPattern.MatchString(s)
}

// Registered reports whether the student has signed in to the app.
func (s Student) Registered() bool {
	return !s.RegisteredAt.IsZero()
}

// NeedsSync reports whether the student's projects were never synced or were
// last synced at least maxAge before now.
func (s Student) NeedsSync(now time.Time, maxAge time.Duration) bool {
	return s.ProjectsSyncedAt.IsZero() || now.Sub(s.ProjectsSyncedAt) >= maxAge
}

// Validated returns the set of the student's validated project IDs.
func (s Student) Validated() curriculum.ProjectSet {
	set := make(curriculum.ProjectSet, len(s.Projects))
	for _, p := range s.Projects {
		set[p.ProjectID] = struct{}{}
	}
	return set
}

// Repository persists students.
type Repository interface {
	// Save inserts the student's profile or updates the stored one with the
	// same ID. A non-zero RegisteredAt marks the student as registered; a
	// zero one never unregisters an existing student.
	Save(ctx context.Context, s Student) error
	// ReplaceProjects stores projects as the student's full list of validated
	// projects, synced at syncedAt.
	ReplaceProjects(ctx context.Context, id ID, projects []CompletedProject, syncedAt time.Time) error
	// FindByID and FindByLogin return the student with Projects, or ErrNotFound.
	FindByID(ctx context.Context, id ID) (Student, error)
	FindByLogin(ctx context.Context, login string) (Student, error)
	// Search returns registered students whose login or display name contains
	// query, ordered by login, and the total number of matches.
	Search(ctx context.Context, query string, limit, offset int) ([]Student, int, error)
	// ListRegistered returns every registered student with ProjectsSyncedAt
	// (no Projects).
	ListRegistered(ctx context.Context) ([]Student, error)
	// ListSynced returns every registered student whose projects were synced
	// at least once, with Projects and ProjectsSyncedAt populated.
	ListSynced(ctx context.Context) ([]Student, error)
}
