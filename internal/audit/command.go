package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

type command string

const (
	initCommand        command = "init"
	resumeCommand      command = "resume"
	checkpointCommand  command = "checkpoint"
	viewCommand        command = "view"
	importCommand      command = "import"
	listCommand        command = "list"
	buildReportCommand command = "build"
	setupAgentCommand  command = "setup"
)
const usage = `Usage: spectre audit <command> [options]
  init        --project <root> [--source <report.json>] [--new]
  resume      --project <root> [--audit <id>]
  checkpoint  --project <root> --checkpoint <file.json> --revision <revision> [--audit <id>]
  view        --project <root> [--audit <id>] [--port <number>]
  import      --project <root> --feedback <file.json> [--audit <id>] [--replace]
  list
  build       <report.json> [-o out.html] [--strict] [--check]
  setup       [--agent codex|claude|gemini|opencode] [--skills-dir <dir>]

State commands print JSON. Resume returns the revision needed by checkpoint.
AUDIT_REPORT_HOME overrides the project index directory (default: ~/.audit-report).
Both modes run in Go. Audit report paths use POSIX paths; use WSL on Windows.
`

// Run executes an audit command, writes results to out and diagnostics to
// diagnostics, and stops a running viewer when ctx is canceled. It preserves
// the audit-report state formats and returns errors without exiting the process.
func Run(ctx context.Context, args []string, out, diagnostics io.Writer) (err error) {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		_, err := io.WriteString(out, usage)
		return err
	}
	selected := command(args[0])
	args = args[1:]
	if selected == buildReportCommand {
		return buildCommand(args, out, diagnostics)
	}
	if runtime.GOOS == "windows" {
		return invalid("Audit paths use POSIX paths. Run Spectre inside WSL.")
	}
	if selected == setupAgentCommand {
		return setupCommand(args, out, diagnostics)
	}
	switch selected {
	case initCommand, resumeCommand, checkpointCommand, viewCommand, importCommand, listCommand:
	default:
		return invalid("Unknown audit command %q\n%s", selected, usage)
	}
	flags := flag.NewFlagSet("spectre audit "+string(selected), flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	flags.Usage = func() { fmt.Fprint(diagnostics, usage) }
	project := flags.String("project", "", "audited project directory")
	id := flags.String("audit", "", "audit ID (default: active audit)")
	var source, checkpointPath, revision, feedbackPath string
	var newAudit, replace bool
	port := 0
	switch selected {
	case initCommand:
		flags.StringVar(&source, "source", "", "report to import")
		flags.BoolVar(&newAudit, "new", false, "start a separate audit")
	case checkpointCommand:
		flags.StringVar(&checkpointPath, "checkpoint", "", "checkpoint JSON")
		flags.StringVar(&revision, "revision", "", "projectRevision from resume")
	case importCommand:
		flags.StringVar(&feedbackPath, "feedback", "", "feedback JSON")
		flags.BoolVar(&replace, "replace", false, "replace conflicting feedback after review")
	case viewCommand:
		flags.IntVar(&port, "port", 0, "loopback port (0 chooses an available port)")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return invalid("Unexpected positional argument: %s", flags.Arg(0))
	}
	if selected != listCommand && *project == "" {
		return invalid("--project is required")
	}
	options := auditOptions{Project: *project, AuditID: auditID(*id)}
	var result any
	switch selected {
	case initCommand:
		result, err = initAudit(ctx, options, source, newAudit)
	case resumeCommand:
		result, err = resumeAudit(ctx, options)
	case checkpointCommand:
		if checkpointPath == "" || revision == "" {
			return invalid("--checkpoint and --revision are required")
		}
		data, readErr := readFile(checkpointPath)
		if readErr != nil {
			return readErr
		}
		var input checkpointInput
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			return invalid("Invalid checkpoint: %v", err)
		}
		result, err = checkpointAudit(ctx, options, input, revision)
	case importCommand:
		if feedbackPath == "" {
			return invalid("--feedback is required")
		}
		data, readErr := readFile(feedbackPath)
		if readErr != nil {
			return readErr
		}
		result, err = importFeedback(options, data, replace)
	case listCommand:
		result, err = listProjects()
	case viewCommand:
		v, startErr := startViewer(options, port, diagnostics)
		if startErr != nil {
			return startErr
		}
		defer func() { err = errors.Join(err, v.close()) }()
		data, err := encode(map[string]string{"url": v.URL})
		if err != nil {
			return err
		}
		if _, err := out.Write(data); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-v.done:
			return v.serveErr
		}
	}
	if err != nil {
		return err
	}
	data, err := encode(result)
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}

type listedProject struct {
	projectEntry
	ActiveAuditID auditID      `json:"activeAuditId,omitempty"`
	Audits        []savedAudit `json:"audits,omitempty"`
	Error         string       `json:"error,omitempty"`
}

func listProjects() (any, error) {
	path, err := indexPath("")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	index, err := parseIndex(data)
	if err != nil {
		return nil, err
	}
	projects := []listedProject{}
	for _, entry := range index.Projects {
		listed := listedProject{projectEntry: entry}
		data, err := readFile(filepath.Join(entry.Root, stateFile))
		if err == nil {
			state, stateErr := parseState(data)
			err = stateErr
			if err == nil && state.ProjectID != entry.ProjectID {
				err = invalid("Project identity changed")
			}
			if err == nil {
				listed.ActiveAuditID = state.ActiveAuditID
				listed.Audits = state.Audits
			}
		}
		if err != nil {
			listed.Error = err.Error()
		}
		projects = append(projects, listed)
	}
	return struct {
		Projects []listedProject `json:"projects"`
	}{projects}, nil
}
