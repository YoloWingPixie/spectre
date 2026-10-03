package audit

import (
	"fmt"
	"strings"
	"testing"
)

func TestMarkdownCodeAndInlineReferences(t *testing.T) {
	for _, test := range []struct{ name, input, want string }{
		{"ordinary code", "Before `x < y & z` after", "Before <code>x &lt; y &amp; z</code> after"},
		{"traversal reference", "Before `../auth.go:12` after", "Before <code>../auth.go:12</code> after"},
		{"zero line", "Before `src/auth.go:0` after", "Before <code>src/auth.go:0</code> after"},
		{"invalid line", "Before `src/auth.go:no` after", "Before <code>src/auth.go:no</code> after"},
		{"root file", "Before `auth.go:12` after", "Before <code>auth.go:12</code> after"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := string(markdown(test.input, "/project", "Ubuntu")); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
	got := string(markdown("First `src/auth.go:12-20` middle `x & y` last `src/app.go:7` end <safe>", "/project", "Ubuntu"))
	for _, want := range []string{`href="vscode://vscode-remote/wsl+Ubuntu/project/src/auth.go:12:1"`, ">src/auth.go:12-20</a>", " middle <code>x &amp; y</code> last ", `href="vscode://vscode-remote/wsl+Ubuntu/project/src/app.go:7:1"`, " end &lt;safe&gt;"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered references lack %q", want)
		}
	}
}

func TestReportSeverityOrderingAndCounts(t *testing.T) {
	doc, err := readReport(starterReport, false)
	if err != nil {
		t.Fatal(err)
	}
	r := doc.Report
	original := r.Findings[0]
	r.Findings = nil
	r.FixFirst = nil
	r.OpenQuestions = nil
	for _, entry := range []struct {
		id       findingID
		severity severity
		status   findingStatus
	}{
		{"low-open", low, open}, {"high-first", high, open}, {"critical-closed", critical, fixed},
		{"high-second", high, open}, {"medium-open", medium, open}, {"critical-open", critical, open},
	} {
		f := original
		f.ID = entry.id
		f.Severity = entry.severity
		f.Status = entry.status
		r.Findings = append(r.Findings, f)
	}
	page, err := renderReport(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	html := string(page)
	previous := -1
	for _, id := range []string{"critical-open", "high-first", "high-second", "medium-open", "low-open", "critical-closed"} {
		position := strings.Index(html, `id="f-`+id+`"`)
		if position <= previous {
			t.Fatalf("finding %s missing or out of order", id)
		}
		previous = position
	}
	for _, entry := range []struct {
		severity string
		count    int
	}{{"critical", 1}, {"high", 2}, {"medium", 1}, {"low", 1}} {
		want := fmt.Sprintf(`<div class="tile %s"><span class="n">%d</span>`, entry.severity, entry.count)
		if !strings.Contains(html, want) {
			t.Errorf("incorrect open severity count: missing %s", want)
		}
	}
}
