package leaderboard

import (
	"testing"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

func withProjects(login string, ids ...curriculum.ProjectID) student.Student {
	s := student.Student{Login: login}
	for _, id := range ids {
		s.Projects = append(s.Projects, student.CompletedProject{ProjectID: id})
	}
	return s
}

func TestRank(t *testing.T) {
	entries := Rank(curriculum.ADBachelor(), []student.Student{
		withProjects("carol"),            // 0
		withProjects("bob", 1314),        // 4
		withProjects("alice", 1314),      // 4
		withProjects("dave", 1314, 1327), // 7
	})

	want := []struct {
		login string
		rank  int
		total int
	}{
		{"dave", 1, 7},
		{"alice", 2, 4},
		{"bob", 2, 4},
		{"carol", 4, 0},
	}
	for i, w := range want {
		e := entries[i]
		if e.Student.Login != w.login || e.Rank != w.rank || e.Progress.Total != w.total {
			t.Errorf("#%d = {%s rank %d total %d}, want {%s rank %d total %d}",
				i, e.Student.Login, e.Rank, e.Progress.Total, w.login, w.rank, w.total)
		}
	}
}

func TestRankUsesEachStudentsStream(t *testing.T) {
	// Both have camagru (mobile-web, 2) and ft_ping + ft_traceroute (network-security, 4).
	both := []curriculum.ProjectID{1396, 1397, 1399}
	chose := withProjects("chose", both...)
	chose.Stream = "mobile-web"           // counts only 2
	best := withProjects("best", both...) // not chosen: best stream, 4

	entries := Rank(curriculum.ADBachelor(), []student.Student{chose, best})
	if entries[0].Student.Login != "best" || entries[0].Progress.Total != 4 || entries[1].Progress.Total != 2 {
		t.Errorf("entries = %s %d, %s %d", entries[0].Student.Login, entries[0].Progress.Total,
			entries[1].Student.Login, entries[1].Progress.Total)
	}
}
