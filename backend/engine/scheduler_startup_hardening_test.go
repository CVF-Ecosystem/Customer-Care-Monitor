package engine

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"
)

type startupHardeningFixture struct {
	base      *incFixture
	tenantID  string
	channelID string
}

func setupStartupHardeningFixture(t *testing.T) *startupHardeningFixture {
	t.Helper()
	base := setupIncFixture(t, false, "qc_analysis")
	f := &startupHardeningFixture{base: base, tenantID: pkg.NewUUID(), channelID: pkg.NewUUID()}
	now := time.Now().UTC()
	base.exec(t, `INSERT INTO tenants (id, name, slug, settings, created_at, updated_at)
		VALUES (?, 'R078 synthetic tenant', ?, '{}', ?, ?)`, f.tenantID, f.tenantID, now, now)
	base.exec(t, `INSERT INTO channels (id, tenant_id, channel_type, name, external_id,
		credentials_encrypted, is_active, metadata, created_at, updated_at)
		VALUES (?, ?, 'pancake', 'R078 synthetic channel', ?, X'00', true, '{}', ?, ?)`,
		f.channelID, f.tenantID, f.channelID, now, now)
	base.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at)
		VALUES (?, ?, 'ai_batch_mode', 'false', ?, ?)`, pkg.NewUUID(), f.tenantID, now, now)
	base.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at)
		VALUES (?, ?, 'ai_batch_size', '30', ?, ?)`, pkg.NewUUID(), f.tenantID, now, now)
	t.Cleanup(func() {
		for _, table := range []string{"job_results", "analysis_snapshots", "job_runs", "ai_usage_logs", "messages", "conversations", "app_settings", "activity_logs", "jobs", "channels"} {
			if err := db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", f.tenantID).Error; err != nil {
				t.Errorf("R078 fixture cleanup %s: %v", table, err)
			}
		}
		if err := db.DB.Exec("DELETE FROM tenants WHERE id = ?", f.tenantID).Error; err != nil {
			t.Errorf("R078 fixture cleanup tenant: %v", err)
		}
	})
	return f
}

func (f *startupHardeningFixture) addJob(t *testing.T, scheduleType, scheduleCron string) models.Job {
	t.Helper()
	id := pkg.NewUUID()
	channelJSON, err := json.Marshal([]string{f.channelID})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	f.base.exec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content,
		rules_config, schedule_type, schedule_cron, is_active, outputs, output_schedule, created_at, updated_at)
		VALUES (?, ?, ?, 'qc_analysis', ?, 'Synthetic startup rule', '[{"name":"Synthetic rule"}]', ?, ?, true, '[]', 'none', ?, ?)`,
		id, f.tenantID, "R078 job "+id[:8], string(channelJSON), scheduleType, scheduleCron, now, now)
	return f.job(t, id)
}

func (f *startupHardeningFixture) addConversation(t *testing.T, label string) string {
	t.Helper()
	id := pkg.NewUUID()
	messageID := pkg.NewUUID()
	at := f.base.clock.Add(-time.Hour).UTC()
	f.base.exec(t, `INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id,
		customer_name, last_message_at, message_count, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'Synthetic customer', ?, 1, '{}', NOW(), NOW())`,
		id, f.tenantID, f.channelID, "R078 external "+label, at)
	f.base.exec(t, `INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type,
		sender_name, content, content_type, attachments, sent_at, created_at)
		VALUES (?, ?, ?, ?, 'customer', 'Synthetic customer', 'Synthetic message', 'text', '[]', ?, NOW())`,
		messageID, f.tenantID, id, "R078 message "+label, at)
	return id
}

func (f *startupHardeningFixture) job(t *testing.T, id string) models.Job {
	t.Helper()
	var job models.Job
	if err := db.DB.First(&job, "id = ? AND tenant_id = ?", id, f.tenantID).Error; err != nil {
		t.Fatal(err)
	}
	return job
}

func (f *startupHardeningFixture) addRun(t *testing.T, jobID, status string) models.JobRun {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	run := models.JobRun{
		ID: pkg.NewUUID(), JobID: jobID, TenantID: f.tenantID,
		StartedAt: now.Add(-time.Hour), Status: status, Summary: `{"fixture":"r078"}`, CreatedAt: now.Add(-time.Hour),
	}
	if status != "running" {
		finished := now
		run.FinishedAt = &finished
	}
	if err := db.DB.Create(&run).Error; err != nil {
		t.Fatalf("insert synthetic JobRun: %v", err)
	}
	return run
}

func (f *startupHardeningFixture) run(t *testing.T, id string) models.JobRun {
	t.Helper()
	var run models.JobRun
	if err := db.DB.First(&run, "id = ? AND tenant_id = ?", id, f.tenantID).Error; err != nil {
		t.Fatal(err)
	}
	return run
}

func (f *startupHardeningFixture) runCount(t *testing.T, jobID string) int64 {
	t.Helper()
	var count int64
	if err := db.DB.Model(&models.JobRun{}).Where("tenant_id = ? AND job_id = ?", f.tenantID, jobID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

type startupRunWriteAttempt struct {
	TargetID string
}

type startupRunWriteTrap struct {
	mu       sync.Mutex
	active   bool
	attempts []startupRunWriteAttempt
}

func installStartupRunWriteTrap(t *testing.T) *startupRunWriteTrap {
	t.Helper()
	trap := &startupRunWriteTrap{active: true}
	name := "r078_startup_jobrun_" + pkg.NewUUID()
	if err := db.DB.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
		table := tx.Statement.Table
		if table == "" && tx.Statement.Schema != nil {
			table = tx.Statement.Schema.Table
		}
		if table != "job_runs" {
			return
		}
		attempt := startupRunWriteAttempt{TargetID: "<unknown>"}
		switch model := tx.Statement.Model.(type) {
		case *models.JobRun:
			attempt.TargetID = model.ID
		case models.JobRun:
			attempt.TargetID = model.ID
		}
		trap.mu.Lock()
		active := trap.active
		if active {
			trap.attempts = append(trap.attempts, attempt)
		}
		trap.mu.Unlock()
		if active {
			tx.AddError(errors.New("R078 startup JobRun update blocked by test trap"))
		}
	}); err != nil {
		t.Fatalf("install JobRun update trap: %v", err)
	}
	t.Cleanup(func() {
		trap.stopObserving()
		if err := db.DB.Callback().Update().Remove(name); err != nil {
			t.Errorf("remove JobRun update trap: %v", err)
		}
	})
	return trap
}

func (trap *startupRunWriteTrap) stopObserving() []startupRunWriteAttempt {
	trap.mu.Lock()
	defer trap.mu.Unlock()
	trap.active = false
	return append([]startupRunWriteAttempt(nil), trap.attempts...)
}

func assertNoStartupRunWrites(t *testing.T, trap *startupRunWriteTrap, contract string) {
	t.Helper()
	attempts := trap.stopObserving()
	if len(attempts) != 0 {
		t.Fatalf("%s startup attempted JobRun update target=%s", contract, attempts[0].TargetID)
	}
}

type startupTestScheduler struct {
	scheduler *Scheduler
	stopOnce  sync.Once
	stopped   chan struct{}
}

func newStartupTestScheduler(t *testing.T, cfg *config.Config) *startupTestScheduler {
	t.Helper()
	scheduler, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	started := &startupTestScheduler{scheduler: scheduler, stopped: make(chan struct{})}
	t.Cleanup(func() { started.stop(t) })
	return started
}

func (s *startupTestScheduler) start() { s.scheduler.Start() }

func (s *startupTestScheduler) stop(t *testing.T) {
	t.Helper()
	s.stopOnce.Do(func() {
		go func() {
			s.scheduler.Stop()
			close(s.stopped)
		}()
	})
	select {
	case <-s.stopped:
	case <-time.After(5 * time.Second):
		t.Errorf("Scheduler.Stop exceeded the 5 second test bound")
	}
}

func jobsTable(tx *gorm.DB) string {
	if tx.Statement.Table != "" {
		return tx.Statement.Table
	}
	if tx.Statement.Schema != nil {
		return tx.Statement.Schema.Table
	}
	return ""
}

func registerJobsQueryFault(t *testing.T, hit *atomic.Int32) func() {
	t.Helper()
	name := "r078_jobs_query_fault_" + pkg.NewUUID()
	if err := db.DB.Callback().Query().Before("gorm:query").Register(name, func(tx *gorm.DB) {
		if jobsTable(tx) == "jobs" {
			hit.Add(1)
			tx.AddError(errors.New("R078 synthetic jobs query fault"))
		}
	}); err != nil {
		t.Fatalf("install jobs query fault: %v", err)
	}
	var once sync.Once
	remove := func() {
		once.Do(func() {
			if err := db.DB.Callback().Query().Remove(name); err != nil {
				t.Errorf("remove jobs query fault: %v", err)
			}
		})
	}
	t.Cleanup(remove)
	return remove
}

func TestSH01LiveOwnerStartPreservesRowAndPersistedBusyAdmission(t *testing.T) {
	f := setupStartupHardeningFixture(t)
	job := f.addJob(t, "manual", "")
	f.addConversation(t, "live-owner")
	ownerA, freshB := newJobRunCoordinator(), newJobRunCoordinator()
	reservation, err := ownerA.reserve(context.Background(), job, 0)
	if err != nil {
		t.Fatal(err)
	}
	provider := newBarrier()
	provider.ignoreCtx = true
	worker := goRun(func() (*models.JobRun, error) {
		return f.base.analyzerWith(provider).RunReserved(reservation, job, "unanalyzed", 0, "", "")
	})
	t.Cleanup(func() {
		provider.release()
		worker.wait(t)
	})
	provider.waitEntered(t)
	before := f.run(t, reservation.RunID())
	trap := installStartupRunWriteTrap(t)
	scheduler := newStartupTestScheduler(t, &config.Config{})
	scheduler.start()
	scheduler.stop(t)
	assertNoStartupRunWrites(t, trap, "SH01")
	if after := f.run(t, reservation.RunID()); !reflect.DeepEqual(before, after) {
		t.Fatalf("SH01 live-owner row changed during Start: before=%+v after=%+v", before, after)
	}
	if provider.callCount() != 1 {
		t.Fatalf("SH01 startup invoked the held provider again: calls=%d", provider.callCount())
	}
	if _, err := freshB.reserve(context.Background(), job, 0); !errors.Is(err, ErrJobBusy) {
		t.Fatalf("SH01 fresh coordinator admission = %v, want ErrJobBusy", err)
	}
	if f.runCount(t, job.ID) != 1 {
		t.Fatalf("SH01 fresh coordinator changed the persisted admission count: %d", f.runCount(t, job.ID))
	}
	provider.release()
	worker.wait(t)
	if worker.err != nil {
		t.Fatalf("SH01 held owner finalizer: %v", worker.err)
	}
	if final := f.run(t, reservation.RunID()); final.Status == "running" || final.FinishedAt == nil {
		t.Fatalf("SH01 owner finalizer did not close its own row: %+v", final)
	}
}

func TestSH02StartupPreservesUnknownRunningAndTerminalRows(t *testing.T) {
	f := setupStartupHardeningFixture(t)
	job := f.addJob(t, "manual", "")
	running := f.addRun(t, job.ID, "running")
	terminal := f.addRun(t, job.ID, "success")
	beforeRunning, beforeTerminal := f.run(t, running.ID), f.run(t, terminal.ID)
	trap := installStartupRunWriteTrap(t)
	scheduler := newStartupTestScheduler(t, &config.Config{})
	scheduler.start()
	scheduler.stop(t)
	assertNoStartupRunWrites(t, trap, "SH02")
	if after := f.run(t, running.ID); !reflect.DeepEqual(beforeRunning, after) {
		t.Fatalf("SH02 unknown running row changed during Start: before=%+v after=%+v", beforeRunning, after)
	}
	if after := f.run(t, terminal.ID); !reflect.DeepEqual(beforeTerminal, after) {
		t.Fatalf("SH02 terminal row changed during Start: before=%+v after=%+v", beforeTerminal, after)
	}
}

func TestSH03FreeJobAdmissionAndOwnerCancellationContrast(t *testing.T) {
	f := setupStartupHardeningFixture(t)
	ownedJob := f.addJob(t, "manual", "")
	freeJob := f.addJob(t, "manual", "")
	f.addConversation(t, "cancel-owner")
	ownerA, freshB := newJobRunCoordinator(), newJobRunCoordinator()
	reservation, err := ownerA.reserve(context.Background(), ownedJob, 0)
	if err != nil {
		t.Fatal(err)
	}
	provider := newBarrier()
	provider.ignoreCtx = true
	worker := goRun(func() (*models.JobRun, error) {
		return f.base.analyzerWith(provider).RunReserved(reservation, ownedJob, "unanalyzed", 0, "", "")
	})
	t.Cleanup(func() {
		provider.release()
		worker.wait(t)
	})
	provider.waitEntered(t)
	free, err := freshB.reserve(context.Background(), freeJob, 0)
	if err != nil {
		t.Fatalf("SH03 unrelated free job admission: %v", err)
	}
	if err := free.Abort(freeJob, "R078 free-job control"); err != nil {
		t.Fatalf("SH03 close free-job control: %v", err)
	}
	if _, err := freshB.cancelRun(ownedJob.TenantID, ownedJob.ID, reservation.RunID()); !errors.Is(err, ErrRunNotOwned) {
		t.Fatalf("SH03 non-owner cancellation = %v, want ErrRunNotOwned", err)
	}
	if _, err := ownerA.cancelRun(ownedJob.TenantID, ownedJob.ID, pkg.NewUUID()); !errors.Is(err, ErrRunNotRunning) {
		t.Fatalf("SH03 mismatched owner cancellation = %v, want ErrRunNotRunning", err)
	}
	if row := f.run(t, reservation.RunID()); row.Status != "running" {
		t.Fatalf("SH03 a refused cancellation changed the row: %+v", row)
	}
	if got, err := ownerA.cancelRun(ownedJob.TenantID, ownedJob.ID, reservation.RunID()); err != nil || got != reservation.RunID() {
		t.Fatalf("SH03 exact owner cancellation = %q, %v", got, err)
	}
	if row := f.run(t, reservation.RunID()); row.Status != "running" {
		t.Fatalf("SH03 accepted request finalized before the worker exited: %+v", row)
	}
	provider.release()
	worker.wait(t)
	if worker.err != nil {
		t.Fatalf("SH03 owner finalizer: %v", worker.err)
	}
	final := f.run(t, reservation.RunID())
	if final.Status != "cancelled" || final.FinishedAt == nil {
		t.Fatalf("SH03 exact owner did not finalize its row as cancelled: %+v", final)
	}
	if f.runCount(t, freeJob.ID) != 1 {
		t.Fatalf("SH03 free job was not independently admitted and finalized: %d rows", f.runCount(t, freeJob.ID))
	}
}

func TestSH04StartPreservesSyncLeaseRecoveryControls(t *testing.T) {
	f := setupSFFixture(t)
	eligible := f.chA
	eligibleReservation := f.mustReserve(t, f.tenantID, eligible)
	f.expireLease(t, eligible)
	ineligible := f.addChannel(t, "zalo_oa")
	f.mustReserve(t, f.tenantID, ineligible)
	if err := db.DB.Exec("UPDATE channels SET sync_lease_until = NOW(3) - INTERVAL 1 SECOND WHERE id = ?", ineligible).Error; err != nil {
		t.Fatal(err)
	}
	beforeEligible, beforeIneligible := f.lease(t, eligible), f.lease(t, ineligible)
	trap := installStartupRunWriteTrap(t)
	scheduler := newStartupTestScheduler(t, f.cfg)
	scheduler.start()
	scheduler.stop(t)
	assertNoStartupRunWrites(t, trap, "SH04")
	afterEligible, afterIneligible := f.lease(t, eligible), f.lease(t, ineligible)
	if afterEligible.Status != "error" || afterEligible.Error != syncLeaseReleasedMessage || afterEligible.RunID != "" || afterEligible.LeaseSet {
		t.Fatalf("SH04 eligible expired lease was not recovered by Start: %+v", afterEligible)
	}
	if !sameStartupTime(beforeEligible.LastSyncAt, afterEligible.LastSyncAt) || beforeEligible.Credentials != afterEligible.Credentials {
		t.Fatalf("SH04 eligible recovery changed checkpoint or credentials: before=%+v after=%+v", beforeEligible, afterEligible)
	}
	if !reflect.DeepEqual(beforeIneligible, afterIneligible) {
		t.Fatalf("SH04 ineligible lease changed during Start: before=%+v after=%+v", beforeIneligible, afterIneligible)
	}
	if f.triggerCount() != 0 || f.fetchCount(eligible) != 0 || f.fetchCount(ineligible) != 0 {
		t.Fatal("SH04 startup lease recovery launched channel work")
	}
	if err := checkSyncOwnership(eligibleReservation); !errors.Is(err, ErrSyncOwnershipLost) {
		t.Fatalf("SH04 recovered lease retained ownership: %v", err)
	}
}

func sameStartupTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func TestSH05UnknownOwnershipAndReservationFaultFailClosed(t *testing.T) {
	f := setupStartupHardeningFixture(t)
	unknownJob := f.addJob(t, "manual", "")
	faultJob := f.addJob(t, "manual", "")
	unknown := f.addRun(t, unknownJob.ID, "running")
	coordinator := newJobRunCoordinator()
	if _, err := coordinator.reserve(context.Background(), unknownJob, 0); !errors.Is(err, ErrJobBusy) {
		t.Fatalf("SH05 stored unknown owner admission = %v, want ErrJobBusy", err)
	}
	if after := f.run(t, unknown.ID); after.Status != "running" || f.runCount(t, unknownJob.ID) != 1 {
		t.Fatalf("SH05 unknown row did not remain a single admission blocker: %+v count=%d", after, f.runCount(t, unknownJob.ID))
	}
	var readFaults atomic.Int32
	removeFault := registerJobsQueryFault(t, &readFaults)
	faultCoordinator := newJobRunCoordinator()
	reservation, err := faultCoordinator.reserve(context.Background(), faultJob, 0)
	removeFault()
	if !errors.Is(err, ErrJobAdmission) || reservation != nil || readFaults.Load() != 1 {
		t.Fatalf("SH05 reservation read fault: reservation=%v err=%v faults=%d", reservation, err, readFaults.Load())
	}
	faultCoordinator.mu.Lock()
	owners := len(faultCoordinator.owners)
	faultCoordinator.mu.Unlock()
	if owners != 0 || f.runCount(t, faultJob.ID) != 0 {
		t.Fatalf("SH05 failed admission leaked an owner or row: owners=%d runs=%d", owners, f.runCount(t, faultJob.ID))
	}
}

func TestSH06CronLoadQueryFaultStillStartsScheduler(t *testing.T) {
	f := setupStartupHardeningFixture(t)
	job := f.addJob(t, "cron", "0 0 1 1 *")
	scheduler := newStartupTestScheduler(t, &config.Config{})
	probe := make(chan struct{})
	var probeOnce sync.Once
	if _, err := scheduler.scheduler.scheduler.NewJob(
		gocron.DurationJob(25*time.Millisecond),
		gocron.NewTask(func() { probeOnce.Do(func() { close(probe) }) }),
		gocron.WithName("r078-cron-load-fault-control"),
	); err != nil {
		t.Fatal(err)
	}
	var queryFaults atomic.Int32
	removeFault := registerJobsQueryFault(t, &queryFaults)
	scheduler.start()
	if queryFaults.Load() != 1 {
		t.Fatalf("SH06 cron load query fault observed %d times", queryFaults.Load())
	}
	select {
	case <-probe:
	case <-time.After(3 * time.Second):
		t.Fatal("SH06 Scheduler.Start did not start the scheduler after loadCronJobs query failed")
	}
	removeFault()
	for _, registered := range scheduler.scheduler.scheduler.Jobs() {
		if registered.Name() == "job-"+job.ID {
			t.Fatal("SH06 failed cron-load query unexpectedly registered the database job")
		}
	}
	scheduler.stop(t)
	if f.runCount(t, job.ID) != 0 {
		t.Fatalf("SH06 cron-load failure created a JobRun: %d", f.runCount(t, job.ID))
	}
}

func TestSH07SchedulerLoadsControlledCronAndStopsWithoutExternalEffects(t *testing.T) {
	f := setupStartupHardeningFixture(t)
	job := f.addJob(t, "cron", "0 0 1 1 *")
	f.addConversation(t, "controlled-cron")
	provider := &incProvider{}
	var analyzerFactories, notifications atomic.Int32
	oldFactory, oldNotifications := newScheduledAnalyzer, sendJobNotifications
	newScheduledAnalyzer = func(cfg *config.Config) *Analyzer {
		analyzerFactories.Add(1)
		return NewAnalyzerWithProvider(cfg, provider)
	}
	sendJobNotifications = func(context.Context, models.Job, models.JobRun) error {
		notifications.Add(1)
		return nil
	}
	t.Cleanup(func() { newScheduledAnalyzer, sendJobNotifications = oldFactory, oldNotifications })
	scheduler := newStartupTestScheduler(t, &config.Config{})
	beforeRuns := f.runCount(t, job.ID)
	scheduler.start()
	loaded := false
	for _, registered := range scheduler.scheduler.scheduler.Jobs() {
		if registered.Name() == "job-"+job.ID {
			loaded = true
			break
		}
	}
	if !loaded {
		t.Fatal("SH07 Scheduler.Start did not load the controlled cron job")
	}
	scheduler.stop(t)
	if analyzerFactories.Load() != 0 || provider.callCount() != 0 || notifications.Load() != 0 {
		t.Fatalf("SH07 startup caused provider/notification work: factories=%d provider=%d notifications=%d",
			analyzerFactories.Load(), provider.callCount(), notifications.Load())
	}
	if afterRuns := f.runCount(t, job.ID); afterRuns != beforeRuns {
		t.Fatalf("SH07 startup created JobRuns: before=%d after=%d", beforeRuns, afterRuns)
	}
}
