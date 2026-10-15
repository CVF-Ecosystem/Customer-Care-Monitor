package handlers

import (
	"errors"
	"net/http"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/gin-gonic/gin"
)

func ApplyInterventionAction(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	var req engine.M3Action
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_action"})
		return
	}
	row, err := engine.M3ApplyAction(c.Request.Context(), middleware.GetTenantID(c), middleware.GetUserID(c), c.Param("caseId"), req)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"case": row})
	case errors.Is(err, engine.ErrM3Missing):
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	case errors.Is(err, engine.ErrM3Forbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "permission_denied"})
	case errors.Is(err, engine.ErrM3Invalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_action"})
	case errors.Is(err, engine.ErrM3Conflict):
		c.JSON(http.StatusConflict, gin.H{"error": "case_conflict"})
	default:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "action_unavailable"})
	}
}

// ListInterventionAssignees returns only a tenant member's id and display
// name. Submission independently checks current membership under a DB lock.
func ListInterventionAssignees(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	var rows []struct {
		UserID string `json:"user_id"`
		Name   string `json:"name"`
	}
	err := db.DB.Table("user_tenants AS ut").Select("ut.user_id, users.name").Joins("JOIN users ON users.id = ut.user_id").
		Where("ut.tenant_id = ? AND ut.role IN ?", tenantID, []string{"owner", "admin", "member"}).Order("users.name ASC, ut.user_id ASC").Limit(100).Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "assignees_unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func m3SourceStatus(row models.InterventionCase) string {
	if row.SnapshotID == nil {
		return "UNAVAILABLE"
	}
	var snap models.AnalysisSnapshot
	if db.DB.Where("id = ? AND tenant_id = ?", *row.SnapshotID, row.TenantID).Take(&snap).Error != nil {
		return "UNAVAILABLE"
	}
	if snap.Digest != row.SnapshotDigest || !engine.VerifySnapshotProvenance(snap, row.TenantID, row.ConversationID, row.RunID) {
		return "UNAVAILABLE"
	}
	if snap.Coverage != "complete" {
		return "INCOMPLETE"
	}
	var messages []models.Message
	if db.DB.Where("tenant_id = ? AND conversation_id = ?", row.TenantID, row.ConversationID).Order("sent_at ASC, id ASC").Find(&messages).Error != nil {
		return "UNAVAILABLE"
	}
	switch engine.CompareSnapshotToCurrentMessages(snap.Manifest, messages) {
	case engine.SourceIntegrityBoundCurrentnessUnverified:
		return "LOCAL_MATCH_CURRENTNESS_UNVERIFIED"
	case engine.SourceIntegrityChangedSinceAnalysis:
		return "CHANGED"
	default:
		return "UNAVAILABLE"
	}
}

func m3AllowedState(state string) bool {
	switch state {
	case "OPEN", "ASSIGNED", "WAITING_ON_DATA", "RESOLVED", "REJECTED":
		return true
	default:
		return false
	}
}
