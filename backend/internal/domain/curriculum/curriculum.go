package curriculum

import (
	"fmt"
	"math"
)

// StreamChoice is a themed set of stream modules. A student picks one stream, and
// only its modules count toward the stream credits; credits from different
// streams do not add up. A module may belong to several streams.
type StreamChoice struct {
	Key     string
	Name    string
	Modules []string // module codes
}

// ModuleSet is a set of module codes.
type ModuleSet map[string]bool

// Curriculum is the full list of modules of a degree.
type Curriculum struct {
	modules []Module
	streams []StreamChoice
	// selfReported modules can be marked completed by the student: their
	// results are recorded outside 42 (e.g. by the Learning Team on ATRIUM).
	selfReported ModuleSet
}

// New validates modules, streams and self-reported module codes and builds a
// Curriculum. Every stream module must belong to at least one stream, and
// streams may only list stream modules.
func New(modules []Module, streams []StreamChoice, selfReported []string) (Curriculum, error) {
	byCode := make(map[string]Module, len(modules))
	for _, m := range modules {
		if _, ok := RequiredCredits[m.Category]; !ok {
			return Curriculum{}, fmt.Errorf("module %s: unknown category %q", m.Code, m.Category)
		}
		if m.Credits <= 0 {
			return Curriculum{}, fmt.Errorf("module %s: credits must be positive", m.Code)
		}
		if _, dup := byCode[m.Code]; dup {
			return Curriculum{}, fmt.Errorf("module %s: duplicate code", m.Code)
		}
		byCode[m.Code] = m
	}

	inStream := map[string]bool{}
	keys := map[string]bool{}
	for _, s := range streams {
		if s.Key == "" || keys[s.Key] {
			return Curriculum{}, fmt.Errorf("stream %q: empty or duplicate key", s.Key)
		}
		keys[s.Key] = true
		for _, code := range s.Modules {
			m, ok := byCode[code]
			if !ok {
				return Curriculum{}, fmt.Errorf("stream %s: unknown module %s", s.Key, code)
			}
			if m.Category != Stream {
				return Curriculum{}, fmt.Errorf("stream %s: module %s is not a stream module", s.Key, code)
			}
			inStream[code] = true
		}
	}
	for _, m := range modules {
		if m.Category == Stream && !inStream[m.Code] {
			return Curriculum{}, fmt.Errorf("module %s: stream module in no stream", m.Code)
		}
	}

	self := make(ModuleSet, len(selfReported))
	for _, code := range selfReported {
		if _, ok := byCode[code]; !ok {
			return Curriculum{}, fmt.Errorf("self-reported module %s: unknown module", code)
		}
		self[code] = true
	}

	return Curriculum{modules: modules, streams: streams, selfReported: self}, nil
}

// SelfReported reports whether the student may mark the module completed.
func (c Curriculum) SelfReported(code string) bool {
	return c.selfReported[code]
}

// Modules returns a copy of all modules.
func (c Curriculum) Modules() []Module {
	return append([]Module(nil), c.modules...)
}

// Streams returns a copy of all streams, in display order.
func (c Curriculum) Streams() []StreamChoice {
	return append([]StreamChoice(nil), c.streams...)
}

// HasStream reports whether key names a stream.
func (c Curriculum) HasStream(key string) bool {
	for _, s := range c.streams {
		if s.Key == key {
			return true
		}
	}
	return false
}

// Progress is a student's earned credits, capped per category.
type Progress struct {
	Credits    map[Category]int
	Total      int
	Percentage float64 // 0–100, rounded to one decimal

	// StreamCredits is the (uncapped) credits earned in each stream.
	StreamCredits map[string]int
	// Stream is the stream counted for Credits[Stream]: the chosen one, or
	// the best one when the student has not chosen yet.
	Stream       string
	StreamChosen bool
}

// Evaluate computes progress for a student with the given validated projects,
// modules they marked completed themselves (only self-reported modules count)
// and chosen stream (empty if not chosen yet).
func (c Curriculum) Evaluate(validated ProjectSet, marked ModuleSet, chosenStream string) Progress {
	earned := make(map[Category]int, len(Categories))
	completed := map[string]bool{}
	for _, m := range c.modules {
		if !m.CompletedBy(validated) && !(c.selfReported[m.Code] && marked[m.Code]) {
			continue
		}
		completed[m.Code] = true
		if m.Category != Stream {
			earned[m.Category] += m.Credits
		}
	}

	p := Progress{
		Credits:       make(map[Category]int, len(Categories)),
		StreamCredits: make(map[string]int, len(c.streams)),
	}
	credits := map[string]int{}
	for _, m := range c.modules {
		credits[m.Code] = m.Credits
	}
	for _, s := range c.streams {
		p.StreamCredits[s.Key] = 0 // list every stream, even without credits
		for _, code := range s.Modules {
			if completed[code] {
				p.StreamCredits[s.Key] += credits[code]
			}
		}
	}

	if c.HasStream(chosenStream) {
		p.Stream, p.StreamChosen = chosenStream, true
	} else {
		// Not chosen: count the best stream; ties go to the first one.
		for _, s := range c.streams {
			if p.Stream == "" || p.StreamCredits[s.Key] > p.StreamCredits[p.Stream] {
				p.Stream = s.Key
			}
		}
	}
	earned[Stream] = p.StreamCredits[p.Stream]

	for _, cat := range Categories {
		credits := min(earned[cat], RequiredCredits[cat])
		p.Credits[cat] = credits
		p.Total += credits
	}
	p.Percentage = math.Round(float64(p.Total)/TotalRequiredCredits*1000) / 10
	return p
}
