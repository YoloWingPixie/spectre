package audit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestReportRejectsFieldAliases(t *testing.T) {
	for _, alias := range []string{"reporoot", "endline"} {
		t.Run(alias, func(t *testing.T) {
			var r map[string]any
			if err := json.Unmarshal(mustRead(t, "templates/report.starter.json"), &r); err != nil {
				t.Fatal(err)
			}
			if alias == "reporoot" {
				r[alias] = "relative"
			} else {
				r["findings"].([]any)[0].(map[string]any)["location"].(map[string]any)[alias] = 0
			}
			doc, err := parseReport(mustJSON(t, r), ".", false)
			if err != nil {
				t.Fatal(err)
			}
			if err := doc.ready(); err == nil || !strings.Contains(err.Error(), alias) {
				t.Fatalf("alias accepted: %v", err)
			}
		})
	}
}

func TestReportNumbersRemainExact(t *testing.T) {
	for _, test := range []struct {
		value string
		want  int64
		valid bool
	}{
		{"42", 42, true}, {"42.0", 42, true}, {"4.2e1", 42, true}, {"420e-1", 42, true},
		{"9007199254740993", 9007199254740993, strconv.IntSize == 64},
		{"9223372036854775807", 9223372036854775807, strconv.IntSize == 64},
		{"1.0000000000000001", 0, false}, {"1e-400", 0, false}, {"9223372036854775808", 0, false},
	} {
		t.Run(test.value, func(t *testing.T) {
			data := bytes.Replace(mustRead(t, "templates/report.starter.json"), []byte(`"line": 42`), []byte(`"line": `+test.value), 1)
			data = bytes.Replace(data, []byte(`"endLine": 58`), []byte(`"endLine": `+test.value), 1)
			doc, err := parseReport(data, ".", false)
			if err == nil {
				err = doc.ready()
			}
			if !test.valid {
				if err == nil {
					t.Fatal("invalid or unrepresentable integer accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if int64(doc.Report.Findings[0].Location.Line) != test.want {
				t.Fatalf("line changed: %d", doc.Report.Findings[0].Location.Line)
			}
			if doc.Revision != digest(data) || !bytes.Equal(doc.Raw, data) {
				t.Fatal("normalization changed original report bytes or revision")
			}
		})
	}
}

func TestSchemaBoundaryPreservesCompatibility(t *testing.T) {
	data := mustRead(t, "templates/report.starter.json")
	if _, _, err := validateJSON(append(bytes.Clone(data), []byte(` {}`)...), "report.schema.json", false); err == nil {
		t.Fatal("trailing JSON accepted")
	}
	var r map[string]any
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	r["extension"] = "An unknown field"
	doc, err := parseReport(mustJSON(t, r), ".", false)
	if err != nil || doc.ready() != nil || len(doc.Warnings) != 1 {
		t.Fatalf("ordinary unknown field compatibility lost: %+v, %v", doc.validation, err)
	}
	_, created := fixture(t)
	state := mustJSON(t, created.ProjectState)
	state = bytes.Replace(state, []byte(`"version": 1`), []byte(`"version": 1.0`), 1)
	if _, err := parseState(state); err != nil {
		t.Fatalf("integral version rejected: %v", err)
	}
}

func TestTouchReferencesUseDecimalLines(t *testing.T) {
	for _, text := range []string{"src/auth.go:010", "src/auth.go:08-010"} {
		ref, ok := parseTouch(text)
		want := 10
		if strings.Contains(text, "08") {
			want = 8
		}
		if !ok || ref.Line != want || strings.Contains(text, "-") && ref.EndLine != 10 {
			t.Fatalf("%s parsed as %+v (valid=%v)", text, ref, ok)
		}
	}
}

func TestInitializationResponseKeepsSelectedAudit(t *testing.T) {
	options, _ := fixture(t)
	selected, err := initializeAudit(t.Context(), options, "", true)
	if err != nil {
		t.Fatal(err)
	}
	first, err := loadAudit(options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := initAudit(t.Context(), options, "", true)
	if err != nil {
		t.Fatal(err)
	}
	response := selected.result()
	if response.AuditID != first.Audit.ID || response.Paths.Report != first.Paths.Report || response.AuditID == second.AuditID {
		t.Fatal("initialization response switched to a later active audit")
	}
}

func TestCopiedProjectRejectedBeforeMutation(t *testing.T) {
	for _, operation := range []string{"init", "checkpoint"} {
		t.Run(operation, func(t *testing.T) {
			options, created := fixture(t)
			copyRoot := filepath.Join(filepath.Dir(options.Project), "copy")
			statePath := filepath.Join(copyRoot, stateFile)
			before := mustRead(t, filepath.Join(options.Project, stateFile))
			mustWrite(t, statePath, before)
			index, err := indexPath("")
			if err != nil {
				t.Fatal(err)
			}
			beforeIndex := mustRead(t, index)
			entries, err := os.ReadDir(filepath.Dir(created.Paths.Report))
			if err != nil {
				t.Fatal(err)
			}
			auditRoot := filepath.Dir(filepath.Dir(created.Paths.Report))
			audits, err := os.ReadDir(auditRoot)
			if err != nil {
				t.Fatal(err)
			}
			copied := auditOptions{Project: copyRoot}
			if operation == "init" {
				_, err = initAudit(t.Context(), copied, "", true)
			} else {
				_, err = checkpointAudit(t.Context(), copied, checkpointInput{Completed: []string{"Changed"}, NextSteps: []string{}}, digest(before))
			}
			if err == nil || !strings.Contains(err.Error(), "Project ID also exists") {
				t.Fatalf("expected identity conflict, got %v", err)
			}
			if !bytes.Equal(before, mustRead(t, statePath)) {
				t.Error("rejected operation changed project state")
			}
			if !bytes.Equal(beforeIndex, mustRead(t, index)) {
				t.Error("rejected operation changed the project index")
			}
			after, err := os.ReadDir(auditRoot)
			if err != nil {
				t.Fatal(err)
			}
			if len(after) != len(audits) {
				t.Error("rejected operation created audit files")
			}
			afterEntries, err := os.ReadDir(filepath.Dir(created.Paths.Report))
			if err != nil || len(afterEntries) != len(entries) {
				t.Fatal("original audit files changed", err)
			}
		})
	}
}
