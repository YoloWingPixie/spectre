package audit

import (
	"bytes"
	"io"
	"path/filepath"
	"regexp"
	"testing"
)

func TestStandaloneReportIdentity(t *testing.T) {
	directory := t.TempDir()
	source := mustRead(t, starterReport)
	first, second := filepath.Join(directory, "first.json"), filepath.Join(directory, "second.json")
	mustWrite(t, first, source)
	mustWrite(t, second, source)
	build := func(input string) string {
		t.Helper()
		output := filepath.Join(directory, "report.html")
		if err := buildCommand([]string{input, "-o", output}, io.Discard, io.Discard); err != nil {
			t.Fatal(err)
		}
		match := regexp.MustCompile(`data-report-id="([^"]+)"`).FindSubmatch(mustRead(t, output))
		if len(match) != 2 {
			t.Fatal("missing storage identity")
		}
		return string(match[1])
	}
	identity := build(first)
	if identity == build(second) {
		t.Fatal("separate legacy source files share identity")
	}
	if !bytes.Equal(source, mustRead(t, first)) {
		t.Fatal("build changed legacy source")
	}
	mustWrite(t, first, bytes.Replace(source, []byte("Project Audit"), []byte("Edited title"), 1))
	if identity != build(first) {
		t.Fatal("editing legacy report changed identity")
	}
	explicit := bytes.Replace(source, []byte(`"title":`), []byte(`"reportId":"stable-report", "title":`), 1)
	mustWrite(t, first, explicit)
	mustWrite(t, second, explicit)
	if build(first) != "stable-report" || build(second) != "stable-report" {
		t.Fatal("explicit identity lost across paths")
	}
}

func TestInitializedReportHasPersistentIdentity(t *testing.T) {
	directory := t.TempDir()
	first, second := filepath.Join(directory, "one.json"), filepath.Join(directory, "two.json")
	for _, path := range []string{first, second} {
		if err := initReport(path, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	a, err := readReport(first, false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := readReport(second, false)
	if err != nil {
		t.Fatal(err)
	}
	if a.Report.ID == "" || a.Report.ID == b.Report.ID {
		t.Fatal("starter identities missing or shared")
	}
	_, initialized := fixture(t)
	persisted, err := readReport(initialized.Paths.Report, true)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Report.ID == "" || string(persisted.Report.ID) != string(initialized.AuditID) {
		t.Fatal("audit report identity not persisted")
	}
}
