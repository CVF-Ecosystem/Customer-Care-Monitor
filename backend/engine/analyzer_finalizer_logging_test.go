package engine

// CCMAI-RUNTIME-046 (R046): finalizeOrdinaryRun driver-error logging containment.
// FL-01..06 acceptance matrix. Synthetic disposable MySQL on TEST_DB_DSN.
// No real provider/channel/credential/network/persistent DB. Race NOT RUN
// (CGO disabled / no C compiler on this host).

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// flBoundaryError is a synthetic error whose Error() method tracks how many times
// it was formatted or converted to string via an atomic int64 counter, verifying
// raw driver error strings never reach application logs or returned error messages.
type flBoundaryError struct {
	formatted *int64
	detail    string
}

func (e flBoundaryError) Error() string {
	if e.formatted != nil {
		atomic.AddInt64(e.formatted, 1)
	}
	if e.detail != "" {
		return e.detail
	}
	return "fl-synthetic-boundary-driver-error@tcp/query-detail"
}

// flBoundaryTx wraps an active *sql.Tx and hooks Commit and Rollback calls
// to simulate transaction boundary failures.
type flBoundaryTx struct {
	*sql.Tx
	pool *flBoundaryPool
}

func (tx *flBoundaryTx) Commit() error {
	atomic.AddInt64(&tx.pool.commitAttempts, 1)
	if atomic.LoadInt64(&tx.pool.failCommitRemaining) > 0 {
		atomic.AddInt64(&tx.pool.failCommitRemaining, -1)
		atomic.AddInt64(&tx.pool.failedCommits, 1)
		if tx.pool.commitCause != nil {
			return tx.pool.commitCause
		}
		return tx.pool.cause
	}
	return tx.Tx.Commit()
}

func (tx *flBoundaryTx) Rollback() error {
	atomic.AddInt64(&tx.pool.rollbackAttempts, 1)
	realErr := tx.Tx.Rollback()
	if errors.Is(realErr, sql.ErrTxDone) {
		realErr = nil
	}
	if atomic.LoadInt64(&tx.pool.failRollbackRemaining) > 0 {
		atomic.AddInt64(&tx.pool.failRollbackRemaining, -1)
		atomic.AddInt64(&tx.pool.failedRollbacks, 1)
		if tx.pool.rollbackCause != nil {
			return tx.pool.rollbackCause
		}
		return tx.pool.cause
	}
	return realErr
}

// flBoundaryPool wraps a gorm.ConnPool and injects synthetic transaction-boundary
// errors on BeginTx, Commit, or Rollback up to specified remaining counts.
type flBoundaryPool struct {
	gorm.ConnPool
	failedBegins          int64
	failBeginRemaining    int64
	failedCommits         int64
	failCommitRemaining   int64
	failedRollbacks       int64
	failRollbackRemaining int64
	beginAttempts         int64
	commitAttempts        int64
	rollbackAttempts      int64
	cause                 error
	commitCause           error
	rollbackCause         error
}

func (p *flBoundaryPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	atomic.AddInt64(&p.beginAttempts, 1)
	if atomic.LoadInt64(&p.failBeginRemaining) > 0 {
		atomic.AddInt64(&p.failBeginRemaining, -1)
		atomic.AddInt64(&p.failedBegins, 1)
		return nil, p.cause
	}
	realTx, err := p.ConnPool.(gorm.TxBeginner).BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &flBoundaryTx{
		Tx:   realTx,
		pool: p,
	}, nil
}

// installBoundaryPool replaces db.DB with a session configured with pool for the
// duration of the test, and restores original db.DB on cleanup.
// It clones the statement via Context to guarantee originalDB.Statement.ConnPool
// is not mutated by the scoped pool injection.
func installBoundaryPool(t *testing.T, pool *flBoundaryPool) {
	t.Helper()
	originalDB := db.DB
	originalPool := originalDB.Statement.ConnPool
	scoped := originalDB.Session(&gorm.Session{Context: context.Background()})
	pool.ConnPool = originalPool
	scoped.Statement.ConnPool = pool
	db.DB = scoped
	t.Cleanup(func() {
		originalDB.Statement.ConnPool = originalPool
		db.DB = originalDB
	})
}

// ---- helpers ----

// flFixture is a minimal per-test fixture for finalizeOrdinaryRun tests. It
// creates synthetic tenant/job/channel/run rows and cleans up on t.Cleanup.
type flFixture struct {
	tenantID string
	jobID    string
	chanID   string
}

func setupFLFixture(t *testing.T) *flFixture {
	t.Helper()
	db.Close()
	connectTestDB(t) // uses TEST_DB_DSN; skips if unset
	s := pkg.NewUUID()[:8]
	f := &flFixture{
		tenantID: "fl-" + s,
		jobID:    "job-fl-" + s,
		chanID:   "ch-fl-" + s,
	}
	exec := func(sql string, args ...interface{}) {
		t.Helper()
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture setup: %v", err)
		}
	}
	exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'FL', ?, '{}', NOW(), NOW())`, f.tenantID, f.tenantID)
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'FL', ?, X'00', true, '{}', NOW(), NOW())`, f.chanID, f.tenantID, "ext-"+f.chanID)
	exec(`INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, schedule_cron, is_active, outputs, output_schedule, created_at, updated_at) VALUES (?, ?, 'FL', 'qc_analysis', ?, 'r', '[{"name":"x"}]', 'manual', '', true, '[]', 'none', NOW(), NOW())`,
		f.jobID, f.tenantID, `["`+f.chanID+`"]`)
	t.Cleanup(func() {
		for _, table := range []string{"job_results", "analysis_snapshots", "job_runs", "activity_logs", "jobs", "channels"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", f.tenantID)
		}
		db.DB.Exec("DELETE FROM tenants WHERE id = ?", f.tenantID)
	})
	return f
}

func (f *flFixture) job(t *testing.T) models.Job {
	t.Helper()
	var j models.Job
	if err := db.DB.First(&j, "id = ?", f.jobID).Error; err != nil {
		t.Fatal(err)
	}
	return j
}

// insertRunning creates a running job_run row and returns its ID.
func (f *flFixture) insertRunning(t *testing.T) string {
	t.Helper()
	id := pkg.NewUUID()
	if err := db.DB.Exec(`INSERT INTO job_runs (id, tenant_id, job_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'running', '{}', NOW())`,
		id, f.tenantID, f.jobID).Error; err != nil {
		t.Fatalf("insertRunning: %v", err)
	}
	return id
}

func (f *flFixture) runRow(t *testing.T, id string) models.JobRun {
	t.Helper()
	var r models.JobRun
	if err := db.DB.First(&r, "id = ?", id).Error; err != nil {
		t.Fatalf("runRow: %v", err)
	}
	return r
}

// addJobRunTrigger installs a MySQL BEFORE UPDATE trigger on job_runs for the
// given run ID and drops it on t.Cleanup.
func (f *flFixture) addJobRunTrigger(t *testing.T, runID, body string) {
	t.Helper()
	name := "fl_trig_" + strings.ReplaceAll(runID[:8], "-", "")
	db.DB.Exec("DROP TRIGGER IF EXISTS " + name)
	sql := "CREATE TRIGGER " + name + " BEFORE UPDATE ON job_runs FOR EACH ROW BEGIN IF NEW.id = '" + runID + "' THEN " + body + " END IF; END"
	if err := db.DB.Exec(sql).Error; err != nil {
		t.Fatalf("addJobRunTrigger: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + name) })
}

// withAppLog replaces the process-wide log.Writer with a buffer.
// Must not run concurrently with other tests that also replace log.Writer.
func withAppLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

// withFLGormSink replaces db.DB.Logger with a capturing logger and runs fn.
// Must wrap only the call under observation (not fixture setup or assertions).
func withFLGormSink(level logger.LogLevel, slow time.Duration, fn func()) string {
	var buf bytes.Buffer
	previous := db.DB.Logger
	db.DB.Logger = logger.New(log.New(&buf, "", 0), logger.Config{
		LogLevel:                  level,
		SlowThreshold:             slow,
		IgnoreRecordNotFoundError: true,
	})
	defer func() { db.DB.Logger = previous }()
	fn()
	return buf.String()
}

// syntheticDriverDetail is a recognizable DSN-like string used to detect
// accidental raw error formatting (FL-04 adversarial error).
// cvf-allow-secret-fixture
const syntheticDriverDetail = "fl-r046-synthetic-driver-detail-secret@tcp(127.0.0.1:3306)/test"

// adversarialError implements error; its Error() returns syntheticDriverDetail.
// Used to detect accidental %v formatting in the normalization path.
type adversarialError struct{}

func (adversarialError) Error() string { return syntheticDriverDetail }

// ---- FL-01: bounded application errors ----

// TestFL01BoundedAppRetryLog verifies that when a trigger forces a write
// failure, the retry application log and returned error exclude raw driver
// text and use bounded sentinel wording only.
func TestFL01BoundedAppRetryLog(t *testing.T) {
	f := setupFLFixture(t)
	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	run := &models.JobRun{ID: f.insertRunning(t), TenantID: f.tenantID, JobID: f.jobID}
	job := f.job(t)

	const triggerDetail = "fl-r046-forced-write-error-r01"
	f.addJobRunTrigger(t, run.ID, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = '"+triggerDetail+"';")

	appLog := withAppLog(t)
	err := finalizeOrdinaryRun(run, job, "error", "errmsg", "{}", time.Now(), nil)

	if err == nil {
		t.Fatal("expected an error; got nil")
	}
	if !errors.Is(err, errFinalizeWrite) {
		t.Fatalf("returned error is not errFinalizeWrite: %v", err)
	}
	if strings.Contains(err.Error(), triggerDetail) {
		t.Fatalf("raw trigger detail %q escaped into returned error: %q", triggerDetail, err.Error())
	}
	logOutput := appLog.String()
	if strings.Contains(logOutput, triggerDetail) {
		t.Fatalf("raw trigger detail %q escaped into application log: %q", triggerDetail, logOutput)
	}
	if !strings.Contains(logOutput, "write error") {
		t.Fatalf("expected 'write error' class in application log; got: %q", logOutput)
	}
}

// TestFL01BoundedAppFallbackLog verifies the fallback write log excludes raw
// driver text and uses bounded class wording (run ID + "write error").
func TestFL01BoundedAppFallbackLog(t *testing.T) {
	f := setupFLFixture(t)
	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	run := &models.JobRun{ID: f.insertRunning(t), TenantID: f.tenantID, JobID: f.jobID}
	job := f.job(t)

	const fallbackDetail = "fl-r046-forced-fallback-error-r01"
	f.addJobRunTrigger(t, run.ID, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = '"+fallbackDetail+"';")

	appLog := withAppLog(t)
	_ = finalizeOrdinaryRun(run, job, "error", "errmsg", "{}", time.Now(), nil)

	logOutput := appLog.String()
	if strings.Contains(logOutput, fallbackDetail) {
		t.Fatalf("raw fallback detail %q escaped into application log: %q", fallbackDetail, logOutput)
	}
	if !strings.Contains(logOutput, "write error") {
		t.Fatalf("expected 'write error' class in fallback log; got: %q", logOutput)
	}
	if !strings.Contains(logOutput, run.ID) {
		t.Fatalf("run ID %q missing from fallback log: %q", run.ID, logOutput)
	}
}

// TestFLBoundaryPoolRestoresOriginalStatement verifies that installBoundaryPool
// isolates the statement and does NOT mutate originalStatement.ConnPool, and that
// cleanup completely restores both the DB pointer and the original pool.
// Maintained control adapted from reviewer's probe (R046-R2-01).
func TestFLBoundaryPoolRestoresOriginalStatement(t *testing.T) {
	setupFLFixture(t)
	originalDB := db.DB
	originalStatement := originalDB.Statement
	originalPool := originalStatement.ConnPool
	defer func() {
		originalStatement.ConnPool = originalPool
		db.DB = originalDB
	}()
	t.Run("install_and_cleanup", func(t *testing.T) {
		installBoundaryPool(t, &flBoundaryPool{})
		if originalStatement.ConnPool != originalPool {
			t.Error("worker helper mutated original Statement.ConnPool before cleanup")
		}
	})
	if db.DB != originalDB {
		t.Error("worker cleanup did not restore DB pointer")
	}
	if originalDB.Statement.ConnPool != originalPool {
		t.Fatal("worker cleanup restored DB pointer but retained injected pool in original Statement")
	}
}

// TestFL01TransactionBoundaryUnknownErrorIsContained exercises raw transaction-boundary
// failure (BEGIN fault) across all retry attempts. Verifies:
// 1. All ordinaryFinalizeAttempts transaction attempts are executed.
// 2. The raw boundary error is never formatted (*formats == 0).
// 3. Raw detail string never leaks to returned error or app log.
// 4. Sentinel errFinalizeWrite is preserved and discoverable via errors.Is.
// 5. Fallback write marks the run status="error" with bounded error message.
func TestFL01TransactionBoundaryUnknownErrorIsContained(t *testing.T) {
	f := setupFLFixture(t)
	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	var formats int64
	pool := &flBoundaryPool{
		failBeginRemaining: ordinaryFinalizeAttempts,
		cause: flBoundaryError{
			formatted: &formats,
			detail:    "fl-r046-boundary-begin-fault@tcp(127.0.0.1:3306)/app",
		},
	}
	installBoundaryPool(t, pool)

	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	appLog := withAppLog(t)
	checkpoint := time.Now().UTC().Truncate(time.Second)
	err := finalizeOrdinaryRun(run, job, "success", "", "{}", time.Now(), &checkpoint)

	if pool.failedBegins != ordinaryFinalizeAttempts {
		t.Fatalf("expected %d boundary attempts; got %d", ordinaryFinalizeAttempts, pool.failedBegins)
	}
	if !errors.Is(err, errFinalizeWrite) {
		t.Fatalf("boundary failure lost bounded sentinel: %v", err)
	}
	if formats != 0 {
		t.Fatalf("BOUNDARY DETECTOR FAILED: raw boundary error formatted %d times", formats)
	}
	if strings.Contains(err.Error(), "fl-r046-boundary-begin-fault") {
		t.Fatalf("raw boundary error leaked into returned error message: %v", err)
	}
	logOutput := appLog.String()
	if strings.Contains(logOutput, "fl-r046-boundary-begin-fault") {
		t.Fatalf("raw boundary error leaked into app log: %s", logOutput)
	}
	// Fallback write check
	stored := f.runRow(t, run.ID)
	if stored.Status != "error" {
		t.Fatalf("fallback must mark run status as 'error'; got %q", stored.Status)
	}
	if stored.ErrorMessage != "Không ghi nhận được kết quả cuối của lượt chạy; mốc quét giữ nguyên." {
		t.Fatalf("unexpected fallback error message: %q", stored.ErrorMessage)
	}
	if stored.FinishedAt == nil {
		t.Fatal("fallback must set finished_at")
	}
}

// TestFL01TransactionBoundaryCommitFaultIsContained exercises raw transaction-boundary
// failure on Commit across all retry attempts. Verifies:
// 1. All ordinaryFinalizeAttempts commit attempts fail and retry.
// 2. Raw commit error is never formatted (*formats == 0).
// 3. Raw detail string never leaks to returned error or app log.
// 4. Sentinel errFinalizeWrite is preserved and discoverable via errors.Is.
// 5. Fallback write marks the run status="error" with bounded error message.
func TestFL01TransactionBoundaryCommitFaultIsContained(t *testing.T) {
	f := setupFLFixture(t)
	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	var formats int64
	pool := &flBoundaryPool{
		failCommitRemaining: ordinaryFinalizeAttempts,
		commitCause: flBoundaryError{
			formatted: &formats,
			detail:    "fl-r046-boundary-commit-fault@tcp(127.0.0.1:3306)/app",
		},
	}
	installBoundaryPool(t, pool)

	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	appLog := withAppLog(t)
	checkpoint := time.Now().UTC().Truncate(time.Second)
	err := finalizeOrdinaryRun(run, job, "success", "", "{}", time.Now(), &checkpoint)

	if pool.failedCommits != ordinaryFinalizeAttempts {
		t.Fatalf("expected %d commit attempts; got %d", ordinaryFinalizeAttempts, pool.failedCommits)
	}
	if !errors.Is(err, errFinalizeWrite) {
		t.Fatalf("commit boundary failure lost bounded sentinel: %v", err)
	}
	if formats != 0 {
		t.Fatalf("BOUNDARY COMMIT DETECTOR FAILED: raw commit error formatted %d times", formats)
	}
	if strings.Contains(err.Error(), "fl-r046-boundary-commit-fault") {
		t.Fatalf("raw commit error leaked into returned error: %v", err)
	}
	logOutput := appLog.String()
	if strings.Contains(logOutput, "fl-r046-boundary-commit-fault") {
		t.Fatalf("raw commit error leaked into app log: %s", logOutput)
	}
	stored := f.runRow(t, run.ID)
	if stored.Status != "error" {
		t.Fatalf("fallback must mark run status as 'error'; got %q", stored.Status)
	}
	if stored.FinishedAt == nil {
		t.Fatal("fallback must set finished_at")
	}
}

// TestFL01TransactionBoundaryRollbackFaultIsContained exercises boundary behavior when
// an internal transaction failure triggers Rollback, and the Rollback call returns a raw driver error.
// Verifies raw rollback error object is not formatted or leaked into returned error or app log.
func TestFL01TransactionBoundaryRollbackFaultIsContained(t *testing.T) {
	f := setupFLFixture(t)
	job := f.job(t)

	var formats int64
	pool := &flBoundaryPool{
		failRollbackRemaining: 1,
		rollbackCause: flBoundaryError{
			formatted: &formats,
			detail:    "fl-r046-boundary-rollback-fault@tcp(127.0.0.1:3306)/app",
		},
	}
	installBoundaryPool(t, pool)

	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	// Trigger a missing row failure so transaction fails and executes rollback
	missingRun := &models.JobRun{ID: pkg.NewUUID(), TenantID: f.tenantID, JobID: f.jobID}
	appLog := withAppLog(t)
	err := finalizeOrdinaryRun(missingRun, job, "error", "", "{}", time.Now(), nil)

	if pool.failedRollbacks < 1 {
		t.Fatalf("expected at least 1 rollback attempt; got %d", pool.failedRollbacks)
	}
	if formats != 0 {
		t.Fatalf("BOUNDARY ROLLBACK DETECTOR FAILED: raw rollback error formatted %d times", formats)
	}
	if !errors.Is(err, errFinalizeMissing) {
		t.Fatalf("expected errFinalizeMissing sentinel to be preserved; got: %v", err)
	}
	if strings.Contains(err.Error(), "fl-r046-boundary-rollback-fault") {
		t.Fatalf("raw rollback error leaked into returned error: %v", err)
	}
	if strings.Contains(appLog.String(), "fl-r046-boundary-rollback-fault") {
		t.Fatalf("raw rollback error leaked into app log: %s", appLog.String())
	}
}

// ---- FL-02: GORM sink ----

// TestFL02GormSinkContained installs a capturing GORM logger at all three
// emission levels and induces real finalizer transaction faults carrying
// synthetic marker text. No fault detail, SQL or bound payload must reach
// the GORM sink.
func TestFL02GormSinkContained(t *testing.T) {
	f := setupFLFixture(t)
	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	job := f.job(t)

	// Positive detector: ordinary db.DB statement must appear in GORM sink at Info.
	const marker = "fl-r046-sink-detector-r02"
	positiveInfo := withFLGormSink(logger.Info, 200*time.Millisecond, func() {
		var n int64
		_ = db.DB.Model(&models.JobRun{}).Where("id = ? AND tenant_id = ?", marker, f.tenantID).Count(&n).Error
	})
	if !strings.Contains(positiveInfo, marker) {
		t.Fatalf("positive detector: ordinary statement did not appear in GORM sink at Info; sink: %q", positiveInfo)
	}

	const sinkDetail = "fl-r046-gorm-sink-detail-r02"

	cases := []struct {
		name  string
		level logger.LogLevel
		slow  time.Duration
	}{
		{"info", logger.Info, 200 * time.Millisecond},
		{"warn_on_error", logger.Warn, time.Hour},
		{"warn_on_slow", logger.Warn, time.Nanosecond},
	}

	for _, cfg := range cases {
		cfg := cfg
		t.Run(cfg.name, func(t *testing.T) {
			subRunID := f.insertRunning(t)
			subRun := &models.JobRun{ID: subRunID, TenantID: f.tenantID, JobID: f.jobID}
			f.addJobRunTrigger(t, subRunID, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = '"+sinkDetail+"';")

			sink := withFLGormSink(cfg.level, cfg.slow, func() {
				_ = finalizeOrdinaryRun(subRun, job, "error", "errmsg", "{}", time.Now(), nil)
			})

			if strings.Contains(sink, sinkDetail) {
				t.Fatalf("GORM sink at %s contains forbidden detail %q: %q", cfg.name, sinkDetail, sink)
			}
			if strings.Contains(sink, subRunID) {
				t.Fatalf("GORM sink at %s contains run ID bound payload: %q", cfg.name, sink)
			}
		})
	}
}

// ---- FL-03: lifecycle invariants ----

// TestFL03SuccessfulTerminalWrite verifies the happy path: terminal run/job
// values are written and the checkpoint is advanced.
func TestFL03SuccessfulTerminalWrite(t *testing.T) {
	f := setupFLFixture(t)
	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	nt := installNotifier(t)

	now := time.Now().UTC().Truncate(time.Second)
	cp := now.Add(-time.Minute)
	err := finalizeOrdinaryRun(run, job, "success", "", `{"total":0}`, now, &cp)
	if err != nil {
		t.Fatalf("expected nil error; got: %v", err)
	}
	if nt.count() != 0 {
		t.Fatalf("finalizeOrdinaryRun must not call sendJobNotifications; got %d calls", nt.count())
	}

	row := f.runRow(t, runID)
	if row.Status != "success" {
		t.Fatalf("run status: want success, got %s", row.Status)
	}
	reloaded := f.job(t)
	if reloaded.LastRunStatus != "success" {
		t.Fatalf("job last_run_status: want success, got %s", reloaded.LastRunStatus)
	}
	if reloaded.LastRunAt == nil || reloaded.LastRunAt.Round(time.Second) != cp.Round(time.Second) {
		t.Fatalf("job last_run_at: want %v, got %v", cp, reloaded.LastRunAt)
	}
}

// TestFL03ExhaustedRetriesReturnsBoundedFailure verifies exhausted retries
// return errFinalizeWrite, no notification sent, checkpoint not advanced.
func TestFL03ExhaustedRetriesReturnsBoundedFailure(t *testing.T) {
	f := setupFLFixture(t)
	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	nt := installNotifier(t)
	f.addJobRunTrigger(t, runID, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'fl-r046-exhausted';")

	origCP := f.job(t).LastRunAt
	err := finalizeOrdinaryRun(run, job, "success", "", `{}`, time.Now(), nil)

	if !errors.Is(err, errFinalizeWrite) {
		t.Fatalf("expected errFinalizeWrite from exhausted retries; got: %v", err)
	}
	if nt.count() != 0 {
		t.Fatalf("expected no notifications on failure; got %d", nt.count())
	}
	reloaded := f.job(t)
	cpChanged := false
	switch {
	case origCP == nil && reloaded.LastRunAt == nil:
		// unchanged: OK
	case origCP != nil && reloaded.LastRunAt != nil && origCP.Equal(*reloaded.LastRunAt):
		// unchanged: OK
	default:
		cpChanged = true
	}
	if cpChanged {
		t.Fatalf("checkpoint must not advance on exhausted failure; was %v, now %v", origCP, reloaded.LastRunAt)
	}
}

// TestFL03MissingRowBreaksRetries verifies missing run returns errFinalizeMissing
// and logs exactly once with "row missing" class (no retries after missing).
func TestFL03MissingRowBreaksRetries(t *testing.T) {
	f := setupFLFixture(t)
	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	job := f.job(t)
	run := &models.JobRun{ID: pkg.NewUUID(), TenantID: f.tenantID, JobID: f.jobID}

	appLog := withAppLog(t)
	err := finalizeOrdinaryRun(run, job, "success", "", `{}`, time.Now(), nil)

	if !errors.Is(err, errFinalizeMissing) {
		t.Fatalf("expected errFinalizeMissing for missing row; got: %v", err)
	}
	logOutput := appLog.String()
	if !strings.Contains(logOutput, "row missing") {
		t.Fatalf("expected 'row missing' class in application log; got: %q", logOutput)
	}
	count := strings.Count(logOutput, "row missing")
	if count != 1 {
		t.Fatalf("expected exactly 1 'row missing' log line (no retries); got %d in: %q", count, logOutput)
	}
}

// TestFL03FallbackBoundedOnFailure verifies fallback log is bounded even when
// the fallback write itself fails.
func TestFL03FallbackBoundedOnFailure(t *testing.T) {
	f := setupFLFixture(t)
	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	const fbDetail = "fl-r046-fallback-trigger"
	f.addJobRunTrigger(t, runID, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = '"+fbDetail+"';")

	appLog := withAppLog(t)
	err := finalizeOrdinaryRun(run, job, "success", "", `{}`, time.Now(), nil)

	if !errors.Is(err, errFinalizeWrite) {
		t.Fatalf("expected errFinalizeWrite; got %v", err)
	}
	logOutput := appLog.String()
	if strings.Contains(logOutput, fbDetail) {
		t.Fatalf("fallback trigger detail %q escaped into log: %q", fbDetail, logOutput)
	}
	if !strings.Contains(logOutput, "fallback error mark failed") {
		t.Fatalf("expected fallback log line; got: %q", logOutput)
	}
	if !strings.Contains(logOutput, run.ID) {
		t.Fatalf("run ID missing from fallback log: %q", logOutput)
	}
}

// TestFL03TransactionBoundaryTransientRecovery verifies that a transient raw boundary
// failure (e.g. 1 failed BeginTx) retries and succeeds on the subsequent attempt,
// successfully committing the terminal run and advancing the proposed checkpoint.
func TestFL03TransactionBoundaryTransientRecovery(t *testing.T) {
	f := setupFLFixture(t)
	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	var formats int64
	pool := &flBoundaryPool{
		failBeginRemaining: 1,
		cause: flBoundaryError{
			formatted: &formats,
			detail:    "fl-r046-transient-boundary-fault@tcp(127.0.0.1:3306)/app",
		},
	}
	installBoundaryPool(t, pool)

	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	checkpoint := time.Now().UTC().Truncate(time.Second)
	err := finalizeOrdinaryRun(run, job, "success", "", "{}", time.Now(), &checkpoint)
	if err != nil {
		t.Fatalf("expected transient recovery to succeed; got err: %v", err)
	}
	if pool.failedBegins != 1 {
		t.Fatalf("expected exactly 1 failed begin attempt; got %d", pool.failedBegins)
	}
	if formats != 0 {
		t.Fatalf("raw boundary error was formatted %d times during transient failure", formats)
	}

	stored := f.runRow(t, run.ID)
	reloaded := f.job(t)
	if stored.Status != "success" {
		t.Fatalf("run status: want 'success', got %q", stored.Status)
	}
	if reloaded.LastRunAt == nil || !reloaded.LastRunAt.Equal(checkpoint) {
		t.Fatalf("checkpoint must advance to proposed checkpoint %v; got %v", checkpoint, reloaded.LastRunAt)
	}
	if reloaded.LastRunStatus != "success" {
		t.Fatalf("job last_run_status: want 'success', got %q", reloaded.LastRunStatus)
	}
}

// TestFL03TransactionBoundaryCommitTransientRecovery verifies that a transient Commit
// failure retries and succeeds on the subsequent attempt, committing terminal status and advancing checkpoint.
func TestFL03TransactionBoundaryCommitTransientRecovery(t *testing.T) {
	f := setupFLFixture(t)
	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	var formats int64
	pool := &flBoundaryPool{
		failCommitRemaining: 1,
		commitCause: flBoundaryError{
			formatted: &formats,
			detail:    "fl-r046-commit-transient-fault@tcp(127.0.0.1:3306)/app",
		},
	}
	installBoundaryPool(t, pool)

	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	checkpoint := time.Now().UTC().Truncate(time.Second)
	err := finalizeOrdinaryRun(run, job, "success", "", "{}", time.Now(), &checkpoint)
	if err != nil {
		t.Fatalf("expected commit transient recovery to succeed; got err: %v", err)
	}
	if pool.failedCommits != 1 {
		t.Fatalf("expected exactly 1 failed commit attempt; got %d", pool.failedCommits)
	}
	if formats != 0 {
		t.Fatalf("raw commit boundary error was formatted %d times during transient failure", formats)
	}

	stored := f.runRow(t, run.ID)
	reloaded := f.job(t)
	if stored.Status != "success" {
		t.Fatalf("run status: want 'success', got %q", stored.Status)
	}
	if reloaded.LastRunAt == nil || !reloaded.LastRunAt.Equal(checkpoint) {
		t.Fatalf("checkpoint must advance to proposed checkpoint %v; got %v", checkpoint, reloaded.LastRunAt)
	}
	if reloaded.LastRunStatus != "success" {
		t.Fatalf("job last_run_status: want 'success', got %q", reloaded.LastRunStatus)
	}
}

// TestFL03TransactionBoundaryExhaustedDoesNotAdvanceCheckpoint verifies that when boundary
// retries are exhausted, a nonnil proposed checkpoint demonstrably does NOT advance,
// preserving the existing checkpoint (or nil) and marking the run as error via fallback.
func TestFL03TransactionBoundaryExhaustedDoesNotAdvanceCheckpoint(t *testing.T) {
	f := setupFLFixture(t)
	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	if job.LastRunAt != nil {
		t.Fatalf("fixture job initial LastRunAt should be nil; got %v", job.LastRunAt)
	}

	var formats int64
	const exhaustedMarker = "fl-r046-exhausted-boundary-fault@tcp(127.0.0.1:3306)/app"
	pool := &flBoundaryPool{
		failBeginRemaining: ordinaryFinalizeAttempts,
		cause: flBoundaryError{
			formatted: &formats,
			detail:    exhaustedMarker,
		},
	}
	installBoundaryPool(t, pool)

	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	appLog := withAppLog(t)
	checkpoint := time.Now().UTC().Truncate(time.Second)
	err := finalizeOrdinaryRun(run, job, "success", "", "{}", time.Now(), &checkpoint)
	if err == nil {
		t.Fatal("expected exhausted retries to return error; got nil")
	}

	// Semantic & attempt observations (R046-R2-02)
	if pool.failedBegins != ordinaryFinalizeAttempts {
		t.Fatalf("expected exactly %d failed begin attempts; got %d", ordinaryFinalizeAttempts, pool.failedBegins)
	}
	if formats != 0 {
		t.Fatalf("BOUNDARY EXHAUSTED DETECTOR FAILED: raw boundary error formatted %d times", formats)
	}
	if !errors.Is(err, errFinalizeWrite) {
		t.Fatalf("expected bounded sentinel errFinalizeWrite; got: %v", err)
	}
	if strings.Contains(err.Error(), exhaustedMarker) {
		t.Fatalf("raw boundary marker %q escaped into returned error: %v", exhaustedMarker, err)
	}
	if strings.Contains(appLog.String(), exhaustedMarker) {
		t.Fatalf("raw boundary marker %q escaped into app log: %s", exhaustedMarker, appLog.String())
	}

	reloadedJob := f.job(t)
	if reloadedJob.LastRunAt != nil {
		t.Fatalf("checkpoint MUST NOT advance on exhausted failure; want nil, got %v", reloadedJob.LastRunAt)
	}
	storedRun := f.runRow(t, run.ID)
	if storedRun.Status != "error" {
		t.Fatalf("fallback must mark run status as 'error'; got %q", storedRun.Status)
	}
}

// ---- FL-04: error contract ----

// TestFL04SentinelDiscovery verifies errors.Is discovers both sentinels
// through the returned wrapper.
func TestFL04SentinelDiscovery(t *testing.T) {
	f := setupFLFixture(t)
	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	job := f.job(t)

	t.Run("write_failure_sentinel", func(t *testing.T) {
		runID := f.insertRunning(t)
		run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}
		f.addJobRunTrigger(t, runID, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'fl-r046-sentinel-write';")

		err := finalizeOrdinaryRun(run, job, "error", "", `{}`, time.Now(), nil)
		if !errors.Is(err, errFinalizeWrite) {
			t.Fatalf("errors.Is(err, errFinalizeWrite) = false; err=%v", err)
		}
		if errors.Is(err, errFinalizeMissing) {
			t.Fatalf("errFinalizeMissing must not be aliased from write failure")
		}
	})

	t.Run("missing_row_sentinel", func(t *testing.T) {
		run := &models.JobRun{ID: pkg.NewUUID(), TenantID: f.tenantID, JobID: f.jobID}
		err := finalizeOrdinaryRun(run, job, "error", "", `{}`, time.Now(), nil)
		if !errors.Is(err, errFinalizeMissing) {
			t.Fatalf("errors.Is(err, errFinalizeMissing) = false; err=%v", err)
		}
		if errors.Is(err, errFinalizeWrite) {
			t.Fatalf("errFinalizeWrite must not be aliased from missing-row failure")
		}
	})
}

// TestFL04AdversarialErrorNotFormatted verifies that a trigger with a DSN-like
// message does not escape into returned error or application log.
func TestFL04AdversarialErrorNotFormatted(t *testing.T) {
	f := setupFLFixture(t)
	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	// cvf-allow-secret-fixture
	const triggerPayload = "fl-r046-adversarial@tcp/testdb"
	f.addJobRunTrigger(t, runID, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = '"+triggerPayload+"';")

	appLog := withAppLog(t)
	err := finalizeOrdinaryRun(run, job, "error", "", `{}`, time.Now(), nil)

	if err != nil && strings.Contains(err.Error(), triggerPayload) {
		t.Fatalf("adversarial trigger payload %q escaped into returned error: %q", triggerPayload, err.Error())
	}
	logOutput := appLog.String()
	if strings.Contains(logOutput, triggerPayload) {
		t.Fatalf("adversarial trigger payload %q escaped into application log: %q", triggerPayload, logOutput)
	}
	if strings.Contains(logOutput, syntheticDriverDetail) {
		t.Fatalf("syntheticDriverDetail escaped into application log: %q", logOutput)
	}
	// adversarialError is defined but its value never flows into finalizeOrdinaryRun;
	// this test documents the invariant. Confirm the type compiles correctly:
	_ = adversarialError{}.Error()
}

// ---- FL-05: semantic detectors ----

// TestFL05RetryLogDetector is a named behavioral control detector. The
// PASS result means the current implementation suppresses raw driver text
// in the retry log. If the fix were absent (raw "%v" of lastErr), the
// trigger marker would appear in the log.
func TestFL05RetryLogDetector(t *testing.T) {
	f := setupFLFixture(t)
	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	const detectorMarker = "fl-r046-detector-retry-marker-05"
	f.addJobRunTrigger(t, runID, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = '"+detectorMarker+"';")

	appLog := withAppLog(t)
	_ = finalizeOrdinaryRun(run, job, "error", "", `{}`, time.Now(), nil)
	logOutput := appLog.String()

	// Current implementation: marker must NOT appear in log (sentinel normalization active).
	if strings.Contains(logOutput, detectorMarker) {
		t.Fatalf("FL-05 RETRY DETECTOR FAILED: raw driver marker %q escaped into log (fix missing or reverted): %q", detectorMarker, logOutput)
	}
	// Positive check: a bounded class line must still be present.
	if !strings.Contains(logOutput, "write error") && !strings.Contains(logOutput, "row missing") {
		t.Fatalf("FL-05 RETRY DETECTOR: no bounded class line in log; log: %q", logOutput)
	}
}

// TestFL05FallbackLogDetector is a named behavioral control detector for the
// fallback write log. PASS means raw driver text does not escape.
func TestFL05FallbackLogDetector(t *testing.T) {
	f := setupFLFixture(t)
	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	const detectorMarker = "fl-r046-detector-fallback-marker-05"
	f.addJobRunTrigger(t, runID, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = '"+detectorMarker+"';")

	appLog := withAppLog(t)
	_ = finalizeOrdinaryRun(run, job, "error", "", `{}`, time.Now(), nil)
	logOutput := appLog.String()

	if strings.Contains(logOutput, detectorMarker) {
		t.Fatalf("FL-05 FALLBACK DETECTOR FAILED: raw driver marker %q escaped into fallback log (fix missing or reverted): %q", detectorMarker, logOutput)
	}
}

// TestFL05GormSinkDetector is a named behavioral control detector for the
// GORM sink. PASS means finalizerDB() scoped session is active.
// If finalizerDB() were replaced by db.DB, the trigger detail would appear
// in the GORM sink at Warn-on-error level.
func TestFL05GormSinkDetector(t *testing.T) {
	f := setupFLFixture(t)
	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	const detectorMarker = "fl-r046-detector-gorm-marker-05"
	f.addJobRunTrigger(t, runID, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = '"+detectorMarker+"';")

	// Warn-on-error: would emit SQL errors if db.DB.Transaction used instead of finalizerDB().
	sink := withFLGormSink(logger.Warn, time.Hour, func() {
		_ = finalizeOrdinaryRun(run, job, "error", "", `{}`, time.Now(), nil)
	})

	// Current implementation (finalizerDB scoped session): marker must NOT appear.
	if strings.Contains(sink, detectorMarker) {
		t.Fatalf("FL-05 GORM DETECTOR FAILED: raw driver marker %q appeared in GORM sink (finalizerDB fix missing or reverted): %q", detectorMarker, sink)
	}
}

// TestFL05TransactionBoundaryNormalizationDetector is a dedicated behavioral detector
// for the transaction-boundary normalization switch in finalizeOrdinaryRun.
// If the normalization switch is bypassed (e.g. lastErr = txErr without mapping to
// errFinalizeWrite), this test detects and fails:
// 1. Returned error does not match errors.Is(err, errFinalizeWrite).
// 2. Raw boundary error leaks or is formatted.
func TestFL05TransactionBoundaryNormalizationDetector(t *testing.T) {
	f := setupFLFixture(t)
	job := f.job(t)
	runID := f.insertRunning(t)
	run := &models.JobRun{ID: runID, TenantID: f.tenantID, JobID: f.jobID}

	const detectorMarker = "fl05-detector-boundary-driver-leak-marker"
	var formats int64
	pool := &flBoundaryPool{
		failBeginRemaining: ordinaryFinalizeAttempts,
		cause: flBoundaryError{
			formatted: &formats,
			detail:    detectorMarker,
		},
	}
	installBoundaryPool(t, pool)

	origDelay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	defer func() { ordinaryFinalizeRetryDelay = origDelay }()

	appLog := withAppLog(t)
	err := finalizeOrdinaryRun(run, job, "success", "", "{}", time.Now(), nil)

	// Invariant 1: must return bounded sentinel errFinalizeWrite
	if !errors.Is(err, errFinalizeWrite) {
		t.Fatalf("FL-05 BOUNDARY DETECTOR FAILED: returned error does not match errFinalizeWrite sentinel (normalization switch missing or bypassed): %v", err)
	}
	// Invariant 2: raw boundary error must never be formatted
	if formats != 0 {
		t.Fatalf("FL-05 BOUNDARY DETECTOR FAILED: raw error formatted %d times", formats)
	}
	// Invariant 3: raw detail marker must not appear in returned error
	if strings.Contains(err.Error(), detectorMarker) {
		t.Fatalf("FL-05 BOUNDARY DETECTOR FAILED: raw driver marker %q leaked in returned error: %v", detectorMarker, err)
	}
	// Invariant 4: raw detail marker must not appear in app log
	if strings.Contains(appLog.String(), detectorMarker) {
		t.Fatalf("FL-05 BOUNDARY DETECTOR FAILED: raw driver marker %q leaked in app log: %s", detectorMarker, appLog.String())
	}
}

// ---- FL-06: isolation / lifecycle regression selection ----

// TestFL06SuccessLifecyclePreservation runs a full ordinary incremental
// analysis on a disposable MySQL and verifies the success lifecycle.
func TestFL06SuccessLifecyclePreservation(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "fl06", []time.Time{f.clock.Add(-time.Hour)})

	p := &incProvider{}
	run := f.mustRun(t, p)

	if run.Status != "success" {
		t.Fatalf("expected success status; got %s", run.Status)
	}
	if run.FinishedAt == nil {
		t.Fatal("expected FinishedAt to be set")
	}
	reloaded := f.job(t)
	if reloaded.LastRunStatus != "success" {
		t.Fatalf("job last_run_status: want success, got %s", reloaded.LastRunStatus)
	}
	if reloaded.LastRunAt == nil {
		t.Fatal("checkpoint must be set after success run")
	}
}

// TestFL06CancellationLeavesNoBoundaryLeak exercises the cancellation path.
// Confirms no driver-detail text leaks into application log.
func TestFL06CancellationLeavesNoBoundaryLeak(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "fl06-cancel", []time.Time{f.clock.Add(-time.Hour)})

	appLog := withAppLog(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _ = NewAnalyzer(&config.Config{}).RunJobWithProvider(ctx, f.job(t), 0, &incProvider{})

	logOutput := appLog.String()
	forbidden := []string{"DSN", "@tcp(", "driver:"}
	for _, needle := range forbidden {
		if strings.Contains(logOutput, needle) {
			t.Fatalf("forbidden driver text %q appeared in app log after cancellation: %q", needle, logOutput)
		}
	}
}

// NOTE: Race detector NOT RUN on this host (CGO disabled / no C compiler).
// Full engine suite and full backend suite NOT RUN in this bounded BUILD.
// Required: TEST_DB_DSN must be set; tests skip automatically if absent.
