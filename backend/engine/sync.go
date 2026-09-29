package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// SyncEngine handles pulling messages from external channels into the database.
type SyncEngine struct {
	cfg *config.Config
	// Narrow per-engine seams for deterministic storage and mid-loop tests.
	storeForTenant    func(string) (storage.Store, error)
	afterConversation func()
}

const (
	maxSyncFailureDetails     = 10
	maxSyncFailureDetailRunes = 300
	maxSyncErrorMessageRunes  = 4000
)

// syncProgress keeps the channel checkpoint honest. A channel run is only a
// success when every selected conversation and message completed without an
// observable failure.
type syncProgress struct {
	conversationsFetched int
	conversationsSynced  int
	messagesSynced       int
	failures             []string
}

func (p *syncProgress) fail(scope, externalID string, err error) {
	detail := []rune(fmt.Sprintf("%s %s: %v", scope, externalID, err))
	if len(detail) > maxSyncFailureDetailRunes {
		detail = append(detail[:maxSyncFailureDetailRunes], '…')
	}
	p.failures = append(p.failures, string(detail))
}

func (p syncProgress) finalStatus() string {
	if len(p.failures) > 0 {
		return "partial"
	}
	return "success"
}

func (p syncProgress) errorMessage() string {
	if len(p.failures) == 0 {
		return ""
	}
	visible := p.failures
	if len(visible) > maxSyncFailureDetails {
		visible = visible[:maxSyncFailureDetails]
	}
	message := strings.Join(visible, "; ")
	if hidden := len(p.failures) - len(visible); hidden > 0 {
		message += fmt.Sprintf("; và %d lỗi khác", hidden)
	}
	summary := fmt.Sprintf("%d/%d conversations hoàn tất, %d messages đã lưu; %d lỗi: %s",
		p.conversationsSynced, p.conversationsFetched, p.messagesSynced, len(p.failures), message)
	runes := []rune(summary)
	if len(runes) > maxSyncErrorMessageRunes {
		return string(runes[:maxSyncErrorMessageRunes]) + "…"
	}
	return summary
}

func NewSyncEngine(cfg *config.Config) *SyncEngine {
	return &SyncEngine{cfg: cfg}
}

// Errors returned by ReserveChannelSync. They carry no SQL or credential text.
var (
	// ErrSyncAlreadyRunning: the channel is already marked syncing.
	ErrSyncAlreadyRunning = errors.New("sync_already_running")
	// ErrSyncChannelMissing: no such channel for the expected tenant.
	ErrSyncChannelMissing = errors.New("sync channel not found for tenant")
	// ErrSyncNotAdmitted: admission could not be recorded (database failure or
	// an unexpected zero-row write).
	ErrSyncNotAdmitted = errors.New("sync not admitted")
	// ErrSyncReservationMismatch: an already-reserved entry was given an empty
	// reservation or one that does not belong to its channel.
	ErrSyncReservationMismatch = errors.New("sync reservation does not match channel")
	ErrSyncOwnershipLost       = errors.New("sync_ownership_lost")
	ErrSyncWriteFailed         = errors.New("sync write failed")
)

// RunWriteDB returns a session for the writes that carry a run ID: the
// reservation, the run-ID-fenced terminal writes and the manual panic write.
// GORM's logger interpolates query values, so at Info, on errors and on slow
// queries it would print the run ID, the SQL values and the raw driver error
// to the process output. This session is silent for exactly those statements;
// every other statement keeps the application's normal DB logging, and the
// caller reports failures as bounded classes instead.
func RunWriteDB() *gorm.DB {
	return db.DB.Session(&gorm.Session{Logger: db.DB.Logger.LogMode(logger.Silent)})
}

// SyncReservation is the proof of one admitted run: the tenant and channel it
// was admitted for and the run ID stored on the channel row. Only
// ReserveChannelSync creates a valid one; every terminal write of that run must
// present its RunID.
type SyncReservation struct {
	TenantID  string
	ChannelID string
	RunID     string
	// Leased reports that the admitting write stored an R016 lease deadline
	// (GET-only channel types only). Only a leased run sends heartbeats.
	Leased bool
}

// CCMAI-RUNTIME-016: lease recovery is allowed only for channel types whose
// sync adapters issue read-only (GET) outbound requests. This allowlist is an
// audited source contract: if either adapter gains a mutating request or a
// token rotation, it must be removed from here until independently reviewed.
// zalo_oa is deliberately absent (single-use token refresh POST).
var leaseEligibleChannelTypes = []interface{}{"facebook", "pancake"}

// Lease timing. Variables only so tests can shorten them; the heartbeat must
// run well inside the lease.
var (
	syncLeaseDuration      = 5 * time.Minute
	syncHeartbeatInterval  = time.Minute
	syncLeaseRecoveryBatch = 100
	// leaseRecoveryBeforeRelease is a test seam that runs between the scan and
	// a row's conditional release; nil in production.
	leaseRecoveryBeforeRelease func(channelID string)
)

const syncLeaseReleasedMessage = "Lượt đồng bộ trước bị thu hồi vì hết hạn lease; kết quả của lượt đó không được ghi nhận."

// leaseDeadlineExpr is the lease deadline computed by the database clock.
func leaseDeadlineExpr() clause.Expr {
	return gorm.Expr("NOW(3) + INTERVAL ? MICROSECOND", syncLeaseDuration.Microseconds())
}

func (r SyncReservation) valid() bool {
	return r.TenantID != "" && r.ChannelID != "" && r.RunID != ""
}

// withOwnedSyncWrite serializes a run-owned side effect with any channel
// ownership change. Checking outside this transaction would permit takeover
// between the check and the write. The silent session also keeps the run ID and
// any values in these statements out of GORM's interpolated SQL output.
func withOwnedSyncWrite(reservation SyncReservation, write func(*gorm.DB) error) error {
	if !reservation.valid() {
		return ErrSyncOwnershipLost
	}
	var innerErr error
	err := RunWriteDB().Transaction(func(tx *gorm.DB) error {
		var owner models.Channel
		lookup := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "last_sync_status", "sync_run_id").
			Where("id = ? AND tenant_id = ?", reservation.ChannelID, reservation.TenantID).
			Take(&owner)
		if errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
			innerErr = ErrSyncOwnershipLost
			return innerErr
		}
		if lookup.Error != nil {
			innerErr = ErrSyncWriteFailed
			return innerErr
		}
		if owner.LastSyncStatus != "syncing" || owner.SyncRunID == nil || *owner.SyncRunID != reservation.RunID {
			innerErr = ErrSyncOwnershipLost
			return innerErr
		}
		innerErr = write(tx)
		return innerErr
	})
	if err != nil {
		if innerErr != nil {
			return innerErr
		}
		return ErrSyncWriteFailed // begin/commit failure; never expose driver text
	}
	return nil
}

func checkSyncOwnership(reservation SyncReservation) error {
	return withOwnedSyncWrite(reservation, func(*gorm.DB) error { return nil })
}

// Test seams (CCMAI-RUNTIME-013): the adapter factory and the after-sync
// trigger are variables so tests can observe dispatch without a real channel.
var (
	newSyncAdapter   = channels.NewAdapter
	triggerAfterSync = func(tenantID, channelID string) {
		if sched := GetDefaultScheduler(); sched != nil {
			sched.TriggerAfterSyncJobs(tenantID, channelID)
		}
	}
)

// ReserveChannelSync is the one admission point shared by the manual,
// scheduler and agent sync paths. The conditional UPDATE is the arbiter: it
// moves exactly one tenant-owned channel to "syncing" only if it is not
// already syncing (NULL counts as not syncing). A read afterwards only
// classifies a zero-row result and never decides admission.
//
// The same update assigns a fresh run ID (CCMAI-RUNTIME-014), which is returned
// only when exactly one row was changed.
func ReserveChannelSync(tenantID, channelID string) (SyncReservation, error) {
	runID := pkg.NewUUID()
	res := RunWriteDB().Model(&models.Channel{}).
		Where("id = ? AND tenant_id = ? AND (last_sync_status IS NULL OR last_sync_status <> ?)", channelID, tenantID, "syncing").
		Updates(map[string]interface{}{
			"last_sync_status": "syncing",
			"last_sync_error":  "",
			"sync_run_id":      runID,
			// R016: the lease marker is written by the same admitting update,
			// from the row's current type and the database clock. Any other
			// type gets NULL and can never be released by expiry.
			"sync_lease_until": gorm.Expr("CASE WHEN channel_type IN ? THEN NOW(3) + INTERVAL ? MICROSECOND ELSE NULL END",
				leaseEligibleChannelTypes, syncLeaseDuration.Microseconds()),
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		log.Printf("[sync] reservation for channel %s failed: write error", channelID)
		return SyncReservation{}, ErrSyncNotAdmitted
	}
	if res.RowsAffected == 1 {
		reservation := SyncReservation{TenantID: tenantID, ChannelID: channelID, RunID: runID}
		// Informational read of what the admitting write stored. On a read
		// failure no heartbeat starts: a leased run may then expire and be
		// released, and R015's ownership fence stops this worker's writes.
		var marks []struct{ Leased bool }
		if err := RunWriteDB().Model(&models.Channel{}).Select("sync_lease_until IS NOT NULL AS leased").
			Where("id = ? AND tenant_id = ? AND sync_run_id = ?", channelID, tenantID, runID).Scan(&marks).Error; err != nil {
			log.Printf("[sync] lease marker for channel %s not read; no heartbeat", channelID)
		} else if len(marks) == 1 {
			reservation.Leased = marks[0].Leased
		}
		return reservation, nil
	}
	if res.RowsAffected > 1 {
		log.Printf("[sync] reservation for channel %s changed %d rows", channelID, res.RowsAffected)
		return SyncReservation{}, ErrSyncNotAdmitted
	}

	var rows []struct{ LastSyncStatus *string }
	if err := db.DB.Model(&models.Channel{}).Select("last_sync_status").
		Where("id = ? AND tenant_id = ?", channelID, tenantID).Limit(1).Scan(&rows).Error; err != nil {
		log.Printf("[sync] reservation for channel %s not classified: read error", channelID)
		return SyncReservation{}, ErrSyncNotAdmitted
	}
	if len(rows) == 0 {
		return SyncReservation{}, ErrSyncChannelMissing
	}
	if rows[0].LastSyncStatus != nil && *rows[0].LastSyncStatus == "syncing" {
		return SyncReservation{}, ErrSyncAlreadyRunning
	}
	log.Printf("[sync] reservation for channel %s not applied although the channel is idle", channelID)
	return SyncReservation{}, ErrSyncNotAdmitted
}

// SyncChannel reserves the channel and then syncs it. Every caller that has
// not already reserved the channel must use this entry point.
func (s *SyncEngine) SyncChannel(ctx context.Context, channel models.Channel) error {
	reservation, err := ReserveChannelSync(channel.TenantID, channel.ID)
	if err != nil {
		return err
	}
	return s.SyncReservedChannel(ctx, channel, reservation)
}

// SyncReservedChannel runs a sync for a channel the caller has already
// reserved with ReserveChannelSync (the manual handler does this before it
// answers 202). It never reserves again. The reservation must belong to this
// channel; otherwise the run is rejected before any adapter work. Every
// terminal write, including the deferred error transition, presents the run ID
// of that reservation, so a stale worker cannot overwrite a newer run. If it
// exits without a recorded final status (failed write, zero rows, panic) it
// attempts an error transition under the same ID without hiding the original
// failure. A stuck "syncing" after a process crash is not handled here
// (separate recovery tranche). Run-owned conversation/message/count and
// refreshed-token writes are fenced by the channel ownership lock below.
func (s *SyncEngine) SyncReservedChannel(ctx context.Context, channel models.Channel, reservation SyncReservation) (runErr error) {
	if !reservation.valid() || reservation.TenantID != channel.TenantID || reservation.ChannelID != channel.ID {
		log.Printf("[sync] channel %s rejected: reservation does not match", channel.ID)
		return ErrSyncReservationMismatch
	}
	// R016: a leased run extends its lease while it works. A failed or
	// zero-row heartbeat cancels runCtx and stops the run (see stopErr).
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	var heartbeat *syncLeaseHeartbeat
	if reservation.Leased {
		heartbeat = startSyncLeaseHeartbeat(reservation, cancelRun)
	}
	defer heartbeat.stop()
	ctx = runCtx
	// stopErr is the heartbeat's failure class once it has failed.
	stopErr := func() error { return heartbeat.failure() }
	// gate stops the run on a failed heartbeat, then checks ownership.
	gate := func() error {
		if err := stopErr(); err != nil {
			return err
		}
		return checkSyncOwnership(reservation)
	}

	recorded := false
	defer func() {
		heartbeat.stop()
		if recorded || errors.Is(runErr, ErrSyncOwnershipLost) {
			return
		}
		r := recover()
		if _, terr := s.recordSyncStatus(reservation, "error", "Đồng bộ dừng trước khi ghi nhận kết quả; xem nhật ký máy chủ."); terr != nil {
			log.Printf("[sync] channel %s could not leave syncing state: %v", channel.ID, terr)
		}
		if r != nil {
			panic(r)
		}
	}()
	// finish records the final status; only a successful write counts as recorded.
	finish := func(status, errMsg string) error {
		heartbeat.stop()
		if err := stopErr(); err != nil {
			return err
		}
		wrote, err := s.recordSyncStatus(reservation, status, errMsg)
		if wrote {
			recorded = true
		}
		return err
	}

	log.Printf("[sync] starting sync for channel %s (%s)", channel.Name, channel.ChannelType)

	// Decrypt credentials
	credBytes, err := pkg.Decrypt(channel.CredentialsEncrypted, s.cfg.EncryptionKey)
	if err != nil {
		return finish("error", fmt.Sprintf("decrypt failed: %v", err))
	}

	adapter, err := newSyncAdapter(channel.ChannelType, credBytes)
	if err != nil {
		return finish("error", fmt.Sprintf("adapter init failed: %v", err))
	}

	// Persist rotated tokens only while this run still owns the channel.
	if zaloAdapter, ok := adapter.(*channels.ZaloOAAdapter); ok {
		zaloAdapter.SetTokenRefreshCallback(func(newAccess, newRefresh string) error {
			return s.persistZaloRefreshedTokens(reservation, newAccess, newRefresh)
		})
	}

	// Determine since — use last_sync_at or default to 7 days ago
	// Subtract 1 hour buffer to avoid missing messages near the boundary
	since := time.Now().AddDate(0, 0, -7)
	if channel.LastSyncAt != nil {
		since = channel.LastSyncAt.Add(-1 * time.Hour)
	}

	// Fetch recent conversations
	if err := gate(); err != nil {
		return err
	}
	conversations, err := adapter.FetchRecentConversations(ctx, since, 100)
	if err != nil {
		if hbErr := stopErr(); hbErr != nil {
			return hbErr
		}
		if errors.Is(err, ErrSyncOwnershipLost) || errors.Is(err, ErrSyncWriteFailed) {
			return err
		}
		return finish("error", fmt.Sprintf("fetch conversations failed: %v", err))
	}

	log.Printf("[sync] channel %s: found %d conversations", channel.Name, len(conversations))
	progress := syncProgress{conversationsFetched: len(conversations)}

	// Check if file sync is enabled for this channel
	syncFiles := false
	if channel.Metadata != "" {
		var meta map[string]interface{}
		if json.Unmarshal([]byte(channel.Metadata), &meta) == nil {
			if sf, ok := meta["sync_files"]; ok {
				syncFiles, _ = sf.(bool)
			}
		}
	}
	log.Printf("[sync] channel %s: sync_files=%v, metadata=%s", channel.Name, syncFiles, channel.Metadata)

	for _, conv := range conversations {
		conversationFailed := false
		if err := gate(); err != nil {
			return err
		}

		// Upsert conversation
		convID, err := s.upsertConversation(reservation, conv)
		if err != nil {
			if errors.Is(err, ErrSyncOwnershipLost) || errors.Is(err, ErrSyncWriteFailed) {
				return err
			}
			log.Printf("[sync] error upserting conversation %s: %v", conv.ExternalID, err)
			progress.fail("conversation", conv.ExternalID, err)
			continue
		}

		// Fetch messages for this conversation
		if err := gate(); err != nil {
			return err
		}
		messages, err := adapter.FetchMessages(ctx, conv.ExternalID, since)
		if err != nil {
			if hbErr := stopErr(); hbErr != nil {
				return hbErr
			}
			if errors.Is(err, ErrSyncOwnershipLost) || errors.Is(err, ErrSyncWriteFailed) {
				return err
			}
			log.Printf("[sync] error fetching messages for %s: %v", conv.ExternalID, err)
			progress.fail("messages", conv.ExternalID, err)
			continue
		}

		// Upsert messages
		for _, msg := range messages {
			if err := gate(); err != nil {
				return err
			}
			var newKeys []string
			if syncFiles {
				var attachmentErr error
				newKeys, attachmentErr = s.downloadAttachments(ctx, reservation, convID, &msg)
				// A failed heartbeat cancels the run whatever the transfer
				// result was: a store operation can finish successfully at the
				// cancellation boundary and must not be published (R016-R1).
				if hbErr := stopErr(); hbErr != nil {
					s.cleanupAttemptKeys(reservation.TenantID, convID, newKeys)
					return hbErr
				}
				if errors.Is(attachmentErr, ErrSyncOwnershipLost) || errors.Is(attachmentErr, ErrSyncWriteFailed) {
					s.cleanupAttemptKeys(reservation.TenantID, convID, newKeys)
					return attachmentErr
				}
				if attachmentErr != nil {
					log.Printf("[sync] attachment coverage incomplete for message %s: %v", msg.ExternalID, attachmentErr)
					progress.fail("attachments", msg.ExternalID, attachmentErr)
					conversationFailed = true
				}
			}
			if err := s.upsertMessage(reservation, convID, msg); err != nil {
				s.cleanupAttemptKeys(reservation.TenantID, convID, newKeys)
				if errors.Is(err, ErrSyncOwnershipLost) || errors.Is(err, ErrSyncWriteFailed) {
					return err
				}
				log.Printf("[sync] error upserting message %s: %v", msg.ExternalID, err)
				progress.fail("message", msg.ExternalID, err)
				conversationFailed = true
			} else {
				progress.messagesSynced++
			}
		}

		// Update conversation message count (a failed heartbeat stops the run
		// before this write too, including when the message loop was empty).
		if err := gate(); err != nil {
			return err
		}
		if err := s.updateOwnedMessageCount(reservation, convID); err != nil {
			if errors.Is(err, ErrSyncOwnershipLost) || errors.Is(err, ErrSyncWriteFailed) {
				return err
			}
			progress.fail("message_count", conv.ExternalID, err)
			conversationFailed = true
		}

		if !conversationFailed {
			progress.conversationsSynced++
		}
		if s.afterConversation != nil {
			s.afterConversation()
		}
	}

	log.Printf("[sync] channel %s: synced %d/%d conversations, %d messages, failures=%d",
		channel.Name, progress.conversationsSynced, progress.conversationsFetched, progress.messagesSynced, len(progress.failures))

	if progress.finalStatus() == "partial" {
		return finish("partial", progress.errorMessage())
	}
	if err := finish("success", ""); err != nil {
		return err
	}

	// Log activity
	db.LogActivity(channel.TenantID, "", "system", "sync.completed", "channel", channel.ID,
		fmt.Sprintf("Sync '%s': %d conversations, %d messages", channel.Name, progress.conversationsSynced, progress.messagesSynced), "", "")

	// Trigger after-sync jobs for this channel (only after a recorded success)
	triggerAfterSync(channel.TenantID, channel.ID)

	return nil
}

// luuFileDinhKem lưu một file, ưu tiên kho chính; kho chính hỏng thì ghi xuống
// đĩa để không mất tấm ảnh đó.
//
// Phải tải lại từ đầu cho lượt thử thứ hai vì luồng dữ liệu của lượt trước đã
// bị đọc mất. Đường này hiếm khi chạy nên tải lại một lần là chấp nhận được,
// đổi lại không bao giờ mất ảnh chỉ vì S3 chập chờn.
func luuFileDinhKem(ctx context.Context, chinh, duPhong storage.Store, key string,
	tai func() (io.ReadCloser, int64, string, error)) (noiDaLuu string, err error) {

	body, size, contentType, err := tai()
	if err != nil {
		return "", err
	}
	err = chinh.Put(ctx, key, body, size, contentType)
	body.Close()
	if err == nil {
		return chinh.Kind(), nil
	}
	if chinh == duPhong {
		return "", err
	}

	log.Printf("[sync] kho chính (%s) không nhận file: storage error — ghi tạm xuống đĩa, chạy migrate-files -up để chuyển lên sau", chinh.Kind())

	body2, size2, contentType2, err2 := tai()
	if err2 != nil {
		return "", fmt.Errorf("kho chính hỏng (%v) và tải lại cũng hỏng: %w", err, err2)
	}
	err2 = duPhong.Put(ctx, key, body2, size2, contentType2)
	body2.Close()
	if err2 != nil {
		return "", fmt.Errorf("kho chính hỏng (%v) và ghi xuống đĩa cũng hỏng: %w", err, err2)
	}
	return duPhong.Kind() + " (tạm)", nil
}

// SyncAllChannels syncs all active channels for a tenant.
func (s *SyncEngine) SyncAllChannels(ctx context.Context, tenantID string) error {
	var chans []models.Channel
	if err := db.DB.Where("tenant_id = ? AND is_active = true", tenantID).Find(&chans).Error; err != nil {
		return fmt.Errorf("load active channels: %w", err)
	}

	var syncErrors []error
	for _, ch := range chans {
		if err := s.SyncChannel(ctx, ch); err != nil {
			log.Printf("[sync] channel %s failed: %v", ch.Name, err)
			syncErrors = append(syncErrors, fmt.Errorf("channel %s: %w", ch.ID, err))
		}
	}
	return errors.Join(syncErrors...)
}

func (s *SyncEngine) upsertConversation(reservation SyncReservation, conv channels.SyncedConversation) (string, error) {
	metadataJSON, err := json.Marshal(conv.Metadata)
	if err != nil {
		return "", fmt.Errorf("marshal conversation metadata: %w", err)
	}
	var conversationID string
	err = withOwnedSyncWrite(reservation, func(tx *gorm.DB) error {
		var existing models.Conversation
		result := tx.Where("tenant_id = ? AND channel_id = ? AND external_conversation_id = ?",
			reservation.TenantID, reservation.ChannelID, conv.ExternalID).First(&existing)
		if result.Error == nil {
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"customer_name": conv.CustomerName, "last_message_at": conv.LastMessageAt,
				"metadata": string(metadataJSON), "updated_at": time.Now(),
			}).Error; err != nil {
				return ErrSyncWriteFailed
			}
			conversationID = existing.ID
			return nil
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return ErrSyncWriteFailed
		}
		newConv := models.Conversation{
			ID: pkg.NewUUID(), TenantID: reservation.TenantID, ChannelID: reservation.ChannelID,
			ExternalConversationID: conv.ExternalID, ExternalUserID: conv.ExternalUserID,
			CustomerName: conv.CustomerName, LastMessageAt: &conv.LastMessageAt,
			Metadata: string(metadataJSON), CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := tx.Create(&newConv).Error; err != nil {
			return ErrSyncWriteFailed
		}
		conversationID = newConv.ID
		return nil
	})
	return conversationID, err
}

func (s *SyncEngine) upsertMessage(reservation SyncReservation, conversationID string, msg channels.SyncedMessage) error {
	return withOwnedSyncWrite(reservation, func(tx *gorm.DB) error {
		var existing models.Message
		result := tx.Where("tenant_id = ? AND conversation_id = ? AND external_message_id = ?",
			reservation.TenantID, conversationID, msg.ExternalID).First(&existing)
		if result.Error == nil {
			return updateExistingMessage(tx, &existing, msg)
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return ErrSyncWriteFailed
		}
		attachmentsJSON, err := json.Marshal(msg.Attachments)
		if err != nil {
			return fmt.Errorf("marshal message attachments: %w", err)
		}
		rawDataJSON, err := json.Marshal(msg.RawData)
		if err != nil {
			return fmt.Errorf("marshal message raw data: %w", err)
		}
		message := models.Message{
			ID: pkg.NewUUID(), TenantID: reservation.TenantID, ConversationID: conversationID,
			ExternalMessageID: msg.ExternalID, SenderType: msg.SenderType,
			SenderName: msg.SenderName, Content: msg.Content, ContentType: msg.ContentType,
			Attachments: string(attachmentsJSON), SentAt: msg.SentAt,
			RawData: string(rawDataJSON), CreatedAt: time.Now(),
		}
		if err := tx.Create(&message).Error; err != nil {
			return ErrSyncWriteFailed
		}
		return nil
	})
}

// updateExistingMessage applies a same-external-ID replay onto an already
// stored row. Only an explicitly supplied, nonempty/nonzero field overwrites
// what is stored — an adapter that merely omits a field on this reply must
// never erase a previously populated value, and this tranche does not infer
// deletion from an empty reply. The message's internal ID and row count are
// never touched. When nothing actually differs, no UPDATE is issued at all,
// so a replay of an unchanged message causes no meaningful DB mutation.
func updateExistingMessage(tx *gorm.DB, existing *models.Message, msg channels.SyncedMessage) error {
	updates := map[string]interface{}{}

	if msg.Content != "" && msg.Content != existing.Content {
		updates["content"] = msg.Content
	}
	if msg.SenderType != "" && msg.SenderType != existing.SenderType {
		updates["sender_type"] = msg.SenderType
	}
	if msg.SenderName != "" && msg.SenderName != existing.SenderName {
		updates["sender_name"] = msg.SenderName
	}
	if msg.ContentType != "" && msg.ContentType != existing.ContentType {
		updates["content_type"] = msg.ContentType
	}
	if !msg.SentAt.IsZero() && !msg.SentAt.Equal(existing.SentAt) {
		updates["sent_at"] = msg.SentAt
	}

	mergedAttachments, changed, err := mergeAttachments(existing.Attachments, msg.Attachments)
	if err != nil {
		return fmt.Errorf("merge message attachments: %w", err)
	}
	if changed {
		attachmentsJSON, err := json.Marshal(mergedAttachments)
		if err != nil {
			return fmt.Errorf("marshal message attachments: %w", err)
		}
		updates["attachments"] = string(attachmentsJSON)
	}

	rawDataJSON, rawDataChanged, err := mergeRawData(existing.RawData, msg.RawData)
	if err != nil {
		return fmt.Errorf("serialize message raw data: %w", err)
	}
	if rawDataChanged {
		updates["raw_data"] = rawDataJSON
	}

	if len(updates) == 0 {
		return nil
	}
	if err := tx.Model(existing).Updates(updates).Error; err != nil {
		return ErrSyncWriteFailed
	}
	return nil
}

func (s *SyncEngine) updateOwnedMessageCount(reservation SyncReservation, conversationID string) error {
	return withOwnedSyncWrite(reservation, func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.Message{}).Where("tenant_id = ? AND conversation_id = ?", reservation.TenantID, conversationID).Count(&count).Error; err != nil {
			return ErrSyncWriteFailed
		}
		result := tx.Model(&models.Conversation{}).
			Where("id = ? AND tenant_id = ? AND channel_id = ?", conversationID, reservation.TenantID, reservation.ChannelID).
			Update("message_count", count)
		if result.Error != nil || result.RowsAffected != 1 {
			return ErrSyncWriteFailed
		}
		return nil
	})
}

func (s *SyncEngine) persistZaloRefreshedTokens(reservation SyncReservation, access, refresh string) error {
	return withOwnedSyncWrite(reservation, func(tx *gorm.DB) error {
		var ch models.Channel
		if err := tx.Where("id = ? AND tenant_id = ?", reservation.ChannelID, reservation.TenantID).Take(&ch).Error; err != nil {
			return ErrSyncWriteFailed
		}
		oldCreds, err := pkg.Decrypt(ch.CredentialsEncrypted, s.cfg.EncryptionKey)
		if err != nil {
			return ErrSyncWriteFailed
		}
		var creds map[string]interface{}
		if err := json.Unmarshal(oldCreds, &creds); err != nil {
			return ErrSyncWriteFailed
		}
		creds["access_token"], creds["refresh_token"] = access, refresh
		encoded, err := json.Marshal(creds)
		if err != nil {
			return ErrSyncWriteFailed
		}
		encrypted, err := pkg.Encrypt(encoded, s.cfg.EncryptionKey)
		if err != nil {
			return ErrSyncWriteFailed
		}
		result := tx.Model(&models.Channel{}).
			Where("id = ? AND tenant_id = ? AND last_sync_status = ? AND sync_run_id = ?",
				reservation.ChannelID, reservation.TenantID, "syncing", reservation.RunID).
			Update("credentials_encrypted", encrypted)
		if result.Error != nil {
			return ErrSyncWriteFailed
		}
		if result.RowsAffected != 1 {
			return ErrSyncOwnershipLost
		}
		return nil
	})
}

// mergeRawData decides whether a replay's raw-data map should overwrite what
// is stored. A nil or empty map is indistinguishable from an adapter that
// simply didn't attach raw data to this reply, so it never erases stored raw
// data and is not treated as a deletion signal. A nonempty map is explicitly
// supplied: it is marshaled with an error check (R003-R1 — a value the
// standard library cannot encode, such as a NaN float, must surface as an
// error here rather than silently keeping stale data with no report) and
// only written when it canonically differs from the stored value, so an
// identical replay issues no UPDATE. This tranche does not act on any
// removal/edit marker inside the raw payload (e.g. Pancake's "is_removed") —
// it is stored as supplied, nothing more.
func mergeRawData(existingJSON string, incoming map[string]interface{}) (value string, changed bool, err error) {
	if len(incoming) == 0 {
		return "", false, nil
	}
	incomingJSON, err := json.Marshal(incoming)
	if err != nil {
		return "", false, fmt.Errorf("marshal message raw data: %w", err)
	}

	trimmed := strings.TrimSpace(existingJSON)
	if trimmed != "" && trimmed != "null" {
		var existing map[string]interface{}
		if err := json.Unmarshal([]byte(existingJSON), &existing); err == nil {
			if existingCanon, err := json.Marshal(existing); err == nil && string(existingCanon) == string(incomingJSON) {
				return "", false, nil
			}
		}
		// A stored value that can't be parsed/re-encoded cleanly falls
		// through to the write below: the incoming value is nonempty, valid
		// and explicitly supplied, so it replaces a value this code can't
		// even confirm is unchanged.
	}
	return string(incomingJSON), true, nil
}

// mergeAttachments applies a replay's attachment list onto what is already
// stored, using the same identity (type, URL, name — see classifyAttachments)
// the snapshot digest fingerprint already keys on. An attachment that keeps
// the same identity as before keeps its previously downloaded LocalPath when
// this reply didn't supply a new one; an attachment with a different identity
// must never inherit a stranger's local path. An empty incoming list is
// indistinguishable from an adapter that simply didn't include attachments in
// this reply (as opposed to the source message losing them), so it never
// erases a previously stored list — this tranche does not infer deletion from
// it; that ambiguity is a documented adapter coverage limit, not a defect.
func mergeAttachments(existingJSON string, incoming []channels.Attachment) (merged []channels.Attachment, changed bool, err error) {
	var existing []channels.Attachment
	trimmed := strings.TrimSpace(existingJSON)
	if trimmed != "" && trimmed != "null" {
		if err := json.Unmarshal([]byte(existingJSON), &existing); err != nil {
			return nil, false, fmt.Errorf("parse stored attachments: %w", err)
		}
	}

	if len(incoming) == 0 {
		return existing, false, nil
	}

	type identity struct{ typ, url, name string }
	priorLocalPath := make(map[identity]string, len(existing))
	for _, att := range existing {
		priorLocalPath[identity{att.Type, att.URL, att.Name}] = att.LocalPath
	}

	merged = make([]channels.Attachment, len(incoming))
	copy(merged, incoming)
	for i := range merged {
		if merged[i].LocalPath == "" {
			if oldPath, ok := priorLocalPath[identity{merged[i].Type, merged[i].URL, merged[i].Name}]; ok {
				merged[i].LocalPath = oldPath
			}
		}
	}

	existingCanon, err := json.Marshal(existing)
	if err != nil {
		return nil, false, fmt.Errorf("re-encode stored attachments: %w", err)
	}
	mergedCanon, err := json.Marshal(merged)
	if err != nil {
		return nil, false, fmt.Errorf("encode merged attachments: %w", err)
	}
	return merged, string(existingCanon) != string(mergedCanon), nil
}

func buildSyncStatusUpdates(status, errMsg string, now time.Time) map[string]interface{} {
	updates := map[string]interface{}{
		"last_sync_status": status,
		"last_sync_error":  errMsg,
		"updated_at":       now,
	}
	if status == "success" {
		updates["last_sync_at"] = &now
	}
	return updates
}

// recordSyncStatus writes the final status of a reserved run. The write is
// scoped to the tenant, the current "syncing" state and the run's own ID, and
// must affect exactly one row; otherwise wrote is false and the error says the
// status was not recorded (a missing, transferred, already-finished or
// re-reserved row is never overwritten). The same write clears the run ID.
// When wrote is true, err is nil for success and the reported failure text for
// partial/error runs.
func (s *SyncEngine) recordSyncStatus(reservation SyncReservation, status, errMsg string) (wrote bool, err error) {
	if !reservation.valid() {
		return false, fmt.Errorf("update sync status %s: %w", status, ErrSyncReservationMismatch)
	}
	tenantID, channelID := reservation.TenantID, reservation.ChannelID
	now := time.Now()
	updates := buildSyncStatusUpdates(status, errMsg, now)
	updates["sync_run_id"] = gorm.Expr("NULL")
	updates["sync_lease_until"] = gorm.Expr("NULL") // R016: every owned terminal write ends the lease
	res := RunWriteDB().Model(&models.Channel{}).
		Where("id = ? AND tenant_id = ? AND last_sync_status = ? AND sync_run_id = ?", channelID, tenantID, "syncing", reservation.RunID).
		Updates(updates)
	if res.Error != nil {
		// Bounded class only: the driver error can echo SQL values and the run ID.
		return false, fmt.Errorf("update sync status %s: write failed", status)
	}
	if res.RowsAffected != 1 {
		return false, fmt.Errorf("update sync status %s: %d rows affected", status, res.RowsAffected)
	}
	if errMsg != "" {
		action := "sync.error"
		label := "Sync failed"
		if status == "partial" {
			action = "sync.partial"
			label = "Sync partial"
		}
		var ch models.Channel
		if db.DB.Where("id = ? AND tenant_id = ?", channelID, tenantID).First(&ch).Error == nil {
			db.LogActivity(ch.TenantID, "", "system", action, "channel", channelID, label+": "+ch.Name, errMsg, "")
		}
		return true, fmt.Errorf("sync %s: %s", status, errMsg)
	}
	return true, nil
}

// downloadAttachments reuses an existing same-identity object when present.
// A new download receives an attempt-unique key, so an old run cannot replace
// a newer run's bytes even if it finishes a network request after takeover.
// Only the owned message upsert may publish the new key in the database.
func (s *SyncEngine) downloadAttachments(ctx context.Context, reservation SyncReservation, convID string, msg *channels.SyncedMessage) ([]string, error) {
	if err := checkSyncOwnership(reservation); err != nil {
		return nil, err
	}
	tenantID := reservation.TenantID
	// Kho trên đĩa luôn dựng sẵn làm lưới an toàn. Mất một tấm ảnh là mất hẳn —
	// link ảnh bên Zalo và Facebook hết hạn sau ít lâu, không tải lại được nữa —
	// nên S3 trục trặc thì thà ghi tạm xuống đĩa rồi chuyển lên sau, hơn là bỏ.
	duPhong, err := storage.NewLocal(s.cfg.StorageLocalDir)
	if err != nil {
		return nil, fmt.Errorf("không dựng được kho trên đĩa: %w", err)
	}

	// Mỗi công ty có kho riêng: công ty này để trên S3, công ty kia vẫn trên đĩa.
	storeForTenant := s.storeForTenant
	if storeForTenant == nil {
		storeForTenant = storage.ForTenant
	}
	store, err := storeForTenant(tenantID)
	if err != nil {
		log.Printf("[sync] không lấy được nơi cất file của công ty %s (%v) — tạm ghi xuống đĩa", tenantID, err)
		store = duPhong
	}
	// Replaying an unchanged attachment identity should not create a new object
	// on every sync. A missing old object is downloaded again under a fresh key.
	prior := map[string]string{}
	var stored models.Message
	lookup := RunWriteDB().Where("tenant_id = ? AND conversation_id = ? AND external_message_id = ?",
		tenantID, convID, msg.ExternalID).Take(&stored)
	if lookup.Error == nil {
		var old []channels.Attachment
		if err := json.Unmarshal([]byte(stored.Attachments), &old); err != nil {
			return nil, fmt.Errorf("parse stored attachments: %w", err)
		}
		for _, att := range old {
			if att.LocalPath != "" {
				prior[att.Type+"\x00"+att.URL+"\x00"+att.Name] = att.LocalPath
			}
		}
	} else if !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
		return nil, ErrSyncWriteFailed
	}

	var failures []string
	var newKeys []string
	for i, att := range msg.Attachments {
		if att.URL == "" {
			continue
		}
		if err := checkSyncOwnership(reservation); err != nil {
			return newKeys, err
		}
		if oldKey := prior[att.Type+"\x00"+att.URL+"\x00"+att.Name]; oldKey != "" {
			if exists, err := store.Exists(ctx, oldKey); err == nil && exists {
				msg.Attachments[i].LocalPath = oldKey
				continue
			}
		}

		// Tên file đến từ API bên ngoài nên không được tin: chỉ lấy phần tên,
		// bỏ mọi thành phần đường dẫn.
		name := filepath.Base(att.Name)
		if name == "" || name == "." || name == "/" || strings.Contains(name, "..") {
			name = fmt.Sprintf("%s-%d", att.Type, time.Now().UnixMilli())
		}
		key := path.Join(tenantID, convID, pkg.NewUUID(), name)

		tai := func() (io.ReadCloser, int64, string, error) {
			client := &http.Client{Timeout: 30 * time.Second}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, att.URL, nil)
			if err != nil {
				return nil, 0, "", err
			}
			resp, err := client.Do(req)
			if err != nil {
				return nil, 0, "", err
			}
			if resp.StatusCode != 200 {
				resp.Body.Close()
				return nil, 0, "", fmt.Errorf("status %d", resp.StatusCode)
			}
			// ContentLength là -1 khi máy chủ không báo độ dài; nơi cất file
			// hiểu -1 là "chưa biết trước".
			return resp.Body, resp.ContentLength, resp.Header.Get("Content-Type"), nil
		}

		noiDaLuu, err := luuFileDinhKem(ctx, store, duPhong, key, tai)
		if err != nil {
			log.Printf("[sync] attachment transfer failed for channel %s", reservation.ChannelID)
			failures = append(failures, "attachment transfer failed")
			continue
		}

		// Trường LocalPath giữ nguyên tên cũ để không phải đổi dữ liệu đã lưu,
		// nay mang nghĩa khoá của file trong nơi cất.
		msg.Attachments[i].LocalPath = key
		newKeys = append(newKeys, key)
		log.Printf("[sync] downloaded attachment for channel %s (%s)", reservation.ChannelID, noiDaLuu)
	}
	if len(failures) > 0 {
		return newKeys, fmt.Errorf("%d attachment không lưu được: %s", len(failures), strings.Join(failures, "; "))
	}
	return newKeys, nil
}

func (s *SyncEngine) cleanupAttemptKeys(tenantID, convID string, keys []string) {
	if len(keys) == 0 {
		return
	}
	local, err := storage.NewLocal(s.cfg.StorageLocalDir)
	if err != nil {
		log.Printf("[sync] cannot initialize attachment cleanup: storage error")
		return
	}
	store, err := storage.ForTenant(tenantID)
	if err != nil {
		store = local
	}
	for _, key := range keys {
		// A failed/unknown COMMIT result is not proof the message write rolled
		// back. Delete only after a successful read proves this key is not
		// referenced; on read failure, leave an orphan instead of breaking a
		// potentially committed attachment.
		var references int64
		if err := RunWriteDB().Model(&models.Message{}).
			Where("tenant_id = ? AND conversation_id = ? AND JSON_SEARCH(attachments, 'one', ?) IS NOT NULL", tenantID, convID, key).
			Count(&references).Error; err != nil {
			log.Printf("[sync] attachment attempt cleanup deferred: reference check failed")
			continue
		}
		if references != 0 {
			continue
		}
		if err := store.Delete(context.Background(), key); err != nil {
			log.Printf("[sync] attachment attempt cleanup failed: storage error")
		}
	}
}

// syncLeaseHeartbeat extends one leased run's deadline (CCMAI-RUNTIME-016).
type syncLeaseHeartbeat struct {
	stopCh   chan struct{}
	done     chan struct{}
	stopOnce sync.Once
	mu       sync.Mutex
	err      error
}

func startSyncLeaseHeartbeat(reservation SyncReservation, cancelRun context.CancelFunc) *syncLeaseHeartbeat {
	h := &syncLeaseHeartbeat{stopCh: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(h.done)
		ticker := time.NewTicker(syncHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-h.stopCh:
				return
			case <-ticker.C:
				if err := extendSyncLease(reservation); err != nil {
					h.mu.Lock()
					h.err = err
					h.mu.Unlock()
					log.Printf("[sync] lease heartbeat for channel %s failed: %v", reservation.ChannelID, err)
					cancelRun()
					return
				}
			}
		}
	}()
	return h
}

// stop ends the heartbeat and waits for it; safe on nil and when repeated.
func (h *syncLeaseHeartbeat) stop() {
	if h == nil {
		return
	}
	h.stopOnce.Do(func() { close(h.stopCh) })
	<-h.done
}

// failure is the bounded class of a failed heartbeat, or nil.
func (h *syncLeaseHeartbeat) failure() error {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.err
}

// extendSyncLease moves the lease deadline forward only for the exact owning
// run of a leased row; a Zalo or legacy run can never gain a lease this way.
// UpdateColumn leaves updated_at alone; the silent session keeps the run ID
// out of GORM's output.
func extendSyncLease(reservation SyncReservation) error {
	res := RunWriteDB().Model(&models.Channel{}).
		Where("id = ? AND tenant_id = ? AND last_sync_status = ? AND sync_run_id = ? AND sync_lease_until IS NOT NULL",
			reservation.ChannelID, reservation.TenantID, "syncing", reservation.RunID).
		UpdateColumn("sync_lease_until", leaseDeadlineExpr())
	if res.Error != nil {
		return ErrSyncWriteFailed
	}
	if res.RowsAffected != 1 {
		return ErrSyncOwnershipLost
	}
	return nil
}

// RecoverExpiredSyncLeases releases, in one bounded batch, runs whose lease
// has expired by database time — only for the GET-only allowlist and only for
// rows carrying an R016 run ID and lease. Expiry revokes local ownership; it
// does not prove the old worker died, and R015's ownership fence keeps that
// worker from writing afterwards. Each release is a single conditional
// update that repeats every eligibility predicate with the observed run ID,
// so a concurrent heartbeat or terminal write wins or loses atomically. It
// never touches last_sync_at, credentials, conversations, messages or
// attachments and launches no worker: the channel re-enters the ordinary
// reservation path after its normal throttle. The returned error is a bounded
// class; rows it could not release stay syncing.
func RecoverExpiredSyncLeases(limit int) (released int, err error) {
	if limit <= 0 {
		return 0, nil
	}
	var candidates []struct {
		ID, TenantID string
		SyncRunID    string
	}
	if err := RunWriteDB().Model(&models.Channel{}).Select("id", "tenant_id", "sync_run_id").
		Where("channel_type IN ? AND last_sync_status = ? AND sync_run_id IS NOT NULL AND sync_lease_until IS NOT NULL AND sync_lease_until < NOW(3)",
			leaseEligibleChannelTypes, "syncing").
		Order("sync_lease_until").Limit(limit).Scan(&candidates).Error; err != nil {
		log.Printf("[sync] lease recovery scan failed: read error")
		return 0, ErrSyncWriteFailed
	}
	var failed bool
	for _, c := range candidates {
		if leaseRecoveryBeforeRelease != nil {
			leaseRecoveryBeforeRelease(c.ID)
		}
		res := RunWriteDB().Model(&models.Channel{}).
			Where("id = ? AND tenant_id = ? AND channel_type IN ? AND last_sync_status = ? AND sync_run_id = ? AND sync_lease_until IS NOT NULL AND sync_lease_until < NOW(3)",
				c.ID, c.TenantID, leaseEligibleChannelTypes, "syncing", c.SyncRunID).
			UpdateColumns(map[string]interface{}{
				"last_sync_status": "error",
				"last_sync_error":  syncLeaseReleasedMessage,
				"sync_run_id":      gorm.Expr("NULL"),
				"sync_lease_until": gorm.Expr("NULL"),
				"updated_at":       time.Now(), // attempt time for scheduler throttling
			})
		if res.Error != nil {
			failed = true
			log.Printf("[sync] lease recovery for channel %s failed: write error", c.ID)
			continue
		}
		if res.RowsAffected == 1 {
			released++
			log.Printf("[sync] released expired sync lease for channel %s", c.ID)
		}
	}
	if failed {
		return released, ErrSyncWriteFailed
	}
	return released, nil
}
