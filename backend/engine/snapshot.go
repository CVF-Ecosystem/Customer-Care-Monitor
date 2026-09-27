package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/pkg"
)

const snapshotSchemaV1 = "ccma.snapshot.v1"

const (
	coverageComplete = "complete"
	coveragePartial  = "partial"
	coverageEmpty    = "empty"

	reasonNoMessages               = "NO_MESSAGES"
	reasonHistoryWindowed          = "HISTORY_WINDOWED"
	reasonUnsupportedContentType   = "UNSUPPORTED_CONTENT_TYPE"
	reasonAttachmentNotRepresented = "ATTACHMENT_NOT_REPRESENTED"
	reasonAttachmentJSONInvalid    = "ATTACHMENT_JSON_INVALID"

	attachmentNone           = "none"
	attachmentNotRepresented = "not_represented"
	attachmentInvalidJSON    = "invalid_json"
)

type snapshotMessage struct {
	MessageID             string `json:"message_id"`
	ExternalMessageID     string `json:"external_message_id"`
	SenderType            string `json:"sender_type"`
	SenderName            string `json:"sender_name"`
	ContentType           string `json:"content_type"`
	SentAt                string `json:"sent_at"`
	ContentSHA256         string `json:"content_sha256"`
	ContentCodePoints     int    `json:"content_code_points"`
	AttachmentCoverage    string `json:"attachment_coverage"`
	AttachmentCount       int    `json:"attachment_count"`
	AttachmentFingerprint string `json:"attachment_fingerprint,omitempty"`
}

// snapshotManifest is the canonical, digest-bearing record. Field order is
// fixed by the struct and slices are sorted, so encoding is deterministic.
type snapshotManifest struct {
	SchemaVersion          string            `json:"schema_version"`
	TenantID               string            `json:"tenant_id"`
	ConversationID         string            `json:"conversation_id"`
	Coverage               string            `json:"coverage"`
	CoverageReasons        []string          `json:"coverage_reasons"`
	OmittedEarlierMessages int64             `json:"omitted_earlier_messages"`
	Messages               []snapshotMessage `json:"messages"`
}

type conversationSnapshot struct {
	Manifest     snapshotManifest
	ManifestJSON []byte
	Digest       string
	Transcript   string
	contents     map[string]string
}

// loadConversationSnapshot is the only path single and batch analysis use to
// read a conversation. Query errors are returned, never treated as empty.
func loadConversationSnapshot(conv models.Conversation, since time.Time) (*conversationSnapshot, error) {
	var messages []models.Message
	q := db.DB.Where("conversation_id = ? AND tenant_id = ?", conv.ID, conv.TenantID)
	if !since.IsZero() {
		q = q.Where("sent_at > ?", since)
	}
	if err := q.Order("sent_at ASC, id ASC").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("load messages for snapshot: %w", err)
	}
	var omitted int64
	if !since.IsZero() {
		if err := db.DB.Model(&models.Message{}).
			Where("conversation_id = ? AND tenant_id = ? AND sent_at <= ?", conv.ID, conv.TenantID, since).
			Count(&omitted).Error; err != nil {
			return nil, fmt.Errorf("count earlier messages for snapshot: %w", err)
		}
	}
	return buildConversationSnapshot(conv, messages, omitted)
}

func buildConversationSnapshot(conv models.Conversation, messages []models.Message, omittedEarlier int64) (*conversationSnapshot, error) {
	sorted := append([]models.Message(nil), messages...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if !sorted[i].SentAt.Equal(sorted[j].SentAt) {
			return sorted[i].SentAt.Before(sorted[j].SentAt)
		}
		return sorted[i].ID < sorted[j].ID
	})

	reasons := map[string]bool{}
	if omittedEarlier > 0 {
		reasons[reasonHistoryWindowed] = true
	}
	manifest := snapshotManifest{
		SchemaVersion:          snapshotSchemaV1,
		TenantID:               conv.TenantID,
		ConversationID:         conv.ID,
		OmittedEarlierMessages: omittedEarlier,
		Messages:               make([]snapshotMessage, 0, len(sorted)),
	}
	chat := make([]ai.ChatMessage, 0, len(sorted))
	contents := make(map[string]string, len(sorted))

	for _, m := range sorted {
		if m.ConversationID != conv.ID || m.TenantID != conv.TenantID {
			return nil, fmt.Errorf("message %s does not belong to conversation %s", m.ID, conv.ID)
		}
		sm := canonicalizeMessage(m)
		if sm.ContentType != "text" {
			reasons[reasonUnsupportedContentType] = true
		}
		switch sm.AttachmentCoverage {
		case attachmentNotRepresented:
			reasons[reasonAttachmentNotRepresented] = true
		case attachmentInvalidJSON:
			reasons[reasonAttachmentJSONInvalid] = true
		}
		manifest.Messages = append(manifest.Messages, sm)
		chat = append(chat, ai.ChatMessage{
			MessageID:  m.ID,
			SenderType: m.SenderType,
			SenderName: m.SenderName,
			Content:    m.Content,
			SentAt:     pkg.ToVN(m.SentAt).Format(time.RFC3339),
		})
		contents[m.ID] = m.Content
	}

	switch {
	case len(sorted) == 0:
		manifest.Coverage = coverageEmpty
		reasons[reasonNoMessages] = true
	case len(reasons) > 0:
		manifest.Coverage = coveragePartial
	default:
		manifest.Coverage = coverageComplete
	}
	manifest.CoverageReasons = make([]string, 0, len(reasons))
	for r := range reasons {
		manifest.CoverageReasons = append(manifest.CoverageReasons, r)
	}
	sort.Strings(manifest.CoverageReasons)

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("encode snapshot manifest: %w", err)
	}
	digest := sha256.Sum256(manifestJSON)
	return &conversationSnapshot{
		Manifest:     manifest,
		ManifestJSON: manifestJSON,
		Digest:       hex.EncodeToString(digest[:]),
		Transcript:   ai.FormatChatTranscript(chat),
		contents:     contents,
	}, nil
}

// classifyAttachments returns coverage, count and a deterministic fingerprint
// of the attachment metadata (not the raw payload). Valid JSON is fingerprinted
// by re-encoding the typed []channels.Attachment slice with json.Marshal: fixed
// struct field order plus JSON's own escaping of every field's content (quotes,
// colons, commas, control characters) makes the encoding unambiguous, so a
// same-count, same-coverage swap to a different attachment still changes the
// fingerprint. A hand-joined delimiter string does not have that guarantee — two
// different attachments can produce the same joined bytes when a field value
// contains the delimiter itself, which is exactly the collision this replaces.
// Invalid JSON is fingerprinted from the untrimmed raw bytes, so any source-byte
// change — including a change to only leading/trailing whitespace — is never
// silently absorbed into "no change".
func classifyAttachments(raw string) (coverage string, count int, fingerprint string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" || trimmed == "[]" {
		return attachmentNone, 0, ""
	}
	var atts []channels.Attachment
	if err := json.Unmarshal([]byte(trimmed), &atts); err != nil {
		sum := sha256.Sum256([]byte(raw))
		return attachmentInvalidJSON, 0, hex.EncodeToString(sum[:])
	}
	if len(atts) == 0 {
		return attachmentNone, 0, ""
	}
	canonical, err := json.Marshal(atts)
	if err != nil {
		sum := sha256.Sum256([]byte(raw))
		return attachmentNotRepresented, len(atts), hex.EncodeToString(sum[:])
	}
	sum := sha256.Sum256(canonical)
	return attachmentNotRepresented, len(atts), hex.EncodeToString(sum[:])
}

// canonicalizeMessage is the single source of truth for a message's
// digest-relevant facts: the exact fields buildConversationSnapshot puts in
// the manifest, and the exact fields CompareSnapshotToCurrentMessages
// compares a stored manifest against. Any caller wanting to know whether a
// message "looks the same" as when it was snapshotted must go through this
// function rather than re-deriving its own notion of sameness.
func canonicalizeMessage(m models.Message) snapshotMessage {
	contentType := m.ContentType
	if contentType == "" {
		contentType = "text"
	}
	attCoverage, attCount, attFingerprint := classifyAttachments(m.Attachments)
	sum := sha256.Sum256([]byte(m.Content))
	return snapshotMessage{
		MessageID:             m.ID,
		ExternalMessageID:     m.ExternalMessageID,
		SenderType:            m.SenderType,
		SenderName:            m.SenderName,
		ContentType:           contentType,
		SentAt:                m.SentAt.UTC().Format(time.RFC3339Nano),
		ContentSHA256:         hex.EncodeToString(sum[:]),
		ContentCodePoints:     utf8.RuneCountInString(m.Content),
		AttachmentCoverage:    attCoverage,
		AttachmentCount:       attCount,
		AttachmentFingerprint: attFingerprint,
	}
}

// Source-integrity status values for CCMAI-RUNTIME-004's read-time
// comparison between a stored analysis snapshot and current source
// messages. These are local, informational signals only — see
// CompareSnapshotToCurrentMessages for exactly what they do and don't prove.
const (
	// SourceIntegrityChangedSinceAnalysis means the local comparison found a
	// concrete difference: an edited message, a missing (deleted) message,
	// or a message the snapshot never saw.
	SourceIntegrityChangedSinceAnalysis = "changed_since_analysis"
	// SourceIntegrityBoundCurrentnessUnverified means the local comparison
	// found no difference, but that is not proof of full upstream
	// currentness — adapter history/removal coverage and timing may be
	// incomplete (see docs/specs/RUNTIME_RESULT_SOURCE_FRESHNESS_S1_2026-09-27.md).
	SourceIntegrityBoundCurrentnessUnverified = "bound_currentness_unverified"
	// SourceIntegrityLegacyUnverified means the result has no snapshot link
	// at all, so no comparison could be attempted.
	SourceIntegrityLegacyUnverified = "legacy_unverified"
	// SourceIntegrityVerificationUnavailable means a snapshot link exists but
	// the comparison itself could not be trusted: a broken link, a
	// malformed/unsupported manifest, or a row-local read error. This must
	// never collapse into "unchanged."
	SourceIntegrityVerificationUnavailable = "verification_unavailable"
)

// CompareSnapshotToCurrentMessages compares a stored ccma.snapshot.v1
// manifest (as persisted in models.AnalysisSnapshot.Manifest) against the
// conversation's current tenant-scoped messages, using exactly the same
// per-message canonicalization canonicalizeMessage uses to build that
// manifest in the first place — this is deliberate reuse, not a parallel
// notion of "changed."
//
// The comparison window is bounded to current messages at or after the
// earliest SentAt recorded in the manifest. The manifest's own
// OmittedEarlierMessages already documents that earlier history may have
// been excluded by design when the snapshot was built; without that original
// cutoff recorded verbatim, treating every current message as in-scope would
// misreport an already-known, never-analyzed older message as "added." A
// message re-inserted with a timestamp older than anything the snapshot
// recorded is therefore out of scope for this comparison — a documented
// limitation, not a claim that such a change would be detected.
//
// This function is read-only and never mutates the snapshot, any result, or
// any message. An unchanged comparison never returns more than
// SourceIntegrityBoundCurrentnessUnverified: a local comparison cannot prove
// the adapter's upstream history and removal coverage are complete.
func CompareSnapshotToCurrentMessages(manifestJSON string, currentMessages []models.Message) string {
	var manifest snapshotManifest
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		return SourceIntegrityVerificationUnavailable
	}
	if manifest.SchemaVersion != snapshotSchemaV1 {
		return SourceIntegrityVerificationUnavailable
	}

	manifestByID := make(map[string]snapshotMessage, len(manifest.Messages))
	var minSentAt time.Time
	for i, sm := range manifest.Messages {
		manifestByID[sm.MessageID] = sm
		t, err := time.Parse(time.RFC3339Nano, sm.SentAt)
		if err != nil {
			return SourceIntegrityVerificationUnavailable
		}
		if i == 0 || t.Before(minSentAt) {
			minSentAt = t
		}
	}

	currentByID := make(map[string]snapshotMessage, len(currentMessages))
	for _, m := range currentMessages {
		if len(manifest.Messages) > 0 && m.SentAt.Before(minSentAt) {
			continue // outside the manifest's own declared window; see doc comment
		}
		currentByID[m.ID] = canonicalizeMessage(m)
	}

	for id, sm := range manifestByID {
		cur, ok := currentByID[id]
		if !ok || cur != sm {
			return SourceIntegrityChangedSinceAnalysis
		}
	}
	for id := range currentByID {
		if _, ok := manifestByID[id]; !ok {
			return SourceIntegrityChangedSinceAnalysis
		}
	}
	return SourceIntegrityBoundCurrentnessUnverified
}

func (s *conversationSnapshot) record(runID string) (models.AnalysisSnapshot, error) {
	reasonsJSON, err := json.Marshal(s.Manifest.CoverageReasons)
	if err != nil {
		return models.AnalysisSnapshot{}, err
	}
	return models.AnalysisSnapshot{
		ID:              pkg.NewUUID(),
		TenantID:        s.Manifest.TenantID,
		JobRunID:        runID,
		ConversationID:  s.Manifest.ConversationID,
		SchemaVersion:   s.Manifest.SchemaVersion,
		Digest:          s.Digest,
		Coverage:        s.Manifest.Coverage,
		CoverageReasons: string(reasonsJSON),
		MessageCount:    len(s.Manifest.Messages),
		Manifest:        string(s.ManifestJSON),
		CreatedAt:       time.Now(),
	}, nil
}

// evidenceRef binds a finding to an exact span of one source message.
// Offsets are Unicode code points, start inclusive, end exclusive.
type evidenceRef struct {
	MessageID string `json:"message_id"`
	Quote     string `json:"quote"`
	Start     *int   `json:"start,omitempty"`
	End       *int   `json:"end,omitempty"`
}

func (s *conversationSnapshot) validateEvidenceRefs(refs []evidenceRef) error {
	if len(refs) == 0 {
		return fmt.Errorf("finding has no evidence_refs")
	}
	for i, ref := range refs {
		content, ok := s.contents[ref.MessageID]
		if !ok {
			return fmt.Errorf("evidence_refs[%d]: message %q is not in the snapshot of conversation %s", i, ref.MessageID, s.Manifest.ConversationID)
		}
		if ref.Quote == "" || !strings.Contains(content, ref.Quote) {
			return fmt.Errorf("evidence_refs[%d]: quote is not an exact substring of message %s", i, ref.MessageID)
		}
		if ref.Start == nil && ref.End == nil {
			continue
		}
		if ref.Start == nil || ref.End == nil {
			return fmt.Errorf("evidence_refs[%d]: start and end must be given together", i)
		}
		runes := []rune(content)
		start, end := *ref.Start, *ref.End
		if start < 0 || end <= start || end > len(runes) || string(runes[start:end]) != ref.Quote {
			return fmt.Errorf("evidence_refs[%d]: offsets [%d,%d) do not match quote", i, start, end)
		}
	}
	return nil
}
