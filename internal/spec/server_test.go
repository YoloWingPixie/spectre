package spec

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// newTestServer serves a copy of the ATC sample spec from a temp dir.
func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	for _, f := range SpecFiles {
		b, err := os.ReadFile(filepath.Join(specDir(t), f))
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
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts, dir
}

var noRedirect = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func TestScenarioIntroductionRemainsReviewable(t *testing.T) {
	ts, dir := newTestServer(t)
	if err := os.WriteFile(filepath.Join(dir, "scenarios.md"), []byte("# Scenarios\n\nIntroduction text.\n\n### S-01 · Introduction scenario\n\nA reviewer opens this scenario.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/id/S-01", "/scenarios/1", "/search?kind=scenario"} {
		code, body := get(t, ts.URL+path)
		if code != http.StatusOK || !strings.Contains(body, "Introduction scenario") {
			t.Errorf("%s: status %d, missing scenario: %v", path, code, !strings.Contains(body, "Introduction scenario"))
		}
	}
	_, body := get(t, ts.URL+"/scenarios/1")
	if !strings.Contains(body, "Introduction text.") || !strings.Contains(body, "Accept") {
		t.Fatal("introduction text or review controls missing")
	}
}

func get(t *testing.T, u string) (int, string) {
	t.Helper()
	resp, err := noRedirect.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func post(t *testing.T, u string, form url.Values) *http.Response {
	t.Helper()
	resp, err := noRedirect.PostForm(u, form)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp
}

func TestHandlersGET(t *testing.T) {
	ts, _ := newTestServer(t)
	tests := []struct {
		path string
		code int
		want []string
	}{
		{"/", 200, []string{"<b>59</b> requirements", "<b>7</b> scenarios", "<b>28</b> tasks", "<b>6</b> open decisions", "0 of 59 reviewed"}},
		{"/req", 200, []string{`href="/req/1"`, "Area index"}},
		{"/req/1", 200, []string{`id="ATC-FR-IDN-001"`, "0 of 12 reviewed", `action="/feedback"`, `href="/id/DE-IDN"`}},
		{"/req/999", 404, nil},
		{"/id/ATC-FR-IDN-001", 200, []string{"Referenced by", `href="/id/S-01"`, `href="/id/T-M1-01"`, `action="/comment"`}},
		{"/id/OD-002", 200, []string{`action="/answer"`, `value="c"`, "mission flag", "recommended"}},
		{"/id/DE-FAC", 200, []string{"Facility controller", "Referenced by"}},
		{"/id/NOPE-1", 404, nil},
		{"/design/1", 200, []string{"All pages"}},
		{"/scenarios/2", 200, []string{`class="card`}},
		{"/tasks/2", 200, []string{`id="T-M`}},
		{"/od", 200, []string{`href="/id/OD-001"`, "resolved"}},
		{"/search?q=say+callsign", 200, []string{"results", "<mark>"}},
		{"/search?kind=req&pri=MVP&st=decided", 200, []string{"ATC-FR-"}}, // all ODs resolved 2026-10-01: no open requirements left
		{"/search?q=zzzqqq", 200, []string{"0 results"}},
		{"/readme", 200, []string{"Counts"}},
		{"/static/style.css", 200, []string{"content-visibility:auto"}},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			code, body := get(t, ts.URL+tt.path)
			if code != tt.code {
				t.Fatalf("status = %d, want %d", code, tt.code)
			}
			for _, w := range tt.want {
				if !strings.Contains(body, w) {
					t.Errorf("body lacks %q", w)
				}
			}
			if len(body) > 150_000 {
				t.Errorf("page is %d bytes", len(body))
			}
		})
	}
}

func TestAllPagesSmall(t *testing.T) {
	ts, _ := newTestServer(t)
	s, _ := Load(specDir(t))
	var paths []string
	for _, ch := range s.Chapters {
		paths = append(paths, "/req/"+strconv.Itoa(ch.Num))
	}
	for name, d := range s.Docs {
		for _, p := range d.Pages {
			paths = append(paths, "/"+name+"/"+strconv.Itoa(p.Num))
		}
	}
	for _, p := range paths {
		code, body := get(t, ts.URL+p)
		if code != 200 || len(body) > 150_000 {
			t.Errorf("%s: status %d, %d bytes", p, code, len(body))
		}
	}
}

func TestSearchFiltersAndPaging(t *testing.T) {
	s, err := Load(specDir(t))
	if err != nil {
		t.Fatal(err)
	}
	rev := &Review{Feedback: map[string][]Entry{"ATC-FR-IDN-001": {{Fields: []Field{{"Verdict", "Accept"}}}}}}
	count := func(q Query) int { return len(Search(s, rev, q)) }
	tests := []struct {
		name string
		q    Query
		want int
	}{
		{"all MVP reqs", Query{Kind: "req", Pri: "MVP"}, 26},
		{"all reqs", Query{Kind: "req"}, 59},
		{"accepted", Query{Rev: "accepted"}, 1},
		{"unreviewed reqs", Query{Kind: "req", Rev: "unreviewed"}, 58},
		{"chapter 1", Query{Chapter: 1}, 12},
		{"tasks M0", Query{Kind: "task", Ms: "M0"}, 16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := count(tt.q); got != tt.want {
				t.Errorf("count = %d, want %d", got, tt.want)
			}
		})
	}
	if n := count(Query{Kind: "req", St: "open"}); n != s.Counts().Status["open"] {
		t.Errorf("open = %d", n)
	}
	if n := count(Query{Kind: "req", Ms: "M1"}); n < 26 {
		t.Errorf("M1 reqs = %d, want at least the 26 MVP", n)
	}

	ts, _ := newTestServer(t)
	_, p1 := get(t, ts.URL+"/search?kind=req&page=1")
	_, p2 := get(t, ts.URL+"/search?kind=req&page=2")
	if strings.Count(p1, `class="res"`) != SearchPageSize {
		t.Errorf("page 1 has %d results", strings.Count(p1, `class="res"`))
	}
	if !strings.Contains(p1, "page=2") || strings.Count(p2, `class="res"`) != 59-SearchPageSize {
		t.Errorf("paging broken: last page has %d results", strings.Count(p2, `class="res"`))
	}
}

func TestPostFeedback(t *testing.T) {
	ts, dir := newTestServer(t)
	resp := post(t, ts.URL+"/feedback", url.Values{"id": {"ATC-FR-IDN-002"}, "verdict": {"change"}, "text": {"Tighten wording"}, "back": {"/req/1"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/req/1#ATC-FR-IDN-002" {
		t.Fatalf("status %d location %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	post(t, ts.URL+"/feedback", url.Values{"id": {"ATC-FR-IDN-002"}, "verdict": {"Accept"}, "back": {"//evil.example"}})
	b, err := os.ReadFile(filepath.Join(dir, "review", FeedbackFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "## ATC-FR-IDN-002 · ") || !strings.Contains(string(b), "- **Verdict:** Change") ||
		!strings.Contains(string(b), "> Tighten wording") {
		t.Errorf("feedback file:\n%s", b)
	}
	_, body := get(t, ts.URL+"/req/1")
	if !strings.Contains(body, "1 of 12 reviewed") || !strings.Contains(body, `<span class="chip v-accept">accepted</span>`) ||
		!strings.Contains(body, "Tighten wording") {
		t.Error("chapter does not show the latest verdict, history and progress")
	}
	_, idx := get(t, ts.URL+"/")
	if !strings.Contains(idx, "1 of 59 reviewed") {
		t.Error("index does not show review progress")
	}
	// next unreviewed skips the reviewed item
	resp, _ = noRedirect.Get(ts.URL + "/next?kind=req&after=ATC-FR-IDN-001")
	if loc := resp.Header.Get("Location"); loc != "/id/ATC-FR-IDN-003" {
		t.Errorf("next = %q", loc)
	}
	resp, _ = noRedirect.Get(ts.URL + "/next?kind=req&in=page&after=ATC-FR-IDN-001")
	if loc := resp.Header.Get("Location"); loc != "/req/1#ATC-FR-IDN-003" {
		t.Errorf("next in page = %q", loc)
	}
	_, unrev := get(t, ts.URL+"/search?kind=req&rev=change")
	if strings.Contains(unrev, "ATC-FR-IDN-002") {
		t.Error("latest verdict (Accept) should win over Change")
	}

	tests := []struct {
		name string
		form url.Values
		code int
	}{
		{"bad verdict", url.Values{"id": {"ATC-FR-IDN-002"}, "verdict": {"Maybe"}}, 400},
		{"unknown id", url.Values{"id": {"ATC-FR-XXX-999"}, "verdict": {"Accept"}}, 400},
		{"scenario", url.Values{"id": {"S-01"}, "verdict": {"Question"}}, 303},
		{"task", url.Values{"id": {"T-M1-01"}, "verdict": {"Reject"}}, 303},
		{"design element", url.Values{"id": {"DE-FAC"}, "verdict": {"Accept"}}, 303},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if r := post(t, ts.URL+"/feedback", tt.form); r.StatusCode != tt.code {
				t.Errorf("status = %d, want %d", r.StatusCode, tt.code)
			}
		})
	}
}

func TestPostCommentAndAnswer(t *testing.T) {
	ts, dir := newTestServer(t)
	if r := post(t, ts.URL+"/comment", url.Values{"id": {"S-01"}, "text": {"Covers IDN-002 too?"}}); r.StatusCode != 303 {
		t.Fatalf("comment status %d", r.StatusCode)
	}
	if r := post(t, ts.URL+"/comment", url.Values{"id": {"S-01"}, "text": {"  "}}); r.StatusCode != 400 {
		t.Errorf("empty comment status %d", r.StatusCode)
	}
	_, body := get(t, ts.URL+"/id/S-01")
	if !strings.Contains(body, `Covers <a class="idl" href="/id/ATC-FR-IDN-002">IDN-002</a> too?`) {
		t.Error("comment not shown under the item")
	}

	tests := []struct {
		name string
		form url.Values
		code int
		want string
	}{
		{"option", url.Values{"id": {"OD-002"}, "choice": {"c"}, "note": {"default off"}}, 303, "- **Choice:** (c) mission flag"},
		{"own", url.Values{"id": {"OD-003"}, "choice": {"own"}, "answer": {"Both, per mission"}}, 303, "> Both, per mission"},
		{"own empty", url.Values{"id": {"OD-003"}, "choice": {"own"}}, 400, ""},
		{"bad option", url.Values{"id": {"OD-002"}, "choice": {"z"}}, 400, ""},
		{"not an OD", url.Values{"id": {"S-01"}, "choice": {"a"}}, 400, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if r := post(t, ts.URL+"/answer", tt.form); r.StatusCode != tt.code {
				t.Fatalf("status = %d, want %d", r.StatusCode, tt.code)
			}
			if tt.want != "" {
				b, _ := os.ReadFile(filepath.Join(dir, "review", AnswersFile))
				if !strings.Contains(string(b), tt.want) {
					t.Errorf("answers.md lacks %q:\n%s", tt.want, b)
				}
			}
		})
	}
	_, body = get(t, ts.URL+"/id/OD-002")
	if !strings.Contains(body, "answered (pending fold-in)") {
		t.Error("answered OD not marked")
	}
	_, list := get(t, ts.URL+"/od?state=answered")
	if !strings.Contains(list, `href="/id/OD-002"`) || strings.Contains(list, `href="/id/OD-001"`) {
		t.Error("answered filter wrong")
	}
	orig, _ := os.ReadFile(filepath.Join(specDir(t), "open-decisions.md"))
	now, _ := os.ReadFile(filepath.Join(dir, "open-decisions.md"))
	if string(orig) != string(now) {
		t.Error("open-decisions.md was modified")
	}
}

func TestGuard(t *testing.T) {
	ts, _ := newTestServer(t)
	req, _ := http.NewRequest("POST", ts.URL+"/comment", strings.NewReader("id=S-01&text=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")
	resp, err := noRedirect.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("cross-origin POST status = %d", resp.StatusCode)
	}
	req, _ = http.NewRequest("GET", ts.URL+"/", nil)
	req.Host = "attacker.example"
	resp, err = noRedirect.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("foreign Host status = %d", resp.StatusCode)
	}
}

func TestReloadOnChange(t *testing.T) {
	ts, dir := newTestServer(t)
	p := filepath.Join(dir, "requirements.md")
	b, _ := os.ReadFile(p)
	b = []byte(strings.Replace(string(b), "· Unidentified caller", "· Unidentified caller EDITED", 1))
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(p, later, later)
	_, body := get(t, ts.URL+"/id/ATC-FR-IDN-001")
	if !strings.Contains(body, "Unidentified caller EDITED") {
		t.Error("edit not picked up")
	}
}
