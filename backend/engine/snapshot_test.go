package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
)

func snapshotFixture() (models.Conversation, []models.Message) {
	conv := models.Conversation{ID: "conv-1", TenantID: "tenant-1"}
	base := time.Date(2026, 9, 27, 7, 0, 0, 0, time.UTC)
	return conv, []models.Message{
		{ID: "m-2", TenantID: "tenant-1", ConversationID: "conv-1", ExternalMessageID: "ext-2", SenderType: "agent", SenderName: "NV Lan",
			Content: "Dạ em kiểm tra ngay ạ", ContentType: "text", Attachments: "[]", SentAt: base.Add(time.Minute)},
		{ID: "m-1", TenantID: "tenant-1", ConversationID: "conv-1", ExternalMessageID: "ext-1", SenderType: "customer", SenderName: "Khách",
			Content: "Đơn hàng chưa tới 😡 3 ngày rồi", ContentType: "text", SentAt: base},
	}
}

func mustSnapshot(t *testing.T, conv models.Conversation, msgs []models.Message, omitted int64) *conversationSnapshot {
	t.Helper()
	snap, err := buildConversationSnapshot(conv, msgs, omitted)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	return snap
}

func TestSnapshotDigestIsDeterministic(t *testing.T) {
	conv, msgs := snapshotFixture()
	a := mustSnapshot(t, conv, msgs, 0)
	reversed := []models.Message{msgs[1], msgs[0]}
	b := mustSnapshot(t, conv, reversed, 0)
	if a.Digest != b.Digest || string(a.ManifestJSON) != string(b.ManifestJSON) {
		t.Fatalf("same data in different load order produced different snapshots")
	}
	if len(a.Digest) != 64 || strings.ToLower(a.Digest) != a.Digest {
		t.Fatalf("digest must be lowercase SHA-256 hex, got %q", a.Digest)
	}

	// Same instant in another timezone is the same source version.
	shifted := append([]models.Message(nil), msgs...)
	shifted[0].SentAt = shifted[0].SentAt.In(time.FixedZone("ICT", 7*3600))
	if c := mustSnapshot(t, conv, shifted, 0); c.Digest != a.Digest {
		t.Fatal("digest depends on the timezone of the loaded timestamps")
	}
	if a.Manifest.Messages[0].MessageID != "m-1" || a.Manifest.Messages[0].SentAt != "2026-09-27T07:00:00Z" {
		t.Fatalf("manifest not ordered/UTC: %+v", a.Manifest.Messages[0])
	}
}

func TestSnapshotDigestChangesWithSource(t *testing.T) {
	conv, msgs := snapshotFixture()
	base := mustSnapshot(t, conv, msgs, 0).Digest
	mutations := map[string]func(m *models.Message){
		"content":      func(m *models.Message) { m.Content += "!" },
		"role":         func(m *models.Message) { m.SenderType = "customer" },
		"timestamp":    func(m *models.Message) { m.SentAt = m.SentAt.Add(time.Nanosecond) },
		"content type": func(m *models.Message) { m.ContentType = "image" },
		"attachment":   func(m *models.Message) { m.Attachments = `[{"type":"image","url":"https://x/y.png"}]` },
	}
	for name, mutate := range mutations {
		changed := append([]models.Message(nil), msgs...)
		mutate(&changed[0])
		if d := mustSnapshot(t, conv, changed, 0).Digest; d == base {
			t.Errorf("changing %s did not change digest", name)
		}
	}
}

func TestSnapshotCoverage(t *testing.T) {
	conv, msgs := snapshotFixture()
	cases := []struct {
		name    string
		msgs    []models.Message
		omitted int64
		want    string
		reason  string
	}{
		{"complete", msgs, 0, coverageComplete, ""},
		{"empty", nil, 0, coverageEmpty, reasonNoMessages},
		{"windowed history", msgs, 3, coveragePartial, reasonHistoryWindowed},
		{"sticker", withMessage(msgs, func(m *models.Message) { m.ContentType = "sticker" }), 0, coveragePartial, reasonUnsupportedContentType},
		{"attachment", withMessage(msgs, func(m *models.Message) { m.Attachments = `[{"type":"file","url":"u"}]` }), 0, coveragePartial, reasonAttachmentNotRepresented},
		{"bad attachment json", withMessage(msgs, func(m *models.Message) { m.Attachments = `{broken` }), 0, coveragePartial, reasonAttachmentJSONInvalid},
	}
	for _, tc := range cases {
		snap := mustSnapshot(t, conv, tc.msgs, tc.omitted)
		if snap.Manifest.Coverage != tc.want {
			t.Errorf("%s: coverage %q, want %q", tc.name, snap.Manifest.Coverage, tc.want)
		}
		if tc.reason == "" && len(snap.Manifest.CoverageReasons) != 0 {
			t.Errorf("%s: unexpected reasons %v", tc.name, snap.Manifest.CoverageReasons)
		}
		if tc.reason != "" && !contains(snap.Manifest.CoverageReasons, tc.reason) {
			t.Errorf("%s: reasons %v lack %s", tc.name, snap.Manifest.CoverageReasons, tc.reason)
		}
	}
}

func TestSnapshotTranscriptCarriesIdentity(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)
	want := "[2026-09-27T14:00:00+07:00 | msg:m-1] Khách (customer): Đơn hàng chưa tới 😡 3 ngày rồi\n" +
		"[2026-09-27T14:01:00+07:00 | msg:m-2] NV Lan (agent): Dạ em kiểm tra ngay ạ\n"
	if snap.Transcript != want {
		t.Fatalf("transcript = %q", snap.Transcript)
	}
	if strings.Contains(string(snap.ManifestJSON), "Đơn hàng") {
		t.Fatal("manifest must store content hashes, not raw content")
	}
}

func TestSnapshotRejectsForeignMessage(t *testing.T) {
	conv, msgs := snapshotFixture()
	msgs[0].ConversationID = "conv-2"
	if _, err := buildConversationSnapshot(conv, msgs, 0); err == nil {
		t.Fatal("snapshot accepted a message from another conversation")
	}
}

func TestEvidenceRefsUnicodeCodePointOffsets(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)
	// "Đơn hàng chưa tới 😡 3 ngày rồi": 😡 is one code point at index 18; "chưa" starts at 9.
	ok := []evidenceRef{
		{MessageID: "m-1", Quote: "chưa tới 😡", Start: intp(9), End: intp(19)},
		{MessageID: "m-1", Quote: "3 ngày rồi", Start: intp(20), End: intp(30)},
		{MessageID: "m-2", Quote: "kiểm tra ngay"},
	}
	if err := snap.validateEvidenceRefs(ok); err != nil {
		t.Fatalf("valid refs rejected: %v", err)
	}
	// Byte offsets for the same quote must fail.
	byteStart := strings.Index(msgs[1].Content, "3 ngày rồi")
	bad := []evidenceRef{{MessageID: "m-1", Quote: "3 ngày rồi", Start: intp(byteStart), End: intp(byteStart + len("3 ngày rồi"))}}
	if err := snap.validateEvidenceRefs(bad); err == nil {
		t.Fatal("byte offsets accepted as code-point offsets")
	}
}

func TestEvidenceRefsRejectInvalid(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)
	cases := map[string][]evidenceRef{
		"empty refs":         {},
		"unknown message":    {{MessageID: "m-404", Quote: "Dạ"}},
		"cross conversation": {{MessageID: "other-conv-msg", Quote: "Đơn hàng"}},
		"wrong quote":        {{MessageID: "m-1", Quote: "don hang chua toi"}},
		"quote in other msg": {{MessageID: "m-2", Quote: "Đơn hàng"}},
		"empty quote":        {{MessageID: "m-1", Quote: ""}},
		"offset mismatch":    {{MessageID: "m-1", Quote: "Đơn", Start: intp(1), End: intp(4)}},
		"only start":         {{MessageID: "m-1", Quote: "Đơn", Start: intp(0)}},
		"out of range":       {{MessageID: "m-1", Quote: "rồi", Start: intp(27), End: intp(99)}},
		"one bad of two":     {{MessageID: "m-1", Quote: "Đơn"}, {MessageID: "m-2", Quote: "không có"}},
	}
	for name, refs := range cases {
		if err := snap.validateEvidenceRefs(refs); err == nil {
			t.Errorf("%s: invalid refs accepted", name)
		}
	}

	// A message from another conversation's snapshot is not citable here.
	other := mustSnapshot(t, models.Conversation{ID: "conv-2", TenantID: "tenant-1"}, []models.Message{
		{ID: "other-conv-msg", TenantID: "tenant-1", ConversationID: "conv-2", Content: "Đơn hàng khác", SentAt: time.Now()},
	}, 0)
	if err := other.validateEvidenceRefs([]evidenceRef{{MessageID: "other-conv-msg", Quote: "Đơn hàng"}}); err != nil {
		t.Fatalf("control: ref valid in its own conversation was rejected: %v", err)
	}
}

func withMessage(msgs []models.Message, mutate func(*models.Message)) []models.Message {
	out := append([]models.Message(nil), msgs...)
	mutate(&out[0])
	return out
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func intp(v int) *int { return &v }
