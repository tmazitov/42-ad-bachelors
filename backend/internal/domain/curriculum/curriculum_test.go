package curriculum

import "testing"

func TestADBachelorIsValid(t *testing.T) {
	// ADBachelor panics on invalid data.
	if n := len(ADBachelor().Modules()); n == 0 {
		t.Fatal("curriculum has no modules")
	}
}

func TestEvaluate(t *testing.T) {
	c := ADBachelor()

	tests := []struct {
		name      string
		validated ProjectSet
		want      map[Category]int
		total     int
	}{
		{
			name:      "nothing validated",
			validated: NewProjectSet(),
			want:      map[Category]int{},
		},
		{
			name:      "alternative project ID counts once",
			validated: NewProjectSet(2724, 2005), // Minitalk and Pipex
			want:      map[Category]int{Standard: 3},
			total:     3,
		},
		{
			name:      "ft_transcendence counts for CCP 161 and BAI 498",
			validated: NewProjectSet(1337),
			want:      map[Category]int{Standard: 8},
			total:     8,
		},
		{
			name: "category credits are capped",
			// 2+3+3+3+3+3+3+3+3 = 26 elective credits, capped at 12
			validated: NewProjectSet(1418, 1385, 1393, 1470, 1403, 1384, 2485, 2600, 1414),
			want:      map[Category]int{Elective: 12},
			total:     12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Evaluate(tt.validated)
			for _, cat := range Categories {
				if got.Credits[cat] != tt.want[cat] {
					t.Errorf("%s credits = %d, want %d", cat, got.Credits[cat], tt.want[cat])
				}
			}
			if got.Total != tt.total {
				t.Errorf("total = %d, want %d", got.Total, tt.total)
			}
		})
	}
}

func TestEvaluatePercentage(t *testing.T) {
	got := ADBachelor().Evaluate(NewProjectSet(1314)) // Libft, 4 credits
	if want := 3.0; got.Percentage != want {          // 4/132 = 3.03 → 3.0
		t.Errorf("percentage = %v, want %v", got.Percentage, want)
	}
}
