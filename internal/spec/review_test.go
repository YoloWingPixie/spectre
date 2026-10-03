package spec

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func fixedStore(t *testing.T) *Store {
	t.Helper()
	st := NewStore(filepath.Join(t.TempDir(), "review"))
	st.now = func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC) }
	return st
}

func TestAppendAndRead(t *testing.T) {
	tests := []struct {
		name   string
		file   string
		id     string
		fields []Field
		want   []Field // after round trip (empty values dropped)
	}{
		{"feedback", FeedbackFile, "ATC-FR-IDN-001", []Field{{"Verdict", "Change"}, {"Text", "line 1\n## not a header\n> quoted"}},
			[]Field{{"Verdict", "Change"}, {"Text", "line 1\n## not a header\n> quoted"}}},
		{"comment", CommentsFile, "S-01", []Field{{"Comment", "short"}}, []Field{{"Comment", "short"}}},
		{"answer", AnswersFile, "OD-012", []Field{{"Choice", "(b) keep"}, {"Note", ""}}, []Field{{"Choice", "(b) keep"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := fixedStore(t)
			if _, err := st.Append(tt.file, tt.id, tt.fields); err != nil {
				t.Fatal(err)
			}
			if _, err := st.Append(tt.file, tt.id, tt.fields); err != nil {
				t.Fatal(err)
			}
			es, err := st.Read(tt.file)
			if err != nil {
				t.Fatal(err)
			}
			if len(es) != 2 {
				t.Fatalf("got %d entries, want 2", len(es))
			}
			e := es[1]
			if e.ID != tt.id || e.Time != "2026-09-30T12:00:00Z" {
				t.Errorf("entry = %s %s", e.ID, e.Time)
			}
			if len(e.Fields) != len(tt.want) {
				t.Fatalf("fields = %+v, want %+v", e.Fields, tt.want)
			}
			for i, f := range tt.want {
				if e.Fields[i] != f {
					t.Errorf("field %d = %+v, want %+v", i, e.Fields[i], f)
				}
			}
			b, _ := os.ReadFile(filepath.Join(st.Dir, tt.file))
			if strings.Count(string(b), "\n# ") > 0 || !strings.HasPrefix(string(b), "# ") {
				t.Errorf("file title missing or repeated:\n%s", b)
			}
		})
	}
}

func TestAppendRejectsBadID(t *testing.T) {
	st := fixedStore(t)
	for _, id := range []string{"", "x\n## OD-001", "../etc", "OD-12"} {
		if _, err := st.Append(CommentsFile, id, []Field{{"Comment", "x"}}); err == nil {
			t.Errorf("Append(%q) succeeded", id)
		}
	}
}

func TestAppendConcurrent(t *testing.T) {
	st := fixedStore(t)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := st.Append(CommentsFile, "S-01", []Field{{"Comment", "a\nb"}}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	es, _ := st.Read(CommentsFile)
	if len(es) != 20 {
		t.Errorf("got %d entries, want 20", len(es))
	}
}

func TestLatestVerdictWins(t *testing.T) {
	st := fixedStore(t)
	for _, v := range []string{"Reject", "Question", "Accept"} {
		if _, err := st.Append(FeedbackFile, "T-M1-01", []Field{{"Verdict", v}}); err != nil {
			t.Fatal(err)
		}
	}
	r, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Verdict("T-M1-01"); got != "Accept" {
		t.Errorf("verdict = %q, want Accept", got)
	}
	if n := len(r.Feedback["T-M1-01"]); n != 3 {
		t.Errorf("history = %d, want 3", n)
	}
	if r.Verdict("S-01") != "" || r.Answered("OD-001") {
		t.Error("unreviewed item has a verdict or answer")
	}
}
