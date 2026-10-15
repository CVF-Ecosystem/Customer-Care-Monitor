package handlers_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	mysqlcfg "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type r102Fixture struct {
	t                                                                                            *testing.T
	tenant, other, owner, writer, noJobs, reader, unknown, channel, conv, job, caseID, snap, run string
	originalDigest                                                                               string
	token                                                                                        map[string]string
	router                                                                                       http.Handler
}

func r102CreateCallback(t *testing.T, name string, cb func(*gorm.DB)) func() {
	t.Helper()
	r102DB(t, db.DB.Callback().Create().Before("gorm:create").Register(name, cb))
	var once sync.Once
	remove := func() {
		once.Do(func() {
			if err := db.DB.Callback().Create().Remove(name); err != nil {
				t.Error("R102 create callback cleanup: ", err)
			}
		})
	}
	t.Cleanup(remove)
	return remove
}

func r102UpdateCallback(t *testing.T, name string, cb func(*gorm.DB)) func() {
	t.Helper()
	r102DB(t, db.DB.Callback().Update().Before("gorm:update").Register(name, cb))
	var once sync.Once
	remove := func() {
		once.Do(func() {
			if err := db.DB.Callback().Update().Remove(name); err != nil {
				t.Error("R102 update callback cleanup: ", err)
			}
		})
	}
	t.Cleanup(remove)
	return remove
}

func r102DB(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal("R102 synthetic DB assertion failed: ", err)
	}
}

func r102ReleaseAndAwait(t *testing.T, release func(), workers *sync.WaitGroup) {
	t.Helper()
	release()
	done := make(chan struct{})
	go func() { workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(12 * time.Second):
		t.Error("R102 fixture goroutine did not finish before DB cleanup")
	}
}

func r102Receive(t *testing.T, ch <-chan int) int {
	t.Helper()
	select {
	case n := <-ch:
		return n
	case <-time.After(15 * time.Second):
		t.Fatal("R102 concurrent request timeout")
		return 0
	}
}

func r102Must(t *testing.T, rows ...any) {
	t.Helper()
	for _, row := range rows {
		if err := db.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
}

// No default application DSN is read. Root supplies only the new owned R102
// loopback fixture; this guard prevents accidental use of preserved R101 data.
func r102Start(t *testing.T) *r102Fixture {
	t.Helper()
	if handlers.InterventionRunner != nil {
		t.Fatal("provider runner must remain nil")
	}
	dsn := os.Getenv("R102_MYSQL_DSN")
	if dsn == "" {
		t.Skip("NOT_RUN: R102_MYSQL_DSN owned synthetic fixture required")
	}
	parsed, err := mysqlcfg.ParseDSN(dsn)
	if err != nil || parsed.Net != "tcp" || !strings.HasPrefix(parsed.Addr, "127.0.0.1:") || parsed.DBName != "r102_fixture" {
		t.Fatal("R102 DSN must target the named loopback synthetic database")
	}
	if err := db.Connect(dsn, false); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB.DB()
		if err != nil {
			t.Error("R102 DB handle cleanup: ", err)
			return
		}
		if err := sqlDB.Close(); err != nil {
			t.Error("R102 DB close: ", err)
		}
	})
	db.DB.Logger = logger.Default.LogMode(logger.Silent)
	if err := db.AutoMigrate(); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(); err != nil {
		t.Fatal("second migration: ", err)
	}
	f := &r102Fixture{t: t, tenant: pkg.NewUUID(), other: pkg.NewUUID(), owner: pkg.NewUUID(), writer: pkg.NewUUID(), noJobs: pkg.NewUUID(), reader: pkg.NewUUID(), unknown: pkg.NewUUID(), channel: pkg.NewUUID(), conv: pkg.NewUUID(), job: pkg.NewUUID(), caseID: pkg.NewUUID(), snap: pkg.NewUUID(), run: pkg.NewUUID(), token: map[string]string{}}
	now := time.Now().UTC().Truncate(time.Millisecond)
	t.Cleanup(func() {
		// Delete only this test's generated tenant rows; never drop the database.
		for _, model := range []any{&models.InterventionRerunRequest{}, &models.InterventionDataRevision{}, &models.InterventionHumanEvent{}, &models.InterventionCase{}, &models.AnalysisSnapshot{}, &models.Message{}, &models.Conversation{}, &models.Job{}, &models.Channel{}, &models.UserTenant{}} {
			if err := db.DB.Unscoped().Where("tenant_id = ?", f.tenant).Delete(model).Error; err != nil {
				t.Error("R102 owned row cleanup: ", err)
			}
		}
		if err := db.DB.Unscoped().Where("tenant_id = ?", f.other).Delete(&models.UserTenant{}).Error; err != nil {
			t.Error("R102 other membership cleanup: ", err)
		}
		if err := db.DB.Unscoped().Where("id IN ?", []string{f.owner, f.writer, f.noJobs, f.reader, f.unknown}).Delete(&models.User{}).Error; err != nil {
			t.Error("R102 user cleanup: ", err)
		}
		if err := db.DB.Unscoped().Where("id IN ?", []string{f.tenant, f.other}).Delete(&models.Tenant{}).Error; err != nil {
			t.Error("R102 tenant cleanup: ", err)
		}
	})
	hash, err := bcrypt.GenerateFromPassword([]byte("R102SyntheticOnly!123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	r102Must(t,
		&models.Tenant{ID: f.tenant, Name: "R102 synthetic", Slug: "r102-" + f.tenant, Settings: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Tenant{ID: f.other, Name: "R102 other", Slug: "r102-" + f.other, Settings: "{}", CreatedAt: now, UpdatedAt: now},
	)
	for _, user := range []struct{ id, role, perms string }{
		{f.owner, "owner", "{}"},
		{f.writer, "member", `{"intervention_cases":"rw","jobs":"w"}`},
		{f.noJobs, "member", `{"intervention_cases":"rw"}`},
		{f.reader, "member", `{"intervention_cases":"r"}`},
		{f.unknown, "unknown", `{"intervention_cases":"rw","jobs":"w"}`},
	} {
		r102Must(t, &models.User{ID: user.id, Email: user.id + "@example.invalid", Name: "R102 " + user.role, PasswordHash: string(hash), CreatedAt: now, UpdatedAt: now}, &models.UserTenant{UserID: user.id, TenantID: f.tenant, Role: user.role, Permissions: user.perms})
	}
	r102Must(t, &models.UserTenant{UserID: f.owner, TenantID: f.other, Role: "owner", Permissions: "{}"})
	emptyManifest, _ := json.Marshal(struct {
		SchemaVersion          string   `json:"schema_version"`
		TenantID               string   `json:"tenant_id"`
		ConversationID         string   `json:"conversation_id"`
		Coverage               string   `json:"coverage"`
		CoverageReasons        []string `json:"coverage_reasons"`
		OmittedEarlierMessages int64    `json:"omitted_earlier_messages"`
		Messages               []any    `json:"messages"`
	}{"ccma.snapshot.v1", f.tenant, f.conv, "empty", []string{"NO_MESSAGES"}, 0, []any{}})
	digestBytes := sha256.Sum256(emptyManifest)
	f.originalDigest = hex.EncodeToString(digestBytes[:])
	r102Must(t,
		&models.Channel{ID: f.channel, TenantID: f.tenant, ChannelType: "pancake", Name: "R102 source", ExternalID: f.channel, CredentialsEncrypted: []byte("synthetic"), IsDemoFixture: true, IsActive: true, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Conversation{ID: f.conv, TenantID: f.tenant, ChannelID: f.channel, ExternalConversationID: f.conv, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Job{ID: f.job, TenantID: f.tenant, Name: "R102 manual synthetic", JobType: "intervention", InputChannelIDs: fmt.Sprintf("[%q]", f.channel), RulesConfig: "{}", Outputs: "[]", ScheduleType: "manual", IsActive: true, CreatedAt: now, UpdatedAt: now},
		&models.AnalysisSnapshot{ID: f.snap, TenantID: f.tenant, JobRunID: f.run, ConversationID: f.conv, SchemaVersion: "ccma.snapshot.v1", Digest: f.originalDigest, Coverage: "empty", CoverageReasons: `["NO_MESSAGES"]`, MessageCount: 0, Manifest: string(emptyManifest), CreatedAt: now},
		&models.InterventionCase{ID: f.caseID, TenantID: f.tenant, JobID: f.job, RunID: f.run, ConversationID: f.conv, UseCase: "pancake_complaint_wait", Version: 1, SnapshotID: &f.snap, SnapshotDigest: f.originalDigest, Eligibility: "WAIT_DATA", Execution: "NOT_RUN", Disposition: "NOT_CREATED", WorkState: "WAITING_ON_DATA", ReasonCode: "SOURCE_INCOMPLETE", ActorID: f.owner, PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "r102-direct-seed", SchemaVersion: "ccma.intervention.v1", CreatedAt: now, UpdatedAt: now},
	)
	middleware.SetJWTSecret(strings.Repeat("s", 32))
	for _, id := range []string{f.owner, f.writer, f.noJobs, f.reader, f.unknown} {
		f.token[id], err = middleware.GenerateAccessToken(id, id+"@example.invalid", false)
		if err != nil {
			t.Fatal(err)
		}
	}
	f.router = api.SetupRouter(&config.Config{RateLimitPerIP: 1000, RateLimitPerUser: 1000})
	return f
}

func (f *r102Fixture) send(method, tenant, actor, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	if body == nil {
		b = nil
	}
	req := httptest.NewRequest(method, "/api/v1/tenants/"+tenant+"/intervention-cases/"+f.caseID+path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+f.token[actor])
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func (f *r102Fixture) source(content, contentType string) string {
	f.t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	id := pkg.NewUUID()
	if err := db.DB.Create(&models.Message{ID: id, TenantID: f.tenant, ConversationID: f.conv, ExternalMessageID: pkg.NewUUID(), SenderType: "customer", Content: content, ContentType: contentType, Attachments: "[]", RawData: "{}", SentAt: now, CreatedAt: now}).Error; err != nil {
		f.t.Fatal("R102 synthetic source insert failed: ", err)
	}
	return id
}

func (f *r102Fixture) revisionBody(version int) map[string]any {
	return map[string]any{"expected_version": version, "expected_snapshot_id": f.snap, "expected_snapshot_digest": f.originalDigest, "reason": "R102 source observed"}
}

func (f *r102Fixture) requestBody(version int, r models.InterventionDataRevision, key string) map[string]any {
	return map[string]any{"expected_version": version, "expected_snapshot_id": f.snap, "expected_snapshot_digest": f.originalDigest, "data_revision_id": r.ID, "data_revision_digest": r.Digest, "reason": "R102 explicit request", "request_key": key}
}

func (f *r102Fixture) captured(t *testing.T) models.InterventionDataRevision {
	t.Helper()
	f.source("Tôi khiếu nại và chờ đơn hàng", "text")
	rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1))
	if rec.Code != 201 {
		t.Fatalf("capture %d %s", rec.Code, rec.Body.String())
	}
	var r models.InterventionDataRevision
	if db.DB.Where("tenant_id = ? AND case_id = ?", f.tenant, f.caseID).Take(&r).Error != nil {
		t.Fatal("revision not stored")
	}
	return r
}

func TestR102NativeRevisionIntent(t *testing.T) {
	f := r102Start(t)
	var beforeRun, beforeAttempt, beforeBudget, beforeProposal int64
	r102DB(t, db.DB.Model(&models.JobRun{}).Where("tenant_id = ?", f.tenant).Count(&beforeRun).Error)
	r102DB(t, db.DB.Model(&models.InterventionAttempt{}).Where("tenant_id = ?", f.tenant).Count(&beforeAttempt).Error)
	r102DB(t, db.DB.Model(&models.InterventionBudget{}).Where("tenant_id = ?", f.tenant).Count(&beforeBudget).Error)
	r102DB(t, db.DB.Model(&models.InterventionProposal{}).Where("tenant_id = ?", f.tenant).Count(&beforeProposal).Error)
	var original models.AnalysisSnapshot
	if db.DB.Where("id = ?", f.snap).Take(&original).Error != nil {
		t.Fatal("original snapshot")
	}
	// Existing human ownership remains in place and must survive both writes.
	ownerID := f.owner
	if db.DB.Model(&models.InterventionCase{}).Where("id = ?", f.caseID).Update("data_owner_id", ownerID).Error != nil {
		t.Fatal("owner seed")
	}
	legacy := models.InterventionHumanEvent{ID: pkg.NewUUID(), TenantID: f.tenant, CaseID: f.caseID, CaseVersion: 1, ActorID: f.owner, Action: "ASSIGN_DATA", Reason: "earlier owner assignment", SnapshotID: f.snap, SnapshotDigest: f.originalDigest, SourceStatus: "INCOMPLETE", Before: `{"data_owner_id":null}`, After: `{"data_owner_id":"synthetic"}`, CreatedAt: time.Now().UTC()}
	r102Must(t, &legacy)
	if unchanged := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1)); unchanged.Code != 409 || !bytes.Contains(unchanged.Body.Bytes(), []byte("source_unchanged")) {
		t.Fatalf("empty source should be unchanged %d %s", unchanged.Code, unchanged.Body.String())
	}
	r := f.captured(t)
	manifestSum := sha256.Sum256([]byte(r.Manifest))
	if r.Coverage != "complete" || r.OriginalSnapshotDigest != f.originalDigest || strings.Contains(r.Manifest, "khiếu nại") || hex.EncodeToString(manifestSum[:]) != r.Digest {
		t.Fatal("revision provenance/content")
	}
	if rec := f.send("GET", f.tenant, f.reader, "/data-revisions?limit=1", nil); rec.Code != 200 || !bytes.Contains(rec.Body.Bytes(), []byte(r.ID)) || bytes.Contains(rec.Body.Bytes(), []byte(`"manifest"`)) {
		t.Fatalf("list %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("GET", f.tenant, f.reader, "/data-revisions/"+r.ID, nil); rec.Code != 200 || !bytes.Contains(rec.Body.Bytes(), []byte(`"manifest"`)) {
		t.Fatalf("detail %d", rec.Code)
	}
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(2)); rec.Code != 409 || !bytes.Contains(rec.Body.Bytes(), []byte("source_unchanged")) {
		t.Fatalf("unchanged %d", rec.Code)
	}
	request := f.requestBody(2, r, "r102-one")
	if rec := f.send("POST", f.tenant, f.noJobs, "/rerun-requests", request); rec.Code != 403 {
		t.Fatalf("jobs denial %d", rec.Code)
	}
	created := f.send("POST", f.tenant, f.writer, "/rerun-requests", request)
	if created.Code != 201 || !bytes.Contains(created.Body.Bytes(), []byte("AWAITING_EXECUTION_AUTHORITY")) || !bytes.Contains(created.Body.Bytes(), []byte("NOT_RUN")) {
		t.Fatalf("request %d %s", created.Code, created.Body.String())
	}
	if replay := f.send("POST", f.tenant, f.writer, "/rerun-requests", request); replay.Code != 200 {
		t.Fatalf("replay %d", replay.Code)
	}
	request["reason"] = "changed"
	if conflict := f.send("POST", f.tenant, f.writer, "/rerun-requests", request); conflict.Code != 409 {
		t.Fatalf("key conflict %d", conflict.Code)
	}
	secondKey := f.requestBody(3, r, "r102-second")
	if conflict := f.send("POST", f.tenant, f.writer, "/rerun-requests", secondKey); conflict.Code != 409 || !bytes.Contains(conflict.Body.Bytes(), []byte("request_pending")) {
		t.Fatalf("second pending intent %d %s", conflict.Code, conflict.Body.String())
	}
	var c models.InterventionCase
	if db.DB.Where("id = ?", f.caseID).Take(&c).Error != nil || c.Version != 3 || c.DataOwnerID == nil || *c.DataOwnerID != f.owner || c.SnapshotDigest != f.originalDigest || c.WorkState != "WAITING_ON_DATA" {
		t.Fatal("original case projection changed")
	}
	var events []models.InterventionHumanEvent
	if db.DB.Where("case_id = ?", f.caseID).Order("case_version").Find(&events).Error != nil || len(events) != 3 || events[0].ID != legacy.ID || events[0].Before != legacy.Before || events[0].After != legacy.After || events[0].DataRevisionID != nil || events[0].RerunRequestID != nil || events[1].DataRevisionID == nil || events[2].RerunRequestID == nil {
		t.Fatal("append-only events")
	}
	if rec := f.send("GET", f.tenant, f.reader, "/rerun-requests?limit=1", nil); rec.Code != 200 || !bytes.Contains(rec.Body.Bytes(), []byte("NOT_RUN")) {
		t.Fatalf("list requests %d", rec.Code)
	}
	var afterRun, afterAttempt, afterBudget, afterProposal int64
	r102DB(t, db.DB.Model(&models.JobRun{}).Where("tenant_id = ?", f.tenant).Count(&afterRun).Error)
	r102DB(t, db.DB.Model(&models.InterventionAttempt{}).Where("tenant_id = ?", f.tenant).Count(&afterAttempt).Error)
	r102DB(t, db.DB.Model(&models.InterventionBudget{}).Where("tenant_id = ?", f.tenant).Count(&afterBudget).Error)
	r102DB(t, db.DB.Model(&models.InterventionProposal{}).Where("tenant_id = ?", f.tenant).Count(&afterProposal).Error)
	var preserved models.AnalysisSnapshot
	r102DB(t, db.DB.Where("id = ?", f.snap).Take(&preserved).Error)
	if handlers.InterventionRunner != nil || beforeRun != afterRun || beforeAttempt != afterAttempt || beforeBudget != afterBudget || beforeProposal != afterProposal || original.Manifest != preserved.Manifest || original.Digest != preserved.Digest {
		t.Fatal("original analysis or execution ledger changed")
	}
}

func TestR102NativeDenialsAndRollback(t *testing.T) {
	f := r102Start(t)
	f.source("Hình ảnh chưa đủ", "image")
	body := f.revisionBody(1)
	for _, tc := range []struct {
		actor, tenant string
		want          int
	}{{f.reader, f.tenant, 403}, {f.unknown, f.tenant, 403}, {f.owner, f.other, 404}} {
		if rec := f.send("POST", tc.tenant, tc.actor, "/data-revisions", body); rec.Code != tc.want {
			t.Fatalf("denial %d", rec.Code)
		}
	}
	body["unexpected"] = true
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", body); rec.Code != 400 {
		t.Fatalf("unknown field %d", rec.Code)
	}
	delete(body, "unexpected")
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", body); rec.Code != 201 {
		t.Fatalf("incomplete revision %d %s", rec.Code, rec.Body.String())
	}
	var r models.InterventionDataRevision
	if db.DB.Where("case_id = ?", f.caseID).Take(&r).Error != nil || r.Coverage != "partial" || r.ObservationStatus != "INCOMPLETE" {
		t.Fatal("incomplete observation")
	}
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(2, r, "incomplete")); rec.Code != 409 {
		t.Fatalf("incomplete request %d", rec.Code)
	}
	// Occupy the next unique human-event version. Revision insertion and case
	// CAS must roll back together when event creation fails.
	r102Must(t, &models.InterventionHumanEvent{ID: pkg.NewUUID(), TenantID: f.tenant, CaseID: f.caseID, CaseVersion: 3, ActorID: f.owner, Action: "SYNTHETIC_CONFLICT", Reason: "forced rollback", SnapshotID: f.snap, SnapshotDigest: f.originalDigest, SourceStatus: "UNAVAILABLE", Before: `{}`, After: `{}`, CreatedAt: time.Now().UTC()})
	f.source("Tôi khiếu nại vì hàng chậm", "text")
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(2)); rec.Code != 503 {
		t.Fatalf("event failure %d", rec.Code)
	}
	var c models.InterventionCase
	var count int64
	r102DB(t, db.DB.Where("id = ?", f.caseID).Take(&c).Error)
	r102DB(t, db.DB.Model(&models.InterventionDataRevision{}).Where("case_id = ?", f.caseID).Count(&count).Error)
	if c.Version != 2 || count != 1 {
		t.Fatal("rollback did not preserve case/revision")
	}
}

func TestR102NativeChannelBarrier(t *testing.T) {
	f := r102Start(t)
	f.source("Tôi chờ hàng", "text")
	locked := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseNow := func() { releaseOnce.Do(func() { close(release) }) }
	var workers sync.WaitGroup
	defer r102ReleaseAndAwait(t, releaseNow, &workers)
	writerErr := make(chan error, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		writerErr <- db.DB.Transaction(func(tx *gorm.DB) error {
			var channel models.Channel
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", f.channel).Take(&channel).Error; err != nil {
				return err
			}
			close(locked)
			<-release
			now := time.Now().UTC().Truncate(time.Millisecond)
			id := pkg.NewUUID()
			return tx.Create(&models.Message{ID: id, TenantID: f.tenant, ConversationID: f.conv, ExternalMessageID: id, SenderType: "customer", Content: "Nguồn đồng bộ đến sau khóa", ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: now, CreatedAt: now}).Error
		})
	}()
	select {
	case <-locked:
	case <-time.After(10 * time.Second):
		t.Fatal("channel lock barrier")
	}
	done := make(chan *httptest.ResponseRecorder, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		done <- f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1))
	}()
	select {
	case <-done:
		t.Fatal("mutation crossed channel lock")
	case <-time.After(150 * time.Millisecond):
	}
	releaseNow()
	select {
	case err := <-writerErr:
		if err != nil {
			t.Fatal("sync-shaped writer failed: ", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("sync-shaped writer completion timeout")
	}
	select {
	case rec := <-done:
		if rec.Code != 201 {
			t.Fatalf("post-barrier %d", rec.Code)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("barrier release")
	}
	var observed models.InterventionDataRevision
	if db.DB.Where("case_id = ?", f.caseID).Take(&observed).Error != nil || observed.MessageCount != 2 {
		t.Fatal("writer-before-observer source missed")
	}
}

func TestR102NativeSourceChangesReplayAndRevocation(t *testing.T) {
	f := r102Start(t)
	r := f.captured(t)
	body := f.requestBody(2, r, "retry-after-source-change")
	message := f.source("Tin mới về đơn hàng", "text")
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", body); rec.Code != 409 {
		t.Fatalf("appended source must invalidate new request %d", rec.Code)
	}
	if err := db.DB.Where("id = ?", message).Delete(&models.Message{}).Error; err != nil {
		t.Fatal(err)
	}
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", body); rec.Code != 201 {
		t.Fatalf("restored source request %d %s", rec.Code, rec.Body.String())
	}
	f.source("Later local source", "text")
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", body); rec.Code != 200 {
		t.Fatalf("exact prior replay after source change %d", rec.Code)
	}
	if err := db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.writer).Update("permissions", `{}`).Error; err != nil {
		t.Fatal(err)
	}
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", body); rec.Code != 403 {
		t.Fatalf("revoked replay %d", rec.Code)
	}
	if rec := f.send("GET", f.tenant, f.writer, "/data-revisions", nil); rec.Code != 403 {
		t.Fatalf("revoked read %d", rec.Code)
	}
}

func TestR102NativeSourceMutationRejection(t *testing.T) {
	for _, kind := range []string{"edit", "delete", "append"} {
		t.Run(kind, func(t *testing.T) {
			f := r102Start(t)
			r := f.captured(t)
			var message models.Message
			if db.DB.Where("tenant_id = ? AND conversation_id = ?", f.tenant, f.conv).Take(&message).Error != nil {
				t.Fatal("captured message")
			}
			switch kind {
			case "edit":
				if err := db.DB.Model(&models.Message{}).Where("id = ?", message.ID).Update("content", "Tôi khiếu nại nội dung khác").Error; err != nil {
					t.Fatal(err)
				}
			case "delete":
				if err := db.DB.Where("id = ?", message.ID).Delete(&models.Message{}).Error; err != nil {
					t.Fatal(err)
				}
			case "append":
				f.source("Tin mới", "text")
			}
			if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(2, r, "source-"+kind)); rec.Code != 409 {
				t.Fatalf("%s source accepted %d", kind, rec.Code)
			}
		})
	}
}

func TestR102NativeCorruptRevisionUnavailable(t *testing.T) {
	f := r102Start(t)
	r := f.captured(t)
	if err := db.DB.Model(&models.InterventionDataRevision{}).Where("id = ?", r.ID).Update("manifest", `{"schema_version":"tampered"}`).Error; err != nil {
		t.Fatal(err)
	}
	if rec := f.send("GET", f.tenant, f.reader, "/data-revisions/"+r.ID, nil); rec.Code != 503 {
		t.Fatalf("corrupt detail %d", rec.Code)
	}
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(2, r, "corrupt")); rec.Code != 503 {
		t.Fatalf("corrupt request %d", rec.Code)
	}
}

func TestR102NativeLatestVersionAndRequestRollback(t *testing.T) {
	f := r102Start(t)
	r1 := f.captured(t)
	f.source("Khách đang chờ phản hồi", "text")
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(2)); rec.Code != 201 {
		t.Fatalf("second revision %d", rec.Code)
	}
	var r2 models.InterventionDataRevision
	if db.DB.Where("case_id = ? AND case_version = 3", f.caseID).Take(&r2).Error != nil {
		t.Fatal("second revision missing")
	}
	// Reverse timestamps to prove source lineage uses the case version.
	if err := db.DB.Model(&models.InterventionDataRevision{}).Where("id = ?", r1.ID).Update("created_at", time.Now().UTC().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(3, r1, "older")); rec.Code != 409 {
		t.Fatalf("older revision %d", rec.Code)
	}
	// Force the request's event insert to fail after request insert/CAS.
	r102Must(t, &models.InterventionHumanEvent{ID: pkg.NewUUID(), TenantID: f.tenant, CaseID: f.caseID, CaseVersion: 4, ActorID: f.owner, Action: "SYNTHETIC_CONFLICT", Reason: "forced request rollback", SnapshotID: f.snap, SnapshotDigest: f.originalDigest, SourceStatus: "UNAVAILABLE", Before: `{}`, After: `{}`, CreatedAt: time.Now().UTC()})
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(3, r2, "rollback")); rec.Code != 503 {
		t.Fatalf("request rollback %d", rec.Code)
	}
	var count int64
	var c models.InterventionCase
	r102DB(t, db.DB.Model(&models.InterventionRerunRequest{}).Where("case_id = ?", f.caseID).Count(&count).Error)
	r102DB(t, db.DB.Where("id = ?", f.caseID).Take(&c).Error)
	if count != 0 || c.Version != 3 {
		t.Fatal("request/event rollback failed")
	}
}

func TestR102NativeCompetingWriters(t *testing.T) {
	f := r102Start(t)
	f.source("Tôi khiếu nại và chờ hàng", "text")
	locked := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseNow := func() { releaseOnce.Do(func() { close(release) }) }
	var workers sync.WaitGroup
	defer r102ReleaseAndAwait(t, releaseNow, &workers)
	barrierErr := make(chan error, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		barrierErr <- db.DB.Transaction(func(tx *gorm.DB) error {
			var channel models.Channel
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", f.channel).Take(&channel).Error; err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	select {
	case <-locked:
	case <-time.After(10 * time.Second):
		t.Fatal("barrier")
	}
	result := make(chan int, 2)
	for _, actor := range []string{f.writer, f.owner} {
		workers.Add(1)
		go func(actor string) {
			defer workers.Done()
			result <- f.send("POST", f.tenant, actor, "/data-revisions", f.revisionBody(1)).Code
		}(actor)
	}
	select {
	case <-result:
		t.Fatal("writer bypassed channel barrier")
	case <-time.After(150 * time.Millisecond):
	}
	releaseNow()
	select {
	case err := <-barrierErr:
		if err != nil {
			t.Fatal("barrier transaction: ", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("barrier completion timeout")
	}
	a, b := r102Receive(t, result), r102Receive(t, result)
	if !((a == 201 && b == 409) || (a == 409 && b == 201)) {
		t.Fatalf("competing results %d %d", a, b)
	}
	var count int64
	r102DB(t, db.DB.Model(&models.InterventionDataRevision{}).Where("case_id = ?", f.caseID).Count(&count).Error)
	if count != 1 {
		t.Fatal("duplicate revision")
	}
}

func TestR102NativeBodyAndCursorBounds(t *testing.T) {
	f := r102Start(t)
	f.source("Tôi chờ hàng", "text")
	body := f.revisionBody(1)
	body["reason"] = strings.Repeat("x", 17*1024)
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", body); rec.Code != 400 {
		t.Fatalf("body cap %d", rec.Code)
	}
	if rec := f.send("GET", f.tenant, f.reader, "/data-revisions?cursor=bad", nil); rec.Code != 400 {
		t.Fatalf("cursor %d", rec.Code)
	}
	if rec := f.send("GET", f.tenant, f.reader, "/data-revisions?limit=101", nil); rec.Code != 400 {
		t.Fatalf("limit %d", rec.Code)
	}
}

func TestR102NativeStateAndIdentityRejections(t *testing.T) {
	f := r102Start(t)
	f.source("Tôi khiếu nại và chờ hàng", "text")
	base := f.revisionBody(1)
	for _, mutate := range []func(map[string]any){
		func(v map[string]any) { v["expected_version"] = 0 },
		func(v map[string]any) { v["expected_snapshot_id"] = "bad" },
		func(v map[string]any) { v["expected_snapshot_digest"] = "not-a-digest" },
	} {
		v := f.revisionBody(1)
		mutate(v)
		if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", v); rec.Code != 400 {
			t.Fatalf("malformed input %d", rec.Code)
		}
	}
	base["expected_snapshot_id"] = pkg.NewUUID()
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", base); rec.Code != 409 {
		t.Fatalf("wrong original snapshot %d", rec.Code)
	}
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(2)); rec.Code != 409 {
		t.Fatalf("stale version %d", rec.Code)
	}
	if err := db.DB.Model(&models.Job{}).Where("id = ?", f.job).Update("schedule_type", "cron").Error; err != nil {
		t.Fatal(err)
	}
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1)); rec.Code != 409 {
		t.Fatalf("nonmanual %d", rec.Code)
	}
	if err := db.DB.Model(&models.Job{}).Where("id = ?", f.job).Update("schedule_type", "manual").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Model(&models.Channel{}).Where("id = ?", f.channel).Update("is_demo_fixture", false).Error; err != nil {
		t.Fatal(err)
	}
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1)); rec.Code != 409 {
		t.Fatalf("nonfixture %d", rec.Code)
	}
	if err := db.DB.Model(&models.Channel{}).Where("id = ?", f.channel).Update("is_demo_fixture", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Model(&models.InterventionCase{}).Where("id = ?", f.caseID).Update("work_state", "RESOLVED").Error; err != nil {
		t.Fatal(err)
	}
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1)); rec.Code != 409 {
		t.Fatalf("wrong state %d", rec.Code)
	}
	if err := db.DB.Model(&models.InterventionCase{}).Where("id = ?", f.caseID).Update("work_state", "WAITING_ON_DATA").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Model(&models.AnalysisSnapshot{}).Where("id = ?", f.snap).Update("digest", strings.Repeat("0", 64)).Error; err != nil {
		t.Fatal(err)
	}
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1)); rec.Code != 409 {
		t.Fatalf("corrupt provenance %d", rec.Code)
	}
	var count int64
	r102DB(t, db.DB.Model(&models.InterventionDataRevision{}).Where("case_id = ?", f.caseID).Count(&count).Error)
	if count != 0 {
		t.Fatal("denied writes persisted")
	}
}

func TestR102NativeForeignRevisionAnchorAndPermissions(t *testing.T) {
	f := r102Start(t)
	f.source("Ordinary update", "text")
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1)); rec.Code != 201 {
		t.Fatalf("capture %d", rec.Code)
	}
	var r models.InterventionDataRevision
	if db.DB.Where("case_id = ?", f.caseID).Take(&r).Error != nil {
		t.Fatal("R102 revision missing")
	}
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(2, r, "no-anchor")); rec.Code != 409 {
		t.Fatalf("no anchor %d", rec.Code)
	}
	foreign := f.requestBody(2, r, "foreign")
	foreign["data_revision_id"] = pkg.NewUUID()
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", foreign); rec.Code != 404 {
		t.Fatalf("foreign revision %d", rec.Code)
	}
	if rec := f.send("GET", f.tenant, f.reader, "/data-revisions/"+pkg.NewUUID(), nil); rec.Code != 404 {
		t.Fatalf("foreign detail %d", rec.Code)
	}
	if err := db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.writer).Update("permissions", "malformed").Error; err != nil {
		t.Fatal(err)
	}
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(2, r, "bad-member")); rec.Code != 403 {
		t.Fatalf("malformed member %d", rec.Code)
	}
}

func TestR102NativeCompetingRequestKeys(t *testing.T) {
	f := r102Start(t)
	r := f.captured(t)
	// Two different request keys for one revision cannot both win.
	a := f.requestBody(2, r, "race-a")
	b := f.requestBody(2, r, "race-b")
	result := make(chan int, 2)
	go func() { result <- f.send("POST", f.tenant, f.writer, "/rerun-requests", a).Code }()
	go func() { result <- f.send("POST", f.tenant, f.owner, "/rerun-requests", b).Code }()
	x, y := r102Receive(t, result), r102Receive(t, result)
	if !((x == 201 && y == 409) || (x == 409 && y == 201)) {
		t.Fatalf("request race %d %d", x, y)
	}
	var n int64
	r102DB(t, db.DB.Model(&models.InterventionRerunRequest{}).Where("case_id = ?", f.caseID).Count(&n).Error)
	if n != 1 {
		t.Fatal("duplicate intent")
	}
}

func TestR102NativeHumanActionCompetition(t *testing.T) {
	f := r102Start(t)
	f.source("Tôi khiếu nại chờ đơn", "text")
	action := map[string]any{"action": "ASSIGN_DATA", "expected_version": 1, "expected_snapshot_id": f.snap, "expected_snapshot_digest": f.originalDigest, "expected_proposal_id": "", "expected_revision": 0, "reason": "R102 parallel human action", "next_action": "Bổ sung nguồn", "assignee_id": f.owner, "due_at": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)}
	result := make(chan int, 2)
	go func() { result <- f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1)).Code }()
	go func() { result <- f.send("POST", f.tenant, f.owner, "/actions", action).Code }()
	a, b := r102Receive(t, result), r102Receive(t, result)
	accepted := (a == 201 || a == 200) && b == 409 || (b == 201 || b == 200) && a == 409
	if !accepted {
		t.Fatalf("human/revision CAS %d %d", a, b)
	}
	var c models.InterventionCase
	var n int64
	r102DB(t, db.DB.Where("id = ?", f.caseID).Take(&c).Error)
	r102DB(t, db.DB.Model(&models.InterventionHumanEvent{}).Where("case_id = ?", f.caseID).Count(&n).Error)
	if c.Version != 2 || n != 1 {
		t.Fatal("human/revision duplicate mutation")
	}
}

func TestR102NativeMembershipRevocationBarrier(t *testing.T) {
	f := r102Start(t)
	r := f.captured(t)
	locked := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	releaseNow := func() { once.Do(func() { close(release) }) }
	var workers sync.WaitGroup
	defer r102ReleaseAndAwait(t, releaseNow, &workers)
	errCh := make(chan error, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		errCh <- db.DB.Transaction(func(tx *gorm.DB) error {
			var member models.UserTenant
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.writer).Take(&member).Error; err != nil {
				return err
			}
			close(locked)
			<-release
			return tx.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.writer).Update("permissions", `{}`).Error
		})
	}()
	select {
	case <-locked:
	case <-time.After(10 * time.Second):
		t.Fatal("membership barrier")
	}
	result := make(chan int, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		result <- f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(2, r, "revoked-in-flight")).Code
	}()
	select {
	case <-result:
		t.Fatal("request crossed membership lock")
	case <-time.After(150 * time.Millisecond):
	}
	releaseNow()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("membership transaction timeout")
	}
	if code := r102Receive(t, result); code != 403 {
		t.Fatalf("revoked request %d", code)
	}
	var n int64
	r102DB(t, db.DB.Model(&models.InterventionRerunRequest{}).Where("case_id = ?", f.caseID).Count(&n).Error)
	if n != 0 {
		t.Fatal("revoked request persisted")
	}
}

func TestR102NativeSyncWriterBeforeRequest(t *testing.T) {
	f := r102Start(t)
	r := f.captured(t)
	locked := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	releaseNow := func() { once.Do(func() { close(release) }) }
	var workers sync.WaitGroup
	defer r102ReleaseAndAwait(t, releaseNow, &workers)
	errCh := make(chan error, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		errCh <- db.DB.Transaction(func(tx *gorm.DB) error {
			var channel models.Channel
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", f.channel).Take(&channel).Error; err != nil {
				return err
			}
			close(locked)
			<-release
			now := time.Now().UTC().Truncate(time.Millisecond)
			id := pkg.NewUUID()
			return tx.Create(&models.Message{ID: id, TenantID: f.tenant, ConversationID: f.conv, ExternalMessageID: id, SenderType: "customer", Content: "Nguồn mới đến", ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: now, CreatedAt: now}).Error
		})
	}()
	select {
	case <-locked:
	case <-time.After(10 * time.Second):
		t.Fatal("sync barrier")
	}
	result := make(chan int, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		result <- f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(2, r, "sync-before-request")).Code
	}()
	select {
	case <-result:
		t.Fatal("request crossed source writer lock")
	case <-time.After(150 * time.Millisecond):
	}
	releaseNow()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("source writer transaction timeout")
	}
	if code := r102Receive(t, result); code != 409 {
		t.Fatalf("changed source request %d", code)
	}
}

func TestR102NativePaginationAndMessageCap(t *testing.T) {
	f := r102Start(t)
	f.source("Tôi khiếu nại", "text")
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1)); rec.Code != 201 {
		t.Fatalf("first %d", rec.Code)
	}
	f.source("Tôi vẫn chờ", "text")
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(2)); rec.Code != 201 {
		t.Fatalf("second %d", rec.Code)
	}
	first := f.send("GET", f.tenant, f.reader, "/data-revisions?limit=1", nil)
	var page struct {
		Items   []map[string]any `json:"items"`
		HasMore bool             `json:"has_more"`
		Cursor  string           `json:"next_cursor"`
	}
	if first.Code != 200 || json.Unmarshal(first.Body.Bytes(), &page) != nil || len(page.Items) != 1 || !page.HasMore || page.Cursor == "" {
		t.Fatalf("first page %d %s", first.Code, first.Body.String())
	}
	second := f.send("GET", f.tenant, f.reader, "/data-revisions?limit=1&cursor="+page.Cursor, nil)
	if second.Code != 200 || json.Unmarshal(second.Body.Bytes(), &page) != nil || len(page.Items) != 1 || page.HasMore {
		t.Fatalf("second page %d %s", second.Code, second.Body.String())
	}
	// The bounded SELECT must detect row 1001 and reject the observation.
	now := time.Now().UTC().Truncate(time.Millisecond)
	rows := make([]models.Message, 999)
	for i := range rows {
		id := pkg.NewUUID()
		rows[i] = models.Message{ID: id, TenantID: f.tenant, ConversationID: f.conv, ExternalMessageID: id, SenderType: "customer", Content: "synthetic", ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: now, CreatedAt: now}
	}
	if err := db.DB.CreateInBatches(&rows, 100).Error; err != nil {
		t.Fatal(err)
	}
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(3)); rec.Code != 503 {
		t.Fatalf("message cap %d", rec.Code)
	}
}

func TestR102NativeLegacyEventSurvivesMigration(t *testing.T) {
	f := r102Start(t)
	legacy := models.InterventionHumanEvent{ID: pkg.NewUUID(), TenantID: f.tenant, CaseID: f.caseID, CaseVersion: 1, ActorID: f.owner, Action: "ASSIGN_DATA", Reason: "legacy synthetic event", SnapshotID: f.snap, SnapshotDigest: f.originalDigest, SourceStatus: "INCOMPLETE", Before: `{}`, After: `{"owner":"set"}`, CreatedAt: time.Now().UTC()}
	r102Must(t, &legacy)
	// Simulate the pre-R102 table shape only inside the exclusively owned DB.
	if err := db.DB.Migrator().DropColumn(&models.InterventionHumanEvent{}, "DataRevisionID"); err != nil {
		t.Fatal("drop synthetic new column: ", err)
	}
	if err := db.DB.Migrator().DropColumn(&models.InterventionHumanEvent{}, "RerunRequestID"); err != nil {
		t.Fatal("drop synthetic new column: ", err)
	}
	if db.DB.Migrator().HasColumn(&models.InterventionHumanEvent{}, "DataRevisionID") || db.DB.Migrator().HasColumn(&models.InterventionHumanEvent{}, "RerunRequestID") {
		t.Fatal("legacy schema was not simulated")
	}
	for i := 0; i < 2; i++ {
		if err := db.AutoMigrate(); err != nil {
			t.Fatal("repeat migration: ", err)
		}
	}
	var actual models.InterventionHumanEvent
	if db.DB.Where("id = ?", legacy.ID).Take(&actual).Error != nil || actual.Before != legacy.Before || actual.After != legacy.After || actual.DataRevisionID != nil || actual.RerunRequestID != nil {
		t.Fatal("legacy event changed by migration")
	}
}

func TestR102NativeSilentSQLSuccessAndFailure(t *testing.T) {
	f := r102Start(t)
	id := f.source("Tôi khiếu nại và chờ hàng", "text")
	r102DB(t, db.DB.Model(&models.Message{}).Where("id = ?", id).Update("sender_name", "R102_MANIFEST_SENTINEL").Error)
	var sink bytes.Buffer
	previous := db.DB.Logger
	infoLogger := logger.New(log.New(&sink, "", 0), logger.Config{LogLevel: logger.Info})
	restore := func() { db.DB.Logger = previous }
	t.Cleanup(restore)
	db.DB.Logger = infoLogger
	body := f.revisionBody(1)
	body["reason"] = "R102_REASON_SENTINEL"
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", body); rec.Code != 201 {
		t.Fatalf("logged success %d", rec.Code)
	}
	if strings.Contains(sink.String(), "R102_REASON_SENTINEL") || strings.Contains(sink.String(), "R102_MANIFEST_SENTINEL") {
		t.Fatal("human reason or manifest leaked to SQL logger on success")
	}
	restore()
	f.source("Nguồn bổ sung", "text")
	r102Must(t, &models.InterventionHumanEvent{ID: pkg.NewUUID(), TenantID: f.tenant, CaseID: f.caseID, CaseVersion: 3, ActorID: f.owner, Action: "SYNTHETIC_CONFLICT", Reason: "forced failure", SnapshotID: f.snap, SnapshotDigest: f.originalDigest, SourceStatus: "UNAVAILABLE", Before: `{}`, After: `{}`, CreatedAt: time.Now().UTC()})
	sink.Reset()
	db.DB.Logger = infoLogger
	body = f.revisionBody(2)
	body["reason"] = "R102_REASON_SENTINEL"
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", body); rec.Code != 503 {
		t.Fatalf("logged failure %d", rec.Code)
	}
	if strings.Contains(sink.String(), "R102_REASON_SENTINEL") || strings.Contains(sink.String(), "R102_MANIFEST_SENTINEL") {
		t.Fatal("human reason or manifest leaked to SQL logger on failure")
	}
}

func TestR102NativeActionBeforeSourceAndRevocation(t *testing.T) {
	f := r102Start(t)
	r := f.captured(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	releaseNow := func() { once.Do(func() { close(release) }) }
	var workers sync.WaitGroup
	defer r102ReleaseAndAwait(t, releaseNow, &workers)
	remove := r102CreateCallback(t, "r102_action_before_writer", func(tx *gorm.DB) {
		event, ok := tx.Statement.Dest.(*models.InterventionHumanEvent)
		if ok && event.Action == "REQUEST_RERUN" {
			close(entered)
			<-release
		}
	})
	action := make(chan int, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		action <- f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(2, r, "action-first")).Code
	}()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("action event barrier")
	}
	writer := make(chan error, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		writer <- db.DB.Transaction(func(tx *gorm.DB) error {
			var channel models.Channel
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", f.channel).Take(&channel).Error; err != nil {
				return err
			}
			now := time.Now().UTC().Truncate(time.Millisecond)
			id := pkg.NewUUID()
			return tx.Create(&models.Message{ID: id, TenantID: f.tenant, ConversationID: f.conv, ExternalMessageID: id, SenderType: "customer", Content: "writer after intent", ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: now, CreatedAt: now}).Error
		})
	}()
	revoked := make(chan error, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		revoked <- db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.writer).Update("permissions", `{}`).Error
	}()
	select {
	case <-writer:
		t.Fatal("source writer crossed action lock")
	case <-time.After(150 * time.Millisecond):
	}
	select {
	case <-revoked:
		t.Fatal("membership revocation crossed action lock")
	case <-time.After(150 * time.Millisecond):
	}
	releaseNow()
	if code := r102Receive(t, action); code != 201 {
		t.Fatalf("action-first request %d", code)
	}
	for name, ch := range map[string]<-chan error{"source writer": writer, "revocation": revoked} {
		select {
		case err := <-ch:
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("%s completion timeout", name)
		}
	}
	remove()
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(2, r, "action-first")); rec.Code != 403 {
		t.Fatalf("revoked replay %d", rec.Code)
	}
	if rec := f.send("POST", f.tenant, f.owner, "/rerun-requests", f.requestBody(3, r, "new-after-source")); rec.Code != 409 {
		t.Fatalf("new request after writer %d", rec.Code)
	}
}

func r102AssertMutationRows(t *testing.T, f *r102Fixture, version int, revisions, requests, events int64) {
	t.Helper()
	var c models.InterventionCase
	r102DB(t, db.DB.Where("id = ?", f.caseID).Take(&c).Error)
	if c.Version != version || c.SnapshotID == nil || *c.SnapshotID != f.snap || c.SnapshotDigest != f.originalDigest {
		t.Fatal("case projection changed after failed mutation")
	}
	for _, item := range []struct {
		model any
		want  int64
	}{{&models.InterventionDataRevision{}, revisions}, {&models.InterventionRerunRequest{}, requests}, {&models.InterventionHumanEvent{}, events}} {
		var got int64
		r102DB(t, db.DB.Model(item.model).Where("case_id = ?", f.caseID).Count(&got).Error)
		if got != item.want {
			t.Fatalf("failed mutation row count got %d want %d", got, item.want)
		}
	}
}

func TestR102NativeInsertAndCASRollbackBranches(t *testing.T) {
	t.Run("revision_insert", func(t *testing.T) {
		f := r102Start(t)
		f.source("Tôi khiếu nại chờ hàng", "text")
		remove := r102CreateCallback(t, "r102_fail_revision_insert", func(tx *gorm.DB) {
			if _, ok := tx.Statement.Dest.(*models.InterventionDataRevision); ok {
				tx.AddError(errors.New("synthetic revision insert failure"))
			}
		})
		if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1)); rec.Code != 503 {
			t.Fatalf("revision insert failure %d", rec.Code)
		}
		remove()
		r102AssertMutationRows(t, f, 1, 0, 0, 0)
	})
	t.Run("request_insert", func(t *testing.T) {
		f := r102Start(t)
		r := f.captured(t)
		remove := r102CreateCallback(t, "r102_fail_request_insert", func(tx *gorm.DB) {
			if _, ok := tx.Statement.Dest.(*models.InterventionRerunRequest); ok {
				tx.AddError(errors.New("synthetic request insert failure"))
			}
		})
		if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(2, r, "fail-insert")); rec.Code != 503 {
			t.Fatalf("request insert failure %d", rec.Code)
		}
		remove()
		r102AssertMutationRows(t, f, 2, 1, 0, 1)
	})
	t.Run("revision_cas_zero_rows", func(t *testing.T) {
		f := r102Start(t)
		f.source("Tôi khiếu nại chờ hàng", "text")
		remove := r102UpdateCallback(t, "r102_zero_cas_revision", func(tx *gorm.DB) {
			if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "intervention_cases" {
				tx.Statement.AddClause(clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "1 = 0"}}})
			}
		})
		if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1)); rec.Code != 409 {
			t.Fatalf("CAS zero revision %d", rec.Code)
		}
		remove()
		r102AssertMutationRows(t, f, 1, 0, 0, 0)
	})
	t.Run("request_cas_zero_rows", func(t *testing.T) {
		f := r102Start(t)
		r := f.captured(t)
		remove := r102UpdateCallback(t, "r102_zero_cas_request", func(tx *gorm.DB) {
			if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "intervention_cases" {
				tx.Statement.AddClause(clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "1 = 0"}}})
			}
		})
		if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(2, r, "fail-cas")); rec.Code != 409 {
			t.Fatalf("CAS zero request %d", rec.Code)
		}
		remove()
		r102AssertMutationRows(t, f, 2, 1, 0, 1)
	})
}
