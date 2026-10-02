package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

func ListCostLogs(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
	provider := c.Query("provider")

	if page < 1 {
		page = 1
	}
	if perPage > 100 {
		perPage = 100
	}

	// CCMAI-RUNTIME-026: Vietnam business days, [from, to+1 day); list and total share the interval.
	rng, ok := requestBusinessRange(c)
	if !ok {
		return
	}

	query := db.DB.Where("tenant_id = ?", tenantID)
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	query = rng.where(query, "created_at")

	var total int64
	query.Model(&models.AIUsageLog{}).Count(&total)

	var logs []models.AIUsageLog
	query.Order("created_at DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&logs)

	// Get exchange rate from tenant settings
	var rateSetting models.AppSetting
	exchangeRate := 26000.0
	if db.DB.Where("tenant_id = ? AND setting_key = ?", tenantID, "exchange_rate_vnd").First(&rateSetting).Error == nil && rateSetting.ValuePlain != "" {
		if r, err := strconv.ParseFloat(rateSetting.ValuePlain, 64); err == nil && r > 0 {
			exchangeRate = r
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":          logs,
		"total":         total,
		"page":          page,
		"per_page":      perPage,
		"exchange_rate":  exchangeRate,
	})
}
