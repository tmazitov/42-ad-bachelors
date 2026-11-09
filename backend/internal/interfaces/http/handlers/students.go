package handlers

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/students"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

const searchPageSize = 10

type progressResponse struct {
	General    int     `json:"general"`
	Standard   int     `json:"standard"`
	Stream     int     `json:"stream"`
	Elective   int     `json:"elective"`
	Total      int     `json:"total"`
	Percentage float64 `json:"percentage"`
	// StreamKey is the stream counted in Stream: the chosen one, or the best
	// one if none is chosen. Streams has the credits earned in every stream.
	StreamKey string         `json:"stream_key"`
	Streams   map[string]int `json:"streams"`
}

type projectResponse struct {
	ProjectID int        `json:"project_id"`
	Slug      string     `json:"slug,omitempty"`
	FinalMark int        `json:"final_mark"`
	MarkedAt  *time.Time `json:"marked_at"`
}

type studentSummary struct {
	Login       string  `json:"login"`
	DisplayName string  `json:"display_name"`
	Image       *string `json:"image"`
}

type studentResponse struct {
	ID int `json:"id"`
	studentSummary
	Staff  bool    `json:"staff"`
	Alumni bool    `json:"alumni"`
	Stream *string `json:"stream"` // chosen stream key, null if not chosen
	// MarkedModules are module codes the student marked completed themselves.
	MarkedModules []string          `json:"marked_modules"`
	Registered    bool              `json:"registered"`
	SyncedAt      *time.Time        `json:"synced_at"`
	Progress      progressResponse  `json:"progress"`
	Projects      []projectResponse `json:"projects"`
}

// Me returns the signed-in student with progress and validated projects.
func Me(svc *students.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		v, err := svc.ByID(c.Context(), SignedIn(c))
		if errors.Is(err, student.ErrNotFound) {
			return fiber.NewError(fiber.StatusUnauthorized, "not signed in")
		}
		if err != nil {
			return err
		}
		return c.JSON(toStudentResponse(v))
	}
}

// Student returns any 42 student by login, fetching them from 42 if needed.
func Student(svc *students.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		v, err := svc.ByLogin(c.Context(), c.Params("login"))
		switch {
		case errors.Is(err, students.ErrInvalidLogin):
			return fiber.NewError(fiber.StatusBadRequest, "invalid login")
		case errors.Is(err, student.ErrNotFound):
			return fiber.NewError(fiber.StatusNotFound, "student not found")
		case err != nil:
			return err
		}
		return c.JSON(toStudentResponse(v))
	}
}

// SetMyStream stores the signed-in student's stream:
// PUT /me/stream {"stream": "<key>"}.
func SetMyStream(svc *students.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			Stream string `json:"stream"`
		}
		if err := c.Bind().JSON(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body")
		}
		v, err := svc.SetStream(c.Context(), SignedIn(c), req.Stream)
		switch {
		case errors.Is(err, students.ErrUnknownStream):
			return fiber.NewError(fiber.StatusBadRequest, "unknown stream")
		case errors.Is(err, student.ErrNotFound):
			return fiber.NewError(fiber.StatusUnauthorized, "not signed in")
		case err != nil:
			return err
		}
		return c.JSON(toStudentResponse(v))
	}
}

// MarkMyModule marks a self-reported module completed or not for the signed-in
// student: PUT /me/modules {"code": "GEN 101", "completed": true}.
func MarkMyModule(svc *students.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			Code      string `json:"code"`
			Completed bool   `json:"completed"`
		}
		if err := c.Bind().JSON(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body")
		}
		v, err := svc.MarkModule(c.Context(), SignedIn(c), req.Code, req.Completed)
		switch {
		case errors.Is(err, students.ErrNotSelfReported):
			return fiber.NewError(fiber.StatusBadRequest, "this module cannot be marked by the student")
		case errors.Is(err, student.ErrNotFound):
			return fiber.NewError(fiber.StatusUnauthorized, "not signed in")
		case err != nil:
			return err
		}
		return c.JSON(toStudentResponse(v))
	}
}

type searchResponse struct {
	Students []studentSummary `json:"students"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// SearchStudents finds registered students: GET /students?search=&page=.
func SearchStudents(svc *students.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		page := max(fiber.Query[int](c, "page"), 1)
		res, err := svc.Search(c.Context(), c.Query("search"), page, searchPageSize)
		if err != nil {
			return err
		}
		resp := searchResponse{Students: make([]studentSummary, len(res.Students)), Total: res.Total,
			Page: page, PageSize: searchPageSize}
		for i, s := range res.Students {
			resp.Students[i] = toSummary(s)
		}
		return c.JSON(resp)
	}
}

func toSummary(s student.Student) studentSummary {
	return studentSummary{Login: s.Login, DisplayName: s.DisplayName, Image: optional(s.ImageURL)}
}

func toStudentResponse(v students.View) studentResponse {
	s := v.Student
	resp := studentResponse{
		ID:             int(s.ID),
		studentSummary: toSummary(s),
		Staff:          s.Staff,
		Alumni:         s.Alumni,
		Stream:         optional(s.Stream),
		MarkedModules:  append([]string{}, s.MarkedModules...),
		Registered:     s.Registered(),
		SyncedAt:       optionalTime(s.ProjectsSyncedAt),
		Progress:       toProgress(v.Progress),
		Projects:       make([]projectResponse, len(s.Projects)),
	}
	for i, p := range s.Projects {
		resp.Projects[i] = projectResponse{
			ProjectID: int(p.ProjectID),
			Slug:      p.Slug,
			FinalMark: p.FinalMark,
			MarkedAt:  optionalTime(p.MarkedAt),
		}
	}
	return resp
}

func toProgress(p curriculum.Progress) progressResponse {
	return progressResponse{
		General:    p.Credits[curriculum.General],
		Standard:   p.Credits[curriculum.Standard],
		Stream:     p.Credits[curriculum.Stream],
		Elective:   p.Credits[curriculum.Elective],
		Total:      p.Total,
		Percentage: p.Percentage,
		StreamKey:  p.Stream,
		Streams:    p.StreamCredits,
	}
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
