package engine

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrM3Missing = errors.New("not_found")
var ErrM3Forbidden = errors.New("permission_denied")
var ErrM3Invalid = errors.New("invalid_action")
var ErrM3Conflict = errors.New("case_conflict")
var ErrM3Unavailable = errors.New("action_unavailable")

// M3Action binds a human decision to the exact case and source the reviewer saw.
// Actor and tenant always come from authenticated server context.
type M3Action struct {
	Action                 string     `json:"action"`
	ExpectedVersion        int        `json:"expected_version"`
	ExpectedSnapshotID     string     `json:"expected_snapshot_id"`
	ExpectedSnapshotDigest string     `json:"expected_snapshot_digest"`
	ExpectedProposalID     string     `json:"expected_proposal_id"`
	ExpectedRevision       int        `json:"expected_revision"`
	Reason                 string     `json:"reason"`
	Explanation            string     `json:"explanation"`
	NextAction             string     `json:"next_action"`
	AssigneeID             string     `json:"assignee_id"`
	DueAt                  *time.Time `json:"due_at"`
	Outcome                string     `json:"outcome"`
}

func boundedM3(s string, max int) bool {
	return strings.TrimSpace(s) != "" && len([]rune(s)) <= max
}

func m3Permitted(ut models.UserTenant) bool {
	switch ut.Role {
	case "owner", "admin":
		return true
	case "member":
		var perms map[string]string
		if json.Unmarshal([]byte(ut.Permissions), &perms) != nil {
			return false
		}
		p := perms["intervention_cases"]
		return strings.Contains(p, "r") && strings.Contains(p, "w")
	default:
		return false
	}
}

func m3Projection(c models.InterventionCase) map[string]any {
	return map[string]any{
		"version": c.Version, "disposition": c.Disposition,
		"work_state": c.WorkState, "human_assessment": c.HumanAssessment,
		"human_explanation": c.HumanExplanation, "human_next_action": c.HumanNextAction,
		"assignee_id": c.AssigneeID, "data_owner_id": c.DataOwnerID,
		"due_at": c.DueAt, "outcome_text": c.OutcomeText,
	}
}

func m3HasHumanOwnership(tx *gorm.DB, prior models.InterventionCase) (bool, error) {
	var n int64
	if err := tx.Model(&models.InterventionHumanEvent{}).Where("tenant_id = ? AND case_id = ?", prior.TenantID, prior.ID).Count(&n).Error; err != nil {
		return true, err
	}
	return prior.Disposition == "PENDING_REVIEW" || prior.HumanAssessment != "" || prior.DataOwnerID != nil || n > 0, nil
}

// M3ApplyAction serializes supported sync writes through the channel lock,
// then checks the current local source and case version before one atomic
// projection/event commit. Out-of-band SQL and upstream history are outside
// this local-currentness contract.
func M3ApplyAction(ctx context.Context, tenantID, actorID, caseID string, a M3Action) (*models.InterventionCase, error) {
	if a.ExpectedVersion < 1 || !boundedM3(a.Reason, 500) || a.ExpectedSnapshotID == "" || a.ExpectedSnapshotDigest == "" {
		return nil, ErrM3Invalid
	}
	// Locate the lock parents outside the transaction. These reads are hints
	// only and cannot establish an old repeatable-read view inside the writer.
	var located models.InterventionCase
	if db.DB.WithContext(ctx).Where("id = ? AND tenant_id = ?", caseID, tenantID).Take(&located).Error != nil {
		return nil, ErrM3Missing
	}
	var locatedConv models.Conversation
	if db.DB.WithContext(ctx).Where("id = ? AND tenant_id = ?", located.ConversationID, tenantID).Take(&locatedConv).Error != nil {
		return nil, ErrM3Conflict
	}
	var result models.InterventionCase
	err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var actor models.UserTenant
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND user_id = ?", tenantID, actorID).Take(&actor).Error != nil || !m3Permitted(actor) {
			return ErrM3Forbidden
		}
		// These first two reads locate parents only. Recheck every identity after
		// taking the locks; neither read authorizes a write.
		var channel models.Channel
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", locatedConv.ChannelID, tenantID).Take(&channel).Error != nil {
			return ErrM3Conflict
		}
		var conv models.Conversation
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ? AND channel_id = ?", located.ConversationID, tenantID, channel.ID).Take(&conv).Error != nil {
			return ErrM3Conflict
		}
		var messages []models.Message
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("conversation_id = ? AND tenant_id = ?", conv.ID, tenantID).Order("sent_at ASC, id ASC").Find(&messages).Error != nil {
			return ErrM3Unavailable
		}
		var current models.InterventionCase
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", caseID, tenantID).Take(&current).Error != nil {
			return ErrM3Missing
		}
		if current.ConversationID != conv.ID || current.Version != a.ExpectedVersion || current.SnapshotID == nil || *current.SnapshotID != a.ExpectedSnapshotID || current.SnapshotDigest != a.ExpectedSnapshotDigest {
			return ErrM3Conflict
		}
		var snap models.AnalysisSnapshot
		snapErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", a.ExpectedSnapshotID, tenantID).Take(&snap).Error
		if snapErr != nil && !errors.Is(snapErr, gorm.ErrRecordNotFound) {
			return ErrM3Unavailable
		}
		provenanceOK := snapErr == nil && snap.Digest == current.SnapshotDigest && VerifySnapshotProvenance(snap, tenantID, conv.ID, current.RunID)
		// Data ownership may proceed on incomplete source. It never certifies
		// source currentness; assessment/assignment/outcome require complete.
		needsCurrent := a.Action != "ASSIGN_DATA" && a.Action != "ESCALATE_DATA"
		sourceStatus := "UNAVAILABLE"
		if provenanceOK {
			if snap.Coverage != coverageComplete {
				sourceStatus = "INCOMPLETE"
			} else {
				sourceStatus = CompareSnapshotToCurrentMessages(snap.Manifest, messages)
			}
		}
		if needsCurrent && sourceStatus != SourceIntegrityBoundCurrentnessUnverified {
			return ErrM3Conflict
		}
		var proposal models.InterventionProposal
		hasProposal := a.Action != "ASSIGN_DATA" && a.Action != "ESCALATE_DATA"
		if hasProposal {
			if a.ExpectedProposalID == "" || a.ExpectedRevision < 1 || tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND case_id = ? AND tenant_id = ?", a.ExpectedProposalID, caseID, tenantID).Take(&proposal).Error != nil || proposal.Revision != a.ExpectedRevision || proposal.RunID != current.RunID || proposal.SnapshotID != snap.ID || proposal.SnapshotDigest != snap.Digest {
				return ErrM3Conflict
			}
		} else if a.ExpectedProposalID != "" || a.ExpectedRevision != 0 {
			return ErrM3Invalid
		}
		before, err := json.Marshal(m3Projection(current))
		if err != nil {
			return ErrM3Unavailable
		}
		change := map[string]any{}
		now := time.Now().UTC()
		validDue := a.DueAt != nil && a.DueAt.After(now)
		if a.DueAt != nil {
			dueUTC := a.DueAt.UTC()
			a.DueAt = &dueUTC
		}
		var assigneeID string
		validateAssignee := func(id string) error {
			if id == "" {
				return ErrM3Invalid
			}
			var member models.UserTenant
			if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND user_id = ?", tenantID, id).Take(&member).Error != nil {
				return ErrM3Invalid
			}
			if member.Role != "owner" && member.Role != "admin" && member.Role != "member" {
				return ErrM3Invalid
			}
			var user models.User
			if tx.Where("id = ?", id).Take(&user).Error != nil {
				return ErrM3Invalid
			}
			assigneeID = id
			return nil
		}
		switch a.Action {
		case "CONFIRM", "CORRECT", "REJECT":
			if current.Disposition != "PENDING_REVIEW" || current.WorkState != "OPEN" || current.HumanAssessment != "" {
				return ErrM3Conflict
			}
			if a.Action == "CORRECT" {
				if !boundedM3(a.Explanation, 2000) || !boundedM3(a.NextAction, 2000) {
					return ErrM3Invalid
				}
				current.HumanExplanation, current.HumanNextAction = strings.TrimSpace(a.Explanation), strings.TrimSpace(a.NextAction)
				change["human_explanation"], change["human_next_action"] = current.HumanExplanation, current.HumanNextAction
			}
			current.HumanAssessment, current.Disposition = a.Action, "HUMAN_"+a.Action
			change["human_assessment"], change["disposition"] = current.HumanAssessment, current.Disposition
			if a.Action == "REJECT" {
				current.WorkState = "REJECTED"
				change["work_state"] = current.WorkState
			}
		case "ASSIGN":
			if (current.HumanAssessment != "CONFIRM" && current.HumanAssessment != "CORRECT") || (current.WorkState != "OPEN" && current.WorkState != "ASSIGNED") {
				return ErrM3Conflict
			}
			if !validDue {
				return ErrM3Invalid
			}
			if err := validateAssignee(a.AssigneeID); err != nil {
				return err
			}
			current.AssigneeID, current.DueAt, current.WorkState = &assigneeID, a.DueAt, "ASSIGNED"
			change["assignee_id"], change["due_at"], change["work_state"] = current.AssigneeID, current.DueAt, current.WorkState
		case "RESOLVE":
			if current.WorkState != "ASSIGNED" || current.AssigneeID == nil || (current.HumanAssessment != "CONFIRM" && current.HumanAssessment != "CORRECT") {
				return ErrM3Conflict
			}
			if err := validateAssignee(*current.AssigneeID); err != nil {
				return ErrM3Conflict
			}
			if !boundedM3(a.Outcome, 2000) {
				return ErrM3Invalid
			}
			current.OutcomeText, current.WorkState = strings.TrimSpace(a.Outcome), "RESOLVED"
			change["outcome_text"], change["work_state"] = current.OutcomeText, current.WorkState
		case "REOPEN":
			if current.WorkState != "RESOLVED" || current.AssigneeID == nil {
				return ErrM3Conflict
			}
			if err := validateAssignee(*current.AssigneeID); err != nil {
				return ErrM3Conflict
			}
			if !validDue {
				return ErrM3Invalid
			}
			current.WorkState, current.DueAt, current.OutcomeText = "ASSIGNED", a.DueAt, ""
			change["work_state"], change["due_at"], change["outcome_text"] = current.WorkState, current.DueAt, current.OutcomeText
		case "ASSIGN_DATA":
			if current.WorkState != "WAITING_ON_DATA" || !validDue || !boundedM3(a.NextAction, 500) {
				return ErrM3Invalid
			}
			if err := validateAssignee(a.AssigneeID); err != nil {
				return err
			}
			current.DataOwnerID, current.DueAt, current.HumanNextAction = &assigneeID, a.DueAt, strings.TrimSpace(a.NextAction)
			change["data_owner_id"], change["due_at"], change["human_next_action"] = current.DataOwnerID, current.DueAt, current.HumanNextAction
		case "ESCALATE_DATA":
			if current.WorkState != "WAITING_ON_DATA" || !boundedM3(a.NextAction, 500) {
				return ErrM3Invalid
			}
			current.HumanNextAction = strings.TrimSpace(a.NextAction)
			change["human_next_action"] = current.HumanNextAction
		default:
			return ErrM3Invalid
		}
		current.Version++
		current.UpdatedAt = now
		change["version"], change["updated_at"] = current.Version, now
		updated := tx.Model(&models.InterventionCase{}).Where("id = ? AND tenant_id = ? AND version = ?", caseID, tenantID, a.ExpectedVersion).Updates(change)
		if updated.Error != nil || updated.RowsAffected != 1 {
			return ErrM3Conflict
		}
		after, err := json.Marshal(m3Projection(current))
		if err != nil {
			return ErrM3Unavailable
		}
		var proposalID *string
		if hasProposal {
			proposalID = &proposal.ID
		}
		if sourceStatus == SourceIntegrityBoundCurrentnessUnverified {
			sourceStatus = "LOCAL_MATCH_CURRENTNESS_UNVERIFIED"
		}
		if sourceStatus == SourceIntegrityChangedSinceAnalysis {
			sourceStatus = "CHANGED"
		}
		event := models.InterventionHumanEvent{ID: pkg.NewUUID(), TenantID: tenantID, CaseID: caseID, CaseVersion: current.Version, ActorID: actorID, Action: a.Action, Reason: strings.TrimSpace(a.Reason), ProposalID: proposalID, SnapshotID: a.ExpectedSnapshotID, SnapshotDigest: a.ExpectedSnapshotDigest, SourceStatus: sourceStatus, Before: string(before), After: string(after), CreatedAt: now}
		if tx.Create(&event).Error != nil {
			return ErrM3Unavailable
		}
		result = current
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrM3Missing) || errors.Is(err, ErrM3Forbidden) || errors.Is(err, ErrM3Invalid) || errors.Is(err, ErrM3Conflict) {
			return nil, err
		}
		return nil, ErrM3Unavailable
	}
	return &result, nil
}
