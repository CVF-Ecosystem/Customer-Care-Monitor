package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// CCMAI-RUNTIME-028 (F06): cron and after-sync use the shared admission. They skip a busy job
// without a fabricated run, release the after-sync callback exactly once, and their owners are
// cancellable by run. The real scheduler entry points run with a barrier synthetic provider.

type schedHarness struct {
	s    *Scheduler
	done chan string
}

func newSchedHarness(t *testing.T, p *barrierProvider, f *incFixture) *schedHarness {
	t.Helper()
	orig, origDone := newScheduledAnalyzer, afterSyncJobFinished
	newScheduledAnalyzer = func(cfg *config.Config) *Analyzer {
		a := NewAnalyzer(cfg)
		a.providerOverride = p
		return a
	}
	h := &schedHarness{s: &Scheduler{cfg: &config.Config{}}, done: make(chan string, 8)}
	afterSyncJobFinished = func(jobID string) { h.done <- jobID }
	t.Cleanup(func() { newScheduledAnalyzer, afterSyncJobFinished = orig, origDone })
	f.exec(t, "UPDATE jobs SET schedule_type = 'after_sync' WHERE id = ?", f.jobID)
	return h
}

func (h *schedHarness) expectCallbackOnce(t *testing.T, jobID string) {
	t.Helper()
	select {
	case got := <-h.done:
		if got != jobID {
			t.Fatalf("callback for %s, want %s", got, jobID)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the after-sync callback was never released")
	}
	select {
	case extra := <-h.done:
		t.Fatalf("the callback fired twice (%s)", extra)
	case <-time.After(300 * time.Millisecond):
	}
}

// An owned job is skipped by cron and by after-sync: no run row, no provider call, and the
// after-sync callback is released exactly once per skipped job.
func TestCronAndAfterSyncSkipAnOccupiedJob(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.twoConvs(t)
	p := newBarrier()
	h := newSchedHarness(t, p, f)
	owner, err := ReserveJobRun(context.Background(), f.job(t), 0)
	if err != nil {
		t.Fatal(err)
	}
	rows := f.runsOf(t, f.jobID)
	h.s.runScheduledJob(f.jobID, "Inc")
	h.s.TriggerAfterSyncJobs(f.tenantID, f.channelID)
	h.expectCallbackOnce(t, f.jobID)
	if f.runsOf(t, f.jobID) != rows || p.callCount() != 0 || f.activityCount(t, "job.run.started") != 0 {
		t.Fatalf("a skipped trigger left a run (%d), a provider call (%d) or an activity", f.runsOf(t, f.jobID)-rows, p.callCount())
	}
	if f.job(t).LastRunStatus == "success" {
		t.Fatal("a skipped trigger recorded a successful run")
	}
	if err := owner.Abort(f.job(t), "test"); err != nil {
		t.Fatal(err)
	}
	// the same entry points run once the slot is free
	p.release()
	h.s.runScheduledJob(f.jobID, "Inc")
	if f.runsOf(t, f.jobID) != rows+1 {
		t.Fatalf("cron did not run after the slot was released (%d rows)", f.runsOf(t, f.jobID))
	}
}

// Cron holding the slot refuses an after-sync trigger for the same job (and vice versa).
func TestCronVersusAfterSyncAdmitExactlyOne(t *testing.T) {
	for _, first := range []string{"cron", "after-sync"} {
		first := first
		t.Run(first+" first", func(t *testing.T) {
			f := setupIncFixture(t, false, "qc_analysis")
			f.twoConvs(t)
			p := newBarrier()
			h := newSchedHarness(t, p, f)
			var started *bgRun
			if first == "cron" {
				started = goRun(func() (*models.JobRun, error) { h.s.runScheduledJob(f.jobID, "Inc"); return nil, nil })
			} else {
				h.s.TriggerAfterSyncJobs(f.tenantID, f.channelID)
			}
			p.waitEntered(t)
			rows := f.runsOf(t, f.jobID)
			if first == "cron" {
				h.s.TriggerAfterSyncJobs(f.tenantID, f.channelID)
				h.expectCallbackOnce(t, f.jobID) // the busy after-sync trigger is released at once
			} else {
				h.s.runScheduledJob(f.jobID, "Inc") // refused synchronously
			}
			if f.runsOf(t, f.jobID) != rows || p.callCount() != 1 {
				t.Fatalf("a second entry path got in: %d rows (was %d), %d provider calls", f.runsOf(t, f.jobID), rows, p.callCount())
			}
			p.release()
			if first == "cron" {
				started.wait(t)
			} else {
				h.expectCallbackOnce(t, f.jobID)
			}
			if f.runsOf(t, f.jobID) != rows {
				t.Fatal("extra runs after the owner finished")
			}
		})
	}
}

// Cron- and after-sync-owned runs are cancellable by their run id; the cancelled worker records
// the terminal state and releases the slot (and the after-sync callback) only after it exits.
func TestSchedulerOwnersAreCancellableByRun(t *testing.T) {
	for _, path := range []string{"cron", "after-sync"} {
		path := path
		t.Run(path, func(t *testing.T) {
			f := setupIncFixture(t, false, "qc_analysis")
			f.twoConvs(t)
			sentinel := f.setSentinel(t)
			p := newBarrier()
			p.ignoreCtx = true
			h := newSchedHarness(t, p, f)
			var bg *bgRun
			if path == "cron" {
				bg = goRun(func() (*models.JobRun, error) { h.s.runScheduledJob(f.jobID, "Inc"); return nil, nil })
			} else {
				h.s.TriggerAfterSyncJobs(f.tenantID, f.channelID)
			}
			p.waitEntered(t)
			runID := f.currentRunID(t)
			if got, err := CancelJobRun(f.tenantID, f.jobID, runID); err != nil || got != runID {
				t.Fatalf("cancel: %q %v", got, err)
			}
			if f.runRow(t, runID).Status != "running" {
				t.Fatal("the request wrote a terminal state")
			}
			if _, err := f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t)); !errors.Is(err, ErrJobBusy) {
				t.Fatalf("the slot was released at the request: %v", err)
			}
			p.release()
			if path == "cron" {
				bg.wait(t)
			} else {
				h.expectCallbackOnce(t, f.jobID)
			}
			stored := f.runRow(t, runID)
			if stored.Status != "cancelled" || stored.FinishedAt == nil || JobRunActive(f.tenantID, f.jobID) {
				t.Fatalf("stored %+v, active %v", stored, JobRunActive(f.tenantID, f.jobID))
			}
			if cp := f.checkpoint(t); cp == nil || !cp.Equal(sentinel) {
				t.Fatalf("an accepted cancel advanced the checkpoint to %v", cp)
			}
		})
	}
}
