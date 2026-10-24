package curriculum

import (
	"fmt"
	"math"
)

// Curriculum is the full list of modules of a degree.
type Curriculum struct {
	modules []Module
}

// New validates modules and builds a Curriculum.
func New(modules []Module) (Curriculum, error) {
	codes := make(map[string]struct{}, len(modules))
	for _, m := range modules {
		if _, ok := RequiredCredits[m.Category]; !ok {
			return Curriculum{}, fmt.Errorf("module %s: unknown category %q", m.Code, m.Category)
		}
		if m.Credits <= 0 {
			return Curriculum{}, fmt.Errorf("module %s: credits must be positive", m.Code)
		}
		if _, dup := codes[m.Code]; dup {
			return Curriculum{}, fmt.Errorf("module %s: duplicate code", m.Code)
		}
		codes[m.Code] = struct{}{}
	}
	return Curriculum{modules: modules}, nil
}

// Modules returns a copy of all modules.
func (c Curriculum) Modules() []Module {
	return append([]Module(nil), c.modules...)
}

// Progress is a student's earned credits, capped per category.
type Progress struct {
	Credits    map[Category]int
	Total      int
	Percentage float64 // 0–100, rounded to one decimal
}

// Evaluate computes progress for a student with the given validated projects.
func (c Curriculum) Evaluate(validated ProjectSet) Progress {
	earned := make(map[Category]int, len(Categories))
	for _, m := range c.modules {
		if m.CompletedBy(validated) {
			earned[m.Category] += m.Credits
		}
	}

	p := Progress{Credits: make(map[Category]int, len(Categories))}
	for _, cat := range Categories {
		credits := min(earned[cat], RequiredCredits[cat])
		p.Credits[cat] = credits
		p.Total += credits
	}
	p.Percentage = math.Round(float64(p.Total)/TotalRequiredCredits*1000) / 10
	return p
}
