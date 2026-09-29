package handlers

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-010: channel credential and OAuth actions are admitted only
// with a validated configuration. The config loader is stubbed (except in the
// real-validation test) and every outbound HTTP request goes to a recording
// transport with synthetic answers; no real OAuth endpoint, channel or provider
// is contacted and all credentials are synthetic.

const (
	channelCfgJWT    = "synthetic-jwt-secret-for-tests-0123456789"
	channelCfgKey    = "synthetic-32-byte-key-0123456789" // exactly 32 bytes
	channelCfgSecret = "ENCRYPTION_KEY=SECRET-DO-NOT-LEAK is invalid"
)

// stubOutbound records every outbound request and answers the known OAuth and
// Graph endpoints with synthetic tokens; anything else gets a 404.
type stubOutbound struct {
	mu       sync.Mutex
	requests []string
	// hook, when set, runs for every request after it is recorded and before
	// the synthetic answer is returned (CCMAI-RUNTIME-012 race fixtures).
	hook func(method, hostPath string)
}

func (s *stubOutbound) RoundTrip(r *http.Request) (*http.Response, error) {
	s.mu.Lock()
	s.requests = append(s.requests, r.Method+" "+r.URL.Host+r.URL.Path)
	s.mu.Unlock()
	if s.hook != nil {
		s.hook(r.Method, r.URL.Host+r.URL.Path)
	}

	status, body := http.StatusNotFound, `{}`
	switch r.URL.Host + r.URL.Path {
	case "oauth.zaloapp.com/v4/oa/access_token":
		status, body = http.StatusOK, `{"access_token":"synthetic-zalo-access","refresh_token":"synthetic-zalo-refresh","expires_in":"3600"}`
	case "openapi.zalo.me/v2.0/oa/getoa":
		status, body = http.StatusOK, `{"error":0,"data":{"oa_id":"synthetic-oa","name":"Synthetic OA"}}`
	case "graph.facebook.com/v21.0/oauth/access_token":
		status, body = http.StatusOK, `{"access_token":"synthetic-fb-user"}`
		if r.Method == http.MethodGet {
			body = `{"access_token":"synthetic-fb-long"}`
		}
	case "graph.facebook.com/v21.0/me/accounts":
		status, body = http.StatusOK, `{"data":[{"id":"synthetic-page","name":"Synthetic Page","access_token":"synthetic-page-token"}]}`
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

func (s *stubOutbound) calls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

type channelConfigFixture struct {
	tenantID, otherTenantID string
	zaloID, fbID            string
	out                     *stubOutbound

	cfg      *config.Config
	cfgErr   error
	cfgLoads int
}

func setupChannelConfigFixture(t *testing.T) *channelConfigFixture {
	t.Helper()
	connectChannelsTestDB(t)
	s := pkg.NewUUID()[:8]
	f := &channelConfigFixture{
		tenantID: "chcfg-" + s, otherTenantID: "chcfg-other-" + s,
		zaloID: "ch-chcfg-zalo-" + s, fbID: "ch-chcfg-fb-" + s,
		out: &stubOutbound{},
		cfg: &config.Config{Env: "test", JWTSecret: channelCfgJWT, EncryptionKey: channelCfgKey},
	}
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, tenant := range []string{f.tenantID, f.otherTenantID} {
			db.DB.Exec("DELETE FROM channels WHERE tenant_id = ?", tenant)
			db.DB.Exec("DELETE FROM activity_logs WHERE tenant_id = ?", tenant)
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenant)
		}
	})
	for _, tenant := range []string{f.tenantID, f.otherTenantID} {
		exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Channel Config', ?, '{}', NOW(), NOW())`, tenant, tenant)
	}
	creds, err := pkg.Encrypt([]byte(`{"app_id":"synthetic-app","app_secret":"synthetic-app-secret"}`), channelCfgKey)
	if err != nil {
		t.Fatalf("encrypt fixture credentials: %v", err)
	}
	for id, kind := range map[string]string{f.zaloID: "zalo_oa", f.fbID: "facebook"} {
		exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, last_sync_status, last_sync_error, metadata, created_at, updated_at) VALUES (?, ?, ?, 'Kenh', 'fixture-ext', ?, true, '', '', '{}', '2026-01-01 00:00:00', '2026-01-01 00:00:00')`,
			id, f.tenantID, kind, creds)
	}

	originalClient, originalTransport := httpClientWithTimeout, http.DefaultTransport
	httpClientWithTimeout = &http.Client{Transport: f.out}
	http.DefaultTransport = f.out
	t.Cleanup(func() { httpClientWithTimeout, http.DefaultTransport = originalClient, originalTransport })

	originalLoad := loadChannelSecurityConfig
	loadChannelSecurityConfig = func() (*config.Config, error) {
		f.cfgLoads++
		if f.cfgErr != nil {
			return nil, f.cfgErr
		}
		return f.cfg, nil
	}
	t.Cleanup(func() { loadChannelSecurityConfig = originalLoad })
	return f
}

// channelRowState is everything a credential/OAuth route could write.
type channelRowState struct {
	ID, TenantID, Name, ExternalID, Credentials, UpdatedAt string
}

func (f *channelConfigFixture) rows(t *testing.T) []channelRowState {
	t.Helper()
	var raw []struct {
		ID, TenantID, Name, ExternalID string
		CredentialsEncrypted           []byte
		UpdatedAt                      string
	}
	if err := db.DB.Raw(`SELECT id, tenant_id, name, external_id, credentials_encrypted, CAST(updated_at AS CHAR) AS updated_at FROM channels WHERE tenant_id IN (?, ?) ORDER BY id`,
		f.tenantID, f.otherTenantID).Scan(&raw).Error; err != nil {
		t.Fatalf("read channels: %v", err)
	}
	out := make([]channelRowState, len(raw))
	for i, r := range raw {
		out[i] = channelRowState{r.ID, r.TenantID, r.Name, r.ExternalID, hex.EncodeToString(r.CredentialsEncrypted), r.UpdatedAt}
	}
	return out
}

func (f *channelConfigFixture) credentials(t *testing.T, channelID string) map[string]string {
	t.Helper()
	var enc []byte
	if err := db.DB.Raw("SELECT credentials_encrypted FROM channels WHERE id = ?", channelID).Row().Scan(&enc); err != nil {
		t.Fatalf("read credentials: %v", err)
	}
	plain, err := pkg.Decrypt(enc, channelCfgKey)
	if err != nil {
		t.Fatalf("stored credentials do not decrypt with the validated key: %v", err)
	}
	var creds map[string]string
	if err := json.Unmarshal(plain, &creds); err != nil {
		t.Fatalf("stored credentials: %v", err)
	}
	return creds
}

func (f *channelConfigFixture) state(tenantID, channelID string) string {
	return signOAuthState(tenantID, channelID, channelCfgJWT)
}

func serveChannelHandler(handler gin.HandlerFunc, tenantID, method, target, body string, params gin.Params) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	if tenantID != "" {
		c.Set("tenant_id", tenantID)
	}
	c.Params = params
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	handler(c)
	return rec
}

const fbCreateBody = `{"channel_type":"facebook","name":"Kenh FB","credentials":{"page_id":"synthetic-page","access_token":"synthetic-user-token"}}`

func (f *channelConfigFixture) callCreate(body string) *httptest.ResponseRecorder {
	return serveChannelHandler(CreateChannel, f.tenantID, "POST", "/api/v1/channels", body, nil)
}

func (f *channelConfigFixture) callTest(tenantID, channelID string) *httptest.ResponseRecorder {
	return serveChannelHandler(TestChannelConnection, tenantID, "POST", "/api/v1/channels/"+channelID+"/test", "", gin.Params{{Key: "channelId", Value: channelID}})
}

func (f *channelConfigFixture) callReauth(tenantID, channelID string) *httptest.ResponseRecorder {
	return serveChannelHandler(ReauthChannel, tenantID, "POST", "/api/v1/channels/"+channelID+"/reauth", "", gin.Params{{Key: "channelId", Value: channelID}})
}

func callOAuthCallback(handler gin.HandlerFunc, path string, query url.Values) *httptest.ResponseRecorder {
	return serveChannelHandler(handler, "", "GET", path+"?"+query.Encode(), "", nil)
}

func (f *channelConfigFixture) callZalo(code, state string) *httptest.ResponseRecorder {
	return callOAuthCallback(ZaloOAuthCallback, "/api/v1/channels/zalo/callback", url.Values{"code": {code}, "state": {state}})
}

func (f *channelConfigFixture) callFacebook(code, state string) *httptest.ResponseRecorder {
	return callOAuthCallback(FacebookOAuthCallback, "/api/v1/channels/facebook/callback", url.Values{"code": {code}, "state": {state}})
}

const (
	authFailedRedirect   = "/login?zalo_auth=error&message=Authorization+failed"
	missingCodeRedirect  = "/login?zalo_auth=error&message=Missing+code+or+state"
	noOutboundOrWriteMsg = "no outbound request and no channel write"
)

// assertResponse checks the exact status and either the exact JSON body or,
// for a redirect, the exact Location.
func assertResponse(t *testing.T, rec *httptest.ResponseRecorder, code int, want string) {
	t.Helper()
	if rec.Code != code {
		t.Fatalf("status %d, want %d; body %s", rec.Code, code, rec.Body.String())
	}
	got := strings.TrimSpace(rec.Body.String())
	if code == http.StatusFound {
		got = rec.Header().Get("Location")
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func (f *channelConfigFixture) assertUntouched(t *testing.T, before []channelRowState) {
	t.Helper()
	if calls := f.out.calls(); len(calls) != 0 {
		t.Fatalf("want %s; outbound requests: %v", noOutboundOrWriteMsg, calls)
	}
	if after := f.rows(t); !reflect.DeepEqual(before, after) {
		t.Fatalf("want %s; channels changed:\nbefore %+v\nafter  %+v", noOutboundOrWriteMsg, before, after)
	}
}

type channelConfigRoute struct {
	name string
	call func(f *channelConfigFixture) *httptest.ResponseRecorder
	code int
	want string // failure body or Location
}

func channelConfigRoutes() []channelConfigRoute {
	return []channelConfigRoute{
		{"CreateChannel", func(f *channelConfigFixture) *httptest.ResponseRecorder { return f.callCreate(fbCreateBody) },
			http.StatusInternalServerError, `{"error":"create_channel_failed"}`},
		{"TestChannelConnection", func(f *channelConfigFixture) *httptest.ResponseRecorder { return f.callTest(f.tenantID, f.zaloID) },
			http.StatusInternalServerError, `{"error":"decrypt_failed"}`},
		{"ReauthChannel", func(f *channelConfigFixture) *httptest.ResponseRecorder { return f.callReauth(f.tenantID, f.zaloID) },
			http.StatusInternalServerError, `{"error":"Decrypt failed"}`},
		{"ZaloOAuthCallback", func(f *channelConfigFixture) *httptest.ResponseRecorder {
			return f.callZalo("synthetic-code", f.state(f.tenantID, f.zaloID))
		}, http.StatusFound, authFailedRedirect},
		{"FacebookOAuthCallback", func(f *channelConfigFixture) *httptest.ResponseRecorder {
			return f.callFacebook("synthetic-code", f.state(f.tenantID, f.fbID))
		}, http.StatusFound, authFailedRedirect},
	}
}

func TestChannelConfigFailureIsNotAdmitted(t *testing.T) {
	for _, route := range channelConfigRoutes() {
		for _, mode := range []string{"error", "nil"} {
			t.Run(route.name+"/"+mode, func(t *testing.T) {
				f := setupChannelConfigFixture(t)
				if mode == "error" {
					f.cfgErr = errors.New(channelCfgSecret)
				} else {
					f.cfg = nil
				}
				before := f.rows(t)
				logs := captureLog(t)

				rec := route.call(f) // a nil config dereference would panic here

				assertResponse(t, rec, route.code, route.want)
				if f.cfgLoads != 1 {
					t.Fatalf("config loaded %d times, want 1", f.cfgLoads)
				}
				f.assertUntouched(t, before)
				visible := rec.Body.String() + rec.Header().Get("Location") + logs.String()
				for _, leak := range []string{"SECRET", "ENCRYPTION_KEY", "state=", "redirect_url", "success", "synthetic-code"} {
					if strings.Contains(visible, leak) {
						t.Fatalf("%q exposed in response or log: %s", leak, visible)
					}
				}
				if !strings.Contains(logs.String(), "configuration invalid") {
					t.Fatalf("log should name the failure class: %s", logs.String())
				}
			})
		}
	}
}

// The accepted paths are the detector for the failure tests above: with a
// valid config the same fixture observes outbound requests and channel writes.
func TestChannelConfigAcceptedPathsReachSuccessBoundary(t *testing.T) {
	t.Run("CreateChannel/facebook", func(t *testing.T) {
		f := setupChannelConfigFixture(t)
		rec := f.callCreate(fbCreateBody)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d, want 201; body %s", rec.Code, rec.Body.String())
		}
		var resp ChannelResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || resp.ChannelType != "facebook" || resp.Name != "Synthetic Page" || resp.ExternalID != "synthetic-page" {
			t.Fatalf("response %+v (%v)", resp, err)
		}
		if calls := f.out.calls(); !reflect.DeepEqual(calls, []string{"GET graph.facebook.com/v21.0/me/accounts"}) {
			t.Fatalf("outbound %v, want the page-token exchange only", calls)
		}
		if creds := f.credentials(t, resp.ID); creds["access_token"] != "synthetic-page-token" || creds["page_id"] != "synthetic-page" {
			t.Fatalf("stored credentials %v", creds)
		}
		if f.cfgLoads != 1 {
			t.Fatalf("config loaded %d times, want 1", f.cfgLoads)
		}
	})

	t.Run("CreateChannel/zalo_oa", func(t *testing.T) {
		f := setupChannelConfigFixture(t)
		rec := f.callCreate(`{"channel_type":"zalo_oa","name":"Kenh Zalo","credentials":{"app_id":"synthetic-app","app_secret":"synthetic-app-secret"}}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d, want 201; body %s", rec.Code, rec.Body.String())
		}
		var resp ChannelResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if creds := f.credentials(t, resp.ID); creds["app_secret"] != "synthetic-app-secret" {
			t.Fatalf("stored credentials %v", creds)
		}
		if calls := f.out.calls(); len(calls) != 0 {
			t.Fatalf("unexpected outbound %v", calls)
		}
	})

	t.Run("TestChannelConnection", func(t *testing.T) {
		f := setupChannelConfigFixture(t)
		assertResponse(t, f.callTest(f.tenantID, f.zaloID), http.StatusOK, `{"message":"connection_successful","status":"ok"}`)
		if f.cfgLoads != 1 {
			t.Fatalf("config loaded %d times, want 1", f.cfgLoads)
		}
	})

	t.Run("ReauthChannel", func(t *testing.T) {
		f := setupChannelConfigFixture(t)
		for id, prefix := range map[string]string{
			f.zaloID: "https://oauth.zaloapp.com/v4/oa/permission?app_id=synthetic-app&",
			f.fbID:   "https://www.facebook.com/v21.0/dialog/oauth?client_id=synthetic-app&",
		} {
			rec := f.callReauth(f.tenantID, id)
			var resp struct {
				RedirectURL string `json:"redirect_url"`
			}
			if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &resp) != nil {
				t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
			}
			if !strings.HasPrefix(resp.RedirectURL, prefix) || !strings.Contains(resp.RedirectURL, "&state="+f.state(f.tenantID, id)) {
				t.Fatalf("redirect %q, want %s... with a state signed by the validated secret", resp.RedirectURL, prefix)
			}
		}
		if calls := f.out.calls(); len(calls) != 0 || f.cfgLoads != 2 {
			t.Fatalf("outbound %v, config loads %d; want none and 2", calls, f.cfgLoads)
		}
	})

	t.Run("ZaloOAuthCallback", func(t *testing.T) {
		f := setupChannelConfigFixture(t)
		rec := f.callZalo("synthetic-code", f.state(f.tenantID, f.zaloID))
		assertResponse(t, rec, http.StatusFound, "/"+f.tenantID+"/channels/"+f.zaloID+"?zalo_auth=success")
		want := []string{"POST oauth.zaloapp.com/v4/oa/access_token", "GET openapi.zalo.me/v2.0/oa/getoa"}
		if calls := f.out.calls(); !reflect.DeepEqual(calls, want) {
			t.Fatalf("outbound %v, want %v", calls, want)
		}
		creds := f.credentials(t, f.zaloID)
		if creds["access_token"] != "synthetic-zalo-access" || creds["refresh_token"] != "synthetic-zalo-refresh" || creds["app_secret"] != "synthetic-app-secret" {
			t.Fatalf("stored credentials %v", creds)
		}
		var externalID string
		if err := db.DB.Raw("SELECT external_id FROM channels WHERE id = ?", f.zaloID).Row().Scan(&externalID); err != nil || externalID != "synthetic-oa" {
			t.Fatalf("external_id %q (%v), want synthetic-oa", externalID, err)
		}
	})

	t.Run("FacebookOAuthCallback", func(t *testing.T) {
		f := setupChannelConfigFixture(t)
		rec := f.callFacebook("synthetic-code", f.state(f.tenantID, f.fbID))
		assertResponse(t, rec, http.StatusFound, "/"+f.tenantID+"/channels/"+f.fbID+"?fb_auth=success")
		want := []string{"POST graph.facebook.com/v21.0/oauth/access_token", "GET graph.facebook.com/v21.0/oauth/access_token", "GET graph.facebook.com/v21.0/me/accounts"}
		if calls := f.out.calls(); !reflect.DeepEqual(calls, want) {
			t.Fatalf("outbound %v, want %v", calls, want)
		}
		if creds := f.credentials(t, f.fbID); creds["access_token"] != "synthetic-page-token" || creds["page_id"] != "synthetic-page" {
			t.Fatalf("stored credentials %v", creds)
		}
	})
}

// A loaded config whose key/secret differ from the fixture's must be the one
// the handler uses: decryption and state verification then fail closed.
func TestChannelConfigRoutesUseTheLoadedConfig(t *testing.T) {
	other := &config.Config{Env: "test", JWTSecret: "another-synthetic-jwt-secret-0123456789", EncryptionKey: "another-32-byte-synthetic-key-01"}

	t.Run("TestChannelConnection", func(t *testing.T) {
		f := setupChannelConfigFixture(t)
		f.cfg = other
		assertResponse(t, f.callTest(f.tenantID, f.zaloID), http.StatusInternalServerError, `{"error":"decrypt_failed"}`)
	})
	t.Run("ZaloOAuthCallback", func(t *testing.T) {
		f := setupChannelConfigFixture(t)
		f.cfg = other
		before := f.rows(t)
		assertResponse(t, f.callZalo("synthetic-code", f.state(f.tenantID, f.zaloID)), http.StatusFound, authFailedRedirect)
		f.assertUntouched(t, before)
	})
	t.Run("FacebookOAuthCallback", func(t *testing.T) {
		f := setupChannelConfigFixture(t)
		f.cfg = other
		before := f.rows(t)
		assertResponse(t, f.callFacebook("synthetic-code", f.state(f.tenantID, f.fbID)), http.StatusFound, authFailedRedirect)
		f.assertUntouched(t, before)
	})
}

func TestChannelConfigRequestChecksRunBeforeConfigLoad(t *testing.T) {
	f := setupChannelConfigFixture(t)
	before := f.rows(t)

	assertResponse(t, f.callTest(f.otherTenantID, f.zaloID), http.StatusNotFound, `{"error":"channel_not_found"}`)
	assertResponse(t, f.callReauth(f.otherTenantID, f.zaloID), http.StatusNotFound, `{"error":"Channel not found"}`)
	if rec := f.callCreate(`{"channel_type":"unknown","name":"x"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed create got %d, want 400", rec.Code)
	}
	for _, q := range []url.Values{{"state": {f.state(f.tenantID, f.zaloID)}}, {"code": {"synthetic-code"}}, {}} {
		assertResponse(t, callOAuthCallback(ZaloOAuthCallback, "/api/v1/channels/zalo/callback", q), http.StatusFound, missingCodeRedirect)
		assertResponse(t, callOAuthCallback(FacebookOAuthCallback, "/api/v1/channels/facebook/callback", q), http.StatusFound, missingCodeRedirect)
	}
	if f.cfgLoads != 0 {
		t.Fatalf("config loaded %d times before request/tenant checks", f.cfgLoads)
	}
	f.assertUntouched(t, before)
}

// After config admission, callbacks still verify the signed state before the
// tenant-scoped channel lookup and never exchange a code for a foreign channel.
func TestChannelOAuthCallbacksKeepStateAndTenantOrder(t *testing.T) {
	f := setupChannelConfigFixture(t)
	before := f.rows(t)

	forged := f.tenantID + ":" + f.zaloID + ":0000000000000000"
	assertResponse(t, f.callZalo("synthetic-code", forged), http.StatusFound, authFailedRedirect)
	assertResponse(t, f.callFacebook("synthetic-code", forged), http.StatusFound, authFailedRedirect)

	foreign := "/" + f.otherTenantID + "/channels?zalo_auth=error&message=Channel+not+found"
	assertResponse(t, f.callZalo("synthetic-code", f.state(f.otherTenantID, f.zaloID)), http.StatusFound, foreign)
	assertResponse(t, f.callFacebook("synthetic-code", f.state(f.otherTenantID, f.fbID)), http.StatusFound, foreign)

	if f.cfgLoads != 4 {
		t.Fatalf("config loaded %d times, want once per admitted callback (4)", f.cfgLoads)
	}
	f.assertUntouched(t, before)
}

// TestChannelConfigUsesRealConfigValidation runs the real config.Load with
// synthetic environment values (t.Setenv restores them; no t.Parallel).
func TestChannelConfigUsesRealConfigValidation(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		f := setupChannelConfigFixture(t)
		loadChannelSecurityConfig = config.Load // restored by the fixture cleanup
		t.Setenv("JWT_SECRET", "too-short")
		t.Setenv("ENCRYPTION_KEY", channelCfgKey)
		t.Setenv("DB_PASSWORD", "synthetic")
		before := f.rows(t)
		logs := captureLog(t)

		recs := []*httptest.ResponseRecorder{
			f.callReauth(f.tenantID, f.zaloID),
			f.callZalo("synthetic-code", f.state(f.tenantID, f.zaloID)),
		}
		assertResponse(t, recs[0], http.StatusInternalServerError, `{"error":"Decrypt failed"}`)
		assertResponse(t, recs[1], http.StatusFound, authFailedRedirect)
		for _, rec := range recs {
			if strings.Contains(rec.Body.String()+logs.String(), "JWT") {
				t.Fatalf("validation detail leaked; body %q, log %q", rec.Body.String(), logs.String())
			}
		}
		f.assertUntouched(t, before)
	})

	t.Run("valid", func(t *testing.T) {
		f := setupChannelConfigFixture(t)
		loadChannelSecurityConfig = config.Load
		t.Setenv("JWT_SECRET", channelCfgJWT)
		t.Setenv("ENCRYPTION_KEY", channelCfgKey)
		t.Setenv("DB_PASSWORD", "synthetic")

		rec := f.callZalo("synthetic-code", f.state(f.tenantID, f.zaloID))
		assertResponse(t, rec, http.StatusFound, "/"+f.tenantID+"/channels/"+f.zaloID+"?zalo_auth=success")
		if creds := f.credentials(t, f.zaloID); creds["access_token"] != "synthetic-zalo-access" {
			t.Fatalf("stored credentials %v", creds)
		}
	})
}
