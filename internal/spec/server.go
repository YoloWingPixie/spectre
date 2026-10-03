package spec

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/YoloWingPixie/spectre/internal/webui"
)

//go:embed assets
var assets embed.FS

// Server serves the spec viewer. The spec is re-parsed when any spec file
// changes; the review files are read on every request.
type Server struct {
	dir   string
	store *Store
	log   *slog.Logger
	pages map[string]*template.Template

	mu    sync.Mutex
	spec  *Spec
	stamp string
}

// New returns a server over the spec in dir, writing review files to
// dir/review. log may be nil.
func New(dir string, log *slog.Logger) (*Server, error) {
	if log == nil {
		log = slog.Default()
	}
	s := &Server{dir: dir, store: NewStore(filepath.Join(dir, "review")), log: log}
	if err := s.parseTemplates(); err != nil {
		return nil, err
	}
	sp, err := s.Spec()
	if err != nil {
		return nil, err
	}
	s.store.Name = sp.Name // review files per spec dir, titled with its name
	return s, nil
}

func (s *Server) parseTemplates() error {
	funcs := template.FuncMap{
		"lower":        strings.ToLower,
		"verdictLabel": verdictLabel,
		"kindLabel":    KindLabel,
		"themeScript":  webui.ThemeScript,
	}
	base, err := webui.Templates(template.New("layout").Funcs(funcs))
	if err == nil {
		base, err = base.ParseFS(assets, "assets/tmpl/layout.html")
	}
	if err != nil {
		return fmt.Errorf("spectre: templates: %w", err)
	}
	s.pages = map[string]*template.Template{}
	names, err := fs.Glob(assets, "assets/tmpl/*.html")
	if err != nil {
		return fmt.Errorf("spectre: templates: %w", err)
	}
	for _, n := range names {
		name := strings.TrimSuffix(filepath.Base(n), ".html")
		if name == "layout" {
			continue
		}
		t, err := template.Must(base.Clone()).ParseFS(assets, n)
		if err != nil {
			return fmt.Errorf("spectre: template %s: %w", name, err)
		}
		s.pages[name] = t
	}
	return nil
}

func verdictLabel(v string) string {
	switch v {
	case "Accept":
		return "accepted"
	case "Change":
		return "change requested"
	case "Reject":
		return "rejected"
	case "Question":
		return "question"
	}
	return strings.ToLower(v)
}

// Spec returns the parsed spec, re-parsing it if a file changed.
func (s *Server) Spec() (*Spec, error) {
	var b strings.Builder
	for _, f := range SpecFiles {
		fi, err := os.Stat(filepath.Join(s.dir, f))
		if err != nil {
			return nil, fmt.Errorf("spectre: stat spec: %w", err)
		}
		fmt.Fprintf(&b, "%s:%d:%d;", f, fi.ModTime().UnixNano(), fi.Size())
	}
	stamp := b.String()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.spec != nil && stamp == s.stamp {
		return s.spec, nil
	}
	sp, err := Load(s.dir)
	if err != nil {
		if s.spec != nil {
			s.log.Warn("spectre: re-parse failed, serving the previous spec", "err", err)
			return s.spec, nil
		}
		return nil, err
	}
	s.spec, s.stamp = sp, stamp
	s.log.Info("spectre: parsed spec", "items", len(sp.Items))
	return sp, nil
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(assets, "assets")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /static/style.css", func(w http.ResponseWriter, r *http.Request) {
		css, err := assets.ReadFile("assets/style.css")
		if err != nil {
			http.Error(w, "cannot load styles", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		fmt.Fprint(w, webui.Styles(css))
	})
	mux.HandleFunc("GET /{$}", s.wrap(s.index))
	mux.HandleFunc("GET /readme", s.wrap(s.readme))
	mux.HandleFunc("GET /req", s.wrap(s.reqOverview))
	mux.HandleFunc("GET /req/{n}", s.wrap(s.chapter))
	mux.HandleFunc("GET /id/{id}", s.wrap(s.item))
	mux.HandleFunc("GET /od", s.wrap(s.odList))
	mux.HandleFunc("GET /search", s.wrap(s.search))
	mux.HandleFunc("GET /next", s.wrap(s.next))
	for _, d := range []string{"design", "scenarios", "tasks"} {
		mux.HandleFunc("GET /"+d, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/"+d+"/1", http.StatusFound)
		})
		mux.HandleFunc("GET /"+d+"/{n}", s.wrap(s.doc(d)))
	}
	mux.HandleFunc("POST /feedback", s.wrap(s.postFeedback))
	mux.HandleFunc("POST /comment", s.wrap(s.postComment))
	mux.HandleFunc("POST /answer", s.wrap(s.postAnswer))
	return s.guard(mux)
}

// guard rejects requests whose Host is not a loopback name (DNS rebinding)
// and cross-site POSTs.
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		host = strings.Trim(host, "[]")
		if host != "127.0.0.1" && host != "localhost" && host != "::1" {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost {
			if o := r.Header.Get("Origin"); o != "" && o != "null" {
				u, err := url.Parse(o)
				if err != nil || u.Host != r.Host {
					http.Error(w, "cross-origin request refused", http.StatusForbidden)
					return
				}
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				http.Error(w, "cross-site request refused", http.StatusForbidden)
				return
			}
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

type ctx struct {
	w   http.ResponseWriter
	r   *http.Request
	sp  *Spec
	rev *Review
	nav string
	q   string
}

// specName is the spec's short name ("ATC"; the default when no spec is loaded).
func (c *ctx) specName() string {
	if c == nil || c.sp == nil || c.sp.Name == "" {
		return defaultPrefix
	}
	return c.sp.Name
}

type handler func(c *ctx) error

type httpError struct {
	code int
	msg  string
}

func (e *httpError) Error() string { return e.msg }

func notFound(format string, a ...any) error {
	return &httpError{http.StatusNotFound, fmt.Sprintf(format, a...)}
}

func badRequest(format string, a ...any) error {
	return &httpError{http.StatusBadRequest, fmt.Sprintf(format, a...)}
}

func (s *Server) wrap(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sp, err := s.Spec()
		if err != nil {
			s.log.Error("spectre: load spec", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rev, err := s.store.Load()
		if err != nil {
			s.log.Error("spectre: load review", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		c := &ctx{w: w, r: r, sp: sp, rev: rev}
		if err := h(c); err != nil {
			code := http.StatusInternalServerError
			msg := err.Error()
			if he, ok := err.(*httpError); ok {
				code = he.code
			} else {
				s.log.Error("spectre: request", "path", r.URL.Path, "err", err)
			}
			if r.Method == http.MethodPost {
				http.Error(w, msg, code)
				return
			}
			w.WriteHeader(code)
			_ = s.render(c, "error", http.StatusText(code), struct{ Title, Msg string }{http.StatusText(code), msg})
		}
	}
}

type pageData struct {
	Title, Nav, Q string
	Data          any
	Header        webui.Header
}

func (s *Server) render(c *ctx, name, title string, data any) error {
	t := s.pages[name]
	if t == nil {
		return fmt.Errorf("spectre: no template %q", name)
	}
	c.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.w.Header().Set("Cache-Control", "no-store")
	return t.ExecuteTemplate(c.w, "layout", pageData{Title: title, Nav: c.nav, Q: c.q, Data: data, Header: webui.Header{Mode: webui.Specs, Name: c.specName() + " Spec", Home: "/"}})
}

// ---------------------------------------------------------------- views

// FieldView is a rendered field.
type FieldView struct {
	Name string
	HTML template.HTML
}

// EntryView is a rendered review entry.
type EntryView struct {
	Time, Verdict string
	Lines         []FieldView
}

// ItemView is everything a card template needs.
type ItemView struct {
	It                 *Item
	Label              string
	Verdict            string
	Answered           bool
	StatusHTML         template.HTML
	EARS, Accept, Body template.HTML
	Meta               []FieldView
	Feedback, Comments []EntryView
	Answers            []EntryView
	Back               string // page path the forms return to
	Full               bool
	FormOpen           bool
	NextURL            string
}

func entryViews(lk *Linker, es []Entry, skip string) []EntryView {
	var out []EntryView
	for _, e := range es {
		ev := EntryView{Time: strings.Replace(strings.TrimSuffix(e.Time, "Z"), "T", " ", 1) + " UTC"}
		for _, f := range e.Fields {
			if f.Name == skip {
				ev.Verdict = NormVerdict(f.MD)
				continue
			}
			ev.Lines = append(ev.Lines, FieldView{f.Name, template.HTML(lk.LinkText(f.MD))})
		}
		out = append(out, ev)
	}
	return out
}

func (c *ctx) view(it *Item, back string, full bool) *ItemView {
	lk := c.sp.Linker
	v := &ItemView{It: it, Label: KindLabel(it.Kind), Verdict: c.rev.Verdict(it.ID), Back: back, Full: full,
		FormOpen: full || it.Kind == KindReq}
	if it.Kind == KindOD {
		v.Answered = c.rev.Answered(it.ID)
	}
	if it.StatusText != "" {
		v.StatusHTML = template.HTML(RenderInline(it.StatusText, lk))
	}
	for _, f := range it.Fields {
		switch {
		case it.Kind == KindReq && f.Name == "EARS":
			v.EARS = template.HTML(RenderInline(f.MD, lk))
		case it.Kind == KindReq && f.Name == "Acceptance":
			v.Accept = template.HTML(RenderMarkdown(f.MD, lk))
		default:
			h := RenderMarkdown(f.MD, lk)
			if !strings.Contains(f.MD, "\n") {
				h = RenderInline(f.MD, lk)
			}
			v.Meta = append(v.Meta, FieldView{f.Name, template.HTML(h)})
		}
	}
	if it.Body != "" {
		v.Body = template.HTML(RenderMarkdown(it.Body, lk))
	}
	v.Feedback = entryViews(lk, c.rev.Feedback[it.ID], "Verdict")
	v.Comments = entryViews(lk, c.rev.Comments[it.ID], "")
	v.Answers = entryViews(lk, c.rev.Answers[it.ID], "")
	return v
}

// nextUnreviewed returns the first unreviewed item of kind after id
// (wrapping), or nil.
func (c *ctx) nextUnreviewed(kind Kind, after string) *Item {
	list := c.sp.ByKind[kind]
	start := 0
	if it := c.sp.Items[after]; it != nil && it.Kind == kind {
		start = it.Seq + 1
	}
	for k := 0; k < len(list); k++ {
		it := list[(start+k)%len(list)]
		if it.ID != after && c.rev.Verdict(it.ID) == "" {
			return it
		}
	}
	return nil
}

type progress struct {
	Label           string
	Done, Total     int
	Pct             int
	Breakdown, Next string
}

func (c *ctx) progress(label string, items []*Item, next string) progress {
	p := progress{Label: label, Total: len(items), Next: next}
	counts := map[string]int{}
	for _, it := range items {
		if v := c.rev.Verdict(it.ID); v != "" {
			p.Done++
			counts[v]++
		}
	}
	if p.Total > 0 {
		p.Pct = p.Done * 100 / p.Total
	}
	var parts []string
	for _, v := range Verdicts {
		if counts[v] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[v], verdictLabel(v)))
		}
	}
	p.Breakdown = strings.Join(parts, ", ")
	if p.Done == p.Total {
		p.Next = ""
	}
	return p
}

type chapterRow struct {
	Num         int
	Title       string
	Done, Total int
}

func (c *ctx) chapterRows() []chapterRow {
	var out []chapterRow
	for _, ch := range c.sp.Chapters {
		p := c.progress("", ch.Items, "")
		out = append(out, chapterRow{ch.Num, ch.Title, p.Done, p.Total})
	}
	return out
}

func (s *Server) index(c *ctx) error {
	answered := 0
	for _, od := range c.sp.ByKind[KindOD] {
		if c.rev.Answered(od.ID) {
			answered++
		}
	}
	var prog []progress
	for _, k := range []struct {
		kind  Kind
		label string
	}{{KindReq, "Requirements"}, {KindScenario, "Scenarios"}, {KindTask, "Tasks"}, {KindDE, "Design elements"}} {
		prog = append(prog, c.progress(k.label, c.sp.ByKind[k.kind], "/next?kind="+string(k.kind)))
	}
	return s.render(c, "index", c.sp.Name+" Spec Review", struct {
		Title      string
		C          Counts
		ODAnswered int
		Progress   []progress
		Chapters   []chapterRow
	}{c.sp.Title, c.sp.Counts(), answered, prog, c.chapterRows()})
}

func (s *Server) readme(c *ctx) error {
	return s.render(c, "md", c.sp.Name+" Spec Read Me", struct {
		Title string
		HTML  template.HTML
	}{c.sp.Title, template.HTML(RenderMarkdown(c.sp.Readme, c.sp.Linker))})
}

func (s *Server) reqOverview(c *ctx) error {
	c.nav = "req"
	intro := RenderMarkdown(c.sp.ReqIntro, c.sp.Linker)
	// Link area codes in the area index to their chapters.
	for _, ch := range c.sp.Chapters {
		if ch.Code != "" {
			intro = strings.Replace(intro, "<td>"+ch.Code+"</td>",
				fmt.Sprintf(`<td><a href="/req/%d">%s</a></td>`, ch.Num, ch.Code), 1)
		}
	}
	return s.render(c, "reqov", c.sp.Name+" Requirements", struct {
		Title    string
		Intro    template.HTML
		Chapters []chapterRow
	}{c.sp.ReqTitle, template.HTML(intro), c.chapterRows()})
}

func atoiPath(c *ctx, name string, max int) (int, error) {
	n, err := strconv.Atoi(c.r.PathValue(name))
	if err != nil || n < 1 || n > max {
		return 0, notFound("no page %q", c.r.PathValue(name))
	}
	return n, nil
}

func (s *Server) chapter(c *ctx) error {
	c.nav = "req"
	n, err := atoiPath(c, "n", len(c.sp.Chapters))
	if err != nil {
		return err
	}
	ch := c.sp.Chapters[n-1]
	back := fmt.Sprintf("/req/%d", n)
	var cards []*ItemView
	for _, it := range ch.Items {
		v := c.view(it, back, false)
		v.NextURL = "/next?kind=req&in=page&after=" + it.ID
		cards = append(cards, v)
	}
	p := c.progress("", ch.Items, "")
	next := ""
	for _, it := range ch.Items {
		if c.rev.Verdict(it.ID) == "" {
			next = "#" + it.ID
			break
		}
	}
	var prev, nextCh *Chapter
	if n > 1 {
		prev = c.sp.Chapters[n-2]
	}
	if n < len(c.sp.Chapters) {
		nextCh = c.sp.Chapters[n]
	}
	var intro template.HTML
	if strings.TrimSpace(ch.Intro) != "" {
		intro = template.HTML(RenderMarkdown(ch.Intro, c.sp.Linker))
	}
	title := c.sp.Name + " Requirements"
	if ch.Code != "" {
		title += " " + ch.Code
	}
	return s.render(c, "chapter", title, struct {
		Ch           *Chapter
		Done, Total  int
		Pct          int
		Next         string
		Intro        template.HTML
		Cards        []*ItemView
		Prev, NextCh *Chapter
	}{ch, p.Done, p.Total, p.Pct, next, intro, cards, prev, nextCh})
}

type pageLink struct {
	Num   int
	Label string
}

type docBlock struct {
	Card *ItemView
	HTML template.HTML
}

func pageLabel(p *DocPage) string {
	if p.Part > 0 {
		return fmt.Sprintf("%s (part %d)", p.Section, p.Part)
	}
	return p.Section
}

func (s *Server) doc(name string) handler {
	return func(c *ctx) error {
		c.nav = name
		d := c.sp.Docs[name]
		n, err := atoiPath(c, "n", len(d.Pages))
		if err != nil {
			return err
		}
		pg := d.Pages[n-1]
		base := "/" + name
		back := fmt.Sprintf("%s/%d", base, n)
		var blocks []docBlock
		var md []string
		flush := func() {
			if len(md) > 0 {
				blocks = append(blocks, docBlock{HTML: template.HTML(RenderMarkdown(strings.Join(md, "\n\n"), c.sp.Linker))})
				md = nil
			}
		}
		for _, u := range pg.Units {
			if it := c.sp.Items[u.ItemID]; it != nil {
				flush()
				v := c.view(it, back, false)
				v.NextURL = "/next?in=page&kind=" + string(it.Kind) + "&after=" + it.ID
				blocks = append(blocks, docBlock{Card: v})
				continue
			}
			md = append(md, u.MD)
		}
		flush()
		var pages []pageLink
		for _, p := range d.Pages {
			pages = append(pages, pageLink{p.Num, pageLabel(p)})
		}
		var prev, next *pageLink
		if n > 1 {
			prev = &pages[n-2]
		}
		if n < len(pages) {
			next = &pages[n]
		}
		title := c.sp.Name + " " + map[string]string{"design": "Design", "scenarios": "Scenarios", "tasks": "Tasks"}[name]
		return s.render(c, "doc", fmt.Sprintf("%s %d", title, n), struct {
			Title      string
			Base       string
			N          int
			Page       pageLink
			Pages      []pageLink
			Blocks     []docBlock
			Prev, Next *pageLink
		}{d.Title, base, len(d.Pages), pages[n-1], pages, blocks, prev, next})
	}
}

type optionView struct {
	Letter, Value string
	HTML          template.HTML
	Rec           bool
}

func (s *Server) item(c *ctx) error {
	id := c.r.PathValue("id")
	it := c.sp.Items[id]
	if it == nil {
		return notFound("no item %s", id)
	}
	c.nav = map[Kind]string{KindReq: "req", KindScenario: "scenarios", KindTask: "tasks", KindOD: "od", KindDE: "design"}[it.Kind]
	v := c.view(it, "/id/"+id, true)
	var opts []optionView
	rec := strings.ToLower(it.Field("Recommendation"))
	for i, o := range it.Options {
		l := string(rune('a' + i))
		opts = append(opts, optionView{Letter: l, Value: l, HTML: template.HTML(RenderInline(o, c.sp.Linker)),
			Rec: strings.HasPrefix(rec, "("+l+")")})
	}
	var back, refs []*Item
	for _, b := range c.sp.Backlinks[id] {
		back = append(back, c.sp.Items[b])
	}
	for _, r := range it.Refs {
		refs = append(refs, c.sp.Items[r])
	}
	list := c.sp.ByKind[it.Kind]
	var prev, next *Item
	if it.Seq > 0 {
		prev = list[it.Seq-1]
	}
	if it.Seq+1 < len(list) {
		next = list[it.Seq+1]
	}
	nu := ""
	if it.Kind != KindOD {
		nu = "/next?kind=" + string(it.Kind) + "&after=" + id
	}
	return s.render(c, "item", it.ID, struct {
		Card            *ItemView
		OD              bool
		Options         []optionView
		Backlinks, Refs []*Item
		Prev, Next      *Item
		NextUnrev       string
	}{v, it.Kind == KindOD, opts, back, refs, prev, next, nu})
}

func (s *Server) next(c *ctx) error {
	q := c.r.URL.Query()
	kind := Kind(q.Get("kind"))
	if kind == "" {
		kind = KindReq
	}
	if _, ok := c.sp.ByKind[kind]; !ok {
		return badRequest("unknown kind %q", kind)
	}
	it := c.nextUnreviewed(kind, q.Get("after"))
	if it == nil {
		http.Redirect(c.w, c.r, "/search?rev=unreviewed&kind="+string(kind), http.StatusSeeOther)
		return nil
	}
	target := "/id/" + it.ID
	if q.Get("in") == "page" && it.Page != "" {
		target = it.Page
	}
	http.Redirect(c.w, c.r, target, http.StatusSeeOther)
	return nil
}

type odRow struct {
	*ItemView
	Rec template.HTML
}

func (s *Server) odList(c *ctx) error {
	c.nav = "od"
	state := c.r.URL.Query().Get("state")
	counts := map[string]int{}
	var rows []odRow
	for _, it := range c.sp.ByKind[KindOD] {
		st := it.Status
		if c.rev.Answered(it.ID) {
			counts["answered"]++
		}
		counts[st]++
		counts[""]++
		switch state {
		case "", st:
		case "answered":
			if !c.rev.Answered(it.ID) {
				continue
			}
		case "unanswered":
			if c.rev.Answered(it.ID) || st == "resolved" {
				continue
			}
		default:
			continue
		}
		v := c.view(it, "/od", false)
		rows = append(rows, odRow{v, template.HTML(RenderInline(strings.Join(strings.Fields(it.Field("Recommendation")), " "), c.sp.Linker))})
	}
	counts["unanswered"] = counts["open"] - 0
	for _, it := range c.sp.ByKind[KindOD] {
		if it.Status == "open" && c.rev.Answered(it.ID) {
			counts["unanswered"]--
		}
	}
	type st struct {
		Value, Label string
		N            int
	}
	return s.render(c, "od", c.sp.Name+" Open Decisions", struct {
		Title  string
		State  string
		States []st
		Intro  template.HTML
		Rows   []odRow
	}{c.sp.ODTitle, state, []st{{"", "all", counts[""]}, {"open", "open", counts["open"]},
		{"unanswered", "open, unanswered", counts["unanswered"]}, {"answered", "answered (pending fold-in)", counts["answered"]},
		{"resolved", "resolved", counts["resolved"]}},
		template.HTML(RenderMarkdown(c.sp.ODIntro, c.sp.Linker)), rows})
}

// ---------------------------------------------------------------- posts

func (c *ctx) postItem() (*Item, error) {
	if err := c.r.ParseForm(); err != nil {
		return nil, badRequest("bad form: %v", err)
	}
	id := c.r.PostForm.Get("id")
	it := c.sp.Items[id]
	if it == nil {
		return nil, badRequest("unknown id %q", id)
	}
	return it, nil
}

// safeBack returns a same-site path to redirect to, with the item anchor.
func safeBack(back, id string) string {
	if !strings.HasPrefix(back, "/") || strings.HasPrefix(back, "//") || strings.ContainsAny(back, "\\\r\n#") {
		back = "/id/" + id
	}
	return back + "#" + id
}

func (s *Server) postFeedback(c *ctx) error {
	c.r.Body = http.MaxBytesReader(c.w, c.r.Body, 64<<10)
	it, err := c.postItem()
	if err != nil {
		return err
	}
	v := NormVerdict(c.r.PostForm.Get("verdict"))
	if v == "" {
		return badRequest("verdict must be one of %s", strings.Join(Verdicts, ", "))
	}
	if _, err := s.store.Append(FeedbackFile, it.ID, []Field{{"Verdict", v}, {"Text", c.r.PostForm.Get("text")}}); err != nil {
		return err
	}
	http.Redirect(c.w, c.r, safeBack(c.r.PostForm.Get("back"), it.ID), http.StatusSeeOther)
	return nil
}

func (s *Server) postComment(c *ctx) error {
	c.r.Body = http.MaxBytesReader(c.w, c.r.Body, 64<<10)
	it, err := c.postItem()
	if err != nil {
		return err
	}
	text := strings.TrimSpace(c.r.PostForm.Get("text"))
	if text == "" {
		return badRequest("empty comment")
	}
	if _, err := s.store.Append(CommentsFile, it.ID, []Field{{"Comment", text}}); err != nil {
		return err
	}
	http.Redirect(c.w, c.r, "/id/"+it.ID+"#comment", http.StatusSeeOther)
	return nil
}

func (s *Server) postAnswer(c *ctx) error {
	c.r.Body = http.MaxBytesReader(c.w, c.r.Body, 64<<10)
	it, err := c.postItem()
	if err != nil {
		return err
	}
	if it.Kind != KindOD {
		return badRequest("%s is not an open decision", it.ID)
	}
	choice := c.r.PostForm.Get("choice")
	own := strings.TrimSpace(c.r.PostForm.Get("answer"))
	var fields []Field
	switch {
	case choice == "own" || choice == "":
		if own == "" {
			return badRequest("pick an option or write an answer")
		}
		fields = []Field{{"Choice", "own answer"}, {"Answer", own}}
	case len(choice) == 1 && choice[0] >= 'a' && int(choice[0]-'a') < len(it.Options):
		fields = []Field{{"Choice", fmt.Sprintf("(%s) %s", choice, it.Options[choice[0]-'a'])}}
		if own != "" {
			fields = append(fields, Field{"Answer", own})
		}
	default:
		return badRequest("unknown option %q", choice)
	}
	fields = append(fields, Field{"Note", c.r.PostForm.Get("note")})
	if _, err := s.store.Append(AnswersFile, it.ID, fields); err != nil {
		return err
	}
	http.Redirect(c.w, c.r, "/id/"+it.ID+"#answer", http.StatusSeeOther)
	return nil
}

// ListenAndServe serves h on addr until ctx-less shutdown via the returned server.
func NewHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 10 * time.Second}
}
