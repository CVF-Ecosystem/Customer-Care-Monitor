package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-009: TestRunJob and TriggerJob admit dispatch only after a
// validated configuration, checked after the tenant-scoped job lookup and
// before any goroutine is launched or cancellation is registered. Config
// loader and launchers are stubbed by setupJobDispatchFixture; no analyzer,
// adapter or provider is ever called by these tests.

type jobDispatchFixture struct {
	tenantID, otherTenantID, jobID string

	cfg      *config.Config
	cfgErr   error
	cfgLoads int

	testRunLaunches    int
	testRunLaunchedCfg *config.Config
	testRunLaunchedJob models.Job
	testRunLimit       int

	triggerLaunches    int
	triggerLaunchedCfg *config.Config
	triggerLaunchedJob models.Job
	triggerParams      triggerJobParams
	reservations       []*engine.JobRunReservation
}

func setupJobDispatchFixture(t *testing.T) *jobDispatchFixture {
	t.Helper()
	connectChannelsTestDB(t)
	s := pkg.NewUUID()[:8]
	f := &jobDispatchFixture{tenantID: "jobdisp-" + s, otherTenantID: "jobdisp-other-" + s, jobID: "job-jobdisp-" + s}
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	for _, tenant := range []string{f.tenantID, f.otherTenantID} {
		exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Job Dispatch', ?, '{}', NOW(), NOW())`, tenant, tenant)
	}
	exec(`INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, 'Dispatch Test', 'qc_analysis', '[]', '', '[]', 'manual', true, '[]', NOW(), NOW())`,
		f.jobID, f.tenantID)
	t.Cleanup(func() {
		// release any reservation a stub launcher received (CCMAI-RUNTIME-028)
		for _, res := range f.reservations {
			_ = res.Abort(models.Job{ID: f.jobID, TenantID: f.tenantID}, "test cleanup")
		}
		db.DB.Exec("DELETE FROM job_runs WHERE job_id = ?", f.jobID)
		db.DB.Exec("DELETE FROM jobs WHERE id = ?", f.jobID)
		for _, tenant := range []string{f.tenantID, f.otherTenantID} {
			db.DB.Exec("DELETE FROM activity_logs WHERE tenant_id = ?", tenant)
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenant)
		}
	})
	if f.jobRunCount(t) != 0 {
		t.Fatalf("fixture job starts with job_runs rows")
	}
	if engine.JobRunActive(f.tenantID, f.jobID) {
		t.Fatalf("fixture job starts with a local owner")
	}

	// The handler now reserves (owner + running row) before it launches, so the stub launchers only
	// record the reservation; the rejection assertions stay non-vacuous because an accepted launch
	// leaves exactly that owner and row (CCMAI-RUNTIME-028).
	originalTestRun := startTestRunJob
	startTestRunJob = func(job models.Job, cfg *config.Config, limit int, res *engine.JobRunReservation) error {
		f.testRunLaunches++
		f.testRunLaunchedCfg = cfg
		f.testRunLaunchedJob = job
		f.testRunLimit = limit
		f.reservations = append(f.reservations, res)
		return nil
	}
	t.Cleanup(func() { startTestRunJob = originalTestRun })

	originalTrigger := startTriggerJob
	startTriggerJob = func(job models.Job, cfg *config.Config, p triggerJobParams, res *engine.JobRunReservation) error {
		f.triggerLaunches++
		f.triggerLaunchedCfg = cfg
		f.triggerLaunchedJob = job
		f.triggerParams = p
		f.reservations = append(f.reservations, res)
		return nil
	}
	t.Cleanup(func() { startTriggerJob = originalTrigger })

	f.cfg = &config.Config{Env: "test"} // synthetic; no real secrets or credentials
	originalLoad := loadJobDispatchConfig
	loadJobDispatchConfig = func() (*config.Config, error) {
		f.cfgLoads++
		if f.cfgErr != nil {
			return nil, f.cfgErr
		}
		return f.cfg, nil
	}
	t.Cleanup(func() { loadJobDispatchConfig = originalLoad })
	return f
}

func (f *jobDispatchFixture) callTestRun(tenantID string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenantID)
	c.Params = gin.Params{{Key: "jobId", Value: f.jobID}}
	c.Request = httptest.NewRequest("POST", "/api/v1/jobs/"+f.jobID+"/test-run", nil)
	TestRunJob(c)
	return rec
}

func (f *jobDispatchFixture) callTrigger(tenantID, query string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenantID)
	c.Params = gin.Params{{Key: "jobId", Value: f.jobID}}
	url := "/api/v1/jobs/" + f.jobID + "/trigger"
	if query != "" {
		url += "?" + query
	}
	c.Request = httptest.NewRequest("POST", url, nil)
	TriggerJob(c)
	return rec
}

func (f *jobDispatchFixture) jobRunCount(t *testing.T) int64 {
	t.Helper()
	var n int64
	if err := db.DB.Model(&models.JobRun{}).Where("job_id = ?", f.jobID).Count(&n).Error; err != nil {
		t.Fatalf("count job_runs: %v", err)
	}
	return n
}

// assertNoRunOrCancel checks the rejection side effects directly: no job run
// row and no cancel handle exist for the fixture job.
func (f *jobDispatchFixture) assertNoRunOrCancel(t *testing.T) {
	t.Helper()
	if n := f.jobRunCount(t); n != 0 {
		t.Fatalf("%d job_runs rows created despite rejection", n)
	}
	if engine.JobRunActive(f.tenantID, f.jobID) {
		t.Fatalf("local owner registered despite rejection")
	}
}

// assertWorkerStarted proves the side-effect detectors fire when a launch
// does happen, so their absence on rejection is meaningful.
func (f *jobDispatchFixture) assertWorkerStarted(t *testing.T) {
	t.Helper()
	if n := f.jobRunCount(t); n != 1 {
		t.Fatalf("job_runs rows %d after accepted launch, want 1", n)
	}
	if !engine.JobRunActive(f.tenantID, f.jobID) {
		t.Fatalf("no local owner after accepted launch")
	}
}

func (f *jobDispatchFixture) assertNoTestRunStart(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code < 400 {
		t.Fatalf("status %d, want non-2xx; body %s", rec.Code, rec.Body.String())
	}
	if f.testRunLaunches != 0 {
		t.Fatalf("test-run worker launched %d times despite rejection", f.testRunLaunches)
	}
	f.assertNoRunOrCancel(t)
}

func (f *jobDispatchFixture) assertNoTriggerStart(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code < 400 {
		t.Fatalf("status %d, want non-2xx; body %s", rec.Code, rec.Body.String())
	}
	if f.triggerLaunches != 0 {
		t.Fatalf("trigger worker launched %d times despite rejection", f.triggerLaunches)
	}
	f.assertNoRunOrCancel(t)
}

// --- TestRunJob ---

func TestTestRunJobConfigFailureStartsNoWorker(t *testing.T) {
	f := setupJobDispatchFixture(t)
	f.cfgErr = errors.New("ENCRYPTION_KEY=SECRET-DO-NOT-LEAK is invalid")
	logs := captureLog(t)

	rec := f.callTestRun(f.tenantID)

	f.assertNoTestRunStart(t, rec)
	if f.cfgLoads != 1 {
		t.Fatalf("config loaded %d times, want 1", f.cfgLoads)
	}
	body := rec.Body.String()
	if strings.TrimSpace(body) != `{"error":"job_start_failed"}` {
		t.Fatalf("response %q, want generic job_start_failed", body)
	}
	for _, leak := range []string{"SECRET", "ENCRYPTION_KEY"} {
		if strings.Contains(body, leak) || strings.Contains(logs.String(), leak) {
			t.Fatalf("config error detail %q leaked; body %q, log %q", leak, body, logs.String())
		}
	}
	if !strings.Contains(logs.String(), "configuration invalid") {
		t.Fatalf("log should name the failure class: %s", logs.String())
	}
}

func TestTestRunJobNilConfigIsNotAdmitted(t *testing.T) {
	f := setupJobDispatchFixture(t)
	f.cfg = nil
	f.assertNoTestRunStart(t, f.callTestRun(f.tenantID))
}

func TestTestRunJobPassesValidatedConfigToWorker(t *testing.T) {
	f := setupJobDispatchFixture(t)
	rec := f.callTestRun(f.tenantID)

	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), `"message":"test_run_started"`) || len(f.reservations) != 1 ||
		!strings.Contains(rec.Body.String(), `"run_id":"`+f.reservations[0].RunID()+`"`) {
		t.Fatalf("got %d %s, want 202 test_run_started with the reserved run_id", rec.Code, rec.Body.String())
	}
	if f.cfgLoads != 1 || f.testRunLaunches != 1 {
		t.Fatalf("config loads %d, launches %d; want 1 and 1", f.cfgLoads, f.testRunLaunches)
	}
	if f.testRunLaunchedCfg != f.cfg {
		t.Fatalf("worker got config %p, want the validated config %p", f.testRunLaunchedCfg, f.cfg)
	}
	if f.testRunLaunchedJob.ID != f.jobID {
		t.Fatalf("worker got job %q, want %q", f.testRunLaunchedJob.ID, f.jobID)
	}
	if f.testRunLimit != 3 {
		t.Fatalf("test-run launched with limit %d, want 3", f.testRunLimit)
	}
	f.assertWorkerStarted(t)
}

func TestTestRunJobWrongTenantSkipsConfigLoad(t *testing.T) {
	f := setupJobDispatchFixture(t)
	rec := f.callTestRun(f.otherTenantID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("other tenant got %d, want 404", rec.Code)
	}
	if f.cfgLoads != 0 {
		t.Fatalf("config loaded %d times before the tenant check", f.cfgLoads)
	}
	f.assertNoTestRunStart(t, rec)
}

// --- TriggerJob ---

func TestTriggerJobConfigFailureStartsNoWorker(t *testing.T) {
	f := setupJobDispatchFixture(t)
	f.cfgErr = errors.New("ENCRYPTION_KEY=SECRET-DO-NOT-LEAK is invalid")
	logs := captureLog(t)

	rec := f.callTrigger(f.tenantID, "")

	f.assertNoTriggerStart(t, rec)
	if f.cfgLoads != 1 {
		t.Fatalf("config loaded %d times, want 1", f.cfgLoads)
	}
	body := rec.Body.String()
	if strings.TrimSpace(body) != `{"error":"job_start_failed"}` {
		t.Fatalf("response %q, want generic job_start_failed", body)
	}
	for _, leak := range []string{"SECRET", "ENCRYPTION_KEY"} {
		if strings.Contains(body, leak) || strings.Contains(logs.String(), leak) {
			t.Fatalf("config error detail %q leaked; body %q, log %q", leak, body, logs.String())
		}
	}
	if !strings.Contains(logs.String(), "configuration invalid") {
		t.Fatalf("log should name the failure class: %s", logs.String())
	}
}

func TestTriggerJobNilConfigIsNotAdmitted(t *testing.T) {
	f := setupJobDispatchFixture(t)
	f.cfg = nil
	f.assertNoTriggerStart(t, f.callTrigger(f.tenantID, ""))
}

func TestTriggerJobPassesValidatedConfigAndParamsToWorker(t *testing.T) {
	f := setupJobDispatchFixture(t)
	rec := f.callTrigger(f.tenantID, "mode=conditional&from=2026-01-01&to=2026-01-31&limit=7")

	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), `"message":"job_triggered"`) || len(f.reservations) != 1 ||
		!strings.Contains(rec.Body.String(), `"run_id":"`+f.reservations[0].RunID()+`"`) {
		t.Fatalf("got %d %s, want 202 job_triggered with the reserved run_id", rec.Code, rec.Body.String())
	}
	if f.cfgLoads != 1 || f.triggerLaunches != 1 {
		t.Fatalf("config loads %d, launches %d; want 1 and 1", f.cfgLoads, f.triggerLaunches)
	}
	if f.triggerLaunchedCfg != f.cfg {
		t.Fatalf("worker got config %p, want the validated config %p", f.triggerLaunchedCfg, f.cfg)
	}
	if f.triggerLaunchedJob.ID != f.jobID {
		t.Fatalf("worker got job %q, want %q", f.triggerLaunchedJob.ID, f.jobID)
	}
	want := triggerJobParams{mode: "conditional", dateFrom: "2026-01-01", dateTo: "2026-01-31", maxConv: 7}
	if f.triggerParams != want {
		t.Fatalf("worker got params %+v, want %+v", f.triggerParams, want)
	}
	f.assertWorkerStarted(t)
}

func TestTriggerJobDefaultModeAndLimitReachWorkerUnchanged(t *testing.T) {
	f := setupJobDispatchFixture(t)
	rec := f.callTrigger(f.tenantID, "")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("got %d, want 202", rec.Code)
	}
	want := triggerJobParams{mode: "since_last", dateFrom: "", dateTo: "", maxConv: 0}
	if f.triggerParams != want {
		t.Fatalf("default params %+v, want %+v", f.triggerParams, want)
	}
}

func TestTriggerJobWrongTenantSkipsConfigLoad(t *testing.T) {
	f := setupJobDispatchFixture(t)
	rec := f.callTrigger(f.otherTenantID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("other tenant got %d, want 404", rec.Code)
	}
	if f.cfgLoads != 0 {
		t.Fatalf("config loaded %d times before the tenant check", f.cfgLoads)
	}
	f.assertNoTriggerStart(t, rec)
}

// TestJobDispatchUsesRealConfigValidation runs the real config.Load with
// synthetic environment values (t.Setenv restores them; no t.Parallel).
func TestJobDispatchUsesRealConfigValidation(t *testing.T) {
	const jwt = "synthetic-jwt-secret-for-tests-0123456789"
	const encKey = "synthetic-32-byte-key-0123456789" // exactly 32 bytes

	t.Run("invalid", func(t *testing.T) {
		f := setupJobDispatchFixture(t)
		loadJobDispatchConfig = config.Load // restored by the fixture cleanup
		t.Setenv("JWT_SECRET", "too-short")
		t.Setenv("ENCRYPTION_KEY", encKey)
		t.Setenv("DB_PASSWORD", "synthetic")
		logs := captureLog(t)

		rec := f.callTrigger(f.tenantID, "")
		f.assertNoTriggerStart(t, rec)
		if strings.Contains(rec.Body.String(), "JWT") || strings.Contains(logs.String(), "JWT") {
			t.Fatalf("validation detail leaked; body %q, log %q", rec.Body.String(), logs.String())
		}
	})

	t.Run("valid", func(t *testing.T) {
		f := setupJobDispatchFixture(t)
		loadJobDispatchConfig = config.Load
		t.Setenv("JWT_SECRET", jwt)
		t.Setenv("ENCRYPTION_KEY", encKey)
		t.Setenv("DB_PASSWORD", "synthetic")

		rec := f.callTestRun(f.tenantID)
		if rec.Code != http.StatusAccepted || f.testRunLaunches != 1 {
			t.Fatalf("got %d with %d launches, want 202 and 1", rec.Code, f.testRunLaunches)
		}
		if f.testRunLaunchedCfg == nil || f.testRunLaunchedCfg.JWTSecret != jwt || f.testRunLaunchedCfg.EncryptionKey != encKey {
			t.Fatalf("worker did not receive the config validated from the synthetic environment")
		}
	})
}
