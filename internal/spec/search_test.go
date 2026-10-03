package spec

import (
	"strings"
	"testing"
)

func TestSnippetHighlightsOriginalText(t *testing.T) {
	for _, test := range []struct {
		name, text string
		words      []string
		want       string
	}{
		{"repeated", "a mark", strings.Fields(strings.Repeat("a ", 10)), "<mark>a</mark> m<mark>a</mark>rk"},
		{"overlap", "banana", []string{"ana", "nan"}, "b<mark>anana</mark>"},
		{"escaping", "<a & b>", []string{"a", "&"}, "&lt;<mark>a</mark> <mark>&amp;</mark> b&gt;"},
		{"case", "Alpha ALPHA", []string{"alpha"}, "<mark>Alpha</mark> <mark>ALPHA</mark>"},
		{"unicode case widths", "ȺKȺ", []string{"ⱥ"}, "<mark>Ⱥ</mark>K<mark>Ⱥ</mark>"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := string(snippet(test.text, test.words)); got != test.want {
				t.Fatalf("got %.300q, want %q", got, test.want)
			}
		})
	}
}
