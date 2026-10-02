package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-028 (F06): shared admission, run-owned cancellation and lifecycle cleanup against
// the real Analyzer and coordinator on a disposable MySQL. Barrier-controlled synthetic providers
// (channels, no sleeps) and a counting notification observer; no real provider/channel call.

// barrierProvider blocks each provider call until gate is closed (or, unless ignoreCtx, until the
// call's context is done) and signals entry on entered.
type barrierProvider struct {
	incProvider
	entered   chan struct{}
	gate      chan struct{}
	ignoreCtx bool        // late output: keep waiting for the gate even after cancellation
	panicWith interface{} // panic instead of answering
}

func newBarrier() *barrierProvider {
	return &barrierProvider{entered: make(chan struct{}, 64), gate: make(chan struct{})}
}

func (b *barrierProvider) wait(ctx context.Context) error {
	select {
	case <-b.gate:
		return nil
	case <-ctx.Done():
		if b.ignoreCtx {
			<-b.gate
			return nil
		}
		return ctx.Err()
	}
}

func (b *barrierProvider) AnalyzeChat(ctx context.Context, sys, transcript string) (ai.AIResponse, error) {
	b.record([]string{transcript})
	b.entered <- struct{}{}
	if b.panicWith != nil {
		panic(b.panicWith)
	}
	if err := b.wait(ctx); err != nil {
		return ai.AIResponse{}, err
	}
	return ai.AIResponse{Content: string(b.result(transcript)), Model: "test-double", Provider: "test-double"}, nil
}

func (b *barrierProvider) AnalyzeChatBatch(ctx context.Context, sys string, items []ai.BatchItem) (ai.AIResponse, error) {
	ts := make([]string, len(items))
	for i, it := range items {
		ts[i] = it.Transcript
	}
	b.record(ts)
	b.entered <- struct{}{}
	if b.panicWith != nil {
		panic(b.panicWith)
	}
	if err := b.wait(ctx); err != nil {
		return ai.AIResponse{}, err
	}
	out := make([]json.RawMessage, len(items))
	for i, it := range items {
		out[i] = b.result(it.Transcript)
	}
	raw, _ := json.Marshal(out)
	return ai.AIResponse{Content: string(raw), Model: "test-double", Provider: "test-double"}, nil
}

func (b *barrierProvider) release() {
	select {
	case <-b.gate:
	default:
		close(b.gate)
	}
}

func (b *barrierProvider) waitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-b.entered:
	case <-time.After(30 * time.Second):
		t.Fatal("the provider call never started")
	}
}

type bgRun struct {
	done chan struct{}
	run  *models.JobRun
	err  error
}

func (r *bgRun) wait(t *testing.T) {
	t.Helper()
	select {
	case <-r.done:
	case <-time.After(60 * time.Second):
		t.Fatal("the run never finished")
	}
}

func goRun(call func() (*models.JobRun, error)) *bgRun {
	r := &bgRun{done: make(chan struct{})}
	go func() {
		defer close(r.done)
		r.run, r.err = call()
	}()
	return r
}

// notifier replaces the notification seam with a counting observer (safe from goroutines).
type notifier struct {
	mu sync.Mutex
	n  int
	// onSend runs inside the dispatch, after the terminal commit and before the slot is released.
	onSend func(job models.Job, run models.JobRun)
}

func installNotifier(t *testing.T) *notifier {
	t.Helper()
	nt := &notifier{}
	sendJobNotifications = func(_ context.Context, job models.Job, run models.JobRun) error {
		nt.mu.Lock()
		nt.n++
		hook := nt.onSend
		nt.mu.Unlock()
		if hook != nil {
			hook(job, run)
		}
		return nil
	}
	return nt
}

func (n *notifier) count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.n
}

func (f *incFixture) analyzerWith(p ai.AIProvider) *Analyzer {
	return NewAnalyzerWithProvider(&config.Config{}, p)
}

func (f *incFixture) runRow(t *testing.T, runID string) models.JobRun {
	t.Helper()
	var r models.JobRun
	if err := db.DB.First(&r, "id = ?", runID).Error; err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *incFixture) runsOf(t *testing.T, jobID string) int64 {
	t.Helper()
	var n int64
	db.DB.Model(&models.JobRun{}).Where("job_id = ?", jobID).Count(&n)
	return n
}

func (f *incFixture) activityCount(t *testing.T, action string) int64 {
	t.Helper()
	var n int64
	db.DB.Model(&models.ActivityLog{}).Where("tenant_id = ? AND action = ?", f.tenantID, action).Count(&n)
	return n
}

// currentRunID is the stored running row of the fixture job.
func (f *incFixture) currentRunID(t *testing.T) string {
	t.Helper()
	var id string
	if err := db.DB.Raw("SELECT id FROM job_runs WHERE job_id = ? AND status = 'running'", f.jobID).Scan(&id).Error; err != nil || id == "" {
		t.Fatalf("no running row: %v", err)
	}
	return id
}

func (f *incFixture) twoConvs(t *testing.T) {
	t.Helper()
	f.addConv(t, f.tenantID, f.channelID, "p", []time.Time{f.clock.Add(-3 * time.Hour)})
	f.addConv(t, f.tenantID, f.channelID, "q", []time.Time{f.clock.Add(-2 * time.Hour)})
}

// ---- group A: admission ----

func TestAdmissionRefusesASecondOwnerForEveryEntryPointAndMode(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "qc_analysis")
		f.twoConvs(t)
		b := newBarrier()
		an := f.analyzerWith(b)
		first := goRun(func() (*models.JobRun, error) { return an.RunJob(context.Background(), f.job(t)) })
		b.waitEntered(t)
		runs, started := f.runsOf(t, f.jobID), f.activityCount(t, "job.run.started")
		calls := b.callCount()
		other := &incProvider{}
		ctx := context.Background()
		job := f.job(t)
		busy := map[string]func() (*models.JobRun, error){
			"ordinary":   func() (*models.JobRun, error) { return f.analyzerWith(other).RunJob(ctx, job) },
			"with limit": func() (*models.JobRun, error) { return f.analyzerWith(other).RunJobWithLimit(ctx, job, 2) },
			"full":       func() (*models.JobRun, error) { return f.analyzerWith(other).RunJobFull(ctx, job) },
			"full+params": func() (*models.JobRun, error) {
				return f.analyzerWith(other).RunJobFullWithParams(ctx, job, "2026-10-02", "2026-10-02", 1)
			},
			"unanalyzed":    func() (*models.JobRun, error) { return f.analyzerWith(other).RunJobUnanalyzed(ctx, job, 1) },
			"since last":    func() (*models.JobRun, error) { return f.analyzerWith(other).RunJobSinceLast(ctx, job, 1) },
			"with provider": func() (*models.JobRun, error) { return f.analyzerWith(other).RunJobWithProvider(ctx, job, 0, other) },
		}
		for name, call := range busy {
			if run, err := call(); !errors.Is(err, ErrJobBusy) || run != nil {
				t.Fatalf("%s: %v %v", name, run, err)
			}
		}
		if f.runsOf(t, f.jobID) != runs || f.activityCount(t, "job.run.started") != started || other.callCount() != 0 || b.callCount() != calls {
			t.Fatalf("a refused admission left a run row, an activity or a provider call")
		}
		b.release()
		first.wait(t)
		if first.err != nil || first.run.Status != "success" {
			t.Fatalf("the owner's run: %v %+v", first.err, first.run)
		}
	})
}

func TestAdmissionDifferentJobsAndTenantsRunConcurrently(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.twoConvs(t)
	// second job of the same tenant, and a job of another tenant with its own conversation
	job2 := f.addOtherJob(t)
	other := f.addConv(t, f.otherTenantID, f.otherTenantChannelID, "o", []time.Time{f.clock.Add(-time.Hour)})
	_ = other
	otherJob := "job-inc-t-" + pkg.NewUUID()[:6]
	f.exec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, schedule_cron, is_active, outputs, output_schedule, created_at, updated_at) VALUES (?, ?, 'T', 'qc_analysis', ?, 'r', '[{"name":"x"}]', 'manual', '', true, '[]', 'none', NOW(), NOW())`,
		otherJob, f.otherTenantID, `["`+f.otherTenantChannelID+`"]`)
	b1, b2, b3 := newBarrier(), newBarrier(), newBarrier()
	load := func(id string) models.Job {
		var j models.Job
		if err := db.DB.First(&j, "id = ?", id).Error; err != nil {
			t.Fatal(err)
		}
		return j
	}
	r1 := goRun(func() (*models.JobRun, error) { return f.analyzerWith(b1).RunJob(context.Background(), load(f.jobID)) })
	r2 := goRun(func() (*models.JobRun, error) { return f.analyzerWith(b2).RunJob(context.Background(), load(job2)) })
	r3 := goRun(func() (*models.JobRun, error) { return f.analyzerWith(b3).RunJob(context.Background(), load(otherJob)) })
	// all three reach their provider before any is released: they are independent
	for i, pair := range []struct {
		b *barrierProvider
		r *bgRun
	}{{b1, r1}, {b2, r2}, {b3, r3}} {
		select {
		case <-pair.b.entered:
		case <-pair.r.done:
			t.Fatalf("run %d ended before reaching its provider: %v %+v", i, pair.r.err, pair.r.run)
		case <-time.After(30 * time.Second):
			t.Fatalf("run %d never reached its provider", i)
		}
	}
	b1.release()
	b2.release()
	b3.release()
	for i, r := range []*bgRun{r1, r2, r3} {
		r.wait(t)
		if r.err != nil || r.run.Status != "success" {
			t.Fatalf("run %d: %v %+v", i, r.err, r.run)
		}
	}
	defer db.DB.Exec("DELETE FROM job_results WHERE tenant_id = ?", f.otherTenantID)
}

func TestAdmissionStoredRunningRowsFailClosed(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.twoConvs(t)
	p := &incProvider{}
	an := f.analyzerWith(p)
	// one stored running row without a local owner (legacy/crashed), then several
	for i := 0; i < 3; i++ {
		f.exec(t, `INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'running', '{}', NOW())`, pkg.NewUUID(), f.jobID, f.tenantID)
		before := f.runsOf(t, f.jobID)
		if run, err := an.RunJob(context.Background(), f.job(t)); !errors.Is(err, ErrJobBusy) || run != nil {
			t.Fatalf("%d stored running rows: %v %v", i+1, run, err)
		}
		if f.runsOf(t, f.jobID) != before || p.callCount() != 0 {
			t.Fatal("a refused admission left a row or a provider call")
		}
		if _, err := CancelJobRun(f.tenantID, f.jobID, ""); !errors.Is(err, ErrRunNotOwned) {
			t.Fatalf("an unowned running row must not be cancellable: %v", err)
		}
	}
}

func TestAdmissionTwoCoordinatorsSerializeOnTheDatabase(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	job := f.job(t)
	c1, c2 := newJobRunCoordinator(), newJobRunCoordinator()
	var wins, busy int
	var mu sync.Mutex
	var winners []*JobRunReservation
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 12; i++ {
		c := c1
		if i%2 == 1 {
			c = c2
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			res, err := c.reserve(context.Background(), job, 0)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				wins++
				winners = append(winners, res)
			case errors.Is(err, ErrJobBusy):
				busy++
			default:
				t.Errorf("unexpected admission error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if wins != 1 || busy != 11 || f.runsOf(t, f.jobID) != 1 {
		t.Fatalf("%d admitted, %d busy, %d rows; want exactly one owner and row", wins, busy, f.runsOf(t, f.jobID))
	}
	// while its row runs, the other instance is refused; once it is closed, admission works again
	if _, err := c2.reserve(context.Background(), job, 0); !errors.Is(err, ErrJobBusy) {
		t.Fatalf("second instance admitted while the row runs: %v", err)
	}
	if err := winners[0].Abort(job, "test"); err != nil {
		t.Fatal(err)
	}
	res, err := c2.reserve(context.Background(), job, 0)
	if err != nil {
		t.Fatalf("admission after the row closed: %v", err)
	}
	if err := res.Abort(job, "test"); err != nil {
		t.Fatal(err)
	}
}

// ---- group C: cancellation ----

func TestCancelTargetsTheExactRunAndHoldsOwnershipUntilExit(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "qc_analysis")
		nt := installNotifier(t)
		f.exec(t, "UPDATE jobs SET output_schedule = 'instant' WHERE id = ?", f.jobID)
		// an earlier committed evaluation must survive a later cancelled run
		prior := f.addConv(t, f.tenantID, f.channelID, "prior", []time.Time{f.clock.Add(-5 * time.Hour)})
		f.mustRun(t, &incProvider{})
		if nt.count() != 1 {
			t.Fatalf("setup notification count %d", nt.count())
		}
		sentinel := f.setSentinel(t)
		f.twoConvs(t)

		b := newBarrier()
		b.ignoreCtx = true // the provider ignores ctx and answers late
		an := f.analyzerWith(b)
		bg := goRun(func() (*models.JobRun, error) { return an.RunJob(context.Background(), f.job(t)) })
		b.waitEntered(t)
		runID := f.currentRunID(t)

		// stale or foreign ids never cancel it
		if _, err := CancelJobRun(f.tenantID, f.jobID, pkg.NewUUID()); !errors.Is(err, ErrRunNotRunning) {
			t.Fatalf("a stale id was accepted: %v", err)
		}
		if _, err := CancelJobRun(f.otherTenantID, f.jobID, runID); !errors.Is(err, ErrRunNotRunning) && !errors.Is(err, ErrRunNotOwned) {
			t.Fatalf("another tenant targeted the run: %v", err)
		}
		if _, err := CancelJobRun(f.tenantID, "no-such-job", runID); !errors.Is(err, ErrRunNotRunning) {
			t.Fatalf("another job targeted the run: %v", err)
		}
		if f.runRow(t, runID).Status != "running" {
			t.Fatal("a rejected cancel changed the row")
		}
		// the exact id is accepted; repeated and legacy (omitted id) requests are accepted too
		for i, id := range []string{runID, runID, ""} {
			got, err := CancelJobRun(f.tenantID, f.jobID, id)
			if err != nil || got != runID {
				t.Fatalf("cancel %d: %q %v", i, got, err)
			}
		}
		// requested is not terminal: the row stays running, the slot stays held, new work is refused
		if f.runRow(t, runID).Status != "running" {
			t.Fatal("the cancel request wrote a terminal state")
		}
		if _, err := f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t)); !errors.Is(err, ErrJobBusy) {
			t.Fatalf("the slot was released at the cancel request: %v", err)
		}
		callsAtCancel := b.callCount()
		b.release() // late provider output arrives after the accepted cancel
		bg.wait(t)
		if bg.err != nil || bg.run.Status != "cancelled" {
			t.Fatalf("cancelled run: %v %+v", bg.err, bg.run)
		}
		stored := f.runRow(t, runID)
		if stored.Status != "cancelled" || stored.FinishedAt == nil {
			t.Fatalf("stored %+v", stored)
		}
		var results, snaps int64
		db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", runID).Count(&results)
		db.DB.Model(&models.AnalysisSnapshot{}).Where("job_run_id = ?", runID).Count(&snaps)
		if results != 0 || snaps != 0 {
			t.Fatalf("late output after an accepted cancel was published: %d results, %d snapshots", results, snaps)
		}
		if b.callCount() != callsAtCancel {
			t.Fatalf("a provider call started after the accepted cancel: %d -> %d", callsAtCancel, b.callCount())
		}
		var prev int64
		db.DB.Model(&models.JobResult{}).Where("conversation_id = ? AND result_type = 'conversation_evaluation'", prior).Count(&prev)
		if prev != 1 {
			t.Fatalf("an earlier committed evaluation did not survive: %d", prev)
		}
		job := f.job(t)
		if job.LastRunAt == nil || !job.LastRunAt.Equal(sentinel) || job.LastRunStatus != "cancelled" {
			t.Fatalf("job after cancel: last_run_at %v status %q", job.LastRunAt, job.LastRunStatus)
		}
		if nt.count() != 1 { // only the setup run notified
			t.Fatalf("a cancelled run dispatched a notification (%d)", nt.count())
		}
		// the slot is free only after the worker exited
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatal("ownership survived the worker exit")
		}
		if _, err := f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t)); err != nil {
			t.Fatalf("admission after exit: %v", err)
		}
	})
}

func TestCancelStaleRunNeverCancelsTheNewRun(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-3 * time.Hour)})
	runA := f.mustRun(t, &incProvider{})
	f.addConv(t, f.tenantID, f.channelID, "b", []time.Time{f.clock.Add(-2 * time.Hour)})
	b := newBarrier()
	bg := goRun(func() (*models.JobRun, error) { return f.analyzerWith(b).RunJob(context.Background(), f.job(t)) })
	b.waitEntered(t)
	runB := f.currentRunID(t)
	if _, err := CancelJobRun(f.tenantID, f.jobID, runA.ID); !errors.Is(err, ErrRunNotRunning) {
		t.Fatalf("a finished run's id targeted the new run: %v", err)
	}
	if f.runRow(t, runB).Status != "running" {
		t.Fatal("the stale cancel changed run B")
	}
	b.release()
	bg.wait(t)
	if bg.err != nil || bg.run.Status != "success" || bg.run.ID != runB {
		t.Fatalf("run B was disturbed: %v %+v", bg.err, bg.run)
	}
}

func TestCancelWithoutAnOwnerIsClassified(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	if _, err := CancelJobRun(f.tenantID, f.jobID, ""); !errors.Is(err, ErrRunNotRunning) {
		t.Fatalf("nothing running: %v", err)
	}
	f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-time.Hour)})
	done := f.mustRun(t, &incProvider{})
	if _, err := CancelJobRun(f.tenantID, f.jobID, done.ID); !errors.Is(err, ErrRunNotRunning) {
		t.Fatalf("a terminal run: %v", err)
	}
	stored := pkg.NewUUID()
	f.exec(t, `INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'running', '{}', NOW())`, stored, f.jobID, f.tenantID)
	if _, err := CancelJobRun(f.tenantID, f.jobID, stored); !errors.Is(err, ErrRunNotOwned) {
		t.Fatalf("a stored running row without an owner: %v", err)
	}
	if _, err := CancelJobRun(f.tenantID, f.jobID, pkg.NewUUID()); !errors.Is(err, ErrRunNotRunning) {
		t.Fatalf("an unknown id while an unowned row runs: %v", err)
	}
}

// A failing checked read reports the failure and signals nothing; the run completes normally.
func TestCancelCheckFailureSignalsNothing(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.twoConvs(t)
	b := newBarrier()
	bg := goRun(func() (*models.JobRun, error) { return f.analyzerWith(b).RunJob(context.Background(), f.job(t)) })
	b.waitEntered(t)
	runID := f.currentRunID(t)
	f.exec(t, "RENAME TABLE job_runs TO job_runs_r028_hold")
	restored := false
	restore := func() {
		if !restored {
			restored = true
			db.DB.Exec("RENAME TABLE job_runs_r028_hold TO job_runs")
		}
	}
	t.Cleanup(restore)
	_, err := CancelJobRun(f.tenantID, f.jobID, runID)
	restore()
	if !errors.Is(err, ErrCancelCheck) {
		t.Fatalf("a failed read must be reported: %v", err)
	}
	defaultJobRuns.mu.Lock()
	owner := defaultJobRuns.owners[jobKey{f.tenantID, f.jobID}]
	defaultJobRuns.mu.Unlock()
	if owner == nil || owner.cancelledNow() {
		t.Fatal("the failed check signalled the owner")
	}
	b.release()
	bg.wait(t)
	if bg.err != nil || bg.run.Status != "success" {
		t.Fatalf("the run was disturbed: %v %+v", bg.err, bg.run)
	}
}

// ---- group D: terminal / cancel serialization and lifecycle ----

func TestTerminalCommitWinsOrCancelWinsDeterministically(t *testing.T) {
	// terminal first: a cancel requested while the notification runs (after the terminal commit,
	// before the slot is released) is refused and the stored success is untouched.
	t.Run("terminal before cancel", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		nt := installNotifier(t)
		f.exec(t, "UPDATE jobs SET output_schedule = 'instant' WHERE id = ?", f.jobID)
		f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-time.Hour)})
		var cancelErr error
		nt.onSend = func(job models.Job, run models.JobRun) {
			_, cancelErr = CancelJobRun(job.TenantID, job.ID, run.ID)
			if !JobRunActive(job.TenantID, job.ID) {
				t.Error("the slot was released before the notification finished")
			}
		}
		run := f.mustRun(t, &incProvider{})
		if !errors.Is(cancelErr, ErrRunNotRunning) || run.Status != "success" || f.runRow(t, run.ID).Status != "success" || nt.count() != 1 {
			t.Fatalf("cancel err %v, status %s, notified %d", cancelErr, run.Status, nt.count())
		}
		if cp := f.checkpoint(t); cp == nil {
			t.Fatal("the committed terminal state lost its checkpoint")
		}
	})
	// cancel first: accepted at the last provider call, the run closes cancelled with no
	// checkpoint and no notification even though every conversation was analyzed.
	t.Run("cancel before terminal", func(t *testing.T) {
		f := setupIncFixture(t, false, "qc_analysis")
		nt := installNotifier(t)
		f.exec(t, "UPDATE jobs SET output_schedule = 'instant' WHERE id = ?", f.jobID)
		f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-time.Hour)})
		sentinel := f.setSentinel(t)
		var got error
		p := &incProvider{}
		p.onCall = func(int) { _, got = CancelJobRun(f.tenantID, f.jobID, "") }
		run := f.mustRun(t, p)
		if got != nil || run.Status != "cancelled" || nt.count() != 0 {
			t.Fatalf("cancel %v, status %s, notified %d", got, run.Status, nt.count())
		}
		if cp := f.checkpoint(t); cp == nil || !cp.Equal(sentinel) {
			t.Fatalf("an accepted cancel advanced the checkpoint to %v", cp)
		}
	})
}

// Publication and an accepted cancel are totally ordered: a cancel that arrives while a result is
// being committed waits for the commit and then wins; one that arrived first suppresses it.
func TestPublicationIsSerializedWithCancel(t *testing.T) {
	c := newJobRunCoordinator()
	newOwner := func() *JobRunOwner {
		ctx, cancel := context.WithCancel(context.Background())
		return &JobRunOwner{coord: c, key: jobKey{"t", "j"}, runID: "r", token: "x", ctx: ctx, cancel: cancel}
	}
	t.Run("publish in flight", func(t *testing.T) {
		o := newOwner()
		inside, release := make(chan struct{}), make(chan struct{})
		published := make(chan bool, 1)
		go func() {
			ok, _ := o.publish(func() error { close(inside); <-release; return nil })
			published <- ok
		}()
		<-inside
		cancelled := make(chan bool, 1)
		go func() { cancelled <- o.requestCancel() }()
		select {
		case <-cancelled:
			t.Fatal("the cancel overtook an in-flight publication")
		case <-time.After(200 * time.Millisecond):
		}
		close(release)
		if !<-published || !<-cancelled || !o.cancelledNow() {
			t.Fatal("the committed publication must precede the accepted cancel")
		}
	})
	t.Run("cancel first", func(t *testing.T) {
		o := newOwner()
		o.requestCancel()
		ran := false
		if ok, _ := o.publish(func() error { ran = true; return nil }); ok || ran {
			t.Fatal("a publication started after an accepted cancel")
		}
	})
	t.Run("terminal first", func(t *testing.T) {
		o := newOwner()
		if o.beginTerminal() || o.requestCancel() {
			t.Fatal("a cancel was accepted after the terminal commit began")
		}
		o.terminalFailed()
		if !o.requestCancel() {
			t.Fatal("a failed terminal write must re-open cancellation")
		}
	})
	t.Run("cancel then terminal", func(t *testing.T) {
		o := newOwner()
		o.requestCancel()
		if !o.beginTerminal() {
			t.Fatal("an accepted cancel must close the run as cancelled")
		}
	})
}

// Cleanup, finalizer and release of run A never touch run B or its owner.
func TestStaleOwnerCleanupNeverTouchesTheNewRun(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	job := f.job(t)
	sentinel := f.setSentinel(t)
	a, err := ReserveJobRun(context.Background(), job, 0)
	if err != nil {
		t.Fatal(err)
	}
	staleRun := a.run
	staleOwner := a.owner
	if err := a.Abort(job, "first"); err != nil {
		t.Fatal(err)
	}
	b, err := ReserveJobRun(context.Background(), job, 0)
	if err != nil {
		t.Fatal(err)
	}
	// every stale effect is a no-op for B
	staleOwner.release()
	staleOwner.release()
	if err := a.Abort(job, "again"); err != nil {
		t.Fatalf("a second abort of a consumed reservation must be a no-op: %v", err)
	}
	if err := finalizeOrdinaryRun(&staleRun, job, "success", "", "{}", analyzerNow(), nil); err == nil {
		t.Fatal("a stale finalizer reported success against another run's scope")
	}
	cp := analyzerNow().Truncate(time.Second)
	if err := finalizeOrdinaryRun(&staleRun, job, "success", "", "{}", analyzerNow(), &cp); err == nil {
		t.Fatal("a stale finalizer with a checkpoint reported success")
	}
	// an owner object that is not the registered one (a leaked/stale handle) cannot unregister B
	ghost := &JobRunOwner{coord: defaultJobRuns, key: jobKey{f.tenantID, f.jobID}, runID: "ghost", token: "ghost"}
	ghost.ctx, ghost.cancel = context.WithCancel(context.Background())
	ghost.release()
	if !JobRunActive(f.tenantID, f.jobID) || f.runRow(t, b.run.ID).Status != "running" {
		t.Fatal("stale cleanup disturbed run B")
	}
	if got := f.job(t); got.LastRunAt == nil || !got.LastRunAt.Equal(sentinel) {
		t.Fatalf("a stale finalizer moved the checkpoint: %v", got.LastRunAt)
	}
	if err := b.Abort(job, "done"); err != nil {
		t.Fatal(err)
	}
}

func TestEarlyFailuresAreCheckedBoundedAndReleaseOwnership(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	cases := []struct {
		name  string
		setup func()
		call  func() (*models.JobRun, error)
	}{
		{"invalid input channel list", func() { f.exec(t, "UPDATE jobs SET input_channel_ids = '{\"a\":1}' WHERE id = ?", f.jobID) },
			func() (*models.JobRun, error) {
				return f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t))
			}},
		{"provider selection failure", func() {
			f.exec(t, "UPDATE jobs SET input_channel_ids = ? WHERE id = ?", `["`+f.channelID+`"]`, f.jobID)
		},
			func() (*models.JobRun, error) {
				// no injected provider and no stored API key: provider resolution fails after admission
				return NewAnalyzer(&config.Config{}).RunJob(context.Background(), f.job(t))
			}},
	}
	for _, c := range cases {
		c.setup()
		before := f.runsOf(t, f.jobID)
		run, err := c.call()
		if err == nil || run == nil || run.Status != "error" {
			t.Fatalf("%s: %v %+v", c.name, err, run)
		}
		stored := f.runRow(t, run.ID)
		if stored.Status != "error" || stored.FinishedAt == nil || stored.ErrorMessage == "" || f.runsOf(t, f.jobID) != before+1 {
			t.Fatalf("%s: stored %+v", c.name, stored)
		}
		if strings.Contains(stored.ErrorMessage, "Error 1") || strings.Contains(stored.ErrorMessage, "json:") {
			t.Fatalf("%s: raw driver/parser text stored: %q", c.name, stored.ErrorMessage)
		}
		if JobRunActive(f.tenantID, f.jobID) {
			t.Fatalf("%s: ownership leaked", c.name)
		}
		if f.job(t).LastRunStatus != "error" {
			t.Fatalf("%s: job status %q", c.name, f.job(t).LastRunStatus)
		}
	}
	// a fresh run is admitted after each failure
	f.exec(t, "UPDATE jobs SET input_channel_ids = ? WHERE id = ?", `["`+f.channelID+`"]`, f.jobID)
	if _, err := f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t)); err != nil {
		t.Fatalf("admission after early failures: %v", err)
	}
}

func TestProviderPanicIsCleanedUpWithoutLeakingTheValue(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-time.Hour)})
	sentinel := f.setSentinel(t)
	var logs bytes.Buffer
	log.SetOutput(&logs)
	defer log.SetOutput(log.Writer())
	b := newBarrier()
	b.panicWith = "secret-token-xyz Error 1146 table"
	run, err := f.analyzerWith(b).RunJob(context.Background(), f.job(t))
	if !errors.Is(err, ErrJobRunPanic) || run == nil || run.Status != "error" {
		t.Fatalf("panic outcome: %v %+v", err, run)
	}
	stored := f.runRow(t, run.ID)
	if stored.Status != "error" || strings.Contains(stored.ErrorMessage, "secret-token") || strings.Contains(err.Error(), "secret-token") || strings.Contains(logs.String(), "secret-token") {
		t.Fatalf("panic value leaked: stored %+v err %v", stored, err)
	}
	if JobRunActive(f.tenantID, f.jobID) {
		t.Fatal("ownership leaked after a panic")
	}
	if cp := f.checkpoint(t); cp == nil || !cp.Equal(sentinel) {
		t.Fatalf("a panicking run moved the checkpoint: %v", cp)
	}
	if _, err := f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t)); err != nil {
		t.Fatalf("admission after a panic: %v", err)
	}
}

func TestTimeoutIsNonSuccessAndReleasesOwnership(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	nt := installNotifier(t)
	f.exec(t, "UPDATE jobs SET output_schedule = 'instant' WHERE id = ?", f.jobID)
	f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-time.Hour)})
	sentinel := f.setSentinel(t)
	res, err := ReserveJobRun(context.Background(), f.job(t), 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	b := newBarrier() // honors ctx: returns when the deadline expires
	run, err := f.analyzerWith(b).RunReserved(res, f.job(t), "unanalyzed", 0, "", "")
	if err != nil || run.Status == "success" || run.Status == "running" || run.Status == "cancelled" {
		t.Fatalf("timeout outcome: %v %+v", err, run)
	}
	if cp := f.checkpoint(t); cp == nil || !cp.Equal(sentinel) || nt.count() != 0 || JobRunActive(f.tenantID, f.jobID) {
		t.Fatalf("checkpoint %v, notified %d, owner %v", cp, nt.count(), JobRunActive(f.tenantID, f.jobID))
	}
}

// When neither the terminal write nor the fallback mark can be stored, the running row stays,
// keeps blocking admission, cannot be cancelled here, and nothing is notified.
func TestUnresolvedRunningRowKeepsBlockingAfterTerminalFailure(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	nt := installNotifier(t)
	f.exec(t, "UPDATE jobs SET output_schedule = 'instant' WHERE id = ?", f.jobID)
	f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-time.Hour)})
	sentinel := f.setSentinel(t)
	trigger := "r028_run_" + f.jobID[len(f.jobID)-6:]
	f.exec(t, fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE ON job_runs FOR EACH ROW BEGIN IF NEW.job_id = '%s' AND NEW.status <> 'running' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'synthetic terminal failure'; END IF; END", trigger, f.jobID))
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trigger) })
	run, err := f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t))
	if err == nil || run == nil || run.Status != "error" || nt.count() != 0 {
		t.Fatalf("failed terminal write: %v %+v notified %d", err, run, nt.count())
	}
	if got := f.runRow(t, run.ID).Status; got != "running" {
		t.Fatalf("the unresolved row must stay running, got %q", got)
	}
	if JobRunActive(f.tenantID, f.jobID) {
		t.Fatal("the exited worker kept the local slot")
	}
	if _, err := f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t)); !errors.Is(err, ErrJobBusy) {
		t.Fatalf("the unresolved row must block admission: %v", err)
	}
	if _, err := CancelJobRun(f.tenantID, f.jobID, run.ID); !errors.Is(err, ErrRunNotOwned) {
		t.Fatalf("an unowned unresolved row must not be cancellable here: %v", err)
	}
	if cp := f.checkpoint(t); cp == nil || !cp.Equal(sentinel) {
		t.Fatalf("checkpoint moved: %v", cp)
	}
}

// Admission, terminal writes, the destructive guard and cancellation all take Job -> JobRun, so
// contention never deadlocks (a MySQL deadlock would surface as an admission/finalize error).
func TestNoLockInversionUnderContention(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	job := f.job(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failures []string
	note := func(format string, args ...interface{}) {
		mu.Lock()
		failures = append(failures, fmt.Sprintf(format, args...))
		mu.Unlock()
	}
	deadline := time.Now().Add(4 * time.Second)
	worker := func(id int) {
		defer wg.Done()
		for time.Now().Before(deadline) {
			switch id % 3 {
			case 0: // admit then close through the checked finalizer
				res, err := ReserveJobRun(context.Background(), job, 0)
				if errors.Is(err, ErrJobBusy) {
					continue
				}
				if err != nil {
					note("reserve: %v", err)
					return
				}
				if err := res.Abort(job, "contention"); err != nil {
					note("abort: %v", err)
					return
				}
			case 1: // destructive guard in a transaction
				err := db.DB.Transaction(func(tx *gorm.DB) error { return GuardJobMutation(tx, job.TenantID, job.ID) })
				if err != nil && !errors.Is(err, ErrJobBusy) {
					note("guard: %v", err)
					return
				}
			default: // cancellation reads
				if _, err := CancelJobRun(job.TenantID, job.ID, ""); err != nil &&
					!errors.Is(err, ErrRunNotRunning) && !errors.Is(err, ErrRunNotOwned) {
					note("cancel: %v", err)
					return
				}
			}
		}
	}
	for i := 0; i < 9; i++ {
		wg.Add(1)
		go worker(i)
	}
	wg.Wait()
	if len(failures) > 0 {
		t.Fatalf("contention errors (deadlock?): %v", failures[:1])
	}
	if JobRunActive(f.tenantID, f.jobID) {
		t.Fatal("an owner leaked after contention")
	}
}

// Admission faults are checked: an insert or read/lock failure refuses the run, stores nothing and
// leaves no owner.
func TestAdmissionFaultsAreCheckedAndLeakNothing(t *testing.T) {
	for _, fault := range []string{"insert", "read"} {
		fault := fault
		t.Run(fault, func(t *testing.T) {
			f := setupIncFixture(t, false, "qc_analysis")
			f.twoConvs(t)
			p := &incProvider{}
			switch fault {
			case "insert":
				trig := "r028_ins_" + f.jobID[len(f.jobID)-6:]
				f.exec(t, fmt.Sprintf("CREATE TRIGGER %s BEFORE INSERT ON job_runs FOR EACH ROW BEGIN IF NEW.job_id = '%s' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'synthetic admission failure'; END IF; END", trig, f.jobID))
				t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trig) })
			default:
				f.exec(t, "RENAME TABLE job_runs TO job_runs_r028_hold")
				t.Cleanup(func() { db.DB.Exec("RENAME TABLE job_runs_r028_hold TO job_runs") })
			}
			run, err := f.analyzerWith(p).RunJob(context.Background(), f.job(t))
			if !errors.Is(err, ErrJobAdmission) || run != nil || p.callCount() != 0 || JobRunActive(f.tenantID, f.jobID) {
				t.Fatalf("%s fault: run %v err %v calls %d active %v", fault, run, err, p.callCount(), JobRunActive(f.tenantID, f.jobID))
			}
			if strings.Contains(err.Error(), "synthetic") || strings.Contains(err.Error(), "Error 1") {
				t.Fatalf("driver text in the admission error: %v", err)
			}
		})
	}
}

// An accepted cancel after some conversations were already published closes the run cancelled and
// dispatches no notification, although analyzed > 0 would otherwise notify.
func TestCancelAfterPartialProgressDoesNotNotify(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	nt := installNotifier(t)
	f.exec(t, "UPDATE jobs SET output_schedule = 'instant' WHERE id = ?", f.jobID)
	f.twoConvs(t)
	sentinel := f.setSentinel(t)
	p := &incProvider{}
	p.onCall = func(n int) {
		if n == 2 { // the first conversation was published; cancel while the second is in flight
			if _, err := CancelJobRun(f.tenantID, f.jobID, ""); err != nil {
				t.Errorf("cancel: %v", err)
			}
		}
	}
	run := f.mustRun(t, p)
	var published int64
	db.DB.Model(&models.JobResult{}).Where("job_run_id = ? AND result_type = 'conversation_evaluation'", run.ID).Count(&published)
	if run.Status != "cancelled" || published != 1 || nt.count() != 0 {
		t.Fatalf("status %s, %d published, %d notifications", run.Status, published, nt.count())
	}
	if cp := f.checkpoint(t); cp == nil || !cp.Equal(sentinel) {
		t.Fatalf("checkpoint moved: %v", cp)
	}
}
