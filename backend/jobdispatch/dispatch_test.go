package jobdispatch

import (
	"bytes"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-044: the shared dispatch service on a disposable MySQL with synthetic rows. No
// provider, channel, notification or real configuration is touched: configuration and worker start
// are injected, and the only worker that runs is a panicking stub.

type fixture struct {
	tenantID, jobID, otherJobID string
	reservations                []*engine.JobRunReservation
	mu                          sync.Mutex
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("bo qua: TEST_DB_DSN chua duoc thiet lap")
	}
	db.Close()
	if err := db.Connect(dsn, false); err != nil {
		t.Skipf("bo qua: khong ket noi duoc DB test: %v", err)
	}
	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate loi: %v", err)
	}
	s := pkg.NewUUID()[:8]
	f := &fixture{tenantID: "jd-" + s, jobID: "job-jd-" + s, otherJobID: "job-jd2-" + s}
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Dispatch', ?, '{}', NOW(), NOW())`, f.tenantID, f.tenantID)
	for _, id := range []string{f.jobID, f.otherJobID} {
		exec(`INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, 'Dispatch Test', 'qc_analysis', '[]', '', '[]', 'manual', true, '[]', NOW(), NOW())`, id, f.tenantID)
	}
	t.Cleanup(func() {
		for _, res := range f.reservations {
			_ = res.Abort(models.Job{}, "test cleanup") // idempotent; releases any parked owner
		}
		db.DB.Exec("DELETE FROM activity_logs WHERE tenant_id = ?", f.tenantID)
		db.DB.Exec("DELETE FROM job_runs WHERE tenant_id = ?", f.tenantID)
		db.DB.Exec("DELETE FROM jobs WHERE tenant_id = ?", f.tenantID)
		db.DB.Exec("DELETE FROM tenants WHERE id = ?", f.tenantID)
	})
	return f
}

func (f *fixture) job(id string) models.Job {
	return models.Job{ID: id, TenantID: f.tenantID, Name: "Dispatch Test"}
}

func (f *fixture) runs(t *testing.T, jobID string) int64 {
	t.Helper()
	var n int64
	if err := db.DB.Model(&models.JobRun{}).Where("job_id = ?", jobID).Count(&n).Error; err != nil {
		t.Fatalf("count runs: %v", err)
	}
	return n
}

func (f *fixture) run(t *testing.T, runID string) models.JobRun {
	t.Helper()
	var r models.JobRun
	if err := db.DB.First(&r, "id = ?", runID).Error; err != nil {
		t.Fatalf("run %s: %v", runID, err)
	}
	return r
}

func okConfig() (*config.Config, error) { return &config.Config{Env: "test"}, nil }

// parkStart records the reservation and starts nothing (the reservation stays owned).
func (f *fixture) parkStart(starts *int) StartFunc {
	return func(_ models.Job, _ *config.Config, _ Params, res *engine.JobRunReservation) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		*starts++
		f.reservations = append(f.reservations, res)
		return nil
	}
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

func failQueries(t *testing.T, table, name string) (disable func()) {
	t.Helper()
	enabled := true
	var mu sync.Mutex
	if err := db.DB.Callback().Query().Before("gorm:query").Register(name, func(tx *gorm.DB) {
		mu.Lock()
		on := enabled
		mu.Unlock()
		if on && tx.Statement.Table == table {
			tx.AddError(errors.New("injected " + name + " SECRET-DRIVER-DETAIL"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	remove := func() { db.DB.Callback().Query().Remove(name) }
	t.Cleanup(remove)
	return func() { mu.Lock(); enabled = false; mu.Unlock(); remove() }
}

// failUpdates makes every UPDATE of table fail, including the engine's best-effort fallback mark,
// which bypasses the query callbacks (a read-only fault does not force a failed abort).
func failUpdates(t *testing.T, table, name string) (disable func()) {
	t.Helper()
	if err := db.DB.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == table {
			tx.AddError(errors.New("injected " + name + " SECRET-DRIVER-DETAIL"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	remove := func() { db.DB.Callback().Update().Remove(name) }
	t.Cleanup(remove)
	return remove
}

// JE-04: an invalid or nil configuration is refused before any reservation, worker or log detail.
func TestConfigFailureReservesNothing(t *testing.T) {
	f := newFixture(t)
	for name, load := range map[string]func() (*config.Config, error){
		"error": func() (*config.Config, error) { return nil, errors.New("ENCRYPTION_KEY=SECRET-DO-NOT-LEAK is invalid") },
		"nil":   func() (*config.Config, error) { return nil, nil },
	} {
		logs := captureLog(t)
		starts := 0
		svc := &Service{Label: "trigger", LoadConfig: load, Start: f.parkStart(&starts)}
		runID, err := svc.Dispatch(f.job(f.jobID), Params{Mode: "since_last"})
		if !errors.Is(err, ErrStartFailed) || runID != "" {
			t.Fatalf("%s: got run %q err %v, want ErrStartFailed", name, runID, err)
		}
		if starts != 0 || f.runs(t, f.jobID) != 0 || engine.JobRunActive(f.tenantID, f.jobID) {
			t.Fatalf("%s: starts %d runs %d owner %v; nothing may be reserved", name, starts, f.runs(t, f.jobID), engine.JobRunActive(f.tenantID, f.jobID))
		}
		if strings.Contains(logs.String(), "SECRET") || !strings.Contains(logs.String(), "configuration invalid") {
			t.Fatalf("%s: log %q must name the failure class without secret detail", name, logs.String())
		}
	}
}

// JE-03: success returns the persisted reservation's identity; the worker gets the exact parameters
// and a bounded context that belongs to the reservation, not to any request.
func TestDispatchReturnsPersistedReservationIdentity(t *testing.T) {
	f := newFixture(t)
	var gotParams Params
	var gotCfg *config.Config
	var deadline time.Time
	var hasDeadline bool
	var res0 *engine.JobRunReservation
	svc := &Service{Label: "trigger", Timeout: 90 * time.Second, LoadConfig: okConfig,
		Start: func(_ models.Job, cfg *config.Config, p Params, res *engine.JobRunReservation) error {
			gotParams, gotCfg, res0 = p, cfg, res
			deadline, hasDeadline = res.Context().Deadline()
			f.reservations = append(f.reservations, res)
			return nil
		}}
	want := Params{Mode: "conditional", DateFrom: "2026-10-01", DateTo: "2026-10-02", Limit: 7}
	before := time.Now()
	runID, err := svc.Dispatch(f.job(f.jobID), want)
	if err != nil || runID == "" {
		t.Fatalf("dispatch: %q %v", runID, err)
	}
	if res0 == nil || res0.RunID() != runID {
		t.Fatalf("returned run id %q is not the handed-off reservation's id", runID)
	}
	if gotParams != want || gotCfg == nil || gotCfg.Env != "test" {
		t.Fatalf("worker got %+v / %+v, want the parameters and validated config unchanged", gotParams, gotCfg)
	}
	row := f.run(t, runID)
	if row.Status != "running" || row.JobID != f.jobID || row.TenantID != f.tenantID || f.runs(t, f.jobID) != 1 {
		t.Fatalf("persisted row %+v, runs %d", row, f.runs(t, f.jobID))
	}
	if !engine.JobRunActive(f.tenantID, f.jobID) {
		t.Fatal("the local owner must exist before Dispatch returns")
	}
	if !hasDeadline || deadline.Before(before.Add(80*time.Second)) || deadline.After(time.Now().Add(91*time.Second)) {
		t.Fatalf("reservation deadline %v (has=%v) must be the configured bounded timeout", deadline, hasDeadline)
	}
	if res0.Context().Err() != nil {
		t.Fatalf("the reservation context must be live: %v", res0.Context().Err())
	}
}

// The zero Timeout falls back to the 30-minute bound (never unbounded).
func TestDefaultTimeoutIsBounded(t *testing.T) {
	f := newFixture(t)
	var deadline time.Time
	var has bool
	svc := &Service{LoadConfig: okConfig, Start: func(_ models.Job, _ *config.Config, _ Params, res *engine.JobRunReservation) error {
		deadline, has = res.Context().Deadline()
		f.reservations = append(f.reservations, res)
		return nil
	}}
	if _, err := svc.Dispatch(f.job(f.jobID), Params{Mode: "since_last"}); err != nil {
		t.Fatal(err)
	}
	if !has || time.Until(deadline) > DefaultTimeout || time.Until(deadline) < DefaultTimeout-time.Minute {
		t.Fatalf("zero Timeout must use DefaultTimeout %v, deadline in %v (has=%v)", DefaultTimeout, time.Until(deadline), has)
	}
}

// JE-05: concurrent dispatches of one tenant/job reserve and start exactly once; other jobs are
// independent; a stored running row without a local owner blocks.
func TestConcurrentDispatchHasOneOwner(t *testing.T) {
	f := newFixture(t)
	starts := 0
	svc := &Service{LoadConfig: okConfig, Start: f.parkStart(&starts)}
	const n = 8
	gate := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			_, err := svc.Dispatch(f.job(f.jobID), Params{Mode: "since_last"})
			results <- err
		}()
	}
	close(gate)
	wg.Wait()
	close(results)
	ok, busy := 0, 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrBusy):
			busy++
		default:
			t.Fatalf("unexpected error %v", err)
		}
	}
	if ok != 1 || busy != n-1 || starts != 1 || f.runs(t, f.jobID) != 1 {
		t.Fatalf("ok %d busy %d starts %d rows %d, want exactly one winner", ok, busy, starts, f.runs(t, f.jobID))
	}
	// A different job of the same tenant is independent of the busy one.
	if _, err := svc.Dispatch(f.job(f.otherJobID), Params{Mode: "since_last"}); err != nil {
		t.Fatalf("independent job refused: %v", err)
	}
	if starts != 2 {
		t.Fatalf("starts %d, want 2", starts)
	}
}

func TestStoredRunningRowWithoutOwnerBlocks(t *testing.T) {
	f := newFixture(t)
	if err := db.DB.Exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'running', '{}', NOW())`,
		pkg.NewUUID(), f.jobID, f.tenantID).Error; err != nil {
		t.Fatal(err)
	}
	starts := 0
	svc := &Service{LoadConfig: okConfig, Start: f.parkStart(&starts)}
	if _, err := svc.Dispatch(f.job(f.jobID), Params{Mode: "since_last"}); !errors.Is(err, ErrBusy) {
		t.Fatalf("got %v, want ErrBusy for a stored running row", err)
	}
	if starts != 0 || engine.JobRunActive(f.tenantID, f.jobID) || f.runs(t, f.jobID) != 1 {
		t.Fatalf("starts %d owner %v rows %d: a blocked dispatch must add nothing", starts, engine.JobRunActive(f.tenantID, f.jobID), f.runs(t, f.jobID))
	}
}

// A job that disappeared before reservation is the missing class, not a start failure.
func TestMissingJobIsReportedAsMissing(t *testing.T) {
	f := newFixture(t)
	starts := 0
	svc := &Service{LoadConfig: okConfig, Start: f.parkStart(&starts)}
	gone := models.Job{ID: "job-gone-" + f.jobID, TenantID: f.tenantID}
	if _, err := svc.Dispatch(gone, Params{Mode: "since_last"}); !errors.Is(err, ErrMissing) {
		t.Fatalf("got %v, want ErrMissing", err)
	}
	if starts != 0 || engine.JobRunActive(f.tenantID, gone.ID) {
		t.Fatalf("starts %d: nothing may start for a missing job", starts)
	}
}

// JE-04: a forced admission failure is a generic start failure with no driver detail, no worker and no row.
func TestAdmissionFailureIsGenericAndStartsNothing(t *testing.T) {
	f := newFixture(t)
	logs := captureLog(t)
	failQueries(t, "job_runs", "r044:fail:job_runs")
	starts := 0
	svc := &Service{LoadConfig: okConfig, Start: f.parkStart(&starts)}
	runID, err := svc.Dispatch(f.job(f.jobID), Params{Mode: "since_last"})
	if !errors.Is(err, ErrStartFailed) || runID != "" || errors.Is(err, ErrBusy) {
		t.Fatalf("got %q %v, want ErrStartFailed", runID, err)
	}
	if strings.Contains(err.Error(), "SECRET") || strings.Contains(logs.String(), "SECRET") {
		t.Fatalf("driver detail leaked: %v %q", err, logs.String())
	}
	if starts != 0 || engine.JobRunActive(f.tenantID, f.jobID) {
		t.Fatalf("starts %d owner %v after a failed admission", starts, engine.JobRunActive(f.tenantID, f.jobID))
	}
}

// JE-06: a synchronous start failure aborts the bound reservation: the row closes as error, the
// local owner is released, and no checkpoint, result or notification is written.
func TestStartFailureAbortsReservationAndReleasesOwnership(t *testing.T) {
	f := newFixture(t)
	sentinel := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if err := db.DB.Exec("UPDATE jobs SET last_run_at = ? WHERE id = ?", sentinel, f.jobID).Error; err != nil {
		t.Fatal(err)
	}
	var seen *engine.JobRunReservation
	svc := &Service{LoadConfig: okConfig, Start: func(_ models.Job, _ *config.Config, _ Params, res *engine.JobRunReservation) error {
		seen = res
		return errors.New("synthetic start failure")
	}}
	runID, err := svc.Dispatch(f.job(f.jobID), Params{Mode: "since_last"})
	if !errors.Is(err, ErrStartFailed) || runID != "" || seen == nil {
		t.Fatalf("got %q %v (reservation %v)", runID, err, seen)
	}
	row := f.run(t, seen.RunID())
	if row.Status != "error" || row.FinishedAt == nil || row.ErrorMessage == "" {
		t.Fatalf("aborted row %+v, want a closed error run", row)
	}
	if engine.JobRunActive(f.tenantID, f.jobID) {
		t.Fatal("the local owner must be released after a successful abort")
	}
	var results, notes int64
	db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", row.ID).Count(&results)
	db.DB.Model(&models.NotificationLog{}).Where("tenant_id = ?", f.tenantID).Count(&notes)
	var job models.Job
	db.DB.First(&job, "id = ?", f.jobID)
	if results != 0 || notes != 0 || job.LastRunAt == nil || !job.LastRunAt.Equal(sentinel) {
		t.Fatalf("abort wrote results %d notifications %d checkpoint %v; it must write none", results, notes, job.LastRunAt)
	}
	// The slot is reusable after the checked abort.
	starts := 0
	again := &Service{LoadConfig: okConfig, Start: f.parkStart(&starts)}
	if _, err := again.Dispatch(f.job(f.jobID), Params{Mode: "since_last"}); err != nil || starts != 1 {
		t.Fatalf("re-dispatch after abort: %v starts %d", err, starts)
	}
}

// JE-06: when the abort cannot be recorded the stored running row keeps blocking later admission,
// even after the local owner is gone; the row is never deleted to simulate cleanup.
func TestFailedAbortKeepsStoredRowBlocking(t *testing.T) {
	f := newFixture(t)
	logs := captureLog(t)
	var disable func()
	var seen *engine.JobRunReservation
	svc := &Service{LoadConfig: okConfig, Start: func(_ models.Job, _ *config.Config, _ Params, res *engine.JobRunReservation) error {
		seen = res
		disable = failUpdates(t, "job_runs", "r044:fail:abort")
		return errors.New("synthetic start failure")
	}}
	if _, err := svc.Dispatch(f.job(f.jobID), Params{Mode: "since_last"}); !errors.Is(err, ErrStartFailed) {
		t.Fatalf("got %v", err)
	}
	disable()
	if seen == nil {
		t.Fatal("start was not reached")
	}
	if row := f.run(t, seen.RunID()); row.Status != "running" {
		t.Fatalf("row is %q; a failed abort must leave it running", row.Status)
	}
	if engine.JobRunActive(f.tenantID, f.jobID) {
		t.Fatal("the local owner is released even when the abort failed (the stored row blocks instead)")
	}
	// This package's own line names the run and says it keeps blocking, without driver detail. (The
	// engine's pre-existing fallback line logs the driver error text; engine logging is out of scope.)
	var own string
	for _, line := range strings.Split(logs.String(), "\n") {
		if strings.Contains(line, "could not be closed") {
			own = line
		}
	}
	if !strings.Contains(own, "keeps blocking admission") || strings.Contains(own, "SECRET") {
		t.Fatalf("dispatch log line %q must say the run keeps blocking, without driver detail", own)
	}
	starts := 0
	again := &Service{LoadConfig: okConfig, Start: f.parkStart(&starts)}
	if _, err := again.Dispatch(f.job(f.jobID), Params{Mode: "since_last"}); !errors.Is(err, ErrBusy) {
		t.Fatalf("later admission got %v, want ErrBusy from the stored running row", err)
	}
	if starts != 0 || f.runs(t, f.jobID) != 1 {
		t.Fatalf("starts %d rows %d after the blocked retry", starts, f.runs(t, f.jobID))
	}
}

// JE-07: a worker panic is recovered without logging its value, closes the run as error and releases
// ownership; the goroutine ends (the test joins it through the slot).
func TestWorkerPanicClosesRunAndReleasesOwnership(t *testing.T) {
	f := newFixture(t)
	logs := captureLog(t)
	svc := &Service{Label: "trigger", LoadConfig: okConfig,
		NewAnalyzer: func(*config.Config) *engine.Analyzer { panic("PANIC-VALUE-MUST-NOT-LOG") }}
	runID, err := svc.Dispatch(f.job(f.jobID), Params{Mode: "since_last"})
	if err != nil || runID == "" {
		t.Fatalf("dispatch: %q %v", runID, err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for engine.JobRunActive(f.tenantID, f.jobID) {
		if time.Now().After(deadline) {
			t.Fatal("the slot was never released after the panic")
		}
		time.Sleep(20 * time.Millisecond)
	}
	row := f.run(t, runID)
	if row.Status != "error" || row.FinishedAt == nil {
		t.Fatalf("panicked run %+v, want closed error", row)
	}
	if strings.Contains(logs.String(), "PANIC-VALUE") || !strings.Contains(logs.String(), "panic in trigger goroutine") {
		t.Fatalf("log %q must record the panic class without its value", logs.String())
	}
}
