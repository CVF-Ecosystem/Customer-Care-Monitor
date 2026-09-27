package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/notifications"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/pkg"
	"github.com/xuri/excelize/v2"
)

// jobCancelFuncs stores cancel functions for running jobs, keyed by job ID
var jobCancelFuncs sync.Map

type CreateJobRequest struct {
	Name            string          `json:"name" binding:"required,min=2,max=255"`
	Description     string          `json:"description"`
	JobType         string          `json:"job_type" binding:"required,oneof=qc_analysis classification"`
	InputChannelIDs []string        `json:"input_channel_ids" binding:"required,min=1"`
	RulesContent    string          `json:"rules_content"`
	RulesConfig     json.RawMessage `json:"rules_config"`
	SkipConditions  string          `json:"skip_conditions"`
	AIProvider      string          `json:"ai_provider" binding:"omitempty,oneof=claude gemini"`
	AIModel         string          `json:"ai_model"`
	Outputs         json.RawMessage `json:"outputs" binding:"required"`
	OutputSchedule  string          `json:"output_schedule" binding:"required,oneof=instant scheduled cron none"`
	OutputCron      string          `json:"output_cron"`
	OutputAt        *time.Time      `json:"output_at"`
	ScheduleType    string          `json:"schedule_type" binding:"required,oneof=cron after_sync manual"`
	ScheduleCron    string          `json:"schedule_cron"`
}

func ListJobs(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var jobs []models.Job
	db.DB.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&jobs)

	c.JSON(http.StatusOK, jobs)
}

func CreateJob(c *gin.Context) {
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "details": err.Error()})
		return
	}

	tenantID := middleware.GetTenantID(c)

	channelIDsJSON, _ := json.Marshal(req.InputChannelIDs)
	rulesConfig := "{}"
	if req.RulesConfig != nil {
		rulesConfig = string(req.RulesConfig)
	}

	now := time.Now()
	job := models.Job{
		ID:              pkg.NewUUID(),
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		JobType:         req.JobType,
		InputChannelIDs: string(channelIDsJSON),
		RulesContent:    req.RulesContent,
		RulesConfig:     rulesConfig,
		SkipConditions:  req.SkipConditions,
		AIProvider:      req.AIProvider,
		AIModel:         req.AIModel,
		Outputs:         string(req.Outputs),
		OutputSchedule:  req.OutputSchedule,
		OutputCron:      req.OutputCron,
		OutputAt:        req.OutputAt,
		ScheduleType:    req.ScheduleType,
		ScheduleCron:    req.ScheduleCron,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := db.DB.Create(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create_job_failed"})
		return
	}

	// Reload cron jobs if this is a scheduled job
	if job.ScheduleType == "cron" {
		if sched := engine.GetDefaultScheduler(); sched != nil {
			sched.ReloadJobs()
		}
	}

	c.JSON(http.StatusCreated, job)
}

func GetJob(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	jobID := c.Param("jobId")

	var job models.Job
	if err := db.DB.Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job_not_found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// allowedJobUpdateFields is a whitelist of fields that can be updated via the UpdateJob API.
var allowedJobUpdateFields = map[string]bool{
	"name": true, "description": true, "type": true, "status": true,
	"input_channel_ids": true, "outputs": true, "rules_config": true,
	"rules_content": true, "skip_conditions": true,
	"ai_provider": true, "ai_model": true, "ai_system_prompt": true,
	"schedule_type": true, "schedule_cron": true, "schedule_enabled": true,
	"date_from": true, "date_to": true, "max_conversations": true,
}

func UpdateJob(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	jobID := c.Param("jobId")

	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	// Filter to allowed fields only — prevents mass assignment of tenant_id, id, etc.
	req := make(map[string]interface{})
	for key, val := range raw {
		if allowedJobUpdateFields[key] {
			req[key] = val
		}
	}

	// JSON-encode array/object fields that are stored as strings in DB
	for _, key := range []string{"outputs", "input_channel_ids", "rules_config"} {
		if v, ok := req[key]; ok {
			switch v.(type) {
			case []interface{}, map[string]interface{}:
				encoded, _ := json.Marshal(v)
				req[key] = string(encoded)
			}
		}
	}

	req["updated_at"] = time.Now()

	result := db.DB.Model(&models.Job{}).Where("id = ? AND tenant_id = ?", jobID, tenantID).Updates(req)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "job_not_found"})
		return
	}

	// Reload cron jobs if schedule changed
	if _, ok := req["schedule_type"]; ok {
		if sched := engine.GetDefaultScheduler(); sched != nil {
			sched.ReloadJobs()
		}
	}
	if _, ok := req["schedule_cron"]; ok {
		if sched := engine.GetDefaultScheduler(); sched != nil {
			sched.ReloadJobs()
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func DeleteJob(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	jobID := c.Param("jobId")

	// Check job exists
	var job models.Job
	if err := db.DB.Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job_not_found"})
		return
	}

	// Cascade delete: results → runs → usage logs → notification logs → job, all
	// inside one transaction. Job runs are locked (FOR UPDATE) first — same
	// parent-first protocol as DeleteChannel/PurgeChannelConversations — so a
	// concurrent saveResults() holding that lock is waited out, and once it
	// releases, the child deletes below see everything it just wrote.
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var runs []models.JobRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("job_id = ? AND tenant_id = ?", jobID, tenantID).
			Find(&runs).Error; err != nil {
			return fmt.Errorf("lock job runs: %w", err)
		}
		runIDs := make([]string, len(runs))
		for i, r := range runs {
			runIDs[i] = r.ID
		}
		if len(runIDs) > 0 {
			if err := tx.Where("job_run_id IN ? AND tenant_id = ?", runIDs, tenantID).Delete(&models.JobResult{}).Error; err != nil {
				return fmt.Errorf("delete job results: %w", err)
			}
			if err := tx.Where("job_run_id IN ? AND tenant_id = ?", runIDs, tenantID).Delete(&models.AnalysisSnapshot{}).Error; err != nil {
				return fmt.Errorf("delete analysis snapshots: %w", err)
			}
		}
		if err := tx.Where("job_id = ? AND tenant_id = ?", jobID, tenantID).Delete(&models.JobRun{}).Error; err != nil {
			return fmt.Errorf("delete job runs: %w", err)
		}
		if err := tx.Where("job_id = ? AND tenant_id = ?", jobID, tenantID).Delete(&models.AIUsageLog{}).Error; err != nil {
			return fmt.Errorf("delete ai usage logs: %w", err)
		}
		if err := tx.Where("job_id = ? AND tenant_id = ?", jobID, tenantID).Delete(&models.NotificationLog{}).Error; err != nil {
			return fmt.Errorf("delete notification logs: %w", err)
		}
		if err := tx.Where("id = ? AND tenant_id = ?", jobID, tenantID).Delete(&models.Job{}).Error; err != nil {
			return fmt.Errorf("delete job: %w", err)
		}
		return nil
	})
	if err != nil {
		log.Printf("[error] delete job %s cascade: %v", jobID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete_job_failed"})
		return
	}

	// Reload cron jobs after deletion
	if job.ScheduleType == "cron" {
		if sched := engine.GetDefaultScheduler(); sched != nil {
			sched.ReloadJobs()
		}
	}

	db.LogActivity(tenantID, middleware.GetUserID(c), middleware.GetUserEmail(c), "job.delete", "job", jobID, "Deleted job: "+job.Name, "", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func ClearJobResults(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	jobID := c.Param("jobId")

	var job models.Job
	if err := db.DB.Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job_not_found"})
		return
	}

	// Delete results + usage logs + notification logs (keep runs)
	var runIDs []string
	db.DB.Model(&models.JobRun{}).Where("job_id = ? AND tenant_id = ?", jobID, tenantID).Pluck("id", &runIDs)
	if len(runIDs) > 0 {
		db.DB.Where("job_run_id IN ? AND tenant_id = ?", runIDs, tenantID).Delete(&models.JobResult{})
		db.DB.Where("job_run_id IN ? AND tenant_id = ?", runIDs, tenantID).Delete(&models.AnalysisSnapshot{})
	}
	db.DB.Where("job_id = ? AND tenant_id = ?", jobID, tenantID).Delete(&models.AIUsageLog{})
	db.DB.Where("job_id = ? AND tenant_id = ?", jobID, tenantID).Delete(&models.NotificationLog{})

	db.LogActivity(tenantID, middleware.GetUserID(c), middleware.GetUserEmail(c), "job.clear_results", "job", jobID, "Cleared results for job: "+job.Name, "", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "cleared"})
}

func ClearJobRuns(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	jobID := c.Param("jobId")

	var job models.Job
	if err := db.DB.Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job_not_found"})
		return
	}

	// Cascade delete: results → runs → usage logs → notification logs, all inside
	// one transaction. Job runs are locked (FOR UPDATE) first, same protocol as
	// DeleteJob, so a concurrent saveResults() holding that lock is waited out and
	// its evidence is caught by the child deletes once it releases.
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var runs []models.JobRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("job_id = ? AND tenant_id = ?", jobID, tenantID).
			Find(&runs).Error; err != nil {
			return fmt.Errorf("lock job runs: %w", err)
		}
		runIDs := make([]string, len(runs))
		for i, r := range runs {
			runIDs[i] = r.ID
		}
		if len(runIDs) > 0 {
			if err := tx.Where("job_run_id IN ? AND tenant_id = ?", runIDs, tenantID).Delete(&models.JobResult{}).Error; err != nil {
				return fmt.Errorf("delete job results: %w", err)
			}
			if err := tx.Where("job_run_id IN ? AND tenant_id = ?", runIDs, tenantID).Delete(&models.AnalysisSnapshot{}).Error; err != nil {
				return fmt.Errorf("delete analysis snapshots: %w", err)
			}
		}
		if err := tx.Where("job_id = ? AND tenant_id = ?", jobID, tenantID).Delete(&models.JobRun{}).Error; err != nil {
			return fmt.Errorf("delete job runs: %w", err)
		}
		if err := tx.Where("job_id = ? AND tenant_id = ?", jobID, tenantID).Delete(&models.AIUsageLog{}).Error; err != nil {
			return fmt.Errorf("delete ai usage logs: %w", err)
		}
		if err := tx.Where("job_id = ? AND tenant_id = ?", jobID, tenantID).Delete(&models.NotificationLog{}).Error; err != nil {
			return fmt.Errorf("delete notification logs: %w", err)
		}
		if err := tx.Model(&job).Updates(map[string]interface{}{
			"last_run_at":     nil,
			"last_run_status": "",
			"updated_at":      time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("reset job run state: %w", err)
		}
		return nil
	})
	if err != nil {
		log.Printf("[error] clear job runs %s: %v", jobID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "clear_runs_failed"})
		return
	}

	db.LogActivity(tenantID, middleware.GetUserID(c), middleware.GetUserEmail(c), "job.clear_runs", "job", jobID, "Cleared all runs for job: "+job.Name, "", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "cleared"})
}

func TestRunJob(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	jobID := c.Param("jobId")

	var job models.Job
	if err := db.DB.Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job_not_found"})
		return
	}

	// A job dispatch that cannot get a valid configuration must not be
	// acknowledged: validate before launching any worker. The error is not
	// logged or returned because validation messages describe secret
	// configuration.
	cfg, err := loadJobDispatchConfig()
	if err != nil || cfg == nil {
		log.Printf("[error] test-run for job %s not admitted: configuration invalid", job.Name)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "job_start_failed"})
		return
	}

	startTestRunJob(job, cfg, testRunConversationLimit)

	c.JSON(http.StatusAccepted, gin.H{"message": "test_run_started"})
}

func TriggerJob(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	jobID := c.Param("jobId")

	var job models.Job
	if err := db.DB.Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job_not_found"})
		return
	}

	// mode: "unanalyzed" | "since_last" | "conditional"
	// backward compat: if full=true treat as conditional
	mode := c.Query("mode")
	if mode == "" && c.Query("full") == "true" {
		mode = "conditional"
	}
	if mode == "" {
		mode = "since_last"
	}
	dateFrom := c.Query("from")
	dateTo := c.Query("to")
	limitStr := c.Query("limit")
	var maxConv int
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			maxConv = n
		}
	}

	// A job dispatch that cannot get a valid configuration must not be
	// acknowledged: validate before launching any worker. The error is not
	// logged or returned because validation messages describe secret
	// configuration.
	cfg, err := loadJobDispatchConfig()
	if err != nil || cfg == nil {
		log.Printf("[error] trigger for job %s not admitted: configuration invalid", job.Name)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "job_start_failed"})
		return
	}

	startTriggerJob(job, cfg, triggerJobParams{mode: mode, dateFrom: dateFrom, dateTo: dateTo, maxConv: maxConv})

	c.JSON(http.StatusAccepted, gin.H{"message": "job_triggered"})
}

// loadJobDispatchConfig loads and validates configuration for a job dispatch
// (test-run or trigger). It is a variable only so handler tests can force a
// load failure without touching real environment/config state.
var loadJobDispatchConfig = config.Load

// testRunConversationLimit caps how many conversations a test run analyzes.
const testRunConversationLimit = 3

// startTestRunJob launches the background test-run worker with the
// configuration already validated for this request and the conversation
// limit chosen by the handler. It is a variable only so handler tests can
// observe dispatch without running a real analyzer/provider.
var startTestRunJob = func(job models.Job, cfg *config.Config, limit int) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[security] panic in test-run goroutine for job %s: %v", job.Name, r)
			}
		}()
		analyzer := engine.NewAnalyzer(cfg)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		jobCancelFuncs.Store(job.ID, cancel)
		defer jobCancelFuncs.Delete(job.ID)
		if _, err := analyzer.RunJobWithLimit(ctx, job, limit); err != nil {
			log.Printf("[test-run] error for job %s: %v", job.Name, err)
		}
	}()
}

// triggerJobParams carries TriggerJob's resolved mode/date/limit parameters
// unchanged into startTriggerJob, so admission never alters trigger semantics.
type triggerJobParams struct {
	mode             string
	dateFrom, dateTo string
	maxConv          int
}

// startTriggerJob launches the background trigger worker with the
// configuration already validated for this request. It is a variable only so
// handler tests can observe dispatch without running a real
// analyzer/provider.
var startTriggerJob = func(job models.Job, cfg *config.Config, p triggerJobParams) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[security] panic in trigger goroutine for job %s: %v", job.Name, r)
			}
		}()
		analyzer := engine.NewAnalyzer(cfg)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		jobCancelFuncs.Store(job.ID, cancel)
		defer jobCancelFuncs.Delete(job.ID)
		var err error
		switch p.mode {
		case "unanalyzed":
			_, err = analyzer.RunJobUnanalyzed(ctx, job, p.maxConv)
		case "conditional":
			_, err = analyzer.RunJobFullWithParams(ctx, job, p.dateFrom, p.dateTo, p.maxConv)
		default: // "since_last"
			_, err = analyzer.RunJobSinceLast(ctx, job, p.maxConv)
		}
		if err != nil {
			log.Printf("[trigger] error for job %s: %v", job.Name, err)
		}
	}()
}

func CancelJob(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	jobID := c.Param("jobId")

	var job models.Job
	if err := db.DB.Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job_not_found"})
		return
	}

	// Cancel the running context
	if cancelFn, ok := jobCancelFuncs.Load(job.ID); ok {
		cancelFn.(context.CancelFunc)()
		jobCancelFuncs.Delete(job.ID)
	}

	// Mark running job_runs as cancelled
	finishedAt := time.Now()
	db.DB.Model(&models.JobRun{}).
		Where("job_id = ? AND status = ?", job.ID, "running").
		Updates(map[string]interface{}{
			"status":        "cancelled",
			"finished_at":   &finishedAt,
			"error_message": "Cancelled by user",
		})

	log.Printf("[security] job cancelled: user=%s job=%s tenant=%s ip=%s", userID, jobID, tenantID, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "job_cancelled"})
}

func ListJobRuns(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	jobID := c.Param("jobId")

	var runs []models.JobRun
	db.DB.Where("job_id = ? AND tenant_id = ?", jobID, tenantID).
		Order("started_at DESC").Limit(50).Find(&runs)

	c.JSON(http.StatusOK, runs)
}

func TestOutput(c *gin.Context) {
	var req struct {
		Type     string `json:"type" binding:"required,oneof=telegram email"`
		BotToken string `json:"bot_token"`
		ChatID   string `json:"chat_id"`
		// Email fields can be added later
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "details": err.Error()})
		return
	}

	ctx := c.Request.Context()

	switch req.Type {
	case "telegram":
		if req.BotToken == "" || req.ChatID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bot_token and chat_id are required"})
			return
		}
		notifier := notifications.NewTelegramNotifier(req.BotToken, req.ChatID)
		err := notifier.Send(ctx, "Customer Care Monitor AI - Test", "Đây là tin nhắn thử nghiệm từ Customer Care Monitor AI.\nKết nối Telegram thành công!")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Telegram message sent"})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported output type"})
	}
}

// JobResultWithIntegrity is a job result plus its read-time, local-only
// source-integrity status (CCMAI-RUNTIME-004 semantics). R005 confidence
// fields are carried unchanged by the embedded JobResult.
type JobResultWithIntegrity struct {
	models.JobResult
	SourceIntegrityStatus string `json:"source_integrity_status"`
}

func ListJobResults(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	runID := c.Param("runId")

	var results []models.JobResult
	if err := db.DB.Where("job_run_id = ? AND tenant_id = ?", runID, tenantID).
		Order("created_at DESC").Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query_failed"})
		return
	}

	refs := make([]sourceIntegrityRef, len(results))
	for i, r := range results {
		refs[i] = sourceIntegrityRef{ConversationID: r.ConversationID, JobRunID: r.JobRunID, AnalysisSnapshotID: r.AnalysisSnapshotID}
	}
	statuses, err := computeSourceIntegrity(tenantID, refs)
	if err != nil {
		log.Printf("[jobs] source integrity: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query_failed"})
		return
	}

	out := make([]JobResultWithIntegrity, len(results))
	for i, r := range results {
		out[i] = JobResultWithIntegrity{JobResult: r, SourceIntegrityStatus: statuses[i]}
	}
	c.JSON(http.StatusOK, out)
}

// JobResultWithConvDate extends JobResult with the conversation's start date,
// customer name and read-time source-integrity status.
type JobResultWithConvDate struct {
	models.JobResult
	ConversationDate      *time.Time `json:"conversation_date"`
	CustomerName          string     `json:"customer_name"`
	SourceIntegrityStatus string     `gorm:"-" json:"source_integrity_status"`
}

// loadJobResultsWithIntegrity loads every result of the job's runs, newest
// first, and attaches source_integrity_status. Any query error is returned so
// callers fail before writing a response body or download headers.
func loadJobResultsWithIntegrity(tenantID, jobID string) ([]JobResultWithConvDate, error) {
	var runIDs []string
	if err := db.DB.Model(&models.JobRun{}).Where("job_id = ? AND tenant_id = ?", jobID, tenantID).
		Pluck("id", &runIDs).Error; err != nil {
		return nil, fmt.Errorf("truy vấn lần chạy: %w", err)
	}
	if len(runIDs) == 0 {
		return []JobResultWithConvDate{}, nil
	}

	var results []JobResultWithConvDate
	if err := db.DB.Model(&models.JobResult{}).
		Select("job_results.*, (SELECT MIN(m.sent_at) FROM messages m WHERE m.conversation_id = job_results.conversation_id) as conversation_date, conversations.customer_name as customer_name").
		Joins("LEFT JOIN conversations ON conversations.id = job_results.conversation_id").
		Where("job_results.job_run_id IN ? AND job_results.tenant_id = ?", runIDs, tenantID).
		Order("job_results.created_at DESC").
		Find(&results).Error; err != nil {
		return nil, fmt.Errorf("truy vấn kết quả: %w", err)
	}

	refs := make([]sourceIntegrityRef, len(results))
	for i, r := range results {
		refs[i] = sourceIntegrityRef{ConversationID: r.ConversationID, JobRunID: r.JobRunID, AnalysisSnapshotID: r.AnalysisSnapshotID}
	}
	statuses, err := computeSourceIntegrity(tenantID, refs)
	if err != nil {
		return nil, err
	}
	for i := range results {
		results[i].SourceIntegrityStatus = statuses[i]
	}
	return results, nil
}

// ListAllJobResults returns all results across all runs for a job.
func ListAllJobResults(c *gin.Context) {
	results, err := loadJobResultsWithIntegrity(middleware.GetTenantID(c), c.Param("jobId"))
	if err != nil {
		log.Printf("[jobs] list all results: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query_failed"})
		return
	}
	c.JSON(http.StatusOK, results)
}

// exportConvRow holds one grouped conversation row for export.
type exportConvRow struct {
	CustomerName          string
	ConversationDate      string
	EvalDate              string
	Review                string
	Verdict               string
	Score                 string
	Issues                string
	SourceIntegrity       string
	SourceIntegrityDetail string
}

// exportIntegrity collects one conversation group's per-result statuses. The
// export groups every run's results for a conversation, so a group can mix
// statuses; all distinct ones are shown, most concerning first, plus a
// per-result "id: label" detail for traceability.
type exportIntegrity struct {
	statuses []string
	details  []string
}

func (e *exportIntegrity) add(r JobResultWithConvDate) {
	e.statuses = append(e.statuses, r.SourceIntegrityStatus)
	e.details = append(e.details, r.ID+": "+sourceIntegrityLabel(r.SourceIntegrityStatus))
}

func (e *exportIntegrity) summary() string { return distinctSourceIntegrityLabels(e.statuses) }
func (e *exportIntegrity) detail() string  { return strings.Join(e.details, "; ") }

const (
	exportIntegrityHeader       = "Tính toàn vẹn nguồn"
	exportIntegrityDetailHeader = "Chi tiết toàn vẹn nguồn (mã kết quả)"
)

// buildExportRows groups raw JobResultWithConvDate records by conversation and returns sorted rows.
func buildExportRows(results []JobResultWithConvDate) []exportConvRow {
	type convGroup struct {
		customerName     string
		conversationDate string
		evalDate         string
		review           string
		verdict          string
		score            string
		violations       []string
		integrity        exportIntegrity
	}
	groups := map[string]*convGroup{}
	order := []string{}

	for _, r := range results {
		cid := r.ConversationID
		if _, ok := groups[cid]; !ok {
			convDate := ""
			if r.ConversationDate != nil {
				convDate = r.ConversationDate.Format("2006-01-02 15:04")
			}
			groups[cid] = &convGroup{
				customerName:     r.CustomerName,
				conversationDate: convDate,
			}
			order = append(order, cid)
		}
		g := groups[cid]
		g.integrity.add(r)
		if r.ResultType == "conversation_evaluation" {
			verdict := r.Severity
			if verdict == "PASS" {
				g.verdict = "Đạt"
			} else if verdict == "SKIP" {
				g.verdict = "Bỏ qua"
			} else {
				g.verdict = "Không đạt"
			}
			g.review = r.Evidence
			g.evalDate = r.CreatedAt.Format("2006-01-02 15:04")
			// Parse score from detail JSON
			var detail map[string]interface{}
			if err := json.Unmarshal([]byte(r.Detail), &detail); err == nil {
				if s, ok := detail["score"]; ok {
					g.score = fmt.Sprintf("%v", s)
				}
			}
		} else if r.ResultType != "conversation_evaluation" {
			issue := r.RuleName
			if r.Evidence != "" {
				issue += ": " + r.Evidence
			}
			g.violations = append(g.violations, issue)
		}
	}

	rows := make([]exportConvRow, 0, len(order))
	for _, cid := range order {
		g := groups[cid]
		rows = append(rows, exportConvRow{
			CustomerName:          g.customerName,
			ConversationDate:      g.conversationDate,
			EvalDate:              g.evalDate,
			Review:                g.review,
			Verdict:               g.verdict,
			Score:                 g.score,
			Issues:                strings.Join(g.violations, "; "),
			SourceIntegrity:       g.integrity.summary(),
			SourceIntegrityDetail: g.integrity.detail(),
		})
	}
	return rows
}

// ExportJobResults returns all results as CSV or XLSX for download. Every
// query (results, source integrity, job type, classification chat text) runs
// and is checked before any download header or file byte is written.
func ExportJobResults(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	jobID := c.Param("jobId")
	format := c.DefaultQuery("format", "csv")

	results, err := loadJobResultsWithIntegrity(tenantID, jobID)
	if err != nil {
		log.Printf("[jobs] export results: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query_failed"})
		return
	}

	var jobType string
	if err := db.DB.Model(&models.Job{}).Where("id = ? AND tenant_id = ?", jobID, tenantID).Pluck("job_type", &jobType).Error; err != nil {
		log.Printf("[jobs] export job type: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query_failed"})
		return
	}

	if jobType == "classification" {
		exportClassification(c, tenantID, results, format)
		return
	}

	rows := buildExportRows(results)
	headers := []string{"Tên", "Ngày phát sinh chat", "Ngày đánh giá", "Kết quả đánh giá chi tiết", "Đánh giá", "Điểm", "Vấn đề", exportIntegrityHeader, exportIntegrityDetailHeader}
	records := make([][]string, 0, len(rows))
	for _, r := range rows {
		records = append(records, []string{r.CustomerName, r.ConversationDate, r.EvalDate, r.Review, r.Verdict, r.Score, r.Issues, r.SourceIntegrity, r.SourceIntegrityDetail})
	}
	writeJobExport(c, "results", format, headers, records)
}

func writeJobExport(c *gin.Context, filename, format string, headers []string, records [][]string) {
	if format == "xlsx" {
		writeResultsXLSX(c, filename, headers, records)
		return
	}
	writeResultsCSV(c, filename, headers, records)
}

// loadChatTextByConversation reads each conversation's messages as they are
// now (not the analyzed snapshot), in chunked tenant-scoped batches.
func loadChatTextByConversation(tenantID string, convIDs []string) (map[string]string, error) {
	lines := map[string][]string{}
	for _, chunk := range chunkStrings(convIDs, sourceIntegrityBatchSize) {
		var messages []models.Message
		if err := db.DB.Where("tenant_id = ? AND conversation_id IN ?", tenantID, chunk).
			Order("conversation_id ASC, sent_at ASC, id ASC").Find(&messages).Error; err != nil {
			return nil, fmt.Errorf("truy vấn nội dung chat: %w", err)
		}
		for _, m := range messages {
			if m.Content == "" {
				continue
			}
			name := m.SenderName
			if name == "" {
				name = m.SenderType
			}
			lines[m.ConversationID] = append(lines[m.ConversationID], fmt.Sprintf("[%s] %s", name, m.Content))
		}
	}
	chat := make(map[string]string, len(lines))
	for cid, l := range lines {
		chat[cid] = strings.Join(l, "\n")
	}
	return chat, nil
}

// exportClassification exports classification results with Tags + Issues + Chat content columns.
func exportClassification(c *gin.Context, tenantID string, results []JobResultWithConvDate, format string) {
	type convGroup struct {
		customerName     string
		conversationDate string
		evalDate         string
		tags             []string
		issues           []string
		integrity        exportIntegrity
	}
	groups := map[string]*convGroup{}
	order := []string{}

	for _, r := range results {
		cid := r.ConversationID
		if _, ok := groups[cid]; !ok {
			convDate := ""
			if r.ConversationDate != nil {
				convDate = r.ConversationDate.Format("2006-01-02 15:04")
			}
			groups[cid] = &convGroup{
				customerName:     r.CustomerName,
				conversationDate: convDate,
			}
			order = append(order, cid)
		}
		g := groups[cid]
		g.integrity.add(r)
		if r.ResultType == "conversation_evaluation" {
			g.evalDate = r.CreatedAt.Format("2006-01-02 15:04")
		} else if r.ResultType == "classification_tag" {
			g.tags = append(g.tags, r.RuleName)
			if r.Evidence != "" {
				g.issues = append(g.issues, r.Evidence)
			}
		}
	}

	chatMap, err := loadChatTextByConversation(tenantID, order)
	if err != nil {
		log.Printf("[jobs] export classification chat: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query_failed"})
		return
	}

	headers := []string{"Tên", "Ngày phát sinh chat", "Ngày đánh giá", "Loại", "Vấn đề",
		exportChatHeader, exportIntegrityHeader, exportIntegrityDetailHeader}
	records := make([][]string, 0, len(order))
	for _, cid := range order {
		g := groups[cid]
		records = append(records, []string{
			g.customerName, g.conversationDate, g.evalDate,
			strings.Join(g.tags, "\n"), strings.Join(g.issues, "\n"), chatMap[cid],
			g.integrity.summary(), g.integrity.detail(),
		})
	}
	writeJobExport(c, "classification", format, headers, records)
}

// exportChatHeader makes clear the chat text is read at export time and may
// differ from what was analyzed.
const exportChatHeader = "Nội dung chat (đọc lúc xuất file, có thể khác bản đã phân tích)"

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
