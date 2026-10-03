package audit

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestSentenceBoundariesAcrossNewlines(t *testing.T) {
	for _, separator := range []string{"\n", "\r\n", " \n\n\t"} {
		t.Run(fmt.Sprintf("%q", separator), func(t *testing.T) {
			got := sentences("First sentence." + separator + "Next sentence.")
			want := []string{"First sentence.", "Next sentence."}
			if !slices.Equal(got, want) {
				t.Fatalf("sentences = %q, want %q", got, want)
			}
		})
	}
}

func TestLintPhraseMatchingAndOrder(t *testing.T) {
	r := report{Scope: "We UTILIZE robust, cutting edge tools. `leverage` and robustly are code or different words.", Notes: "A cutting-edge tool is robust."}
	want := []string{
		`(root).scope: avoid "utilize"`,
		`(root).scope: avoid "robust"`,
		`(root).scope: avoid "cutting-edge"`,
		`(root).notes: avoid "robust"`,
		`(root).notes: avoid "cutting-edge"`,
	}
	if got := lintReport(r); !slices.Equal(got, want) {
		t.Fatalf("warnings = %q, want %q", got, want)
	}
}

func BenchmarkLintReport(b *testing.B) {
	doc, err := readReport("templates/report.starter.json", false)
	if err != nil {
		b.Fatal(err)
	}
	for _, count := range []int{1, 100} {
		b.Run(fmt.Sprintf("findings_%d", count), func(b *testing.B) {
			r := doc.Report
			r.Findings = make([]finding, count)
			for i := range r.Findings {
				r.Findings[i] = doc.Report.Findings[0]
				r.Findings[i].ID = findingID(fmt.Sprintf("F-%d", i))
			}
			if warnings := lintReport(r); len(warnings) != 0 {
				b.Fatal(strings.Join(warnings, "\n"))
			}
			b.ReportAllocs()
			for b.Loop() {
				lintReport(r)
			}
		})
	}
}
