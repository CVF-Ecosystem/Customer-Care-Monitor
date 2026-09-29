package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
)

// CCMAI-RUNTIME-012: an OAuth callback reports success only when the final
// tenant-scoped credential update changed exactly one row. The fixture and
// recording transport are shared with the R010 tests; all tokens, states and
// replies are synthetic and no real endpoint is contacted.

type oauthPersistCase struct {
	name      string
	channelID func(f *channelConfigFixture) string
	call      func(f *channelConfigFixture) *httptest.ResponseRecorder
	success   func(f *channelConfigFixture) string // success Location
	outbound  []string                             // exchange calls expected before persistence
	tokenPath string                               // request after which the race hook fires
	tokens    []string                             // synthetic values that must never persist on failure
}

func oauthPersistCases() []oauthPersistCase {
	return []oauthPersistCase{
		{
			name:      "zalo",
			channelID: func(f *channelConfigFixture) string { return f.zaloID },
			call: func(f *channelConfigFixture) *httptest.ResponseRecorder {
				return f.callZalo("synthetic-code", f.state(f.tenantID, f.zaloID))
			},
			success: func(f *channelConfigFixture) string {
				return "/" + f.tenantID + "/channels/" + f.zaloID + "?zalo_auth=success"
			},
			outbound:  []string{"POST oauth.zaloapp.com/v4/oa/access_token", "GET openapi.zalo.me/v2.0/oa/getoa"},
			tokenPath: "openapi.zalo.me/v2.0/oa/getoa",
			tokens:    []string{"synthetic-zalo-access", "synthetic-zalo-refresh", "synthetic-oa"},
		},
		{
			name:      "facebook",
			channelID: func(f *channelConfigFixture) string { return f.fbID },
			call: func(f *channelConfigFixture) *httptest.ResponseRecorder {
				return f.callFacebook("synthetic-code", f.state(f.tenantID, f.fbID))
			},
			success: func(f *channelConfigFixture) string {
				return "/" + f.tenantID + "/channels/" + f.fbID + "?fb_auth=success"
			},
			outbound:  []string{"POST graph.facebook.com/v21.0/oauth/access_token", "GET graph.facebook.com/v21.0/oauth/access_token", "GET graph.facebook.com/v21.0/me/accounts"},
			tokenPath: "graph.facebook.com/v21.0/me/accounts",
			tokens:    []string{"synthetic-page-token", "synthetic-page", "Synthetic Page"},
		},
	}
}

func (f *channelConfigFixture) tenantRedirect() string {
	return "/" + f.tenantID + "/channels?zalo_auth=error&message=Authorization+failed"
}

func (f *channelConfigFixture) channelTenant(t *testing.T, channelID string) string {
	t.Helper()
	var tenant string
	if err := db.DB.Raw("SELECT tenant_id FROM channels WHERE id = ?", channelID).Row().Scan(&tenant); err != nil {
		t.Fatalf("read channel tenant: %v", err)
	}
	return tenant
}

func (f *channelConfigFixture) assertPersistenceFailure(t *testing.T, tc oauthPersistCase, rec *httptest.ResponseRecorder, logs string) {
	t.Helper()
	assertResponse(t, rec, http.StatusFound, f.tenantRedirect())
	if calls := f.out.calls(); !reflect.DeepEqual(calls, tc.outbound) {
		t.Fatalf("outbound %v, want %v (exchange must precede the failed write)", calls, tc.outbound)
	}
	visible := rec.Body.String() + rec.Header().Get("Location") + logs
	leaks := append([]string{"success", "SECRET", "state=", "synthetic-code", "forced", "UPDATE", "Error 1", "synthetic-page-token", "synthetic-zalo"}, tc.tokens...)
	for _, leak := range leaks {
		if strings.Contains(visible, leak) {
			t.Fatalf("%q exposed in response or log: %s", leak, visible)
		}
	}
	if !strings.Contains(logs, "not persisted") || !strings.Contains(logs, tc.channelID(f)) {
		t.Fatalf("log should name the failure class and channel: %s", logs)
	}
}

func (f *channelConfigFixture) addChannelUpdateTrigger(t *testing.T, channelID string) {
	t.Helper()
	name := "trg_oauthp_" + strings.ReplaceAll(channelID, "-", "_")
	// The ID is generated alphanumerics; trigger bodies cannot bind parameters.
	sql := fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE ON channels FOR EACH ROW BEGIN IF OLD.id = '%s' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced update failure'; END IF; END", name, channelID)
	if err := db.DB.Exec(sql).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + name) })
}

func TestOAuthCallbackUpdateErrorIsNotSuccess(t *testing.T) {
	for _, tc := range oauthPersistCases() {
		t.Run(tc.name, func(t *testing.T) {
			f := setupChannelConfigFixture(t)
			f.addChannelUpdateTrigger(t, tc.channelID(f))
			before := f.rows(t)
			logs := captureLog(t)

			rec := tc.call(f)

			f.assertPersistenceFailure(t, tc, rec, logs.String())
			if after := f.rows(t); !reflect.DeepEqual(before, after) {
				t.Fatalf("channels changed despite the failed write:\nbefore %+v\nafter  %+v", before, after)
			}
		})
	}
}

// The channel is reassigned to another tenant after the authorized lookup, at
// the token step. An ID-only write would still hit it (and store the caller's
// tokens on a foreign tenant's row); the tenant-scoped write changes 0 rows.
func TestOAuthCallbackZeroRowUpdateIsNotSuccess(t *testing.T) {
	for _, tc := range oauthPersistCases() {
		t.Run(tc.name, func(t *testing.T) {
			f := setupChannelConfigFixture(t)
			channelID := tc.channelID(f)
			before := f.rows(t)
			reassigned := 0
			f.out.hook = func(_, hostPath string) {
				if hostPath == tc.tokenPath {
					reassigned++
					if err := db.DB.Exec("UPDATE channels SET tenant_id = ? WHERE id = ?", f.otherTenantID, channelID).Error; err != nil {
						t.Errorf("reassign channel: %v", err)
					}
				}
			}
			logs := captureLog(t)

			rec := tc.call(f)

			if reassigned != 1 {
				t.Fatalf("race hook fired %d times, want 1; the zero-row case would be vacuous", reassigned)
			}
			f.assertPersistenceFailure(t, tc, rec, logs.String())
			if got := f.channelTenant(t, channelID); got != f.otherTenantID {
				t.Fatalf("channel tenant %q, want the reassigned %q", got, f.otherTenantID)
			}
			// Only the tenant moved: no token, external ID, name or timestamp
			// was written to the foreign tenant's row.
			after := f.rows(t)
			for i := range before {
				if before[i].ID == channelID {
					before[i].TenantID = f.otherTenantID
				}
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("channel data written to a foreign tenant's row:\nbefore %+v\nafter  %+v", before, after)
			}
		})
	}
}

// Detector for the failure tests: the same fixture, hooks and observers see
// the accepted callback change exactly one tenant-scoped row.
func TestOAuthCallbackSuccessPersistsExactlyOneRow(t *testing.T) {
	for _, tc := range oauthPersistCases() {
		t.Run(tc.name, func(t *testing.T) {
			f := setupChannelConfigFixture(t)
			channelID := tc.channelID(f)
			hooked := 0
			f.out.hook = func(_, hostPath string) {
				if hostPath == tc.tokenPath {
					hooked++
				}
			}
			before := f.rows(t)

			rec := tc.call(f)

			assertResponse(t, rec, http.StatusFound, tc.success(f))
			if hooked != 1 {
				t.Fatalf("hook fired %d times, want 1", hooked)
			}
			if calls := f.out.calls(); !reflect.DeepEqual(calls, tc.outbound) {
				t.Fatalf("outbound %v, want %v", calls, tc.outbound)
			}
			after := f.rows(t)
			changed := 0
			for i := range before {
				if !reflect.DeepEqual(before[i], after[i]) {
					changed++
					if before[i].ID != channelID || after[i].TenantID != f.tenantID {
						t.Fatalf("unexpected row changed: %+v -> %+v", before[i], after[i])
					}
				}
			}
			if changed != 1 {
				t.Fatalf("%d rows changed, want exactly 1", changed)
			}
			creds := f.credentials(t, channelID)
			if tc.name == "zalo" && (creds["access_token"] != "synthetic-zalo-access" || creds["refresh_token"] != "synthetic-zalo-refresh") {
				t.Fatalf("stored credentials %v", creds)
			}
			if tc.name == "facebook" && (creds["access_token"] != "synthetic-page-token" || creds["page_id"] != "synthetic-page") {
				t.Fatalf("stored credentials %v", creds)
			}
		})
	}
}
