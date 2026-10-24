package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/progress"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
)

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

type leaderboardResponse struct {
	UpdatedAt *time.Time         `json:"updated_at"`
	Students  []leaderboardEntry `json:"students"`
}

// Leaderboard serves all synced students ranked by credits.
func Leaderboard(svc *progress.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		board, err := svc.Leaderboard(c.Context())
		if err != nil {
			return err
		}

		resp := leaderboardResponse{Students: make([]leaderboardEntry, len(board.Entries))}
		if !board.UpdatedAt.IsZero() {
			resp.UpdatedAt = &board.UpdatedAt
		}
		for i, e := range board.Entries {
			var image *string
			if e.Student.ImageURL != "" {
				image = &e.Student.ImageURL
			}
			resp.Students[i] = leaderboardEntry{
				Rank:        e.Rank,
				Login:       e.Student.Login,
				DisplayName: e.Student.DisplayName,
				Image:       image,
				General:     e.Progress.Credits[curriculum.General],
				Standard:    e.Progress.Credits[curriculum.Standard],
				Stream:      e.Progress.Credits[curriculum.Stream],
				Elective:    e.Progress.Credits[curriculum.Elective],
				Total:       e.Progress.Total,
				Percentage:  e.Progress.Percentage,
				SyncedAt:    e.Student.ProjectsSyncedAt,
			}
		}

		c.Set(fiber.HeaderCacheControl, "no-cache")
		return c.JSON(resp)
	}
}
