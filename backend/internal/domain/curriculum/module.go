package curriculum

// ProjectID is a 42 intra project identifier.
type ProjectID int

// ProjectSet is a set of validated 42 projects of a student.
type ProjectSet map[ProjectID]struct{}

// NewProjectSet builds a ProjectSet from a list of IDs.
func NewProjectSet(ids ...ProjectID) ProjectSet {
	set := make(ProjectSet, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

// Has reports whether the set contains id.
func (s ProjectSet) Has(id ProjectID) bool {
	_, ok := s[id]
	return ok
}

// Team tells whether a module's project is done alone or in a group on intra.
type Team string

const (
	TeamUnknown Team = "" // not tracked through the API
	Solo        Team = "solo"
	Group       Team = "group"
	// Mixed: the module's alternative projects, or a project's sessions, differ.
	Mixed Team = "mixed"
)

// Module is a bachelor course worth a number of credits. It is completed when
// any of its ProjectIDs is validated (alternatives such as "Minitalk or Pipex",
// or old and new versions of the same project). A module without ProjectIDs
// cannot be tracked through the 42 API (e.g. seminars, the capstone project).
type Module struct {
	Code       string
	Name       string
	Credits    int
	Category   Category
	ProjectIDs []ProjectID
	Team       Team
}

// Trackable reports whether completion can be derived from 42 projects.
func (m Module) Trackable() bool {
	return len(m.ProjectIDs) > 0
}

// CompletedBy reports whether any of the module's projects is in validated.
func (m Module) CompletedBy(validated ProjectSet) bool {
	for _, id := range m.ProjectIDs {
		if validated.Has(id) {
			return true
		}
	}
	return false
}
