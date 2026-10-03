package audit

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckpointRejectsUnreadableFeedbackBeforeSaving(t *testing.T) {
	options, created := fixture(t)
	statePath := filepath.Join(options.Project, stateFile)
	before := mustRead(t, statePath)
	mustWrite(t, created.Paths.Feedback, []byte("{broken"))
	_, err := checkpointAudit(t.Context(), options, checkpointInput{Completed: []string{"New work"}, NextSteps: []string{}, Status: complete}, created.ProjectRevision)
	if err == nil {
		t.Fatal("checkpoint accepted corrupt feedback")
	}
	if !bytes.Equal(before, mustRead(t, statePath)) {
		t.Fatal("failed checkpoint changed saved state")
	}
}

func TestFirstInitializationValidatesIndexBeforeCreatingAudit(t *testing.T) {
	for _, indexData := range []string{"", "{broken", `{"version":9,"projects":[]}`} {
		t.Run(indexData, func(t *testing.T) {
			dir := t.TempDir()
			project := filepath.Join(dir, "project")
			if err := os.Mkdir(project, 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("AUDIT_REPORT_HOME", filepath.Join(dir, "index"))
			t.Setenv("XDG_CACHE_HOME", filepath.Join(dir, "cache"))
			index := filepath.Join(dir, "index", "projects.json")
			mustWrite(t, index, []byte(indexData))
			if _, err := initAudit(t.Context(), auditOptions{Project: project}, "", false); err == nil {
				t.Fatal("initialization accepted invalid index")
			}
			if _, err := os.Stat(filepath.Join(project, stateFile)); !os.IsNotExist(err) {
				t.Fatal("failed initialization created state", err)
			}
			if _, err := os.Stat(filepath.Join(dir, "project.audits")); !os.IsNotExist(err) {
				t.Fatal("failed initialization created audit files", err)
			}
			if !bytes.Equal([]byte(indexData), mustRead(t, index)) {
				t.Fatal("failed initialization replaced the index")
			}
		})
	}
}

func TestCheckpointDoesNotRegisterAgainAfterCommit(t *testing.T) {
	options, created := fixture(t)
	index, err := indexPath("")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	git := filepath.Join(bin, "git")
	mustWrite(t, git, []byte(`#!/bin/sh
case "$(/bin/cat "$2/.audit-report.json")" in
  *POST_COMMIT_INDEX_LOCK*) printf '%s' test > "$SPECTRE_TEST_INDEX_LOCK" ;;
esac
exit 1
`))
	if err := os.Chmod(git, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("SPECTRE_TEST_INDEX_LOCK", index+".lock")
	result, err := checkpointAudit(t.Context(), options, checkpointInput{Completed: []string{"POST_COMMIT_INDEX_LOCK"}, NextSteps: []string{}}, created.ProjectRevision)
	if err != nil {
		t.Fatalf("response failed after checkpoint commit: %v", err)
	}
	if result.Audit.Checkpoint.Completed[0] != "POST_COMMIT_INDEX_LOCK" {
		t.Fatal("response lacks committed checkpoint")
	}
	if result.ProjectRevision != digest(mustRead(t, filepath.Join(options.Project, stateFile))) {
		t.Fatal("response revision differs from committed bytes")
	}
}
