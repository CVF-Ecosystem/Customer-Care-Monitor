package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-025 (F03): ordinary unlimited analysis selects by source version (the latest
// committed evaluation's snapshot of the same job), not by event time, result time or the
// wall-clock checkpoint; the checkpoint is the scan start and is written only with a checked
// terminal run write. A recording synthetic provider runs through the real Analyzer on a
// disposable MySQL; no real AI or channel is called. Every scenario runs in single and batch
// mode.

// ---- recording provider ----

type incCall struct {
	transcripts []string
}

type incProvider struct {
	mu      sync.Mutex
	calls   []incCall
	jobType string
	verdict string // QC verdict, default FAIL
	fail    bool   // return a provider error
	badRef  bool   // cite text that is not in the transcript (save fails)
	onCall  func(n int)
}

func (p *incProvider) result(transcript string) json.RawMessage {
	src := firstTranscriptRef(transcript)
	quote := src.content
	if p.badRef {
		quote = "khong co trong tin nhan"
	}
	ref := map[string]interface{}{"message_id": src.messageID, "quote": quote}
	if p.jobType == "classification" {
		b, _ := json.Marshal(map[string]interface{}{
			"summary": "Khach hoi don hang.",
			"tags": []map[string]interface{}{{"rule_name": "Hoi don", "confidence": 0.8, "evidence": quote,
				"evidence_refs": []interface{}{ref}, "explanation": "Khach hoi."}},
		})
		return b
	}
	if p.verdict == "PASS" {
		b, _ := json.Marshal(map[string]interface{}{"verdict": "PASS", "score": 95, "review": "Tot.", "summary": "Dat.", "violations": []interface{}{}})
		return b
	}
	return qcFailWithRef(ref)
}

func (p *incProvider) record(transcripts []string) int {
	p.mu.Lock()
	p.calls = append(p.calls, incCall{transcripts: transcripts})
	n := len(p.calls)
	hook := p.onCall
	p.mu.Unlock()
	if hook != nil {
		hook(n)
	}
	return n
}

func (p *incProvider) AnalyzeChat(_ context.Context, _ string, transcript string) (ai.AIResponse, error) {
	p.record([]string{transcript})
	if p.fail {
		return ai.AIResponse{}, errors.New("synthetic provider failure")
	}
	return ai.AIResponse{Content: string(p.result(transcript)), Model: "test-double", Provider: "test-double"}, nil
}

func (p *incProvider) AnalyzeChatBatch(_ context.Context, _ string, items []ai.BatchItem) (ai.AIResponse, error) {
	ts := make([]string, len(items))
	for i, it := range items {
		ts[i] = it.Transcript
	}
	p.record(ts)
	if p.fail {
		return ai.AIResponse{}, errors.New("synthetic provider failure")
	}
	out := make([]json.RawMessage, len(items))
	for i, it := range items {
		out[i] = p.result(it.Transcript)
	}
	b, _ := json.Marshal(out)
	return ai.AIResponse{Content: string(b), Model: "test-double", Provider: "test-double"}, nil
}

// analyzed returns the conversation IDs seen by the provider (from message IDs in transcripts).
func (p *incProvider) transcripts() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []string
	for _, c := range p.calls {
		out = append(out, c.transcripts...)
	}
	return out
}

func (p *incProvider) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.calls)
}

func (p *incProvider) reset() {
	p.mu.Lock()
	p.calls = nil
	p.mu.Unlock()
}

func transcriptMessageIDs(transcript string) []string {
	var ids []string
	for _, line := range strings.Split(transcript, "\n") {
		if m := transcriptLine.FindStringSubmatch(line); m != nil {
			ids = append(ids, m[1])
		}
	}
	sort.Strings(ids)
	return ids
}

// ---- fixture ----

type incFixture struct {
	tenantID, otherTenantID   string
	channelID, otherChannelID string
	otherTenantChannelID      string
	jobID                     string
	batch                     bool
	clock                     time.Time
}

func setupIncFixture(t *testing.T, batch bool, jobType string) *incFixture {
	t.Helper()
	// db.Connect never closes its predecessor; these tests create many fixtures, so close the
	// previous pool first or later tests in the package hit "Too many connections" and skip.
	db.Close()
	connectIncDB(t)
	s := pkg.NewUUID()[:8]
	f := &incFixture{
		tenantID: "inc-" + s, otherTenantID: "inc-o-" + s,
		channelID: "ch-inc-" + s, otherChannelID: "ch-inc-x-" + s, otherTenantChannelID: "ch-inc-ot-" + s,
		jobID: "job-inc-" + s, batch: batch,
	}
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	for _, tn := range []string{f.tenantID, f.otherTenantID} {
		exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Inc', ?, '{}', NOW(), NOW())`, tn, tn)
	}
	for ch, tn := range map[string]string{f.channelID: f.tenantID, f.otherChannelID: f.tenantID, f.otherTenantChannelID: f.otherTenantID} {
		exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh', ?, X'00', true, '{}', NOW(), NOW())`, ch, tn, "ext-"+ch)
	}
	chJSON, _ := json.Marshal([]string{f.channelID})
	exec(`INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, schedule_cron, is_active, outputs, output_schedule, created_at, updated_at) VALUES (?, ?, 'Inc', ?, ?, 'Phan hoi dung han.', '[{"name":"Hoi don"}]', 'manual', '', true, '[]', 'none', NOW(), NOW())`,
		f.jobID, f.tenantID, jobType, string(chJSON))
	mode := "false"
	if batch {
		mode = "true"
	}
	exec(`INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at) VALUES (?, ?, 'ai_batch_mode', ?, NOW(), NOW())`, pkg.NewUUID(), f.tenantID, mode)
	exec(`INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at) VALUES (?, ?, 'ai_batch_size', '30', NOW(), NOW())`, pkg.NewUUID(), f.tenantID)

	// Deterministic analyzer clock with a fractional second, advanced explicitly per run.
	f.clock = time.Now().UTC().Truncate(time.Second).Add(-10 * time.Minute).Add(437 * time.Millisecond)
	origNow, origDelay, origNotify := analyzerNow, ordinaryFinalizeRetryDelay, sendJobNotifications
	analyzerNow = func() time.Time { return f.clock }
	ordinaryFinalizeRetryDelay = 10 * time.Millisecond
	sendJobNotifications = func(context.Context, models.Job, models.JobRun) error {
		t.Fatalf("notifications are not expected unless a test installs its own sender")
		return nil
	}
	t.Cleanup(func() {
		analyzerNow, ordinaryFinalizeRetryDelay, sendJobNotifications = origNow, origDelay, origNotify
		for _, tn := range []string{f.tenantID, f.otherTenantID} {
			for _, table := range []string{"job_results", "analysis_snapshots", "job_runs", "ai_usage_logs", "messages", "conversations", "app_settings", "activity_logs", "jobs", "channels"} {
				db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tn)
			}
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tn)
		}
	})
	return f
}

func (f *incFixture) exec(t *testing.T, sql string, args ...interface{}) {
	t.Helper()
	if err := db.DB.Exec(sql, args...).Error; err != nil {
		t.Fatalf("exec: %v", err)
	}
}

// addConv inserts a conversation with messages at the given times (oldest first); the
// conversation's last_message_at is the last message time unless lastAt overrides it.
func (f *incFixture) addConv(t *testing.T, tenantID, channelID, label string, msgTimes []time.Time) string {
	t.Helper()
	id := "conv-" + label + "-" + pkg.NewUUID()[:6]
	var last interface{}
	if len(msgTimes) > 0 {
		last = msgTimes[len(msgTimes)-1].UTC()
	}
	f.exec(t, `INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, 'Khach', ?, ?, '{}', NOW(), NOW())`,
		id, tenantID, channelID, "ext-"+id, last, len(msgTimes))
	for i, at := range msgTimes {
		f.addMsg(t, tenantID, id, fmt.Sprintf("%s-m%d", label, i), at)
	}
	return id
}

func (f *incFixture) addMsg(t *testing.T, tenantID, convID, label string, at time.Time) string {
	t.Helper()
	id := pkg.NewUUID()
	f.exec(t, `INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, ?, 'customer', 'Khach', ?, 'text', '[]', ?, NOW())`,
		id, tenantID, convID, "x-"+label, "Noi dung "+label, at.UTC())
	return id
}

func (f *incFixture) job(t *testing.T) models.Job {
	t.Helper()
	var job models.Job
	if err := db.DB.First(&job, "id = ?", f.jobID).Error; err != nil {
		t.Fatal(err)
	}
	return job
}

// run executes one ordinary incremental run at the fixture clock, then advances the clock.
func (f *incFixture) run(t *testing.T, p ai.AIProvider) (*models.JobRun, error) {
	t.Helper()
	run, err := NewAnalyzer(&config.Config{}).RunJobWithProvider(context.Background(), f.job(t), 0, p)
	f.clock = f.clock.Add(time.Hour)
	return run, err
}

func (f *incFixture) mustRun(t *testing.T, p ai.AIProvider) *models.JobRun {
	t.Helper()
	run, err := f.run(t, p)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return run
}

// evaluatedIn returns the conversation IDs with a conversation_evaluation in the run.
func evaluatedIn(t *testing.T, runID string) []string {
	t.Helper()
	var ids []string
	if err := db.DB.Model(&models.JobResult{}).Where("job_run_id = ? AND result_type = 'conversation_evaluation'", runID).
		Order("conversation_id").Pluck("conversation_id", &ids).Error; err != nil {
		t.Fatal(err)
	}
	return ids
}

func (f *incFixture) count(t *testing.T, table string) int64 {
	t.Helper()
	var n int64
	if err := db.DB.Table(table).Where("tenant_id = ?", f.tenantID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *incFixture) checkpoint(t *testing.T) *time.Time {
	t.Helper()
	return f.job(t).LastRunAt
}

func sortedIDs(ids ...string) []string {
	out := append([]string(nil), ids...)
	sort.Strings(out)
	return out
}

func eqIDs(a, b []string) bool { return strings.Join(a, ",") == strings.Join(b, ",") }

func forModes(t *testing.T, body func(t *testing.T, batch bool)) {
	for _, batch := range []bool{false, true} {
		batch := batch
		t.Run(fmt.Sprintf("batch=%v", batch), func(t *testing.T) { body(t, batch) })
	}
}

// ---- tests ----

// CCMAI-RUNTIME-027: the mode is chosen by the entry point, never inferred from the cap or dates.
func TestOrdinaryPlanIsOnlyTheUncappedOrdinaryMode(t *testing.T) {
	if p := ordinaryPlan(); p.explicit() || p.limit != 0 {
		t.Fatalf("ordinary plan %+v", p)
	}
	for _, m := range []analysisMode{modeTestRun, modeFull, modeUnanalyzed, modeSinceLast} {
		if !(runPlan{mode: m}).explicit() {
			t.Errorf("%s must be explicit", m)
		}
	}
	if _, err := newPlan(modeOrdinary, 3, pkg.BusinessRange{}); err == nil {
		t.Error("ordinary with a cap must be rejected")
	}
}

// A conversation inserted after candidate selection, whose last message is before the first
// run's finish, is analyzed by the next run; the third run analyzes nothing. The checkpoint is
// exactly the scan start truncated to the second.
func TestOrdinaryRunAnalyzesConversationInsertedDuringPreviousRun(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "qc_analysis")
		a := f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-2 * time.Hour)})
		var lateNoMore, lateMore string
		p := &incProvider{}
		p.onCall = func(n int) {
			if n == 1 { // during run 1, after selection and preparation
				lateNoMore = f.addConv(t, f.tenantID, f.channelID, "l1", []time.Time{f.clock.Add(-30 * time.Minute)})
				lateMore = f.addConv(t, f.tenantID, f.channelID, "l2", []time.Time{f.clock.Add(-20 * time.Minute)})
			}
		}
		scan1 := f.clock
		run1 := f.mustRun(t, p)
		if run1.Status != "success" || !eqIDs(evaluatedIn(t, run1.ID), []string{a}) {
			t.Fatalf("run 1: status %s, evaluated %v", run1.Status, evaluatedIn(t, run1.ID))
		}
		if cp := f.checkpoint(t); cp == nil || !cp.Equal(scan1.Truncate(time.Second)) {
			t.Fatalf("checkpoint %v, want scan start %v", cp, scan1.Truncate(time.Second))
		}
		// A later message after run 1 finished for one of them.
		f.addMsg(t, f.tenantID, lateMore, "l2-after", f.clock.Add(-time.Minute))
		f.exec(t, "UPDATE conversations SET last_message_at = ? WHERE id = ?", f.clock.Add(-time.Minute).UTC(), lateMore)

		p.reset()
		p.onCall = nil
		run2 := f.mustRun(t, p)
		if got := evaluatedIn(t, run2.ID); !eqIDs(got, sortedIDs(lateNoMore, lateMore)) {
			t.Fatalf("run 2 evaluated %v, want both late conversations %v", got, sortedIDs(lateNoMore, lateMore))
		}
		snaps, results := f.count(t, "analysis_snapshots"), f.count(t, "job_results")
		p.reset()
		run3 := f.mustRun(t, p)
		if p.callCount() != 0 || len(evaluatedIn(t, run3.ID)) != 0 || f.count(t, "analysis_snapshots") != snaps || f.count(t, "job_results") != results {
			t.Fatalf("run 3 re-analyzed unchanged source: %d calls", p.callCount())
		}
		if run3.Status != "success" {
			t.Fatalf("run 3 status %s", run3.Status)
		}
	})
}

// An empty first run still records its scan start; a conversation ingested later with an old
// timestamp (before that checkpoint and before the initial 24h window) is analyzed next.
func TestOrdinaryRunAnalyzesLateIngestedOldConversation(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "qc_analysis")
		p := &incProvider{}
		scan1 := f.clock
		run1 := f.mustRun(t, p)
		if run1.Status != "success" || p.callCount() != 0 {
			t.Fatalf("empty run: %s, %d calls", run1.Status, p.callCount())
		}
		if cp := f.checkpoint(t); cp == nil || !cp.Equal(scan1.Truncate(time.Second)) {
			t.Fatalf("zero-work checkpoint %v, want %v", cp, scan1.Truncate(time.Second))
		}
		old := f.addConv(t, f.tenantID, f.channelID, "old", []time.Time{scan1.Add(-72 * time.Hour), scan1.Add(-71 * time.Hour)})
		run2 := f.mustRun(t, p)
		if got := evaluatedIn(t, run2.ID); !eqIDs(got, []string{old}) {
			t.Fatalf("run 2 evaluated %v, want the late old conversation", got)
		}
		p.reset()
		f.mustRun(t, p)
		if p.callCount() != 0 {
			t.Fatalf("unchanged replay called the provider %d times", p.callCount())
		}
	})
}

// For an analyzed conversation, a late message dated before the checkpoint (last_message_at
// unchanged) and an in-place edit of a selected message are each analyzed once, with the
// change in the transcript and the saved snapshot, then skipped.
func TestOrdinaryRunAnalyzesLateMessageAndEditWithoutNewerTimestamp(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "qc_analysis")
		t0 := f.clock.Add(-5 * time.Hour)
		a := f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{t0, t0.Add(10 * time.Minute)})
		p := &incProvider{}
		f.mustRun(t, p)

		// Late message between the two existing ones; last_message_at unchanged.
		lateID := f.addMsg(t, f.tenantID, a, "late", t0.Add(5*time.Minute))
		p.reset()
		run2 := f.mustRun(t, p)
		ts := p.transcripts()
		if len(ts) != 1 || !strings.Contains(ts[0], "Noi dung late") {
			t.Fatalf("run 2 transcripts %q", ts)
		}
		f.assertSavedSnapshotMatches(t, run2.ID, a, ts[0], 3)
		if !hasID(transcriptMessageIDs(ts[0]), lateID) {
			t.Fatal("late message id missing from transcript")
		}
		p.reset()
		f.mustRun(t, p)
		if p.callCount() != 0 {
			t.Fatalf("replay after late message: %d calls", p.callCount())
		}

		// Edit an existing message without changing any timestamp.
		f.exec(t, "UPDATE messages SET content = 'Noi dung da sua' WHERE conversation_id = ? AND external_message_id = 'x-a-m0'", a)
		p.reset()
		run4 := f.mustRun(t, p)
		ts = p.transcripts()
		if len(ts) != 1 || !strings.Contains(ts[0], "Noi dung da sua") {
			t.Fatalf("edit not analyzed: %q", ts)
		}
		f.assertSavedSnapshotMatches(t, run4.ID, a, ts[0], 3)
		p.reset()
		f.mustRun(t, p)
		if p.callCount() != 0 {
			t.Fatalf("replay after edit: %d calls", p.callCount())
		}
	})
}

func hasID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// assertSavedSnapshotMatches checks that the snapshot saved for the run is the source the
// provider saw: same message IDs and count, and a valid digest.
func (f *incFixture) assertSavedSnapshotMatches(t *testing.T, runID, convID, transcript string, wantMessages int) {
	t.Helper()
	var snap models.AnalysisSnapshot
	if err := db.DB.Where("job_run_id = ? AND conversation_id = ?", runID, convID).First(&snap).Error; err != nil {
		t.Fatalf("no snapshot saved: %v", err)
	}
	if !VerifySnapshotProvenance(snap, f.tenantID, convID, runID) || snap.MessageCount != wantMessages {
		t.Fatalf("snapshot provenance/count: %+v", snap)
	}
	var manifest snapshotManifest
	if err := json.Unmarshal([]byte(snap.Manifest), &manifest); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range manifest.Messages {
		ids = append(ids, m.MessageID)
	}
	sort.Strings(ids)
	if !eqIDs(ids, transcriptMessageIDs(transcript)) {
		t.Fatalf("saved snapshot messages %v differ from the submitted transcript %v", ids, transcriptMessageIDs(transcript))
	}
}

// A message inserted and an edit made after the snapshot was prepared (during the provider call)
// are not attributed to that result: the saved snapshot is the submitted one, and the next run
// analyzes the conversation again.
func TestOrdinaryRunDetectsChangeAfterPreparation(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "qc_analysis")
		t0 := f.clock.Add(-3 * time.Hour)
		a := f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{t0, t0.Add(time.Minute)})
		p := &incProvider{}
		p.onCall = func(n int) {
			if n == 1 {
				f.addMsg(t, f.tenantID, a, "during", t0.Add(30*time.Second))
			}
		}
		run1 := f.mustRun(t, p)
		ts := p.transcripts()
		if len(ts) != 1 || strings.Contains(ts[0], "Noi dung during") {
			t.Fatalf("run 1 transcript %q", ts)
		}
		f.assertSavedSnapshotMatches(t, run1.ID, a, ts[0], 2)

		p.reset()
		p.onCall = nil
		run2 := f.mustRun(t, p)
		ts = p.transcripts()
		if len(ts) != 1 || !strings.Contains(ts[0], "Noi dung during") {
			t.Fatalf("run 2 did not re-analyze the changed source: %q", ts)
		}
		f.assertSavedSnapshotMatches(t, run2.ID, a, ts[0], 3)
		p.reset()
		f.mustRun(t, p)
		if p.callCount() != 0 {
			t.Fatalf("run 3: %d calls", p.callCount())
		}
	})
}

// Receipts are bound to this job and tenant: another job's evaluation, a newer result timestamp,
// a legacy evaluation without a snapshot and an orphan snapshot without an evaluation never hide
// changed source; other tenants/channels are never candidates. A PASS evaluation without any
// finding is a valid receipt.
func TestOrdinaryRunReceiptsAreJobBound(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "qc_analysis")
		base := f.clock.Add(-4 * time.Hour)
		otherJobConv := f.addConv(t, f.tenantID, f.channelID, "oj", []time.Time{base})
		legacyConv := f.addConv(t, f.tenantID, f.channelID, "legacy", []time.Time{base})
		orphanConv := f.addConv(t, f.tenantID, f.channelID, "orphan", []time.Time{base})
		f.addConv(t, f.tenantID, f.otherChannelID, "otherch", []time.Time{base})
		f.addConv(t, f.otherTenantID, f.otherTenantChannelID, "othert", []time.Time{base})

		// Another job of the same tenant already evaluated otherJobConv with a far-future time.
		otherJob := "job-inc-other-" + pkg.NewUUID()[:6]
		f.exec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, output_schedule, created_at, updated_at) VALUES (?, ?, 'Other', 'qc_analysis', '[]', '', '[]', 'manual', true, '[]', 'none', NOW(), NOW())`, otherJob, f.tenantID)
		otherRun := pkg.NewUUID()
		f.exec(t, `INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'success', '{}', NOW())`, otherRun, otherJob, f.tenantID)
		f.exec(t, `INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, detail, created_at) VALUES (?, ?, ?, ?, 'conversation_evaluation', 'PASS', '{}', ?)`,
			pkg.NewUUID(), otherRun, f.tenantID, otherJobConv, time.Now().Add(24*time.Hour).UTC())
		// A legacy evaluation of this job without a snapshot, and an orphan snapshot of this job's
		// run with the current digest but no evaluation.
		thisRun := pkg.NewUUID()
		f.exec(t, `INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'success', '{}', NOW())`, thisRun, f.jobID, f.tenantID)
		f.exec(t, `INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, detail, created_at) VALUES (?, ?, ?, ?, 'conversation_evaluation', 'FAIL', '{}', ?)`,
			pkg.NewUUID(), thisRun, f.tenantID, legacyConv, time.Now().Add(-24*time.Hour).UTC())
		var orphan models.Conversation
		db.DB.First(&orphan, "id = ?", orphanConv)
		snap, err := loadConversationSnapshot(orphan, time.Time{})
		if err != nil {
			t.Fatal(err)
		}
		row, _ := snap.record(thisRun)
		if err := db.DB.Create(&row).Error; err != nil {
			t.Fatal(err)
		}

		p := &incProvider{verdict: "PASS"}
		run := f.mustRun(t, p)
		if got := evaluatedIn(t, run.ID); !eqIDs(got, sortedIDs(otherJobConv, legacyConv, orphanConv)) {
			t.Fatalf("evaluated %v, want exactly this tenant/channel's three conversations", got)
		}
		// PASS evaluations without findings are receipts: nothing to do next time.
		p.reset()
		f.mustRun(t, p)
		if p.callCount() != 0 {
			t.Fatalf("PASS receipts not honoured: %d calls", p.callCount())
		}
	})
}

// A linked snapshot that is missing, tampered or bound to another conversation is a
// verification error: the run is not a success and the checkpoint does not move.
func TestOrdinaryRunUnverifiableReceiptFailsClosed(t *testing.T) {
	for _, damage := range []string{"tampered", "missing", "cross-bound"} {
		damage := damage
		t.Run(damage, func(t *testing.T) {
			f := setupIncFixture(t, false, "qc_analysis")
			a := f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-2 * time.Hour)})
			b := f.addConv(t, f.tenantID, f.channelID, "b", []time.Time{f.clock.Add(-2 * time.Hour)})
			p := &incProvider{}
			f.mustRun(t, p)
			before := f.checkpoint(t)
			var eval models.JobResult
			if err := db.DB.Where("conversation_id = ? AND result_type = 'conversation_evaluation'", a).First(&eval).Error; err != nil {
				t.Fatal(err)
			}
			switch damage {
			case "tampered":
				f.exec(t, "UPDATE analysis_snapshots SET manifest = CONCAT(manifest, ' ') WHERE id = ?", *eval.AnalysisSnapshotID)
			case "missing":
				f.exec(t, "DELETE FROM analysis_snapshots WHERE id = ?", *eval.AnalysisSnapshotID)
			case "cross-bound":
				var other models.JobResult
				db.DB.Where("conversation_id = ? AND result_type = 'conversation_evaluation'", b).First(&other)
				f.exec(t, "UPDATE job_results SET analysis_snapshot_id = ? WHERE id = ?", *other.AnalysisSnapshotID, eval.ID)
			}
			p.reset()
			run, err := f.run(t, p)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if run.Status == "success" || summaryInt(t, run, "conversations_errors") != 1 {
				t.Fatalf("unverifiable receipt: status %s summary %s", run.Status, run.Summary)
			}
			if cp := f.checkpoint(t); cp == nil || before == nil || !cp.Equal(*before) {
				t.Fatalf("checkpoint moved: %v -> %v", before, cp)
			}
			if p.callCount() != 0 {
				t.Fatalf("an unverifiable or unchanged conversation was sent: %d calls", p.callCount())
			}
		})
	}
}

// Classification uses the same source-version decision.
func TestOrdinaryRunClassificationSharesSelection(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "classification")
		t0 := f.clock.Add(-3 * time.Hour)
		a := f.addConv(t, f.tenantID, f.channelID, "c", []time.Time{t0})
		p := &incProvider{jobType: "classification"}
		run1 := f.mustRun(t, p)
		if !eqIDs(evaluatedIn(t, run1.ID), []string{a}) {
			t.Fatalf("classification run 1 evaluated %v", evaluatedIn(t, run1.ID))
		}
		p.reset()
		f.mustRun(t, p)
		if p.callCount() != 0 {
			t.Fatalf("classification replay: %d calls", p.callCount())
		}
		f.addMsg(t, f.tenantID, a, "late", t0.Add(-time.Minute))
		run3 := f.mustRun(t, p)
		if !eqIDs(evaluatedIn(t, run3.ID), []string{a}) {
			t.Fatalf("classification late message not analyzed")
		}
	})
}

// Provider failure, result-save failure and cancellation keep the checkpoint and are not success.
func TestOrdinaryRunFailuresKeepCheckpoint(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		for _, mode := range []string{"provider", "save", "cancel"} {
			f := setupIncFixture(t, batch, "qc_analysis")
			f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-2 * time.Hour)})
			f.mustRun(t, &incProvider{verdict: "PASS"}) // establishes a checkpoint
			before := f.checkpoint(t)
			f.addConv(t, f.tenantID, f.channelID, "b", []time.Time{f.clock.Add(-90 * time.Minute)})
			p := &incProvider{fail: mode == "provider", badRef: mode == "save"}
			ctx, cancel := context.WithCancel(context.Background())
			if mode == "cancel" {
				cancel()
			}
			run, err := NewAnalyzer(&config.Config{}).RunJobWithProvider(ctx, f.job(t), 0, p)
			cancel()
			f.clock = f.clock.Add(time.Hour)
			if err != nil {
				t.Fatalf("%s: %v", mode, err)
			}
			if run.Status == "success" {
				t.Fatalf("%s: status success", mode)
			}
			if cp := f.checkpoint(t); cp == nil || !cp.Equal(*before) {
				t.Fatalf("%s: checkpoint moved %v -> %v", mode, before, cp)
			}
		}
	})
}

// A failed or missing terminal write: the run reports an error, the checkpoint stays, the run row
// is not left successful and no notification is sent; the positive control notifies once.
func TestOrdinaryRunTerminalWriteFailure(t *testing.T) {
	for _, mode := range []string{"job-write-error", "job-row-missing", "control"} {
		mode := mode
		t.Run(mode, func(t *testing.T) {
			f := setupIncFixture(t, false, "qc_analysis")
			f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-2 * time.Hour)})
			f.exec(t, "UPDATE jobs SET output_schedule = 'instant', last_run_at = ? WHERE id = ?", f.clock.Add(-48*time.Hour).UTC().Truncate(time.Second), f.jobID)
			before := f.checkpoint(t)
			notified := 0
			sendJobNotifications = func(context.Context, models.Job, models.JobRun) error { notified++; return nil }
			p := &incProvider{}
			switch mode {
			case "job-write-error":
				trigger := "r025_job_" + f.jobID[len(f.jobID)-6:]
				f.exec(t, fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE ON jobs FOR EACH ROW BEGIN IF NEW.id = '%s' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'synthetic terminal write failure'; END IF; END", trigger, f.jobID))
				t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trigger) })
			case "job-row-missing":
				// The job leaves this tenant while the run works (a delete is blocked by the
				// job_runs foreign key): the tenant-scoped terminal write finds no target.
				p.onCall = func(int) { f.exec(t, "UPDATE jobs SET tenant_id = ? WHERE id = ?", f.otherTenantID, f.jobID) }
			}
			job := f.job(t)
			run, err := NewAnalyzer(&config.Config{}).RunJobWithProvider(context.Background(), job, 0, p)
			if mode == "control" {
				if err != nil || run.Status != "success" || notified != 1 {
					t.Fatalf("control: err %v status %s notified %d", err, run.Status, notified)
				}
				return
			}
			if err == nil || run.Status != "error" || notified != 0 {
				t.Fatalf("%s: err %v status %s notified %d", mode, err, run.Status, notified)
			}
			var stored models.JobRun
			if e := db.DB.First(&stored, "id = ?", run.ID).Error; e != nil || stored.Status == "success" || stored.Status == "running" {
				t.Fatalf("%s: stored run %q (%v)", mode, stored.Status, e)
			}
			if mode == "job-write-error" {
				if cp := f.checkpoint(t); cp == nil || !cp.Equal(*before) {
					t.Fatalf("checkpoint moved %v -> %v", before, cp)
				}
			}
		})
	}
}

// The other modes keep their previous contracts (F05 is not changed here).
func TestNonOrdinaryModesKeepTheirContracts(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	a := f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-2 * time.Hour)})
	p := &incProvider{}
	f.mustRun(t, p)
	cp := f.checkpoint(t)
	an := NewAnalyzer(&config.Config{})
	an.providerOverride = p

	// Positive limit: excludes already analyzed conversations and never moves the checkpoint.
	p.reset()
	if _, err := an.RunJobWithProvider(context.Background(), f.job(t), 5, p); err != nil {
		t.Fatal(err)
	}
	if p.callCount() != 0 || !f.checkpoint(t).Equal(*cp) {
		t.Fatalf("limited run: %d calls, checkpoint %v", p.callCount(), f.checkpoint(t))
	}
	// Unanalyzed-only: an analyzed conversation with a new message is still excluded (F05).
	f.addMsg(t, f.tenantID, a, "new", f.clock.Add(-time.Minute))
	p.reset()
	if _, err := an.RunJobUnanalyzed(context.Background(), f.job(t), 0); err != nil {
		t.Fatal(err)
	}
	if p.callCount() != 0 {
		t.Fatalf("unanalyzed mode re-analyzed an analyzed conversation: %d calls", p.callCount())
	}
	// Full rerun repeats analysis of unchanged source on request.
	p.reset()
	if _, err := an.RunJobFull(context.Background(), f.job(t)); err != nil {
		t.Fatal(err)
	}
	if p.callCount() == 0 {
		t.Fatal("full rerun did not analyze")
	}
	// A date range outside the data selects nothing.
	p.reset()
	if _, err := an.RunJobFullWithParams(context.Background(), f.job(t), "2000-01-01", "2000-01-02", 0); err != nil {
		t.Fatal(err)
	}
	if p.callCount() != 0 {
		t.Fatalf("date range outside the data analyzed %d", p.callCount())
	}
}
