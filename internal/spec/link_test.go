package spec

import (
	"strings"
	"testing"
)

func testLinker() *Linker {
	return NewLinker([]string{"ATC-FR-IDN-001", "ATC-FR-IDN-002", "ATC-NFR-PERF-001", "ATC-NFR-PERF-002",
		"ATC-FR-PERF-009", "OD-012", "T-M1-01", "S-06", "DE-FAC", "ATC-FR-GUA-003"})
}

func TestLinkText(t *testing.T) {
	lk := testLinker()
	tests := []struct {
		name, in, want string
	}{
		{"full", "see ATC-FR-IDN-001.", `see <a class="idl" href="/id/ATC-FR-IDN-001">ATC-FR-IDN-001</a>.`},
		{"short after full", "ATC-FR-IDN-001, IDN-002", `<a class="idl" href="/id/ATC-FR-IDN-001">ATC-FR-IDN-001</a>, <a class="idl" href="/id/ATC-FR-IDN-002">IDN-002</a>`},
		{"nfr prefix carries", "ATC-NFR-PERF-001, PERF-002", `<a class="idl" href="/id/ATC-NFR-PERF-001">ATC-NFR-PERF-001</a>, <a class="idl" href="/id/ATC-NFR-PERF-002">PERF-002</a>`},
		{"unique short", "GUA-003", `<a class="idl" href="/id/ATC-FR-GUA-003">GUA-003</a>`},
		{"od task scenario de", "OD-012 T-M1-01 S-06a DE-FAC", `<a class="idl" href="/id/OD-012">OD-012</a> <a class="idl" href="/id/T-M1-01">T-M1-01</a> <a class="idl" href="/id/S-06">S-06a</a> <a class="idl" href="/id/DE-FAC">DE-FAC</a>`},
		{"unknown stays text", "OD-999 and XYZ-001", "OD-999 and XYZ-001"},
		{"no partial match", "ATC-FR-IDN-0011 xATC-FR-IDN-001", "ATC-FR-IDN-0011 xATC-FR-IDN-001"},
		{"escapes", "<b> & OD-012", `&lt;b&gt; &amp; <a class="idl" href="/id/OD-012">OD-012</a>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lk.LinkText(tt.in); got != tt.want {
				t.Errorf("LinkText(%q)\n got %s\nwant %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestRefs(t *testing.T) {
	lk := testLinker()
	got := lk.Refs("ATC-FR-IDN-001, IDN-002, IDN-001; OD-012")
	want := []string{"ATC-FR-IDN-001", "ATC-FR-IDN-002", "OD-012"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Refs = %v, want %v", got, want)
	}
}

func TestRenderMarkdown(t *testing.T) {
	lk := testLinker()
	tests := []struct {
		name, in string
		want     []string // substrings
		not      []string
	}{
		{"paragraph links", "Needs ATC-FR-IDN-001 and **OD-012**.", []string{`<p>Needs <a class="idl" href="/id/ATC-FR-IDN-001">`, `<strong><a class="idl" href="/id/OD-012">OD-012</a></strong>`}, nil},
		{"raw html escaped", "a <script>x</script> <name>", []string{"&lt;script&gt;", "&lt;name&gt;"}, []string{"<script>"}},
		{"code fence not linked", "```\nATC-FR-IDN-001 <x>\n```", []string{"<pre><code>ATC-FR-IDN-001 &lt;x&gt;</code></pre>"}, []string{"href"}},
		{"inline code linked", "`OD-012`", []string{`<code><a class="idl" href="/id/OD-012">OD-012</a></code>`}, nil},
		{"table", "| ID | x |\n|---|---|\n| DE-FAC | a `b|c` |", []string{`<tr id="DE-FAC">`, "<th>ID</th>", "<code>b|c</code>"}, nil},
		{"nested list", "- a\n  - b\n- c", []string{"<ul>\n<li>a\n<ul>\n<li>b</li>", "<li>c</li>"}, nil},
		{"ordered", "1. one\n2. two", []string{"<ol>", "<li>one</li>", "<li>two</li>"}, nil},
		{"heading", "### 3.1 Layout", []string{`<h3 id="3-1-layout">3.1 Layout</h3>`}, nil},
		{"emphasis", "*the tower* and snake_case_name", []string{"<em>the tower</em>", "snake_case_name"}, []string{"<em>case</em>"}},
		{"md link mapped", "[req](requirements.md) [x](javascript:alert(1))", []string{`<a href="/req">req</a>`, `<a href="#">`}, []string{"javascript"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderMarkdown(tt.in, lk)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("missing %q in\n%s", w, got)
				}
			}
			for _, w := range tt.not {
				if strings.Contains(got, w) {
					t.Errorf("unexpected %q in\n%s", w, got)
				}
			}
		})
	}
}
