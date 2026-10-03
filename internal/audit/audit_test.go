package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func fixture(t *testing.T) (auditOptions, resumeResult) {
	t.Helper()
	directory := t.TempDir()
	project := filepath.Join(directory, "my project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(directory, "cache"))
	t.Setenv("AUDIT_REPORT_HOME", filepath.Join(directory, "index"))
	options := auditOptions{Project: project}
	result, err := initAudit(t.Context(), options, "templates/report.starter.json", false)
	if err != nil {
		t.Fatal(err)
	}
	return options, result
}
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := encode(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestImportedReportsBuildWithoutExternalRuntime(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	for _, path := range []string{"templates/report.starter.json", "testdata/example/report.json"} {
		t.Run(path, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "report.html")
			var diagnostics bytes.Buffer
			if err := Run(t.Context(), []string{"build", path, "--strict", "-o", output}, io.Discard, &diagnostics); err != nil {
				t.Fatal(err, diagnostics.String())
			}
			html := string(mustRead(t, output))
			if !strings.HasPrefix(html, "<title>") || strings.Contains(html, "<html>") || strings.Contains(html, "#ZgotmplZ") {
				t.Fatal("invalid standalone page contract")
			}
			for _, contract := range []string{`class="row st-`, `data-kind="sev"`, `data-kind="status"`, `data-fb="coaNote"`, `data-fb="reason"`, `id="fb-copy"`, `vscode://vscode-remote/`, `function contentOf(`, `function editorHref(`} {
				if !strings.Contains(html, contract) {
					t.Errorf("page lacks %s", contract)
				}
			}
			if path == "testdata/example/report.json" && !strings.Contains(html, "data:image/jpeg;base64,") {
				t.Error("example screenshot was not embedded")
			}
		})
	}
}

func TestValidationPreventsInvalidBuilds(t *testing.T) {
	data := mustRead(t, "templates/report.starter.json")
	tests := []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{"duplicate findings", func(r map[string]any) { fs := r["findings"].([]any); r["findings"] = append(fs, fs[0]) }, "duplicate finding ID"},
		{"unknown area", func(r map[string]any) { r["findings"].([]any)[0].(map[string]any)["area"] = "missing" }, "unknown area"},
		{"invalid path", func(r map[string]any) {
			r["findings"].([]any)[0].(map[string]any)["location"].(map[string]any)["path"] = "../secret"
		}, "relative path"},
		{"missing command output", func(r map[string]any) {
			delete(r["findings"].([]any)[0].(map[string]any)["evidence"].([]any)[1].(map[string]any), "output")
		}, "output"},
		{"duplicate recommended options", func(r map[string]any) {
			c := r["findings"].([]any)[0].(map[string]any)["coas"].([]any)[1].(map[string]any)
			c["recommended"] = true
			c["recommendedReason"] = "Another recommendation."
		}, "more than one recommended"},
		{"unknown related finding", func(r map[string]any) {
			r["findings"].([]any)[0].(map[string]any)["relatedIds"] = []string{"NO-SUCH-ID"}
		}, "unknown related"},
		{"unknown field", func(r map[string]any) { r["scop"] = "Typo" }, "unknown field"},
		{"wordy prose", func(r map[string]any) { r["scope"] = "We utilize a token." }, "utilize"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var r map[string]any
			if err := json.Unmarshal(data, &r); err != nil {
				t.Fatal(err)
			}
			test.mutate(r)
			path := filepath.Join(t.TempDir(), "invalid.json")
			mustWrite(t, path, mustJSON(t, r))
			var diagnostics bytes.Buffer
			err := Run(t.Context(), []string{"build", path, "--strict"}, io.Discard, &diagnostics)
			if err == nil || !strings.Contains(err.Error()+diagnostics.String(), test.want) {
				t.Fatalf("error = %v; diagnostics = %s", err, diagnostics.String())
			}
			if _, err := os.Stat(strings.TrimSuffix(path, ".json") + ".html"); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("invalid report wrote HTML")
			}
		})
	}
}

func TestRenderingEscapesReportText(t *testing.T) {
	doc, err := readReport("templates/report.starter.json", false)
	if err != nil {
		t.Fatal(err)
	}
	attack := `</script><script>alert("audit")</script><img src=x onerror=alert(1)>`
	doc.Report.Title = attack
	doc.Report.Findings[0].Summary = attack
	doc.Report.Findings[0].Evidence[0].Excerpt = attack
	doc.Report.Findings[0].COAs[0].Name = attack
	html, err := renderReport(doc.Report, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(html, []byte(attack)) || bytes.Contains(html, []byte(`onerror=alert(1)>`)) {
		t.Fatal("report text became markup")
	}
	if !bytes.Contains(html, []byte("&lt;script&gt;")) {
		t.Fatal("escaped report text is missing")
	}
}

func TestResumeCheckpointAndAuditIsolation(t *testing.T) {
	options, created := fixture(t)
	ctx := t.Context()
	again, err := initAudit(ctx, options, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if again.AuditID != created.AuditID || again.ProjectRevision != created.ProjectRevision {
		t.Fatal("init did not resume idempotently")
	}
	input := checkpointInput{Completed: []string{"Read the source"}, NextSteps: []string{"Review decisions"}, Status: awaitingReview}
	saved, err := checkpointAudit(ctx, options, input, created.ProjectRevision)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Audit.Status != awaitingReview || saved.Audit.Checkpoint.NextSteps[0] != "Review decisions" {
		t.Fatal("checkpoint was not saved")
	}
	before := mustRead(t, filepath.Join(options.Project, stateFile))
	if _, err := checkpointAudit(ctx, options, input, created.ProjectRevision); err == nil {
		t.Fatal("stale checkpoint accepted")
	}
	if !bytes.Equal(before, mustRead(t, filepath.Join(options.Project, stateFile))) {
		t.Fatal("stale save replaced state")
	}
	second, err := initAudit(ctx, options, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if second.AuditID == created.AuditID || len(second.ProjectState.Audits) != 2 {
		t.Fatal("new audit was not independent")
	}
	options.AuditID = created.AuditID
	old, err := resumeAudit(ctx, options)
	if err != nil {
		t.Fatal(err)
	}
	if old.Audit.Checkpoint.NextSteps[0] != "Review decisions" {
		t.Fatal("first checkpoint was overwritten")
	}
	projects, err := listProjects()
	if err != nil {
		t.Fatal(err)
	}
	data := mustJSON(t, projects)
	if !bytes.Contains(data, []byte(created.ProjectID)) {
		t.Fatal("project discovery did not persist")
	}
}

func TestLegacyStateAndFeedbackArePreserved(t *testing.T) {
	directory := t.TempDir()
	project := filepath.Join(directory, "legacy")
	mustWrite(t, filepath.Join(project, "file.txt"), []byte("source"))
	t.Setenv("AUDIT_REPORT_HOME", filepath.Join(directory, "index"))
	const projectID = "a5c418f5-19c6-44c4-a102-b5975f6ab125"
	const auditID = "cb4249e0-29eb-4ee0-8a2c-501764fab011"
	dir := filepath.Join(directory, "legacy.audits", auditID)
	reportData := mustRead(t, "templates/report.starter.json")
	mustWrite(t, filepath.Join(dir, reportFile), reportData)
	state := fmt.Sprintf(`{"version":1,"projectId":%q,"dataDir":"../legacy.audits","activeAuditId":%q,"audits":[{"id":%q,"status":"awaiting-review","checkpoint":{"updatedAt":"2026-09-26T18:00:00.000Z","completed":["Read authentication"],"nextSteps":["Review feedback"],"repository":{"head":null,"fingerprint":null},"reportRevision":%q}}]}`, projectID, auditID, auditID, digest(reportData))
	mustWrite(t, filepath.Join(project, stateFile), []byte(state))
	feedback := fmt.Sprintf(`{"version":1,"projectId":%q,"auditId":%q,"findings":{"SEC-01":{"findingId":"SEC-01","decision":{"type":"coa","coaId":"expire-one-hour"},"coaNotes":{"expire-one-hour":"Keep this note"},"note":"Resume tomorrow","updatedAt":"2026-09-26T18:00:00.000Z","updatedBy":{"id":"reader"},"notify":{"threadId":"legacy-thread"}}}}`, projectID, auditID)
	mustWrite(t, filepath.Join(dir, feedbackFilename), []byte(feedback))
	result, err := resumeAudit(t.Context(), auditOptions{Project: project})
	if err != nil {
		t.Fatal(err)
	}
	if string(result.ProjectID) != projectID || string(result.AuditID) != auditID || result.Feedback["SEC-01"].Note != "Resume tomorrow" || result.Feedback["SEC-01"].Decision.COAID != "expire-one-hour" {
		t.Fatal("legacy identities or feedback changed")
	}
	if !bytes.Equal([]byte(state), mustRead(t, filepath.Join(project, stateFile))) || !bytes.Equal([]byte(feedback), mustRead(t, filepath.Join(dir, feedbackFilename))) {
		t.Fatal("resume rewrote legacy state")
	}
}

func TestImportIsIdempotentAndProtectsConflicts(t *testing.T) {
	options, _ := fixture(t)
	input := []byte(`{"findings":{"SEC-01":{"findingId":"SEC-01","decision":{"type":"coa","coaId":"expire-one-hour"},"coaNotes":{},"note":"Keep me"}}}`)
	if _, err := importFeedback(options, input, false); err != nil {
		t.Fatal(err)
	}
	c, err := loadAudit(options)
	if err != nil {
		t.Fatal(err)
	}
	before := mustRead(t, c.Paths.Feedback)
	if _, err := importFeedback(options, input, false); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, mustRead(t, c.Paths.Feedback)) {
		t.Fatal("identical import changed records")
	}
	changed := bytes.ReplaceAll(input, []byte("Keep me"), []byte("Replace me"))
	if _, err := importFeedback(options, changed, false); err == nil {
		t.Fatal("conflicting import was accepted")
	}
	if !bytes.Equal(before, mustRead(t, c.Paths.Feedback)) {
		t.Fatal("conflicting import replaced data")
	}
	if _, err := importFeedback(options, changed, true); err != nil {
		t.Fatal(err)
	}
	foreign := []byte(`{"projectId":"some-other-project","findings":{}}`)
	if _, err := importFeedback(options, foreign, true); err == nil {
		t.Fatal("foreign feedback was accepted")
	}
}

func TestViewerSavesFeedbackAndRejectsUnsafeRequests(t *testing.T) {
	options, _ := fixture(t)
	v, err := startViewer(options, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := v.close(); err != nil {
			t.Error(err)
		}
	})
	response, err := http.Get(v.URL)
	if err != nil {
		t.Fatal(err)
	}
	page, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 {
		t.Fatal("viewer page failed", err)
	}
	tokenMatch := regexp.MustCompile(`data-feedback-token="([^"]+)"`).FindSubmatch(page)
	if len(tokenMatch) != 2 {
		t.Fatal("page lacks feedback token")
	}
	token := string(tokenMatch[1])
	origin := strings.TrimSuffix(v.URL, "/")
	state, err := getFeedback(options)
	if err != nil {
		t.Fatal(err)
	}
	input := feedbackUpdate{FindingID: "SEC-01", Content: json.RawMessage(`{"decision":{"type":"coa","coaId":"expire-one-hour"},"coaNotes":{},"note":"Saved from the browser"}`), ReportRevision: state.ReportRevision}
	request := func(method, host, requestOrigin, requestToken string, body []byte) int {
		t.Helper()
		req, err := http.NewRequest(method, v.URL+"feedback", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		if host != "" {
			req.Host = host
		}
		req.Header.Set("Origin", requestOrigin)
		req.Header.Set(feedbackTokenHeader, requestToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)
		return resp.StatusCode
	}
	for _, test := range []struct {
		name, method, host, origin, token string
		body                              []byte
		code                              int
	}{
		{"foreign host", "PUT", "evil.example", origin, token, mustJSON(t, input), 403},
		{"foreign origin", "PUT", "", "https://evil.example", token, mustJSON(t, input), 403},
		{"no origin", "PUT", "", "", token, mustJSON(t, input), 403},
		{"no token", "GET", "", "", "", nil, 403},
		{"malformed JSON", "PUT", "", origin, token, []byte("{"), 400},
		{"missing revision", "PUT", "", origin, token, []byte(`{"findingId":"SEC-01","content":{},"reportRevision":"x"}`), 422},
		{"oversize body", "PUT", "", origin, token, bytes.Repeat([]byte("x"), feedbackBodyLimit+1), 413},
		{"unsupported method", "POST", "", origin, token, nil, 405},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := request(test.method, test.host, test.origin, test.token, test.body); got != test.code {
				t.Fatalf("status %d, want %d", got, test.code)
			}
		})
	}
	if got := request("PUT", "", origin, token, mustJSON(t, input)); got != 200 {
		t.Fatalf("save status %d", got)
	}
	resumed, err := resumeAudit(t.Context(), options)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Feedback["SEC-01"].Note != "Saved from the browser" {
		t.Fatal("HTTP feedback did not persist")
	}
	before := mustRead(t, resumed.Paths.Feedback)
	if got := request("PUT", "", origin, token, mustJSON(t, input)); got != 409 {
		t.Fatal("stale save status", got)
	}
	if !bytes.Equal(before, mustRead(t, resumed.Paths.Feedback)) {
		t.Fatal("stale HTTP save replaced feedback")
	}
	state, err = getFeedback(options)
	if err != nil {
		t.Fatal(err)
	}
	input.ExpectedRevision = state.Revisions["SEC-01"]
	input.ReportRevision = "stale-report"
	if got := request("PUT", "", origin, token, mustJSON(t, input)); got != 409 {
		t.Fatal("stale report status", got)
	}
	if err := v.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := startViewer(options, 0)
	if err != nil {
		t.Fatal("viewer lock was not released", err)
	}
	if err := reopened.close(); err != nil {
		t.Fatal(err)
	}
}

func TestSavedRemovedOptionsSurviveNewNotes(t *testing.T) {
	options, result := fixture(t)
	if _, err := importFeedback(options, []byte(`{"findings":{"SEC-01":{"decision":{"type":"coa","coaId":"removed"},"coaNotes":{"removed":"Keep the old choice"},"note":"Earlier note"}}}`), false); err != nil {
		t.Fatal(err)
	}
	state, err := getFeedback(options)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Review) != 2 {
		t.Fatal("removed option feedback was not identified")
	}
	_, err = saveFeedback(options, feedbackUpdate{FindingID: "SEC-01", Content: json.RawMessage(`{"decision":null,"coaNotes":{},"note":"New note"}`), ExpectedRevision: state.Revisions["SEC-01"], ReportRevision: state.ReportRevision})
	if err != nil {
		t.Fatal(err)
	}
	feedback := mustRead(t, result.Paths.Feedback)
	if !bytes.Contains(feedback, []byte("Keep the old choice")) || !bytes.Contains(feedback, []byte(`"coaId": "removed"`)) {
		t.Fatal("removed option feedback was erased")
	}
}

func TestLockedWritesPreserveData(t *testing.T) {
	options, result := fixture(t)
	release, err := acquireLock(result.Paths.Feedback + ".lock")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	before := mustRead(t, result.Paths.Feedback)
	if _, err := importFeedback(options, []byte(`{"findings":{"SEC-01":{"note":"new"}}}`), false); err == nil {
		t.Fatal("locked write succeeded")
	}
	if !bytes.Equal(before, mustRead(t, result.Paths.Feedback)) {
		t.Fatal("locked write changed data")
	}
}

func TestRepositoryFingerprintDetectsUncommittedChanges(t *testing.T) {
	options, result := fixture(t)
	if err := exec.Command("git", "init", "--quiet", options.Project).Run(); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(options.Project, "source.go"), []byte("first"))
	first, err := repositoryState(t.Context(), options.Project)
	if err != nil || first.Fingerprint == nil {
		t.Fatal("Git fingerprint unavailable", err)
	}
	input := checkpointInput{Completed: []string{}, NextSteps: []string{"Continue"}}
	saved, err := checkpointAudit(t.Context(), options, input, result.ProjectRevision)
	if err != nil {
		t.Fatal(err)
	}
	if saved.RepositoryChanged == nil || *saved.RepositoryChanged {
		t.Fatal("checkpoint fingerprint changed from its own state write")
	}
	mustWrite(t, filepath.Join(options.Project, "source.go"), []byte("other"))
	changed, err := resumeAudit(t.Context(), options)
	if err != nil {
		t.Fatal(err)
	}
	if changed.RepositoryChanged == nil || !*changed.RepositoryChanged {
		t.Fatal("uncommitted content change was missed")
	}
}

func TestViewerCommandCancellationReleasesLock(t *testing.T) {
	options, result := fixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	reader, writer := io.Pipe()
	defer reader.Close()
	done := make(chan error, 1)
	go func() {
		err := Run(ctx, []string{"view", "--project", options.Project}, writer, io.Discard)
		writer.Close()
		done <- err
	}()
	var resultURL map[string]string
	if err := json.NewDecoder(reader).Decode(&resultURL); err != nil {
		t.Fatal(err)
	}
	if _, err := url.ParseRequestURI(resultURL["url"]); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(result.Paths.Report), ".viewer.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("viewer lock survived cancellation")
	}
}

func TestSetupPreservesExistingSkillsAndWorksWithoutNode(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(directory, "cache"))
	skills := filepath.Join(directory, "skills")
	legacy := filepath.Join(skills, "audit-report", "SKILL.md")
	mustWrite(t, legacy, []byte("Legacy registration"))
	var out bytes.Buffer
	args := []string{"setup", "--agent", "codex", "--skills-dir", skills}
	if err := Run(t.Context(), args, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(skills, skillName)
	target, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "schema/report.schema.json")); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(mustRead(t, filepath.Join(target, "SKILL.md")), []byte("spectre audit resume")) {
		t.Fatal("installed instructions do not use Spectre")
	}
	if err := Run(t.Context(), args, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Already installed") {
		t.Fatal("setup is not idempotent")
	}
	if err := Run(t.Context(), append(args, "--remove"), &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, legacy)) != "Legacy registration" {
		t.Fatal("setup changed legacy registration")
	}
	mustWrite(t, filepath.Join(link, "SKILL.md"), []byte("User instructions"))
	if err := Run(t.Context(), append(args, "--replace"), &out, io.Discard); err == nil {
		t.Fatal("setup replaced a real directory")
	}
	if string(mustRead(t, filepath.Join(link, "SKILL.md"))) != "User instructions" {
		t.Fatal("setup changed existing instructions")
	}
}

func TestAbsoluteScreenshotImportKeepsTheLegacyContract(t *testing.T) {
	options, _ := fixture(t)
	doc, err := readReport("templates/report.starter.json", true)
	if err != nil {
		t.Fatal(err)
	}
	image, err := filepath.Abs("testdata/example/shots/wiki-desktop-night.jpg")
	if err != nil {
		t.Fatal(err)
	}
	doc.Report.Findings[0].Evidence = append(doc.Report.Findings[0].Evidence, evidence{Type: screenshotEvidence, Path: image, Caption: "Imported screenshot"})
	source := filepath.Join(t.TempDir(), "report.json")
	mustWrite(t, source, mustJSON(t, doc.Report))
	imported, err := initAudit(t.Context(), options, source, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(imported.Paths.Report), "screenshots/1.jpg")); err != nil {
		t.Fatal(err)
	}
}

func TestProjectMoveKeepsSavedDecisions(t *testing.T) {
	options, created := fixture(t)
	if _, err := importFeedback(options, []byte(`{"findings":{"SEC-01":{"note":"Keep this after moving"}}}`), false); err != nil {
		t.Fatal(err)
	}
	renamed := filepath.Join(filepath.Dir(options.Project), "renamed project")
	if err := os.Rename(options.Project, renamed); err != nil {
		t.Fatal(err)
	}
	options.Project = renamed
	resumed, err := resumeAudit(t.Context(), options)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.AuditID != created.AuditID || resumed.Feedback["SEC-01"].Note != "Keep this after moving" || !resumed.PathChanged {
		t.Fatal("move lost identity, decisions, or path-change detection")
	}
}

func TestInvalidStoredJSONIsNeverReplaced(t *testing.T) {
	for _, original := range [][]byte{{}, []byte("{broken")} {
		t.Run(fmt.Sprintf("%d bytes", len(original)), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state.json")
			mustWrite(t, path, original)
			_, err := updateFile(path, func(previous []byte) ([]byte, error) { return []byte(`{"replacement":true}`), nil })
			if err == nil {
				t.Fatal("invalid stored JSON was replaced")
			}
			if !bytes.Equal(original, mustRead(t, path)) {
				t.Fatal("failed save changed stored data")
			}
		})
	}
}

func TestMalformedFeedbackCannotReplaceSavedNotes(t *testing.T) {
	for _, input := range []string{
		`{"findings":{"SEC-01":{"coaNotes":{"expire-one-hour":null}}}}`,
		`{"findings":{"SEC-01":{"Note":null}}}`,
		`{"findings":{"SEC-01":{"COANotes":{"expire-one-hour":null}}}}`,
		`{"findings":{"SEC-01":{"findingId":"","note":"Wrong identity"}}}`,
		`{"findings":{"SEC-01":{"FindingId":"SEC-02","note":"Wrong identity"}}}`,
		`{"findings":{"SEC-01":{"findingId":"SEC-01","FindingId":"SEC-02"}}}`,
		`{"findings":{"SEC-01":{"findingId":"SEC-02","FindingId":"SEC-01"}}}`,
		`{"findings":{"SEC-01":{"FindingId":null}}}`,
		`{"findings":{"SEC-01":{"FindingId":42}}}`,
	} {
		t.Run(input, func(t *testing.T) {
			options, result := fixture(t)
			if _, err := importFeedback(options, []byte(`{"findings":{"SEC-01":{"note":"Preserve this note"}}}`), false); err != nil {
				t.Fatal(err)
			}
			before := mustRead(t, result.Paths.Feedback)
			if _, err := importFeedback(options, []byte(input), true); err == nil {
				t.Error("malformed feedback was accepted")
			}
			if !bytes.Equal(before, mustRead(t, result.Paths.Feedback)) {
				t.Error("malformed import replaced saved notes")
			}
		})
	}
}

func TestFeedbackCaseVariantsRetainValidNotes(t *testing.T) {
	options, _ := fixture(t)
	result, err := importFeedback(options, []byte(`{"findings":{"SEC-01":{"findingId":"SEC-01","FindingId":"SEC-01","Note":" Keep this ","COANotes":{"expire-one-hour":" Keep this option "},"updatedBy":{"id":"reader"}}}}`), false)
	if err != nil {
		t.Fatal(err)
	}
	record := result.Findings["SEC-01"]
	if record.Note != "Keep this" || record.COANotes["expire-one-hour"] != "Keep this option" {
		t.Fatal("case-variant keys lost their normalized notes")
	}
}

func TestFailedImportPreservesFeedbackWhenReportCannotBeRead(t *testing.T) {
	for _, reportState := range []string{"missing", "invalid JSON"} {
		t.Run(reportState, func(t *testing.T) {
			options, result := fixture(t)
			if _, err := importFeedback(options, []byte(`{"findings":{"SEC-01":{"note":"Keep this"}}}`), false); err != nil {
				t.Fatal(err)
			}
			before := mustRead(t, result.Paths.Feedback)
			if reportState == "missing" {
				if err := os.Remove(result.Paths.Report); err != nil {
					t.Fatal(err)
				}
			} else {
				mustWrite(t, result.Paths.Report, []byte("{broken"))
			}
			_, err := importFeedback(options, []byte(`{"findings":{"SEC-01":{"note":"Replace this"}}}`), true)
			if err == nil {
				t.Fatal("import succeeded without a readable report")
			}
			if !bytes.Equal(before, mustRead(t, result.Paths.Feedback)) {
				t.Fatal("failed import changed saved feedback")
			}
		})
	}
}

func TestReportAcceptsIntegralJSONNumbers(t *testing.T) {
	data := bytes.Replace(mustRead(t, "templates/report.starter.json"), []byte(`"line": 42`), []byte(`"line": 42.0`), 1)
	doc, err := parseReport(data, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ready(); err != nil {
		t.Fatal(err)
	}
	if doc.Report.Findings[0].Location.Line != 42 {
		t.Fatal("integral JSON number changed")
	}
}
