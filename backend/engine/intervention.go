package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInterventionManualOnly = errors.New("intervention_manual_only")
var ErrInterventionUnavailable = errors.New("intervention_unavailable")
var ErrInterventionDenied = errors.New("intervention_source_denied")
var ErrInterventionBusy = errors.New("intervention_busy")
var ErrInterventionMissing = errors.New("intervention_not_found")

// InterventionTransport is installed only by an explicit server-owned configuration.
// A production implementation must consume exactly one request per invocation.
type InterventionTransport interface {
	AnalyzeChatOnce(context.Context, string, string, int64) (ai.AIResponse, error)
}
type InterventionService struct {
	Transport       InterventionTransport
	PolicyVersion   string
	PromptVersion   string
	Prompt          string
	MaxRequestBytes int64
	Timeout         time.Duration
}

type interventionOutput struct {
	Schema         string        `json:"schema"`
	ConversationID string        `json:"conversation_id"`
	SnapshotDigest string        `json:"snapshot_digest"`
	Kind           string        `json:"kind"`
	Explanation    string        `json:"explanation"`
	NextAction     string        `json:"next_action"`
	Evidence       []evidenceRef `json:"evidence_refs"`
}

func (s *InterventionService) configured() bool {
	return s != nil && s.Transport != nil && s.PromptVersion != "" && s.Prompt != "" && s.MaxRequestBytes > 0 && s.MaxRequestBytes <= 6000 && s.Timeout > 0 && s.Timeout <= 60*time.Second
}

// RunManual admits one selected synthetic conversation. All error codes are safe
// to expose; neither provider text nor transcript enters a receipt or log.
func (s *InterventionService) RunManual(ctx context.Context, tenantID, actorID, jobID, conversationID string) (*models.InterventionCase, error) {
	if !s.configured() {
		return nil, ErrInterventionUnavailable
	}
	if s.PolicyVersion != "ccma.synthetic.intervention.v1" {
		return nil, ErrInterventionDenied
	}
	var job models.Job
	if db.DB.Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&job).Error != nil {
		return nil, ErrInterventionMissing
	}
	if job.JobType != "intervention" || job.ScheduleType != "manual" || !job.IsActive {
		return nil, ErrInterventionManualOnly
	}
	var conv models.Conversation
	if db.DB.Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&conv).Error != nil {
		return nil, ErrInterventionMissing
	}
	var channel models.Channel
	if db.DB.Where("id = ? AND tenant_id = ?", conv.ChannelID, tenantID).First(&channel).Error != nil {
		return nil, ErrInterventionMissing
	}
	// The marker is a server-owned column; channel metadata or a caller parameter is insufficient.
	allowedChannel := false
	var inputs []string
	if json.Unmarshal([]byte(job.InputChannelIDs), &inputs) != nil {
		return nil, ErrInterventionUnavailable
	}
	for _, id := range inputs {
		if id == channel.ID {
			allowedChannel = true
		}
	}
	if !allowedChannel || !channel.IsDemoFixture || channel.ChannelType != "pancake" {
		return nil, ErrInterventionDenied
	}
	snap, err := loadConversationSnapshot(conv, time.Time{})
	if err != nil {
		return nil, ErrInterventionUnavailable
	}
	res, err := ReserveJobRun(ctx, job, s.Timeout+time.Minute)
	if err != nil {
		if errors.Is(err, ErrJobBusy) {
			return nil, ErrInterventionBusy
		}
		return nil, ErrInterventionUnavailable
	}
	consumed := false
	res.used.Do(func() { consumed = true })
	if !consumed {
		return nil, ErrInterventionUnavailable
	}
	defer res.owner.release()
	runID := res.RunID()
	now := time.Now().UTC()
	caseRow := models.InterventionCase{ID: pkg.NewUUID(), TenantID: tenantID, JobID: jobID, RunID: runID, ConversationID: conversationID, UseCase: "pancake_complaint_wait", Version: 1, Eligibility: "WAIT_DATA", Execution: "NOT_RUN", Disposition: "NOT_CREATED", WorkState: "WAITING_ON_DATA", ReasonCode: "SOURCE_INCOMPLETE", NextAction: "complete_synthetic_source", ActorID: actorID, PolicyVersion: s.PolicyVersion, PromptVersion: s.PromptVersion, SchemaVersion: "ccma.intervention.v1", CreatedAt: now, UpdatedAt: now}
	saved, err := snap.record(runID)
	if err != nil {
		_, _ = closeOwnedRun(res.owner, &res.run, res.bound(), "snapshot_unavailable")
		return nil, ErrInterventionUnavailable
	}
	snapshotRow := &saved
	caseRow.SnapshotID = &saved.ID
	caseRow.SnapshotDigest = saved.Digest
	anchorAvailable := false
	for _, m := range snap.Manifest.Messages {
		content := strings.ToLower(snap.contents[m.MessageID])
		if m.SenderType == "customer" && (strings.Contains(content, "chờ") || strings.Contains(content, "khiếu nại") || strings.Contains(content, "phàn nàn")) {
			anchorAvailable = true
		}
	}
	if snap.Manifest.Coverage == coverageComplete && anchorAvailable {
		caseRow.Eligibility = "ELIGIBLE"
		caseRow.WorkState = "OPEN"
		caseRow.ReasonCode = ""
		caseRow.NextAction = ""
	}
	// The accepted coordinator owns the JobRun; update only the stable case and
	// this run's immutable snapshot here. Job is locked before case/snapshot.
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.Job
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&locked).Error != nil {
			return ErrInterventionMissing
		}
		if locked.JobType != "intervention" || locked.ScheduleType != "manual" || !locked.IsActive {
			return ErrInterventionManualOnly
		}
		var prior models.InterventionCase
		find := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND job_id = ? AND conversation_id = ? AND use_case = ?", tenantID, jobID, conversationID, caseRow.UseCase).First(&prior)
		if find.Error != nil && !errors.Is(find.Error, gorm.ErrRecordNotFound) {
			return ErrInterventionUnavailable
		}
		if find.Error == nil {
			caseRow.ID = prior.ID
			caseRow.CreatedAt = prior.CreatedAt
			caseRow.Version = prior.Version + 1
			// A pending human review belongs to its prior immutable proposal; do not
			// silently replace it with a rerun before M3 decides.
			owned, ownerErr := m3HasHumanOwnership(tx, prior)
			if ownerErr != nil {
				return ErrInterventionUnavailable
			}
			if owned {
				return ErrInterventionBusy
			}
		}
		if snapshotRow != nil {
			if tx.Create(snapshotRow).Error != nil {
				return ErrInterventionUnavailable
			}
		}
		if find.Error == nil {
			r := tx.Model(&models.InterventionCase{}).Where("id = ? AND version = ?", prior.ID, prior.Version).Updates(map[string]any{"run_id": runID, "version": caseRow.Version, "snapshot_id": caseRow.SnapshotID, "snapshot_digest": caseRow.SnapshotDigest, "eligibility": caseRow.Eligibility, "execution": caseRow.Execution, "disposition": caseRow.Disposition, "work_state": caseRow.WorkState, "reason_code": caseRow.ReasonCode, "next_action": caseRow.NextAction, "actor_id": actorID, "policy_version": s.PolicyVersion, "prompt_version": s.PromptVersion, "schema_version": "ccma.intervention.v1", "updated_at": now})
			if r.Error != nil || r.RowsAffected != 1 {
				return ErrInterventionUnavailable
			}
		} else if tx.Create(&caseRow).Error != nil {
			return ErrInterventionUnavailable
		}
		return nil
	})
	if err != nil {
		_, _ = closeOwnedRun(res.owner, &res.run, res.bound(), "intervention_unavailable")
		return nil, err
	}
	if caseRow.Eligibility != "ELIGIBLE" {
		if err := s.finish(res, &caseRow, "WAIT_DATA", "UNKNOWN"); err != nil {
			return nil, err
		}
		return &caseRow, nil
	}
	// The ledger is persistent and keyed to the stable tenant/job, never to a
	// caller-selected run ID. A missing cap denies the call.
	var attempt models.InterventionAttempt
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var budget models.InterventionBudget
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND job_id = ?", tenantID, jobID).First(&budget).Error != nil {
			return ErrInterventionUnavailable
		}
		if budget.Cap <= 0 || budget.Consumed >= budget.Cap {
			return ErrInterventionUnavailable
		}
		r := tx.Model(&models.InterventionBudget{}).Where("id = ? AND consumed = ? AND cap = ?", budget.ID, budget.Consumed, budget.Cap).Update("consumed", budget.Consumed+1)
		if r.Error != nil || r.RowsAffected != 1 {
			return ErrInterventionUnavailable
		}
		attempt = models.InterventionAttempt{ID: pkg.NewUUID(), TenantID: tenantID, JobID: jobID, RunID: runID, BudgetID: budget.ID, State: "CONSUMED", UsageState: "UNKNOWN", CreatedAt: time.Now().UTC()}
		if tx.Create(&attempt).Error != nil {
			return ErrInterventionUnavailable
		}
		r = tx.Model(&models.InterventionCase{}).Where("id = ? AND execution = ?", caseRow.ID, "NOT_RUN").Update("execution", "LLM_PENDING")
		if r.Error != nil || r.RowsAffected != 1 {
			return ErrInterventionUnavailable
		}
		return nil
	})
	if err != nil {
		if err := s.finish(res, &caseRow, "BUDGET_UNAVAILABLE", "UNKNOWN"); err != nil {
			return nil, err
		}
		return &caseRow, nil
	}
	caseRow.Execution = "LLM_PENDING"
	callCtx, cancel := context.WithTimeout(res.Context(), s.Timeout)
	defer cancel()
	prompt := s.Prompt + "\nReturn one JSON object with schema ccma.intervention.v1, conversation_id " + conversationID + ", snapshot_digest " + snap.Digest + ", kind complaint or waiting, explanation, next_action, and evidence_refs with canonical message_id and exact quote."
	response, callErr := s.Transport.AnalyzeChatOnce(callCtx, prompt, snap.Transcript, s.MaxRequestBytes)
	if callErr != nil {
		if ai.IsObservedChatOnceResponse(callErr) && response.Model != "" && response.Provider != "" && response.UsagePresence != nil {
			usage := settledUsage(response)
			if err := settleInterventionResponse(attempt.ID, response, usage); err != nil {
				if endErr := s.finish(res, &caseRow, "SETTLEMENT_FAILED", "UNKNOWN"); endErr != nil {
					return nil, endErr
				}
				return &caseRow, nil
			}
			if err := s.finish(res, &caseRow, "OUTPUT_INVALID", usage); err != nil {
				return nil, err
			}
			return &caseRow, nil
		}
		if err := s.finish(res, &caseRow, "PROVIDER_UNKNOWN", "UNKNOWN"); err != nil {
			return nil, err
		}
		return &caseRow, nil
	}
	usage := settledUsage(response)
	if err := settleInterventionResponse(attempt.ID, response, usage); err != nil {
		if endErr := s.finish(res, &caseRow, "SETTLEMENT_FAILED", "UNKNOWN"); endErr != nil {
			return nil, endErr
		}
		return &caseRow, nil
	}
	var out interventionOutput
	if response.Model == "" || response.Provider == "" || json.Unmarshal([]byte(response.Content), &out) != nil || validateInterventionOutput(snap, conversationID, out) != nil {
		if err := s.finish(res, &caseRow, "OUTPUT_INVALID", usage); err != nil {
			return nil, err
		}
		return &caseRow, nil
	}
	if res.Context().Err() != nil {
		if err := s.finish(res, &caseRow, "CANCELLED", usage); err != nil {
			return nil, err
		}
		return &caseRow, nil
	}
	// Owner publication serializes cancellation against this atomic write. Lock
	// Job before Run to honor the repository lock order; source is re-read here.
	allowed, pubErr := res.owner.publish(func() error {
		return db.DB.Transaction(func(tx *gorm.DB) error {
			var lockedJob models.Job
			if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&lockedJob).Error != nil {
				return errors.New("STALE_SOURCE")
			}
			var run models.JobRun
			if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ? AND status = ?", runID, tenantID, "running").First(&run).Error != nil || res.Context().Err() != nil {
				return errors.New("CANCELLED")
			}
			var current models.Conversation
			if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&current).Error != nil {
				return errors.New("STALE_SOURCE")
			}
			var messages []models.Message
			if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("conversation_id = ? AND tenant_id = ?", conversationID, tenantID).Order("sent_at ASC, id ASC").Find(&messages).Error != nil {
				return errors.New("STALE_SOURCE")
			}
			if CompareSnapshotToCurrentMessages(string(snap.ManifestJSON), messages) != SourceIntegrityBoundCurrentnessUnverified {
				return errors.New("STALE_SOURCE")
			}
			ev, _ := json.Marshal(out.Evidence)
			proposal := models.InterventionProposal{ID: pkg.NewUUID(), TenantID: tenantID, CaseID: caseRow.ID, Revision: caseRow.Version, RunID: runID, SnapshotID: *caseRow.SnapshotID, SnapshotDigest: snap.Digest, PromptVersion: s.PromptVersion, PolicyVersion: s.PolicyVersion, SchemaVersion: "ccma.intervention.v1", Kind: out.Kind, Explanation: out.Explanation, NextAction: out.NextAction, Evidence: string(ev), Provider: response.Provider, Model: response.Model, UsageState: usage, CreatedAt: time.Now().UTC()}
			if tx.Create(&proposal).Error != nil {
				return errors.New("PERSIST_FAILED")
			}
			r := tx.Model(&models.InterventionCase{}).Where("id = ? AND run_id = ? AND version = ? AND execution = ?", caseRow.ID, runID, caseRow.Version, "LLM_PENDING").Updates(map[string]any{"execution": "LLM_SUCCEEDED", "disposition": "PENDING_REVIEW", "updated_at": time.Now().UTC()})
			if r.Error != nil || r.RowsAffected != 1 {
				return errors.New("PERSIST_FAILED")
			}
			if tx.Create(&models.InterventionReceipt{ID: pkg.NewUUID(), TenantID: tenantID, CaseID: caseRow.ID, RunID: runID, Code: "PROPOSAL_VALID", UsageState: usage, CreatedAt: time.Now().UTC()}).Error != nil {
				return errors.New("PERSIST_FAILED")
			}
			end := time.Now().UTC()
			jr := tx.Model(&models.JobRun{}).Where("id = ? AND tenant_id = ? AND status = ?", runID, tenantID, "running").Updates(map[string]any{"status": "success", "finished_at": end})
			if jr.Error != nil || jr.RowsAffected != 1 {
				return errors.New("PERSIST_FAILED")
			}
			return nil
		})
	})
	if !allowed {
		pubErr = errors.New("CANCELLED")
	}
	if pubErr != nil {
		if err := s.finish(res, &caseRow, safeInterventionPublicationCode(pubErr), usage); err != nil {
			return nil, err
		}
		return &caseRow, nil
	}
	caseRow.Execution = "LLM_SUCCEEDED"
	caseRow.Disposition = "PENDING_REVIEW"
	return &caseRow, nil
}

func validateInterventionOutput(snap *conversationSnapshot, conversationID string, out interventionOutput) error {
	if out.Schema != "ccma.intervention.v1" || out.ConversationID != conversationID || out.SnapshotDigest != snap.Digest || (out.Kind != "complaint" && out.Kind != "waiting") || strings.TrimSpace(out.Explanation) == "" || strings.TrimSpace(out.NextAction) == "" {
		return fmt.Errorf("output_contract")
	}
	if err := snap.validateEvidenceRefs(out.Evidence); err != nil {
		return err
	}
	anchor := false
	for _, ref := range out.Evidence {
		for _, m := range snap.Manifest.Messages {
			if m.MessageID == ref.MessageID && m.SenderType == "customer" && ((out.Kind == "complaint" && (strings.Contains(strings.ToLower(ref.Quote), "khiếu nại") || strings.Contains(strings.ToLower(ref.Quote), "phàn nàn"))) || (out.Kind == "waiting" && strings.Contains(strings.ToLower(ref.Quote), "chờ"))) {
				anchor = true
			}
		}
	}
	if !anchor {
		return fmt.Errorf("complaint_anchor_missing")
	}
	return nil
}

func safeInterventionPublicationCode(err error) string {
	if err == nil {
		return "PERSIST_FAILED"
	}
	switch err.Error() {
	case "STALE_SOURCE":
		return "STALE_SOURCE"
	case "CANCELLED":
		return "CANCELLED"
	default:
		return "PERSIST_FAILED"
	}
}

type interventionUsage struct {
	state          string
	inputPresence  string
	outputPresence string
	input          *int64
	output         *int64
}

func usageFromResponse(response ai.AIResponse) interventionUsage {
	u := interventionUsage{state: "UNKNOWN", inputPresence: "unknown", outputPresence: "unknown"}
	if response.UsagePresence == nil {
		return u
	}
	p := response.UsagePresence
	u.inputPresence = string(p.InputTokens.Status)
	u.outputPresence = string(p.OutputTokens.Status)
	if p.InputTokens.Status == ai.AIUsagePresenceKnown && p.InputTokens.Value != nil {
		v := *p.InputTokens.Value
		u.input = &v
	}
	if p.OutputTokens.Status == ai.AIUsagePresenceKnown && p.OutputTokens.Value != nil {
		v := *p.OutputTokens.Value
		u.output = &v
	}
	if u.input != nil && u.output != nil && p.Complete {
		u.state = "KNOWN"
	} else if u.input != nil || u.output != nil {
		u.state = "PARTIAL"
	}
	return u
}

func settledUsage(response ai.AIResponse) string { return usageFromResponse(response).state }

// Settle the physical response independently of proposal validation. A failed
// settlement leaves the already consumed budget slot in place and blocks publish.
func settleInterventionResponse(attemptID string, response ai.AIResponse, usageState string) error {
	u := usageFromResponse(response)
	if u.state != usageState {
		return ErrInterventionUnavailable
	}
	r := db.DB.Model(&models.InterventionAttempt{}).Where("id = ? AND state = ?", attemptID, "CONSUMED").Updates(map[string]any{
		"state": "RESPONSE_RECEIVED", "usage_state": u.state, "input_usage_presence": u.inputPresence, "output_usage_presence": u.outputPresence, "input_tokens": u.input, "output_tokens": u.output, "response_model": response.Model,
	})
	if r.Error != nil || r.RowsAffected != 1 {
		return ErrInterventionUnavailable
	}
	return nil
}

func (s *InterventionService) finish(res *JobRunReservation, c *models.InterventionCase, code, usage string) error {
	now := time.Now().UTC()
	cancelled := res.owner.beginTerminal()
	if cancelled {
		code = "CANCELLED"
	}
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var job models.Job
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", c.JobID, c.TenantID).First(&job).Error != nil {
			return ErrInterventionUnavailable
		}
		var run models.JobRun
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ? AND status = ?", c.RunID, c.TenantID, "running").First(&run).Error != nil {
			return ErrInterventionUnavailable
		}
		state := "ERROR"
		if code == "WAIT_DATA" {
			state = "NOT_RUN"
		}
		if code == "CANCELLED" {
			state = "CANCELLED"
		}
		if code == "PROVIDER_UNKNOWN" {
			state = "USAGE_UNKNOWN"
		}
		r := tx.Model(&models.InterventionCase{}).Where("id = ? AND tenant_id = ? AND run_id = ?", c.ID, c.TenantID, c.RunID).Updates(map[string]any{"execution": state, "reason_code": code, "updated_at": now})
		if r.Error != nil || r.RowsAffected != 1 {
			return ErrInterventionUnavailable
		}
		if tx.Create(&models.InterventionReceipt{ID: pkg.NewUUID(), TenantID: c.TenantID, CaseID: c.ID, RunID: c.RunID, Code: code, UsageState: usage, CreatedAt: now}).Error != nil {
			return ErrInterventionUnavailable
		}
		if code == "PROVIDER_UNKNOWN" || code == "CANCELLED" {
			ar := tx.Model(&models.InterventionAttempt{}).Where("run_id = ? AND state = ?", c.RunID, "CONSUMED").Update("state", "UNKNOWN")
			if ar.Error != nil {
				return ErrInterventionUnavailable
			}
		}
		jobStatus := "error"
		if code == "CANCELLED" {
			jobStatus = "cancelled"
		}
		jr := tx.Model(&models.JobRun{}).Where("id = ? AND tenant_id = ? AND status = ?", c.RunID, c.TenantID, "running").Updates(map[string]any{"status": jobStatus, "finished_at": now})
		if jr.Error != nil || jr.RowsAffected != 1 {
			return ErrInterventionUnavailable
		}
		c.Execution = state
		c.ReasonCode = code
		return nil
	})
	if err != nil {
		res.owner.terminalFailed()
		return ErrInterventionUnavailable
	}
	return nil
}
