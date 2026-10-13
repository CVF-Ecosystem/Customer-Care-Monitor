package notifications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type deliverySet struct {
	job      models.Job
	run      models.JobRun
	manifest models.NotificationDeliveryManifest
	results  []models.JobResult
	outputs  []resolvedOutput
}

type manifestDigestInput struct {
	TenantID          string   `json:"tenant_id"`
	JobID             string   `json:"job_id"`
	JobRunID          string   `json:"job_run_id"`
	ConfigurationHash string   `json:"configuration_hash"`
	ResultIDs         []string `json:"result_ids"`
	OutputKeys        []string `json:"output_keys"`
}

func parseOutputConfigs(raw string) ([]OutputConfig, error) {
	var outputs []OutputConfig
	if err := json.Unmarshal([]byte(raw), &outputs); err == nil {
		return outputs, nil
	}
	var wrapped string
	if err := json.Unmarshal([]byte(raw), &wrapped); err != nil {
		return nil, errors.New("invalid notification output configuration")
	}
	if err := json.Unmarshal([]byte(wrapped), &outputs); err != nil {
		return nil, errors.New("invalid notification output configuration")
	}
	return outputs, nil
}

func freezeDeliverySet(ctx context.Context, requestedJob models.Job, requestedRun models.JobRun) (*deliverySet, error) {
	var result *deliverySet
	err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job models.Job
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", requestedJob.ID, requestedJob.TenantID).First(&job).Error; err != nil {
			return errors.New("notification job binding could not be verified")
		}
		var run models.JobRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ? AND job_id = ?", requestedRun.ID, job.TenantID, job.ID).First(&run).Error; err != nil {
			return errors.New("notification run binding could not be verified")
		}
		configs, err := parseOutputConfigs(job.Outputs)
		if err != nil {
			return err
		}
		outputs, configHash, err := resolveOutputs(configs)
		if err != nil {
			return errors.New("notification output identity could not be derived")
		}
		var existing models.NotificationDeliveryManifest
		err = tx.Where("job_run_id = ? AND tenant_id = ? AND job_id = ?", run.ID, job.TenantID, job.ID).First(&existing).Error
		hasManifest := err == nil
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("notification manifest could not be read")
		}
		if !hasManifest && len(outputs) == 0 {
			result = nil // An empty output list is not a delivery and has no completion marker.
			return nil
		}
		outputKeys := make([]string, 0, len(outputs))
		outputTypes := make(map[string]string, len(outputs))
		for _, output := range outputs {
			outputKeys = append(outputKeys, output.Key)
			outputTypes[output.Key] = output.Identity.Type
		}
		sort.Strings(outputKeys)
		if hasManifest {
			if existing.ConfigurationHash != configHash {
				return errors.New("notification output configuration conflicts with the frozen run manifest")
			}
			var frozenKeys []string
			if err := json.Unmarshal([]byte(existing.RequiredOutputKeys), &frozenKeys); err != nil {
				return errors.New("notification manifest is invalid")
			}
			sort.Strings(frozenKeys)
			if !equalStrings(frozenKeys, outputKeys) || existing.JobRunID != run.ID || existing.TenantID != job.TenantID || existing.JobID != job.ID {
				return errors.New("notification output configuration conflicts with the frozen run manifest")
			}
		}
		var results []models.JobResult
		if hasManifest {
			var resultIDs []string
			if err := json.Unmarshal([]byte(existing.CapturedResultIDs), &resultIDs); err != nil || len(resultIDs) == 0 {
				return errors.New("notification manifest result set is invalid")
			}
			if err := tx.Where("tenant_id = ? AND job_run_id = ? AND id IN ?", job.TenantID, run.ID, resultIDs).Order("id ASC").Find(&results).Error; err != nil {
				return errors.New("captured notification results could not be read")
			}
			if len(results) != len(resultIDs) {
				return errors.New("captured notification result set no longer matches the run manifest")
			}
			actualIDs := make([]string, 0, len(results))
			for _, r := range results {
				actualIDs = append(actualIDs, r.ID)
			}
			sort.Strings(actualIDs)
			if !equalStrings(actualIDs, resultIDs) {
				return errors.New("captured notification result set no longer matches the run manifest")
			}
			if manifestHash(job.TenantID, job.ID, run.ID, configHash, resultIDs, outputKeys) != existing.ManifestHash {
				return errors.New("notification manifest integrity check failed")
			}
		} else {
			if err := tx.Where("tenant_id = ? AND job_run_id = ? AND notified_at IS NULL", job.TenantID, run.ID).Order("id ASC").Find(&results).Error; err != nil {
				return errors.New("unnotified run results could not be read")
			}
			if len(results) == 0 {
				result = nil
				return nil
			}
			resultIDs := make([]string, 0, len(results))
			for _, r := range results {
				resultIDs = append(resultIDs, r.ID)
			}
			sort.Strings(resultIDs)
			keysJSON, _ := json.Marshal(outputKeys)
			idsJSON, _ := json.Marshal(resultIDs)
			manifest := models.NotificationDeliveryManifest{
				ID: pkg.NewUUID(), TenantID: job.TenantID, JobID: job.ID, JobRunID: run.ID,
				ManifestHash:      manifestHash(job.TenantID, job.ID, run.ID, configHash, resultIDs, outputKeys),
				ConfigurationHash: configHash, CapturedResultIDs: string(idsJSON), RequiredOutputKeys: string(keysJSON), CreatedAt: time.Now().UTC(),
			}
			if err := tx.Create(&manifest).Error; err != nil {
				return errors.New("notification manifest could not be persisted")
			}
			for _, output := range outputs {
				receipt := models.NotificationDeliveryReceipt{
					ID: pkg.NewUUID(), TenantID: job.TenantID, JobID: job.ID, JobRunID: run.ID,
					ManifestHash: manifest.ManifestHash, OutputKey: output.Key, ChannelType: output.Identity.Type,
					Status: deliveryPending, CreatedAt: manifest.CreatedAt, UpdatedAt: manifest.CreatedAt,
				}
				if err := tx.Create(&receipt).Error; err != nil {
					return errors.New("notification receipts could not be initialized")
				}
			}
			existing = manifest
		}
		if existing.ConfigurationHash != configHash {
			return errors.New("notification output configuration conflicts with the frozen run manifest")
		}
		result = &deliverySet{job: job, run: run, manifest: existing, results: results, outputs: outputs}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func manifestHash(tenantID, jobID, runID, configHash string, resultIDs, outputKeys []string) string {
	ids := append([]string(nil), resultIDs...)
	keys := append([]string(nil), outputKeys...)
	sort.Strings(ids)
	sort.Strings(keys)
	encoded, _ := json.Marshal(manifestDigestInput{TenantID: tenantID, JobID: jobID, JobRunID: runID, ConfigurationHash: configHash, ResultIDs: ids, OutputKeys: keys})
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func updateRunNotifiedAt(set *deliverySet) error {
	if set == nil || len(set.results) == 0 {
		return nil
	}
	ids := make([]string, 0, len(set.results))
	for _, result := range set.results {
		ids = append(ids, result.ID)
	}
	sort.Strings(ids)
	now := time.Now().UTC()
	markedAt := make(map[string]*time.Time, len(ids))
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var captured []models.JobResult
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "notified_at").
			Where("tenant_id = ? AND job_run_id = ? AND id IN ?", set.job.TenantID, set.run.ID, ids).
			Order("id ASC").Find(&captured).Error; err != nil {
			return errors.New("captured notification results could not be verified")
		}
		if len(captured) != len(ids) {
			return errors.New("captured notification result set no longer matches the run manifest")
		}
		unmarkedIDs := make([]string, 0, len(captured))
		for _, result := range captured {
			markedAt[result.ID] = result.NotifiedAt
			if result.NotifiedAt == nil {
				unmarkedIDs = append(unmarkedIDs, result.ID)
			}
		}
		if len(unmarkedIDs) > 0 {
			update := tx.Model(&models.JobResult{}).
				Where("tenant_id = ? AND job_run_id = ? AND id IN ? AND notified_at IS NULL", set.job.TenantID, set.run.ID, unmarkedIDs).
				Update("notified_at", &now)
			if update.Error != nil {
				return errors.New("captured notification results could not be marked delivered")
			}
			if update.RowsAffected != int64(len(unmarkedIDs)) {
				return errors.New("captured notification result set changed while marking delivered")
			}
			for _, id := range unmarkedIDs {
				markedAt[id] = &now
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for i := range set.results {
		set.results[i].NotifiedAt = markedAt[set.results[i].ID]
	}
	return nil
}
