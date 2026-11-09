package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/projects"
)

const projectSearchLimit = 15

type projectModuleResponse struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type projectResultResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	// Modules are the bachelor modules the project counts for; empty when it
	// is outside the bachelor.
	Modules []projectModuleResponse `json:"modules"`
}

// SearchProjects finds 42cursus projects by name: GET /projects?search=.
func SearchProjects(svc *projects.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		results, err := svc.Search(c.Context(), c.Query("search"), projectSearchLimit)
		if err != nil {
			return err
		}
		resp := make([]projectResultResponse, len(results))
		for i, r := range results {
			mods := make([]projectModuleResponse, len(r.Modules))
			for j, m := range r.Modules {
				mods[j] = projectModuleResponse{Code: m.Code, Name: m.Name, Category: string(m.Category)}
			}
			resp[i] = projectResultResponse{ID: int(r.Project.ID), Name: r.Project.Name, Slug: r.Project.Slug, Modules: mods}
		}
		return c.JSON(fiber.Map{"projects": resp})
	}
}
