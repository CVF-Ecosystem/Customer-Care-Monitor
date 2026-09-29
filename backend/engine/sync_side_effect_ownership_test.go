package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/storage"
	"gorm.io/gorm"
)

func TestStaleRunCannotPersistConversationMessageCountOrCredentials(t *testing.T) {
	f := setupSFFixture(t)
	eng := NewSyncEngine(f.cfg)
	old := f.mustReserve(t, f.tenantID, f.chA)
	f.simulateRecoveryRelease(t, f.chA)
	owner := f.mustReserve(t, f.tenantID, f.chA)
	conv := channels.SyncedConversation{ExternalID: "new-generation", CustomerName: "B", LastMessageAt: time.Now()}
	convID, err := eng.upsertConversation(owner, conv)
	if err != nil {
		t.Fatalf("owner conversation: %v", err)
	}
	msg := channels.SyncedMessage{ExternalID: "message-B", SenderType: "customer", Content: "B", SentAt: time.Now()}
	if err := eng.upsertMessage(owner, convID, msg); err != nil {
		t.Fatalf("owner message: %v", err)
	}
	if err := eng.updateOwnedMessageCount(owner, convID); err != nil {
		t.Fatalf("owner count: %v", err)
	}
	var before models.Channel
	if err := db.DB.Take(&before, "id = ?", f.chA).Error; err != nil {
		t.Fatal(err)
	}
	for name, write := range map[string]func() error{
		"conversation": func() error {
			_, err := eng.upsertConversation(old, channels.SyncedConversation{ExternalID: "stale", CustomerName: "A"})
			return err
		},
		"message": func() error {
			return eng.upsertMessage(old, convID, channels.SyncedMessage{ExternalID: "message-B", Content: "A"})
		},
		"count":       func() error { return eng.updateOwnedMessageCount(old, convID) },
		"credentials": func() error { return eng.persistZaloRefreshedTokens(old, "stale-access", "stale-refresh") },
	} {
		if err := write(); !errors.Is(err, ErrSyncOwnershipLost) {
			t.Errorf("%s: got %v, want ownership lost", name, err)
		}
	}
	var staleCount int64
	if err := db.DB.Model(&models.Conversation{}).Where("tenant_id = ? AND channel_id = ? AND external_conversation_id = 'stale'", f.tenantID, f.chA).Count(&staleCount).Error; err != nil || staleCount != 0 {
		t.Fatalf("stale conversation count %d, err %v", staleCount, err)
	}
	var stored models.Message
	if err := db.DB.Where("conversation_id = ? AND external_message_id = ?", convID, msg.ExternalID).Take(&stored).Error; err != nil || stored.Content != "B" {
		t.Fatalf("stale worker replaced B's message: %+v, %v", stored, err)
	}
	var conversation models.Conversation
	if err := db.DB.Take(&conversation, "id = ?", convID).Error; err != nil || conversation.MessageCount != 1 {
		t.Fatalf("stale count write: %+v, %v", conversation, err)
	}
	var after models.Channel
	if err := db.DB.Take(&after, "id = ?", f.chA).Error; err != nil || string(after.CredentialsEncrypted) != string(before.CredentialsEncrypted) || after.SyncRunID == nil || *after.SyncRunID != owner.RunID {
		t.Fatalf("stale credentials or owner changed: %v, %v", after.SyncRunID, err)
	}
	if err := eng.persistZaloRefreshedTokens(owner, "fresh-access", "fresh-refresh"); err != nil {
		t.Fatalf("owner token persist: %v", err)
	}
	if err := db.DB.Take(&after, "id = ?", f.chA).Error; err != nil {
		t.Fatal(err)
	}
	clear, err := pkg.Decrypt(after.CredentialsEncrypted, f.cfg.EncryptionKey)
	if err != nil || !strings.Contains(string(clear), "fresh-access") {
		t.Fatalf("owner token was not persisted: %v", err)
	}
}

func TestOwnedSideEffectRejectsForgedTenantAndBoundsDBError(t *testing.T) {
	f := setupSFFixture(t)
	eng := NewSyncEngine(f.cfg)
	owner := f.mustReserve(t, f.tenantID, f.chA)
	forged := owner
	forged.RunID = pkg.NewUUID()
	wrongTenant := owner
	wrongTenant.TenantID = f.otherTenantID
	for _, bad := range []SyncReservation{forged, wrongTenant, {TenantID: owner.TenantID, ChannelID: owner.ChannelID}} {
		if _, err := eng.upsertConversation(bad, channels.SyncedConversation{ExternalID: "forged"}); !errors.Is(err, ErrSyncOwnershipLost) {
			t.Fatalf("bad reservation %+v: %v", bad, err)
		}
	}
	convID, err := eng.upsertConversation(owner, channels.SyncedConversation{ExternalID: "db-failure", LastMessageAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	trigger := "r015_msg_" + f.chA[len(f.chA)-8:]
	if err := db.DB.Exec(fmt.Sprintf("CREATE TRIGGER %s BEFORE INSERT ON messages FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'secret raw DB failure'", trigger)).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trigger) })
	_, err = eng.upsertConversation(owner, channels.SyncedConversation{ExternalID: "other", LastMessageAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	sink := withGormSink(gormSinkConfigs[0], func() {
		err = eng.upsertMessage(owner, convID, channels.SyncedMessage{ExternalID: "forced", SenderType: "customer", SentAt: time.Now()})
	})
	if !errors.Is(err, ErrSyncWriteFailed) || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), owner.RunID) {
		t.Fatalf("raw DB error escaped: %v", err)
	}
	if strings.Contains(sink, "secret raw DB failure") || strings.Contains(sink, owner.RunID) || strings.Contains(sink, "sync_run_id") {
		t.Fatalf("run or SQL values reached GORM sink: %q", sink)
	}
}

func TestZaloRefreshWriteErrorIsBoundedAndDoesNotChangeCredential(t *testing.T) {
	f := setupSFFixture(t)
	owner := f.mustReserve(t, f.tenantID, f.chA)
	var before models.Channel
	if err := db.DB.Take(&before, "id = ?", f.chA).Error; err != nil {
		t.Fatal(err)
	}
	f.addTrigger(t, f.chA, "IF OLD.credentials_encrypted <> NEW.credentials_encrypted THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'raw credential write failure'; END IF;")
	err := NewSyncEngine(f.cfg).persistZaloRefreshedTokens(owner, "new-access", "new-refresh")
	if !errors.Is(err, ErrSyncWriteFailed) || strings.Contains(err.Error(), "raw credential") {
		t.Fatalf("credential write error escaped: %v", err)
	}
	var after models.Channel
	if err := db.DB.Take(&after, "id = ?", f.chA).Error; err != nil || string(after.CredentialsEncrypted) != string(before.CredentialsEncrypted) {
		t.Fatalf("credential changed despite failed write: %v", err)
	}
}

func TestOwnedWriteSerializesWithOwnershipChange(t *testing.T) {
	f := setupSFFixture(t)
	old := f.mustReserve(t, f.tenantID, f.chA)
	entered, release := make(chan struct{}), make(chan struct{})
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- withOwnedSyncWrite(old, func(tx *gorm.DB) error {
			close(entered)
			<-release
			return tx.Model(&models.Channel{}).Where("id = ?", f.chA).Update("name", "written-before-release").Error
		})
	}()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("owned write did not acquire the channel lock")
	}
	releaseDone := make(chan error, 1)
	go func() {
		releaseDone <- db.DB.Exec("UPDATE channels SET last_sync_status = 'idle', sync_run_id = NULL WHERE id = ?", f.chA).Error
	}()
	select {
	case err := <-releaseDone:
		t.Fatalf("ownership changed before the owned write committed: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	if err := <-writeDone; err != nil {
		t.Fatal(err)
	}
	if err := <-releaseDone; err != nil {
		t.Fatal(err)
	}
	newOwner := f.mustReserve(t, f.tenantID, f.chA)
	if newOwner.RunID == old.RunID || !errors.Is(checkSyncOwnership(old), ErrSyncOwnershipLost) {
		t.Fatal("old ownership remained after takeover")
	}
}

func TestAttachmentReplayReusesObjectAndNewAttemptKeyIsIsolated(t *testing.T) {
	f := setupSFFixture(t)
	storage.SetConfigLoader(func(string) (storage.Config, error) {
		return storage.Config{Backend: "local", BaseDir: f.cfg.StorageLocalDir}, nil
	})
	t.Cleanup(func() { storage.SetConfigLoader(nil) })
	eng := NewSyncEngine(f.cfg)
	owner := f.mustReserve(t, f.tenantID, f.chA)
	appLogs := captureEngineLog(t)
	convID, err := eng.upsertConversation(owner, channels.SyncedConversation{ExternalID: "files", LastMessageAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, "original bytes")
	}))
	defer server.Close()
	attachment := channels.Attachment{Type: "image", URL: server.URL + "/one?token=synthetic-secret", Name: "same.jpg"}
	message := channels.SyncedMessage{ExternalID: "file-1", SenderType: "customer", SentAt: time.Now(), Attachments: []channels.Attachment{attachment}}
	keys, err := eng.downloadAttachments(context.Background(), owner, convID, &message)
	if err != nil || len(keys) != 1 {
		t.Fatalf("initial download: %v, keys %v", err, keys)
	}
	if err := eng.upsertMessage(owner, convID, message); err != nil {
		t.Fatal(err)
	}
	replay := channels.SyncedMessage{ExternalID: message.ExternalID, SenderType: "customer", SentAt: message.SentAt, Attachments: []channels.Attachment{attachment}}
	replayKeys, err := eng.downloadAttachments(context.Background(), owner, convID, &replay)
	if err != nil || len(replayKeys) != 0 || replay.Attachments[0].LocalPath != keys[0] || calls.Load() != 1 {
		t.Fatalf("unchanged attachment caused a second object: %v, %v, %d calls", err, replayKeys, calls.Load())
	}
	f.simulateRecoveryRelease(t, f.chA)
	newOwner := f.mustReserve(t, f.tenantID, f.chA)
	newMessage := channels.SyncedMessage{ExternalID: message.ExternalID, SenderType: "customer", SentAt: message.SentAt,
		Attachments: []channels.Attachment{{Type: "image", URL: server.URL + "/two", Name: "same.jpg"}}}
	newKeys, err := eng.downloadAttachments(context.Background(), newOwner, convID, &newMessage)
	if err != nil || len(newKeys) != 1 || newKeys[0] == keys[0] {
		t.Fatalf("new run shared old object key: %v, %v", err, newKeys)
	}
	if err := eng.upsertMessage(newOwner, convID, newMessage); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.downloadAttachments(context.Background(), owner, convID, &replay); !errors.Is(err, ErrSyncOwnershipLost) {
		t.Fatalf("stale download: %v", err)
	}
	var stored models.Message
	if err := db.DB.Where("conversation_id = ? AND external_message_id = ?", convID, message.ExternalID).Take(&stored).Error; err != nil {
		t.Fatal(err)
	}
	var attachments []channels.Attachment
	if err := json.Unmarshal([]byte(stored.Attachments), &attachments); err != nil || len(attachments) != 1 || attachments[0].LocalPath != newKeys[0] {
		t.Fatalf("new run attachment was not published: %s, %v", stored.Attachments, err)
	}
	if strings.Contains(appLogs.String(), "synthetic-secret") {
		t.Fatal("signed attachment URL reached application logs")
	}
}

func TestLegacyAttachmentPathIsReusedWithoutDownload(t *testing.T) {
	f := setupSFFixture(t)
	storage.SetConfigLoader(func(string) (storage.Config, error) {
		return storage.Config{Backend: "local", BaseDir: f.cfg.StorageLocalDir}, nil
	})
	t.Cleanup(func() { storage.SetConfigLoader(nil) })
	eng := NewSyncEngine(f.cfg)
	owner := f.mustReserve(t, f.tenantID, f.chA)
	convID, err := eng.upsertConversation(owner, channels.SyncedConversation{ExternalID: "legacy-file", LastMessageAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.ForTenant(f.tenantID)
	if err != nil {
		t.Fatal(err)
	}
	legacyKey := path.Join(f.tenantID, convID, "legacy.jpg")
	if err := store.Put(context.Background(), legacyKey, strings.NewReader("legacy bytes"), 12, "image/jpeg"); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, "unexpected download")
	}))
	defer server.Close()
	attachment := channels.Attachment{Type: "image", URL: server.URL + "/legacy", Name: "legacy.jpg", LocalPath: legacyKey}
	stored := channels.SyncedMessage{ExternalID: "legacy", SenderType: "customer", SentAt: time.Now(), Attachments: []channels.Attachment{attachment}}
	if err := eng.upsertMessage(owner, convID, stored); err != nil {
		t.Fatal(err)
	}
	attachment.LocalPath = ""
	replay := channels.SyncedMessage{ExternalID: stored.ExternalID, SenderType: stored.SenderType, SentAt: stored.SentAt, Attachments: []channels.Attachment{attachment}}
	keys, err := eng.downloadAttachments(context.Background(), owner, convID, &replay)
	if err != nil || len(keys) != 0 || replay.Attachments[0].LocalPath != legacyKey || calls.Load() != 0 {
		t.Fatalf("legacy object was replaced: %v, %v, %q, calls %d", err, keys, replay.Attachments[0].LocalPath, calls.Load())
	}
}

func TestInFlightOldAttachmentCannotReplaceNewRunObject(t *testing.T) {
	f := setupSFFixture(t)
	storage.SetConfigLoader(func(string) (storage.Config, error) {
		return storage.Config{Backend: "local", BaseDir: f.cfg.StorageLocalDir}, nil
	})
	t.Cleanup(func() { storage.SetConfigLoader(nil) })
	eng := NewSyncEngine(f.cfg)
	old := f.mustReserve(t, f.tenantID, f.chA)
	convID, err := eng.upsertConversation(old, channels.SyncedConversation{ExternalID: "file-race", LastMessageAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			close(entered)
			<-release
			_, _ = io.WriteString(w, "old bytes")
			return
		}
		_, _ = io.WriteString(w, "new bytes")
	}))
	defer server.Close()
	makeMessage := func(url string) channels.SyncedMessage {
		return channels.SyncedMessage{ExternalID: "same-message", SenderType: "customer", SentAt: time.Now(),
			Attachments: []channels.Attachment{{Type: "image", URL: url, Name: "same.jpg"}}}
	}
	oldMsg := makeMessage(server.URL + "/slow")
	type downloadResult struct {
		keys []string
		err  error
	}
	done := make(chan downloadResult, 1)
	go func() {
		keys, err := eng.downloadAttachments(context.Background(), old, convID, &oldMsg)
		done <- downloadResult{keys, err}
	}()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("old download did not start")
	}
	f.simulateRecoveryRelease(t, f.chA)
	newOwner := f.mustReserve(t, f.tenantID, f.chA)
	newMsg := makeMessage(server.URL + "/new")
	newKeys, err := eng.downloadAttachments(context.Background(), newOwner, convID, &newMsg)
	if err != nil || len(newKeys) != 1 {
		t.Fatalf("new download: %v, %v", err, newKeys)
	}
	if err := eng.upsertMessage(newOwner, convID, newMsg); err != nil {
		t.Fatal(err)
	}
	close(release)
	oldDownload := <-done
	if oldDownload.err != nil || len(oldDownload.keys) != 1 || oldDownload.keys[0] == newKeys[0] {
		t.Fatalf("attempt keys collided or old download failed: %+v", oldDownload)
	}
	if err := eng.upsertMessage(old, convID, oldMsg); !errors.Is(err, ErrSyncOwnershipLost) {
		t.Fatalf("old attempt published after takeover: %v", err)
	}
	eng.cleanupAttemptKeys(old.TenantID, convID, oldDownload.keys)
	store, err := storage.ForTenant(f.tenantID)
	if err != nil {
		t.Fatal(err)
	}
	if exists, _ := store.Exists(context.Background(), oldDownload.keys[0]); exists {
		t.Fatal("rejected old attempt object was not cleaned up")
	}
	eng.cleanupAttemptKeys(newOwner.TenantID, convID, newKeys) // uncertain commit must never delete a referenced key
	body, _, _, err := store.Get(context.Background(), newKeys[0])
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := io.ReadAll(body)
	body.Close()
	if err != nil || string(bytes) != "new bytes" {
		t.Fatalf("new run object was overwritten: %q, %v", bytes, err)
	}
	var stored models.Message
	if err := db.DB.Where("conversation_id = ? AND external_message_id = ?", convID, newMsg.ExternalID).Take(&stored).Error; err != nil {
		t.Fatal(err)
	}
	var attached []channels.Attachment
	if err := json.Unmarshal([]byte(stored.Attachments), &attached); err != nil || len(attached) != 1 || attached[0].LocalPath != newKeys[0] {
		t.Fatalf("old attempt displaced new DB path: %s, %v", stored.Attachments, err)
	}
}
