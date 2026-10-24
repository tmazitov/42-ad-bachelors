// Package student holds the Student aggregate: a 42 user who signed in to the app.
package student

import (
	"context"
	"errors"
)

// ID is the 42 intra user id.
type ID int

// Student is a 42 user registered in the app.
type Student struct {
	ID          ID
	Login       string
	DisplayName string
	ImageURL    string // may be empty
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

// Repository persists students.
type Repository interface {
	// Save inserts the student or updates the stored one with the same ID.
	Save(ctx context.Context, s Student) error
}
