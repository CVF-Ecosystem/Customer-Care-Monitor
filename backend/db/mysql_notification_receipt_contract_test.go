package db

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

func TestR091NotificationDeliveryMigrationIsIdempotentAndReceiptKeyFitsMySQL(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Fatal("R091 migration test requires TEST_DB_DSN; setup failure is not a skip")
	}
	Close()
	if err := Connect(dsn, true); err != nil {
		t.Fatalf("connect owned synthetic MySQL: %v", err)
	}
	if err := AutoMigrate(); err != nil {
		t.Fatalf("first AutoMigrate: %v", err)
	}
	for _, table := range []string{"notification_delivery_manifests", "notification_delivery_receipts"} {
		if !DB.Migrator().HasTable(table) {
			t.Fatalf("AutoMigrate did not register %s", table)
		}
	}
	if err := AutoMigrate(); err != nil {
		t.Fatalf("second AutoMigrate: %v", err)
	}

	type indexColumn struct {
		Column string `gorm:"column:column_name"`
		Seq    int    `gorm:"column:seq_in_index"`
	}
	var columns []indexColumn
	if err := DB.Raw(`SELECT column_name, seq_in_index FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = 'notification_delivery_receipts'
		AND index_name = 'uq_delivery_receipt_run_output' AND non_unique = 0 ORDER BY seq_in_index`).Scan(&columns).Error; err != nil {
		t.Fatalf("read receipt uniqueness index: %v", err)
	}
	if !reflect.DeepEqual(columns, []indexColumn{{Column: "job_run_id", Seq: 1}, {Column: "output_key", Seq: 2}}) {
		t.Fatalf("receipt uniqueness index=%+v, want (job_run_id, output_key)", columns)
	}
	type keyMetadata struct {
		MaxBytes int64  `gorm:"column:max_bytes"`
		Charset  string `gorm:"column:charset"`
	}
	var key keyMetadata
	if err := DB.Raw(`SELECT SUM(c.character_maximum_length * 4) AS max_bytes,
		MIN(c.character_set_name) AS charset
		FROM information_schema.statistics s
		JOIN information_schema.columns c ON c.table_schema = s.table_schema AND c.table_name = s.table_name AND c.column_name = s.column_name
		WHERE s.table_schema = DATABASE() AND s.table_name = 'notification_delivery_receipts'
		AND s.index_name = 'uq_delivery_receipt_run_output' AND s.non_unique = 0`).Scan(&key).Error; err != nil {
		t.Fatalf("read utf8mb4 receipt index key length: %v", err)
	}
	if key.Charset != "utf8mb4" || key.MaxBytes != 400 || key.MaxBytes > 3072 {
		t.Fatalf("receipt uniqueness key charset/maximum bytes=%s/%d, want utf8mb4/400 <= 3072", key.Charset, key.MaxBytes)
	}

	first := models.NotificationDeliveryReceipt{
		ID: pkg.NewUUID(), TenantID: pkg.NewUUID(), JobID: pkg.NewUUID(), JobRunID: pkg.NewUUID(),
		ManifestHash: "a", OutputKey: "b", ChannelType: "email", Status: "pending", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := DB.Create(&first).Error; err != nil {
		t.Fatalf("insert first unique receipt: %v", err)
	}
	t.Cleanup(func() { DB.Where("id = ?", first.ID).Delete(&models.NotificationDeliveryReceipt{}) })
	duplicate := first
	duplicate.ID = pkg.NewUUID()
	if err := DB.Create(&duplicate).Error; err == nil {
		t.Fatal("the database accepted a duplicate run/output receipt key")
	}
}
