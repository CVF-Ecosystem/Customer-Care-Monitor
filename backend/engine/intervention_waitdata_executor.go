package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// This installation is process-local and server-owned. Ordinary routing leaves
// it nil; neither the request nor a grant supplies a transport or secret.
var WaitDataExecutor *InterventionService

var ErrWaitDataLineage = errors.New("intervention_lineage_preserved")
var ErrWaitDataAdmissionUncertain = errors.New("admission_uncertain")
var ErrWaitDataCancelled = errors.New("execution_cancelled")
var ErrWaitDataDenied = errors.New("execution_denied_before_call")

func waitDataReadError(err,errorIfMissing error) error {
	if err == nil { return nil }
	if errors.Is(err,gorm.ErrRecordNotFound) { return errorIfMissing }
	return ErrWaitDataUnavailable
}

// GuardWaitDataLineage is used by supported destructive writers before they
// touch source or immutable run evidence. Tenant is the common admission parent.
// Empty job/channel scopes mean the tenant-wide demo reset.
func GuardWaitDataLineage(tx *gorm.DB, tenantID, jobID, channelID string) error {
	if err := waitDataLockTenant(tx, tenantID); err != nil {
		return err
	}
	var n int64
	q := tx.Model(&models.InterventionExecution{}).Where("intervention_executions.tenant_id = ?", tenantID)
	if jobID != "" {
		q = q.Where("intervention_executions.job_id = ?", jobID)
	}
	if channelID != "" {
		q = q.Joins("JOIN conversations ON conversations.id = intervention_executions.conversation_id AND conversations.tenant_id = intervention_executions.tenant_id").Where("conversations.channel_id = ?", channelID)
	}
	if q.Count(&n).Error != nil {
		return ErrWaitDataUnavailable
	}
	if n != 0 {
		return ErrWaitDataLineage
	}
	return nil
}

type WaitDataExecuteInput struct {
	ExpectedCaseVersion int    `json:"expected_case_version"`
	GrantID             string `json:"grant_id"`
}

type waitDataExecution struct {
	row      models.InterventionExecution
	owner    *JobRunOwner
	snapshot *conversationSnapshot
}

func HashWaitDataExecute(tenantID, caseID, requestID string, in WaitDataExecuteInput) string {
	b, _ := json.Marshal(struct {
		TenantID            string `json:"tenant_id"`
		CaseID              string `json:"case_id"`
		RequestID           string `json:"request_id"`
		ExpectedCaseVersion int    `json:"expected_case_version"`
		GrantID             string `json:"grant_id"`
	}{strings.ToLower(tenantID), strings.ToLower(caseID), strings.ToLower(requestID), in.ExpectedCaseVersion, strings.ToLower(in.GrantID)})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func waitDataLockTenant(tx *gorm.DB, tenantID string) error {
	var tenant models.Tenant
	return waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", tenantID).Take(&tenant).Error,ErrWaitDataMissing)
}

func waitDataCheckActor(tx *gorm.DB, tenantID, actorID string) error {
	var actor models.UserTenant
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND user_id = ?", tenantID, actorID).Take(&actor).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrWaitDataUnavailable
	}
	if err != nil || !waitDataJobPermitted(actor) {
		return ErrWaitDataForbidden
	}
	return nil
}

// The job/channel/source locks are shared by admission, pre-call and publication.
// Caller has locked tenant then membership. All hint identities are rechecked.
type waitDataExecutionLocked struct {
	job      models.Job
	channel  models.Channel
	conv     models.Conversation
	messages []models.Message
	caseRow  models.InterventionCase
	original models.AnalysisSnapshot
	observed *conversationSnapshot
}

func lockWaitDataExecutionSource(tx *gorm.DB, tenantID, caseID string) (*waitDataExecutionLocked, error) {
	var hint models.InterventionCase
	if err:=waitDataReadError(tx.Where("tenant_id = ? AND id = ?", tenantID, caseID).Take(&hint).Error,ErrWaitDataMissing);err!=nil{return nil,err}
	var convHint models.Conversation
	if err:=waitDataReadError(tx.Where("tenant_id = ? AND id = ?", tenantID, hint.ConversationID).Take(&convHint).Error,ErrWaitDataConflict);err!=nil{return nil,err}
	l := new(waitDataExecutionLocked)
	if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, hint.JobID).Take(&l.job).Error,ErrWaitDataConflict);err!=nil{return nil,err}
	if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, convHint.ChannelID).Take(&l.channel).Error,ErrWaitDataConflict);err!=nil{return nil,err}
	if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ? AND channel_id = ?", tenantID, hint.ConversationID, l.channel.ID).Take(&l.conv).Error,ErrWaitDataConflict);err!=nil{return nil,err}
	if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND conversation_id = ?", tenantID, l.conv.ID).Order("sent_at ASC, id ASC").Limit(waitDataMaxMessages+1).Find(&l.messages).Error != nil || len(l.messages) > waitDataMaxMessages {
		return nil, ErrWaitDataUnavailable
	}
	if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, caseID).Take(&l.caseRow).Error,ErrWaitDataConflict);err!=nil{return nil,err}
	if l.caseRow.JobID != l.job.ID || l.caseRow.ConversationID != l.conv.ID || l.caseRow.UseCase != "pancake_complaint_wait" || l.job.JobType != "intervention" || l.job.ScheduleType != "manual" || !l.job.IsActive || l.channel.ChannelType != "pancake" || !l.channel.IsDemoFixture || l.caseRow.SnapshotID == nil {
		return nil, ErrWaitDataConflict
	}
	var inputs []string
	if json.Unmarshal([]byte(l.job.InputChannelIDs), &inputs) != nil {
		return nil, ErrWaitDataUnavailable
	}
	included := false
	for _, id := range inputs {
		if id == l.channel.ID {
			included = true
		}
	}
	if !included {
		return nil, ErrWaitDataConflict
	}
	if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, *l.caseRow.SnapshotID).Take(&l.original).Error,ErrWaitDataConflict);err!=nil{return nil,err}
	if l.original.Digest != l.caseRow.SnapshotDigest || !VerifySnapshotProvenance(l.original, tenantID, l.conv.ID, l.caseRow.RunID) {
		return nil, ErrWaitDataConflict
	}
	n := 0
	for _, m := range l.messages {
		n += len(m.Content)
		if n > waitDataMaxContentBytes {
			return nil, ErrWaitDataUnavailable
		}
	}
	observed, err := buildConversationSnapshot(l.conv, l.messages, 0)
	if err != nil || len(observed.ManifestJSON) > waitDataMaxManifestBytes {
		return nil, ErrWaitDataUnavailable
	}
	l.observed = observed
	return l, nil
}

func waitDataLatestRevision(tx *gorm.DB, l *waitDataExecutionLocked, request models.InterventionRerunRequest) error {
	var revision, latest models.InterventionDataRevision
	if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND case_id = ? AND id = ?", request.TenantID, request.CaseID, request.DataRevisionID).Take(&revision).Error,ErrWaitDataConflict);err!=nil{return err}
	if err:=waitDataReadError(tx.Where("tenant_id = ? AND case_id = ?", request.TenantID, request.CaseID).Order("case_version DESC, id DESC").Take(&latest).Error,ErrWaitDataConflict);err!=nil{return err}
	if !VerifyInterventionDataRevision(revision) || !VerifyInterventionDataRevision(latest) || latest.ID != revision.ID || revision.Digest != request.DataRevisionDigest || revision.Coverage != coverageComplete || revision.Digest != l.observed.Digest || l.observed.Manifest.Coverage != coverageComplete || revision.OriginalSnapshotID != request.OriginalSnapshotID || revision.OriginalSnapshotDigest != request.OriginalSnapshotDigest {
		return ErrWaitDataConflict
	}
	return nil
}

func waitDataOwner(tenantID, jobID string, timeout time.Duration) (*JobRunOwner, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout+time.Minute)
	o := &JobRunOwner{coord: defaultJobRuns, key: jobKey{tenantID, jobID}, runID: pkg.NewUUID(), token: pkg.NewUUID(), ctx: ctx, cancel: cancel}
	defaultJobRuns.mu.Lock()
	defer defaultJobRuns.mu.Unlock()
	if _, ok := defaultJobRuns.owners[o.key]; ok {
		cancel()
		return nil, ErrWaitDataConflict
	}
	defaultJobRuns.owners[o.key] = o
	return o, nil
}

// ClaimWaitDataExecution is transaction-aware admission. Only a returned new
// claim may be driven by its local owner. An uncertain commit never starts a call.
func ClaimWaitDataExecution(ctx context.Context, tenantID, actorID, caseID, requestID string, in WaitDataExecuteInput, s *InterventionService) (*models.InterventionExecution, bool, error) {
	if in.ExpectedCaseVersion < 1 || !waitDataUUIDPattern.MatchString(tenantID) || !waitDataUUIDPattern.MatchString(caseID) || !waitDataUUIDPattern.MatchString(requestID) || !waitDataUUIDPattern.MatchString(in.GrantID) {
		return nil, false, ErrWaitDataInvalid
	}
	tenantID, actorID, caseID, requestID = strings.ToLower(tenantID), strings.ToLower(actorID), strings.ToLower(caseID), strings.ToLower(requestID)
	in.GrantID = strings.ToLower(in.GrantID)
	if s == nil || !s.configured() || s.PolicyVersion != "ccma.synthetic.intervention.v1" || s.Provider == "" || s.Model == "" || s.ConfigRef == "" {
		return nil, false, ErrInterventionUnavailable
	}
	hash := HashWaitDataExecute(tenantID, caseID, requestID, in)
	var prior models.InterventionExecution
	var newClaim *waitDataExecution
	// A preliminary current permission/request check permits a safe replay even
	// when another process owns the local job, or the case projection advanced.
	err := waitDataDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := waitDataLockTenant(tx, tenantID); err != nil {
			return err
		}
		if err := waitDataCheckActor(tx, tenantID, actorID); err != nil {
			return err
		}
		var request models.InterventionRerunRequest
		if err:=waitDataReadError(tx.Where("tenant_id = ? AND case_id = ? AND id = ?", tenantID, caseID, requestID).Take(&request).Error,ErrWaitDataMissing);err!=nil{return err}
		q := tx.Where("tenant_id = ? AND request_id = ?", tenantID, requestID).Take(&prior)
		if q.Error == nil {
			if prior.CaseID != caseID || prior.BodyHash != hash {
				return ErrWaitDataConflict
			}
			return nil
		}
		if !errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return ErrWaitDataUnavailable
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if prior.ID != "" {
		return &prior, true, nil
	}
	var hint models.InterventionCase
	if err:=waitDataReadError(waitDataDB(ctx).Where("tenant_id = ? AND id = ?", tenantID, caseID).Take(&hint).Error,ErrWaitDataMissing);err!=nil{return nil,false,err}
	owner, err := waitDataOwner(tenantID, hint.JobID, s.Timeout)
	if err != nil {
		// Another local admission may be between owner reservation and its DB
		// commit. Wait only to observe its unique claim; never reserve a run.
		for i := 0; i < 20; i++ {
			lookup:=waitDataDB(ctx).Where("tenant_id = ? AND request_id = ?", tenantID, requestID).Take(&prior)
			if lookup.Error == nil {
				if prior.BodyHash != hash {
					return nil, false, ErrWaitDataConflict
				}
				return &prior, true, nil
			}
			if !errors.Is(lookup.Error,gorm.ErrRecordNotFound){return nil,false,ErrWaitDataAdmissionUncertain}
			if ctx.Err() != nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		return nil, false, err
	}
	committed := false
	defer func() {
		if !committed {
			owner.release()
		}
	}()
	now := time.Now().UTC()
	err = waitDataDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := waitDataLockTenant(tx, tenantID); err != nil {
			return err
		}
		if err := waitDataCheckActor(tx, tenantID, actorID); err != nil {
			return err
		}
		var request models.InterventionRerunRequest
		if err:=waitDataReadError(tx.Where("tenant_id = ? AND case_id = ? AND id = ?", tenantID, caseID, requestID).Take(&request).Error,ErrWaitDataMissing);err!=nil{return err}
		q := tx.Where("tenant_id = ? AND request_id = ?", tenantID, requestID).Take(&prior)
		if q.Error == nil {
			if prior.BodyHash != hash {
				return ErrWaitDataConflict
			}
			return nil
		}
		if !errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return ErrWaitDataUnavailable
		}
		l, err := lockWaitDataExecutionSource(tx, tenantID, caseID)
		if err != nil {
			return err
		}
		if l.job.ID != owner.key.jobID {
			return ErrWaitDataConflict
		}
		if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND case_id = ? AND id = ?", tenantID, caseID, requestID).Take(&request).Error,ErrWaitDataMissing);err!=nil{return err}
		if l.caseRow.Version != in.ExpectedCaseVersion || l.caseRow.ActiveExecutionID != nil || l.caseRow.WorkState != "WAITING_ON_DATA" || l.caseRow.Eligibility != "WAIT_DATA" || l.caseRow.Execution != "NOT_RUN" || l.caseRow.Disposition == "PENDING_REVIEW" || l.caseRow.HumanAssessment != "" || l.caseRow.OutcomeText != "" || request.JobID != l.job.ID || request.ConversationID != l.conv.ID || request.OriginalSnapshotID != l.original.ID || request.OriginalSnapshotDigest != l.original.Digest || request.Status != "AWAITING_EXECUTION_AUTHORITY" || request.Execution != "NOT_RUN" || request.CaseVersion != l.caseRow.Version {
			return ErrWaitDataConflict
		}
		if err := waitDataLatestRevision(tx, l, request); err != nil {
			return err
		}
		var n int64
		if err:=tx.Model(&models.InterventionProposal{}).Where("tenant_id = ? AND case_id = ?", tenantID, caseID).Count(&n).Error;err!=nil{return ErrWaitDataUnavailable};if n!=0{return ErrWaitDataConflict}
		var grant models.InterventionExecutionGrant
		if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, in.GrantID).Take(&grant).Error,ErrWaitDataConflict);err!=nil{return err}
		if grant.CaseID != caseID || grant.JobID != l.job.ID || grant.RequestID != requestID || grant.DataRevisionID != request.DataRevisionID || grant.ActorID != actorID || grant.State != "ISSUED" || !grant.Deadline.After(now) || grant.Provider != s.Provider || grant.Model != s.Model || grant.PolicyVersion != s.PolicyVersion || grant.PromptVersion != s.PromptVersion || grant.ConfigRef != s.ConfigRef || grant.IssuerID == "" || grant.EvidenceRef == "" {
			return ErrWaitDataConflict
		}
		if err:=tx.Model(&models.JobRun{}).Where("tenant_id = ? AND job_id = ? AND status = ?", tenantID, l.job.ID, "running").Count(&n).Error;err!=nil{return ErrWaitDataUnavailable};if n!=0{return ErrWaitDataConflict}
		run := models.JobRun{ID: owner.runID, TenantID: tenantID, JobID: l.job.ID, StartedAt: now, Status: "running", Summary: "{}", CreatedAt: now}
		snap, err := l.observed.record(owner.runID)
		if err != nil {
			return ErrWaitDataUnavailable
		}
		row := models.InterventionExecution{ID: pkg.NewUUID(), TenantID: tenantID, CaseID: caseID, JobID: l.job.ID, ConversationID: l.conv.ID, RequestID: requestID, RevisionID: request.DataRevisionID, GrantID: grant.ID, RunID: run.ID, ActorID: actorID, BodyHash: hash, OriginalRunID: l.caseRow.RunID, OriginalSnapshotID: l.original.ID, OriginalSnapshotDigest: l.original.Digest, SnapshotID: snap.ID, SnapshotDigest: snap.Digest, ManifestDigest: snap.Digest, ExpectedCaseVersion: in.ExpectedCaseVersion, ClaimedCaseVersion: l.caseRow.Version, ConfigRef: grant.ConfigRef, State: "CLAIMED", CreatedAt: now, UpdatedAt: now}
		if tx.Create(&run).Error != nil || tx.Create(&snap).Error != nil || tx.Create(&row).Error != nil {
			return ErrWaitDataUnavailable
		}
		if r := tx.Model(&models.InterventionExecutionGrant{}).Where("id = ? AND state = ?", grant.ID, "ISSUED").Updates(map[string]any{"state": "CONSUMED", "consumed_execution_id": row.ID}); r.Error != nil || r.RowsAffected != 1 {
			return ErrWaitDataUnavailable
		}
		if r := tx.Model(&models.InterventionCase{}).Where("tenant_id = ? AND id = ? AND version = ? AND active_execution_id IS NULL", tenantID, caseID, l.caseRow.Version).Update("active_execution_id", row.ID); r.Error != nil || r.RowsAffected != 1 {
			return ErrWaitDataConflict
		}
		newClaim = &waitDataExecution{row: row, owner: owner, snapshot: l.observed}
		return nil
	})
	if err != nil {
		// Commit outcome can be uncertain. Resolve unique identity; never launch
		// transport based solely on an error or retry admission.
		if waitDataDB(ctx).Where("tenant_id = ? AND request_id = ?", tenantID, requestID).Take(&prior).Error == nil && prior.BodyHash == hash {
			return &prior, true, nil
		}
		if !errors.Is(err,ErrWaitDataMissing) && !errors.Is(err,ErrWaitDataConflict) && !errors.Is(err,ErrWaitDataForbidden) && !errors.Is(err,ErrWaitDataInvalid) {
			return nil, false, ErrWaitDataAdmissionUncertain
		}
		return nil, false, err
	}
	if newClaim == nil {
		return &prior, true, nil
	}
	committed = true
	go s.driveWaitDataExecution(newClaim)
	return &newClaim.row, false, nil
}

func (s *InterventionService) driveWaitDataExecution(e *waitDataExecution) {
	defer e.owner.release()
	row := e.row
	attemptID, err := s.markWaitDataCall(e)
	if err != nil {
		// A marker commit error can mean a debit already committed. Only a
		// confirmed untouched CLAIMED row and absent attempt permit no-call
		// terminalization. Every uncertainty leaves the running claim held.
		var current models.InterventionExecution
		var attempts int64
		read := waitDataDB(context.Background()).Where("tenant_id = ? AND id = ?", e.row.TenantID, e.row.ID).Take(&current).Error
		count := waitDataDB(context.Background()).Model(&models.InterventionAttempt{}).Where("tenant_id = ? AND execution_id = ?", e.row.TenantID, e.row.ID).Count(&attempts).Error
		if (errors.Is(err,ErrWaitDataDenied) || errors.Is(err,ErrWaitDataConflict) || errors.Is(err,ErrWaitDataForbidden)) && read == nil && count == nil && current.State == "CLAIMED" && attempts == 0 {
			s.terminalWaitData(e, "DENIED", "UNKNOWN", false)
		}
		return
	}
	prompt := s.Prompt + "\nReturn one JSON object with schema ccma.intervention.v1, conversation_id " + row.ConversationID + ", snapshot_digest " + row.SnapshotDigest + ", kind complaint or waiting, explanation, next_action, and evidence_refs with canonical message_id and exact quote."
	ctx, cancel := context.WithTimeout(e.owner.ctx, s.Timeout)
	defer cancel()
	response, callErr := s.Transport.AnalyzeChatOnce(ctx, prompt, e.snapshot.Transcript, s.MaxRequestBytes)
	usage := settledUsage(response)
	if callErr != nil && !ai.IsObservedChatOnceResponse(callErr) {
		s.terminalWaitData(e, "PROVIDER_UNKNOWN", "UNKNOWN", true)
		return
	}
	if response.Model == "" || response.Provider == "" || response.UsagePresence == nil {
		s.terminalWaitData(e, "PROVIDER_UNKNOWN", "UNKNOWN", true)
		return
	}
	if settleInterventionResponse(attemptID, response, usage) != nil {
		s.terminalWaitData(e, "SETTLEMENT_FAILED", "UNKNOWN", true)
		return
	}
	recorded:=waitDataDB(context.Background()).Model(&models.InterventionExecution{}).Where("id = ? AND state = ?", row.ID, "CALL_MAY_HAVE_STARTED").Updates(map[string]any{"state": "RESPONSE_RECORDED", "updated_at": time.Now().UTC()})
	if recorded.Error!=nil || recorded.RowsAffected != 1 {
		return
	}
	if response.Provider != s.Provider || response.Model != s.Model {
		s.terminalWaitData(e, "OUTPUT_INVALID", usage, true)
		return
	}
	if callErr != nil {
		s.terminalWaitData(e, "OUTPUT_INVALID", usage, true)
		return
	}
	var out interventionOutput
	if json.Unmarshal([]byte(response.Content), &out) != nil || validateInterventionOutput(e.snapshot, row.ConversationID, out) != nil {
		s.terminalWaitData(e, "OUTPUT_INVALID", usage, true)
		return
	}
	allowed, pubErr := e.owner.publish(func() error { return s.publishWaitData(e, response, out, usage) })
	if !allowed || errors.Is(pubErr, ErrWaitDataCancelled) {
		s.terminalWaitData(e, "CANCELLED", usage, true)
		return
	}
	if errors.Is(pubErr, ErrWaitDataUnavailable) {
		// Only this still-live owner retains the validated body. One DB-only
		// retry performs fresh locks/source checks; no provider request repeats.
		allowed, pubErr = e.owner.publish(func() error { return s.publishWaitData(e, response, out, usage) })
		if !allowed {
			s.terminalWaitData(e, "CANCELLED", usage, true)
			return
		}
	}
	if pubErr != nil {
		if errors.Is(pubErr, ErrWaitDataConflict) || errors.Is(pubErr, ErrWaitDataForbidden) {
			s.terminalWaitData(e, "STALE_SOURCE", usage, true)
		}
		// Storage/commit uncertainty remains RESPONSE_RECORDED with a blocking
		// run. The owner exits without inventing a source-stale receipt.
	}
}

func (s *InterventionService) markWaitDataCall(e *waitDataExecution) (string, error) {
	var attemptID string
	err := waitDataDB(e.owner.ctx).Transaction(func(tx *gorm.DB) error {
		r := e.row
		if err := waitDataLockTenant(tx, r.TenantID); err != nil {
			return err
		}
		if err := waitDataCheckActor(tx, r.TenantID, r.ActorID); err != nil {
			return err
		}
		l, err := lockWaitDataExecutionSource(tx, r.TenantID, r.CaseID)
		if err != nil {
			return err
		}
		var claim models.InterventionExecution
		if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", r.ID, r.TenantID).Take(&claim).Error,ErrWaitDataConflict);err!=nil{return err}
		if claim.State != "CLAIMED" || l.caseRow.ActiveExecutionID == nil || *l.caseRow.ActiveExecutionID != r.ID || l.caseRow.Version != r.ClaimedCaseVersion || l.observed.Digest != r.SnapshotDigest {
			return ErrWaitDataConflict
		}
		var request models.InterventionRerunRequest
		if err:=waitDataReadError(tx.Where("tenant_id = ? AND id = ?", r.TenantID, r.RequestID).Take(&request).Error,ErrWaitDataConflict);err!=nil{return err}
		if err:=waitDataLatestRevision(tx,l,request);err!=nil{return err}
		var grant models.InterventionExecutionGrant
		if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", r.TenantID, r.GrantID).Take(&grant).Error,ErrWaitDataConflict);err!=nil{return err}
		if grant.State != "CONSUMED" || grant.ConsumedExecutionID == nil || *grant.ConsumedExecutionID != r.ID || !grant.Deadline.After(time.Now().UTC()) || grant.ConfigRef != r.ConfigRef || grant.Provider != s.Provider || grant.Model != s.Model || grant.PolicyVersion != s.PolicyVersion || grant.PromptVersion != s.PromptVersion {
			return ErrWaitDataConflict
		}
		var budget models.InterventionBudget
		if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND job_id = ?", r.TenantID, r.JobID).Take(&budget).Error,ErrWaitDataDenied);err!=nil{return err}
		if budget.Cap <= 0 || budget.Consumed >= budget.Cap {return ErrWaitDataDenied}
		if u := tx.Model(&models.InterventionBudget{}).Where("id = ? AND consumed = ?", budget.ID, budget.Consumed).Update("consumed", budget.Consumed+1); u.Error != nil || u.RowsAffected != 1 {
			return ErrWaitDataUnavailable
		}
		attemptID = pkg.NewUUID()
		attempt := models.InterventionAttempt{ID: attemptID, ExecutionID: &r.ID, TenantID: r.TenantID, JobID: r.JobID, RunID: r.RunID, BudgetID: budget.ID, State: "CONSUMED", UsageState: "UNKNOWN", CreatedAt: time.Now().UTC()}
		if tx.Create(&attempt).Error != nil {
			return ErrWaitDataUnavailable
		}
		if u := tx.Model(&models.InterventionExecution{}).Where("id = ? AND state = ?", r.ID, "CLAIMED").Updates(map[string]any{"state": "CALL_MAY_HAVE_STARTED", "updated_at": time.Now().UTC()}); u.Error != nil || u.RowsAffected != 1 {
			return ErrWaitDataUnavailable
		}
		return nil
	})
	return attemptID, err
}

func (s *InterventionService) terminalWaitData(e *waitDataExecution, code, usage string, called bool) {
	// A failed terminal transaction intentionally leaves a blocking running row.
	if e.owner.beginTerminal() {code="CANCELLED"}
	err := waitDataDB(context.Background()).Transaction(func(tx *gorm.DB) error {
		r := e.row
		if err := waitDataLockTenant(tx, r.TenantID); err != nil {
			return err
		}
		// Terminal settlement remains permitted after membership revocation. The
		// original projection is untouched and the consumed attempt is preserved.
		var job models.Job
		if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", r.TenantID, r.JobID).Take(&job).Error,ErrWaitDataUnavailable);err!=nil{return err}
		var claim models.InterventionExecution
		if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", r.TenantID, r.ID).Take(&claim).Error,ErrWaitDataUnavailable);err!=nil{return err}
		if claim.State == "PUBLISHED" || claim.TerminalCode != "" {
			return nil
		}
		now := time.Now().UTC()
		if u:=tx.Model(&models.InterventionExecution{}).Where("id = ? AND state = ?", r.ID, claim.State).Updates(map[string]any{"state": code, "terminal_code": code, "updated_at": now});u.Error!=nil || u.RowsAffected != 1 {
			return ErrWaitDataUnavailable
		}
		if called && (code == "PROVIDER_UNKNOWN" || code == "SETTLEMENT_FAILED") {
			if u:=tx.Model(&models.InterventionAttempt{}).Where("execution_id = ? AND state = ?", r.ID, "CONSUMED").Update("state", "UNKNOWN");u.Error != nil || u.RowsAffected != 1 {
				return ErrWaitDataUnavailable
			}
		}
		if tx.Create(&models.InterventionReceipt{ID: pkg.NewUUID(), TenantID: r.TenantID, CaseID: r.CaseID, RunID: r.RunID, Code: code, UsageState: usage, CreatedAt: now}).Error != nil {
			return ErrWaitDataUnavailable
		}
		if u:=tx.Model(&models.JobRun{}).Where("tenant_id = ? AND id = ? AND status = ?", r.TenantID, r.RunID, "running").Updates(map[string]any{"status": "error", "finished_at": now});u.Error!=nil || u.RowsAffected != 1 {
			return ErrWaitDataUnavailable
		}
		return nil
	})
	if err != nil { e.owner.terminalFailed() }
}

func (s *InterventionService) publishWaitData(e *waitDataExecution, response ai.AIResponse, out interventionOutput, usage string) error {
	r := e.row
	return waitDataDB(e.owner.ctx).Transaction(func(tx *gorm.DB) error {
		if err := waitDataLockTenant(tx, r.TenantID); err != nil {
			return err
		}
		if err := waitDataCheckActor(tx, r.TenantID, r.ActorID); err != nil {
			return err
		}
		l, err := lockWaitDataExecutionSource(tx, r.TenantID, r.CaseID)
		if err != nil {
			return err
		}
		if l.caseRow.ActiveExecutionID == nil || *l.caseRow.ActiveExecutionID != r.ID || l.caseRow.Version != r.ClaimedCaseVersion || l.observed.Digest != r.SnapshotDigest || l.caseRow.RunID != r.OriginalRunID || l.caseRow.SnapshotDigest != r.OriginalSnapshotDigest {
			return ErrWaitDataConflict
		}
		var request models.InterventionRerunRequest
		if err:=waitDataReadError(tx.Where("tenant_id = ? AND id = ?", r.TenantID, r.RequestID).Take(&request).Error,ErrWaitDataConflict);err!=nil{return err}
		if err:=waitDataLatestRevision(tx,l,request);err!=nil{return err}
		var claim models.InterventionExecution
		if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", r.TenantID, r.ID).Take(&claim).Error,ErrWaitDataConflict);err!=nil{return err}
		if claim.State!="RESPONSE_RECORDED" {return ErrWaitDataConflict}
		var n int64
		if err:=tx.Model(&models.InterventionProposal{}).Where("tenant_id = ? AND case_id = ?", r.TenantID, r.CaseID).Count(&n).Error;err!=nil{return ErrWaitDataUnavailable};if n!=0{return ErrWaitDataConflict}
		var run models.JobRun
		if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ? AND status = ?", r.TenantID, r.RunID, "running").Take(&run).Error,ErrWaitDataConflict);err!=nil{return err}
		if e.owner.ctx.Err() != nil {
			return ErrWaitDataCancelled
		}
		now := time.Now().UTC()
		ev, _ := json.Marshal(out.Evidence)
		proposal := models.InterventionProposal{ID: pkg.NewUUID(), TenantID: r.TenantID, CaseID: r.CaseID, Revision: l.caseRow.Version + 1, RunID: r.RunID, SnapshotID: r.SnapshotID, SnapshotDigest: r.SnapshotDigest, PromptVersion: s.PromptVersion, PolicyVersion: s.PolicyVersion, SchemaVersion: "ccma.intervention.v1", Kind: out.Kind, Explanation: out.Explanation, NextAction: out.NextAction, Evidence: string(ev), Provider: response.Provider, Model: response.Model, UsageState: usage, CreatedAt: now}
		if tx.Create(&proposal).Error != nil {
			return ErrWaitDataUnavailable
		}
		beforeProjection := m3Projection(l.caseRow)
		beforeProjection["run_id"] = l.caseRow.RunID
		beforeProjection["snapshot_id"] = l.original.ID
		beforeProjection["snapshot_digest"] = l.original.Digest
		beforeProjection["execution_id"] = r.ID
		beforeProjection["request_id"] = r.RequestID
		beforeProjection["revision_id"] = r.RevisionID
		before, _ := json.Marshal(beforeProjection)
		if u := tx.Model(&models.InterventionCase{}).Where("tenant_id = ? AND id = ? AND version = ? AND active_execution_id = ?", r.TenantID, r.CaseID, l.caseRow.Version, r.ID).Updates(map[string]any{"version": proposal.Revision, "run_id": r.RunID, "snapshot_id": r.SnapshotID, "snapshot_digest": r.SnapshotDigest, "eligibility": "ELIGIBLE", "execution": "LLM_SUCCEEDED", "disposition": "PENDING_REVIEW", "work_state": "OPEN", "reason_code": "", "next_action": "", "active_proposal_id": proposal.ID, "policy_version": s.PolicyVersion, "prompt_version": s.PromptVersion, "updated_at": now}); u.Error != nil || u.RowsAffected != 1 {
			return ErrWaitDataUnavailable
		}
		afterRow := l.caseRow
		afterRow.Version = proposal.Revision
		afterRow.RunID = r.RunID
		afterRow.SnapshotID = &r.SnapshotID
		afterRow.SnapshotDigest = r.SnapshotDigest
		afterRow.Disposition = "PENDING_REVIEW"
		afterRow.WorkState = "OPEN"
		afterProjection := m3Projection(afterRow)
		afterProjection["run_id"] = r.RunID
		afterProjection["snapshot_id"] = r.SnapshotID
		afterProjection["snapshot_digest"] = r.SnapshotDigest
		afterProjection["active_proposal_id"] = proposal.ID
		afterProjection["execution_id"] = r.ID
		afterProjection["request_id"] = r.RequestID
		afterProjection["revision_id"] = r.RevisionID
		after, _ := json.Marshal(afterProjection)
		event := models.InterventionHumanEvent{ID: pkg.NewUUID(), TenantID: r.TenantID, CaseID: r.CaseID, CaseVersion: proposal.Revision, ActorID: r.ActorID, Action: "EXECUTE_RERUN", Reason: "approved execution", ProposalID: &proposal.ID, DataRevisionID: &r.RevisionID, RerunRequestID: &r.RequestID, SnapshotID: r.SnapshotID, SnapshotDigest: r.SnapshotDigest, SourceStatus: SourceIntegrityBoundCurrentnessUnverified, Before: string(before), After: string(after), CreatedAt: now}
		if tx.Create(&event).Error != nil {
			return ErrWaitDataUnavailable
		}
		if tx.Create(&models.InterventionReceipt{ID: pkg.NewUUID(), TenantID: r.TenantID, CaseID: r.CaseID, RunID: r.RunID, Code: "PROPOSAL_VALID", UsageState: usage, CreatedAt: now}).Error != nil {
			return ErrWaitDataUnavailable
		}
		if u:=tx.Model(&models.InterventionExecution{}).Where("id = ? AND state = ?", r.ID, "RESPONSE_RECORDED").Updates(map[string]any{"state": "PUBLISHED", "terminal_code": "PROPOSAL_VALID", "updated_at": now});u.Error!=nil || u.RowsAffected != 1 {
			return ErrWaitDataUnavailable
		}
		if u:=tx.Model(&models.JobRun{}).Where("tenant_id = ? AND id = ? AND status = ?", r.TenantID, r.RunID, "running").Updates(map[string]any{"status": "success", "finished_at": now});u.Error!=nil || u.RowsAffected != 1 {
			return ErrWaitDataUnavailable
		}
		return nil
	})
}

// MarkWaitDataRecoveryRequired is an explicit operator observation of a
// committed call marker after the original process owner is gone. It never
// replays transport or reconstructs a proposal. A recorded response without
// the original validated body remains a blocking manual hold with no receipt.
func MarkWaitDataRecoveryRequired(ctx context.Context, tenantID, executionID string) error {
	var hint models.InterventionExecution
	if err:=waitDataReadError(waitDataDB(ctx).Where("tenant_id = ? AND id = ?", tenantID, executionID).Take(&hint).Error,ErrWaitDataMissing);err!=nil{return err}
	if JobRunActive(tenantID, hint.JobID) {
		return ErrWaitDataConflict
	}
	return waitDataDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := waitDataLockTenant(tx, tenantID); err != nil {
			return err
		}
		var job models.Job
		if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, hint.JobID).Take(&job).Error,ErrWaitDataConflict);err!=nil{return err}
		if JobRunActive(tenantID, hint.JobID) {
			return ErrWaitDataConflict
		}
		var claim models.InterventionExecution
		if err:=waitDataReadError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, executionID).Take(&claim).Error,ErrWaitDataConflict);err!=nil{return err}
		if claim.State != "CALL_MAY_HAVE_STARTED" && claim.State != "RESPONSE_RECORDED" {
			return ErrWaitDataConflict
		}
		now := time.Now().UTC()
		if u := tx.Model(&models.InterventionExecution{}).Where("id = ? AND state = ?", claim.ID, claim.State).Updates(map[string]any{"state": "RECOVERY_REQUIRED", "terminal_code": "RECOVERY_REQUIRED", "updated_at": now}); u.Error != nil || u.RowsAffected != 1 {
			return ErrWaitDataUnavailable
		}
		if claim.State == "RESPONSE_RECORDED" {
			return nil
		}
		if u := tx.Model(&models.InterventionAttempt{}).Where("tenant_id = ? AND execution_id = ? AND state = ?", tenantID, claim.ID, "CONSUMED").Update("state", "UNKNOWN"); u.Error != nil || u.RowsAffected != 1 {
			return ErrWaitDataUnavailable
		}
		if tx.Create(&models.InterventionReceipt{ID: pkg.NewUUID(), TenantID: tenantID, CaseID: claim.CaseID, RunID: claim.RunID, Code: "RECOVERY_REQUIRED", UsageState: "UNKNOWN", CreatedAt: now}).Error != nil {
			return ErrWaitDataUnavailable
		}
		if u := tx.Model(&models.JobRun{}).Where("tenant_id = ? AND id = ? AND status = ?", tenantID, claim.RunID, "running").Updates(map[string]any{"status": "error", "finished_at": now}); u.Error != nil || u.RowsAffected != 1 {
			return ErrWaitDataUnavailable
		}
		return nil
	})
}
