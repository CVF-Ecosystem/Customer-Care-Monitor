package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-028 (F06): the real handlers, coordinator and Analyzer on a disposable MySQL with a
// barrier-controlled synthetic provider. No real provider/channel call, no notification (jobs have
// output_schedule none).

type hBarrier struct {
	mu        sync.Mutex
	calls     int
	entered   chan struct{}
	gate      chan struct{}
	ignoreCtx bool
}

func newHBarrier() *hBarrier {
	return &hBarrier{entered: make(chan struct{}, 32), gate: make(chan struct{})}
}

func (b *hBarrier) AnalyzeChat(ctx context.Context, _ string, _ string) (ai.AIResponse, error) {
	b.mu.Lock()
	b.calls++
	b.mu.Unlock()
	b.entered <- struct{}{}
	select {
	case <-b.gate:
	case <-ctx.Done():
		if !b.ignoreCtx {
			return ai.AIResponse{}, ctx.Err()
		}
		<-b.gate
	}
	return passProvider{}.AnalyzeChat(context.Background(), "", "")
}

func (b *hBarrier) AnalyzeChatBatch(context.Context, string, []ai.BatchItem) (ai.AIResponse, error) {
	return ai.AIResponse{}, errors.New("batch not used")
}

func (b *hBarrier) callCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.calls
}

func (b *hBarrier) release() {
	select {
	case <-b.gate:
	default:
		close(b.gate)
	}
}

func (b *hBarrier) waitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-b.entered:
	case <-time.After(30 * time.Second):
		t.Fatal("the provider call never started")
	}
}

// ownFx is the dispatch fixture plus real launchers, a channel with conversations, and the barrier.
type ownFx struct {
	*jobDispatchFixture
	prov   *hBarrier
	chanID string
}

// setupOwnership builds the fixture with the REAL launchers (the dispatch fixture stubs them) and a
// synthetic barrier provider routed through the real Analyzer.
func setupOwnership(t *testing.T, realLaunchers bool, convs int) *ownFx {
	t.Helper()
	realTrigger, realTest := startTriggerJob, startTestRunJob
	db.Close()
	base := setupJobDispatchFixture(t)
	if realLaunchers {
		startTriggerJob, startTestRunJob = realTrigger, realTest
	}
	fx := &ownFx{jobDispatchFixture: base, prov: newHBarrier(), chanID: "ch-" + base.jobID}
	origAnalyzer := newTriggerAnalyzer
	newTriggerAnalyzer = func(cfg *config.Config) *engine.Analyzer { return engine.NewAnalyzerWithProvider(cfg, fx.prov) }
	t.Cleanup(func() { newTriggerAnalyzer = origAnalyzer })

	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh', ?, X'00', true, '{}', NOW(), NOW())`, fx.chanID, base.tenantID, "ext-"+fx.chanID)
	exec(`UPDATE jobs SET input_channel_ids = ?, rules_content = 'Phan hoi dung han.' WHERE id = ?`, `["`+fx.chanID+`"]`, base.jobID)
	exec(`INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at) VALUES (?, ?, 'ai_batch_mode', 'false', NOW(), NOW())`, pkg.NewUUID(), base.tenantID)
	t.Cleanup(func() {
		fx.prov.release() // never leave a worker parked on the barrier
		time.Sleep(200 * time.Millisecond)
		for _, table := range []string{"job_results", "analysis_snapshots", "ai_usage_logs", "messages", "conversations", "app_settings", "channels"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", base.tenantID)
		}
	})
	for i := 0; i < convs; i++ {
		id := fmt.Sprintf("conv-%d-%s", i, base.jobID)
		at := time.Now().UTC().Add(-time.Duration(i+2) * time.Hour)
		exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, 'Khach', ?, 1, '{}', NOW(), NOW())`, id, base.tenantID, fx.chanID, "ext-"+id, at)
		exec(`INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, ?, 'customer', 'Khach', ?, 'text', '[]', ?, NOW())`, pkg.NewUUID(), base.tenantID, id, "x-"+id, "Noi dung "+id, at)
	}
	return fx
}

func (f *jobDispatchFixture) ginCtx(tenantID, method, rawURL string) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenantID)
	c.Params = gin.Params{{Key: "jobId", Value: f.jobID}}
	c.Request = httptest.NewRequest(method, rawURL, nil)
	return rec, c
}

func (f *jobDispatchFixture) callCancel(tenantID, rawQuery string) *httptest.ResponseRecorder {
	u := "/api/v1/jobs/" + f.jobID + "/cancel"
	if rawQuery != "" {
		u += "?" + rawQuery
	}
	rec, c := f.ginCtx(tenantID, "POST", u)
	CancelJob(c)
	return rec
}

func (f *jobDispatchFixture) callDelete(tenantID string) *httptest.ResponseRecorder {
	rec, c := f.ginCtx(tenantID, "DELETE", "/api/v1/jobs/"+f.jobID)
	DeleteJob(c)
	return rec
}

func (f *jobDispatchFixture) callClearRuns(tenantID string) *httptest.ResponseRecorder {
	rec, c := f.ginCtx(tenantID, "DELETE", "/api/v1/jobs/"+f.jobID+"/runs")
	ClearJobRuns(c)
	return rec
}

func bodyField(t *testing.T, rec *httptest.ResponseRecorder, key string) string {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("body %q: %v", rec.Body.String(), err)
	}
	v, _ := m[key].(string)
	return v
}

func (f *jobDispatchFixture) runStatus(t *testing.T, runID string) string {
	t.Helper()
	var r models.JobRun
	if err := db.DB.First(&r, "id = ?", runID).Error; err != nil {
		t.Fatalf("run %s: %v", runID, err)
	}
	return r.Status
}

func (f *jobDispatchFixture) waitRunDone(t *testing.T, runID string) models.JobRun {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		var r models.JobRun
		err := db.DB.First(&r, "id = ?", runID).Error
		if err == nil && r.Status != "running" {
			return r
		}
		if time.Now().After(deadline) {
			t.Fatalf("run %s did not finish: %v %+v", runID, err, r)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// waitIdle waits until the worker has exited and released the slot.
func (f *jobDispatchFixture) waitIdle(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for engine.JobRunActive(f.tenantID, f.jobID) {
		if time.Now().After(deadline) {
			t.Fatal("the slot was never released")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// ---- group B: HTTP reservation ----

func TestRouteReservesBeforeAcceptAndConsumesTheSameRun(t *testing.T) {
	for _, entry := range []string{"trigger", "test-run"} {
		entry := entry
		t.Run(entry, func(t *testing.T) {
			fx := setupOwnership(t, true, 2)
			var rec *httptest.ResponseRecorder
			if entry == "trigger" {
				rec = fx.callTrigger(fx.tenantID, "mode=unanalyzed")
			} else {
				rec = fx.callTestRun(fx.tenantID)
			}
			wantMsg := map[string]string{"trigger": "job_triggered", "test-run": "test_run_started"}[entry]
			runID := bodyField(t, rec, "run_id")
			if rec.Code != http.StatusAccepted || bodyField(t, rec, "message") != wantMsg || runID == "" {
				t.Fatalf("got %d %s", rec.Code, rec.Body.String())
			}
			// 202 means the row and the owner already exist, and they are the worker's own
			if fx.runStatus(t, runID) != "running" || !engine.JobRunActive(fx.tenantID, fx.jobID) || fx.jobRunCount(t) != 1 {
				t.Fatalf("after 202: status %q, owner %v, rows %d", fx.runStatus(t, runID), engine.JobRunActive(fx.tenantID, fx.jobID), fx.jobRunCount(t))
			}
			fx.prov.waitEntered(t)
			fx.prov.release()
			done := fx.waitRunDone(t, runID)
			fx.waitIdle(t)
			if done.Status != "success" || fx.jobRunCount(t) != 1 {
				t.Fatalf("worker run %+v, rows %d (the worker must consume the reservation, not create another row)", done, fx.jobRunCount(t))
			}
			var other int64
			db.DB.Model(&models.JobResult{}).Where("tenant_id = ? AND job_run_id <> ?", fx.tenantID, runID).Count(&other)
			var mine int64
			db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", runID).Count(&mine)
			if other != 0 || mine == 0 {
				t.Fatalf("results: %d for the reserved run, %d for others", mine, other)
			}
		})
	}
}

func TestRouteBusyIs409WithoutAnExtraRowOrProviderCall(t *testing.T) {
	fx := setupOwnership(t, true, 2)
	first := fx.callTrigger(fx.tenantID, "mode=unanalyzed")
	runID := bodyField(t, first, "run_id")
	fx.prov.waitEntered(t)
	calls := fx.prov.callCount()
	loads := fx.cfgLoads
	for name, rec := range map[string]*httptest.ResponseRecorder{
		"trigger":   fx.callTrigger(fx.tenantID, "mode=conditional"),
		"since":     fx.callTrigger(fx.tenantID, ""),
		"test-run":  fx.callTestRun(fx.tenantID),
		"unlimited": fx.callTrigger(fx.tenantID, "full=true"),
	} {
		if rec.Code != http.StatusConflict || bodyField(t, rec, "error") != "job_already_running" {
			t.Fatalf("%s: %d %s", name, rec.Code, rec.Body.String())
		}
	}
	if fx.jobRunCount(t) != 1 || fx.prov.callCount() != calls || fx.cfgLoads < loads {
		t.Fatalf("a busy request created %d rows or a provider call", fx.jobRunCount(t)-1)
	}
	fx.prov.release()
	fx.waitRunDone(t, runID)
	fx.waitIdle(t)
	// and the test-run entry is the one that now gets in
	if rec := fx.callTestRun(fx.tenantID); rec.Code != http.StatusAccepted {
		t.Fatalf("test-run after the owner exited: %d %s", rec.Code, rec.Body.String())
	}
}

// A cancel that arrives before the worker body starts reaches the exact context: the run closes
// cancelled without a single provider call.
func TestRouteImmediateCancelReachesTheReservedWorker(t *testing.T) {
	fx := setupOwnership(t, false, 2) // stub launcher: records the reservation, the body has not started
	rec := fx.callTrigger(fx.tenantID, "mode=unanalyzed")
	runID := bodyField(t, rec, "run_id")
	if rec.Code != http.StatusAccepted || len(fx.reservations) != 1 || fx.reservations[0].RunID() != runID {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	cancel := fx.callCancel(fx.tenantID, "run_id="+runID)
	if cancel.Code != http.StatusAccepted || bodyField(t, cancel, "message") != "job_cancel_requested" || bodyField(t, cancel, "run_id") != runID {
		t.Fatalf("cancel: %d %s", cancel.Code, cancel.Body.String())
	}
	if fx.runStatus(t, runID) != "running" {
		t.Fatal("the handler wrote a terminal state")
	}
	run, err := engine.NewAnalyzerWithProvider(fx.cfg, fx.prov).RunReserved(fx.reservations[0], fx.triggerLaunchedJob, "unanalyzed", 0, "", "")
	if err != nil || run.Status != "cancelled" || fx.prov.callCount() != 0 {
		t.Fatalf("worker body after an early cancel: %v %+v, %d provider calls", err, run, fx.prov.callCount())
	}
	if fx.runStatus(t, runID) != "cancelled" || engine.JobRunActive(fx.tenantID, fx.jobID) {
		t.Fatal("cancelled terminal state/ownership not settled")
	}
}

func TestRouteReservationFaultsNeverAcceptOrLeakWork(t *testing.T) {
	cases := []struct {
		name  string
		fault func(t *testing.T, fx *ownFx)
	}{
		{"insert fails", func(t *testing.T, fx *ownFx) {
			trig := "r028_ins_" + fx.jobID[len(fx.jobID)-6:]
			if err := db.DB.Exec(fmt.Sprintf("CREATE TRIGGER %s BEFORE INSERT ON job_runs FOR EACH ROW BEGIN IF NEW.job_id = '%s' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'synthetic admission failure'; END IF; END", trig, fx.jobID)).Error; err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trig) })
		}},
		{"read/lock fails", func(t *testing.T, fx *ownFx) {
			if err := db.DB.Exec("RENAME TABLE job_runs TO job_runs_r028_hold").Error; err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { db.DB.Exec("RENAME TABLE job_runs_r028_hold TO job_runs") })
		}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			fx := setupOwnership(t, true, 1)
			c.fault(t, fx)
			for name, rec := range map[string]*httptest.ResponseRecorder{
				"trigger": fx.callTrigger(fx.tenantID, "mode=unanalyzed"), "test-run": fx.callTestRun(fx.tenantID),
			} {
				if rec.Code != http.StatusInternalServerError || bodyField(t, rec, "error") != "job_start_failed" {
					t.Fatalf("%s: %d %s", name, rec.Code, rec.Body.String())
				}
				if strings.Contains(rec.Body.String(), "synthetic") || strings.Contains(rec.Body.String(), "Error 1") {
					t.Fatalf("%s: driver text in the response: %s", name, rec.Body.String())
				}
			}
			if engine.JobRunActive(fx.tenantID, fx.jobID) || fx.prov.callCount() != 0 {
				t.Fatal("a failed admission left an owner or called the provider")
			}
		})
	}
}

// A setup failure after the reservation is closed through the checked finalizer; if even that cannot
// be recorded the running row stays and keeps blocking admission (never an orphan accepted start).
func TestRoutePostReservationSetupFailureIsCheckedOrBlocks(t *testing.T) {
	for _, finalizeFails := range []bool{false, true} {
		finalizeFails := finalizeFails
		t.Run(fmt.Sprintf("finalize fails=%v", finalizeFails), func(t *testing.T) {
			fx := setupOwnership(t, false, 1)
			startTriggerJob = func(job models.Job, cfg *config.Config, p triggerJobParams, res *engine.JobRunReservation) error {
				return errors.New("synthetic launcher failure")
			}
			if finalizeFails {
				trig := "r028_run_" + fx.jobID[len(fx.jobID)-6:]
				if err := db.DB.Exec(fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE ON job_runs FOR EACH ROW BEGIN IF NEW.job_id = '%s' AND NEW.status <> 'running' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'synthetic finalize failure'; END IF; END", trig, fx.jobID)).Error; err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trig) })
			}
			rec := fx.callTrigger(fx.tenantID, "mode=unanalyzed")
			if rec.Code != http.StatusInternalServerError || bodyField(t, rec, "error") != "job_start_failed" {
				t.Fatalf("%d %s", rec.Code, rec.Body.String())
			}
			if engine.JobRunActive(fx.tenantID, fx.jobID) {
				t.Fatal("the failed start kept the local slot")
			}
			var runs []models.JobRun
			db.DB.Where("job_id = ?", fx.jobID).Find(&runs)
			if len(runs) != 1 {
				t.Fatalf("%d rows", len(runs))
			}
			if finalizeFails {
				if runs[0].Status != "running" {
					t.Fatalf("an unrecordable abort must leave the row running, got %q", runs[0].Status)
				}
				// restore a working launcher: admission stays blocked by the unresolved row
				startTriggerJob = func(job models.Job, cfg *config.Config, p triggerJobParams, res *engine.JobRunReservation) error {
					fx.reservations = append(fx.reservations, res)
					return nil
				}
				if again := fx.callTrigger(fx.tenantID, "mode=unanalyzed"); again.Code != http.StatusConflict {
					t.Fatalf("an unresolved row must block: %d %s", again.Code, again.Body.String())
				}
			} else if runs[0].Status != "error" {
				t.Fatalf("a recordable abort must close the row as error, got %q", runs[0].Status)
			}
		})
	}
}

// ---- group C: cancellation over HTTP ----

func TestRouteCancelClassification(t *testing.T) {
	fx := setupOwnership(t, false, 1)
	// nothing running
	if rec := fx.callCancel(fx.tenantID, ""); rec.Code != http.StatusConflict || bodyField(t, rec, "error") != "job_not_running" {
		t.Fatalf("idle: %d %s", rec.Code, rec.Body.String())
	}
	// malformed ids
	for name, q := range map[string]string{
		"empty": "run_id=", "malformed": "run_id=not-a-uuid", "duplicate": "run_id=" + pkg.NewUUID() + "&run_id=" + pkg.NewUUID(),
		"sql-ish": "run_id=" + "1%27%20OR%201=1",
	} {
		if rec := fx.callCancel(fx.tenantID, q); rec.Code != http.StatusBadRequest || bodyField(t, rec, "error") != "invalid_run_id" {
			t.Fatalf("%s: %d %s", name, rec.Code, rec.Body.String())
		}
	}
	// wrong tenant / unknown job
	if rec := fx.callCancel(fx.otherTenantID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("other tenant: %d", rec.Code)
	}
	// stored running row without a local owner
	stored := pkg.NewUUID()
	if err := db.DB.Exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'running', '{}', NOW())`, stored, fx.jobID, fx.tenantID).Error; err != nil {
		t.Fatal(err)
	}
	if rec := fx.callCancel(fx.tenantID, "run_id="+stored); rec.Code != http.StatusConflict || bodyField(t, rec, "error") != "job_run_not_owned" {
		t.Fatalf("unowned: %d %s", rec.Code, rec.Body.String())
	}
	db.DB.Exec("UPDATE job_runs SET status = 'error', finished_at = NOW() WHERE id = ?", stored)
	// terminal id
	if rec := fx.callCancel(fx.tenantID, "run_id="+stored); rec.Code != http.StatusConflict || bodyField(t, rec, "error") != "job_not_running" {
		t.Fatalf("terminal: %d %s", rec.Code, rec.Body.String())
	}
}

func TestRouteCancelEndToEndHoldsTheSlotUntilTheWorkerExits(t *testing.T) {
	fx := setupOwnership(t, true, 2)
	fx.prov.ignoreCtx = true
	rec := fx.callTrigger(fx.tenantID, "mode=unanalyzed")
	runID := bodyField(t, rec, "run_id")
	fx.prov.waitEntered(t)

	// stale/other ids never cancel; the exact id, a repeat and the legacy form are accepted
	if rec := fx.callCancel(fx.tenantID, "run_id="+pkg.NewUUID()); rec.Code != http.StatusConflict || bodyField(t, rec, "error") != "job_not_running" {
		t.Fatalf("stale id: %d %s", rec.Code, rec.Body.String())
	}
	for _, q := range []string{"run_id=" + runID, "run_id=" + runID, ""} {
		rec := fx.callCancel(fx.tenantID, q)
		if rec.Code != http.StatusAccepted || bodyField(t, rec, "message") != "job_cancel_requested" || bodyField(t, rec, "run_id") != runID {
			t.Fatalf("cancel %q: %d %s", q, rec.Code, rec.Body.String())
		}
	}
	if fx.runStatus(t, runID) != "running" {
		t.Fatal("202 job_cancel_requested must not be a terminal write")
	}
	if again := fx.callTrigger(fx.tenantID, "mode=unanalyzed"); again.Code != http.StatusConflict {
		t.Fatalf("the slot was freed at the cancel request: %d", again.Code)
	}
	calls := fx.prov.callCount()
	fx.prov.release() // the provider ignored ctx and answers late
	done := fx.waitRunDone(t, runID)
	fx.waitIdle(t)
	var results, snaps int64
	db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", runID).Count(&results)
	db.DB.Model(&models.AnalysisSnapshot{}).Where("job_run_id = ?", runID).Count(&snaps)
	if done.Status != "cancelled" || results != 0 || snaps != 0 || fx.prov.callCount() != calls {
		t.Fatalf("late output after an accepted cancel: status %s, %d results, %d snapshots, calls %d->%d", done.Status, results, snaps, calls, fx.prov.callCount())
	}
	if next := fx.callTrigger(fx.tenantID, "mode=unanalyzed"); next.Code != http.StatusAccepted {
		t.Fatalf("admission after the worker exited: %d %s", next.Code, next.Body.String())
	}
}

// A failed checked read reports 500 and signals nothing.
func TestRouteCancelCheckFailureIs500AndSignalsNothing(t *testing.T) {
	fx := setupOwnership(t, false, 1)
	rec := fx.callTrigger(fx.tenantID, "mode=unanalyzed")
	runID := bodyField(t, rec, "run_id")
	res := fx.reservations[0]
	if err := db.DB.Exec("RENAME TABLE job_runs TO job_runs_r028_hold").Error; err != nil {
		t.Fatal(err)
	}
	restored := false
	restore := func() {
		if !restored {
			restored = true
			db.DB.Exec("RENAME TABLE job_runs_r028_hold TO job_runs")
		}
	}
	t.Cleanup(restore)
	cancel := fx.callCancel(fx.tenantID, "run_id="+runID)
	restore()
	if cancel.Code != http.StatusInternalServerError || bodyField(t, cancel, "error") != "job_cancel_failed" || strings.Contains(cancel.Body.String(), "Error 1") {
		t.Fatalf("%d %s", cancel.Code, cancel.Body.String())
	}
	if res.Context().Err() != nil {
		t.Fatal("a failed cancel check signalled the worker context")
	}
}

// ---- group E: destructive guards ----

func TestRouteDeleteAndClearRunsRefuseAnActiveRun(t *testing.T) {
	for _, kind := range []string{"local owner", "stored running row"} {
		kind := kind
		t.Run(kind, func(t *testing.T) {
			fx := setupOwnership(t, false, 1)
			var res *engine.JobRunReservation
			if kind == "local owner" {
				var err error
				res, err = engine.ReserveJobRun(context.Background(), models.Job{ID: fx.jobID, TenantID: fx.tenantID}, 0)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = res.Abort(models.Job{ID: fx.jobID, TenantID: fx.tenantID}, "test") })
			} else if err := db.DB.Exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'running', '{}', NOW())`, pkg.NewUUID(), fx.jobID, fx.tenantID).Error; err != nil {
				t.Fatal(err)
			}
			// something that a destructive path would erase
			if err := db.DB.Exec(`UPDATE jobs SET last_run_status = 'success' WHERE id = ?`, fx.jobID).Error; err != nil {
				t.Fatal(err)
			}
			rowsBefore := fx.jobRunCount(t)
			for name, rec := range map[string]*httptest.ResponseRecorder{"delete": fx.callDelete(fx.tenantID), "clear runs": fx.callClearRuns(fx.tenantID)} {
				if rec.Code != http.StatusConflict || bodyField(t, rec, "error") != "job_already_running" {
					t.Fatalf("%s: %d %s", name, rec.Code, rec.Body.String())
				}
			}
			var job models.Job
			if err := db.DB.First(&job, "id = ?", fx.jobID).Error; err != nil || job.LastRunStatus != "success" || fx.jobRunCount(t) != rowsBefore {
				t.Fatalf("a refused destructive call changed state: %v %+v rows %d->%d", err, job, rowsBefore, fx.jobRunCount(t))
			}
		})
	}
}

func TestRouteDeleteAndClearRunsStillCascadeWhenIdle(t *testing.T) {
	fx := setupOwnership(t, false, 1)
	if err := db.DB.Exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, finished_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), NOW(), 'success', '{}', NOW())`, pkg.NewUUID(), fx.jobID, fx.tenantID).Error; err != nil {
		t.Fatal(err)
	}
	if rec := fx.callClearRuns(fx.tenantID); rec.Code != http.StatusOK || fx.jobRunCount(t) != 0 {
		t.Fatalf("clear runs: %d %s rows %d", rec.Code, rec.Body.String(), fx.jobRunCount(t))
	}
	if rec := fx.callDelete(fx.tenantID); rec.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}
	var n int64
	db.DB.Model(&models.Job{}).Where("id = ?", fx.jobID).Count(&n)
	if n != 0 {
		t.Fatal("the idle job was not deleted")
	}
}

// Admission and DeleteJob serialize on the Job parent: a delete holding the parent makes the
// admission wait (its local owner is already visible, so the delete guard would refuse), and after
// the delete commits the admission finds no job. The other order is refused with 409.
func TestAdmissionSerializesWithDeleteOnTheJobParent(t *testing.T) {
	fx := setupOwnership(t, false, 0)
	job := models.Job{ID: fx.jobID, TenantID: fx.tenantID}
	tx := db.DB.Begin()
	if err := engine.GuardJobMutation(tx, fx.tenantID, fx.jobID); err != nil {
		t.Fatalf("guard on an idle job: %v", err)
	}
	type outcome struct {
		res *engine.JobRunReservation
		err error
	}
	got := make(chan outcome, 1)
	go func() {
		res, err := engine.ReserveJobRun(context.Background(), job, 0)
		got <- outcome{res, err}
	}()
	// the admission is parked on the parent lock; its owner is already visible to the guards
	deadline := time.Now().Add(10 * time.Second)
	for !engine.JobRunActive(fx.tenantID, fx.jobID) {
		if time.Now().After(deadline) {
			t.Fatal("the pending admission never became visible")
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case o := <-got:
		t.Fatalf("admission completed while the delete held the parent: %+v", o)
	case <-time.After(300 * time.Millisecond):
	}
	if err := tx.Exec("DELETE FROM jobs WHERE id = ? AND tenant_id = ?", fx.jobID, fx.tenantID).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
	select {
	case o := <-got:
		if !errors.Is(o.err, engine.ErrJobMissing) || o.res != nil {
			t.Fatalf("admission after the delete: %+v", o)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the admission never resumed")
	}
	if engine.JobRunActive(fx.tenantID, fx.jobID) || fx.jobRunCount(t) != 0 {
		t.Fatal("a refused admission leaked an owner or row")
	}
}
