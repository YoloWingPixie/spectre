package audit

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

var optionPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var findingPattern = regexp.MustCompile(`^[A-Za-z0-9_.~:@+-]+$`)

func cleanText(text string) string {
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' || r == 127 {
			return -1
		}
		return r
	}, text))
}
func normalizeContent(content feedbackContent) (feedbackContent, error) {
	if d := content.Decision; d != nil {
		copy := *d
		content.Decision = &copy
		switch d.Type {
		case chooseOption:
			if d.COAID == "__proto__" || !optionPattern.MatchString(string(d.COAID)) {
				return content, invalid("Invalid decision option ID")
			}
			copy.Reason = ""
		case chooseNone:
			copy.Reason = cleanText(d.Reason)
			copy.COAID = ""
			if copy.Reason == "" {
				return content, invalid("Say why none of the options fit")
			}
		default:
			return content, invalid("Decision type must be coa or none")
		}
	}
	notes := map[optionID]string{}
	for id, note := range content.COANotes {
		if id == "__proto__" || !optionPattern.MatchString(string(id)) {
			return content, invalid("Invalid option note ID")
		}
		if note = cleanText(note); note != "" {
			notes[id] = note
		}
	}
	content.COANotes = notes
	content.Note = cleanText(content.Note)
	return content, nil
}
func decodeContent(data []byte) (feedbackContent, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil || raw == nil {
		return feedbackContent{}, invalid("Expected a feedback object")
	}
	for _, key := range []string{"note", "coaNotes"} {
		for name, value := range raw {
			if strings.EqualFold(name, key) && string(value) == "null" {
				return feedbackContent{}, invalid("feedback.%s cannot be null", key)
			}
		}
	}
	for name, value := range raw {
		if !strings.EqualFold(name, "coaNotes") {
			continue
		}
		var notes map[string]json.RawMessage
		if err := json.Unmarshal(value, &notes); err != nil {
			return feedbackContent{}, invalid("feedback.coaNotes: expected an object")
		}
		for id, note := range notes {
			var text string
			if string(note) == "null" || json.Unmarshal(note, &text) != nil {
				return feedbackContent{}, invalid("feedback.coaNotes.%s: expected text", id)
			}
		}
	}
	var content feedbackContent
	if err := json.Unmarshal(data, &content); err != nil {
		return content, invalid("Invalid feedback: %v", err)
	}
	return normalizeContent(content)
}
func parseFeedback(data []byte, c auditContext) (feedbackFile, error) {
	var feedback feedbackFile
	if err := json.Unmarshal(data, &feedback); err != nil {
		return feedback, invalid("Invalid feedback JSON: %v", err)
	}
	if feedback.Version != stateVersion || feedback.ProjectID != c.State.ProjectID || feedback.AuditID != c.Audit.ID || feedback.Findings == nil {
		return feedback, invalid("Feedback identity or version does not match this audit")
	}
	for id, record := range feedback.Findings {
		if id != record.FindingID {
			return feedback, invalid("Feedback record identity does not match %s", id)
		}
		if _, err := normalizeContent(record.feedbackContent); err != nil {
			return feedback, err
		}
	}
	return feedback, nil
}
func readFeedback(c auditContext) (feedbackFile, error) {
	data, err := readFile(c.Paths.Feedback)
	if err != nil {
		return feedbackFile{}, err
	}
	return parseFeedback(data, c)
}
func recordRevision(record feedbackRecord) string {
	data, _ := json.Marshal(record)
	return digest(data)
}
func reviewFeedback(r report, findings map[findingID]feedbackRecord) []string {
	review := []string{}
	ids := slices.Sorted(maps.Keys(findings))
	for _, id := range ids {
		record := findings[id]
		index := slices.IndexFunc(r.Findings, func(f finding) bool { return f.ID == id })
		if index < 0 {
			review = append(review, fmt.Sprintf("%s: saved feedback refers to a removed finding", id))
			continue
		}
		options := map[optionID]bool{}
		for _, c := range r.Findings[index].COAs {
			options[c.ID] = true
		}
		if record.Decision != nil && record.Decision.Type == chooseOption && !options[record.Decision.COAID] {
			review = append(review, fmt.Sprintf("%s: selected option %s was removed", id, record.Decision.COAID))
		}
		for option := range record.COANotes {
			if !options[option] {
				review = append(review, fmt.Sprintf("%s: note for removed option %s", id, option))
			}
		}
	}
	return review
}

type feedbackResult struct {
	ProjectID      projectID                    `json:"projectId"`
	AuditID        auditID                      `json:"auditId"`
	Findings       map[findingID]feedbackRecord `json:"findings"`
	Revisions      map[findingID]*string        `json:"revisions"`
	ReportRevision string                       `json:"reportRevision"`
	Review         []string                     `json:"review"`
}

func getFeedback(options auditOptions) (feedbackResult, error) {
	c, err := loadAudit(options)
	if err != nil {
		return feedbackResult{}, err
	}
	doc, err := readReport(c.Paths.Report, true)
	if err != nil {
		return feedbackResult{}, err
	}
	if err := doc.ready(); err != nil {
		return feedbackResult{}, err
	}
	feedback, err := readFeedback(c)
	if err != nil {
		return feedbackResult{}, err
	}
	revisions := map[findingID]*string{}
	for _, f := range doc.Report.Findings {
		if record, ok := feedback.Findings[f.ID]; ok {
			revision := recordRevision(record)
			revisions[f.ID] = &revision
		} else {
			revisions[f.ID] = nil
		}
	}
	return feedbackResult{c.State.ProjectID, c.Audit.ID, feedback.Findings, revisions, doc.Revision, reviewFeedback(doc.Report, feedback.Findings)}, nil
}

type feedbackUpdate struct {
	FindingID        findingID       `json:"findingId"`
	Content          json.RawMessage `json:"content"`
	ExpectedRevision *string         `json:"expectedRevision"`
	ReportRevision   string          `json:"reportRevision"`
}
type saveResult struct {
	Record   feedbackRecord `json:"record"`
	Revision string         `json:"revision"`
}

func saveFeedback(options auditOptions, input feedbackUpdate) (saveResult, error) {
	c, err := loadAudit(options)
	if err != nil {
		return saveResult{}, err
	}
	content, err := decodeContent(input.Content)
	if err != nil {
		return saveResult{}, err
	}
	var record feedbackRecord
	_, err = updateFile(c.Paths.Feedback, func(data []byte) ([]byte, error) {
		feedback, err := parseFeedback(data, c)
		if err != nil {
			return nil, err
		}
		doc, err := readReport(c.Paths.Report, true)
		if err != nil {
			return nil, err
		}
		if err := doc.ready(); err != nil {
			return nil, err
		}
		if err := expectRevision(doc.Revision, &input.ReportRevision); err != nil {
			return nil, err
		}
		index := slices.IndexFunc(doc.Report.Findings, func(f finding) bool { return f.ID == input.FindingID })
		if index < 0 {
			return nil, missing("Unknown finding ID")
		}
		previous, exists := feedback.Findings[input.FindingID]
		revision := ""
		if exists {
			revision = recordRevision(previous)
		}
		if err := expectRevision(revision, input.ExpectedRevision); err != nil {
			return nil, err
		}
		options := map[optionID]bool{}
		for _, option := range doc.Report.Findings[index].COAs {
			options[option.ID] = true
		}
		if content.Decision != nil && content.Decision.Type == chooseOption && !options[content.Decision.COAID] {
			return nil, invalid("Decision refers to an unknown option")
		}
		for id := range content.COANotes {
			if !options[id] {
				return nil, invalid("Note refers to an unknown option: %s", id)
			}
		}
		for id, note := range previous.COANotes {
			if !options[id] {
				content.COANotes[id] = note
			}
		}
		if content.Decision == nil && previous.Decision != nil && previous.Decision.Type == chooseOption && !options[previous.Decision.COAID] {
			content.Decision = previous.Decision
		}
		record = feedbackRecord{FindingID: input.FindingID, feedbackContent: content, UpdatedAt: timestamp(), Notify: map[string]string{}}
		feedback.Findings[input.FindingID] = record
		return encode(feedback)
	})
	if err != nil {
		return saveResult{}, err
	}
	return saveResult{record, recordRevision(record)}, nil
}

type importResult struct {
	Findings map[findingID]feedbackRecord `json:"findings"`
	Review   []string                     `json:"review"`
}

func importFeedback(options auditOptions, data []byte, replace bool) (importResult, error) {
	c, err := loadAudit(options)
	if err != nil {
		return importResult{}, err
	}
	var input struct {
		ProjectID projectID                     `json:"projectId"`
		AuditID   auditID                       `json:"auditId"`
		Findings  map[findingID]json.RawMessage `json:"findings"`
	}
	if err := json.Unmarshal(data, &input); err != nil || input.Findings == nil {
		return importResult{}, invalid("Expected a feedback export with findings")
	}
	if input.ProjectID != "" && input.ProjectID != c.State.ProjectID || input.AuditID != "" && input.AuditID != c.Audit.ID {
		return importResult{}, invalid("Feedback export belongs to a different project or audit")
	}
	entries := map[findingID]feedbackContent{}
	for id, data := range input.Findings {
		if !findingPattern.MatchString(string(id)) || id == "." || id == ".." || id == "__proto__" {
			return importResult{}, invalid("Invalid finding ID: %s", id)
		}
		var record map[string]json.RawMessage
		if err := json.Unmarshal(data, &record); err != nil || record == nil {
			return importResult{}, invalid("Expected a feedback record for %s", id)
		}
		for key, identity := range record {
			if !strings.EqualFold(key, "findingId") {
				continue
			}
			var value findingID
			if err := json.Unmarshal(identity, &value); err != nil || value != id {
				return importResult{}, invalid("Feedback record identity does not match %s", id)
			}
		}
		content, err := decodeContent(data)
		if err != nil {
			return importResult{}, err
		}
		entries[id] = content
	}
	var result feedbackFile
	var doc reportDocument
	_, err = updateFile(c.Paths.Feedback, func(data []byte) ([]byte, error) {
		feedback, err := parseFeedback(data, c)
		if err != nil {
			return nil, err
		}
		doc, err = readReport(c.Paths.Report, true)
		if err != nil {
			return nil, err
		}
		changed := false
		for id, content := range entries {
			if previous, ok := feedback.Findings[id]; ok {
				normalized, err := normalizeContent(previous.feedbackContent)
				if err != nil {
					return nil, err
				}
				if reflect.DeepEqual(normalized, content) {
					continue
				}
				if !replace {
					return nil, conflict("Import conflict for %s. Review both records; use --replace only to replace saved feedback.", id)
				}
			}
			feedback.Findings[id] = feedbackRecord{FindingID: id, feedbackContent: content, UpdatedAt: timestamp(), Notify: map[string]string{}}
			changed = true
		}
		result = feedback
		if !changed {
			return data, nil
		}
		return encode(feedback)
	})
	if err != nil {
		return importResult{}, err
	}
	return importResult{result.Findings, reviewFeedback(doc.Report, result.Findings)}, nil
}
