package handlers

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-027 (F05): TriggerJob admits only a valid mode, a positive integer cap and Vietnam
// dates that belong to the conditional mode. A rejection happens before configuration loading and
// before any worker, run row or cancel handle. The mode is never inferred from the cap.

func TestTriggerJobStrictAdmissionRejectsBeforeAnySideEffect(t *testing.T) {
	cases := []struct {
		name, query, wantErr string
	}{
		{"unknown mode", "mode=bogus", "invalid_run_parameters"},
		{"mode case matters", "mode=Conditional", "invalid_run_parameters"},
		{"limit signed plus", "mode=unanalyzed&limit=%2B2", "invalid_run_parameters"},
		{"limit whitespace", "mode=unanalyzed&limit=%202", "invalid_run_parameters"},
		{"limit overflow", "mode=unanalyzed&limit=99999999999999999999", "invalid_run_parameters"},
		{"limit hex", "mode=unanalyzed&limit=0x10", "invalid_run_parameters"},
		{"full conflicts with unanalyzed", "mode=unanalyzed&full=true", "invalid_run_parameters"},
		{"full conflicts with since_last", "mode=since_last&full=true", "invalid_run_parameters"},
		{"full is not a boolean", "mode=conditional&full=maybe", "invalid_run_parameters"},
		{"full uppercase", "full=TRUE", "invalid_run_parameters"},
		{"limit negative", "mode=unanalyzed&limit=-3", "invalid_run_parameters"},
		{"limit not a number", "mode=since_last&limit=abc", "invalid_run_parameters"},
		{"limit fractional", "mode=conditional&from=2026-10-01&to=2026-10-02&limit=1.5", "invalid_run_parameters"},
		{"dates on unanalyzed", "mode=unanalyzed&from=2026-10-01&to=2026-10-02", "invalid_run_parameters"},
		{"dates on since_last", "mode=since_last&from=2026-10-01", "invalid_run_parameters"},
		{"dates without a mode (since_last default)", "from=2026-10-01&to=2026-10-02", "invalid_run_parameters"},
		{"reversed dates", "mode=conditional&from=2026-10-03&to=2026-10-02", "invalid_date_range"},
		{"bad date", "mode=conditional&from=2026-13-01&to=2026-13-02", "invalid_date_range"},
		{"date-time instead of a date", "mode=conditional&from=2026-10-01T00:00:00Z&to=2026-10-02", "invalid_date_range"},
		{"slash date", "mode=conditional&from=01/10/2026&to=02/10/2026", "invalid_date_range"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			db.Close() // this file adds many fixtures; release the previous pool first
			f := setupJobDispatchFixture(t)
			rec := f.callTrigger(f.tenantID, c.query)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), c.wantErr) {
				t.Fatalf("got %d %s, want 400 %s", rec.Code, rec.Body.String(), c.wantErr)
			}
			if f.cfgLoads != 0 {
				t.Fatalf("configuration loaded %d times before parameter admission", f.cfgLoads)
			}
			f.assertNoTriggerStart(t, rec)
		})
	}
}

func TestTriggerJobAdmitsModeAndCapIndependently(t *testing.T) {
	cases := []struct {
		query string
		want  triggerJobParams
	}{
		{"mode=since_last&limit=3", triggerJobParams{mode: "since_last", maxConv: 3}},
		{"mode=unanalyzed&limit=5", triggerJobParams{mode: "unanalyzed", maxConv: 5}},
		{"mode=unanalyzed", triggerJobParams{mode: "unanalyzed"}},
		{"mode=conditional&limit=2", triggerJobParams{mode: "conditional", maxConv: 2}},
		{"mode=conditional&from=2026-10-02&to=2026-10-02", triggerJobParams{mode: "conditional", dateFrom: "2026-10-02", dateTo: "2026-10-02"}},
		{"mode=conditional&from=2026-10-02", triggerJobParams{mode: "conditional", dateFrom: "2026-10-02"}},
		{"full=true", triggerJobParams{mode: "conditional"}},
		{"limit=4", triggerJobParams{mode: "since_last", maxConv: 4}},
		{"mode=unanalyzed&limit=0", triggerJobParams{mode: "unanalyzed"}},
		{"mode=unanalyzed&limit=", triggerJobParams{mode: "unanalyzed"}},
		{"mode=since_last&limit=007", triggerJobParams{mode: "since_last", maxConv: 7}},
		{"mode=since_last&full=false", triggerJobParams{mode: "since_last"}},
		{"mode=conditional&full=true&limit=2", triggerJobParams{mode: "conditional", maxConv: 2}},
		{"full=true&from=2026-10-01&to=2026-10-02", triggerJobParams{mode: "conditional", dateFrom: "2026-10-01", dateTo: "2026-10-02"}},
		{"full=false", triggerJobParams{mode: "since_last"}},
		{"full=", triggerJobParams{mode: "since_last"}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.query, func(t *testing.T) {
			db.Close() // this file adds many fixtures; release the previous pool first
			f := setupJobDispatchFixture(t)
			rec := f.callTrigger(f.tenantID, c.query)
			if rec.Code != http.StatusAccepted {
				t.Fatalf("got %d %s", rec.Code, rec.Body.String())
			}
			if f.triggerParams != c.want {
				t.Fatalf("worker got %+v, want %+v", f.triggerParams, c.want)
			}
			f.assertWorkerStarted(t)
		})
	}
}

// ---- one real route -> Analyzer execution on a disposable DB (synthetic provider) ----

type passProvider struct{}

func (passProvider) AnalyzeChat(context.Context, string, string) (ai.AIResponse, error) {
	return ai.AIResponse{Content: `{"verdict":"PASS","score":95,"review":"Tot.","summary":"Dat.","violations":[]}`, Model: "test-double", Provider: "test-double"}, nil
}

func (passProvider) AnalyzeChatBatch(context.Context, string, []ai.BatchItem) (ai.AIResponse, error) {
	return ai.AIResponse{}, context.Canceled
}

// Capped since-last through the real handler and the real worker: the mode stays since_last (not
// unanalyzed), the cap applies after eligibility, dates are not involved, the checkpoint is kept.
func TestTriggerJobRealRouteRunsTheRequestedModeWithCap(t *testing.T) {
	realTrigger := startTriggerJob
	db.Close()
	f := setupJobDispatchFixture(t)
	startTriggerJob = realTrigger
	origAnalyzer := newTriggerAnalyzer
	newTriggerAnalyzer = func(cfg *config.Config) *engine.Analyzer { return engine.NewAnalyzerWithProvider(cfg, passProvider{}) }
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
	base := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Second)
	conv := func(label string, at time.Time) string {
		id := "conv-" + label + "-" + f.jobID
		exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, 'Khach', ?, 1, '{}', NOW(), NOW())`, id, f.tenantID, chID, "ext-"+id, at)
		exec(`INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, ?, 'customer', 'Khach', ?, 'text', '[]', ?, NOW())`, pkg.NewUUID(), f.tenantID, id, "x-"+id, "Noi dung "+label, at)
		return id
	}
	// No evaluation exists, so since_last behaves as unanalyzed: oldest first, cap 2 of 3.
	first := conv("a", base)
	second := conv("b", base.Add(time.Minute))
	conv("c", base.Add(2*time.Minute))
	sentinel := base.Add(-72 * time.Hour).Truncate(time.Second)
	exec("UPDATE jobs SET last_run_at = ? WHERE id = ?", sentinel, f.jobID)

	rec := f.callTrigger(f.tenantID, "mode=since_last&limit=2")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	var run models.JobRun
	deadline := time.Now().Add(20 * time.Second)
	for {
		err := db.DB.Where("job_id = ?", f.jobID).Order("created_at DESC").First(&run).Error
		if err == nil && run.Status != "running" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("run did not finish: %v %+v", err, run)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if run.Status != "success" {
		t.Fatalf("run status %s (%s)", run.Status, run.ErrorMessage)
	}
	var evaluated []string
	if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ? AND result_type = 'conversation_evaluation'", run.ID).
		Order("conversation_id").Pluck("conversation_id", &evaluated).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Join(evaluated, ",") != first+","+second {
		t.Fatalf("evaluated %v, want %v", evaluated, []string{first, second})
	}
	var job models.Job
	if err := db.DB.First(&job, "id = ?", f.jobID).Error; err != nil {
		t.Fatal(err)
	}
	if job.LastRunAt == nil || !job.LastRunAt.Equal(sentinel) || job.LastRunStatus != "success" {
		t.Fatalf("job after capped run: last_run_at %v status %q, want checkpoint %v preserved", job.LastRunAt, job.LastRunStatus, sentinel)
	}
}
