package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
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

func TestSnapshotDigestChangesWithAttachmentIdentity(t *testing.T) {
	conv, msgs := snapshotFixture()
	withAttachment := append([]models.Message(nil), msgs...)
	withAttachment[0].Attachments = `[{"type":"image","url":"https://x/a.png","name":"a.png"}]`
	base := mustSnapshot(t, conv, withAttachment, 0)

	mutations := map[string]string{
		"url":        `[{"type":"image","url":"https://x/b.png","name":"a.png"}]`,
		"name":       `[{"type":"image","url":"https://x/a.png","name":"b.png"}]`,
		"type":       `[{"type":"file","url":"https://x/a.png","name":"a.png"}]`,
		"local_path": `[{"type":"image","url":"https://x/a.png","name":"a.png","local_path":"/tmp/a.png"}]`,
	}
	for name, raw := range mutations {
		changed := append([]models.Message(nil), withAttachment...)
		changed[0].Attachments = raw
		snap := mustSnapshot(t, conv, changed, 0)
		if snap.Digest == base.Digest {
			t.Errorf("changing attachment %s did not change digest", name)
		}
		if snap.Manifest.Coverage != coveragePartial || !contains(snap.Manifest.CoverageReasons, reasonAttachmentNotRepresented) {
			t.Errorf("attachment %s: unexpected coverage %+v", name, snap.Manifest)
		}
	}

	// Same valid attachment JSON produces the same digest.
	repeat := append([]models.Message(nil), msgs...)
	repeat[0].Attachments = withAttachment[0].Attachments
	if mustSnapshot(t, conv, repeat, 0).Digest != base.Digest {
		t.Fatal("identical attachment JSON produced a different digest")
	}

	// Same typed values under different whitespace/key order still collapse to
	// the same digest, because the fingerprint re-encodes the typed struct
	// rather than hashing the original bytes.
	reordered := append([]models.Message(nil), msgs...)
	reordered[0].Attachments = `[ { "url" : "https://x/a.png" , "type": "image" , "name":"a.png" } ]`
	if mustSnapshot(t, conv, reordered, 0).Digest != base.Digest {
		t.Fatal("same typed attachment values under different whitespace/key order produced a different digest")
	}

	// R2-RR2: two different attachments whose fields, joined with an unescaped
	// delimiter, previously collided to the same bytes and therefore the same
	// digest. json.Marshal must distinguish them.
	collisionA := append([]models.Message(nil), msgs...)
	collisionA[0].Attachments = `[{"type":"a\u001fb","url":"c"}]`
	collisionB := append([]models.Message(nil), msgs...)
	collisionB[0].Attachments = `[{"type":"a","url":"b\u001fc"}]`
	collDigestA := mustSnapshot(t, conv, collisionA, 0).Digest
	collDigestB := mustSnapshot(t, conv, collisionB, 0).Digest
	if collDigestA == collDigestB {
		t.Fatal("delimiter-collision attachment pair produced the same digest")
	}

	// Invalid JSON is hashed from the untrimmed raw bytes: a whitespace-only
	// change still moves the digest, since coverage/parsing never trims first.
	badNoSpace := append([]models.Message(nil), msgs...)
	badNoSpace[0].Attachments = `{broken`
	badWithSpace := append([]models.Message(nil), msgs...)
	badWithSpace[0].Attachments = ` {broken`
	if mustSnapshot(t, conv, badNoSpace, 0).Digest == mustSnapshot(t, conv, badWithSpace, 0).Digest {
		t.Fatal("whitespace-only change to invalid attachment JSON did not change digest")
	}

	// Invalid JSON fingerprints the raw bytes, so a source change still moves the digest.
	badA := append([]models.Message(nil), msgs...)
	badA[0].Attachments = `{broken-a`
	badB := append([]models.Message(nil), msgs...)
	badB[0].Attachments = `{broken-b`
	da := mustSnapshot(t, conv, badA, 0).Digest
	db := mustSnapshot(t, conv, badB, 0).Digest
	if da == db {
		t.Fatal("two different invalid attachment payloads produced the same digest")
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
