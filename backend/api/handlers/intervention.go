package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// InterventionRunner defaults to nil. Ordinary startup installs it only when
// the server-owned synthetic configuration has been validated.
var InterventionRunner *engine.InterventionService

type interventionRunRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
}

func RunIntervention(c *gin.Context) {
	var req interventionRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if InterventionRunner == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "intervention_unavailable"})
		return
	}
	result, err := InterventionRunner.RunManual(c.Request.Context(), middleware.GetTenantID(c), middleware.GetUserID(c), c.Param("jobId"), req.ConversationID)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, result)
	case errors.Is(err, engine.ErrInterventionMissing):
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	case errors.Is(err, engine.ErrInterventionBusy):
		c.JSON(http.StatusConflict, gin.H{"error": "intervention_busy"})
	case errors.Is(err, engine.ErrInterventionManualOnly):
		c.JSON(http.StatusForbidden, gin.H{"error": "intervention_manual_only"})
	case errors.Is(err, engine.ErrInterventionDenied):
		c.JSON(http.StatusForbidden, gin.H{"eligibility": "DENY", "execution": "NOT_RUN", "reason_code": "SOURCE_DENIED"})
	default:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "intervention_unavailable"})
	}
}

func ListInterventionCases(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		n, e := strconv.Atoi(raw)
		if e != nil || n < 1 || n > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_limit"})
			return
		}
		limit = n
	}
	state := c.Query("state")
	if state != "" && !m3AllowedState(state) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_state"})
		return
	}
	q := db.DB.Model(&models.InterventionCase{}).Where("tenant_id = ?", tenantID)
	if state != "" {
		q = q.Where("work_state = ?", state)
	} else {
		q = q.Where("work_state IN ?", []string{"OPEN", "ASSIGNED", "WAITING_ON_DATA"})
	}
	var total int64
	if q.Count(&total).Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_failed"})
		return
	}
	if cursor := c.Query("cursor"); cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		parts := strings.SplitN(string(decoded), "|", 2)
		if err != nil || len(parts) != 2 || parts[1] == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_cursor"})
			return
		}
		at, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_cursor"})
			return
		}
		q = q.Where("created_at > ? OR (created_at = ? AND id > ?)", at, at, parts[1])
	}
	var rows []models.InterventionCase
	if q.Order("created_at ASC, id ASC").Limit(limit+1).Find(&rows).Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_failed"})
		return
	}
	next := ""
	hasMore := len(rows) > limit
	if hasMore {
		last := rows[limit-1]
		next = base64.RawURLEncoding.EncodeToString([]byte(last.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + last.ID))
		rows = rows[:limit]
	}
	c.JSON(http.StatusOK, gin.H{"items": rows, "next_cursor": next, "has_more": hasMore, "total": total, "order": "created_at_asc_id_asc"})
}

func GetInterventionCase(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	var row models.InterventionCase
	if db.DB.Where("id = ? AND tenant_id = ?", c.Param("caseId"), tenantID).First(&row).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	var proposals []models.InterventionProposal
	if db.DB.Where("case_id = ? AND tenant_id = ?", row.ID, tenantID).Order("revision ASC").Find(&proposals).Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "detail_failed"})
		return
	}
	var receipts []models.InterventionReceipt
	if db.DB.Where("case_id = ? AND tenant_id = ?", row.ID, tenantID).Order("created_at ASC").Find(&receipts).Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "detail_failed"})
		return
	}
	var events []models.InterventionHumanEvent
	if db.DB.Where("case_id = ? AND tenant_id = ?", row.ID, tenantID).Order("case_version ASC").Find(&events).Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "detail_failed"})
		return
	}
	ids := map[string]struct{}{}
	addID := func(id string) {
		if _, err := uuid.Parse(id); err == nil {
			ids[id] = struct{}{}
		}
	}
	for _, id := range []*string{row.AssigneeID, row.DataOwnerID} {
		if id != nil {
			addID(*id)
		}
	}
	for _, event := range events {
		addID(event.ActorID)
		for _, projection := range []string{event.Before, event.After} {
			var fields map[string]json.RawMessage
			if json.Unmarshal([]byte(projection), &fields) != nil {
				continue
			}
			for _, key := range []string{"assignee_id", "data_owner_id"} {
				var id string
				if json.Unmarshal(fields[key], &id) == nil {
					addID(id)
				}
			}
		}
	}
	participants := []struct {
		UserID string `json:"user_id"`
		Name   string `json:"name"`
	}{}
	if len(ids) != 0 {
		selected := make([]string, 0, len(ids))
		for id := range ids {
			selected = append(selected, id)
		}
		sort.Strings(selected)
		if db.DB.Table("user_tenants AS ut").Select("ut.user_id, users.name").Joins("JOIN users ON users.id = ut.user_id").Where("ut.tenant_id = ? AND ut.user_id IN ?", tenantID, selected).Order("ut.user_id ASC").Scan(&participants).Error != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "participants_unavailable"})
			return
		}
	}
	sourceStatus := m3SourceStatus(row)
	currentDigest := ""
	reviewedCurrent := false
	if sourceStatus == "CHANGED" {
		var conv models.Conversation
		var messages []models.Message
		if db.DB.Where("id = ? AND tenant_id = ?", row.ConversationID, tenantID).Take(&conv).Error == nil && db.DB.Where("conversation_id = ? AND tenant_id = ?", row.ConversationID, tenantID).Order("sent_at ASC, id ASC").Find(&messages).Error == nil {
			if digest, coverage, err := engine.M3CurrentSourceObservation(conv, messages); err == nil && coverage == "complete" {
				currentDigest = digest
				if row.ReviewedDataRevisionID != nil {
					var revision models.InterventionDataRevision
					if db.DB.Where("id = ? AND tenant_id = ? AND case_id = ?", *row.ReviewedDataRevisionID, tenantID, row.ID).Take(&revision).Error == nil && engine.VerifyInterventionDataRevision(revision) && revision.JobID == row.JobID && revision.ConversationID == row.ConversationID && revision.Coverage == "complete" && revision.Digest == digest && row.SnapshotID != nil && revision.OriginalSnapshotID == *row.SnapshotID && revision.OriginalSnapshotDigest == row.SnapshotDigest {
						reviewedCurrent = true
					}
				}
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"case": row, "active_proposal_id": row.ActiveProposalID, "proposals": proposals, "receipts": receipts, "human_events": events, "participants": participants, "source_status": sourceStatus, "current_source_digest": currentDigest, "reviewed_source_current": reviewedCurrent})
}
