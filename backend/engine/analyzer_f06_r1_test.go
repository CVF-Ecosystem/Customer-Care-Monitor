package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-028-R1: every terminal path takes the same owner decision (accepted cancel wins,
// else the terminal commit wins and later cancel is refused) with the checked finalizer, and a
// reservation executes only the tenant/job it was admitted for.

type terminalPath struct {
	name string
	// run closes the reserved run through this path. job is the bound job (possibly with a corrupt
	// input list for the early-failure paths).
	run func(t *testing.T, f *incFixture, res *JobRunReservation, job models.Job) (*models.JobRun, error)
}

func terminalPaths() []terminalPath {
	return []terminalPath{
		{"early failure: invalid input list", func(t *testing.T, f *incFixture, res *JobRunReservation, job models.Job) (*models.JobRun, error) {
			job.InputChannelIDs = "{"
			return f.analyzerWith(&incProvider{}).RunReserved(res, job, "unanalyzed", 0, "", "")
		}},
		{"early failure: provider selection", func(t *testing.T, f *incFixture, res *JobRunReservation, job models.Job) (*models.JobRun, error) {
			// no injected provider and no stored API key; add eligible source so provider boundary is reached (LP-07)
			f.addConv(t, f.tenantID, f.channelID, "prov-fail", []time.Time{f.clock.Add(-time.Hour)})
			return NewAnalyzer(&config.Config{}).RunReserved(res, job, "unanalyzed", 0, "", "")
		}},
		{"early failure: invalid run parameters", func(t *testing.T, f *incFixture, res *JobRunReservation, job models.Job) (*models.JobRun, error) {
			return f.analyzerWith(&incProvider{}).RunReserved(res, job, "no-such-mode", 0, "", "")
		}},
		{"abort / setup failure", func(t *testing.T, f *incFixture, res *JobRunReservation, job models.Job) (*models.JobRun, error) {
			err := res.Abort(job, "Không khởi động được tiến trình chạy")
			run := f.runRow(t, res.RunID())
			return &run, err
		}},
		{"provider panic", func(t *testing.T, f *incFixture, res *JobRunReservation, job models.Job) (*models.JobRun, error) {
			f.addConv(t, f.tenantID, f.channelID, "pan", []time.Time{f.clock.Add(-time.Hour)})
			b := newBarrier()
			b.panicWith = "synthetic panic"
			return f.analyzerWith(b).RunReserved(res, job, "unanalyzed", 0, "", "")
		}},
	}
}

func TestEveryTerminalPathHonorsAnAcceptedCancel(t *testing.T) {
	for _, tp := range terminalPaths() {
		tp := tp
		t.Run(tp.name, func(t *testing.T) {
			f := setupIncFixture(t, false, "qc_analysis")
			nt := installNotifier(t)
			f.exec(t, "UPDATE jobs SET output_schedule = 'instant' WHERE id = ?", f.jobID)
			sentinel := f.setSentinel(t)
			job := f.job(t)
			res, err := ReserveJobRun(context.Background(), job, 0)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := CancelJobRun(job.TenantID, job.ID, res.RunID()); err != nil || got != res.RunID() {
				t.Fatalf("cancel: %q %v", got, err)
			}
			_, _ = tp.run(t, f, res, job)
			stored := f.runRow(t, res.RunID())
			if stored.Status != "cancelled" || stored.FinishedAt == nil || f.job(t).LastRunStatus != "cancelled" {
				t.Fatalf("accepted cancel lost: run %s job %s finished %v", stored.Status, f.job(t).LastRunStatus, stored.FinishedAt)
			}
			if cp := f.checkpoint(t); cp == nil || !cp.Equal(sentinel) || nt.count() != 0 || JobRunActive(f.tenantID, f.jobID) {
				t.Fatalf("checkpoint %v, notified %d, owner %v", cp, nt.count(), JobRunActive(f.tenantID, f.jobID))
			}
		})
	}
}

// Terminal first: once the terminal decision was taken a cancel request is refused and the run
// closes error (never cancelled), through the same path.
func TestEveryTerminalPathWinsOverALateCancel(t *testing.T) {
	for _, tp := range terminalPaths() {
		tp := tp
		t.Run(tp.name, func(t *testing.T) {
			f := setupIncFixture(t, false, "qc_analysis")
			job := f.job(t)
			res, err := ReserveJobRun(context.Background(), job, 0)
			if err != nil {
				t.Fatal(err)
			}
			// the terminal decision is taken first (as closeOwnedRun/beginTerminal does)...
			res.owner.beginTerminal()
			// ...so the cancel is refused and nothing is signalled
			if _, err := CancelJobRun(job.TenantID, job.ID, res.RunID()); !errors.Is(err, ErrRunNotRunning) {
				t.Fatalf("a cancel after the terminal decision: %v", err)
			}
			if res.owner.cancelledNow() || res.Context().Err() != nil {
				t.Fatal("a refused cancel signalled the owner")
			}
			_, _ = tp.run(t, f, res, job)
			stored := f.runRow(t, res.RunID())
			if stored.Status != "error" || stored.FinishedAt == nil {
				t.Fatalf("terminal-first close stored %q", stored.Status)
			}
			if strings.Contains(stored.ErrorMessage, "synthetic") || strings.Contains(stored.ErrorMessage, "Error 1") {
				t.Fatalf("unbounded stored message: %q", stored.ErrorMessage)
			}
		})
	}
}

// A panic after the terminal state was committed (inside the notification) does not re-close the
// run: the stored success and the checkpoint stay, the slot is released, the panic is bounded.
func TestPanicAfterCommitKeepsTheStoredTerminalState(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	nt := installNotifier(t)
	f.exec(t, "UPDATE jobs SET output_schedule = 'instant' WHERE id = ?", f.jobID)
	nt.onSend = func(models.Job, models.JobRun) { panic("synthetic notification panic") }
	f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-time.Hour)})
	scan := f.clock
	run, err := f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t))
	if !errors.Is(err, ErrJobRunPanic) || run == nil {
		t.Fatalf("%v %+v", err, run)
	}
	if got := f.runRow(t, run.ID).Status; got != "success" || run.Status != "success" {
		t.Fatalf("a post-commit panic rewrote the state: stored %q, returned %q", got, run.Status)
	}
	if cp := f.checkpoint(t); cp == nil || !cp.Equal(scan.Truncate(time.Second)) || JobRunActive(f.tenantID, f.jobID) {
		t.Fatalf("checkpoint %v, owner %v", cp, JobRunActive(f.tenantID, f.jobID))
	}
}

// When the shared terminal write cannot be stored the run stays running, blocks admission and
// cancellation is re-opened (the terminal commit did not win).
func TestFailedTerminalCloseKeepsBlockingAndReopensCancel(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	job := f.job(t)
	res, err := ReserveJobRun(context.Background(), job, 0)
	if err != nil {
		t.Fatal(err)
	}
	trig := "r028r1_run_" + f.jobID[len(f.jobID)-6:]
	f.exec(t, fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE ON job_runs FOR EACH ROW BEGIN IF NEW.job_id = '%s' AND NEW.status <> 'running' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'synthetic'; END IF; END", trig, f.jobID))
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trig) })
	owner := res.owner
	if err := res.Abort(job, "setup failure"); err == nil || strings.Contains(err.Error(), "synthetic") {
		t.Fatalf("a failed close must be reported without driver text: %v", err)
	}
	if f.runRow(t, res.RunID()).Status != "running" || JobRunActive(f.tenantID, f.jobID) {
		t.Fatal("the unrecordable abort must leave a running row and no local owner")
	}
	if _, err := f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t)); !errors.Is(err, ErrJobBusy) {
		t.Fatalf("an unresolved row must block: %v", err)
	}
	if !owner.requestCancel() { // the failed terminal write re-opened cancellation on the owner
		t.Fatal("a failed terminal write must not leave cancellation closed")
	}
}

// ---- reservation binding ----

func newSameTenantJob(t *testing.T, f *incFixture, src models.Job) models.Job {
	t.Helper()
	b := src
	b.ID = pkg.NewUUID()
	b.Name = "bound other job"
	if err := db.DB.Create(&b).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.DB.Where("job_id = ?", b.ID).Delete(&models.JobRun{})
		db.DB.Where("id = ?", b.ID).Delete(&models.Job{})
	})
	return b
}

func TestReservationRunsOnlyItsOwnTenantAndJob(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "src", []time.Time{f.clock.Add(-time.Hour)})
	a := f.job(t)
	otherJob := newSameTenantJob(t, f, a)
	crossTenant := a
	crossTenant.TenantID = f.otherTenantID
	cases := []struct {
		name string
		job  models.Job
		// occupy: another reservation already owns that job
		occupy bool
	}{
		{"same tenant, different job", otherJob, false},
		{"same tenant, different job that is occupied", otherJob, true},
		{"different tenant, same job id", crossTenant, false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			res, err := ReserveJobRun(context.Background(), a, 0)
			if err != nil {
				t.Fatal(err)
			}
			var occupier *JobRunReservation
			if c.occupy {
				if occupier, err = ReserveJobRun(context.Background(), c.job, 0); err != nil {
					t.Fatal(err)
				}
				defer occupier.Abort(c.job, "cleanup")
			}
			started := f.activityCount(t, "job.run.started")
			rowsA, rowsB := f.runsOf(t, a.ID), f.runsOf(t, c.job.ID)
			p := &incProvider{}
			run, err := f.analyzerWith(p).RunReserved(res, c.job, "unanalyzed", 0, "", "")
			var results int64
			db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", res.RunID()).Count(&results)
			if !errors.Is(err, ErrReservationMismatch) || run != nil || p.callCount() != 0 || results != 0 {
				t.Fatalf("mismatch accepted: run %v err %v provider %d published %d", run, err, p.callCount(), results)
			}
			if f.activityCount(t, "job.run.started") != started || f.runsOf(t, a.ID) != rowsA || f.runsOf(t, c.job.ID) != rowsB {
				t.Fatal("a refused mismatch left an activity or a run row")
			}
			if c.occupy && (!JobRunActive(c.job.TenantID, c.job.ID) || f.runRow(t, occupier.RunID()).Status != "running") {
				t.Fatal("the other job's owner or row was disturbed")
			}
			// the reservation is untouched and still usable by its own job, exactly once
			good, err := f.analyzerWith(&incProvider{}).RunReserved(res, a, "unanalyzed", 0, "", "")
			if err != nil || good.Status != "success" || good.ID != res.RunID() {
				t.Fatalf("the reservation was spoiled by the refused call: %v %+v", err, good)
			}
			if _, err := f.analyzerWith(&incProvider{}).RunReserved(res, a, "unanalyzed", 0, "", ""); !errors.Is(err, ErrJobAdmission) {
				t.Fatalf("a reservation must be consumed exactly once: %v", err)
			}
			if JobRunActive(a.TenantID, a.ID) {
				t.Fatal("the slot leaked")
			}
		})
	}
}

// Abort never mutates anything but the reservation's own run, whatever job it is given.
func TestAbortUsesTheBoundIdentity(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	a := f.job(t)
	otherJob := newSameTenantJob(t, f, a)
	resB, err := ReserveJobRun(context.Background(), otherJob, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer resB.Abort(otherJob, "cleanup")
	resA, err := ReserveJobRun(context.Background(), a, 0)
	if err != nil {
		t.Fatal(err)
	}
	// abort A while naming B (and a foreign tenant): only A's row closes
	if err := resA.Abort(otherJob, "mixed up"); err != nil {
		t.Fatal(err)
	}
	if f.runRow(t, resA.RunID()).Status != "error" || f.runRow(t, resB.RunID()).Status != "running" || !JobRunActive(otherJob.TenantID, otherJob.ID) {
		t.Fatal("Abort touched a run other than its own")
	}
}
