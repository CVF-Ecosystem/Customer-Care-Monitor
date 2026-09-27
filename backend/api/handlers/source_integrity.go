package handlers

import (
	"fmt"
	"strings"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/engine"
)

// sourceIntegrityBatchSize bounds each IN (...) list so a large job export
// issues a few chunked queries rather than one query per result or one
// unbounded parameter list.
const sourceIntegrityBatchSize = 500

// sourceIntegrityRef is the minimal identity of one result row needed to
// verify its linked snapshot (CCMAI-RUNTIME-004 semantics).
type sourceIntegrityRef struct {
	ConversationID     string
	JobRunID           string
	AnalysisSnapshotID *string
}

// computeSourceIntegrity returns one engine.SourceIntegrity* status per ref,
// in order. It is read-only and tenant-scoped: linked snapshots and their
// conversations' current messages are loaded in chunked batches, never one
// query per result. A row-local problem (no link -> legacy_unverified; missing,
// cross-tenant, mislinked or corrupt snapshot -> verification_unavailable)
// never reads as unchanged, while any batch query failure is returned so the
// caller can fail the whole request before writing a body.
func computeSourceIntegrity(tenantID string, refs []sourceIntegrityRef) ([]string, error) {
	var snapshotIDs, convIDs []string
	seenSnapshot, seenConv := map[string]bool{}, map[string]bool{}
	for _, r := range refs {
		if r.AnalysisSnapshotID == nil || *r.AnalysisSnapshotID == "" {
			continue
		}
		if !seenSnapshot[*r.AnalysisSnapshotID] {
			seenSnapshot[*r.AnalysisSnapshotID] = true
			snapshotIDs = append(snapshotIDs, *r.AnalysisSnapshotID)
		}
		if !seenConv[r.ConversationID] {
			seenConv[r.ConversationID] = true
			convIDs = append(convIDs, r.ConversationID)
		}
	}

	snapshotByID := map[string]models.AnalysisSnapshot{}
	for _, chunk := range chunkStrings(snapshotIDs, sourceIntegrityBatchSize) {
		var snaps []models.AnalysisSnapshot
		if err := db.DB.Where("tenant_id = ? AND id IN ?", tenantID, chunk).Find(&snaps).Error; err != nil {
			return nil, fmt.Errorf("truy vấn snapshot: %w", err)
		}
		for _, s := range snaps {
			snapshotByID[s.ID] = s
		}
	}

	messagesByConv := map[string][]models.Message{}
	for _, chunk := range chunkStrings(convIDs, sourceIntegrityBatchSize) {
		var msgs []models.Message
		if err := db.DB.Where("tenant_id = ? AND conversation_id IN ?", tenantID, chunk).Find(&msgs).Error; err != nil {
			return nil, fmt.Errorf("truy vấn tin nhắn: %w", err)
		}
		for _, m := range msgs {
			messagesByConv[m.ConversationID] = append(messagesByConv[m.ConversationID], m)
		}
	}

	statuses := make([]string, len(refs))
	for i, r := range refs {
		if r.AnalysisSnapshotID == nil || *r.AnalysisSnapshotID == "" {
			statuses[i] = engine.SourceIntegrityLegacyUnverified
			continue
		}
		snap, ok := snapshotByID[*r.AnalysisSnapshotID]
		if !ok || !engine.VerifySnapshotProvenance(snap, tenantID, r.ConversationID, r.JobRunID) {
			statuses[i] = engine.SourceIntegrityVerificationUnavailable
			continue
		}
		statuses[i] = engine.CompareSnapshotToCurrentMessages(snap.Manifest, messagesByConv[r.ConversationID])
	}
	return statuses, nil
}

func chunkStrings(items []string, size int) [][]string {
	var chunks [][]string
	for start := 0; start < len(items); start += size {
		end := start + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[start:end])
	}
	return chunks
}

// sourceIntegrityDisplayOrder puts the most concerning status first so a
// grouped export cell or badge row can never lead with a reassuring value.
var sourceIntegrityDisplayOrder = []string{
	engine.SourceIntegrityChangedSinceAnalysis,
	engine.SourceIntegrityVerificationUnavailable,
	engine.SourceIntegrityLegacyUnverified,
	engine.SourceIntegrityBoundCurrentnessUnverified,
}

// distinctSourceIntegrityLabels renders every distinct status in a group, in
// sourceIntegrityDisplayOrder, as Vietnamese export labels joined by "; ".
func distinctSourceIntegrityLabels(statuses []string) string {
	present := map[string]bool{}
	for _, s := range statuses {
		present[s] = true
	}
	var labels []string
	for _, s := range sourceIntegrityDisplayOrder {
		if present[s] {
			labels = append(labels, sourceIntegrityLabel(s))
		}
	}
	return strings.Join(labels, "; ")
}
