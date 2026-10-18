package handlers_test

// R095 is an explicitly invoked, isolated application harness. Ordinary go test
// runs skip it. A Python admission wrapper pins this compiled binary before the
// only permitted real-provider runs; this file is not installed app activation.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
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
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/logger"
)

const r095Model = "qwen3.7-flash"
const r095Tokens = 758

type r095Input struct {
	Mode     string      `json:"mode"`
	Role     string      `json:"role"`
	DSN      string      `json:"dsn"`
	Endpoint string      `json:"endpoint"`
	Key      string      `json:"api_key"`
	Fixture  r095Fixture `json:"fixture"`
}

type r095Fixture struct {
	Tenant               string `json:"tenant"`
	User                 string `json:"user"`
	Job                  string `json:"job"`
	Channel              string `json:"channel"`
	WorkerConversation   string `json:"worker_conversation"`
	ReviewerConversation string `json:"reviewer_conversation"`
	WaitConversation     string `json:"wait_conversation"`
	DenyJob              string `json:"deny_job"`
	DenyConversation     string `json:"deny_conversation"`
}

type r095Wire struct {
	Attempts         int    `json:"attempts"`
	Status           int    `json:"status"`
	Request          string `json:"request"`
	Response         string `json:"response"`
	TransportCode    string `json:"transport_code"`
	ResponseTooLarge bool   `json:"response_too_large"`
}

type r095Result struct {
	SchemaVersion  int                           `json:"schema_version"`
	Role           string                        `json:"role"`
	Fixture        r095Fixture                   `json:"fixture"`
	WaitStatus     int                           `json:"wait_status"`
	DenyStatus     int                           `json:"deny_status"`
	PositiveStatus int                           `json:"positive_status"`
	PositiveBody   string                        `json:"positive_body"`
	DetailStatus   int                           `json:"detail_status"`
	DetailBody     string                        `json:"detail_body"`
	BudgetCap      int                           `json:"budget_cap"`
	BudgetConsumed int                           `json:"budget_consumed"`
	Attempt        *models.InterventionAttempt   `json:"attempt"`
	Case           *models.InterventionCase      `json:"case"`
	Snapshot       *models.AnalysisSnapshot      `json:"snapshot"`
	Messages       []models.Message              `json:"messages"`
	Proposals      []models.InterventionProposal `json:"proposals"`
	Receipts       []models.InterventionReceipt  `json:"receipts"`
	Wire           r095Wire                      `json:"wire"`
	ErrorCode      string                        `json:"error_code"`
}

func r095Output(result r095Result, key string) {
	raw, _ := json.Marshal(result)
	fmt.Println("R095_RESULT=" + r095Redact(string(raw), key))
}

func r095ApprovedBinary() bool {
	raw, err := os.ReadFile(".cvf/runtime/r095_live/root_approval.json")
	if err != nil || len(raw) > 20000 {
		return false
	}
	var approval struct {
		ApprovedBy            string `json:"approvedBy"`
		BinarySha256          string `json:"binarySha256"`
		Model                 string `json:"model"`
		RequestedTokens       int    `json:"requestedTokens"`
		PhysicalRequestsTotal int    `json:"physicalRequestsTotal"`
	}
	if json.Unmarshal(raw, &approval) != nil || approval.ApprovedBy != "Codex /root" || approval.Model != r095Model || approval.RequestedTokens != r095Tokens || approval.PhysicalRequestsTotal != 2 {
		return false
	}
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	file, err := os.Open(executable)
	if err != nil {
		return false
	}
	defer file.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return false
	}
	return approval.BinarySha256 == hex.EncodeToString(hash.Sum(nil))
}

func r095LocalObject(path string, out any) bool {
	return r095LocalObjectBound(path, 65536, out)
}

func r095LocalObjectBound(path string, max int64, out any) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, max+1))
	return err == nil && int64(len(raw)) <= max && json.Unmarshal(raw, out) == nil
}

func r095ReviewerUsageSafe() bool {
	var receipt struct {
		Actual struct {
			Attempt struct {
				OutputTokens *int64 `json:"output_tokens"`
			} `json:"attempt"`
			Wire struct {
				Response string `json:"response"`
			} `json:"wire"`
		} `json:"actual"`
	}
	if !r095LocalObjectBound(".cvf/runtime/r095_live/worker_receipt.json", 4<<20, &receipt) {
		return false
	}
	if receipt.Actual.Attempt.OutputTokens != nil && *receipt.Actual.Attempt.OutputTokens > 768 {
		return false
	}
	if receipt.Actual.Wire.Response != "" {
		var body struct {
			Usage struct {
				CompletionTokens *int64 `json:"completion_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal([]byte(receipt.Actual.Wire.Response), &body) == nil && body.Usage.CompletionTokens != nil && *body.Usage.CompletionTokens > 768 {
			return false
		}
	}
	return true
}

func r095ExactDSN(port int) string {
	return fmt.Sprintf("root:r095-synthetic-only@tcp(127.0.0.1:%d)/r095_fixture?parseTime=true&charset=utf8mb4", port)
}

func r095StartOnce(name string) bool {
	file, err := os.OpenFile(".cvf/runtime/r095_live/"+name+".go-start", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return false
	}
	defer file.Close()
	_, err = fmt.Fprintln(file, time.Now().UTC().Format(time.RFC3339))
	return err == nil && file.Sync() == nil
}

func r095Boundary(in r095Input) bool {
	var state struct {
		CurrentMode   string `json:"currentMode"`
		ActiveTranche string `json:"activeTranche"`
		ActiveHandoff string `json:"activeHandoff"`
		ActiveRole    string `json:"activeRole"`
	}
	if !r095LocalObjectBound("CVF_SESSION/ACTIVE_SESSION_STATE.json", 256<<10, &state) || state.ActiveTranche != "CCMAI-RUNTIME-095" || state.ActiveHandoff != "CVF_SESSION/handoffs/AGENT_HANDOFF_M2_LIVE_R095_2026-10-14.md" {
		return false
	}
	var ownership struct {
		Port         int               `json:"port"`
		ProtectedIDs map[string]string `json:"protectedIds"`
	}
	if !r095LocalObject(".cvf/runtime/r095_live/ownership.json", &ownership) || ownership.Port < 1024 || len(ownership.ProtectedIDs) != 3 || in.DSN != r095ExactDSN(ownership.Port) {
		return false
	}
	var pin struct {
		ContainerID string   `json:"containerId"`
		NetworkID   string   `json:"networkId"`
		ImageID     string   `json:"imageId"`
		VolumeIDs   []string `json:"volumeIds"`
	}
	if !r095LocalObject(".cvf/runtime/r095_live/container_pin.json", &pin) || len(pin.ContainerID) != 64 || len(pin.NetworkID) != 64 || !strings.HasPrefix(pin.ImageID, "sha256:") || len(pin.VolumeIDs) != 1 {
		return false
	}
	if in.Mode == "seed" {
		return state.CurrentMode == "BUILD" && (strings.Contains(state.ActiveRole, "IMPLEMENTATION_WORKER") || strings.Contains(state.ActiveRole, "REPAIR_WORKER")) && in.Role == "seed" && in.Key == "" && r095StartOnce("seed")
	}
	if in.Mode != "live" || (in.Role != "worker" && in.Role != "reviewer") || len(in.Key) < 16 || len(in.Key) > 512 {
		return false
	}
	for _, ch := range in.Key {
		if ch < 33 || ch > 126 {
			return false
		}
	}
	if in.Role == "worker" && (state.CurrentMode != "BUILD" || !(strings.Contains(state.ActiveRole, "IMPLEMENTATION_WORKER") || strings.Contains(state.ActiveRole, "REPAIR_WORKER"))) {
		return false
	}
	if in.Role == "reviewer" && (state.CurrentMode != "REVIEW" || !strings.Contains(state.ActiveRole, "REVIEWER")) {
		return false
	}
	normalized := strings.TrimSuffix(in.Endpoint, "/")
	h := sha256.Sum256([]byte(normalized))
	if hex.EncodeToString(h[:]) != "b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f" {
		return false
	}
	u, err := url.Parse(normalized)
	if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/compatible-mode/v1" || !strings.HasSuffix(u.Hostname(), ".ap-southeast-1.maas.aliyuncs.com") || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	var fixture struct {
		ContainerID   string      `json:"containerId"`
		Port          int         `json:"port"`
		Fixture       r095Fixture `json:"fixture"`
		SeedBinarySha string      `json:"seedBinarySha256"`
	}
	if !r095LocalObject(".cvf/runtime/r095_live/fixture.json", &fixture) || fixture.ContainerID != pin.ContainerID || fixture.Port != ownership.Port || fixture.Fixture != in.Fixture {
		return false
	}
	var approval struct {
		BinarySha string `json:"binarySha256"`
	}
	if !r095LocalObject(".cvf/runtime/r095_live/root_approval.json", &approval) || fixture.SeedBinarySha != approval.BinarySha {
		return false
	}
	var campaign struct {
		StartedEpoch float64 `json:"startedEpoch"`
		MaxSeconds   int     `json:"maxSeconds"`
	}
	if !r095LocalObject(".cvf/runtime/r095_live/campaign.json", &campaign) || campaign.MaxSeconds != 300 || time.Since(time.Unix(int64(campaign.StartedEpoch), 0)) < 0 || time.Since(time.Unix(int64(campaign.StartedEpoch), 0)) > 300*time.Second {
		return false
	}
	var claim struct {
		Role  string `json:"role"`
		State string `json:"state"`
	}
	if !r095LocalObject(".cvf/runtime/r095_live/"+in.Role+".claim", &claim) || claim.Role != in.Role || claim.State != "CONSUMED_BEFORE_HTTP" {
		return false
	}
	var dispatch struct {
		Role      string `json:"role"`
		BinarySha string `json:"binarySha256"`
	}
	if !r095LocalObject(".cvf/runtime/r095_live/"+in.Role+".dispatch", &dispatch) || dispatch.Role != in.Role || dispatch.BinarySha != approval.BinarySha {
		return false
	}
	if in.Role == "reviewer" {
		var workerClaim struct {
			Role string `json:"role"`
		}
		if !r095LocalObject(".cvf/runtime/r095_live/worker.claim", &workerClaim) || workerClaim.Role != "worker" || !r095ReviewerUsageSafe() {
			return false
		}
		if _, err := os.Stat(".cvf/runtime/r095_live/worker.go-start"); err != nil {
			return false
		}
	}
	return r095StartOnce(in.Role)
}

// The recorder delegates the exact validated request bytes to the original
// network transport. It records neither Authorization nor any other headers.
type r095Recorder struct {
	mu       sync.Mutex
	next     http.RoundTripper
	key      string
	endpoint string
	wire     r095Wire
}

func r095Redact(value, key string) string {
	if key == "" {
		return value
	}
	encoded, _ := json.Marshal(key)
	value = strings.ReplaceAll(value, key, "[REDACTED]")
	if len(encoded) >= 2 {
		value = strings.ReplaceAll(value, string(encoded[1:len(encoded)-1]), "[REDACTED]")
	}
	return value
}

func (r *r095Recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.wire.Attempts != 0 {
		return nil, errors.New("R095_SECOND_NETWORK_ATTEMPT_BLOCKED")
	}
	if req.Method != http.MethodPost || req.URL.String() != strings.TrimSuffix(r.endpoint, "/")+"/chat/completions" || req.URL.Scheme != "https" || req.Header.Get("Authorization") != "Bearer "+r.key {
		return nil, errors.New("R095_REQUEST_SHAPE_REJECTED")
	}
	request, err := io.ReadAll(io.LimitReader(req.Body, 6001))
	if err != nil || len(request) > 6000 {
		return nil, errors.New("R095_REQUEST_SIZE_REJECTED")
	}
	var shape struct {
		Model    string `json:"model"`
		Max      int    `json:"max_completion_tokens"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if json.Unmarshal(request, &shape) != nil || shape.Model != r095Model || shape.Max != r095Tokens || len(shape.Messages) != 2 || shape.Messages[0].Role != "system" || shape.Messages[1].Role != "user" || strings.Contains(string(request), "enable_thinking") {
		return nil, errors.New("R095_REQUEST_SHAPE_REJECTED")
	}
	r.wire.Attempts = 1 // This marker precedes the physical transport attempt.
	r.wire.Request = r095Redact(string(request), r.key)
	req.Body = io.NopCloser(bytes.NewReader(request))
	response, transportErr := r.next.RoundTrip(req)
	if transportErr != nil {
		r.wire.TransportCode = "TRANSPORT_UNKNOWN"
		return nil, errors.New("R095_TRANSPORT_UNKNOWN")
	}
	r.wire.Status = response.StatusCode
	body, readErr := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	_ = response.Body.Close()
	if readErr != nil {
		r.wire.TransportCode = "RESPONSE_READ_FAILED"
		return nil, errors.New("R095_RESPONSE_READ_FAILED")
	}
	if len(body) > 1<<20 {
		r.wire.ResponseTooLarge = true
		return nil, errors.New("R095_RESPONSE_TOO_LARGE")
	}
	r.wire.Response = r095Redact(string(body), r.key)
	response.Body = io.NopCloser(bytes.NewReader(body))
	return response, nil
}

func r095Send(router http.Handler, token, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func r095Seed() (r095Fixture, error) {
	now := time.Now().UTC()
	f := r095Fixture{Tenant: pkg.NewUUID(), User: pkg.NewUUID(), Job: pkg.NewUUID(), Channel: pkg.NewUUID(), WorkerConversation: pkg.NewUUID(), ReviewerConversation: pkg.NewUUID(), WaitConversation: pkg.NewUUID(), DenyJob: pkg.NewUUID(), DenyConversation: pkg.NewUUID()}
	deniedChannel := pkg.NewUUID()
	rows := []any{
		&models.Tenant{ID: f.Tenant, Name: "R095 fabricated", Slug: "r095-" + f.Tenant, Settings: "{}", CreatedAt: now, UpdatedAt: now},
		&models.User{ID: f.User, Email: f.User + "@example.invalid", PasswordHash: "fixture", CreatedAt: now, UpdatedAt: now},
		&models.UserTenant{UserID: f.User, TenantID: f.Tenant, Role: "owner", Permissions: "{}"},
		&models.Channel{ID: f.Channel, TenantID: f.Tenant, ChannelType: "pancake", Name: "R095 fabricated", ExternalID: f.Channel, CredentialsEncrypted: []byte("fixture"), IsDemoFixture: true, IsActive: true, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Channel{ID: deniedChannel, TenantID: f.Tenant, ChannelType: "pancake", Name: "R095 unmarked", ExternalID: deniedChannel, CredentialsEncrypted: []byte("fixture"), IsDemoFixture: false, IsActive: true, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Job{ID: f.Job, TenantID: f.Tenant, Name: "R095 selected intervention", JobType: "intervention", InputChannelIDs: fmt.Sprintf("[%q]", f.Channel), RulesConfig: "{}", Outputs: "[]", ScheduleType: "manual", IsActive: true, CreatedAt: now, UpdatedAt: now},
		&models.Job{ID: f.DenyJob, TenantID: f.Tenant, Name: "R095 denied intervention", JobType: "intervention", InputChannelIDs: fmt.Sprintf("[%q]", deniedChannel), RulesConfig: "{}", Outputs: "[]", ScheduleType: "manual", IsActive: true, CreatedAt: now, UpdatedAt: now},
		&models.InterventionBudget{ID: pkg.NewUUID(), TenantID: f.Tenant, JobID: f.Job, Cap: 2, Consumed: 0, CreatedAt: now},
		&models.Conversation{ID: f.WorkerConversation, TenantID: f.Tenant, ChannelID: f.Channel, ExternalConversationID: f.WorkerConversation, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Conversation{ID: f.ReviewerConversation, TenantID: f.Tenant, ChannelID: f.Channel, ExternalConversationID: f.ReviewerConversation, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Conversation{ID: f.WaitConversation, TenantID: f.Tenant, ChannelID: f.Channel, ExternalConversationID: f.WaitConversation, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Conversation{ID: f.DenyConversation, TenantID: f.Tenant, ChannelID: deniedChannel, ExternalConversationID: f.DenyConversation, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
	}
	for _, row := range rows {
		if err := db.DB.Create(row).Error; err != nil {
			return f, err
		}
	}
	for _, item := range []struct{ conversation, content string }{{f.WorkerConversation, "Tôi khiếu nại đơn hàng thử nghiệm SYN-204 bị giao sai; đã chờ ba ngày và chưa được phản hồi."}, {f.ReviewerConversation, "Tôi vẫn đang chờ phản hồi về đơn hàng thử nghiệm SYN-305 từ ba ngày trước."}} {
		if err := db.DB.Create(&models.Message{ID: pkg.NewUUID(), TenantID: f.Tenant, ConversationID: item.conversation, ExternalMessageID: pkg.NewUUID(), SenderType: "customer", Content: item.content, ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: now, CreatedAt: now}).Error; err != nil {
			return f, err
		}
	}
	return f, nil
}

func TestR095Application(t *testing.T) {
	if os.Getenv("R095_EXPLICIT_BINARY") != "1" {
		t.Skip("NOT_RUN: explicit R095 binary invocation required")
	}
	result := r095Result{SchemaVersion: 1, ErrorCode: "PRECONDITION_FAILED"}
	var in r095Input
	defer func() { r095Output(result, in.Key) }()
	// Approval is checked before stdin, which may contain a provider key.
	if !r095ApprovedBinary() {
		result.ErrorCode = "ROOT_APPROVAL_MISSING"
		return
	}
	if err := json.NewDecoder(io.LimitReader(os.Stdin, 20000)).Decode(&in); err != nil {
		result.ErrorCode = "INPUT_INVALID"
		return
	}
	result.Role = in.Role
	result.Fixture = in.Fixture
	if !r095Boundary(in) {
		result.ErrorCode = "BOUNDARY_REJECTED"
		return
	}
	if err := db.Connect(in.DSN, false); err != nil {
		result.ErrorCode = "DB_CONNECT_FAILED"
		return
	}
	defer db.Close()
	db.DB.Logger = logger.Default.LogMode(logger.Silent)
	gin.SetMode(gin.ReleaseMode)
	if in.Mode == "seed" {
		if in.Key != "" {
			result.ErrorCode = "SEED_KEY_REJECTED"
			return
		}
		if err := db.AutoMigrate(); err != nil {
			result.ErrorCode = "SCHEMA_FAILED"
			return
		}
		fixture, err := r095Seed()
		if err != nil {
			result.ErrorCode = "SEED_FAILED"
			return
		}
		result.Fixture = fixture
		result.ErrorCode = ""
		return
	}
	if in.Mode != "live" || (in.Role != "worker" && in.Role != "reviewer") || len(in.Key) < 16 || in.Endpoint == "" {
		result.ErrorCode = "LIVE_INPUT_INVALID"
		return
	}
	f := in.Fixture
	var budget models.InterventionBudget
	if err := db.DB.Where("tenant_id = ? AND job_id = ?", f.Tenant, f.Job).First(&budget).Error; err != nil || budget.Cap != 2 || budget.Consumed != map[string]int{"worker": 0, "reviewer": 1}[in.Role] {
		result.ErrorCode = "BUDGET_PRECONDITION_FAILED"
		return
	}
	result.BudgetCap = budget.Cap
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		result.ErrorCode = "TRANSPORT_UNSUPPORTED"
		return
	}
	isolatedTransport := baseTransport.Clone()
	isolatedTransport.DisableKeepAlives = true
	isolatedTransport.MaxIdleConnsPerHost = 0
	defer isolatedTransport.CloseIdleConnections()
	recorder := &r095Recorder{next: isolatedTransport, key: in.Key, endpoint: in.Endpoint}
	previousTransport := http.DefaultTransport
	http.DefaultTransport = recorder
	defer func() { http.DefaultTransport = previousTransport }()
	defer func() { result.Wire = recorder.wire }()
	handlers.InterventionRunner = &engine.InterventionService{Transport: ai.NewOpenAIProvider(in.Key, r095Model, r095Tokens, in.Endpoint), PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "r095-live-v1", Prompt: "Analyze only this fabricated Pancake complaint or waiting conversation. Quote the customer's exact words and cite canonical message IDs. Return strictly the requested JSON object with no markdown.", MaxRequestBytes: 6000, Timeout: 60 * time.Second}
	defer func() { handlers.InterventionRunner = nil }()
	middleware.SetJWTSecret(strings.Repeat("r", 32))
	token, err := middleware.GenerateAccessToken(f.User, "r095@example.invalid", false)
	if err != nil {
		result.ErrorCode = "AUTH_FAILED"
		return
	}
	router := api.SetupRouter(&config.Config{RateLimitPerIP: 1000, RateLimitPerUser: 1000})
	prefix := "/api/v1/tenants/" + f.Tenant
	request := func(job, conversation string) *httptest.ResponseRecorder {
		return r095Send(router, token, "POST", prefix+"/jobs/"+job+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, conversation))
	}
	wait := request(f.Job, f.WaitConversation)
	result.WaitStatus = wait.Code
	deny := request(f.DenyJob, f.DenyConversation)
	result.DenyStatus = deny.Code
	if wait.Code != 200 || !bytes.Contains(wait.Body.Bytes(), []byte(`"eligibility":"WAIT_DATA"`)) || deny.Code != 403 || recorder.wire.Attempts != 0 {
		result.ErrorCode = "NEGATIVE_PRECONDITION_FAILED"
		return
	}
	conversation := f.WorkerConversation
	if in.Role == "reviewer" {
		conversation = f.ReviewerConversation
	}
	positive := request(f.Job, conversation)
	result.PositiveStatus = positive.Code
	result.PositiveBody = r095Redact(positive.Body.String(), in.Key)
	var caseRow models.InterventionCase
	if db.DB.Where("tenant_id = ? AND job_id = ? AND conversation_id = ?", f.Tenant, f.Job, conversation).First(&caseRow).Error == nil {
		result.Case = &caseRow
		detail := r095Send(router, token, "GET", prefix+"/intervention-cases/"+caseRow.ID, "")
		result.DetailStatus = detail.Code
		result.DetailBody = r095Redact(detail.Body.String(), in.Key)
		_ = db.DB.Where("case_id = ?", caseRow.ID).Order("revision ASC").Find(&result.Proposals).Error
		_ = db.DB.Where("case_id = ?", caseRow.ID).Order("created_at ASC").Find(&result.Receipts).Error
		var attempt models.InterventionAttempt
		if db.DB.Where("run_id = ?", caseRow.RunID).First(&attempt).Error == nil {
			result.Attempt = &attempt
		}
		var snapshot models.AnalysisSnapshot
		if db.DB.Where("job_run_id = ? AND conversation_id = ?", caseRow.RunID, conversation).First(&snapshot).Error == nil {
			result.Snapshot = &snapshot
		}
		_ = db.DB.Where("tenant_id = ? AND conversation_id = ?", f.Tenant, conversation).Order("sent_at ASC, id ASC").Find(&result.Messages).Error
	}
	if db.DB.Where("tenant_id = ? AND job_id = ?", f.Tenant, f.Job).First(&budget).Error == nil {
		result.BudgetConsumed = budget.Consumed
	}
	if result.PositiveStatus != 200 || recorder.wire.Attempts != 1 || result.BudgetConsumed != map[string]int{"worker": 1, "reviewer": 2}[in.Role] {
		result.ErrorCode = "POSITIVE_OBSERVATION_FAILED"
		return
	}
	result.ErrorCode = ""
}

func TestR095RecorderOffline(t *testing.T) {
	key := "sk-R095-synthetic-offline-key"                                                 // cvf-allow-secret-fixture
	if strings.Contains(r095Redact(`{"key":"sk-R095-synthetic-offline-key"}`, key), key) { // cvf-allow-secret-fixture
		t.Fatal("redaction failed")
	}
}
