package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-006: job-specific result endpoints and exports carry R004's
// local source-integrity status. All rows are synthetic; no provider output.

type jobIntegrityFixture struct {
	tenantID, otherTenantID, channelID    string
	qcJobID, classJobID                   string
	oldRunID, newRunID, classRunID        string
	convMixed, convLegacy, convCorrupt    string
	convCross                             string
	mixedOldEval, mixedNewEval, mixedViol string
	legacyEval, corruptEval, crossEval    string
	classEval, classTag                   string
}

func (f *jobIntegrityFixture) exec(t *testing.T, sql string, args ...interface{}) {
	t.Helper()
	if err := db.DB.Exec(sql, args...).Error; err != nil {
		t.Fatalf("fixture: %v", err)
	}
}

// insertSnapshot stores a valid ccma.snapshot.v1 row (real digest, matching
// message count) for one text message and returns its ID.
func (f *jobIntegrityFixture) insertSnapshot(t *testing.T, tenantID, runID, convID, msgID, content string, sentAt time.Time) string {
	t.Helper()
	manifest, _ := json.Marshal(map[string]interface{}{
		"schema_version": "ccma.snapshot.v1", "tenant_id": tenantID, "conversation_id": convID,
		"coverage": "complete", "coverage_reasons": []string{}, "omitted_earlier_messages": 0,
		"messages": []interface{}{snapshotMessageEntry(msgID, "ext-"+msgID, "customer", "Khach", sentAt, content)},
	})
	sum := sha256.Sum256(manifest)
	id := pkg.NewUUID()
	f.exec(t, `INSERT INTO analysis_snapshots (id, tenant_id, job_run_id, conversation_id, schema_version, digest, coverage, coverage_reasons, message_count, manifest, created_at) VALUES (?, ?, ?, ?, 'ccma.snapshot.v1', ?, 'complete', '[]', 1, ?, NOW())`,
		id, tenantID, runID, convID, hex.EncodeToString(sum[:]), string(manifest))
	return id
}

func (f *jobIntegrityFixture) insertResult(t *testing.T, runID, convID, resultType, severity, rule string, snapshotID interface{}, at time.Time) string {
	t.Helper()
	id := pkg.NewUUID()
	f.exec(t, `INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, confidence_basis, analysis_snapshot_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, 'bang chung', '{"score":70}', NULL, 'unavailable', ?, ?)`,
		id, runID, f.tenantID, convID, resultType, severity, rule, snapshotID, at)
	return id
}

func setupJobIntegrityFixture(t *testing.T) *jobIntegrityFixture {
	t.Helper()
	connectChannelsTestDB(t)
	s := pkg.NewUUID()[:8]
	f := &jobIntegrityFixture{
		tenantID: "jobint-" + s, otherTenantID: "jobint-other-" + s, channelID: "ch-jobint-" + s,
		qcJobID: "job-qc-" + s, classJobID: "job-cls-" + s,
		oldRunID: "run-old-" + s, newRunID: "run-new-" + s, classRunID: "run-cls-" + s,
		convMixed: "conv-mixed-" + s, convLegacy: "conv-legacy-" + s, convCorrupt: "conv-corrupt-" + s, convCross: "conv-cross-" + s,
	}
	t.Cleanup(func() {
		for _, tenant := range []string{f.tenantID, f.otherTenantID} {
			for _, table := range []string{"job_results", "analysis_snapshots", "messages", "job_runs", "jobs", "conversations"} {
				db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenant)
			}
			db.DB.Exec("DELETE FROM channels WHERE tenant_id = ?", tenant)
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenant)
		}
	})

	for _, tenant := range []string{f.tenantID, f.otherTenantID} {
		f.exec(t, `INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Job Integrity', ?, '{}', NOW(), NOW())`, tenant, tenant)
	}
	f.exec(t, `INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh', 'fake', X'00', true, '{}', NOW(), NOW())`, f.channelID, f.tenantID)
	sentAt := time.Now().Add(-3 * time.Hour).UTC().Truncate(time.Second)
	for _, cv := range []string{f.convMixed, f.convLegacy, f.convCorrupt, f.convCross} {
		f.exec(t, `INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, 'Khach', NOW(), 1, '{}', NOW(), NOW())`, cv, f.tenantID, f.channelID, "ext-"+cv)
		f.exec(t, `INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, ?, 'customer', 'Khach', 'Noi dung hien tai', 'text', '[]', ?, NOW())`,
			"m-"+cv, f.tenantID, cv, "ext-m-"+cv, sentAt)
	}
	for _, j := range []struct{ id, kind string }{{f.qcJobID, "qc_analysis"}, {f.classJobID, "classification"}} {
		f.exec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, 'Job', ?, ?, '', '[]', 'manual', true, '[]', NOW(), NOW())`,
			j.id, f.tenantID, j.kind, `["`+f.channelID+`"]`)
	}
	for _, r := range []struct{ id, job string }{{f.oldRunID, f.qcJobID}, {f.newRunID, f.qcJobID}, {f.classRunID, f.classJobID}} {
		f.exec(t, `INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'success', '{}', NOW())`, r.id, r.job, f.tenantID)
	}

	older, newer := time.Now().Add(-2*time.Hour), time.Now().Add(-1*time.Hour)
	// Mixed conversation: the old run analyzed text that has since been edited
	// (changed); the new run analyzed the current text (locally unchanged).
	oldSnap := f.insertSnapshot(t, f.tenantID, f.oldRunID, f.convMixed, "m-"+f.convMixed, "Noi dung truoc khi sua", sentAt)
	newSnap := f.insertSnapshot(t, f.tenantID, f.newRunID, f.convMixed, "m-"+f.convMixed, "Noi dung hien tai", sentAt)
	f.mixedOldEval = f.insertResult(t, f.oldRunID, f.convMixed, "conversation_evaluation", "NGHIEM_TRONG", "", oldSnap, older)
	f.mixedNewEval = f.insertResult(t, f.newRunID, f.convMixed, "conversation_evaluation", "PASS", "", newSnap, newer)
	f.mixedViol = f.insertResult(t, f.newRunID, f.convMixed, "qc_violation", "CAN_CAI_THIEN", "Chao hoi", newSnap, newer)
	f.legacyEval = f.insertResult(t, f.newRunID, f.convLegacy, "conversation_evaluation", "PASS", "", nil, newer)

	corruptSnap := f.insertSnapshot(t, f.tenantID, f.newRunID, f.convCorrupt, "m-"+f.convCorrupt, "Noi dung hien tai", sentAt)
	f.exec(t, `UPDATE analysis_snapshots SET digest = ? WHERE id = ?`, strings.Repeat("0", 64), corruptSnap)
	f.corruptEval = f.insertResult(t, f.newRunID, f.convCorrupt, "conversation_evaluation", "PASS", "", corruptSnap, newer)

	// A well-formed snapshot owned by another tenant must never be used.
	crossSnap := f.insertSnapshot(t, f.otherTenantID, f.newRunID, f.convCross, "m-"+f.convCross, "Noi dung hien tai", sentAt)
	f.crossEval = f.insertResult(t, f.newRunID, f.convCross, "conversation_evaluation", "PASS", "", crossSnap, newer)

	classSnap := f.insertSnapshot(t, f.tenantID, f.classRunID, f.convMixed, "m-"+f.convMixed, "Noi dung hien tai", sentAt)
	f.classEval = f.insertResult(t, f.classRunID, f.convMixed, "conversation_evaluation", "PASS", "", classSnap, newer)
	f.classTag = f.insertResult(t, f.classRunID, f.convMixed, "classification_tag", "", "Hoi gia", classSnap, newer)
	return f
}

func callJobHandler(tenantID string, handler gin.HandlerFunc, params gin.Params, query string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenantID)
	c.Params = params
	c.Request = httptest.NewRequest("GET", "/api/v1/jobs/x/results"+query, nil)
	handler(c)
	return rec
}

func decodeStatuses(t *testing.T, rec *httptest.ResponseRecorder) map[string]map[string]interface{} {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
	}
	var items []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	out := map[string]map[string]interface{}{}
	for _, it := range items {
		out[it["id"].(string)] = it
	}
	return out
}

func TestJobResultEndpointsCarrySourceIntegrityStatus(t *testing.T) {
	f := setupJobIntegrityFixture(t)
	want := map[string]string{
		f.mixedOldEval: engine.SourceIntegrityChangedSinceAnalysis,
		f.mixedNewEval: engine.SourceIntegrityBoundCurrentnessUnverified,
		f.mixedViol:    engine.SourceIntegrityBoundCurrentnessUnverified,
		f.legacyEval:   engine.SourceIntegrityLegacyUnverified,
		f.corruptEval:  engine.SourceIntegrityVerificationUnavailable,
		f.crossEval:    engine.SourceIntegrityVerificationUnavailable,
	}
	check := func(t *testing.T, items map[string]map[string]interface{}, ids []string) {
		t.Helper()
		if len(items) != len(ids) {
			t.Fatalf("got %d results, want %d", len(items), len(ids))
		}
		for _, id := range ids {
			it, ok := items[id]
			if !ok {
				t.Fatalf("result %s missing", id)
			}
			if it["source_integrity_status"] != want[id] {
				t.Fatalf("result %s status = %v, want %s", id, it["source_integrity_status"], want[id])
			}
			// R005 confidence semantics and existing fields stay intact.
			if v, present := it["confidence"]; !present || v != nil || it["confidence_basis"] != "unavailable" {
				t.Fatalf("result %s confidence fields changed: %v/%v", id, it["confidence"], it["confidence_basis"])
			}
			if it["evidence"] != "bang chung" || it["evidence_status"] == nil {
				t.Fatalf("result %s lost existing fields: %v", id, it)
			}
		}
	}

	t.Run("ListJobResults", func(t *testing.T) {
		items := decodeStatuses(t, callJobHandler(f.tenantID, ListJobResults,
			gin.Params{{Key: "jobId", Value: f.qcJobID}, {Key: "runId", Value: f.newRunID}}, ""))
		check(t, items, []string{f.mixedNewEval, f.mixedViol, f.legacyEval, f.corruptEval, f.crossEval})
	})
	t.Run("ListAllJobResults", func(t *testing.T) {
		items := decodeStatuses(t, callJobHandler(f.tenantID, ListAllJobResults, gin.Params{{Key: "jobId", Value: f.qcJobID}}, ""))
		check(t, items, []string{f.mixedOldEval, f.mixedNewEval, f.mixedViol, f.legacyEval, f.corruptEval, f.crossEval})
	})
	t.Run("other tenant sees nothing", func(t *testing.T) {
		all := decodeStatuses(t, callJobHandler(f.otherTenantID, ListAllJobResults, gin.Params{{Key: "jobId", Value: f.qcJobID}}, ""))
		run := decodeStatuses(t, callJobHandler(f.otherTenantID, ListJobResults,
			gin.Params{{Key: "jobId", Value: f.qcJobID}, {Key: "runId", Value: f.newRunID}}, ""))
		if len(all) != 0 || len(run) != 0 {
			t.Fatalf("other tenant saw %d/%d results", len(all), len(run))
		}
	})
}

// exportTable returns header + data rows for either format.
func exportTable(t *testing.T, rec *httptest.ResponseRecorder, format string) [][]string {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("%s export status %d, body %s", format, rec.Code, rec.Body.String())
	}
	if format == "xlsx" {
		wb, err := excelize.OpenReader(bytes.NewReader(rec.Body.Bytes()))
		if err != nil {
			t.Fatalf("open xlsx: %v", err)
		}
		rows, err := wb.GetRows("Results")
		if err != nil {
			t.Fatalf("xlsx rows: %v", err)
		}
		return rows
	}
	rows, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(rec.Body.Bytes(), []byte("\xEF\xBB\xBF")))).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	return rows
}

// rowFor finds the exported row whose detail column mentions resultID.
func rowFor(t *testing.T, rows [][]string, resultID string) []string {
	t.Helper()
	for _, r := range rows[1:] {
		if strings.Contains(r[len(r)-1], resultID) {
			return r
		}
	}
	t.Fatalf("no exported row traces result %s", resultID)
	return nil
}

func TestExportJobResultsShowsDistinctGroupStatuses(t *testing.T) {
	f := setupJobIntegrityFixture(t)
	label := sourceIntegrityLabel
	for _, format := range []string{"csv", "xlsx"} {
		t.Run("qc_"+format, func(t *testing.T) {
			rows := exportTable(t, callJobHandler(f.tenantID, ExportJobResults, gin.Params{{Key: "jobId", Value: f.qcJobID}}, "?format="+format), format)
			wantHeader := []string{"Tên", "Ngày phát sinh chat", "Ngày đánh giá", "Kết quả đánh giá chi tiết", "Đánh giá", "Điểm", "Vấn đề", exportIntegrityHeader, exportIntegrityDetailHeader}
			if strings.Join(rows[0], "|") != strings.Join(wantHeader, "|") {
				t.Fatalf("header = %q", rows[0])
			}
			if len(rows) != 5 {
				t.Fatalf("got %d data rows, want 4 conversations", len(rows)-1)
			}

			mixed := rowFor(t, rows, f.mixedOldEval)
			wantMixed := label(engine.SourceIntegrityChangedSinceAnalysis) + "; " + label(engine.SourceIntegrityBoundCurrentnessUnverified)
			if mixed[7] != wantMixed {
				t.Fatalf("mixed group status = %q, want %q (changed must lead, never collapsed)", mixed[7], wantMixed)
			}
			for _, id := range []string{f.mixedOldEval, f.mixedNewEval, f.mixedViol} {
				if !strings.Contains(mixed[8], id+": ") {
					t.Fatalf("mixed detail missing %s: %q", id, mixed[8])
				}
			}
			if !strings.Contains(mixed[8], f.mixedOldEval+": "+label(engine.SourceIntegrityChangedSinceAnalysis)) {
				t.Fatalf("mixed detail does not attribute the change: %q", mixed[8])
			}
			if mixed[6] != "Chao hoi: bang chung" || mixed[5] != "70" {
				t.Fatalf("existing columns changed: %q", mixed)
			}

			for id, status := range map[string]string{
				f.legacyEval:  engine.SourceIntegrityLegacyUnverified,
				f.corruptEval: engine.SourceIntegrityVerificationUnavailable,
				f.crossEval:   engine.SourceIntegrityVerificationUnavailable,
			} {
				if r := rowFor(t, rows, id); r[7] != label(status) {
					t.Fatalf("result %s status cell = %q, want %q", id, r[7], label(status))
				}
			}
		})
		t.Run("classification_"+format, func(t *testing.T) {
			rows := exportTable(t, callJobHandler(f.tenantID, ExportJobResults, gin.Params{{Key: "jobId", Value: f.classJobID}}, "?format="+format), format)
			wantHeader := []string{"Tên", "Ngày phát sinh chat", "Ngày đánh giá", "Loại", "Vấn đề", exportChatHeader, exportIntegrityHeader, exportIntegrityDetailHeader}
			if strings.Join(rows[0], "|") != strings.Join(wantHeader, "|") {
				t.Fatalf("header = %q", rows[0])
			}
			if len(rows) != 2 {
				t.Fatalf("got %d data rows, want 1", len(rows)-1)
			}
			r := rowFor(t, rows, f.classTag)
			if r[3] != "Hoi gia" || r[5] != "[Khach] Noi dung hien tai" {
				t.Fatalf("existing classification columns changed: %q", r)
			}
			if r[6] != label(engine.SourceIntegrityBoundCurrentnessUnverified) || !strings.Contains(r[7], f.classEval+": ") {
				t.Fatalf("classification status = %q / %q", r[6], r[7])
			}
		})
	}
	t.Run("other tenant export is empty", func(t *testing.T) {
		rows := exportTable(t, callJobHandler(f.otherTenantID, ExportJobResults, gin.Params{{Key: "jobId", Value: f.qcJobID}}, "?format=csv"), "csv")
		if len(rows) != 1 {
			t.Fatalf("other tenant exported %d rows", len(rows)-1)
		}
	})
}

// forceTableUnreadable renames a table so its SELECTs fail (MySQL triggers
// cannot intercept a SELECT). Call after the fixture so the LIFO cleanup
// restores it before the fixture's DELETEs run.
func forceTableUnreadable(t *testing.T, table string) {
	t.Helper()
	if err := db.DB.Exec("RENAME TABLE " + table + " TO " + table + "_forced_failure").Error; err != nil {
		t.Fatalf("rename %s: %v", table, err)
	}
	t.Cleanup(func() {
		if err := db.DB.Exec("RENAME TABLE " + table + "_forced_failure TO " + table).Error; err != nil {
			t.Errorf("restore %s: %v", table, err)
		}
	})
}

func TestJobResultQueryFailuresAreObservable(t *testing.T) {
	for _, table := range []string{"analysis_snapshots", "messages"} {
		t.Run(table, func(t *testing.T) {
			f := setupJobIntegrityFixture(t)
			forceTableUnreadable(t, table)

			calls := map[string]*httptest.ResponseRecorder{
				"ListJobResults":    callJobHandler(f.tenantID, ListJobResults, gin.Params{{Key: "jobId", Value: f.qcJobID}, {Key: "runId", Value: f.newRunID}}, ""),
				"ListAllJobResults": callJobHandler(f.tenantID, ListAllJobResults, gin.Params{{Key: "jobId", Value: f.qcJobID}}, ""),
			}
			for _, format := range []string{"csv", "xlsx"} {
				calls["qc export "+format] = callJobHandler(f.tenantID, ExportJobResults, gin.Params{{Key: "jobId", Value: f.qcJobID}}, "?format="+format)
				calls["classification export "+format] = callJobHandler(f.tenantID, ExportJobResults, gin.Params{{Key: "jobId", Value: f.classJobID}}, "?format="+format)
			}
			for name, rec := range calls {
				body := rec.Body.Bytes()
				if rec.Code != http.StatusInternalServerError {
					t.Fatalf("%s: status %d, want 500, body %q", name, rec.Code, body)
				}
				if cd := rec.Header().Get("Content-Disposition"); cd != "" {
					t.Fatalf("%s: download header %q on failure", name, cd)
				}
				if bytes.HasPrefix(body, []byte("\xEF\xBB\xBF")) || bytes.HasPrefix(body, []byte("PK")) {
					t.Fatalf("%s: partial file body %q", name, body)
				}
				var resp struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(body, &resp); err != nil || resp.Error != "query_failed" {
					t.Fatalf("%s: body %q, want {\"error\":\"query_failed\"}", name, body)
				}
			}
		})
	}
}

func TestChunkStringsCoversEveryItemOnce(t *testing.T) {
	for _, n := range []int{0, 1, sourceIntegrityBatchSize, sourceIntegrityBatchSize + 1, 2*sourceIntegrityBatchSize + 3} {
		items := make([]string, n)
		for i := range items {
			items[i] = pkg.NewUUID()
		}
		var seen []string
		for _, chunk := range chunkStrings(items, sourceIntegrityBatchSize) {
			if len(chunk) == 0 || len(chunk) > sourceIntegrityBatchSize {
				t.Fatalf("n=%d: chunk of size %d", n, len(chunk))
			}
			seen = append(seen, chunk...)
		}
		if strings.Join(seen, ",") != strings.Join(items, ",") {
			t.Fatalf("n=%d: chunks lost or reordered items", n)
		}
	}
}

func TestDistinctSourceIntegrityLabelsLeadsWithMostConcerning(t *testing.T) {
	got := distinctSourceIntegrityLabels([]string{
		engine.SourceIntegrityBoundCurrentnessUnverified, engine.SourceIntegrityLegacyUnverified,
		engine.SourceIntegrityBoundCurrentnessUnverified, engine.SourceIntegrityChangedSinceAnalysis,
		engine.SourceIntegrityVerificationUnavailable,
	})
	want := strings.Join([]string{
		sourceIntegrityLabel(engine.SourceIntegrityChangedSinceAnalysis),
		sourceIntegrityLabel(engine.SourceIntegrityVerificationUnavailable),
		sourceIntegrityLabel(engine.SourceIntegrityLegacyUnverified),
		sourceIntegrityLabel(engine.SourceIntegrityBoundCurrentnessUnverified),
	}, "; ")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
