package models

import "time"

// InterventionDataRevision records one immutable local source observation. It
// does not belong to a JobRun and does not replace an AnalysisSnapshot.
type InterventionDataRevision struct {
	ID                     string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID               string    `gorm:"type:char(36);not null;index:idx_data_revision_case,priority:1" json:"tenant_id"`
	CaseID                 string    `gorm:"type:char(36);not null;index:idx_data_revision_case,priority:2" json:"case_id"`
	JobID                  string    `gorm:"type:char(36);not null" json:"job_id"`
	ConversationID         string    `gorm:"type:char(36);not null" json:"conversation_id"`
	ActorID                string    `gorm:"type:char(36);not null" json:"actor_id"`
	OriginatingCaseVersion int       `gorm:"not null" json:"originating_case_version"`
	CaseVersion            int       `gorm:"not null" json:"case_version"`
	OriginalSnapshotID     string    `gorm:"type:char(36);not null" json:"original_snapshot_id"`
	OriginalSnapshotDigest string    `gorm:"type:char(64);not null" json:"original_snapshot_digest"`
	Digest                 string    `gorm:"type:char(64);not null" json:"digest"`
	Manifest               string    `gorm:"type:mediumtext;not null" json:"-"`
	Coverage               string    `gorm:"type:varchar(16);not null" json:"coverage"`
	CoverageReasons        string    `gorm:"type:json;not null" json:"coverage_reasons"`
	MessageCount           int       `gorm:"not null" json:"message_count"`
	ObservationStatus      string    `gorm:"type:varchar(40);not null" json:"observation_status"`
	Reason                 string    `gorm:"type:varchar(500);not null" json:"reason"`
	CreatedAt              time.Time `gorm:"not null;index:idx_data_revision_case,priority:3" json:"created_at"`
}

// InterventionRerunRequest is only a nonexecuting intent. No worker consumes
// these rows in R102.
type InterventionRerunRequest struct {
	ID                     string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID               string    `gorm:"type:char(36);not null;uniqueIndex:uq_rerun_request_key,priority:1" json:"tenant_id"`
	CaseID                 string    `gorm:"type:char(36);not null;uniqueIndex:uq_rerun_request_key,priority:2;uniqueIndex:uq_rerun_revision,priority:1" json:"case_id"`
	JobID                  string    `gorm:"type:char(36);not null" json:"job_id"`
	ConversationID         string    `gorm:"type:char(36);not null" json:"conversation_id"`
	DataRevisionID         string    `gorm:"type:char(36);not null;uniqueIndex:uq_rerun_revision,priority:2" json:"data_revision_id"`
	DataRevisionDigest     string    `gorm:"type:char(64);not null" json:"data_revision_digest"`
	OriginalSnapshotID     string    `gorm:"type:char(36);not null" json:"original_snapshot_id"`
	OriginalSnapshotDigest string    `gorm:"type:char(64);not null" json:"original_snapshot_digest"`
	ExpectedVersion        int       `gorm:"not null" json:"expected_version"`
	CaseVersion            int       `gorm:"not null" json:"case_version"`
	ActorID                string    `gorm:"type:char(36);not null" json:"actor_id"`
	Reason                 string    `gorm:"type:varchar(500);not null" json:"reason"`
	RequestKey             string    `gorm:"type:varchar(128);not null;uniqueIndex:uq_rerun_request_key,priority:3" json:"request_key"`
	PayloadHash            string    `gorm:"type:char(64);not null" json:"-"`
	Status                 string    `gorm:"type:varchar(40);not null" json:"status"`
	Execution              string    `gorm:"type:varchar(24);not null" json:"execution"`
	CreatedAt              time.Time `gorm:"not null;index:idx_rerun_request_case,priority:3" json:"created_at"`
}

// InterventionExecutionGrant is installed by a server-owned, separately
// authorized path. No HTTP route issues grants or accepts provider settings.
type InterventionExecutionGrant struct {
	ID                  string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID            string    `gorm:"type:char(36);not null;uniqueIndex:uq_intervention_grant_request,priority:1" json:"tenant_id"`
	JobID               string    `gorm:"type:char(36);not null" json:"job_id"`
	CaseID              string    `gorm:"type:char(36);not null" json:"case_id"`
	RequestID           string    `gorm:"type:char(36);not null;uniqueIndex:uq_intervention_grant_request,priority:2" json:"request_id"`
	DataRevisionID      string    `gorm:"type:char(36);not null" json:"data_revision_id"`
	ActorID             string    `gorm:"type:char(36);not null" json:"actor_id"`
	Provider            string    `gorm:"type:varchar(40);not null" json:"provider"`
	Model               string    `gorm:"type:varchar(100);not null" json:"model"`
	PolicyVersion       string    `gorm:"type:varchar(40);not null" json:"policy_version"`
	PromptVersion       string    `gorm:"type:varchar(40);not null" json:"prompt_version"`
	ConfigRef           string    `gorm:"type:varchar(120);not null" json:"config_ref"`
	IssuerID            string    `gorm:"type:char(36);not null" json:"issuer_id"`
	EvidenceRef         string    `gorm:"type:varchar(255);not null" json:"evidence_ref"`
	Deadline            time.Time `gorm:"not null" json:"deadline"`
	State               string    `gorm:"type:varchar(20);not null" json:"state"`
	ConsumedExecutionID *string   `gorm:"type:char(36)" json:"consumed_execution_id,omitempty"`
	CreatedAt           time.Time `gorm:"not null" json:"created_at"`
}

// One execution per tenant/request is permanent, including terminal holds.
type InterventionExecution struct {
	ID                     string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID               string    `gorm:"type:char(36);not null;uniqueIndex:uq_intervention_execution_request,priority:1;uniqueIndex:uq_intervention_execution_run,priority:1;uniqueIndex:uq_intervention_execution_grant,priority:1" json:"tenant_id"`
	CaseID                 string    `gorm:"type:char(36);not null" json:"case_id"`
	JobID                  string    `gorm:"type:char(36);not null" json:"job_id"`
	ConversationID         string    `gorm:"type:char(36);not null" json:"conversation_id"`
	RequestID              string    `gorm:"type:char(36);not null;uniqueIndex:uq_intervention_execution_request,priority:2" json:"request_id"`
	RevisionID             string    `gorm:"type:char(36);not null" json:"revision_id"`
	GrantID                string    `gorm:"type:char(36);not null;uniqueIndex:uq_intervention_execution_grant,priority:2" json:"grant_id"`
	RunID                  string    `gorm:"type:char(36);not null;uniqueIndex:uq_intervention_execution_run,priority:2" json:"run_id"`
	ActorID                string    `gorm:"type:char(36);not null" json:"actor_id"`
	BodyHash               string    `gorm:"type:char(64);not null" json:"-"`
	OriginalRunID          string    `gorm:"type:char(36);not null" json:"original_run_id"`
	OriginalSnapshotID     string    `gorm:"type:char(36);not null" json:"original_snapshot_id"`
	OriginalSnapshotDigest string    `gorm:"type:char(64);not null" json:"original_snapshot_digest"`
	SnapshotID             string    `gorm:"type:char(36);not null" json:"snapshot_id"`
	SnapshotDigest         string    `gorm:"type:char(64);not null" json:"snapshot_digest"`
	ManifestDigest         string    `gorm:"type:char(64);not null" json:"manifest_digest"`
	ExpectedCaseVersion    int       `gorm:"not null" json:"expected_case_version"`
	ClaimedCaseVersion     int       `gorm:"not null" json:"claimed_case_version"`
	ConfigRef              string    `gorm:"type:varchar(120);not null" json:"config_ref"`
	State                  string    `gorm:"type:varchar(30);not null" json:"state"`
	TerminalCode           string    `gorm:"type:varchar(60)" json:"terminal_code"`
	CreatedAt              time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt              time.Time `gorm:"not null" json:"updated_at"`
}
