package db

import (
	"fmt"
	"log"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(dsn string, isProduction bool) error {
	logLevel := logger.Info
	if isProduction {
		logLevel = logger.Warn
	}

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("Database connected successfully")
	return nil
}

func AutoMigrate() error {
	err := DB.AutoMigrate(
		&models.User{},
		&models.Tenant{},
		&models.UserTenant{},
		&models.Channel{},
		&models.Conversation{},
		&models.Message{},
		&models.Job{},
		&models.JobRun{},
		&models.JobResult{},
		&models.AnalysisSnapshot{},
		&models.AppSetting{},
		&models.NotificationLog{},
		&models.AIUsageLog{},
		&models.OAuthClient{},
		&models.OAuthAuthorizationCode{},
		&models.OAuthToken{},
		&models.ActivityLog{},
	)
	if err != nil {
		return fmt.Errorf("auto-migrate: %w", err)
	}

	// Add unique constraints that GORM can't express directly.
	if err := addUniqueConstraints(); err != nil {
		return err
	}

	// Mark the historical demo fixture channels (CCMAI-RUNTIME-018).
	if err := backfillDemoFixtureChannels(); err != nil {
		return err
	}

	log.Println("Database migration completed")
	return nil
}

func addUniqueConstraints() error {
	constraints := []struct {
		table   string
		name    string
		columns string
	}{
		{"channels", "uq_channel_tenant_type_ext", "tenant_id, channel_type, external_id"},
		{"conversations", "uq_conv_tenant_channel_ext", "tenant_id, channel_id, external_conversation_id"},
		{"messages", "uq_msg_tenant_conv_ext", "tenant_id, conversation_id, external_message_id"},
	}

	for _, c := range constraints {
		if DB.Migrator().HasIndex(c.table, c.name) {
			continue
		}

		sql := fmt.Sprintf(
			"ALTER TABLE `%s` ADD UNIQUE INDEX `%s` (%s)",
			c.table, c.name, c.columns,
		)
		if err := DB.Exec(sql).Error; err != nil {
			return fmt.Errorf("add unique index %s on %s: %w", c.name, c.table, err)
		}
	}

	return nil
}

// legacyDemoCredentials is the exact plaintext the old demo importer stored in
// the credential column. No real channel can hold it: real credentials are
// encrypted.
const legacyDemoCredentials = `{"demo":true}`

// backfillDemoFixtureChannels marks, once and idempotently, only rows that
// match the exact historical fixture identity: a known demo external ID with
// its channel type, the exact plaintext demo credential bytes, and a tenant
// whose settings carry is_demo_data=true. A real channel in a demo tenant, or a
// same-named channel with encrypted credentials, is never marked.
//
// For marked rows only, the old fixture-only decrypt failure is cleared back to
// the never-synced state, and only when no run owns the row. Checkpoints
// (last_sync_at), run IDs, leases and activity logs are never touched.
func backfillDemoFixtureChannels() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE channels c JOIN tenants t ON t.id = c.tenant_id
			SET c.is_demo_fixture = TRUE
			WHERE c.is_demo_fixture = FALSE
			  AND ((c.channel_type = 'zalo_oa' AND c.external_id = 'demo-zalo-oa')
			    OR (c.channel_type = 'facebook' AND c.external_id = 'demo-fb-page'))
			  AND c.credentials_encrypted = ?
			  AND JSON_EXTRACT(t.settings, '$.is_demo_data') = TRUE`, []byte(legacyDemoCredentials)).Error; err != nil {
			return fmt.Errorf("mark demo fixture channels: %w", err)
		}
		if err := tx.Exec(`UPDATE channels
			SET last_sync_status = '', last_sync_error = ''
			WHERE is_demo_fixture = TRUE
			  AND last_sync_at IS NULL
			  AND last_sync_status = 'error'
			  AND last_sync_error LIKE 'decrypt failed:%'
			  AND sync_run_id IS NULL
			  AND sync_lease_until IS NULL`).Error; err != nil {
			return fmt.Errorf("clear demo fixture sync error: %w", err)
		}
		return nil
	})
}

func Close() {
	if DB != nil {
		sqlDB, _ := DB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
}
