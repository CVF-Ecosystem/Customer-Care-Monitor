package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-028 old-source probe (cancel targeting). It uses only the pre-F06 handler surface
// (CancelJob with a query string) and direct rows, so the same file compiles against the base source
// and fails there behaviorally: the old handler ignored run_id and bulk-marked EVERY running row of
// the job cancelled (answering 200 job_cancelled), so a stale/unknown target cancelled another run.
func TestF06ProbeCancelNeverTouchesARunItDoesNotOwn(t *testing.T) {
	db.Close()
	connectChannelsTestDB(t)
	s := pkg.NewUUID()[:8]
	tenant, jobID := "f06probe-"+s, "job-f06probe-"+s
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Probe', ?, '{}', NOW(), NOW())`, tenant, tenant)
	exec(`INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, 'Probe', 'qc_analysis', '[]', '', '[]', 'manual', true, '[]', NOW(), NOW())`, jobID, tenant)
	t.Cleanup(func() {
		db.DB.Exec("DELETE FROM job_runs WHERE job_id = ?", jobID)
		db.DB.Exec("DELETE FROM jobs WHERE id = ?", jobID)
		db.DB.Exec("DELETE FROM activity_logs WHERE tenant_id = ?", tenant)
		db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenant)
	})
	a, b := pkg.NewUUID(), pkg.NewUUID()
	for _, id := range []string{a, b} {
		exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'running', '{}', NOW())`, id, jobID, tenant)
	}
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenant)
	c.Params = gin.Params{{Key: "jobId", Value: jobID}}
	c.Request = httptest.NewRequest("POST", "/api/v1/jobs/"+jobID+"/cancel?run_id="+a, nil)
	CancelJob(c)

	var running int64
	db.DB.Raw("SELECT COUNT(*) FROM job_runs WHERE job_id = ? AND status = 'running'", jobID).Scan(&running)
	if rec.Code == http.StatusOK || running != 2 {
		t.Fatalf("a cancel for a run this process does not own must change nothing: %d %s, %d of 2 rows still running",
			rec.Code, rec.Body.String(), running)
	}
}
