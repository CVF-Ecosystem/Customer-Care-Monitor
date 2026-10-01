package engine

import (
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// CCMAI-RUNTIME-025 (F03): both scheduler entry points (cron runScheduledJob and
// TriggerAfterSyncJobs) load the job from the database and run the real Analyzer, so they get the
// ordinary incremental contract. Only the provider is synthetic (private seam); job lookup,
// routing, candidate/snapshot/result/checkpoint writes are real. Notifications stay disabled.

func (f *incFixture) runIDs(t *testing.T) []string {
	t.Helper()
	var ids []string
	if err := db.DB.Model(&models.JobRun{}).Where("job_id = ? AND tenant_id = ?", f.jobID, f.tenantID).
		Order("created_at, id").Pluck("id", &ids).Error; err != nil {
		t.Fatal(err)
	}
	return ids
}

func TestScheduledEntryPointsUseIncrementalCoverage(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	p := &incProvider{}
	orig, origDone := newScheduledAnalyzer, afterSyncJobFinished
	newScheduledAnalyzer = func(cfg *config.Config) *Analyzer {
		a := NewAnalyzer(cfg)
		a.providerOverride = p
		return a
	}
	done := make(chan string, 8)
	afterSyncJobFinished = func(jobID string) { done <- jobID }
	t.Cleanup(func() { newScheduledAnalyzer, afterSyncJobFinished = orig, origDone })
	s := &Scheduler{cfg: &config.Config{}}

	lastRun := func() string {
		ids := f.runIDs(t)
		if len(ids) == 0 {
			t.Fatal("no run recorded")
		}
		return ids[len(ids)-1]
	}

	// Cron path, three invocations.
	a := f.addConv(t, f.tenantID, f.channelID, "cron-a", []time.Time{f.clock.Add(-2 * time.Hour)})
	s.runScheduledJob(f.jobID, "Inc")
	f.clock = f.clock.Add(time.Hour)
	if got := evaluatedIn(t, lastRun()); !eqIDs(got, []string{a}) {
		t.Fatalf("cron run 1 evaluated %v", got)
	}
	late := f.addConv(t, f.tenantID, f.channelID, "cron-late", []time.Time{f.clock.Add(-72 * time.Hour)})
	s.runScheduledJob(f.jobID, "Inc")
	f.clock = f.clock.Add(time.Hour)
	if got := evaluatedIn(t, lastRun()); !eqIDs(got, []string{late}) {
		t.Fatalf("cron run 2 evaluated %v, want the late old conversation", got)
	}
	p.reset()
	s.runScheduledJob(f.jobID, "Inc")
	f.clock = f.clock.Add(time.Hour)
	if p.callCount() != 0 || len(f.runIDs(t)) != 3 {
		t.Fatalf("cron run 3: %d calls, %d runs", p.callCount(), len(f.runIDs(t)))
	}
	if cp := f.checkpoint(t); cp == nil {
		t.Fatal("cron runs left no checkpoint")
	}

	// After-sync path: tenant/channel routing is synchronous; matching jobs run in a goroutine.
	f.exec(t, "UPDATE jobs SET schedule_type = 'after_sync' WHERE id = ?", f.jobID)
	late2 := f.addConv(t, f.tenantID, f.channelID, "sync-late", []time.Time{f.clock.Add(-96 * time.Hour)})
	runsBefore := len(f.runIDs(t))
	s.TriggerAfterSyncJobs(f.tenantID, f.otherChannelID)
	s.TriggerAfterSyncJobs(f.otherTenantID, f.channelID)
	if n := len(f.runIDs(t)); n != runsBefore {
		t.Fatalf("after-sync routed to a job for another channel/tenant: %d runs", n)
	}
	select {
	case id := <-done:
		t.Fatalf("unexpected after-sync worker for %s", id)
	default:
	}

	waitAfterSync := func() {
		t.Helper()
		select {
		case id := <-done:
			if id != f.jobID {
				t.Fatalf("after-sync finished job %s", id)
			}
		case <-time.After(60 * time.Second):
			t.Fatal("after-sync worker did not finish")
		}
	}
	p.reset()
	s.TriggerAfterSyncJobs(f.tenantID, f.channelID)
	waitAfterSync()
	f.clock = f.clock.Add(time.Hour)
	if got := evaluatedIn(t, lastRun()); !eqIDs(got, []string{late2}) {
		t.Fatalf("after-sync run 1 evaluated %v, want the late conversation", got)
	}
	p.reset()
	s.TriggerAfterSyncJobs(f.tenantID, f.channelID)
	waitAfterSync()
	f.clock = f.clock.Add(time.Hour)
	var last models.JobRun
	if err := db.DB.First(&last, "id = ?", lastRun()).Error; err != nil {
		t.Fatal(err)
	}
	if p.callCount() != 0 || last.Status != "success" || len(f.runIDs(t)) != runsBefore+2 {
		t.Fatalf("after-sync run 2: %d calls, status %s, %d runs", p.callCount(), last.Status, len(f.runIDs(t)))
	}
}
