package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-017: GET /dashboard additive qc_violation_count. The real
// handler runs against a disposable MySQL with synthetic rows only; no provider
// output and no governance claim.

const (
	dashQCWindowDay = "2026-03-10"
	dashQCType      = "qc_violation"
	dashEvalType    = "conversation_evaluation"
	dashTagType     = "classification_tag"
)

// CCMAI-RUNTIME-026: the window is the Vietnam day 2026-03-10, i.e. [2026-03-09T17:00:00Z,
// 2026-03-10T17:00:00Z); From is its first instant and To its last millisecond (the old UTC
// edges 00:00:00 / 23:59:59 with an inclusive BETWEEN no longer describe the contract).
var (
	dashQCFrom = time.Date(2026, 3, 9, 17, 0, 0, 0, time.UTC)
	dashQCTo   = time.Date(2026, 3, 10, 16, 59, 59, 999_000_000, time.UTC)
)

type dashQCFixture struct {
	tenantA, tenantB, tenantEmpty string
	convA, convB, convEmpty       string
	runA, runB, runEmpty          string
}

func (f *dashQCFixture) exec(t *testing.T, sql string, args ...interface{}) {
	t.Helper()
	if err := db.DB.Exec(sql, args...).Error; err != nil {
		t.Fatalf("fixture: %v", err)
	}
}

func (f *dashQCFixture) insertResult(t *testing.T, tenantID, runID, convID, resultType string, at time.Time) {
	t.Helper()
	f.exec(t, `INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, created_at) VALUES (?, ?, ?, ?, ?, 'CAN_CAI_THIEN', 'Chao hoi', 'bang chung', '{}', ?)`,
		pkg.NewUUID(), runID, tenantID, convID, resultType, at)
}

func setupDashQCFixture(t *testing.T) *dashQCFixture {
	t.Helper()
	connectChannelsTestDB(t)
	s := pkg.NewUUID()[:8]
	f := &dashQCFixture{
		tenantA: "dashqc-a-" + s, tenantB: "dashqc-b-" + s, tenantEmpty: "dashqc-e-" + s,
		convA: "conv-dashqc-a-" + s, convB: "conv-dashqc-b-" + s, convEmpty: "conv-dashqc-e-" + s,
		runA: "run-dashqc-a-" + s, runB: "run-dashqc-b-" + s, runEmpty: "run-dashqc-e-" + s,
	}
	tenants := []string{f.tenantA, f.tenantB, f.tenantEmpty}
	t.Cleanup(func() {
		for _, tenant := range tenants {
			for _, table := range []string{"job_results", "job_runs", "jobs", "conversations"} {
				db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenant)
			}
			db.DB.Exec("DELETE FROM channels WHERE tenant_id = ?", tenant)
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenant)
		}
	})
	for i, tenant := range tenants {
		conv := []string{f.convA, f.convB, f.convEmpty}[i]
		run := []string{f.runA, f.runB, f.runEmpty}[i]
		ch, job := "ch-"+tenant, "job-"+tenant
		f.exec(t, `INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Dash QC', ?, '{}', NOW(), NOW())`, tenant, tenant)
		f.exec(t, `INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh', 'fake', X'00', true, '{}', NOW(), NOW())`, ch, tenant)
		f.exec(t, `INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, 'Khach', NOW(), 1, '{}', NOW(), NOW())`, conv, tenant, ch, "ext-"+conv)
		f.exec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, 'Job', 'qc_analysis', '[]', '', '[]', 'manual', true, '[]', NOW(), NOW())`, job, tenant)
		f.exec(t, `INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'success', '{}', NOW())`, run, job, tenant)
	}

	inside := time.Date(2026, 3, 10, 9, 30, 0, 0, time.UTC)
	// Tenant A, inside the window: two violations on ONE conversation, one
	// evaluation, one classification tag.
	f.insertResult(t, f.tenantA, f.runA, f.convA, dashQCType, inside)
	f.insertResult(t, f.tenantA, f.runA, f.convA, dashQCType, inside.Add(time.Minute))
	f.insertResult(t, f.tenantA, f.runA, f.convA, dashEvalType, inside)
	f.insertResult(t, f.tenantA, f.runA, f.convA, dashTagType, inside)
	// 01:00 Vietnam time is still 2026-03-09 in UTC: the old UTC-day window missed it, and the
	// pairing below (one old-window row lost at the start edge, one gained at the end edge)
	// would otherwise cancel in the count, so this row makes a shifted window visible.
	f.insertResult(t, f.tenantA, f.runA, f.convA, dashQCType, dashQCFrom.Add(time.Hour))
	// The first instant and the last millisecond of the Vietnam day: counted.
	f.insertResult(t, f.tenantA, f.runA, f.convA, dashQCType, dashQCFrom)
	f.insertResult(t, f.tenantA, f.runA, f.convA, dashQCType, dashQCTo)
	// One millisecond outside either edge (previous day / next midnight): neither field counts them.
	f.insertResult(t, f.tenantA, f.runA, f.convA, dashQCType, dashQCFrom.Add(-time.Millisecond))
	f.insertResult(t, f.tenantA, f.runA, f.convA, dashQCType, dashQCTo.Add(time.Millisecond))
	f.insertResult(t, f.tenantA, f.runA, f.convA, dashEvalType, dashQCFrom.Add(-time.Millisecond))
	// Tenant B: three violations inside the same window.
	for i := 0; i < 3; i++ {
		f.insertResult(t, f.tenantB, f.runB, f.convB, dashQCType, inside.Add(time.Duration(i)*time.Minute))
	}
	// Tenant with no violations in the window (only an evaluation).
	f.insertResult(t, f.tenantEmpty, f.runEmpty, f.convEmpty, dashEvalType, inside)
	return f
}

func callDashboard(tenantID string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenantID)
	c.Request = httptest.NewRequest("GET", "/api/v1/dashboard?from="+dashQCWindowDay+"&to="+dashQCWindowDay, nil)
	GetDashboard(c)
	return rec
}

func dashboardFields(t *testing.T, rec *httptest.ResponseRecorder) (issues, qc float64, body map[string]interface{}) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	var ok bool
	if issues, ok = body["issues"].(float64); !ok {
		t.Fatalf("issues is not a JSON number: %v", body["issues"])
	}
	if qc, ok = body["qc_violation_count"].(float64); !ok {
		t.Fatalf("qc_violation_count missing or not a JSON number: %v", body["qc_violation_count"])
	}
	return issues, qc, body
}

func TestDashboardQCViolationCountIsScopedByTypeTenantAndInterval(t *testing.T) {
	f := setupDashQCFixture(t)

	issues, qc, body := dashboardFields(t, callDashboard(f.tenantA))
	// 5 violations in the interval (two on one conversation + 01:00 VN + both edges).
	if qc != 5 {
		t.Fatalf("qc_violation_count = %v, want 5", qc)
	}
	// issues keeps its old meaning: every result type in the interval
	// (5 violations + 1 evaluation + 1 tag), without the outside-edge rows.
	if issues != 7 {
		t.Fatalf("issues = %v, want 7 (all result types in the interval)", issues)
	}
	for _, key := range []string{"total_conversations", "active_channels", "active_jobs", "conversations_by_channel", "qc_alerts", "classification_recent", "cost_period", "cost_today", "cost_this_month", "cost_by_day", "messages_by_day", "exchange_rate"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("existing response key %q is missing", key)
		}
	}

	// Tenant isolation, both directions.
	issuesB, qcB, _ := dashboardFields(t, callDashboard(f.tenantB))
	if qcB != 3 || issuesB != 3 {
		t.Fatalf("tenant B: qc=%v issues=%v, want 3/3", qcB, issuesB)
	}
}

func TestDashboardQCViolationCountIsNumericZeroWhenNothingMatches(t *testing.T) {
	f := setupDashQCFixture(t)
	issues, qc, body := dashboardFields(t, callDashboard(f.tenantEmpty))
	if qc != 0 {
		t.Fatalf("qc_violation_count = %v, want 0", qc)
	}
	if issues != 1 {
		t.Fatalf("issues = %v, want 1 (the evaluation still counts there)", issues)
	}
	if !strings.Contains(callDashboard(f.tenantEmpty).Body.String(), `"qc_violation_count":0`) {
		t.Fatalf("zero must be serialized as a number, got %v", body["qc_violation_count"])
	}
}

func TestDashboardQCViolationCountChangesWhenAQualifyingRowIsAdded(t *testing.T) {
	f := setupDashQCFixture(t)
	_, before, _ := dashboardFields(t, callDashboard(f.tenantA))
	f.insertResult(t, f.tenantA, f.runA, f.convA, dashQCType, time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC))
	issuesAfter, after, _ := dashboardFields(t, callDashboard(f.tenantA))
	if after != before+1 || after != 6 {
		t.Fatalf("qc_violation_count %v -> %v, want +1 (6)", before, after)
	}
	if issuesAfter != 8 {
		t.Fatalf("issues = %v, want 8", issuesAfter)
	}

	// Non-qualifying rows (other type, other tenant) leave the field unchanged.
	f.insertResult(t, f.tenantA, f.runA, f.convA, dashEvalType, time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC))
	f.insertResult(t, f.tenantB, f.runB, f.convB, dashQCType, time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC))
	_, still, _ := dashboardFields(t, callDashboard(f.tenantA))
	if still != 6 {
		t.Fatalf("non-qualifying rows changed qc_violation_count to %v", still)
	}
}

func TestDashboardQCViolationCountQueryFailureReturnsGeneric500(t *testing.T) {
	f := setupDashQCFixture(t)

	// Fail only the COUNT query that filters on qc_violation; every other
	// dashboard query keeps working, so a false zero would be the only other
	// possible outcome.
	const cbName = "ccma:test_fail_qc_count"
	failed := 0
	if err := db.DB.Callback().Query().After("gorm:query").Register(cbName, func(tx *gorm.DB) {
		if !strings.Contains(strings.ToLower(tx.Statement.SQL.String()), "count(") {
			return
		}
		for _, v := range tx.Statement.Vars {
			if s, ok := v.(string); ok && s == dashQCType {
				failed++
				tx.AddError(errors.New("forced qc count failure: SELECT secret_table_detail"))
				return
			}
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() { db.DB.Callback().Query().Remove(cbName) })

	rec := callDashboard(f.tenantA)
	if failed == 0 {
		t.Fatal("the failure hook never fired: the new count query is not the one being exercised")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500; body %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "dashboard_unavailable" || len(body) != 1 {
		t.Fatalf("body = %s, want only {\"error\":\"dashboard_unavailable\"}", rec.Body.String())
	}
	lower := strings.ToLower(rec.Body.String())
	for _, leak := range []string{"select", "forced", "secret_table", "job_results", "qc_violation_count"} {
		if strings.Contains(lower, leak) {
			t.Fatalf("500 body leaks %q: %s", leak, rec.Body.String())
		}
	}

	// Once the failure is removed the same request succeeds again.
	db.DB.Callback().Query().Remove(cbName)
	if _, qc, _ := dashboardFields(t, callDashboard(f.tenantA)); qc != 5 {
		t.Fatalf("after removing the failure, qc_violation_count = %v, want 5", qc)
	}
}
