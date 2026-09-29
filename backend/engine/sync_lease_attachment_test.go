package engine

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/storage"
)

// CCMAI-RUNTIME-016 R016-R1: a failed heartbeat cancels the run even when the
// attachment transfer itself completed successfully at the cancellation
// boundary. The synthetic store's Put waits for the run context to end and
// then still stores the object and returns success; nothing may be published
// afterwards.

// cancelBoundaryStore stores successfully, but only after the run context is
// cancelled. onPut runs first (the test uses it to break the heartbeat).
type cancelBoundaryStore struct {
	storage.Store
	onPut func()
	puts  atomic.Int32
}

func (s *cancelBoundaryStore) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	s.puts.Add(1)
	if s.onPut != nil {
		s.onPut()
		select {
		case <-ctx.Done():
		case <-time.After(10 * time.Second):
			return errors.New("run context was never cancelled")
		}
	}
	return s.Store.Put(context.Background(), key, r, size, contentType)
}
func (s *cancelBoundaryStore) Kind() string { return "synthetic" }

func setupAttachmentLeaseRun(t *testing.T, f *sfFixture) (*SyncEngine, *cancelBoundaryStore, storage.Store, string) {
	t.Helper()
	if err := db.DB.Model(&models.Channel{}).Where("id = ?", f.chA).Update("metadata", `{"sync_files":true}`).Error; err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "attachment bytes")
	}))
	t.Cleanup(server.Close)
	adapter := &repairAdapter{
		conversations: []channels.SyncedConversation{{ExternalID: "hb-conv", CustomerName: "Customer", LastMessageAt: time.Now()}},
		messages: map[string][]channels.SyncedMessage{"hb-conv": {{
			ExternalID: "hb-file", SenderType: "customer", Content: "file", SentAt: time.Now(),
			Attachments: []channels.Attachment{{Type: "image", URL: server.URL + "/img", Name: "img.jpg"}},
		}}},
	}
	newSyncAdapter = func(string, []byte) (channels.ChannelAdapter, error) { return adapter, nil }
	local, err := storage.NewLocal(f.cfg.StorageLocalDir)
	if err != nil {
		t.Fatal(err)
	}
	store := &cancelBoundaryStore{Store: local}
	eng := NewSyncEngine(f.cfg)
	eng.storeForTenant = func(string) (storage.Store, error) { return store, nil }
	return eng, store, local, "hb-conv"
}

func TestHeartbeatFailureDuringSuccessfulAttachmentTransferPublishesNothing(t *testing.T) {
	f := setupSFFixture(t)
	withLeaseTiming(t, 5*time.Minute, 30*time.Millisecond)
	eng, store, local, extConv := setupAttachmentLeaseRun(t, f)
	owner := f.mustReserve(t, f.tenantID, f.chA)
	if !owner.Leased {
		t.Fatal("the fixture run must be leased for this test")
	}
	// The heartbeat starts to fail only once the transfer is in flight.
	store.onPut = func() {
		f.addTrigger(t, f.chA, "IF NEW.sync_lease_until IS NOT NULL AND NOT (NEW.sync_lease_until <=> OLD.sync_lease_until) THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced heartbeat failure'; END IF;")
	}
	logs := captureEngineLog(t)

	err := eng.SyncReservedChannel(context.Background(), f.channel(t, f.chA), owner)

	if !errors.Is(err, ErrSyncWriteFailed) {
		t.Fatalf("run returned %v, want the bounded heartbeat write-failure class", err)
	}
	if store.puts.Load() != 1 {
		t.Fatalf("store Put ran %d times; the boundary was not exercised", store.puts.Load())
	}
	var messages int64
	if err := db.DB.Model(&models.Message{}).Where("tenant_id = ? AND external_message_id = 'hb-file'", f.tenantID).Count(&messages).Error; err != nil || messages != 0 {
		t.Fatalf("a message was persisted after the heartbeat failed: count %d, err %v", messages, err)
	}
	var conv models.Conversation
	if err := db.DB.Where("tenant_id = ? AND external_conversation_id = ?", f.tenantID, extConv).Take(&conv).Error; err != nil || conv.MessageCount != 0 {
		t.Fatalf("count written after heartbeat failure: %+v, %v", conv, err)
	}
	row := f.lease(t, f.chA)
	if row.Status != "error" || row.RunID != "" || row.LeaseSet || row.LastSyncAt != nil {
		t.Fatalf("owned terminal state after the abort: %+v", row)
	}
	if f.triggerCount() != 0 {
		t.Fatalf("analysis triggered after heartbeat failure: %v", f.triggers)
	}
	if strings.Contains(logs.String(), "forced") || strings.Contains(logs.String(), owner.RunID) {
		t.Fatalf("detail leaked: %q", logs.String())
	}
	// Only the run's new attempt key is cleaned up: the object it stored is gone.
	keys := 0
	if err := local.List(context.Background(), f.tenantID+"/", func(string, int64) error { keys++; return nil }); err != nil || keys != 0 {
		t.Fatalf("new attempt object left behind: %d keys, err %v", keys, err)
	}
}

// Detector: with a healthy heartbeat the same setup publishes the message, so
// the absence assertions above are meaningful, and normal per-attachment
// partial semantics are untouched.
func TestSuccessfulAttachmentTransferWithHealthyHeartbeatIsPublished(t *testing.T) {
	f := setupSFFixture(t)
	withLeaseTiming(t, 5*time.Minute, 30*time.Millisecond)
	eng, store, _, extConv := setupAttachmentLeaseRun(t, f)
	owner := f.mustReserve(t, f.tenantID, f.chA)

	if err := eng.SyncReservedChannel(context.Background(), f.channel(t, f.chA), owner); err != nil {
		t.Fatalf("healthy run: %v", err)
	}
	if store.puts.Load() != 1 {
		t.Fatalf("store Put ran %d times", store.puts.Load())
	}
	var messages int64
	if err := db.DB.Model(&models.Message{}).Where("tenant_id = ? AND external_message_id = 'hb-file'", f.tenantID).Count(&messages).Error; err != nil || messages != 1 {
		t.Fatalf("message not published: count %d, err %v", messages, err)
	}
	var conv models.Conversation
	if err := db.DB.Where("tenant_id = ? AND external_conversation_id = ?", f.tenantID, extConv).Take(&conv).Error; err != nil || conv.MessageCount != 1 {
		t.Fatalf("count not written: %+v, %v", conv, err)
	}
	if row := f.lease(t, f.chA); row.Status != "success" || row.LastSyncAt == nil || f.triggerCount() != 1 {
		t.Fatalf("final state %+v, triggers %d", row, f.triggerCount())
	}
}

// A failed attachment transfer with a healthy heartbeat keeps the existing
// partial semantics (the message is still stored, the channel ends partial).
func TestFailedAttachmentTransferWithHealthyHeartbeatStaysPartial(t *testing.T) {
	f := setupSFFixture(t)
	withLeaseTiming(t, 5*time.Minute, 30*time.Millisecond)
	eng, _, _, _ := setupAttachmentLeaseRun(t, f)
	eng.storeForTenant = func(string) (storage.Store, error) {
		return &failingProbeStore{Store: mustLocal(t, f.cfg.StorageLocalDir)}, nil
	}
	// The local fallback (same dir) succeeds even though the primary fails, so
	// force a transfer failure with an unreachable attachment URL instead.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }))
	t.Cleanup(server.Close)
	newSyncAdapter = func(string, []byte) (channels.ChannelAdapter, error) {
		return &repairAdapter{
			conversations: []channels.SyncedConversation{{ExternalID: "hb-conv", CustomerName: "Customer", LastMessageAt: time.Now()}},
			messages: map[string][]channels.SyncedMessage{"hb-conv": {{
				ExternalID: "hb-file", SenderType: "customer", Content: "file", SentAt: time.Now(),
				Attachments: []channels.Attachment{{Type: "image", URL: server.URL + "/img", Name: "img.jpg"}},
			}}},
		}, nil
	}
	owner := f.mustReserve(t, f.tenantID, f.chA)

	err := eng.SyncReservedChannel(context.Background(), f.channel(t, f.chA), owner)

	if err == nil || errors.Is(err, ErrSyncWriteFailed) || errors.Is(err, ErrSyncOwnershipLost) {
		t.Fatalf("got %v, want the ordinary partial report", err)
	}
	var messages int64
	if e := db.DB.Model(&models.Message{}).Where("tenant_id = ? AND external_message_id = 'hb-file'", f.tenantID).Count(&messages).Error; e != nil || messages != 1 {
		t.Fatalf("message not stored on partial: %d, %v", messages, e)
	}
	if row := f.lease(t, f.chA); row.Status != "partial" || row.LastSyncAt != nil || row.LeaseSet {
		t.Fatalf("final state %+v, want partial without checkpoint", row)
	}
}

func mustLocal(t *testing.T, dir string) storage.Store {
	t.Helper()
	s, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
