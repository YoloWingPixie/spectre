package audit

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf16"
)

const stateFile = ".audit-report.json"
const reportFile = "report.json"
const feedbackFilename = "feedback.json"
const stateVersion = 1
const timestampLayout = "2006-01-02T15:04:05.000Z"

func digest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
func timestamp() string         { return time.Now().UTC().Format(timestampLayout) }
func uuid() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	id[6] = id[6]&0x0f | 0x40
	id[8] = id[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:]), nil
}
func encode(value any) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, missing("Cannot read %s: file not found", path)
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if !json.Valid(data) {
		return nil, invalid("Invalid JSON in %s. Repair this file; it has not been replaced.", path)
	}
	return data, nil
}

func acquireLock(path string) (func() error, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		owner, _ := os.ReadFile(path)
		return nil, conflict("Locked: %s (process %s). Retry after the writer exits. Verify it has exited before removing a stale lock.", path, strings.TrimSpace(string(owner)))
	}
	if err != nil {
		return nil, err
	}
	_, writeErr := fmt.Fprint(f, os.Getpid())
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return nil, errors.Join(err, os.Remove(path))
	}
	return func() error { return os.Remove(path) }, nil
}

func updateFile(path string, transform func([]byte) ([]byte, error)) (data []byte, err error) {
	release, err := acquireLock(path + ".lock")
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, release()) }()
	previous, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil && !json.Valid(previous) {
		return nil, invalid("Invalid JSON in %s. Repair this file; it has not been replaced.", path)
	}
	data, err = transform(previous)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(data, previous) {
		return data, nil
	}
	if !json.Valid(data) {
		return nil, invalid("Refusing to write invalid JSON to %s", path)
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".audit-write-*")
	if err != nil {
		return nil, err
	}
	temporary := f.Name()
	defer os.Remove(temporary)
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return nil, err
	}
	if err := os.Rename(temporary, path); err != nil {
		return nil, err
	}
	return data, nil
}

func expectRevision(actual string, expected *string) error {
	if (expected == nil && actual != "") || (expected != nil && *expected != actual) {
		return conflict("Save conflict: data changed. Resume or reload before applying your changes again.")
	}
	return nil
}

type auditOptions struct {
	Project  string
	AuditID  auditID
	IndexDir string
}
type auditPaths struct {
	Report   string `json:"report"`
	Feedback string `json:"feedback"`
}
type auditContext struct {
	Root          string
	StatePath     string
	State         projectState
	StateRevision string
	Audit         savedAudit
	Directory     string
	Paths         auditPaths
}

func parseState(data []byte) (projectState, error) {
	checks, normalized, err := validateJSON(data, "project-state.schema.json", true)
	if err != nil {
		return projectState{}, err
	}
	if len(checks.Errors) > 0 {
		return projectState{}, invalid("Invalid project state:\n%s", strings.Join(checks.Errors, "\n"))
	}
	var state projectState
	if err := json.Unmarshal(normalized, &state); err != nil {
		return state, err
	}
	ids := map[auditID]bool{}
	for _, audit := range state.Audits {
		if ids[audit.ID] {
			return state, invalid("Duplicate audit ID")
		}
		ids[audit.ID] = true
	}
	if !ids[state.ActiveAuditID] {
		return state, invalid("Unknown active audit ID")
	}
	return state, nil
}
func projectRoot(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	root, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", invalid("Project must be a directory")
	}
	return root, nil
}
func loadAudit(options auditOptions) (auditContext, error) {
	root, err := projectRoot(options.Project)
	if err != nil {
		return auditContext{}, err
	}
	path := filepath.Join(root, stateFile)
	data, err := readFile(path)
	if err != nil {
		return auditContext{}, err
	}
	state, err := parseState(data)
	if err != nil {
		return auditContext{}, err
	}
	return contextForAudit(root, state, digest(data), options.AuditID)
}

func contextForAudit(root string, state projectState, revision string, id auditID) (auditContext, error) {
	if id == "" {
		id = state.ActiveAuditID
	}
	for _, audit := range state.Audits {
		if audit.ID == id {
			directory := filepath.Join(root, filepath.FromSlash(state.DataDir), string(id))
			return auditContext{root, filepath.Join(root, stateFile), state, revision, audit, directory, auditPaths{filepath.Join(directory, reportFile), filepath.Join(directory, feedbackFilename)}}, nil
		}
	}
	return auditContext{}, missing("Unknown audit ID")
}

func repositoryState(ctx context.Context, project string) (repository, error) {
	git := func(args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "git", append([]string{"-C", project}, args...)...).Output()
	}
	if _, err := git("rev-parse", "--show-toplevel"); err != nil {
		return repository{}, ctx.Err()
	}
	var state repository
	if head, err := git("rev-parse", "--verify", "HEAD"); err == nil {
		value := strings.TrimSpace(string(head))
		state.Head = &value
	}
	names, err := git("ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return state, err
	}
	paths := strings.Split(string(names), "\x00")
	slices.SortFunc(paths, func(a, b string) int { return slices.Compare(utf16.Encode([]rune(a)), utf16.Encode([]rune(b))) })
	paths = slices.Compact(paths)
	hash := sha256.New()
	if state.Head == nil {
		hash.Write([]byte("unborn"))
	} else {
		hash.Write([]byte(*state.Head))
	}
	for _, name := range paths {
		if name == "" || name == stateFile || strings.HasPrefix(name, stateFile+".") {
			continue
		}
		hash.Write([]byte(name + "\x00"))
		path := filepath.Join(project, filepath.FromSlash(name))
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			hash.Write([]byte("missing"))
			hash.Write([]byte{0})
			continue
		}
		if err != nil {
			return state, err
		}
		mode := uint32(info.Mode().Perm())
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			mode |= 0120000
		case info.IsDir():
			mode |= 0040000
		default:
			mode |= 0100000
		}
		if info.Mode()&fs.ModeSetuid != 0 {
			mode |= 04000
		}
		if info.Mode()&fs.ModeSetgid != 0 {
			mode |= 02000
		}
		if info.Mode()&fs.ModeSticky != 0 {
			mode |= 01000
		}
		fmt.Fprintf(hash, "%d\x00", mode)
		var contents []byte
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return state, err
			}
			contents = []byte(target)
		case info.IsDir():
			nested, err := repositoryState(ctx, path)
			if err != nil {
				return state, err
			}
			contents, err = json.Marshal(nested)
			if err != nil {
				return state, err
			}
		default:
			contents, err = os.ReadFile(path)
			if err != nil {
				return state, err
			}
		}
		hash.Write(contents)
		hash.Write([]byte{0})
	}
	fingerprint := fmt.Sprintf("%x", hash.Sum(nil))
	state.Fingerprint = &fingerprint
	return state, nil
}

type projectEntry struct {
	ProjectID projectID `json:"projectId"`
	Root      string    `json:"root"`
}
type projectIndex struct {
	Version  int            `json:"version"`
	Projects []projectEntry `json:"projects"`
}

func indexPath(directory string) (string, error) {
	if directory == "" {
		directory = os.Getenv("AUDIT_REPORT_HOME")
	}
	if directory == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		directory = filepath.Join(home, ".audit-report")
	}
	return filepath.Join(directory, "projects.json"), nil
}
func parseIndex(data []byte) (projectIndex, error) {
	if len(data) == 0 {
		return projectIndex{stateVersion, []projectEntry{}}, nil
	}
	var index projectIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return index, err
	}
	if index.Version != stateVersion || index.Projects == nil {
		return index, invalid("Invalid project index")
	}
	for _, entry := range index.Projects {
		if !uuidPattern.MatchString(string(entry.ProjectID)) || !filepath.IsAbs(entry.Root) {
			return index, invalid("Invalid project index entry")
		}
	}
	return index, nil
}
func registerProject(c auditContext, directory string) error {
	path, err := indexPath(directory)
	if err != nil {
		return err
	}
	_, err = updateFile(path, func(data []byte) ([]byte, error) { return registeredProjectData(c, data) })
	return err
}

func registeredProjectData(c auditContext, data []byte) ([]byte, error) {
	index, err := parseIndex(data)
	if err != nil {
		return nil, err
	}
	for _, entry := range index.Projects {
		if entry.ProjectID != c.State.ProjectID {
			continue
		}
		if entry.Root == c.Root {
			return data, nil
		}
		old, err := os.ReadFile(filepath.Join(entry.Root, stateFile))
		if err == nil {
			state, err := parseState(old)
			if err != nil {
				return nil, err
			}
			if state.ProjectID == entry.ProjectID {
				return nil, conflict("Project ID also exists at %s. Do not register a copied dotfile as a different project.", entry.Root)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	index.Projects = slices.DeleteFunc(index.Projects, func(entry projectEntry) bool {
		return entry.ProjectID == c.State.ProjectID || entry.Root == c.Root
	})
	index.Projects = append(index.Projects, projectEntry{c.State.ProjectID, c.Root})
	return encode(index)
}

type resumeResult struct {
	ProjectID         projectID                    `json:"projectId"`
	AuditID           auditID                      `json:"auditId"`
	ProjectRevision   string                       `json:"projectRevision"`
	ProjectState      projectState                 `json:"projectState"`
	Audit             savedAudit                   `json:"audit"`
	Paths             auditPaths                   `json:"paths"`
	Report            json.RawMessage              `json:"report"`
	ReportErrors      []string                     `json:"reportErrors"`
	ReportWarnings    []string                     `json:"reportWarnings"`
	Feedback          map[findingID]feedbackRecord `json:"feedback"`
	FeedbackReview    []string                     `json:"feedbackReview"`
	CurrentRepository repository                   `json:"currentRepository"`
	RepositoryChanged *bool                        `json:"repositoryChanged"`
	ReportChanged     bool                         `json:"reportChanged"`
	PathChanged       bool                         `json:"pathChanged"`
}

type auditSnapshot struct {
	context    auditContext
	document   reportDocument
	feedback   feedbackFile
	repository repository
}

func readAuditSnapshot(ctx context.Context, c auditContext) (auditSnapshot, error) {
	doc, err := readReport(c.Paths.Report, true)
	if err != nil {
		return auditSnapshot{}, err
	}
	feedback, err := readFeedback(c)
	if err != nil {
		return auditSnapshot{}, err
	}
	repository, err := repositoryState(ctx, c.Root)
	if err != nil {
		return auditSnapshot{}, err
	}
	return auditSnapshot{c, doc, feedback, repository}, nil
}

func (snapshot auditSnapshot) result() resumeResult {
	c, doc, feedback, repository := snapshot.context, snapshot.document, snapshot.feedback, snapshot.repository
	var changed *bool
	if previous := c.Audit.Checkpoint.Repository.Fingerprint; previous != nil && repository.Fingerprint != nil {
		value := *previous != *repository.Fingerprint
		changed = &value
	}
	review := reviewFeedback(doc.Report, feedback.Findings)
	if len(doc.Errors) > 0 {
		review = []string{"Repair the report before reconciling saved feedback."}
	}
	return resumeResult{c.State.ProjectID, c.Audit.ID, c.StateRevision, c.State, c.Audit, c.Paths, doc.Raw, doc.Errors, doc.Warnings, feedback.Findings, review, repository, changed, c.Audit.Checkpoint.ReportRevision != doc.Revision, doc.Report.RepoRoot != c.Root}
}

func resumeAudit(ctx context.Context, options auditOptions) (resumeResult, error) {
	c, err := loadAudit(options)
	if err != nil {
		return resumeResult{}, err
	}
	snapshot, err := readAuditSnapshot(ctx, c)
	if err != nil {
		return resumeResult{}, err
	}
	if err := registerProject(c, options.IndexDir); err != nil {
		return resumeResult{}, err
	}
	return snapshot.result(), nil
}

func initAudit(ctx context.Context, options auditOptions, source string, newAudit bool) (resumeResult, error) {
	snapshot, err := initializeAudit(ctx, options, source, newAudit)
	if err != nil {
		return resumeResult{}, err
	}
	return snapshot.result(), nil
}

func initializeAudit(ctx context.Context, options auditOptions, source string, newAudit bool) (auditSnapshot, error) {
	var snapshot auditSnapshot
	root, err := projectRoot(options.Project)
	if err != nil {
		return snapshot, err
	}
	index, err := indexPath(options.IndexDir)
	if err != nil {
		return snapshot, err
	}
	_, err = updateFile(index, func(indexData []byte) ([]byte, error) {
		if _, err := parseIndex(indexData); err != nil {
			return nil, err
		}
		var registration []byte
		committed, err := updateFile(filepath.Join(root, stateFile), func(data []byte) ([]byte, error) {
			var state projectState
			if len(data) > 0 {
				var err error
				state, err = parseState(data)
				if err != nil {
					return nil, err
				}
			} else {
				id, err := uuid()
				if err != nil {
					return nil, err
				}
				resources, err := resourceDir()
				if err != nil {
					return nil, err
				}
				state = projectState{Schema: filepath.Join(resources, "schema/project-state.schema.json"), Version: stateVersion, ProjectID: projectID(id), DataDir: "../" + filepath.Base(root) + ".audits", Audits: []savedAudit{}}
			}
			var err error
			registration, err = registeredProjectData(auditContext{Root: root, State: state}, indexData)
			if err != nil {
				return nil, err
			}
			if len(data) > 0 && !newAudit {
				c, err := contextForAudit(root, state, digest(data), state.ActiveAuditID)
				if err != nil {
					return nil, err
				}
				snapshot, err = readAuditSnapshot(ctx, c)
				return data, err
			}

			id, err := uuid()
			if err != nil {
				return nil, err
			}
			directory := filepath.Join(root, filepath.FromSlash(state.DataDir), id)
			report, err := prepareReport(root, source, directory)
			if err != nil {
				return nil, err
			}
			report.ID = reportID(id)
			reportData, err := encode(report)
			if err != nil {
				return nil, err
			}
			if _, err := updateFile(filepath.Join(directory, reportFile), func(previous []byte) ([]byte, error) {
				if len(previous) > 0 {
					return nil, conflict("Audit report already exists")
				}
				return reportData, nil
			}); err != nil {
				return nil, err
			}
			feedback := feedbackFile{stateVersion, state.ProjectID, auditID(id), map[findingID]feedbackRecord{}}
			if _, err := updateFile(filepath.Join(directory, feedbackFilename), func(previous []byte) ([]byte, error) {
				if len(previous) > 0 {
					return nil, conflict("Audit feedback already exists")
				}
				return encode(feedback)
			}); err != nil {
				return nil, err
			}
			repository, err := repositoryState(ctx, root)
			if err != nil {
				return nil, err
			}
			next := "Record the audit scope and start investigating."
			if source != "" {
				next = "Review the imported report and continue the audit."
			}
			state.ActiveAuditID = auditID(id)
			state.Audits = append(state.Audits, savedAudit{auditID(id), inProgress, checkpoint{timestamp(), []string{}, []string{next}, repository, digest(reportData)}})
			stateData, err := encode(state)
			if err != nil {
				return nil, err
			}
			c, err := contextForAudit(root, state, digest(stateData), state.ActiveAuditID)
			if err != nil {
				return nil, err
			}
			doc, err := parseReport(reportData, directory, true)
			if err != nil {
				return nil, err
			}
			snapshot = auditSnapshot{c, doc, feedback, repository}
			return stateData, nil
		})
		if err != nil {
			return nil, err
		}
		snapshot.context.StateRevision = digest(committed)
		return registration, nil
	})
	if err != nil {
		return auditSnapshot{}, err
	}
	return snapshot, nil
}

func prepareReport(root, source, directory string) (report, error) {
	r := report{Title: filepath.Base(root) + " audit", Project: filepath.Base(root), RepoRoot: root, Date: time.Now().UTC().Format(time.DateOnly), Scope: "Scope has not been recorded yet.", Areas: []area{{"general", "General"}}, Findings: []finding{}}
	if source != "" {
		doc, err := readReport(source, true)
		if err != nil {
			return r, err
		}
		if err := doc.ready(); err != nil {
			return r, err
		}
		r = doc.Report
		images := map[string]string{}
		for i := range r.Findings {
			for j := range r.Findings[i].Evidence {
				e := &r.Findings[i].Evidence[j]
				if e.Type != screenshotEvidence {
					continue
				}
				if _, ok := images[e.Path]; !ok {
					target := fmt.Sprintf("screenshots/%d%s", len(images)+1, strings.ToLower(filepath.Ext(e.Path)))
					data, err := os.ReadFile(screenshotPath(filepath.Dir(source), e.Path))
					if err != nil {
						return r, err
					}
					if err := os.MkdirAll(filepath.Join(directory, "screenshots"), 0o700); err != nil {
						return r, err
					}
					if err := os.WriteFile(filepath.Join(directory, target), data, 0o600); err != nil {
						return r, err
					}
					images[e.Path] = target
				}
				e.Path = images[e.Path]
			}
		}
	}
	resources, err := resourceDir()
	if err != nil {
		return r, err
	}
	r.RepoRoot = root
	r.Schema = filepath.Join(resources, "schema/report.schema.json")
	return r, nil
}

func checkpointAudit(ctx context.Context, options auditOptions, input checkpointInput, revision string) (resumeResult, error) {
	if input.Completed == nil || input.NextSteps == nil {
		return resumeResult{}, invalid("Checkpoint requires completed and nextSteps arrays")
	}
	for _, value := range slices.Concat(input.Completed, input.NextSteps) {
		if strings.TrimSpace(value) == "" {
			return resumeResult{}, invalid("Checkpoint entries must not be empty")
		}
	}
	if input.Status != "" && input.Status != inProgress && input.Status != awaitingReview && input.Status != complete {
		return resumeResult{}, invalid("Invalid audit status")
	}
	c, err := loadAudit(options)
	if err != nil {
		return resumeResult{}, err
	}
	snapshot, err := readAuditSnapshot(ctx, c)
	if err != nil {
		return resumeResult{}, err
	}
	committed, err := updateFile(c.StatePath, func(data []byte) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := expectRevision(digest(data), &revision); err != nil {
			return nil, err
		}
		state, err := parseState(data)
		if err != nil {
			return nil, err
		}
		if err := registerProject(auditContext{Root: c.Root, State: state}, options.IndexDir); err != nil {
			return nil, err
		}
		for i := range state.Audits {
			a := &state.Audits[i]
			if a.ID != c.Audit.ID {
				continue
			}
			if input.Status != "" {
				a.Status = input.Status
			}
			a.Checkpoint = checkpoint{timestamp(), input.Completed, input.NextSteps, snapshot.repository, snapshot.document.Revision}
			c.Audit = *a
		}
		state.ActiveAuditID = c.Audit.ID
		c.State = state
		return encode(state)
	})
	if err != nil {
		return resumeResult{}, err
	}
	c.StateRevision = digest(committed)
	snapshot.context = c
	return snapshot.result(), nil
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
