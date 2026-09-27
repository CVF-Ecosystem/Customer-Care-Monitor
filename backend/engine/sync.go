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
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/pkg"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/storage"
	"gorm.io/gorm"
)

// SyncEngine handles pulling messages from external channels into the database.
type SyncEngine struct {
	cfg *config.Config
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

// SyncChannel syncs a single channel: fetches conversations + messages and upserts into DB.
func (s *SyncEngine) SyncChannel(ctx context.Context, channel models.Channel) error {
	log.Printf("[sync] starting sync for channel %s (%s)", channel.Name, channel.ChannelType)

	// Decrypt credentials
	credBytes, err := pkg.Decrypt(channel.CredentialsEncrypted, s.cfg.EncryptionKey)
	if err != nil {
		return s.updateSyncStatus(channel.ID, "error", fmt.Sprintf("decrypt failed: %v", err))
	}

	adapter, err := channels.NewAdapter(channel.ChannelType, credBytes)
	if err != nil {
		return s.updateSyncStatus(channel.ID, "error", fmt.Sprintf("adapter init failed: %v", err))
	}

	// Set token refresh callback for Zalo — persist new tokens to DB after refresh
	if zaloAdapter, ok := adapter.(*channels.ZaloOAAdapter); ok {
		chID := channel.ID
		encKey := s.cfg.EncryptionKey
		zaloAdapter.SetTokenRefreshCallback(func(newAccess, newRefresh string) {
			var ch models.Channel
			if db.DB.First(&ch, "id = ?", chID).Error != nil {
				return
			}
			oldCreds, err := pkg.Decrypt(ch.CredentialsEncrypted, encKey)
			if err != nil {
				log.Printf("[sync] decrypt failed for token persist: %v", err)
				return
			}
			var credsMap map[string]interface{}
			if err := json.Unmarshal(oldCreds, &credsMap); err != nil {
				log.Printf("[sync] unmarshal creds failed: %v", err)
				return
			}
			credsMap["access_token"] = newAccess
			credsMap["refresh_token"] = newRefresh
			newCredJSON, _ := json.Marshal(credsMap)
			encrypted, err := pkg.Encrypt(newCredJSON, encKey)
			if err != nil {
				log.Printf("[sync] encrypt failed for token persist: %v", err)
				return
			}
			db.DB.Model(&ch).Update("credentials_encrypted", encrypted)
			log.Printf("[sync] persisted refreshed Zalo tokens for channel %s", chID)
		})
	}

	// Determine since — use last_sync_at or default to 7 days ago
	// Subtract 1 hour buffer to avoid missing messages near the boundary
	since := time.Now().AddDate(0, 0, -7)
	if channel.LastSyncAt != nil {
		since = channel.LastSyncAt.Add(-1 * time.Hour)
	}

	// Fetch recent conversations
	conversations, err := adapter.FetchRecentConversations(ctx, since, 100)
	if err != nil {
		return s.updateSyncStatus(channel.ID, "error", fmt.Sprintf("fetch conversations failed: %v", err))
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

		// Upsert conversation
		convID, err := s.upsertConversation(channel.TenantID, channel.ID, conv)
		if err != nil {
			log.Printf("[sync] error upserting conversation %s: %v", conv.ExternalID, err)
			progress.fail("conversation", conv.ExternalID, err)
			continue
		}

		// Fetch messages for this conversation
		messages, err := adapter.FetchMessages(ctx, conv.ExternalID, since)
		if err != nil {
			log.Printf("[sync] error fetching messages for %s: %v", conv.ExternalID, err)
			progress.fail("messages", conv.ExternalID, err)
			continue
		}

		// Upsert messages
		for _, msg := range messages {
			if syncFiles {
				if err := s.downloadAttachments(channel.TenantID, convID, &msg); err != nil {
					log.Printf("[sync] attachment coverage incomplete for message %s: %v", msg.ExternalID, err)
					progress.fail("attachments", msg.ExternalID, err)
					conversationFailed = true
				}
			}
			if err := s.upsertMessage(channel.TenantID, convID, msg); err != nil {
				log.Printf("[sync] error upserting message %s: %v", msg.ExternalID, err)
				progress.fail("message", msg.ExternalID, err)
				conversationFailed = true
			} else {
				progress.messagesSynced++
			}
		}

		// Update conversation message count
		var count int64
		if err := db.DB.Model(&models.Message{}).Where("conversation_id = ?", convID).Count(&count).Error; err != nil {
			progress.fail("message_count", conv.ExternalID, err)
			conversationFailed = true
		} else if err := db.DB.Model(&models.Conversation{}).Where("id = ?", convID).Update("message_count", count).Error; err != nil {
			progress.fail("message_count", conv.ExternalID, err)
			conversationFailed = true
		}

		if !conversationFailed {
			progress.conversationsSynced++
		}
	}

	log.Printf("[sync] channel %s: synced %d/%d conversations, %d messages, failures=%d",
		channel.Name, progress.conversationsSynced, progress.conversationsFetched, progress.messagesSynced, len(progress.failures))

	if progress.finalStatus() == "partial" {
		return s.updateSyncStatus(channel.ID, "partial", progress.errorMessage())
	}
	if err := s.updateSyncStatus(channel.ID, "success", ""); err != nil {
		return err
	}

	// Log activity
	db.LogActivity(channel.TenantID, "", "system", "sync.completed", "channel", channel.ID,
		fmt.Sprintf("Sync '%s': %d conversations, %d messages", channel.Name, progress.conversationsSynced, progress.messagesSynced), "", "")

	// Trigger after-sync jobs for this channel
	if sched := GetDefaultScheduler(); sched != nil {
		sched.TriggerAfterSyncJobs(channel.TenantID, channel.ID)
	}

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

	log.Printf("[sync] kho chính (%s) không nhận file %s: %v — ghi tạm xuống đĩa, chạy migrate-files -up để chuyển lên sau",
		chinh.Kind(), key, err)

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

func (s *SyncEngine) upsertConversation(tenantID, channelID string, conv channels.SyncedConversation) (string, error) {
	var existing models.Conversation
	result := db.DB.Where("tenant_id = ? AND channel_id = ? AND external_conversation_id = ?",
		tenantID, channelID, conv.ExternalID).First(&existing)

	metadataJSON, err := json.Marshal(conv.Metadata)
	if err != nil {
		return "", fmt.Errorf("marshal conversation metadata: %w", err)
	}

	if result.Error == nil {
		// Update existing
		if err := db.DB.Model(&existing).Updates(map[string]interface{}{
			"customer_name":   conv.CustomerName,
			"last_message_at": conv.LastMessageAt,
			"metadata":        string(metadataJSON),
			"updated_at":      time.Now(),
		}).Error; err != nil {
			return "", err
		}
		return existing.ID, nil
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("find existing conversation: %w", result.Error)
	}

	// Create new
	newConv := models.Conversation{
		ID:                     pkg.NewUUID(),
		TenantID:               tenantID,
		ChannelID:              channelID,
		ExternalConversationID: conv.ExternalID,
		ExternalUserID:         conv.ExternalUserID,
		CustomerName:           conv.CustomerName,
		LastMessageAt:          &conv.LastMessageAt,
		MessageCount:           0,
		Metadata:               string(metadataJSON),
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}
	if err := db.DB.Create(&newConv).Error; err != nil {
		return "", err
	}
	return newConv.ID, nil
}

func (s *SyncEngine) upsertMessage(tenantID, conversationID string, msg channels.SyncedMessage) error {
	// Check if message already exists (dedup by external_message_id)
	var existing models.Message
	result := db.DB.Where("tenant_id = ? AND conversation_id = ? AND external_message_id = ?",
		tenantID, conversationID, msg.ExternalID).First(&existing)
	if result.Error == nil {
		// Message exists — update attachments if we have new local paths
		hasLocalPath := false
		for _, att := range msg.Attachments {
			if att.LocalPath != "" {
				hasLocalPath = true
				break
			}
		}
		if hasLocalPath {
			attachmentsJSON, err := json.Marshal(msg.Attachments)
			if err != nil {
				return fmt.Errorf("marshal message attachments: %w", err)
			}
			if err := db.DB.Model(&existing).Update("attachments", string(attachmentsJSON)).Error; err != nil {
				return err
			}
		}
		return nil
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("find existing message: %w", result.Error)
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
		ID:                pkg.NewUUID(),
		TenantID:          tenantID,
		ConversationID:    conversationID,
		ExternalMessageID: msg.ExternalID,
		SenderType:        msg.SenderType,
		SenderName:        msg.SenderName,
		Content:           msg.Content,
		ContentType:       msg.ContentType,
		Attachments:       string(attachmentsJSON),
		SentAt:            msg.SentAt,
		RawData:           string(rawDataJSON),
		CreatedAt:         time.Now(),
	}
	return db.DB.Create(&message).Error
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

func (s *SyncEngine) updateSyncStatus(channelID, status, errMsg string) error {
	now := time.Now()
	updates := buildSyncStatusUpdates(status, errMsg, now)
	if err := db.DB.Model(&models.Channel{}).Where("id = ?", channelID).Updates(updates).Error; err != nil {
		return fmt.Errorf("update sync status %s: %w", status, err)
	}
	if errMsg != "" {
		action := "sync.error"
		label := "Sync failed"
		if status == "partial" {
			action = "sync.partial"
			label = "Sync partial"
		}
		var ch models.Channel
		if db.DB.Where("id = ?", channelID).First(&ch).Error == nil {
			db.LogActivity(ch.TenantID, "", "system", action, "channel", channelID, label+": "+ch.Name, errMsg, "")
		}
		return fmt.Errorf("sync %s: %s", status, errMsg)
	}
	return nil
}

// downloadAttachments tải file đính kèm về nơi cất file đang cấu hình — đĩa
// máy chủ hoặc S3. Khoá của file vẫn là "<tenant>/<cuộc chat>/<tên file>" như
// trước, nên đổi nơi cất không phải đụng vào dữ liệu đã lưu.
func (s *SyncEngine) downloadAttachments(tenantID, convID string, msg *channels.SyncedMessage) error {
	// Kho trên đĩa luôn dựng sẵn làm lưới an toàn. Mất một tấm ảnh là mất hẳn —
	// link ảnh bên Zalo và Facebook hết hạn sau ít lâu, không tải lại được nữa —
	// nên S3 trục trặc thì thà ghi tạm xuống đĩa rồi chuyển lên sau, hơn là bỏ.
	duPhong, err := storage.NewLocal(s.cfg.StorageLocalDir)
	if err != nil {
		return fmt.Errorf("không dựng được kho trên đĩa: %w", err)
	}

	// Mỗi công ty có kho riêng: công ty này để trên S3, công ty kia vẫn trên đĩa.
	store, err := storage.ForTenant(tenantID)
	if err != nil {
		log.Printf("[sync] không lấy được nơi cất file của công ty %s (%v) — tạm ghi xuống đĩa", tenantID, err)
		store = duPhong
	}

	var failures []string
	for i, att := range msg.Attachments {
		if att.URL == "" {
			continue
		}

		// Tên file đến từ API bên ngoài nên không được tin: chỉ lấy phần tên,
		// bỏ mọi thành phần đường dẫn.
		name := filepath.Base(att.Name)
		if name == "" || name == "." || name == "/" || strings.Contains(name, "..") {
			name = fmt.Sprintf("%s-%d", att.Type, time.Now().UnixMilli())
		}
		key := path.Join(tenantID, convID, name)

		tai := func() (io.ReadCloser, int64, string, error) {
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Get(att.URL)
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

		noiDaLuu, err := luuFileDinhKem(context.Background(), store, duPhong, key, tai)
		if err != nil {
			log.Printf("[sync] lưu file %s hỏng: %v", key, err)
			failures = append(failures, fmt.Sprintf("%s: %v", name, err))
			continue
		}

		// Trường LocalPath giữ nguyên tên cũ để không phải đổi dữ liệu đã lưu,
		// nay mang nghĩa khoá của file trong nơi cất.
		msg.Attachments[i].LocalPath = key
		log.Printf("[sync] downloaded %s → %s (%s)", att.URL, key, noiDaLuu)
	}
	if len(failures) > 0 {
		return fmt.Errorf("%d attachment không lưu được: %s", len(failures), strings.Join(failures, "; "))
	}
	return nil
}
