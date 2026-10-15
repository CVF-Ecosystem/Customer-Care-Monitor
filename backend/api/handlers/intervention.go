package handlers

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/gin-gonic/gin"
)

// InterventionRunner is deliberately nil in ordinary startup. A future
// server-owned source-reviewed configuration must install it explicitly.
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
	c.JSON(http.StatusOK, gin.H{"case": row, "proposals": proposals, "receipts": receipts, "human_events": events, "source_status": m3SourceStatus(row)})
}
