package engine

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-050 (LP-01..08): source-first application provider initialization.
// The Analyzer delays provider settings/key resolution and provider construction until
// candidate snapshots have been prepared and eligible work requiring inference remains.

// countingProvider wraps an incProvider and counts AnalyzeChat invocations.
type countingProvider struct {
	*incProvider
	chatCalls int64
}

func newCountingProvider(jobType string) *countingProvider {
	return &countingProvider{
		incProvider: &incProvider{jobType: jobType, verdict: "PASS"},
	}
}

func (cp *countingProvider) AnalyzeChat(ctx context.Context, systemPrompt, transcript string) (ai.AIResponse, error) {
	atomic.AddInt64(&cp.chatCalls, 1)
	return cp.incProvider.AnalyzeChat(ctx, systemPrompt, transcript)
}

func (cp *countingProvider) AnalyzeChatBatch(ctx context.Context, systemPrompt string, items []ai.BatchItem) (ai.AIResponse, error) {
	atomic.AddInt64(&cp.chatCalls, int64(len(items)))
	return cp.incProvider.AnalyzeChatBatch(ctx, systemPrompt, items)
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

			var resolverCalls int64
			analyzer := NewAnalyzer(&config.Config{EncryptionKey: "01234567890123456789012345678901"})
			analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
				atomic.AddInt64(&resolverCalls, 1)
				return nil, errors.New("resolver should not be called when no work is prepared")
			}

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

			// Verify provider resolver was never invoked
			if calls := atomic.LoadInt64(&resolverCalls); calls != 0 {
				t.Fatalf("expected 0 provider resolver calls, got %d", calls)
			}

			// Verify summary reflects 0 work
			var summary map[string]interface{}
			if err := json.Unmarshal([]byte(run.Summary), &summary); err != nil {
				t.Fatalf("parse summary: %v", err)
			}
			if found, _ := summary["conversations_found"].(float64); found != 0 {
				t.Fatalf("expected conversations_found=0, got %v", found)
			}
			if analyzed, _ := summary["conversations_analyzed"].(float64); analyzed != 0 {
				t.Fatalf("expected conversations_analyzed=0, got %v", analyzed)
			}

			// Verify zero AI usage logs
			var usageCount int64
			db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&usageCount)
			if usageCount != 0 {
				t.Fatalf("expected 0 AI usage logs, got %d", usageCount)
			}

			// Verify zero job results
			var resultCount int64
			db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", run.ID).Count(&resultCount)
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

		var resolverCalls int64
		analyzer := NewAnalyzer(&config.Config{})
		analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
			atomic.AddInt64(&resolverCalls, 1)
			return nil, errors.New("resolver should not be called")
		}

		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err == nil || run == nil {
			t.Fatalf("expected error on invalid channel JSON, got run=%v, err=%v", run, err)
		}
		if run.Status != "error" {
			t.Fatalf("expected error status, got %q", run.Status)
		}
		if atomic.LoadInt64(&resolverCalls) != 0 {
			t.Fatalf("expected 0 resolver calls, got %d", atomic.LoadInt64(&resolverCalls))
		}
		if f.checkpoint(t) != nil {
			t.Fatal("failed run must not set checkpoint")
		}
	})

	t.Run("all-failed preparation closes as error with zero provider initialization", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "bad", []time.Time{f.clock.Add(-time.Hour)})

		// Make snapshot loading fail by inducing a query error on the messages table
		failTable(t, "messages", "SYNTHETIC_MSG_QUERY_FAIL")

		var resolverCalls int64
		analyzer := NewAnalyzer(&config.Config{})
		analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
			atomic.AddInt64(&resolverCalls, 1)
			return nil, errors.New("resolver should not be called on all-failed preparation")
		}

		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err != nil {
			// RunJob returns nil err when run completes through ordinary terminal path with error status
		}
		if run == nil || run.Status != "error" {
			t.Fatalf("all-failed preparation must close as error, got status=%v", run)
		}
		if atomic.LoadInt64(&resolverCalls) != 0 {
			t.Fatalf("expected 0 provider resolver calls on all-failed prep, got %d", atomic.LoadInt64(&resolverCalls))
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
		analyzer := NewAnalyzer(&config.Config{})
		analyzer.providerResolver = func(job models.Job) (ai.AIProvider, error) {
			atomic.AddInt64(&resolverCalls, 1)
			return cp, nil
		}

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
		if chatCalls := atomic.LoadInt64(&cp.chatCalls); chatCalls != 1 {
			t.Fatalf("expected exactly 1 AnalyzeChat call for valid conversation, got %d", chatCalls)
		}
		// Checkpoint must NOT advance on partial run
		if f.checkpoint(t) != nil {
			t.Fatal("partial run must not set checkpoint")
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
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership must be released on provider failure")
		}
	})

	t.Run("undecryptable API key fails with earlyProviderUnavailable on eligible work", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "work", []time.Time{f.clock.Add(-time.Hour)})
		f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, value_encrypted, created_at, updated_at) VALUES (?, ?, 'ai_api_key', '', X'01020304', NOW(), NOW())`,
			pkg.NewUUID(), f.tenantID)

		analyzer := NewAnalyzer(&config.Config{EncryptionKey: "01234567890123456789012345678901"})
		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err == nil || run == nil {
			t.Fatalf("expected error on undecryptable key, got run=%v, err=%v", run, err)
		}
		if run.Status != "error" {
			t.Fatalf("expected error status, got %q", run.Status)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership must be released on provider failure")
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

		// Under original eager initialization, NewAnalyzer without settings would fail here:
		// getProvider -> "API key not configured" -> failOwnedRun(..., earlyProviderUnavailable).
		// Under lazy initialization, this must succeed cleanly.
		analyzer := NewAnalyzer(&config.Config{})
		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err != nil || run.Status != "success" {
			t.Fatalf("negative control failed: expected success without AI settings, got run=%v, err=%v", run, err)
		}
	})

	t.Run("positive control: eligible work strictly requires provider initialization", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "pos-work", []time.Time{f.clock.Add(-time.Hour)})

		// Delete all app_settings for AI
		f.exec(t, `DELETE FROM app_settings WHERE tenant_id = ? AND setting_key LIKE 'ai_%'`, f.tenantID)

		// With eligible work and missing settings, lazy initialization MUST detect and fail
		analyzer := NewAnalyzer(&config.Config{})
		run, err := analyzer.RunJob(context.Background(), f.job(t))
		if err == nil || run == nil || run.Status != "error" {
			t.Fatalf("positive control failed: expected provider_unavailable error, got run=%v, err=%v", run, err)
		}
		if !strings.Contains(err.Error(), "Không khởi tạo được AI provider") {
			t.Fatalf("expected provider unavailable error message, got %v", err)
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
		analyzed := atomic.LoadInt64(&cp.chatCalls)

		// Exactly 1 constructor access
		if constructed != 1 {
			t.Fatalf("expected constructorAccess == 1, got %d", constructed)
		}
		// 2 conversations analyzed
		if analyzed != 2 {
			t.Fatalf("expected chatCalls == 2, got %d", analyzed)
		}

		// Constructor access was distinct and happened once, while chat calls matched conversation count
		if constructed == analyzed {
			t.Fatalf("constructor access (%d) should be distinct from inference call count (%d)", constructed, analyzed)
		}
	})
}
