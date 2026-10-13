package models

import "time"

// NotificationDeliveryManifest freezes the result IDs and required output identities
// captured for one terminal job run. It never contains credentials or rendered content.
type NotificationDeliveryManifest struct {
	ID                 string    `gorm:"type:char(36);primaryKey" json:"-"`
	TenantID           string    `gorm:"type:char(36);not null;index:idx_delivery_manifest_tenant_run" json:"-"`
	JobRunID           string    `gorm:"type:char(36);not null;uniqueIndex:uq_delivery_manifest_run" json:"-"`
	JobID              string    `gorm:"type:char(36);not null" json:"-"`
	ManifestHash       string    `gorm:"type:char(64);not null" json:"-"`
	ConfigurationHash  string    `gorm:"type:char(64);not null" json:"-"`
	CapturedResultIDs  string    `gorm:"type:json;not null" json:"-"`
	RequiredOutputKeys string    `gorm:"type:json;not null" json:"-"`
	CreatedAt          time.Time `gorm:"not null" json:"-"`
}

func (NotificationDeliveryManifest) TableName() string { return "notification_delivery_manifests" }

// NotificationDeliveryReceipt is the single durable state for one required output
// of a frozen run manifest. A crash left in sending stays ambiguous and is not retried.
type NotificationDeliveryReceipt struct {
	ID              string     `gorm:"type:char(36);primaryKey" json:"-"`
	TenantID        string     `gorm:"type:char(36);not null;index:idx_delivery_receipt_tenant_run" json:"-"`
	JobID           string     `gorm:"type:char(36);not null" json:"-"`
	JobRunID        string     `gorm:"type:char(36);not null;uniqueIndex:uq_delivery_receipt_run_output,priority:1" json:"-"`
	ManifestHash    string     `gorm:"type:char(64);not null" json:"-"`
	OutputKey       string     `gorm:"type:char(64);not null;uniqueIndex:uq_delivery_receipt_run_output,priority:2" json:"-"`
	ChannelType     string     `gorm:"type:varchar(20);not null" json:"-"`
	Status          string     `gorm:"type:varchar(20);not null;index:idx_delivery_receipt_run_status" json:"-"`
	AttemptCount    int        `gorm:"not null;default:0" json:"-"`
	ActiveAttemptID string     `gorm:"type:char(36)" json:"-"`
	SafeError       string     `gorm:"type:varchar(255)" json:"-"`
	SentAt          *time.Time `json:"-"`
	CreatedAt       time.Time  `gorm:"not null" json:"-"`
	UpdatedAt       time.Time  `gorm:"not null" json:"-"`
}

func (NotificationDeliveryReceipt) TableName() string { return "notification_delivery_receipts" }
