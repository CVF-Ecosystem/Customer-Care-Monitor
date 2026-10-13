package handlers

import (
	"net/http"
	"strconv"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/gin-gonic/gin"
)

func GetDashboard(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	role := middleware.GetTenantRole(c)
	if role != "owner" && role != "admin" && role != "member" {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission_denied"})
		return
	}
	permissions := c.GetString("tenant_permissions")
	isPrivileged := role == "owner" || role == "admin"
	canRead := func(resource string) bool {
		return isPrivileged || middleware.PermissionDenial(role, permissions, resource, "r") == ""
	}
	canChannels := canRead("channels")
	canMessages := canRead("messages")
	canJobs := canRead("jobs")
	canSettings := canRead("settings")
	if !canChannels && !canMessages && !canJobs && !canSettings {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission_denied"})
		return
	}

	// CCMAI-RUNTIME-026: every date boundary is a Vietnam calendar boundary derived from one
	// clock instant; the filter is [from, to+1 day) and the default is today.
	w := newDashboardWindow(businessClock())
	rng, err := dashboardRange(w, c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_date_range"})
		return
	}
	unavailable := func() { c.JSON(http.StatusInternalServerError, gin.H{"error": "dashboard_unavailable"}) }
	response := gin.H{}
	if canChannels {
		var activeChannels int64
		db.DB.Model(&models.Channel{}).Where("tenant_id = ? AND is_active = true", tenantID).Count(&activeChannels)
		response["active_channels"] = activeChannels
	}
	if canMessages {
		var totalConversations int64
		if err := rng.where(db.DB.Model(&models.Conversation{}).Where("tenant_id = ?", tenantID), "last_message_at").Count(&totalConversations).Error; err != nil {
			unavailable()
			return
		}
		type ChannelCount struct {
			ChannelType string `json:"channel_type"`
			Count       int64  `json:"count"`
		}
		var channelCounts []ChannelCount
		if err := rng.where(db.DB.Model(&models.Conversation{}).
			Joins("JOIN channels ON channels.id = conversations.channel_id").
			Where("conversations.tenant_id = ?", tenantID), "conversations.last_message_at").
			Select("channels.channel_type, COUNT(*) as count").
			Group("channels.channel_type").Scan(&channelCounts).Error; err != nil {
			unavailable()
			return
		}
		messagesByDay, err := dashboardMessagesByDay(tenantID, w)
		if err != nil {
			unavailable()
			return
		}
		response["total_conversations"] = totalConversations
		response["conversations_by_channel"] = channelCounts
		response["messages_by_day"] = messagesByDay
	}
	if canJobs {
		var activeJobs, issuesInPeriod int64
		db.DB.Model(&models.Job{}).Where("tenant_id = ? AND is_active = true", tenantID).Count(&activeJobs)
		if err := rng.where(db.DB.Model(&models.JobResult{}).Where("tenant_id = ?", tenantID), "created_at").Count(&issuesInPeriod).Error; err != nil {
			unavailable()
			return
		}
		var qcViolationCount int64
		if err := rng.where(db.DB.Model(&models.JobResult{}).Where("tenant_id = ? AND result_type = ?", tenantID, "qc_violation"), "created_at").
			Count(&qcViolationCount).Error; err != nil {
			unavailable()
			return
		}
		const recentActivityLimit = 10
		var qcAlerts []models.JobResult
		if err := rng.where(db.DB.Where("tenant_id = ? AND result_type = 'qc_violation'", tenantID), "created_at").
			Order("created_at DESC").Limit(recentActivityLimit).Find(&qcAlerts).Error; err != nil {
			unavailable()
			return
		}
		type ClassificationItem struct {
			models.JobResult
			CustomerName string `json:"customer_name"`
		}
		var classRecent []ClassificationItem
		if err := rng.where(db.DB.Model(&models.JobResult{}).
			Select("job_results.*, conversations.customer_name").
			Joins("LEFT JOIN conversations ON conversations.id = job_results.conversation_id").
			Where("job_results.tenant_id = ? AND job_results.result_type = 'classification_tag'", tenantID), "job_results.created_at").
			Order("job_results.created_at DESC").Limit(recentActivityLimit).Find(&classRecent).Error; err != nil {
			unavailable()
			return
		}
		response["active_jobs"] = activeJobs
		response["issues"] = issuesInPeriod
		response["qc_violation_count"] = qcViolationCount
		response["qc_alerts"] = qcAlerts
		response["classification_recent"] = classRecent
	}
	if canSettings {
		var costPeriod float64
		if err := rng.where(db.DB.Model(&models.AIUsageLog{}).Where("tenant_id = ?", tenantID), "created_at").
			Select("COALESCE(SUM(cost_usd), 0)").Scan(&costPeriod).Error; err != nil {
			unavailable()
			return
		}
		costToday, err := dashboardCostSum(tenantID, w.todayStart, w.tomorrowStart)
		if err != nil {
			unavailable()
			return
		}
		costMonth, err := dashboardCostSum(tenantID, w.monthStart, w.nextMonth)
		if err != nil {
			unavailable()
			return
		}
		costByDay, err := dashboardCostByDay(tenantID, w)
		if err != nil {
			unavailable()
			return
		}
		exchangeRate := 26000.0
		var rateSetting models.AppSetting
		if db.DB.Where("tenant_id = ? AND setting_key = ?", tenantID, "exchange_rate_vnd").First(&rateSetting).Error == nil && rateSetting.ValuePlain != "" {
			if rate, err := strconv.ParseFloat(rateSetting.ValuePlain, 64); err == nil && rate > 0 {
				exchangeRate = rate
			}
		}
		response["cost_period"] = costPeriod
		response["cost_today"] = costToday
		response["cost_this_month"] = costMonth
		response["cost_by_day"] = costByDay
		response["exchange_rate"] = exchangeRate
	}
	c.JSON(http.StatusOK, response)
}

func GetOnboardingStatus(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	// Step 1: Has channels?
	var channelCount int64
	db.DB.Model(&models.Channel{}).Where("tenant_id = ?", tenantID).Count(&channelCount)

	// Step 2: Has conversations (synced)?
	var convCount int64
	db.DB.Model(&models.Conversation{}).Where("tenant_id = ?", tenantID).Count(&convCount)

	// Step 3: AI configured?
	var aiSetting models.AppSetting
	aiConfigured := db.DB.Where("tenant_id = ? AND setting_key = ? AND value_plain != ''", tenantID, "ai_provider").First(&aiSetting).Error == nil

	// Step 4: Has jobs?
	var jobCount int64
	db.DB.Model(&models.Job{}).Where("tenant_id = ?", tenantID).Count(&jobCount)

	// Step 5: Has job runs?
	var runCount int64
	db.DB.Model(&models.JobRun{}).Where("tenant_id = ?", tenantID).Count(&runCount)

	// Check if dismissed
	var dismissSetting models.AppSetting
	dismissed := false
	if db.DB.Where("tenant_id = ? AND setting_key = ?", tenantID, "onboarding_dismissed").First(&dismissSetting).Error == nil {
		dismissed = dismissSetting.ValuePlain == "true"
	}

	c.JSON(http.StatusOK, gin.H{
		"dismissed": dismissed,
		"steps": []gin.H{
			{"key": "channel", "title": "Kết nối kênh chat", "done": channelCount > 0, "link": "channels"},
			{"key": "sync", "title": "Đồng bộ tin nhắn", "done": convCount > 0, "link": "messages"},
			{"key": "ai", "title": "Cấu hình AI Provider", "done": aiConfigured, "link": "settings"},
			{"key": "job", "title": "Tạo công việc phân tích", "done": jobCount > 0, "link": "jobs/create"},
			{"key": "run", "title": "Chạy thử phân tích", "done": runCount > 0, "link": "jobs"},
		},
	})
}

func ListNotificationLogs(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage > 100 {
		perPage = 100
	}

	var total int64
	db.DB.Model(&models.NotificationLog{}).Where("tenant_id = ?", tenantID).Count(&total)

	var logs []models.NotificationLog
	db.DB.Where("tenant_id = ?", tenantID).Order("sent_at DESC").
		Offset((page - 1) * perPage).Limit(perPage).Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"data":     logs,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}
