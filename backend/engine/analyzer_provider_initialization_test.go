package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-050 (LP-01..08) & CCMAI-RUNTIME-051 (RP-01..04):
// source-first application provider initialization and test repair.
// The Analyzer delays provider settings/key resolution and provider construction until
// candidate snapshots have been prepared and eligible work requiring inference remains.

// countingProvider wraps an incProvider and tracks single inference calls, batch inference calls,
// and processed items distinctly (RP-02 / R050-R1-02).
type countingProvider struct {
	*incProvider
	singleCalls int64
	batchCalls  int64
	itemsCount  int64
	chatCalls   int64
}

func newCountingProvider(jobType string) *countingProvider {
	return &countingProvider{
		incProvider: &incProvider{jobType: jobType, verdict: "PASS"},
	}
}

func (cp *countingProvider) AnalyzeChat(ctx context.Context, systemPrompt, transcript string) (ai.AIResponse, error) {
	atomic.AddInt64(&cp.singleCalls, 1)
	atomic.AddInt64(&cp.itemsCount, 1)
	atomic.AddInt64(&cp.chatCalls, 1)
	return cp.incProvider.AnalyzeChat(ctx, systemPrompt, transcript)
}

func (cp *countingProvider) AnalyzeChatBatch(ctx context.Context, systemPrompt string, items []ai.BatchItem) (ai.AIResponse, error) {
	atomic.AddInt64(&cp.batchCalls, 1)
	atomic.AddInt64(&cp.itemsCount, int64(len(items)))
	atomic.AddInt64(&cp.chatCalls, 1)
	return cp.incProvider.AnalyzeChatBatch(ctx, systemPrompt, items)
}

func (cp *countingProvider) totalInferenceCalls() int64 {
	return atomic.LoadInt64(&cp.singleCalls) + atomic.LoadInt64(&cp.batchCalls)
}

// trapAISettingsQueries registers a GORM query callback that counts queries targeting app_settings
// for AI settings (ai_provider, ai_api_key, ai_model, etc.) during getProvider resolution.
// It returns a pointer to an atomic int64 query counter and unregisters on test cleanup.
func trapAISettingsQueries(t *testing.T) *int64 {
	t.Helper()
	var queriesCount int64
	callbackName := "trap_ai_settings_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
	err := db.DB.Callback().Query().After("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		sql := tx.Statement.SQL.String()
		vars := fmt.Sprintf("%v", tx.Statement.Vars)
		if tx.Statement.Table == "app_settings" || strings.Contains(sql, "app_settings") {
			if strings.Contains(sql, "ai_") || strings.Contains(vars, "ai_") {
				atomic.AddInt64(&queriesCount, 1)
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.DB.Callback().Query().Remove(callbackName)
	})
	return &queriesCount
}

// trapJobNotifications registers a test-local dispatcher for sendJobNotifications that records
// notification dispatch count and arguments, restoring the original function on test cleanup.
func trapJobNotifications(t *testing.T) *notifier {
	t.Helper()
	orig := sendJobNotifications
	nt := &notifier{}
	sendJobNotifications = func(_ context.Context, job models.Job, run models.JobRun) error {
		nt.mu.Lock()
		nt.n++
		hook := nt.onSend
		nt.mu.Unlock()
		if hook != nil {
			hook(job, run)
		}
		return nil
	}
	t.Cleanup(func() {
		sendJobNotifications = orig
	})
	return nt
}

// LP-01: Empty channel list, no conversations, no messages and all-unchanged ordinary
// snapshots finish through the existing truthful no-work path without provider-setting/key
// lookup or decryption. Missing/invalid AI config must not mask an empty-source outcome.
func TestLP01NoWorkSkipsProviderInitialization(t *testing.T) {
	cases := []struct {
		name         string
		mode         string
		setupSource  func(t *testing.T, f *incFixture)
		expectCPMove bool
	}{
		{
			name: "empty input channel list",
			mode: "ordinary",
			setupSource: func(t *testing.T, f *incFixture) {
				f.exec(t, `UPDATE jobs SET input_channel_ids = '[]' WHERE id = ?`, f.jobID)
			},
			expectCPMove: true,
		},
		{
			name: "channel with zero conversations",
			mode: "ordinary",
			setupSource: func(t *testing.T, f *incFixture) {
				// zero conversations in channel
			},
			expectCPMove: true,
		},
		{
			name: "conversation with zero messages (coverageEmpty)",
			mode: "ordinary",
			setupSource: func(t *testing.T, f *incFixture) {
				f.addConv(t, f.tenantID, f.channelID, "empty", nil)
			},
			expectCPMove: true,
		},
		{
			name: "all conversations unchanged in ordinary mode",
			mode: "ordinary",
			setupSource: func(t *testing.T, f *incFixture) {
				// First run analyzes one conversation and establishes committed baseline
				f.addConv(t, f.tenantID, f.channelID, "init", []time.Time{f.clock.Add(-time.Hour)})
				p := &incProvider{jobType: "qc_analysis", verdict: "PASS"}
				run1 := f.mustRun(t, p)
				if run1.Status != "success" {
					t.Fatalf("setup run failed: %v", run1.ErrorMessage)
				}
				// Verify one evaluation committed
				if evs := evaluatedIn(t, run1.ID); len(evs) != 1 {
					t.Fatalf("expected 1 evaluation in setup run, got %d", len(evs))
				}
			},
			expectCPMove: true,
		},
		{
			name: "explicit unanalyzed mode with zero conversations",
			mode: "unanalyzed",
			setupSource: func(t *testing.T, f *incFixture) {
				// zero conversations
			},
			expectCPMove: false, // explicit modes preserve checkpoint
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			f := setupIncFixture(t, false, "qc_analysis")
			c.setupSource(t, f)

			// Deliberately DO NOT configure any ai_api_key in app_settings.
			// Furthermore, insert a corrupt encrypted key that would error if pkg.Decrypt were called.
			f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, value_encrypted, created_at, updated_at) VALUES (?, ?, 'ai_api_key', '', X'DEADBEEF', NOW(), NOW())`,
				pkg.NewUUID(), f.tenantID)

			queries := trapAISettingsQueries(t)
			nt := trapJobNotifications(t)
			// Exercise the actual production getProvider path without resolver or override:
			// under eager init, getProvider would query ai_api_key, attempt pkg.Decrypt(X'DEADBEEF'),
			// fail with decryption error and fail the run. Under lazy init, getProvider is bypassed.
			analyzer := NewAnalyzer(&config.Config{EncryptionKey: "01234567890123456789012345678901"})

			initialCP := f.checkpoint(t)

			var run *models.JobRun
			var err error
			if c.mode == "ordinary" {
				run, err = analyzer.RunJob(context.Background(), f.job(t))
			} else {
				run, err = analyzer.RunJobUnanalyzed(context.Background(), f.job(t), 0)
			}

			if err != nil {
				t.Fatalf("expected no-work run to succeed cleanly without provider, got error: %v", err)
			}
			if run == nil || run.Status != "success" {
				t.Fatalf("expected success status, got %v (%+v)", run, err)
			}

			// Verify notifications were never sent on no-work run
			if calls := nt.count(); calls != 0 {
				t.Fatalf("expected 0 job notifications on no-work run, got %d", calls)
			}

			// Verify AI settings queries were never issued and provider resolution was never attempted
			if calls := atomic.LoadInt64(queries); calls != 0 {
				t.Fatalf("expected 0 AI settings queries on no-work run, got %d", calls)
			}

			// Reload stored run from DB to observe persisted state directly
			var storedRun models.JobRun
			if err := db.DB.Where("id = ?", run.ID).First(&storedRun).Error; err != nil {
				t.Fatalf("reload stored run %s: %v", run.ID, err)
			}
			if storedRun.Status != "success" {
				t.Fatalf("expected stored run status success, got %q", storedRun.Status)
			}
			if storedRun.FinishedAt == nil {
				t.Fatal("expected stored run finished_at to be set on clean completion")
			}
			if JobRunActive(f.tenantID, f.jobID) {
				t.Fatal("expected job run slot ownership to be released on no-work completion")
			}

			// Reload stored job to observe persisted run tracking
			var storedJob models.Job
			if err := db.DB.Where("id = ?", f.jobID).First(&storedJob).Error; err != nil {
				t.Fatalf("reload stored job %s: %v", f.jobID, err)
			}
			if c.expectCPMove && storedJob.LastRunAt == nil {
				t.Fatal("expected stored job last_run_at to be updated on ordinary clean completion")
			}

			// Verify stored summary reflects 0 work
			var summary map[string]interface{}
			if err := json.Unmarshal([]byte(storedRun.Summary), &summary); err != nil {
				t.Fatalf("parse stored summary: %v", err)
			}
			if found, _ := summary["conversations_found"].(float64); found != 0 {
				t.Fatalf("expected conversations_found=0, got %v", found)
			}
			if analyzed, _ := summary["conversations_analyzed"].(float64); analyzed != 0 {
				t.Fatalf("expected conversations_analyzed=0, got %v", analyzed)
			}

			// Verify zero AI usage logs, failing immediately on observational query error
			var usageCount int64
			if err := db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&usageCount).Error; err != nil {
				t.Fatalf("count AI usage logs: %v", err)
			}
			if usageCount != 0 {
				t.Fatalf("expected 0 AI usage logs, got %d", usageCount)
			}

			// Verify zero job results, failing immediately on observational query error
			var resultCount int64
			if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", run.ID).Count(&resultCount).Error; err != nil {
				t.Fatalf("count job results: %v", err)
			}
			if resultCount != 0 {
				t.Fatalf("expected 0 job results, got %d", resultCount)
			}

			// Checkpoint verification
			newCP := f.checkpoint(t)
			if c.expectCPMove {
				if newCP == nil {
					t.Fatal("expected checkpoint to advance on clean ordinary run, got nil")
				}
				if initialCP != nil && newCP.Equal(*initialCP) {
					t.Fatalf("expected checkpoint to advance past initial %v, got %v", initialCP, newCP)
				}
			} else {
				if initialCP == nil && newCP != nil {
					t.Fatalf("explicit mode should not create checkpoint, got %v", newCP)
				}
				if initialCP != nil && (newCP == nil || !newCP.Equal(*initialCP)) {
					t.Fatalf("explicit mode should preserve checkpoint %v, got %v", initialCP, newCP)
				}
			}
		})
	}
}

// LP-02: Invalid channel JSON and candidate/preparation failures remain errors or partial;
// all-failed preparation must never become success. No provider initialization when no
// prepared snapshot remains. Mixed valid/failed preparation processes valid source and
// retains error counters/terminal/checkpoint rules.
func TestLP02CandidateAndPreparationFailuresDoNotInitializeProvider(t *testing.T) {
	t.Run("invalid channel JSON fails before candidate selection and provider", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.exec(t, `UPDATE jobs SET input_channel_ids = '{"not":"an_array"}' WHERE id = ?`, f.jobID)
		// Insert corrupt encrypted key: must not be touched
		f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, value_encrypted, created_at, updated_at) VALUES (?, ?, 'ai_api_key', '', X'DEADBEEF', NOW(), NOW())`,
			pkg.NewUUID(), f.tenantID)

		queries := trapAISettingsQueries(t)
		nt := trapJobNotifications(t)
		analyzer := NewAnalyzer(&config.Config{EncryptionKey: "01234567890123456789012345678901"})

		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err == nil || run == nil {
			t.Fatalf("expected error on invalid channel JSON, got run=%v, err=%v", run, err)
		}
		if run.Status != "error" {
			t.Fatalf("expected error status, got %q", run.Status)
		}
		if calls := nt.count(); calls != 0 {
			t.Fatalf("expected 0 notifications on invalid input, got %d", calls)
		}
		if calls := atomic.LoadInt64(queries); calls != 0 {
			t.Fatalf("expected 0 AI settings queries on invalid input, got %d", calls)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership must be released on invalid input")
		}
		var storedRun models.JobRun
		if err := db.DB.Where("id = ?", run.ID).First(&storedRun).Error; err != nil {
			t.Fatalf("reload stored run %s: %v", run.ID, err)
		}
		if storedRun.Status != "error" || storedRun.FinishedAt == nil {
			t.Fatalf("expected stored run error/finished, got %+v", storedRun)
		}
		var usageCount int64
		if err := db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&usageCount).Error; err != nil {
			t.Fatalf("count usage logs: %v", err)
		}
		if usageCount != 0 {
			t.Fatalf("expected 0 usage logs, got %d", usageCount)
		}
		var resultCount int64
		if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", run.ID).Count(&resultCount).Error; err != nil {
			t.Fatalf("count results: %v", err)
		}
		if resultCount != 0 {
			t.Fatalf("expected 0 results, got %d", resultCount)
		}
		if f.checkpoint(t) != nil {
			t.Fatal("failed run must not set checkpoint")
		}
	})

	t.Run("candidate selection query failure fails before provider and skips settings/key lookup", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "cand-err", []time.Time{f.clock.Add(-time.Hour)})
		f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, value_encrypted, created_at, updated_at) VALUES (?, ?, 'ai_api_key', '', X'DEADBEEF', NOW(), NOW())`,
			pkg.NewUUID(), f.tenantID)

		queries := trapAISettingsQueries(t)
		nt := trapJobNotifications(t)
		failTable(t, "conversations", "SYNTHETIC_CANDIDATE_QUERY_FAIL")

		analyzer := NewAnalyzer(&config.Config{EncryptionKey: "01234567890123456789012345678901"})
		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err == nil || run == nil {
			t.Fatalf("expected error on candidate query failure, got run=%v, err=%v", run, err)
		}
		if run.Status != "error" {
			t.Fatalf("expected error status, got %q", run.Status)
		}
		if calls := nt.count(); calls != 0 {
			t.Fatalf("expected 0 notifications on candidate query failure, got %d", calls)
		}
		if calls := atomic.LoadInt64(queries); calls != 0 {
			t.Fatalf("expected 0 AI settings queries on candidate query failure, got %d", calls)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership must be released on candidate query failure")
		}
		var storedRun models.JobRun
		if err := db.DB.Where("id = ?", run.ID).First(&storedRun).Error; err != nil {
			t.Fatalf("reload stored run %s: %v", run.ID, err)
		}
		if storedRun.Status != "error" || storedRun.FinishedAt == nil {
			t.Fatalf("expected stored run error/finished, got %+v", storedRun)
		}
		var usageCount int64
		if err := db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&usageCount).Error; err != nil {
			t.Fatalf("count usage logs: %v", err)
		}
		if usageCount != 0 {
			t.Fatalf("expected 0 usage logs, got %d", usageCount)
		}
		var resultCount int64
		if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", run.ID).Count(&resultCount).Error; err != nil {
			t.Fatalf("count results: %v", err)
		}
		if resultCount != 0 {
			t.Fatalf("expected 0 results, got %d", resultCount)
		}
		if f.checkpoint(t) != nil {
			t.Fatal("candidate failure must not set checkpoint")
		}
	})

	t.Run("all-failed preparation closes as error with zero provider initialization", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "bad", []time.Time{f.clock.Add(-time.Hour)})
		f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, value_encrypted, created_at, updated_at) VALUES (?, ?, 'ai_api_key', '', X'DEADBEEF', NOW(), NOW())`,
			pkg.NewUUID(), f.tenantID)

		// Make snapshot loading fail by inducing a query error on the messages table
		failTable(t, "messages", "SYNTHETIC_MSG_QUERY_FAIL")

		queries := trapAISettingsQueries(t)
		nt := trapJobNotifications(t)
		analyzer := NewAnalyzer(&config.Config{EncryptionKey: "01234567890123456789012345678901"})

		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err != nil {
			// RunJob returns nil err when run completes through ordinary terminal path with error status
		}
		if run == nil || run.Status != "error" {
			t.Fatalf("all-failed preparation must close as error, got status=%v", run)
		}
		if calls := nt.count(); calls != 0 {
			t.Fatalf("expected 0 notifications on all-failed prep, got %d", calls)
		}
		if calls := atomic.LoadInt64(queries); calls != 0 {
			t.Fatalf("expected 0 AI settings queries on all-failed prep, got %d", calls)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership must be released on all-failed prep")
		}
		var storedRun models.JobRun
		if err := db.DB.Where("id = ?", run.ID).First(&storedRun).Error; err != nil {
			t.Fatalf("reload stored run %s: %v", run.ID, err)
		}
		if storedRun.Status != "error" || storedRun.FinishedAt == nil {
			t.Fatalf("expected stored run error/finished, got %+v", storedRun)
		}
		var usageCount int64
		if err := db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&usageCount).Error; err != nil {
			t.Fatalf("count usage logs: %v", err)
		}
		if usageCount != 0 {
			t.Fatalf("expected 0 usage logs, got %d", usageCount)
		}
		var resultCount int64
		if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", run.ID).Count(&resultCount).Error; err != nil {
			t.Fatalf("count results: %v", err)
		}
		if resultCount != 0 {
			t.Fatalf("expected 0 results, got %d", resultCount)
		}
		if f.checkpoint(t) != nil {
			t.Fatal("all-failed preparation must not advance checkpoint")
		}
	})

	t.Run("mixed valid and failed preparation initializes provider once and reports partial", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "valid", []time.Time{f.clock.Add(-time.Hour)})
		corruptID := f.addConv(t, f.tenantID, f.channelID, "corrupt", []time.Time{f.clock.Add(-30 * time.Minute)})

		// Insert an evaluation with missing linked snapshot so sourceVersionChanged fails with ErrSourceVersionUnverifiable
		dummyRunID := pkg.NewUUID()
		f.exec(t, `INSERT INTO job_runs (id, tenant_id, job_id, status, summary, started_at, created_at) VALUES (?, ?, ?, 'success', '{}', NOW(), NOW())`, dummyRunID, f.tenantID, f.jobID)
		f.exec(t, `INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, analysis_snapshot_id, result_type, severity, rule_name, evidence, detail, ai_raw_response, created_at) VALUES (?, ?, ?, ?, ?, 'conversation_evaluation', 'PASS', '', '', '{}', '{}', NOW())`,
			pkg.NewUUID(), dummyRunID, f.tenantID, corruptID, pkg.NewUUID())

		var resolverCalls int64
		cp := newCountingProvider("qc_analysis")
		nt := trapJobNotifications(t)
		analyzer := NewAnalyzer(&config.Config{})
		analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
			atomic.AddInt64(&resolverCalls, 1)
			return cp, nil
		}

		f.exec(t, "UPDATE jobs SET output_schedule = 'instant' WHERE id = ?", f.jobID)

		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if run.Status != "partial" {
			t.Fatalf("expected partial status for mixed prep outcome, got %q (%s)", run.Status, run.ErrorMessage)
		}
		// Provider initialized exactly once for the valid conversation
		if calls := atomic.LoadInt64(&resolverCalls); calls != 1 {
			t.Fatalf("expected exactly 1 provider resolver call, got %d", calls)
		}
		if singleCalls := atomic.LoadInt64(&cp.singleCalls); singleCalls != 1 {
			t.Fatalf("expected exactly 1 AnalyzeChat call for valid conversation, got %d", singleCalls)
		}
		if items := atomic.LoadInt64(&cp.itemsCount); items != 1 {
			t.Fatalf("expected exactly 1 item processed, got %d", items)
		}
		if calls := nt.count(); calls != 1 {
			t.Fatalf("expected exactly 1 notification on partial run with analyzed work, got %d", calls)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership must be released on partial run")
		}
		var storedRun models.JobRun
		if err := db.DB.Where("id = ?", run.ID).First(&storedRun).Error; err != nil {
			t.Fatalf("reload stored run %s: %v", run.ID, err)
		}
		if storedRun.Status != "partial" || storedRun.FinishedAt == nil {
			t.Fatalf("expected stored run partial/finished, got %+v", storedRun)
		}
		var resultCount int64
		if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", run.ID).Count(&resultCount).Error; err != nil {
			t.Fatalf("count results: %v", err)
		}
		if resultCount != 1 {
			t.Fatalf("expected 1 job result for valid conversation, got %d", resultCount)
		}
		// Checkpoint must NOT advance on partial run
		if f.checkpoint(t) != nil {
			t.Fatal("partial run must not set checkpoint")
		}
	})

	t.Run("mixed valid and failed preparation with corrupt key on production path fails at provider initialization", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "valid", []time.Time{f.clock.Add(-time.Hour)})
		corruptID := f.addConv(t, f.tenantID, f.channelID, "corrupt", []time.Time{f.clock.Add(-30 * time.Minute)})

		dummyRunID := pkg.NewUUID()
		f.exec(t, `INSERT INTO job_runs (id, tenant_id, job_id, status, summary, started_at, created_at) VALUES (?, ?, ?, 'success', '{}', NOW(), NOW())`, dummyRunID, f.tenantID, f.jobID)
		f.exec(t, `INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, analysis_snapshot_id, result_type, severity, rule_name, evidence, detail, ai_raw_response, created_at) VALUES (?, ?, ?, ?, ?, 'conversation_evaluation', 'PASS', '', '', '{}', '{}', NOW())`,
			pkg.NewUUID(), dummyRunID, f.tenantID, corruptID, pkg.NewUUID())

		// Insert corrupt encrypted key: because valid preparation remains, provider initialization IS reached and must fail
		f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, value_encrypted, created_at, updated_at) VALUES (?, ?, 'ai_api_key', '', X'DEADBEEF', NOW(), NOW())`,
			pkg.NewUUID(), f.tenantID)

		queries := trapAISettingsQueries(t)
		nt := trapJobNotifications(t)
		analyzer := NewAnalyzer(&config.Config{EncryptionKey: "01234567890123456789012345678901"})

		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err == nil || run == nil || run.Status != "error" {
			t.Fatalf("expected error on corrupt key with valid preparation, got run=%v, err=%v", run, err)
		}
		if calls := nt.count(); calls != 0 {
			t.Fatalf("expected 0 notifications on provider failure, got %d", calls)
		}
		if calls := atomic.LoadInt64(queries); calls == 0 {
			t.Fatalf("expected AI settings query attempted on valid work, got %d", calls)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership must be released on provider failure")
		}
		var storedRun models.JobRun
		if err := db.DB.Where("id = ?", run.ID).First(&storedRun).Error; err != nil {
			t.Fatalf("reload stored run %s: %v", run.ID, err)
		}
		if storedRun.Status != "error" || storedRun.FinishedAt == nil {
			t.Fatalf("expected stored run error/finished, got %+v", storedRun)
		}
		if !strings.Contains(storedRun.ErrorMessage, "Không khởi tạo được AI provider") {
			t.Fatalf("expected stored error message, got %q", storedRun.ErrorMessage)
		}
		var resultCount int64
		if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", run.ID).Count(&resultCount).Error; err != nil {
			t.Fatalf("count results: %v", err)
		}
		if resultCount != 0 {
			t.Fatalf("expected 0 results on provider failure, got %d", resultCount)
		}
		if f.checkpoint(t) != nil {
			t.Fatal("failed provider initialization must not set checkpoint")
		}
	})
}

// LP-03: Eligible work initializes the selected provider once per run after preparation;
// injected/test override precedence remains unchanged. Missing/undecryptable synthetic key
// still yields the existing bounded provider-unavailable error, stored failure and ownership release.
func TestLP03EligibleWorkInitializesProviderOnceAndRespectsPrecedence(t *testing.T) {
	t.Run("missing API key fails with earlyProviderUnavailable on eligible work", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "work", []time.Time{f.clock.Add(-time.Hour)})
		// No API key in app_settings

		nt := trapJobNotifications(t)
		analyzer := NewAnalyzer(&config.Config{})
		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err == nil || run == nil {
			t.Fatalf("expected error on missing API key with eligible work, got run=%v, err=%v", run, err)
		}
		if run.Status != "error" {
			t.Fatalf("expected run status error, got %q", run.Status)
		}
		if !strings.Contains(err.Error(), "Không khởi tạo được AI provider") {
			t.Fatalf("expected provider unavailable error message, got %v", err)
		}
		if calls := nt.count(); calls != 0 {
			t.Fatalf("expected 0 notifications on provider failure, got %d", calls)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership must be released on provider failure")
		}
		var storedRun models.JobRun
		if err := db.DB.Where("id = ?", run.ID).First(&storedRun).Error; err != nil {
			t.Fatalf("reload stored run %s: %v", run.ID, err)
		}
		if storedRun.Status != "error" || storedRun.FinishedAt == nil {
			t.Fatalf("expected stored run error/finished, got %+v", storedRun)
		}
		if !strings.Contains(storedRun.ErrorMessage, "Không khởi tạo được AI provider") {
			t.Fatalf("expected stored error message, got %q", storedRun.ErrorMessage)
		}
		var usageCount int64
		if err := db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&usageCount).Error; err != nil {
			t.Fatalf("count usage: %v", err)
		}
		if usageCount != 0 {
			t.Fatalf("expected 0 usage logs on provider failure, got %d", usageCount)
		}
		var resultCount int64
		if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", run.ID).Count(&resultCount).Error; err != nil {
			t.Fatalf("count results: %v", err)
		}
		if resultCount != 0 {
			t.Fatalf("expected 0 results on provider failure, got %d", resultCount)
		}
		if f.checkpoint(t) != nil {
			t.Fatal("failed run must not set checkpoint")
		}
	})

	t.Run("undecryptable API key fails with earlyProviderUnavailable on eligible work", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "work", []time.Time{f.clock.Add(-time.Hour)})
		f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, value_encrypted, created_at, updated_at) VALUES (?, ?, 'ai_api_key', '', X'01020304', NOW(), NOW())`,
			pkg.NewUUID(), f.tenantID)

		nt := trapJobNotifications(t)
		analyzer := NewAnalyzer(&config.Config{EncryptionKey: "01234567890123456789012345678901"})
		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err == nil || run == nil {
			t.Fatalf("expected error on undecryptable key, got run=%v, err=%v", run, err)
		}
		if run.Status != "error" {
			t.Fatalf("expected error status, got %q", run.Status)
		}
		if !strings.Contains(err.Error(), "Không khởi tạo được AI provider") {
			t.Fatalf("expected provider unavailable error message, got %v", err)
		}
		if calls := nt.count(); calls != 0 {
			t.Fatalf("expected 0 notifications on provider failure, got %d", calls)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership must be released on provider failure")
		}
		var storedRun models.JobRun
		if err := db.DB.Where("id = ?", run.ID).First(&storedRun).Error; err != nil {
			t.Fatalf("reload stored run %s: %v", run.ID, err)
		}
		if storedRun.Status != "error" || storedRun.FinishedAt == nil {
			t.Fatalf("expected stored run error/finished, got %+v", storedRun)
		}
		if !strings.Contains(storedRun.ErrorMessage, "Không khởi tạo được AI provider") {
			t.Fatalf("expected stored error message, got %q", storedRun.ErrorMessage)
		}
		var usageCount int64
		if err := db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&usageCount).Error; err != nil {
			t.Fatalf("count usage: %v", err)
		}
		if usageCount != 0 {
			t.Fatalf("expected 0 usage logs on provider failure, got %d", usageCount)
		}
		var resultCount int64
		if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", run.ID).Count(&resultCount).Error; err != nil {
			t.Fatalf("count results: %v", err)
		}
		if resultCount != 0 {
			t.Fatalf("expected 0 results on provider failure, got %d", resultCount)
		}
		if f.checkpoint(t) != nil {
			t.Fatal("failed run must not set checkpoint")
		}
	})

	t.Run("precedence: injectedProvider > providerOverride > providerResolver > settings", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "work", []time.Time{f.clock.Add(-time.Hour)})

		injectedP := newCountingProvider("qc_analysis")
		overrideP := newCountingProvider("qc_analysis")
		resolverP := newCountingProvider("qc_analysis")

		analyzer := NewAnalyzer(&config.Config{})
		analyzer.providerOverride = overrideP
		analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
			return resolverP, nil
		}

		// When injectedProvider is provided to RunJobWithProvider, it must win
		run, err := analyzer.RunJobWithProvider(context.Background(), f.job(t), 0, injectedP)
		if err != nil || run.Status != "success" {
			t.Fatalf("run failed: %v", err)
		}
		if atomic.LoadInt64(&injectedP.chatCalls) != 1 {
			t.Fatalf("expected injected provider to receive call, got %d", atomic.LoadInt64(&injectedP.chatCalls))
		}
		if atomic.LoadInt64(&overrideP.chatCalls) != 0 || atomic.LoadInt64(&resolverP.chatCalls) != 0 {
			t.Fatal("lower precedence providers received calls")
		}

		// When injectedProvider is nil, providerOverride wins
		f.addConv(t, f.tenantID, f.channelID, "work2", []time.Time{f.clock.Add(-40 * time.Minute)})
		atomic.StoreInt64(&injectedP.chatCalls, 0)
		run2, err := analyzer.RunJob(context.Background(), f.job(t))
		if err != nil || run2.Status != "success" {
			t.Fatalf("run failed: %v", err)
		}
		if atomic.LoadInt64(&overrideP.chatCalls) != 1 {
			t.Fatalf("expected override provider to receive call, got %d", atomic.LoadInt64(&overrideP.chatCalls))
		}
		if atomic.LoadInt64(&resolverP.chatCalls) != 0 {
			t.Fatal("resolver should not be called when override is set")
		}

		// When override is nil, providerResolver wins
		f.addConv(t, f.tenantID, f.channelID, "work3", []time.Time{f.clock.Add(-20 * time.Minute)})
		analyzer.providerOverride = nil
		run3, err := analyzer.RunJob(context.Background(), f.job(t))
		if err != nil || run3.Status != "success" {
			t.Fatalf("run failed: %v", err)
		}
		if atomic.LoadInt64(&resolverP.chatCalls) != 1 {
			t.Fatalf("expected resolver provider to receive call, got %d", atomic.LoadInt64(&resolverP.chatCalls))
		}
	})

	t.Run("provider is initialized exactly once per run across multiple conversations", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "c1", []time.Time{f.clock.Add(-time.Hour)})
		f.addConv(t, f.tenantID, f.channelID, "c2", []time.Time{f.clock.Add(-50 * time.Minute)})
		f.addConv(t, f.tenantID, f.channelID, "c3", []time.Time{f.clock.Add(-40 * time.Minute)})

		var resolverCalls int64
		cp := newCountingProvider("qc_analysis")
		analyzer := NewAnalyzer(&config.Config{})
		analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
			atomic.AddInt64(&resolverCalls, 1)
			return cp, nil
		}

		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err != nil || run.Status != "success" {
			t.Fatalf("run failed: %v", err)
		}
		if calls := atomic.LoadInt64(&resolverCalls); calls != 1 {
			t.Fatalf("expected exactly 1 provider resolution for multi-conversation run, got %d", calls)
		}
		if chatCalls := atomic.LoadInt64(&cp.chatCalls); chatCalls != 3 {
			t.Fatalf("expected 3 AnalyzeChat calls for 3 conversations, got %d", chatCalls)
		}
	})
}

// LP-04: Existing ordinary/explicit, full/unanalyzed/since_last, finite cap/date/test-run and
// batch/single execution semantics unchanged. Observe representative real Analyzer entry methods
// plus shared RunReserved.
func TestLP04SemanticsAcrossModesAndEntryPoints(t *testing.T) {
	entryPoints := []struct {
		name string
		call func(ctx context.Context, a *Analyzer, job models.Job, res *JobRunReservation) (*models.JobRun, error)
	}{
		{
			name: "RunJob (ordinary)",
			call: func(ctx context.Context, a *Analyzer, job models.Job, res *JobRunReservation) (*models.JobRun, error) {
				return a.RunJob(ctx, job)
			},
		},
		{
			name: "RunJobWithLimit (ordinary capped)",
			call: func(ctx context.Context, a *Analyzer, job models.Job, res *JobRunReservation) (*models.JobRun, error) {
				return a.RunJobWithLimit(ctx, job, 5)
			},
		},
		{
			name: "RunJobFull",
			call: func(ctx context.Context, a *Analyzer, job models.Job, res *JobRunReservation) (*models.JobRun, error) {
				return a.RunJobFull(ctx, job)
			},
		},
		{
			name: "RunJobUnanalyzed",
			call: func(ctx context.Context, a *Analyzer, job models.Job, res *JobRunReservation) (*models.JobRun, error) {
				return a.RunJobUnanalyzed(ctx, job, 10)
			},
		},
		{
			name: "RunJobSinceLast",
			call: func(ctx context.Context, a *Analyzer, job models.Job, res *JobRunReservation) (*models.JobRun, error) {
				return a.RunJobSinceLast(ctx, job, 10)
			},
		},
		{
			name: "RunReserved (shared HTTP/MCP/scheduler forwarding)",
			call: func(ctx context.Context, a *Analyzer, job models.Job, res *JobRunReservation) (*models.JobRun, error) {
				return a.RunReserved(res, job, "unanalyzed", 10, "", "")
			},
		},
	}

	for _, ep := range entryPoints {
		ep := ep
		t.Run(ep.name+" with eligible work", func(t *testing.T) {
			f := setupIncFixture(t, false, "qc_analysis")
			f.addConv(t, f.tenantID, f.channelID, "ep-work", []time.Time{f.clock.Add(-time.Hour)})

			var resolverCalls int64
			cp := newCountingProvider("qc_analysis")
			analyzer := NewAnalyzer(&config.Config{})
			analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
				atomic.AddInt64(&resolverCalls, 1)
				return cp, nil
			}

			var res *JobRunReservation
			var err error
			if strings.Contains(ep.name, "RunReserved") {
				res, err = ReserveJobRun(context.Background(), f.job(t), 0)
				if err != nil {
					t.Fatal(err)
				}
			}

			run, err := ep.call(context.Background(), analyzer, f.job(t), res)
			if err != nil || run == nil || run.Status != "success" {
				t.Fatalf("%s failed: %v %+v", ep.name, err, run)
			}
			if calls := atomic.LoadInt64(&resolverCalls); calls != 1 {
				t.Fatalf("expected 1 resolver call, got %d", calls)
			}
			if chatCalls := atomic.LoadInt64(&cp.chatCalls); chatCalls != 1 {
				t.Fatalf("expected 1 AnalyzeChat call, got %d", chatCalls)
			}
		})

		t.Run(ep.name+" without eligible work skips provider", func(t *testing.T) {
			f := setupIncFixture(t, false, "qc_analysis")
			// Zero conversations

			var resolverCalls int64
			analyzer := NewAnalyzer(&config.Config{})
			analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
				atomic.AddInt64(&resolverCalls, 1)
				return nil, errors.New("resolver should not be called")
			}

			var res *JobRunReservation
			var err error
			if strings.Contains(ep.name, "RunReserved") {
				res, err = ReserveJobRun(context.Background(), f.job(t), 0)
				if err != nil {
					t.Fatal(err)
				}
			}

			run, err := ep.call(context.Background(), analyzer, f.job(t), res)
			if err != nil || run == nil || run.Status != "success" {
				t.Fatalf("%s failed: %v %+v", ep.name, err, run)
			}
			if calls := atomic.LoadInt64(&resolverCalls); calls != 0 {
				t.Fatalf("expected 0 resolver calls, got %d", calls)
			}
		})
	}
}

// LP-05: Cancellation/time exhaustion before inference prevents provider initialization/new
// calls and retains cancelled/partial/ownership/checkpoint behavior.
func TestLP05CancellationPreventsProviderInitialization(t *testing.T) {
	t.Run("accepted cancellation before inference avoids provider initialization", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "c-work", []time.Time{f.clock.Add(-time.Hour)})
		job := f.job(t)

		res, err := ReserveJobRun(context.Background(), job, 0)
		if err != nil {
			t.Fatal(err)
		}

		// Request cancellation before executing
		if _, err := CancelJobRun(job.TenantID, job.ID, res.RunID()); err != nil {
			t.Fatal(err)
		}

		var resolverCalls int64
		analyzer := NewAnalyzer(&config.Config{})
		analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
			atomic.AddInt64(&resolverCalls, 1)
			return nil, errors.New("provider must not be resolved after cancellation")
		}

		run, err := analyzer.RunReserved(res, job, "unanalyzed", 10, "", "")
		if err != nil {
			// RunReserved returns nil error on accepted cancel
		}
		if run == nil || run.Status != "cancelled" {
			t.Fatalf("expected run status cancelled, got %v", run)
		}
		if calls := atomic.LoadInt64(&resolverCalls); calls != 0 {
			t.Fatalf("expected 0 provider resolver calls on cancelled run, got %d", calls)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("slot leaked on cancelled run")
		}
		if f.checkpoint(t) != nil {
			t.Fatal("cancelled run must not set checkpoint")
		}
	})

	t.Run("expired context before inference avoids provider initialization", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "ctx-work", []time.Time{f.clock.Add(-time.Hour)})
		job := f.job(t)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // context is cancelled immediately

		var resolverCalls int64
		analyzer := NewAnalyzer(&config.Config{})
		analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
			atomic.AddInt64(&resolverCalls, 1)
			return nil, errors.New("provider must not be resolved when context is done")
		}

		run, _ := analyzer.RunJobWithLimit(ctx, job, 5)
		if run == nil || (run.Status != "partial" && run.Status != "cancelled") {
			t.Fatalf("expected non-success status on expired context, got %v", run)
		}
		if calls := atomic.LoadInt64(&resolverCalls); calls != 0 {
			t.Fatalf("expected 0 provider resolver calls on expired context, got %d", calls)
		}
		if f.checkpoint(t) != nil {
			t.Fatal("expired context run must not set checkpoint")
		}
	})
}

// LP-06: Dedicated application regressions reject original eager initialization.
// Include actual source-selection no-work negative and eligible-source positive controls;
// count provider settings/decryption/constructor access distinctly from AnalyzeChat calls.
func TestLP06EagerInitializationDetectorNegativeAndPositive(t *testing.T) {
	t.Run("negative control: no work succeeds without AI settings/decryption", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		// Zero work in channel

		// Delete all app_settings for AI
		f.exec(t, `DELETE FROM app_settings WHERE tenant_id = ? AND setting_key LIKE 'ai_%'`, f.tenantID)
		// Insert corrupt encrypted key: under eager init, getProvider would fail decrypting this
		f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, value_encrypted, created_at, updated_at) VALUES (?, ?, 'ai_api_key', '', X'DEADBEEF', NOW(), NOW())`,
			pkg.NewUUID(), f.tenantID)

		queries := trapAISettingsQueries(t)
		nt := trapJobNotifications(t)
		analyzer := NewAnalyzer(&config.Config{EncryptionKey: "01234567890123456789012345678901"})
		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err != nil || run.Status != "success" {
			t.Fatalf("negative control failed: expected success without AI settings, got run=%v, err=%v", run, err)
		}
		if calls := nt.count(); calls != 0 {
			t.Fatalf("negative control: expected 0 notifications on no-work run, got %d", calls)
		}
		if calls := atomic.LoadInt64(queries); calls != 0 {
			t.Fatalf("negative control: expected 0 settings queries on no-work run, got %d", calls)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership must be released on no-work run")
		}
		var storedRun models.JobRun
		if err := db.DB.Where("id = ?", run.ID).First(&storedRun).Error; err != nil {
			t.Fatalf("reload stored run %s: %v", run.ID, err)
		}
		if storedRun.Status != "success" || storedRun.FinishedAt == nil {
			t.Fatalf("expected stored run success/finished, got %+v", storedRun)
		}
		var usageCount int64
		if err := db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&usageCount).Error; err != nil {
			t.Fatalf("count usage logs: %v", err)
		}
		if usageCount != 0 {
			t.Fatalf("expected 0 usage logs, got %d", usageCount)
		}
		var resultCount int64
		if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", run.ID).Count(&resultCount).Error; err != nil {
			t.Fatalf("count results: %v", err)
		}
		if resultCount != 0 {
			t.Fatalf("expected 0 results, got %d", resultCount)
		}
	})

	t.Run("positive control: eligible work strictly requires provider initialization (missing key)", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "pos-work", []time.Time{f.clock.Add(-time.Hour)})

		// Delete all app_settings for AI
		f.exec(t, `DELETE FROM app_settings WHERE tenant_id = ? AND setting_key LIKE 'ai_%'`, f.tenantID)

		queries := trapAISettingsQueries(t)
		nt := trapJobNotifications(t)
		analyzer := NewAnalyzer(&config.Config{})
		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err == nil || run == nil || run.Status != "error" {
			t.Fatalf("positive control failed: expected provider_unavailable error, got run=%v, err=%v", run, err)
		}
		if !strings.Contains(err.Error(), "Không khởi tạo được AI provider") {
			t.Fatalf("expected provider unavailable error message, got %v", err)
		}
		if calls := nt.count(); calls != 0 {
			t.Fatalf("positive control: expected 0 notifications on provider failure, got %d", calls)
		}
		if calls := atomic.LoadInt64(queries); calls == 0 {
			t.Fatalf("positive control: expected settings queries > 0, got %d", calls)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership must be released on provider failure")
		}
		var storedRun models.JobRun
		if err := db.DB.Where("id = ?", run.ID).First(&storedRun).Error; err != nil {
			t.Fatalf("reload stored run %s: %v", run.ID, err)
		}
		if storedRun.Status != "error" || storedRun.FinishedAt == nil {
			t.Fatalf("expected stored run error/finished, got %+v", storedRun)
		}
		if !strings.Contains(storedRun.ErrorMessage, "Không khởi tạo được AI provider") {
			t.Fatalf("expected stored error message, got %q", storedRun.ErrorMessage)
		}
		var usageCount int64
		if err := db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&usageCount).Error; err != nil {
			t.Fatalf("count usage logs: %v", err)
		}
		if usageCount != 0 {
			t.Fatalf("expected 0 usage logs, got %d", usageCount)
		}
		var resultCount int64
		if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", run.ID).Count(&resultCount).Error; err != nil {
			t.Fatalf("count results: %v", err)
		}
		if resultCount != 0 {
			t.Fatalf("expected 0 results, got %d", resultCount)
		}
		if f.checkpoint(t) != nil {
			t.Fatal("failed run must not set checkpoint")
		}
	})

	t.Run("positive control: eligible work strictly requires provider initialization (corrupt key)", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "pos-work-corrupt", []time.Time{f.clock.Add(-time.Hour)})

		f.exec(t, `DELETE FROM app_settings WHERE tenant_id = ? AND setting_key LIKE 'ai_%'`, f.tenantID)
		f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, value_encrypted, created_at, updated_at) VALUES (?, ?, 'ai_api_key', '', X'DEADBEEF', NOW(), NOW())`,
			pkg.NewUUID(), f.tenantID)

		queries := trapAISettingsQueries(t)
		nt := trapJobNotifications(t)
		analyzer := NewAnalyzer(&config.Config{EncryptionKey: "01234567890123456789012345678901"})
		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err == nil || run == nil || run.Status != "error" {
			t.Fatalf("positive control corrupt key failed: expected error, got run=%v, err=%v", run, err)
		}
		if !strings.Contains(err.Error(), "Không khởi tạo được AI provider") {
			t.Fatalf("expected provider unavailable error message, got %v", err)
		}
		if calls := nt.count(); calls != 0 {
			t.Fatalf("positive control: expected 0 notifications on provider failure, got %d", calls)
		}
		if calls := atomic.LoadInt64(queries); calls == 0 {
			t.Fatalf("positive control: expected settings queries > 0, got %d", calls)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership must be released on provider failure")
		}
		var storedRun models.JobRun
		if err := db.DB.Where("id = ?", run.ID).First(&storedRun).Error; err != nil {
			t.Fatalf("reload stored run %s: %v", run.ID, err)
		}
		if storedRun.Status != "error" || storedRun.FinishedAt == nil {
			t.Fatalf("expected stored run error/finished, got %+v", storedRun)
		}
		if !strings.Contains(storedRun.ErrorMessage, "Không khởi tạo được AI provider") {
			t.Fatalf("expected stored error message, got %q", storedRun.ErrorMessage)
		}
		var usageCount int64
		if err := db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&usageCount).Error; err != nil {
			t.Fatalf("count usage logs: %v", err)
		}
		if usageCount != 0 {
			t.Fatalf("expected 0 usage logs, got %d", usageCount)
		}
		var resultCount int64
		if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", run.ID).Count(&resultCount).Error; err != nil {
			t.Fatalf("count results: %v", err)
		}
		if resultCount != 0 {
			t.Fatalf("expected 0 results, got %d", resultCount)
		}
		if f.checkpoint(t) != nil {
			t.Fatal("failed run must not set checkpoint")
		}
	})

	t.Run("distinct counting of provider constructor access vs AnalyzeChat calls", func(t *testing.T) {
		f := setupIncFixture(t, true, "qc_analysis") // batch mode enabled
		f.addConv(t, f.tenantID, f.channelID, "w1", []time.Time{f.clock.Add(-time.Hour)})
		f.addConv(t, f.tenantID, f.channelID, "w2", []time.Time{f.clock.Add(-50 * time.Minute)})

		var constructorAccess int64
		cp := newCountingProvider("qc_analysis")
		analyzer := NewAnalyzer(&config.Config{})
		analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
			atomic.AddInt64(&constructorAccess, 1)
			return cp, nil
		}

		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err != nil || run.Status != "success" {
			t.Fatalf("run failed: %v", err)
		}

		constructed := atomic.LoadInt64(&constructorAccess)
		singleCalls := atomic.LoadInt64(&cp.singleCalls)
		batchCalls := atomic.LoadInt64(&cp.batchCalls)
		itemsCount := atomic.LoadInt64(&cp.itemsCount)
		totalCalls := cp.totalInferenceCalls()

		// Exactly 1 constructor access (provider resolved once)
		if constructed != 1 {
			t.Fatalf("expected constructorAccess == 1, got %d", constructed)
		}
		// In batch mode with 2 conversations: exactly 0 single calls, 1 batch call, 2 items
		if singleCalls != 0 {
			t.Fatalf("expected singleCalls == 0 in batch mode, got %d", singleCalls)
		}
		if batchCalls != 1 {
			t.Fatalf("expected batchCalls == 1 for two-item batch, got %d", batchCalls)
		}
		if totalCalls != 1 {
			t.Fatalf("expected totalInferenceCalls == 1, got %d", totalCalls)
		}
		if itemsCount != 2 {
			t.Fatalf("expected itemsCount == 2 across batches, got %d", itemsCount)
		}
	})

	t.Run("distinct counting in non-batch mode: constructor once, single calls match conversation count", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis") // non-batch mode
		f.addConv(t, f.tenantID, f.channelID, "nb1", []time.Time{f.clock.Add(-time.Hour)})
		f.addConv(t, f.tenantID, f.channelID, "nb2", []time.Time{f.clock.Add(-50 * time.Minute)})

		var constructorAccess int64
		cp := newCountingProvider("qc_analysis")
		analyzer := NewAnalyzer(&config.Config{})
		analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
			atomic.AddInt64(&constructorAccess, 1)
			return cp, nil
		}

		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err != nil || run.Status != "success" {
			t.Fatalf("run failed: %v", err)
		}

		constructed := atomic.LoadInt64(&constructorAccess)
		singleCalls := atomic.LoadInt64(&cp.singleCalls)
		batchCalls := atomic.LoadInt64(&cp.batchCalls)
		itemsCount := atomic.LoadInt64(&cp.itemsCount)
		totalCalls := cp.totalInferenceCalls()

		if constructed != 1 {
			t.Fatalf("expected constructorAccess == 1, got %d", constructed)
		}
		if singleCalls != 2 {
			t.Fatalf("expected singleCalls == 2 for two conversations, got %d", singleCalls)
		}
		if batchCalls != 0 {
			t.Fatalf("expected batchCalls == 0 in non-batch mode, got %d", batchCalls)
		}
		if totalCalls != 2 {
			t.Fatalf("expected totalInferenceCalls == 2, got %d", totalCalls)
		}
		if itemsCount != 2 {
			t.Fatalf("expected itemsCount == 2, got %d", itemsCount)
		}
	})
}
