package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-027-R1 (F05-R1-04): named Vietnam date cases through the real admission, and the
// real route -> worker -> Analyzer -> DB conditional date+cap repeat evaluation. Synthetic provider.

// The named cases shared with the engine DB test (analyzer_modes_acceptance_test.go) and the
// mounted JobDetail spec (job-run-dialog-modes.spec.ts).
var namedTriggerDateCases = []struct {
	name, from, to string
	valid          bool
}{
	{"same day", "2026-10-02", "2026-10-02", true},
	{"month rollover", "2026-09-30", "2026-10-01", true},
	{"year rollover", "2025-12-31", "2026-01-01", true},
	{"leap day", "2028-02-29", "2028-02-29", true},
	{"from only", "2026-10-02", "", true},
	{"to only", "", "2026-10-02", true},
	{"reversed", "2026-10-03", "2026-10-02", false},
	{"not a leap year", "2100-02-29", "2100-02-29", false},
	{"impossible calendar date", "2026-02-30", "", false},
	{"time string", "2026-10-02T00:00:00", "", false},
	{"before the minimum", "0999-12-31", "", false},
	{"supplied maximum overflows", "", "9999-12-31", false},
}

func TestTriggerJobNamedDateCasesAdmission(t *testing.T) {
	for _, c := range namedTriggerDateCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			db.Close()
			f := setupJobDispatchFixture(t)
			q := url.Values{"mode": {"conditional"}}
			if c.from != "" {
				q.Set("from", c.from)
			}
			if c.to != "" {
				q.Set("to", c.to)
			}
			rec := f.callTrigger(f.tenantID, q.Encode())
			if c.valid {
				want := triggerJobParams{mode: "conditional", dateFrom: c.from, dateTo: c.to}
				if rec.Code != http.StatusAccepted || f.triggerParams != want {
					t.Fatalf("got %d %s params %+v, want 202 %+v", rec.Code, rec.Body.String(), f.triggerParams, want)
				}
				f.assertWorkerStarted(t)
				return
			}
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_date_range") || f.cfgLoads != 0 {
				t.Fatalf("got %d %s config loads %d", rec.Code, rec.Body.String(), f.cfgLoads)
			}
			f.assertNoTriggerStart(t, rec)
		})
	}
}

// recProvider records the transcripts it is given and always passes.
type recProvider struct {
	mu          sync.Mutex
	transcripts []string
}

func (p *recProvider) AnalyzeChat(_ context.Context, _ string, transcript string) (ai.AIResponse, error) {
	p.mu.Lock()
	p.transcripts = append(p.transcripts, transcript)
	p.mu.Unlock()
	return passProvider{}.AnalyzeChat(context.Background(), "", "")
}

func (p *recProvider) AnalyzeChatBatch(ctx context.Context, s string, items []ai.BatchItem) (ai.AIResponse, error) {
	return passProvider{}.AnalyzeChatBatch(ctx, s, items)
}

// Through the real handler and the real trigger worker, a conditional date + cap run repeats the
// evaluation of an already evaluated conversation with the full local snapshot, keeps the mode
// (conditional, not unanalyzed), applies the cap after the Vietnam date bounds and preserves the
// checkpoint.
func TestTriggerJobRealRouteConditionalDateCapRepeatsEvaluation(t *testing.T) {
	realTrigger := startTriggerJob
	db.Close()
	f := setupJobDispatchFixture(t)
	startTriggerJob = realTrigger
	prov := &recProvider{}
	origAnalyzer := newTriggerAnalyzer
	newTriggerAnalyzer = func(cfg *config.Config) *engine.Analyzer { return engine.NewAnalyzerWithProvider(cfg, prov) }
	t.Cleanup(func() { newTriggerAnalyzer = origAnalyzer })

	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	chID := "ch-" + f.jobID
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh', ?, X'00', true, '{}', NOW(), NOW())`, chID, f.tenantID, "ext-"+chID)
	exec(`UPDATE jobs SET input_channel_ids = ?, rules_content = 'Phan hoi dung han.' WHERE id = ?`, `["`+chID+`"]`, f.jobID)
	exec(`INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at) VALUES (?, ?, 'ai_batch_mode', 'false', NOW(), NOW())`, pkg.NewUUID(), f.tenantID)
	t.Cleanup(func() {
		for _, table := range []string{"job_results", "analysis_snapshots", "ai_usage_logs", "messages", "conversations", "app_settings", "channels"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", f.tenantID)
		}
	})
	at := func(s string) time.Time { v, _ := time.Parse(time.RFC3339, s); return v }
	conv := func(label string, times ...time.Time) string {
		id := "conv-" + label + "-" + f.jobID
		exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, 'Khach', ?, ?, '{}', NOW(), NOW())`, id, f.tenantID, chID, "ext-"+id, times[len(times)-1], len(times))
		for i, tm := range times {
			exec(`INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, ?, 'customer', 'Khach', ?, 'text', '[]', ?, NOW())`, pkg.NewUUID(), f.tenantID, id, fmt.Sprintf("x-%s-%d", id, i), fmt.Sprintf("Noi dung %s %d", label, i), tm)
		}
		return id
	}
	// Vietnam 2026-10-02 is [2026-10-01T17:00:00Z, 2026-10-02T17:00:00Z).
	outside := conv("c", at("2026-10-01T16:59:59Z"))                          // VN Oct 1 23:59:59: before the day
	edge := conv("d", at("2026-09-20T03:00:00Z"), at("2026-10-01T17:00:00Z")) // VN Oct 2 00:00:00, with an earlier message
	mid := conv("a", at("2026-10-02T03:00:00Z"))
	late := conv("b", at("2026-10-02T04:00:00Z"))
	sentinel := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	exec("UPDATE jobs SET last_run_at = ? WHERE id = ?", sentinel, f.jobID)

	// waitRun returns the finished run of the job that is not in the excluded set.
	waitRun := func(exclude ...string) models.JobRun {
		var run models.JobRun
		deadline := time.Now().Add(20 * time.Second)
		for {
			q := db.DB.Where("job_id = ?", f.jobID)
			if len(exclude) > 0 {
				q = q.Where("id NOT IN ?", exclude)
			}
			err := q.Order("started_at DESC").First(&run).Error
			if err == nil && run.Status != "running" {
				return run
			}
			if time.Now().After(deadline) {
				t.Fatalf("run did not finish: %v %+v", err, run)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	evaluatedIDs := func(runID string) string {
		var ids []string
		db.DB.Model(&models.JobResult{}).Where("job_run_id = ? AND result_type = 'conversation_evaluation'", runID).Order("conversation_id").Pluck("conversation_id", &ids)
		return strings.Join(ids, ",")
	}

	// 1) unanalyzed evaluates the four conversations once
	if rec := f.callTrigger(f.tenantID, "mode=unanalyzed"); rec.Code != http.StatusAccepted {
		t.Fatalf("unanalyzed: %d %s", rec.Code, rec.Body.String())
	}
	first := waitRun()
	if first.Status != "success" {
		t.Fatalf("first run %s (%s)", first.Status, first.ErrorMessage)
	}
	prov.mu.Lock()
	prov.transcripts = nil
	prov.mu.Unlock()

	// 2) conditional + the Vietnam day + cap 2: the two oldest conversations of the day, although
	// both were already evaluated, each with its full snapshot.
	rec := f.callTrigger(f.tenantID, "mode=conditional&from=2026-10-02&to=2026-10-02&limit=2")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("conditional: %d %s", rec.Code, rec.Body.String())
	}
	second := waitRun(first.ID)
	if second.Status != "success" {
		t.Fatalf("second run %s (%s)", second.Status, second.ErrorMessage)
	}
	want := []string{edge, mid}
	sort.Strings(want)
	if got := evaluatedIDs(second.ID); got != strings.Join(want, ",") {
		t.Fatalf("second run evaluated %q, want %q (not %s / %s)", got, strings.Join(want, ","), outside, late)
	}
	var evalsOfEdge int64
	db.DB.Model(&models.JobResult{}).Where("conversation_id = ? AND result_type = 'conversation_evaluation'", edge).Count(&evalsOfEdge)
	if evalsOfEdge != 2 {
		t.Fatalf("%d evaluations of the repeated conversation, want 2", evalsOfEdge)
	}
	prov.mu.Lock()
	seen := strings.Join(prov.transcripts, "\n")
	prov.mu.Unlock()
	if !strings.Contains(seen, "Noi dung d 0") || !strings.Contains(seen, "Noi dung d 1") {
		t.Fatalf("the date bound cut the conversation context: %q", seen)
	}
	var snap models.AnalysisSnapshot
	if err := db.DB.Where("job_run_id = ? AND conversation_id = ?", second.ID, edge).First(&snap).Error; err != nil || snap.MessageCount != 2 {
		t.Fatalf("saved snapshot for the repeated evaluation: %v %+v", err, snap)
	}
	var job models.Job
	if err := db.DB.First(&job, "id = ?", f.jobID).Error; err != nil {
		t.Fatal(err)
	}
	if job.LastRunAt == nil || !job.LastRunAt.Equal(sentinel) || job.LastRunStatus != "success" {
		t.Fatalf("job after the conditional run: last_run_at %v status %q, want checkpoint %v preserved", job.LastRunAt, job.LastRunStatus, sentinel)
	}
}
