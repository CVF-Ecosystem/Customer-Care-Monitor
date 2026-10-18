package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func waitDataDecode(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if d.Decode(dst) != nil || d.Decode(new(any)) != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return false
	}
	return true
}

func waitDataError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, engine.ErrWaitDataInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
	case errors.Is(err, engine.ErrWaitDataForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "permission_denied"})
	case errors.Is(err, engine.ErrWaitDataMissing):
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	case errors.Is(err, engine.ErrWaitDataUnchanged):
		c.JSON(http.StatusConflict, gin.H{"error": "source_unchanged"})
	case errors.Is(err, engine.ErrWaitDataKeyConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "request_key_conflict"})
	case errors.Is(err, engine.ErrWaitDataPending):
		c.JSON(http.StatusConflict, gin.H{"error": "request_pending"})
	case errors.Is(err, engine.ErrWaitDataConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "case_conflict"})
	default:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "source_unavailable"})
	}
}

func CreateInterventionDataRevision(c *gin.Context) {
	var in engine.DataRevisionInput
	if !waitDataDecode(c, &in) {
		return
	}
	r, err := engine.CaptureInterventionDataRevision(c.Request.Context(), middleware.GetTenantID(c), middleware.GetUserID(c), c.Param("caseId"), in)
	if err != nil {
		waitDataError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"revision": revisionView(*r), "case_version": r.CaseVersion})
}

func CreateInterventionRerunRequest(c *gin.Context) {
	var in engine.RerunRequestInput
	if !waitDataDecode(c, &in) {
		return
	}
	r, replayed, err := engine.RequestInterventionRerun(c.Request.Context(), middleware.GetTenantID(c), middleware.GetUserID(c), c.Param("caseId"), in)
	if err != nil {
		waitDataError(c, err)
		return
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"request": requestView(*r), "case_version": r.CaseVersion, "replayed": replayed})
}

// ExecuteInterventionRerun accepts only a grant reference and expected case
// version. The service is installed explicitly by server-owned configuration.
func ExecuteInterventionRerun(c *gin.Context) {
	var in engine.WaitDataExecuteInput
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 512)
	d := json.NewDecoder(c.Request.Body)
	start, err := d.Token()
	if err != nil || start != json.Delim('{') { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_request"}); return }
	fields := map[string]json.RawMessage{}
	for d.More() {
		key, err := d.Token()
		name, ok := key.(string)
		if err != nil || !ok || (name != "expected_case_version" && name != "grant_id") { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_request"}); return }
		if _, duplicate := fields[name]; duplicate { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_request"}); return }
		var raw json.RawMessage
		if d.Decode(&raw) != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_request"}); return }
		fields[name] = raw
	}
	end, endErr := d.Token()
	if endErr != nil || end != json.Delim('}') || len(fields) != 2 || d.Decode(new(any)) != io.EOF || json.Unmarshal(fields["expected_case_version"], &in.ExpectedCaseVersion) != nil || json.Unmarshal(fields["grant_id"], &in.GrantID) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	r, replayed, err := engine.ClaimWaitDataExecution(c.Request.Context(), middleware.GetTenantID(c), middleware.GetUserID(c), c.Param("caseId"), c.Param("requestId"), in, engine.WaitDataExecutor)
	if err != nil {
		waitDataError(c, err)
		return
	}
	status := http.StatusAccepted
	if replayed && r.State != "CLAIMED" && r.State != "CALL_MAY_HAVE_STARTED" && r.State != "RESPONSE_RECORDED" {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"execution": gin.H{"id": r.ID, "state": r.State, "terminal_code": r.TerminalCode, "run_id": r.RunID, "snapshot_id": r.SnapshotID}, "replayed": replayed})
}

func waitDataReadCase(c *gin.Context) (models.InterventionCase, bool) {
	tenant, actor := middleware.GetTenantID(c), middleware.GetUserID(c)
	var membership models.UserTenant
	membershipErr := db.DB.Where("tenant_id = ? AND user_id = ?", tenant, actor).Take(&membership).Error
	if membershipErr != nil && !errors.Is(membershipErr, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "source_unavailable"})
		return models.InterventionCase{}, false
	}
	if membershipErr != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission_denied"})
		return models.InterventionCase{}, false
	}
	allowed := membership.Role == "owner" || membership.Role == "admin"
	if membership.Role == "member" {
		var p map[string]string
		allowed = json.Unmarshal([]byte(membership.Permissions), &p) == nil && strings.Contains(p["intervention_cases"], "r")
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission_denied"})
		return models.InterventionCase{}, false
	}
	var row models.InterventionCase
	if db.DB.Where("tenant_id = ? AND id = ?", tenant, c.Param("caseId")).Take(&row).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return models.InterventionCase{}, false
	}
	return row, true
}

func waitDataPage(c *gin.Context) (*gorm.DB, int, bool) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_limit"})
			return nil, 0, false
		}
		limit = n
	}
	q := db.DB.Where("tenant_id = ? AND case_id = ?", middleware.GetTenantID(c), c.Param("caseId"))
	if cursor := c.Query("cursor"); cursor != "" {
		if len(cursor) > 256 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_cursor"})
			return nil, 0, false
		}
		b, err := base64.RawURLEncoding.DecodeString(cursor)
		parts := strings.SplitN(string(b), "|", 2)
		if err != nil || len(parts) != 2 || len(parts[1]) != 36 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_cursor"})
			return nil, 0, false
		}
		at, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_cursor"})
			return nil, 0, false
		}
		q = q.Where("created_at > ? OR (created_at = ? AND id > ?)", at, at, parts[1])
	}
	return q, limit, true
}

func waitDataNext(at time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func revisionView(r models.InterventionDataRevision) gin.H {
	return gin.H{"id": r.ID, "case_id": r.CaseID, "job_id": r.JobID, "conversation_id": r.ConversationID, "actor_id": r.ActorID, "originating_case_version": r.OriginatingCaseVersion, "case_version": r.CaseVersion, "original_snapshot_id": r.OriginalSnapshotID, "original_snapshot_digest": r.OriginalSnapshotDigest, "digest": r.Digest, "coverage": r.Coverage, "coverage_reasons": json.RawMessage(r.CoverageReasons), "message_count": r.MessageCount, "observation_status": r.ObservationStatus, "reason": r.Reason, "created_at": r.CreatedAt}
}

func requestView(r models.InterventionRerunRequest) gin.H {
	return gin.H{"id": r.ID, "case_id": r.CaseID, "job_id": r.JobID, "conversation_id": r.ConversationID, "data_revision_id": r.DataRevisionID, "data_revision_digest": r.DataRevisionDigest, "original_snapshot_id": r.OriginalSnapshotID, "original_snapshot_digest": r.OriginalSnapshotDigest, "actor_id": r.ActorID, "reason": r.Reason, "request_key": r.RequestKey, "status": r.Status, "execution": r.Execution, "case_version": r.CaseVersion, "created_at": r.CreatedAt}
}

func ListInterventionDataRevisions(c *gin.Context) {
	if _, ok := waitDataReadCase(c); !ok {
		return
	}
	q, limit, ok := waitDataPage(c)
	if !ok {
		return
	}
	var rows []models.InterventionDataRevision
	if q.Order("created_at ASC, id ASC").Limit(limit+1).Find(&rows).Error != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "source_unavailable"})
		return
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		items = append(items, revisionView(r))
	}
	next := ""
	if hasMore {
		last := rows[len(rows)-1]
		next = waitDataNext(last.CreatedAt, last.ID)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "has_more": hasMore, "next_cursor": next})
}

func GetInterventionDataRevision(c *gin.Context) {
	if _, ok := waitDataReadCase(c); !ok {
		return
	}
	var r models.InterventionDataRevision
	if db.DB.Where("tenant_id = ? AND case_id = ? AND id = ?", middleware.GetTenantID(c), c.Param("caseId"), c.Param("revisionId")).Take(&r).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	if !engine.VerifyInterventionDataRevision(r) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "source_unavailable"})
		return
	}
	view := revisionView(r)
	view["manifest"] = json.RawMessage(r.Manifest)
	c.JSON(http.StatusOK, gin.H{"revision": view})
}

func ListInterventionRerunRequests(c *gin.Context) {
	if _, ok := waitDataReadCase(c); !ok {
		return
	}
	q, limit, ok := waitDataPage(c)
	if !ok {
		return
	}
	var rows []models.InterventionRerunRequest
	if q.Order("created_at ASC, id ASC").Limit(limit+1).Find(&rows).Error != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "source_unavailable"})
		return
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		items = append(items, requestView(r))
	}
	next := ""
	if hasMore {
		last := rows[len(rows)-1]
		next = waitDataNext(last.CreatedAt, last.ID)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "has_more": hasMore, "next_cursor": next})
}
