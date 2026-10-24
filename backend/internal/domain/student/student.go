// Package student holds the Student aggregate: a 42 user who signed in to the app.
package student

import (
	"context"
	"errors"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
)

// ID is the 42 intra user id.
type ID int

// CompletedProject is a 42 project the student has validated.
type CompletedProject struct {
	ProjectID curriculum.ProjectID
	FinalMark int
	MarkedAt  time.Time // zero if unknown
}

// Student is a 42 user registered in the app.
type Student struct {
	ID          ID
	Login       string
	DisplayName string
	ImageURL    string // may be empty

	// Projects is set when loaded by ListSynced. ProjectsSyncedAt is zero
	// until the first sync.
	Projects         []CompletedProject
	ProjectsSyncedAt time.Time
}

// New validates and builds a Student.
func New(id ID, login, displayName, imageURL string) (Student, error) {
	if id <= 0 {
		return Student{}, errors.New("student: id must be positive")
	}
	if login == "" {
		return Student{}, errors.New("student: login is required")
	}
	if displayName == "" {
		displayName = login
	}
	return Student{ID: id, Login: login, DisplayName: displayName, ImageURL: imageURL}, nil
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
	// Save inserts the student's profile or updates the stored one with the same ID.
	Save(ctx context.Context, s Student) error
	// ReplaceProjects stores projects as the student's full list of validated
	// projects, synced at syncedAt.
	ReplaceProjects(ctx context.Context, id ID, projects []CompletedProject, syncedAt time.Time) error
	// List returns every registered student with ProjectsSyncedAt (no Projects).
	List(ctx context.Context) ([]Student, error)
	// ListSynced returns every student whose projects were synced at least
	// once, with Projects and ProjectsSyncedAt populated.
	ListSynced(ctx context.Context) ([]Student, error)
}
