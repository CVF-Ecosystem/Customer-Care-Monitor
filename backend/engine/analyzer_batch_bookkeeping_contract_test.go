package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

func TestR091BatchBackoffDelayScheduleAndCanceledContext(t *testing.T) {
	old := batchBackoffSleep
	defer func() { batchBackoffSleep = old }()
	var got []time.Duration
	var contexts []context.Context
	batchBackoffSleep = func(ctx context.Context, delay time.Duration) {
		contexts = append(contexts, ctx)
		got = append(got, delay)
	}
	waitBatchBackoff(context.Background(), false, 0)
	waitBatchBackoff(context.Background(), true, 1)
	waitBatchBackoff(context.Background(), true, 2)
	waitBatchBackoff(context.Background(), true, 3)
	batchBackoffSleep = old
	if !reflect.DeepEqual(got, []time.Duration{2 * time.Second, 10 * time.Second, 10 * time.Second, 30 * time.Second}) {
		t.Fatalf("batch backoff schedule=%v", got)
	}
	for _, ctx := range contexts {
		if ctx == nil || ctx.Err() != nil {
			t.Fatalf("batch backoff did not receive the live run context: %v", ctx)
		}
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	waitBatchBackoff(canceled, true, 1)
	if time.Since(started) > 150*time.Millisecond {
		t.Fatalf("canceled backoff did not stop promptly: %s", time.Since(started))
	}
}

type r091BatchFaultProvider struct {
	incProvider
	firstFault      string
	beforeBatchCall func(int)
}

func (p *r091BatchFaultProvider) AnalyzeChatBatch(_ context.Context, _ string, items []ai.BatchItem) (ai.AIResponse, error) {
	transcripts := make([]string, len(items))
	for i, item := range items {
		transcripts[i] = item.Transcript
	}
	n := p.record(transcripts)
	if p.beforeBatchCall != nil {
		p.beforeBatchCall(n)
	}
	if n == 1 && p.firstFault == "parse" {
		return ai.AIResponse{Content: "not-json", Model: "test-double", Provider: "test-double"}, nil
	}

	p.badRef = n == 1 && p.firstFault == "save"
	out := make([]json.RawMessage, len(items))
	for i, item := range items {
		out[i] = p.result(item.Transcript)
	}
	body, err := json.Marshal(out)
	if err != nil {
		return ai.AIResponse{}, err
	}
	return ai.AIResponse{Content: string(body), Model: "test-double", Provider: "test-double"}, nil
}

func r091ReadBatchProgress(t *testing.T, jobID string) map[string]any {
	t.Helper()
	var run models.JobRun
	if err := db.DB.Where("job_id = ?", jobID).Order("started_at DESC").First(&run).Error; err != nil {
		t.Fatalf("read synthetic job-run progress: %v", err)
	}
	var summary map[string]any
	if err := json.Unmarshal([]byte(run.Summary), &summary); err != nil {
		t.Fatalf("decode synthetic job-run progress: %v", err)
	}
	return summary
}

// Parse and persistence failures share the same batch tail as provider failures. The injected
// sleep records the exact production delay selection without waiting ten real seconds; progress
// is read at the tail and again at each next provider invocation.
func TestR091ParseAndSaveBatchFailuresPersistProgressAndUseCommonBackoffTail(t *testing.T) {
	originalSleep := batchBackoffSleep
	defer func() { batchBackoffSleep = originalSleep }()

	for _, fault := range []string{"parse", "save"} {
		t.Run(fault, func(t *testing.T) {
			f := setupIncFixture(t, true, "qc_analysis")
			f.exec(t, "UPDATE app_settings SET value_plain = '1' WHERE tenant_id = ? AND setting_key = 'ai_batch_size'", f.tenantID)
			f.addConv(t, f.tenantID, f.channelID, "first", []time.Time{f.clock.Add(-3 * time.Hour)})
			f.addConv(t, f.tenantID, f.channelID, "second", []time.Time{f.clock.Add(-2 * time.Hour)})
			f.addConv(t, f.tenantID, f.channelID, "third", []time.Time{f.clock.Add(-time.Hour)})

			var delays []time.Duration
			batchBackoffSleep = func(ctx context.Context, delay time.Duration) {
				if ctx == nil || ctx.Err() != nil {
					t.Errorf("batch delay did not receive a live run context: %v", ctx)
				}
				delays = append(delays, delay)
				summary := r091ReadBatchProgress(t, f.jobID)
				if got, _ := summary["conversations_errors"].(float64); got < 1 {
					t.Errorf("%s failure was not persisted before the shared batch wait: %+v", fault, summary)
				}
			}

			provider := &r091BatchFaultProvider{firstFault: fault}
			provider.beforeBatchCall = func(call int) {
				if call == 1 {
					return
				}
				if len(delays) != call-1 {
					t.Errorf("provider batch %d started without the prior common wait; observed delays=%v", call, delays)
				}
				summary := r091ReadBatchProgress(t, f.jobID)
				if got, _ := summary["conversations_errors"].(float64); got < 1 {
					t.Errorf("provider batch %d started before failure progress was stored: %+v", call, summary)
				}
			}
			run, err := f.run(t, provider)
			if err != nil {
				t.Fatalf("synthetic batch run: %v", err)
			}
			if provider.callCount() != 3 {
				t.Fatalf("provider batch calls=%d, want three singleton batches", provider.callCount())
			}
			if !reflect.DeepEqual(delays, []time.Duration{10 * time.Second, 2 * time.Second}) {
				t.Fatalf("%s error then healthy batch delays=%v, want [10s 2s]", fault, delays)
			}
			if run.Status != "partial" {
				t.Fatalf("run status=%q, want partial after one %s failure", run.Status, fault)
			}
			summary := r091ReadBatchProgress(t, f.jobID)
			if errors, _ := summary["conversations_errors"].(float64); errors != 1 {
				t.Fatalf("final error accounting=%v, want one failed conversation; summary=%+v", summary["conversations_errors"], summary)
			}
			if analyzed, _ := summary["conversations_analyzed"].(float64); analyzed != 2 {
				t.Fatalf("final success accounting=%v, want two later successes; summary=%+v", summary["conversations_analyzed"], summary)
			}
		})
	}
}

type r091BatchStreakProvider struct {
	incProvider
	failThrough int
}

func (p *r091BatchStreakProvider) AnalyzeChatBatch(_ context.Context, _ string, items []ai.BatchItem) (ai.AIResponse, error) {
	transcripts := make([]string, len(items))
	for i, item := range items {
		transcripts[i] = item.Transcript
	}
	n := p.record(transcripts)
	if n <= p.failThrough {
		return ai.AIResponse{}, errors.New("synthetic provider batch failure")
	}
	results := make([]json.RawMessage, len(items))
	for i, item := range items {
		results[i] = p.result(item.Transcript)
	}
	body, err := json.Marshal(results)
	if err != nil {
		return ai.AIResponse{}, err
	}
	return ai.AIResponse{Content: string(body), Model: "test-double", Provider: "test-double"}, nil
}

func TestR091R1AnalyzerUsesThreeErrorEscalationHealthyResetAndCanceledWait(t *testing.T) {
	originalSleep := batchBackoffSleep
	defer func() { batchBackoffSleep = originalSleep }()
	originalNotify := sendJobNotifications
	defer func() { sendJobNotifications = originalNotify }()
	sendJobNotifications = func(context.Context, models.Job, models.JobRun) error { return nil }

	t.Run("actual analyzer streak and reset", func(t *testing.T) {
		f := setupIncFixture(t, true, "qc_analysis")
		sendJobNotifications = func(context.Context, models.Job, models.JobRun) error { return nil }
		f.exec(t, "UPDATE app_settings SET value_plain = '1' WHERE tenant_id = ? AND setting_key = 'ai_batch_size'", f.tenantID)
		for i := 0; i < 5; i++ {
			f.addConv(t, f.tenantID, f.channelID, fmt.Sprintf("streak-%d", i), []time.Time{f.clock.Add(time.Duration(i-6) * time.Hour)})
		}
		var delays []time.Duration
		batchBackoffSleep = func(ctx context.Context, delay time.Duration) {
			if ctx == nil || ctx.Err() != nil {
				t.Errorf("actual Analyzer delay did not receive live owner context: %v", ctx)
			}
			delays = append(delays, delay)
			summary := r091ReadBatchProgress(t, f.jobID)
			wantErrors := len(delays)
			if wantErrors > 3 {
				wantErrors = 3
			}
			if got, _ := summary["conversations_errors"].(float64); got != float64(wantErrors) {
				t.Errorf("actual Analyzer persisted member errors=%v at delay %d; want %d", got, len(delays), wantErrors)
			}
		}
		provider := &r091BatchStreakProvider{incProvider: incProvider{jobType: "qc_analysis"}, failThrough: 3}
		provider.onCall = func(call int) {
			if call > 1 && len(delays) != call-1 {
				t.Errorf("actual provider call %d began without common wait; delays=%v", call, delays)
			}
			summary := r091ReadBatchProgress(t, f.jobID)
			wantErrors := call - 1
			if wantErrors > 3 {
				wantErrors = 3
			}
			if call > 1 {
				if got, _ := summary["conversations_errors"].(float64); got != float64(wantErrors) {
					t.Errorf("provider call %d started with conversations_errors=%v; want %d", call, got, wantErrors)
				}
			}
		}
		run, err := f.run(t, provider)
		if err != nil {
			t.Fatalf("real Analyzer streak run failed: %v", err)
		}
		if provider.callCount() != 5 || run.Status != "partial" {
			t.Fatalf("Analyzer run status=%q calls=%d; want partial and five calls", run.Status, provider.callCount())
		}
		if !reflect.DeepEqual(delays, []time.Duration{10 * time.Second, 10 * time.Second, 30 * time.Second, 2 * time.Second}) {
			t.Fatalf("actual Analyzer error streak/reset delays=%v", delays)
		}
		summary := r091ReadBatchProgress(t, f.jobID)
		if got, _ := summary["conversations_errors"].(float64); got != 3 {
			t.Fatalf("actual Analyzer final member-error accounting=%v; want 3", got)
		}
	})

	t.Run("cancellation during common wait blocks next provider call", func(t *testing.T) {
		f := setupIncFixture(t, true, "qc_analysis")
		sendJobNotifications = func(context.Context, models.Job, models.JobRun) error { return nil }
		f.exec(t, "UPDATE app_settings SET value_plain = '1' WHERE tenant_id = ? AND setting_key = 'ai_batch_size'", f.tenantID)
		for i := 0; i < 2; i++ {
			f.addConv(t, f.tenantID, f.channelID, fmt.Sprintf("cancel-%d", i), []time.Time{f.clock.Add(time.Duration(i-3) * time.Hour)})
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		waitObserved := false
		batchBackoffSleep = func(_ context.Context, delay time.Duration) {
			if delay != 10*time.Second {
				t.Errorf("error batch selected delay=%s; want 10s", delay)
			}
			waitObserved = true
			cancel()
		}
		provider := &r091BatchStreakProvider{incProvider: incProvider{jobType: "qc_analysis"}, failThrough: 1}
		run, err := NewAnalyzer(&config.Config{}).RunJobWithProvider(ctx, f.job(t), 0, provider)
		if err != nil {
			t.Fatalf("canceling actual Analyzer run failed: %v", err)
		}
		if !waitObserved || provider.callCount() != 1 || run == nil {
			t.Fatalf("common-wait cancel result: wait=%v calls=%d runNil=%v; want true/1/non-nil", waitObserved, provider.callCount(), run == nil)
		}
	})
}
