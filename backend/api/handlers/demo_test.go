package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// demoResetFixtureIDs names every row seedDemoResetFixture creates. snapshotID
// is optional: leave it empty to seed a legacy job_result with no
// analysis_snapshot link (the pre-R3-E1-T1 shape of these tests), or set it to
// also seed an analysis_snapshot and link the job_result to it.
type demoResetFixtureIDs struct {
	tenantID   string
	channelID  string
	convID     string
	msgID      string
	jobID      string
	runID      string
	resultID   string
	snapshotID string
}

func seedDemoResetFixture(t *testing.T, ids demoResetFixtureIDs) {
	t.Helper()
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Demo Reset Test', ?, '{"is_demo_data":true}', NOW(), NOW())`,
		ids.tenantID, ids.tenantID)
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh demo', 'fake', X'00', true, '{}', NOW(), NOW())`,
		ids.channelID, ids.tenantID)
	exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, 'ext-demo', 'Khach', NOW(), 1, '{}', NOW(), NOW())`,
		ids.convID, ids.tenantID, ids.channelID)
	exec(`INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, 'm1', 'customer', 'Khach', 'Xin chao', 'text', '[]', NOW(), NOW())`,
		ids.msgID, ids.tenantID, ids.convID)
	exec(`INSERT INTO jobs (id, tenant_id, name, description, job_type, input_channel_ids, outputs, is_active, created_at, updated_at) VALUES (?, ?, 'QC demo', '', 'qc_analysis', ?, '[]', true, NOW(), NOW())`,
		ids.jobID, ids.tenantID, fmt.Sprintf(`["%s"]`, ids.channelID))
	exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, created_at) VALUES (?, ?, ?, NOW(), 'success', NOW())`,
		ids.runID, ids.jobID, ids.tenantID)

	if ids.snapshotID != "" {
		exec(`INSERT INTO analysis_snapshots (id, tenant_id, job_run_id, conversation_id, schema_version, digest, coverage, coverage_reasons, message_count, manifest, created_at) VALUES (?, ?, ?, ?, 'ccma.snapshot.v1', 'deadbeef', 'complete', '[]', 1, '{}', NOW())`,
			ids.snapshotID, ids.tenantID, ids.runID, ids.convID)
		exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, analysis_snapshot_id, created_at) VALUES (?, ?, ?, ?, 'qc_violation', 'NGHIEM_TRONG', 'rule', 'evidence', '{}', 1, ?, NOW())`,
			ids.resultID, ids.runID, ids.tenantID, ids.convID, ids.snapshotID)
		return
	}
	// Legacy shape: no snapshot, result.analysis_snapshot_id stays NULL. Kept as
	// an option (rather than removed) so this fixture still covers the
	// pre-snapshot/legacy result path this handler must also reset correctly;
	// the JobResult.AfterFind "legacy_unverified" derivation itself is already
	// covered by backend/engine's TestLegacyResultsAreMarkedUnverified.
	exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, created_at) VALUES (?, ?, ?, ?, 'qc_violation', 'NGHIEM_TRONG', 'rule', 'evidence', '{}', 1, NOW())`,
		ids.resultID, ids.runID, ids.tenantID, ids.convID)
}

func cleanupDemoResetFixture(tenantID, channelID string) {
	for _, table := range []string{"job_results", "analysis_snapshots", "messages", "conversations", "job_runs", "jobs", "activity_logs"} {
		db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenantID)
	}
	db.DB.Exec("DELETE FROM channels WHERE id = ?", channelID)
	db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
}

// TestResetDemoDataHappyPathClearsAllTenantData proves ResetDemoData still
// deletes every tenant row — including a snapshot-bound analysis_snapshot and
// its linked job_result — and clears the demo flag on the ordinary success
// path, after it was rewritten to run inside db.DB.Transaction.
func TestResetDemoDataHappyPathClearsAllTenantData(t *testing.T) {
	connectChannelsTestDB(t)
	suffix := pkg.NewUUID()[:8]
	ids := demoResetFixtureIDs{
		tenantID:   "reset-ok-" + suffix,
		channelID:  "ch-reset-ok-" + suffix,
		convID:     "conv-reset-ok-" + suffix,
		msgID:      pkg.NewUUID(),
		jobID:      "job-reset-ok-" + suffix,
		runID:      "run-reset-ok-" + suffix,
		resultID:   pkg.NewUUID(),
		snapshotID: pkg.NewUUID(),
	}

	seedDemoResetFixture(t, ids)
	t.Cleanup(func() { cleanupDemoResetFixture(ids.tenantID, ids.channelID) })

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", ids.tenantID)
	c.Request = httptest.NewRequest("POST", "/api/v1/demo/reset", nil)

	ResetDemoData(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("ResetDemoData status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var channelCount, convCount, msgCount, jobCount, runCount, resultCount, snapCount int64
	if err := db.DB.Model(&models.Channel{}).Where("tenant_id = ?", ids.tenantID).Count(&channelCount).Error; err != nil {
		t.Fatalf("count channels: %v", err)
	}
	if err := db.DB.Model(&models.Conversation{}).Where("tenant_id = ?", ids.tenantID).Count(&convCount).Error; err != nil {
		t.Fatalf("count conversations: %v", err)
	}
	if err := db.DB.Model(&models.Message{}).Where("tenant_id = ?", ids.tenantID).Count(&msgCount).Error; err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if err := db.DB.Model(&models.Job{}).Where("tenant_id = ?", ids.tenantID).Count(&jobCount).Error; err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if err := db.DB.Model(&models.JobRun{}).Where("tenant_id = ?", ids.tenantID).Count(&runCount).Error; err != nil {
		t.Fatalf("count job_runs: %v", err)
	}
	if err := db.DB.Model(&models.JobResult{}).Where("tenant_id = ?", ids.tenantID).Count(&resultCount).Error; err != nil {
		t.Fatalf("count job_results: %v", err)
	}
	if err := db.DB.Model(&models.AnalysisSnapshot{}).Where("tenant_id = ?", ids.tenantID).Count(&snapCount).Error; err != nil {
		t.Fatalf("count analysis_snapshots: %v", err)
	}
	if channelCount != 0 || convCount != 0 || msgCount != 0 || jobCount != 0 || runCount != 0 || resultCount != 0 || snapCount != 0 {
		t.Errorf("du lieu tenant chua duoc xoa het: channels=%d conversations=%d messages=%d jobs=%d job_runs=%d job_results=%d analysis_snapshots=%d",
			channelCount, convCount, msgCount, jobCount, runCount, resultCount, snapCount)
	}

	var tenant models.Tenant
	if err := db.DB.Where("id = ?", ids.tenantID).First(&tenant).Error; err != nil {
		t.Fatalf("reload tenant: %v", err)
	}
	if tenant.Settings != "{}" {
		t.Errorf("demo flag khong duoc xoa, settings = %q", tenant.Settings)
	}
}

// TestResetDemoDataFailureRollsBackEverything is the permanent regression test
// for R3-E1/R3-E1-T1: ResetDemoData used to ignore every DELETE/UPDATE
// statement's .Error inside a hand-rolled tx.Begin()/Commit(), so a mid-reset
// failure (forced here via a MySQL BEFORE DELETE trigger on job_results,
// matching the review's failure-injection reproduction) still committed,
// deleting parents while leaving an orphaned job_result and reporting
// success. This seeds a snapshot-bound job_result (analysis_snapshot_id set,
// per R3-E1-T1) rather than a legacy one, so the assertions below cover the
// snapshot-preservation acceptance the work order requires. After the fix,
// the same forced failure must roll back the whole reset: a non-2xx response,
// and every seeded row — including the message and the analysis_snapshot —
// plus the demo flag, left exactly as they were.
func TestResetDemoDataFailureRollsBackEverything(t *testing.T) {
	connectChannelsTestDB(t)
	suffix := pkg.NewUUID()[:8]
	failResultID := "FORCEFAILRESET" + suffix
	ids := demoResetFixtureIDs{
		tenantID:   "reset-fail-" + suffix,
		channelID:  "ch-reset-fail-" + suffix,
		convID:     "conv-reset-fail-" + suffix,
		msgID:      pkg.NewUUID(),
		jobID:      "job-reset-fail-" + suffix,
		runID:      "run-reset-fail-" + suffix,
		resultID:   failResultID,
		snapshotID: pkg.NewUUID(),
	}
	triggerName := "trg_ccma_test_resetfail_" + suffix

	seedDemoResetFixture(t, ids)

	// failResultID is a generated alphanumeric string with no quotes, safe to
	// inline: a trigger body cannot bind query parameters.
	if err := db.DB.Exec(fmt.Sprintf(`CREATE TRIGGER %s BEFORE DELETE ON job_results
FOR EACH ROW
BEGIN
	IF OLD.id = '%s' THEN
		SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced failure for reset-demo-data rollback test';
	END IF;
END`, triggerName, failResultID)).Error; err != nil {
		t.Fatalf("fixture trigger: %v", err)
	}

	t.Cleanup(func() {
		db.DB.Exec("DROP TRIGGER IF EXISTS " + triggerName)
		cleanupDemoResetFixture(ids.tenantID, ids.channelID)
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	gin.SetMode(gin.TestMode)
	c.Set("tenant_id", ids.tenantID)
	c.Request = httptest.NewRequest("POST", "/api/v1/demo/reset", nil)

	ResetDemoData(c)

	if rec.Code == http.StatusOK || rec.Code < 300 {
		t.Fatalf("ResetDemoData status = %d, muon non-2xx vi delete bi ep loi, body = %s", rec.Code, rec.Body.String())
	}

	var channelCount, convCount, msgCount, jobCount, runCount, resultCount, snapCount int64
	if err := db.DB.Model(&models.Channel{}).Where("id = ?", ids.channelID).Count(&channelCount).Error; err != nil {
		t.Fatalf("count channels: %v", err)
	}
	if err := db.DB.Model(&models.Conversation{}).Where("id = ?", ids.convID).Count(&convCount).Error; err != nil {
		t.Fatalf("count conversations: %v", err)
	}
	if err := db.DB.Model(&models.Message{}).Where("id = ?", ids.msgID).Count(&msgCount).Error; err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if err := db.DB.Model(&models.Job{}).Where("id = ?", ids.jobID).Count(&jobCount).Error; err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if err := db.DB.Model(&models.JobRun{}).Where("id = ?", ids.runID).Count(&runCount).Error; err != nil {
		t.Fatalf("count job_runs: %v", err)
	}
	if err := db.DB.Model(&models.JobResult{}).Where("id = ?", failResultID).Count(&resultCount).Error; err != nil {
		t.Fatalf("count job_results: %v", err)
	}
	if err := db.DB.Model(&models.AnalysisSnapshot{}).Where("id = ?", ids.snapshotID).Count(&snapCount).Error; err != nil {
		t.Fatalf("count analysis_snapshots: %v", err)
	}

	if channelCount != 1 {
		t.Errorf("channel khong con nguyen ven sau reset that bai: %d", channelCount)
	}
	if convCount != 1 {
		t.Errorf("conversation khong con nguyen ven sau reset that bai: %d", convCount)
	}
	if msgCount != 1 {
		t.Errorf("message khong con nguyen ven sau reset that bai: %d", msgCount)
	}
	if jobCount != 1 {
		t.Errorf("job khong con nguyen ven sau reset that bai: %d", jobCount)
	}
	if runCount != 1 {
		t.Errorf("job_run khong con nguyen ven sau reset that bai (orphan neu bi xoa): %d", runCount)
	}
	if resultCount != 1 {
		t.Errorf("job_result khong con nguyen ven sau reset that bai: %d", resultCount)
	}
	if snapCount != 1 {
		t.Errorf("analysis_snapshot khong con nguyen ven sau reset that bai (mat snapshot-preservation acceptance): %d", snapCount)
	}

	var result models.JobResult
	if err := db.DB.Where("id = ?", failResultID).First(&result).Error; err != nil {
		t.Fatalf("reload job_result: %v", err)
	}
	if result.AnalysisSnapshotID == nil || *result.AnalysisSnapshotID != ids.snapshotID {
		t.Errorf("job_result mat lien ket toi snapshot sau reset that bai: %+v", result.AnalysisSnapshotID)
	}

	var tenant models.Tenant
	if err := db.DB.Where("id = ?", ids.tenantID).First(&tenant).Error; err != nil {
		t.Fatalf("reload tenant: %v", err)
	}
	// MySQL's JSON column type re-serializes stored text (e.g. adds a space
	// after ':'), so compare the decoded flag rather than the raw string.
	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(tenant.Settings), &settings); err != nil {
		t.Fatalf("parse tenant settings %q: %v", tenant.Settings, err)
	}
	if isDemo, _ := settings["is_demo_data"].(bool); !isDemo {
		t.Errorf("demo flag bi xoa du reset that bai (khong atomic), settings = %q", tenant.Settings)
	}
}

// TestImportDemoDataStoresNoInventedConfidence covers CCMAI-RUNTIME-005 for the
// demo writer: demo rows are invented, not model output, so every seeded
// result must store confidence NULL with basis "unavailable" and serialize
// without a number. Before the tranche, demo rows stored 0.92/0.88/0.90 and a
// random 0.85-0.99 for tags.
func TestImportDemoDataStoresNoInventedConfidence(t *testing.T) {
	connectChannelsTestDB(t)
	tenantID := "demoimp-" + pkg.NewUUID()[:8]
	if err := db.DB.Exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Demo Import Test', ?, '{}', NOW(), NOW())`, tenantID, tenantID).Error; err != nil {
		t.Fatalf("fixture tenant: %v", err)
	}
	t.Cleanup(func() {
		for _, table := range []string{"job_results", "analysis_snapshots", "ai_usage_logs", "messages", "conversations", "job_runs", "jobs", "channels", "activity_logs"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenantID)
		}
		db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
	})

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenantID)
	c.Request = httptest.NewRequest("POST", "/api/v1/demo/import", nil)
	ImportDemoData(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("ImportDemoData status %d, body %s", rec.Code, rec.Body.String())
	}

	type row struct {
		ResultType      string
		Confidence      *float64
		ConfidenceBasis *string
	}
	var rows []row
	if err := db.DB.Raw("SELECT result_type, confidence, confidence_basis FROM job_results WHERE tenant_id = ?", tenantID).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	types := map[string]int{}
	for _, r := range rows {
		types[r.ResultType]++
		if r.Confidence != nil || r.ConfidenceBasis == nil || *r.ConfidenceBasis != models.ConfidenceBasisUnavailable {
			t.Fatalf("demo %s stored confidence=%v basis=%v, want NULL/unavailable", r.ResultType, r.Confidence, r.ConfidenceBasis)
		}
	}
	for _, want := range []string{"conversation_evaluation", "qc_violation", "classification_tag"} {
		if types[want] == 0 {
			t.Fatalf("demo seeded no %s rows (%v); test would be vacuous", want, types)
		}
	}

	var loaded []models.JobResult
	if err := db.DB.Where("tenant_id = ? AND result_type = ?", tenantID, "classification_tag").Find(&loaded).Error; err != nil {
		t.Fatal(err)
	}
	for _, r := range loaded {
		b, _ := json.Marshal(r)
		var m map[string]interface{}
		json.Unmarshal(b, &m)
		if m["confidence"] != nil || m["confidence_basis"] != models.ConfidenceBasisUnavailable {
			t.Fatalf("demo tag serialized confidence=%v basis=%v", m["confidence"], m["confidence_basis"])
		}
	}
}

// TestImportDemoDataPlacesNothingInTheFuture covers CCMAI-UX-001a (finding
// UX-01): with "today" pinned to 07:30 in Vietnam, same-day demo conversations
// used to be scheduled up to 20:00 that day (and, via UTC midnight truncation,
// even later), so the dashboard showed negative "minutes ago". Every demo
// conversation must now start between 08:00 and 20:00 Vietnam time and end,
// with its results, before the demo runs start and finish.
func TestImportDemoDataPlacesNothingInTheFuture(t *testing.T) {
	connectChannelsTestDB(t)
	fixedNow := time.Date(2026, 9, 28, 0, 30, 0, 0, time.UTC) // 07:30 at UTC+07:00
	prevNow := demoNow
	demoNow = func() time.Time { return fixedNow }
	t.Cleanup(func() { demoNow = prevNow })

	tenantID := "demotime-" + pkg.NewUUID()[:8]
	if err := db.DB.Exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Demo Time Test', ?, '{}', NOW(), NOW())`, tenantID, tenantID).Error; err != nil {
		t.Fatalf("fixture tenant: %v", err)
	}
	t.Cleanup(func() {
		for _, table := range []string{"job_results", "analysis_snapshots", "ai_usage_logs", "messages", "conversations", "job_runs", "jobs", "channels", "activity_logs"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenantID)
		}
		db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
	})

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenantID)
	c.Request = httptest.NewRequest("POST", "/api/v1/demo/import", nil)
	ImportDemoData(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("ImportDemoData status %d, body %s", rec.Code, rec.Body.String())
	}

	runsStart := fixedNow.Add(-2 * time.Hour)
	runsFinish := fixedNow.Add(-30 * time.Minute)

	var convs []models.Conversation
	if err := db.DB.Where("tenant_id = ?", tenantID).Find(&convs).Error; err != nil {
		t.Fatal(err)
	}
	if len(convs) == 0 {
		t.Fatal("demo seeded no conversations; test would be vacuous")
	}
	vn := time.FixedZone("ICT", 7*60*60)
	for _, cv := range convs {
		start := cv.CreatedAt.In(vn)
		if start.Hour() < 8 || start.Hour() > 20 {
			t.Fatalf("conversation %s starts at %s, outside 08:00-20:00 Vietnam time", cv.ID, start.Format(time.RFC3339))
		}
		if cv.LastMessageAt == nil || cv.LastMessageAt.After(runsStart) {
			t.Fatalf("conversation %s last message %v is not before the demo runs start %s", cv.ID, cv.LastMessageAt, runsStart)
		}
	}

	count := func(sql string, args ...interface{}) int64 {
		var n int64
		if err := db.DB.Raw(sql, args...).Scan(&n).Error; err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := count("SELECT COUNT(*) FROM messages WHERE tenant_id = ? AND sent_at > ?", tenantID, runsStart); n != 0 {
		t.Fatalf("%d demo messages are later than the demo runs start", n)
	}
	if n := count("SELECT COUNT(*) FROM job_results WHERE tenant_id = ? AND created_at > ?", tenantID, runsFinish); n != 0 {
		t.Fatalf("%d demo results are later than the demo runs finish", n)
	}
	if n := count("SELECT COUNT(*) FROM job_results WHERE tenant_id = ?", tenantID); n == 0 {
		t.Fatal("demo seeded no results; test would be vacuous")
	}
}
