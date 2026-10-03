package spec

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// asocDir is the ASOC sample spec (testdata/asoc-spec): a second spec with
// its own requirement prefix, read by the same viewer.
func asocDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "asoc-spec")
}

func TestASOCSpecLoads(t *testing.T) {
	s, err := Load(asocDir(t))
	if err != nil {
		t.Fatal(err)
	}
	c := s.Counts()
	if s.Prefix != "ASOC" || s.Name != "ASOC" {
		t.Errorf("prefix, name = %q, %q; want ASOC, ASOC", s.Prefix, s.Name)
	}
	if c.Reqs == 0 || c.Scenarios == 0 || c.Tasks == 0 || c.ODs == 0 || c.DEs == 0 {
		t.Fatalf("counts = %+v", c)
	}
	for _, it := range s.ByKind[KindReq] {
		if !strings.HasPrefix(it.ID, "ASOC-FR-") && !strings.HasPrefix(it.ID, "ASOC-NFR-") {
			t.Errorf("requirement %s has a foreign prefix", it.ID)
		}
		if it.Page == "" || it.Field("EARS") == "" {
			t.Errorf("%s: page %q, EARS empty %v", it.ID, it.Page, it.Field("EARS") == "")
		}
		if len(it.Milestones) == 0 {
			t.Errorf("%s is in no milestone", it.ID)
		}
	}
	for _, od := range s.ByKind[KindOD] {
		if len(od.Options) < 2 {
			t.Errorf("%s: %d options parsed", od.ID, len(od.Options))
		}
	}
	// Short IDs in Covers resolve against the ASOC prefix.
	backs := []struct{ target, from string }{
		{"ASOC-FR-CHK-001", "T-M1-04"},  // first full ID of the Covers line
		{"ASOC-FR-CHK-002", "T-M1-04"},  // short ID after a full one
		{"ASOC-NFR-TST-002", "T-M1-09"}, // NFR prefix carries
		{"ASOC-FR-IDN-001", "S-01"},     // scenario Covers
		{"ASOC-FR-JTC-002", "OD-001"},   // OD Blocks
		{"DE-ACO", "ASOC-FR-ACO-001"},   // requirement Design field
	}
	for _, b := range backs {
		found := false
		for _, x := range s.Backlinks[b.target] {
			found = found || x == b.from
		}
		if !found {
			t.Errorf("%s not referenced by %s (backlinks %v)", b.target, b.from, s.Backlinks[b.target])
		}
	}
}

func TestATCSpecPrefixUnchanged(t *testing.T) {
	s := loadSpec(t)
	if s.Prefix != "ATC" || s.Name != "ATC" {
		t.Errorf("prefix, name = %q, %q; want ATC, ATC", s.Prefix, s.Name)
	}
}

func TestSpecPrefixAndName(t *testing.T) {
	reqs := []*Item{{ID: "ASOC-FR-CHK-001"}, {ID: "ASOC-NFR-PERF-001"}, {ID: "ATC-FR-IDN-001"}}
	tests := []struct {
		name, readme, title string
		reqs                []*Item
		prefix, short       string
	}{
		{"from ids", "", "ASOC role: specification (SDD)", reqs, "ASOC", "ASOC"},
		{"declared", "<!-- spectre: prefix=XYZ -->", "Thing: spec", reqs, "XYZ", "Thing"},
		{"legacy declaration", "<!-- specview: prefix=XYZ -->", "Thing: spec", reqs, "XYZ", "Thing"},
		{"no ids", "", "", nil, "ATC", "ATC"},
		{"title without colon", "", "Radio spec", reqs, "ASOC", "Radio spec"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := specPrefix(tt.readme, tt.reqs)
			if p != tt.prefix {
				t.Errorf("prefix = %q, want %q", p, tt.prefix)
			}
			if n := shortName(tt.title, p); n != tt.short {
				t.Errorf("name = %q, want %q", n, tt.short)
			}
		})
	}
}

func TestLinkerASOCPrefix(t *testing.T) {
	lk := NewLinkerPrefix([]string{"ASOC-FR-CHK-001", "ASOC-FR-CHK-002", "ASOC-NFR-PERF-001", "ASOC-NFR-PERF-002", "ATC-FR-CHK-002"}, "ASOC")
	tests := []struct{ in, want string }{
		{"CHK-002", `href="/id/ASOC-FR-CHK-002"`},                        // default prefix before any full ID
		{"ATC-FR-CHK-002, CHK-002", `href="/id/ATC-FR-CHK-002">CHK-002`}, // the last full prefix wins
		{"ASOC-NFR-PERF-001, PERF-002", `href="/id/ASOC-NFR-PERF-002">PERF-002`},
	}
	for _, tt := range tests {
		if got := lk.LinkText(tt.in); !strings.Contains(got, tt.want) {
			t.Errorf("LinkText(%q) = %s, want it to contain %s", tt.in, got, tt.want)
		}
	}
	if got := NewLinker([]string{"ASOC-FR-CHK-001", "OD-001"}).Prefix(); got != "ASOC" {
		t.Errorf("NewLinker prefix = %q, want ASOC", got)
	}
}

func TestASOCServer(t *testing.T) {
	dir := t.TempDir()
	for _, f := range SpecFiles {
		b, err := os.ReadFile(filepath.Join(asocDir(t), f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s, err := New(dir, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	ts := newHTTPTest(t, s)
	pages := []struct {
		path string
		want []string
	}{
		{"/", []string{"<title>ASOC Spec Review</title>", `<a href="/">ASOC Spec</a>`}},
		{"/req/1", []string{"<title>ASOC Requirements", `id="ASOC-FR-ROL-001"`}},
		{"/id/ASOC-FR-CHK-002", []string{"Referenced by", `href="/id/T-M1-04"`}},
		{"/od", []string{"<title>ASOC Open Decisions</title>", "OD-017"}},
	}
	for _, p := range pages {
		code, body := get(t, ts+p.path)
		if code != 200 {
			t.Errorf("%s: %d", p.path, code)
			continue
		}
		for _, w := range p.want {
			if !strings.Contains(body, w) {
				t.Errorf("%s: missing %q", p.path, w)
			}
		}
	}
	resp := post(t, ts+"/feedback", map[string][]string{"id": {"ASOC-FR-CHK-002"}, "verdict": {"Accept"}, "back": {"/req/3"}})
	if resp.StatusCode != 303 {
		t.Fatalf("feedback: %d", resp.StatusCode)
	}
	b, err := os.ReadFile(filepath.Join(dir, "review", FeedbackFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "# ASOC spec review: feedback") || !strings.Contains(string(b), "## ASOC-FR-CHK-002 · ") {
		t.Errorf("feedback file:\n%s", b)
	}
}

func newHTTPTest(t *testing.T, s *Server) string {
	t.Helper()
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

func TestATCTitlesUnchanged(t *testing.T) {
	ts, dir := newTestServer(t)
	for path, want := range map[string]string{
		"/":         "<title>ATC Spec Review</title>",
		"/od":       "<title>ATC Open Decisions</title>",
		"/design/1": "<title>ATC Design",
	} {
		_, body := get(t, ts.URL+path)
		if !strings.Contains(body, want) || !strings.Contains(body, `<a href="/">ATC Spec</a>`) {
			t.Errorf("%s: missing %q or the ATC header", path, want)
		}
	}
	post(t, ts.URL+"/comment", map[string][]string{"id": {"S-01"}, "text": {"x"}, "back": {"/"}})
	b, err := os.ReadFile(filepath.Join(dir, "review", CommentsFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "# ATC spec review: comments") {
		t.Errorf("comments file title:\n%s", b)
	}
}
