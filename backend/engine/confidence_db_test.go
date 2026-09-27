package engine

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/pkg"
)

// CCMAI-RUNTIME-005: result confidence truth. Payloads below are synthetic
// fixtures fed straight to saveResults; they prove persistence semantics only,
// not model quality or confidence calibration.

type storedConfidence struct {
	value sql.NullFloat64
	basis sql.NullString
}

func rawConfidence(t *testing.T, resultID string) storedConfidence {
	t.Helper()
	var s storedConfidence
	if err := db.DB.Raw("SELECT confidence, confidence_basis FROM job_results WHERE id = ?", resultID).
		Row().Scan(&s.value, &s.basis); err != nil {
		t.Fatalf("read stored confidence of %s: %v", resultID, err)
	}
	return s
}

// saveForFirstConversation runs saveResults for the fixture's first
// conversation under a fresh job run and returns that run's ID.
func saveForFirstConversation(t *testing.T, f *snapshotDBFixture, jobType string, payload func(ref map[string]interface{}) []byte) (string, error) {
	t.Helper()
	var conv models.Conversation
	if err := db.DB.First(&conv, "id = ?", f.convIDs[0]).Error; err != nil {
		t.Fatal(err)
	}
	snap, err := loadConversationSnapshot(conv, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	ref := (&refProvider{}).refFor([]string{snap.Transcript}, 0)
	runID := "run-conf-" + pkg.NewUUID()[:8]
	if err := db.DB.Exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'running', '{}', NOW())`,
		runID, f.jobID, f.tenantID).Error; err != nil {
		t.Fatalf("fixture job_run: %v", err)
	}
	_, _, err = NewAnalyzer(&config.Config{}).saveResults(runID, snap, jobType, string(payload(ref)))
	return runID, err
}

func classificationPayload(tagConfidence interface{}) func(ref map[string]interface{}) []byte {
	return func(ref map[string]interface{}) []byte {
		tag := map[string]interface{}{
			"rule_name": "Khieu nai giao hang", "evidence": ref["quote"],
			"evidence_refs": []interface{}{ref}, "explanation": "Khach phan nan.",
		}
		if tagConfidence != nil {
			tag["confidence"] = tagConfidence
		}
		b, _ := json.Marshal(map[string]interface{}{"summary": "Khach khieu nai.", "tags": []interface{}{tag}})
		return b
	}
}

func assertUnavailable(t *testing.T, r models.JobResult) {
	t.Helper()
	stored := rawConfidence(t, r.ID)
	if stored.value.Valid || !stored.basis.Valid || stored.basis.String != models.ConfidenceBasisUnavailable {
		t.Fatalf("%s %s stored confidence=%v basis=%v, want NULL/unavailable", r.ResultType, r.ID, stored.value, stored.basis)
	}
	if r.ReportedConfidence != nil || r.ReportedConfidenceBasis != models.ConfidenceBasisUnavailable {
		t.Fatalf("%s %s reported %v/%q, want nil/unavailable", r.ResultType, r.ID, r.ReportedConfidence, r.ReportedConfidenceBasis)
	}
}

// TestQCResultsStoreNoFabricatedConfidence is the central regression: before
// this tranche every QC evaluation and violation was stored with 1.0.
func TestQCResultsStoreNoFabricatedConfidence(t *testing.T) {
	f := setupSnapshotDBFixture(t, false)
	runID, err := saveForFirstConversation(t, f, "qc_analysis", func(ref map[string]interface{}) []byte { return qcFailWithRef(ref) })
	if err != nil {
		t.Fatalf("saveResults: %v", err)
	}
	results := resultsFor(t, runID)
	if len(results) != 2 {
		t.Fatalf("got %d results, want evaluation + violation", len(results))
	}
	for _, r := range results {
		assertUnavailable(t, r)
		if r.AnalysisSnapshotID == nil || r.EvidenceStatus != "snapshot_bound" {
			t.Fatalf("%s lost its snapshot binding", r.ResultType)
		}
	}
}

func TestClassificationTagKeepsOnlyModelReportedUncalibratedValue(t *testing.T) {
	f := setupSnapshotDBFixture(t, false)
	runID, err := saveForFirstConversation(t, f, "classification", classificationPayload(0.7))
	if err != nil {
		t.Fatalf("saveResults: %v", err)
	}
	var sawTag, sawEval bool
	for _, r := range resultsFor(t, runID) {
		switch r.ResultType {
		case "classification_tag":
			sawTag = true
			stored := rawConfidence(t, r.ID)
			if !stored.value.Valid || stored.value.Float64 != 0.7 || stored.basis.String != models.ConfidenceBasisModelReportedUncalibrated {
				t.Fatalf("tag stored %v/%v, want 0.7/model_reported_uncalibrated", stored.value, stored.basis)
			}
			if r.ReportedConfidence == nil || *r.ReportedConfidence != 0.7 || r.ReportedConfidenceBasis != models.ConfidenceBasisModelReportedUncalibrated {
				t.Fatalf("tag reported %v/%q", r.ReportedConfidence, r.ReportedConfidenceBasis)
			}
		case "conversation_evaluation":
			sawEval = true
			assertUnavailable(t, r)
		}
	}
	if !sawTag || !sawEval {
		t.Fatalf("missing rows: tag=%v eval=%v", sawTag, sawEval)
	}
}

func TestClassificationSkipEvaluationHasNoConfidence(t *testing.T) {
	f := setupSnapshotDBFixture(t, false)
	runID, err := saveForFirstConversation(t, f, "classification", func(map[string]interface{}) []byte {
		return []byte(`{"summary":"Khong khop nhan nao.","tags":[]}`)
	})
	if err != nil {
		t.Fatalf("saveResults: %v", err)
	}
	results := resultsFor(t, runID)
	if len(results) != 1 || results[0].Severity != "SKIP" {
		t.Fatalf("got %+v, want one SKIP evaluation", results)
	}
	assertUnavailable(t, results[0])
}

// TestInvalidModelTagConfidenceIsRejected proves the existing validation still
// refuses an out-of-range or missing tag confidence, writing nothing at all.
func TestInvalidModelTagConfidenceIsRejected(t *testing.T) {
	for name, value := range map[string]interface{}{"above one": 1.5, "negative": -0.1, "missing": nil} {
		t.Run(name, func(t *testing.T) {
			f := setupSnapshotDBFixture(t, false)
			runID, err := saveForFirstConversation(t, f, "classification", classificationPayload(value))
			if err == nil {
				t.Fatal("invalid tag confidence was accepted")
			}
			if n := len(resultsFor(t, runID)); n != 0 {
				t.Fatalf("%d results saved from an invalid response", n)
			}
			if n := len(snapshotsFor(t, runID)); n != 0 {
				t.Fatalf("%d snapshots saved from an invalid response", n)
			}
		})
	}
}

// TestLegacyAndMislabeledConfidenceIsNotExposed covers rows written before
// confidence_basis existed (NULL basis, old numbers kept) and rows whose basis
// or type must never expose a number. A Save after reading must not rewrite
// the historical stored value.
func TestLegacyAndMislabeledConfidenceIsNotExposed(t *testing.T) {
	connectTestDB(t)
	tenantID := "conf-legacy-" + pkg.NewUUID()[:8]
	t.Cleanup(func() { db.DB.Exec("DELETE FROM job_results WHERE tenant_id = ?", tenantID) })

	rows := []struct {
		resultType string
		value      interface{}
		basis      interface{}
	}{
		{"conversation_evaluation", 1.0, nil},
		{"qc_violation", 1.0, nil},
		{"classification_tag", 0.73, nil},
		{"qc_violation", 0.9, models.ConfidenceBasisModelReportedUncalibrated},
		{"classification_tag", 1.5, models.ConfidenceBasisModelReportedUncalibrated},
		{"classification_tag", 0.8, "calibrated"},
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = pkg.NewUUID()
		if err := db.DB.Exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, confidence_basis, created_at) VALUES (?, 'legacy-run', ?, 'legacy-conv', ?, 'NGHIEM_TRONG', 'r', 'e', '{}', ?, ?, NOW())`,
			ids[i], tenantID, r.resultType, r.value, r.basis).Error; err != nil {
			t.Fatalf("fixture row %d: %v", i, err)
		}
	}

	for i, id := range ids {
		var r models.JobResult
		if err := db.DB.First(&r, "id = ?", id).Error; err != nil {
			t.Fatal(err)
		}
		if r.ReportedConfidence != nil || r.ReportedConfidenceBasis != models.ConfidenceBasisUnavailable {
			t.Fatalf("row %d (%s) exposed %v/%q, want nil/unavailable", i, rows[i].resultType, r.ReportedConfidence, r.ReportedConfidenceBasis)
		}
		if err := db.DB.Save(&r).Error; err != nil {
			t.Fatalf("save row %d: %v", i, err)
		}
		stored := rawConfidence(t, id)
		if !stored.value.Valid || stored.value.Float64 != rows[i].value.(float64) {
			t.Fatalf("row %d stored value rewritten to %v", i, stored.value)
		}
		if (rows[i].basis == nil) == stored.basis.Valid {
			t.Fatalf("row %d stored basis changed to %v", i, stored.basis)
		}
	}
}
