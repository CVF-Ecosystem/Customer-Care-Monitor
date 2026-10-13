package notifications

import (
	"errors"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type deliveryClaim struct {
	claimed bool
	logID   string
}

func claimDeliveryOutput(set *deliverySet, output resolvedOutput, subject, body string) (deliveryClaim, error) {
	var claim deliveryClaim
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var receipt models.NotificationDeliveryReceipt
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(
			"tenant_id = ? AND job_id = ? AND job_run_id = ? AND output_key = ?",
			set.job.TenantID, set.job.ID, set.run.ID, output.Key,
		).First(&receipt).Error; err != nil {
			return errors.New("notification output receipt could not be read")
		}
		if receipt.ManifestHash != set.manifest.ManifestHash {
			return errors.New("notification output receipt does not match the frozen manifest")
		}
		switch receipt.Status {
		case deliverySent:
			return nil
		case deliverySending, deliveryUnknown:
			return nil // Ambiguous outcomes never initiate an automatic retry.
		case deliveryPending, deliveryFailedStatus:
		default:
			return errors.New("notification output receipt has an unsupported state")
		}
		started := time.Now().UTC()
		logID := pkg.NewUUID()
		entry := models.NotificationLog{
			ID: logID, TenantID: set.job.TenantID, JobID: set.job.ID, JobRunID: set.run.ID,
			ChannelType: output.Identity.Type, Recipient: notificationRecipient(output.Config),
			Subject: subject, Body: body, Status: deliverySending, SentAt: started, CreatedAt: started,
			DeliveryOutputKey: output.Key, DeliveryAttempt: receipt.AttemptCount + 1,
		}
		if err := tx.Create(&entry).Error; err != nil {
			return errors.New("notification attempt log could not be created")
		}
		updated := tx.Model(&models.NotificationDeliveryReceipt{}).Where("id = ? AND status IN ?", receipt.ID, []string{deliveryPending, deliveryFailedStatus}).Updates(map[string]interface{}{
			"status": deliverySending, "attempt_count": receipt.AttemptCount + 1,
			"active_attempt_id": logID, "safe_error": "", "updated_at": started,
		})
		if updated.Error != nil {
			return errors.New("notification output claim could not be persisted")
		}
		if updated.RowsAffected != 1 {
			return errors.New("notification output was claimed concurrently")
		}
		claim = deliveryClaim{claimed: true, logID: logID}
		return nil
	})
	if err != nil {
		return deliveryClaim{}, err
	}
	return claim, nil
}

func notificationRecipient(output OutputConfig) string {
	if output.Type == "email" {
		return output.To
	}
	return output.ChatID
}

func finishDeliveryAttempt(set *deliverySet, output resolvedOutput, logID, status, safeError string) error {
	if status != deliverySent && status != deliveryFailedStatus && status != deliveryUnknown {
		status, safeError = deliveryUnknown, "delivery outcome unknown"
	}
	now := time.Now().UTC()
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var receipt models.NotificationDeliveryReceipt
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(
			"tenant_id = ? AND job_id = ? AND job_run_id = ? AND output_key = ?",
			set.job.TenantID, set.job.ID, set.run.ID, output.Key,
		).First(&receipt).Error; err != nil {
			return errors.New("notification output receipt could not be read after delivery")
		}
		if receipt.Status != deliverySending || receipt.ActiveAttemptID != logID {
			return errors.New("notification output outcome no longer matches its active attempt")
		}
		updates := map[string]interface{}{"status": status, "safe_error": safeError, "updated_at": now}
		if status == deliverySent {
			updates["sent_at"] = now
		}
		changed := tx.Model(&models.NotificationDeliveryReceipt{}).Where("id = ? AND status = ? AND active_attempt_id = ?", receipt.ID, deliverySending, logID).Updates(updates)
		if changed.Error != nil {
			return errors.New("notification output outcome could not be persisted")
		}
		if changed.RowsAffected != 1 {
			return errors.New("notification output outcome lost its active claim")
		}
		logUpdates := map[string]interface{}{"status": status, "error_message": safeError}
		logged := tx.Model(&models.NotificationLog{}).Where("id = ? AND delivery_output_key = ?", logID, output.Key).Updates(logUpdates)
		if logged.Error != nil {
			return errors.New("notification attempt outcome could not be logged")
		}
		if logged.RowsAffected != 1 {
			return errors.New("notification attempt outcome log is missing")
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func requireAllOutputsSent(set *deliverySet) error {
	keys := make([]string, 0, len(set.outputs))
	for _, output := range set.outputs {
		keys = append(keys, output.Key)
	}
	var receipts []models.NotificationDeliveryReceipt
	if err := db.DB.Where("tenant_id = ? AND job_id = ? AND job_run_id = ? AND manifest_hash = ? AND output_key IN ?",
		set.job.TenantID, set.job.ID, set.run.ID, set.manifest.ManifestHash, keys).Find(&receipts).Error; err != nil {
		return errors.New("notification delivery receipts could not be verified")
	}
	if len(receipts) != len(keys) {
		return errors.New("notification delivery is incomplete: required receipts are missing")
	}
	for _, receipt := range receipts {
		if receipt.Status != deliverySent {
			return errors.New("notification delivery is incomplete: one or more outputs are not confirmed sent")
		}
	}
	return nil
}
