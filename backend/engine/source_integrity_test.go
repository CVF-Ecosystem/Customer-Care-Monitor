package engine

import (
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
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

// TestCompareSnapshotIgnoresMessageOlderThanAnalyzedWindow documents the
// bounded-comparison limitation on purpose: a message timestamped before
// anything the snapshot recorded is presumed to belong to the manifest's own
// declared OmittedEarlierMessages bucket, not a new signal, so it must not
// be misreported as "added."
func TestCompareSnapshotIgnoresMessageOlderThanAnalyzedWindow(t *testing.T) {
	conv, msgs := snapshotFixture()
	snap := mustSnapshot(t, conv, msgs, 3) // pretend 3 earlier messages were windowed out

	minSentAt := msgs[1].SentAt // snapshotFixture's m-1 is the earlier message
	withOlder := append([]models.Message(nil), msgs...)
	withOlder = append(withOlder, models.Message{
		ID: "m-0", TenantID: conv.TenantID, ConversationID: conv.ID, ExternalMessageID: "ext-0",
		SenderType: "customer", SenderName: "Khách", Content: "Tin nhắn rất cũ",
		ContentType: "text", SentAt: minSentAt.Add(-time.Hour),
	})

	got := CompareSnapshotToCurrentMessages(string(snap.ManifestJSON), withOlder)
	if got != SourceIntegrityBoundCurrentnessUnverified {
		t.Fatalf("older-than-window comparison = %q, want %q (documented limitation)", got, SourceIntegrityBoundCurrentnessUnverified)
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
