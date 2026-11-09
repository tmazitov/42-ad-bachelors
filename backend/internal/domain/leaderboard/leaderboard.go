// Package leaderboard ranks students by their bachelor progress.
package leaderboard

import (
	"sort"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

// Entry is one ranked student.
type Entry struct {
	Rank     int
	Student  student.Student
	Progress curriculum.Progress
}

// Rank evaluates each student against c and orders them by total credits,
// highest first. Students with equal credits share a rank (1, 2, 2, 4) and are
// ordered by login.
func Rank(c curriculum.Curriculum, students []student.Student) []Entry {
	entries := make([]Entry, len(students))
	for i, s := range students {
		entries[i] = Entry{Student: s, Progress: c.Evaluate(s.Validated(), s.Marked(), s.Stream)}
	}

	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Progress.Total != b.Progress.Total {
			return a.Progress.Total > b.Progress.Total
		}
		return a.Student.Login < b.Student.Login
	})

	for i := range entries {
		if i > 0 && entries[i].Progress.Total == entries[i-1].Progress.Total {
			entries[i].Rank = entries[i-1].Rank
		} else {
			entries[i].Rank = i + 1
		}
	}
	return entries
}
