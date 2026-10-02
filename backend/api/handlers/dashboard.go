package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

func GetDashboard(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	// CCMAI-RUNTIME-026: every date boundary is a Vietnam calendar boundary derived from one
	// clock instant; the filter is [from, to+1 day) and the default is today.
	w := newDashboardWindow(businessClock())
	rng, err := dashboardRange(w, c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_date_range"})
		return
	}
	unavailable := func() { c.JSON(http.StatusInternalServerError, gin.H{"error": "dashboard_unavailable"}) }

	// Static stats (not time-dependent)
	var activeChannels, activeJobs int64
	db.DB.Model(&models.Channel{}).Where("tenant_id = ? AND is_active = true", tenantID).Count(&activeChannels)
	db.DB.Model(&models.Job{}).Where("tenant_id = ? AND is_active = true", tenantID).Count(&activeJobs)

	// Time-dependent stats
	var totalConversations, issuesInPeriod int64
	if err := rng.where(db.DB.Model(&models.Conversation{}).Where("tenant_id = ?", tenantID), "last_message_at").Count(&totalConversations).Error; err != nil {
		unavailable()
		return
	}
	if err := rng.where(db.DB.Model(&models.JobResult{}).Where("tenant_id = ?", tenantID), "created_at").Count(&issuesInPeriod).Error; err != nil {
		unavailable()
		return
	}

	// Số dòng vi phạm QC (không đếm đánh giá hội thoại hay nhãn phân loại). `issues`
	// ở trên vẫn đếm mọi kết quả nên giữ nguyên; lỗi truy vấn này trả 500 chung
	// thay vì một số 0 sai.
	var qcViolationCount int64
	if err := rng.where(db.DB.Model(&models.JobResult{}).Where("tenant_id = ? AND result_type = ?", tenantID, "qc_violation"), "created_at").
		Count(&qcViolationCount).Error; err != nil {
		unavailable()
		return
	}

	// Conversations by channel type
	type ChannelCount struct {
		ChannelType string `json:"channel_type"`
		Count       int64  `json:"count"`
	}
	var channelCounts []ChannelCount
	if err := rng.where(db.DB.Model(&models.Conversation{}).
		Joins("JOIN channels ON channels.id = conversations.channel_id").
		Where("conversations.tenant_id = ?", tenantID), "conversations.last_message_at").
		Select("channels.channel_type, COUNT(*) as count").
		Group("channels.channel_type").
		Scan(&channelCounts).Error; err != nil {
		unavailable()
		return
	}

	// Thẻ "Hoạt động gần đây" gộp hai danh sách rồi lấy recentActivityLimit dòng
	// mới nhất. Mỗi danh sách vì thế phải lấy đủ recentActivityLimit: trước đây
	// phía QC chỉ lấy 5 nên công ty không dùng phân loại thì thẻ vĩnh viễn chỉ có
	// 5 dòng, nhìn như thiếu dữ liệu.
	const recentActivityLimit = 10

	// QC Alerts: only qc_violation (real quality issues)
	var qcAlerts []models.JobResult
	if err := rng.where(db.DB.Where("tenant_id = ? AND result_type = 'qc_violation'", tenantID), "created_at").
		Order("created_at DESC").Limit(recentActivityLimit).Find(&qcAlerts).Error; err != nil {
		unavailable()
		return
	}

	// Classification recent: only classification_tag
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

	// AI cost over the selected interval
	var costPeriod float64
	if err := rng.where(db.DB.Model(&models.AIUsageLog{}).Where("tenant_id = ?", tenantID), "created_at").
		Select("COALESCE(SUM(cost_usd), 0)").Scan(&costPeriod).Error; err != nil {
		unavailable()
		return
	}

	// Chi phí hôm nay tính riêng, không phụ thuộc khoảng thời gian đang lọc — nếu
	// dùng chung một con số thì lọc 28 ngày sẽ ra "hôm nay" lớn hơn "tháng này".
	// Both are bounded above: the current VN day and the current VN month only.
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

	// Cost / messages by Vietnam business day (independent of the selected interval)
	costByDay, err := dashboardCostByDay(tenantID, w)
	if err != nil {
		unavailable()
		return
	}
	messagesByDay, err := dashboardMessagesByDay(tenantID, w)
	if err != nil {
		unavailable()
		return
	}

	// Exchange rate from tenant settings
	exchangeRate := 26000.0
	var rateSetting models.AppSetting
	if db.DB.Where("tenant_id = ? AND setting_key = ?", tenantID, "exchange_rate_vnd").First(&rateSetting).Error == nil && rateSetting.ValuePlain != "" {
		if r, err := strconv.ParseFloat(rateSetting.ValuePlain, 64); err == nil && r > 0 {
			exchangeRate = r
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_conversations":      totalConversations,
		"active_channels":          activeChannels,
		"active_jobs":              activeJobs,
		"issues":                   issuesInPeriod,
		"qc_violation_count":       qcViolationCount,
		"conversations_by_channel": channelCounts,
		"qc_alerts":                qcAlerts,
		"classification_recent":    classRecent,
		"cost_period":              costPeriod,
		"cost_today":               costToday,
		"cost_this_month":          costMonth,
		"cost_by_day":              costByDay,
		"messages_by_day":          messagesByDay,
		"exchange_rate":            exchangeRate,
	})
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
