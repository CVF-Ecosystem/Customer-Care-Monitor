package db

import (
	"os"
	"reflect"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// CCMAI-RUNTIME-014: the nullable internal sync_run_id column arrives through
// the ordinary startup AutoMigrate, is idempotent, and leaves legacy rows
// (including a NULL-ID "syncing" row) untouched.
//
// Other packages' tests share this disposable database and run in parallel, so
// the live channels table is never altered here. The "old schema" is a
// throwaway table whose model lacks the column, upgraded through GORM
// AutoMigrate (the call the application makes at startup) by a model that adds
// it with the very tag models.Channel uses, which the test asserts.

func connectMigrationTestDB(t *testing.T) {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("bo qua: TEST_DB_DSN chua duoc thiet lap")
	}
	if err := Connect(dsn, false); err != nil {
		t.Skipf("bo qua: khong ket noi duoc DB test: %v", err)
	}
	if err := AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
}

type columnInfo struct {
	DataType   string
	IsNullable string
	MaxLen     int
}

func columnOf(t *testing.T, table, column string) (columnInfo, bool) {
	t.Helper()
	var rows []columnInfo
	err := DB.Raw(`SELECT DATA_TYPE AS data_type, IS_NULLABLE AS is_nullable, COALESCE(CHARACTER_MAXIMUM_LENGTH, 0) AS max_len
		FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`, table, column).Scan(&rows).Error
	if err != nil {
		t.Fatalf("read column: %v", err)
	}
	if len(rows) == 0 {
		return columnInfo{}, false
	}
	return rows[0], true
}

func TestChannelSyncRunIDColumnOnFreshSchema(t *testing.T) {
	connectMigrationTestDB(t)

	col, ok := columnOf(t, "channels", "sync_run_id")
	if !ok || col.DataType != "varchar" || col.IsNullable != "YES" || col.MaxLen != 36 {
		t.Fatalf("channels.sync_run_id = %+v (present %v), want nullable varchar(36)", col, ok)
	}
	if err := AutoMigrate(); err != nil {
		t.Fatalf("second AutoMigrate on the current schema: %v", err)
	}
}

type channelOld014 struct {
	ID             string `gorm:"type:char(36);primaryKey"`
	TenantID       string `gorm:"type:char(36);not null"`
	LastSyncStatus string `gorm:"type:varchar(20)"`
}

func (channelOld014) TableName() string { return "channels_014_old" }

type channelNew014 struct {
	ID             string  `gorm:"type:char(36);primaryKey"`
	TenantID       string  `gorm:"type:char(36);not null"`
	LastSyncStatus string  `gorm:"type:varchar(20)"`
	SyncRunID      *string `gorm:"type:varchar(36)"`
}

func (channelNew014) TableName() string { return "channels_014_old" }

func TestChannelSyncRunIDMigratesOldSchemaIdempotentlyAndKeepsLegacyRows(t *testing.T) {
	connectMigrationTestDB(t)

	// The emulation is only meaningful if it declares the column exactly as the real model does.
	real, _ := reflect.TypeOf(models.Channel{}).FieldByName("SyncRunID")
	emulated, _ := reflect.TypeOf(channelNew014{}).FieldByName("SyncRunID")
	if real.Tag.Get("gorm") != emulated.Tag.Get("gorm") || real.Tag.Get("json") != "-" {
		t.Fatalf("model tags differ: real %q/%q, emulated %q", real.Tag.Get("gorm"), real.Tag.Get("json"), emulated.Tag.Get("gorm"))
	}

	const old = "channels_014_old"
	DB.Exec("DROP TABLE IF EXISTS " + old)
	t.Cleanup(func() { DB.Exec("DROP TABLE IF EXISTS " + old) })

	if err := DB.AutoMigrate(&channelOld014{}); err != nil {
		t.Fatalf("create old schema: %v", err)
	}
	if _, ok := columnOf(t, old, "sync_run_id"); ok {
		t.Fatal("the emulated old schema already has sync_run_id")
	}
	// A legacy "syncing" row and an idle row exist before the migration.
	for _, r := range []channelOld014{{"legacy-syncing", "legacy-tenant", "syncing"}, {"legacy-idle", "legacy-tenant", "success"}} {
		if err := DB.Create(&r).Error; err != nil {
			t.Fatalf("seed legacy row: %v", err)
		}
	}

	for pass := 1; pass <= 2; pass++ {
		if err := DB.AutoMigrate(&channelNew014{}); err != nil {
			t.Fatalf("pass %d: %v", pass, err)
		}
		col, ok := columnOf(t, old, "sync_run_id")
		if !ok || col.DataType != "varchar" || col.IsNullable != "YES" || col.MaxLen != 36 {
			t.Fatalf("pass %d: sync_run_id = %+v (present %v)", pass, col, ok)
		}
	}

	var rows []channelNew014
	if err := DB.Order("id").Find(&rows).Error; err != nil || len(rows) != 2 {
		t.Fatalf("read legacy rows: %v (%d)", err, len(rows))
	}
	for _, r := range rows {
		if r.SyncRunID != nil {
			t.Fatalf("legacy row %s was backfilled with %q", r.ID, *r.SyncRunID)
		}
	}
	if rows[0].ID != "legacy-idle" || rows[0].LastSyncStatus != "success" || rows[1].ID != "legacy-syncing" || rows[1].LastSyncStatus != "syncing" {
		t.Fatalf("legacy statuses changed: %+v", rows)
	}
}
