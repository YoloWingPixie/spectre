package spec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Review files, appended by the viewer and folded into the spec by hand.
const (
	AnswersFile  = "answers.md"
	CommentsFile = "comments.md"
	FeedbackFile = "feedback.md"
)

// Verdicts a reviewer can give an item.
var Verdicts = []string{"Accept", "Change", "Reject", "Question"}

// NormVerdict returns the canonical verdict for v, or "" if unknown.
func NormVerdict(v string) string {
	for _, x := range Verdicts {
		if strings.EqualFold(x, strings.TrimSpace(v)) {
			return x
		}
	}
	return ""
}

// Entry is one appended review record.
type Entry struct {
	ID     string
	Time   string // RFC 3339, UTC
	Fields []Field
}

// Get returns a field value.
func (e Entry) Get(name string) string {
	for _, f := range e.Fields {
		if f.Name == name {
			return f.MD
		}
	}
	return ""
}

// Store appends to and reads the review files in a directory.
type Store struct {
	Dir  string
	Name string // spec short name for new files' titles ("ATC"; "" = "ATC")
	mu   sync.Mutex
	now  func() time.Time
}

// NewStore returns a store writing to dir (created on first write).
func NewStore(dir string) *Store { return &Store{Dir: dir, now: time.Now} }

var fileTitles = map[string]string{
	AnswersFile:  "# %s spec review: open-decision answers\n\nAppended by `cmd/spectre`, newest last. Fold into `open-decisions.md` by hand.\n",
	CommentsFile: "# %s spec review: comments\n\nAppended by `cmd/spectre`, newest last.\n",
	FeedbackFile: "# %s spec review: feedback\n\nAppended by `cmd/spectre`, newest last; the latest verdict per ID counts.\n",
}

// fileTitle is the heading a new review file starts with.
func (st *Store) fileTitle(file string) string {
	t, ok := fileTitles[file]
	if !ok {
		return ""
	}
	name := st.Name
	if name == "" {
		name = defaultPrefix
	}
	return fmt.Sprintf(t, name)
}

// MaxText limits the size of one submitted text field.
const MaxText = 20_000

// Append writes one entry to file. Multi-line values are written as a block
// quote so no user text can start a new entry.
func (st *Store) Append(file, id string, fields []Field) (Entry, error) {
	if !reAnyID.MatchString(id) {
		return Entry{}, fmt.Errorf("spectre: bad id %q", id)
	}
	e := Entry{ID: id, Time: st.now().UTC().Format(time.RFC3339), Fields: fields}
	var b strings.Builder
	fmt.Fprintf(&b, "\n## %s · %s\n", e.ID, e.Time)
	for _, f := range fields {
		v := strings.TrimSpace(strings.ReplaceAll(f.MD, "\r\n", "\n"))
		if len(v) > MaxText {
			return Entry{}, fmt.Errorf("spectre: %s too long", f.Name)
		}
		if v == "" {
			continue
		}
		if !strings.Contains(v, "\n") && len(v) <= 120 && f.Name != "Note" && f.Name != "Text" && f.Name != "Answer" {
			fmt.Fprintf(&b, "- **%s:** %s\n", f.Name, v)
			continue
		}
		fmt.Fprintf(&b, "- **%s:**\n", f.Name)
		for _, l := range strings.Split(v, "\n") {
			b.WriteString(strings.TrimRight("> "+l, " ") + "\n")
		}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if err := os.MkdirAll(st.Dir, 0o755); err != nil {
		return Entry{}, fmt.Errorf("spectre: review dir: %w", err)
	}
	path := filepath.Join(st.Dir, file)
	_, statErr := os.Stat(path)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return Entry{}, fmt.Errorf("spectre: open %s: %w", file, err)
	}
	defer f.Close()
	out := b.String()
	if errors.Is(statErr, os.ErrNotExist) {
		out = st.fileTitle(file) + out
	}
	if _, err := f.WriteString(out); err != nil {
		return Entry{}, fmt.Errorf("spectre: write %s: %w", file, err)
	}
	return e, nil
}

var (
	reAnyID    = regexp.MustCompile(`^(?:` + fullReqPattern + `|OD-\d{3}|T-M\d-\d{2}|S-\d{2}|DE-[A-Z0-9]+)$`)
	reEntryHdr = regexp.MustCompile(`^## (\S+) · (\S+)\s*$`)
)

// Read parses all entries of a review file (missing file: none).
func (st *Store) Read(file string) ([]Entry, error) {
	b, err := os.ReadFile(filepath.Join(st.Dir, file))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("spectre: read %s: %w", file, err)
	}
	return ParseEntries(string(b)), nil
}

// ParseEntries parses the review-file format written by Append.
func ParseEntries(md string) []Entry {
	var out []Entry
	var cur *Entry
	var f *Field
	for _, l := range strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n") {
		if m := reEntryHdr.FindStringSubmatch(l); m != nil {
			if cur != nil {
				out = append(out, *cur)
			}
			cur, f = &Entry{ID: m[1], Time: m[2]}, nil
			continue
		}
		if cur == nil {
			continue
		}
		if m := reFieldLine.FindStringSubmatch(l); m != nil {
			cur.Fields = append(cur.Fields, Field{Name: m[1], MD: m[2]})
			f = &cur.Fields[len(cur.Fields)-1]
			continue
		}
		if f != nil && (strings.HasPrefix(l, ">")) {
			t := strings.TrimPrefix(strings.TrimPrefix(l, ">"), " ")
			if f.MD == "" {
				f.MD = t
			} else {
				f.MD += "\n" + t
			}
		}
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out
}

// Review is everything read from the review files, grouped by ID.
type Review struct {
	Answers  map[string][]Entry
	Comments map[string][]Entry
	Feedback map[string][]Entry
}

// Verdict returns the latest verdict for id ("" if unreviewed).
func (r *Review) Verdict(id string) string {
	fb := r.Feedback[id]
	for i := len(fb) - 1; i >= 0; i-- {
		if v := NormVerdict(fb[i].Get("Verdict")); v != "" {
			return v
		}
	}
	return ""
}

// Answered reports whether an OD has an answer awaiting fold-in.
func (r *Review) Answered(id string) bool { return len(r.Answers[id]) > 0 }

// Load reads the three review files.
func (st *Store) Load() (*Review, error) {
	r := &Review{}
	for _, x := range []struct {
		file string
		dst  *map[string][]Entry
	}{{AnswersFile, &r.Answers}, {CommentsFile, &r.Comments}, {FeedbackFile, &r.Feedback}} {
		es, err := st.Read(x.file)
		if err != nil {
			return nil, err
		}
		m := map[string][]Entry{}
		for _, e := range es {
			m[e.ID] = append(m[e.ID], e)
		}
		*x.dst = m
	}
	return r, nil
}
