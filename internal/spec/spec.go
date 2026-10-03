package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Kind is the type of a spec item.
type Kind string

// Item kinds.
const (
	KindReq      Kind = "req"
	KindScenario Kind = "scenario"
	KindTask     Kind = "task"
	KindOD       Kind = "od"
	KindDE       Kind = "de"
)

// KindLabel is the human name of a kind.
func KindLabel(k Kind) string {
	switch k {
	case KindReq:
		return "Requirement"
	case KindScenario:
		return "Scenario"
	case KindTask:
		return "Task"
	case KindOD:
		return "Open decision"
	case KindDE:
		return "Design element"
	}
	return "Section"
}

// SpecFiles are the Markdown files the viewer reads.
var SpecFiles = []string{"README.md", "requirements.md", "design.md", "scenarios.md", "tasks.md", "open-decisions.md"}

// Field is one "- **Name:** value" field of an item.
type Field struct {
	Name string
	MD   string // dedented Markdown value
}

// Item is one addressable spec entry.
type Item struct {
	ID, Title  string
	Kind       Kind
	Seq        int    // order within its kind
	Section    string // chapter or section title
	Page       string // URL of the page that shows it in context (with anchor)
	Chapter    int    // requirement chapter number (1-based), or doc page number
	Pri        string // MVP / P2 / P3 (requirements)
	Status     string // decided / derived / open (requirements); open / resolved (ODs)
	StatusText string
	Milestones []string
	Fields     []Field
	Body       string   // Markdown body (for items without fields, or the rest)
	Options    []string // OD options, without the "(a)" marker
	Refs       []string // resolved outgoing IDs
	text       string   // lower-case search text
}

// Field returns the Markdown of a named field.
func (it *Item) Field(name string) string {
	for _, f := range it.Fields {
		if f.Name == name {
			return f.MD
		}
	}
	return ""
}

// Chapter is one requirements chapter.
type Chapter struct {
	Num   int
	Title string
	Code  string
	Intro string // Markdown before the first requirement
	Items []*Item
}

// Unit is a piece of a document page: either an item or plain Markdown.
type Unit struct {
	ItemID string
	MD     string
}

// DocPage is one page of a paginated document.
type DocPage struct {
	Num     int
	Section string // the ## section title
	Part    int    // 1-based part within the section; 0 when the section fits one page
	Units   []Unit
}

// Doc is a paginated document (design, scenarios, tasks).
type Doc struct {
	Name, Title string
	Pages       []*DocPage
}

// Spec is a parsed spec folder.
type Spec struct {
	Title     string
	Prefix    string // requirement ID prefix ("ATC", "ASOC"): README declaration, else the IDs found
	Name      string // short name for page titles ("ATC"): the README title before ":" without " role", else Prefix
	Readme    string // Markdown body of README.md
	ReqTitle  string
	ReqIntro  string // Markdown of the requirements overview
	Chapters  []*Chapter
	Items     map[string]*Item
	ByKind    map[Kind][]*Item
	Docs      map[string]*Doc // design, scenarios, tasks
	ODIntro   string
	ODTitle   string
	Backlinks map[string][]string
	Linker    *Linker
}

// Counts summarizes the spec.
type Counts struct {
	Reqs, Scenarios, Tasks, ODs, DEs int
	Pri, Status                      map[string]int
	ODResolved                       int
}

// Counts returns the item counts.
func (s *Spec) Counts() Counts {
	c := Counts{Pri: map[string]int{}, Status: map[string]int{}}
	c.Reqs = len(s.ByKind[KindReq])
	c.Scenarios = len(s.ByKind[KindScenario])
	c.Tasks = len(s.ByKind[KindTask])
	c.ODs = len(s.ByKind[KindOD])
	c.DEs = len(s.ByKind[KindDE])
	for _, it := range s.ByKind[KindReq] {
		c.Pri[it.Pri]++
		c.Status[it.Status]++
	}
	for _, it := range s.ByKind[KindOD] {
		if it.Status == "resolved" {
			c.ODResolved++
		}
	}
	return c
}

var (
	reFieldLine  = regexp.MustCompile(`^- \*\*([^*]+?):\*\*[ \t]*(.*)$`)
	reItemHead   = regexp.MustCompile(`^#{3,4} (` + fullReqPattern + `|S-\d{2}|T-M\d-\d{2}|OD-\d{3}) · (.*)$`)
	reDeclPrefix = regexp.MustCompile(`<!--\s*(?:spectre|specview):\s*prefix\s*=\s*([A-Z][A-Z0-9]{1,7})\s*-->`)
	reChapCode   = regexp.MustCompile(`\(([A-Z0-9]+)\)\s*$`)
	reStatus     = regexp.MustCompile(`\*\*Status:\*\*\s*(.+)$`)
	reOption     = regexp.MustCompile(`\(([a-z])\)\s*`)
	reMsRow      = regexp.MustCompile(`^M(\d)\b`)
	reScenRange  = regexp.MustCompile(`S-(\d{2})(?:\s*(?:…|\.\.\.)\s*S-(\d{2}))?`)
)

// Load reads and parses the spec files in dir.
func Load(dir string) (*Spec, error) {
	src := map[string]string{}
	for _, f := range SpecFiles {
		b, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return nil, fmt.Errorf("spectre: read spec: %w", err)
		}
		src[f] = strings.ReplaceAll(string(b), "\r\n", "\n")
	}
	return Parse(src)
}

func splitTitle(md string) (string, string) {
	first, rest, _ := strings.Cut(md, "\n")
	return strings.TrimSpace(strings.TrimLeft(first, "# ")), rest
}

// Parse parses the spec from file contents keyed by file name.
func Parse(src map[string]string) (*Spec, error) {
	s := &Spec{Items: map[string]*Item{}, ByKind: map[Kind][]*Item{}, Docs: map[string]*Doc{}}
	s.Title, s.Readme = splitTitle(src["README.md"])
	if err := s.parseRequirements(src["requirements.md"]); err != nil {
		return nil, err
	}
	s.Prefix = specPrefix(src["README.md"], s.ByKind[KindReq])
	s.Name = shortName(s.Title, s.Prefix)
	s.Docs["design"] = s.parseDoc("design", src["design.md"], "")
	s.Docs["scenarios"] = s.parseDoc("scenarios", src["scenarios.md"], "S-")
	s.Docs["tasks"] = s.parseDoc("tasks", src["tasks.md"], "T-")
	s.parseODs(src["open-decisions.md"])
	s.parseDEs(src["design.md"])

	var ids []string
	for id := range s.Items {
		ids = append(ids, id)
	}
	s.Linker = NewLinkerPrefix(ids, s.Prefix)
	s.computeRefs()
	s.computeMilestones(src["tasks.md"])
	for _, it := range s.Items {
		it.text = strings.ToLower(it.ID + " " + it.Title + " " + it.Section + " " + it.Body + " " + fieldsText(it.Fields))
	}
	return s, nil
}

func fieldsText(fs []Field) string {
	var b strings.Builder
	for _, f := range fs {
		b.WriteString(f.Name + " " + f.MD + "\n")
	}
	return b.String()
}

func (s *Spec) add(it *Item) {
	if _, dup := s.Items[it.ID]; dup {
		return
	}
	it.Seq = len(s.ByKind[it.Kind])
	s.Items[it.ID] = it
	s.ByKind[it.Kind] = append(s.ByKind[it.Kind], it)
}

// splitSections splits text at lines starting with "## " (outside code
// fences). The first element is the text before the first heading.
func splitSections(text string) (pre string, titles, bodies []string) {
	lines := strings.Split(text, "\n")
	var cur []string
	inCode := false
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inCode = !inCode
		}
		if !inCode && strings.HasPrefix(l, "## ") {
			if titles == nil {
				pre = strings.Join(cur, "\n")
			} else {
				bodies = append(bodies, strings.Join(cur, "\n"))
			}
			titles = append(titles, strings.TrimSpace(l[3:]))
			cur = nil
			continue
		}
		cur = append(cur, l)
	}
	if titles == nil {
		pre = strings.Join(cur, "\n")
	} else {
		bodies = append(bodies, strings.Join(cur, "\n"))
	}
	return pre, titles, bodies
}

// splitUnits splits a section body at ### / #### headings (outside code).
// Each returned chunk starts with its heading line, except the first.
func splitUnits(body string) []string {
	lines := strings.Split(body, "\n")
	var out []string
	var cur []string
	inCode := false
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inCode = !inCode
		}
		if !inCode && (strings.HasPrefix(l, "### ") || strings.HasPrefix(l, "#### ")) {
			out = append(out, strings.Join(cur, "\n"))
			cur = nil
		}
		cur = append(cur, l)
	}
	return append(out, strings.Join(cur, "\n"))
}

// trimRule removes a trailing "---" separator and surrounding blank lines.
func trimRule(s string) string {
	s = strings.TrimSpace(s)
	for strings.HasSuffix(s, "\n---") || s == "---" {
		s = strings.TrimSpace(strings.TrimSuffix(s, "---"))
	}
	return s
}

// parseFields splits an item body into its "- **Name:** value" fields and
// any remaining Markdown (after the fields).
func parseFields(body string) ([]Field, string) {
	lines := strings.Split(body, "\n")
	var fields []Field
	var rest []string
	var cur *Field
	var val []string
	done := func() {
		if cur != nil {
			cur.MD = dedent(strings.TrimRight(strings.Join(val, "\n"), "\n "))
			fields = append(fields, *cur)
			cur = nil
		}
	}
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if len(rest) > 0 {
			rest = append(rest, l)
			continue
		}
		if m := reFieldLine.FindStringSubmatch(l); m != nil {
			done()
			cur = &Field{Name: m[1]}
			val = []string{m[2]}
			continue
		}
		if cur != nil && (strings.HasPrefix(l, " ") || blank(l)) {
			if blank(l) {
				// A blank line ends the fields unless indented text follows.
				n := i + 1
				for n < len(lines) && blank(lines[n]) {
					n++
				}
				if n >= len(lines) || !strings.HasPrefix(lines[n], " ") {
					done()
					continue
				}
			}
			val = append(val, l)
			continue
		}
		done()
		if !blank(l) || len(rest) > 0 {
			rest = append(rest, l)
		}
	}
	done()
	return fields, strings.TrimSpace(strings.Join(rest, "\n"))
}

// dedent removes the common indentation of continuation lines (line 2 on);
// the first line is the text after the field label.
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	min := -1
	for _, l := range lines[1:] {
		if blank(l) {
			continue
		}
		if n := indentOf(l); min < 0 || n < min {
			min = n
		}
	}
	if min <= 0 {
		return strings.TrimSpace(s)
	}
	for i := 1; i < len(lines); i++ {
		if len(lines[i]) >= min {
			lines[i] = lines[i][min:]
		} else {
			lines[i] = strings.TrimSpace(lines[i])
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (s *Spec) parseRequirements(md string) error {
	title, body := splitTitle(md)
	s.ReqTitle = title
	pre, titles, bodies := splitSections(body)
	intro := []string{strings.TrimSpace(pre)}
	for i, t := range titles {
		units := splitUnits(bodies[i])
		var reqs []string
		for _, u := range units[1:] {
			if reItemHead.MatchString(strings.SplitN(u, "\n", 2)[0]) {
				reqs = append(reqs, u)
			}
		}
		if len(reqs) == 0 {
			intro = append(intro, "## "+t+"\n\n"+trimRule(bodies[i]))
			continue
		}
		ch := &Chapter{Num: len(s.Chapters) + 1, Title: t, Intro: trimRule(units[0])}
		if m := reChapCode.FindStringSubmatch(t); m != nil {
			ch.Code = m[1]
		}
		for _, u := range reqs {
			head, rest, _ := strings.Cut(u, "\n")
			m := reItemHead.FindStringSubmatch(head)
			fields, extra := parseFields(trimRule(rest))
			it := &Item{ID: m[1], Title: strings.TrimSpace(m[2]), Kind: KindReq, Section: t,
				Chapter: ch.Num, Page: fmt.Sprintf("/req/%d#%s", ch.Num, m[1]), Fields: fields, Body: extra}
			pl := it.Field("Priority")
			it.Pri = strings.Fields(pl + " ?")[0]
			if sm := reStatus.FindStringSubmatch(pl); sm != nil {
				it.StatusText = strings.TrimSpace(sm[1])
				it.Status = strings.Fields(it.StatusText)[0]
			}
			// Priority and status are shown as chips, not as a field.
			it.Fields = without(it.Fields, "Priority")
			ch.Items = append(ch.Items, it)
			s.add(it)
		}
		s.Chapters = append(s.Chapters, ch)
	}
	s.ReqIntro = strings.Join(intro, "\n\n")
	if len(s.ByKind[KindReq]) == 0 {
		return fmt.Errorf("spectre: no requirements found")
	}
	return nil
}

func without(fs []Field, name string) []Field {
	out := fs[:0:0]
	for _, f := range fs {
		if f.Name != name {
			out = append(out, f)
		}
	}
	return out
}

// PageBudget is the Markdown size (bytes) a document page aims for.
const PageBudget = 36_000

// parseDoc paginates a document by ## section, splitting long sections at
// ###/#### headings, blank lines and table rows. Items whose ID starts with
// itemPrefix become addressable items.
func (s *Spec) parseDoc(name, md, itemPrefix string) *Doc {
	title, body := splitTitle(md)
	d := &Doc{Name: name, Title: title}
	pre, titles, bodies := splitSections(body)
	type sec struct {
		title string
		units []Unit
	}
	var secs []sec
	if strings.TrimSpace(trimRule(pre)) != "" {
		titles = append([]string{"Introduction"}, titles...)
		bodies = append([]string{pre}, bodies...)
	}
	for i, t := range titles {
		var units []Unit
		for k, u := range splitUnits(bodies[i]) {
			u = trimRule(u)
			if u == "" {
				continue
			}
			head, rest, _ := strings.Cut(u, "\n")
			if m := reItemHead.FindStringSubmatch(head); k > 0 && m != nil && itemPrefix != "" && strings.HasPrefix(m[1], itemPrefix) {
				it := &Item{ID: m[1], Title: strings.TrimSpace(m[2]), Section: t}
				if itemPrefix == "S-" {
					it.Kind = KindScenario
					it.Body = trimRule(rest)
				} else {
					it.Kind = KindTask
					it.Fields, it.Body = parseFields(trimRule(rest))
				}
				s.add(it)
				units = append(units, Unit{ItemID: it.ID})
				continue
			}
			units = append(units, Unit{MD: u})
		}
		secs = append(secs, sec{title: t, units: units})
	}
	for _, sc := range secs {
		var pages [][]Unit
		var cur []Unit
		size := 0
		for _, u := range sc.units {
			n := s.unitSize(u)
			if n > PageBudget && u.ItemID == "" {
				for _, piece := range splitMarkdown(u.MD, PageBudget) {
					if len(cur) > 0 && size+len(piece) > PageBudget {
						pages = append(pages, cur)
						cur, size = nil, 0
					}
					cur = append(cur, Unit{MD: piece})
					size += len(piece)
				}
				continue
			}
			if len(cur) > 0 && size+n > PageBudget {
				pages = append(pages, cur)
				cur, size = nil, 0
			}
			cur = append(cur, u)
			size += n
		}
		if len(cur) > 0 || len(pages) == 0 {
			pages = append(pages, cur)
		}
		for k, p := range pages {
			dp := &DocPage{Num: len(d.Pages) + 1, Section: sc.title, Units: p}
			if len(pages) > 1 {
				dp.Part = k + 1
			}
			if k == 0 {
				// The section heading opens its first page.
				dp.Units = append([]Unit{{MD: "## " + sc.title}}, dp.Units...)
				if sc.title == "Introduction" {
					dp.Units = dp.Units[1:]
				}
			}
			for _, u := range dp.Units {
				if it := s.Items[u.ItemID]; it != nil {
					it.Chapter = dp.Num
					it.Page = fmt.Sprintf("/%s/%d#%s", name, dp.Num, it.ID)
				}
			}
			d.Pages = append(d.Pages, dp)
		}
	}
	return d
}

func (s *Spec) unitSize(u Unit) int {
	if it := s.Items[u.ItemID]; it != nil {
		return len(it.Body) + len(fieldsText(it.Fields)) + len(it.Title) + 400
	}
	return len(u.MD)
}

// splitMarkdown cuts md into pieces of about budget bytes at blank lines;
// a table longer than the budget is cut by rows, repeating its header.
func splitMarkdown(md string, budget int) []string {
	var blocks []string
	lines := strings.Split(md, "\n")
	var cur []string
	inCode := false
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inCode = !inCode
		}
		if !inCode && blank(l) {
			if len(cur) > 0 {
				blocks = append(blocks, strings.Join(cur, "\n"))
				cur = nil
			}
			continue
		}
		cur = append(cur, l)
		_ = i
	}
	if len(cur) > 0 {
		blocks = append(blocks, strings.Join(cur, "\n"))
	}
	var out []string
	var page []string
	size := 0
	emit := func(b string) {
		if len(page) > 0 && size+len(b) > budget {
			out = append(out, strings.Join(page, "\n\n"))
			page, size = nil, 0
		}
		page = append(page, b)
		size += len(b) + 2
	}
	for _, b := range blocks {
		bl := strings.Split(b, "\n")
		if len(b) > budget && len(bl) > 3 && isTableStart(bl, 0) {
			head := bl[0] + "\n" + bl[1]
			var rows []string
			rsize := len(head)
			for _, r := range bl[2:] {
				if len(rows) > 0 && size+rsize+len(r) > budget {
					emit(head + "\n" + strings.Join(rows, "\n"))
					rows, rsize = nil, len(head)
				}
				rows = append(rows, r)
				rsize += len(r) + 1
			}
			if len(rows) > 0 {
				emit(head + "\n" + strings.Join(rows, "\n"))
			}
			continue
		}
		emit(b)
	}
	if len(page) > 0 {
		out = append(out, strings.Join(page, "\n\n"))
	}
	return out
}

func (s *Spec) parseODs(md string) {
	title, body := splitTitle(md)
	s.ODTitle = title
	units := splitUnits(body)
	s.ODIntro = trimRule(units[0])
	for _, u := range units[1:] {
		head, rest, _ := strings.Cut(u, "\n")
		m := reItemHead.FindStringSubmatch(head)
		if m == nil || !strings.HasPrefix(m[1], "OD-") {
			continue
		}
		it := &Item{ID: m[1], Title: strings.TrimSpace(m[2]), Kind: KindOD, Section: "Open decisions", Page: "/od#" + m[1]}
		it.Fields, it.Body = parseFields(trimRule(rest))
		it.Status = "open"
		if strings.Contains(it.Title, "(resolved)") || it.Field("Resolution") != "" {
			it.Status = "resolved"
		}
		it.StatusText = it.Status
		it.Options = parseOptions(it.Field("Options"))
		s.add(it)
	}
}

// parseOptions splits "(a) x; (b) y" (inline or as a list) into options.
func parseOptions(md string) []string {
	// Text after a blank line is commentary on the options, not an option.
	if i := strings.Index(md, "\n\n"); i >= 0 {
		md = md[:i]
	}
	locs := reOption.FindAllStringSubmatchIndex(md, -1)
	var starts [][2]int // marker start, text start
	want := byte('a')
	for _, l := range locs {
		before := strings.TrimRight(md[:l[0]], " ")
		atStart := before == "" || strings.HasSuffix(before, ";") || strings.HasSuffix(before, "\n") ||
			strings.HasSuffix(before, "\n-") || before == "-" || strings.HasSuffix(before, ":")
		if md[l[2]] == want && atStart {
			starts = append(starts, [2]int{l[0], l[1]})
			want++
		}
	}
	var out []string
	for i, st := range starts {
		end := len(md)
		if i+1 < len(starts) {
			end = starts[i+1][0]
		}
		o := strings.TrimSpace(md[st[1]:end])
		o = strings.TrimSpace(strings.TrimSuffix(o, "-"))
		o = strings.TrimRight(o, ";,. \n")
		o = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(o), " or"))
		out = append(out, strings.Join(strings.Fields(o), " "))
	}
	return out
}

func (s *Spec) parseDEs(md string) {
	_, body := splitTitle(md)
	_, titles, bodies := splitSections(body)
	for i, t := range titles {
		for _, l := range strings.Split(bodies[i], "\n") {
			if !strings.HasPrefix(l, "| DE-") {
				continue
			}
			cells := SplitCells(l)
			if len(cells) < 2 || !reDEID.MatchString(cells[0]) {
				continue
			}
			it := &Item{ID: cells[0], Title: cells[1], Kind: KindDE, Section: t}
			names := []string{"Element", "Package / files", "Responsibility"}
			for k, c := range cells[2:] {
				n := fmt.Sprintf("Column %d", k+3)
				if k+1 < len(names) {
					n = names[k+1]
				}
				it.Fields = append(it.Fields, Field{Name: n, MD: c})
			}
			// Find the design page with this table row.
			for _, p := range s.Docs["design"].Pages {
				for _, u := range p.Units {
					if strings.Contains(u.MD, "| "+it.ID+" |") {
						it.Chapter = p.Num
						it.Page = fmt.Sprintf("/design/%d#%s", p.Num, it.ID)
					}
				}
				if it.Page != "" {
					break
				}
			}
			s.add(it)
		}
	}
}

func (s *Spec) computeRefs() {
	s.Backlinks = map[string][]string{}
	for _, k := range []Kind{KindReq, KindScenario, KindTask, KindOD, KindDE} {
		for _, it := range s.ByKind[k] {
			var refs []string
			for _, r := range s.Linker.Refs(fieldsText(it.Fields) + "\n" + it.Body) {
				if r != it.ID {
					refs = append(refs, r)
				}
			}
			it.Refs = refs
			for _, r := range refs {
				s.Backlinks[r] = append(s.Backlinks[r], it.ID)
			}
		}
	}
}

// computeMilestones assigns milestones: tasks from their ID, requirements
// from the tasks that cover them, scenarios from the milestone table's exit
// column, open decisions from the requirements they block.
func (s *Spec) computeMilestones(tasksMD string) {
	add := func(it *Item, m string) {
		for _, x := range it.Milestones {
			if x == m {
				return
			}
		}
		it.Milestones = append(it.Milestones, m)
	}
	for _, t := range s.ByKind[KindTask] {
		m := t.ID[2:4]
		add(t, m)
		for _, r := range s.Linker.Refs(t.Field("Covers")) {
			if it := s.Items[r]; it != nil && it.Kind == KindReq {
				add(it, m)
			}
		}
	}
	// The spec defines MVP as milestone M1.
	for _, it := range s.ByKind[KindReq] {
		if it.Pri == "MVP" {
			add(it, "M1")
		}
	}
	for _, l := range strings.Split(tasksMD, "\n") {
		cells := SplitCells(l)
		if !strings.HasPrefix(strings.TrimSpace(l), "|") || len(cells) < 3 {
			continue
		}
		mm := reMsRow.FindStringSubmatch(cells[0])
		if mm == nil {
			continue
		}
		ms := "M" + mm[1]
		for _, r := range reScenRange.FindAllStringSubmatch(cells[len(cells)-1], -1) {
			a, _ := strconv.Atoi(r[1])
			b := a
			if r[2] != "" {
				b, _ = strconv.Atoi(r[2])
			}
			for n := a; n <= b && n-a < 100; n++ {
				if it := s.Items[fmt.Sprintf("S-%02d", n)]; it != nil {
					add(it, ms)
				}
			}
		}
	}
	for _, od := range s.ByKind[KindOD] {
		for _, r := range s.Linker.Refs(od.Field("Blocks")) {
			if it := s.Items[r]; it != nil && it.Kind == KindReq {
				for _, m := range it.Milestones {
					add(od, m)
				}
			}
		}
	}
	for _, it := range s.Items {
		sort.Strings(it.Milestones)
	}
}

// specPrefix is the spec's requirement ID prefix: a README declaration
// ("<!-- spectre: prefix=ASOC -->"), else the most common prefix of the
// requirement IDs (the first one on a tie).
func specPrefix(readme string, reqs []*Item) string {
	if m := reDeclPrefix.FindStringSubmatch(readme); m != nil {
		return m[1]
	}
	ids := make([]string, len(reqs))
	for i, it := range reqs {
		ids[i] = it.ID
	}
	return dominantPrefix(ids)
}

// shortName is the name page titles use: the README title before its
// first ":", without a trailing " role" ("ATC role: specification" ->
// "ATC"); prefix when the title gives nothing.
func shortName(title, prefix string) string {
	n, _, _ := strings.Cut(title, ":")
	n = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(n), " role"))
	if n == "" {
		return prefix
	}
	return n
}
