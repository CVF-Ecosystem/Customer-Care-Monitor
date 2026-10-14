package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/handlers"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm/logger"
)

// Native software integration only. No mocked response in this test is CVF
// governance proof; it exercises actual MySQL and the authenticated app route.
func TestR094InterventionRouteNative(t *testing.T) {
	dsn := os.Getenv("R094_MYSQL_DSN")
	if dsn == "" {
		t.Skip("NOT_RUN: R094_MYSQL_DSN disposable MySQL not set")
	}
	if err := db.Connect(dsn, false); err != nil {
		t.Fatal(err)
	}
	db.DB.Logger = logger.Default.LogMode(logger.Silent)
	defer db.Close()
	if err := db.AutoMigrate(); err != nil {
		t.Fatal(err)
	}
	tenantID, userID, jobID, channelID := pkg.NewUUID(), pkg.NewUUID(), pkg.NewUUID(), pkg.NewUUID()
	now := time.Now().UTC()
	for _, v := range []any{
		&models.Tenant{ID: tenantID, Name: "R094 Synthetic", Slug: "r094-" + tenantID, Settings: "{}", CreatedAt: now, UpdatedAt: now},
		&models.User{ID: userID, Email: userID + "@example.invalid", PasswordHash: "fixture", CreatedAt: now, UpdatedAt: now},
		&models.UserTenant{UserID: userID, TenantID: tenantID, Role: "owner", Permissions: "{}"},
		&models.Channel{ID: channelID, TenantID: tenantID, ChannelType: "pancake", Name: "R094 fixture", ExternalID: "r094-" + channelID, CredentialsEncrypted: []byte("fixture"), IsDemoFixture: true, IsActive: true, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Job{ID: jobID, TenantID: tenantID, Name: "R094 Intervention", JobType: "intervention", InputChannelIDs: fmt.Sprintf("[%q]", channelID), RulesConfig: "{}", Outputs: "[]", ScheduleType: "manual", IsActive: true, CreatedAt: now, UpdatedAt: now},
		&models.InterventionBudget{ID: pkg.NewUUID(), TenantID: tenantID, JobID: jobID, Cap: 1, Consumed: 0, CreatedAt: now},
	} {
		if err := db.DB.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	makeConversation := func(content string) (string, string) {
		convID, msgID := pkg.NewUUID(), pkg.NewUUID()
		if err := db.DB.Create(&models.Conversation{ID: convID, TenantID: tenantID, ChannelID: channelID, ExternalConversationID: convID, Metadata: "{}", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
		if content != "" {
			if err := db.DB.Create(&models.Message{ID: msgID, TenantID: tenantID, ConversationID: convID, ExternalMessageID: msgID, SenderType: "customer", Content: content, ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: now, CreatedAt: now}).Error; err != nil {
				t.Fatal(err)
			}
		}
		return convID, msgID
	}
	goodConv, msgID := makeConversation("Tôi đang chờ đơn hàng ba ngày")
	waitConv, _ := makeConversation("")
	var calls atomic.Int64
	var responseMode atomic.Int32
	var staleMsgID, cancelMsgID, cancelJobID, validationMsgID string
	digestRE := regexp.MustCompile(`snapshot_digest ([a-f0-9]{64})`)
	convRE := regexp.MustCompile(`conversation_id ([a-f0-9-]{36})`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if responseMode.Load() == 11 {
			time.Sleep(150 * time.Millisecond)
		}
		if responseMode.Load() == 2 {
			if h, ok := w.(http.Hijacker); ok {
				conn, _, _ := h.Hijack()
				_ = conn.Close()
			}
			return
		}
		if responseMode.Load() == 3 {
			if err := db.DB.Model(&models.Message{}).Where("id = ?", staleMsgID).Update("content", "Đã thay đổi nguồn").Error; err != nil {
				t.Errorf("stale source mutation: %v", err)
			}
		}
		if responseMode.Load() == 4 {
			if _, err := engine.CancelJobRun(tenantID, cancelJobID, ""); err != nil {
				t.Errorf("cancel admitted run: %v", err)
			}
		}
		if responseMode.Load() == 12 { // force a checked attempt-settlement write failure in the disposable schema
			if err := db.DB.Exec("DROP TABLE intervention_attempts").Error; err != nil {
				t.Errorf("drop attempt fixture table: %v", err)
			}
		}
		if responseMode.Load() == 13 { // reviewer detector: fail inside atomic proposal publication
			if err := db.DB.Exec("DROP TABLE intervention_receipts").Error; err != nil {
				t.Errorf("drop publication fixture table: %v", err)
			}
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path %s", r.URL.Path)
		}
		var body struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || len(body.Messages) != 2 {
			t.Error("request malformed")
			return
		}
		prompt := body.Messages[0].Content
		digest := digestRE.FindStringSubmatch(prompt)
		conv := convRE.FindStringSubmatch(prompt)
		if len(digest) != 2 || len(conv) != 2 {
			t.Error("request lacks bound IDs")
			return
		}
		refID := msgID
		if responseMode.Load() == 1 {
			refID = pkg.NewUUID()
		}
		if responseMode.Load() == 3 {
			refID = staleMsgID
		}
		if responseMode.Load() == 4 {
			refID = cancelMsgID
		}
		if responseMode.Load() >= 5 {
			refID = validationMsgID
		}
		evidence := map[string]any{"message_id": refID, "quote": "chờ đơn hàng"}
		output := map[string]any{"schema": "ccma.intervention.v1", "conversation_id": conv[1], "snapshot_digest": digest[1], "kind": "waiting", "explanation": "Khách đang chờ", "next_action": "Kiểm tra đơn hàng", "evidence_refs": []map[string]any{evidence}}
		switch responseMode.Load() {
		case 5:
			evidence["quote"] = "không có trong tin nhắn"
		case 6:
			evidence["start"] = 99
			evidence["end"] = 101
		case 7:
			output["kind"] = "complaint"
		case 8:
			output["snapshot_digest"] = strings.Repeat("0", 64)
		case 9:
			output["schema"] = "wrong.schema"
		}
		raw, _ := json.Marshal(output)
		if responseMode.Load() == 10 {
			raw = []byte(`{"unfinished":`)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"model": "fixture-model", "choices": []map[string]any{{"message": map[string]string{"content": string(raw)}}}, "usage": map[string]int{"prompt_tokens": 5, "completion_tokens": 7}})
	}))
	defer server.Close()
	handlers.InterventionRunner = &engine.InterventionService{Transport: ai.NewOpenAIProvider("synthetic", "fixture-model", 128, server.URL+"/v1"), PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "r094-v1", Prompt: "Synthetic complaint/waiting analysis", MaxRequestBytes: 6000, Timeout: 5 * time.Second}
	defer func() { handlers.InterventionRunner = nil }()
	middleware.SetJWTSecret(strings.Repeat("x", 32))
	token, err := middleware.GenerateAccessToken(userID, "fixture@example.invalid", false)
	if err != nil {
		t.Fatal(err)
	}
	router := api.SetupRouter(&config.Config{RateLimitPerIP: 1000, RateLimitPerUser: 1000})
	send := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	prefix := "/api/v1/tenants/" + tenantID
	wait := send("POST", prefix+"/jobs/"+jobID+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, waitConv))
	if wait.Code != 200 || !bytes.Contains(wait.Body.Bytes(), []byte(`"eligibility":"WAIT_DATA"`)) || calls.Load() != 0 {
		t.Fatalf("wait path code=%d calls=%d body=%s", wait.Code, calls.Load(), wait.Body.String())
	}
	good := send("POST", prefix+"/jobs/"+jobID+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, goodConv))
	if good.Code != 200 || !bytes.Contains(good.Body.Bytes(), []byte(`"disposition":"PENDING_REVIEW"`)) || calls.Load() != 1 {
		t.Fatalf("good path code=%d calls=%d body=%s", good.Code, calls.Load(), good.Body.String())
	}
	var created models.InterventionCase
	if json.Unmarshal(good.Body.Bytes(), &created) != nil || created.ID == "" {
		t.Fatal("case body")
	}
	detail := send("GET", prefix+"/intervention-cases/"+created.ID, "")
	if detail.Code != 200 || !bytes.Contains(detail.Body.Bytes(), []byte(`"proposals"`)) || !bytes.Contains(detail.Body.Bytes(), []byte("chờ đơn hàng")) {
		t.Fatalf("detail %d %s", detail.Code, detail.Body.String())
	}
	list := send("GET", prefix+"/intervention-cases?limit=1", "")
	if list.Code != 200 || !bytes.Contains(list.Body.Bytes(), []byte(`"next_cursor"`)) {
		t.Fatalf("list %d %s", list.Code, list.Body.String())
	}
	var budget models.InterventionBudget
	if db.DB.Where("tenant_id = ? AND job_id = ?", tenantID, jobID).First(&budget).Error != nil || budget.Consumed != 1 {
		t.Fatalf("budget consumed %d", budget.Consumed)
	}
	// Reconstructed service cannot reset the stable persisted cap.
	handlers.InterventionRunner = &engine.InterventionService{Transport: ai.NewOpenAIProvider("synthetic", "fixture-model", 128, server.URL+"/v1"), PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "r094-v1", Prompt: "Synthetic complaint/waiting analysis", MaxRequestBytes: 6000, Timeout: 5 * time.Second}
	deniedConv, _ := makeConversation("Tôi đang chờ đơn khác")
	denied := send("POST", prefix+"/jobs/"+jobID+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, deniedConv))
	if denied.Code != 200 || !bytes.Contains(denied.Body.Bytes(), []byte("BUDGET_UNAVAILABLE")) || calls.Load() != 1 {
		t.Fatalf("cap path %d calls=%d body=%s", denied.Code, calls.Load(), denied.Body.String())
	}
	legacy := send("POST", prefix+"/jobs/"+jobID+"/test-run", `{}`)
	if legacy.Code != 403 || calls.Load() != 1 {
		t.Fatalf("legacy bypass %d calls=%d", legacy.Code, calls.Load())
	}
	legacy = send("POST", prefix+"/jobs/"+jobID+"/trigger", `{}`)
	if legacy.Code != 403 || calls.Load() != 1 {
		t.Fatalf("legacy trigger %d calls=%d", legacy.Code, calls.Load())
	}
	// Source-selected job cannot be converted to a scheduled path through mutation.
	mutation := send("PUT", prefix+"/jobs/"+jobID, `{"schedule_type":"cron"}`)
	if mutation.Code != 403 {
		t.Fatalf("intervention job mutation %d", mutation.Code)
	}
	memberID := pkg.NewUUID()
	if db.DB.Create(&models.User{ID: memberID, Email: memberID + "@example.invalid", PasswordHash: "fixture", CreatedAt: now, UpdatedAt: now}).Error != nil {
		t.Fatal("member seed")
	}
	if db.DB.Create(&models.UserTenant{UserID: memberID, TenantID: tenantID, Role: "member", Permissions: `{"intervention_cases":"r"}`}).Error != nil {
		t.Fatal("membership seed")
	}
	memberToken, err := middleware.GenerateAccessToken(memberID, "member@example.invalid", false)
	if err != nil {
		t.Fatal(err)
	}
	sendMember := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+memberToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	if rec := sendMember("POST", prefix+"/jobs/"+jobID+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, goodConv)); rec.Code != 403 || calls.Load() != 1 {
		t.Fatalf("member run %d", rec.Code)
	}
	if rec := sendMember("GET", prefix+"/intervention-cases", ""); rec.Code != 200 {
		t.Fatalf("member read %d", rec.Code)
	}
	if db.DB.Model(&models.UserTenant{}).Where("user_id = ? AND tenant_id = ?", memberID, tenantID).Update("permissions", "bad-json").Error != nil {
		t.Fatal("permission update")
	}
	if rec := sendMember("GET", prefix+"/intervention-cases", ""); rec.Code != 403 {
		t.Fatalf("malformed permission %d", rec.Code)
	}
	newJob := func() string {
		id := pkg.NewUUID()
		if db.DB.Create(&models.Job{ID: id, TenantID: tenantID, Name: "R094 intervention extra", JobType: "intervention", InputChannelIDs: fmt.Sprintf("[%q]", channelID), RulesConfig: "{}", Outputs: "[]", ScheduleType: "manual", IsActive: true, CreatedAt: now, UpdatedAt: now}).Error != nil {
			t.Fatal("job seed")
		}
		if db.DB.Create(&models.InterventionBudget{ID: pkg.NewUUID(), TenantID: tenantID, JobID: id, Cap: 1, CreatedAt: now}).Error != nil {
			t.Fatal("budget seed")
		}
		return id
	}
	badJob := newJob()
	badConv, _ := makeConversation("Tôi đang chờ đơn hàng ba ngày")
	responseMode.Store(1)
	bad := send("POST", prefix+"/jobs/"+badJob+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, badConv))
	if bad.Code != 200 || !bytes.Contains(bad.Body.Bytes(), []byte("OUTPUT_INVALID")) || calls.Load() != 2 {
		t.Fatalf("bad ref %d calls=%d body=%s", bad.Code, calls.Load(), bad.Body.String())
	}
	var badCase models.InterventionCase
	if json.Unmarshal(bad.Body.Bytes(), &badCase) != nil {
		t.Fatal("bad case body")
	}
	var badProposals int64
	if db.DB.Model(&models.InterventionProposal{}).Where("case_id = ?", badCase.ID).Count(&badProposals).Error != nil || badProposals != 0 {
		t.Fatalf("bad output proposal count %d", badProposals)
	}
	unknownJob := newJob()
	unknownConv, _ := makeConversation("Tôi đang chờ đơn hàng ba ngày")
	responseMode.Store(2)
	unknown := send("POST", prefix+"/jobs/"+unknownJob+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, unknownConv))
	if unknown.Code != 200 || !bytes.Contains(unknown.Body.Bytes(), []byte("PROVIDER_UNKNOWN")) || calls.Load() != 3 {
		t.Fatalf("unknown %d calls=%d body=%s", unknown.Code, calls.Load(), unknown.Body.String())
	}
	var unknownBudget models.InterventionBudget
	if db.DB.Where("tenant_id = ? AND job_id = ?", tenantID, unknownJob).First(&unknownBudget).Error != nil || unknownBudget.Consumed != 1 {
		t.Fatalf("unknown consumed %d", unknownBudget.Consumed)
	}
	var unknownAttempt models.InterventionAttempt
	if db.DB.Where("tenant_id = ? AND job_id = ?", tenantID, unknownJob).First(&unknownAttempt).Error != nil || unknownAttempt.State != "UNKNOWN" {
		t.Fatalf("unknown attempt %s", unknownAttempt.State)
	}
	staleJob := newJob()
	staleConv, sm := makeConversation("Tôi đang chờ đơn hàng ba ngày")
	staleMsgID = sm
	responseMode.Store(3)
	stale := send("POST", prefix+"/jobs/"+staleJob+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, staleConv))
	if stale.Code != 200 || !bytes.Contains(stale.Body.Bytes(), []byte("STALE_SOURCE")) || calls.Load() != 4 {
		t.Fatalf("stale %d calls=%d body=%s", stale.Code, calls.Load(), stale.Body.String())
	}
	var staleCase models.InterventionCase
	if json.Unmarshal(stale.Body.Bytes(), &staleCase) != nil {
		t.Fatal("stale body")
	}
	var staleProposals int64
	if db.DB.Model(&models.InterventionProposal{}).Where("case_id = ?", staleCase.ID).Count(&staleProposals).Error != nil || staleProposals != 0 {
		t.Fatalf("stale proposal count %d", staleProposals)
	}
	cancelJobID = newJob()
	cancelConv, cm := makeConversation("Tôi đang chờ đơn hàng ba ngày")
	cancelMsgID = cm
	responseMode.Store(4)
	cancelled := send("POST", prefix+"/jobs/"+cancelJobID+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, cancelConv))
	if cancelled.Code != 200 || !bytes.Contains(cancelled.Body.Bytes(), []byte("CANCELLED")) || calls.Load() != 5 {
		t.Fatalf("cancel %d calls=%d body=%s", cancelled.Code, calls.Load(), cancelled.Body.String())
	}
	var cancelledCase models.InterventionCase
	if json.Unmarshal(cancelled.Body.Bytes(), &cancelledCase) != nil {
		t.Fatal("cancel body")
	}
	var cancelledProposals int64
	if db.DB.Model(&models.InterventionProposal{}).Where("case_id = ?", cancelledCase.ID).Count(&cancelledProposals).Error != nil || cancelledProposals != 0 {
		t.Fatalf("cancel proposal count %d", cancelledProposals)
	}
	// F3: disabled installation and foreign source never reach transport.
	baselineCalls := calls.Load()
	currentService := handlers.InterventionRunner
	handlers.InterventionRunner = nil
	if rec := send("POST", prefix+"/jobs/"+jobID+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, goodConv)); rec.Code != 503 || calls.Load() != baselineCalls {
		t.Fatalf("default disabled %d", rec.Code)
	}
	handlers.InterventionRunner = currentService
	currentService.PolicyVersion = "unknown.policy"
	if rec := send("POST", prefix+"/jobs/"+jobID+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, goodConv)); rec.Code != 403 || calls.Load() != baselineCalls {
		t.Fatalf("unknown policy %d", rec.Code)
	}
	currentService.PolicyVersion = "ccma.synthetic.intervention.v1"
	otherTenant := pkg.NewUUID()
	otherChannel := pkg.NewUUID()
	otherConv := pkg.NewUUID()
	if err := db.DB.Create(&models.Tenant{ID: otherTenant, Name: "Other synthetic", Slug: "other-" + otherTenant, Settings: "{}", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&models.Channel{ID: otherChannel, TenantID: otherTenant, ChannelType: "pancake", Name: "Other", ExternalID: otherChannel, CredentialsEncrypted: []byte("fixture"), IsDemoFixture: true, IsActive: true, Metadata: "{}", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&models.Conversation{ID: otherConv, TenantID: otherTenant, ChannelID: otherChannel, ExternalConversationID: otherConv, Metadata: "{}", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if rec := send("POST", prefix+"/jobs/"+jobID+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, otherConv)); rec.Code != 404 || calls.Load() != baselineCalls {
		t.Fatalf("foreign source %d", rec.Code)
	}
	foreignCaseID := pkg.NewUUID()
	if err := db.DB.Create(&models.InterventionCase{ID: foreignCaseID, TenantID: otherTenant, JobID: jobID, RunID: pkg.NewUUID(), ConversationID: otherConv, UseCase: "pancake_complaint_wait", Version: 1, Eligibility: "WAIT_DATA", Execution: "NOT_RUN", Disposition: "NOT_CREATED", WorkState: "WAITING_ON_DATA", ActorID: userID, PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "r094-v1", SchemaVersion: "ccma.intervention.v1", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if rec := send("GET", prefix+"/intervention-cases/"+foreignCaseID, ""); rec.Code != 404 {
		t.Fatalf("foreign detail %d", rec.Code)
	}
	realChannel := pkg.NewUUID()
	realJob := pkg.NewUUID()
	realConv := pkg.NewUUID()
	if err := db.DB.Create(&models.Channel{ID: realChannel, TenantID: tenantID, ChannelType: "pancake", Name: "Unmarked", ExternalID: realChannel, CredentialsEncrypted: []byte("fixture"), IsDemoFixture: false, IsActive: true, Metadata: "{}", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&models.Job{ID: realJob, TenantID: tenantID, Name: "Unmarked job", JobType: "intervention", InputChannelIDs: fmt.Sprintf("[%q]", realChannel), RulesConfig: "{}", Outputs: "[]", ScheduleType: "manual", IsActive: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&models.Conversation{ID: realConv, TenantID: tenantID, ChannelID: realChannel, ExternalConversationID: realConv, Metadata: "{}", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if rec := send("POST", prefix+"/jobs/"+realJob+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, realConv)); rec.Code != 403 || !bytes.Contains(rec.Body.Bytes(), []byte(`"eligibility":"DENY"`)) || calls.Load() != baselineCalls {
		t.Fatalf("source deny %d %s", rec.Code, rec.Body.String())
	}
	// F4: independent source/evidence failure variants must settle a received
	// response and consume exactly one slot without creating a proposal.
	for mode := int32(5); mode <= 10; mode++ {
		variantJob := newJob()
		variantConv, variantMsg := makeConversation("Tôi đang chờ đơn hàng ba ngày")
		validationMsgID = variantMsg
		responseMode.Store(mode)
		before := calls.Load()
		rec := send("POST", prefix+"/jobs/"+variantJob+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, variantConv))
		if rec.Code != 200 || !bytes.Contains(rec.Body.Bytes(), []byte("OUTPUT_INVALID")) || calls.Load() != before+1 {
			t.Fatalf("invalid mode=%d status=%d body=%s", mode, rec.Code, rec.Body.String())
		}
		var row models.InterventionCase
		if json.Unmarshal(rec.Body.Bytes(), &row) != nil {
			t.Fatal("variant body")
		}
		var n int64
		if db.DB.Model(&models.InterventionProposal{}).Where("case_id = ?", row.ID).Count(&n).Error != nil || n != 0 {
			t.Fatalf("variant proposal mode=%d count=%d", mode, n)
		}
		var attempt models.InterventionAttempt
		if db.DB.Where("run_id = ?", row.RunID).First(&attempt).Error != nil || attempt.State != "RESPONSE_RECEIVED" || attempt.UsageState != "KNOWN" || attempt.InputTokens == nil || attempt.OutputTokens == nil {
			t.Fatalf("variant settlement mode=%d state=%s usage=%s", mode, attempt.State, attempt.UsageState)
		}
	}
	// Two concurrent requests for one job cannot make two physical calls.
	contentionJob := newJob()
	convA, msgA := makeConversation("Tôi đang chờ đơn hàng ba ngày")
	convB, _ := makeConversation("Tôi đang chờ đơn hàng ba ngày")
	validationMsgID = msgA
	responseMode.Store(11)
	before := calls.Load()
	results := make([]*httptest.ResponseRecorder, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		results[0] = send("POST", prefix+"/jobs/"+contentionJob+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, convA))
	}()
	go func() {
		defer wg.Done()
		results[1] = send("POST", prefix+"/jobs/"+contentionJob+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, convB))
	}()
	wg.Wait()
	if calls.Load() != before+1 || !((results[0].Code == 200 && results[1].Code == 409) || (results[1].Code == 200 && results[0].Code == 409)) {
		t.Fatalf("contention calls=%d statuses=%d/%d", calls.Load()-before, results[0].Code, results[1].Code)
	}
	// Reviewer-added independent detector: response settlement succeeds, but a
	// later publication write fails. No partial proposal/case success may survive.
	publicationJob := newJob()
	publicationConv, publicationMsg := makeConversation("Tôi đang chờ đơn hàng ba ngày")
	validationMsgID = publicationMsg
	responseMode.Store(13)
	publication := send("POST", prefix+"/jobs/"+publicationJob+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, publicationConv))
	if publication.Code != 503 || strings.Contains(publication.Body.String(), "intervention_receipts") {
		t.Fatalf("atomic publication failure %d %s", publication.Code, publication.Body.String())
	}
	var publicationCase models.InterventionCase
	if err := db.DB.Where("job_id = ? AND tenant_id = ?", publicationJob, tenantID).First(&publicationCase).Error; err != nil {
		t.Fatal("publication case absent")
	}
	var publicationCount int64
	if err := db.DB.Model(&models.InterventionProposal{}).Where("case_id = ?", publicationCase.ID).Count(&publicationCount).Error; err != nil || publicationCount != 0 || publicationCase.Disposition == "PENDING_REVIEW" || publicationCase.Execution == "LLM_SUCCEEDED" {
		t.Fatalf("partial publication survived: proposals=%d disposition=%s execution=%s", publicationCount, publicationCase.Disposition, publicationCase.Execution)
	}
	var publicationAttempt models.InterventionAttempt
	var publicationBudget models.InterventionBudget
	if err := db.DB.Where("job_id = ? AND tenant_id = ?", publicationJob, tenantID).First(&publicationAttempt).Error; err != nil || publicationAttempt.State != "RESPONSE_RECEIVED" {
		t.Fatalf("physical response truth lost: %s", publicationAttempt.State)
	}
	if err := db.DB.Where("job_id = ? AND tenant_id = ?", publicationJob, tenantID).First(&publicationBudget).Error; err != nil || publicationBudget.Consumed != 1 {
		t.Fatalf("publication slot refunded: %d", publicationBudget.Consumed)
	}
	if err := db.DB.AutoMigrate(&models.InterventionReceipt{}); err != nil {
		t.Fatal("restore publication fixture table")
	}
	responseMode.Store(0)
	if retry := send("POST", prefix+"/jobs/"+publicationJob+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, publicationConv)); retry.Code != 409 {
		t.Fatalf("crash-left run did not block reentry: %d", retry.Code)
	}
	// Last, force settlement failure; the consumed cap survives and proposal is absent.
	settlementJob := newJob()
	settlementConv, settlementMsg := makeConversation("Tôi đang chờ đơn hàng ba ngày")
	validationMsgID = settlementMsg
	responseMode.Store(12)
	settlement := send("POST", prefix+"/jobs/"+settlementJob+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, settlementConv))
	if settlement.Code != 200 || !bytes.Contains(settlement.Body.Bytes(), []byte("SETTLEMENT_FAILED")) {
		t.Fatalf("settlement failure %d %s", settlement.Code, settlement.Body.String())
	}
	var consumed models.InterventionBudget
	if db.DB.Where("tenant_id = ? AND job_id = ?", tenantID, settlementJob).First(&consumed).Error != nil || consumed.Consumed != 1 {
		t.Fatalf("settlement cap consumed=%d", consumed.Consumed)
	}
}
