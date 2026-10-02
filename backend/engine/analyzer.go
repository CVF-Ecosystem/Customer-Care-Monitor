package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm/clause"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/notifications"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// Analyzer executes analysis jobs: loads messages, calls AI, saves results.
type Analyzer struct {
	cfg *config.Config
	// providerOverride is a private test seam (CCMAI-RUNTIME-025) that lets scheduler tests
	// route a synthetic provider through the real Analyzer. Never set in production.
	providerOverride ai.AIProvider
}

func NewAnalyzer(cfg *config.Config) *Analyzer {
	return &Analyzer{cfg: cfg}
}

// execute reserves the tenant/job (CCMAI-RUNTIME-028: one shared admission for every entry path,
// including the public Analyzer methods) and runs the planned analysis under that ownership.
func (a *Analyzer) execute(ctx context.Context, job models.Job, plan runPlan, injectedProvider ai.AIProvider) (*models.JobRun, error) {
	res, err := ReserveJobRun(ctx, job, 0)
	if err != nil {
		return nil, err
	}
	return a.executeReserved(res, job, plan, injectedProvider)
}

// executeReserved runs one planned analysis (CCMAI-RUNTIME-027) for an already reserved, owned and
// persisted running row. The reservation is consumed exactly once; ownership is released only after
// the worker's last effect (terminal write, notification, cleanup). The plan fixes the mode, the
// cap and the date bounds; the ordinary mode keeps the R025 source-version selection, every
// explicit mode uses its own candidate query (analyzer_modes.go), and all modes analyze the full
// local snapshot that was prepared before the provider call and is the one saved with the result.
func (a *Analyzer) executeReserved(res *JobRunReservation, job models.Job, plan runPlan, injectedProvider ai.AIProvider) (retRun *models.JobRun, retErr error) {
	// The reservation authorizes exactly its own tenant/job: validated before it is consumed and
	// before any activity, provider resolution, source selection or publication.
	if !res.matches(job) {
		return nil, ErrReservationMismatch
	}
	consumed := false
	res.used.Do(func() { consumed = true })
	if !consumed {
		return nil, ErrJobAdmission // a reservation is consumed exactly once
	}
	owner := res.owner
	ctx := owner.ctx
	run := res.run
	committed := false    // the terminal state is stored: a later panic must not re-close the run
	defer owner.release() // runs last: the slot is held until every effect below has finished
	defer func() {
		if r := recover(); r != nil {
			// The panic value is never logged or returned (it may carry SQL/config/credentials).
			log.Printf("[security] panic in job run %s of job %s", run.ID, job.ID)
			if !committed {
				// the same owner decision and checked finalizer as every other terminal path
				if _, err := closeOwnedRun(owner, &run, res.bound(), "Lượt chạy dừng đột ngột; xem nhật ký máy chủ."); err != nil {
					log.Printf("[analyzer] job %s: panic state not recorded; the running row keeps blocking admission", job.ID)
					finishedAt := analyzerNow()
					run.Status = "error"
					run.FinishedAt = &finishedAt
				}
			}
			retRun, retErr = &run, ErrJobRunPanic
		}
	}()
	// CCMAI-RUNTIME-025: the start instant is also the scan start; an ordinary incremental run that
	// completes cleanly records it (second-truncated) as its checkpoint, and it is the test-run
	// window's clock (it must not move while the run works).
	now := run.StartedAt
	scanStart := now

	// Log run started
	db.LogActivity(job.TenantID, "", "system", "job.run.started", "job", job.ID,
		fmt.Sprintf("Job '%s': started analysis (mode=%s, max=%d)", job.Name, plan.mode, plan.limit), "", "")

	// Get AI provider (use injected if provided, otherwise from settings)
	var provider ai.AIProvider
	var err error
	if injectedProvider != nil {
		provider = injectedProvider
	} else if a.providerOverride != nil {
		provider = a.providerOverride
	} else {
		provider, err = a.getProvider(job)
		if err != nil {
			return a.failOwnedRun(owner, &run, job, err.Error(), err)
		}
	}

	// Parse input channel IDs
	var channelIDs []string
	if err := json.Unmarshal([]byte(job.InputChannelIDs), &channelIDs); err != nil {
		return a.failOwnedRun(owner, &run, job, "Danh sách kênh đầu vào của job không hợp lệ", err)
	}

	issuesFound := 0
	passCount := 0
	analyzedCount := 0
	errorCount := 0
	truncated := false

	// Select the conversations and prepare their snapshots.
	var conversations []models.Conversation
	var prepared []preparedConversation
	unchangedCount := 0
	if !plan.explicit() {
		// Every tenant/input-channel conversation is a candidate; the source-version check
		// below, not event timestamps or last_run_at, decides what needs analysis.
		conversations, err = ordinaryIncrementalCandidates(job, channelIDs)
		if err != nil {
			return a.failOwnedRun(owner, &run, job, "Không đọc được danh sách cuộc chat; xem nhật ký máy chủ", err)
		}
		var prepErrors int
		prepared, unchangedCount, prepErrors, truncated = prepareOrdinaryIncremental(ctx, job, conversations)
		errorCount += prepErrors
	} else {
		conversations, err = explicitCandidates(job, channelIDs, plan, now)
		if err != nil {
			return a.failOwnedRun(owner, &run, job, "Không đọc được danh sách cuộc chat; xem nhật ký máy chủ", err)
		}
		var prepErrors int
		prepared, prepErrors, truncated = prepareExplicit(ctx, conversations)
		errorCount += prepErrors
	}

	// found is the number of conversations this run has to handle: those with source to send
	// plus those that could not be prepared (or verified).
	found := len(prepared) + errorCount

	if !plan.explicit() {
		log.Printf("[analyzer] job %s: ordinary incremental, channelIDs=%v, %d candidates, %d changed, %d unchanged, %d errors",
			job.Name, channelIDs, len(conversations), len(prepared), unchangedCount, errorCount)
	} else {
		log.Printf("[analyzer] job %s: mode=%s cap=%d, channelIDs=%v, %d selected, %d with source, %d errors",
			job.Name, plan.mode, plan.limit, channelIDs, len(conversations), len(prepared), errorCount)
	}

	// Set initial total so frontend can show progress immediately
	initialSummary, _ := json.Marshal(map[string]interface{}{
		"conversations_found": found,
	})
	if err := db.DB.Model(&run).Update("summary", string(initialSummary)).Error; err != nil {
		log.Printf("[analyzer] DB update error (initial summary): %v", err)
	}
	if truncated {
		goto complete
	}

	// Check batch mode setting (default: enabled with batch size 5)
	{
		batchMode := true
		batchSize := 5
		var batchSetting models.AppSetting
		if err := db.DB.Where("tenant_id = ? AND setting_key = ?", job.TenantID, "ai_batch_mode").First(&batchSetting).Error; err == nil {
			batchMode = batchSetting.ValuePlain != "false"
		}
		var batchSizeSetting models.AppSetting
		if err := db.DB.Where("tenant_id = ? AND setting_key = ?", job.TenantID, "ai_batch_size").First(&batchSizeSetting).Error; err == nil {
			var n int
			if _, err := fmt.Sscanf(batchSizeSetting.ValuePlain, "%d", &n); err == nil && n > 0 && n <= 30 {
				batchSize = n
			}
		}
		// Safety net: prevent infinite loop if batchSize somehow becomes 0
		if batchSize < 1 {
			batchSize = 5
		}

		if batchMode {
			var bIssues, bPass, bAnalyzed, bErrors int
			bIssues, bPass, bAnalyzed, bErrors, truncated = a.runBatchMode(owner, provider, job, run, prepared, found, batchSize)
			issuesFound += bIssues
			passCount += bPass
			analyzedCount += bAnalyzed
			errorCount += bErrors
		} else {
			// analyzeSingle sends one prepared snapshot and saves its result.
			analyzeSingle := func(conv models.Conversation, snap *conversationSnapshot) {
				transcript := snap.Transcript

				// Build prompt based on job type
				var systemPrompt string
				switch job.JobType {
				case "qc_analysis":
					systemPrompt = ai.BuildQCPrompt(job.RulesContent, job.SkipConditions)
				case "classification":
					systemPrompt = ai.BuildClassificationPrompt(job.RulesConfig)
				default:
					return
				}

				// Call AI (with rate limit delay)
				if analyzedCount > 0 {
					sleepCtx(ctx, 500*time.Millisecond) // Avoid rate limiting
				}
				// No new provider invocation after an accepted cancellation.
				if owner.cancelledNow() {
					return
				}
				aiResp, err := provider.AnalyzeChat(ctx, systemPrompt, transcript)
				if err != nil {
					log.Printf("[analyzer] AI error for conversation %s: %v", conv.ID, err)
					errorCount++
					// Update progress even on error
					errProgressJSON, _ := json.Marshal(map[string]interface{}{
						"conversations_found":    found,
						"conversations_analyzed": analyzedCount,
						"conversations_passed":   passCount,
						"conversations_errors":   errorCount,
						"issues_found":           issuesFound,
					})
					if err := db.DB.Model(&run).Update("summary", string(errProgressJSON)).Error; err != nil {
						log.Printf("[analyzer] DB update error (error progress): %v", err)
					}
					return
				}
				// Log AI usage + cost
				cost, priceKnown := ai.CalculateCost(aiResp.Model, aiResp.InputTokens, aiResp.OutputTokens)
				if !priceKnown {
					log.Printf("[ai] chưa có đơn giá cho model %s (provider=%s), chi phí lượt gọi này không được tính", aiResp.Model, aiResp.Provider)
				}
				usageLog := models.AIUsageLog{
					ID:           pkg.NewUUID(),
					TenantID:     job.TenantID,
					JobID:        job.ID,
					JobRunID:     run.ID,
					Provider:     aiResp.Provider,
					Model:        aiResp.Model,
					InputTokens:  aiResp.InputTokens,
					OutputTokens: aiResp.OutputTokens,
					CostUSD:      cost,
					CreatedAt:    time.Now(),
				}
				db.DB.Create(&usageLog)

				// Parse and save results, unless a cancellation was accepted while the provider ran.
				var count int
				var passed bool
				published, err := owner.publish(func() error {
					var saveErr error
					count, passed, saveErr = a.saveResults(run.ID, snap, job.JobType, aiResp.Content)
					return saveErr
				})
				if !published {
					return
				}
				if err != nil {
					log.Printf("[analyzer] save results error for %s: %v", conv.ID, err)
					errorCount++
					return
				}
				analyzedCount++
				issuesFound += count
				if passed {
					passCount++
				}

				// Update progress so frontend can poll real-time status
				progressJSON, _ := json.Marshal(map[string]interface{}{
					"conversations_found":    found,
					"conversations_analyzed": analyzedCount,
					"conversations_passed":   passCount,
					"conversations_errors":   errorCount,
					"issues_found":           issuesFound,
				})
				if err := db.DB.Model(&run).Update("summary", string(progressJSON)).Error; err != nil {
					log.Printf("[analyzer] DB update error (progress): %v", err)
				}
			}

			for _, p := range prepared {
				if ctx.Err() != nil {
					log.Printf("[analyzer] job %s: context cancelled, stopping after %d/%d conversations", job.Name, analyzedCount, found)
					truncated = true
					break
				}
				analyzeSingle(p.Conv, p.Snap)
			}
		} // end else (non-batch mode)
	}

complete:
	// Complete run. beginTerminal is the serialization point with a cancel request: an accepted
	// cancellation closes the run as cancelled (never success, never a checkpoint); otherwise the
	// terminal commit wins and later cancellation requests are refused.
	cancelled := owner.beginTerminal()
	if ctx.Err() != nil {
		truncated = true // timeout or a cancelled parent: non-success
	}
	if cancelled {
		truncated = true
	}
	finishedAt := analyzerNow()
	summaryJSON, _ := json.Marshal(map[string]interface{}{
		"conversations_found":    found,
		"conversations_analyzed": analyzedCount,
		"conversations_passed":   passCount,
		"conversations_errors":   errorCount,
		"issues_found":           issuesFound,
	})
	runStatus := "success"
	if cancelled {
		runStatus = "cancelled"
		run.ErrorMessage = "Cancelled by user"
	} else if analyzedCount == 0 && errorCount > 0 {
		runStatus = "error"
		run.ErrorMessage = fmt.Sprintf("Analysis errors: %d/%d conversations failed", errorCount, found)
	} else if truncated {
		runStatus = "partial"
		run.ErrorMessage = fmt.Sprintf("Hết thời gian chạy, mới xử lý %d/%d cuộc chat. Phần còn lại vào lần chạy sau.",
			analyzedCount, found)
	} else if errorCount > 0 {
		runStatus = "partial"
		run.ErrorMessage = fmt.Sprintf("Analysis errors: %d/%d conversations failed", errorCount, found)
	}

	// Only the ordinary incremental scan owns last_run_at, and only after a complete, error-free
	// run written together with the terminal run status (CCMAI-RUNTIME-025). Every explicit mode,
	// capped or not, preserves the checkpoint and still records last_run_status/updated_at.
	var checkpoint *time.Time
	if !plan.explicit() && !truncated && errorCount == 0 {
		cp := scanStart.Truncate(time.Second)
		checkpoint = &cp
	}
	finalizeErr := finalizeOrdinaryRun(&run, job, runStatus, run.ErrorMessage, string(summaryJSON), finishedAt, checkpoint)
	if err := finalizeErr; err != nil {
		owner.terminalFailed()
		log.Printf("[analyzer] job %s: %v", job.Name, err)
		run.Status = "error"
		run.FinishedAt = &finishedAt
		run.Summary = string(summaryJSON)
		run.ErrorMessage = "Không ghi nhận được kết quả cuối của lượt chạy; mốc quét giữ nguyên."
		return &run, err
	}

	committed = true
	run.Status = runStatus
	run.FinishedAt = &finishedAt
	run.Summary = string(summaryJSON)

	log.Printf("[analyzer] job %s completed: %d conversations, %d issues", job.Name, found, issuesFound)

	// Log activity
	db.LogActivity(job.TenantID, "", "system", "job.run.completed", "job", job.ID,
		fmt.Sprintf("Job '%s': %d analyzed, %d passed, %d issues, %d errors", job.Name, analyzedCount, passCount, issuesFound, errorCount),
		run.ErrorMessage, "")

	// Send notifications via configured outputs (telegram, email, etc.). A cancelled run sends
	// none (CCMAI-RUNTIME-028); the slot stays held until this returns.
	if analyzedCount > 0 && job.OutputSchedule != "none" && runStatus != "cancelled" {
		if err := sendJobNotifications(ctx, job, run); err != nil {
			log.Printf("[analyzer] notification error for job %s: %v", job.Name, err)
			db.LogActivity(job.TenantID, "", "system", "notification.error", "job", job.ID, "", err.Error(), "")
		}
	}

	return &run, nil
}

func defaultSendJobNotifications(ctx context.Context, job models.Job, run models.JobRun) error {
	return notifications.NewDispatcher().SendJobResults(ctx, job, run)
}

func (a *Analyzer) getProvider(job models.Job) (ai.AIProvider, error) {
	// Get AI provider from tenant settings (fallback to job's ai_provider)
	provider := job.AIProvider
	var providerSetting models.AppSetting
	if err := db.DB.Where("tenant_id = ? AND setting_key = ?", job.TenantID, "ai_provider").First(&providerSetting).Error; err == nil {
		provider = providerSetting.ValuePlain
	}

	// Get API key from tenant settings
	var setting models.AppSetting
	result := db.DB.Where("tenant_id = ? AND setting_key = ?", job.TenantID, "ai_api_key").First(&setting)
	if result.Error != nil {
		return nil, fmt.Errorf("API key not configured - go to Settings > AI Config")
	}

	apiKey := setting.ValuePlain
	if setting.ValueEncrypted != nil && len(setting.ValueEncrypted) > 0 {
		decrypted, err := pkg.Decrypt(setting.ValueEncrypted, a.cfg.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt API key: %w", err)
		}
		apiKey = string(decrypted)
	}

	// Get model from tenant settings (fallback to job's ai_model)
	model := job.AIModel
	var modelSetting models.AppSetting
	if err := db.DB.Where("tenant_id = ? AND setting_key = ?", job.TenantID, "ai_model").First(&modelSetting).Error; err == nil && modelSetting.ValuePlain != "" {
		model = modelSetting.ValuePlain
	}

	// Get base URL from tenant settings (optional)
	var baseURL string
	var baseURLSetting models.AppSetting
	if err := db.DB.Where("tenant_id = ? AND setting_key = ?", job.TenantID, "ai_base_url").First(&baseURLSetting).Error; err == nil {
		baseURL = baseURLSetting.ValuePlain
	}

	switch provider {
	case "claude":
		return ai.NewClaudeProvider(apiKey, model, a.cfg.AIMaxTokens, baseURL), nil
	case "gemini":
		return ai.NewGeminiProvider(apiKey, model, baseURL), nil
	case "openai":
		return ai.NewOpenAIProvider(apiKey, model, a.cfg.AIMaxTokens, baseURL), nil
	case "xai":
		return ai.NewXAIProvider(apiKey, model, a.cfg.AIMaxTokens, baseURL), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", provider)
	}
}

func (a *Analyzer) saveResults(runID string, snap *conversationSnapshot, jobType, aiResponse string) (int, bool, error) {
	if snap == nil {
		return 0, false, fmt.Errorf("analysis snapshot is required")
	}
	tenantID := snap.Manifest.TenantID
	conversationID := snap.Manifest.ConversationID
	now := time.Now()
	count := 0
	passed := false

	// Strip markdown code fences (```json ... ```) that AI sometimes wraps around JSON
	aiResponse = strings.TrimSpace(aiResponse)
	if strings.HasPrefix(aiResponse, "```") {
		// Remove opening fence (```json or ```)
		if idx := strings.Index(aiResponse, "\n"); idx != -1 {
			aiResponse = aiResponse[idx+1:]
		}
		// Remove closing fence
		if idx := strings.LastIndex(aiResponse, "```"); idx != -1 {
			aiResponse = aiResponse[:idx]
		}
		aiResponse = strings.TrimSpace(aiResponse)
	}
	if err := validateAIResult(jobType, []byte(aiResponse)); err != nil {
		return 0, false, err
	}

	var qcResult struct {
		Verdict    string `json:"verdict"`
		Violations []struct {
			Severity     string        `json:"severity"`
			Rule         string        `json:"rule"`
			Evidence     string        `json:"evidence"`
			EvidenceRefs []evidenceRef `json:"evidence_refs"`
			Explanation  string        `json:"explanation"`
			Suggestion   string        `json:"suggestion"`
		} `json:"violations"`
		Score   int    `json:"score"`
		Review  string `json:"review"`
		Summary string `json:"summary"`
	}
	var classResult struct {
		Tags []struct {
			RuleName     string        `json:"rule_name"`
			Confidence   float64       `json:"confidence"`
			Evidence     string        `json:"evidence"`
			EvidenceRefs []evidenceRef `json:"evidence_refs"`
			Explanation  string        `json:"explanation"`
		} `json:"tags"`
		Summary string `json:"summary"`
	}

	// Every finding must cite the snapshot before anything is written, so one
	// bad ref rejects the whole conversation result.
	switch jobType {
	case "qc_analysis":
		if err := json.Unmarshal([]byte(aiResponse), &qcResult); err != nil {
			return 0, false, fmt.Errorf("failed to parse QC response: %w", err)
		}
		for i, v := range qcResult.Violations {
			if err := snap.validateEvidenceRefs(v.EvidenceRefs); err != nil {
				return 0, false, fmt.Errorf("violation %d: %w", i, err)
			}
		}
	case "classification":
		if err := json.Unmarshal([]byte(aiResponse), &classResult); err != nil {
			return 0, false, fmt.Errorf("failed to parse classification response: %w", err)
		}
		for i, t := range classResult.Tags {
			if err := snap.validateEvidenceRefs(t.EvidenceRefs); err != nil {
				return 0, false, fmt.Errorf("tag %d: %w", i, err)
			}
		}
	}

	snapshotRow, err := snap.record(runID)
	if err != nil {
		return 0, false, fmt.Errorf("prepare analysis snapshot: %w", err)
	}
	snapshotID := snapshotRow.ID

	tx := db.DB.Begin()
	if tx.Error != nil {
		return 0, false, tx.Error
	}
	defer tx.Rollback()

	// R2-RR3: lock and confirm the parents are still there before writing any
	// evidence. This FOR UPDATE read on the same Conversation/JobRun rows that
	// every parent-deletion path (DeleteChannel, PurgeChannelConversations,
	// DeleteJob, ClearJobRuns, demo reset) now locks first is what closes the
	// writer/deletion race: whichever side gets there first is waited out by the
	// other, so a snapshot/result can never commit referencing a conversation or
	// job run that deletion has already removed (or is mid-removing).
	var parentConv models.Conversation
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&parentConv).Error; err != nil {
		return 0, false, fmt.Errorf("conversation %s is gone, evidence not saved: %w", conversationID, err)
	}
	var parentRun models.JobRun
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND tenant_id = ?", runID, tenantID).First(&parentRun).Error; err != nil {
		return 0, false, fmt.Errorf("job run %s is gone, evidence not saved: %w", runID, err)
	}

	if err := tx.Create(&snapshotRow).Error; err != nil {
		return 0, false, fmt.Errorf("persist analysis snapshot: %w", err)
	}

	// QC evaluations/violations and classification evaluations have no
	// measured confidence; store an explicit unknown rather than a placeholder.
	noConfidence, unavailableBasis := models.UnavailableConfidence()

	switch jobType {
	case "qc_analysis":
		// Determine pass/fail (SKIP counts as not passed)
		passed = qcResult.Verdict == "PASS"

		// Save conversation evaluation record (for every conversation)
		evalDetailJSON, _ := json.Marshal(map[string]interface{}{
			"review":  qcResult.Review,
			"score":   qcResult.Score,
			"summary": qcResult.Summary,
		})
		evalResult := models.JobResult{
			ID:                 pkg.NewUUID(),
			JobRunID:           runID,
			TenantID:           tenantID,
			ConversationID:     conversationID,
			AnalysisSnapshotID: &snapshotID,
			ResultType:         "conversation_evaluation",
			Severity:           qcResult.Verdict,
			Evidence:           qcResult.Review,
			Detail:             string(evalDetailJSON),
			AIRawResponse:      aiResponse,
			Confidence:         noConfidence,
			ConfidenceBasis:    unavailableBasis,
			CreatedAt:          now,
		}
		if err := tx.Create(&evalResult).Error; err != nil {
			return 0, false, err
		}

		// SKIP conversations have no violations — stop here
		if qcResult.Verdict == "SKIP" {
			return 0, false, tx.Commit().Error
		}

		// Save individual violations
		for _, v := range qcResult.Violations {
			detailJSON, _ := json.Marshal(map[string]interface{}{
				"explanation":   v.Explanation,
				"suggestion":    v.Suggestion,
				"score":         qcResult.Score,
				"summary":       qcResult.Summary,
				"evidence_refs": v.EvidenceRefs,
			})
			result := models.JobResult{
				ID:                 pkg.NewUUID(),
				JobRunID:           runID,
				TenantID:           tenantID,
				ConversationID:     conversationID,
				AnalysisSnapshotID: &snapshotID,
				ResultType:         "qc_violation",
				Severity:           v.Severity,
				RuleName:           v.Rule,
				Evidence:           v.Evidence,
				Detail:             string(detailJSON),
				AIRawResponse:      aiResponse,
				Confidence:         noConfidence,
				ConfidenceBasis:    unavailableBasis,
				CreatedAt:          now,
			}
			if err := tx.Create(&result).Error; err != nil {
				return 0, false, err
			}
			count++
		}

	case "classification":
		for _, t := range classResult.Tags {
			// validateAIResult already rejected a missing or out-of-[0,1] value;
			// what remains is the model's own uncalibrated estimate.
			tagConfidence, tagBasis := models.ModelReportedConfidence(t.Confidence)
			detailJSON, _ := json.Marshal(map[string]interface{}{
				"explanation":   t.Explanation,
				"summary":       classResult.Summary,
				"evidence_refs": t.EvidenceRefs,
			})
			result := models.JobResult{
				ID:                 pkg.NewUUID(),
				JobRunID:           runID,
				TenantID:           tenantID,
				ConversationID:     conversationID,
				AnalysisSnapshotID: &snapshotID,
				ResultType:         "classification_tag",
				RuleName:           t.RuleName,
				Evidence:           t.Evidence,
				Detail:             string(detailJSON),
				AIRawResponse:      aiResponse,
				Confidence:         tagConfidence,
				ConfidenceBasis:    tagBasis,
				CreatedAt:          now,
			}
			if err := tx.Create(&result).Error; err != nil {
				return 0, false, err
			}
			count++
		}

		// Create conversation_evaluation record for classified conversations
		if len(classResult.Tags) > 0 {
			evalDetail, _ := json.Marshal(map[string]interface{}{
				"summary": classResult.Summary,
			})
			if err := tx.Create(&models.JobResult{
				ID:                 pkg.NewUUID(),
				JobRunID:           runID,
				TenantID:           tenantID,
				ConversationID:     conversationID,
				AnalysisSnapshotID: &snapshotID,
				ResultType:         "conversation_evaluation",
				Severity:           "PASS",
				Evidence:           classResult.Summary,
				Detail:             string(evalDetail),
				AIRawResponse:      aiResponse,
				Confidence:         noConfidence,
				ConfidenceBasis:    unavailableBasis,
				CreatedAt:          now,
			}).Error; err != nil {
				return 0, false, err
			}
		}

		// No tags matched — mark conversation as SKIP
		if len(classResult.Tags) == 0 {
			skipDetail, _ := json.Marshal(map[string]interface{}{
				"summary": classResult.Summary,
			})
			if err := tx.Create(&models.JobResult{
				ID:                 pkg.NewUUID(),
				JobRunID:           runID,
				TenantID:           tenantID,
				ConversationID:     conversationID,
				AnalysisSnapshotID: &snapshotID,
				ResultType:         "conversation_evaluation",
				Severity:           "SKIP",
				Evidence:           "Cuộc chat không khớp với bất kỳ nhãn phân loại nào.",
				Detail:             string(skipDetail),
				AIRawResponse:      aiResponse,
				Confidence:         noConfidence,
				ConfidenceBasis:    unavailableBasis,
				CreatedAt:          now,
			}).Error; err != nil {
				return 0, false, err
			}
		}
	}

	return count, passed, tx.Commit().Error
}

// runBatchMode processes the prepared conversations in batches of batchSize, sending multiple
// conversations per AI call. The snapshots were prepared before this call (ordinary and explicit
// modes alike) and are sent and saved unchanged.
func (a *Analyzer) runBatchMode(owner *JobRunOwner, provider ai.AIProvider, job models.Job, run models.JobRun, prepared []preparedConversation, found, batchSize int) (issuesFound, passCount, analyzedCount, errorCount int, cancelled bool) {
	ctx := owner.ctx
	// Build system prompt once
	var systemPrompt string
	switch job.JobType {
	case "qc_analysis":
		systemPrompt = ai.BuildQCPrompt(job.RulesContent, job.SkipConditions)
	case "classification":
		systemPrompt = ai.BuildClassificationPrompt(job.RulesConfig)
	default:
		return
	}

	// Process in batches
	consecutiveErrors := 0
	for i := 0; i < len(prepared); i += batchSize {
		batchHadError := false
		// Check if context cancelled
		select {
		case <-ctx.Done():
			log.Printf("[analyzer-batch] job %s: context cancelled, stopping after %d/%d conversations", job.Name, analyzedCount, found)
			cancelled = true
			return
		default:
		}

		end := i + batchSize
		if end > len(prepared) {
			end = len(prepared)
		}
		batch := prepared[i:end]

		// Build batch items
		items := make([]ai.BatchItem, len(batch))
		for j, b := range batch {
			items[j] = ai.BatchItem{
				ConversationID: b.Conv.ID,
				Transcript:     b.Snap.Transcript,
			}
		}

		// No new provider invocation after an accepted cancellation.
		if owner.cancelledNow() {
			cancelled = true
			return
		}
		// Call AI batch
		aiResp, err := provider.AnalyzeChatBatch(ctx, systemPrompt, items)
		if err != nil {
			log.Printf("[analyzer-batch] AI error for batch starting at %d: %v", i, err)
			errorCount += len(batch)
			batchHadError = true
			continue
		}

		// Log AI usage
		cost, priceKnown := ai.CalculateCost(aiResp.Model, aiResp.InputTokens, aiResp.OutputTokens)
		if !priceKnown {
			log.Printf("[ai] chưa có đơn giá cho model %s (provider=%s), chi phí lượt gọi này không được tính", aiResp.Model, aiResp.Provider)
		}
		usageLog := models.AIUsageLog{
			ID:           pkg.NewUUID(),
			TenantID:     job.TenantID,
			JobID:        job.ID,
			JobRunID:     run.ID,
			Provider:     aiResp.Provider,
			Model:        aiResp.Model,
			InputTokens:  aiResp.InputTokens,
			OutputTokens: aiResp.OutputTokens,
			CostUSD:      cost,
			CreatedAt:    time.Now(),
		}
		db.DB.Create(&usageLog)

		// Parse batch response — expect JSON array
		content := strings.TrimSpace(aiResp.Content)
		if strings.HasPrefix(content, "```") {
			if idx := strings.Index(content, "\n"); idx != -1 {
				content = content[idx+1:]
			}
			if idx := strings.LastIndex(content, "```"); idx != -1 {
				content = content[:idx]
			}
			content = strings.TrimSpace(content)
		}

		expectedIDs := make([]string, len(batch))
		for j, b := range batch {
			expectedIDs[j] = b.Conv.ID
		}
		batchResults, parseErr := parseBatchResults(expectedIDs, []byte(content))
		if parseErr != nil {
			log.Printf("[analyzer-batch] rejected batch starting at %d: %v", i, parseErr)
			errorCount += len(batch)
			batchHadError = true
		} else {
			// Process each result
			for j, rawResult := range batchResults {
				convID := batch[j].Conv.ID
				var count int
				var passed bool
				published, saveErr := owner.publish(func() error {
					var e error
					count, passed, e = a.saveResults(run.ID, batch[j].Snap, job.JobType, string(rawResult))
					return e
				})
				if !published {
					// cancellation accepted while the provider ran: nothing further is published
					cancelled = true
					return
				}
				if saveErr != nil {
					log.Printf("[analyzer-batch] save error for %s: %v", convID, saveErr)
					errorCount++
					batchHadError = true
				} else {
					analyzedCount++
					issuesFound += count
					if passed {
						passCount++
					}
				}
			}
		}

		// Update progress
		progressJSON, _ := json.Marshal(map[string]interface{}{
			"conversations_found":    found,
			"conversations_analyzed": analyzedCount,
			"conversations_passed":   passCount,
			"conversations_errors":   errorCount,
			"issues_found":           issuesFound,
		})
		if err := db.DB.Model(&run).Update("summary", string(progressJSON)).Error; err != nil {
			log.Printf("[analyzer-batch] DB update error (progress): %v", err)
		}

		// Adaptive rate limit between batches
		if end < len(prepared) {
			if batchHadError {
				consecutiveErrors++
				if consecutiveErrors >= 3 {
					sleepCtx(ctx, 30*time.Second)
				} else {
					sleepCtx(ctx, 10*time.Second)
				}
			} else {
				consecutiveErrors = 0
				sleepCtx(ctx, 2*time.Second)
			}
		}
	}

	log.Printf("[analyzer-batch] job %s: %d conversations in %d batches of %d", job.Name, len(prepared), (len(prepared)+batchSize-1)/batchSize, batchSize)
	return
}

// failOwnedRun closes a reserved run that failed before any analysis (provider selection, input
// parsing, candidate selection) through the shared terminal path (closeOwnedRun): an accepted
// cancellation wins and the run closes cancelled, otherwise it closes error. publicMsg is stored and
// shown; cause is logged only. When the terminal write cannot be recorded the running row stays and
// keeps blocking admission (fail closed).
func (a *Analyzer) failOwnedRun(owner *JobRunOwner, run *models.JobRun, job models.Job, publicMsg string, cause error) (*models.JobRun, error) {
	log.Printf("[analyzer] job %s: run %s failed before analysis: %v", job.ID, run.ID, cause)
	status, err := closeOwnedRun(owner, run, job, publicMsg)
	if err != nil {
		log.Printf("[analyzer] job %s: early failure not recorded: %v", job.ID, err)
		return run, err
	}
	if status == "cancelled" {
		return run, nil // the accepted cancellation, not the failure, is the outcome
	}
	return run, errors.New(publicMsg)
}

// sleepCtx waits for d or until ctx is done.
func sleepCtx(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
