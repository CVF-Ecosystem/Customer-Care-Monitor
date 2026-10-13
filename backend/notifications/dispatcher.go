package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// OutputConfig represents a single output destination from job.outputs JSON.
type OutputConfig struct {
	Type           string `json:"type"`            // "telegram" | "email"
	BotToken       string `json:"bot_token"`       // telegram
	ChatID         string `json:"chat_id"`         // telegram
	SMTPHost       string `json:"smtp_host"`       // email
	SMTPPort       int    `json:"smtp_port"`       // email
	SMTPUser       string `json:"smtp_user"`       // email
	SMTPPass       string `json:"smtp_pass"`       // email
	From           string `json:"from"`            // email
	To             string `json:"to"`              // email (comma-separated)
	Template       string `json:"template"`        // "default" | "custom"
	CustomTemplate string `json:"custom_template"` // user-defined template (HTML)
}

// Dispatcher sends notifications for job results and logs every send.
type Dispatcher struct{}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

// SendJobResults sends notifications for a job run based on the job's output config.
func (d *Dispatcher) SendJobResults(ctx context.Context, requestedJob models.Job, requestedRun models.JobRun) error {
	set, err := freezeDeliverySet(ctx, requestedJob, requestedRun)
	if err != nil || set == nil {
		return err
	}

	var summary map[string]interface{}
	_ = json.Unmarshal([]byte(set.run.Summary), &summary)
	total := int(getFloat(summary, "conversations_analyzed"))
	passed := int(getFloat(summary, "conversations_passed"))
	failed := total - passed
	issues := int(getFloat(summary, "issues_found"))
	subject := fmt.Sprintf("[Customer Care Monitor AI] %s - %d issues found", set.job.Name, issues)
	link := fmt.Sprintf("%s/%s/jobs/%s", d.getBaseURL(set.job.TenantID), set.job.TenantID, set.job.ID)

	for _, output := range set.outputs {
		if err := ctx.Err(); err != nil {
			return errors.New("notification delivery canceled before all outputs were attempted")
		}
		defaultBody := d.buildNotificationBody(set.job, set.results, output.Identity.Type)
		body := defaultBody
		if output.Config.Template == "custom" && output.Config.CustomTemplate != "" {
			body = d.renderCustomTemplate(output.Config.CustomTemplate, set.job.Name, total, passed, failed, issues, defaultBody, link)
		}
		claim, err := claimDeliveryOutput(set, output, subject, body)
		if err != nil {
			return err
		}
		if !claim.claimed {
			continue
		}

		notifier, createErr := d.createNotifier(output.Config)
		var sendErr error
		if createErr != nil {
			sendErr = failedDelivery("configuration")
		} else {
			sendErr = notifier.Send(ctx, subject, body)
		}
		status, safeError := deliveryOutcome(sendErr)
		if err := finishDeliveryAttempt(set, output, claim.logID, status, safeError); err != nil {
			return err
		}
		if status != deliverySent {
			log.Printf("[dispatcher] output delivery ended as %s for channel %s", status, output.Identity.Type)
		}
	}

	if err := requireAllOutputsSent(set); err != nil {
		return err
	}
	return updateRunNotifiedAt(set)
}

func (d *Dispatcher) createNotifier(cfg OutputConfig) (Notifier, error) {
	switch cfg.Type {
	case "telegram":
		return NewTelegramNotifier(cfg.BotToken, cfg.ChatID), nil
	case "email":
		return NewEmailNotifier(
			cfg.SMTPHost, cfg.SMTPPort,
			cfg.SMTPUser, cfg.SMTPPass,
			cfg.From, splitComma(cfg.To),
		), nil
	default:
		return nil, fmt.Errorf("unsupported output type: %s", cfg.Type)
	}
}

func (d *Dispatcher) buildNotificationBody(job models.Job, results []models.JobResult, outputType string) string {
	body := fmt.Sprintf("<b>Kết quả phân tích: %s</b>\n\n", job.Name)

	for i, r := range results {
		if i >= 10 {
			body += fmt.Sprintf("\n... và %d vấn đề khác\n", len(results)-10)
			break
		}
		switch r.ResultType {
		case "qc_violation":
			emoji := "⚠️"
			if r.Severity == "NGHIEM_TRONG" {
				emoji = "🔴"
			}
			body += fmt.Sprintf("%s <b>%s</b> — %s\n📌 %s\n\n", emoji, r.Severity, r.RuleName, r.Evidence)
		case "classification_tag":
			body += fmt.Sprintf("🏷 <b>%s</b>%s\n📌 %s\n\n", r.RuleName, tagConfidenceNote(r), r.Evidence)
		}
	}

	return body
}

// tagConfidenceNote shows a tag's number only when AfterFind exposed it as
// model-reported, and always says it is the model's own uncalibrated estimate.
// Unknown, invalid or legacy values produce no percentage at all.
func tagConfidenceNote(r models.JobResult) string {
	if r.ReportedConfidence == nil || r.ReportedConfidenceBasis != models.ConfidenceBasisModelReportedUncalibrated {
		return ""
	}
	return fmt.Sprintf(" — mô hình tự ước lượng %.0f%%, chưa hiệu chuẩn", *r.ReportedConfidence*100)
}

func splitComma(s string) []string {
	var result []string
	for _, p := range splitBy(s, ',') {
		p = trimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func splitBy(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func (d *Dispatcher) getBaseURL(tenantID string) string {
	// Priority 1: tenant setting from DB
	var setting models.AppSetting
	if err := db.DB.Where("tenant_id = ? AND setting_key = ?", tenantID, "app_url").First(&setting).Error; err == nil && setting.ValuePlain != "" {
		return setting.ValuePlain
	}
	// Priority 2: environment variable
	if u := os.Getenv("APP_URL"); u != "" {
		return u
	}
	// Priority 3: fallback
	return "http://localhost:8080"
}

func (d *Dispatcher) renderCustomTemplate(tmpl, jobName string, total, passed, failed, issues int, content, link string) string {
	result := tmpl
	replacements := map[string]string{
		"{{job_name}}": jobName,
		"{{total}}":    fmt.Sprintf("%d", total),
		"{{passed}}":   fmt.Sprintf("%d", passed),
		"{{failed}}":   fmt.Sprintf("%d", failed),
		"{{issues}}":   fmt.Sprintf("%d", issues),
		"{{content}}":  content,
		"{{link}}":     link,
	}
	for k, v := range replacements {
		for {
			idx := -1
			for i := 0; i <= len(result)-len(k); i++ {
				if result[i:i+len(k)] == k {
					idx = i
					break
				}
			}
			if idx < 0 {
				break
			}
			result = result[:idx] + v + result[idx+len(k):]
		}
	}
	return result
}

func getFloat(m map[string]interface{}, key string) float64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}
