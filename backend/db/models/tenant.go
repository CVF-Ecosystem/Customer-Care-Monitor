package models

import "time"

// SingleWorkspaceID is an internal compatibility key. The product has one
// organization; existing tenant-scoped tables still use this key for isolation.
const SingleWorkspaceID = "single-workspace"

type Tenant struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Slug      string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"slug"`
	Settings  string    `gorm:"type:json" json:"settings"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`

	Users    []User    `gorm:"many2many:user_tenants;" json:"users,omitempty"`
	Channels []Channel `gorm:"foreignKey:TenantID" json:"channels,omitempty"`
}
