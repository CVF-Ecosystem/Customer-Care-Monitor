package engine

import (
 "errors"
 "strings"
 "sync/atomic"
 "testing"
 "time"
 "gorm.io/gorm/logger"
 "github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
 "github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// Reviewer-only mounted observations; no change to worker maintained tests.
func TestR046R2ReviewerBoundarySinksAndCheckpoint(t *testing.T) {
 for _, name := range []string{"commit_exhausted", "commit_transient", "rollback_failure"} {
  t.Run(name, func(t *testing.T) {
   f := setupFLFixture(t)
   job := f.job(t)
   var formats int64
   marker := "r046-reviewer-boundary-" + name
   cause := flBoundaryError{formatted: &formats, detail: marker}
   pool := &flBoundaryPool{}
   if name == "commit_exhausted" { pool.failCommitRemaining = ordinaryFinalizeAttempts; pool.commitCause = cause }
   if name == "commit_transient" { pool.failCommitRemaining = 1; pool.commitCause = cause }
   if name == "rollback_failure" { pool.failRollbackRemaining = 1; pool.rollbackCause = cause }
   installBoundaryPool(t, pool)
   delay := ordinaryFinalizeRetryDelay
   ordinaryFinalizeRetryDelay = time.Millisecond
   defer func() { ordinaryFinalizeRetryDelay = delay }()
   id := pkg.NewUUID()
   if name != "rollback_failure" { id = f.insertRunning(t) }
   run := &models.JobRun{ID: id, TenantID: f.tenantID, JobID: f.jobID}
   checkpoint := time.Now().UTC().Truncate(time.Second)
   app := withAppLog(t)
   var err error
   sink := withFLGormSink(logger.Info, time.Nanosecond, func() {
    err = finalizeOrdinaryRun(run, job, "success", "", "{}", time.Now(), &checkpoint)
   })
   if sink != "" { t.Fatalf("scoped GORM sink emitted boundary SQL/detail: %q", sink) }
   if atomic.LoadInt64(&formats) != 0 { t.Fatalf("raw boundary error formatted %d times", formats) }
   if strings.Contains(app.String(), marker) { t.Fatal("boundary marker escaped application log") }
   storedJob := f.job(t)
   switch name {
   case "commit_exhausted":
    if !errors.Is(err, errFinalizeWrite) { t.Fatalf("lost bounded commit sentinel: %v", err) }
    if atomic.LoadInt64(&pool.failedCommits) != ordinaryFinalizeAttempts { t.Fatalf("commit attempt count: %d", pool.failedCommits) }
    if atomic.LoadInt64(&pool.rollbackAttempts) != ordinaryFinalizeAttempts { t.Fatalf("unresolved failed commit transaction: rollbacks=%d", pool.rollbackAttempts) }
    if storedJob.LastRunAt != nil { t.Fatal("exhausted COMMIT failure advanced checkpoint") }
    if f.runRow(t, id).Status != "error" { t.Fatal("exhausted COMMIT fallback not terminal error") }
   case "commit_transient":
    if err != nil { t.Fatalf("transient COMMIT did not recover: %v", err) }
    if atomic.LoadInt64(&pool.commitAttempts) != 2 || atomic.LoadInt64(&pool.rollbackAttempts) != 1 { t.Fatal("transient COMMIT retry/rollback counts differ") }
    if storedJob.LastRunAt == nil || !storedJob.LastRunAt.Equal(checkpoint) { t.Fatal("recovered COMMIT checkpoint differs") }
    if f.runRow(t, id).Status != "success" { t.Fatal("recovered COMMIT not terminal success") }
   case "rollback_failure":
    if !errors.Is(err, errFinalizeMissing) { t.Fatalf("rollback changed missing sentinel: %v", err) }
    if atomic.LoadInt64(&pool.failedRollbacks) != 1 { t.Fatal("raw ROLLBACK fault was not observed") }
    if storedJob.LastRunAt != nil { t.Fatal("missing run/ROLLBACK advanced checkpoint") }
   }
  })
 }
}
