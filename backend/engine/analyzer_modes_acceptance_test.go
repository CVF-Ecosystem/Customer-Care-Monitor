package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-027-R1 (F05-R1-04): the remaining SPEC acceptance groups. Synthetic provider, real
// Analyzer, disposable MySQL, no real AI/channel call.

// insertEval writes a historical result row of this job directly (a legacy evaluation has no
// snapshot). kind is the result_type; the parent run is created on demand per (job, tenant).
func (f *incFixture) insertEval(t *testing.T, jobID, tenantID, convID, kind, severity string) {
	t.Helper()
	runID := pkg.NewUUID()
	f.exec(t, `INSERT INTO job_runs (id, job_id, tenant_id, started_at, finished_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), NOW(), 'success', '{}', NOW())`, runID, jobID, tenantID)
	f.exec(t, `INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, created_at) VALUES (?, ?, ?, ?, ?, ?, '', '', '{}', NOW())`,
		pkg.NewUUID(), runID, tenantID, convID, kind, severity)
}

func (f *incFixture) addOtherJob(t *testing.T) string {
	t.Helper()
	id := "job-inc-o-" + pkg.NewUUID()[:6]
	f.exec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, schedule_cron, is_active, outputs, output_schedule, created_at, updated_at) SELECT ?, tenant_id, 'Other', job_type, input_channel_ids, rules_content, rules_config, schedule_type, schedule_cron, is_active, outputs, output_schedule, NOW(), NOW() FROM jobs WHERE id = ?`, id, f.jobID)
	return id
}

func (f *incFixture) setSentinel(t *testing.T) time.Time {
	t.Helper()
	s := f.clock.Add(-48 * time.Hour).UTC().Truncate(time.Second)
	f.exec(t, "UPDATE jobs SET last_run_at = ? WHERE id = ?", s, f.jobID)
	return s
}

func (f *incFixture) evaluated(t *testing.T, run *models.JobRun, err error) []string {
	t.Helper()
	if err != nil || run == nil || run.Status != "success" {
		t.Fatalf("run: %v %+v", err, run)
	}
	return evaluatedIn(t, run.ID)
}

// ---- evaluation / anchor isolation ----

func TestExplicitSelectionIsolationControls(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "qc_analysis")
		T := f.clock
		at := func(d time.Duration) []time.Time { return []time.Time{T.Add(d)} }
		passEv := f.addConv(t, f.tenantID, f.channelID, "pass", at(-5*time.Hour))
		skipEv := f.addConv(t, f.tenantID, f.channelID, "skip", at(-3*time.Hour)) // the anchor
		findingOnly := f.addConv(t, f.tenantID, f.channelID, "find", at(-2*time.Hour))
		newer := f.addConv(t, f.tenantID, f.channelID, "newer", at(-time.Hour))
		otherJobEv := f.addConv(t, f.tenantID, f.channelID, "ojev", at(-4*time.Hour))
		oldFresh := f.addConv(t, f.tenantID, f.channelID, "old", at(-30*24*time.Hour))
		foreignEv := f.addConv(t, f.tenantID, f.otherChannelID, "fch", at(-30*time.Minute)) // evaluated, not a current channel
		f.addConv(t, f.otherTenantID, f.otherTenantChannelID, "ften", at(-time.Hour))

		other := f.addOtherJob(t)
		f.insertEval(t, f.jobID, f.tenantID, passEv, "conversation_evaluation", "PASS")
		f.insertEval(t, f.jobID, f.tenantID, skipEv, "conversation_evaluation", "SKIP")
		f.insertEval(t, f.jobID, f.tenantID, findingOnly, "qc_violation", "CAN_CAI_THIEN") // finding only: not evaluated
		f.insertEval(t, f.jobID, f.tenantID, foreignEv, "conversation_evaluation", "PASS")
		f.insertEval(t, other, f.tenantID, otherJobEv, "conversation_evaluation", "PASS")
		sentinel := f.setSentinel(t)
		an := NewAnalyzerWithProvider(&config.Config{}, &incProvider{})
		ctx := context.Background()

		cases := []struct {
			name string
			run  func() (*models.JobRun, error)
			want []string
		}{
			// PASS and SKIP evaluations (legacy, without snapshot) count; finding-only and other-job do not.
			{"unanalyzed", func() (*models.JobRun, error) { return an.RunJobUnanalyzed(ctx, f.job(t), 0) }, []string{findingOnly, newer, otherJobEv, oldFresh}},
			// anchor = the skip evaluation's last message; the other-channel conversation (newer, evaluated)
			// is not a current input channel and does not move it; the equal-time conversation is excluded.
			{"since last", func() (*models.JobRun, error) { return an.RunJobSinceLast(ctx, f.job(t), 0) }, []string{findingOnly, newer}},
			// test run: 7-day window, never evaluated by this job; the 30-day-old conversation is outside it.
			{"test run", func() (*models.JobRun, error) { return an.RunJobWithLimit(ctx, f.job(t), 50) }, []string{findingOnly, newer, otherJobEv}},
		}
		for _, c := range cases {
			run, err := c.run()
			if got := f.evaluated(t, run, err); !eqIDs(got, sortedIDs(c.want...)) {
				t.Fatalf("%s: evaluated %v, want %v", c.name, got, sortedIDs(c.want...))
			}
			if cp := f.checkpoint(t); cp == nil || !cp.Equal(sentinel) {
				t.Fatalf("%s moved the checkpoint to %v", c.name, cp)
			}
			f.exec(t, "DELETE FROM job_results WHERE job_run_id = ?", run.ID) // each case starts from the same evaluations
		}
	})
}

// A historical conversation is selected by the unanalyzed mode regardless of its age, while the
// test run keeps its explicit 7-day window.
func TestUnanalyzedIgnoresAgeAndTestRunKeepsItsWindow(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	old := f.addConv(t, f.tenantID, f.channelID, "old", []time.Time{f.clock.Add(-400 * 24 * time.Hour)})
	edge := f.addConv(t, f.tenantID, f.channelID, "edge", []time.Time{f.clock.Add(-7 * 24 * time.Hour)}) // exactly 7 days: strict > excludes it
	in := f.addConv(t, f.tenantID, f.channelID, "in", []time.Time{f.clock.Add(-7*24*time.Hour + time.Second)})
	an := NewAnalyzerWithProvider(&config.Config{}, &incProvider{})
	run, err := an.RunJobWithLimit(context.Background(), f.job(t), 10)
	if got := f.evaluated(t, run, err); !eqIDs(got, []string{in}) {
		t.Fatalf("test run %v, want only %v", got, in)
	}
	run, err = an.RunJobUnanalyzed(context.Background(), f.job(t), 0)
	if got := f.evaluated(t, run, err); !eqIDs(got, sortedIDs(old, edge)) { // `in` is evaluated now
		t.Fatalf("unanalyzed %v, want %v", got, sortedIDs(old, edge))
	}
}

// An anchor needs a non-NULL last_message_at: evaluated conversations with only NULL times give no
// anchor, so since-last behaves as unanalyzed.
func TestSinceLastNullOnlyAnchorFallsBackToUnanalyzed(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	nullEv := f.addConv(t, f.tenantID, f.channelID, "nullev", []time.Time{f.clock.Add(-time.Hour)})
	f.exec(t, "UPDATE conversations SET last_message_at = NULL WHERE id = ?", nullEv)
	f.insertEval(t, f.jobID, f.tenantID, nullEv, "conversation_evaluation", "PASS")
	oldU := f.addConv(t, f.tenantID, f.channelID, "oldu", []time.Time{f.clock.Add(-90 * 24 * time.Hour)})
	newU := f.addConv(t, f.tenantID, f.channelID, "newu", []time.Time{f.clock.Add(-time.Hour)})
	run, err := NewAnalyzerWithProvider(&config.Config{}, &incProvider{}).RunJobSinceLast(context.Background(), f.job(t), 0)
	if got := f.evaluated(t, run, err); !eqIDs(got, sortedIDs(oldU, newU)) {
		t.Fatalf("evaluated %v, want %v", got, sortedIDs(oldU, newU))
	}
}

// Deterministic order: equal last_message_at conversations are cut by id, for every explicit mode.
func TestExplicitCapOrdersEqualTimesByID(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "qc_analysis")
		same := f.clock.Add(-time.Hour)
		a := f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{same})
		b := f.addConv(t, f.tenantID, f.channelID, "b", []time.Time{same})
		f.addConv(t, f.tenantID, f.channelID, "c", []time.Time{same})
		an := NewAnalyzerWithProvider(&config.Config{}, &incProvider{})
		ctx := context.Background()
		for name, call := range map[string]func() (*models.JobRun, error){
			"full":       func() (*models.JobRun, error) { return an.RunJobFullWithParams(ctx, f.job(t), "", "", 2) },
			"unanalyzed": func() (*models.JobRun, error) { return an.RunJobUnanalyzed(ctx, f.job(t), 2) },
			"test run":   func() (*models.JobRun, error) { return an.RunJobWithLimit(ctx, f.job(t), 2) },
		} {
			run, err := call()
			if got := f.evaluated(t, run, err); !eqIDs(got, sortedIDs(a, b)) {
				t.Fatalf("%s: evaluated %v, want %v", name, got, sortedIDs(a, b))
			}
			f.exec(t, "DELETE FROM job_results WHERE job_run_id = ?", run.ID) // keep the pool unevaluated for the next mode
		}
	})
}

// A classification job's evaluation counts as evaluated for the explicit modes.
func TestClassificationEvaluationCountsForUnanalyzed(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "classification")
		p := &incProvider{jobType: "classification"}
		done := f.addConv(t, f.tenantID, f.channelID, "done", []time.Time{f.clock.Add(-2 * time.Hour)})
		f.mustRun(t, p)
		fresh := f.addConv(t, f.tenantID, f.channelID, "fresh", []time.Time{f.clock.Add(-time.Hour)})
		run, err := NewAnalyzerWithProvider(&config.Config{}, p).RunJobUnanalyzed(context.Background(), f.job(t), 0)
		if got := f.evaluated(t, run, err); !eqIDs(got, []string{fresh}) {
			t.Fatalf("evaluated %v, want only %v (not %v)", got, fresh, done)
		}
	})
}

// Zero-message sources never fabricate an evaluation or a provider call.
func TestEmptySourceNeverFabricatesEvaluation(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "empty", nil)
		f.addConv(t, f.tenantID, f.channelID, "empty2", nil)
		p := &incProvider{}
		an := NewAnalyzerWithProvider(&config.Config{}, p)
		for name, call := range map[string]func() (*models.JobRun, error){
			"unanalyzed": func() (*models.JobRun, error) { return an.RunJobUnanalyzed(context.Background(), f.job(t), 0) },
			"full":       func() (*models.JobRun, error) { return an.RunJobFull(context.Background(), f.job(t)) },
		} {
			run, err := call()
			if got := f.evaluated(t, run, err); len(got) != 0 || p.callCount() != 0 {
				t.Fatalf("%s: evaluations %v, provider calls %d", name, got, p.callCount())
			}
		}
	})
}

// ---- repeated evaluation with exact full snapshots ----

func TestFullAndDateCapReEvaluateAlreadyEvaluatedConversation(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "qc_analysis")
		at := func(s string) time.Time { v, _ := time.Parse(time.RFC3339, s); return v }
		x := f.addConv(t, f.tenantID, f.channelID, "x", []time.Time{at("2026-09-28T02:00:00Z"), at("2026-10-02T03:00:00Z")})
		p := &incProvider{}
		f.mustRun(t, p)
		sentinel := f.setSentinel(t)
		countEval := func() int64 {
			var n int64
			db.DB.Model(&models.JobResult{}).Where("conversation_id = ? AND result_type = 'conversation_evaluation'", x).Count(&n)
			return n
		}
		if countEval() != 1 {
			t.Fatalf("setup: %d evaluations", countEval())
		}
		an := NewAnalyzerWithProvider(&config.Config{}, p)
		for i, c := range []struct {
			name           string
			from, to       string
			limit, wantEvs int
		}{{"full cap 1", "", "", 1, 2}, {"date + cap 1", "2026-10-02", "2026-10-02", 1, 3}} {
			p.reset()
			run, err := an.RunJobFullWithParams(context.Background(), f.job(t), c.from, c.to, c.limit)
			if got := f.evaluated(t, run, err); !eqIDs(got, []string{x}) {
				t.Fatalf("%s: evaluated %v", c.name, got)
			}
			if int(countEval()) != c.wantEvs || len(p.transcripts()) != 1 {
				t.Fatalf("%s: %d evaluations, %d transcripts (case %d)", c.name, countEval(), len(p.transcripts()), i)
			}
			// the exact full local snapshot: both messages, the early one outside the date bound
			f.assertSavedSnapshotMatches(t, run.ID, x, p.transcripts()[0], 2)
			var sum map[string]int
			if err := jsonUnmarshalSummary(run.ID, &sum); err != nil || sum["conversations_found"] != 1 || sum["conversations_analyzed"] != 1 {
				t.Fatalf("%s: summary %v %v", c.name, sum, err)
			}
			if cp := f.checkpoint(t); cp == nil || !cp.Equal(sentinel) {
				t.Fatalf("%s moved the checkpoint to %v", c.name, cp)
			}
		}
	})
}

// ---- named Vietnam date cases against an independent oracle, both storage locations, both modes ----

type namedDateCase struct {
	name, from, to string
	valid          bool
}

// The same named cases are used by the TriggerJob admission test and the mounted JobDetail spec.
var namedDateCases = []namedDateCase{
	{"same day", "2026-10-02", "2026-10-02", true},
	{"month rollover", "2026-09-30", "2026-10-01", true},
	{"year rollover", "2025-12-31", "2026-01-01", true},
	{"leap day", "2028-02-29", "2028-02-29", true},
	{"from only", "2026-10-02", "", true},
	{"to only", "", "2026-10-02", true},
	{"reversed", "2026-10-03", "2026-10-02", false},
	{"not a leap year", "2100-02-29", "2100-02-29", false},
	{"impossible calendar date", "2026-02-30", "", false},
	{"time string", "2026-10-02T00:00:00", "", false},
	{"before the minimum", "0999-12-31", "", false},
	{"supplied maximum overflows", "", "9999-12-31", false},
}

// vnStartUTC is the UTC instant of 00:00 Vietnam time on the date (independent of the parser).
func vnStartUTC(t *testing.T, date string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatal(err)
	}
	return d.Add(-7 * time.Hour)
}

func TestVietnamDateCasesSelectExactlyTheOracleSet(t *testing.T) {
	for _, loc := range []string{"UTC", "Asia/Ho_Chi_Minh"} {
		loc := loc
		t.Run("loc="+strings.ReplaceAll(loc, "/", "_"), func(t *testing.T) {
			withIncLoc(t, loc)
			forModes(t, func(t *testing.T, batch bool) {
				f := setupIncFixture(t, batch, "qc_analysis")
				// instants just around each VN midnight used below, plus the 00:00-07:00 VN window
				var instants []time.Time
				for _, d := range []string{"2025-12-31", "2026-01-01", "2026-01-02", "2026-09-30", "2026-10-01", "2026-10-02", "2026-10-03", "2028-02-28", "2028-02-29", "2028-03-01"} {
					s := vnStartUTC(t, d)
					instants = append(instants, s.Add(-time.Second), s, s.Add(3*time.Hour), s.Add(24*time.Hour-time.Second))
				}
				ids := map[string]time.Time{}
				var all []string
				for i, in := range instants {
					id := f.addConv(t, f.tenantID, f.channelID, fmt.Sprintf("d%02d", i), []time.Time{in})
					ids[id] = in
					all = append(all, id)
				}
				an := NewAnalyzerWithProvider(&config.Config{}, &incProvider{})
				for _, c := range namedDateCases {
					p := &incProvider{}
					an.providerOverride = p
					before := f.count(t, "job_runs")
					run, err := an.RunJobFullWithParams(context.Background(), f.job(t), c.from, c.to, 0)
					if !c.valid {
						if err == nil || f.count(t, "job_runs") != before || p.callCount() != 0 {
							t.Fatalf("%s: invalid range accepted or left side effects: %v", c.name, err)
						}
						continue
					}
					var want []string
					for id, in := range ids {
						ok := true
						if c.from != "" && in.Before(vnStartUTC(t, c.from)) {
							ok = false
						}
						if c.to != "" && !in.Before(vnStartUTC(t, c.to).Add(24*time.Hour)) {
							ok = false
						}
						if ok {
							want = append(want, id)
						}
					}
					if got := f.evaluated(t, run, err); !eqIDs(got, sortedIDs(want...)) {
						t.Fatalf("%s: selected %d conversations %v, oracle %d", c.name, len(got), got, len(want))
					}
				}
				_ = all
			})
		})
	}
}

// ---- terminal state / checkpoint across modes, cap classes and outcomes ----

func TestExplicitCheckpointPreservedAcrossModesCapsAndOutcomes(t *testing.T) {
	type mode struct {
		name string
		run  func(an *Analyzer, ctx context.Context, job models.Job, cap int) (*models.JobRun, error)
		caps []int
	}
	modes := []mode{
		{"full", func(an *Analyzer, ctx context.Context, j models.Job, c int) (*models.JobRun, error) {
			return an.RunJobFullWithParams(ctx, j, "", "", c)
		}, []int{0, 1}},
		{"unanalyzed", func(an *Analyzer, ctx context.Context, j models.Job, c int) (*models.JobRun, error) {
			return an.RunJobUnanalyzed(ctx, j, c)
		}, []int{0, 1}},
		{"since last", func(an *Analyzer, ctx context.Context, j models.Job, c int) (*models.JobRun, error) {
			return an.RunJobSinceLast(ctx, j, c)
		}, []int{0, 1}},
		{"test run", func(an *Analyzer, ctx context.Context, j models.Job, c int) (*models.JobRun, error) {
			return an.RunJobWithLimit(ctx, j, c)
		}, []int{1, 50}},
	}
	outcomes := []struct {
		name, wantStatus string
	}{
		{"success", "success"}, {"zero work", "success"}, {"provider error", "error"},
		{"cancel at the last provider call", "partial"}, {"cancel with an empty selection", "partial"},
	}
	for _, m := range modes {
		for _, capN := range m.caps {
			for _, o := range outcomes {
				m, capN, o := m, capN, o
				t.Run(fmt.Sprintf("%s/cap=%d/%s", m.name, capN, o.name), func(t *testing.T) {
					f := setupIncFixture(t, false, "qc_analysis")
					f.addConv(t, f.tenantID, f.channelID, "ev", []time.Time{f.clock.Add(-3 * time.Hour)})
					f.addConv(t, f.tenantID, f.channelID, "n1", []time.Time{f.clock.Add(-2 * time.Hour)})
					f.addConv(t, f.tenantID, f.channelID, "n2", []time.Time{f.clock.Add(-time.Hour)})
					f.insertEval(t, f.jobID, f.tenantID, f.firstConvID(t, "ev"), "conversation_evaluation", "PASS")
					sentinel := f.setSentinel(t)
					p := &incProvider{}
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					switch o.name {
					case "zero work", "cancel with an empty selection":
						f.exec(t, "UPDATE jobs SET input_channel_ids = '[\"none\"]' WHERE id = ?", f.jobID)
					case "provider error":
						p.fail = true
					case "cancel at the last provider call":
						p.onCall = func(int) { cancel() }
					}
					if o.name == "cancel with an empty selection" {
						cancel()
					}
					run, err := m.run(NewAnalyzerWithProvider(&config.Config{}, p), ctx, f.job(t), capN)
					if err != nil || run.Status != o.wantStatus {
						t.Fatalf("status %s (err %v), want %s", run.Status, err, o.wantStatus)
					}
					if o.name == "provider error" || strings.HasPrefix(o.name, "cancel at") {
						if p.callCount() == 0 {
							t.Fatalf("the provider was never reached: the outcome was not exercised")
						}
					}
					job := f.job(t)
					if job.LastRunAt == nil || !job.LastRunAt.Equal(sentinel) {
						t.Fatalf("checkpoint moved to %v", job.LastRunAt)
					}
					if job.LastRunStatus != run.Status {
						t.Fatalf("job status %q, run status %q", job.LastRunStatus, run.Status)
					}
				})
			}
		}
	}
}

func (f *incFixture) firstConvID(t *testing.T, label string) string {
	t.Helper()
	var id string
	if err := db.DB.Raw("SELECT id FROM conversations WHERE tenant_id = ? AND id LIKE ?", f.tenantID, "conv-"+label+"-%").Scan(&id).Error; err != nil || id == "" {
		t.Fatalf("conversation %s: %v", label, err)
	}
	return id
}

func jsonUnmarshalSummary(runID string, out *map[string]int) error {
	var raw string
	if err := db.DB.Raw("SELECT CAST(summary AS CHAR) FROM job_runs WHERE id = ?", runID).Scan(&raw).Error; err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), out)
}
