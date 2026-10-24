package curriculum

// Category groups modules; each category has its own credit cap.
type Category string

const (
	General  Category = "general"
	Standard Category = "standard"
	Stream   Category = "stream"
	Elective Category = "elective"
)

// Categories lists all categories in display order.
var Categories = []Category{General, Standard, Stream, Elective}

// RequiredCredits is the number of credits needed per category. Credits
// earned above the requirement do not count toward the total.
var RequiredCredits = map[Category]int{
	General:  5,
	Standard: 101,
	Stream:   14,
	Elective: 12,
}

// TotalRequiredCredits is the number of credits needed to graduate.
const TotalRequiredCredits = 132
