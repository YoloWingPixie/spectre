package spec

import (
	"html"
	"regexp"
	"strings"
)

// reID matches every spec ID form, including short requirement IDs
// ("IDN-002") that take the prefix of the last full ID before them.
// fullReqPattern is a full requirement ID of any spec: "<PREFIX>-FR-" or
// "<PREFIX>-NFR-", an area code and three digits ("ATC-FR-IDN-001",
// "ASOC-NFR-PERF-002").
const fullReqPattern = `[A-Z][A-Z0-9]{1,7}-(?:FR|NFR)-[A-Z0-9]+-\d{3}`

// defaultPrefix is the requirement prefix when nothing else names one.
const defaultPrefix = "ATC"

var reID = regexp.MustCompile(fullReqPattern +
	`|OD-\d{3}|T-M\d-\d{2}|S-\d{2}[a-z]?|DE-[A-Z0-9]+` +
	`|[A-Z][A-Z0-9]{1,4}-\d{3}`)

var (
	reShortID = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,4}-\d{3}$`)
	reSubScen = regexp.MustCompile(`^S-\d{2}[a-z]$`)
	reDEID    = regexp.MustCompile(`^DE-[A-Z0-9]+$`)
	reFullReq = regexp.MustCompile(`^([A-Z][A-Z0-9]{1,7})-(?:FR|NFR)-`)
)

// Linker resolves ID tokens against the set of known IDs.
type Linker struct {
	prefix string // default requirement prefix for short IDs before any full one ("ATC")
	known  map[string]bool
	short  map[string][]string // "IDN-002" -> full IDs ending in it
}

// NewLinker builds a linker over the given IDs; short IDs before any full
// one take the most common requirement prefix among them.
func NewLinker(ids []string) *Linker { return NewLinkerPrefix(ids, dominantPrefix(ids)) }

// NewLinkerPrefix builds a linker whose short IDs default to prefix
// ("ASOC": "CHK-001" -> "ASOC-FR-CHK-001"); "" = the IDs' most common.
func NewLinkerPrefix(ids []string, prefix string) *Linker {
	if prefix == "" {
		prefix = dominantPrefix(ids)
	}
	lk := &Linker{prefix: prefix, known: map[string]bool{}, short: map[string][]string{}}
	for _, id := range ids {
		lk.known[id] = true
		if reFullReq.MatchString(id) {
			parts := strings.SplitN(id, "-", 3)
			lk.short[parts[2]] = append(lk.short[parts[2]], id)
		}
	}
	return lk
}

// dominantPrefix is the most common requirement prefix of ids (the first
// seen on a tie), defaultPrefix when there is none.
func dominantPrefix(ids []string) string {
	n := map[string]int{}
	best := ""
	for _, id := range ids {
		m := reFullReq.FindStringSubmatch(id)
		if m == nil {
			continue
		}
		n[m[1]]++
		if best == "" || n[m[1]] > n[best] {
			best = m[1]
		}
	}
	if best == "" {
		return defaultPrefix
	}
	return best
}

// Prefix is the default requirement prefix of short IDs ("ATC").
func (lk *Linker) Prefix() string {
	if lk == nil || lk.prefix == "" {
		return defaultPrefix
	}
	return lk.prefix
}

// Known reports whether id is a known item ID.
func (lk *Linker) Known(id string) bool { return lk != nil && lk.known[id] }

// linkState walks text in order, remembering the last full requirement prefix.
type linkState struct {
	lk     *Linker
	prefix string
}

func (lk *Linker) state() *linkState { return &linkState{lk: lk, prefix: lk.Prefix() + "-FR-"} }

// IDMatch is one resolved ID mention in a text.
type IDMatch struct {
	Start, End int
	Token      string // as written
	Target     string // resolved full ID
}

func boundaryOK(s string, start, end int) bool {
	if start > 0 {
		b := s[start-1]
		if isWord(b) || b == '-' {
			return false
		}
	}
	if end < len(s) {
		b := s[end]
		if b == '-' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' {
			return false
		}
	}
	return true
}

func (st *linkState) resolve(tok string) string {
	if st.lk == nil {
		return ""
	}
	if m := reFullReq.FindString(tok); m != "" {
		st.prefix = m // "ASOC-NFR-"
	}
	if st.lk.known[tok] {
		return tok
	}
	if reSubScen.MatchString(tok) && st.lk.known[tok[:len(tok)-1]] {
		return tok[:len(tok)-1]
	}
	if reShortID.MatchString(tok) && !strings.HasPrefix(tok, "OD-") {
		if c := st.prefix + tok; st.lk.known[c] {
			return c
		}
		if s := st.lk.short[tok]; len(s) == 1 {
			return s[0]
		}
	}
	return ""
}

// scan returns the resolved ID mentions in s, in order.
func (st *linkState) scan(s string) []IDMatch {
	var out []IDMatch
	for _, loc := range reID.FindAllStringIndex(s, -1) {
		// A long match may hide a shorter valid one; the regexp is
		// leftmost-first, so re-check the boundary on the match as found.
		if !boundaryOK(s, loc[0], loc[1]) {
			continue
		}
		tok := s[loc[0]:loc[1]]
		if t := st.resolve(tok); t != "" {
			out = append(out, IDMatch{Start: loc[0], End: loc[1], Token: tok, Target: t})
		}
	}
	return out
}

// link escapes plain text and wraps each resolved ID in a link to /id/<ID>.
func (st *linkState) link(s string) string {
	if st == nil || st.lk == nil {
		return html.EscapeString(s)
	}
	ms := st.scan(s)
	if len(ms) == 0 {
		return html.EscapeString(s)
	}
	var b strings.Builder
	pos := 0
	for _, m := range ms {
		b.WriteString(html.EscapeString(s[pos:m.Start]))
		b.WriteString(`<a class="idl" href="/id/` + m.Target + `">` + html.EscapeString(m.Token) + `</a>`)
		pos = m.End
	}
	b.WriteString(html.EscapeString(s[pos:]))
	return b.String()
}

// Refs returns the distinct IDs that text mentions, in first-mention order.
func (lk *Linker) Refs(text string) []string {
	st := lk.state()
	seen := map[string]bool{}
	var out []string
	for _, m := range st.scan(text) {
		if !seen[m.Target] {
			seen[m.Target] = true
			out = append(out, m.Target)
		}
	}
	return out
}

// LinkText escapes plain text and links the IDs in it.
func (lk *Linker) LinkText(s string) string { return lk.state().link(s) }
