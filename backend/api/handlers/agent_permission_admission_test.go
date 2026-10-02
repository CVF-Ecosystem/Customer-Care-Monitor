package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-020 (F01-A): the HTTP agent API applies the requested tenant's stored
// role/permissions to the exact action or query resource. The matrix below is written out
// independently of the handler's table so drift in either is caught. Dispatchers and
// transport are the R011 synthetic observers; no engine, provider or channel is used.

type agentOp struct {
	agent, name string // name: run action or query resource
	query       bool
	needs       []string // resource:letter, all required
}

var agentMatrix = []agentOp{
	{"cqa.sync", "sync_all", false, []string{"channels:w", "messages:w"}},
	{"cqa.sync", "sync_channel", false, []string{"channels:w", "messages:w"}},
	{"cqa.qc", "analyze_quality", false, []string{"jobs:w", "messages:r"}},
	{"cqa.classify", "classify_conversations", false, []string{"jobs:w", "messages:r"}},
	{"cqa.sync", "conversations", true, []string{"messages:r"}},
	{"cqa.sync", "messages", true, []string{"messages:r"}},
	{"cqa.qc", "violations", true, []string{"jobs:r"}},
	{"cqa.classify", "tags", true, []string{"jobs:r"}},
}

// newPermFixture closes the previous test's pool before the R011 fixture opens a new one;
// db.Connect never closes its predecessor, and these tests would otherwise exhaust the
// disposable server's connection limit for later packages' tests.
func newPermFixture(t *testing.T) *agentRunFixture {
	t.Helper()
	db.Close()
	return setupAgentRunFixture(t)
}

func permsJSON(needs []string, skip int) string {
	m := map[string]string{}
	for i, n := range needs {
		if i == skip {
			continue
		}
		parts := strings.SplitN(n, ":", 2)
		m[parts[0]] += parts[1]
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func (f *agentRunFixture) exec(t *testing.T, sql string, args ...interface{}) {
	t.Helper()
	if err := db.DB.Exec(sql, args...).Error; err != nil {
		t.Fatalf("fixture: %v", err)
	}
}

// reset clears the observers between iterations that share one fixture, so the tests do not
// open a new database pool per case.
func (f *agentRunFixture) reset(t *testing.T) {
	t.Helper()
	f.cfgLoads, f.syncCalls, f.analysisCalls = 0, 0, 0
	f.exec(t, `UPDATE channels SET last_sync_status = 'success' WHERE id = ?`, f.channelID)
	f.exec(t, `DELETE FROM job_runs WHERE tenant_id = ?`, f.tenantID)
	f.outbound.requests.Store(0)
}

func (f *agentRunFixture) setMember(t *testing.T, role, perms string) {
	t.Helper()
	f.exec(t, `UPDATE user_tenants SET role = ?, permissions = ? WHERE user_id = ? AND tenant_id = ?`, role, perms, f.userID, f.tenantID)
}

// seed inserts one distinguishable conversation, message, qc_violation and classification_tag
// per tenant; markers identify the owning tenant in any response.
func (f *agentRunFixture) seed(t *testing.T, tenantID, marker string) (convID string) {
	t.Helper()
	convID = "conv-" + marker + "-" + tenantID
	chID := "ch-" + marker + "-" + tenantID
	f.exec(t, `INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh', ?, X'00', true, '{}', NOW(), NOW())`, chID, tenantID, "fake-"+marker+tenantID)
	f.exec(t, `INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NOW(), 1, '{}', NOW(), NOW())`, convID, tenantID, chID, "ext-"+marker+tenantID, "CUST-"+marker)
	f.exec(t, `INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, ?, 'customer', 'Khach', ?, 'text', '[]', NOW(), NOW())`, pkg.NewUUID(), tenantID, convID, "m-"+marker+tenantID, "MSG-"+marker)
	for _, rt := range []string{"qc_violation", "classification_tag"} {
		f.exec(t, `INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, created_at) VALUES (?, ?, ?, ?, ?, 'NGHIEM_TRONG', ?, ?, '{}', 1, NOW())`, pkg.NewUUID(), "run-"+marker, tenantID, convID, rt, "RULE-"+marker, "EVID-"+marker)
	}
	t.Cleanup(func() {
		for _, table := range []string{"job_results", "messages", "conversations", "channels"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenantID)
		}
	})
	return convID
}

func (f *agentRunFixture) bodyForAction(agent, action, tenantID string) string {
	return `{"tenant_id":"` + tenantID + `","action":"` + action + `","params":{"channel_id":"` + f.channelID + `","role":"owner","permissions":"{\"channels\":\"rw\"}"}}`
}

func (f *agentRunFixture) runOp(op agentOp, tenantID string) *httptest.ResponseRecorder {
	return f.call(op.agent, f.bodyForAction(op.agent, op.name, tenantID))
}

func (f *agentRunFixture) queryOp(agent, tenantID, resource, convID string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("user_id", f.userID)
	c.Params = gin.Params{{Key: "agentName", Value: agent}}
	q := url.Values{"tenant_id": {tenantID}, "resource": {resource}, "conversation_id": {convID}}
	c.Request = httptest.NewRequest("GET", "/api/v1/agents/"+agent+"/query?"+q.Encode(), nil)
	AgentQuery(c)
	return rec
}

const deniedBody = `{"error":"permission_denied"}`

func (f *agentRunFixture) assertDeniedRun(t *testing.T, rec *httptest.ResponseRecorder, wantCode int, wantBody string) {
	t.Helper()
	if rec.Code != wantCode || strings.TrimSpace(rec.Body.String()) != wantBody {
		t.Fatalf("got %d %s, want %d %s", rec.Code, rec.Body.String(), wantCode, wantBody)
	}
	if f.cfgLoads != 0 {
		t.Fatalf("config loaded %d times on a rejected run", f.cfgLoads)
	}
	f.assertNoDispatch(t)
}

func TestAgentRunAndQueryRequireMatrixPermissions(t *testing.T) {
	for _, op := range agentMatrix {
		op := op
		kind := "run"
		if op.query {
			kind = "query"
		}
		t.Run(kind+"/"+op.agent+"/"+op.name, func(t *testing.T) {
			f := newPermFixture(t)
			convID := f.seed(t, f.tenantID, "own")
			f.seed(t, f.otherTenantID, "oth")

			// Missing each required right in turn (and both for a pair): denied, generic body.
			for skip := -1; skip < len(op.needs); skip++ {
				perms := permsJSON(op.needs, skip)
				f.setMember(t, "member", perms)
				if skip == -1 {
					continue
				}
				f.cfgLoads = 0
				if op.query {
					rec := f.queryOp(op.agent, f.tenantID, op.name, convID)
					if rec.Code != http.StatusForbidden || strings.TrimSpace(rec.Body.String()) != deniedBody {
						t.Fatalf("perms %s: got %d %s, want 403 %s", perms, rec.Code, rec.Body.String(), deniedBody)
					}
					for _, leak := range []string{"CUST-", "MSG-", "RULE-", "EVID-"} {
						if strings.Contains(rec.Body.String(), leak) {
							t.Fatalf("denied query leaked %q", leak)
						}
					}
				} else {
					f.assertDeniedRun(t, f.runOp(op, f.tenantID), http.StatusForbidden, deniedBody)
				}
			}
			// Read-only where write is required is denied too.
			if !op.query {
				f.setMember(t, "member", `{"channels":"r","messages":"r","jobs":"r"}`)
				f.assertDeniedRun(t, f.runOp(op, f.tenantID), http.StatusForbidden, deniedBody)
			}

			// Complete rights: the same permitted path, detector observers fire, tenant-scoped data.
			f.setMember(t, "member", permsJSON(op.needs, -1))
			if op.query {
				rec := f.queryOp(op.agent, f.tenantID, op.name, convID)
				body := rec.Body.String()
				if rec.Code != http.StatusOK || !strings.Contains(body, "-own") || strings.Contains(body, "-oth") {
					t.Fatalf("authorized query: got %d %s; want own-tenant rows only", rec.Code, body)
				}
				// Cross-tenant: same rights, other tenant id, no membership there.
				rec = f.queryOp(op.agent, f.otherTenantID, op.name, "conv-oth")
				if rec.Code != http.StatusForbidden || strings.Contains(rec.Body.String(), "-oth") {
					t.Fatalf("cross-tenant query: got %d %s", rec.Code, rec.Body.String())
				}
				return
			}
			f.cfgLoads = 0
			rec := f.runOp(op, f.tenantID)
			if rec.Code != http.StatusOK || f.cfgLoads != 1 || f.syncCalls+f.analysisCalls != 1 {
				t.Fatalf("authorized run: got %d, loads %d, dispatches %d", rec.Code, f.cfgLoads, f.syncCalls+f.analysisCalls)
			}
			if got := f.channelStatus(t); got != "syncing" || f.jobRunCount(t) != 1 || f.outbound.requests.Load() != 1 {
				t.Fatalf("detector observers did not fire (status %q); rejection checks would be vacuous", got)
			}
			if (op.agent == "cqa.sync") != (f.syncCalls == 1) || f.gotAgent != op.agent || f.gotReq.Action != op.name {
				t.Fatalf("wrong dispatch: %s %q", f.gotAgent, f.gotReq.Action)
			}
		})
	}
}

func TestAgentOwnerAndAdminAreAdmittedForSupportedPairsOnly(t *testing.T) {
	f := newPermFixture(t)
	convID := f.seed(t, f.tenantID, "own")
	for _, role := range []string{"owner", "admin"} {
		for _, perms := range []string{"", "not-json", `{}`} {
			f.setMember(t, role, perms)
			for _, op := range agentMatrix {
				f.reset(t)
				if op.query {
					rec := f.queryOp(op.agent, f.tenantID, op.name, convID)
					if rec.Code != http.StatusOK {
						t.Fatalf("%s/%q %s: got %d %s", role, perms, op.name, rec.Code, rec.Body.String())
					}
				} else if rec := f.runOp(op, f.tenantID); rec.Code != http.StatusOK || f.syncCalls+f.analysisCalls != 1 {
					t.Fatalf("%s/%q %s: got %d, dispatches %d", role, perms, op.name, rec.Code, f.syncCalls+f.analysisCalls)
				}
			}
		}
		// Even owner/admin cannot run or query an unsupported pair; nothing is loaded or dispatched.
		f.reset(t)
		for _, bad := range []struct{ agent, action string }{{"cqa.qc", "sync_all"}, {"cqa.sync", "analyze_quality"}, {"cqa.classify", "analyze_quality"}, {"cqa.sync", "bogus"}, {"cqa.qc", "query:scores"}} {
			f.assertDeniedRun(t, f.call(bad.agent, f.bodyForAction(bad.agent, bad.action, f.tenantID)), http.StatusBadRequest, `{"error":"unsupported_action"}`)
		}
		for _, bad := range []struct{ agent, res string }{{"cqa.qc", "scores"}, {"cqa.classify", "rules"}, {"cqa.sync", "violations"}, {"cqa.qc", "tags"}, {"cqa.sync", ""}} {
			rec := f.queryOp(bad.agent, f.tenantID, bad.res, "")
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "unknown resource") {
				t.Fatalf("%s %s: got %d %s, want 400 unknown resource", bad.agent, bad.res, rec.Code, rec.Body.String())
			}
		}
	}
}

func TestAgentMemberWithBadPermissionDataFailsClosed(t *testing.T) {
	cases := []struct{ name, role, perms string }{
		{"empty permissions", "member", ""},
		{"empty object", "member", `{}`},
		{"malformed json", "member", `{"channels":`},
		{"non-string values", "member", `{"channels":5,"messages":true,"jobs":[]}`},
		{"array", "member", `["channels:w"]`},
		{"json null", "member", `null`},
		{"unrecognized role with full rights", "guest", `{"channels":"rw","messages":"rw","jobs":"rw"}`},
		{"empty role with full rights", "", `{"channels":"rw","messages":"rw","jobs":"rw"}`},
		{"uppercase role", "OWNER", `{}`},
	}
	f := newPermFixture(t)
	convID := f.seed(t, f.tenantID, "own")
	for _, tc := range cases {
		f.setMember(t, tc.role, tc.perms)
		for _, op := range agentMatrix {
			f.reset(t)
			if op.query {
				rec := f.queryOp(op.agent, f.tenantID, op.name, convID)
				if rec.Code != http.StatusForbidden || strings.TrimSpace(rec.Body.String()) != deniedBody {
					t.Fatalf("%s %s: got %d %s, want 403", tc.name, op.name, rec.Code, rec.Body.String())
				}
				continue
			}
			f.assertDeniedRun(t, f.runOp(op, f.tenantID), http.StatusForbidden, deniedBody)
		}
	}
}

func TestAgentAdmissionOrderAndCrossTenant(t *testing.T) {
	t.Run("rights in another tenant do not help", func(t *testing.T) {
		f := newPermFixture(t)
		f.exec(t, `INSERT INTO user_tenants (user_id, tenant_id, role, permissions) VALUES (?, ?, 'member', ?)`, f.userID, f.otherTenantID, agentAllRightsJSON)
		t.Cleanup(func() { db.DB.Exec("DELETE FROM user_tenants WHERE user_id = ? AND tenant_id = ?", f.userID, f.otherTenantID) })
		f.setMember(t, "member", `{}`) // requested tenant grants nothing
		for _, op := range agentMatrix {
			if op.query {
				continue
			}
			f.assertDeniedRun(t, f.runOp(op, f.tenantID), http.StatusForbidden, deniedBody)
		}
		// The rights in the other tenant work there, proving the denial is per requested tenant.
		if rec := f.runOp(agentMatrix[0], f.otherTenantID); rec.Code != http.StatusOK {
			t.Fatalf("other tenant with rights: got %d %s", rec.Code, rec.Body.String())
		}
	})
	t.Run("nonmember stays 403 tenant_access_denied ahead of everything", func(t *testing.T) {
		f := newPermFixture(t)
		f.setMember(t, "owner", "")
		f.assertDeniedRun(t, f.call("cqa.sync", f.bodyForAction("cqa.sync", "sync_all", f.otherTenantID)), http.StatusForbidden, `{"error":"tenant_access_denied"}`)
		rec := f.queryOp("cqa.unknown", f.otherTenantID, "x", "")
		if rec.Code != http.StatusForbidden || strings.TrimSpace(rec.Body.String()) != `{"error":"tenant_access_denied"}` {
			t.Fatalf("got %d %s", rec.Code, rec.Body.String())
		}
	})
	t.Run("unknown agent is 404 after tenant admission, even without rights", func(t *testing.T) {
		f := newPermFixture(t)
		f.setMember(t, "member", `{}`)
		f.assertDeniedRun(t, f.call("cqa.unknown", f.bodyForAction("cqa.unknown", "sync_all", f.tenantID)), http.StatusNotFound, `{"error":"agent_not_found"}`)
		rec := f.queryOp("cqa.unknown", f.tenantID, "conversations", "")
		if rec.Code != http.StatusNotFound || strings.TrimSpace(rec.Body.String()) != `{"error":"agent_not_found"}` {
			t.Fatalf("got %d %s", rec.Code, rec.Body.String())
		}
	})
	t.Run("permission is checked before config admission", func(t *testing.T) {
		f := newPermFixture(t)
		f.setMember(t, "member", `{}`)
		f.cfgErr = http.ErrAbortHandler // would be a 500 if config were loaded first
		f.assertDeniedRun(t, f.runOp(agentMatrix[0], f.tenantID), http.StatusForbidden, deniedBody)
	})
	t.Run("denial log names classes only", func(t *testing.T) {
		f := newPermFixture(t)
		f.setMember(t, "member", `{"channels":"r","SECRET-KEY":"x"}`)
		logs := captureLog(t)
		f.assertDeniedRun(t, f.runOp(agentMatrix[0], f.tenantID), http.StatusForbidden, deniedBody)
		if !strings.Contains(logs.String(), "permission denied") || !strings.Contains(logs.String(), "run:sync_all") {
			t.Fatalf("expected a class-only denial log, got %q", logs.String())
		}
		for _, leak := range []string{"SECRET-KEY", agentRunSecretParam, "channels"} {
			if strings.Contains(logs.String(), leak) {
				t.Fatalf("denial log leaked %q: %s", leak, logs.String())
			}
		}
	})
}

// Mounted routes with real JWT parsing: JWT alone grants nothing to a member; discovery
// and health stay JWT-only and return no tenant data.
func TestAgentRoutesWithJWT(t *testing.T) {
	f := newPermFixture(t)
	f.seed(t, f.tenantID, "own")
	middleware.SetJWTSecret("synthetic-jwt-secret-for-agent-route-tests-0123456789")
	t.Cleanup(func() { middleware.SetJWTSecret("") })
	token, err := middleware.GenerateAccessToken(f.userID, "agrun@example.invalid", false)
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1/agents", middleware.JWTAuth())
	g.GET("", ListAgents)
	g.GET("/capabilities", ListAgents)
	g.POST("/:agentName/run", AgentRun)
	g.GET("/:agentName/query", AgentQuery)
	g.GET("/:agentName/health", AgentHealth)

	do := func(method, path, body string, auth bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if auth {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	runBody := f.bodyForAction("cqa.sync", "sync_all", f.tenantID)
	queryPath := "/api/v1/agents/cqa.sync/query?resource=conversations&tenant_id=" + f.tenantID

	if rec := do("POST", "/api/v1/agents/cqa.sync/run", runBody, false); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: got %d", rec.Code)
	}
	f.setMember(t, "member", `{}`)
	if rec := do("POST", "/api/v1/agents/cqa.sync/run", runBody, true); rec.Code != http.StatusForbidden || strings.TrimSpace(rec.Body.String()) != deniedBody {
		t.Fatalf("member with JWT only: got %d %s", rec.Code, rec.Body.String())
	}
	f.assertNoDispatch(t)
	if rec := do("GET", queryPath, "", true); rec.Code != http.StatusForbidden || strings.Contains(rec.Body.String(), "CUST-") {
		t.Fatalf("member query with JWT only: got %d %s", rec.Code, rec.Body.String())
	}
	for _, path := range []string{"/api/v1/agents", "/api/v1/agents/capabilities", "/api/v1/agents/cqa.sync/health"} {
		rec := do("GET", path, "", true)
		if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "-own") || strings.Contains(rec.Body.String(), f.tenantID) {
			t.Fatalf("%s: got %d %s, want JWT-only discovery without tenant data", path, rec.Code, rec.Body.String())
		}
	}
	f.setMember(t, "member", `{"channels":"w","messages":"rw"}`)
	if rec := do("POST", "/api/v1/agents/cqa.sync/run", runBody, true); rec.Code != http.StatusOK || f.syncCalls != 1 {
		t.Fatalf("authorized member run: got %d, sync dispatches %d", rec.Code, f.syncCalls)
	}
	if rec := do("GET", queryPath, "", true); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "CUST-own") {
		t.Fatalf("authorized member query: got %d %s", rec.Code, rec.Body.String())
	}
}
