package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
)

type moduleResponse struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Credits    int    `json:"credits"`
	ProjectIDs []int  `json:"project_ids"`
}

type categoryResponse struct {
	Key             string           `json:"key"`
	RequiredCredits int              `json:"required_credits"`
	Modules         []moduleResponse `json:"modules"`
}

type curriculumResponse struct {
	TotalRequiredCredits int                `json:"total_required_credits"`
	Categories           []categoryResponse `json:"categories"`
}

// Curriculum serves the degree's modules grouped by category. The response is
// static, so it is built once.
func Curriculum(c curriculum.Curriculum) fiber.Handler {
	byCategory := map[curriculum.Category][]moduleResponse{}
	for _, m := range c.Modules() {
		ids := make([]int, len(m.ProjectIDs))
		for i, id := range m.ProjectIDs {
			ids[i] = int(id)
		}
		byCategory[m.Category] = append(byCategory[m.Category], moduleResponse{
			Code:       m.Code,
			Name:       m.Name,
			Credits:    m.Credits,
			ProjectIDs: ids,
		})
	}

	resp := curriculumResponse{TotalRequiredCredits: curriculum.TotalRequiredCredits}
	for _, cat := range curriculum.Categories {
		resp.Categories = append(resp.Categories, categoryResponse{
			Key:             string(cat),
			RequiredCredits: curriculum.RequiredCredits[cat],
			Modules:         byCategory[cat],
		})
	}

	return func(ctx fiber.Ctx) error {
		ctx.Set(fiber.HeaderCacheControl, "public, max-age=3600")
		return ctx.JSON(resp)
	}
}
