package models

import (
	"time"

	"gorm.io/gorm"
)

type Job struct {
	ID          string `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID    string `gorm:"type:char(36);not null;index:idx_job_tenant_active" json:"tenant_id"`
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	JobType     string `gorm:"type:varchar(30);not null" json:"job_type"` // qc_analysis | classification

	// Input
	InputChannelIDs string `gorm:"type:json;not null" json:"input_channel_ids"` // JSON array of channel UUIDs

	// Rules
	RulesContent   string `gorm:"type:text" json:"rules_content"`   // Markdown for QC
	RulesConfig    string `gorm:"type:json" json:"rules_config"`    // JSON array for classification
	SkipConditions string `gorm:"type:text" json:"skip_conditions"` // Conditions to skip evaluation (QC only)

	// AI
	AIProvider string `gorm:"column:ai_provider;type:varchar(20);default:'claude'" json:"ai_provider"` // claude | gemini
	AIModel    string `gorm:"type:varchar(100)" json:"ai_model"`

	// Output
	Outputs        string     `gorm:"type:json;not null" json:"outputs"`                         // [{type, config...}]
	OutputSchedule string     `gorm:"type:varchar(20);default:'instant'" json:"output_schedule"` // instant | scheduled | cron
	OutputCron     string     `gorm:"type:varchar(100)" json:"output_cron"`
	OutputAt       *time.Time `json:"output_at"`

	// Analysis schedule
	ScheduleType string `gorm:"type:varchar(20);default:'cron'" json:"schedule_type"` // cron | after_sync | manual
	ScheduleCron string `gorm:"type:varchar(100)" json:"schedule_cron"`

	// State
	IsActive      bool       `gorm:"default:true;index:idx_job_tenant_active" json:"is_active"`
	LastRunAt     *time.Time `json:"last_run_at"`
	LastRunStatus string     `gorm:"type:varchar(20)" json:"last_run_status"`
	CreatedAt     time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"not null" json:"updated_at"`

	Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

type JobRun struct {
	ID           string     `gorm:"type:char(36);primaryKey" json:"id"`
	JobID        string     `gorm:"type:char(36);not null;index:idx_jobrun_job_started" json:"job_id"`
	TenantID     string     `gorm:"type:char(36);not null" json:"tenant_id"`
	StartedAt    time.Time  `gorm:"not null;index:idx_jobrun_job_started" json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Status       string     `gorm:"type:varchar(20);default:'running'" json:"status"` // running | success | error
	Summary      string     `gorm:"type:json" json:"summary"`
	ErrorMessage string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt    time.Time  `gorm:"not null" json:"created_at"`

	Job Job `gorm:"foreignKey:JobID" json:"job,omitempty"`
}

type JobResult struct {
	ID             string `gorm:"type:char(36);primaryKey" json:"id"`
	JobRunID       string `gorm:"type:char(36);not null;index:idx_result_run" json:"job_run_id"`
	TenantID       string `gorm:"type:char(36);not null;index:idx_result_tenant_type" json:"tenant_id"`
	ConversationID string `gorm:"type:char(36);not null;index:idx_result_tenant_conv" json:"conversation_id"`
	// Nil for results created before snapshots existed (legacy/unverified).
	AnalysisSnapshotID *string    `gorm:"type:char(36);index" json:"analysis_snapshot_id"`
	ResultType         string     `gorm:"type:varchar(30);not null;index:idx_result_tenant_type" json:"result_type"` // qc_violation | classification_tag
	Severity           string     `gorm:"type:varchar(30)" json:"severity"`
	RuleName           string     `gorm:"type:varchar(255)" json:"rule_name"`
	Evidence           string     `gorm:"type:text" json:"evidence"`
	Detail             string     `gorm:"type:json" json:"detail"`
	AIRawResponse      string     `gorm:"type:text" json:"ai_raw_response,omitempty"`
	NotifiedAt         *time.Time `json:"notified_at"`
	CreatedAt          time.Time  `gorm:"not null;index:idx_result_tenant_type" json:"created_at"`

	// Confidence and ConfidenceBasis are stored exactly as written and are
	// never serialized directly. Rows written before confidence_basis existed
	// keep their old number (often a placeholder 1.0) with a NULL basis, which
	// means "provenance unknown" and is never exposed.
	Confidence      *float64 `gorm:"column:confidence" json:"-"`
	ConfidenceBasis *string  `gorm:"type:varchar(40)" json:"-"`

	// ReportedConfidence/ReportedConfidenceBasis are what API JSON and
	// notifications show, derived on read by AfterFind. Being separate from
	// the stored columns, a later Save can never rewrite historical values.
	ReportedConfidence      *float64 `gorm:"-" json:"confidence"`
	ReportedConfidenceBasis string   `gorm:"-" json:"confidence_basis"`

	// EvidenceStatus is derived on read: "snapshot_bound" or "legacy_unverified".
	EvidenceStatus string `gorm:"-" json:"evidence_status"`
}

// Confidence basis values. There is deliberately no "calibrated" value: no
// result confidence in this system has been measured against outcomes.
const (
	ConfidenceBasisUnavailable               = "unavailable"
	ConfidenceBasisModelReportedUncalibrated = "model_reported_uncalibrated"
)

// UnavailableConfidence returns the stored pair for a result with no
// numeric confidence (QC evaluations/violations, classification evaluations).
func UnavailableConfidence() (*float64, *string) {
	basis := ConfidenceBasisUnavailable
	return nil, &basis
}

// ModelReportedConfidence returns the stored pair for a classification tag's
// model-supplied number, which callers must already have validated to [0,1].
func ModelReportedConfidence(v float64) (*float64, *string) {
	basis := ConfidenceBasisModelReportedUncalibrated
	return &v, &basis
}

func (r *JobResult) AfterFind(_ *gorm.DB) error {
	r.EvidenceStatus = "legacy_unverified"
	if r.AnalysisSnapshotID != nil && *r.AnalysisSnapshotID != "" {
		r.EvidenceStatus = "snapshot_bound"
	}

	r.ReportedConfidence = nil
	r.ReportedConfidenceBasis = ConfidenceBasisUnavailable
	if r.ResultType == "classification_tag" &&
		r.ConfidenceBasis != nil && *r.ConfidenceBasis == ConfidenceBasisModelReportedUncalibrated &&
		r.Confidence != nil && *r.Confidence >= 0 && *r.Confidence <= 1 {
		v := *r.Confidence
		r.ReportedConfidence = &v
		r.ReportedConfidenceBasis = ConfidenceBasisModelReportedUncalibrated
	}
	return nil
}
