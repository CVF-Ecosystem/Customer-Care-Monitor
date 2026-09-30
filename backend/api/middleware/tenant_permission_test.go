package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// CCMAI-RUNTIME-020: PermissionDenial was extracted from RequirePermission so the HTTP agent
// API and tenant routes share one decision. These cases pin the unchanged semantics, and the
// route test proves RequirePermission still returns the same codes through the extraction.
func TestPermissionDenialSemantics(t *testing.T) {
	cases := []struct {
		name, role, perms, resource, action, want string
	}{
		{"owner bypass", "owner", "", "jobs", "w", ""},
		{"admin bypass with malformed json", "admin", "not-json", "jobs", "w", ""},
		{"member empty", "member", "", "jobs", "r", "no_permissions"},
		{"member malformed", "member", `{"jobs":`, "jobs", "r", "invalid_permissions"},
		{"member wrong value type", "member", `{"jobs":5}`, "jobs", "r", "invalid_permissions"},
		{"member resource missing", "member", `{"channels":"rw"}`, "jobs", "r", "permission_denied"},
		{"member action missing", "member", `{"jobs":"r"}`, "jobs", "w", "permission_denied"},
		{"member allowed", "member", `{"jobs":"rw"}`, "jobs", "w", ""},
		{"non-owner role uses member rule", "viewer", `{"jobs":"r"}`, "jobs", "r", ""},
	}
	for _, tc := range cases {
		if got := PermissionDenial(tc.role, tc.perms, tc.resource, tc.action); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestRequirePermissionStillUsesSharedDecision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		role, perms string
		code        int
		body        string
	}{
		{"owner", "", http.StatusOK, ""},
		{"member", `{"jobs":"rw"}`, http.StatusOK, ""},
		{"member", "", http.StatusForbidden, "no_permissions"},
		{"member", "{", http.StatusForbidden, "invalid_permissions"},
		{"member", `{"jobs":"r"}`, http.StatusForbidden, "permission_denied"},
	} {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("tenant_role", tc.role)
			c.Set("tenant_permissions", tc.perms)
		})
		r.GET("/x", RequirePermission("jobs", "w"), func(c *gin.Context) { c.Status(http.StatusOK) })
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
		if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.body) {
			t.Errorf("%s %q: got %d %s, want %d %s", tc.role, tc.perms, rec.Code, rec.Body.String(), tc.code, tc.body)
		}
	}
}
