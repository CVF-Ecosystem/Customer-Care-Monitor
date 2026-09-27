package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/pkg"
)

func TestParseScore(t *testing.T) {
	cases := []struct {
		ten    string
		detail string
		muon   *float64
	}{
		{"khong co detail", "", nil},
		{"detail hong", "{khong phai json", nil},
		{"khong co khoa score", `{"summary":"abc"}`, nil},
		{"score la so", `{"score":88}`, ptrFloat(88)},
		{"score la chuoi", `{"score":"72.5"}`, ptrFloat(72.5)},
	}
	for _, c := range cases {
		t.Run(c.ten, func(t *testing.T) {
			got := parseScore(c.detail)
			if c.muon == nil {
				if got != nil {
					t.Fatalf("muon nil, nhan %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("muon %v, nhan nil", *c.muon)
			}
			if *got != *c.muon {
				t.Fatalf("muon %v, nhan %v", *c.muon, *got)
			}
		})
	}
}

func ptrFloat(v float64) *float64 { return &v }

func TestSplitCSVParam(t *testing.T) {
	if got := splitCSVParam("  "); got != nil {
		t.Fatalf("chuoi rong phai tra nil, nhan %v", got)
	}
	got := splitCSVParam("a, b ,,c")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("tach sai: %v", got)
	}
}

func TestVerdictLabel(t *testing.T) {
	if verdictLabel("PASS") != "Đạt" || verdictLabel("SKIP") != "Bỏ qua" || verdictLabel("NGHIEM_TRONG") != "Không đạt" {
		t.Fatal("nhãn kết quả sai")
	}
}

func TestJoinIssues(t *testing.T) {
	if joinIssues(nil) != "" {
		t.Fatal("khong co van de thi phai tra chuoi rong")
	}
	got := joinIssues([]issueRow{
		{RuleName: "Chào hỏi"},
		{RuleName: "Phản hồi", Evidence: "chậm 18 phút"},
	})
	if got != "Chào hỏi; Phản hồi: chậm 18 phút" {
		t.Fatalf("gộp vấn đề sai: %s", got)
	}
}

// resultsFixture dựng dữ liệu thật trong MySQL test: một tác vụ QC chạy hai lần
// trên cùng một cuộc chat, cộng thêm một cuộc chat bị bỏ qua.
type resultsFixture struct {
	tenantID  string
	channelID string
	convID    string
	conv2ID   string
	jobID     string
}

func setupResultsFixture(t *testing.T) *resultsFixture {
	t.Helper()

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "cqa:cqa_password@tcp(127.0.0.1:3306)/cqa?charset=utf8mb4&parseTime=True&loc=UTC"
	}
	if err := db.Connect(dsn, false); err != nil {
		t.Skipf("bo qua: khong ket noi duoc DB test: %v", err)
	}
	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate loi: %v", err)
	}

	f := &resultsFixture{
		tenantID:  "restest-" + pkg.NewUUID()[:8],
		channelID: "ch-restest-" + pkg.NewUUID()[:8],
		convID:    "conv-restest-" + pkg.NewUUID()[:8],
		conv2ID:   "conv2-restest-" + pkg.NewUUID()[:8],
		jobID:     "job-restest-" + pkg.NewUUID()[:8],
	}
	channelIDsJSON, _ := json.Marshal([]string{f.channelID})
	convAt := time.Now().Add(-48 * time.Hour)

	db.DB.Exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, ?, ?, '{}', NOW(), NOW())`,
		f.tenantID, "Results Test", f.tenantID)
	db.DB.Exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'zalo_oa', 'Kenh ket qua', 'fake', X'00', true, '{}', NOW(), NOW())`,
		f.channelID, f.tenantID)
	db.DB.Exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, 'ext-1', 'Khach Mot', ?, 2, '{}', NOW(), NOW())`,
		f.convID, f.tenantID, f.channelID, convAt)
	db.DB.Exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, 'ext-2', 'Khach Hai', ?, 1, '{}', NOW(), NOW())`,
		f.conv2ID, f.tenantID, f.channelID, convAt)
	db.DB.Exec(`INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, schedule_cron, is_active, outputs, output_schedule, created_at, updated_at) VALUES (?, ?, 'QC Ket qua', 'qc_analysis', ?, 'Quy tac', '[]', 'cron', '0 7 * * *', true, '[]', 'none', NOW(), NOW())`,
		f.jobID, f.tenantID, string(channelIDsJSON))

	// Lần chạy cũ: cuộc chat 1 bị đánh Không đạt
	runCu := "run-cu-" + pkg.NewUUID()[:8]
	runMoi := "run-moi-" + pkg.NewUUID()[:8]
	cu := time.Now().Add(-24 * time.Hour)
	moi := time.Now().Add(-1 * time.Hour)
	db.DB.Exec(`INSERT INTO job_runs (id, job_id, tenant_id, status, summary, started_at, created_at) VALUES (?, ?, ?, 'success', '{}', ?, ?)`,
		runCu, f.jobID, f.tenantID, cu, cu)
	db.DB.Exec(`INSERT INTO job_runs (id, job_id, tenant_id, status, summary, started_at, created_at) VALUES (?, ?, ?, 'success', '{}', ?, ?)`,
		runMoi, f.jobID, f.tenantID, moi, moi)

	themKetQua(f.tenantID, runCu, f.convID, "conversation_evaluation", "NGHIEM_TRONG", "", "Lan cu", `{"score":40}`, cu)
	themKetQua(f.tenantID, runCu, f.convID, "qc_violation", "NGHIEM_TRONG", "Chao hoi", "Khong chao", `{}`, cu)
	// Lần chạy mới: cùng cuộc chat nhưng đã Đạt
	themKetQua(f.tenantID, runMoi, f.convID, "conversation_evaluation", "PASS", "", "Lan moi", `{"score":90}`, moi)
	// Cuộc chat 2 bị bỏ qua
	themKetQua(f.tenantID, runMoi, f.conv2ID, "conversation_evaluation", "SKIP", "", "Chi co 1 tin nhan", `{}`, moi)

	t.Cleanup(func() {
		db.DB.Exec("DELETE FROM job_results WHERE tenant_id = ?", f.tenantID)
		db.DB.Exec("DELETE FROM job_runs WHERE tenant_id = ?", f.tenantID)
		db.DB.Exec("DELETE FROM jobs WHERE id = ?", f.jobID)
		db.DB.Exec("DELETE FROM conversations WHERE tenant_id = ?", f.tenantID)
		db.DB.Exec("DELETE FROM channels WHERE id = ?", f.channelID)
		db.DB.Exec("DELETE FROM tenants WHERE id = ?", f.tenantID)
	})
	return f
}

func themKetQua(tenantID, runID, convID, resultType, severity, ruleName, evidence, detail string, at time.Time) {
	db.DB.Create(&models.JobResult{
		ID:             pkg.NewUUID(),
		JobRunID:       runID,
		TenantID:       tenantID,
		ConversationID: convID,
		ResultType:     resultType,
		Severity:       severity,
		RuleName:       ruleName,
		Evidence:       evidence,
		Detail:         detail,
		Confidence:     1,
		CreatedAt:      at,
	})
}

func (f *resultsFixture) loc() resultFilter {
	return resultFilter{tenantID: f.tenantID, jobType: "qc_analysis", verdict: "all", dateField: "conv", sort: "recent"}
}

// Chạy lại tác vụ không được sinh ra dòng trùng: mỗi cuộc chat chỉ còn lần đánh giá mới nhất.
func TestFetchRowsChiLayLanDanhGiaMoiNhat(t *testing.T) {
	f := setupResultsFixture(t)

	rows, err := f.loc().fetchRows(50, 0)
	if err != nil {
		t.Fatalf("fetchRows loi: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("muon 2 cuoc chat, nhan %d", len(rows))
	}

	var conv1 *resultRow
	for i := range rows {
		if rows[i].ConversationID == f.convID {
			conv1 = &rows[i]
		}
	}
	if conv1 == nil {
		t.Fatal("khong thay cuoc chat 1")
	}
	if conv1.Severity != "PASS" {
		t.Fatalf("phai lay lan chay moi nhat (PASS), nhan %s", conv1.Severity)
	}
	if conv1.Score == nil || *conv1.Score != 90 {
		t.Fatalf("diem phai la 90 cua lan moi, nhan %v", conv1.Score)
	}
	// Vi phạm của lần chạy cũ không được dính sang kết quả mới
	if len(conv1.Issues) != 0 {
		t.Fatalf("lan chay moi khong co vi pham, nhan %d", len(conv1.Issues))
	}
}

func TestVerdictCountsVaLocTheoNhan(t *testing.T) {
	f := setupResultsFixture(t)

	counts, err := f.loc().verdictCounts()
	if err != nil {
		t.Fatalf("verdictCounts loi: %v", err)
	}
	if counts["all"] != 2 || counts["pass"] != 1 || counts["skip"] != 1 || counts["fail"] != 0 {
		t.Fatalf("dem sai: %v", counts)
	}

	locSkip := f.loc()
	locSkip.verdict = "skip"
	rows, err := locSkip.fetchRows(50, 0)
	if err != nil {
		t.Fatalf("fetchRows loi: %v", err)
	}
	if len(rows) != 1 || rows[0].ConversationID != f.conv2ID {
		t.Fatalf("loc Bo qua sai: %+v", rows)
	}
}

func TestLocTheoDiemVaTenKhach(t *testing.T) {
	f := setupResultsFixture(t)

	min80, max100 := 80.0, 100.0
	locDiem := f.loc()
	locDiem.scoreMin = &min80
	locDiem.scoreMax = &max100
	rows, err := locDiem.fetchRows(50, 0)
	if err != nil {
		t.Fatalf("fetchRows loi: %v", err)
	}
	if len(rows) != 1 || rows[0].ConversationID != f.convID {
		t.Fatalf("loc theo diem sai: %+v", rows)
	}

	locTen := f.loc()
	locTen.keyword = "Khach Hai"
	rows, err = locTen.fetchRows(50, 0)
	if err != nil {
		t.Fatalf("fetchRows loi: %v", err)
	}
	if len(rows) != 1 || rows[0].CustomerName != "Khach Hai" {
		t.Fatalf("tim theo ten sai: %+v", rows)
	}
}

// Công ty khác không được nhìn thấy kết quả của công ty này.
func TestKhongLoDuLieuSangCongTyKhac(t *testing.T) {
	f := setupResultsFixture(t)

	locKhac := f.loc()
	locKhac.tenantID = "cong-ty-khac-" + pkg.NewUUID()[:8]
	rows, err := locKhac.fetchRows(50, 0)
	if err != nil {
		t.Fatalf("fetchRows loi: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("cong ty khac phai khong thay gi, nhan %d dong", len(rows))
	}
}

// --- CCMAI-RUNTIME-004: source_integrity_status ---
//
// These tests hand-construct a ccma.snapshot.v1 manifest JSON string matching
// the exact wire shape backend/engine's canonicalizeMessage/buildConversationSnapshot
// produce (message_id, external_message_id, sender_type, sender_name,
// content_type, sent_at, content_sha256, content_code_points,
// attachment_coverage, attachment_count) rather than calling those unexported
// engine functions directly (this test file is in package handlers). The
// comparison algorithm itself (engine.CompareSnapshotToCurrentMessages) has
// its own focused unit tests in backend/engine/source_integrity_test.go;
// these tests instead prove results.go's query/batching/status-assignment
// wiring is correct end to end against a real disposable-MySQL snapshot row.

type sourceIntegrityFixture struct {
	tenantID, channelID, convID, jobID, runID string
}

func setupSourceIntegrityFixture(t *testing.T) *sourceIntegrityFixture {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "cqa:cqa_password@tcp(127.0.0.1:3306)/cqa?charset=utf8mb4&parseTime=True&loc=UTC"
	}
	if err := db.Connect(dsn, false); err != nil {
		t.Skipf("bo qua: khong ket noi duoc DB test: %v", err)
	}
	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate loi: %v", err)
	}

	suffix := pkg.NewUUID()[:8]
	f := &sourceIntegrityFixture{
		tenantID:  "srcint-" + suffix,
		channelID: "ch-srcint-" + suffix,
		convID:    "conv-srcint-" + suffix,
		jobID:     "job-srcint-" + suffix,
		runID:     "run-srcint-" + suffix,
	}
	channelIDsJSON, _ := json.Marshal([]string{f.channelID})
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Source Integrity Test', ?, '{}', NOW(), NOW())`,
		f.tenantID, f.tenantID)
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh srcint', 'fake', X'00', true, '{}', NOW(), NOW())`,
		f.channelID, f.tenantID)
	exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, 'ext-srcint', 'Khach', NOW(), 1, '{}', NOW(), NOW())`,
		f.convID, f.tenantID, f.channelID)
	exec(`INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, schedule_cron, is_active, outputs, output_schedule, created_at, updated_at) VALUES (?, ?, 'QC srcint', 'qc_analysis', ?, 'Quy tac', '[]', 'manual', '', true, '[]', 'none', NOW(), NOW())`,
		f.jobID, f.tenantID, string(channelIDsJSON))
	exec(`INSERT INTO job_runs (id, job_id, tenant_id, status, summary, started_at, created_at) VALUES (?, ?, ?, 'success', '{}', NOW(), NOW())`,
		f.runID, f.jobID, f.tenantID)

	t.Cleanup(func() {
		for _, table := range []string{"job_results", "analysis_snapshots", "messages", "job_runs", "jobs", "conversations"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", f.tenantID)
		}
		db.DB.Exec("DELETE FROM channels WHERE id = ?", f.channelID)
		db.DB.Exec("DELETE FROM tenants WHERE id = ?", f.tenantID)
	})
	return f
}

func (f *sourceIntegrityFixture) loc() resultFilter {
	return resultFilter{tenantID: f.tenantID, jobType: "qc_analysis", verdict: "all", dateField: "conv", sort: "recent"}
}

func (f *sourceIntegrityFixture) insertMessage(t *testing.T, id, externalID, senderType, senderName, content string, sentAt time.Time) {
	t.Helper()
	if err := db.DB.Exec(`INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, 'text', '[]', ?, NOW())`,
		id, f.tenantID, f.convID, externalID, senderType, senderName, content, sentAt).Error; err != nil {
		t.Fatalf("fixture message: %v", err)
	}
}

// snapshotMessageEntry mirrors backend/engine's snapshotMessage wire shape
// for a plain-text, no-attachment message — exactly what canonicalizeMessage
// produces for such a message.
func snapshotMessageEntry(id, externalID, senderType, senderName string, sentAt time.Time, content string) map[string]interface{} {
	sum := sha256.Sum256([]byte(content))
	return map[string]interface{}{
		"message_id":          id,
		"external_message_id": externalID,
		"sender_type":         senderType,
		"sender_name":         senderName,
		"content_type":        "text",
		"sent_at":             sentAt.UTC().Format(time.RFC3339Nano),
		"content_sha256":      hex.EncodeToString(sum[:]),
		"content_code_points": utf8.RuneCountInString(content),
		"attachment_coverage": "none",
		"attachment_count":    0,
	}
}

func (f *sourceIntegrityFixture) manifestJSON(t *testing.T, entries ...map[string]interface{}) string {
	t.Helper()
	m := map[string]interface{}{
		"schema_version":           "ccma.snapshot.v1",
		"tenant_id":                f.tenantID,
		"conversation_id":          f.convID,
		"coverage":                 "complete",
		"coverage_reasons":         []string{},
		"omitted_earlier_messages": int64(0),
		"messages":                 entries,
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	return string(b)
}

// insertSnapshotBoundResult inserts an analysis_snapshots row with the given
// manifest and a conversation_evaluation job_results row linked to it —
// exactly the shape ListResults/ExportResults read.
func (f *sourceIntegrityFixture) insertSnapshotBoundResult(t *testing.T, manifest string) {
	t.Helper()
	snapID := pkg.NewUUID()
	resultID := pkg.NewUUID()
	if err := db.DB.Exec(`INSERT INTO analysis_snapshots (id, tenant_id, job_run_id, conversation_id, schema_version, digest, coverage, coverage_reasons, message_count, manifest, created_at) VALUES (?, ?, ?, ?, 'ccma.snapshot.v1', 'deadbeef', 'complete', '[]', 1, ?, NOW())`,
		snapID, f.tenantID, f.runID, f.convID, manifest).Error; err != nil {
		t.Fatalf("fixture snapshot: %v", err)
	}
	if err := db.DB.Exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, analysis_snapshot_id, created_at) VALUES (?, ?, ?, ?, 'conversation_evaluation', 'PASS', '', 'Danh gia dat', '{"score":90}', 1, ?, NOW())`,
		resultID, f.runID, f.tenantID, f.convID, snapID).Error; err != nil {
		t.Fatalf("fixture result: %v", err)
	}
}

func (f *sourceIntegrityFixture) fetchOneRow(t *testing.T) resultRow {
	t.Helper()
	rows, err := f.loc().fetchRows(50, 0)
	if err != nil {
		t.Fatalf("fetchRows loi: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("muon 1 dong, nhan %d", len(rows))
	}
	return rows[0]
}

// TestSourceIntegrityUnchangedIsBoundCurrentnessUnverified is the central
// CCMAI-RUNTIME-004 case: a snapshot compared against exactly the source it
// was built from must never be reported as anything stronger than
// "unverified but locally unchanged." This field/behavior does not exist in
// pre-tranche source, so this test fails to compile against it.
func TestSourceIntegrityUnchangedIsBoundCurrentnessUnverified(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-1", "ext-1", "customer", "Khach", "Cho hoi gia san pham a", sentAt)
	f.insertSnapshotBoundResult(t, f.manifestJSON(t, snapshotMessageEntry("m-srcint-1", "ext-1", "customer", "Khach", sentAt, "Cho hoi gia san pham a")))

	row := f.fetchOneRow(t)
	if row.SourceIntegrityStatus != engine.SourceIntegrityBoundCurrentnessUnverified {
		t.Fatalf("status = %q, want %q", row.SourceIntegrityStatus, engine.SourceIntegrityBoundCurrentnessUnverified)
	}
}

// TestSourceIntegrityEditedMessageIsChangedSinceAnalysis is the key stale
// case the work order names explicitly: a message edited after analysis must
// not be reported as unchanged. This fails against pre-tranche behavior,
// which has no source_integrity_status field at all.
func TestSourceIntegrityEditedMessageIsChangedSinceAnalysis(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	originalContent := "Cho hoi gia san pham a"
	f.insertMessage(t, "m-srcint-2", "ext-2", "customer", "Khach", originalContent, sentAt)
	f.insertSnapshotBoundResult(t, f.manifestJSON(t, snapshotMessageEntry("m-srcint-2", "ext-2", "customer", "Khach", sentAt, originalContent)))

	// Source edited after analysis: same internal message ID, changed content.
	if err := db.DB.Exec(`UPDATE messages SET content = ? WHERE id = ?`, "Gia san pham da doi", "m-srcint-2").Error; err != nil {
		t.Fatalf("edit message: %v", err)
	}

	row := f.fetchOneRow(t)
	if row.SourceIntegrityStatus != engine.SourceIntegrityChangedSinceAnalysis {
		t.Fatalf("status = %q, want %q", row.SourceIntegrityStatus, engine.SourceIntegrityChangedSinceAnalysis)
	}
}

func TestSourceIntegrityMissingMessageIsChangedSinceAnalysis(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-3", "ext-3", "customer", "Khach", "Tin nhan se bi xoa", sentAt)
	f.insertSnapshotBoundResult(t, f.manifestJSON(t, snapshotMessageEntry("m-srcint-3", "ext-3", "customer", "Khach", sentAt, "Tin nhan se bi xoa")))

	if err := db.DB.Exec(`DELETE FROM messages WHERE id = ?`, "m-srcint-3").Error; err != nil {
		t.Fatalf("delete message: %v", err)
	}

	row := f.fetchOneRow(t)
	if row.SourceIntegrityStatus != engine.SourceIntegrityChangedSinceAnalysis {
		t.Fatalf("status = %q, want %q", row.SourceIntegrityStatus, engine.SourceIntegrityChangedSinceAnalysis)
	}
}

func TestSourceIntegrityAddedMessageIsChangedSinceAnalysis(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-4a", "ext-4a", "customer", "Khach", "Tin nhan duoc phan tich", sentAt)
	f.insertSnapshotBoundResult(t, f.manifestJSON(t, snapshotMessageEntry("m-srcint-4a", "ext-4a", "customer", "Khach", sentAt, "Tin nhan duoc phan tich")))

	// A new message arrives after analysis — the snapshot never saw it.
	f.insertMessage(t, "m-srcint-4b", "ext-4b", "agent", "NV", "Tin nhan moi sau khi phan tich", sentAt.Add(10*time.Minute))

	row := f.fetchOneRow(t)
	if row.SourceIntegrityStatus != engine.SourceIntegrityChangedSinceAnalysis {
		t.Fatalf("status = %q, want %q", row.SourceIntegrityStatus, engine.SourceIntegrityChangedSinceAnalysis)
	}
}

func TestSourceIntegrityLegacyResultHasNoSnapshotLink(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-5", "ext-5", "customer", "Khach", "Tin nhan cu", sentAt)
	resultID := pkg.NewUUID()
	if err := db.DB.Exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, created_at) VALUES (?, ?, ?, ?, 'conversation_evaluation', 'PASS', '', 'Danh gia cu', '{"score":80}', 1, NOW())`,
		resultID, f.runID, f.tenantID, f.convID).Error; err != nil {
		t.Fatalf("fixture legacy result: %v", err)
	}

	row := f.fetchOneRow(t)
	if row.SourceIntegrityStatus != engine.SourceIntegrityLegacyUnverified {
		t.Fatalf("status = %q, want %q", row.SourceIntegrityStatus, engine.SourceIntegrityLegacyUnverified)
	}
}

// TestSourceIntegrityBrokenSnapshotLinkIsVerificationUnavailable covers a
// result whose analysis_snapshot_id points at a row that no longer exists —
// this must never collapse into "unchanged."
func TestSourceIntegrityBrokenSnapshotLinkIsVerificationUnavailable(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-6", "ext-6", "customer", "Khach", "Tin nhan", sentAt)

	danglingSnapID := pkg.NewUUID() // deliberately never inserted into analysis_snapshots
	resultID := pkg.NewUUID()
	if err := db.DB.Exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, analysis_snapshot_id, created_at) VALUES (?, ?, ?, ?, 'conversation_evaluation', 'PASS', '', 'Danh gia', '{"score":85}', 1, ?, NOW())`,
		resultID, f.runID, f.tenantID, f.convID, danglingSnapID).Error; err != nil {
		t.Fatalf("fixture dangling-link result: %v", err)
	}

	row := f.fetchOneRow(t)
	if row.SourceIntegrityStatus != engine.SourceIntegrityVerificationUnavailable {
		t.Fatalf("status = %q, want %q", row.SourceIntegrityStatus, engine.SourceIntegrityVerificationUnavailable)
	}
}

// TestSourceIntegrityCorruptManifestIsVerificationUnavailable covers a
// snapshot row whose stored manifest is not valid ccma.snapshot.v1 JSON —
// this must never collapse into "unchanged" either.
func TestSourceIntegrityCorruptManifestIsVerificationUnavailable(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-7", "ext-7", "customer", "Khach", "Tin nhan", sentAt)
	f.insertSnapshotBoundResult(t, "khong-phai-json-hop-le")

	row := f.fetchOneRow(t)
	if row.SourceIntegrityStatus != engine.SourceIntegrityVerificationUnavailable {
		t.Fatalf("status = %q, want %q", row.SourceIntegrityStatus, engine.SourceIntegrityVerificationUnavailable)
	}
}

// TestSourceIntegrityLabelCoversAllFourStatuses proves the CSV/XLSX export
// label helper (sourceIntegrityLabel) has an explicit, non-empty mapping for
// every SPEC-defined status value — the same field the page and the export
// both read.
func TestSourceIntegrityLabelCoversAllFourStatuses(t *testing.T) {
	for _, status := range []string{
		engine.SourceIntegrityChangedSinceAnalysis,
		engine.SourceIntegrityBoundCurrentnessUnverified,
		engine.SourceIntegrityLegacyUnverified,
		engine.SourceIntegrityVerificationUnavailable,
	} {
		if label := sourceIntegrityLabel(status); label == "" {
			t.Errorf("sourceIntegrityLabel(%q) is empty", status)
		}
	}
}
