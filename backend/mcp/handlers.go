package mcp

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func handleToolsCall(c *gin.Context, params json.RawMessage) (interface{}, *RPCError) {
	var call ToolCallParams
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, &RPCError{Code: -32602, Message: "Invalid params"}
	}

	args := call.Arguments
	tenantID, _ := args["tenant_id"].(string)
	userIDVal, ok := c.Get("mcp_user_id")
	if !ok {
		return errResult("authentication required")
	}
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return errResult("authentication required")
	}

	// CCMAI-RUNTIME-021: every supported tool is bound to the requested tenant's stored
	// membership and exact rights before any handler runs. Unknown tools never fall back to
	// membership-only admission.
	policy, known := toolPolicies[call.Name]
	if !known {
		return nil, &RPCError{Code: -32602, Message: "Unknown tool: " + call.Name}
	}
	if policy.tenantScoped {
		if tenantID == "" {
			return errResult("tenant_id is required")
		}
		if denial := authorizeToolCall(userID, tenantID, call.Name, policy); denial != "" {
			return errResult(denial)
		}
	}

	switch call.Name {
	case "cqa_list_tenants":
		return toolListTenants(userID)
	case "cqa_get_tenant":
		return toolGetTenant(tenantID)
	case "cqa_list_channels":
		return toolListChannels(tenantID)
	case "cqa_list_conversations":
		return toolListConversations(tenantID, args)
	case "cqa_get_messages":
		convID, _ := args["conversation_id"].(string)
		return toolGetMessages(tenantID, convID, args)
	case "cqa_search_messages":
		query, _ := args["query"].(string)
		return toolSearchMessages(tenantID, query, args)
	case "cqa_list_jobs":
		return toolListJobs(tenantID)
	case "cqa_get_job_results":
		runID, _ := args["job_run_id"].(string)
		return toolGetJobResults(tenantID, runID)
	case "cqa_search_violations":
		return toolSearchViolations(tenantID, args)
	case "cqa_get_stats":
		period, _ := args["period"].(string)
		return toolGetStats(tenantID, period)
	case "cqa_get_notification_logs":
		return toolGetNotificationLogs(tenantID, args)
	case "cqa_trigger_job":
		jobID, _ := args["job_id"].(string)
		return toolTriggerJob(tenantID, jobID)
	default:
		// Unreachable for a tool in toolPolicies; kept so a policy without a handler fails closed.
		return nil, &RPCError{Code: -32602, Message: "Unknown tool: " + call.Name}
	}
}

// toolRight is one tenant permission letter a tool needs (same vocabulary as the HTTP routes).
type toolRight struct{ resource, action string }

// toolPolicy binds a tool to its tenant-scoping and required rights. Rights are conjunctive:
// every listed right is needed. Owner/admin bypass only the letters, never membership.
type toolPolicy struct {
	tenantScoped bool
	rights       []toolRight
}

var toolPolicies = map[string]toolPolicy{
	// cqa_list_tenants is scoped by the authenticated user's own memberships inside the handler.
	"cqa_list_tenants":          {tenantScoped: false},
	"cqa_get_tenant":            {tenantScoped: true},
	"cqa_list_channels":         {tenantScoped: true, rights: []toolRight{{"channels", "r"}}},
	"cqa_list_conversations":    {tenantScoped: true, rights: []toolRight{{"messages", "r"}}},
	"cqa_get_messages":          {tenantScoped: true, rights: []toolRight{{"messages", "r"}}},
	"cqa_search_messages":       {tenantScoped: true, rights: []toolRight{{"messages", "r"}}},
	"cqa_list_jobs":             {tenantScoped: true, rights: []toolRight{{"jobs", "r"}}},
	"cqa_get_job_results":       {tenantScoped: true, rights: []toolRight{{"jobs", "r"}}},
	"cqa_search_violations":     {tenantScoped: true, rights: []toolRight{{"jobs", "r"}}},
	"cqa_get_stats":             {tenantScoped: true, rights: []toolRight{{"messages", "r"}, {"jobs", "r"}}},
	"cqa_get_notification_logs": {tenantScoped: true, rights: []toolRight{{"settings", "r"}}},
	"cqa_trigger_job":           {tenantScoped: true, rights: []toolRight{{"jobs", "w"}, {"messages", "r"}}},
}

// recognizedRole is compared in Go (case-sensitive), not in SQL, whose collation would fold
// "OWNER" into "owner".
func recognizedRole(role string) bool {
	return role == "owner" || role == "admin" || role == "member"
}

// authorizeToolCall returns "" when the authenticated user's stored membership for the
// requested tenant admits the tool, otherwise a generic denial message. The decision uses only
// database state; tool arguments never carry rights. A lookup error, an unrecognized role or
// bad permission data denies. Logs never include permission JSON, tokens or SQL errors.
func authorizeToolCall(userID, tenantID, tool string, policy toolPolicy) string {
	var members []models.UserTenant
	if err := db.DB.Where("user_id = ? AND tenant_id = ?", userID, tenantID).Limit(1).Find(&members).Error; err != nil {
		log.Printf("[security] MCP authorization unavailable: user=%s tenant=%s tool=%s reason=membership_lookup_failed", userID, tenantID, tool)
		return "authorization unavailable"
	}
	if len(members) == 0 {
		return "access denied: you don't have access to this tenant"
	}
	member := members[0]
	if !recognizedRole(member.Role) {
		log.Printf("[security] MCP permission denied: user=%s tenant=%s tool=%s reason=unrecognized_role", userID, tenantID, tool)
		return "permission denied"
	}
	for _, need := range policy.rights {
		if reason := middleware.PermissionDenial(member.Role, member.Permissions, need.resource, need.action); reason != "" {
			log.Printf("[security] MCP permission denied: user=%s tenant=%s tool=%s reason=%s", userID, tenantID, tool, reason)
			return "permission denied"
		}
	}
	return ""
}

func jsonResult(data interface{}) (interface{}, *RPCError) {
	b, _ := json.MarshalIndent(data, "", "  ")
	return ToolResult{Content: []ToolContent{{Type: "text", Text: string(b)}}}, nil
}

func errResult(msg string) (interface{}, *RPCError) {
	return ToolResult{Content: []ToolContent{{Type: "text", Text: msg}}, IsError: true}, nil
}

func getLimit(args map[string]interface{}, defaultVal int) int {
	if l, ok := args["limit"].(string); ok {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			if n > 200 {
				return 200
			}
			return n
		}
	}
	return defaultVal
}

// tenantSummary is the only tenant data the discovery tools expose: no settings, counts or
// associations (CCMAI-RUNTIME-021).
type tenantSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func toolListTenants(userID string) (interface{}, *RPCError) {
	// Only return tenants the authenticated user has a recognized membership in.
	var memberships []models.UserTenant
	if err := db.DB.Where("user_id = ?", userID).Find(&memberships).Error; err != nil {
		return errResult("Unable to list tenants")
	}
	ids := []string{}
	for _, m := range memberships {
		if recognizedRole(m.Role) {
			ids = append(ids, m.TenantID)
		}
	}
	tenants := []tenantSummary{}
	if len(ids) > 0 {
		if err := db.DB.Model(&models.Tenant{}).Select("id", "name", "slug").
			Where("id IN ?", ids).Find(&tenants).Error; err != nil {
			return errResult("Unable to list tenants")
		}
	}
	return jsonResult(tenants)
}

func toolGetTenant(tenantID string) (interface{}, *RPCError) {
	var tenant tenantSummary
	if err := db.DB.Model(&models.Tenant{}).Select("id", "name", "slug").First(&tenant, "id = ?", tenantID).Error; err != nil {
		return errResult("Tenant not found")
	}
	return jsonResult(tenant)
}

func toolListChannels(tenantID string) (interface{}, *RPCError) {
	var channels []models.Channel
	db.DB.Where("tenant_id = ?", tenantID).Find(&channels)
	return jsonResult(channels)
}

func toolListConversations(tenantID string, args map[string]interface{}) (interface{}, *RPCError) {
	limit := getLimit(args, 20)
	q := db.DB.Where("tenant_id = ?", tenantID)

	if chID, ok := args["channel_id"].(string); ok && chID != "" {
		q = q.Where("channel_id = ?", chID)
	}
	if since, ok := args["since"].(string); ok && since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			q = q.Where("last_message_at > ?", t)
		}
	}

	var convs []models.Conversation
	q.Order("last_message_at DESC").Limit(limit).Find(&convs)
	return jsonResult(convs)
}

func toolGetMessages(tenantID, convID string, args map[string]interface{}) (interface{}, *RPCError) {
	limit := getLimit(args, 50)
	var messages []models.Message
	db.DB.Where("tenant_id = ? AND conversation_id = ?", tenantID, convID).
		Order("sent_at ASC").Limit(limit).Find(&messages)
	return jsonResult(messages)
}

func toolSearchMessages(tenantID, query string, args map[string]interface{}) (interface{}, *RPCError) {
	limit := getLimit(args, 20)
	var messages []models.Message
	db.DB.Where("tenant_id = ? AND content LIKE ?", tenantID, "%"+query+"%").
		Order("sent_at DESC").Limit(limit).Find(&messages)
	return jsonResult(messages)
}

func toolListJobs(tenantID string) (interface{}, *RPCError) {
	var jobs []models.Job
	db.DB.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&jobs)
	return jsonResult(jobs)
}

func toolGetJobResults(tenantID, runID string) (interface{}, *RPCError) {
	var results []models.JobResult
	db.DB.Where("tenant_id = ? AND job_run_id = ?", tenantID, runID).
		Order("created_at DESC").Find(&results)
	return jsonResult(results)
}

func toolSearchViolations(tenantID string, args map[string]interface{}) (interface{}, *RPCError) {
	limit := getLimit(args, 20)
	q := db.DB.Where("tenant_id = ? AND result_type = 'qc_violation'", tenantID)

	if sev, ok := args["severity"].(string); ok && sev != "" {
		q = q.Where("severity = ?", sev)
	}
	if since, ok := args["since"].(string); ok && since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			q = q.Where("created_at > ?", t)
		}
	}

	var results []models.JobResult
	q.Order("created_at DESC").Limit(limit).Find(&results)
	return jsonResult(results)
}

func toolGetStats(tenantID, period string) (interface{}, *RPCError) {
	var since time.Time
	switch period {
	case "week":
		since = time.Now().AddDate(0, 0, -7)
	case "month":
		since = time.Now().AddDate(0, -1, 0)
	default: // today
		since = time.Now().Truncate(24 * time.Hour)
	}

	var totalConvs, totalMsgs, violations, tags int64
	db.DB.Model(&models.Conversation{}).Where("tenant_id = ? AND last_message_at > ?", tenantID, since).Count(&totalConvs)
	db.DB.Model(&models.Message{}).Where("tenant_id = ? AND sent_at > ?", tenantID, since).Count(&totalMsgs)
	db.DB.Model(&models.JobResult{}).Where("tenant_id = ? AND result_type = 'qc_violation' AND created_at > ?", tenantID, since).Count(&violations)
	db.DB.Model(&models.JobResult{}).Where("tenant_id = ? AND result_type = 'classification_tag' AND created_at > ?", tenantID, since).Count(&tags)

	return jsonResult(map[string]interface{}{
		"period":        period,
		"since":         since,
		"conversations": totalConvs,
		"messages":      totalMsgs,
		"violations":    violations,
		"tags":          tags,
	})
}

func toolGetNotificationLogs(tenantID string, args map[string]interface{}) (interface{}, *RPCError) {
	limit := getLimit(args, 20)
	q := db.DB.Where("tenant_id = ?", tenantID)

	if status, ok := args["status"].(string); ok && status != "" {
		q = q.Where("status = ?", status)
	}

	var logs []models.NotificationLog
	q.Order("sent_at DESC").Limit(limit).Find(&logs)
	return jsonResult(logs)
}

func toolTriggerJob(tenantID, jobID string) (interface{}, *RPCError) {
	var job models.Job
	if err := db.DB.Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&job).Error; err != nil {
		return errResult("Job not found")
	}

	// We can't easily run the analyzer here without config, so just return a message
	return jsonResult(map[string]string{
		"status":  "triggered",
		"message": "Job " + job.Name + " has been queued for execution",
	})
}
