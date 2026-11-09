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
			name:      "ft_transcendence counts for CCP 161 only, not the capstone",
			validated: NewProjectSet(1337),
			want:      map[Category]int{Standard: 4},
			total:     4,
		},
		{
			name:      "Work Experience I completes the internship",
			validated: NewProjectSet(1638),
			want:      map[Category]int{Standard: 4},
			total:     4,
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
			got := c.Evaluate(tt.validated, nil, "")
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
	got := ADBachelor().Evaluate(NewProjectSet(1314), nil, "") // Libft, 4 credits
	if want := 3.0; got.Percentage != want {                   // 4/132 = 3.03 → 3.0
		t.Errorf("percentage = %v, want %v", got.Percentage, want)
	}
}

func TestTeamSetForTrackableModules(t *testing.T) {
	for _, m := range ADBachelor().Modules() {
		if m.Trackable() && m.Team == TeamUnknown {
			t.Errorf("module %s is trackable but has no team", m.Code)
		}
		if !m.Trackable() && m.Team != TeamUnknown {
			t.Errorf("module %s is not trackable but has team %q", m.Code, m.Team)
		}
	}
}

func TestStreamCredits(t *testing.T) {
	c := ADBachelor()
	// 42run (GAM 222, 4) + camagru (WEB 101, 2) + matcha (WEB 202, 3) + Peace_Break (MOB 241, 3, in both
	// gaming-graphics and mobile-web) + ft_ping (NET 111, 2).
	validated := NewProjectSet(1387, 1396, 1401, 2552, 1397)

	tests := []struct {
		name   string
		chosen string
		stream string
		want   int
	}{
		{"chosen stream ignores the others", "network-security", "network-security", 2},
		{"shared module counts in each of its streams", "gaming-graphics", "gaming-graphics", 7},
		{"not chosen: best stream", "", "mobile-web", 8},
		{"unknown key behaves as not chosen", "nope", "mobile-web", 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := c.Evaluate(validated, nil, tt.chosen)
			if p.Stream != tt.stream || p.Credits[Stream] != tt.want {
				t.Errorf("stream = %s (%d credits), want %s (%d)", p.Stream, p.Credits[Stream], tt.stream, tt.want)
			}
			if p.StreamChosen != (tt.chosen == tt.stream) {
				t.Errorf("StreamChosen = %v", p.StreamChosen)
			}
		})
	}

	p := c.Evaluate(validated, nil, "")
	want := map[string]int{"gaming-graphics": 7, "mobile-web": 8, "kernel-systems": 0, "network-security": 2}
	for k, v := range want {
		if p.StreamCredits[k] != v {
			t.Errorf("StreamCredits[%s] = %d, want %d", k, p.StreamCredits[k], v)
		}
	}
}

func TestStreamCreditsCappedAndTieGoesToFirst(t *testing.T) {
	c := ADBachelor()
	// Nothing validated: every stream has 0, so the first stream is used.
	if p := c.Evaluate(NewProjectSet(), nil, ""); p.Stream != "gaming-graphics" || p.Credits[Stream] != 0 {
		t.Errorf("empty: stream = %s, credits = %d", p.Stream, p.Credits[Stream])
	}

	// All of Kernel and Systems: 3+3+2+2+2+3+2 = 17, capped at 14.
	all := NewProjectSet(1416, 1421, 1415, 1479, 1330, 1381, 2606)
	p := c.Evaluate(all, nil, "kernel-systems")
	if p.StreamCredits["kernel-systems"] != 17 || p.Credits[Stream] != 14 {
		t.Errorf("kernel-systems = %d uncapped, %d counted", p.StreamCredits["kernel-systems"], p.Credits[Stream])
	}
}

func TestNewValidatesStreams(t *testing.T) {
	modules := []Module{
		{"STR 1", "s1", 2, Stream, ids(1), Solo},
		{"STD 1", "std", 2, Standard, ids(2), Solo},
	}
	for name, streams := range map[string][]StreamChoice{
		"unknown module":        {{"a", "A", []string{"STR 1", "NOPE"}}},
		"non-stream module":     {{"a", "A", []string{"STR 1", "STD 1"}}},
		"stream module in none": {{"a", "A", nil}},
		"duplicate key":         {{"a", "A", []string{"STR 1"}}, {"a", "B", []string{"STR 1"}}},
	} {
		if _, err := New(modules, streams, nil); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
	if _, err := New(modules, []StreamChoice{{"a", "A", []string{"STR 1"}}}, nil); err != nil {
		t.Errorf("valid: %v", err)
	}
}

func TestSelfReportedModules(t *testing.T) {
	c := ADBachelor()
	marked := ModuleSet{"GEN 101": true, "GEN 201": true, "BAI 498": true, "CCP 101": true}
	p := c.Evaluate(NewProjectSet(), marked, "")
	// GEN 101 (1) + GEN 201 (2) count as General; BAI 498 (4) as Standard.
	// CCP 101 (Libft) is tracked through 42 only, so marking it does nothing.
	if p.Credits[General] != 3 || p.Credits[Standard] != 4 {
		t.Errorf("credits = %v, want general 3, standard 4", p.Credits)
	}
	// A module completed on 42 and also marked counts once.
	if p := c.Evaluate(NewProjectSet(2267), ModuleSet{"ARI 111": true}, ""); p.Credits[Standard] != 2 {
		t.Errorf("ARI 111 standard = %d, want 2", p.Credits[Standard])
	}
	if !c.SelfReported("GEN 301") || c.SelfReported("CCP 101") {
		t.Error("SelfReported: GEN 301 should be, CCP 101 should not")
	}
	if _, err := New([]Module{{"STD 1", "std", 2, Standard, ids(2), Solo}}, nil, []string{"NOPE"}); err == nil {
		t.Error("unknown self-reported module: want error")
	}
}
