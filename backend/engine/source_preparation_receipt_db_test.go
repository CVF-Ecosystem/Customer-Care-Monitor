package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

func spRead(t *testing.T, summary string) preparationReceipt {
	t.Helper()
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(summary), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope["source_preparation"]) == 0 {
		t.Fatal("missing source_preparation in stored terminal row")
	}
	var r preparationReceipt
	if err := json.Unmarshal(envelope["source_preparation"], &r); err != nil {
		t.Fatal(err)
	}
	if r.Version != preparationVersion || r.Scope != "preparation_only" {
		t.Fatalf("invalid receipt %+v", r)
	}
	return r
}
func spReload(t *testing.T, run *models.JobRun) preparationReceipt {
	t.Helper()
	var stored models.JobRun
	if err := db.DB.Where("id = ? AND tenant_id = ?", run.ID, run.TenantID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	r := spRead(t, stored.Summary)
	returned := spRead(t, run.Summary)
	a, _ := json.Marshal(r)
	b, _ := json.Marshal(returned)
	if string(a) != string(b) {
		t.Fatal("stored/returned receipt mismatch")
	}
	return r
}

func TestSPStoredEarlyTerminalReceipt(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "early", []time.Time{f.clock.Add(-time.Hour)})
	run, err := NewAnalyzer(&config.Config{}).RunJob(context.Background(), f.job(t))
	if err == nil || run == nil || run.Status != "error" {
		t.Fatalf("provider failure changed: %v %+v", err, run)
	}
	r := spReload(t, run)
	if r.Counts["PREPARED_FOR_INFERENCE"] != 1 || !r.ScanComplete {
		t.Fatalf("early receipt %+v", r)
	}
}

func TestSPOrdinaryExplicitAndProgressPersistence(t *testing.T) {
	for _, batch := range []bool{false, true} {
		f := setupIncFixture(t, batch, "qc_analysis")
		empty := f.addConv(t, f.tenantID, f.channelID, "empty", nil)
		changed := f.addConv(t, f.tenantID, f.channelID, "changed", []time.Time{f.clock.Add(-time.Hour)})
		_ = empty
		f.exec(t, "UPDATE messages SET content_type = 'image', attachments = '[{}]' WHERE conversation_id = ?", changed)
		var progress []string
		name := "sp_summary_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
		if err := db.DB.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
			if tx.Statement.Table == "job_runs" {
				if values, ok := tx.Statement.Dest.(map[string]interface{}); ok {
					if s, ok := values["summary"].(string); ok {
						progress = append(progress, s)
					}
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.DB.Callback().Update().Remove(name) })
		p := &incProvider{verdict: "PASS"}
		run := f.mustRun(t, p)
		r := spReload(t, run)
		if r.Counts["EMPTY_SOURCE"] != 1 || r.Counts["PREPARED_FOR_INFERENCE"] != 1 || p.callCount() != 1 {
			t.Fatalf("ordinary %+v calls%d", r, p.callCount())
		}
		if len(progress) < 3 {
			t.Fatalf("missing initial/progress/final writes: %d", len(progress))
		}
		frozen, _ := json.Marshal(r)
		for index, s := range progress {
			pr := spRead(t, s)
			observed, _ := json.Marshal(pr)
			if string(observed) != string(frozen) {
				t.Fatal("progress receipt lost")
			}
			var scalars map[string]interface{}
			json.Unmarshal([]byte(s), &scalars)
			delete(scalars, "source_preparation")
			if _, ok := scalars["source_execution"]; !ok {
				t.Fatal("missing source_execution in Analyzer summary")
			}
			delete(scalars, "source_execution")
			if scalars["conversations_found"] != float64(1) {
				t.Fatalf("found scalar changed %+v", scalars)
			}
			if index == 0 {
				if len(scalars) != 1 {
					t.Fatalf("initial scalar keys %+v", scalars)
				}
			} else {
				if len(scalars) != 5 || scalars["conversations_analyzed"] != float64(1) || scalars["conversations_passed"] != float64(1) || scalars["conversations_errors"] != float64(0) || scalars["issues_found"] != float64(0) {
					t.Fatalf("progress/final scalars %+v", scalars)
				}
			}
		}
		for _, e := range r.Entries {
			if e.Outcome == "PREPARED_FOR_INFERENCE" && e.Coverage != "partial" {
				t.Fatal("partial became complete")
			}
		}
		run = f.mustRun(t, p)
		r = spReload(t, run)
		if r.Counts["UNCHANGED_VERIFIED"] != 1 || r.Entries[1].Reference.State == "LOOKUP_FAILED" {
			t.Fatalf("unchanged %+v", r)
		}
		a := NewAnalyzerWithProvider(&config.Config{}, p)
		run, err := a.RunJobFull(context.Background(), f.job(t))
		if err != nil {
			t.Fatal(err)
		}
		r = spReload(t, run)
		if r.Mode != "conditional" || r.Counts["UNCHANGED_VERIFIED"] != 0 || r.Counts["PREPARED_FOR_INFERENCE"] != 1 {
			t.Fatalf("explicit %+v", r)
		}
		if err := db.DB.Callback().Update().Remove(name); err != nil {
			t.Fatal(err)
		}
		// Fixture cleanups run at test end. Closing/reconnecting for the next mode is already the legacy fixture contract.
	}
}

func TestSPObservedQueriesAndReferenceErrors(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	id := f.addConv(t, f.tenantID, f.channelID, "ref", []time.Time{f.clock.Add(-time.Hour)})
	run := f.mustRun(t, &incProvider{verdict: "PASS"})
	_ = run
	job := f.job(t)
	var conv models.Conversation
	if err := db.DB.First(&conv, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	queryCount := 0
	name := "sp_queries_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
	if err := db.DB.Callback().Query().After("gorm:query").Register(name, func(*gorm.DB) { queryCount++ }); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.DB.Callback().Query().Remove(name) })
	legacy, unchanged, errs, _ := prepareOrdinaryIncremental(context.Background(), job, []models.Conversation{conv})
	n := queryCount
	queryCount = 0
	c := newPreparationCollector(job, models.JobRun{ID: pkg.NewUUID(), TenantID: job.TenantID, JobID: job.ID}, ordinaryPlan())
	c.selection(1)
	observed, u, e, _ := prepareOrdinaryIncrementalObserved(context.Background(), job, []models.Conversation{conv}, c)
	if queryCount != n || len(legacy) != len(observed) || u != unchanged || e != errs || u != 1 || c.freeze().Entries[0].Reference.State != "VERIFIED" {
		t.Fatalf("query/outcome drift: %d/%d %+v", queryCount, n, c.freeze())
	}
	f.exec(t, "UPDATE analysis_snapshots SET digest = ? WHERE job_run_id = ?", strings.Repeat("0", 64), run.ID)
	c = newPreparationCollector(job, models.JobRun{ID: pkg.NewUUID(), TenantID: job.TenantID, JobID: job.ID}, ordinaryPlan())
	c.selection(1)
	_, u, e, _ = prepareOrdinaryIncrementalObserved(context.Background(), job, []models.Conversation{conv}, c)
	r := c.freeze()
	if u != 0 || e != 1 || r.Counts["SOURCE_VERSION_ERROR"] != 1 || r.Entries[0].Reference.State != "PROVENANCE_UNVERIFIABLE" {
		t.Fatalf("corrupt reference %+v", r)
	}
}

func TestSPInputSelectionAndPreparationPanic(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	job := f.job(t)
	job.InputChannelIDs = "not-json"
	run, err := NewAnalyzer(&config.Config{}).RunJob(context.Background(), job)
	if err == nil {
		t.Fatal("input failure lost")
	}
	r := spReload(t, run)
	if r.Selected != nil || r.SelectionStatus != "NOT_ATTEMPTED" {
		t.Fatalf("input %+v", r)
	}
	job = f.job(t)
	f.addConv(t, f.tenantID, f.channelID, "panic", []time.Time{f.clock.Add(-time.Hour)})
	name := "sp_fault_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
	mode := "selection"
	if err := db.DB.Callback().Query().After("gorm:query").Register(name, func(tx *gorm.DB) {
		if mode == "selection" && tx.Statement.Table == "conversations" {
			tx.AddError(errors.New("synthetic secret driver text"))
		}
		if mode == "panic" && tx.Statement.Table == "messages" {
			panic("synthetic secret panic text")
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.DB.Callback().Query().Remove(name) })
	run, err = NewAnalyzer(&config.Config{}).RunJob(context.Background(), job)
	if err == nil {
		t.Fatal("selection failure lost")
	}
	r = spReload(t, run)
	if r.Selected != nil || r.SelectionStatus != "FAILED" {
		t.Fatalf("selection %+v", r)
	}
	mode = "panic"
	run, err = NewAnalyzer(&config.Config{}).RunJob(context.Background(), job)
	if !errors.Is(err, ErrJobRunPanic) {
		t.Fatalf("panic %v", err)
	}
	r = spReload(t, run)
	if r.Visited != 1 || r.Counts["PREPARATION_INTERRUPTED"] != 1 || r.ScanComplete {
		t.Fatalf("panic trace %+v", r)
	}
}

func TestSPCancellationAndProviderPanic(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "cancel", []time.Time{f.clock.Add(-time.Hour)})
	f.addConv(t, f.tenantID, f.channelID, "remaining", []time.Time{f.clock.Add(-30 * time.Minute)})
	job := f.job(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	run, err := NewAnalyzerWithProvider(&config.Config{}, &incProvider{verdict: "PASS"}).RunJob(ctx, job)
	if run != nil {
		r := spReload(t, run)
		if r.Visited != 0 {
			t.Fatal("cancelled prep visited")
		}
	} else if err == nil {
		t.Fatal("cancelled admission false success")
	}
	name := "sp_cancel_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
	if err := db.DB.Callback().Query().After("gorm:query").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == "messages" {
			if _, e := CancelJobRun(job.TenantID, job.ID, ""); e != nil {
				t.Errorf("cancel: %v", e)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.DB.Callback().Query().Remove(name) })
	constructs := 0
	a := NewAnalyzer(&config.Config{})
	a.providerResolver = func(models.Job) (ai.AIProvider, error) { constructs++; return &incProvider{verdict: "PASS"}, nil }
	run, err = a.RunJob(context.Background(), job)
	if err := db.DB.Callback().Query().Remove(name); err != nil {
		t.Fatal(err)
	}
	if err != nil || run.Status != "cancelled" {
		t.Fatalf("cancel precedence %v %+v", err, run)
	}
	cr := spReload(t, run)
	if cr.Visited != 1 || cr.Unvisited == nil || *cr.Unvisited != 1 || cr.StopReason != "CONTEXT_CANCELLED" || constructs != 0 {
		t.Fatalf("midprep prefix %+v constructors%d", cr, constructs)
	}
	p := &incProvider{verdict: "PASS", onCall: func(int) { panic("synthetic secret provider panic") }}
	run, err = f.run(t, p)
	if !errors.Is(err, ErrJobRunPanic) {
		t.Fatalf("provider panic %v", err)
	}
	r := spReload(t, run)
	if !r.ScanComplete || r.Counts["PREPARATION_INTERRUPTED"] != 0 || r.Counts["PREPARED_FOR_INFERENCE"] != 2 {
		t.Fatalf("completed preparation damaged %+v", r)
	}
}

func TestSPTerminalFailureAndLegacyClose(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "terminal", []time.Time{f.clock.Add(-time.Hour)})
	job := f.job(t)
	trigger := "sp_terminal_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
	f.exec(t, "CREATE TRIGGER "+trigger+" BEFORE UPDATE ON job_runs FOR EACH ROW BEGIN IF NEW.job_id = '"+job.ID+"' AND NEW.status <> 'running' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'synthetic terminal error'; END IF; END")
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trigger) })
	run, err := NewAnalyzer(&config.Config{}).RunJob(context.Background(), job)
	if err == nil || run == nil {
		t.Fatal("terminal failure lost")
	}
	returned := spRead(t, run.Summary)
	if returned.Counts["PREPARED_FOR_INFERENCE"] != 1 {
		t.Fatal("failed-write observation missing")
	}
	var stored models.JobRun
	if err := db.DB.First(&stored, "id = ?", run.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != "running" {
		t.Fatal("failed terminal recorded false success")
	}
	f.exec(t, "DROP TRIGGER "+trigger)
	f.exec(t, "UPDATE job_runs SET status='error' WHERE id = ?", run.ID)
	res, err := ReserveJobRun(context.Background(), job, 2*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer res.owner.release()
	legacy := res.run
	legacy.Summary = "returned sentinel"
	if _, err := closeOwnedRun(res.owner, &legacy, job, "synthetic legacy error"); err != nil {
		t.Fatal(err)
	}
	if legacy.Summary != "returned sentinel" {
		t.Fatal("legacy returned Summary changed")
	}
	stored = models.JobRun{}
	if err := db.DB.First(&stored, "id = ?", legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Summary != "{}" {
		t.Fatalf("legacy persistence %q", stored.Summary)
	}
}

func TestSPSingleProviderErrorReceipt(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "error", []time.Time{f.clock.Add(-time.Hour)})
	p := &incProvider{fail: true}
	run, err := f.run(t, p)
	if err != nil || run.Status != "error" || p.callCount() != 1 {
		t.Fatalf("provider error %v %+v", err, run)
	}
	r := spReload(t, run)
	if r.Counts["PREPARED_FOR_INFERENCE"] != 1 {
		t.Fatal("provider error trace missing")
	}
}

func TestSPAllExplicitModesCapAndContext(t *testing.T) {
	for _, mode := range []analysisMode{modeTestRun, modeUnanalyzed, modeSinceLast, modeFull} {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "first", []time.Time{f.clock.Add(-2 * time.Hour), f.clock.Add(-time.Hour)})
		f.addConv(t, f.tenantID, f.channelID, "second", []time.Time{f.clock.Add(-30 * time.Minute)})
		job := f.job(t)
		p := &incProvider{verdict: "PASS"}
		a := NewAnalyzerWithProvider(&config.Config{}, p)
		var run *models.JobRun
		var err error
		switch mode {
		case modeTestRun:
			run, err = a.RunJobWithLimit(context.Background(), job, 1)
		case modeUnanalyzed:
			run, err = a.RunJobUnanalyzed(context.Background(), job, 1)
		case modeSinceLast:
			run, err = a.RunJobSinceLast(context.Background(), job, 1)
		case modeFull:
			run, err = a.RunJobFullWithParams(context.Background(), job, pkg.ToVN(f.clock.Add(-24*time.Hour)).Format("2006-01-02"), pkg.ToVN(f.clock).Format("2006-01-02"), 1)
		}
		if err != nil || run.Status != "success" {
			t.Fatalf("mode%s %v %+v", mode, err, run)
		}
		r := spReload(t, run)
		if r.Mode != string(mode) || r.Selected == nil || *r.Selected != 1 || r.Visited != 1 || r.Counts["PREPARED_FOR_INFERENCE"] != 1 || r.Counts["UNCHANGED_VERIFIED"] != 0 || len(r.Entries) != 1 || r.Entries[0].Reference.State != "NONE" {
			t.Fatalf("explicit%s %+v", mode, r)
		}
		if p.callCount() != 1 || len(p.transcripts()) != 1 {
			t.Fatal("explicit call count changed")
		}
		transcript := p.transcripts()[0]
		if strings.Contains(transcript, "first") && len(transcriptMessageIDs(transcript)) != 2 {
			t.Fatal("capped selection truncated full context")
		}
		if f.checkpoint(t) != nil {
			t.Fatal("explicit advanced ordinary checkpoint")
		}
	}
}

func TestSPSnapshotLookupMissingAndMisboundObservations(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	id := f.addConv(t, f.tenantID, f.channelID, "source", []time.Time{f.clock.Add(-time.Hour)})
	run := f.mustRun(t, &incProvider{verdict: "PASS"})
	job := f.job(t)
	var conv models.Conversation
	if err := db.DB.First(&conv, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	observe := func() preparationReceipt {
		c := newPreparationCollector(job, models.JobRun{ID: pkg.NewUUID(), JobID: job.ID, TenantID: job.TenantID}, ordinaryPlan())
		c.selection(1)
		prepared, u, e, _ := prepareOrdinaryIncrementalObserved(context.Background(), job, []models.Conversation{conv}, c)
		if len(prepared) != 0 || u != 0 || e != 1 {
			t.Fatalf("failure became skip %d/%d/%d", len(prepared), u, e)
		}
		return *c.freeze()
	}
	for _, table := range []string{"messages", "job_results", "analysis_snapshots"} {
		name := "sp_query_fault_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
		if err := db.DB.Callback().Query().After("gorm:query").Register(name, func(tx *gorm.DB) {
			if tx.Statement.Table == table {
				tx.AddError(errors.New("synthetic query fault"))
			}
		}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.DB.Callback().Query().Remove(name) })
		r := observe()
		if table == "messages" {
			if r.Counts["SNAPSHOT_ERROR"] != 1 || r.Entries[0].Digest != "" {
				t.Fatalf("snapshot error %+v", r)
			}
		} else {
			if r.Counts["SOURCE_VERSION_ERROR"] != 1 || r.Entries[0].Reference.State != "LOOKUP_FAILED" {
				t.Fatalf("lookup error %+v", r)
			}
		}
		if err := db.DB.Callback().Query().Remove(name); err != nil {
			t.Fatal(err)
		}
	}
	var result models.JobResult
	if err := db.DB.Where("job_run_id = ? AND result_type = ?", run.ID, "conversation_evaluation").First(&result).Error; err != nil {
		t.Fatal(err)
	}
	if result.AnalysisSnapshotID == nil {
		t.Fatal("fixture snapshot missing")
	}
	original := *result.AnalysisSnapshotID
	f.exec(t, "UPDATE job_results SET analysis_snapshot_id = ? WHERE id = ?", pkg.NewUUID(), result.ID)
	r := observe()
	if r.Entries[0].Reference.State != "PROVENANCE_UNVERIFIABLE" {
		t.Fatalf("missing %+v", r)
	}
	f.exec(t, "UPDATE job_results SET analysis_snapshot_id = ? WHERE id = ?", original, result.ID)
	other := f.addConv(t, f.tenantID, f.channelID, "other", []time.Time{f.clock.Add(-time.Hour)})
	f.exec(t, "UPDATE analysis_snapshots SET conversation_id = ? WHERE id = ?", other, original)
	r = observe()
	if r.Entries[0].Reference.State != "PROVENANCE_UNVERIFIABLE" || r.Entries[0].Reference.SnapshotID != "" {
		t.Fatalf("misbound %+v", r)
	}
}

func TestSPInitialAndProgressRecordingGap(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		f := setupIncFixture(t, false, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "gap", []time.Time{f.clock.Add(-time.Hour)})
		name := "sp_write_fault_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
		writes := 0
		if err := db.DB.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
			if tx.Statement.Table == "job_runs" {
				if m, ok := tx.Statement.Dest.(map[string]interface{}); ok {
					if _, ok := m["summary"]; ok {
						writes++
						if writes == failAt {
							tx.AddError(errors.New("synthetic_write_canary"))
						}
					}
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.DB.Callback().Update().Remove(name) })
		var logs bytes.Buffer
		old := log.Writer()
		log.SetOutput(&logs)
		t.Cleanup(func() { log.SetOutput(old) })
		run, err := f.run(t, &incProvider{verdict: "PASS"})
		log.SetOutput(old)
		if err := db.DB.Callback().Update().Remove(name); err != nil {
			t.Fatal(err)
		}
		if err != nil || run.Status != "success" || f.checkpoint(t) == nil {
			t.Fatalf("observation write failure changed outcome %v %+v", err, run)
		}
		r := spReload(t, run)
		if r.Counts["PREPARED_FOR_INFERENCE"] != 1 {
			t.Fatal("final trace lost")
		}
		found := false
		for _, line := range strings.Split(logs.String(), "\n") {
			if strings.Contains(line, "source preparation receipt recording gap") {
				found = true
				if strings.Contains(line, "synthetic_write_canary") || strings.Contains(line, f.jobID) || strings.Contains(line, f.tenantID) {
					t.Fatal("gap diagnostic leaked raw identifiers/error")
				}
			}
		}
		if !found {
			t.Fatal("missing recording-gap diagnostic")
		}
	}
}

// Compile-time interface check protects use of a synthetic application provider only.
var _ ai.AIProvider = (*incProvider)(nil)
