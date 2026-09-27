package models

import "time"

// AnalysisSnapshot records which version of a conversation a job run analysed.
// Manifest holds IDs, metadata and content hashes, not the raw transcript,
// stored byte-for-byte so SHA-256(manifest) == digest can be re-checked.
type AnalysisSnapshot struct {
	ID              string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID        string    `gorm:"type:char(36);not null;index:idx_snapshot_tenant_conv,priority:1" json:"tenant_id"`
	JobRunID        string    `gorm:"type:char(36);not null;uniqueIndex:idx_snapshot_run_conv,priority:1" json:"job_run_id"`
	ConversationID  string    `gorm:"type:char(36);not null;uniqueIndex:idx_snapshot_run_conv,priority:2;index:idx_snapshot_tenant_conv,priority:2" json:"conversation_id"`
	SchemaVersion   string    `gorm:"type:varchar(40);not null" json:"schema_version"`
	Digest          string    `gorm:"type:char(64);not null;index" json:"digest"`
	Coverage        string    `gorm:"type:varchar(20);not null" json:"coverage"`
	CoverageReasons string    `gorm:"type:json;not null" json:"coverage_reasons"`
	MessageCount    int       `gorm:"not null" json:"message_count"`
	Manifest        string    `gorm:"type:mediumtext;not null" json:"manifest"` // canonical bytes; a JSON column would re-serialize them
	CreatedAt       time.Time `gorm:"not null" json:"created_at"`
}
