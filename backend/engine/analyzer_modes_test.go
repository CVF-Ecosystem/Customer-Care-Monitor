package engine

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-027 (F05): analyzer mode, count cap, Vietnam date bounds, snapshot scope and
// checkpoint policy are independent. Synthetic provider through the real Analyzer on a disposable
// MySQL; no real AI or channel is called. Every scenario runs in single and batch mode.

// incLoc is the driver location of the fixture connection ("" = the test DSN as given).
var incLoc string

var incLocRe = regexp.MustCompile(`loc=[^&]*`)

func connectIncDB(t *testing.T) {
	t.Helper()
	if incLoc == "" {
		connectTestDB(t)
		return
	}
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("bo qua: TEST_DB_DSN chua duoc thiet lap")
	}
	repl := "loc=" + url.QueryEscape(incLoc)
	if incLocRe.MatchString(dsn) {
		dsn = incLocRe.ReplaceAllString(dsn, repl)
	} else {
		dsn += "&" + repl
	}
	if err := db.Connect(dsn, false); err != nil {
		t.Skipf("bo qua: khong ket noi duoc DB test: %v", err)
	}
	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate loi: %v", err)
	}
}

func withIncLoc(t *testing.T, loc string) {
	t.Helper()
	prev := incLoc
	incLoc = loc
	t.Cleanup(func() { incLoc = prev })
}

// ---- plan validation ----

func TestPlanValidationRejectsBeforeAnySideEffect(t *testing.T) {
	dates := pkg.BusinessRange{}
	if r, err := pkg.ParseBusinessRange("2026-10-02", "2026-10-02"); err != nil {
		t.Fatal(err)
	} else {
		dates = r
	}
	bad := []struct {
		name  string
		mode  analysisMode
		limit int
		dates pkg.BusinessRange
	}{
		{"negative cap", modeUnanalyzed, -1, pkg.BusinessRange{}},
		{"test run without cap", modeTestRun, 0, pkg.BusinessRange{}},
		{"ordinary with cap", modeOrdinary, 2, pkg.BusinessRange{}},
		{"dates on unanalyzed", modeUnanalyzed, 0, dates},
		{"dates on since_last", modeSinceLast, 0, dates},
		{"dates on test run", modeTestRun, 3, dates},
	}
	for _, c := range bad {
		if _, err := newPlan(c.mode, c.limit, c.dates); !errors.Is(err, ErrInvalidRunParameters) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	for _, c := range []struct {
		mode  analysisMode
		limit int
		dates pkg.BusinessRange
	}{
		{modeOrdinary, 0, pkg.BusinessRange{}}, {modeTestRun, 3, pkg.BusinessRange{}}, {modeFull, 0, pkg.BusinessRange{}},
		{modeFull, 5, dates}, {modeUnanalyzed, 4, pkg.BusinessRange{}}, {modeSinceLast, 0, pkg.BusinessRange{}},
	} {
		if p, err := newPlan(c.mode, c.limit, c.dates); err != nil || p.mode != c.mode || p.limit != c.limit {
			t.Errorf("valid plan %v/%d: %+v %v", c.mode, c.limit, p, err)
		}
	}
}

func TestEntryPointsRejectInvalidParametersWithoutRunOrProvider(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-2 * time.Hour)})
	p := &incProvider{}
	an := NewAnalyzerWithProvider(&config.Config{}, p)
	ctx := context.Background()
	before := f.count(t, "job_runs")
	for name, call := range map[string]func() error{
		"test run cap 0":      func() error { _, e := an.RunJobWithLimit(ctx, f.job(t), 0); return e },
		"test run cap -2":     func() error { _, e := an.RunJobWithLimit(ctx, f.job(t), -2); return e },
		"unanalyzed cap -1":   func() error { _, e := an.RunJobUnanalyzed(ctx, f.job(t), -1); return e },
		"since last cap -1":   func() error { _, e := an.RunJobSinceLast(ctx, f.job(t), -1); return e },
		"full cap -1":         func() error { _, e := an.RunJobFullWithParams(ctx, f.job(t), "", "", -1); return e },
		"provider run cap -1": func() error { _, e := an.RunJobWithProvider(ctx, f.job(t), -1, p); return e },
	} {
		if err := call(); !errors.Is(err, ErrInvalidRunParameters) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, r := range map[string][2]string{
		"reversed": {"2026-10-03", "2026-10-02"}, "bad month": {"2026-13-01", ""}, "not a date": {"", "hom nay"}, "datetime": {"2026-10-01T00:00:00Z", ""},
	} {
		if _, err := an.RunJobFullWithParams(ctx, f.job(t), r[0], r[1], 0); !errors.Is(err, pkg.ErrInvalidDateRange) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if got := f.count(t, "job_runs"); got != before || p.callCount() != 0 {
		t.Fatalf("a rejected call left %d new runs and %d provider calls", got-before, p.callCount())
	}
}

// ---- mode matrix ----

type modesData struct {
	ev, otherEval, newUn, oldUn, nullLast string
	foreignChannel, foreignTenant         string
	sentinel                              time.Time
	base                                  time.Time
}

// newModesData builds the dataset: ev was evaluated by the main job; otherEval only by another job
// of the same tenant (so it is still unevaluated for the main job); newUn/oldUn/nullLast were never
// evaluated; the two foreign conversations belong to another channel / tenant.
func newModesData(t *testing.T, f *incFixture) modesData {
	t.Helper()
	d := modesData{}
	d.ev = f.addConv(t, f.tenantID, f.channelID, "ev", []time.Time{f.clock.Add(-2 * time.Hour)})
	f.mustRun(t, &incProvider{}) // evaluates ev only; the clock advances one hour
	d.base = f.clock

	// another job of the same tenant and channel evaluates ev and otherEval
	otherJob := "job-inc-o-" + pkg.NewUUID()[:6]
	f.exec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, schedule_cron, is_active, outputs, output_schedule, created_at, updated_at) SELECT ?, tenant_id, 'Other', job_type, input_channel_ids, rules_content, rules_config, schedule_type, schedule_cron, is_active, outputs, output_schedule, NOW(), NOW() FROM jobs WHERE id = ?`, otherJob, f.jobID)
	d.otherEval = f.addConv(t, f.tenantID, f.channelID, "oe", []time.Time{d.base.Add(-4 * time.Hour)})
	var oj models.Job
	if err := db.DB.First(&oj, "id = ?", otherJob).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewAnalyzerWithProvider(&config.Config{}, &incProvider{}).RunJob(context.Background(), oj); err != nil {
		t.Fatal(err)
	}

	d.newUn = f.addConv(t, f.tenantID, f.channelID, "nu", []time.Time{d.base.Add(-time.Hour)})
	d.oldUn = f.addConv(t, f.tenantID, f.channelID, "ou", []time.Time{d.base.Add(-30 * 24 * time.Hour)})
	d.nullLast = f.addConv(t, f.tenantID, f.channelID, "nl", []time.Time{d.base.Add(-time.Hour)})
	f.exec(t, "UPDATE conversations SET last_message_at = NULL WHERE id = ?", d.nullLast)
	d.foreignChannel = f.addConv(t, f.tenantID, f.otherChannelID, "fc", []time.Time{d.base.Add(-time.Hour)})
	d.foreignTenant = f.addConv(t, f.otherTenantID, f.otherTenantChannelID, "ft", []time.Time{d.base.Add(-time.Hour)})

	d.sentinel = d.base.Add(-48 * time.Hour).UTC().Truncate(time.Second)
	f.exec(t, "UPDATE jobs SET last_run_at = ? WHERE id = ?", d.sentinel, f.jobID)
	return d
}

func TestExplicitModeMatrix(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		type tc struct {
			name  string
			run   func(an *Analyzer, job models.Job) (*models.JobRun, error)
			want  func(d modesData) []string
			wcnt  int
			isRun bool
		}
		ctx := context.Background()
		cases := []tc{
			{"unanalyzed", func(an *Analyzer, j models.Job) (*models.JobRun, error) { return an.RunJobUnanalyzed(ctx, j, 0) },
				func(d modesData) []string { return []string{d.otherEval, d.newUn, d.oldUn, d.nullLast} }, 4, false},
			{"unanalyzed cap 2 keeps the unanalyzed mode and the oldest-first order", func(an *Analyzer, j models.Job) (*models.JobRun, error) { return an.RunJobUnanalyzed(ctx, j, 2) },
				func(d modesData) []string { return []string{d.nullLast, d.oldUn} }, 2, false},
			{"full", func(an *Analyzer, j models.Job) (*models.JobRun, error) { return an.RunJobFull(ctx, j) },
				func(d modesData) []string { return []string{d.ev, d.otherEval, d.newUn, d.oldUn, d.nullLast} }, 5, false},
			{"full cap 2 still re-analyzes (cap is not unanalyzed)", func(an *Analyzer, j models.Job) (*models.JobRun, error) {
				return an.RunJobFullWithParams(ctx, j, "", "", 2)
			},
				func(d modesData) []string { return []string{d.nullLast, d.oldUn} }, 2, false},
			{"since last uses the newest evaluated last message as anchor", func(an *Analyzer, j models.Job) (*models.JobRun, error) { return an.RunJobSinceLast(ctx, j, 0) },
				func(d modesData) []string { return []string{d.newUn} }, 1, false},
			{"test run window excludes old, null and evaluated", func(an *Analyzer, j models.Job) (*models.JobRun, error) { return an.RunJobWithLimit(ctx, j, 5) },
				func(d modesData) []string { return []string{d.otherEval, d.newUn} }, 2, true},
			{"test run cap 1 takes the oldest candidate", func(an *Analyzer, j models.Job) (*models.JobRun, error) { return an.RunJobWithLimit(ctx, j, 1) },
				func(d modesData) []string { return []string{d.otherEval} }, 1, true},
		}
		for _, c := range cases {
			c := c
			t.Run(c.name, func(t *testing.T) {
				f := setupIncFixture(t, batch, "qc_analysis")
				d := newModesData(t, f)
				beforeJob := f.job(t)
				p := &incProvider{}
				run, err := c.run(NewAnalyzerWithProvider(&config.Config{}, p), beforeJob)
				if err != nil || run.Status != "success" {
					t.Fatalf("run: %v %v", err, run)
				}
				if got := evaluatedIn(t, run.ID); !eqIDs(got, sortedIDs(c.want(d)...)) {
					t.Fatalf("evaluated %v, want %v", got, sortedIDs(c.want(d)...))
				}
				if len(p.transcripts()) != c.wcnt {
					t.Fatalf("provider saw %d transcripts, want %d", len(p.transcripts()), c.wcnt)
				}
				after := f.job(t)
				if after.LastRunAt == nil || !after.LastRunAt.Equal(d.sentinel) {
					t.Fatalf("checkpoint moved: %v, want %v", after.LastRunAt, d.sentinel)
				}
				// every explicit mode, a test run included, records the terminal job status and
				// updated_at without a checkpoint
				if after.LastRunStatus != "success" || !after.UpdatedAt.After(beforeJob.UpdatedAt) {
					t.Fatalf("explicit run did not record its status: %q, updated_at %v -> %v", after.LastRunStatus, beforeJob.UpdatedAt, after.UpdatedAt)
				}
			})
		}
	})
}

// With no evaluated conversation, since-last behaves as the unanalyzed mode.
func TestSinceLastWithoutAnchorIsUnanalyzed(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	a := f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-40 * 24 * time.Hour)})
	b := f.addConv(t, f.tenantID, f.channelID, "b", []time.Time{f.clock.Add(-time.Hour)})
	run, err := NewAnalyzerWithProvider(&config.Config{}, &incProvider{}).RunJobSinceLast(context.Background(), f.job(t), 0)
	if err != nil || !eqIDs(evaluatedIn(t, run.ID), sortedIDs(a, b)) {
		t.Fatalf("%v %v", err, evaluatedIn(t, run.ID))
	}
}

// A failing anchor query fails the run: it must not fall back to "no anchor" and analyze
// everything. The fault is proved reached (the error names the anchor and nothing was analyzed).
func TestSinceLastAnchorErrorFailsTheRun(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-2 * time.Hour)})
	f.mustRun(t, &incProvider{})
	f.addConv(t, f.tenantID, f.channelID, "b", []time.Time{f.clock.Add(-time.Hour)})
	f.exec(t, "RENAME TABLE job_results TO job_results_r027_hold")
	restored := false
	restore := func() {
		if !restored {
			restored = true
			db.DB.Exec("RENAME TABLE job_results_r027_hold TO job_results")
		}
	}
	t.Cleanup(restore)
	p := &incProvider{}
	run, err := NewAnalyzerWithProvider(&config.Config{}, p).RunJobSinceLast(context.Background(), f.job(t), 0)
	restore()
	// the failure is checked and bounded: no SQL/table text reaches the error or the stored run
	if err == nil || strings.Contains(err.Error(), "job_results") || !strings.Contains(err.Error(), "danh sách cuộc chat") {
		t.Fatalf("expected a bounded candidate-selection error, got %v", err)
	}
	if run == nil || run.Status != "error" || p.callCount() != 0 {
		t.Fatalf("run %+v, provider calls %d", run, p.callCount())
	}
	var stored models.JobRun
	if e := db.DB.First(&stored, "id = ?", run.ID).Error; e != nil || stored.Status != "error" {
		t.Fatalf("stored run %q (%v)", stored.Status, e)
	}
}

// ---- Vietnam dates and full snapshots ----

func TestFullModeVietnamDatesKeepFullSnapshot(t *testing.T) {
	for _, loc := range []string{"UTC", "Asia/Ho_Chi_Minh"} {
		loc := loc
		t.Run("loc="+strings.ReplaceAll(loc, "/", "_"), func(t *testing.T) {
			withIncLoc(t, loc)
			f := setupIncFixture(t, false, "qc_analysis")
			at := func(s string) time.Time {
				v, err := time.Parse(time.RFC3339, s)
				if err != nil {
					t.Fatal(err)
				}
				return v
			}
			a := f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{at("2026-09-20T03:00:00Z"), at("2026-10-01T17:00:00Z")}) // VN Oct 2 00:00:00
			b := f.addConv(t, f.tenantID, f.channelID, "b", []time.Time{at("2026-10-01T16:59:59Z")})                             // VN Oct 1 23:59:59
			c := f.addConv(t, f.tenantID, f.channelID, "c", []time.Time{at("2026-10-02T16:59:59Z")})                             // VN Oct 2 23:59:59
			d := f.addConv(t, f.tenantID, f.channelID, "d", []time.Time{at("2026-10-02T17:00:00Z")})                             // VN Oct 3 00:00:00
			cases := []struct {
				name, from, to string
				limit          int
				want           []string
			}{
				{"one VN day includes both its edges", "2026-10-02", "2026-10-02", 0, []string{a, c}},
				{"the previous VN day", "2026-10-01", "2026-10-01", 0, []string{b}},
				{"from only", "2026-10-03", "", 0, []string{d}},
				{"to only", "", "2026-10-01", 0, []string{b}},
				{"cap after the date filter", "2026-10-02", "2026-10-02", 1, []string{a}},
			}
			for _, tc := range cases {
				p := &incProvider{}
				run, err := NewAnalyzerWithProvider(&config.Config{}, p).RunJobFullWithParams(context.Background(), f.job(t), tc.from, tc.to, tc.limit)
				if err != nil || run.Status != "success" {
					t.Fatalf("%s: %v %v", tc.name, err, run)
				}
				if got := evaluatedIn(t, run.ID); !eqIDs(got, sortedIDs(tc.want...)) {
					t.Fatalf("%s: evaluated %v, want %v", tc.name, got, sortedIDs(tc.want...))
				}
			}
			// The date bound selects the conversation; it never cuts its context: a's
			// September message is sent and saved with the October one.
			p := &incProvider{}
			run, err := NewAnalyzerWithProvider(&config.Config{}, p).RunJobFullWithParams(context.Background(), f.job(t), "2026-10-02", "2026-10-02", 1)
			if err != nil || len(p.transcripts()) != 1 {
				t.Fatalf("%v %d", err, len(p.transcripts()))
			}
			if ids := transcriptMessageIDs(p.transcripts()[0]); len(ids) != 2 {
				t.Fatalf("snapshot sent %d messages, want 2: %v", len(ids), ids)
			}
			f.assertSavedSnapshotMatches(t, run.ID, a, p.transcripts()[0], 2)
		})
	}
}

// ---- terminal state ----

func TestExplicitModeCancellationKeepsCheckpoint(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		for _, kind := range []string{"context", "accepted-cancel"} {
			kind := kind
			t.Run(kind, func(t *testing.T) {
				f := setupIncFixture(t, batch, "qc_analysis")
				d := newModesData(t, f)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				p := &incProvider{}
				switch kind {
				case "context":
					cancel()
				case "accepted-cancel":
					p.onCall = func(n int) {
						if n == 1 {
							if _, err := CancelJobRun(f.tenantID, f.jobID, ""); err != nil {
								t.Errorf("cancel request: %v", err)
							}
						}
					}
				}
				run, err := NewAnalyzerWithProvider(&config.Config{}, p).RunJobUnanalyzed(ctx, f.job(t), 0)
				if err != nil {
					t.Fatal(err)
				}
				if kind == "context" && (p.callCount() != 0 || run.Status != "partial") {
					t.Fatalf("cancelled context: %d calls, status %s", p.callCount(), run.Status)
				}
				if kind == "accepted-cancel" && (p.callCount() == 0 || run.Status != "cancelled") {
					t.Fatalf("stored cancel: %d calls, status %s", p.callCount(), run.Status)
				}
				var stored models.JobRun
				if e := db.DB.First(&stored, "id = ?", run.ID).Error; e != nil || stored.Status == "success" || stored.Status == "running" {
					t.Fatalf("stored run %q (%v)", stored.Status, e)
				}
				if cp := f.checkpoint(t); cp == nil || !cp.Equal(d.sentinel) {
					t.Fatalf("checkpoint moved: %v", cp)
				}
			})
		}
	})
}

func runExplicit(an *Analyzer, job models.Job, mode string) (*models.JobRun, error) {
	ctx := context.Background()
	switch mode {
	case "unanalyzed":
		return an.RunJobUnanalyzed(ctx, job, 1)
	case "full":
		return an.RunJobFullWithParams(ctx, job, "", "", 1)
	case "since-last":
		return an.RunJobSinceLast(ctx, job, 1)
	default:
		return an.RunJobWithLimit(ctx, job, 1)
	}
}

var explicitModeNames = []string{"unanalyzed", "full", "since-last", "test-run"}

func TestExplicitModeFinalizeFailure(t *testing.T) {
	for _, mode := range explicitModeNames {
		mode := mode
		for _, fault := range []string{"job-write-error", "job-row-missing"} {
			fault := fault
			t.Run(mode+"/"+fault, func(t *testing.T) {
				f := setupIncFixture(t, false, "qc_analysis")
				d := newModesData(t, f)
				f.exec(t, "UPDATE jobs SET output_schedule = 'instant' WHERE id = ?", f.jobID)
				notified := 0
				sendJobNotifications = func(context.Context, models.Job, models.JobRun) error { notified++; return nil }
				p := &incProvider{}
				if fault == "job-write-error" {
					trigger := "r027_job_" + f.jobID[len(f.jobID)-6:]
					f.exec(t, fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE ON jobs FOR EACH ROW BEGIN IF NEW.id = '%s' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'synthetic terminal write failure'; END IF; END", trigger, f.jobID))
					t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trigger) })
				} else {
					// the job leaves the tenant while the run works: the scoped terminal write finds no target
					p.onCall = func(int) { f.exec(t, "UPDATE jobs SET tenant_id = ? WHERE id = ?", f.otherTenantID, f.jobID) }
				}
				run, err := runExplicit(NewAnalyzerWithProvider(&config.Config{}, p), f.job(t), mode)
				// the fault was reached (the provider ran) and the failure is reported, not hidden
				if p.callCount() == 0 {
					t.Fatalf("the run never reached the provider, so the fault was not exercised")
				}
				if err == nil || run.Status != "error" || notified != 0 {
					t.Fatalf("finalize failure not reported: err %v status %s notified %d", err, run.Status, notified)
				}
				var stored models.JobRun
				if e := db.DB.First(&stored, "id = ?", run.ID).Error; e != nil || stored.Status == "success" || stored.Status == "running" {
					t.Fatalf("stored run %q (%v)", stored.Status, e)
				}
				if fault == "job-write-error" {
					if cp := f.checkpoint(t); cp == nil || !cp.Equal(d.sentinel) {
						t.Fatalf("checkpoint moved: %v", cp)
					}
				}
			})
		}
	}
}

// Positive control: without a fault the same explicit runs finish, notify once and keep the checkpoint.
func TestExplicitModeFinalizeControlNotifiesOnce(t *testing.T) {
	for _, mode := range explicitModeNames {
		mode := mode
		t.Run(mode, func(t *testing.T) {
			f := setupIncFixture(t, false, "qc_analysis")
			d := newModesData(t, f)
			f.exec(t, "UPDATE jobs SET output_schedule = 'instant' WHERE id = ?", f.jobID)
			notified := 0
			sendJobNotifications = func(context.Context, models.Job, models.JobRun) error { notified++; return nil }
			run, err := runExplicit(NewAnalyzerWithProvider(&config.Config{}, &incProvider{}), f.job(t), mode)
			if err != nil || run.Status != "success" || notified != 1 {
				t.Fatalf("control: err %v status %s notified %d", err, run.Status, notified)
			}
			if cp := f.checkpoint(t); cp == nil || !cp.Equal(d.sentinel) {
				t.Fatalf("checkpoint moved: %v", cp)
			}
		})
	}
}

// Only the ordinary unlimited run advances the checkpoint, even after explicit runs.
func TestOnlyOrdinaryRunWritesCheckpoint(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	d := newModesData(t, f)
	an := NewAnalyzerWithProvider(&config.Config{}, &incProvider{})
	if _, err := an.RunJobFull(context.Background(), f.job(t)); err != nil {
		t.Fatal(err)
	}
	if cp := f.checkpoint(t); !cp.Equal(d.sentinel) {
		t.Fatalf("full run moved the checkpoint to %v", cp)
	}
	scan := f.clock
	if _, err := an.RunJob(context.Background(), f.job(t)); err != nil {
		t.Fatal(err)
	}
	if cp := f.checkpoint(t); cp == nil || !cp.Equal(scan.Truncate(time.Second)) {
		t.Fatalf("ordinary run checkpoint %v, want %v", cp, scan.Truncate(time.Second))
	}
}
