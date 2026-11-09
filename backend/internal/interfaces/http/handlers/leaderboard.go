package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/progress"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/leaderboard"
)

const leaderboardPageSize = 10

type leaderboardEntry struct {
	Rank        int       `json:"rank"`
	Login       string    `json:"login"`
	DisplayName string    `json:"display_name"`
	Image       *string   `json:"image"`
	General     int       `json:"general"`
	Standard    int       `json:"standard"`
	Stream      int       `json:"stream"`
	Elective    int       `json:"elective"`
	Total       int       `json:"total"`
	Percentage  float64   `json:"percentage"`
	SyncedAt    time.Time `json:"synced_at"`
}

type myRankResponse struct {
	leaderboardEntry
	// Page holds the viewer in the unfiltered leaderboard.
	Page int `json:"page"`
	// Percentile is the share of other students ranked below; null when
	// the viewer is the only one.
	Percentile *int `json:"percentile"`
}

type leaderboardResponse struct {
	UpdatedAt *time.Time `json:"updated_at"`
	// TotalStudents and Average cover the whole board, not only matches.
	TotalStudents int                `json:"total_students"`
	Average       float64            `json:"average"`
	Students      []leaderboardEntry `json:"students"` // this page
	Total         int                `json:"total"`    // students matching the search
	Page          int                `json:"page"`
	PageSize      int                `json:"page_size"`
	Me            *myRankResponse    `json:"me"`
}

// Leaderboard serves one page of the ranked synced students:
// GET /leaderboard?page=&search=. The ranking is cached for 2 minutes.
func Leaderboard(svc *progress.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		page := max(fiber.Query[int](c, "page"), 1)
		board, err := svc.LeaderboardPage(c.Context(), c.Query("search"), page, leaderboardPageSize, SignedIn(c))
		if err != nil {
			return err
		}

		resp := leaderboardResponse{
			TotalStudents: board.Students,
			Average:       board.Average,
			Students:      make([]leaderboardEntry, len(board.Entries)),
			Total:         board.Matches,
			Page:          board.Page,
			PageSize:      board.PageSize,
		}
		if !board.UpdatedAt.IsZero() {
			resp.UpdatedAt = &board.UpdatedAt
		}
		for i, e := range board.Entries {
			resp.Students[i] = toLeaderboardEntry(e)
		}
		if me := board.Me; me != nil {
			resp.Me = &myRankResponse{leaderboardEntry: toLeaderboardEntry(me.Entry), Page: me.Page}
			if me.Percentile >= 0 {
				resp.Me.Percentile = &me.Percentile
			}
		}

		c.Set(fiber.HeaderCacheControl, "no-cache")
		return c.JSON(resp)
	}
}

func toLeaderboardEntry(e leaderboard.Entry) leaderboardEntry {
	return leaderboardEntry{
		Rank:        e.Rank,
		Login:       e.Student.Login,
		DisplayName: e.Student.DisplayName,
		Image:       optional(e.Student.ImageURL),
		General:     e.Progress.Credits[curriculum.General],
		Standard:    e.Progress.Credits[curriculum.Standard],
		Stream:      e.Progress.Credits[curriculum.Stream],
		Elective:    e.Progress.Credits[curriculum.Elective],
		Total:       e.Progress.Total,
		Percentage:  e.Progress.Percentage,
		SyncedAt:    e.Student.ProjectsSyncedAt,
	}
}
