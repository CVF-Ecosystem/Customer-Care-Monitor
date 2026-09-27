package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/pkg"
)

func seedDemoResetFixture(t *testing.T, tenantID, channelID, convID, jobID, runID, resultID string) {
	t.Helper()
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Demo Reset Test', ?, '{"is_demo_data":true}', NOW(), NOW())`,
		tenantID, tenantID)
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh demo', 'fake', X'00', true, '{}', NOW(), NOW())`,
		channelID, tenantID)
	exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, 'ext-demo', 'Khach', NOW(), 1, '{}', NOW(), NOW())`,
		convID, tenantID, channelID)
	exec(`INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, 'm1', 'customer', 'Khach', 'Xin chao', 'text', '[]', NOW(), NOW())`,
		pkg.NewUUID(), tenantID, convID)
	exec(`INSERT INTO jobs (id, tenant_id, name, description, job_type, input_channel_ids, outputs, is_active, created_at, updated_at) VALUES (?, ?, 'QC demo', '', 'qc_analysis', ?, '[]', true, NOW(), NOW())`,
		jobID, tenantID, fmt.Sprintf(`["%s"]`, channelID))
	exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, created_at) VALUES (?, ?, ?, NOW(), 'success', NOW())`,
		runID, jobID, tenantID)
	exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, created_at) VALUES (?, ?, ?, ?, 'qc_violation', 'NGHIEM_TRONG', 'rule', 'evidence', '{}', 1, NOW())`,
		resultID, runID, tenantID, convID)
}

func cleanupDemoResetFixture(tenantID, channelID string) {
	for _, table := range []string{"job_results", "analysis_snapshots", "messages", "conversations", "job_runs", "jobs", "activity_logs"} {
		db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenantID)
	}
	db.DB.Exec("DELETE FROM channels WHERE id = ?", channelID)
	db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
}

// TestResetDemoDataHappyPathClearsAllTenantData proves ResetDemoData still
// deletes every tenant row and clears the demo flag on the ordinary success
// path after it was rewritten to run inside db.DB.Transaction.
func TestResetDemoDataHappyPathClearsAllTenantData(t *testing.T) {
	connectChannelsTestDB(t)
	suffix := pkg.NewUUID()[:8]
	tenantID := "reset-ok-" + suffix
	channelID := "ch-reset-ok-" + suffix
	convID := "conv-reset-ok-" + suffix
	jobID := "job-reset-ok-" + suffix
	runID := "run-reset-ok-" + suffix
	resultID := pkg.NewUUID()

	seedDemoResetFixture(t, tenantID, channelID, convID, jobID, runID, resultID)
	t.Cleanup(func() { cleanupDemoResetFixture(tenantID, channelID) })

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenantID)
	c.Request = httptest.NewRequest("POST", "/api/v1/demo/reset", nil)

	ResetDemoData(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("ResetDemoData status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var channelCount, convCount, jobCount, runCount, resultCount int64
	db.DB.Model(&models.Channel{}).Where("tenant_id = ?", tenantID).Count(&channelCount)
	db.DB.Model(&models.Conversation{}).Where("tenant_id = ?", tenantID).Count(&convCount)
	db.DB.Model(&models.Job{}).Where("tenant_id = ?", tenantID).Count(&jobCount)
	db.DB.Model(&models.JobRun{}).Where("tenant_id = ?", tenantID).Count(&runCount)
	db.DB.Model(&models.JobResult{}).Where("tenant_id = ?", tenantID).Count(&resultCount)
	if channelCount != 0 || convCount != 0 || jobCount != 0 || runCount != 0 || resultCount != 0 {
		t.Errorf("du lieu tenant chua duoc xoa het: channels=%d conversations=%d jobs=%d job_runs=%d job_results=%d",
			channelCount, convCount, jobCount, runCount, resultCount)
	}

	var tenant models.Tenant
	if err := db.DB.Where("id = ?", tenantID).First(&tenant).Error; err != nil {
		t.Fatalf("reload tenant: %v", err)
	}
	if tenant.Settings != "{}" {
		t.Errorf("demo flag khong duoc xoa, settings = %q", tenant.Settings)
	}
}

// TestResetDemoDataFailureRollsBackEverything is the permanent regression test
// for R3-E1: ResetDemoData used to ignore every DELETE/UPDATE statement's
// .Error inside a hand-rolled tx.Begin()/Commit(), so a mid-reset failure
// (forced here via a MySQL BEFORE DELETE trigger on job_results, matching the
// review's failure-injection reproduction) still committed, deleting parents
// while leaving an orphaned job_result and reporting success. After the fix,
// the same forced failure must roll back the whole reset: a non-2xx response,
// and every row plus the demo flag left exactly as they were.
func TestResetDemoDataFailureRollsBackEverything(t *testing.T) {
	connectChannelsTestDB(t)
	suffix := pkg.NewUUID()[:8]
	tenantID := "reset-fail-" + suffix
	channelID := "ch-reset-fail-" + suffix
	convID := "conv-reset-fail-" + suffix
	jobID := "job-reset-fail-" + suffix
	runID := "run-reset-fail-" + suffix
	failResultID := "FORCEFAILRESET" + suffix
	triggerName := "trg_ccma_test_resetfail_" + suffix

	seedDemoResetFixture(t, tenantID, channelID, convID, jobID, runID, failResultID)

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
		cleanupDemoResetFixture(tenantID, channelID)
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	gin.SetMode(gin.TestMode)
	c.Set("tenant_id", tenantID)
	c.Request = httptest.NewRequest("POST", "/api/v1/demo/reset", nil)

	ResetDemoData(c)

	if rec.Code == http.StatusOK || rec.Code < 300 {
		t.Fatalf("ResetDemoData status = %d, muon non-2xx vi delete bi ep loi, body = %s", rec.Code, rec.Body.String())
	}

	var channelCount, convCount, jobCount, runCount, resultCount int64
	db.DB.Model(&models.Channel{}).Where("id = ?", channelID).Count(&channelCount)
	db.DB.Model(&models.Conversation{}).Where("id = ?", convID).Count(&convCount)
	db.DB.Model(&models.Job{}).Where("id = ?", jobID).Count(&jobCount)
	db.DB.Model(&models.JobRun{}).Where("id = ?", runID).Count(&runCount)
	db.DB.Model(&models.JobResult{}).Where("id = ?", failResultID).Count(&resultCount)

	if channelCount != 1 {
		t.Errorf("channel khong con nguyen ven sau reset that bai: %d", channelCount)
	}
	if convCount != 1 {
		t.Errorf("conversation khong con nguyen ven sau reset that bai: %d", convCount)
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

	var tenant models.Tenant
	if err := db.DB.Where("id = ?", tenantID).First(&tenant).Error; err != nil {
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
