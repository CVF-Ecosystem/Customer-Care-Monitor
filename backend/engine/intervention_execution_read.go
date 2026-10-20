package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"gorm.io/gorm"
)

var ErrExecutionReadInvalid = errors.New("invalid_request")
var ErrExecutionReadForbidden = errors.New("permission_denied")
var ErrExecutionReadMissing = errors.New("not_found")
var ErrExecutionReadUnavailable = errors.New("read_unavailable")

type ExecutionReadiness struct {
	State               string     `json:"state"`
	RequestID           string     `json:"request_id"`
	ServerNow           time.Time  `json:"server_now"`
	ExpectedCaseVersion *int       `json:"expected_case_version,omitempty"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	GrantHandle         string     `json:"grant_handle,omitempty"`
}

type ExecutionStatus struct {
	RequestID            string    `json:"request_id"`
	RequestIntent        string    `json:"request_intent"`
	ExecutionPresent     bool      `json:"execution_present"`
	Phase                string    `json:"phase"`
	ObservedAt           time.Time `json:"observed_at"`
	TerminalCode         string    `json:"terminal_code,omitempty"`
	PublishedCaseVersion *int      `json:"published_case_version,omitempty"`
	ProposalAvailable    *bool     `json:"proposal_available,omitempty"`
}

func executionReadIdentity(tenantID, actorID, caseID, requestID string) error {
	if (tenantID != models.SingleWorkspaceID && !waitDataUUIDPattern.MatchString(tenantID)) || !waitDataUUIDPattern.MatchString(actorID) || !waitDataUUIDPattern.MatchString(caseID) || !waitDataUUIDPattern.MatchString(requestID) {
		return ErrExecutionReadInvalid
	}
	return nil
}

func executionReadBase(tx *gorm.DB, tenantID, actorID, caseID, requestID string, writer bool) (models.InterventionCase, models.InterventionRerunRequest, error) {
	var member models.UserTenant
	if err := tx.Where("tenant_id = ? AND user_id = ?", tenantID, actorID).Take(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.InterventionCase{}, models.InterventionRerunRequest{}, ErrExecutionReadForbidden
		}
		return models.InterventionCase{}, models.InterventionRerunRequest{}, ErrExecutionReadUnavailable
	}
	allowed := false
	if writer {
		allowed = waitDataJobPermitted(member)
	} else if member.Role == "owner" || member.Role == "admin" {
		allowed = true
	} else if member.Role == "member" {
		var p map[string]string
		allowed = json.Unmarshal([]byte(member.Permissions), &p) == nil && strings.Contains(p["intervention_cases"], "r")
	}
	if !allowed {
		return models.InterventionCase{}, models.InterventionRerunRequest{}, ErrExecutionReadForbidden
	}
	var c models.InterventionCase
	if err := tx.Where("tenant_id = ? AND id = ?", tenantID, caseID).Take(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c, models.InterventionRerunRequest{}, ErrExecutionReadMissing
		}
		return c, models.InterventionRerunRequest{}, ErrExecutionReadUnavailable
	}
	var request models.InterventionRerunRequest
	if err := tx.Where("tenant_id = ? AND case_id = ? AND id = ?", tenantID, caseID, requestID).Take(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c, request, ErrExecutionReadMissing
		}
		return c, request, ErrExecutionReadUnavailable
	}
	return c, request, nil
}

func executionReadTransaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return db.DB.WithContext(ctx).Transaction(fn, &sql.TxOptions{ReadOnly: true})
}

// ReadExecutionReadiness is an advisory observation only. The POST rechecks
// every condition under its own locks and may still deny the execution.
func ReadExecutionReadiness(ctx context.Context, tenantID, actorID, caseID, requestID string, service *InterventionService) (*ExecutionReadiness, error) {
	if err := executionReadIdentity(tenantID, actorID, caseID, requestID); err != nil {
		return nil, err
	}
	tenantID, actorID, caseID, requestID = strings.ToLower(tenantID), strings.ToLower(actorID), strings.ToLower(caseID), strings.ToLower(requestID)
	result := &ExecutionReadiness{RequestID: requestID, ServerNow: time.Now().UTC(), State: "STALE_OR_INELIGIBLE"}
	err := executionReadTransaction(ctx, func(tx *gorm.DB) error {
		caseRow, request, err := executionReadBase(tx, tenantID, actorID, caseID, requestID, true)
		if err != nil {
			return err
		}
		var existing models.InterventionExecution
		if err = tx.Where("tenant_id = ? AND request_id = ?", tenantID, requestID).Take(&existing).Error; err == nil {
			result.State = "ALREADY_STARTED"
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrExecutionReadUnavailable
		}
		if service == nil || !service.SyntheticConfigured() {
			result.State = "PROCESS_DISABLED"
			return nil
		}
		if caseRow.Version < 1 || caseRow.ActiveExecutionID != nil || caseRow.WorkState != "WAITING_ON_DATA" || caseRow.Eligibility != "WAIT_DATA" || caseRow.Execution != "NOT_RUN" || caseRow.Disposition == "PENDING_REVIEW" || caseRow.HumanAssessment != "" || caseRow.OutcomeText != "" || request.Status != "AWAITING_EXECUTION_AUTHORITY" || request.Execution != "NOT_RUN" || request.CaseVersion != caseRow.Version || request.JobID != caseRow.JobID || request.ConversationID != caseRow.ConversationID || caseRow.SnapshotID == nil || request.OriginalSnapshotID != *caseRow.SnapshotID || request.OriginalSnapshotDigest != caseRow.SnapshotDigest || request.ActorID != actorID {
			return nil
		}
		var job models.Job
		if err = tx.Where("tenant_id = ? AND id = ?", tenantID, caseRow.JobID).Take(&job).Error; err != nil {
			return executionReadDependency(err)
		}
		var conv models.Conversation
		if err = tx.Where("tenant_id = ? AND id = ?", tenantID, caseRow.ConversationID).Take(&conv).Error; err != nil {
			return executionReadDependency(err)
		}
		var channel models.Channel
		if err = tx.Where("tenant_id = ? AND id = ?", tenantID, conv.ChannelID).Take(&channel).Error; err != nil {
			return executionReadDependency(err)
		}
		if job.JobType != "intervention" || job.ScheduleType != "manual" || !job.IsActive || channel.ChannelType != "pancake" || !channel.IsDemoFixture || caseRow.UseCase != "pancake_complaint_wait" {
			return nil
		}
		var inputs []string
		if json.Unmarshal([]byte(job.InputChannelIDs), &inputs) != nil {
			return ErrExecutionReadUnavailable
		}
		included := false
		for _, id := range inputs {
			if id == channel.ID {
				included = true
			}
		}
		if !included {
			return nil
		}
		var original models.AnalysisSnapshot
		if err = tx.Where("tenant_id = ? AND id = ?", tenantID, *caseRow.SnapshotID).Take(&original).Error; err != nil {
			return executionReadDependency(err)
		}
		if original.Digest != caseRow.SnapshotDigest || !VerifySnapshotProvenance(original, tenantID, conv.ID, caseRow.RunID) {
			return nil
		}
		var messages []models.Message
		if err = tx.Where("tenant_id = ? AND conversation_id = ?", tenantID, conv.ID).Order("sent_at ASC, id ASC").Limit(waitDataMaxMessages + 1).Find(&messages).Error; err != nil {
			return ErrExecutionReadUnavailable
		}
		if len(messages) > waitDataMaxMessages {
			return ErrExecutionReadUnavailable
		}
		digest, coverage, sourceErr := M3CurrentSourceObservation(conv, messages)
		if sourceErr != nil {
			return ErrExecutionReadUnavailable
		}
		var revision, latest models.InterventionDataRevision
		if err = tx.Where("tenant_id = ? AND case_id = ? AND id = ?", tenantID, caseID, request.DataRevisionID).Take(&revision).Error; err != nil {
			return executionReadDependency(err)
		}
		if err = tx.Where("tenant_id = ? AND case_id = ?", tenantID, caseID).Order("case_version DESC, id DESC").Take(&latest).Error; err != nil {
			return executionReadDependency(err)
		}
		if !VerifyInterventionDataRevision(revision) || !VerifyInterventionDataRevision(latest) || latest.ID != revision.ID || revision.Digest != request.DataRevisionDigest || revision.Coverage != coverageComplete || revision.Digest != digest || coverage != coverageComplete || revision.OriginalSnapshotID != original.ID || revision.OriginalSnapshotDigest != original.Digest {
			return nil
		}
		var proposalCount, runningCount int64
		if tx.Model(&models.InterventionProposal{}).Where("tenant_id = ? AND case_id = ?", tenantID, caseID).Count(&proposalCount).Error != nil || tx.Model(&models.JobRun{}).Where("tenant_id = ? AND job_id = ? AND status = ?", tenantID, job.ID, "running").Count(&runningCount).Error != nil {
			return ErrExecutionReadUnavailable
		}
		if proposalCount != 0 || runningCount != 0 {
			return nil
		}
		var grant models.InterventionExecutionGrant
		if err = tx.Where("tenant_id = ? AND request_id = ? AND actor_id = ?", tenantID, requestID, actorID).Take(&grant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				result.State = "AWAITING_OPERATOR_AUTHORITY"
				return nil
			}
			return ErrExecutionReadUnavailable
		}
		result.ServerNow = time.Now().UTC()
		if grant.CaseID != caseID || grant.JobID != job.ID || grant.DataRevisionID != revision.ID || grant.State != "ISSUED" || !grant.Deadline.After(result.ServerNow) || grant.Provider != service.Provider || grant.Model != service.Model || grant.PolicyVersion != service.PolicyVersion || grant.PromptVersion != service.PromptVersion || grant.ConfigRef != service.ConfigRef || grant.IssuerID == "" || grant.EvidenceRef == "" {
			result.State = "AWAITING_OPERATOR_AUTHORITY"
			return nil
		}
		var budget models.InterventionBudget
		if err = tx.Where("tenant_id = ? AND job_id = ?", tenantID, job.ID).Take(&budget).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return ErrExecutionReadUnavailable
		}
		if budget.Cap <= 0 || budget.Consumed >= budget.Cap {
			return nil
		}
		result.ServerNow = time.Now().UTC()
		if !grant.Deadline.After(result.ServerNow) {
			result.State = "AWAITING_OPERATOR_AUTHORITY"
			return nil
		}
		version := caseRow.Version
		deadline := grant.Deadline
		result.State, result.ExpectedCaseVersion, result.ExpiresAt, result.GrantHandle = "READY", &version, &deadline, grant.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func executionReadDependency(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil // A known missing prerequisite is an ineligible source, not a failed read.
	}
	return ErrExecutionReadUnavailable
}

func executionRowMatchesRequest(row models.InterventionExecution, request models.InterventionRerunRequest) bool {
	return row.TenantID == request.TenantID && row.CaseID == request.CaseID && row.JobID == request.JobID && row.ConversationID == request.ConversationID && row.RequestID == request.ID && row.RevisionID == request.DataRevisionID && row.ActorID == request.ActorID && row.OriginalSnapshotID == request.OriginalSnapshotID && row.OriginalSnapshotDigest == request.OriginalSnapshotDigest && row.ExpectedCaseVersion == request.CaseVersion && row.ClaimedCaseVersion == request.CaseVersion
}

func publishedLinksMatch(row models.InterventionExecution, proposal models.InterventionProposal, event models.InterventionHumanEvent, after map[string]any) bool {
	return proposal.TenantID == row.TenantID && proposal.CaseID == row.CaseID && proposal.RunID == row.RunID && proposal.SnapshotID == row.SnapshotID && proposal.SnapshotDigest == row.SnapshotDigest && proposal.Revision > row.ClaimedCaseVersion &&
		event.TenantID == row.TenantID && event.CaseID == row.CaseID && event.ActorID == row.ActorID && event.Action == "EXECUTE_RERUN" && event.RerunRequestID != nil && *event.RerunRequestID == row.RequestID && event.ProposalID != nil && *event.ProposalID == proposal.ID && event.DataRevisionID != nil && *event.DataRevisionID == row.RevisionID && event.CaseVersion == proposal.Revision && event.SnapshotID == proposal.SnapshotID && event.SnapshotDigest == proposal.SnapshotDigest &&
		after["execution_id"] == row.ID && after["request_id"] == row.RequestID && after["revision_id"] == row.RevisionID && after["run_id"] == row.RunID && after["snapshot_id"] == row.SnapshotID && after["snapshot_digest"] == row.SnapshotDigest && after["active_proposal_id"] == proposal.ID
}

func publishedExecutionLineage(tx *gorm.DB, row models.InterventionExecution, request models.InterventionRerunRequest) (int, error) {
	var run models.JobRun
	if err := tx.Where("tenant_id = ? AND job_id = ? AND id = ? AND status = ?", row.TenantID, row.JobID, row.RunID, "success").Take(&run).Error; err != nil {
		return 0, ErrExecutionReadUnavailable
	}
	if run.FinishedAt == nil {
		return 0, ErrExecutionReadUnavailable
	}
	var proposal models.InterventionProposal
	if err := tx.Where("tenant_id = ? AND case_id = ? AND run_id = ?", row.TenantID, row.CaseID, row.RunID).Take(&proposal).Error; err != nil {
		return 0, ErrExecutionReadUnavailable
	}
	var event models.InterventionHumanEvent
	if err := tx.Where("tenant_id = ? AND case_id = ? AND rerun_request_id = ? AND action = ?", row.TenantID, row.CaseID, row.RequestID, "EXECUTE_RERUN").Take(&event).Error; err != nil {
		return 0, ErrExecutionReadUnavailable
	}
	var after map[string]any
	if json.Unmarshal([]byte(event.After), &after) != nil || !executionRowMatchesRequest(row, request) || !publishedLinksMatch(row, proposal, event, after) {
		return 0, ErrExecutionReadUnavailable
	}
	return proposal.Revision, nil
}

func ReadExecutionStatus(ctx context.Context, tenantID, actorID, caseID, requestID string) (*ExecutionStatus, error) {
	if err := executionReadIdentity(tenantID, actorID, caseID, requestID); err != nil {
		return nil, err
	}
	tenantID, actorID, caseID, requestID = strings.ToLower(tenantID), strings.ToLower(actorID), strings.ToLower(caseID), strings.ToLower(requestID)
	result := &ExecutionStatus{RequestID: requestID, Phase: "NOT_STARTED", ObservedAt: time.Now().UTC()}
	err := executionReadTransaction(ctx, func(tx *gorm.DB) error {
		_, request, err := executionReadBase(tx, tenantID, actorID, caseID, requestID, false)
		if err != nil {
			return err
		}
		result.RequestIntent = request.Status
		var row models.InterventionExecution
		if err = tx.Where("tenant_id = ? AND request_id = ?", tenantID, requestID).Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return ErrExecutionReadUnavailable
		}
		if !executionRowMatchesRequest(row, request) {
			return ErrExecutionReadUnavailable
		}
		result.ExecutionPresent = true
		switch row.State {
		case "CLAIMED", "CALL_MAY_HAVE_STARTED", "RESPONSE_RECORDED":
			result.Phase = row.State
		case "PUBLISHED":
			version, err := publishedExecutionLineage(tx, row, request)
			if err != nil {
				return err
			}
			available := true
			result.Phase, result.PublishedCaseVersion, result.ProposalAvailable = "PUBLISHED", &version, &available
		case "DENIED", "STALE_SOURCE", "OUTPUT_INVALID", "PROVIDER_UNKNOWN", "SETTLEMENT_FAILED", "CANCELLED", "RECOVERY_REQUIRED":
			result.Phase, result.TerminalCode = "MANUAL_HOLD", row.State
			if row.TerminalCode != "" && row.TerminalCode != row.State {
				return ErrExecutionReadUnavailable
			}
		default:
			return ErrExecutionReadUnavailable
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
