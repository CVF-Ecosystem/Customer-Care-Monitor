package engine

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/storage"
)

type repairAdapter struct {
	conversations []channels.SyncedConversation
	messages      map[string][]channels.SyncedMessage
}

func (a *repairAdapter) FetchRecentConversations(context.Context, time.Time, int) ([]channels.SyncedConversation, error) {
	return a.conversations, nil
}
func (a *repairAdapter) FetchMessages(_ context.Context, id string, _ time.Time) ([]channels.SyncedMessage, error) {
	return a.messages[id], nil
}
func (a *repairAdapter) HealthCheck(context.Context) error { return nil }

type failingProbeStore struct {
	storage.Store
	probes atomic.Int32
	puts   atomic.Int32
}

func (s *failingProbeStore) Exists(context.Context, string) (bool, error) {
	s.probes.Add(1)
	return false, errors.New("synthetic primary probe unavailable")
}
func (s *failingProbeStore) Put(context.Context, string, io.Reader, int64, string) error {
	s.puts.Add(1)
	return errors.New("synthetic primary put unavailable")
}
func (s *failingProbeStore) Kind() string { return "synthetic-s3" }

func TestSyncReservedChannelContinuesAfterOldAttachmentProbeError(t *testing.T) {
	f := setupSFFixture(t)
	if err := db.DB.Model(&models.Channel{}).Where("id = ?", f.chA).Update("metadata", `{"sync_files":true}`).Error; err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "new attachment bytes")
	}))
	defer server.Close()
	attachment := channels.Attachment{Type: "image", URL: server.URL + "/image", Name: "image.jpg"}
	adapter := &repairAdapter{
		conversations: []channels.SyncedConversation{{ExternalID: "probe-conv", CustomerName: "Customer", LastMessageAt: time.Now()}},
		messages: map[string][]channels.SyncedMessage{"probe-conv": {
			{ExternalID: "file", SenderType: "customer", Content: "file", SentAt: time.Now(), Attachments: []channels.Attachment{attachment}},
			{ExternalID: "later", SenderType: "customer", Content: "later", SentAt: time.Now()},
		}},
	}
	newSyncAdapter = func(string, []byte) (channels.ChannelAdapter, error) { return adapter, nil }
	eng := NewSyncEngine(f.cfg)
	owner := f.mustReserve(t, f.tenantID, f.chA)
	convID, err := eng.upsertConversation(owner, adapter.conversations[0])
	if err != nil {
		t.Fatal(err)
	}
	old := adapter.messages["probe-conv"][0]
	old.Attachments = []channels.Attachment{{Type: attachment.Type, URL: attachment.URL, Name: attachment.Name, LocalPath: "old/unconfirmed.jpg"}}
	if err := eng.upsertMessage(owner, convID, old); err != nil {
		t.Fatal(err)
	}
	local, err := storage.NewLocal(f.cfg.StorageLocalDir)
	if err != nil {
		t.Fatal(err)
	}
	primary := &failingProbeStore{Store: local}
	eng.storeForTenant = func(string) (storage.Store, error) { return primary, nil }
	if err := eng.SyncReservedChannel(context.Background(), f.channel(t, f.chA), owner); err != nil {
		t.Fatalf("storage probe error should use fresh download/local fallback: %v", err)
	}
	if primary.probes.Load() != 1 || primary.puts.Load() != 1 {
		t.Fatalf("probe/primary put counts = %d/%d, want 1/1", primary.probes.Load(), primary.puts.Load())
	}
	var stored models.Message
	if err := db.DB.Where("tenant_id = ? AND conversation_id = ? AND external_message_id = 'file'", f.tenantID, convID).Take(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Attachments == "" || stored.Attachments == "[]" {
		t.Fatal("fresh attachment was not published")
	}
	var saved []channels.Attachment
	if err := json.Unmarshal([]byte(stored.Attachments), &saved); err != nil || len(saved) != 1 || saved[0].LocalPath == "" || saved[0].LocalPath == "old/unconfirmed.jpg" {
		t.Fatalf("fresh attachment key not published: %q, %v", stored.Attachments, err)
	}
	if exists, err := local.Exists(context.Background(), saved[0].LocalPath); err != nil || !exists {
		t.Fatalf("local fallback did not save the new object: %v, %v", exists, err)
	}
	var later int64
	if err := db.DB.Model(&models.Message{}).Where("tenant_id = ? AND conversation_id = ? AND external_message_id = 'later'", f.tenantID, convID).Count(&later).Error; err != nil || later != 1 {
		t.Fatalf("later message not persisted: count %d, error %v", later, err)
	}
	if status := f.statusOf(t, f.chA).Status; status != "success" {
		t.Fatalf("run status %q, want success", status)
	}
}

func TestSyncReservedChannelStopsAfterFirstConversationLosesOwnership(t *testing.T) {
	f := setupSFFixture(t)
	adapter := &repairAdapter{
		conversations: []channels.SyncedConversation{
			{ExternalID: "first-A", CustomerName: "A", LastMessageAt: time.Now()},
			{ExternalID: "second-A", CustomerName: "A", LastMessageAt: time.Now()},
		},
		messages: map[string][]channels.SyncedMessage{
			"first-A":  {{ExternalID: "first-message-A", SenderType: "customer", Content: "A", SentAt: time.Now()}},
			"second-A": {{ExternalID: "second-message-A", SenderType: "customer", Content: "A", SentAt: time.Now()}},
		},
	}
	newSyncAdapter = func(string, []byte) (channels.ChannelAdapter, error) { return adapter, nil }
	eng := NewSyncEngine(f.cfg)
	old := f.mustReserve(t, f.tenantID, f.chA)
	var newOwner SyncReservation
	completed := 0
	eng.afterConversation = func() {
		completed++
		if completed != 1 {
			return
		}
		f.simulateRecoveryRelease(t, f.chA)
		newOwner = f.mustReserve(t, f.tenantID, f.chA)
		convID, err := eng.upsertConversation(newOwner, channels.SyncedConversation{ExternalID: "owned-B", CustomerName: "B", LastMessageAt: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
		if err := eng.upsertMessage(newOwner, convID, channels.SyncedMessage{ExternalID: "message-B", SenderType: "customer", Content: "B", SentAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
		if err := eng.updateOwnedMessageCount(newOwner, convID); err != nil {
			t.Fatal(err)
		}
	}
	err := eng.SyncReservedChannel(context.Background(), f.channel(t, f.chA), old)
	if !errors.Is(err, ErrSyncOwnershipLost) || completed != 1 || newOwner.RunID == old.RunID {
		t.Fatalf("A continued or returned partial after takeover: %v, completed %d", err, completed)
	}
	for id, want := range map[string]int64{"first-A": 1, "second-A": 0, "owned-B": 1} {
		var count int64
		if err := db.DB.Model(&models.Conversation{}).Where("tenant_id = ? AND channel_id = ? AND external_conversation_id = ?", f.tenantID, f.chA, id).Count(&count).Error; err != nil || count != want {
			t.Fatalf("conversation %s: count %d, want %d, error %v", id, count, want, err)
		}
	}
	for id, want := range map[string]int64{"first-message-A": 1, "second-message-A": 0, "message-B": 1} {
		var count int64
		if err := db.DB.Model(&models.Message{}).Where("tenant_id = ? AND external_message_id = ?", f.tenantID, id).Count(&count).Error; err != nil || count != want {
			t.Fatalf("message %s: count %d, want %d, error %v", id, count, want, err)
		}
	}
	var b models.Conversation
	if err := db.DB.Where("tenant_id = ? AND channel_id = ? AND external_conversation_id = 'owned-B'", f.tenantID, f.chA).Take(&b).Error; err != nil || b.MessageCount != 1 {
		t.Fatalf("B count changed: %+v, error %v", b, err)
	}
	if status := f.statusOf(t, f.chA).Status; status != "syncing" {
		t.Fatalf("B status changed to %q", status)
	}
	if runID := f.runIDOf(t, f.chA); runID == nil || *runID != newOwner.RunID {
		t.Fatal("B run ID changed after A stopped")
	}
}
