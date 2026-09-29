package db

import (
	"reflect"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// CCMAI-RUNTIME-016: the nullable internal sync_lease_until column arrives
// through the ordinary AutoMigrate, is idempotent, and is never backfilled, so
// rows from older binaries (including a syncing row with a run ID) keep a NULL
// lease and can never be released by expiry. As in the R014 test, the "old
// schema" is a throwaway table so the shared live channels table is untouched.

type channelOld016 struct {
	ID             string  `gorm:"type:char(36);primaryKey"`
	LastSyncStatus string  `gorm:"type:varchar(20)"`
	SyncRunID      *string `gorm:"type:varchar(36)"`
}

func (channelOld016) TableName() string { return "channels_016_old" }

type channelNew016 struct {
	ID             string     `gorm:"type:char(36);primaryKey"`
	LastSyncStatus string     `gorm:"type:varchar(20)"`
	SyncRunID      *string    `gorm:"type:varchar(36)"`
	SyncLeaseUntil *time.Time `gorm:"type:datetime(3)"`
}

func (channelNew016) TableName() string { return "channels_016_old" }

func TestChannelSyncLeaseColumnOnFreshSchema(t *testing.T) {
	connectMigrationTestDB(t)
	col, ok := columnOf(t, "channels", "sync_lease_until")
	if !ok || col.DataType != "datetime" || col.IsNullable != "YES" {
		t.Fatalf("channels.sync_lease_until = %+v (present %v), want nullable datetime", col, ok)
	}
	if err := AutoMigrate(); err != nil {
		t.Fatalf("second AutoMigrate: %v", err)
	}
}

func TestChannelSyncLeaseMigratesOldSchemaWithoutBackfill(t *testing.T) {
	connectMigrationTestDB(t)
	real, _ := reflect.TypeOf(models.Channel{}).FieldByName("SyncLeaseUntil")
	emulated, _ := reflect.TypeOf(channelNew016{}).FieldByName("SyncLeaseUntil")
	if real.Tag.Get("gorm") != emulated.Tag.Get("gorm") || real.Tag.Get("json") != "-" {
		t.Fatalf("model tags differ: real %q/%q, emulated %q", real.Tag.Get("gorm"), real.Tag.Get("json"), emulated.Tag.Get("gorm"))
	}
	const old = "channels_016_old"
	DB.Exec("DROP TABLE IF EXISTS " + old)
	t.Cleanup(func() { DB.Exec("DROP TABLE IF EXISTS " + old) })

	if err := DB.AutoMigrate(&channelOld016{}); err != nil {
		t.Fatalf("create old schema: %v", err)
	}
	runID := "r015-run-id"
	for _, r := range []channelOld016{{"r015-syncing", "syncing", &runID}, {"legacy-syncing", "syncing", nil}, {"idle", "success", nil}} {
		if err := DB.Create(&r).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	for pass := 1; pass <= 2; pass++ {
		if err := DB.AutoMigrate(&channelNew016{}); err != nil {
			t.Fatalf("pass %d: %v", pass, err)
		}
		if col, ok := columnOf(t, old, "sync_lease_until"); !ok || col.DataType != "datetime" || col.IsNullable != "YES" {
			t.Fatalf("pass %d: sync_lease_until = %+v (present %v)", pass, col, ok)
		}
	}
	var rows []channelNew016
	if err := DB.Order("id").Find(&rows).Error; err != nil || len(rows) != 3 {
		t.Fatalf("read rows: %v (%d)", err, len(rows))
	}
	for _, r := range rows {
		if r.SyncLeaseUntil != nil {
			t.Fatalf("row %s was backfilled with a lease", r.ID)
		}
	}
	if rows[2].ID != "r015-syncing" || rows[2].LastSyncStatus != "syncing" || rows[2].SyncRunID == nil {
		t.Fatalf("old-binary run changed: %+v", rows[2])
	}
}
