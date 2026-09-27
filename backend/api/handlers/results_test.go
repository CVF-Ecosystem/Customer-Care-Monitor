package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

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
// exactly the shape ListResults/ExportResults read. The digest is always the
// real SHA-256 of the manifest bytes and message_count is derived from the
// manifest's own "messages" array (R004-R1: VerifySnapshotProvenance now
// rejects a mismatched digest/count, so a fixture using a placeholder like
// the prior hardcoded "deadbeef" digest would falsely report every
// otherwise-valid fixture as verification_unavailable). It returns the
// inserted snapshot ID so a test can deliberately corrupt one field after
// insertion.
func (f *sourceIntegrityFixture) insertSnapshotBoundResult(t *testing.T, manifest string) string {
	t.Helper()
	return f.insertSnapshotBoundResultWithLink(t, manifest, f.convID, f.runID)
}

// insertSnapshotBoundResultWithLink is insertSnapshotBoundResult but lets a
// test record the snapshot under a different conversation/job-run than the
// linking result — the R004-R1 wrong-conversation/wrong-run provenance
// regressions.
func (f *sourceIntegrityFixture) insertSnapshotBoundResultWithLink(t *testing.T, manifest, snapshotConvID, snapshotRunID string) string {
	t.Helper()
	sum := sha256.Sum256([]byte(manifest))
	digest := hex.EncodeToString(sum[:])
	messageCount := manifestMessageCount(manifest)

	snapID := pkg.NewUUID()
	resultID := pkg.NewUUID()
	if err := db.DB.Exec(`INSERT INTO analysis_snapshots (id, tenant_id, job_run_id, conversation_id, schema_version, digest, coverage, coverage_reasons, message_count, manifest, created_at) VALUES (?, ?, ?, ?, 'ccma.snapshot.v1', ?, 'complete', '[]', ?, ?, NOW())`,
		snapID, f.tenantID, snapshotRunID, snapshotConvID, digest, messageCount, manifest).Error; err != nil {
		t.Fatalf("fixture snapshot: %v", err)
	}
	if err := db.DB.Exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, analysis_snapshot_id, created_at) VALUES (?, ?, ?, ?, 'conversation_evaluation', 'PASS', '', 'Danh gia dat', '{"score":90}', 1, ?, NOW())`,
		resultID, f.runID, f.tenantID, f.convID, snapID).Error; err != nil {
		t.Fatalf("fixture result: %v", err)
	}
	return snapID
}

// manifestMessageCount best-effort parses a manifest's "messages" array
// length; an unparsable manifest (the deliberately-corrupt-manifest test
// case) yields 0, which is fine since VerifySnapshotProvenance rejects that
// manifest on the JSON-unmarshal step regardless of the recorded count.
func manifestMessageCount(manifest string) int {
	var m struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal([]byte(manifest), &m); err != nil {
		return 0
	}
	return len(m.Messages)
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

// --- R004-R1 repair: linked-snapshot provenance regressions ---
//
// engine.CompareSnapshotToCurrentMessages's own unit tests (source_integrity_test.go)
// prove VerifySnapshotProvenance's field-level rejections directly. These
// tests instead prove results.go's attachSourceIntegrity wiring actually
// calls it before ever reporting bound_currentness_unverified.

// TestSourceIntegrityCorruptDigestIsVerificationUnavailable covers a
// snapshot row whose stored digest does not match its own manifest bytes
// (e.g. a hand-edited manifest column) — this must never collapse into
// "unchanged."
func TestSourceIntegrityCorruptDigestIsVerificationUnavailable(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-8", "ext-8", "customer", "Khach", "Tin nhan", sentAt)
	snapID := f.insertSnapshotBoundResult(t, f.manifestJSON(t, snapshotMessageEntry("m-srcint-8", "ext-8", "customer", "Khach", sentAt, "Tin nhan")))

	corruptDigest := strings.Repeat("deadbeef", 8) // 64 hex chars, deliberately wrong
	if err := db.DB.Exec(`UPDATE analysis_snapshots SET digest = ? WHERE id = ?`, corruptDigest, snapID).Error; err != nil {
		t.Fatalf("corrupt digest: %v", err)
	}

	row := f.fetchOneRow(t)
	if row.SourceIntegrityStatus != engine.SourceIntegrityVerificationUnavailable {
		t.Fatalf("status = %q, want %q", row.SourceIntegrityStatus, engine.SourceIntegrityVerificationUnavailable)
	}
}

// TestSourceIntegrityWrongConversationLinkIsVerificationUnavailable covers a
// snapshot recorded (correct digest and message count) for a different
// conversation than the result linking to it — a mislinked row, not
// necessarily malicious, must never be presented as locally matching.
func TestSourceIntegrityWrongConversationLinkIsVerificationUnavailable(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-9", "ext-9", "customer", "Khach", "Tin nhan", sentAt)
	manifest := f.manifestJSON(t, snapshotMessageEntry("m-srcint-9", "ext-9", "customer", "Khach", sentAt, "Tin nhan"))
	f.insertSnapshotBoundResultWithLink(t, manifest, "conv-khac-"+pkg.NewUUID()[:8], f.runID)

	row := f.fetchOneRow(t)
	if row.SourceIntegrityStatus != engine.SourceIntegrityVerificationUnavailable {
		t.Fatalf("status = %q, want %q", row.SourceIntegrityStatus, engine.SourceIntegrityVerificationUnavailable)
	}
}

// TestSourceIntegrityWrongJobRunLinkIsVerificationUnavailable is the same
// regression as above for the job-run identity field.
func TestSourceIntegrityWrongJobRunLinkIsVerificationUnavailable(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-9b", "ext-9b", "customer", "Khach", "Tin nhan", sentAt)
	manifest := f.manifestJSON(t, snapshotMessageEntry("m-srcint-9b", "ext-9b", "customer", "Khach", sentAt, "Tin nhan"))
	f.insertSnapshotBoundResultWithLink(t, manifest, f.convID, "run-khac-"+pkg.NewUUID()[:8])

	row := f.fetchOneRow(t)
	if row.SourceIntegrityStatus != engine.SourceIntegrityVerificationUnavailable {
		t.Fatalf("status = %q, want %q", row.SourceIntegrityStatus, engine.SourceIntegrityVerificationUnavailable)
	}
}

// TestSourceIntegrityCrossTenantSnapshotLinkIsVerificationUnavailable is the
// R004-R1 cross-tenant regression: even though this fixture inserts a
// snapshot with a well-formed digest/manifest and a job_results row of this
// test's own tenant links to it by ID, the snapshot itself belongs to a
// different tenant. attachSourceIntegrity's batched snapshot lookup stays
// tenant-scoped, so it never loads that row, and the link is reported as
// verification_unavailable rather than silently comparing across tenants.
func TestSourceIntegrityCrossTenantSnapshotLinkIsVerificationUnavailable(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-10", "ext-10", "customer", "Khach", "Tin nhan", sentAt)

	otherTenantID := "other-tenant-" + pkg.NewUUID()[:8]
	manifest := f.manifestJSON(t, snapshotMessageEntry("m-srcint-10", "ext-10", "customer", "Khach", sentAt, "Tin nhan"))
	sum := sha256.Sum256([]byte(manifest))
	digest := hex.EncodeToString(sum[:])
	snapID := pkg.NewUUID()
	if err := db.DB.Exec(`INSERT INTO analysis_snapshots (id, tenant_id, job_run_id, conversation_id, schema_version, digest, coverage, coverage_reasons, message_count, manifest, created_at) VALUES (?, ?, ?, ?, 'ccma.snapshot.v1', ?, 'complete', '[]', 1, ?, NOW())`,
		snapID, otherTenantID, f.runID, f.convID, digest, manifest).Error; err != nil {
		t.Fatalf("fixture cross-tenant snapshot: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec("DELETE FROM analysis_snapshots WHERE tenant_id = ?", otherTenantID) })

	resultID := pkg.NewUUID()
	if err := db.DB.Exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, analysis_snapshot_id, created_at) VALUES (?, ?, ?, ?, 'conversation_evaluation', 'PASS', '', 'Danh gia', '{"score":90}', 1, ?, NOW())`,
		resultID, f.runID, f.tenantID, f.convID, snapID).Error; err != nil {
		t.Fatalf("fixture cross-tenant-linked result: %v", err)
	}

	row := f.fetchOneRow(t)
	if row.SourceIntegrityStatus != engine.SourceIntegrityVerificationUnavailable {
		t.Fatalf("status = %q, want %q", row.SourceIntegrityStatus, engine.SourceIntegrityVerificationUnavailable)
	}
}

// --- R004-R3 repair: endpoint/export/isolation/error-path evidence ---

// TestListResultsResponseIncludesSourceIntegrityStatus exercises the actual
// ListResults handler (not just fetchRows) end to end, proving the page
// response carries the same source_integrity_status the earlier
// fetchRows-only tests checked directly.
func TestListResultsResponseIncludesSourceIntegrityStatus(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-11", "ext-11", "customer", "Khach", "Tin nhan goc", sentAt)
	f.insertSnapshotBoundResult(t, f.manifestJSON(t, snapshotMessageEntry("m-srcint-11", "ext-11", "customer", "Khach", sentAt, "Tin nhan goc")))
	if err := db.DB.Exec(`UPDATE messages SET content = ? WHERE id = ?`, "Tin nhan da doi", "m-srcint-11").Error; err != nil {
		t.Fatalf("edit message: %v", err)
	}

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", f.tenantID)
	c.Request = httptest.NewRequest("GET", "/api/v1/results?job_type=qc_analysis", nil)

	ListResults(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("ListResults status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []struct {
			SourceIntegrityStatus string `json:"source_integrity_status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].SourceIntegrityStatus != engine.SourceIntegrityChangedSinceAnalysis {
		t.Fatalf("items = %+v, want one item with status %q", resp.Items, engine.SourceIntegrityChangedSinceAnalysis)
	}
}

func callExportResults(t *testing.T, tenantID, format string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenantID)
	c.Request = httptest.NewRequest("GET", "/api/v1/results/export?job_type=qc_analysis&format="+format, nil)
	ExportResults(c)
	return rec
}

// csvLastColumn parses the exported CSV (BOM + header + data rows) and returns
// the last cell of every data row, so the status is checked in its own column
// rather than as a substring anywhere in the file.
func csvLastColumn(t *testing.T, body []byte) []string {
	t.Helper()
	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(body, []byte("\xEF\xBB\xBF"))))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(records) < 1 || records[0][len(records[0])-1] != "Tính toàn vẹn nguồn" {
		t.Fatalf("csv header = %+v, want last column %q", records, "Tính toàn vẹn nguồn")
	}
	out := make([]string, 0, len(records)-1)
	for _, rec := range records[1:] {
		out = append(out, rec[len(rec)-1])
	}
	return out
}

func xlsxLastColumn(t *testing.T, body []byte) []string {
	t.Helper()
	wb, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	sheetRows, err := wb.GetRows("Results")
	if err != nil {
		t.Fatalf("read xlsx rows: %v", err)
	}
	if len(sheetRows) < 1 || sheetRows[0][len(sheetRows[0])-1] != "Tính toàn vẹn nguồn" {
		t.Fatalf("xlsx header = %+v, want last column %q", sheetRows, "Tính toàn vẹn nguồn")
	}
	out := make([]string, 0, len(sheetRows)-1)
	for _, row := range sheetRows[1:] {
		out = append(out, row[len(row)-1])
	}
	return out
}

// TestExportResultsCSVAndXLSXIncludeSourceIntegrityColumn proves both export
// formats carry the same source-integrity label for the same underlying row,
// for a nonchanged source and for a source edited after analysis.
func TestExportResultsCSVAndXLSXIncludeSourceIntegrityColumn(t *testing.T) {
	cases := []struct {
		name       string
		editSource bool
		wantStatus string
	}{
		{"unchanged", false, engine.SourceIntegrityBoundCurrentnessUnverified},
		{"changed", true, engine.SourceIntegrityChangedSinceAnalysis},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := setupSourceIntegrityFixture(t)
			sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
			f.insertMessage(t, "m-srcint-12", "ext-12", "customer", "Khach", "Tin nhan goc", sentAt)
			f.insertSnapshotBoundResult(t, f.manifestJSON(t, snapshotMessageEntry("m-srcint-12", "ext-12", "customer", "Khach", sentAt, "Tin nhan goc")))
			if tc.editSource {
				if err := db.DB.Exec(`UPDATE messages SET content = ? WHERE id = ? AND tenant_id = ?`, "Tin nhan da doi sau danh gia", "m-srcint-12", f.tenantID).Error; err != nil {
					t.Fatalf("edit message: %v", err)
				}
			}
			wantLabel := sourceIntegrityLabel(tc.wantStatus)

			csvRec := callExportResults(t, f.tenantID, "csv")
			if csvRec.Code != http.StatusOK {
				t.Fatalf("csv export status = %d, body = %s", csvRec.Code, csvRec.Body.String())
			}
			if got := csvLastColumn(t, csvRec.Body.Bytes()); len(got) != 1 || got[0] != wantLabel {
				t.Fatalf("csv source-integrity column = %q, want [%q]", got, wantLabel)
			}

			xlsxRec := callExportResults(t, f.tenantID, "xlsx")
			if xlsxRec.Code != http.StatusOK {
				t.Fatalf("xlsx export status = %d, body = %s", xlsxRec.Code, xlsxRec.Body.String())
			}
			if got := xlsxLastColumn(t, xlsxRec.Body.Bytes()); len(got) != 1 || got[0] != wantLabel {
				t.Fatalf("xlsx source-integrity column = %q, want [%q]", got, wantLabel)
			}
		})
	}
}

// forceSnapshotQueryFailure renames analysis_snapshots so attachSourceIntegrity's
// batched snapshot SELECT fails. MySQL triggers cannot intercept a SELECT, so
// this disposable-MySQL-only rename is used instead; it must be called after
// setupSourceIntegrityFixture so the LIFO cleanup restores the table before the
// fixture's own DELETEs run, even if the test fails or panics.
func forceSnapshotQueryFailure(t *testing.T) {
	t.Helper()
	if err := db.DB.Exec("RENAME TABLE analysis_snapshots TO analysis_snapshots_forced_failure").Error; err != nil {
		t.Fatalf("rename table to force failure: %v", err)
	}
	t.Cleanup(func() {
		if err := db.DB.Exec("RENAME TABLE analysis_snapshots_forced_failure TO analysis_snapshots").Error; err != nil {
			t.Errorf("restore renamed table: %v", err)
		}
	})
}

// TestListResultsSnapshotBatchQueryFailureIsObservable proves
// attachSourceIntegrity's batched snapshot query failing the whole request
// is real, not just a documented intent: it forces that exact query to
// error and checks the handler returns a non-2xx response instead of a
// page that silently omits or mis-states the affected rows' status.
func TestListResultsSnapshotBatchQueryFailureIsObservable(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-13", "ext-13", "customer", "Khach", "Tin nhan", sentAt)
	f.insertSnapshotBoundResult(t, f.manifestJSON(t, snapshotMessageEntry("m-srcint-13", "ext-13", "customer", "Khach", sentAt, "Tin nhan")))
	forceSnapshotQueryFailure(t)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", f.tenantID)
	c.Request = httptest.NewRequest("GET", "/api/v1/results?job_type=qc_analysis", nil)

	ListResults(c)

	if rec.Code == http.StatusOK || rec.Code < 300 {
		t.Fatalf("ListResults status = %d, muon loi khi bang snapshot khong doc duoc (khong duoc thanh cong mot phan), body = %s", rec.Code, rec.Body.String())
	}
}

// TestExportResultsSnapshotBatchQueryFailureIsObservable is the export-side
// counterpart: with the batched snapshot query forced to fail, neither CSV nor
// XLSX may return a success status, a download header or any partial file.
func TestExportResultsSnapshotBatchQueryFailureIsObservable(t *testing.T) {
	f := setupSourceIntegrityFixture(t)
	sentAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	f.insertMessage(t, "m-srcint-14", "ext-14", "customer", "Khach", "Tin nhan", sentAt)
	f.insertSnapshotBoundResult(t, f.manifestJSON(t, snapshotMessageEntry("m-srcint-14", "ext-14", "customer", "Khach", sentAt, "Tin nhan")))
	forceSnapshotQueryFailure(t)

	for _, format := range []string{"csv", "xlsx"} {
		t.Run(format, func(t *testing.T) {
			rec := callExportResults(t, f.tenantID, format)
			body := rec.Body.Bytes()

			if rec.Code < 400 {
				t.Fatalf("%s export status = %d, want a 4xx/5xx error, body = %q", format, rec.Code, body)
			}
			if cd := rec.Header().Get("Content-Disposition"); cd != "" {
				t.Fatalf("%s export set download header %q on failure", format, cd)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("%s export content type = %q, want JSON error, not a file", format, ct)
			}
			if bytes.HasPrefix(body, []byte("\xEF\xBB\xBF")) || bytes.HasPrefix(body, []byte("PK")) {
				t.Fatalf("%s export body starts like a CSV/XLSX file on failure: %q", format, body)
			}
			var resp struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(body, &resp); err != nil || resp.Error != "query_failed" {
				t.Fatalf("%s export body = %q, want JSON {\"error\":\"query_failed\"} (unmarshal err: %v)", format, body, err)
			}
		})
	}
}
