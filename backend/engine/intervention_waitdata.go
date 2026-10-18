package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var ErrWaitDataInvalid = errors.New("invalid_request")
var ErrWaitDataForbidden = errors.New("permission_denied")
var ErrWaitDataMissing = errors.New("not_found")
var ErrWaitDataConflict = errors.New("case_conflict")
var ErrWaitDataUnchanged = errors.New("source_unchanged")
var ErrWaitDataKeyConflict = errors.New("request_key_conflict")
var ErrWaitDataPending = errors.New("request_pending")
var ErrWaitDataUnavailable = errors.New("source_unavailable")

const waitDataMaxMessages = 1000
const waitDataMaxContentBytes = 4 << 20
const waitDataMaxManifestBytes = 1 << 20

var waitDataKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]{1,128}$`)
var waitDataUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type DataRevisionInput struct {
	ExpectedVersion        int    `json:"expected_version"`
	ExpectedSnapshotID     string `json:"expected_snapshot_id"`
	ExpectedSnapshotDigest string `json:"expected_snapshot_digest"`
	Reason                 string `json:"reason"`
}

type RerunRequestInput struct {
	ExpectedVersion        int    `json:"expected_version"`
	ExpectedSnapshotID     string `json:"expected_snapshot_id"`
	ExpectedSnapshotDigest string `json:"expected_snapshot_digest"`
	DataRevisionID         string `json:"data_revision_id"`
	DataRevisionDigest     string `json:"data_revision_digest"`
	Reason                 string `json:"reason"`
	RequestKey             string `json:"request_key"`
}

func validWaitDataCommon(version int, snapshotID, snapshotDigest, reason string) bool {
	_, digestErr := hex.DecodeString(snapshotDigest)
	return version > 0 && waitDataUUIDPattern.MatchString(snapshotID) && len(snapshotDigest) == 64 && digestErr == nil && boundedM3(reason, 500)
}

// HashRerunPayload covers every submitted semantic field. Actor, tenant and
// creation time are server-owned; the key itself is scoped by a unique index.
func HashRerunPayload(in RerunRequestInput) string {
	b, _ := json.Marshal(in)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func waitDataJobPermitted(actor models.UserTenant) bool {
	if !m3Permitted(actor) {
		return false
	}
	if actor.Role == "owner" || actor.Role == "admin" {
		return true
	}
	var p map[string]string
	return json.Unmarshal([]byte(actor.Permissions), &p) == nil && strings.Contains(p["jobs"], "w")
}

// A retry proves only that the exact prior intent exists. It does not assert
// that the source, job or case is still eligible for a new request.
func replayRerunRequest(ctx context.Context, tenantID, actorID, caseID, key, hash string) (*models.InterventionRerunRequest, error) {
	var found *models.InterventionRerunRequest
	err := waitDataDB(ctx).Transaction(func(tx *gorm.DB) error {
		var actor models.UserTenant
		actorErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND user_id = ?", tenantID, actorID).Take(&actor).Error
		if actorErr != nil && !errors.Is(actorErr, gorm.ErrRecordNotFound) {
			return ErrWaitDataUnavailable
		}
		if actorErr != nil || !waitDataJobPermitted(actor) {
			return ErrWaitDataForbidden
		}
		var c models.InterventionCase
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, caseID).Take(&c).Error != nil {
			return ErrWaitDataMissing
		}
		var prior models.InterventionRerunRequest
		q := tx.Where("tenant_id = ? AND case_id = ? AND request_key = ?", tenantID, caseID, key).Take(&prior)
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if q.Error != nil {
			return ErrWaitDataUnavailable
		}
		if prior.PayloadHash != hash {
			return ErrWaitDataKeyConflict
		}
		found = &prior
		return nil
	})
	if err == nil || errors.Is(err, ErrWaitDataForbidden) || errors.Is(err, ErrWaitDataMissing) || errors.Is(err, ErrWaitDataKeyConflict) || errors.Is(err, ErrWaitDataUnavailable) {
		return found, err
	}
	return nil, ErrWaitDataUnavailable
}

type waitDataLocked struct {
	tx       *gorm.DB
	caseRow  models.InterventionCase
	snapshot models.AnalysisSnapshot
	conv     models.Conversation
	messages []models.Message
}

// A per-call silent session protects reasons and manifests from SQL logging
// without changing the application's global logger.
func waitDataDB(ctx context.Context) *gorm.DB {
	return db.DB.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
}

// VerifyInterventionDataRevision checks the exact persisted manifest bytes as
// well as its denormalized observation fields before any later use.
func VerifyInterventionDataRevision(r models.InterventionDataRevision) bool {
	sum := sha256.Sum256([]byte(r.Manifest))
	if hex.EncodeToString(sum[:]) != r.Digest {
		return false
	}
	var m snapshotManifest
	if json.Unmarshal([]byte(r.Manifest), &m) != nil || m.SchemaVersion != snapshotSchemaV1 || m.TenantID != r.TenantID || m.ConversationID != r.ConversationID || m.Coverage != r.Coverage || len(m.Messages) != r.MessageCount {
		return false
	}
	var reasons []string
	if json.Unmarshal([]byte(r.CoverageReasons), &reasons) != nil || len(reasons) != len(m.CoverageReasons) {
		return false
	}
	for i := range reasons {
		if reasons[i] != m.CoverageReasons[i] {
			return false
		}
	}
	return true
}

// Parent reads are lookup hints only. All identities and current permission
// are rechecked in this transaction. The order follows supported channel
// writers: membership, job, channel, conversation, messages, case, snapshot.
func withWaitDataLocked(ctx context.Context, tenantID, actorID, caseID string, needsJobs bool, fn func(*waitDataLocked) error) error {
	var hint models.InterventionCase
	if waitDataDB(ctx).Where("id = ? AND tenant_id = ?", caseID, tenantID).Take(&hint).Error != nil {
		return ErrWaitDataMissing
	}
	var convHint models.Conversation
	if waitDataDB(ctx).Where("id = ? AND tenant_id = ?", hint.ConversationID, tenantID).Take(&convHint).Error != nil {
		return ErrWaitDataConflict
	}
	err := waitDataDB(ctx).Transaction(func(tx *gorm.DB) error {
		var actor models.UserTenant
		actorErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND user_id = ?", tenantID, actorID).Take(&actor).Error
		if actorErr != nil && !errors.Is(actorErr, gorm.ErrRecordNotFound) {
			return ErrWaitDataUnavailable
		}
		if actorErr != nil || !m3Permitted(actor) || (needsJobs && !waitDataJobPermitted(actor)) {
			return ErrWaitDataForbidden
		}
		var job models.Job
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", hint.JobID, tenantID).Take(&job).Error != nil {
			return ErrWaitDataConflict
		}
		var channel models.Channel
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", convHint.ChannelID, tenantID).Take(&channel).Error != nil {
			return ErrWaitDataConflict
		}
		var conv models.Conversation
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ? AND channel_id = ?", hint.ConversationID, tenantID, channel.ID).Take(&conv).Error != nil {
			return ErrWaitDataConflict
		}
		var messages []models.Message
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND conversation_id = ?", tenantID, conv.ID).Order("sent_at ASC, id ASC").Limit(waitDataMaxMessages+1).Find(&messages).Error != nil {
			return ErrWaitDataUnavailable
		}
		if len(messages) > waitDataMaxMessages {
			return ErrWaitDataUnavailable
		}
		var current models.InterventionCase
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", caseID, tenantID).Take(&current).Error != nil {
			return ErrWaitDataMissing
		}
		if current.JobID != job.ID || current.ConversationID != conv.ID || current.UseCase != "pancake_complaint_wait" || job.JobType != "intervention" || job.ScheduleType != "manual" || !job.IsActive || channel.ChannelType != "pancake" || !channel.IsDemoFixture {
			return ErrWaitDataConflict
		}
		var inputs []string
		if json.Unmarshal([]byte(job.InputChannelIDs), &inputs) != nil {
			return ErrWaitDataUnavailable
		}
		channelAllowed := false
		for _, id := range inputs {
			if id == channel.ID {
				channelAllowed = true
			}
		}
		if !channelAllowed || current.SnapshotID == nil || current.WorkState != "WAITING_ON_DATA" || current.Eligibility != "WAIT_DATA" || current.Execution != "NOT_RUN" || current.HumanAssessment != "" || current.OutcomeText != "" || current.WorkState == "RESOLVED" || current.WorkState == "REJECTED" {
			return ErrWaitDataConflict
		}
		var snap models.AnalysisSnapshot
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", *current.SnapshotID, tenantID).Take(&snap).Error != nil {
			return ErrWaitDataConflict
		}
		if snap.Digest != current.SnapshotDigest || !VerifySnapshotProvenance(snap, tenantID, conv.ID, current.RunID) {
			return ErrWaitDataConflict
		}
		return fn(&waitDataLocked{tx: tx, caseRow: current, snapshot: snap, conv: conv, messages: messages})
	})
	if err == nil || errors.Is(err, ErrWaitDataMissing) || errors.Is(err, ErrWaitDataForbidden) || errors.Is(err, ErrWaitDataInvalid) || errors.Is(err, ErrWaitDataConflict) || errors.Is(err, ErrWaitDataUnchanged) || errors.Is(err, ErrWaitDataKeyConflict) || errors.Is(err, ErrWaitDataPending) || errors.Is(err, ErrWaitDataUnavailable) {
		return err
	}
	return ErrWaitDataUnavailable
}

func (l *waitDataLocked) observation() (*conversationSnapshot, error) {
	bytes := 0
	for _, m := range l.messages {
		bytes += len(m.Content)
		if bytes > waitDataMaxContentBytes {
			return nil, ErrWaitDataUnavailable
		}
	}
	s, err := buildConversationSnapshot(l.conv, l.messages, 0)
	if err != nil || len(s.ManifestJSON) > waitDataMaxManifestBytes {
		return nil, ErrWaitDataUnavailable
	}
	return s, nil
}

func (l *waitDataLocked) matches(version int, snapshotID, digest string) bool {
	return l.caseRow.Version == version && l.snapshot.ID == snapshotID && l.snapshot.Digest == digest
}

func (l *waitDataLocked) appendMutation(actorID, action, reason, sourceStatus string, revisionID, requestID *string, at time.Time) error {
	before, err := json.Marshal(m3Projection(l.caseRow))
	if err != nil {
		return ErrWaitDataUnavailable
	}
	current := l.caseRow
	current.Version++
	current.UpdatedAt = at
	changed := l.tx.Model(&models.InterventionCase{}).Where("id = ? AND tenant_id = ? AND version = ?", current.ID, current.TenantID, l.caseRow.Version).Updates(map[string]any{"version": current.Version, "updated_at": at})
	if changed.Error != nil || changed.RowsAffected != 1 {
		return ErrWaitDataConflict
	}
	after, err := json.Marshal(m3Projection(current))
	if err != nil {
		return ErrWaitDataUnavailable
	}
	event := models.InterventionHumanEvent{ID: pkg.NewUUID(), TenantID: current.TenantID, CaseID: current.ID, CaseVersion: current.Version, ActorID: actorID, Action: action, Reason: strings.TrimSpace(reason), SnapshotID: l.snapshot.ID, SnapshotDigest: l.snapshot.Digest, DataRevisionID: revisionID, RerunRequestID: requestID, SourceStatus: sourceStatus, Before: string(before), After: string(after), CreatedAt: at}
	if l.tx.Create(&event).Error != nil {
		return ErrWaitDataUnavailable
	}
	l.caseRow = current
	return nil
}

func CaptureInterventionDataRevision(ctx context.Context, tenantID, actorID, caseID string, in DataRevisionInput) (*models.InterventionDataRevision, error) {
	if !validWaitDataCommon(in.ExpectedVersion, in.ExpectedSnapshotID, in.ExpectedSnapshotDigest, in.Reason) {
		return nil, ErrWaitDataInvalid
	}
	var result models.InterventionDataRevision
	err := withWaitDataLocked(ctx, tenantID, actorID, caseID, false, func(l *waitDataLocked) error {
		if !l.matches(in.ExpectedVersion, in.ExpectedSnapshotID, in.ExpectedSnapshotDigest) {
			return ErrWaitDataConflict
		}
		observed, err := l.observation()
		if err != nil {
			return err
		}
		var latest models.InterventionDataRevision
		q := l.tx.Where("tenant_id = ? AND case_id = ?", tenantID, caseID).Order("case_version DESC, id DESC").Take(&latest)
		if q.Error != nil && !errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return ErrWaitDataUnavailable
		}
		baseline := l.snapshot.Digest
		if q.Error == nil {
			if !VerifyInterventionDataRevision(latest) {
				return ErrWaitDataUnavailable
			}
			baseline = latest.Digest
		}
		if observed.Digest == baseline {
			return ErrWaitDataUnchanged
		}
		reasons, _ := json.Marshal(observed.Manifest.CoverageReasons)
		now := time.Now().UTC()
		status := "INCOMPLETE"
		if observed.Manifest.Coverage == coverageComplete {
			status = "LOCAL_MATCH_CURRENTNESS_UNVERIFIED"
		}
		result = models.InterventionDataRevision{ID: pkg.NewUUID(), TenantID: tenantID, CaseID: caseID, JobID: l.caseRow.JobID, ConversationID: l.conv.ID, ActorID: actorID, OriginatingCaseVersion: l.caseRow.Version, CaseVersion: l.caseRow.Version + 1, OriginalSnapshotID: l.snapshot.ID, OriginalSnapshotDigest: l.snapshot.Digest, Digest: observed.Digest, Manifest: string(observed.ManifestJSON), Coverage: observed.Manifest.Coverage, CoverageReasons: string(reasons), MessageCount: len(observed.Manifest.Messages), ObservationStatus: status, Reason: strings.TrimSpace(in.Reason), CreatedAt: now}
		if l.tx.Create(&result).Error != nil {
			return ErrWaitDataUnavailable
		}
		return l.appendMutation(actorID, "OBSERVE_DATA", in.Reason, result.ObservationStatus, &result.ID, nil, now)
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func RequestInterventionRerun(ctx context.Context, tenantID, actorID, caseID string, in RerunRequestInput) (*models.InterventionRerunRequest, bool, error) {
	_, revisionDigestErr := hex.DecodeString(in.DataRevisionDigest)
	if !validWaitDataCommon(in.ExpectedVersion, in.ExpectedSnapshotID, in.ExpectedSnapshotDigest, in.Reason) || !waitDataUUIDPattern.MatchString(in.DataRevisionID) || len(in.DataRevisionDigest) != 64 || revisionDigestErr != nil || !waitDataKeyPattern.MatchString(in.RequestKey) {
		return nil, false, ErrWaitDataInvalid
	}
	hash := HashRerunPayload(in)
	prior, replayErr := replayRerunRequest(ctx, tenantID, actorID, caseID, in.RequestKey, hash)
	if replayErr != nil {
		return nil, false, replayErr
	}
	if prior != nil {
		return prior, true, nil
	}
	var result models.InterventionRerunRequest
	replayed := false
	err := withWaitDataLocked(ctx, tenantID, actorID, caseID, true, func(l *waitDataLocked) error {
		// Authorize and scope the case first, then honor an identical key even
		// though its original expected version has since advanced.
		var prior models.InterventionRerunRequest
		q := l.tx.Where("tenant_id = ? AND case_id = ? AND request_key = ?", tenantID, caseID, in.RequestKey).Take(&prior)
		if q.Error == nil {
			if prior.PayloadHash != hash {
				return ErrWaitDataKeyConflict
			}
			result, replayed = prior, true
			return nil
		}
		if !errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return ErrWaitDataUnavailable
		}
		if !l.matches(in.ExpectedVersion, in.ExpectedSnapshotID, in.ExpectedSnapshotDigest) {
			return ErrWaitDataConflict
		}
		var selected models.InterventionDataRevision
		selectedQuery := l.tx.Where("tenant_id = ? AND case_id = ? AND id = ?", tenantID, caseID, in.DataRevisionID).Take(&selected)
		if errors.Is(selectedQuery.Error, gorm.ErrRecordNotFound) {
			return ErrWaitDataMissing
		}
		if selectedQuery.Error != nil {
			return ErrWaitDataUnavailable
		}
		if !VerifyInterventionDataRevision(selected) {
			return ErrWaitDataUnavailable
		}
		var latest models.InterventionDataRevision
		if l.tx.Where("tenant_id = ? AND case_id = ?", tenantID, caseID).Order("case_version DESC, id DESC").Take(&latest).Error != nil {
			return ErrWaitDataMissing
		}
		if !VerifyInterventionDataRevision(latest) {
			return ErrWaitDataUnavailable
		}
		if latest.ID != in.DataRevisionID || latest.Digest != in.DataRevisionDigest || latest.OriginalSnapshotID != l.snapshot.ID || latest.OriginalSnapshotDigest != l.snapshot.Digest || latest.Coverage != coverageComplete {
			return ErrWaitDataConflict
		}
		observed, err := l.observation()
		if err != nil {
			return err
		}
		if observed.Digest != latest.Digest || observed.Manifest.Coverage != coverageComplete {
			return ErrWaitDataConflict
		}
		anchor := false
		for _, m := range observed.Manifest.Messages {
			content := strings.ToLower(observed.contents[m.MessageID])
			if m.SenderType == "customer" && (strings.Contains(content, "chờ") || strings.Contains(content, "khiếu nại") || strings.Contains(content, "phàn nàn")) {
				anchor = true
			}
		}
		if !anchor || l.caseRow.Disposition == "PENDING_REVIEW" {
			return ErrWaitDataConflict
		}
		var n int64
		if l.tx.Model(&models.InterventionProposal{}).Where("tenant_id = ? AND case_id = ?", tenantID, caseID).Count(&n).Error != nil {
			return ErrWaitDataUnavailable
		}
		if n != 0 {
			return ErrWaitDataConflict
		}
		if l.tx.Model(&models.InterventionRerunRequest{}).Where("tenant_id = ? AND case_id = ? AND data_revision_id = ?", tenantID, caseID, latest.ID).Count(&n).Error != nil {
			return ErrWaitDataUnavailable
		}
		if n != 0 {
			return ErrWaitDataPending
		}
		now := time.Now().UTC()
		result = models.InterventionRerunRequest{ID: pkg.NewUUID(), TenantID: tenantID, CaseID: caseID, JobID: l.caseRow.JobID, ConversationID: l.conv.ID, DataRevisionID: latest.ID, DataRevisionDigest: latest.Digest, OriginalSnapshotID: l.snapshot.ID, OriginalSnapshotDigest: l.snapshot.Digest, ExpectedVersion: in.ExpectedVersion, CaseVersion: in.ExpectedVersion + 1, ActorID: actorID, Reason: strings.TrimSpace(in.Reason), RequestKey: in.RequestKey, PayloadHash: hash, Status: "AWAITING_EXECUTION_AUTHORITY", Execution: "NOT_RUN", CreatedAt: now}
		if l.tx.Create(&result).Error != nil {
			return ErrWaitDataUnavailable
		}
		return l.appendMutation(actorID, "REQUEST_RERUN", in.Reason, "LOCAL_MATCH_CURRENTNESS_UNVERIFIED", &latest.ID, &result.ID, now)
	})
	if err != nil {
		return nil, false, err
	}
	return &result, replayed, nil
}
