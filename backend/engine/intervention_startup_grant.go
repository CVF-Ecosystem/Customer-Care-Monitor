package engine

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInterventionGrantInvalid = errors.New("grant_invalid")
var ErrInterventionGrantConflict = errors.New("grant_conflict")
var ErrInterventionGrantForbidden = errors.New("grant_forbidden")
var ErrInterventionGrantUnavailable = errors.New("grant_unavailable")

// IssueSyntheticInterventionGrant is the server-side authority behind the
// operator CLI. It does not call a provider or consume budget. The subsequent
// authenticated execute request independently rechecks every bound field.
func IssueSyntheticInterventionGrant(ctx context.Context, tenantID, issuerID, actorID, caseID, requestID, evidenceRef string, deadline time.Time, service *InterventionService) (*models.InterventionExecutionGrant, bool, error) {
	if (tenantID != models.SingleWorkspaceID && !waitDataUUIDPattern.MatchString(tenantID)) || !waitDataUUIDPattern.MatchString(issuerID) || !waitDataUUIDPattern.MatchString(actorID) || !waitDataUUIDPattern.MatchString(caseID) || !waitDataUUIDPattern.MatchString(requestID) || !boundedM3(evidenceRef, 255) || deadline.Before(time.Now().UTC().Add(time.Minute)) || deadline.After(time.Now().UTC().Add(24*time.Hour)) || service == nil || !service.SyntheticConfigured() {
		return nil, false, ErrInterventionGrantInvalid
	}
	tenantID, issuerID, actorID, caseID, requestID = strings.ToLower(tenantID), strings.ToLower(issuerID), strings.ToLower(actorID), strings.ToLower(caseID), strings.ToLower(requestID)
	var result models.InterventionExecutionGrant
	replayed := false
	err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := waitDataLockTenant(tx, tenantID); err != nil {
			return ErrInterventionGrantUnavailable
		}
		var issuer, actor models.UserTenant
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND user_id = ?", tenantID, issuerID).Take(&issuer).Error != nil || (issuer.Role != "owner" && issuer.Role != "admin") {
			return ErrInterventionGrantForbidden
		}
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND user_id = ?", tenantID, actorID).Take(&actor).Error != nil || !waitDataJobPermitted(actor) {
			return ErrInterventionGrantForbidden
		}
		var existing models.InterventionExecutionGrant
		q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND request_id = ?", tenantID, requestID).Take(&existing)
		if q.Error == nil {
			if existing.CaseID != caseID || existing.ActorID != actorID || existing.IssuerID != issuerID || existing.EvidenceRef != strings.TrimSpace(evidenceRef) || !existing.Deadline.Equal(deadline.UTC()) || existing.Provider != service.Provider || existing.Model != service.Model || existing.PolicyVersion != service.PolicyVersion || existing.PromptVersion != service.PromptVersion || existing.ConfigRef != service.ConfigRef {
				return ErrInterventionGrantConflict
			}
			result, replayed = existing, true
			return nil
		}
		if !errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return ErrInterventionGrantUnavailable
		}
		var request models.InterventionRerunRequest
		if tx.Where("tenant_id = ? AND case_id = ? AND id = ?", tenantID, caseID, requestID).Take(&request).Error != nil {
			return ErrInterventionGrantConflict
		}
		l, err := lockWaitDataExecutionSource(tx, tenantID, caseID)
		if err != nil {
			return ErrInterventionGrantConflict
		}
		if l.caseRow.Version != request.CaseVersion || l.caseRow.ActiveExecutionID != nil || l.caseRow.WorkState != "WAITING_ON_DATA" || l.caseRow.Eligibility != "WAIT_DATA" || l.caseRow.Execution != "NOT_RUN" || l.caseRow.HumanAssessment != "" || l.caseRow.OutcomeText != "" || request.Status != "AWAITING_EXECUTION_AUTHORITY" || request.Execution != "NOT_RUN" || request.JobID != l.job.ID || request.ConversationID != l.conv.ID || request.OriginalSnapshotID != l.original.ID || request.OriginalSnapshotDigest != l.original.Digest || request.ActorID != actorID {
			return ErrInterventionGrantConflict
		}
		if err := waitDataLatestRevision(tx, l, request); err != nil {
			return ErrInterventionGrantConflict
		}
		var budget models.InterventionBudget
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND job_id = ?", tenantID, l.job.ID).Take(&budget).Error != nil || budget.Cap <= 0 || budget.Consumed >= budget.Cap {
			return ErrInterventionGrantConflict
		}
		now := time.Now().UTC()
		result = models.InterventionExecutionGrant{ID: pkg.NewUUID(), TenantID: tenantID, JobID: l.job.ID, CaseID: caseID, RequestID: requestID, DataRevisionID: request.DataRevisionID, ActorID: actorID, Provider: service.Provider, Model: service.Model, PolicyVersion: service.PolicyVersion, PromptVersion: service.PromptVersion, ConfigRef: service.ConfigRef, IssuerID: issuerID, EvidenceRef: strings.TrimSpace(evidenceRef), Deadline: deadline.UTC(), State: "ISSUED", CreatedAt: now}
		if tx.Create(&result).Error != nil {
			return ErrInterventionGrantConflict
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return &result, replayed, nil
}
