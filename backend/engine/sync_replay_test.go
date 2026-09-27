package engine

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/pkg"
)

// syncReplayFixture is a bare tenant/channel/conversation, reusing the
// connectTestDB helper already shared by snapshot_db_test.go.
type syncReplayFixture struct {
	suffix, tenantID, channelID, convID string
}

func setupSyncReplayFixture(t *testing.T) *syncReplayFixture {
	t.Helper()
	connectTestDB(t)
	suffix := pkg.NewUUID()[:8]
	f := &syncReplayFixture{
		suffix:    suffix,
		tenantID:  "replay-" + suffix,
		channelID: "ch-replay-" + suffix,
		convID:    "conv-replay-" + suffix,
	}
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Replay Test', ?, '{}', NOW(), NOW())`,
		f.tenantID, f.tenantID)
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh replay', 'fake', X'00', true, '{}', NOW(), NOW())`,
		f.channelID, f.tenantID)
	exec(`INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, 'ext-replay', 'Khach', NOW(), 0, '{}', NOW(), NOW())`,
		f.convID, f.tenantID, f.channelID)

	t.Cleanup(func() {
		db.DB.Exec("DELETE FROM messages WHERE tenant_id = ?", f.tenantID)
		db.DB.Exec("DELETE FROM conversations WHERE id = ?", f.convID)
		db.DB.Exec("DELETE FROM channels WHERE id = ?", f.channelID)
		db.DB.Exec("DELETE FROM tenants WHERE id = ?", f.tenantID)
	})
	return f
}

func (f *syncReplayFixture) conversation() models.Conversation {
	return models.Conversation{ID: f.convID, TenantID: f.tenantID}
}

// digest builds the same shared snapshot single/batch analysis uses, so a
// test asserting "digest changed" is asserting exactly what analysis would
// see, not a proxy for it.
func (f *syncReplayFixture) digest(t *testing.T) string {
	t.Helper()
	snap, err := loadConversationSnapshot(f.conversation(), time.Time{})
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	return snap.Digest
}

func (f *syncReplayFixture) loadMessage(t *testing.T, externalID string) models.Message {
	t.Helper()
	var m models.Message
	if err := db.DB.Where("tenant_id = ? AND conversation_id = ? AND external_message_id = ?",
		f.tenantID, f.convID, externalID).First(&m).Error; err != nil {
		t.Fatalf("load message %s: %v", externalID, err)
	}
	return m
}

func (f *syncReplayFixture) countMessages(t *testing.T, externalID string) int64 {
	t.Helper()
	var n int64
	if err := db.DB.Model(&models.Message{}).Where("tenant_id = ? AND conversation_id = ? AND external_message_id = ?",
		f.tenantID, f.convID, externalID).Count(&n).Error; err != nil {
		t.Fatalf("count message %s: %v", externalID, err)
	}
	return n
}

func (f *syncReplayFixture) failOnUpdateTrigger(t *testing.T, name, messageID, reason string) {
	t.Helper()
	// messageID is a UUID (hex + hyphens only), safe to inline: a trigger body
	// cannot bind query parameters.
	if err := db.DB.Exec(fmt.Sprintf(`CREATE TRIGGER %s BEFORE UPDATE ON messages
FOR EACH ROW
BEGIN
	IF OLD.id = '%s' THEN
		SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = '%s';
	END IF;
END`, name, messageID, reason)).Error; err != nil {
		t.Fatalf("fixture trigger: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + name) })
}

// TestUpsertMessageReplayUpdatesStaleContentAndDigest is the central
// stale-message regression named by the spec. Before this tranche,
// upsertMessage only updated an existing row's attachments, and only when a
// fresh local path was supplied — a replay carrying edited Vietnamese/emoji
// text, sender name and timestamp left the stored row, and the snapshot
// digest describing it, silently stale. This test fails against that
// pre-change behavior (it would see the old content/digest survive).
func TestUpsertMessageReplayUpdatesStaleContentAndDigest(t *testing.T) {
	f := setupSyncReplayFixture(t)
	eng := &SyncEngine{}
	const extID = "m-stale-1"
	sentAt1 := time.Now().Add(-10 * time.Minute).UTC().Truncate(time.Second)

	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach A",
		Content: "Đơn hàng của mình đâu rồi ạ", ContentType: "text", SentAt: sentAt1,
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}
	digestBefore := f.digest(t)

	sentAt2 := sentAt1.Add(3 * time.Minute)
	newContent := "Đơn hàng của mình đâu rồi ạ 😡 — sao lâu thế!!"
	newSender := "Khach A (đã đổi tên) 😅"
	// R003-R2: the SPEC acceptance is a changed sender role/name together —
	// a customer message getting corrected/relabeled as an agent one is an
	// artificial but valid role transition for exercising the assertion.
	newSenderType := "agent"
	newContentType := "sticker"
	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: newSenderType, SenderName: newSender,
		Content: newContent, ContentType: newContentType, SentAt: sentAt2,
	}); err != nil {
		t.Fatalf("replay upsert: %v", err)
	}

	if n := f.countMessages(t, extID); n != 1 {
		t.Fatalf("replay produced %d rows for the same external ID, want exactly 1", n)
	}
	updated := f.loadMessage(t, extID)
	if updated.Content != newContent {
		t.Errorf("content not updated by replay: %q", updated.Content)
	}
	if updated.SenderName != newSender {
		t.Errorf("sender_name not updated by replay: %q", updated.SenderName)
	}
	if updated.SenderType != newSenderType {
		t.Errorf("sender_type (role) not updated by replay: %q", updated.SenderType)
	}
	if updated.ContentType != newContentType {
		t.Errorf("content_type not updated by replay: %q", updated.ContentType)
	}
	if !updated.SentAt.Equal(sentAt2) {
		t.Errorf("sent_at not updated by replay: got %v want %v", updated.SentAt, sentAt2)
	}

	if digestAfter := f.digest(t); digestAfter == digestBefore {
		t.Fatal("snapshot digest did not change after a replay updated stored content")
	}
}

// TestUpsertMessageIdenticalReplayIsNoOpAndPreservesAttachmentLocalPath
// proves two acceptance points together: an identical replay causes no
// meaningful DB mutation, and an attachment that keeps the same source
// identity (type, URL, name) keeps its previously downloaded local path even
// when this reply doesn't carry one (e.g. file sync is off, or it simply
// wasn't re-downloaded). The no-op claim is proven with a BEFORE UPDATE
// trigger that turns any UPDATE on this row into an error — if
// upsertMessage attempted one, the call would return that error instead of
// nil.
func TestUpsertMessageIdenticalReplayIsNoOpAndPreservesAttachmentLocalPath(t *testing.T) {
	f := setupSyncReplayFixture(t)
	eng := &SyncEngine{}
	const extID = "m-idem-1"
	sentAt := time.Now().Add(-5 * time.Minute).UTC().Truncate(time.Second)
	stored := channels.Attachment{Type: "image", URL: "https://cdn.example/a.jpg", Name: "a.jpg", LocalPath: "tenant/conv/a.jpg"}

	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach B",
		Content: "Cho mình hỏi giá ạ", ContentType: "attachment", SentAt: sentAt,
		Attachments: []channels.Attachment{stored},
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}
	before := f.loadMessage(t, extID)
	f.failOnUpdateTrigger(t, "trg_ccma_test_replay_noupdate_"+f.suffix, before.ID, "unexpected UPDATE on identical replay")

	// Same content/sender/time/content-type; same attachment identity but
	// WITHOUT a local path, as a real replay looks when the file isn't
	// re-downloaded.
	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach B",
		Content: "Cho mình hỏi giá ạ", ContentType: "attachment", SentAt: sentAt,
		Attachments: []channels.Attachment{{Type: stored.Type, URL: stored.URL, Name: stored.Name}},
	}); err != nil {
		t.Fatalf("identical replay should be a no-op; got an error (likely the no-update trigger firing): %v", err)
	}

	after := f.loadMessage(t, extID)
	var atts []channels.Attachment
	if err := json.Unmarshal([]byte(after.Attachments), &atts); err != nil {
		t.Fatalf("parse stored attachments: %v", err)
	}
	if len(atts) != 1 || atts[0].LocalPath != stored.LocalPath {
		t.Errorf("same-identity replay must retain the stored local path, got %+v", atts)
	}
}

// TestUpsertMessageAttachmentIdentityChangeDoesNotInheritLocalPath proves the
// other half of the attachment-identity acceptance: a genuinely different
// attachment (new URL/name) must never inherit a stranger's local path just
// because it landed in the same message.
func TestUpsertMessageAttachmentIdentityChangeDoesNotInheritLocalPath(t *testing.T) {
	f := setupSyncReplayFixture(t)
	eng := &SyncEngine{}
	const extID = "m-attid-1"
	sentAt := time.Now().Add(-4 * time.Minute).UTC().Truncate(time.Second)
	oldAtt := channels.Attachment{Type: "image", URL: "https://cdn.example/old.jpg", Name: "old.jpg", LocalPath: "tenant/conv/old.jpg"}

	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach C",
		Content: "Ảnh sản phẩm đây ạ", ContentType: "attachment", SentAt: sentAt,
		Attachments: []channels.Attachment{oldAtt},
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	newAtt := channels.Attachment{Type: "image", URL: "https://cdn.example/new.jpg", Name: "new.jpg"}
	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach C",
		Content: "Ảnh sản phẩm đây ạ", ContentType: "attachment", SentAt: sentAt,
		Attachments: []channels.Attachment{newAtt},
	}); err != nil {
		t.Fatalf("replay upsert: %v", err)
	}

	after := f.loadMessage(t, extID)
	var atts []channels.Attachment
	if err := json.Unmarshal([]byte(after.Attachments), &atts); err != nil {
		t.Fatalf("parse stored attachments: %v", err)
	}
	if len(atts) != 1 {
		t.Fatalf("expected exactly one stored attachment, got %d: %+v", len(atts), atts)
	}
	if atts[0].LocalPath != "" {
		t.Errorf("a different attachment identity must not inherit the old local path, got %q", atts[0].LocalPath)
	}
	if atts[0].URL != newAtt.URL {
		t.Errorf("stored attachment URL = %q, want the replayed %q", atts[0].URL, newAtt.URL)
	}
}

// TestUpsertMessageEmptyAttachmentReplayRetainsStoredList covers spec
// contract point 2: an empty incoming attachment list is indistinguishable
// from an adapter reply that simply didn't carry attachment data, so it must
// never erase a previously stored list — even while other fields on the same
// replay (here: content) do legitimately change.
func TestUpsertMessageEmptyAttachmentReplayRetainsStoredList(t *testing.T) {
	f := setupSyncReplayFixture(t)
	eng := &SyncEngine{}
	const extID = "m-emptyatt-1"
	sentAt := time.Now().Add(-2 * time.Minute).UTC().Truncate(time.Second)
	att := channels.Attachment{Type: "file", URL: "https://cdn.example/f.pdf", Name: "f.pdf", LocalPath: "tenant/conv/f.pdf"}

	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "agent", SenderName: "NV",
		Content: "Đây là hóa đơn của anh/chị", ContentType: "attachment", SentAt: sentAt,
		Attachments: []channels.Attachment{att},
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	newContent := "Đây là hóa đơn đã cập nhật của anh/chị"
	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "agent", SenderName: "NV",
		Content: newContent, ContentType: "attachment", SentAt: sentAt,
	}); err != nil {
		t.Fatalf("replay upsert: %v", err)
	}

	after := f.loadMessage(t, extID)
	if after.Content != newContent {
		t.Errorf("content should still update independently of the attachment field: %q", after.Content)
	}
	var atts []channels.Attachment
	if err := json.Unmarshal([]byte(after.Attachments), &atts); err != nil {
		t.Fatalf("parse stored attachments: %v", err)
	}
	if len(atts) != 1 || atts[0].LocalPath != att.LocalPath {
		t.Errorf("an empty incoming attachment list must not erase the stored list, got %+v", atts)
	}
}

// TestUpsertMessageWriteFailureIsReturnedAndLeavesRowUnchanged is the
// permanent failure-injection regression for the replay path: a forced
// BEFORE UPDATE failure (same trigger technique used by the Gate B
// demo/channel regressions) must surface as a returned error rather than a
// silently stale row. SyncChannel's unchanged handling of that error
// (progress.fail + conversationFailed=true, asserted by
// TestSyncProgressFinalStatus and TestBuildSyncStatusUpdatesOnlyAdvancesSuccessfulCheckpoint
// in sync_status_test.go) is what turns this into the channel's honest
// `partial` status without advancing last_sync_at or running after-sync
// jobs — a direct SyncChannel-level test would need a real adapter, so this
// test proves the boundary it depends on: the write error is observable and
// the row is not left half-mutated.
func TestUpsertMessageWriteFailureIsReturnedAndLeavesRowUnchanged(t *testing.T) {
	f := setupSyncReplayFixture(t)
	eng := &SyncEngine{}
	const extID = "m-fail-1"
	sentAt := time.Now().Add(-1 * time.Minute).UTC().Truncate(time.Second)

	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach D",
		Content: "Noi dung goc", ContentType: "text", SentAt: sentAt,
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}
	before := f.loadMessage(t, extID)
	f.failOnUpdateTrigger(t, "trg_ccma_test_replay_forcefail_"+f.suffix, before.ID, "forced failure for replay write-error test")

	err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach D",
		Content: "Noi dung da sua nhung ghi se loi", ContentType: "text", SentAt: sentAt.Add(time.Minute),
	})
	if err == nil {
		t.Fatal("expected the forced UPDATE failure to be returned as an error, got nil")
	}

	after := f.loadMessage(t, extID)
	if after.Content != before.Content || !after.SentAt.Equal(before.SentAt) {
		t.Errorf("a failed write must not leave a partially applied row: before=%+v after=%+v", before, after)
	}
}

// TestUpsertMessageReplayUpdatesChangedRawData is the R003-R1 regression: a
// replay carrying a changed, nonempty raw-data map (here including a newly
// observed Pancake-style "is_removed" marker) must actually persist, even
// though raw data isn't part of the snapshot digest. This tranche does not
// act on the marker's meaning — it is stored as supplied, nothing more.
func TestUpsertMessageReplayUpdatesChangedRawData(t *testing.T) {
	f := setupSyncReplayFixture(t)
	eng := &SyncEngine{}
	const extID = "m-rawdata-1"
	sentAt := time.Now().Add(-6 * time.Minute).UTC().Truncate(time.Second)

	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach E",
		Content: "Tin nhan goc", ContentType: "text", SentAt: sentAt,
		RawData: map[string]interface{}{"id": "raw-1", "is_removed": false},
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach E",
		Content: "Tin nhan goc", ContentType: "text", SentAt: sentAt,
		RawData: map[string]interface{}{"id": "raw-1", "is_removed": true},
	}); err != nil {
		t.Fatalf("replay upsert: %v", err)
	}

	if n := f.countMessages(t, extID); n != 1 {
		t.Fatalf("replay produced %d rows for the same external ID, want exactly 1", n)
	}
	var stored map[string]interface{}
	after := f.loadMessage(t, extID)
	if err := json.Unmarshal([]byte(after.RawData), &stored); err != nil {
		t.Fatalf("parse stored raw_data: %v", err)
	}
	if removed, _ := stored["is_removed"].(bool); !removed {
		t.Errorf("changed raw_data was not persisted by replay: %+v", stored)
	}
}

// TestUpsertMessageIdenticalRawDataReplayIsNoOp proves an identical raw-data
// replay causes no UPDATE, using the same no-update trigger technique as the
// content/attachment idempotency test.
func TestUpsertMessageIdenticalRawDataReplayIsNoOp(t *testing.T) {
	f := setupSyncReplayFixture(t)
	eng := &SyncEngine{}
	const extID = "m-rawdata-idem-1"
	sentAt := time.Now().Add(-7 * time.Minute).UTC().Truncate(time.Second)
	raw := map[string]interface{}{"id": "raw-2", "note": "khong doi"}

	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach F",
		Content: "Tin nhan on dinh", ContentType: "text", SentAt: sentAt,
		RawData: raw,
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}
	before := f.loadMessage(t, extID)
	f.failOnUpdateTrigger(t, "trg_ccma_test_replay_rawdata_noupdate_"+f.suffix, before.ID, "unexpected UPDATE on identical raw-data replay")

	// Same content/sender/time/content-type and a structurally identical (but
	// distinct map value) raw-data payload — a real replay would decode a
	// fresh map from JSON each time, not reuse the same Go value.
	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach F",
		Content: "Tin nhan on dinh", ContentType: "text", SentAt: sentAt,
		RawData: map[string]interface{}{"id": "raw-2", "note": "khong doi"},
	}); err != nil {
		t.Fatalf("identical raw-data replay should be a no-op; got an error (likely the no-update trigger firing): %v", err)
	}
}

// TestUpsertMessageUnmarshalableRawDataReturnsErrorAndLeavesRowUnchanged is
// the R003-R1 write-error regression: a source raw-data value the standard
// library cannot encode (a NaN float, used only here to force the failure)
// must surface as an error rather than silently keeping the row stale with
// no report.
func TestUpsertMessageUnmarshalableRawDataReturnsErrorAndLeavesRowUnchanged(t *testing.T) {
	f := setupSyncReplayFixture(t)
	eng := &SyncEngine{}
	const extID = "m-rawdata-bad-1"
	sentAt := time.Now().Add(-8 * time.Minute).UTC().Truncate(time.Second)

	if err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach G",
		Content: "Tin nhan hop le", ContentType: "text", SentAt: sentAt,
		RawData: map[string]interface{}{"id": "raw-3"},
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}
	before := f.loadMessage(t, extID)

	err := eng.upsertMessage(f.tenantID, f.convID, channels.SyncedMessage{
		ExternalID: extID, SenderType: "customer", SenderName: "Khach G",
		Content: "Tin nhan hop le", ContentType: "text", SentAt: sentAt,
		RawData: map[string]interface{}{"id": "raw-3", "bad": math.NaN()},
	})
	if err == nil {
		t.Fatal("expected an unmarshalable raw-data value to return an error, got nil")
	}

	after := f.loadMessage(t, extID)
	if after.Content != before.Content || after.RawData != before.RawData {
		t.Errorf("a raw-data marshal failure must not leave a partially applied row: before=%+v after=%+v", before, after)
	}
}
