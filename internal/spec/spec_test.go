package spec

import (
	"path/filepath"
	"strings"
	"testing"
)

// specDir is the ATC sample spec (testdata/atc-spec): a trimmed snapshot
// with the "ATC" prefix.
func specDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "atc-spec")
}

func loadSpec(t *testing.T) *Spec {
	t.Helper()
	s, err := Load(specDir(t))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestParseCounts(t *testing.T) {
	s := loadSpec(t)
	c := s.Counts()
	tests := []struct {
		name      string
		got, want int
	}{
		{"requirements", c.Reqs, 59},
		{"scenarios", c.Scenarios, 7},
		{"tasks", c.Tasks, 28},
		{"open decisions", c.ODs, 6},
		{"MVP", c.Pri["MVP"], 26},
		{"P2", c.Pri["P2"], 30},
		{"P3", c.Pri["P3"], 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
	if c.DEs == 0 {
		t.Error("no design elements")
	}
	if got := c.Status["decided"] + c.Status["derived"] + c.Status["open"]; got != 59 {
		t.Errorf("status total = %d (%v)", got, c.Status)
	}
}

func TestItemsHavePagesAndBacklinks(t *testing.T) {
	s := loadSpec(t)
	for id, it := range s.Items {
		if it.Page == "" {
			t.Errorf("%s has no page", id)
		}
		if it.Title == "" {
			t.Errorf("%s has no title", id)
		}
	}
	tests := []struct {
		target, from string
	}{
		{"ATC-FR-IDN-001", "S-01"},    // scenario Covers
		{"ATC-FR-IDN-001", "T-M1-01"}, // task Covers
		{"DE-IDN", "ATC-FR-IDN-001"},  // requirement Design field
		{"ATC-FR-FAC-018", "OD-002"},  // OD Blocks
		{"ATC-FR-IDN-002", "T-M1-02"}, // short ID after a full one
	}
	for _, tt := range tests {
		t.Run(tt.target+"<-"+tt.from, func(t *testing.T) {
			if !contains(s.Backlinks[tt.target], tt.from) {
				t.Errorf("backlinks of %s = %v, want %s", tt.target, s.Backlinks[tt.target], tt.from)
			}
		})
	}
}

func TestMilestonesAndOptions(t *testing.T) {
	s := loadSpec(t)
	tests := []struct {
		id, ms string
	}{
		{"T-M0-01", "M0"},
		{"ATC-FR-IDN-001", "M1"}, // covered by T-M1-01
		{"S-01", "M1"},           // "S-01…S-21" in the M1 exit column
		{"S-27", "M6"},           // "S-26…S-29"
	}
	for _, tt := range tests {
		if !contains(s.Items[tt.id].Milestones, tt.ms) {
			t.Errorf("%s milestones = %v, want %s", tt.id, s.Items[tt.id].Milestones, tt.ms)
		}
	}
	if got := s.Items["OD-002"].Options; len(got) != 3 || got[2] != "mission flag" {
		t.Errorf("OD-002 options = %q", got)
	}
	if got := s.Items["OD-076"].Options; len(got) != 3 {
		t.Errorf("OD-076 (list-form) options = %q", got)
	}
	for _, od := range s.ByKind[KindOD] {
		if len(od.Options) < 2 {
			t.Errorf("%s has %d options", od.ID, len(od.Options))
		}
	}
	if st := s.Items["OD-008"].Status; st != "resolved" {
		t.Errorf("OD-008 status = %q, want resolved", st)
	}
}

func TestParseOptions(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"(a) one; (b) two (see (c) below); (c) three.", []string{"one", "two (see (c) below)", "three"}},
		{"\n- (a) first;\n- (b) second;\n\nCommentary (a) here.", []string{"first", "second"}},
		{"no options", nil},
	}
	for _, tt := range tests {
		got := parseOptions(tt.in)
		if strings.Join(got, "|") != strings.Join(tt.want, "|") {
			t.Errorf("parseOptions(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPagesStaySmall(t *testing.T) {
	s := loadSpec(t)
	for name, d := range s.Docs {
		for _, p := range d.Pages {
			size := 0
			for _, u := range p.Units {
				size += s.unitSize(u)
			}
			if size > 2*PageBudget {
				t.Errorf("%s page %d (%s) is %d bytes", name, p.Num, p.Section, size)
			}
		}
	}
}
