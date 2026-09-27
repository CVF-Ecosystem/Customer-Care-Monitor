package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/pkg"
)

// CCMAI-RUNTIME-005: both job-specific result endpoints must serialize
// confidence as number|null plus confidence_basis, exposing a number only for
// a classification tag whose stored basis is model_reported_uncalibrated.

type confidenceCase struct {
	id, resultType string
	value, basis   interface{}
	wantConfidence interface{} // nil or float64
	wantBasis      string
}

func TestJobResultEndpointsSerializeConfidenceTruth(t *testing.T) {
	connectChannelsTestDB(t)
	suffix := pkg.NewUUID()[:8]
	tenantID, channelID, convID := "confapi-"+suffix, "ch-confapi-"+suffix, "conv-confapi-"+suffix
	jobID, runID := "job-confapi-"+suffix, "run-confapi-"+suffix

	exec := func(sql string, args ...interface{}) {
		t.Helper()
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, table := range []string{"job_results", "job_runs", "jobs", "conversations"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenantID)
		}
		db.DB.Exec("DELETE FROM channels WHERE id = ?", channelID)
		db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
	})
	exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Confidence API', ?, '{}', NOW(), NOW())`, tenantID, tenantID)
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh', 'fake', X'00', true, '{}', NOW(), NOW())`, channelID, tenantID)
	exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, 'ext-confapi', 'Khach', NOW(), 1, '{}', NOW(), NOW())`, convID, tenantID, channelID)
	exec(`INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, 'Job', 'classification', ?, '', '[]', 'manual', true, '[]', NOW(), NOW())`, jobID, tenantID, `["`+channelID+`"]`)
	exec(`INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'success', '{}', NOW())`, runID, jobID, tenantID)

	cases := []confidenceCase{
		{pkg.NewUUID(), "conversation_evaluation", 1.0, nil, nil, models.ConfidenceBasisUnavailable}, // legacy QC placeholder
		{pkg.NewUUID(), "qc_violation", 1.0, nil, nil, models.ConfidenceBasisUnavailable},            // legacy QC placeholder
		{pkg.NewUUID(), "classification_tag", 0.73, nil, nil, models.ConfidenceBasisUnavailable},     // legacy tag, provenance unknown
		{pkg.NewUUID(), "conversation_evaluation", nil, "unavailable", nil, models.ConfidenceBasisUnavailable},
		{pkg.NewUUID(), "classification_tag", 0.64, "model_reported_uncalibrated", 0.64, models.ConfidenceBasisModelReportedUncalibrated},
	}
	for _, tc := range cases {
		exec(`INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, confidence_basis, created_at) VALUES (?, ?, ?, ?, ?, 'PASS', 'r', 'e', '{}', ?, ?, NOW())`,
			tc.id, runID, tenantID, convID, tc.resultType, tc.value, tc.basis)
	}

	gin.SetMode(gin.TestMode)
	endpoints := map[string]struct {
		handler gin.HandlerFunc
		params  gin.Params
	}{
		"ListJobResults":    {ListJobResults, gin.Params{{Key: "jobId", Value: jobID}, {Key: "runId", Value: runID}}},
		"ListAllJobResults": {ListAllJobResults, gin.Params{{Key: "jobId", Value: jobID}}},
	}
	for name, ep := range endpoints {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Set("tenant_id", tenantID)
			c.Params = ep.params
			c.Request = httptest.NewRequest("GET", "/api/v1/jobs/"+jobID+"/results", nil)
			ep.handler(c)
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
			}

			var items []map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
				t.Fatalf("decode: %v", err)
			}
			byID := map[string]map[string]interface{}{}
			for _, it := range items {
				byID[it["id"].(string)] = it
			}
			if len(byID) != len(cases) {
				t.Fatalf("got %d results, want %d", len(byID), len(cases))
			}
			for _, tc := range cases {
				it := byID[tc.id]
				got, present := it["confidence"]
				if !present {
					t.Fatalf("%s: confidence key omitted, want explicit null or number", tc.resultType)
				}
				if got != tc.wantConfidence {
					t.Fatalf("%s (stored %v/%v): confidence = %v, want %v", tc.resultType, tc.value, tc.basis, got, tc.wantConfidence)
				}
				if it["confidence_basis"] != tc.wantBasis {
					t.Fatalf("%s: confidence_basis = %v, want %s", tc.resultType, it["confidence_basis"], tc.wantBasis)
				}
			}
		})
	}
}
