package spec

import (
	"fmt"
	"html"
	"html/template"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// SearchPageSize is the number of results per search page.
const SearchPageSize = 40

// Query is a parsed search request.
type Query struct {
	Q, Kind, Pri, St, Ms, Rev string
	Chapter                   int
	Page                      int
}

// Hit is one search result.
type Hit struct {
	Item  *Item  // nil for a document section
	URL   string // for sections
	Title string
	Kind  string
	Text  string // plain text, for the snippet
}

// ParseQuery reads a query from URL values.
func ParseQuery(v url.Values) Query {
	q := Query{Q: strings.TrimSpace(v.Get("q")), Kind: v.Get("kind"), Pri: v.Get("pri"), St: v.Get("st"),
		Ms: v.Get("ms"), Rev: v.Get("rev")}
	q.Page, _ = strconv.Atoi(v.Get("page"))
	if q.Page < 1 {
		q.Page = 1
	}
	q.Chapter, _ = strconv.Atoi(v.Get("chapter"))
	return q
}

// Values encodes the query (page included when > 1).
func (q Query) Values(page int) url.Values {
	v := url.Values{}
	for k, x := range map[string]string{"q": q.Q, "kind": q.Kind, "pri": q.Pri, "st": q.St, "ms": q.Ms, "rev": q.Rev} {
		if x != "" {
			v.Set(k, x)
		}
	}
	if q.Chapter > 0 {
		v.Set("chapter", strconv.Itoa(q.Chapter))
	}
	if page > 1 {
		v.Set("page", strconv.Itoa(page))
	}
	return v
}

func (q Query) itemFilters() bool {
	return q.Pri != "" || q.St != "" || q.Ms != "" || q.Rev != "" || q.Chapter > 0
}

// reviewState maps a verdict to the review-state filter value.
func reviewState(verdict string) string {
	switch verdict {
	case "":
		return "unreviewed"
	case "Accept":
		return "accepted"
	case "Change":
		return "change"
	case "Reject":
		return "rejected"
	}
	return "question"
}

// Search returns all hits for q, in document order.
func Search(sp *Spec, rev *Review, q Query) []Hit {
	words := strings.Fields(strings.ToLower(q.Q))
	match := func(text string) bool {
		for _, w := range words {
			if !strings.Contains(text, w) {
				return false
			}
		}
		return true
	}
	var hits []Hit
	for _, k := range []Kind{KindReq, KindScenario, KindTask, KindOD, KindDE} {
		if q.Kind != "" && q.Kind != string(k) {
			continue
		}
		for _, it := range sp.ByKind[k] {
			if q.Pri != "" && it.Pri != q.Pri {
				continue
			}
			if q.St != "" && it.Status != q.St {
				continue
			}
			if q.Chapter > 0 && (it.Kind != KindReq || it.Chapter != q.Chapter) {
				continue
			}
			if q.Ms != "" && !contains(it.Milestones, q.Ms) {
				continue
			}
			if q.Rev != "" && reviewState(rev.Verdict(it.ID)) != q.Rev {
				continue
			}
			if !match(it.text) {
				continue
			}
			hits = append(hits, Hit{Item: it, Title: it.Title, Kind: KindLabel(it.Kind),
				Text: it.Field("EARS") + " " + it.Body + " " + fieldsText(it.Fields)})
		}
	}
	if q.itemFilters() || (q.Kind != "" && q.Kind != "section") || len(words) == 0 {
		return hits
	}
	// Document sections (non-item text).
	add := func(url, title, md string) {
		if match(strings.ToLower(title + " " + md)) {
			hits = append(hits, Hit{URL: url, Title: title, Kind: "Section", Text: md})
		}
	}
	add("/readme", "Read me", sp.Readme)
	add("/req", "Requirements overview", sp.ReqIntro)
	for _, ch := range sp.Chapters {
		if ch.Intro != "" {
			add(fmt.Sprintf("/req/%d", ch.Num), ch.Title+" (introduction)", ch.Intro)
		}
	}
	for _, name := range []string{"design", "scenarios", "tasks"} {
		for _, p := range sp.Docs[name].Pages {
			var md []string
			for _, u := range p.Units {
				if u.ItemID == "" {
					md = append(md, u.MD)
				}
			}
			add(fmt.Sprintf("/%s/%d", name, p.Num), sp.Docs[name].Title+": "+pageLabel(p), strings.Join(md, "\n"))
		}
	}
	add("/od", "Open decisions (introduction)", sp.ODIntro)
	return hits
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// snippet returns an escaped excerpt of text around the first query word,
// with the words highlighted.
func snippet(text string, words []string) template.HTML {
	plain := strings.Join(strings.Fields(strings.NewReplacer("**", "", "`", "", "|", " ", "#", "").Replace(text)), " ")
	lower := strings.ToLower(plain)
	start := 0
	if len(lower) == len(plain) {
		for _, w := range words {
			if i := strings.Index(lower, w); i >= 0 {
				start = i - 60
				break
			}
		}
	}
	if start < 0 {
		start = 0
	}
	for start > 0 && start < len(plain) && (plain[start]&0xC0) == 0x80 {
		start--
	}
	end := start + 220
	if end > len(plain) {
		end = len(plain)
	}
	for end < len(plain) && (plain[end]&0xC0) == 0x80 {
		end++
	}
	ex := plain[start:end]
	original := []rune(ex)
	folded := []rune(strings.ToLower(ex))
	marked := make([]bool, len(original))
	for _, word := range words {
		match := []rune(word)
		if len(match) == 0 {
			continue
		}
		for pos := 0; pos+len(match) <= len(folded); pos++ {
			if slices.Equal(folded[pos:pos+len(match)], match) {
				for i := pos; i < pos+len(match); i++ {
					marked[i] = true
				}
			}
		}
	}
	var rendered strings.Builder
	// Case-insensitive matches use rune positions because lowercasing can change UTF-8 widths.
	// Generated tags must not become match input.
	for first := 0; first < len(original); {
		last := first + 1
		for last < len(original) && marked[last] == marked[first] {
			last++
		}
		if marked[first] {
			rendered.WriteString("<mark>")
		}
		rendered.WriteString(html.EscapeString(string(original[first:last])))
		if marked[first] {
			rendered.WriteString("</mark>")
		}
		first = last
	}
	out := rendered.String()
	if start > 0 {
		out = "…" + out
	}
	if end < len(plain) {
		out += "…"
	}
	return template.HTML(out)
}

type resultView struct {
	URL, Head, Title, Kind string
	View                   *ItemView
	Snip                   template.HTML
}

type optVal struct{ Value, Label string }

func (s *Server) search(c *ctx) error {
	q := ParseQuery(c.r.URL.Query())
	c.q = q.Q
	hits := Search(c.sp, c.rev, q)
	pages := (len(hits) + SearchPageSize - 1) / SearchPageSize
	if pages == 0 {
		pages = 1
	}
	if q.Page > pages {
		q.Page = pages
	}
	lo := (q.Page - 1) * SearchPageSize
	hi := lo + SearchPageSize
	if hi > len(hits) {
		hi = len(hits)
	}
	words := strings.Fields(strings.ToLower(q.Q))
	var res []resultView
	for _, h := range hits[lo:hi] {
		r := resultView{URL: h.URL, Head: h.Title, Kind: h.Kind, Snip: snippet(h.Text, words)}
		if h.Item != nil {
			r.URL, r.Head, r.Title = "/id/"+h.Item.ID, h.Item.ID, h.Item.Title
			r.View = c.view(h.Item, "", false)
		}
		res = append(res, r)
	}
	link := func(p int) string {
		if p < 1 || p > pages {
			return ""
		}
		return "/search?" + q.Values(p).Encode()
	}
	noCh := q
	noCh.Chapter = 0
	return s.render(c, "search", c.sp.Name+" Spec Search", struct {
		Q                Query
		Total, Pages     int
		Results          []resultView
		PrevURL, NextURL string
		NoChapter        string
		Kinds, Revs      []optVal
		Pris, Sts, Mss   []string
	}{q, len(hits), pages, res, link(q.Page - 1), link(q.Page + 1), "/search?" + noCh.Values(1).Encode(),
		[]optVal{{"req", "Requirements"}, {"scenario", "Scenarios"}, {"task", "Tasks"}, {"od", "Open decisions"},
			{"de", "Design elements"}, {"section", "Document sections"}},
		[]optVal{{"unreviewed", "Unreviewed"}, {"accepted", "Accepted"}, {"change", "Change requested"},
			{"rejected", "Rejected"}, {"question", "Question"}},
		[]string{"MVP", "P2", "P3"}, []string{"decided", "derived", "open", "resolved"},
		[]string{"M0", "M1", "M2", "M3", "M4", "M5", "M6", "M7", "M8"}})
}
