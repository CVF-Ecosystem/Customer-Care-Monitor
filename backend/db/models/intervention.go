package models

import "time"

// InterventionBudget is server provisioned. Its tenant/job identity is stable across callers.
// Consumed is monotone; an uncertain request is never refunded automatically.
type InterventionBudget struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	TenantID  string `gorm:"type:char(36);not null;uniqueIndex:uq_intervention_budget,priority:1"`
	JobID     string `gorm:"type:char(36);not null;uniqueIndex:uq_intervention_budget,priority:2"`
	Cap       int    `gorm:"not null"`
	Consumed  int    `gorm:"not null;default:0"`
	CreatedAt time.Time
}

type InterventionAttempt struct {
	ID                  string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID            string    `gorm:"type:char(36);not null;index" json:"tenant_id"`
	JobID               string    `gorm:"type:char(36);not null" json:"job_id"`
	RunID               string    `gorm:"type:char(36);not null;uniqueIndex" json:"run_id"`
	BudgetID            string    `gorm:"type:char(36);not null" json:"-"`
	State               string    `gorm:"type:varchar(30);not null" json:"state"` // CONSUMED / RESPONSE_RECEIVED / UNKNOWN
	UsageState          string    `gorm:"type:varchar(16);not null" json:"usage_state"`
	InputUsagePresence  string    `gorm:"type:varchar(16);not null;default:'unknown'" json:"input_usage_presence"`
	OutputUsagePresence string    `gorm:"type:varchar(16);not null;default:'unknown'" json:"output_usage_presence"`
	InputTokens         *int64    `json:"input_tokens"`
	OutputTokens        *int64    `json:"output_tokens"`
	ResponseModel       string    `gorm:"type:varchar(100)" json:"response_model"`
	CreatedAt           time.Time `json:"created_at"`
}

type InterventionCase struct {
	ID               string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID         string     `gorm:"type:char(36);not null;index:idx_intervention_queue,priority:1;uniqueIndex:uq_intervention_case,priority:1" json:"tenant_id"`
	JobID            string     `gorm:"type:char(36);not null;uniqueIndex:uq_intervention_case,priority:2" json:"job_id"`
	RunID            string     `gorm:"type:char(36);not null" json:"run_id"`
	ConversationID   string     `gorm:"type:char(36);not null;uniqueIndex:uq_intervention_case,priority:3" json:"conversation_id"`
	UseCase          string     `gorm:"type:varchar(40);not null;uniqueIndex:uq_intervention_case,priority:4" json:"use_case"`
	Version          int        `gorm:"not null" json:"version"`
	SnapshotID       *string    `gorm:"type:char(36)" json:"snapshot_id"`
	SnapshotDigest   string     `gorm:"type:char(64)" json:"snapshot_digest"`
	Eligibility      string     `gorm:"type:varchar(20);not null" json:"eligibility"`
	Execution        string     `gorm:"type:varchar(24);not null" json:"execution"`
	Disposition      string     `gorm:"type:varchar(24);not null" json:"disposition"`
	WorkState        string     `gorm:"type:varchar(24);not null;index:idx_intervention_queue,priority:2" json:"work_state"`
	ReasonCode       string     `gorm:"type:varchar(60)" json:"reason_code"`
	NextAction       string     `gorm:"type:varchar(120)" json:"next_action"`
	DataOwnerID      *string    `gorm:"type:char(36)" json:"data_owner_id"`
	AssigneeID       *string    `gorm:"type:char(36)" json:"assignee_id"`
	DueAt            *time.Time `json:"due_at"`
	HumanAssessment  string     `gorm:"type:varchar(24)" json:"human_assessment"`
	HumanExplanation string     `gorm:"type:text" json:"human_explanation"`
	HumanNextAction  string     `gorm:"type:text" json:"human_next_action"`
	OutcomeText      string     `gorm:"type:text" json:"outcome_text"`
	ActorID          string     `gorm:"type:char(36);not null" json:"actor_id"`
	PolicyVersion    string     `gorm:"type:varchar(40);not null" json:"policy_version"`
	PromptVersion    string     `gorm:"type:varchar(40);not null" json:"prompt_version"`
	SchemaVersion    string     `gorm:"type:varchar(40);not null" json:"schema_version"`
	CreatedAt        time.Time  `gorm:"not null;index:idx_intervention_queue,priority:3" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"not null" json:"updated_at"`
}

type InterventionProposal struct {
	ID             string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID       string    `gorm:"type:char(36);not null;index" json:"tenant_id"`
	CaseID         string    `gorm:"type:char(36);not null;uniqueIndex:uq_intervention_proposal_revision,priority:1" json:"case_id"`
	Revision       int       `gorm:"not null;uniqueIndex:uq_intervention_proposal_revision,priority:2" json:"revision"`
	RunID          string    `gorm:"type:char(36);not null;uniqueIndex" json:"run_id"`
	SnapshotID     string    `gorm:"type:char(36);not null" json:"snapshot_id"`
	SnapshotDigest string    `gorm:"type:char(64);not null" json:"snapshot_digest"`
	PromptVersion  string    `gorm:"type:varchar(40);not null" json:"prompt_version"`
	PolicyVersion  string    `gorm:"type:varchar(40);not null" json:"policy_version"`
	SchemaVersion  string    `gorm:"type:varchar(40);not null" json:"schema_version"`
	Kind           string    `gorm:"type:varchar(40);not null" json:"kind"`
	Explanation    string    `gorm:"type:text;not null" json:"explanation"`
	NextAction     string    `gorm:"type:text;not null" json:"next_action"`
	Evidence       string    `gorm:"type:json;not null" json:"evidence"`
	Provider       string    `gorm:"type:varchar(40)" json:"provider"`
	Model          string    `gorm:"type:varchar(100)" json:"model"`
	UsageState     string    `gorm:"type:varchar(16);not null" json:"usage_state"`
	CreatedAt      time.Time `gorm:"not null" json:"created_at"`
}

type InterventionReceipt struct {
	ID         string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID   string    `gorm:"type:char(36);not null;index" json:"tenant_id"`
	CaseID     string    `gorm:"type:char(36);not null;index" json:"case_id"`
	RunID      string    `gorm:"type:char(36);not null" json:"run_id"`
	Code       string    `gorm:"type:varchar(60);not null" json:"code"`
	UsageState string    `gorm:"type:varchar(16);not null" json:"usage_state"`
	CreatedAt  time.Time `gorm:"not null" json:"created_at"`
}

// InterventionHumanEvent is append-only application history. No route updates or
// deletes these rows; the case projection and one event commit in one transaction.
type InterventionHumanEvent struct {
	ID             string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID       string    `gorm:"type:char(36);not null;index" json:"tenant_id"`
	CaseID         string    `gorm:"type:char(36);not null;uniqueIndex:uq_intervention_human_version,priority:1" json:"case_id"`
	CaseVersion    int       `gorm:"not null;uniqueIndex:uq_intervention_human_version,priority:2" json:"case_version"`
	ActorID        string    `gorm:"type:char(36);not null" json:"actor_id"`
	Action         string    `gorm:"type:varchar(32);not null" json:"action"`
	Reason         string    `gorm:"type:varchar(500);not null" json:"reason"`
	ProposalID     *string   `gorm:"type:char(36)" json:"proposal_id"`
	DataRevisionID *string   `gorm:"type:char(36)" json:"data_revision_id,omitempty"`
	RerunRequestID *string   `gorm:"type:char(36)" json:"rerun_request_id,omitempty"`
	SnapshotID     string    `gorm:"type:char(36);not null" json:"snapshot_id"`
	SnapshotDigest string    `gorm:"type:char(64);not null" json:"snapshot_digest"`
	SourceStatus   string    `gorm:"type:varchar(40);not null" json:"source_status"`
	Before         string    `gorm:"type:json;not null" json:"before"`
	After          string    `gorm:"type:json;not null" json:"after"`
	CreatedAt      time.Time `gorm:"not null" json:"created_at"`
}
