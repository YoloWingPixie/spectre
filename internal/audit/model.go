package audit

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type findingID string
type optionID string
type projectID string
type auditID string
type severity string
type findingStatus string
type evidenceKind string
type effort string
type risk string
type auditStatus string
type decisionKind string

const (
	critical           severity      = "critical"
	high               severity      = "high"
	medium             severity      = "medium"
	low                severity      = "low"
	open               findingStatus = "open"
	fixed              findingStatus = "fixed"
	accepted           findingStatus = "accepted"
	wontFix            findingStatus = "wont-fix"
	fileEvidence       evidenceKind  = "file"
	screenshotEvidence evidenceKind  = "screenshot"
	commandEvidence    evidenceKind  = "command"
	small              effort        = "S"
	mediumEffort       effort        = "M"
	large              effort        = "L"
	lowRisk            risk          = "low"
	mediumRisk         risk          = "med"
	highRisk           risk          = "high"
	inProgress         auditStatus   = "in-progress"
	awaitingReview     auditStatus   = "awaiting-review"
	complete           auditStatus   = "complete"
	chooseOption       decisionKind  = "coa"
	chooseNone         decisionKind  = "none"
)

type report struct {
	Schema        string         `json:"$schema,omitempty"`
	Title         string         `json:"title"`
	Project       string         `json:"project"`
	RepoRoot      string         `json:"repoRoot"`
	WSLDistro     string         `json:"wslDistro,omitempty"`
	Commit        string         `json:"commit,omitempty"`
	Branch        string         `json:"branch,omitempty"`
	Date          string         `json:"date"`
	Stamp         string         `json:"stamp,omitempty"`
	Scope         string         `json:"scope"`
	Areas         []area         `json:"areas"`
	Findings      []finding      `json:"findings"`
	FixFirst      []priorityItem `json:"fixFirst,omitempty"`
	OpenQuestions []question     `json:"openQuestions,omitempty"`
	Clean         []cleanArea    `json:"clean,omitempty"`
	Notes         string         `json:"notes,omitempty"`
}

type area struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}
type cleanArea struct {
	Area string `json:"area"`
	Text string `json:"text"`
}
type location struct {
	Path    string `json:"path"`
	Line    int    `json:"line,omitempty"`
	EndLine int    `json:"endLine,omitempty"`
	Col     int    `json:"col,omitempty"`
}
type currentDesign struct {
	Description string   `json:"description"`
	Pros        []string `json:"pros"`
	Cons        []string `json:"cons"`
}
type finding struct {
	ID         findingID     `json:"id"`
	Area       string        `json:"area"`
	Severity   severity      `json:"severity"`
	Status     findingStatus `json:"status"`
	StatusNote string        `json:"statusNote,omitempty"`
	Title      string        `json:"title"`
	Summary    string        `json:"summary"`
	Location   location      `json:"location"`
	RelatedIDs []findingID   `json:"relatedIds,omitempty"`
	Evidence   []evidence    `json:"evidence,omitempty"`
	Current    currentDesign `json:"current"`
	COAs       []option      `json:"coas,omitempty"`
	Tags       []string      `json:"tags,omitempty"`
}
type evidence struct {
	Type    evidenceKind `json:"type"`
	Path    string       `json:"path,omitempty"`
	Line    int          `json:"line,omitempty"`
	EndLine int          `json:"endLine,omitempty"`
	Col     int          `json:"col,omitempty"`
	Excerpt string       `json:"excerpt,omitempty"`
	Note    string       `json:"note,omitempty"`
	Caption string       `json:"caption,omitempty"`
	Command string       `json:"command,omitempty"`
	Output  string       `json:"output"`
	CWD     string       `json:"cwd,omitempty"`
}

func (e evidence) MarshalJSON() ([]byte, error) {
	type plain evidence
	var output *string
	if e.Type == commandEvidence {
		output = &e.Output
	}
	return json.Marshal(struct {
		plain
		Output *string `json:"output,omitempty"`
	}{plain(e), output})
}

type option struct {
	ID                optionID `json:"id,omitempty"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Pros              []string `json:"pros"`
	Cons              []string `json:"cons"`
	Effort            effort   `json:"effort"`
	Risk              risk     `json:"risk"`
	Touches           []string `json:"touches,omitempty"`
	Recommended       bool     `json:"recommended,omitempty"`
	RecommendedReason string   `json:"recommendedReason,omitempty"`
}
type priorityItem struct {
	ID  findingID `json:"id"`
	Why string    `json:"why,omitempty"`
}

func (p *priorityItem) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		return json.Unmarshal(b, &p.ID)
	}
	type plain priorityItem
	return json.Unmarshal(b, (*plain)(p))
}

type question struct {
	Text      string    `json:"text"`
	FindingID findingID `json:"findingId,omitempty"`
}

func (q *question) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		return json.Unmarshal(b, &q.Text)
	}
	type plain question
	return json.Unmarshal(b, (*plain)(q))
}

type projectState struct {
	Schema        string       `json:"$schema,omitempty"`
	Version       int          `json:"version"`
	ProjectID     projectID    `json:"projectId"`
	DataDir       string       `json:"dataDir"`
	ActiveAuditID auditID      `json:"activeAuditId"`
	Audits        []savedAudit `json:"audits"`
}
type savedAudit struct {
	ID         auditID     `json:"id"`
	Status     auditStatus `json:"status"`
	Checkpoint checkpoint  `json:"checkpoint"`
}
type checkpoint struct {
	UpdatedAt      string     `json:"updatedAt"`
	Completed      []string   `json:"completed"`
	NextSteps      []string   `json:"nextSteps"`
	Repository     repository `json:"repository"`
	ReportRevision string     `json:"reportRevision"`
}
type checkpointInput struct {
	Completed []string    `json:"completed"`
	NextSteps []string    `json:"nextSteps"`
	Status    auditStatus `json:"status,omitempty"`
}
type repository struct {
	Head        *string `json:"head"`
	Fingerprint *string `json:"fingerprint"`
}
type decision struct {
	Type   decisionKind `json:"type"`
	COAID  optionID     `json:"coaId,omitempty"`
	Reason string       `json:"reason,omitempty"`
}
type feedbackContent struct {
	Decision *decision           `json:"decision"`
	COANotes map[optionID]string `json:"coaNotes"`
	Note     string              `json:"note"`
}
type feedbackRecord struct {
	FindingID findingID `json:"findingId"`
	feedbackContent
	UpdatedAt string            `json:"updatedAt"`
	Notify    map[string]string `json:"notify"`
	UpdatedBy json.RawMessage   `json:"updatedBy,omitempty"`
}

func (r *feedbackRecord) UnmarshalJSON(data []byte) error {
	if _, err := decodeContent(data); err != nil {
		return err
	}
	type plain feedbackRecord
	return json.Unmarshal(data, (*plain)(r))
}

type feedbackFile struct {
	Version   int                          `json:"version"`
	ProjectID projectID                    `json:"projectId"`
	AuditID   auditID                      `json:"auditId"`
	Findings  map[findingID]feedbackRecord `json:"findings"`
}

type auditError struct {
	status  int
	message string
}

func (e *auditError) Error() string { return e.message }
func invalid(format string, args ...any) error {
	return &auditError{http.StatusUnprocessableEntity, fmt.Sprintf(format, args...)}
}
func conflict(format string, args ...any) error {
	return &auditError{http.StatusConflict, fmt.Sprintf(format, args...)}
}
func missing(format string, args ...any) error {
	return &auditError{http.StatusNotFound, fmt.Sprintf(format, args...)}
}
