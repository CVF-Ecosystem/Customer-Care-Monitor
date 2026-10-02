package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
)

// verifyTenantAccess checks user has access to the specified tenant.
func verifyTenantAccess(c *gin.Context, tenantID string) bool {
	_, ok := tenantMembership(c, tenantID)
	return ok
}

// tenantMembership loads the caller's membership row for the requested tenant only.
func tenantMembership(c *gin.Context, tenantID string) (models.UserTenant, bool) {
	userID := middleware.GetUserID(c)
	var ut models.UserTenant
	if db.DB.Where("user_id = ? AND tenant_id = ?", userID, tenantID).First(&ut).Error != nil {
		log.Printf("[security] agent API tenant access denied: user=%s tenant=%s ip=%s", userID, tenantID, c.ClientIP())
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_access_denied"})
		return ut, false
	}
	return ut, true
}

// agentPerm is one tenant permission an agent operation needs; all listed rights are conjunctive.
type agentPerm struct{ resource, action string }

// CCMAI-RUNTIME-020 (F01-A): exact HTTP agent action/resource matrix. A pair that is not
// listed is unsupported and is rejected before any configuration load, dispatch or read;
// it never falls back to a broader permission.
var agentRunPerms = map[string]map[string][]agentPerm{
	"cqa.sync": {
		"sync_all":     {{"channels", "w"}, {"messages", "w"}},
		"sync_channel": {{"channels", "w"}, {"messages", "w"}},
	},
	"cqa.qc":       {"analyze_quality": {{"jobs", "w"}, {"messages", "r"}}},
	"cqa.classify": {"classify_conversations": {{"jobs", "w"}, {"messages", "r"}}},
}

var agentQueryPerms = map[string]map[string][]agentPerm{
	"cqa.sync": {
		"conversations": {{"messages", "r"}},
		"messages":      {{"messages", "r"}},
	},
	"cqa.qc":       {"violations": {{"jobs", "r"}}},
	"cqa.classify": {"tags": {{"jobs", "r"}}},
}

// authorizeAgentOperation applies the requested tenant's stored role and permissions to the
// operation. Only owner, admin and member roles are recognized; anything else, and any
// missing or malformed permission JSON, is denied. The 403 body is generic and the log
// carries classes only (never the permission JSON or request parameters).
func authorizeAgentOperation(c *gin.Context, ut models.UserTenant, agentName, op string, need []agentPerm) bool {
	if ut.Role != "owner" && ut.Role != "admin" && ut.Role != "member" {
		log.Printf("[security] agent API permission denied: user=%s tenant=%s agent=%s op=%s reason=unrecognized_role", middleware.GetUserID(c), ut.TenantID, agentName, op)
		c.JSON(http.StatusForbidden, gin.H{"error": "permission_denied"})
		return false
	}
	for _, p := range need {
		if reason := middleware.PermissionDenial(ut.Role, ut.Permissions, p.resource, p.action); reason != "" {
			log.Printf("[security] agent API permission denied: user=%s tenant=%s agent=%s op=%s reason=%s", middleware.GetUserID(c), ut.TenantID, agentName, op, reason)
			c.JSON(http.StatusForbidden, gin.H{"error": "permission_denied"})
			return false
		}
	}
	return true
}

// Agent capability descriptor for Company OS discovery.
type AgentInfo struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

func ListAgents(c *gin.Context) {
	agents := []AgentInfo{
		{
			Name:         "cqa.sync",
			Description:  "Sync chat messages from external channels (Zalo OA, Facebook) into Customer Care Monitor AI",
			Version:      "1.0.0",
			Capabilities: []string{"sync_all", "sync_channel", "query:conversations", "query:messages"},
		},
		{
			Name:         "cqa.qc",
			Description:  "Analyze customer service chat quality against defined rules using AI",
			Version:      "1.0.0",
			Capabilities: []string{"analyze_quality", "query:violations", "query:scores"},
		},
		{
			Name:         "cqa.classify",
			Description:  "Classify and tag conversations using AI-powered rule matching",
			Version:      "1.0.0",
			Capabilities: []string{"classify_conversations", "query:tags", "query:rules"},
		},
	}
	c.JSON(http.StatusOK, agents)
}

type AgentRunRequest struct {
	TenantID string                 `json:"tenant_id" binding:"required"`
	Action   string                 `json:"action" binding:"required"`
	Params   map[string]interface{} `json:"params"`
}

type AgentRunResponse struct {
	Status  string                 `json:"status"`
	Summary map[string]interface{} `json:"summary,omitempty"`
	Errors  []string               `json:"errors,omitempty"`
}

func AgentRun(c *gin.Context) {
	agentName := c.Param("agentName")

	var req AgentRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "details": err.Error()})
		return
	}

	// Verify user has access to the requested tenant
	ut, ok := tenantMembership(c, req.TenantID)
	if !ok {
		return
	}

	// An unknown agent name is rejected before configuration is loaded.
	switch agentName {
	case "cqa.sync", "cqa.qc", "cqa.classify":
	default:
		c.JSON(http.StatusNotFound, gin.H{"error": "agent_not_found"})
		return
	}

	// F01-A: the exact agent/action pair must be supported and the caller must hold its
	// tenant permissions, all before configuration load, dispatch or any read/write.
	need, supported := agentRunPerms[agentName][req.Action]
	if !supported {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_action"})
		return
	}
	if !authorizeAgentOperation(c, ut, agentName, "run:"+req.Action, need) {
		return
	}

	// Admit the run only with a validated configuration (CCMAI-RUNTIME-011).
	// The failure is logged by class only: the loader error and request
	// params can carry secrets.
	cfg, err := loadAgentRunConfig()
	if err != nil || cfg == nil {
		log.Printf("[agent] run %s not admitted: configuration invalid", agentName)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent_run_failed"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if agentName == "cqa.sync" {
		c.JSON(http.StatusOK, dispatchSyncAgent(ctx, cfg, req))
		return
	}
	c.JSON(http.StatusOK, dispatchAnalysisAgent(ctx, cfg, req, agentName))
}

// Seams for tests: configuration loader and engine dispatchers.
var (
	loadAgentRunConfig    = config.Load
	dispatchSyncAgent     = handleSyncAgent
	dispatchAnalysisAgent = handleAnalysisAgent
)

func AgentQuery(c *gin.Context) {
	agentName := c.Param("agentName")
	tenantID := c.Query("tenant_id")
	resource := c.Query("resource")

	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id_required"})
		return
	}

	// Verify user has access to the requested tenant
	ut, ok := tenantMembership(c, tenantID)
	if !ok {
		return
	}

	switch agentName {
	case "cqa.sync", "cqa.qc", "cqa.classify":
	default:
		c.JSON(http.StatusNotFound, gin.H{"error": "agent_not_found"})
		return
	}

	// F01-A: an unsupported resource keeps its existing 400; a supported one needs the
	// tenant permission before any data is read.
	need, supported := agentQueryPerms[agentName][resource]
	if !supported {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown resource: " + resource})
		return
	}
	if !authorizeAgentOperation(c, ut, agentName, "query:"+resource, need) {
		return
	}

	switch agentName {
	case "cqa.sync":
		handleSyncQuery(c, tenantID, resource)
	case "cqa.qc":
		handleQCQuery(c, tenantID, resource)
	case "cqa.classify":
		handleClassifyQuery(c, tenantID, resource)
	default:
		c.JSON(http.StatusNotFound, gin.H{"error": "agent_not_found"})
	}
}

func AgentHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now()})
}

func handleSyncAgent(ctx context.Context, cfg *config.Config, req AgentRunRequest) AgentRunResponse {
	syncEngine := engine.NewSyncEngine(cfg)

	switch req.Action {
	case "sync_all":
		err := syncEngine.SyncAllChannels(ctx, req.TenantID)
		if err != nil {
			return AgentRunResponse{Status: "error", Errors: []string{err.Error()}}
		}
		return AgentRunResponse{Status: "success"}
	case "sync_channel":
		channelID, _ := req.Params["channel_id"].(string)
		var channel models.Channel
		if err := db.DB.Where("id = ? AND tenant_id = ?", channelID, req.TenantID).First(&channel).Error; err != nil {
			return AgentRunResponse{Status: "error", Errors: []string{"channel not found"}}
		}
		err := syncEngine.SyncChannel(ctx, channel)
		if errors.Is(err, engine.ErrSyncAlreadyRunning) {
			// Another entry path owns this channel: not a success, bounded reason.
			return AgentRunResponse{Status: "error", Errors: []string{"sync_already_running"}}
		}
		if err != nil {
			return AgentRunResponse{Status: "error", Errors: []string{err.Error()}}
		}
		return AgentRunResponse{Status: "success"}
	default:
		return AgentRunResponse{Status: "error", Errors: []string{"unknown action: " + req.Action}}
	}
}

func handleAnalysisAgent(ctx context.Context, cfg *config.Config, req AgentRunRequest, agentName string) AgentRunResponse {
	analyzer := engine.NewAnalyzer(cfg)

	jobType := "qc_analysis"
	if agentName == "cqa.classify" {
		jobType = "classification"
	}

	// Find matching active jobs
	var jobs []models.Job
	db.DB.Where("tenant_id = ? AND job_type = ? AND is_active = true", req.TenantID, jobType).Find(&jobs)

	if len(jobs) == 0 {
		return AgentRunResponse{Status: "error", Errors: []string{"no active jobs found for type: " + jobType}}
	}

	// CCMAI-RUNTIME-028: every job goes through the shared admission. Errors are bounded codes (no
	// raw SQL/config text): all jobs failed or busy => error; some completed, some not => partial;
	// all completed => success.
	var errs []string
	completed, failed, partial := 0, 0, 0
	for _, job := range jobs {
		run, err := analyzer.RunJob(ctx, job)
		switch {
		case errors.Is(err, engine.ErrJobBusy):
			failed++
			errs = append(errs, "job_already_running")
		case err != nil || run == nil || run.Status == "error" || run.Status == "cancelled":
			failed++
			errs = append(errs, "job_run_failed")
		case run.Status == "partial":
			completed++
			partial++
		default:
			completed++
		}
	}

	status := "success"
	switch {
	case failed > 0 && completed == 0:
		status = "error"
	case failed > 0 || partial > 0:
		status = "partial"
	}
	return AgentRunResponse{Status: status, Errors: errs}
}

func handleSyncQuery(c *gin.Context, tenantID, resource string) {
	switch resource {
	case "conversations":
		var convs []models.Conversation
		db.DB.Where("tenant_id = ?", tenantID).Order("last_message_at DESC").Limit(50).Find(&convs)
		c.JSON(http.StatusOK, convs)
	case "messages":
		convID := c.Query("conversation_id")
		var msgs []models.Message
		db.DB.Where("tenant_id = ? AND conversation_id = ?", tenantID, convID).Order("sent_at ASC").Limit(100).Find(&msgs)
		c.JSON(http.StatusOK, msgs)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown resource: " + resource})
	}
}

func handleQCQuery(c *gin.Context, tenantID, resource string) {
	switch resource {
	case "violations":
		var results []models.JobResult
		db.DB.Where("tenant_id = ? AND result_type = 'qc_violation'", tenantID).
			Order("created_at DESC").Limit(50).Find(&results)
		c.JSON(http.StatusOK, results)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown resource: " + resource})
	}
}

func handleClassifyQuery(c *gin.Context, tenantID, resource string) {
	switch resource {
	case "tags":
		var results []models.JobResult
		db.DB.Where("tenant_id = ? AND result_type = 'classification_tag'", tenantID).
			Order("created_at DESC").Limit(50).Find(&results)
		c.JSON(http.StatusOK, results)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown resource: " + resource})
	}
}
