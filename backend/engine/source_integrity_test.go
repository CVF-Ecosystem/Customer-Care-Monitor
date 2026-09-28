package engine

import (
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// TestCompareSnapshotToCurrentMessagesUnchangedIsBoundCurrentnessUnverified
// is the central CCMAI-RUNTIME-004 regression: comparing a snapshot against
// the exact messages it was built from must never claim more than
// "unverified but locally unchanged" — never a positive freshness claim.
// This fails to even compile against pre-tranche source, since neither the
// function nor the status constants existed before this BUILD.
func TestCompareSnapshotToCurrentMessagesUnchangedIsBoundCurrentnessUnverified(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)

	got := CompareSnapshotToCurrentMessages(string(snap.ManifestJSON), msgs)
	if got != SourceIntegrityBoundCurrentnessUnverified {
		t.Fatalf("unchanged comparison = %q, want %q", got, SourceIntegrityBoundCurrentnessUnverified)
	}
}

func TestCompareSnapshotDetectsEditedContent(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)

	edited := append([]models.Message(nil), msgs...)
	edited[0].Content = edited[0].Content + " — vẫn chưa thấy đâu! 😡"

	got := CompareSnapshotToCurrentMessages(string(snap.ManifestJSON), edited)
	if got != SourceIntegrityChangedSinceAnalysis {
		t.Fatalf("edited content comparison = %q, want %q", got, SourceIntegrityChangedSinceAnalysis)
	}
}

func TestCompareSnapshotDetectsEditedSenderRoleAndName(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)

	edited := append([]models.Message(nil), msgs...)
	edited[0].SenderType = "agent"
	edited[0].SenderName = "NV khác"

	got := CompareSnapshotToCurrentMessages(string(snap.ManifestJSON), edited)
	if got != SourceIntegrityChangedSinceAnalysis {
		t.Fatalf("edited sender role/name comparison = %q, want %q", got, SourceIntegrityChangedSinceAnalysis)
	}
}

func TestCompareSnapshotDetectsChangedAttachmentIdentity(t *testing.T) {
	conv, msgs := snapshotFixture()
	withAttachment := append([]models.Message(nil), msgs...)
	withAttachment[0].Attachments = `[{"type":"image","url":"https://x/a.png","name":"a.png"}]`
	snap := mustSnapshot(t, conv, withAttachment, 0)

	changed := append([]models.Message(nil), withAttachment...)
	changed[0].Attachments = `[{"type":"image","url":"https://x/b.png","name":"a.png"}]`

	got := CompareSnapshotToCurrentMessages(string(snap.ManifestJSON), changed)
	if got != SourceIntegrityChangedSinceAnalysis {
		t.Fatalf("changed attachment identity comparison = %q, want %q", got, SourceIntegrityChangedSinceAnalysis)
	}
}

func TestCompareSnapshotDetectsMissingMessage(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)

	remaining := []models.Message{msgs[0]} // drop one of the two seeded messages

	got := CompareSnapshotToCurrentMessages(string(snap.ManifestJSON), remaining)
	if got != SourceIntegrityChangedSinceAnalysis {
		t.Fatalf("missing-message comparison = %q, want %q", got, SourceIntegrityChangedSinceAnalysis)
	}
}

func TestCompareSnapshotDetectsAddedMessageAfterAnalyzedWindow(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)

	maxSentAt := msgs[0].SentAt // snapshotFixture's m-2 is the later message
	added := append([]models.Message(nil), msgs...)
	added = append(added, models.Message{
		ID: "m-3", TenantID: conv.TenantID, ConversationID: conv.ID, ExternalMessageID: "ext-3",
		SenderType: "customer", SenderName: "Khách", Content: "Còn ai đó không ạ?",
		ContentType: "text", SentAt: maxSentAt.Add(time.Minute),
	})

	got := CompareSnapshotToCurrentMessages(string(snap.ManifestJSON), added)
	if got != SourceIntegrityChangedSinceAnalysis {
		t.Fatalf("added-message comparison = %q, want %q", got, SourceIntegrityChangedSinceAnalysis)
	}
}

// TestCompareSnapshotWithMatchingOmittedCountStaysBoundCurrentnessUnverified
// documents the bounded-comparison limitation on purpose: when the current
// count of messages older than the manifest's window exactly matches the
// manifest's own declared OmittedEarlierMessages, individual earlier
// messages cannot be identified from the stored manifest alone, so this must
// not be misreported as a difference. R004-R2 repair: this replaces a prior
// version of this test that added only one older message against a
// recorded omitted count of three, which the corrected count-comparison
// logic below would now (correctly) flag as changed.
func TestCompareSnapshotWithMatchingOmittedCountStaysBoundCurrentnessUnverified(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 3) // 3 earlier messages were windowed out when the snapshot was built

	minSentAt := msgs[1].SentAt // snapshotFixture's m-1 is the earlier message
	withOlder := append([]models.Message(nil), msgs...)
	withOlder = append(withOlder,
		models.Message{ID: "m-old-1", TenantID: conv.TenantID, ConversationID: conv.ID, ExternalMessageID: "ext-old-1",
			SenderType: "customer", SenderName: "Khách", Content: "Tin nhắn rất cũ 1",
			ContentType: "text", SentAt: minSentAt.Add(-1 * time.Hour)},
		models.Message{ID: "m-old-2", TenantID: conv.TenantID, ConversationID: conv.ID, ExternalMessageID: "ext-old-2",
			SenderType: "customer", SenderName: "Khách", Content: "Tin nhắn rất cũ 2",
			ContentType: "text", SentAt: minSentAt.Add(-2 * time.Hour)},
		models.Message{ID: "m-old-3", TenantID: conv.TenantID, ConversationID: conv.ID, ExternalMessageID: "ext-old-3",
			SenderType: "customer", SenderName: "Khách", Content: "Tin nhắn rất cũ 3",
			ContentType: "text", SentAt: minSentAt.Add(-3 * time.Hour)},
	)

	got := CompareSnapshotToCurrentMessages(string(snap.ManifestJSON), withOlder)
	if got != SourceIntegrityBoundCurrentnessUnverified {
		t.Fatalf("matching-omitted-count comparison = %q, want %q (documented limitation)", got, SourceIntegrityBoundCurrentnessUnverified)
	}
}

// TestCompareSnapshotDetectsAddedEarlierMessageWithZeroOmittedCount is the
// R004-R2 central case: the snapshot recorded zero omitted earlier history,
// so any current message older than the manifest's window is a proven
// change (it did not exist, or was not omitted, when the snapshot was
// built) — it must not be silently ignored.
func TestCompareSnapshotDetectsAddedEarlierMessageWithZeroOmittedCount(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0) // no earlier history was omitted

	minSentAt := msgs[1].SentAt // snapshotFixture's m-1 is the earlier message
	withOlder := append([]models.Message(nil), msgs...)
	withOlder = append(withOlder, models.Message{
		ID: "m-0", TenantID: conv.TenantID, ConversationID: conv.ID, ExternalMessageID: "ext-0",
		SenderType: "customer", SenderName: "Khách", Content: "Tin nhắn cũ mới xuất hiện",
		ContentType: "text", SentAt: minSentAt.Add(-time.Hour),
	})

	got := CompareSnapshotToCurrentMessages(string(snap.ManifestJSON), withOlder)
	if got != SourceIntegrityChangedSinceAnalysis {
		t.Fatalf("added-earlier-message-with-zero-omitted comparison = %q, want %q", got, SourceIntegrityChangedSinceAnalysis)
	}
}

// TestCompareSnapshotDetectsEarlierMessageCountChangeWithNonzeroOmittedCount
// is the R004-R2 windowed case: the manifest recorded a nonzero omitted
// count, but the current earlier-message count no longer matches it — a
// proven count change, not merely an unidentifiable individual message.
func TestCompareSnapshotDetectsEarlierMessageCountChangeWithNonzeroOmittedCount(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 3) // 3 earlier messages were windowed out when the snapshot was built

	minSentAt := msgs[1].SentAt
	withOnlyOneOlder := append([]models.Message(nil), msgs...)
	withOnlyOneOlder = append(withOnlyOneOlder, models.Message{
		ID: "m-old-only", TenantID: conv.TenantID, ConversationID: conv.ID, ExternalMessageID: "ext-old-only",
		SenderType: "customer", SenderName: "Khách", Content: "Chỉ còn 1 tin nhắn cũ",
		ContentType: "text", SentAt: minSentAt.Add(-time.Hour),
	})

	got := CompareSnapshotToCurrentMessages(string(snap.ManifestJSON), withOnlyOneOlder)
	if got != SourceIntegrityChangedSinceAnalysis {
		t.Fatalf("earlier-count-mismatch comparison = %q, want %q", got, SourceIntegrityChangedSinceAnalysis)
	}
}

func TestCompareSnapshotMalformedManifestIsVerificationUnavailable(t *testing.T) {
	_, msgs := snapshotFixture()
	got := CompareSnapshotToCurrentMessages("not json", msgs)
	if got != SourceIntegrityVerificationUnavailable {
		t.Fatalf("malformed manifest comparison = %q, want %q", got, SourceIntegrityVerificationUnavailable)
	}
}

func TestCompareSnapshotUnsupportedSchemaIsVerificationUnavailable(t *testing.T) {
	_, msgs := snapshotFixture()
	got := CompareSnapshotToCurrentMessages(`{"schema_version":"ccma.snapshot.v0","messages":[]}`, msgs)
	if got != SourceIntegrityVerificationUnavailable {
		t.Fatalf("unsupported schema comparison = %q, want %q", got, SourceIntegrityVerificationUnavailable)
	}
}

// --- R004-R1: VerifySnapshotProvenance ---
//
// mustSnapshotRecord builds the exact models.AnalysisSnapshot the production
// (*conversationSnapshot).record path produces, so these tests validate
// VerifySnapshotProvenance against a genuinely valid row before corrupting
// one field at a time.
func mustSnapshotRecord(t *testing.T, snap *conversationSnapshot, runID string) models.AnalysisSnapshot {
	t.Helper()
	record, err := snap.record(runID)
	if err != nil {
		t.Fatalf("record snapshot: %v", err)
	}
	return record
}

func TestVerifySnapshotProvenanceAcceptsValidSnapshot(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)
	record := mustSnapshotRecord(t, snap, "run-1")

	if !VerifySnapshotProvenance(record, conv.TenantID, conv.ID, "run-1") {
		t.Fatal("valid snapshot was rejected")
	}
}

func TestVerifySnapshotProvenanceRejectsCorruptDigest(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)
	record := mustSnapshotRecord(t, snap, "run-1")
	record.Digest = "deadbeef"

	if VerifySnapshotProvenance(record, conv.TenantID, conv.ID, "run-1") {
		t.Fatal("snapshot with a digest that does not match its manifest bytes was accepted")
	}
}

func TestVerifySnapshotProvenanceRejectsWrongConversationLink(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)
	record := mustSnapshotRecord(t, snap, "run-1")

	if VerifySnapshotProvenance(record, conv.TenantID, "conv-other", "run-1") {
		t.Fatal("snapshot linked to the wrong conversation was accepted")
	}
}

func TestVerifySnapshotProvenanceRejectsWrongJobRunLink(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)
	record := mustSnapshotRecord(t, snap, "run-1")

	if VerifySnapshotProvenance(record, conv.TenantID, conv.ID, "run-other") {
		t.Fatal("snapshot linked to the wrong job run was accepted")
	}
}

// TestVerifySnapshotProvenanceRejectsCrossTenantLink is the R004-R1
// cross-tenant regression: a snapshot recorded for one tenant must never
// verify as belonging to another tenant's result, even independently of the
// tenant-scoped snapshot lookup query that is the first line of defense.
func TestVerifySnapshotProvenanceRejectsCrossTenantLink(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)
	record := mustSnapshotRecord(t, snap, "run-1")

	if VerifySnapshotProvenance(record, "tenant-other", conv.ID, "run-1") {
		t.Fatal("snapshot linked across tenants was accepted")
	}
}

func TestVerifySnapshotProvenanceRejectsMessageCountMismatch(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)
	record := mustSnapshotRecord(t, snap, "run-1")
	record.MessageCount++

	if VerifySnapshotProvenance(record, conv.TenantID, conv.ID, "run-1") {
		t.Fatal("snapshot with a message count inconsistent with its manifest was accepted")
	}
}

func TestVerifySnapshotProvenanceRejectsUnsupportedSchema(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 0)
	record := mustSnapshotRecord(t, snap, "run-1")
	record.SchemaVersion = "ccma.snapshot.v0"

	if VerifySnapshotProvenance(record, conv.TenantID, conv.ID, "run-1") {
		t.Fatal("snapshot with an unsupported schema version was accepted")
	}
}
