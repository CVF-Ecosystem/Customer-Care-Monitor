package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vietbui/chat-quality-agent/api/middleware"
	"github.com/vietbui/chat-quality-agent/db"
	"github.com/vietbui/chat-quality-agent/db/models"
	"gorm.io/gorm"
)

// The existing tenant ID remains an internal scope key for the chat, job and
// storage tables. An installation has exactly one workspace.
type TenantResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	ChannelsCount int64  `json:"channels_count"`
	JobsCount     int64  `json:"jobs_count"`
}

func workspaceResponse(t models.Tenant) TenantResponse {
	var channelsCount, jobsCount int64
	db.DB.Model(&models.Channel{}).Where("tenant_id = ?", t.ID).Count(&channelsCount)
	db.DB.Model(&models.Job{}).Where("tenant_id = ?", t.ID).Count(&jobsCount)
	return TenantResponse{
		ID: t.ID, Name: t.Name, Slug: t.Slug,
		ChannelsCount: channelsCount, JobsCount: jobsCount,
	}
}

func ListTenants(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var workspace models.Tenant
	err := db.DB.Joins("JOIN user_tenants ON user_tenants.tenant_id = tenants.id").
		Where("user_tenants.user_id = ?", userID).First(&workspace).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, []TenantResponse{})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database_error"})
		}
		return
	}
	c.JSON(http.StatusOK, []TenantResponse{workspaceResponse(workspace)})
}

func CreateTenant(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{"error": "single_workspace_only"})
}

func GetTenant(c *gin.Context) {
	var workspace models.Tenant
	if err := db.DB.First(&workspace, "id = ?", middleware.GetTenantID(c)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace_not_found"})
		return
	}
	c.JSON(http.StatusOK, workspaceResponse(workspace))
}

func UpdateTenant(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required,min=2,max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "details": err.Error()})
		return
	}
	result := db.DB.Model(&models.Tenant{}).
		Where("id = ?", middleware.GetTenantID(c)).
		Updates(map[string]interface{}{"name": req.Name, "updated_at": time.Now()})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update_workspace_failed"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace_not_found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func DeleteTenant(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{"error": "single_workspace_cannot_be_deleted"})
}
