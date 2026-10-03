package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func relativeFile(path string) bool {
	if strings.TrimSpace(path) == "" || strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\\:\x00") {
		return false
	}
	return !slices.Contains(strings.Split(path, "/"), "..")
}

type reportDocument struct {
	Raw      json.RawMessage
	Report   report
	Revision string
	validation
}

func readReport(path string, persistent bool) (reportDocument, error) {
	data, err := readFile(path)
	if err != nil {
		return reportDocument{}, err
	}
	return parseReport(data, filepath.Dir(path), persistent)
}

func parseReport(data []byte, baseDir string, persistent bool) (reportDocument, error) {
	checks, normalized, err := validateJSON(data, "report.schema.json", false)
	if err != nil {
		return reportDocument{}, err
	}
	doc := reportDocument{Raw: data, Revision: digest(data), validation: checks}
	if len(checks.Errors) > 0 {
		return doc, nil
	}
	if err := json.Unmarshal(normalized, &doc.Report); err != nil {
		return doc, err
	}
	ids := map[findingID]bool{}
	anchors := map[string]bool{}
	areas := map[string]bool{}
	for _, a := range doc.Report.Areas {
		if areas[a.Key] {
			doc.Errors = append(doc.Errors, "duplicate area: "+a.Key)
		}
		areas[a.Key] = true
	}
	for i := range doc.Report.Findings {
		f := &doc.Report.Findings[i]
		path := fmt.Sprintf("findings[%d] (%s)", i, f.ID)
		if ids[f.ID] {
			doc.Errors = append(doc.Errors, path+": duplicate finding ID")
		}
		ids[f.ID] = true
		if f.ID == "." || f.ID == ".." || f.ID == "__proto__" {
			doc.Errors = append(doc.Errors, path+": reserved finding ID")
		}
		key := anchor(string(f.ID))
		if anchors[key] {
			doc.Errors = append(doc.Errors, path+": finding IDs produce the same page anchor")
		}
		anchors[key] = true
		if !areas[f.Area] {
			doc.Errors = append(doc.Errors, path+".area: unknown area")
		}
		if f.Location.EndLine != 0 && f.Location.EndLine < f.Location.Line {
			doc.Errors = append(doc.Errors, path+".location.endLine: must be >= line")
		}
		for j, e := range f.Evidence {
			q := fmt.Sprintf("%s.evidence[%d]", path, j)
			if e.Type == screenshotEvidence {
				if _, err := os.Stat(screenshotPath(baseDir, e.Path)); err != nil {
					doc.Errors = append(doc.Errors, q+": screenshot file not found")
				}
			} else if e.Type == fileEvidence {
				if e.EndLine != 0 && e.EndLine < e.Line {
					doc.Errors = append(doc.Errors, q+".endLine: must be >= line")
				}
				if e.Excerpt == "" && e.Note == "" {
					doc.Warnings = append(doc.Warnings, q+": add an excerpt or a note")
				}
				if strings.Count(e.Excerpt, "\n") >= 15 {
					doc.Warnings = append(doc.Warnings, q+": keep excerpts to 15 lines or fewer")
				}
			}
		}
		recommended := 0
		options := map[optionID]bool{}
		for j := range f.COAs {
			c := &f.COAs[j]
			q := fmt.Sprintf("%s.coas[%d]", path, j)
			if c.ID == "" {
				if persistent {
					doc.Errors = append(doc.Errors, q+".id: stable option ID is required")
				} else {
					c.ID = optionID(fmt.Sprintf("coa-%d", j+1))
					doc.Warnings = append(doc.Warnings, q+".id: missing; using "+string(c.ID))
				}
			}
			if c.ID == "__proto__" {
				doc.Errors = append(doc.Errors, q+".id: reserved option ID")
			}
			if options[c.ID] {
				doc.Errors = append(doc.Errors, q+".id: duplicate option ID")
			}
			options[c.ID] = true
			if c.Recommended {
				recommended++
				if strings.TrimSpace(c.RecommendedReason) == "" {
					doc.Errors = append(doc.Errors, q+".recommendedReason: is required")
				}
			} else if c.RecommendedReason != "" {
				doc.Warnings = append(doc.Warnings, q+".recommendedReason: ignored without recommended")
			}
			for _, touch := range c.Touches {
				if ref, ok := parseTouch(touch); !ok || !relativeFile(ref.Path) || ref.EndLine != 0 && ref.EndLine < ref.Line {
					doc.Errors = append(doc.Errors, q+".touches: invalid file reference")
				}
			}
		}
		if recommended > 1 {
			doc.Errors = append(doc.Errors, path+".coas: more than one recommended option")
		}
	}
	for _, f := range doc.Report.Findings {
		for _, id := range f.RelatedIDs {
			if !ids[id] {
				doc.Errors = append(doc.Errors, "unknown related finding: "+string(id))
			}
		}
	}
	for _, p := range doc.Report.FixFirst {
		if !ids[p.ID] {
			doc.Errors = append(doc.Errors, "unknown fixFirst finding: "+string(p.ID))
		}
	}
	for _, q := range doc.Report.OpenQuestions {
		if q.FindingID != "" && !ids[q.FindingID] {
			doc.Errors = append(doc.Errors, "unknown question finding: "+string(q.FindingID))
		}
	}
	return doc, nil
}

func (doc reportDocument) ready() error {
	if len(doc.Errors) > 0 {
		return invalid("Report is not ready:\n%s", strings.Join(doc.Errors, "\n"))
	}
	return nil
}

var touchPattern = regexp.MustCompile(`^([^:\s][^:]*?)(?::(\d+)(?:-(\d+))?)?$`)

func parseTouch(text string) (location, bool) {
	m := touchPattern.FindStringSubmatch(text)
	if m == nil {
		return location{}, false
	}
	ref := location{Path: m[1]}
	if m[2] != "" {
		line, err := strconv.Atoi(m[2])
		ref.Line = line
		if err != nil || ref.Line < 1 {
			return ref, false
		}
	}
	if m[3] != "" {
		line, err := strconv.Atoi(m[3])
		ref.EndLine = line
		if err != nil || ref.EndLine < 1 {
			return ref, false
		}
	}
	return ref, true
}
