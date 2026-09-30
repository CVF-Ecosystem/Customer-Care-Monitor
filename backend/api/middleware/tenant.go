package middleware

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// TenantContext extracts tenant_id from URL param and verifies user has access.
func TenantContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Param("tenantId")
		if tenantID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "tenant_id_required"})
			return
		}

		userID := GetUserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization_required"})
			return
		}

		// Check user has access to this tenant
		var ut models.UserTenant
		result := db.DB.Where("user_id = ? AND tenant_id = ?", userID, tenantID).First(&ut)
		if result.Error != nil {
			log.Printf("[security] tenant access denied: user=%s tenant=%s ip=%s", userID, tenantID, c.ClientIP())
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "tenant_access_denied"})
			return
		}

		c.Set("tenant_id", tenantID)
		c.Set("tenant_role", ut.Role)
		c.Set("tenant_permissions", ut.Permissions)
		c.Next()
	}
}

// GetTenantID extracts tenant ID from gin context.
func GetTenantID(c *gin.Context) string {
	if v, exists := c.Get("tenant_id"); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetTenantRole extracts tenant role from gin context.
func GetTenantRole(c *gin.Context) string {
	if v, exists := c.Get("tenant_role"); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// RequireRole checks that the user has at least the specified role.
func RequireRole(roles ...string) gin.HandlerFunc {
	roleMap := make(map[string]bool)
	for _, r := range roles {
		roleMap[r] = true
	}
	return func(c *gin.Context) {
		role := GetTenantRole(c)
		if !roleMap[role] {
			log.Printf("[security] RBAC denied: user=%s role=%s required=%v path=%s", GetUserID(c), role, roles, c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient_role"})
			return
		}
		c.Next()
	}
}

// RequirePermission checks role (owner/admin always pass) or member permission for a resource+action.
// resource: "channels", "jobs", "messages", "settings"
// action: "r" (read), "w" (write), "d" (delete)
func RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deny := PermissionDenial(GetTenantRole(c), c.GetString("tenant_permissions"), resource, action); deny != "" {
			if deny == "permission_denied" {
				log.Printf("[security] permission denied: user=%s resource=%s action=%s path=%s", GetUserID(c), resource, action, c.Request.URL.Path)
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": deny})
			return
		}
		c.Next()
	}
}

// PermissionDenial is the shared tenant permission decision (also used by the HTTP agent
// API). It returns "" when allowed, otherwise the error code: owner/admin always pass; a
// member needs the action letter under the resource in the permissions JSON, and missing
// or malformed permissions are denied.
func PermissionDenial(role, permissionsJSON, resource, action string) string {
	if role == "owner" || role == "admin" {
		return ""
	}
	if permissionsJSON == "" {
		return "no_permissions"
	}
	var permMap map[string]string
	if err := json.Unmarshal([]byte(permissionsJSON), &permMap); err != nil {
		return "invalid_permissions"
	}
	resourcePerms, ok := permMap[resource]
	if !ok || !containsChar(resourcePerms, action) {
		return "permission_denied"
	}
	return ""
}

func containsChar(s, char string) bool {
	for _, c := range s {
		if string(c) == char {
			return true
		}
	}
	return false
}
