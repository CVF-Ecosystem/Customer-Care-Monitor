package handlers_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/handlers"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type m3BarrierKey struct{}

func m3Wait(t *testing.T, signal <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(15 * time.Second):
		t.Fatalf("barrier not reached: %s", label)
	}
}
func m3AssertNoHumanWrite(t *testing.T, caseID string) {
	t.Helper()
	var c models.InterventionCase
	if db.DB.Where("id = ?", caseID).Take(&c).Error != nil || c.Version != 1 {
		t.Fatal("case changed on denied action")
	}
	var n int64
	if db.DB.Model(&models.InterventionHumanEvent{}).Where("case_id = ?", caseID).Count(&n).Error != nil || n != 0 {
		t.Fatalf("partial event count=%d", n)
	}
}

// These tests seed a synthetic proposal and full snapshot directly. They do
// not construct a provider, invoke an AI route, or assert CVF governance.
type m3Fixture struct {
	tenant, otherTenant, owner, member, reader, unknown, assignee, caseID, waitID, proposal, snapshot, job, conversation, message string
	token                                                                                                                         map[string]string
	router                                                                                                                        http.Handler
}

func m3MustCreate(t *testing.T, rows ...any) {
	t.Helper()
	for _, row := range rows {
		if err := db.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func m3FixtureStart(t *testing.T) *m3Fixture {
	t.Helper()
	if handlers.InterventionRunner != nil {
		t.Fatal("R101 must not install an intervention transport")
	}
	dsn := os.Getenv("R101_MYSQL_DSN")
	if dsn == "" {
		t.Skip("NOT_RUN: R101_MYSQL_DSN owned disposable MySQL required")
	}
	if err := db.Connect(dsn, false); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.DB.Logger = logger.Default.LogMode(logger.Silent)
	if err := db.AutoMigrate(); err != nil {
		t.Fatal(err)
	}
	f := &m3Fixture{tenant: pkg.NewUUID(), otherTenant: pkg.NewUUID(), owner: pkg.NewUUID(), member: pkg.NewUUID(), reader: pkg.NewUUID(), unknown: pkg.NewUUID(), assignee: pkg.NewUUID(), caseID: pkg.NewUUID(), waitID: pkg.NewUUID(), proposal: pkg.NewUUID(), snapshot: pkg.NewUUID(), job: pkg.NewUUID(), conversation: pkg.NewUUID(), message: pkg.NewUUID(), token: map[string]string{}}
	now := time.Now().UTC().Truncate(time.Millisecond)
	password := "R101SyntheticOnly!123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	m3MustCreate(t,
		&models.Tenant{ID: f.tenant, Name: "R101 Synthetic", Slug: "r101-" + f.tenant, Settings: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Tenant{ID: f.otherTenant, Name: "R101 Other", Slug: "r101-" + f.otherTenant, Settings: "{}", CreatedAt: now, UpdatedAt: now},
	)
	for _, user := range []struct{ id, role, permissions string }{
		{f.owner, "owner", "{}"}, {f.member, "member", `{"intervention_cases":"rw"}`},
		{f.reader, "member", `{"intervention_cases":"r"}`}, {f.unknown, "unknown", `{"intervention_cases":"rw"}`},
		{f.assignee, "member", `{"intervention_cases":"r"}`},
	} {
		m3MustCreate(t, &models.User{ID: user.id, Email: user.id + "@example.invalid", Name: "R101 " + user.role, PasswordHash: string(hash), CreatedAt: now, UpdatedAt: now},
			&models.UserTenant{UserID: user.id, TenantID: f.tenant, Role: user.role, Permissions: user.permissions})
	}
	channel := pkg.NewUUID()
	run := pkg.NewUUID()
	content := "Tôi khiếu nại vì đơn hàng bị chậm"
	m3MustCreate(t,
		&models.Channel{ID: channel, TenantID: f.tenant, ChannelType: "pancake", Name: "R101 synthetic source", ExternalID: channel, CredentialsEncrypted: []byte("synthetic"), IsDemoFixture: true, IsActive: true, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Conversation{ID: f.conversation, TenantID: f.tenant, ChannelID: channel, ExternalConversationID: f.conversation, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Message{ID: f.message, TenantID: f.tenant, ConversationID: f.conversation, ExternalMessageID: f.message, SenderType: "customer", Content: content, ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: now, CreatedAt: now},
		&models.Job{ID: f.job, TenantID: f.tenant, Name: "R101 local human review", JobType: "intervention", InputChannelIDs: fmt.Sprintf("[%q]", channel), RulesConfig: "{}", Outputs: "[]", ScheduleType: "manual", IsActive: true, CreatedAt: now, UpdatedAt: now},
	)
	var storedMessage models.Message
	if err := db.DB.Where("id = ?", f.message).Take(&storedMessage).Error; err != nil {
		t.Fatal(err)
	}
	contentHash := sha256.Sum256([]byte(storedMessage.Content))
	manifest := map[string]any{
		"schema_version": "ccma.snapshot.v1", "tenant_id": f.tenant, "conversation_id": f.conversation,
		"coverage": "complete", "coverage_reasons": []string{}, "omitted_earlier_messages": 0,
		"messages": []map[string]any{{"message_id": f.message, "external_message_id": f.message, "sender_type": "customer", "sender_name": "", "content_type": "text", "sent_at": storedMessage.SentAt.UTC().Format(time.RFC3339Nano), "content_sha256": hex.EncodeToString(contentHash[:]), "content_code_points": len([]rune(storedMessage.Content)), "attachment_coverage": "none", "attachment_count": 0}},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	digestBytes := sha256.Sum256(manifestJSON)
	digest := hex.EncodeToString(digestBytes[:])
	m3MustCreate(t,
		&models.AnalysisSnapshot{ID: f.snapshot, TenantID: f.tenant, JobRunID: run, ConversationID: f.conversation, SchemaVersion: "ccma.snapshot.v1", Digest: digest, Coverage: "complete", CoverageReasons: "[]", MessageCount: 1, Manifest: string(manifestJSON), CreatedAt: now},
		&models.InterventionCase{ID: f.caseID, TenantID: f.tenant, JobID: f.job, RunID: run, ConversationID: f.conversation, UseCase: "pancake_complaint_wait", Version: 1, SnapshotID: &f.snapshot, SnapshotDigest: digest, Eligibility: "ELIGIBLE", Execution: "LLM_SUCCEEDED", Disposition: "PENDING_REVIEW", WorkState: "OPEN", ActorID: f.owner, PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "r101-direct-seed", SchemaVersion: "ccma.intervention.v1", CreatedAt: now, UpdatedAt: now},
		&models.InterventionProposal{ID: f.proposal, TenantID: f.tenant, CaseID: f.caseID, Revision: 1, RunID: run, SnapshotID: f.snapshot, SnapshotDigest: digest, PromptVersion: "r101-direct-seed", PolicyVersion: "ccma.synthetic.intervention.v1", SchemaVersion: "ccma.intervention.v1", Kind: "complaint", Explanation: "Đơn hàng chậm", NextAction: "Kiểm tra đơn hàng", Evidence: fmt.Sprintf(`[{"message_id":%q,"quote":"khiếu nại"}]`, f.message), Provider: "synthetic-seed-not-provider", Model: "none", UsageState: "UNKNOWN", CreatedAt: now},
	)
	// An empty-source case exercises human WAIT_DATA ownership without AI.
	waitConv := pkg.NewUUID()
	waitSnap := pkg.NewUUID()
	waitRun := pkg.NewUUID()
	waitManifest := map[string]any{"schema_version": "ccma.snapshot.v1", "tenant_id": f.tenant, "conversation_id": waitConv, "coverage": "empty", "coverage_reasons": []string{"NO_MESSAGES"}, "omitted_earlier_messages": 0, "messages": []any{}}
	waitJSON, _ := json.Marshal(waitManifest)
	waitDigestBytes := sha256.Sum256(waitJSON)
	waitDigest := hex.EncodeToString(waitDigestBytes[:])
	m3MustCreate(t,
		&models.Conversation{ID: waitConv, TenantID: f.tenant, ChannelID: channel, ExternalConversationID: waitConv, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.AnalysisSnapshot{ID: waitSnap, TenantID: f.tenant, JobRunID: waitRun, ConversationID: waitConv, SchemaVersion: "ccma.snapshot.v1", Digest: waitDigest, Coverage: "empty", CoverageReasons: `["NO_MESSAGES"]`, MessageCount: 0, Manifest: string(waitJSON), CreatedAt: now},
		&models.InterventionCase{ID: f.waitID, TenantID: f.tenant, JobID: f.job, RunID: waitRun, ConversationID: waitConv, UseCase: "pancake_complaint_wait", Version: 1, SnapshotID: &waitSnap, SnapshotDigest: waitDigest, Eligibility: "WAIT_DATA", Execution: "NOT_RUN", Disposition: "WAIT_DATA", WorkState: "WAITING_ON_DATA", ReasonCode: "SOURCE_INCOMPLETE", NextAction: "complete_synthetic_source", ActorID: f.owner, PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "r101-direct-seed", SchemaVersion: "ccma.intervention.v1", CreatedAt: now, UpdatedAt: now},
	)
	middleware.SetJWTSecret(strings.Repeat("r", 32))
	for _, id := range []string{f.owner, f.member, f.reader, f.unknown, f.assignee} {
		f.token[id], err = middleware.GenerateAccessToken(id, id+"@example.invalid", false)
		if err != nil {
			t.Fatal(err)
		}
	}
	f.router = api.SetupRouter(&config.Config{RateLimitPerIP: 1000, RateLimitPerUser: 1000})
	return f
}

func (f *m3Fixture) send(method, tenant, id, path string, body any) *httptest.ResponseRecorder {
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, "/api/v1/tenants/"+tenant+path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+f.token[id])
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func (f *m3Fixture) action(id, caseID, name string, version int, more map[string]any) map[string]any {
	var row models.InterventionCase
	_ = db.DB.Where("id = ?", caseID).Take(&row).Error
	proposalID, revision := "", 0
	if caseID == f.caseID {
		proposalID, revision = f.proposal, 1
	}
	req := map[string]any{"action": name, "expected_version": version, "expected_snapshot_id": *row.SnapshotID, "expected_snapshot_digest": row.SnapshotDigest, "expected_proposal_id": proposalID, "expected_revision": revision, "reason": "R101 synthetic reviewer reason"}
	for k, v := range more {
		req[k] = v
	}
	return req
}

func TestM3CaseParticipantsAreScopedToHistory(t *testing.T) {
	f := m3FixtureStart(t)
	for id, name := range map[string]string{f.owner: "Case owner", f.member: "Current assignee", f.assignee: "Previous assignee"} {
		if err := db.DB.Model(&models.User{}).Where("id = ?", id).Update("name", name).Error; err != nil {
			t.Fatal(err)
		}
	}
	foreignID := pkg.NewUUID()
	m3MustCreate(t, &models.User{ID: foreignID, Email: foreignID + "@example.invalid", Name: "Foreign private name", PasswordHash: "synthetic", CreatedAt: time.Now(), UpdatedAt: time.Now()}, &models.UserTenant{UserID: foreignID, TenantID: f.otherTenant, Role: "member", Permissions: `{"intervention_cases":"r"}`})
	casePath := "/intervention-cases/" + f.caseID
	waitPath := "/intervention-cases/" + f.waitID
	due := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	steps := []struct {
		path, actor, caseID, action string
		version                     int
		fields                      map[string]any
	}{
		{casePath, f.member, f.caseID, "CORRECT", 1, map[string]any{"explanation": "Human correction", "next_action": "Check order"}},
		{casePath, f.owner, f.caseID, "ASSIGN", 2, map[string]any{"assignee_id": f.assignee, "due_at": due}},
		{casePath, f.owner, f.caseID, "ASSIGN", 3, map[string]any{"assignee_id": f.member, "due_at": due}},
		{casePath, f.member, f.caseID, "RESOLVE", 4, map[string]any{"outcome": "Completed"}},
		{waitPath, f.owner, f.waitID, "ASSIGN_DATA", 1, map[string]any{"assignee_id": f.assignee, "due_at": due, "next_action": "Collect source"}},
		{waitPath, f.owner, f.waitID, "ASSIGN_DATA", 2, map[string]any{"assignee_id": f.member, "due_at": due, "next_action": "Collect missing source"}},
	}
	for _, step := range steps {
		if rec := f.send(http.MethodPost, f.tenant, step.actor, step.path+"/actions", f.action(step.actor, step.caseID, step.action, step.version, step.fields)); rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", step.action, rec.Code, rec.Body.String())
		}
	}
	var event models.InterventionHumanEvent
	if err := db.DB.Where("case_id = ?", f.caseID).Order("case_version ASC").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	var before map[string]any
	if err := json.Unmarshal([]byte(event.Before), &before); err != nil {
		t.Fatal(err)
	}
	before["assignee_id"], before["data_owner_id"] = foreignID, "malformed-id"
	forged, _ := json.Marshal(before)
	if err := db.DB.Model(&event).Update("before", string(forged)).Error; err != nil {
		t.Fatal(err)
	}
	check := func(path string, want map[string]string) {
		t.Helper()
		rec := f.send(http.MethodGet, f.tenant, f.reader, path, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("reader detail: %d %s", rec.Code, rec.Body.String())
		}
		var detail struct {
			Participants []map[string]any `json:"participants"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		for _, item := range detail.Participants {
			if len(item) != 2 {
				t.Fatalf("participant exposed extra fields: %v", item)
			}
			id, idOK := item["user_id"].(string)
			name, nameOK := item["name"].(string)
			if !idOK || !nameOK {
				t.Fatalf("invalid participant: %v", item)
			}
			got[id] = name
		}
		if len(got) != len(want) {
			t.Fatalf("participants = %v, want %v", got, want)
		}
		for id, name := range want {
			if got[id] != name {
				t.Fatalf("participant %s = %q, want %q", id, got[id], name)
			}
		}
		if _, found := got[foreignID]; found {
			t.Fatal("foreign tenant name leaked")
		}
	}
	check(casePath, map[string]string{f.owner: "Case owner", f.member: "Current assignee", f.assignee: "Previous assignee"})
	check(waitPath, map[string]string{f.owner: "Case owner", f.member: "Current assignee", f.assignee: "Previous assignee"})
	for _, id := range []string{f.owner, f.member} {
		if rec := f.send(http.MethodGet, f.tenant, id, "/intervention-assignees", nil); rec.Code != http.StatusOK {
			t.Fatalf("editor assignees: %d", rec.Code)
		}
	}
	if rec := f.send(http.MethodGet, f.tenant, f.reader, "/intervention-assignees", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("reader assignees: %d", rec.Code)
	}
}

func TestM3CaseParticipantLookupFailureIsClosed(t *testing.T) {
	f := m3FixtureStart(t)
	path := "/intervention-cases/" + f.caseID
	if rec := f.send(http.MethodPost, f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "CORRECT", 1, map[string]any{"explanation": "Reviewed", "next_action": "Follow up"})); rec.Code != http.StatusOK {
		t.Fatalf("seed actor event: %d %s", rec.Code, rec.Body.String())
	}
	const hook = "m3:participant_batch_failure"
	fired := 0
	if err := db.DB.Callback().Row().Before("gorm:row").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.TableExpr == nil || tx.Statement.TableExpr.SQL != "user_tenants AS ut" || len(tx.Statement.Selects) != 1 || tx.Statement.Selects[0] != "ut.user_id, users.name" {
			return
		}
		where, ok := tx.Statement.Clauses["WHERE"].Expression.(clause.Where)
		if !ok {
			return
		}
		for _, condition := range where.Exprs {
			if expression, ok := condition.(clause.Expr); ok && strings.Contains(expression.SQL, "ut.user_id IN") {
				fired++
				tx.AddError(errors.New("forced participant lookup failure"))
				return
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	removed := false
	t.Cleanup(func() {
		if !removed {
			_ = db.DB.Callback().Row().Remove(hook)
		}
	})
	rec := f.send(http.MethodGet, f.tenant, f.reader, path, nil)
	if fired != 1 {
		t.Fatalf("participant batch failure callback fired %d times", fired)
	}
	if rec.Code != http.StatusServiceUnavailable || strings.TrimSpace(rec.Body.String()) != `{"error":"participants_unavailable"}` {
		t.Fatalf("lookup failure returned %d %s", rec.Code, rec.Body.String())
	}
	if err := db.DB.Callback().Row().Remove(hook); err != nil {
		t.Fatal(err)
	}
	removed = true
	if rec := f.send(http.MethodGet, f.tenant, f.reader, path, nil); rec.Code != http.StatusOK {
		t.Fatalf("detail after hook removal: %d %s", rec.Code, rec.Body.String())
	}
}

// A human-owned case must retain a way to act after a supported local source
// writer adds a message; the old AI proposal remains bound to its first source.
func TestM3ChangedSourceAfterAssignmentRequiresExplicitReview(t *testing.T) {
	f := m3FixtureStart(t)
	path := "/intervention-cases/" + f.caseID
	due := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "CONFIRM", 1, nil)); rec.Code != 200 {
		t.Fatalf("confirm: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "ASSIGN", 2, map[string]any{"assignee_id": f.assignee, "due_at": due})); rec.Code != 200 {
		t.Fatalf("assign: %d %s", rec.Code, rec.Body.String())
	}
	now := time.Now().UTC().Add(time.Second)
	m3MustCreate(t, &models.Message{ID: pkg.NewUUID(), TenantID: f.tenant, ConversationID: f.conversation, ExternalMessageID: pkg.NewUUID(), SenderType: "customer", Content: "Tôi vẫn chờ phản hồi", ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: now, CreatedAt: now})
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "RESOLVE", 3, map[string]any{"outcome": "Đã xử lý"})); rec.Code != 409 {
		t.Fatalf("stale resolution: %d %s", rec.Code, rec.Body.String())
	}
	var after models.InterventionCase
	if err := db.DB.Where("id = ?", f.caseID).Take(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after.Version != 3 || after.WorkState != "ASSIGNED" || after.AssigneeID == nil || *after.AssigneeID != f.assignee {
		t.Fatalf("source guard changed owner/state: %+v", after)
	}
	var events int64
	if err := db.DB.Model(&models.InterventionHumanEvent{}).Where("case_id = ?", f.caseID).Count(&events).Error; err != nil || events != 2 {
		t.Fatalf("guard history count=%d err=%v", events, err)
	}
	var detail struct {
		CurrentSourceDigest   string `json:"current_source_digest"`
		ReviewedSourceCurrent bool   `json:"reviewed_source_current"`
	}
	if rec := f.send("GET", f.tenant, f.owner, path, nil); rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &detail) != nil || len(detail.CurrentSourceDigest) != 64 || detail.ReviewedSourceCurrent {
		t.Fatalf("changed detail: %d %s", rec.Code, rec.Body.String())
	}
	ack := f.action(f.owner, f.caseID, "ACKNOWLEDGE_SOURCE_CHANGE", 3, map[string]any{"expected_current_digest": strings.Repeat("0", 64)})
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", ack); rec.Code != 409 {
		t.Fatalf("blind acknowledgement: %d %s", rec.Code, rec.Body.String())
	}
	ack["expected_current_digest"] = detail.CurrentSourceDigest
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "ACKNOWLEDGE_SOURCE_CHANGE", 2, map[string]any{"expected_current_digest": detail.CurrentSourceDigest})); rec.Code != 409 {
		t.Fatalf("stale acknowledgement version: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.reader, path+"/actions", ack); rec.Code != 403 {
		t.Fatalf("reader acknowledgement: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.member, path+"/actions", ack); rec.Code != 403 {
		t.Fatalf("reviewer without source access: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.otherTenant, f.owner, path+"/actions", ack); rec.Code != 403 {
		t.Fatalf("foreign tenant acknowledgement: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", ack); rec.Code != 200 {
		t.Fatalf("acknowledgement: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "RESOLVE", 4, map[string]any{"outcome": "Đã xử lý"})); rec.Code != 200 {
		t.Fatalf("resolution after review: %d %s", rec.Code, rec.Body.String())
	}
	if err := db.DB.Where("id = ?", f.caseID).Take(&after).Error; err != nil || after.Version != 5 || after.WorkState != "RESOLVED" || after.AssigneeID == nil || *after.AssigneeID != f.assignee || after.ReviewedDataRevisionID == nil {
		t.Fatalf("recovered state: %+v err=%v", after, err)
	}
	var review models.InterventionDataRevision
	if err := db.DB.Where("id = ?", *after.ReviewedDataRevisionID).Take(&review).Error; err != nil || !engine.VerifyInterventionDataRevision(review) || review.Digest != detail.CurrentSourceDigest {
		t.Fatalf("reviewed revision invalid: %+v err=%v", review, err)
	}
	if err := db.DB.Model(&models.InterventionHumanEvent{}).Where("case_id = ?", f.caseID).Count(&events).Error; err != nil || events != 4 {
		t.Fatalf("recovered history count=%d err=%v", events, err)
	}
	var original models.InterventionProposal
	if err := db.DB.Where("id = ?", f.proposal).Take(&original).Error; err != nil || original.SnapshotID != f.snapshot || original.SnapshotDigest != after.SnapshotDigest || after.HumanAssessment != "CONFIRM" || after.Disposition != "HUMAN_CONFIRM" {
		t.Fatalf("original AI proposal or human decision changed: case=%+v proposal=%+v err=%v", after, original, err)
	}
	var reviewEvent models.InterventionHumanEvent
	if err := db.DB.Where("case_id = ? AND action = ?", f.caseID, "ACKNOWLEDGE_SOURCE_CHANGE").Order("case_version ASC").Take(&reviewEvent).Error; err != nil || reviewEvent.DataRevisionID == nil || *reviewEvent.DataRevisionID != review.ID || reviewEvent.SourceStatus != "CHANGED" {
		t.Fatalf("review audit event missing: %+v err=%v", reviewEvent, err)
	}
	// A second source change invalidates the acknowledgement, including after
	// resolution. The reviewer can explicitly bind the newer version to reopen.
	newer := time.Now().UTC().Add(2 * time.Second)
	m3MustCreate(t, &models.Message{ID: pkg.NewUUID(), TenantID: f.tenant, ConversationID: f.conversation, ExternalMessageID: pkg.NewUUID(), SenderType: "customer", Content: "Tôi vẫn chưa nhận được kết quả", ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: newer, CreatedAt: newer})
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "REOPEN", 5, map[string]any{"due_at": due})); rec.Code != 409 {
		t.Fatalf("reopen after newer change: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("GET", f.tenant, f.owner, path, nil); rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &detail) != nil || detail.ReviewedSourceCurrent || detail.CurrentSourceDigest == review.Digest {
		t.Fatalf("newer source detail: %d %s", rec.Code, rec.Body.String())
	}
	ack = f.action(f.owner, f.caseID, "ACKNOWLEDGE_SOURCE_CHANGE", 5, map[string]any{"expected_current_digest": detail.CurrentSourceDigest})
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", ack); rec.Code != 200 {
		t.Fatalf("newer source acknowledgement: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "REOPEN", 6, map[string]any{"due_at": due})); rec.Code != 200 {
		t.Fatalf("reopen after newer review: %d %s", rec.Code, rec.Body.String())
	}
}

func TestM3ChangedSourceBeforeAssessmentStillConflicts(t *testing.T) {
	f := m3FixtureStart(t)
	path := "/intervention-cases/" + f.caseID
	now := time.Now().UTC().Add(time.Second)
	m3MustCreate(t, &models.Message{ID: pkg.NewUUID(), TenantID: f.tenant, ConversationID: f.conversation, ExternalMessageID: pkg.NewUUID(), SenderType: "customer", Content: "Tôi vẫn chờ", ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: now, CreatedAt: now})
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "CONFIRM", 1, nil)); rec.Code != 409 {
		t.Fatalf("confirm on changed source: %d %s", rec.Code, rec.Body.String())
	}
	var detail struct {
		CurrentSourceDigest string `json:"current_source_digest"`
	}
	if rec := f.send("GET", f.tenant, f.owner, path, nil); rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &detail) != nil {
		t.Fatalf("changed detail: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "ACKNOWLEDGE_SOURCE_CHANGE", 1, map[string]any{"expected_current_digest": detail.CurrentSourceDigest})); rec.Code != 409 {
		t.Fatalf("acknowledgement before assessment: %d %s", rec.Code, rec.Body.String())
	}
}

func TestM3ChangedSourceAfterAssessmentBeforeAssignmentCanRecover(t *testing.T) {
	f := m3FixtureStart(t)
	path := "/intervention-cases/" + f.caseID
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "CONFIRM", 1, nil)); rec.Code != 200 {
		t.Fatalf("confirm: %d %s", rec.Code, rec.Body.String())
	}
	now := time.Now().UTC().Add(time.Second)
	m3MustCreate(t, &models.Message{ID: pkg.NewUUID(), TenantID: f.tenant, ConversationID: f.conversation, ExternalMessageID: pkg.NewUUID(), SenderType: "customer", Content: "Tôi vẫn chờ", ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: now, CreatedAt: now})
	due := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "ASSIGN", 2, map[string]any{"assignee_id": f.assignee, "due_at": due})); rec.Code != 409 {
		t.Fatalf("assign before review: %d %s", rec.Code, rec.Body.String())
	}
	var detail struct {
		CurrentSourceDigest string `json:"current_source_digest"`
	}
	if rec := f.send("GET", f.tenant, f.owner, path, nil); rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &detail) != nil {
		t.Fatalf("changed detail: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "ACKNOWLEDGE_SOURCE_CHANGE", 2, map[string]any{"expected_current_digest": detail.CurrentSourceDigest})); rec.Code != 200 {
		t.Fatalf("ack after assessment: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.owner, path+"/actions", f.action(f.owner, f.caseID, "ASSIGN", 3, map[string]any{"assignee_id": f.assignee, "due_at": due})); rec.Code != 200 {
		t.Fatalf("assign after review: %d %s", rec.Code, rec.Body.String())
	}
}

func TestR101HumanWorkflowNative(t *testing.T) {
	f := m3FixtureStart(t)
	prefix := "/intervention-cases/" + f.caseID
	var budgetBefore, attemptBefore int64
	_ = db.DB.Model(&models.InterventionBudget{}).Count(&budgetBefore).Error
	_ = db.DB.Model(&models.InterventionAttempt{}).Count(&attemptBefore).Error
	if rec := f.send("POST", f.tenant, f.reader, prefix+"/actions", f.action(f.reader, f.caseID, "CONFIRM", 1, nil)); rec.Code != 403 {
		t.Fatalf("read-only write %d", rec.Code)
	}
	if rec := f.send("POST", f.tenant, f.unknown, prefix+"/actions", f.action(f.unknown, f.caseID, "CONFIRM", 1, nil)); rec.Code != 403 {
		t.Fatalf("unknown-role write %d", rec.Code)
	}
	if rec := f.send("POST", f.otherTenant, f.owner, prefix+"/actions", f.action(f.owner, f.caseID, "CONFIRM", 1, nil)); rec.Code != 403 {
		t.Fatalf("foreign-tenant write %d", rec.Code)
	}
	if rec := f.send("POST", f.tenant, f.member, prefix+"/actions", f.action(f.member, f.caseID, "CORRECT", 1, map[string]any{"explanation": "Người duyệt thấy đơn chậm", "next_action": "Gọi nội bộ kiểm tra"})); rec.Code != 200 {
		t.Fatalf("correct %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.owner, prefix+"/actions", f.action(f.owner, f.caseID, "CONFIRM", 1, nil)); rec.Code != 409 {
		t.Fatalf("stale review %d", rec.Code)
	}
	due := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	if rec := f.send("POST", f.tenant, f.owner, prefix+"/actions", f.action(f.owner, f.caseID, "ASSIGN", 2, map[string]any{"assignee_id": pkg.NewUUID(), "due_at": due})); rec.Code != 400 {
		t.Fatalf("invalid assignee %d", rec.Code)
	}
	if rec := f.send("POST", f.tenant, f.owner, prefix+"/actions", f.action(f.owner, f.caseID, "ASSIGN", 2, map[string]any{"assignee_id": f.assignee, "due_at": due})); rec.Code != 200 {
		t.Fatalf("assign %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.member, prefix+"/actions", f.action(f.member, f.caseID, "RESOLVE", 3, map[string]any{"outcome": "Đã xác nhận và giao bộ phận xử lý"})); rec.Code != 200 {
		t.Fatalf("resolve %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.owner, prefix+"/actions", f.action(f.owner, f.caseID, "REOPEN", 4, map[string]any{"due_at": due})); rec.Code != 200 {
		t.Fatalf("reopen %d %s", rec.Code, rec.Body.String())
	}
	var proposal models.InterventionProposal
	if db.DB.Where("id = ?", f.proposal).Take(&proposal).Error != nil || proposal.Explanation != "Đơn hàng chậm" {
		t.Fatal("original proposal changed")
	}
	var events []models.InterventionHumanEvent
	if db.DB.Where("case_id = ?", f.caseID).Order("case_version").Find(&events).Error != nil || len(events) != 4 || events[0].ActorID != f.member || events[0].Reason == "" || events[0].Before == events[0].After || events[2].After == events[3].After {
		t.Fatalf("human history invalid: %d", len(events))
	}
	if rec := f.send("GET", f.tenant, f.owner, prefix, nil); rec.Code != 200 || !bytes.Contains(rec.Body.Bytes(), []byte(`"human_events"`)) {
		t.Fatalf("detail %d", rec.Code)
	}
	// Changed source cannot authorize another work action.
	if db.DB.Model(&models.Message{}).Where("id = ?", f.message).Update("content", "Nguồn đã đổi").Error != nil {
		t.Fatal("source edit")
	}
	if rec := f.send("POST", f.tenant, f.owner, prefix+"/actions", f.action(f.owner, f.caseID, "ASSIGN", 5, map[string]any{"assignee_id": f.assignee, "due_at": due})); rec.Code != 409 {
		t.Fatalf("changed source %d", rec.Code)
	}
	var budgetAfter, attemptAfter int64
	_ = db.DB.Model(&models.InterventionBudget{}).Count(&budgetAfter).Error
	_ = db.DB.Model(&models.InterventionAttempt{}).Count(&attemptAfter).Error
	if handlers.InterventionRunner != nil || budgetAfter != budgetBefore || attemptAfter != attemptBefore {
		t.Fatal("provider route or budget effect")
	}
}

func TestR101WaitDataAndAtomicity(t *testing.T) {
	f := m3FixtureStart(t)
	waitPath := "/intervention-cases/" + f.waitID
	// Even a missing snapshot must not silently complete a WAIT_DATA case.
	var wait models.InterventionCase
	_ = db.DB.Where("id = ?", f.waitID).Take(&wait).Error
	if err := db.DB.Delete(&models.AnalysisSnapshot{}, "id = ?", *wait.SnapshotID).Error; err != nil {
		t.Fatal(err)
	}
	due := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	if rec := f.send("POST", f.tenant, f.owner, waitPath+"/actions", f.action(f.owner, f.waitID, "ASSIGN_DATA", 1, map[string]any{"assignee_id": f.assignee, "due_at": due, "next_action": "Bổ sung lịch sử hội thoại"})); rec.Code != 200 {
		t.Fatalf("assign data %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.owner, waitPath+"/actions", f.action(f.owner, f.waitID, "ESCALATE_DATA", 2, map[string]any{"next_action": "Báo quản lý nguồn dữ liệu"})); rec.Code != 200 {
		t.Fatalf("escalate %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.owner, waitPath+"/actions", f.action(f.owner, f.waitID, "CONFIRM", 3, nil)); rec.Code != 409 {
		t.Fatalf("WAIT_DATA review %d", rec.Code)
	}
	var events []models.InterventionHumanEvent
	_ = db.DB.Where("case_id = ?", f.waitID).Order("case_version").Find(&events)
	if len(events) != 2 || events[0].SourceStatus != "UNAVAILABLE" || events[1].SourceStatus != "UNAVAILABLE" {
		t.Fatal("WAIT_DATA source truth")
	}
	// A unique event conflict after case CAS must roll the case version back.
	var c models.InterventionCase
	_ = db.DB.Where("id = ?", f.caseID).Take(&c).Error
	m3MustCreate(t, &models.InterventionHumanEvent{ID: pkg.NewUUID(), TenantID: f.tenant, CaseID: f.caseID, CaseVersion: 2, ActorID: f.owner, Action: "SYNTHETIC_COLLISION", Reason: "R101 rollback probe", SnapshotID: f.snapshot, SnapshotDigest: c.SnapshotDigest, SourceStatus: "PROBE", Before: "{}", After: "{}", CreatedAt: time.Now().UTC()})
	if rec := f.send("POST", f.tenant, f.owner, "/intervention-cases/"+f.caseID+"/actions", f.action(f.owner, f.caseID, "CONFIRM", 1, nil)); rec.Code != 503 {
		t.Fatalf("event insert failure %d", rec.Code)
	}
	var after models.InterventionCase
	_ = db.DB.Where("id = ?", f.caseID).Take(&after).Error
	if after.Version != 1 || after.HumanAssessment != "" {
		t.Fatal("projection committed without event")
	}
}

func TestR101CompetingReviewers(t *testing.T) {
	f := m3FixtureStart(t)
	path := "/intervention-cases/" + f.caseID + "/actions"
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for _, actor := range []string{f.owner, f.member} {
		wg.Add(1)
		go func(actor string) {
			defer wg.Done()
			codes <- f.send("POST", f.tenant, actor, path, f.action(actor, f.caseID, "CONFIRM", 1, nil)).Code
		}(actor)
	}
	wg.Wait()
	close(codes)
	counts := map[int]int{}
	for code := range codes {
		counts[code]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatalf("competing review result %+v", counts)
	}
	var events int64
	_ = db.DB.Model(&models.InterventionHumanEvent{}).Where("case_id = ?", f.caseID).Count(&events)
	if events != 1 {
		t.Fatalf("accepted events=%d", events)
	}
}

func TestR101SourceAndPermissionConflicts(t *testing.T) {
	f := m3FixtureStart(t)
	path := "/intervention-cases/" + f.caseID + "/actions"
	request := f.action(f.owner, f.caseID, "CONFIRM", 1, nil)
	deny := func(label string) {
		t.Helper()
		if rec := f.send("POST", f.tenant, f.owner, path, request); rec.Code != 409 {
			t.Fatalf("%s: code %d", label, rec.Code)
		}
	}
	var original models.Message
	if db.DB.Where("id = ?", f.message).Take(&original).Error != nil {
		t.Fatal("source")
	}
	if db.DB.Delete(&models.Message{}, "id = ?", f.message).Error != nil {
		t.Fatal("delete")
	}
	deny("deleted")
	m3MustCreate(t, &original)
	added := models.Message{ID: pkg.NewUUID(), TenantID: f.tenant, ConversationID: f.conversation, ExternalMessageID: pkg.NewUUID(), SenderType: "customer", Content: "Thêm nguồn", ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: original.SentAt.Add(time.Second), CreatedAt: original.CreatedAt}
	m3MustCreate(t, &added)
	deny("appended")
	if db.DB.Delete(&models.Message{}, "id = ?", added.ID).Error != nil {
		t.Fatal("remove addition")
	}
	var snap models.AnalysisSnapshot
	if db.DB.Where("id = ?", f.snapshot).Take(&snap).Error != nil {
		t.Fatal("snapshot")
	}
	if db.DB.Model(&models.AnalysisSnapshot{}).Where("id = ?", f.snapshot).Update("manifest", "{}").Error != nil {
		t.Fatal("corrupt")
	}
	deny("corrupt manifest")
	if db.DB.Model(&models.AnalysisSnapshot{}).Where("id = ?", f.snapshot).Update("manifest", snap.Manifest).Error != nil {
		t.Fatal("restore")
	}
	if db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.member).Update("permissions", `{"intervention_cases":"r"}`).Error != nil {
		t.Fatal("revoke")
	}
	if rec := f.send("POST", f.tenant, f.member, path, f.action(f.member, f.caseID, "CONFIRM", 1, nil)); rec.Code != 403 {
		t.Fatalf("revoked write %d", rec.Code)
	}
	if _, err := engine.M3ApplyAction(context.Background(), f.tenant, f.member, f.caseID, engine.M3Action{Action: "CONFIRM", ExpectedVersion: 1, ExpectedSnapshotID: f.snapshot, ExpectedSnapshotDigest: snap.Digest, ExpectedProposalID: f.proposal, ExpectedRevision: 1, Reason: "Direct service revocation check"}); err != engine.ErrM3Forbidden {
		t.Fatalf("direct service permission recheck: %v", err)
	}
	var c models.InterventionCase
	if db.DB.Where("id = ?", f.caseID).Take(&c).Error != nil || c.Version != 1 {
		t.Fatal("denied action changed case")
	}
}

func TestR101SourceLockSerialization(t *testing.T) {
	t.Run("source commits before waiting action", func(t *testing.T) {
		f := m3FixtureStart(t)
		var conv models.Conversation
		_ = db.DB.Where("id = ?", f.conversation).Take(&conv).Error
		writerLocked := make(chan struct{})
		releaseWriter := make(chan struct{})
		writerDone := make(chan error, 1)
		go func() {
			writerDone <- db.DB.Transaction(func(tx *gorm.DB) error {
				var ch models.Channel
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", conv.ChannelID).Take(&ch).Error; err != nil {
					return err
				}
				close(writerLocked)
				<-releaseWriter
				return tx.Model(&models.Message{}).Where("id = ?", f.message).Update("content", "R101 source writer changed content").Error
			})
		}()
		m3Wait(t, writerLocked, "source writer holds channel")
		atChannel := make(chan struct{})
		hook := "r101:source_first"
		if err := db.DB.Callback().Query().Before("gorm:query").Register(hook, func(tx *gorm.DB) {
			if tx.Statement.Table == "channels" && tx.Statement.Context.Value(m3BarrierKey{}) == "action" {
				select {
				case <-atChannel:
				default:
					close(atChannel)
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer db.DB.Callback().Query().Remove(hook)
		result := make(chan error, 1)
		go func() {
			_, err := engine.M3ApplyAction(context.WithValue(context.Background(), m3BarrierKey{}, "action"), f.tenant, f.owner, f.caseID, engine.M3Action{Action: "CONFIRM", ExpectedVersion: 1, ExpectedSnapshotID: f.snapshot, ExpectedSnapshotDigest: m3Digest(f.caseID), ExpectedProposalID: f.proposal, ExpectedRevision: 1, Reason: "R101 source-first race"})
			result <- err
		}()
		m3Wait(t, atChannel, "action reaches channel lock")
		close(releaseWriter)
		if err := <-writerDone; err != nil {
			t.Fatal(err)
		}
		if err := <-result; err != engine.ErrM3Conflict {
			t.Fatalf("source-first action result %v", err)
		}
		m3AssertNoHumanWrite(t, f.caseID)
	})
	t.Run("action commits before waiting source writer", func(t *testing.T) {
		f := m3FixtureStart(t)
		var conv models.Conversation
		_ = db.DB.Where("id = ?", f.conversation).Take(&conv).Error
		actionLocked := make(chan struct{})
		releaseAction := make(chan struct{})
		writerAtChannel := make(chan struct{})
		hook := "r101:action_first"
		if err := db.DB.Callback().Query().After("gorm:query").Register(hook, func(tx *gorm.DB) {
			if tx.Statement.Table == "channels" && tx.Statement.Context.Value(m3BarrierKey{}) == "action" {
				close(actionLocked)
				<-releaseAction
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer db.DB.Callback().Query().Remove(hook)
		if err := db.DB.Callback().Query().Before("gorm:query").Register(hook+":writer", func(tx *gorm.DB) {
			if tx.Statement.Table == "channels" && tx.Statement.Context.Value(m3BarrierKey{}) == "writer" {
				select {
				case <-writerAtChannel:
				default:
					close(writerAtChannel)
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer db.DB.Callback().Query().Remove(hook + ":writer")
		result := make(chan error, 1)
		go func() {
			_, err := engine.M3ApplyAction(context.WithValue(context.Background(), m3BarrierKey{}, "action"), f.tenant, f.owner, f.caseID, engine.M3Action{Action: "CONFIRM", ExpectedVersion: 1, ExpectedSnapshotID: f.snapshot, ExpectedSnapshotDigest: m3Digest(f.caseID), ExpectedProposalID: f.proposal, ExpectedRevision: 1, Reason: "R101 action-first race"})
			result <- err
		}()
		m3Wait(t, actionLocked, "action owns channel")
		writerDone := make(chan error, 1)
		go func() {
			writerDone <- db.DB.WithContext(context.WithValue(context.Background(), m3BarrierKey{}, "writer")).Transaction(func(tx *gorm.DB) error {
				var ch models.Channel
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", conv.ChannelID).Take(&ch).Error; err != nil {
					return err
				}
				return tx.Model(&models.Message{}).Where("id = ?", f.message).Update("content", "R101 later source edit").Error
			})
		}()
		m3Wait(t, writerAtChannel, "writer reaches channel")
		close(releaseAction)
		if err := <-result; err != nil {
			t.Fatalf("action-first action %v", err)
		}
		if err := <-writerDone; err != nil {
			t.Fatal(err)
		}
		var c models.InterventionCase
		_ = db.DB.Where("id = ?", f.caseID).Take(&c).Error
		var n int64
		_ = db.DB.Model(&models.InterventionHumanEvent{}).Where("case_id = ?", f.caseID).Count(&n)
		if c.Version != 2 || n != 1 {
			t.Fatalf("action-first case/event %d/%d", c.Version, n)
		}
	})
}

func m3Digest(caseID string) string {
	var c models.InterventionCase
	_ = db.DB.Where("id = ?", caseID).Take(&c).Error
	return c.SnapshotDigest
}

func TestR101PermissionLockSerialization(t *testing.T) {
	t.Run("revocation commits before waiting action", func(t *testing.T) {
		f := m3FixtureStart(t)
		held := make(chan struct{})
		release := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			done <- db.DB.Transaction(func(tx *gorm.DB) error {
				var ut models.UserTenant
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.member).Take(&ut).Error; err != nil {
					return err
				}
				close(held)
				<-release
				return tx.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.member).Update("permissions", `{"intervention_cases":"r"}`).Error
			})
		}()
		m3Wait(t, held, "revoker owns membership")
		atMembership := make(chan struct{})
		hook := "r101:permission_first"
		if err := db.DB.Callback().Query().Before("gorm:query").Register(hook, func(tx *gorm.DB) {
			if tx.Statement.Table == "user_tenants" && tx.Statement.Context.Value(m3BarrierKey{}) == "action" {
				select {
				case <-atMembership:
				default:
					close(atMembership)
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer db.DB.Callback().Query().Remove(hook)
		result := make(chan error, 1)
		go func() {
			_, err := engine.M3ApplyAction(context.WithValue(context.Background(), m3BarrierKey{}, "action"), f.tenant, f.member, f.caseID, engine.M3Action{Action: "CONFIRM", ExpectedVersion: 1, ExpectedSnapshotID: f.snapshot, ExpectedSnapshotDigest: m3Digest(f.caseID), ExpectedProposalID: f.proposal, ExpectedRevision: 1, Reason: "R101 revoked before action"})
			result <- err
		}()
		m3Wait(t, atMembership, "action reaches membership")
		close(release)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if err := <-result; err != engine.ErrM3Forbidden {
			t.Fatalf("revoked actor result %v", err)
		}
		m3AssertNoHumanWrite(t, f.caseID)
	})
	t.Run("action commits before waiting membership removal", func(t *testing.T) {
		f := m3FixtureStart(t)
		actionLocked := make(chan struct{})
		releaseAction := make(chan struct{})
		removerStarted := make(chan struct{})
		hook := "r101:action_membership_first"
		if err := db.DB.Callback().Query().After("gorm:query").Register(hook, func(tx *gorm.DB) {
			if tx.Statement.Table == "user_tenants" && tx.Statement.Context.Value(m3BarrierKey{}) == "action" {
				select {
				case <-actionLocked:
				default:
					close(actionLocked)
					<-releaseAction
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer db.DB.Callback().Query().Remove(hook)
		if err := db.DB.Callback().Delete().Before("gorm:delete").Register(hook+":delete", func(tx *gorm.DB) {
			if tx.Statement.Table == "user_tenants" && tx.Statement.Context.Value(m3BarrierKey{}) == "remover" {
				select {
				case <-removerStarted:
				default:
					close(removerStarted)
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer db.DB.Callback().Delete().Remove(hook + ":delete")
		result := make(chan error, 1)
		go func() {
			_, err := engine.M3ApplyAction(context.WithValue(context.Background(), m3BarrierKey{}, "action"), f.tenant, f.member, f.caseID, engine.M3Action{Action: "CONFIRM", ExpectedVersion: 1, ExpectedSnapshotID: f.snapshot, ExpectedSnapshotDigest: m3Digest(f.caseID), ExpectedProposalID: f.proposal, ExpectedRevision: 1, Reason: "R101 action before removal"})
			result <- err
		}()
		m3Wait(t, actionLocked, "action owns membership")
		removed := make(chan error, 1)
		go func() {
			removed <- db.DB.WithContext(context.WithValue(context.Background(), m3BarrierKey{}, "remover")).Where("tenant_id = ? AND user_id = ?", f.tenant, f.member).Delete(&models.UserTenant{}).Error
		}()
		m3Wait(t, removerStarted, "remover reaches membership")
		close(releaseAction)
		if err := <-result; err != nil {
			t.Fatalf("action before removal %v", err)
		}
		if err := <-removed; err != nil {
			t.Fatal(err)
		}
		var c models.InterventionCase
		_ = db.DB.Where("id = ?", f.caseID).Take(&c).Error
		var n int64
		_ = db.DB.Model(&models.InterventionHumanEvent{}).Where("case_id = ?", f.caseID).Count(&n)
		var membership int64
		_ = db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.member).Count(&membership)
		if c.Version != 2 || n != 1 || membership != 0 {
			t.Fatalf("serialized action/removal case=%d event=%d member=%d", c.Version, n, membership)
		}
	})
}

// A compiled test binary can run this exact router on loopback for Root's
// browser capture. It never installs InterventionRunner. The fixture is owned
// by the calling campaign and must be stopped and cleaned by that campaign.
func TestR101BrowserFixture(t *testing.T) {
	if os.Getenv("R101_BROWSER_FIXTURE") != "1" {
		t.Skip("NOT_RUN: browser fixture opt-in")
	}
	f := m3FixtureStart(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: f.router}
	defer server.Close()
	go func() { _ = server.Serve(listener) }()
	fmt.Printf("R101_BROWSER_READY http://127.0.0.1:%d tenant=%s case=%s reader=%s\n", listener.Addr().(*net.TCPAddr).Port, f.tenant, f.caseID, f.reader)
	// The public synthetic login is <owner UUID>@example.invalid with the
	// literal fixture-only password in this source file; no key or real account.
	fmt.Printf("R101_BROWSER_OWNER %s@example.invalid\n", f.owner)
	<-time.After(30 * time.Minute)
}
