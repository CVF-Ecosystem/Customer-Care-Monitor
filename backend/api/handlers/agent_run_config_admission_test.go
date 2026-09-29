package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-011: AgentRun admits a known agent only with a validated
// configuration, checked after request binding, tenant authorization and the
// unknown-name check, and before any engine is dispatched. Loader and
// dispatchers are stubbed by setupAgentRunFixture; the stub dispatchers
// reproduce a channel write so the "no write" assertions are not vacuous. No
// engine, adapter, credential or provider is ever called.

const agentRunSecretParam = "PARAM-SECRET-DO-NOT-LEAK"

var knownAgents = []string{"cqa.sync", "cqa.qc", "cqa.classify"}

type agentRunFixture struct {
	userID, tenantID, otherTenantID, channelID string

	cfg      *config.Config
	cfgErr   error
	cfgLoads int

	syncCalls, analysisCalls int
	gotCfg                   *config.Config
	gotAgent                 string
	gotReq                   AgentRunRequest
}

func setupAgentRunFixture(t *testing.T) *agentRunFixture {
	t.Helper()
	connectChannelsTestDB(t)
	s := pkg.NewUUID()[:8]
	f := &agentRunFixture{
		userID: pkg.NewUUID(), tenantID: "agrun-" + s, otherTenantID: "agrun-other-" + s, channelID: "ch-agrun-" + s,
	}
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	for _, tenant := range []string{f.tenantID, f.otherTenantID} {
		exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Agent Run', ?, '{}', NOW(), NOW())`, tenant, tenant)
	}
	exec(`INSERT INTO users (id, email, password_hash, name, is_admin, token_version, language, created_at, updated_at) VALUES (?, ?, 'x', 'Agent Run', false, 0, 'vi', NOW(), NOW())`,
		f.userID, "agrun-"+s+"@example.invalid")
	exec(`INSERT INTO user_tenants (user_id, tenant_id, role, permissions) VALUES (?, ?, 'member', '{}')`, f.userID, f.tenantID)
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, last_sync_status, last_sync_error, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh', 'fake', X'00', true, 'success', '', '{}', NOW(), NOW())`,
		f.channelID, f.tenantID)
	t.Cleanup(func() {
		db.DB.Exec("DELETE FROM channels WHERE id = ?", f.channelID)
		db.DB.Exec("DELETE FROM user_tenants WHERE user_id = ?", f.userID)
		db.DB.Exec("DELETE FROM users WHERE id = ?", f.userID)
		for _, tenant := range []string{f.tenantID, f.otherTenantID} {
			db.DB.Exec("DELETE FROM activity_logs WHERE tenant_id = ?", tenant)
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenant)
		}
	})

	// The stubs reproduce an observable engine side effect (a channel write).
	touch := func() {
		exec(`UPDATE channels SET last_sync_status = 'syncing' WHERE id = ?`, f.channelID)
	}
	origSync, origAnalysis := dispatchSyncAgent, dispatchAnalysisAgent
	dispatchSyncAgent = func(_ context.Context, cfg *config.Config, req AgentRunRequest) AgentRunResponse {
		f.syncCalls++
		f.gotCfg, f.gotAgent, f.gotReq = cfg, "cqa.sync", req
		touch()
		return AgentRunResponse{Status: "stub-cqa.sync"}
	}
	dispatchAnalysisAgent = func(_ context.Context, cfg *config.Config, req AgentRunRequest, agentName string) AgentRunResponse {
		f.analysisCalls++
		f.gotCfg, f.gotAgent, f.gotReq = cfg, agentName, req
		touch()
		return AgentRunResponse{Status: "stub-" + agentName}
	}
	t.Cleanup(func() { dispatchSyncAgent, dispatchAnalysisAgent = origSync, origAnalysis })

	f.cfg = &config.Config{Env: "test"} // synthetic; no real secrets or credentials
	origLoad := loadAgentRunConfig
	loadAgentRunConfig = func() (*config.Config, error) {
		f.cfgLoads++
		if f.cfgErr != nil {
			return nil, f.cfgErr
		}
		return f.cfg, nil
	}
	t.Cleanup(func() { loadAgentRunConfig = origLoad })
	return f
}

func (f *agentRunFixture) call(agent, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("user_id", f.userID)
	c.Params = gin.Params{{Key: "agentName", Value: agent}}
	c.Request = httptest.NewRequest("POST", "/api/v1/agents/"+agent+"/run", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	AgentRun(c)
	return rec
}

func (f *agentRunFixture) body(tenantID string) string {
	return `{"tenant_id":"` + tenantID + `","action":"sync_all","params":{"token":"` + agentRunSecretParam + `"}}`
}

func (f *agentRunFixture) channelStatus(t *testing.T) string {
	t.Helper()
	var ch models.Channel
	if err := db.DB.Where("id = ?", f.channelID).First(&ch).Error; err != nil {
		t.Fatalf("load channel: %v", err)
	}
	return ch.LastSyncStatus
}

func (f *agentRunFixture) assertNoDispatch(t *testing.T) {
	t.Helper()
	if f.syncCalls != 0 || f.analysisCalls != 0 {
		t.Fatalf("engine dispatched (sync %d, analysis %d) without a validated config", f.syncCalls, f.analysisCalls)
	}
	if got := f.channelStatus(t); got != "success" {
		t.Fatalf("channel written despite rejection: status %q", got)
	}
}

func TestAgentRunConfigFailureIsNotAdmitted(t *testing.T) {
	for _, agent := range knownAgents {
		for _, mode := range []string{"loader-error", "nil-config"} {
			t.Run(agent+"/"+mode, func(t *testing.T) {
				f := setupAgentRunFixture(t)
				if mode == "loader-error" {
					f.cfgErr = errors.New("ENCRYPTION_KEY=SECRET-DO-NOT-LEAK is invalid")
				} else {
					f.cfg = nil
				}
				logs := captureLog(t)

				rec := f.call(agent, f.body(f.tenantID))

				if rec.Code != http.StatusInternalServerError || strings.TrimSpace(rec.Body.String()) != `{"error":"agent_run_failed"}` {
					t.Fatalf("got %d %s, want generic 500 agent_run_failed", rec.Code, rec.Body.String())
				}
				if f.cfgLoads != 1 {
					t.Fatalf("config loaded %d times, want 1", f.cfgLoads)
				}
				f.assertNoDispatch(t)
				for _, leak := range []string{"SECRET", "ENCRYPTION_KEY", agentRunSecretParam, "is invalid"} {
					if strings.Contains(rec.Body.String(), leak) || strings.Contains(logs.String(), leak) {
						t.Fatalf("detail %q leaked; body %q, log %q", leak, rec.Body.String(), logs.String())
					}
				}
				if !strings.Contains(logs.String(), agent) || !strings.Contains(logs.String(), "configuration invalid") {
					t.Fatalf("log should name the agent and the failure class: %s", logs.String())
				}
			})
		}
	}
}

// Accepted-path detector: with a valid config the same observers see the
// dispatch, the exact config pointer, the route and the unchanged body shape.
func TestAgentRunPassesValidatedConfigToEngine(t *testing.T) {
	for _, agent := range knownAgents {
		t.Run(agent, func(t *testing.T) {
			f := setupAgentRunFixture(t)
			rec := f.call(agent, f.body(f.tenantID))

			want := `{"status":"stub-` + agent + `"}`
			if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
				t.Fatalf("got %d %s, want 200 %s", rec.Code, rec.Body.String(), want)
			}
			if f.cfgLoads != 1 || f.syncCalls+f.analysisCalls != 1 {
				t.Fatalf("loads %d, dispatches %d; want 1 and 1", f.cfgLoads, f.syncCalls+f.analysisCalls)
			}
			if (agent == "cqa.sync") != (f.syncCalls == 1) {
				t.Fatalf("wrong route for %s: sync %d, analysis %d", agent, f.syncCalls, f.analysisCalls)
			}
			if f.gotCfg != f.cfg || f.gotAgent != agent || f.gotReq.TenantID != f.tenantID {
				t.Fatalf("dispatch got cfg %p (want %p), agent %q, tenant %q", f.gotCfg, f.cfg, f.gotAgent, f.gotReq.TenantID)
			}
			if got := f.channelStatus(t); got != "syncing" {
				t.Fatalf("detector observer did not see the write (status %q); rejection checks would be vacuous", got)
			}
		})
	}
}

func TestAgentRunAdmissionOrder(t *testing.T) {
	t.Run("malformed request", func(t *testing.T) {
		f := setupAgentRunFixture(t)
		rec := f.call("cqa.sync", `{"tenant_id":`)
		if rec.Code != http.StatusBadRequest || f.cfgLoads != 0 {
			t.Fatalf("got %d with %d loads, want 400 and 0", rec.Code, f.cfgLoads)
		}
		f.assertNoDispatch(t)
	})
	t.Run("missing required field", func(t *testing.T) {
		f := setupAgentRunFixture(t)
		rec := f.call("cqa.sync", `{"tenant_id":"`+f.tenantID+`"}`)
		if rec.Code != http.StatusBadRequest || f.cfgLoads != 0 {
			t.Fatalf("got %d with %d loads, want 400 and 0", rec.Code, f.cfgLoads)
		}
		f.assertNoDispatch(t)
	})
	for _, agent := range append([]string{"cqa.unknown"}, knownAgents...) {
		t.Run("unauthorized tenant "+agent, func(t *testing.T) {
			f := setupAgentRunFixture(t)
			rec := f.call(agent, f.body(f.otherTenantID))
			if rec.Code != http.StatusForbidden || strings.TrimSpace(rec.Body.String()) != `{"error":"tenant_access_denied"}` || f.cfgLoads != 0 {
				t.Fatalf("got %d %s with %d loads, want 403 and 0 loads", rec.Code, rec.Body.String(), f.cfgLoads)
			}
			f.assertNoDispatch(t)
		})
	}
	t.Run("unknown agent after tenant authorization", func(t *testing.T) {
		f := setupAgentRunFixture(t)
		f.cfgErr = errors.New("would fail")
		rec := f.call("cqa.unknown", f.body(f.tenantID))
		if rec.Code != http.StatusNotFound || strings.TrimSpace(rec.Body.String()) != `{"error":"agent_not_found"}` || f.cfgLoads != 0 {
			t.Fatalf("got %d %s with %d loads, want 404 agent_not_found and 0 loads", rec.Code, rec.Body.String(), f.cfgLoads)
		}
		f.assertNoDispatch(t)
	})
}

// TestAgentRunUsesRealConfigValidation runs the real config.Load with
// synthetic environment values (t.Setenv restores them; no t.Parallel).
func TestAgentRunUsesRealConfigValidation(t *testing.T) {
	const jwt = "synthetic-jwt-secret-for-tests-0123456789"
	const encKey = "synthetic-32-byte-key-0123456789" // exactly 32 bytes

	for _, agent := range knownAgents {
		t.Run("invalid/"+agent, func(t *testing.T) {
			f := setupAgentRunFixture(t)
			loadAgentRunConfig = config.Load // restored by the fixture cleanup
			t.Setenv("JWT_SECRET", "too-short")
			t.Setenv("ENCRYPTION_KEY", encKey)
			t.Setenv("DB_PASSWORD", "synthetic")
			logs := captureLog(t)

			rec := f.call(agent, f.body(f.tenantID))
			if rec.Code != http.StatusInternalServerError || strings.TrimSpace(rec.Body.String()) != `{"error":"agent_run_failed"}` {
				t.Fatalf("got %d %s, want generic 500", rec.Code, rec.Body.String())
			}
			f.assertNoDispatch(t)
			if strings.Contains(rec.Body.String(), "JWT") || strings.Contains(logs.String(), "JWT") {
				t.Fatalf("validation detail leaked; body %q, log %q", rec.Body.String(), logs.String())
			}
		})
		t.Run("valid/"+agent, func(t *testing.T) {
			f := setupAgentRunFixture(t)
			loadAgentRunConfig = config.Load
			t.Setenv("JWT_SECRET", jwt)
			t.Setenv("ENCRYPTION_KEY", encKey)
			t.Setenv("DB_PASSWORD", "synthetic")

			rec := f.call(agent, f.body(f.tenantID))
			if rec.Code != http.StatusOK || f.syncCalls+f.analysisCalls != 1 {
				t.Fatalf("got %d with %d dispatches, want 200 and 1", rec.Code, f.syncCalls+f.analysisCalls)
			}
			if f.gotCfg == nil || f.gotCfg.JWTSecret != jwt || f.gotCfg.EncryptionKey != encKey {
				t.Fatalf("engine did not receive the config validated from the synthetic environment")
			}
		})
	}
}
