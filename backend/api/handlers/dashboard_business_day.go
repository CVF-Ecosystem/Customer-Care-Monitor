package handlers

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// CCMAI-RUNTIME-026 (F04): Dashboard calendar aggregates in Vietnam business days.
//
// Storage-aware grouping: a day bucket is chosen by comparing the column with typed day-boundary
// instants (CASE WHEN col >= boundary_i ...), never by DATE(col) on the stored wall time. The
// driver serializes each boundary with its own location, exactly as it does for every other
// predicate, so the buckets are correct whether the connection runs loc=UTC or loc=Local/VN and
// no named-zone tables, session time_zone or DSN change are needed. Cost: one grouped scan per
// series over the tenant's rows in the 31-day window (the same rows the DATE() version scanned)
// plus a 31-way CASE per row.

// dashboardSeriesDays is the number of calendar days before today the series reach back
// (today minus 30 days -> 31 dates including today), as before this tranche.
const dashboardSeriesDays = 30

type dashboardWindow struct {
	todayStart    time.Time
	tomorrowStart time.Time
	monthStart    time.Time
	nextMonth     time.Time
	seriesStart   time.Time
}

// newDashboardWindow derives every calendar boundary from one clock instant.
func newDashboardWindow(now time.Time) dashboardWindow {
	today := businessDayStart(now)
	month := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, businessLocation())
	return dashboardWindow{
		todayStart:    today,
		tomorrowStart: today.AddDate(0, 0, 1),
		monthStart:    month,
		nextMonth:     month.AddDate(0, 1, 0),
		seriesStart:   today.AddDate(0, 0, -dashboardSeriesDays),
	}
}

// dashboardRange is the filter interval: today when no date is supplied, otherwise only the sides
// the caller gave.
func dashboardRange(w dashboardWindow, fromStr, toStr string) (businessRange, error) {
	if fromStr == "" && toStr == "" {
		from, to := w.todayStart, w.tomorrowStart
		return businessRange{From: &from, ToExclusive: &to}, nil
	}
	return parseBusinessRange(fromStr, toStr)
}

// dayBucketExpr returns a CASE expression that maps col to the index (0..n-1) of the business day
// starting at starts[i], and its arguments. Rows must already be restricted to [starts[0], end).
func dayBucketExpr(col string, starts []time.Time) (string, []interface{}) {
	var sb strings.Builder
	sb.WriteString("CASE")
	args := make([]interface{}, 0, len(starts))
	for i := len(starts) - 1; i >= 1; i-- {
		sb.WriteString(" WHEN " + col + " >= ? THEN ")
		sb.WriteString(strconv.Itoa(i))
		args = append(args, starts[i])
	}
	sb.WriteString(" ELSE 0 END")
	return sb.String(), args
}

func (w dashboardWindow) seriesDayStarts() []time.Time {
	starts := make([]time.Time, 0, dashboardSeriesDays+1)
	for i := 0; i <= dashboardSeriesDays; i++ {
		starts = append(starts, w.seriesStart.AddDate(0, 0, i))
	}
	return starts
}

type dashboardDayCost struct {
	Date         string  `json:"date"`
	TotalCost    float64 `json:"total_cost"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CallCount    int64   `json:"call_count"`
}

type dashboardDayMessages struct {
	Date       string `json:"date"`
	Count      int64  `json:"count"`
	ChatCount  int64  `json:"chat_count"`  // distinct conversations with customer messages
	ReplyCount int64  `json:"reply_count"` // agent replies
}

// dashboardCostByDay returns the series newest first, like the DATE() version.
func dashboardCostByDay(tenantID string, w dashboardWindow) ([]dashboardDayCost, error) {
	starts := w.seriesDayStarts()
	expr, args := dayBucketExpr("created_at", starts)
	var rows []struct {
		Bucket       int
		TotalCost    float64
		InputTokens  int64
		OutputTokens int64
		CallCount    int64
	}
	err := db.DB.Model(&models.AIUsageLog{}).
		Where("tenant_id = ? AND created_at >= ? AND created_at < ?", tenantID, w.seriesStart, w.tomorrowStart).
		Select(expr+" AS bucket, SUM(cost_usd) AS total_cost, SUM(input_tokens) AS input_tokens, SUM(output_tokens) AS output_tokens, COUNT(*) AS call_count", args...).
		Group("bucket").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]dashboardDayCost, 0, len(rows))
	for _, r := range rows {
		if r.Bucket < 0 || r.Bucket >= len(starts) {
			continue
		}
		out = append(out, dashboardDayCost{Date: businessDayKey(starts[r.Bucket]), TotalCost: r.TotalCost, InputTokens: r.InputTokens, OutputTokens: r.OutputTokens, CallCount: r.CallCount})
	}
	sortDayCostsDesc(out)
	return out, nil
}

// dashboardMessagesByDay returns the series oldest first.
func dashboardMessagesByDay(tenantID string, w dashboardWindow) ([]dashboardDayMessages, error) {
	starts := w.seriesDayStarts()
	expr, args := dayBucketExpr("sent_at", starts)
	var rows []struct {
		Bucket     int
		Count      int64
		ChatCount  int64
		ReplyCount int64
	}
	err := db.DB.Model(&models.Message{}).
		Where("tenant_id = ? AND sent_at >= ? AND sent_at < ?", tenantID, w.seriesStart, w.tomorrowStart).
		Select(expr+` AS bucket, COUNT(*) AS count,
			COUNT(DISTINCT CASE WHEN sender_type = 'customer' THEN conversation_id END) AS chat_count,
			SUM(CASE WHEN sender_type = 'agent' THEN 1 ELSE 0 END) AS reply_count`, args...).
		Group("bucket").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]dashboardDayMessages, 0, len(rows))
	for _, r := range rows {
		if r.Bucket < 0 || r.Bucket >= len(starts) {
			continue
		}
		out = append(out, dashboardDayMessages{Date: businessDayKey(starts[r.Bucket]), Count: r.Count, ChatCount: r.ChatCount, ReplyCount: r.ReplyCount})
	}
	sortDayMessagesAsc(out)
	return out, nil
}

func sortDayCostsDesc(rows []dashboardDayCost) {
	sort.Slice(rows, func(i, j int) bool { return rows[i].Date > rows[j].Date })
}

func sortDayMessagesAsc(rows []dashboardDayMessages) {
	sort.Slice(rows, func(i, j int) bool { return rows[i].Date < rows[j].Date })
}

// dashboardCostSum returns COALESCE(SUM(cost_usd), 0) over [from, to) for the tenant.
func dashboardCostSum(tenantID string, from, to time.Time) (float64, error) {
	var total float64
	err := db.DB.Model(&models.AIUsageLog{}).
		Where("tenant_id = ? AND created_at >= ? AND created_at < ?", tenantID, from, to).
		Select("COALESCE(SUM(cost_usd), 0)").Scan(&total).Error
	return total, err
}
