package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/pkg"
)

// Test doubles below exercise parsing and persistence only. They are not
// evidence of AI quality or of governance over a real provider.

var transcriptLine = regexp.MustCompile(`\| msg:([^\]]+)\] .*? \((?:customer|agent|system)\): (.*)`)

type transcriptRef struct{ messageID, content string }

func firstTranscriptRef(transcript string) transcriptRef {
	for _, line := range strings.Split(transcript, "\n") {
		if m := transcriptLine.FindStringSubmatch(line); m != nil {
			return transcriptRef{messageID: m[1], content: m[2]}
		}
	}
	return transcriptRef{}
}

func qcFailWithRef(ref map[string]interface{}) json.RawMessage {
	b, _ := json.Marshal(map[string]interface{}{
		"verdict": "FAIL", "score": 40, "review": "Chua xu ly khieu nai.", "summary": "Khach cho lau.",
		"violations": []map[string]interface{}{{
			"severity": "NGHIEM_TRONG", "rule": "Phan hoi dung han", "evidence": ref["quote"],
			"evidence_refs": []interface{}{ref}, "explanation": "Khach cho qua lau.",
		}},
	})
	return b
}

// refProvider cites the first transcript message. mode "badquote" corrupts
// the quote; mode "cross" makes batch item i cite item i+1's message.
type refProvider struct{ mode string }

func (p *refProvider) refFor(transcripts []string, i int) map[string]interface{} {
	src := firstTranscriptRef(transcripts[i])
	if p.mode == "cross" && i == 0 && len(transcripts) > 1 {
		src = firstTranscriptRef(transcripts[1])
	}
	quote := []rune(src.content)
	if len(quote) > 8 {
		quote = quote[:8]
	}
	ref := map[string]interface{}{"message_id": src.messageID, "quote": string(quote), "start": 0, "end": len(quote)}
	if p.mode == "badquote" {
		ref["quote"] = "khong co trong tin nhan"
		delete(ref, "start")
		delete(ref, "end")
	}
	return ref
}

func (p *refProvider) AnalyzeChat(_ context.Context, _ string, transcript string) (ai.AIResponse, error) {
	return ai.AIResponse{Content: string(qcFailWithRef(p.refFor([]string{transcript}, 0))), Model: "test-double", Provider: "test-double"}, nil
}

func (p *refProvider) AnalyzeChatBatch(_ context.Context, _ string, items []ai.BatchItem) (ai.AIResponse, error) {
	transcripts := make([]string, len(items))
	for i, it := range items {
		transcripts[i] = it.Transcript
	}
	out := make([]json.RawMessage, len(items))
	for i := range items {
		out[i] = qcFailWithRef(p.refFor(transcripts, i))
	}
	b, _ := json.Marshal(out)
	return ai.AIResponse{Content: string(b), Model: "test-double", Provider: "test-double"}, nil
}

type snapshotDBFixture struct {
	tenantID, channelID, jobID string
	convIDs                    []string
}

func connectTestDB(t *testing.T) {
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

func setupSnapshotDBFixture(t *testing.T, batchMode bool) *snapshotDBFixture {
	t.Helper()
	connectTestDB(t)
	suffix := pkg.NewUUID()[:8]
	f := &snapshotDBFixture{tenantID: "snap-" + suffix, channelID: "ch-snap-" + suffix, jobID: "job-snap-" + suffix,
		convIDs: []string{"conv-snap-a-" + suffix, "conv-snap-b-" + suffix}}
	channelIDs, _ := json.Marshal([]string{f.channelID})
	lastAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)

	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Snapshot Test', ?, '{}', NOW(), NOW())`, f.tenantID, f.tenantID)
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh test', 'fake', X'00', true, '{}', NOW(), NOW())`, f.channelID, f.tenantID)
	contents := [][2]string{
		{"Đơn hàng chưa tới 😡 ba ngày rồi", "Dạ em kiểm tra ngay ạ"},
		{"Shop ơi hoàn tiền giúp mình 🙏", "Bên em đã ghi nhận ạ"},
	}
	for i, convID := range f.convIDs {
		exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, 'Khach', ?, 2, '{}', NOW(), NOW())`,
			convID, f.tenantID, f.channelID, "ext-"+convID, lastAt)
		exec(`INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, 'm1', 'customer', 'Khách', ?, 'text', '[]', ?, NOW())`,
			pkg.NewUUID(), f.tenantID, convID, contents[i][0], lastAt.Add(-5*time.Minute))
		exec(`INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, 'm2', 'agent', 'NV', ?, 'text', '[]', ?, NOW())`,
			pkg.NewUUID(), f.tenantID, convID, contents[i][1], lastAt)
	}
	exec(`INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, schedule_cron, is_active, outputs, output_schedule, created_at, updated_at) VALUES (?, ?, 'QC Snapshot', 'qc_analysis', ?, 'Phan hoi dung han.', '[]', 'manual', '', true, '[]', 'none', NOW(), NOW())`,
		f.jobID, f.tenantID, string(channelIDs))
	mode := "true"
	if !batchMode {
		mode = "false"
	}
	exec(`INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at) VALUES (?, ?, 'ai_batch_mode', ?, NOW(), NOW())`, pkg.NewUUID(), f.tenantID, mode)

	t.Cleanup(func() {
		for _, table := range []string{"job_results", "analysis_snapshots", "job_runs", "ai_usage_logs", "messages", "conversations", "app_settings", "activity_logs"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", f.tenantID)
		}
		db.DB.Exec("DELETE FROM jobs WHERE id = ?", f.jobID)
		db.DB.Exec("DELETE FROM channels WHERE id = ?", f.channelID)
		db.DB.Exec("DELETE FROM tenants WHERE id = ?", f.tenantID)
	})
	return f
}

func (f *snapshotDBFixture) run(t *testing.T, provider ai.AIProvider) *models.JobRun {
	t.Helper()
	var job models.Job
	if err := db.DB.First(&job, "id = ?", f.jobID).Error; err != nil {
		t.Fatal(err)
	}
	run, err := NewAnalyzer(&config.Config{}).RunJobWithProvider(context.Background(), job, 0, provider)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return run
}

func snapshotsFor(t *testing.T, runID string) map[string]models.AnalysisSnapshot {
	t.Helper()
	var rows []models.AnalysisSnapshot
	if err := db.DB.Where("job_run_id = ?", runID).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	out := map[string]models.AnalysisSnapshot{}
	for _, r := range rows {
		out[r.ConversationID] = r
	}
	return out
}

func resultsFor(t *testing.T, runID string) []models.JobResult {
	t.Helper()
	var rows []models.JobResult
	if err := db.DB.Where("job_run_id = ?", runID).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestSingleAndBatchShareSnapshotContract(t *testing.T) {
	for _, batch := range []bool{false, true} {
		f := setupSnapshotDBFixture(t, batch)
		run := f.run(t, &refProvider{})
		if got := summaryInt(t, run, "conversations_analyzed"); got != 2 {
			t.Fatalf("batch=%v: analyzed %d, want 2 (%s)", batch, got, run.Summary)
		}
		snaps := snapshotsFor(t, run.ID)
		if len(snaps) != 2 {
			t.Fatalf("batch=%v: %d snapshots, want 2", batch, len(snaps))
		}
		results := resultsFor(t, run.ID)
		if len(results) != 4 {
			t.Fatalf("batch=%v: %d results, want 4", batch, len(results))
		}
		for _, r := range results {
			snap, ok := snaps[r.ConversationID]
			if r.AnalysisSnapshotID == nil || !ok || *r.AnalysisSnapshotID != snap.ID {
				t.Fatalf("batch=%v: result %s not linked to its conversation snapshot", batch, r.ID)
			}
			if r.EvidenceStatus != "snapshot_bound" {
				t.Fatalf("batch=%v: evidence status %q", batch, r.EvidenceStatus)
			}
		}
		// Both paths must store exactly what the shared loader rebuilds.
		for _, convID := range f.convIDs {
			s := snaps[convID]
			if s.SchemaVersion != snapshotSchemaV1 || s.Coverage != coverageComplete || s.MessageCount != 2 {
				t.Fatalf("batch=%v: unexpected snapshot %+v", batch, s)
			}
			var conv models.Conversation
			db.DB.First(&conv, "id = ?", convID)
			rebuilt, err := loadConversationSnapshot(conv, time.Time{})
			if err != nil || rebuilt.Digest != s.Digest || string(rebuilt.ManifestJSON) != s.Manifest {
				t.Fatalf("batch=%v: stored snapshot does not match shared builder (err=%v)", batch, err)
			}
			if sum := sha256.Sum256([]byte(s.Manifest)); hex.EncodeToString(sum[:]) != s.Digest {
				t.Fatalf("batch=%v: stored manifest does not hash to stored digest", batch)
			}
		}
	}
}

func TestInvalidEvidenceRejectsWholeConversation(t *testing.T) {
	for _, batch := range []bool{false, true} {
		f := setupSnapshotDBFixture(t, batch)
		run := f.run(t, &refProvider{mode: "badquote"})
		if got := summaryInt(t, run, "conversations_errors"); got != 2 {
			t.Fatalf("batch=%v: errors %d, want 2 (%s)", batch, got, run.Summary)
		}
		if n := len(resultsFor(t, run.ID)); n != 0 {
			t.Fatalf("batch=%v: %d results saved from invalid evidence", batch, n)
		}
		if n := len(snapshotsFor(t, run.ID)); n != 0 {
			t.Fatalf("batch=%v: %d orphan snapshots saved", batch, n)
		}
	}
}

func TestBatchRejectsCrossConversationRef(t *testing.T) {
	f := setupSnapshotDBFixture(t, true)
	run := f.run(t, &refProvider{mode: "cross"})
	if got := summaryInt(t, run, "conversations_errors"); got != 1 {
		t.Fatalf("errors %d, want 1 (%s)", got, run.Summary)
	}
	snaps := snapshotsFor(t, run.ID)
	if _, ok := snaps[f.convIDs[0]]; ok || len(snaps) != 1 {
		t.Fatalf("cross-conversation result persisted: %v", snaps)
	}
	for _, r := range resultsFor(t, run.ID) {
		if r.ConversationID == f.convIDs[0] {
			t.Fatal("result saved for conversation citing another conversation")
		}
	}
}

func TestSaveResultsRollsBackSnapshotOnFailure(t *testing.T) {
	f := setupSnapshotDBFixture(t, false)
	var conv models.Conversation
	db.DB.First(&conv, "id = ?", f.convIDs[0])
	snap, err := loadConversationSnapshot(conv, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	ref := (&refProvider{}).refFor([]string{snap.Transcript}, 0)
	analyzer := NewAnalyzer(&config.Config{})

	// A rule name longer than varchar(255) fails after the snapshot insert.
	var payload map[string]interface{}
	json.Unmarshal(qcFailWithRef(ref), &payload)
	payload["violations"].([]interface{})[0].(map[string]interface{})["rule"] = strings.Repeat("r", 300)
	tooLong, _ := json.Marshal(payload)
	runID := "run-rollback-" + pkg.NewUUID()[:8]
	// R2-RR3: saveResults now locks/confirms the parent JobRun before writing
	// evidence, so a real row must exist for it to find.
	if err := db.DB.Exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'running', '{}', NOW())`,
		runID, f.jobID, f.tenantID).Error; err != nil {
		t.Fatalf("fixture: insert job_run: %v", err)
	}
	if _, _, err := analyzer.saveResults(runID, snap, "qc_analysis", string(tooLong)); err == nil {
		t.Fatal("expected insert failure for oversized rule name (is MySQL strict mode on?)")
	}
	if n := len(snapshotsFor(t, runID)); n != 0 {
		t.Fatalf("snapshot survived a failed transaction: %d rows", n)
	}
	if n := len(resultsFor(t, runID)); n != 0 {
		t.Fatalf("results survived a failed transaction: %d rows", n)
	}

	// Snapshot persist failure (duplicate run/conversation) creates nothing new.
	if _, _, err := analyzer.saveResults(runID, snap, "qc_analysis", string(qcFailWithRef(ref))); err != nil {
		t.Fatalf("valid save failed: %v", err)
	}
	if _, _, err := analyzer.saveResults(runID, snap, "qc_analysis", string(qcFailWithRef(ref))); err == nil {
		t.Fatal("second snapshot for the same run/conversation was accepted")
	}
	if n := len(resultsFor(t, runID)); n != 2 {
		t.Fatalf("duplicate save left %d results, want 2", n)
	}
}

// deleteConversationLockingChildrenFirst mirrors DeleteChannel's parent-first
// cascade for a single conversation: lock the conversation row, delete its
// evidence, then delete the conversation itself, all in one transaction.
func deleteConversationLockingChildrenFirst(tenantID, convID string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var conv models.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND tenant_id = ?", convID, tenantID).First(&conv).Error; err != nil {
			return err
		}
		if err := tx.Where("conversation_id = ? AND tenant_id = ?", convID, tenantID).Delete(&models.JobResult{}).Error; err != nil {
			return err
		}
		if err := tx.Where("conversation_id = ? AND tenant_id = ?", convID, tenantID).Delete(&models.AnalysisSnapshot{}).Error; err != nil {
			return err
		}
		if err := tx.Where("conversation_id = ? AND tenant_id = ?", convID, tenantID).Delete(&models.Message{}).Error; err != nil {
			return err
		}
		return tx.Delete(&conv).Error
	})
}

// TestSaveResultsFailsWhenParentDeletedFirst is the "delete wins" ordering
// from R2-RR3 acceptance #4: the conversation is gone before saveResults ever
// starts, so its new locking check must fail cleanly and create nothing.
func TestSaveResultsFailsWhenParentDeletedFirst(t *testing.T) {
	f := setupSnapshotDBFixture(t, false)
	var conv models.Conversation
	if err := db.DB.First(&conv, "id = ?", f.convIDs[0]).Error; err != nil {
		t.Fatal(err)
	}
	snap, err := loadConversationSnapshot(conv, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	ref := (&refProvider{}).refFor([]string{snap.Transcript}, 0)
	runID := "run-delfirst-" + pkg.NewUUID()[:8]
	if err := db.DB.Exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'running', '{}', NOW())`,
		runID, f.jobID, f.tenantID).Error; err != nil {
		t.Fatalf("fixture: insert job_run: %v", err)
	}

	if err := deleteConversationLockingChildrenFirst(f.tenantID, conv.ID); err != nil {
		t.Fatalf("delete conversation: %v", err)
	}

	analyzer := NewAnalyzer(&config.Config{})
	if _, _, err := analyzer.saveResults(runID, snap, "qc_analysis", string(qcFailWithRef(ref))); err == nil {
		t.Fatal("saveResults accepted a snapshot for an already-deleted conversation")
	}
	if n := len(snapshotsFor(t, runID)); n != 0 {
		t.Fatalf("orphan snapshot created for deleted conversation: %d rows", n)
	}
	if n := len(resultsFor(t, runID)); n != 0 {
		t.Fatalf("orphan result created for deleted conversation: %d rows", n)
	}
}

// TestWriterHoldsParentLockDeleteWaitsThenCleansEvidence is the "writer wins"
// ordering from R2-RR3 acceptance #4: a transaction holding the same
// Conversation/JobRun locks saveResults now takes must make a concurrent
// delete wait, and once the writer commits, the delete's child-cleanup step
// must catch the evidence it just wrote — nothing survives half-applied.
func TestWriterHoldsParentLockDeleteWaitsThenCleansEvidence(t *testing.T) {
	f := setupSnapshotDBFixture(t, false)
	var conv models.Conversation
	if err := db.DB.First(&conv, "id = ?", f.convIDs[0]).Error; err != nil {
		t.Fatal(err)
	}
	snap, err := loadConversationSnapshot(conv, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	ref := (&refProvider{}).refFor([]string{snap.Transcript}, 0)
	runID := "run-writerfirst-" + pkg.NewUUID()[:8]
	if err := db.DB.Exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'running', '{}', NOW())`,
		runID, f.jobID, f.tenantID).Error; err != nil {
		t.Fatalf("fixture: insert job_run: %v", err)
	}

	writerLocked := make(chan struct{})
	proceedToCommit := make(chan struct{})
	writerDone := make(chan error, 1)

	go func() {
		// Reproduces saveResults' own locking prefix directly so the test can
		// pause it between "lock acquired" and "commit" — saveResults itself has
		// no such hook, and shouldn't grow one just for a test.
		tx := db.DB.Begin()
		defer tx.Rollback()
		var lockedConv models.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND tenant_id = ?", conv.ID, f.tenantID).First(&lockedConv).Error; err != nil {
			writerDone <- err
			return
		}
		var lockedRun models.JobRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND tenant_id = ?", runID, f.tenantID).First(&lockedRun).Error; err != nil {
			writerDone <- err
			return
		}
		close(writerLocked)
		<-proceedToCommit

		snapRow, err := snap.record(runID)
		if err != nil {
			writerDone <- err
			return
		}
		if err := tx.Create(&snapRow).Error; err != nil {
			writerDone <- err
			return
		}
		detail, _ := json.Marshal(map[string]interface{}{"evidence_refs": []interface{}{ref}})
		result := models.JobResult{
			ID: pkg.NewUUID(), JobRunID: runID, TenantID: f.tenantID, ConversationID: conv.ID,
			AnalysisSnapshotID: &snapRow.ID, ResultType: "qc_violation", Severity: "NGHIEM_TRONG",
			RuleName: "r", Evidence: ref["quote"].(string), Detail: string(detail), Confidence: 1, CreatedAt: time.Now(),
		}
		if err := tx.Create(&result).Error; err != nil {
			writerDone <- err
			return
		}
		writerDone <- tx.Commit().Error
	}()

	select {
	case <-writerLocked:
	case <-time.After(5 * time.Second):
		t.Fatal("writer never acquired its parent locks")
	}

	deleteDone := make(chan error, 1)
	go func() {
		deleteDone <- deleteConversationLockingChildrenFirst(f.tenantID, conv.ID)
	}()

	// The delete must not complete while the writer still holds the lock.
	select {
	case err := <-deleteDone:
		t.Fatalf("delete completed before writer released its lock (err=%v) — locking did not block it", err)
	case <-time.After(300 * time.Millisecond):
	}

	close(proceedToCommit)
	if err := <-writerDone; err != nil {
		t.Fatalf("writer failed: %v", err)
	}

	select {
	case err := <-deleteDone:
		if err != nil {
			t.Fatalf("delete failed after writer released its lock: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("delete did not complete after writer committed")
	}

	var resultCount, snapCount, convCount int64
	db.DB.Model(&models.JobResult{}).Where("conversation_id = ? AND tenant_id = ?", conv.ID, f.tenantID).Count(&resultCount)
	db.DB.Model(&models.AnalysisSnapshot{}).Where("conversation_id = ? AND tenant_id = ?", conv.ID, f.tenantID).Count(&snapCount)
	db.DB.Model(&models.Conversation{}).Where("id = ?", conv.ID).Count(&convCount)
	if resultCount != 0 {
		t.Errorf("result the writer committed survived the delete: %d rows", resultCount)
	}
	if snapCount != 0 {
		t.Errorf("snapshot the writer committed survived the delete: %d rows", snapCount)
	}
	if convCount != 0 {
		t.Errorf("conversation survived its own delete: %d rows", convCount)
	}
}

func TestLegacyResultsAreMarkedUnverified(t *testing.T) {
	f := setupSnapshotDBFixture(t, false)
	id := pkg.NewUUID()
	if err := db.DB.Exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, created_at) VALUES (?, 'legacy-run', ?, ?, 'qc_violation', 'NGHIEM_TRONG', 'Cu', 'trich dan cu', '{}', 1, NOW())`,
		id, f.tenantID, f.convIDs[0]).Error; err != nil {
		t.Fatal(err)
	}
	var r models.JobResult
	if err := db.DB.First(&r, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if r.AnalysisSnapshotID != nil || r.EvidenceStatus != "legacy_unverified" {
		t.Fatalf("legacy row: snapshot=%v status=%q", r.AnalysisSnapshotID, r.EvidenceStatus)
	}
	b, _ := json.Marshal(r)
	if !strings.Contains(string(b), `"analysis_snapshot_id":null`) || !strings.Contains(string(b), `"evidence_status":"legacy_unverified"`) {
		t.Fatalf("API JSON does not expose legacy state: %s", b)
	}
}
