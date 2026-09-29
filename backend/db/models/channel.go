package models

import "time"

type Channel struct {
	ID                   string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID             string     `gorm:"type:char(36);not null;index:idx_channel_tenant_active" json:"tenant_id"`
	ChannelType          string     `gorm:"type:varchar(20);not null" json:"channel_type"` // zalo_oa | facebook | pancake
	Name                 string     `gorm:"type:varchar(255);not null" json:"name"`
	ExternalID           string     `gorm:"type:varchar(255)" json:"external_id"`
	CredentialsEncrypted []byte     `gorm:"type:varbinary(2048);not null" json:"-"`
	IsActive             bool       `gorm:"default:true;index:idx_channel_tenant_active" json:"is_active"`
	LastSyncAt           *time.Time `json:"last_sync_at"`
	LastSyncStatus       string     `gorm:"type:varchar(20)" json:"last_sync_status"`
	LastSyncError        string     `gorm:"type:text" json:"last_sync_error,omitempty"`
	Metadata             string     `gorm:"type:json" json:"metadata"`
	CreatedAt            time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt            time.Time  `gorm:"not null" json:"updated_at"`

	// SyncRunID identifies the admitted run that currently owns "syncing"
	// (CCMAI-RUNTIME-014). Internal only: never serialized, never backfilled;
	// NULL on idle rows and on legacy rows created before this column existed.
	SyncRunID *string `gorm:"type:varchar(36)" json:"-"`
	// SyncLeaseUntil is the database-time lease deadline of a run admitted by
	// an R016 binary for a GET-only channel type (CCMAI-RUNTIME-016). Internal
	// only; NULL for every other run and for rows from older binaries, which
	// therefore can never be released by lease expiry.
	SyncLeaseUntil *time.Time `gorm:"type:datetime(3)" json:"-"`

	// IsDemoFixture marks the invented channels created by the demo import
	// (CCMAI-RUNTIME-018). Server-owned: set only by the demo importer and the
	// exact legacy backfill, never serialized and never taken from request
	// input or channel metadata. Marked rows can never be admitted to a sync.
	IsDemoFixture bool `gorm:"not null;default:false" json:"-"`

	Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (Channel) TableName() string {
	return "channels"
}

// Unique constraint: tenant + type + external_id
// Added via migration hook
