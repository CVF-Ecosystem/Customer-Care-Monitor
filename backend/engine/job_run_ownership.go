package engine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-028 (F06): one shared admission and ownership model for every job entry path
// (HTTP test-run/trigger, cron, after-sync, the analysis agent and the public Analyzer methods).
//
// Ownership is the tenant/job/run tuple plus an opaque in-process token. At most one owned
// execution per tenant/job exists in this process from reservation until the worker has actually
// exited (terminal write, notification and cleanup included). The database side of admission is a
// short transaction that locks the tenant-scoped Job parent, refuses when any stored running row
// exists and inserts exactly one running JobRun, so cooperating revised processes serialize on the
// same parent. It is NOT a distributed lease: cross-process cancellation, crash recovery, timeout
// reclaim and mixed-binary rollout are out of scope, and a stored running row without a local owner
// fails closed (it blocks admission and cannot be cancelled from here).
//
// Lock order (every new transaction that locks both): Job row -> JobRun rows. The R025 terminal
// finalizer was changed to the same order. No database lock is held across a provider call or a
// worker join. Coordinator.mu guards only the owner map; owner.mu serializes cancellation against
// result publication and terminal commit (the "serialization point" of the SPEC) and is never held
// while waiting for another lock.

var (
	// ErrJobBusy: the tenant/job already has an owned execution (local) or a stored running row.
	ErrJobBusy = errors.New("job_already_running")
	// ErrJobAdmission: reservation could not be created, read, locked or committed.
	ErrJobAdmission = errors.New("job_start_failed")
	// ErrJobMissing: the tenant-scoped job does not exist (any more).
	ErrJobMissing = errors.New("job_not_found")
	// ErrRunNotRunning: no current run to cancel (none, terminal, or a different run).
	ErrRunNotRunning = errors.New("job_not_running")
	// ErrRunNotOwned: a stored running row exists but this process does not own it.
	ErrRunNotOwned = errors.New("job_run_not_owned")
	// ErrCancelCheck: the checked read that validates the cancel target failed.
	ErrCancelCheck = errors.New("job_cancel_failed")
	// ErrJobRunPanic: the worker panicked; details are logged without the panic value.
	ErrJobRunPanic = errors.New("job_run_panic")
	// ErrReservationMismatch: the job handed to a reserved run is not the tenant/job the
	// reservation was admitted for. Nothing runs and the reservation is left untouched.
	ErrReservationMismatch = errors.New("job_reservation_mismatch")
)

// jobRunTimeout bounds one owned execution started over HTTP (the worker never uses the short
// request context). It is a variable only so tests can shorten it.
var jobRunTimeout = 30 * time.Minute

type jobKey struct{ tenantID, jobID string }

// JobRunCoordinator is the in-process owner registry. The default instance serves production; tests
// create independent instances to prove that the database admission alone serializes them.
type JobRunCoordinator struct {
	mu     sync.Mutex
	owners map[jobKey]*JobRunOwner
}

func newJobRunCoordinator() *JobRunCoordinator {
	return &JobRunCoordinator{owners: map[jobKey]*JobRunOwner{}}
}

var defaultJobRuns = newJobRunCoordinator()

// JobRunOwner is the opaque owner of one execution. Workers receive it only through a
// JobRunReservation; it is never looked up by a caller-supplied run id.
type JobRunOwner struct {
	coord    *JobRunCoordinator
	key      jobKey
	runID    string
	token    string
	ctx      context.Context
	cancel   context.CancelFunc
	released sync.Once

	mu              sync.Mutex
	cancelRequested bool
	finalizing      bool // a terminal commit is being written (cancellation is no longer accepted)
}

// JobRunReservation is a reserved, owned, already-persisted running JobRun. It is consumed exactly
// once by a RunReserved call (or abandoned with Abort).
type JobRunReservation struct {
	owner *JobRunOwner
	run   models.JobRun
	used  sync.Once
}

// RunID is the stored run identity returned to HTTP callers.
func (r *JobRunReservation) RunID() string { return r.run.ID }

// Context is the owner's cancellable context (derived from the reservation parent, never from an
// HTTP request).
func (r *JobRunReservation) Context() context.Context { return r.owner.ctx }

// ReserveJobRun admits one execution of the tenant-scoped job and persists its running row.
// parent is the context the worker derives from (context.Background() for HTTP workers); timeout
// 0 adds no deadline. Errors: ErrJobBusy, ErrJobMissing, ErrJobAdmission.
func ReserveJobRun(parent context.Context, job models.Job, timeout time.Duration) (*JobRunReservation, error) {
	return defaultJobRuns.reserve(parent, job, timeout)
}

func (c *JobRunCoordinator) reserve(parent context.Context, job models.Job, timeout time.Duration) (*JobRunReservation, error) {
	key := jobKey{job.TenantID, job.ID}
	var ctx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(parent, timeout)
	} else {
		ctx, cancel = context.WithCancel(parent)
	}
	owner := &JobRunOwner{coord: c, key: key, runID: pkg.NewUUID(), token: pkg.NewUUID(), ctx: ctx, cancel: cancel}

	// Local exclusion first: the owner is visible to cancel/delete guards before the database
	// round trip, and a concurrent local caller is refused without a transaction.
	c.mu.Lock()
	if _, taken := c.owners[key]; taken {
		c.mu.Unlock()
		cancel()
		return nil, ErrJobBusy
	}
	c.owners[key] = owner
	c.mu.Unlock()

	run, err := admitJobRun(job, owner.runID)
	if err != nil {
		owner.release()
		return nil, err
	}
	return &JobRunReservation{owner: owner, run: run}, nil
}

// admitJobRun is the checked database admission: lock the Job parent, refuse when a running row
// exists, insert exactly one running row. Job -> JobRun lock order.
func admitJobRun(job models.Job, runID string) (models.JobRun, error) {
	now := analyzerNow()
	run := models.JobRun{
		ID:        runID,
		JobID:     job.ID,
		TenantID:  job.TenantID,
		StartedAt: now,
		Status:    "running",
		Summary:   "{}",
		CreatedAt: now,
	}
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var parents []models.Job
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").
			Where("id = ? AND tenant_id = ?", job.ID, job.TenantID).Find(&parents).Error; err != nil {
			return ErrJobAdmission
		}
		if len(parents) != 1 {
			return ErrJobMissing
		}
		// A plain read: the Job parent lock already serializes every same-job writer (admission,
		// finalizer, guards all lock the parent first), and a locking range read here would take
		// gap locks that deadlock admissions of different jobs.
		var running int64
		if err := tx.Model(&models.JobRun{}).
			Where("job_id = ? AND tenant_id = ? AND status = ?", job.ID, job.TenantID, "running").
			Count(&running).Error; err != nil {
			return ErrJobAdmission
		}
		if running > 0 {
			return ErrJobBusy
		}
		if err := tx.Create(&run).Error; err != nil {
			return ErrJobAdmission
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrJobBusy) || errors.Is(err, ErrJobMissing) {
			return models.JobRun{}, err
		}
		log.Printf("[jobs] admission for job %s failed", job.ID) // no SQL/driver text
		return models.JobRun{}, ErrJobAdmission
	}
	return run, nil
}

// release removes this owner (and only this owner) from the registry and cancels its context. It
// runs exactly once, after the worker has exited its last effect.
func (o *JobRunOwner) release() {
	o.released.Do(func() {
		o.coord.mu.Lock()
		if cur, ok := o.coord.owners[o.key]; ok && cur == o {
			delete(o.coord.owners, o.key)
		}
		o.coord.mu.Unlock()
		o.cancel()
	})
}

// cancelledNow reports an accepted cancellation request.
func (o *JobRunOwner) cancelledNow() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.cancelRequested
}

// publish runs fn (a result/snapshot/evaluation write) unless cancellation was accepted. It holds
// the owner lock for the duration of fn so an accepted cancel and a publication are totally
// ordered: either the cancel waits for the commit, or fn never starts.
func (o *JobRunOwner) publish(fn func() error) (bool, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.cancelRequested {
		return false, nil
	}
	return true, fn()
}

// beginTerminal fixes the outcome class under the serialization point: when a cancellation was
// accepted the run must close as cancelled (never success/checkpoint); otherwise cancellation is
// refused from now on (the terminal commit wins). It returns whether the run is cancelled.
func (o *JobRunOwner) beginTerminal() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.cancelRequested {
		return true
	}
	o.finalizing = true
	return false
}

// terminalFailed re-opens cancellation when the terminal write did not commit (the row is still
// running and this worker is about to exit).
func (o *JobRunOwner) terminalFailed() {
	o.mu.Lock()
	o.finalizing = false
	o.mu.Unlock()
}

// requestCancel accepts a cancellation unless the terminal commit already won.
func (o *JobRunOwner) requestCancel() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.finalizing {
		return false
	}
	o.cancelRequested = true
	o.cancel()
	return true
}

// CancelJobRun targets the run of a tenant/job. runID empty selects the unique locally owned
// current run at the serialization point; a supplied runID must be that run. It validates the
// stored row with checked reads before signalling the exact context and writes nothing: the
// worker records the cancelled terminal state. It returns the targeted run id on acceptance
// (repeated requests are accepted again).
func CancelJobRun(tenantID, jobID, runID string) (string, error) {
	return defaultJobRuns.cancelRun(tenantID, jobID, runID)
}

func (c *JobRunCoordinator) cancelRun(tenantID, jobID, runID string) (string, error) {
	key := jobKey{tenantID, jobID}
	c.mu.Lock()
	owner := c.owners[key]
	c.mu.Unlock()

	if owner == nil {
		// No local owner: distinguish "nothing running" from a stored running row this process
		// does not own (fails closed; recovery is a separate work order).
		q := db.DB.Model(&models.JobRun{}).Where("job_id = ? AND tenant_id = ? AND status = ?", jobID, tenantID, "running")
		if runID != "" {
			q = q.Where("id = ?", runID)
		}
		var n int64
		if err := q.Count(&n).Error; err != nil {
			log.Printf("[jobs] cancel check for job %s failed", jobID)
			return "", ErrCancelCheck
		}
		if n > 0 {
			return "", ErrRunNotOwned
		}
		return "", ErrRunNotRunning
	}
	if runID != "" && runID != owner.runID {
		return "", ErrRunNotRunning // stale id: never cancels another run
	}
	// Checked read of the targeted stored row before any signal.
	var rows []models.JobRun
	if err := db.DB.Select("id", "status").
		Where("id = ? AND tenant_id = ? AND job_id = ?", owner.runID, tenantID, jobID).Find(&rows).Error; err != nil {
		log.Printf("[jobs] cancel check for run %s failed", owner.runID)
		return "", ErrCancelCheck
	}
	if len(rows) != 1 || rows[0].Status != "running" {
		return "", ErrRunNotRunning
	}
	if !owner.requestCancel() {
		return "", ErrRunNotRunning // the terminal commit won the serialization point
	}
	return owner.runID, nil
}

// JobRunActive reports whether this process owns an execution of the tenant/job.
func JobRunActive(tenantID, jobID string) bool {
	defaultJobRuns.mu.Lock()
	defer defaultJobRuns.mu.Unlock()
	_, ok := defaultJobRuns.owners[jobKey{tenantID, jobID}]
	return ok
}

// GuardJobMutation is called inside the caller's transaction before a destructive job operation
// (delete job, clear runs). It locks the Job parent (so it serializes with admission), refuses an
// active local owner or any stored running row with ErrJobBusy, and returns ErrJobMissing when the
// tenant-scoped job does not exist. Job -> JobRun lock order.
func GuardJobMutation(tx *gorm.DB, tenantID, jobID string) error {
	var parents []models.Job
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").
		Where("id = ? AND tenant_id = ?", jobID, tenantID).Find(&parents).Error; err != nil {
		return ErrJobAdmission
	}
	if len(parents) != 1 {
		return ErrJobMissing
	}
	if JobRunActive(tenantID, jobID) {
		return ErrJobBusy
	}
	var running int64
	if err := tx.Model(&models.JobRun{}).
		Where("job_id = ? AND tenant_id = ? AND status = ?", jobID, tenantID, "running").
		Count(&running).Error; err != nil {
		return ErrJobAdmission
	}
	if running > 0 {
		return ErrJobBusy
	}
	return nil
}

// bound is the identity the reservation was admitted for; every terminal write uses it, never a
// caller-supplied job.
func (r *JobRunReservation) bound() models.Job {
	return models.Job{ID: r.owner.key.jobID, TenantID: r.owner.key.tenantID}
}

// matches reports whether job is the tenant/job this reservation was admitted for.
func (r *JobRunReservation) matches(job models.Job) bool {
	return r != nil && r.owner != nil && job.ID == r.owner.key.jobID && job.TenantID == r.owner.key.tenantID
}

// closeOwnedRun is the single terminal path for every close that is not the analysis completion
// (early failure, panic, abort/setup failure). It takes the same owner decision as the normal
// completion: an accepted cancellation closes the run as cancelled, otherwise the terminal commit
// wins and later cancellation is refused. The write is the checked, tenant/job/run-scoped finalizer
// with no checkpoint; if it cannot be recorded cancellation is re-opened and the (still running)
// row keeps blocking admission. job must be the bound identity.
func closeOwnedRun(owner *JobRunOwner, run *models.JobRun, job models.Job, publicMsg string) (string, error) {
	status, msg := "error", publicMsg
	if owner.beginTerminal() {
		status, msg = "cancelled", "Cancelled by user"
	}
	finishedAt := analyzerNow()
	if err := finalizeOrdinaryRun(run, job, status, msg, "{}", finishedAt, nil); err != nil {
		owner.terminalFailed()
		return status, fmt.Errorf("close run: %w", err)
	}
	run.Status = status
	run.FinishedAt = &finishedAt
	run.ErrorMessage = msg
	return status, nil
}

// Abort closes a reservation whose worker will never run, on the reservation's own bound identity
// (the argument is accepted for compatibility and ignored, so an Abort can never mutate another
// job). The stored row is finalized through the shared terminal path (cancelled when a cancel was
// accepted, else error) and the owner released. If the write cannot be recorded the running row
// stays and keeps blocking admission (fail closed).
func (r *JobRunReservation) Abort(_ models.Job, reason string) error {
	var err error
	r.used.Do(func() {
		defer r.owner.release()
		_, err = closeOwnedRun(r.owner, &r.run, r.bound(), reason)
	})
	return err
}
