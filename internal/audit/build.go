package audit

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	auditdocs "github.com/YoloWingPixie/spectre/docs/audit"
)

const buildUsage = `Usage:
  spectre audit build <report.json> [-o out.html] [--strict]
  spectre audit build <report.json> --check [--strict]
  spectre audit build --init <path/report.json>

--check validates without writing output. --strict rejects lint warnings.
Screenshots are embedded up to 3 MiB, then copied to a matching .assets folder.
`

func buildCommand(args []string, out, diagnostics io.Writer) error {
	var input, output, starter string
	strict, check := false, false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h", "--help", "-help":
			_, err := io.WriteString(out, buildUsage)
			return err
		case "-o", "--out", "--init":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return invalid("%s requires a path", arg)
			}
			i++
			if arg == "--init" {
				starter = args[i]
			} else {
				output = args[i]
			}
		case "--strict":
			strict = true
		case "--check":
			check = true
		default:
			if strings.HasPrefix(arg, "-") || input != "" {
				return invalid("Unexpected argument: %s", arg)
			}
			input = arg
		}
	}
	if starter != "" {
		if input != "" || output != "" || check || strict {
			return invalid("Use --init without build options")
		}
		return initReport(starter, diagnostics)
	}
	if input == "" {
		return invalid("%s", buildUsage)
	}
	doc, err := readReport(input, false)
	if err != nil {
		return err
	}
	if err := doc.ready(); err != nil {
		return err
	}
	warnings := append(doc.Warnings, lintReport(doc.Report)...)
	for _, warning := range warnings {
		if _, err := fmt.Fprintln(diagnostics, "warning: "+warning); err != nil {
			return err
		}
	}
	if strict && len(warnings) > 0 {
		return invalid("%d warning(s) and --strict is set. Nothing written.", len(warnings))
	}
	if check {
		_, err := fmt.Fprintf(diagnostics, "ok: %d findings valid, %d warning(s)\n", len(doc.Report.Findings), len(warnings))
		return err
	}
	if output == "" {
		output = strings.TrimSuffix(input, filepath.Ext(input)) + ".html"
	}
	if doc.Report.ID == "" {
		absolute, err := filepath.Abs(input)
		if err != nil {
			return err
		}
		doc.Report.ID = reportID("path-" + digest([]byte(absolute)))
	}
	images, err := collectImages(doc.Report, filepath.Dir(input), output, false)
	if err != nil {
		return err
	}
	html, err := renderReport(doc.Report, images, nil)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(output, html, 0o644); err != nil {
		return err
	}
	_, err = fmt.Fprintf(diagnostics, "ok: wrote %s (%d bytes; %d screenshot(s))\npublish: artifact feedback capabilities {\"db\":{},\"comments\":{},\"user\":{}}\n", output, len(html), len(images))
	return err
}
func initReport(path string, diagnostics io.Writer) error {
	data, err := auditdocs.Files.ReadFile("templates/report.starter.json")
	if err != nil {
		return err
	}
	doc, err := parseReport(data, ".", false)
	if err != nil {
		return err
	}
	if err := doc.ready(); err != nil {
		return err
	}
	directory, err := resourceDir()
	if err != nil {
		return err
	}
	r := doc.Report
	id, err := uuid()
	if err != nil {
		return err
	}
	r.ID = reportID(id)
	r.Schema = filepath.Join(directory, "schema/report.schema.json")
	r.Date = time.Now().UTC().Format(time.DateOnly)
	if distro := os.Getenv("WSL_DISTRO_NAME"); distro != "" {
		r.WSLDistro = distro
	}
	data, err = encode(r)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("write starter (existing files are preserved): %w", err)
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	_, err = fmt.Fprintf(diagnostics, "ok: wrote starter report to %s\nnext: edit it, then run: spectre audit build %q --strict\n", path, path)
	return err
}
