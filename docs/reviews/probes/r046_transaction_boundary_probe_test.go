package engine

// Reviewer-only synthetic transaction-boundary probe. Mount in an isolated exact-BUILD
// archive, never in persistent/customer application sources or databases.
import (
	"context"
	"database/sql"
	"errors"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"gorm.io/gorm"
	"strings"
	"testing"
	"time"
)

type r046BoundaryError struct{ formatted *int }

func (e r046BoundaryError) Error() string {
	*e.formatted++
	return "r046-review-synthetic-boundary@tcp/query-detail"
}

type r046BoundaryPool struct {
	gorm.ConnPool
	failedBegins  int
	failRemaining int
	cause         error
}

func (p *r046BoundaryPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if p.failRemaining > 0 {
		p.failRemaining--
		p.failedBegins++
		return nil, p.cause
	}
	return p.ConnPool.(gorm.TxBeginner).BeginTx(ctx, opts)
}

func TestR046ReviewerUnknownBeginErrorIsContained(t *testing.T) {
	f := setupFLFixture(t)
	job := f.job(t)
	run := &models.JobRun{ID: f.insertRunning(t), TenantID: f.tenantID, JobID: f.jobID}
	original := db.DB
	scoped := original.Session(&gorm.Session{NewDB: true})
	statement := *scoped.Statement
	scoped.Statement = &statement
	formats := 0
	pool := &r046BoundaryPool{ConnPool: scoped.Statement.ConnPool, failRemaining: ordinaryFinalizeAttempts, cause: r046BoundaryError{formatted: &formats}}
	scoped.Statement.ConnPool = pool
	db.DB = scoped
	t.Cleanup(func() { db.DB = original })
	delay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	t.Cleanup(func() { ordinaryFinalizeRetryDelay = delay })
	sink := withAppLog(t)
	err := finalizeOrdinaryRun(run, job, "success", "", "{}", time.Now(), nil)
	if pool.failedBegins != ordinaryFinalizeAttempts {
		t.Fatalf("raw boundary did not reach all transaction attempts: %d", pool.failedBegins)
	}
	if !errors.Is(err, errFinalizeWrite) {
		t.Fatalf("boundary failure lost bounded sentinel: %v", err)
	}
	if formats != 0 {
		t.Fatalf("BOUNDARY DETECTOR FAILED: raw boundary error formatted %d times", formats)
	}
	if strings.Contains(err.Error(), "r046-review-synthetic-boundary") || strings.Contains(sink.String(), "r046-review-synthetic-boundary") {
		t.Fatal("BOUNDARY DETECTOR FAILED: raw begin detail escaped")
	}
}

func TestR046ReviewerTransientBeginFailureRecovers(t *testing.T) {
	f := setupFLFixture(t)
	job := f.job(t)
	run := &models.JobRun{ID: f.insertRunning(t), TenantID: f.tenantID, JobID: f.jobID}
	original := db.DB
	scoped := original.Session(&gorm.Session{NewDB: true})
	statement := *scoped.Statement
	scoped.Statement = &statement
	formats := 0
	pool := &r046BoundaryPool{ConnPool: scoped.Statement.ConnPool, failRemaining: 1, cause: r046BoundaryError{formatted: &formats}}
	scoped.Statement.ConnPool = pool
	db.DB = scoped
	t.Cleanup(func() { db.DB = original })
	delay := ordinaryFinalizeRetryDelay
	ordinaryFinalizeRetryDelay = time.Millisecond
	t.Cleanup(func() { ordinaryFinalizeRetryDelay = delay })
	checkpoint := time.Now().UTC().Truncate(time.Second)
	err := finalizeOrdinaryRun(run, job, "success", "", "{}", time.Now(), &checkpoint)
	if err != nil || pool.failedBegins != 1 || formats != 0 {
		t.Fatalf("transient boundary recovery failed: err=%v failures=%d formats=%d", err, pool.failedBegins, formats)
	}
	stored := f.runRow(t, run.ID)
	reloaded := f.job(t)
	if stored.Status != "success" || reloaded.LastRunAt == nil || !reloaded.LastRunAt.Equal(checkpoint) {
		t.Fatal("transient retry did not preserve terminal/checkpoint success")
	}
}
