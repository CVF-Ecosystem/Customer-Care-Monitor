package handlers_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm"
)

// No R116 native test can silently run against an ambient application DSN.
func r116Fixture(t *testing.T) *r102Fixture {
	t.Helper()
	if os.Getenv("CCMAI_R116_NATIVE") != "1" {
		t.Skip("NOT_RUN: CCMAI_R116_NATIVE=1 required")
	}
	dsn := os.Getenv("CCMAI_R116_MYSQL_DSN")
	if dsn == "" {
		t.Fatal("R116 native opt-in requires owned CCMAI_R116_MYSQL_DSN")
	}
	f := r102StartWithDSN(t, dsn, "r116_fixture")
	now := time.Now().UTC().Truncate(time.Millisecond)
	r102Must(t,
		&models.JobRun{ID: f.run, TenantID: f.tenant, JobID: f.job, StartedAt: now, FinishedAt: &now, Status: "error", Summary: `{}`, CreatedAt: now},
		&models.InterventionReceipt{ID: pkg.NewUUID(), TenantID: f.tenant, CaseID: f.caseID, RunID: f.run, Code: "WAIT_DATA", UsageState: "UNKNOWN", CreatedAt: now},
	)
	// The inherited R102 helper seeds only a new R116 tenant here. Remove every
	// new child row before its own cleanup, leaving old fixture history alone.
	t.Cleanup(func() {
		if os.Getenv("CCMAI_R116_LIVE") == "1" && os.Getenv("CCMAI_R116_PRESERVE") == "1" {
			return
		}
		for _, model := range []any{&models.InterventionReceipt{}, &models.InterventionProposal{}, &models.InterventionAttempt{}, &models.InterventionExecution{}, &models.InterventionExecutionGrant{}, &models.InterventionBudget{}, &models.JobRun{}} {
			if err := db.DB.Unscoped().Where("tenant_id = ?", f.tenant).Delete(model).Error; err != nil {
				t.Error("R116 cleanup: ", err)
			}
		}
	})
	return f
}

type r116OriginalHistory struct {
	Run models.JobRun
	Snapshot models.AnalysisSnapshot
	Receipt models.InterventionReceipt
	Revision models.InterventionDataRevision
	Request models.InterventionRerunRequest
	Events []models.InterventionHumanEvent
}

func r116History(t *testing.T, f *r102Fixture, s r116Seed) r116OriginalHistory {
	t.Helper()
	var h r116OriginalHistory
	r102DB(t, db.DB.Where("tenant_id = ? AND id = ?", f.tenant, f.run).Take(&h.Run).Error)
	r102DB(t, db.DB.Where("tenant_id = ? AND id = ?", f.tenant, f.snap).Take(&h.Snapshot).Error)
	r102DB(t, db.DB.Where("tenant_id = ? AND run_id = ?", f.tenant, f.run).Take(&h.Receipt).Error)
	r102DB(t, db.DB.Where("tenant_id = ? AND id = ?", f.tenant, s.request.DataRevisionID).Take(&h.Revision).Error)
	r102DB(t, db.DB.Where("tenant_id = ? AND id = ?", f.tenant, s.request.ID).Take(&h.Request).Error)
	r102DB(t, db.DB.Where("tenant_id = ? AND case_id = ?", f.tenant, f.caseID).Order("case_version, id").Find(&h.Events).Error)
	return h
}

func r116AssertHistory(t *testing.T, f *r102Fixture, s r116Seed, before r116OriginalHistory) {
	t.Helper()
	after := r116History(t, f, s)
	// Publication may append events, but it may never rewrite prior history.
	if len(after.Events) < len(before.Events) || !reflect.DeepEqual(after.Run, before.Run) || !reflect.DeepEqual(after.Snapshot, before.Snapshot) || !reflect.DeepEqual(after.Receipt, before.Receipt) || !reflect.DeepEqual(after.Revision, before.Revision) || !reflect.DeepEqual(after.Request, before.Request) || !reflect.DeepEqual(after.Events[:len(before.Events)], before.Events) {
		t.Fatal("R116 original run/snapshot/receipt/revision/request/event history changed")
	}
}

type r116Transport struct {
	mu        sync.Mutex
	calls     int
	messageID string
	mode      string
	started   chan struct{}
	release   chan struct{}
}

var r116Digest = regexp.MustCompile(`snapshot_digest ([0-9a-f]{64})`)
var r116Conv = regexp.MustCompile(`conversation_id ([0-9a-f-]{36})`)

func (p *r116Transport) AnalyzeChatOnce(ctx context.Context, prompt, transcript string, maxBytes int64) (ai.AIResponse, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	if p.started != nil {
		select {
		case p.started <- struct{}{}:
		default:
		}
	}
	if p.release != nil {
		select {
		case <-p.release:
		case <-ctx.Done():
			return ai.AIResponse{}, ctx.Err()
		}
	}
	if p.mode == "transport_error" {
		return ai.AIResponse{}, errors.New("synthetic transport uncertainty")
	}
	one, two := int64(20), int64(12)
	response := ai.AIResponse{Provider: "alibaba_maas", Model: "qwen3.7-flash", UsagePresence: &ai.AIUsagePresence{SchemaVersion: "1", Source: ai.AIUsagePresenceSourceOpenAIWire, InputTokens: ai.AIUsageCount{Status: ai.AIUsagePresenceKnown, Value: &one}, OutputTokens: ai.AIUsageCount{Status: ai.AIUsagePresenceKnown, Value: &two}, Complete: true}}
	digest, conv := r116Digest.FindStringSubmatch(prompt), r116Conv.FindStringSubmatch(prompt)
	if len(digest) != 2 || len(conv) != 2 || !strings.Contains(transcript, "khiếu nại") || maxBytes > 6000 {
		return ai.AIResponse{}, errors.New("synthetic input mismatch")
	}
	result := map[string]any{"schema": "ccma.intervention.v1", "conversation_id": conv[1], "snapshot_digest": digest[1], "kind": "complaint", "explanation": "Khách khiếu nại đơn hàng", "next_action": "Liên hệ khách", "evidence_refs": []map[string]string{{"message_id": p.messageID, "quote": "khiếu nại"}}}
	raw, _ := json.Marshal(result)
	response.Content = string(raw)
	if p.mode == "invalid" {
		response.Content = `{"invalid":true}`
	}
	if p.mode == "wrong_model" {
		response.Model = "other-model"
	}
	return response, nil
}

type r116Seed struct {
	f         *r102Fixture
	request   models.InterventionRerunRequest
	grant     models.InterventionExecutionGrant
	transport *r116Transport
}

func r116SeedExecution(t *testing.T, f *r102Fixture, p *r116Transport) r116Seed {
	t.Helper()
	p.messageID = f.source("Tôi khiếu nại và chờ đơn hàng", "text")
	if rec := f.send("POST", f.tenant, f.writer, "/data-revisions", f.revisionBody(1)); rec.Code != 201 {
		t.Fatalf("revision %d %s", rec.Code, rec.Body.String())
	}
	var revision models.InterventionDataRevision
	r102DB(t, db.DB.Where("tenant_id = ? AND case_id = ?", f.tenant, f.caseID).Take(&revision).Error)
	if rec := f.send("POST", f.tenant, f.writer, "/rerun-requests", f.requestBody(2, revision, "r116-new")); rec.Code != 201 {
		t.Fatalf("request %d %s", rec.Code, rec.Body.String())
	}
	var request models.InterventionRerunRequest
	r102DB(t, db.DB.Where("tenant_id = ? AND case_id = ?", f.tenant, f.caseID).Take(&request).Error)
	grant := models.InterventionExecutionGrant{ID: pkg.NewUUID(), TenantID: f.tenant, JobID: f.job, CaseID: f.caseID, RequestID: request.ID, DataRevisionID: revision.ID, ActorID: f.writer, Provider: "alibaba_maas", Model: "qwen3.7-flash", PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "r116-synthetic-v1", ConfigRef: "r116-owned-config", IssuerID: f.owner, EvidenceRef: "r116-synthetic-grant", Deadline: time.Now().UTC().Add(5 * time.Minute), State: "ISSUED", CreatedAt: time.Now().UTC()}
	r102Must(t, &grant, &models.InterventionBudget{ID: pkg.NewUUID(), TenantID: f.tenant, JobID: f.job, Cap: 1, Consumed: 0, CreatedAt: time.Now().UTC()})
	engine.WaitDataExecutor = &engine.InterventionService{Transport: p, Provider: grant.Provider, Model: grant.Model, ConfigRef: grant.ConfigRef, PolicyVersion: grant.PolicyVersion, PromptVersion: grant.PromptVersion, Prompt: "Analyze only fabricated Vietnamese complaint. Return the required JSON.", MaxRequestBytes: 6000, Timeout: 5 * time.Second}
	t.Cleanup(func() { engine.WaitDataExecutor = nil })
	return r116Seed{f: f, request: request, grant: grant, transport: p}
}

func (s r116Seed) execute(actor string, version int, grantID string) int {
	return s.f.send("POST", s.f.tenant, actor, "/rerun-requests/"+s.request.ID+"/execute", map[string]any{"expected_case_version": version, "grant_id": grantID}).Code
}

func r116Await(t *testing.T, id string) models.InterventionExecution {
	return r116AwaitFor(t, id, 8*time.Second)
}

func r116AwaitFor(t *testing.T, id string, limit time.Duration) models.InterventionExecution {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		var row models.InterventionExecution
		if db.DB.Where("id = ?", id).Take(&row).Error == nil && row.TerminalCode != "" {
			return row
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("R116 execution did not reach a terminal row")
	return models.InterventionExecution{}
}

func r116Delete(f *r102Fixture, path string) int {
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tenants/"+f.tenant+path, nil)
	req.Header.Set("Authorization", "Bearer "+f.token[f.owner])
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec.Code
}

func r116RawExecute(f *r102Fixture, actor, caseID, requestID, body string) int {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/"+f.tenant+"/intervention-cases/"+caseID+"/rerun-requests/"+requestID+"/execute", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.token[actor])
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec.Code
}

func TestR116WaitDataNativeA3A4InvalidAdmission(t *testing.T) {
	f := r116Fixture(t)
	p := &r116Transport{}
	s := r116SeedExecution(t, f, p)
	before := r116History(t, f, s)
	valid := `{"expected_case_version":3,"grant_id":"` + s.grant.ID + `"}`
	for _, body := range []string{
		`{"expected_case_version":3,"expected_case_version":3,"grant_id":"` + s.grant.ID + `"}`,
		`{"expected_case_version":3}`,
		`{"expected_case_version":3,"grant_id":"` + s.grant.ID + `","extra":1}`,
		valid + ` {}`,
		strings.Repeat(" ", 513) + valid,
	} {
		if got := r116RawExecute(f, f.writer, f.caseID, s.request.ID, body); got != 400 {
			t.Fatalf("invalid execute body status %d", got)
		}
	}
	if got := s.execute(f.reader, 3, s.grant.ID); got != 403 {
		t.Fatalf("reader permission %d", got)
	}
	if got := s.execute(f.writer, 3, pkg.NewUUID()); got != 409 {
		t.Fatalf("missing grant %d", got)
	}
	deadline := time.Now().UTC().Add(-time.Minute)
	r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", s.grant.ID).Update("deadline", deadline).Error)
	if got := s.execute(f.writer, 3, s.grant.ID); got != 409 {
		t.Fatalf("expired grant %d", got)
	}
	r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", s.grant.ID).Updates(map[string]any{"deadline": time.Now().UTC().Add(5 * time.Minute), "actor_id": f.owner}).Error)
	if got := s.execute(f.writer, 3, s.grant.ID); got != 409 {
		t.Fatalf("actor-mismatched grant %d", got)
	}
	r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", s.grant.ID).Update("actor_id", f.writer).Error)
	var claims, attempts int64
	r102DB(t, db.DB.Model(&models.InterventionExecution{}).Where("tenant_id = ?", f.tenant).Count(&claims).Error)
	r102DB(t, db.DB.Model(&models.InterventionAttempt{}).Where("tenant_id = ?", f.tenant).Count(&attempts).Error)
	if claims != 0 || attempts != 0 || p.calls != 0 {
		t.Fatal("invalid admissions had effects")
	}
	r116AssertHistory(t, f, s, before)
	if got := r116RawExecute(f, f.writer, strings.ToUpper(f.caseID), strings.ToUpper(s.request.ID), `{"expected_case_version":3,"grant_id":"`+strings.ToUpper(s.grant.ID)+`"}`); got != 202 {
		t.Fatalf("canonical UUID-case execute %d", got)
	}
	var claim models.InterventionExecution
	r102DB(t, db.DB.Where("tenant_id = ? AND request_id = ?", f.tenant, s.request.ID).Take(&claim).Error)
	if row := r116Await(t, claim.ID); row.State != "PUBLISHED" {
		t.Fatalf("canonical UUID-case publication %s", row.State)
	}
}

func TestR116WaitDataNativeA1A2A8(t *testing.T) {
	f := r116Fixture(t)
	p := &r116Transport{}
	s := r116SeedExecution(t, f, p)
	originalHistory := r116History(t, f, s)
	if got := s.execute(f.noJobs, 3, s.grant.ID); got != 403 {
		t.Fatalf("A3 jobs permission got %d", got)
	}
	if got := s.execute(f.writer, 2, s.grant.ID); got != 409 {
		t.Fatalf("A2 stale first claim got %d", got)
	}
	if got := s.execute(f.writer, 3, s.grant.ID); got != 202 {
		t.Fatalf("A1 claim got %d", got)
	}
	var claim models.InterventionExecution
	r102DB(t, db.DB.Where("tenant_id = ? AND request_id = ?", f.tenant, s.request.ID).Take(&claim).Error)
	terminal := r116Await(t, claim.ID)
	if terminal.State != "PUBLISHED" || terminal.TerminalCode != "PROPOSAL_VALID" {
		t.Fatalf("A1 terminal %s %s", terminal.State, terminal.TerminalCode)
	}
	if got := s.execute(f.writer, 3, s.grant.ID); got != 200 {
		t.Fatalf("A2 replay got %d", got)
	}
	if got := s.execute(f.writer, 4, s.grant.ID); got != 409 {
		t.Fatalf("A2 changed body got %d", got)
	}
	p.mu.Lock()
	calls := p.calls
	p.mu.Unlock()
	if calls != 1 {
		t.Fatalf("A1 physical double call %d", calls)
	}
	var c models.InterventionCase
	r102DB(t, db.DB.Where("id = ?", f.caseID).Take(&c).Error)
	if c.ID != f.caseID || c.Version != 4 || c.ActiveProposalID == nil || c.ActiveExecutionID == nil || c.RunID != claim.RunID || c.SnapshotID == nil || *c.SnapshotID != claim.SnapshotID {
		t.Fatal("A1 same-case active projection")
	}
	var runs, snaps, attempts, proposals, receipts int64
	r102DB(t, db.DB.Model(&models.JobRun{}).Where("tenant_id = ? AND id = ?", f.tenant, claim.RunID).Count(&runs).Error)
	r102DB(t, db.DB.Model(&models.AnalysisSnapshot{}).Where("tenant_id = ? AND job_run_id = ?", f.tenant, claim.RunID).Count(&snaps).Error)
	r102DB(t, db.DB.Model(&models.InterventionAttempt{}).Where("tenant_id = ? AND run_id = ?", f.tenant, claim.RunID).Count(&attempts).Error)
	r102DB(t, db.DB.Model(&models.InterventionProposal{}).Where("tenant_id = ? AND run_id = ?", f.tenant, claim.RunID).Count(&proposals).Error)
	r102DB(t, db.DB.Model(&models.InterventionReceipt{}).Where("tenant_id = ? AND run_id = ?", f.tenant, claim.RunID).Count(&receipts).Error)
	if runs != 1 || snaps != 1 || attempts != 1 || proposals != 1 || receipts != 1 {
		t.Fatalf("A1 row counts %d %d %d %d %d", runs, snaps, attempts, proposals, receipts)
	}
	var budget models.InterventionBudget
	r102DB(t, db.DB.Where("tenant_id = ? AND job_id = ?", f.tenant, f.job).Take(&budget).Error)
	if budget.Consumed != 1 {
		t.Fatal("A1 debit")
	}
	r116AssertHistory(t, f, s, originalHistory)
	// A8 exercises supported destructive routes with a terminal linked claim.
	for _, path := range []string{"/jobs/" + f.job, "/jobs/" + f.job + "/runs", "/jobs/" + f.job + "/results", "/channels/" + f.channel, "/channels/" + f.channel + "/conversations"} {
		if got := r116Delete(f, path); got != 409 {
			t.Fatalf("A8 %s got %d", path, got)
		}
	}
	r102DB(t, db.DB.Model(&models.Tenant{}).Where("id = ?", f.tenant).Update("settings", `{"is_demo_data":true}`).Error)
	if got := r116Delete(f, "/demo/reset"); got != 409 {
		t.Fatalf("A8 demo reset got %d", got)
	}
	r102DB(t, db.DB.Model(&models.AnalysisSnapshot{}).Where("tenant_id = ? AND id = ?", f.tenant, f.snap).Count(&snaps).Error)
	var original models.AnalysisSnapshot
	r102DB(t, db.DB.Where("tenant_id = ? AND id = ?", f.tenant, f.snap).Take(&original).Error)
	if original.Digest != f.originalDigest {
		t.Fatal("A8 original snapshot changed")
	}
	// A7: a new execution can never use the legacy NULL-pointer fallback.
	r102DB(t, db.DB.Model(&models.InterventionCase{}).Where("id = ?", f.caseID).Update("active_proposal_id", nil).Error)
	_, actionErr := engine.M3ApplyAction(context.Background(), f.tenant, f.writer, f.caseID, engine.M3Action{Action: "CONFIRM", ExpectedVersion: 4, ExpectedSnapshotID: claim.SnapshotID, ExpectedSnapshotDigest: claim.SnapshotDigest, ExpectedProposalID: *c.ActiveProposalID, ExpectedRevision: 4, Reason: "Synthetic reviewer decision"})
	if !errors.Is(actionErr, engine.ErrM3Conflict) {
		t.Fatalf("A7 new execution legacy fallback: %v", actionErr)
	}
}

func TestR116WaitDataNativeA5(t *testing.T) {
	for _, mode := range []string{"invalid", "wrong_model", "transport_error"} {
		t.Run(mode, func(t *testing.T) {
			f := r116Fixture(t)
			p := &r116Transport{mode: mode}
			s := r116SeedExecution(t, f, p)
			if got := s.execute(f.writer, 3, s.grant.ID); got != 202 {
				t.Fatalf("claim %d", got)
			}
			var claim models.InterventionExecution
			r102DB(t, db.DB.Where("tenant_id = ? AND request_id = ?", f.tenant, s.request.ID).Take(&claim).Error)
			terminal := r116Await(t, claim.ID)
			want := "OUTPUT_INVALID"
			if mode == "transport_error" {
				want = "PROVIDER_UNKNOWN"
			}
			if terminal.TerminalCode != want {
				t.Fatalf("%s terminal %s", mode, terminal.TerminalCode)
			}
			var n int64
			r102DB(t, db.DB.Model(&models.InterventionProposal{}).Where("tenant_id = ?", f.tenant).Count(&n).Error)
			if n != 0 {
				t.Fatal("invalid response published")
			}
			var budget models.InterventionBudget
			r102DB(t, db.DB.Where("tenant_id = ?", f.tenant).Take(&budget).Error)
			if budget.Consumed != 1 {
				t.Fatal("consumed call refunded or omitted")
			}
		})
	}
}

func TestR116WaitDataNativeA3A4(t *testing.T) {
	for _, change := range []string{"revoke", "source"} {
		t.Run(change, func(t *testing.T) {
			f := r116Fixture(t)
			p := &r116Transport{started: make(chan struct{}, 1), release: make(chan struct{})}
			s := r116SeedExecution(t, f, p)
			if got := s.execute(f.writer, 3, s.grant.ID); got != 202 {
				t.Fatalf("claim %d", got)
			}
			select {
			case <-p.started:
			case <-time.After(5 * time.Second):
				t.Fatal("call marker not reached")
			}
			var claim models.InterventionExecution
			r102DB(t, db.DB.Where("tenant_id = ? AND request_id = ?", f.tenant, s.request.ID).Take(&claim).Error)
			var marked models.InterventionExecution
			r102DB(t, db.DB.Where("id = ?", claim.ID).Take(&marked).Error)
			if marked.State != "CALL_MAY_HAVE_STARTED" {
				t.Fatalf("marker %s", marked.State)
			}
			var originalRun models.JobRun
			var originalSnapshot models.AnalysisSnapshot
			r102DB(t, db.DB.Where("id = ?", f.run).Take(&originalRun).Error)
			r102DB(t, db.DB.Where("id = ?", f.snap).Take(&originalSnapshot).Error)
			r102DB(t, db.DB.Model(&models.Tenant{}).Where("id = ?", f.tenant).Update("settings", `{"is_demo_data":true}`).Error)
			paths := []string{"/jobs/" + f.job, "/jobs/" + f.job + "/runs", "/jobs/" + f.job + "/results", "/channels/" + f.channel, "/channels/" + f.channel + "/conversations", "/demo/reset"}
			results := make(chan int, len(paths))
			for _, path := range paths { go func(path string) { results <- r116Delete(f, path) }(path) }
			for range paths {
				select {
				case status := <-results:
					if status != 409 { t.Fatalf("blocked destructive endpoint %d", status) }
				case <-time.After(5 * time.Second): t.Fatal("destructive endpoint held DB lock over transport")
				}
			}
			var afterRun models.JobRun
			var afterSnapshot models.AnalysisSnapshot
			r102DB(t, db.DB.Where("id = ?", f.run).Take(&afterRun).Error)
			r102DB(t, db.DB.Where("id = ?", f.snap).Take(&afterSnapshot).Error)
			if !reflect.DeepEqual(originalRun, afterRun) || !reflect.DeepEqual(originalSnapshot, afterSnapshot) { t.Fatal("destructive writer changed original lineage") }
			if change == "revoke" {
				r102DB(t, db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.writer).Update("permissions", `{"intervention_cases":"r"}`).Error)
			} else {
				f.source("Thông tin mới làm thay đổi hồ sơ", "text")
			}
			close(p.release)
			terminal := r116Await(t, claim.ID)
			if terminal.State != "STALE_SOURCE" {
				t.Fatalf("post-marker change published: %s", terminal.State)
			}
			if got := r116Delete(f, "/jobs/"+f.job+"/runs"); got != 409 {
				t.Fatalf("A8 manual hold deletion %d", got)
			}
			var proposals int64
			r102DB(t, db.DB.Model(&models.InterventionProposal{}).Where("tenant_id = ?", f.tenant).Count(&proposals).Error)
			if proposals != 0 {
				t.Fatal("stale result published")
			}
			var budget models.InterventionBudget
			r102DB(t, db.DB.Where("tenant_id = ?", f.tenant).Take(&budget).Error)
			if budget.Consumed != 1 {
				t.Fatal("marker debit not retained")
			}
		})
	}
}

func TestR116WaitDataNativeA2Concurrent(t *testing.T) {
	f := r116Fixture(t)
	p := &r116Transport{started: make(chan struct{}, 1), release: make(chan struct{})}
	s := r116SeedExecution(t, f, p)
	start := make(chan struct{})
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() { <-start; results <- s.execute(f.writer, 3, s.grant.ID) }()
	}
	close(start)
	first, second := <-results, <-results
	if (first != 202 && first != 409) || (second != 202 && second != 409) || (first != 202 && second != 202) {
		t.Fatalf("A2 same-body concurrent statuses %d %d", first, second)
	}
	select {
	case <-p.started:
	case <-time.After(5 * time.Second):
		t.Fatal("A2 provider not reached")
	}
	var count int64
	r102DB(t, db.DB.Model(&models.InterventionExecution{}).Where("tenant_id = ?", f.tenant).Count(&count).Error)
	if count != 1 {
		t.Fatalf("A2 claims %d", count)
	}
	close(p.release)
	var claim models.InterventionExecution
	r102DB(t, db.DB.Where("tenant_id = ? AND request_id = ?", f.tenant, s.request.ID).Take(&claim).Error)
	terminal := r116Await(t, claim.ID)
	if terminal.State != "PUBLISHED" {
		t.Fatalf("A2 terminal %s", terminal.State)
	}
	if got := s.execute(f.writer, 3, s.grant.ID); got != 200 {
		t.Fatalf("A2 terminal replay %d", got)
	}
	p.mu.Lock()
	calls := p.calls
	p.mu.Unlock()
	if calls != 1 {
		t.Fatalf("A2 calls %d", calls)
	}
}

func TestR116WaitDataNativeA7HumanAction(t *testing.T) {
	f := r116Fixture(t)
	p := &r116Transport{}
	s := r116SeedExecution(t, f, p)
	if got := s.execute(f.writer, 3, s.grant.ID); got != 202 {
		t.Fatalf("claim %d", got)
	}
	var claim models.InterventionExecution
	r102DB(t, db.DB.Where("tenant_id = ? AND request_id = ?", f.tenant, s.request.ID).Take(&claim).Error)
	if row := r116Await(t, claim.ID); row.State != "PUBLISHED" {
		t.Fatalf("state %s", row.State)
	}
	var proposal models.InterventionProposal
	r102DB(t, db.DB.Where("tenant_id = ? AND run_id = ?", f.tenant, claim.RunID).Take(&proposal).Error)
	action := map[string]any{"action": "CONFIRM", "expected_version": 4, "expected_snapshot_id": claim.SnapshotID, "expected_snapshot_digest": claim.SnapshotDigest, "expected_proposal_id": proposal.ID, "expected_revision": proposal.Revision, "reason": "Đã kiểm tra đề xuất"}
	if rec := f.send("POST", f.tenant, f.writer, "/actions", action); rec.Code != 200 {
		t.Fatalf("A7 action %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.send("POST", f.tenant, f.writer, "/actions", action); rec.Code != 409 {
		t.Fatalf("A7 stale action %d", rec.Code)
	}
	var c models.InterventionCase
	r102DB(t, db.DB.Where("tenant_id = ? AND id = ?", f.tenant, f.caseID).Take(&c).Error)
	if c.Version != 5 || c.HumanAssessment != "CONFIRM" || c.ActiveProposalID == nil || *c.ActiveProposalID != proposal.ID {
		t.Fatal("A7 action projection")
	}
}

func TestR116WaitDataNativeA7LegacyUnique(t *testing.T) {
	f := r116Fixture(t)
	revision := f.captured(t)
	// Construct an old complete projection with a NULL active pointer; this
	// fixture is synthetic and contains no execution claim.
	r102DB(t, db.DB.Model(&models.AnalysisSnapshot{}).Where("tenant_id = ? AND id = ?", f.tenant, f.snap).Updates(map[string]any{"manifest": revision.Manifest, "digest": revision.Digest, "coverage": "complete", "coverage_reasons": revision.CoverageReasons, "message_count": revision.MessageCount}).Error)
	r102DB(t, db.DB.Model(&models.InterventionCase{}).Where("tenant_id = ? AND id = ?", f.tenant, f.caseID).Updates(map[string]any{"snapshot_digest": revision.Digest, "eligibility": "ELIGIBLE", "execution": "LLM_SUCCEEDED", "disposition": "PENDING_REVIEW", "work_state": "OPEN"}).Error)
	proposal := models.InterventionProposal{ID: pkg.NewUUID(), TenantID: f.tenant, CaseID: f.caseID, Revision: 2, RunID: f.run, SnapshotID: f.snap, SnapshotDigest: revision.Digest, PromptVersion: "legacy", PolicyVersion: "ccma.synthetic.intervention.v1", SchemaVersion: "ccma.intervention.v1", Kind: "complaint", Explanation: "Khách khiếu nại", NextAction: "Liên hệ khách", Evidence: "[]", Provider: "legacy", Model: "legacy", UsageState: "UNKNOWN", CreatedAt: time.Now().UTC()}
	r102Must(t, &proposal)
	duplicate := proposal
	duplicate.ID = pkg.NewUUID()
	duplicate.Revision = 3
	if err := db.DB.Create(&duplicate).Error; err == nil {
		t.Fatal("legacy same-run second candidate bypassed unique-run constraint")
	}
	action := engine.M3Action{Action: "CONFIRM", ExpectedVersion: 2, ExpectedSnapshotID: f.snap, ExpectedSnapshotDigest: revision.Digest, ExpectedProposalID: proposal.ID, ExpectedRevision: 2, Reason: "Đã kiểm tra hồ sơ cũ"}
	c, err := engine.M3ApplyAction(context.Background(), f.tenant, f.writer, f.caseID, action)
	if err != nil || c.Version != 3 || c.ActiveProposalID == nil || *c.ActiveProposalID != proposal.ID {
		t.Fatalf("A7 legacy unique: %v", err)
	}
}

func TestR116WaitDataNativeA6AdmissionFault(t *testing.T) {
	f := r116Fixture(t)
	p := &r116Transport{}
	s := r116SeedExecution(t, f, p)
	remove := r102CreateCallback(t, "r116:deny_execution_create", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Name == "InterventionExecution" {
			tx.AddError(errors.New("synthetic admission fault"))
		}
	})
	status := s.execute(f.writer, 3, s.grant.ID)
	remove()
	if status != 503 {
		t.Fatalf("A6 admission fault status %d", status)
	}
	var claims, runs, attempts int64
	r102DB(t, db.DB.Model(&models.InterventionExecution{}).Where("tenant_id = ?", f.tenant).Count(&claims).Error)
	r102DB(t, db.DB.Model(&models.JobRun{}).Where("tenant_id = ?", f.tenant).Count(&runs).Error)
	r102DB(t, db.DB.Model(&models.InterventionAttempt{}).Where("tenant_id = ?", f.tenant).Count(&attempts).Error)
	p.mu.Lock()
	calls := p.calls
	p.mu.Unlock()
	if claims != 0 || runs != 1 || attempts != 0 || calls != 0 {
		t.Fatalf("A6 rollback claims=%d runs=%d attempts=%d calls=%d", claims, runs, attempts, calls)
	}
}

func r116Claim(t *testing.T, s r116Seed) models.InterventionExecution {
	t.Helper()
	if got := s.execute(s.f.writer, 3, s.grant.ID); got != 202 {
		t.Fatalf("claim status %d", got)
	}
	var claim models.InterventionExecution
	r102DB(t, db.DB.Where("tenant_id = ? AND request_id = ?", s.f.tenant, s.request.ID).Take(&claim).Error)
	return claim
}

func r116Count(t *testing.T, f *r102Fixture, model any, query string, args ...any) int64 {
	t.Helper()
	var count int64
	r102DB(t, db.DB.Model(model).Where("tenant_id = ? AND "+query, append([]any{f.tenant}, args...)...).Count(&count).Error)
	return count
}

func TestR116WaitDataNativeA6PreMarkerAndRecoveryWindows(t *testing.T) {
	for _, window := range []string{"rollback", "marker", "response"} {
		t.Run(window, func(t *testing.T) {
			f := r116Fixture(t)
			p := &r116Transport{}
			s := r116SeedExecution(t, f, p)
			before := r116History(t, f, s)
			remove := r102CreateCallback(t, "r116:premarker_"+window, func(tx *gorm.DB) {
				if tx.Statement.Schema != nil && tx.Statement.Schema.Name == "InterventionAttempt" {
					tx.AddError(errors.New("synthetic premarker rollback"))
				}
			})
			claim := r116Claim(t, s)
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) && engine.JobRunActive(f.tenant, f.job) {
				time.Sleep(20 * time.Millisecond)
			}
			remove()
			if engine.JobRunActive(f.tenant, f.job) { t.Fatal("original owner did not exit") }
			var row models.InterventionExecution
			r102DB(t, db.DB.Where("id = ?", claim.ID).Take(&row).Error)
			if row.State != "CLAIMED" || p.calls != 0 || r116Count(t, f, &models.InterventionAttempt{}, "execution_id = ?", claim.ID) != 0 {
				t.Fatal("premarker rollback launched call or debit")
			}
			var budget models.InterventionBudget
			r102DB(t, db.DB.Where("tenant_id = ? AND job_id = ?", f.tenant, f.job).Take(&budget).Error)
			if budget.Consumed != 0 { t.Fatal("premarker debit") }
			if window != "rollback" {
				// Persist a controlled crash-window marker and debit after the owner
				// has exited. This is not a process-kill test.
				r102DB(t, db.DB.Transaction(func(tx *gorm.DB) error {
					if err := tx.Model(&models.InterventionBudget{}).Where("id = ? AND consumed = 0", budget.ID).Update("consumed", 1).Error; err != nil { return err }
					if err := tx.Create(&models.InterventionAttempt{ID: pkg.NewUUID(), ExecutionID: &claim.ID, TenantID: f.tenant, JobID: f.job, RunID: claim.RunID, BudgetID: budget.ID, State: "CONSUMED", UsageState: "UNKNOWN", CreatedAt: time.Now().UTC()}).Error; err != nil { return err }
					state := "CALL_MAY_HAVE_STARTED"
					if window == "response" { state = "RESPONSE_RECORDED" }
					return tx.Model(&models.InterventionExecution{}).Where("id = ? AND state = ?", claim.ID, "CLAIMED").Update("state", state).Error
				}))
				if err := engine.MarkWaitDataRecoveryRequired(context.Background(), f.tenant, claim.ID); err != nil { t.Fatal(err) }
				r102DB(t, db.DB.Where("id = ?", claim.ID).Take(&row).Error)
				if row.State != "RECOVERY_REQUIRED" || p.calls != 0 || r116Count(t, f, &models.InterventionProposal{}, "run_id = ?", claim.RunID) != 0 { t.Fatal("ownerless recovery reconstructed output") }
				wantReceipt := int64(1)
				wantRunStatus := "error"
				if window == "response" { wantReceipt, wantRunStatus = 0, "running" }
				if r116Count(t, f, &models.InterventionReceipt{}, "run_id = ?", claim.RunID) != wantReceipt { t.Fatal("ownerless recovery receipt") }
				var run models.JobRun
				r102DB(t, db.DB.Where("id = ?", claim.RunID).Take(&run).Error)
				if run.Status != wantRunStatus { t.Fatal("ownerless recovery run status") }
				if got := s.execute(f.writer, 3, s.grant.ID); got != 200 { t.Fatalf("ownerless replay %d", got) }
			}
			r116AssertHistory(t, f, s, before)
		})
	}
}

func TestR116WaitDataNativeA6PublicationAndTerminalFaults(t *testing.T) {
	for _, mode := range []string{"publication_once", "publication_hold", "terminal_hold"} {
		t.Run(mode, func(t *testing.T) {
			f := r116Fixture(t)
			p := &r116Transport{}
			if mode == "terminal_hold" { p.mode = "invalid" }
			s := r116SeedExecution(t, f, p)
			var mu sync.Mutex
			injected := 0
			remove := r102CreateCallback(t, "r116:publication_"+mode, func(tx *gorm.DB) {
				if tx.Statement.Schema == nil { return }
				name := tx.Statement.Schema.Name
				if (mode == "terminal_hold" && name == "InterventionReceipt") || (mode != "terminal_hold" && name == "InterventionProposal") {
					mu.Lock(); injected++; n := injected; mu.Unlock()
					if mode != "publication_once" || n == 1 { tx.AddError(errors.New("synthetic durable writer fault")) }
				}
			})
			claim := r116Claim(t, s)
			deadline := time.Now().Add(8 * time.Second)
			for time.Now().Before(deadline) && engine.JobRunActive(f.tenant, f.job) { time.Sleep(20 * time.Millisecond) }
			remove()
			if engine.JobRunActive(f.tenant, f.job) { t.Fatal("owner did not exit") }
			var row models.InterventionExecution
			r102DB(t, db.DB.Where("id = ?", claim.ID).Take(&row).Error)
			if mode == "publication_once" {
				if row.State != "PUBLISHED" || r116Count(t, f, &models.InterventionProposal{}, "run_id = ?", claim.RunID) != 1 { t.Fatal("original owner did not retry DB publication") }
			} else {
				if row.State != "RESPONSE_RECORDED" || row.TerminalCode != "" || r116Count(t, f, &models.InterventionProposal{}, "run_id = ?", claim.RunID) != 0 || r116Count(t, f, &models.InterventionReceipt{}, "run_id = ?", claim.RunID) != 0 { t.Fatal("writer fault produced false terminal outcome") }
				var run models.JobRun
				r102DB(t, db.DB.Where("id = ?", claim.RunID).Take(&run).Error)
				if run.Status != "running" { t.Fatal("failed terminal did not remain blocked") }
			}
			wantReplay := 202
			if row.State == "PUBLISHED" { wantReplay = 200 }
			if got := s.execute(f.writer, 3, s.grant.ID); got != wantReplay || p.calls != 1 { t.Fatalf("fault replay status=%d want=%d calls=%d", got, wantReplay, p.calls) }
		})
	}
}

func TestR116WaitDataNativeA6SettlementFault(t *testing.T) {
	f := r116Fixture(t)
	p := &r116Transport{}
	s := r116SeedExecution(t, f, p)
	var injected sync.Once
	remove := r102UpdateCallback(t, "r116:settlement_failure", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Name == "InterventionAttempt" {
			injected.Do(func() { tx.AddError(errors.New("synthetic settlement write failure")) })
		}
	})
	claim := r116Claim(t, s)
	row := r116Await(t, claim.ID)
	remove()
	if row.TerminalCode != "SETTLEMENT_FAILED" || p.calls != 1 { t.Fatalf("settlement status=%s calls=%d", row.TerminalCode, p.calls) }
	var attempt models.InterventionAttempt
	r102DB(t, db.DB.Where("tenant_id = ? AND execution_id = ?", f.tenant, claim.ID).Take(&attempt).Error)
	if attempt.State != "UNKNOWN" || attempt.UsageState != "UNKNOWN" { t.Fatal("failed settlement fabricated usage") }
	var budget models.InterventionBudget
	r102DB(t, db.DB.Where("tenant_id = ? AND job_id = ?", f.tenant, f.job).Take(&budget).Error)
	if budget.Consumed != 1 || r116Count(t, f, &models.InterventionProposal{}, "run_id = ?", claim.RunID) != 0 { t.Fatal("failed settlement refunded or published") }
}

// This wrapper is armed only by Root's admitted child environment. The
// exclusive effect marker is durable before the one physical RoundTrip.
type r116WireTransport struct {
	next              http.RoundTripper
	key               string
	endpoint          string
	effectDir         string
	mu                sync.Mutex
	attempts          int
	request, response string
	status            int
}

func r116SafeJSON(raw []byte, key string) bool {
	if len(raw) > 1<<20 || bytes.Contains(raw, []byte(key)) {
		return false
	}
	var logical any
	return json.Unmarshal(raw, &logical) == nil && !r100SecretUnsafe(logical, key, 0)
}

func (w *r116WireTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.attempts != 0 || req.Method != http.MethodPost || req.URL.Scheme != "https" || req.URL.String() != strings.TrimSuffix(w.endpoint, "/")+"/chat/completions" || req.Header.Get("Authorization") != "Bearer "+w.key {
		return nil, errors.New("r116_wire_boundary")
	}
	raw, err := io.ReadAll(io.LimitReader(req.Body, 6001))
	if err != nil || len(raw) > 6000 || !r116SafeJSON(raw, w.key) {
		return nil, errors.New("r116_request_rejected")
	}
	var shape struct {
		Model    string `json:"model"`
		Max      int    `json:"max_completion_tokens"`
		Thinking *bool  `json:"enable_thinking"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	var keys map[string]json.RawMessage
	if json.Unmarshal(raw, &shape) != nil || json.Unmarshal(raw, &keys) != nil || len(keys) != 4 || shape.Model != "qwen3.7-flash" || shape.Max != 768 || shape.Thinking == nil || *shape.Thinking || len(shape.Messages) != 2 || shape.Messages[0].Role != "system" || shape.Messages[1].Role != "user" {
		return nil, errors.New("r116_request_shape")
	}
	f, err := os.OpenFile(filepath.Join(w.effectDir, "slot-1.effect"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, errors.New("r116_physical_slot_unavailable")
	}
	marker, _ := json.Marshal(map[string]any{"slot": 1, "request_sha256": func() string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }(), "state": "MAY_HAVE_STARTED"})
	if _, err = f.Write(marker); err != nil {
		f.Close()
		return nil, errors.New("r116_marker_write_failed")
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return nil, errors.New("r116_marker_sync_failed")
	}
	if err = f.Close(); err != nil {
		return nil, errors.New("r116_marker_close_failed")
	}
	w.attempts = 1
	w.request = string(raw)
	req.Body = io.NopCloser(bytes.NewReader(raw))
	response, err := w.next.RoundTrip(req)
	if err != nil {
		return nil, errors.New("r116_transport_uncertain")
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	_ = response.Body.Close()
	if readErr != nil || !r116SafeJSON(body, w.key) {
		return nil, errors.New("r116_response_unsafe_or_unreadable")
	}
	w.status = response.StatusCode
	w.response = string(body)
	response.Body = io.NopCloser(bytes.NewReader(body))
	return response, nil
}

func TestR116WaitDataRealAlibaba(t *testing.T) {
	if os.Getenv("CCMAI_R116_LIVE") != "1" {
		t.Skip("NOT_RUN: Root live admission required")
	}
	f := r116Fixture(t)
	key, endpoint := os.Getenv("CCMAI_R116_ALIBABA_KEY"), os.Getenv("CCMAI_R116_ALIBABA_ENDPOINT")
	effectDir, evidenceDir := os.Getenv("CCMAI_R116_EFFECT_DIR"), os.Getenv("CCMAI_R116_EVIDENCE_DIR")
	admissionSHA, ownerGrant := os.Getenv("CCMAI_R116_ADMISSION_SHA"), os.Getenv("CCMAI_R116_OWNER_GRANT")
	endpointHash := sha256.Sum256([]byte(endpoint))
	if len(key) < 16 || hex.EncodeToString(endpointHash[:]) != "b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f" || effectDir == "" || evidenceDir == "" || ownerGrant != "CCMAI-RUNTIME-117-OWNER-NEW1-20261017" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(admissionSHA) {
		t.Fatal("R116 live admission input missing or mismatched")
	}
	entries, err := os.ReadDir(effectDir)
	if err != nil || len(entries) != 0 {
		t.Fatal("R116 fresh physical ledger required")
	}
	if stat, err := os.Stat(evidenceDir); err != nil || !stat.IsDir() {
		t.Fatal("R116 evidence directory required")
	}
	source := &r116Transport{}
	s := r116SeedExecution(t, f, source)
	originalHistory := r116History(t, f, s)
	// The OpenAI-compatible adapter honestly reports "openai" even when its
	// pinned HTTPS endpoint is Alibaba. Persist the adapter identity as such.
	liveRef := "r116-alibaba-" + hex.EncodeToString(endpointHash[:])
	liveEvidence := ownerGrant + ":" + admissionSHA
	r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ? AND state = ?", s.grant.ID, "ISSUED").Updates(map[string]any{"provider": "openai", "config_ref": liveRef, "prompt_version": "r116-alibaba-v1", "evidence_ref": liveEvidence}).Error)
	engine.WaitDataExecutor.Provider = "openai"
	engine.WaitDataExecutor.ConfigRef = liveRef
	engine.WaitDataExecutor.PromptVersion = "r116-alibaba-v1"
	engine.WaitDataExecutor.Prompt = "Chỉ phân tích hội thoại giả lập này. Trả về đúng một đối tượng JSON gọn, không dùng markdown. Các trường bắt buộc: schema, conversation_id, snapshot_digest, kind, explanation, next_action, evidence_refs. schema phải là ccma.intervention.v1; kind là complaint. Dùng đúng conversation_id, snapshot_digest và message_id trong chỉ dẫn kèm theo. evidence_refs có đúng một phần tử với message_id và quote; quote là đoạn trích chính xác từ lời khách, chứa 'khiếu nại'. Viết explanation và next_action bằng tiếng Việt dễ hiểu cho nhân viên chăm sóc khách hàng. Không bịa thông tin."
	isolated, err := r096IsolatedTransport()
	if err != nil {
		t.Fatal("R116 pinned TLS transport unavailable")
	}
	defer isolated.CloseIdleConnections()
	wire := &r116WireTransport{next: isolated, key: key, endpoint: endpoint, effectDir: effectDir}
	previous := http.DefaultTransport
	http.DefaultTransport = wire
	defer func() { http.DefaultTransport = previous }()
	provider, err := ai.NewOpenAIProvider(key, "qwen3.7-flash", 768, endpoint).WithAlibabaQwen37OnceThinkingDisabled()
	if err != nil {
		t.Fatal("R116 thinking-disabled adapter unavailable")
	}
	engine.WaitDataExecutor.Transport = provider
	engine.WaitDataExecutor.Timeout = 60 * time.Second
	if got := s.execute(f.writer, 3, s.grant.ID); got != 202 {
		t.Fatalf("R116 live claim %d", got)
	}
	var claim models.InterventionExecution
	r102DB(t, db.DB.Where("tenant_id = ? AND request_id = ?", f.tenant, s.request.ID).Take(&claim).Error)
	defer func() {
		deadline := time.Now().Add(75 * time.Second)
		for time.Now().Before(deadline) {
			var row models.InterventionExecution
			if db.DB.Where("id = ?", claim.ID).Take(&row).Error == nil && row.TerminalCode != "" {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Error("R116 live child remained nonterminal before transport restore")
	}()
	terminal := r116AwaitFor(t, claim.ID, 75*time.Second)
	wire.mu.Lock()
	attempts, request, response, status := wire.attempts, wire.request, wire.response, wire.status
	wire.mu.Unlock()
	if attempts != 1 || response == "" || request == "" || !r116SafeJSON([]byte(request), key) || !r116SafeJSON([]byte(response), key) {
		t.Fatal("R116 live wire receipt incomplete")
	}
	var attempt models.InterventionAttempt
	r102DB(t, db.DB.Where("tenant_id = ? AND execution_id = ?", f.tenant, claim.ID).Take(&attempt).Error)
	var budget models.InterventionBudget
	r102DB(t, db.DB.Where("tenant_id = ? AND job_id = ?", f.tenant, f.job).Take(&budget).Error)
	var proposal models.InterventionProposal
	proposalErr := db.DB.Where("tenant_id = ? AND run_id = ?", f.tenant, claim.RunID).Take(&proposal).Error
	var afterCase models.InterventionCase
	r102DB(t, db.DB.Where("tenant_id = ? AND id = ?", f.tenant, f.caseID).Take(&afterCase).Error)
	var original models.AnalysisSnapshot
	r102DB(t, db.DB.Where("tenant_id = ? AND id = ?", f.tenant, f.snap).Take(&original).Error)
	r116AssertHistory(t, f, s, originalHistory)
	var storedRequest models.InterventionRerunRequest
	r102DB(t, db.DB.Where("tenant_id = ? AND id = ?", f.tenant, s.request.ID).Take(&storedRequest).Error)
	var eventCount, runCount, snapshotCount, attemptCount, proposalCount, receiptCount int64
	r102DB(t, db.DB.Model(&models.InterventionHumanEvent{}).Where("tenant_id = ? AND case_id = ?", f.tenant, f.caseID).Count(&eventCount).Error)
	r102DB(t, db.DB.Model(&models.JobRun{}).Where("tenant_id = ? AND id = ?", f.tenant, claim.RunID).Count(&runCount).Error)
	r102DB(t, db.DB.Model(&models.AnalysisSnapshot{}).Where("tenant_id = ? AND job_run_id = ?", f.tenant, claim.RunID).Count(&snapshotCount).Error)
	r102DB(t, db.DB.Model(&models.InterventionAttempt{}).Where("tenant_id = ? AND execution_id = ?", f.tenant, claim.ID).Count(&attemptCount).Error)
	r102DB(t, db.DB.Model(&models.InterventionProposal{}).Where("tenant_id = ? AND run_id = ?", f.tenant, claim.RunID).Count(&proposalCount).Error)
	r102DB(t, db.DB.Model(&models.InterventionReceipt{}).Where("tenant_id = ? AND run_id = ?", f.tenant, claim.RunID).Count(&receiptCount).Error)
	actionStatus := 0
	if terminal.State == "PUBLISHED" && proposalErr == nil && afterCase.ActiveProposalID != nil {
		action := map[string]any{"action": "CONFIRM", "expected_version": afterCase.Version, "expected_snapshot_id": claim.SnapshotID, "expected_snapshot_digest": claim.SnapshotDigest, "expected_proposal_id": proposal.ID, "expected_revision": proposal.Revision, "reason": "Đã xác nhận đề xuất cho khách hàng giả lập"}
		actionStatus = f.send("POST", f.tenant, f.writer, "/actions", action).Code
	}
	receipt := map[string]any{"physical_provider": "alibaba_maas", "adapter_provider": "openai", "endpoint_sha256": hex.EncodeToString(endpointHash[:]), "owner_grant": ownerGrant, "admission_sha256": admissionSHA, "grant_evidence_ref": liveEvidence, "config_ref": liveRef, "prompt_version": "r116-alibaba-v1", "tenant_id": f.tenant, "case_id": f.caseID, "actor_id": f.writer, "job_id": f.job, "revision_id": s.request.DataRevisionID, "execution_id": claim.ID, "grant_id": s.grant.ID, "request_id": s.request.ID, "run_id": claim.RunID, "snapshot_id": claim.SnapshotID, "source_digest": claim.SnapshotDigest, "original_run_id": originalHistory.Run.ID, "original_run_status": originalHistory.Run.Status, "original_receipt_id": originalHistory.Receipt.ID, "original_receipt_code": originalHistory.Receipt.Code, "original_snapshot_id": f.snap, "original_snapshot_digest": original.Digest, "original_revision_id": originalHistory.Revision.ID, "original_revision_digest": originalHistory.Revision.Digest, "original_request_status": storedRequest.Status, "original_event_count": len(originalHistory.Events), "original_history_equal": true, "state": terminal.State, "terminal_code": terminal.TerminalCode, "attempts": attempts, "http_status": status, "request": json.RawMessage(request), "response": json.RawMessage(response), "usage_state": attempt.UsageState, "input_tokens": attempt.InputTokens, "output_tokens": attempt.OutputTokens, "response_model": attempt.ResponseModel, "budget_cap": budget.Cap, "budget_consumed": budget.Consumed, "run_count": runCount, "snapshot_count": snapshotCount, "attempt_count": attemptCount, "proposal_count": proposalCount, "receipt_count": receiptCount, "event_count_before_action": eventCount, "active_proposal_id": afterCase.ActiveProposalID, "case_version_before_action": afterCase.Version, "human_confirm_status": actionStatus}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil || !r116SafeJSON(encoded, key) {
		t.Fatal("R116 live receipt secret guard")
	}
	if err := os.WriteFile(filepath.Join(evidenceDir, "r116_live_wire.json"), encoded, 0600); err != nil {
		t.Fatal("R116 live receipt write failed")
	}
	if terminal.State != "PUBLISHED" || budget.Consumed != 1 || runCount != 1 || snapshotCount != 1 || attemptCount != 1 || proposalCount != 1 || receiptCount != 1 || original.Digest != f.originalDigest || storedRequest.Status != "AWAITING_EXECUTION_AUTHORITY" || actionStatus != 200 {
		t.Fatalf("R116 live product acceptance failed: state=%s action=%d", terminal.State, actionStatus)
	}
}
