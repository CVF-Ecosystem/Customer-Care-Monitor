package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/pkg"
)

func connectChannelsTestDB(t *testing.T) {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("bo qua: TEST_DB_DSN chua duoc thiet lap")
	}
	if err := db.Connect(dsn, false); err != nil {
		t.Skipf("bo qua: khong ket noi duoc DB test: %v", err)
	}
	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate loi: %v", err)
	}
}

// TestDeleteChannelRemovesResultsAndSnapshotsTogether là regression test cho
// R2-B1: DeleteChannel từng xoá analysis_snapshots nhưng bỏ sót job_results,
// để lại result với analysis_snapshot_id trỏ tới một snapshot đã mất (evidence
// lineage sai, AfterFind vẫn báo snapshot_bound dù snapshot không còn).
func TestDeleteChannelRemovesResultsAndSnapshotsTogether(t *testing.T) {
	connectChannelsTestDB(t)
	suffix := pkg.NewUUID()[:8]
	tenantID := "del-" + suffix
	channelID := "ch-del-" + suffix
	convID := "conv-del-" + suffix
	runID := "run-del-" + suffix
	snapID := pkg.NewUUID()
	resultID := pkg.NewUUID()

	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Del Test', ?, '{}', NOW(), NOW())`,
		tenantID, tenantID)
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh xoa', 'fake', X'00', true, '{}', NOW(), NOW())`,
		channelID, tenantID)
	exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, 'ext-del', 'Khach', NOW(), 1, '{}', NOW(), NOW())`,
		convID, tenantID, channelID)
	exec(`INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, 'm1', 'customer', 'Khach', 'Xin chao', 'text', '[]', NOW(), NOW())`,
		pkg.NewUUID(), tenantID, convID)
	exec(`INSERT INTO analysis_snapshots (id, tenant_id, job_run_id, conversation_id, schema_version, digest, coverage, coverage_reasons, message_count, manifest, created_at) VALUES (?, ?, ?, ?, 'ccma.snapshot.v1', 'deadbeef', 'complete', '[]', 1, '{}', NOW())`,
		snapID, tenantID, runID, convID)
	exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, analysis_snapshot_id, created_at) VALUES (?, ?, ?, ?, 'qc_violation', 'NGHIEM_TRONG', 'rule', 'evidence', '{}', 1, ?, NOW())`,
		resultID, runID, tenantID, convID, snapID)

	t.Cleanup(func() {
		for _, table := range []string{"job_results", "analysis_snapshots", "messages", "conversations", "activity_logs"} {
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
	c.Params = gin.Params{{Key: "channelId", Value: channelID}}
	c.Request = httptest.NewRequest("DELETE", "/api/v1/channels/"+channelID, nil)

	DeleteChannel(c)

	if w.Code != 200 {
		t.Fatalf("DeleteChannel status = %d, body = %s", w.Code, w.Body.String())
	}

	var resultCount, snapCount, channelCount int64
	db.DB.Model(&models.JobResult{}).Where("tenant_id = ?", tenantID).Count(&resultCount)
	db.DB.Model(&models.AnalysisSnapshot{}).Where("tenant_id = ?", tenantID).Count(&snapCount)
	db.DB.Model(&models.Channel{}).Where("id = ?", channelID).Count(&channelCount)
	if resultCount != 0 {
		t.Errorf("job_results con lai sau khi xoa kenh: %d", resultCount)
	}
	if snapCount != 0 {
		t.Errorf("analysis_snapshots con lai sau khi xoa kenh (dangling evidence lineage): %d", snapCount)
	}
	if channelCount != 0 {
		t.Errorf("channel van con sau khi xoa: %d", channelCount)
	}
}

// TestDeleteChannelFailureRollsBackWholeCascade là regression test cho R2-RR1:
// trước đây conversation IDs được đọc bằng một SELECT thường ngoài transaction,
// và dọn file chạy trước khi DB commit — một lỗi giữa chừng cascade không có gì
// đảm bảo mọi thứ (kể cả file đã xoá) trở lại nguyên vẹn. Test này ép bước xoá
// một job_result cụ thể báo lỗi bằng trigger MySQL, gọi DeleteChannel qua đúng
// đường HTTP handler, rồi xác nhận channel/conversation/message/result/snapshot
// đều còn nguyên — không có gì bị xoá một phần.
func TestDeleteChannelFailureRollsBackWholeCascade(t *testing.T) {
	connectChannelsTestDB(t)
	suffix := pkg.NewUUID()[:8]
	tenantID := "delfail-" + suffix
	channelID := "ch-delfail-" + suffix
	convID := "conv-delfail-" + suffix
	runID := "run-delfail-" + suffix
	snapID := pkg.NewUUID()
	okResultID := pkg.NewUUID()
	failResultID := "FORCEFAIL" + suffix
	triggerName := "trg_ccma_test_fail_" + suffix

	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Del Fail Test', ?, '{}', NOW(), NOW())`,
		tenantID, tenantID)
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh xoa loi', 'fake', X'00', true, '{}', NOW(), NOW())`,
		channelID, tenantID)
	exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, 'ext-delfail', 'Khach', NOW(), 1, '{}', NOW(), NOW())`,
		convID, tenantID, channelID)
	exec(`INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, 'm1', 'customer', 'Khach', 'Xin chao', 'text', '[]', NOW(), NOW())`,
		pkg.NewUUID(), tenantID, convID)
	exec(`INSERT INTO analysis_snapshots (id, tenant_id, job_run_id, conversation_id, schema_version, digest, coverage, coverage_reasons, message_count, manifest, created_at) VALUES (?, ?, ?, ?, 'ccma.snapshot.v1', 'deadbeef', 'complete', '[]', 1, '{}', NOW())`,
		snapID, tenantID, runID, convID)
	exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, analysis_snapshot_id, created_at) VALUES (?, ?, ?, ?, 'qc_violation', 'NGHIEM_TRONG', 'rule-ok', 'evidence', '{}', 1, ?, NOW())`,
		okResultID, runID, tenantID, convID, snapID)
	exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, analysis_snapshot_id, created_at) VALUES (?, ?, ?, ?, 'qc_violation', 'NGHIEM_TRONG', 'rule-fail', 'evidence', '{}', 1, ?, NOW())`,
		failResultID, runID, tenantID, convID, snapID)

	// failResultID is a generated alphanumeric string with no quotes, safe to
	// inline: a trigger body cannot bind query parameters.
	exec(fmt.Sprintf(`CREATE TRIGGER %s BEFORE DELETE ON job_results
FOR EACH ROW
BEGIN
	IF OLD.id = '%s' THEN
		SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced failure for delete-channel rollback test';
	END IF;
END`, triggerName, failResultID))

	t.Cleanup(func() {
		db.DB.Exec("DROP TRIGGER IF EXISTS " + triggerName)
		for _, table := range []string{"job_results", "analysis_snapshots", "messages", "conversations", "activity_logs"} {
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
	c.Params = gin.Params{{Key: "channelId", Value: channelID}}
	c.Request = httptest.NewRequest("DELETE", "/api/v1/channels/"+channelID, nil)

	DeleteChannel(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("DeleteChannel status = %d, want %d (forced failure), body = %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}

	var channelCount, convCount, msgCount, resultCount, snapCount int64
	db.DB.Model(&models.Channel{}).Where("id = ?", channelID).Count(&channelCount)
	db.DB.Model(&models.Conversation{}).Where("id = ?", convID).Count(&convCount)
	db.DB.Model(&models.Message{}).Where("conversation_id = ?", convID).Count(&msgCount)
	db.DB.Model(&models.JobResult{}).Where("tenant_id = ?", tenantID).Count(&resultCount)
	db.DB.Model(&models.AnalysisSnapshot{}).Where("tenant_id = ?", tenantID).Count(&snapCount)

	if channelCount != 1 {
		t.Errorf("channel khong con nguyen ven sau cascade that bai: %d", channelCount)
	}
	if convCount != 1 {
		t.Errorf("conversation khong con nguyen ven sau cascade that bai: %d", convCount)
	}
	if msgCount != 1 {
		t.Errorf("message khong con nguyen ven sau cascade that bai: %d", msgCount)
	}
	if resultCount != 2 {
		t.Errorf("job_results con %d, muon ca 2 dong (ke ca dong gay loi) van nguyen ven", resultCount)
	}
	if snapCount != 1 {
		t.Errorf("analysis_snapshot khong con nguyen ven sau cascade that bai: %d", snapCount)
	}
}
