package engine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// CCMAI-RUNTIME-025 (F03): ordinary unlimited incremental analysis decides what to analyze by
// comparing each conversation's current full source snapshot with the snapshot of the latest
// committed evaluation of the same job, instead of by event timestamps and a wall-clock
// checkpoint. A late-ingested conversation, a late message with an old timestamp and an edit of
// an already analyzed message therefore stay eligible until a result with their exact source is
// committed; an unchanged conversation is never analyzed again.

// Test seams (private): the analyzer clock, the finalization retry delay and the notification
// sender. Production values are the real ones.
var (
	analyzerNow                = time.Now
	ordinaryFinalizeRetryDelay = 2 * time.Second
	sendJobNotifications       = defaultSendJobNotifications
)

const ordinaryFinalizeAttempts = 3

// isOrdinaryIncremental names the only mode this contract covers: an ordinary run without a
// full rerun, a positive conversation limit, a date range, a since-override or the
// unanalyzed-only filter. Every other mode keeps its previous selection and checkpoint policy.
func isOrdinaryIncremental(fullRerun bool, maxConversations int, dateFrom, dateTo string, sinceOverride *time.Time, excludeAnalyzed bool) bool {
	return !fullRerun && maxConversations <= 0 && dateFrom == "" && dateTo == "" && sinceOverride == nil && !excludeAnalyzed
}

// preparedConversation is a conversation with the exact snapshot that will be sent to the
// provider and persisted with its result.
type preparedConversation struct {
	Conv models.Conversation
	Snap *conversationSnapshot
}

// ErrSourceVersionUnverifiable means the latest evaluation links a snapshot that cannot be
// trusted (missing, misbound, malformed or failing provenance). It is never a successful skip.
var ErrSourceVersionUnverifiable = errors.New("source version of the latest evaluation cannot be verified")

// ordinaryIncrementalCandidates returns every conversation of the job's tenant and input
// channels in a deterministic order (MySQL sorts NULL last_message_at first; nothing is
// filtered by time).
func ordinaryIncrementalCandidates(job models.Job, channelIDs []string) ([]models.Conversation, error) {
	var conversations []models.Conversation
	if len(channelIDs) == 0 {
		return conversations, nil
	}
	err := db.DB.Where("tenant_id = ? AND channel_id IN ?", job.TenantID, channelIDs).
		Order("last_message_at ASC, id ASC").
		Find(&conversations).Error
	return conversations, err
}

// sourceVersionChanged reports whether the conversation's current full snapshot differs from
// the source recorded by the latest committed conversation_evaluation of this job (tenant-,
// job- and conversation-bound; newest created_at, then id). No evaluation, or a legacy
// evaluation without a snapshot, counts as changed. A linked snapshot that is missing or fails
// VerifySnapshotProvenance returns ErrSourceVersionUnverifiable; query errors are returned.
func sourceVersionChanged(job models.Job, conv models.Conversation, snap *conversationSnapshot) (bool, error) {
	var latest []models.JobResult
	if err := db.DB.Model(&models.JobResult{}).
		Select("job_results.id, job_results.job_run_id, job_results.analysis_snapshot_id").
		Joins("JOIN job_runs ON job_runs.id = job_results.job_run_id").
		Where("job_results.tenant_id = ? AND job_results.conversation_id = ? AND job_results.result_type = ?",
			job.TenantID, conv.ID, "conversation_evaluation").
		Where("job_runs.job_id = ? AND job_runs.tenant_id = ?", job.ID, job.TenantID).
		Order("job_results.created_at DESC, job_results.id DESC").
		Limit(1).
		Find(&latest).Error; err != nil {
		return false, fmt.Errorf("find latest evaluation: %w", err)
	}
	if len(latest) == 0 || latest[0].AnalysisSnapshotID == nil {
		return true, nil
	}
	var linked []models.AnalysisSnapshot
	if err := db.DB.Where("id = ? AND tenant_id = ?", *latest[0].AnalysisSnapshotID, job.TenantID).
		Limit(1).Find(&linked).Error; err != nil {
		return false, fmt.Errorf("load linked snapshot: %w", err)
	}
	if len(linked) == 0 || !VerifySnapshotProvenance(linked[0], job.TenantID, conv.ID, latest[0].JobRunID) {
		return false, ErrSourceVersionUnverifiable
	}
	return strings.ToLower(strings.TrimSpace(linked[0].Digest)) != snap.Digest, nil
}

// prepareOrdinaryIncremental loads each candidate's full current snapshot once and keeps only
// conversations whose source changed. The returned snapshots are the ones analyzed and saved.
// Empty source is skipped (it becomes eligible again when messages arrive). Snapshot and
// verification errors are counted as errors, never as skips. Cancellation stops preparation.
func prepareOrdinaryIncremental(ctx context.Context, job models.Job, candidates []models.Conversation) (prepared []preparedConversation, unchanged, errorCount int, cancelled bool) {
	for _, conv := range candidates {
		if ctx.Err() != nil {
			return prepared, unchanged, errorCount, true
		}
		snap, err := loadConversationSnapshot(conv, time.Time{})
		if err != nil {
			log.Printf("[analyzer] snapshot error for conversation %s: %v", conv.ID, err)
			errorCount++
			continue
		}
		if snap.Manifest.Coverage == coverageEmpty {
			continue
		}
		changed, err := sourceVersionChanged(job, conv, snap)
		if err != nil {
			log.Printf("[analyzer] source version check failed for conversation %s: %v", conv.ID, err)
			errorCount++
			continue
		}
		if !changed {
			unchanged++
			continue
		}
		prepared = append(prepared, preparedConversation{Conv: conv, Snap: snap})
	}
	return prepared, unchanged, errorCount, false
}

var (
	errFinalizeWrite   = errors.New("terminal write failed")
	errFinalizeMissing = errors.New("terminal write target is missing")
)

// finalizeOrdinaryRun writes the terminal run status and the job's status/checkpoint together in
// one transaction, scoped to the run's tenant and job, after locking both rows. It retries a
// failed write up to ordinaryFinalizeAttempts times; a missing row is not retried. When every
// attempt fails it makes a best-effort error mark on the run without touching the checkpoint and
// returns the failure, so the caller neither reports success nor sends notifications.
func finalizeOrdinaryRun(run *models.JobRun, job models.Job, runStatus, errorMessage, summary string, finishedAt time.Time, checkpoint *time.Time) error {
	// Match the database's datetime(3) precision for exact read-back verification;
	// completion time keeps milliseconds, independently of the second checkpoint.
	persistedFinish := finishedAt.Round(time.Millisecond)
	var lastErr error
	for attempt := 0; attempt < ordinaryFinalizeAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(ordinaryFinalizeRetryDelay)
		}
		lastErr = db.DB.Transaction(func(tx *gorm.DB) error {
			// Lock order: Job parent first, then the run (CCMAI-RUNTIME-028; same order as admission
			// and the destructive guards).
			var lockedJob []models.Job
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").
				Where("id = ? AND tenant_id = ?", job.ID, job.TenantID).
				Find(&lockedJob).Error; err != nil {
				return errFinalizeWrite
			}
			var lockedRun []models.JobRun
			// Only a run that is still running can be finalized (CCMAI-RUNTIME-028): a stale
			// finalizer for an already terminal run never rewrites the job's status/checkpoint.
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").
				Where("id = ? AND tenant_id = ? AND job_id = ? AND status = ?", run.ID, job.TenantID, job.ID, "running").
				Find(&lockedRun).Error; err != nil {
				return errFinalizeWrite
			}
			if len(lockedRun) != 1 || len(lockedJob) != 1 {
				return errFinalizeMissing
			}
			res := tx.Model(&models.JobRun{}).
				Where("id = ? AND tenant_id = ? AND job_id = ?", run.ID, job.TenantID, job.ID).
				Updates(map[string]interface{}{
					"status":        runStatus,
					"finished_at":   &persistedFinish,
					"summary":       summary,
					"error_message": errorMessage,
				})
			if res.Error != nil || res.RowsAffected > 1 {
				return errFinalizeWrite
			}
			updates := map[string]interface{}{
				"last_run_status": runStatus,
				"updated_at":      persistedFinish,
			}
			if checkpoint != nil {
				updates["last_run_at"] = checkpoint
			}
			res = tx.Model(&models.Job{}).Where("id = ? AND tenant_id = ?", job.ID, job.TenantID).Updates(updates)
			if res.Error != nil || res.RowsAffected > 1 {
				return errFinalizeWrite
			}
			// MySQL may report zero changed rows for an already-identical job update.
			// Accept that only if both locked records actually contain the intended
			// terminal values; a trigger can silently suppress even a one-row write.
			var verified int64
			if err := tx.Model(&models.JobRun{}).
				Where("id = ? AND tenant_id = ? AND job_id = ?", run.ID, job.TenantID, job.ID).
				Where("status = ? AND finished_at = ? AND error_message = ?", runStatus, persistedFinish, errorMessage).
				Where("summary = CAST(? AS JSON)", summary).
				Count(&verified).Error; err != nil || verified != 1 {
				return errFinalizeWrite
			}
			verified = 0
			if err := tx.Model(&models.Job{}).
				Where("id = ? AND tenant_id = ?", job.ID, job.TenantID).
				Where(updates).Count(&verified).Error; err != nil || verified != 1 {
				return errFinalizeWrite
			}
			return nil
		})
		if lastErr == nil {
			return nil
		}
		log.Printf("[analyzer] final run write failed for job %s (attempt %d): %v", job.ID, attempt+1, lastErr)
		if errors.Is(lastErr, errFinalizeMissing) {
			break
		}
	}
	// Best effort, never a checkpoint: do not leave the run "running" if the row still exists.
	if err := db.DB.Model(&models.JobRun{}).
		Where("id = ? AND tenant_id = ? AND job_id = ? AND status = ?", run.ID, job.TenantID, job.ID, "running").
		Updates(map[string]interface{}{
			"status":        "error",
			"finished_at":   &finishedAt,
			"error_message": "Không ghi nhận được kết quả cuối của lượt chạy; mốc quét giữ nguyên.",
		}).Error; err != nil {
		log.Printf("[analyzer] fallback error mark failed for run %s: %v", run.ID, err)
	}
	return fmt.Errorf("finalize job run: %w", lastErr)
}
