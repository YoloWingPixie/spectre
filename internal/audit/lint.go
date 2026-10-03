package audit

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

const maxSentenceWords = 30

type phraseRule struct {
	phrase  string
	pattern *regexp.Regexp
}

var wordyPhrases = compilePhraseRules("leverage", "leveraging", "utilize", "utilise", "utilization", "robust", "seamless", "seamlessly", "holistic", "synergy", "synergies", "in order to", "it is worth noting", "it should be noted", "needless to say", "going forward", "at the end of the day", "cutting-edge", "best-in-class", "world-class", "game-changer", "paradigm", "empower", "streamline", "facilitate", "delve")

func compilePhraseRules(phrases ...string) []phraseRule {
	rules := make([]phraseRule, len(phrases))
	for i, phrase := range phrases {
		pattern := `(?i)\b` + strings.ReplaceAll(regexp.QuoteMeta(phrase), "-", "[- ]") + `\b`
		rules[i] = phraseRule{phrase, regexp.MustCompile(pattern)}
	}
	return rules
}

func sentences(text string) []string {
	text = codePattern.ReplaceAllString(text, "CODE")
	parts := []string{}
	start := 0
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\n' {
			if s := strings.TrimSpace(string(runes[start:i])); s != "" {
				parts = append(parts, s)
			}
			start = i + 1
			continue
		}
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		next := i + 1
		for next < len(runes) && unicode.IsSpace(runes[next]) {
			next++
		}
		if next > i+1 && next < len(runes) && (unicode.IsUpper(runes[next]) || unicode.IsDigit(runes[next]) || strings.ContainsRune("(\"'", runes[next])) {
			if s := strings.TrimSpace(string(runes[start : i+1])); s != "" {
				parts = append(parts, s)
			}
			start = next
			i = next - 1
		}
	}
	if s := strings.TrimSpace(string(runes[start:])); s != "" {
		parts = append(parts, s)
	}
	return parts
}
func lintReport(r report) []string {
	warnings := []string{}
	text := func(path, value string) {
		prose := codePattern.ReplaceAllString(value, "CODE")
		for _, sentence := range sentences(value) {
			count := 0
			for _, word := range strings.Fields(sentence) {
				if strings.ContainsFunc(word, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) {
					count++
				}
			}
			if count > maxSentenceWords {
				warnings = append(warnings, fmt.Sprintf("%s: sentence has %d words (max %d)", path, count, maxSentenceWords))
			}
		}
		for _, rule := range wordyPhrases {
			if rule.pattern.MatchString(prose) {
				warnings = append(warnings, fmt.Sprintf("%s: avoid %q", path, rule.phrase))
			}
		}
	}
	list := func(path string, values []string) {
		for i, value := range values {
			text(fmt.Sprintf("%s[%d]", path, i), value)
		}
	}
	text("(root).scope", r.Scope)
	text("(root).notes", r.Notes)
	for i, f := range r.Findings {
		p := fmt.Sprintf("findings[%d] (%s)", i, f.ID)
		text(p+".title", f.Title)
		text(p+".summary", f.Summary)
		text(p+".statusNote", f.StatusNote)
		if len(sentences(f.Summary)) > 2 {
			warnings = append(warnings, p+".summary: keep it to 1-2 sentences")
		}
		if len(f.Evidence) == 0 {
			warnings = append(warnings, p+".evidence: no evidence")
		}
		for j, e := range f.Evidence {
			q := fmt.Sprintf("%s.evidence[%d]", p, j)
			text(q+".note", e.Note)
			text(q+".caption", e.Caption)
		}
		text(p+".current.description", f.Current.Description)
		list(p+".current.pros", f.Current.Pros)
		list(p+".current.cons", f.Current.Cons)
		if len(f.Current.Pros) == 0 {
			warnings = append(warnings, p+".current.pros: empty")
		}
		if len(f.Current.Cons) == 0 {
			warnings = append(warnings, p+".current.cons: empty")
		}
		if len(f.COAs) < 2 || len(f.COAs) > 4 {
			warnings = append(warnings, p+".coas: provide 2-4 courses of action")
		}
		recommended := false
		for j, c := range f.COAs {
			q := fmt.Sprintf("%s.coas[%d]", p, j)
			recommended = recommended || c.Recommended
			text(q+".name", c.Name)
			text(q+".description", c.Description)
			text(q+".recommendedReason", c.RecommendedReason)
			list(q+".pros", c.Pros)
			list(q+".cons", c.Cons)
			if len(c.Pros) == 0 {
				warnings = append(warnings, q+".pros: empty")
			}
			if len(c.Cons) == 0 {
				warnings = append(warnings, q+".cons: empty")
			}
		}
		if len(f.COAs) > 1 && f.Status == open && !recommended {
			warnings = append(warnings, p+".coas: choose one recommended option")
		}
	}
	for i, p := range r.FixFirst {
		text(fmt.Sprintf("fixFirst[%d].why", i), p.Why)
	}
	for i, q := range r.OpenQuestions {
		text(fmt.Sprintf("openQuestions[%d]", i), q.Text)
	}
	for i, c := range r.Clean {
		text(fmt.Sprintf("clean[%d].text", i), c.Text)
	}
	return warnings
}
