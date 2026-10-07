package engine

import (
	"encoding/json"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

const executionVersion = "ccmai.source-execution.v1"
const executionCallLimit = 200
const executionMemberLimit = 200
const executionByteLimit = 128 * 1024

type executionCall struct {
	Sequence           int      `json:"sequence"`
	Method             string   `json:"method"`
	ItemCount          int      `json:"item_count"`
	MemberIDs          []string `json:"member_ids"`
	OmittedMembers     int      `json:"omitted_members"`
	MembersComplete    bool     `json:"members_complete"`
	MetadataIncomplete bool     `json:"metadata_incomplete"`
	Invocation         string   `json:"invocation_outcome"`
	UsageWrite         string   `json:"usage_write_outcome"`
	Parsing            string   `json:"parsing_outcome"`
	ItemsSaved         int      `json:"items_saved"`
	ItemsSaveFailed    int      `json:"items_save_failed"`
	ItemsNotPublished  int      `json:"items_not_published"`
	ItemsPending       int      `json:"items_pending"`
}

type executionReceipt struct {
	Version            string          `json:"version"`
	Scope              string          `json:"scope"`
	TenantID           string          `json:"tenant_id,omitempty"`
	JobID              string          `json:"job_id,omitempty"`
	RunID              string          `json:"run_id,omitempty"`
	Mode               string          `json:"mode,omitempty"`
	MetadataIncomplete bool            `json:"metadata_incomplete"`
	CallsBegun         int             `json:"calls_begun"`
	ResponseReturned   int             `json:"response_returned"`
	ErrorReturned      int             `json:"error_returned"`
	Interrupted        int             `json:"interrupted"`
	InFlight           int             `json:"in_flight"`
	ItemCount          int             `json:"item_count"`
	ItemsSaved         int             `json:"items_saved"`
	ItemsSaveFailed    int             `json:"items_save_failed"`
	ItemsNotPublished  int             `json:"items_not_published"`
	ItemsPending       int             `json:"items_pending"`
	UsageWrites        map[string]int  `json:"usage_writes"`
	Parsing            map[string]int  `json:"parsing"`
	Calls              []executionCall `json:"calls"`
	OmittedCalls       int             `json:"omitted_calls"`
	OmittedMembers     int             `json:"omitted_members"`
	EntriesComplete    bool            `json:"entries_complete"`
	MembersComplete    bool            `json:"members_complete"`
	ExecutionComplete  bool            `json:"execution_complete"`
	StopReason         string          `json:"stop_reason"`
}

// Run-owned observation only; nothing reads this collector to choose an effect.
type executionCollector struct {
	receipt   executionReceipt
	retained  []*executionCall
	active    *executionCall
	tenant    string
	members   int
	byteLimit int
}

func newExecutionCollector(job models.Job, run models.JobRun, plan runPlan) *executionCollector {
	r := executionReceipt{Version: executionVersion, Scope: "analyzer_provider_interface_only", StopReason: "NONE", Calls: []executionCall{}, UsageWrites: map[string]int{"NOT_ATTEMPTED": 0, "WRITE_SUCCEEDED": 0, "WRITE_FAILED": 0, "WRITE_OUTCOME_UNKNOWN": 0}, Parsing: map[string]int{"NOT_ATTEMPTED": 0, "ACCEPTED": 0, "REJECTED": 0, "NOT_SEPARATELY_OBSERVABLE": 0}}
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
	return &executionCollector{receipt: r, tenant: job.TenantID, byteLimit: executionByteLimit}
}

func (c *executionCollector) begin(method string, members []models.Conversation) {
	if c == nil {
		return
	}
	switch method {
	case "SINGLE", "BATCH":
	default:
		c.receipt.MetadataIncomplete = true
		return
	}
	e := &executionCall{Sequence: c.receipt.CallsBegun + 1, Method: method, ItemCount: len(members), MemberIDs: []string{}, Invocation: "IN_FLIGHT", UsageWrite: "NOT_ATTEMPTED", Parsing: "NOT_ATTEMPTED", ItemsPending: len(members)}
	if method == "SINGLE" {
		e.Parsing = "NOT_SEPARATELY_OBSERVABLE"
	}
	c.active = e
	c.receipt.CallsBegun++
	c.receipt.InFlight++
	c.receipt.ItemCount += len(members)
	c.receipt.ItemsPending += len(members)
	c.receipt.UsageWrites[e.UsageWrite]++
	c.receipt.Parsing[e.Parsing]++
	if len(c.retained) < executionCallLimit {
		for _, m := range members {
			invalid := m.TenantID != c.tenant
			id := ""
			if !invalid {
				id = preparationID(m.ID, &invalid)
			}
			if invalid {
				e.MetadataIncomplete = true
				c.receipt.MetadataIncomplete = true
			}
			if id != "" && c.members < executionMemberLimit {
				e.MemberIDs = append(e.MemberIDs, id)
				c.members++
			}
		}
		c.retained = append(c.retained, e)
	}
	e.OmittedMembers = e.ItemCount - len(e.MemberIDs)
	e.MembersComplete = e.OmittedMembers == 0
}

func (c *executionCollector) returned(err error) {
	if c == nil || c.active == nil || c.active.Invocation != "IN_FLIGHT" {
		return
	}
	c.receipt.InFlight--
	if err != nil {
		c.active.Invocation = "ERROR_RETURNED"
		c.receipt.ErrorReturned++
		c.notPublished()
	} else {
		c.active.Invocation = "RESPONSE_RETURNED"
		c.receipt.ResponseReturned++
	}
}
func (c *executionCollector) usageBegin() {
	if c == nil || c.active == nil {
		return
	}
	c.receipt.UsageWrites[c.active.UsageWrite]--
	c.active.UsageWrite = "WRITE_OUTCOME_UNKNOWN"
	c.receipt.UsageWrites[c.active.UsageWrite]++
}
func (c *executionCollector) usageReturned(err error) {
	if c == nil || c.active == nil {
		return
	}
	c.receipt.UsageWrites[c.active.UsageWrite]--
	if err != nil {
		c.active.UsageWrite = "WRITE_FAILED"
	} else {
		c.active.UsageWrite = "WRITE_SUCCEEDED"
	}
	c.receipt.UsageWrites[c.active.UsageWrite]++
}
func (c *executionCollector) parsed(err error) {
	if c == nil || c.active == nil {
		return
	}
	c.receipt.Parsing[c.active.Parsing]--
	if err != nil {
		c.active.Parsing = "REJECTED"
		c.notPublished()
	} else {
		c.active.Parsing = "ACCEPTED"
	}
	c.receipt.Parsing[c.active.Parsing]++
}
func (c *executionCollector) publication(published bool, err error) {
	if c == nil || c.active == nil || c.active.ItemsPending == 0 {
		return
	}
	if !published {
		c.notPublished()
		c.stop("OWNERSHIP_NOT_PUBLISHED")
		return
	}
	c.active.ItemsPending--
	c.receipt.ItemsPending--
	if err != nil {
		c.active.ItemsSaveFailed++
		c.receipt.ItemsSaveFailed++
	} else {
		c.active.ItemsSaved++
		c.receipt.ItemsSaved++
	}
}
func (c *executionCollector) notPublished() {
	if c == nil || c.active == nil {
		return
	}
	n := c.active.ItemsPending
	c.active.ItemsPending = 0
	c.receipt.ItemsPending -= n
	c.active.ItemsNotPublished += n
	c.receipt.ItemsNotPublished += n
}
func (c *executionCollector) stop(reason string) {
	if c == nil {
		return
	}
	switch reason {
	case "NONE", "CANCELLED", "OWNERSHIP_NOT_PUBLISHED", "PANIC", "PROVIDER_SETUP_FAILED", "PREPARATION_STOPPED", "EXECUTION_ERROR":
		c.receipt.StopReason = reason
	default:
		c.receipt.MetadataIncomplete = true
	}
}
func (c *executionCollector) panicStop() {
	if c == nil {
		return
	}
	if c.active != nil && c.active.Invocation == "IN_FLIGHT" {
		c.active.Invocation = "INTERRUPTED"
		c.receipt.InFlight--
		c.receipt.Interrupted++
	}
	c.receipt.ExecutionComplete = false
	c.stop("PANIC")
}
func (c *executionCollector) complete(truncated, cancelled bool, errors int) {
	if c == nil {
		return
	}
	c.receipt.ExecutionComplete = !truncated && !cancelled
	if cancelled {
		c.stop("CANCELLED")
	} else if truncated {
		c.stop("PREPARATION_STOPPED")
	} else if errors > 0 {
		if c.receipt.CallsBegun == 0 {
			c.stop("PREPARATION_STOPPED")
		} else {
			c.stop("EXECUTION_ERROR")
		}
	}
	if c.receipt.StopReason == "OWNERSHIP_NOT_PUBLISHED" {
		c.receipt.ExecutionComplete = false
	}
}
func (c *executionCollector) freeze() *executionReceipt {
	if c == nil {
		return nil
	}
	r := c.receipt
	r.Calls = []executionCall{}
	r.UsageWrites = map[string]int{}
	for k, v := range c.receipt.UsageWrites {
		r.UsageWrites[k] = v
	}
	r.Parsing = map[string]int{}
	for k, v := range c.receipt.Parsing {
		r.Parsing[k] = v
	}
	for _, call := range c.retained {
		e := *call
		e.MemberIDs = append([]string{}, call.MemberIDs...)
		r.Calls = append(r.Calls, e)
	}
	for {
		retainedMembers := 0
		for _, call := range r.Calls {
			retainedMembers += len(call.MemberIDs)
		}
		r.OmittedCalls = r.CallsBegun - len(r.Calls)
		r.OmittedMembers = r.ItemCount - retainedMembers
		r.EntriesComplete = r.OmittedCalls == 0
		r.MembersComplete = r.OmittedMembers == 0
		b, err := json.Marshal(r)
		if err == nil && len(b) <= c.byteLimit {
			break
		}
		if len(r.Calls) == 0 {
			break
		}
		r.Calls = r.Calls[:len(r.Calls)-1]
	}
	return &r
}
