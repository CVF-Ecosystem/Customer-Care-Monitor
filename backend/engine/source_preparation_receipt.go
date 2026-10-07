package engine

import (
	"encoding/json"
	"log"
	"regexp"
	"strings"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

const preparationVersion = "ccmai.source-preparation.v1"
const preparationEntryLimit = 200
const preparationByteLimit = 256 * 1024

var preparationUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var preparationDigest = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

type preparationReference struct {
	State        string `json:"state"`
	EvaluationID string `json:"evaluation_id,omitempty"`
	RunID        string `json:"run_id,omitempty"`
	SnapshotID   string `json:"snapshot_id,omitempty"`
}

type preparationEntry struct {
	ConversationID     string               `json:"conversation_id,omitempty"`
	Outcome            string               `json:"outcome"`
	SchemaVersion      string               `json:"snapshot_schema,omitempty"`
	Digest             string               `json:"snapshot_digest,omitempty"`
	Coverage           string               `json:"coverage,omitempty"`
	CoverageReasons    []string             `json:"coverage_reasons,omitempty"`
	Reference          preparationReference `json:"reference"`
	MetadataIncomplete bool                 `json:"metadata_incomplete"`
}

type preparationReceipt struct {
	Version            string             `json:"version"`
	Scope              string             `json:"scope"`
	TenantID           string             `json:"tenant_id,omitempty"`
	JobID              string             `json:"job_id,omitempty"`
	RunID              string             `json:"run_id,omitempty"`
	Mode               string             `json:"mode,omitempty"`
	MetadataIncomplete bool               `json:"metadata_incomplete"`
	SelectionStatus    string             `json:"selection_status"`
	Selected           *int               `json:"selected"`
	Visited            int                `json:"visited"`
	Unvisited          *int               `json:"unvisited"`
	Counts             map[string]int     `json:"counts"`
	Entries            []preparationEntry `json:"entries"`
	OmittedEntries     int                `json:"omitted_entries"`
	EntriesComplete    bool               `json:"entries_complete"`
	ScanComplete       bool               `json:"scan_complete"`
	StopReason         string             `json:"stop_reason,omitempty"`
}

// The collector observes existing operations only. It never controls their result.
type preparationCollector struct {
	receipt   preparationReceipt
	active    *preparationEntry
	tenant    string
	byteLimit int
}

func preparationID(s string, invalid *bool) string {
	if !preparationUUID.MatchString(s) {
		*invalid = true
		return ""
	}
	return strings.ToLower(s)
}

func newPreparationCollector(job models.Job, run models.JobRun, plan runPlan) *preparationCollector {
	r := preparationReceipt{Version: preparationVersion, Scope: "preparation_only", SelectionStatus: "NOT_ATTEMPTED", Counts: map[string]int{}, Entries: []preparationEntry{}, EntriesComplete: true}
	r.TenantID = preparationID(job.TenantID, &r.MetadataIncomplete)
	r.JobID = preparationID(job.ID, &r.MetadataIncomplete)
	r.RunID = preparationID(run.ID, &r.MetadataIncomplete)
	if run.TenantID != job.TenantID || run.JobID != job.ID {
		r.TenantID = ""
		r.JobID = ""
		r.RunID = ""
		r.MetadataIncomplete = true
	}
	switch plan.mode {
	case modeOrdinary, modeTestRun, modeFull, modeUnanalyzed, modeSinceLast:
		r.Mode = string(plan.mode)
	default:
		r.MetadataIncomplete = true
	}
	return &preparationCollector{receipt: r, tenant: job.TenantID, byteLimit: preparationByteLimit}
}

func (c *preparationCollector) selection(n int) {
	if c != nil {
		c.receipt.SelectionStatus = "COMPLETE"
		c.receipt.Selected = &n
	}
}
func (c *preparationCollector) selectionFailed() {
	if c != nil {
		c.receipt.SelectionStatus = "FAILED"
		c.receipt.StopReason = "SELECTION_FAILED"
	}
}
func (c *preparationCollector) start(conv models.Conversation) {
	if c == nil {
		return
	}
	e := preparationEntry{Reference: preparationReference{State: "NONE"}}
	if conv.TenantID != c.tenant {
		e.MetadataIncomplete = true
	} else {
		e.ConversationID = preparationID(conv.ID, &e.MetadataIncomplete)
	}
	c.active = &e
	c.receipt.Visited++
}
func (c *preparationCollector) snapshot(conv models.Conversation, s *conversationSnapshot) {
	if c == nil || c.active == nil || s == nil {
		return
	}
	e := c.active
	if conv.TenantID != c.tenant || s.Manifest.TenantID != c.tenant || s.Manifest.ConversationID != conv.ID {
		e.MetadataIncomplete = true
		return
	}
	if s.Manifest.SchemaVersion == snapshotSchemaV1 {
		e.SchemaVersion = snapshotSchemaV1
	} else {
		e.MetadataIncomplete = true
	}
	if preparationDigest.MatchString(s.Digest) {
		e.Digest = strings.ToLower(s.Digest)
	} else {
		e.MetadataIncomplete = true
	}
	switch s.Manifest.Coverage {
	case coverageComplete, coveragePartial, coverageEmpty:
		e.Coverage = s.Manifest.Coverage
	default:
		e.MetadataIncomplete = true
	}
	for _, reason := range s.Manifest.CoverageReasons {
		switch reason {
		case reasonNoMessages, reasonHistoryWindowed, reasonUnsupportedContentType, reasonAttachmentNotRepresented, reasonAttachmentJSONInvalid:
			found := false
			for _, v := range e.CoverageReasons {
				if v == reason {
					found = true
				}
			}
			if !found {
				e.CoverageReasons = append(e.CoverageReasons, reason)
			}
		default:
			e.MetadataIncomplete = true
		}
	}
}
func (c *preparationCollector) reference(r preparationReference) {
	if c == nil || c.active == nil {
		return
	}
	e := c.active
	switch r.State {
	case "NONE", "LEGACY_NO_SNAPSHOT", "VERIFIED", "LOOKUP_FAILED", "PROVENANCE_UNVERIFIABLE":
		e.Reference.State = r.State
	default:
		e.MetadataIncomplete = true
		return
	}
	// Only a successfully verified linkage can expose bound reference identities.
	if r.State == "VERIFIED" {
		e.Reference.EvaluationID = preparationID(r.EvaluationID, &e.MetadataIncomplete)
		e.Reference.RunID = preparationID(r.RunID, &e.MetadataIncomplete)
		e.Reference.SnapshotID = preparationID(r.SnapshotID, &e.MetadataIncomplete)
	}
}
func (c *preparationCollector) finish(outcome string) {
	if c == nil || c.active == nil {
		return
	}
	switch outcome {
	case "EMPTY_SOURCE", "UNCHANGED_VERIFIED", "PREPARED_FOR_INFERENCE", "SNAPSHOT_ERROR", "SOURCE_VERSION_ERROR", "PREPARATION_INTERRUPTED":
	default:
		outcome = "PREPARATION_INTERRUPTED"
		c.active.MetadataIncomplete = true
	}
	c.active.Outcome = outcome
	c.receipt.Counts[outcome]++
	if len(c.receipt.Entries) < preparationEntryLimit {
		c.receipt.Entries = append(c.receipt.Entries, *c.active)
	}
	c.active = nil
}
func (c *preparationCollector) stop(reason string) {
	if c != nil {
		switch reason {
		case "CONTEXT_CANCELLED", "PREPARATION_INTERRUPTED", "SELECTION_FAILED":
			c.receipt.StopReason = reason
		default:
			c.receipt.MetadataIncomplete = true
		}
	}
}
func (c *preparationCollector) complete() {
	if c != nil {
		c.receipt.ScanComplete = true
	}
}
func (c *preparationCollector) interrupted() {
	if c != nil {
		if c.active != nil {
			c.finish("PREPARATION_INTERRUPTED")
			c.stop("PREPARATION_INTERRUPTED")
		}
	}
}

func (c *preparationCollector) freeze() *preparationReceipt {
	if c == nil {
		return nil
	}
	r := c.receipt
	r.Counts = make(map[string]int, len(c.receipt.Counts))
	for k, v := range c.receipt.Counts {
		r.Counts[k] = v
	}
	r.Entries = append([]preparationEntry{}, c.receipt.Entries...)
	for i := range r.Entries {
		r.Entries[i].CoverageReasons = append([]string(nil), r.Entries[i].CoverageReasons...)
	}
	if r.Selected != nil {
		n := *r.Selected
		r.Selected = &n
		u := n - r.Visited
		r.Unvisited = &u
	}
	for {
		r.OmittedEntries = r.Visited - len(r.Entries)
		r.EntriesComplete = r.OmittedEntries == 0
		b, err := json.Marshal(r)
		if err == nil && len(b) <= c.byteLimit {
			break
		}
		if len(r.Entries) == 0 {
			break
		} // envelope has only bounded fixed fields/counters
		r.Entries = r.Entries[:len(r.Entries)-1]
	}
	return &r
}

func preparationSummary(values map[string]interface{}, receipt *preparationReceipt) string {
	return observedSummary(values, receipt, nil)
}

func observedSummary(values map[string]interface{}, receipt *preparationReceipt, execution *executionReceipt) string {
	if receipt != nil {
		values["source_preparation"] = receipt
	}
	if execution != nil {
		values["source_execution"] = execution
	}
	b, _ := json.Marshal(values) // all values are typed primitives and frozen receipt fields
	return string(b)
}

func preparationGap(stage string) {
	log.Printf("[analyzer] source preparation receipt recording gap (stage=%s)", stage)
}
