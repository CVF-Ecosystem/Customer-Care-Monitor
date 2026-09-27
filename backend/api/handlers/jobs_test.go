package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/pkg"
)

// TestDeleteJobRemovesRunsAndEvidence là regression test cho R2-RR3: DeleteJob
// được viết lại thành transaction có khoá job_run trước (cùng khuôn mẫu
// DeleteChannel/PurgeChannelConversations) để đóng writer/deletion race; test
// này xác nhận luồng bình thường vẫn xoá đúng run/result/snapshot/job.
func TestDeleteJobRemovesRunsAndEvidence(t *testing.T) {
	connectChannelsTestDB(t)
	suffix := pkg.NewUUID()[:8]
	tenantID := "jobdel-" + suffix
	channelID := "ch-jobdel-" + suffix
	convID := "conv-jobdel-" + suffix
	jobID := "job-jobdel-" + suffix
	runID := "run-jobdel-" + suffix
	snapID := pkg.NewUUID()
	resultID := pkg.NewUUID()

	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Job Del Test', ?, '{}', NOW(), NOW())`,
		tenantID, tenantID)
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh', 'fake', X'00', true, '{}', NOW(), NOW())`,
		channelID, tenantID)
	exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, 'ext-jobdel', 'Khach', NOW(), 1, '{}', NOW(), NOW())`,
		convID, tenantID, channelID)
	exec(`INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, 'Job xoa', 'qc_analysis', ?, '', '[]', 'manual', true, '[]', NOW(), NOW())`,
		jobID, tenantID, `["`+channelID+`"]`)
	exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'success', '{}', NOW())`,
		runID, jobID, tenantID)
	exec(`INSERT INTO analysis_snapshots (id, tenant_id, job_run_id, conversation_id, schema_version, digest, coverage, coverage_reasons, message_count, manifest, created_at) VALUES (?, ?, ?, ?, 'ccma.snapshot.v1', 'deadbeef', 'complete', '[]', 1, '{}', NOW())`,
		snapID, tenantID, runID, convID)
	exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, analysis_snapshot_id, created_at) VALUES (?, ?, ?, ?, 'qc_violation', 'NGHIEM_TRONG', 'rule', 'evidence', '{}', 1, ?, NOW())`,
		resultID, runID, tenantID, convID, snapID)
	exec(`INSERT INTO ai_usage_logs (id, tenant_id, job_id, job_run_id, provider, model, input_tokens, output_tokens, cost_usd, created_at) VALUES (?, ?, ?, ?, 'test-double', 'mock', 10, 10, 0, NOW())`,
		pkg.NewUUID(), tenantID, jobID, runID)

	t.Cleanup(func() {
		for _, table := range []string{"job_results", "analysis_snapshots", "ai_usage_logs", "notification_logs", "job_runs", "jobs", "conversations", "activity_logs"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenantID)
		}
		db.DB.Exec("DELETE FROM channels WHERE id = ?", channelID)
		db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
	})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", tenantID)
	c.Set("user_id", "u-test")
	c.Set("user_email", "test@example.com")
	c.Params = gin.Params{{Key: "jobId", Value: jobID}}
	c.Request = httptest.NewRequest("DELETE", "/api/v1/jobs/"+jobID, nil)

	DeleteJob(c)

	if w.Code != http.StatusOK {
		t.Fatalf("DeleteJob status = %d, body = %s", w.Code, w.Body.String())
	}

	var resultCount, snapCount, runCount, usageCount, jobCount int64
	db.DB.Model(&models.JobResult{}).Where("tenant_id = ?", tenantID).Count(&resultCount)
	db.DB.Model(&models.AnalysisSnapshot{}).Where("tenant_id = ?", tenantID).Count(&snapCount)
	db.DB.Model(&models.JobRun{}).Where("tenant_id = ?", tenantID).Count(&runCount)
	db.DB.Model(&models.AIUsageLog{}).Where("tenant_id = ?", tenantID).Count(&usageCount)
	db.DB.Model(&models.Job{}).Where("id = ?", jobID).Count(&jobCount)
	if resultCount != 0 {
		t.Errorf("job_results con lai sau khi xoa job: %d", resultCount)
	}
	if snapCount != 0 {
		t.Errorf("analysis_snapshots con lai sau khi xoa job: %d", snapCount)
	}
	if runCount != 0 {
		t.Errorf("job_runs con lai sau khi xoa job: %d", runCount)
	}
	if usageCount != 0 {
		t.Errorf("ai_usage_logs con lai sau khi xoa job: %d", usageCount)
	}
	if jobCount != 0 {
		t.Errorf("job van con sau khi xoa: %d", jobCount)
	}
}
