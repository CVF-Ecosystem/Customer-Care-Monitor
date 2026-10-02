package handlers

import (
	"context"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-028 (F06): the analysis agent runs every job through the shared admission and
// reports bounded results: all busy/failed => error, some completed and some not => partial, all
// completed => success. Permissions and routing are untouched (R020/R021 tests keep covering them).
// Jobs here have no conversations, so a run completes without any provider call.

func TestAnalysisAgentAggregatesSharedAdmission(t *testing.T) {
	db.Close()
	fx := setupJobDispatchFixture(t)
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	// a configured (synthetic) provider so a zero-candidate run completes without a call
	exec(`INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at) VALUES (?, ?, 'ai_api_key', 'synthetic', NOW(), NOW())`, pkg.NewUUID(), fx.tenantID)
	exec(`INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at) VALUES (?, ?, 'ai_provider', 'claude', NOW(), NOW())`, pkg.NewUUID(), fx.tenantID)
	second := "job-jobdisp-2-" + pkg.NewUUID()[:6]
	exec(`INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, 'Second', 'qc_analysis', '[]', '', '[]', 'manual', true, '[]', NOW(), NOW())`, second, fx.tenantID)
	t.Cleanup(func() {
		db.DB.Exec("DELETE FROM activity_logs WHERE tenant_id = ?", fx.tenantID)
		db.DB.Exec("DELETE FROM job_runs WHERE job_id = ?", second)
		db.DB.Exec("DELETE FROM jobs WHERE id = ?", second)
		db.DB.Exec("DELETE FROM app_settings WHERE tenant_id = ?", fx.tenantID)
	})
	req := AgentRunRequest{TenantID: fx.tenantID}
	cfg := &config.Config{}
	ctx := context.Background()
	hold := func(jobID string) *engine.JobRunReservation {
		res, err := engine.ReserveJobRun(ctx, models.Job{ID: jobID, TenantID: fx.tenantID}, 0)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	abort := func(jobID string, res *engine.JobRunReservation) {
		if err := res.Abort(models.Job{ID: jobID, TenantID: fx.tenantID}, "test"); err != nil {
			t.Fatal(err)
		}
	}

	// all jobs free: success
	if got := handleAnalysisAgent(ctx, cfg, req, "cqa.qc"); got.Status != "success" || len(got.Errors) != 0 {
		t.Fatalf("all free: %+v", got)
	}
	// one busy, one free: partial with a bounded code
	a := hold(fx.jobID)
	got := handleAnalysisAgent(ctx, cfg, req, "cqa.qc")
	if got.Status != "partial" || len(got.Errors) != 1 || got.Errors[0] != "job_already_running" {
		t.Fatalf("one busy: %+v", got)
	}
	// all busy: error
	b := hold(second)
	got = handleAnalysisAgent(ctx, cfg, req, "cqa.qc")
	if got.Status != "error" || len(got.Errors) != 2 {
		t.Fatalf("all busy: %+v", got)
	}
	for _, e := range got.Errors {
		if e != "job_already_running" {
			t.Fatalf("unbounded error text %q", e)
		}
	}
	abort(fx.jobID, a)
	abort(second, b)
}
