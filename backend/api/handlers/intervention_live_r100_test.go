package handlers_test

// R100 is an explicitly invoked, isolated application harness. Ordinary go test
// runs skip it. A Python admission wrapper pins this compiled binary before the
// only permitted real-provider runs; this file is not installed app activation.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
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

const r100Model = "qwen3.7-flash"
const r100Tokens = 758

type r100Input struct {
	Mode     string      `json:"mode"`
	Role     string      `json:"role"`
	DSN      string      `json:"dsn"`
	Endpoint string      `json:"endpoint"`
	Key      string      `json:"api_key"`
	Fixture  r100Fixture `json:"fixture"`
}

type r100Fixture struct {
	Tenant           string    `json:"tenant"`
	User             string    `json:"user"`
	Job              string    `json:"job"`
	Channel          string    `json:"channel"`
	Conversations    [4]string `json:"conversations"`
	WaitConversation string    `json:"wait_conversation"`
	DenyJob          string    `json:"deny_job"`
	DenyConversation string    `json:"deny_conversation"`
}

type r100Wire struct {
	Attempts         int    `json:"attempts"`
	Status           int    `json:"status"`
	Request          string `json:"request"`
	Response         string `json:"response"`
	TransportCode    string `json:"transport_code"`
	TransportClass   string `json:"transport_class"`
	TransportTimeout bool   `json:"transport_timeout"`
	URLWrapped       bool   `json:"url_wrapped"`
	ResponseTooLarge bool   `json:"response_too_large"`
}

type r100Result struct {
	SchemaVersion  int                           `json:"schema_version"`
	Role           string                        `json:"role"`
	Fixture        r100Fixture                   `json:"fixture"`
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
	Wire           r100Wire                      `json:"wire"`
	ErrorCode      string                        `json:"error_code"`
	Slot           int                           `json:"slot"`
	Decision       string                        `json:"decision"`
	DBReadOK       bool                          `json:"db_read_ok"`
}

func r100Output(result r100Result, key string) {
	raw, safe := r100SafeResult(result, key)
	if !safe {
		raw = r100MinimalSecretFailure(result)
	}
	fmt.Println("R100_RESULT=" + string(raw))
}

func r100MinimalSecretFailure(result r100Result) []byte {
	raw, _ := json.Marshal(map[string]any{"schema_version": 1, "slot": result.Slot, "decision": "HARD_STOP_SECRET", "error_code": "RECEIPT_SECRET_GUARD", "wire": map[string]any{"attempts": result.Wire.Attempts, "status": result.Wire.Status}})
	return raw
}

// Inspect decoded fields and nested JSON strings. Any uncertain escape or
// over-deep value fails closed so no provider-controlled text reaches a file
// or stdout when it might contain the credential.
func r100SecretUnsafe(value any, key string, depth int) bool {
	if depth > 16 {
		return true
	}
	switch item := value.(type) {
	case string:
		if strings.Contains(item, key) {
			return true
		}
		if strings.Contains(item, "\\u") {
			var nested any
			if json.Unmarshal([]byte(item), &nested) != nil {
				return true
			}
			return r100SecretUnsafe(nested, key, depth+1)
		}
		if json.Valid([]byte(item)) {
			var nested any
			if json.Unmarshal([]byte(item), &nested) != nil {
				return true
			}
			return r100SecretUnsafe(nested, key, depth+1)
		}
	case []any:
		if len(item) > 10000 {
			return true
		}
		for _, child := range item {
			if r100SecretUnsafe(child, key, depth+1) {
				return true
			}
		}
	case map[string]any:
		if len(item) > 10000 {
			return true
		}
		for name, child := range item {
			if r100SecretUnsafe(name, key, depth+1) || r100SecretUnsafe(child, key, depth+1) {
				return true
			}
		}
	}
	return false
}

func r100SafeResult(result r100Result, key string) ([]byte, bool) {
	raw, err := json.Marshal(result)
	if err != nil || len(raw) > 4<<20 {
		return nil, false
	}
	if key == "" {
		return raw, true
	}
	var logical any
	if json.Unmarshal(raw, &logical) != nil || r100SecretUnsafe(logical, key, 0) || bytes.Contains(raw, []byte(key)) {
		return nil, false
	}
	return raw, true
}

func r100ApprovedBinary() bool {
	raw, err := os.ReadFile(".cvf/runtime/r100_live/root_approval.json")
	if err != nil || len(raw) > 20000 {
		return false
	}
	var approval struct {
		ApprovedBy            string            `json:"approvedBy"`
		BinarySha256          string            `json:"binarySha256"`
		Model                 string            `json:"model"`
		RequestedTokens       int               `json:"requestedTokens"`
		PhysicalRequestsTotal int               `json:"physicalRequestsTotal"`
		ParentManifestSha256  string            `json:"parentManifestSha256"`
		FreshOwnerGrantID     string            `json:"freshOwnerGrantId"`
		TLSDependencySha256   string            `json:"tlsDependencySha256"`
		SourceSha256          map[string]string `json:"sourceSha256"`
		FrozenSourceSha256    map[string]string `json:"frozenSourceSha256"`
	}
	if json.Unmarshal(raw, &approval) != nil || approval.ApprovedBy != "Codex /root" || approval.Model != r100Model || approval.RequestedTokens != r100Tokens || approval.PhysicalRequestsTotal != 4 || approval.FreshOwnerGrantID != r100FreshOwnerGrantID || approval.TLSDependencySha256 != r100TLSDependencySHA {
		return false
	}
	dependency, err := os.ReadFile("backend/api/handlers/intervention_live_r096_test.go")
	if err != nil {
		return false
	}
	dependencySHA := sha256.Sum256(dependency)
	if hex.EncodeToString(dependencySHA[:]) != r100TLSDependencySHA {
		return false
	}
	if len(approval.SourceSha256) != 3 || len(approval.FrozenSourceSha256) != 13 {
		return false
	}
	for path, expected := range approval.SourceSha256 {
		if !r100FileSHA(path, expected) {
			return false
		}
	}
	for path, expected := range approval.FrozenSourceSha256 {
		if !r100FileSHA(path, expected) {
			return false
		}
	}
	parentHash, parentOK := r100ParentKnown()
	if !parentOK || approval.ParentManifestSha256 != parentHash {
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

func r100FileSHA(path, expected string) bool {
	if expected == "" || strings.Contains(path, "..") || strings.HasPrefix(path, "/") || strings.Contains(path, "\\") {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]) == expected
}

const r100ParentDir = ".cvf/runtime/owner_alibaba_r100_new4_20261015"
const r100FreshOwnerGrantID = "CCMAI-RUNTIME-100-OWNER-NEW4-20261015"
const r100SeedCommit = "b33d1fe83be35aa0e20d18590a5462e7f0ecd6fd"
const r100TLSDependencySHA = "3d2340131829f7afcb54d81be7d903c29fc8b36f160d065a05cd338a5df50dad"
const r100HistoricalReceipt = "docs/reviews/probes/r098_root/live_outcome_summary.json"

func r100ParentKnown() (string, bool) {
	parentRaw, err := os.ReadFile(r100ParentDir + "/owner_cap.json")
	if err != nil || len(parentRaw) > 65536 {
		return "", false
	}
	var parent struct {
		ApprovedBy              string `json:"approvedBy"`
		OwnerCap                int    `json:"ownerCap"`
		FreshOwnerGrantID       string `json:"freshOwnerGrantId"`
		FreshGrantCap           int    `json:"freshGrantCap"`
		HistoricalR098Calls     int    `json:"historicalR098Calls"`
		SeedCommit              string `json:"seedCommit"`
		HistoricalReceipt       string `json:"historicalReceipt"`
		HistoricalReceiptSha256 string `json:"historicalReceiptSha256"`
		Provider                string `json:"provider"`
		Model                   string `json:"model"`
		Region                  string `json:"region"`
		EndpointSha256          string `json:"endpointSha256"`
		AutomaticRetries        int    `json:"automaticRetries"`
		WorkerSlots             []int  `json:"workerSlots"`
	}
	if json.Unmarshal(parentRaw, &parent) != nil || parent.ApprovedBy != "Codex /root" || parent.OwnerCap != 4 || parent.FreshOwnerGrantID != r100FreshOwnerGrantID || parent.FreshGrantCap != 4 || parent.HistoricalR098Calls != 2 || parent.SeedCommit != r100SeedCommit || parent.HistoricalReceipt != r100HistoricalReceipt || parent.Provider != "Alibaba" || parent.Model != r100Model || parent.Region != "ap-southeast-1" || parent.EndpointSha256 != "b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f" || parent.AutomaticRetries != 0 || len(parent.WorkerSlots) != 4 || parent.WorkerSlots[0] != 1 || parent.WorkerSlots[1] != 2 || parent.WorkerSlots[2] != 3 || parent.WorkerSlots[3] != 4 {
		return "", false
	}
	prior, err := os.ReadFile(r100HistoricalReceipt)
	if err != nil || len(prior) > 65536 {
		return "", false
	}
	priorSHA := sha256.Sum256(prior)
	if hex.EncodeToString(priorSHA[:]) != parent.HistoricalReceiptSha256 {
		return "", false
	}
	var old []struct {
		Attempts int    `json:"attempts"`
		Model    string `json:"model"`
	}
	if json.Unmarshal(prior, &old) != nil || len(old) != 2 || old[0].Attempts != 1 || old[1].Attempts != 1 || old[0].Model != r100Model || old[1].Model != r100Model {
		return "", false
	}
	h := sha256.Sum256(parentRaw)
	return hex.EncodeToString(h[:]), true
}

func r100LocalObject(path string, out any) bool {
	return r100LocalObjectBound(path, 65536, out)
}

func r100LocalObjectBound(path string, max int64, out any) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, max+1))
	return err == nil && int64(len(raw)) <= max && json.Unmarshal(raw, out) == nil
}

func r100ExactDSN(port int) string {
	return fmt.Sprintf("root:r100-synthetic-only@tcp(127.0.0.1:%d)/r100_fixture?parseTime=true&charset=utf8mb4", port)
}

func r100StartOnce(name string) bool {
	file, err := os.OpenFile(".cvf/runtime/r100_live/"+name+".go-start", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return false
	}
	defer file.Close()
	_, err = fmt.Fprintln(file, time.Now().UTC().Format(time.RFC3339))
	return err == nil && file.Sync() == nil
}

func r100CampaignOpen() bool {
	deadline, ok := r100CampaignDeadline()
	return ok && time.Now().Before(deadline)
}

func r100CampaignDeadline() (time.Time, bool) {
	var campaign struct {
		StartedEpoch float64 `json:"startedEpoch"`
		MaxSeconds   int     `json:"maxSeconds"`
	}
	if !r100LocalObject(".cvf/runtime/r100_live/campaign.json", &campaign) || campaign.MaxSeconds != 300 {
		return time.Time{}, false
	}
	start := time.Unix(0, int64(campaign.StartedEpoch*float64(time.Second)))
	if time.Now().Before(start) {
		return time.Time{}, false
	}
	return start.Add(300 * time.Second), true
}

func r100Boundary(in r100Input) bool {
	var state struct {
		CurrentMode   string `json:"currentMode"`
		ActiveTranche string `json:"activeTranche"`
		ActiveHandoff string `json:"activeHandoff"`
		ActiveRole    string `json:"activeRole"`
	}
	if !r100LocalObjectBound("CVF_SESSION/ACTIVE_SESSION_STATE.json", 256<<10, &state) || state.ActiveTranche != "CCMAI-RUNTIME-100" || state.ActiveHandoff != "CVF_SESSION/handoffs/AGENT_HANDOFF_M2_LIVE_R100_2026-10-15.md" {
		return false
	}
	var ownership struct {
		Port         int               `json:"port"`
		ProtectedIDs map[string]string `json:"protectedIds"`
	}
	if !r100LocalObject(".cvf/runtime/r100_live/ownership.json", &ownership) || ownership.Port < 1024 || len(ownership.ProtectedIDs) != 3 || in.DSN != r100ExactDSN(ownership.Port) {
		return false
	}
	var pin struct {
		ContainerID string   `json:"containerId"`
		NetworkID   string   `json:"networkId"`
		ImageID     string   `json:"imageId"`
		VolumeIDs   []string `json:"volumeIds"`
	}
	if !r100LocalObject(".cvf/runtime/r100_live/container_pin.json", &pin) || len(pin.ContainerID) != 64 || len(pin.NetworkID) != 64 || !strings.HasPrefix(pin.ImageID, "sha256:") || len(pin.VolumeIDs) != 1 {
		return false
	}
	if in.Mode == "seed" {
		return state.CurrentMode == "BUILD" && (strings.Contains(state.ActiveRole, "IMPLEMENTATION_WORKER") || strings.Contains(state.ActiveRole, "REPAIR_WORKER")) && in.Role == "seed" && in.Key == "" && r100StartOnce("seed")
	}
	if in.Mode != "live" || in.Role != "worker" || len(in.Key) < 16 || len(in.Key) > 512 {
		return false
	}
	for _, ch := range in.Key {
		if ch < 33 || ch > 126 {
			return false
		}
	}
	if state.CurrentMode != "BUILD" || !(strings.Contains(state.ActiveRole, "IMPLEMENTATION_WORKER") || strings.Contains(state.ActiveRole, "REPAIR_WORKER")) {
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
		Fixture       r100Fixture `json:"fixture"`
		SeedBinarySha string      `json:"seedBinarySha256"`
	}
	if !r100LocalObject(".cvf/runtime/r100_live/fixture.json", &fixture) || fixture.ContainerID != pin.ContainerID || fixture.Port != ownership.Port || fixture.Fixture != in.Fixture {
		return false
	}
	var approval struct {
		BinarySha string `json:"binarySha256"`
	}
	if !r100LocalObject(".cvf/runtime/r100_live/root_approval.json", &approval) || fixture.SeedBinarySha != approval.BinarySha {
		return false
	}
	if !r100CampaignOpen() {
		return false
	}
	var claim struct {
		Role  string `json:"role"`
		State string `json:"state"`
	}
	if !r100LocalObject(".cvf/runtime/r100_live/"+in.Role+".claim", &claim) || claim.Role != in.Role || claim.State != "CONSUMED_BEFORE_HTTP" {
		return false
	}
	var dispatch struct {
		Role      string `json:"role"`
		BinarySha string `json:"binarySha256"`
	}
	if !r100LocalObject(".cvf/runtime/r100_live/"+in.Role+".dispatch", &dispatch) || dispatch.Role != in.Role || dispatch.BinarySha != approval.BinarySha {
		return false
	}
	return r100StartOnce(in.Role)
}

// The recorder delegates the exact validated request bytes to the original
// network transport. It records neither Authorization nor any other headers.
type r100Recorder struct {
	mu       sync.Mutex
	next     http.RoundTripper
	key      string
	endpoint string
	slot     int
	wire     r100Wire
}

func r100ConsumePhysicalSlot(slot int) bool {
	if slot < 1 || slot > 4 || !r100CampaignOpen() {
		return false
	}
	parentHash, ok := r100ParentKnown()
	if !ok {
		return false
	}
	var approval struct {
		ParentManifestSha256 string `json:"parentManifestSha256"`
	}
	if !r100LocalObject(".cvf/runtime/r100_live/root_approval.json", &approval) || approval.ParentManifestSha256 != parentHash {
		return false
	}
	entries, err := os.ReadDir(r100ParentDir)
	if err != nil || len(entries) != slot {
		return false
	}
	expectedEntries := map[string]bool{"owner_cap.json": true}
	for previous := 1; previous < slot; previous++ {
		expectedEntries[fmt.Sprintf("slot-%d.effect", previous)] = true
	}
	for _, entry := range entries {
		if entry.IsDir() || !expectedEntries[entry.Name()] {
			return false
		}
	}
	for previous := 1; previous < slot; previous++ {
		var prior struct {
			Role                 string `json:"role"`
			Slot                 int    `json:"slot"`
			ParentManifestSha256 string `json:"parentManifestSha256"`
		}
		if !r100LocalObject(fmt.Sprintf("%s/slot-%d.effect", r100ParentDir, previous), &prior) || prior.Role != "worker" || prior.Slot != previous || prior.ParentManifestSha256 != parentHash {
			return false
		}
	}
	path := fmt.Sprintf("%s/slot-%d.effect", r100ParentDir, slot)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return false
	}
	defer file.Close()
	value := map[string]any{"role": "worker", "slot": slot, "parentManifestSha256": parentHash, "consumedAt": time.Now().UTC().Format(time.RFC3339)}
	if err = json.NewEncoder(file).Encode(value); err != nil {
		return false
	}
	return file.Sync() == nil
}

func r100Redact(value, key string) string {
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

// Only bounded class names and booleans leave this function. Error strings,
// URLs, headers and credentials are never written to the receipt.
func r100TransportDiagnostic(err error) (string, bool, bool) {
	var urlErr *url.Error
	wrapped := errors.As(err, &urlErr)
	var timeoutErr net.Error
	timedOut := errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeoutErr) && timeoutErr.Timeout())
	if timedOut {
		return "TIMEOUT", true, wrapped
	}
	var unknownCA x509.UnknownAuthorityError
	var invalidCert x509.CertificateInvalidError
	if errors.As(err, &unknownCA) || errors.As(err, &invalidCert) {
		return "CERTIFICATE", false, wrapped
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "DNS", false, wrapped
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return "NETWORK_OPERATION", false, wrapped
	}
	if errors.Is(err, io.EOF) {
		return "EOF", false, wrapped
	}
	if strings.Contains(err.Error(), "malformed HTTP response") {
		return "HTTP_PROTOCOL", false, wrapped
	}
	return "UNKNOWN", false, wrapped
}

func (r *r100Recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.wire.Attempts != 0 {
		return nil, errors.New("R100_SECOND_NETWORK_ATTEMPT_BLOCKED")
	}
	if req.Method != http.MethodPost || req.URL.String() != strings.TrimSuffix(r.endpoint, "/")+"/chat/completions" || req.URL.Scheme != "https" || req.Header.Get("Authorization") != "Bearer "+r.key {
		return nil, errors.New("R100_REQUEST_SHAPE_REJECTED")
	}
	request, err := io.ReadAll(io.LimitReader(req.Body, 6001))
	if err != nil || len(request) > 6000 {
		return nil, errors.New("R100_REQUEST_SIZE_REJECTED")
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
	if json.Unmarshal(request, &shape) != nil || json.Unmarshal(request, &keys) != nil || len(keys) != 4 || shape.Model != r100Model || shape.Max != r100Tokens || shape.Thinking == nil || *shape.Thinking || len(shape.Messages) != 2 || shape.Messages[0].Role != "system" || shape.Messages[1].Role != "user" {
		return nil, errors.New("R100_REQUEST_SHAPE_REJECTED")
	}
	if !r100ConsumePhysicalSlot(r.slot) {
		r.wire.TransportCode = "OWNER_PHYSICAL_CAP_DENIED"
		return nil, errors.New("R100_OWNER_PHYSICAL_CAP_DENIED")
	}
	r.wire.Attempts = 1 // This marker precedes the physical transport attempt.
	r.wire.Request = r100Redact(string(request), r.key)
	req.Body = io.NopCloser(bytes.NewReader(request))
	response, transportErr := r.next.RoundTrip(req)
	if transportErr != nil {
		r.wire.TransportCode = "TRANSPORT_UNKNOWN"
		r.wire.TransportClass, r.wire.TransportTimeout, r.wire.URLWrapped = r100TransportDiagnostic(transportErr)
		return nil, errors.New("R100_TRANSPORT_UNKNOWN")
	}
	r.wire.Status = response.StatusCode
	body, readErr := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	_ = response.Body.Close()
	bounded := body
	if len(bounded) > 1<<20 {
		bounded = bounded[:1<<20]
	}
	r.wire.Response = r100Redact(string(bounded), r.key)
	if readErr != nil {
		r.wire.TransportCode = "RESPONSE_READ_FAILED"
		return nil, errors.New("R100_RESPONSE_READ_FAILED")
	}
	if len(body) > 1<<20 {
		r.wire.ResponseTooLarge = true
		return nil, errors.New("R100_RESPONSE_TOO_LARGE")
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	return response, nil
}

func r100Send(router http.Handler, token, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func r100SendBound(router http.Handler, token, method, path, body string, deadline time.Time) *httptest.ResponseRecorder {
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	req := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func r100Seed() (r100Fixture, error) {
	now := time.Now().UTC()
	f := r100Fixture{Tenant: pkg.NewUUID(), User: pkg.NewUUID(), Job: pkg.NewUUID(), Channel: pkg.NewUUID(), WaitConversation: pkg.NewUUID(), DenyJob: pkg.NewUUID(), DenyConversation: pkg.NewUUID()}
	for i := range f.Conversations {
		f.Conversations[i] = pkg.NewUUID()
	}
	deniedChannel := pkg.NewUUID()
	rows := []any{
		&models.Tenant{ID: f.Tenant, Name: "R100 fabricated", Slug: "r100-" + f.Tenant, Settings: "{}", CreatedAt: now, UpdatedAt: now},
		&models.User{ID: f.User, Email: f.User + "@example.invalid", PasswordHash: "fixture", CreatedAt: now, UpdatedAt: now},
		&models.UserTenant{UserID: f.User, TenantID: f.Tenant, Role: "owner", Permissions: "{}"},
		&models.Channel{ID: f.Channel, TenantID: f.Tenant, ChannelType: "pancake", Name: "R100 fabricated", ExternalID: f.Channel, CredentialsEncrypted: []byte("fixture"), IsDemoFixture: true, IsActive: true, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Channel{ID: deniedChannel, TenantID: f.Tenant, ChannelType: "pancake", Name: "R100 unmarked", ExternalID: deniedChannel, CredentialsEncrypted: []byte("fixture"), IsDemoFixture: false, IsActive: true, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Job{ID: f.Job, TenantID: f.Tenant, Name: "R100 selected intervention", JobType: "intervention", InputChannelIDs: fmt.Sprintf("[%q]", f.Channel), RulesConfig: "{}", Outputs: "[]", ScheduleType: "manual", IsActive: true, CreatedAt: now, UpdatedAt: now},
		&models.Job{ID: f.DenyJob, TenantID: f.Tenant, Name: "R100 denied intervention", JobType: "intervention", InputChannelIDs: fmt.Sprintf("[%q]", deniedChannel), RulesConfig: "{}", Outputs: "[]", ScheduleType: "manual", IsActive: true, CreatedAt: now, UpdatedAt: now},
		&models.InterventionBudget{ID: pkg.NewUUID(), TenantID: f.Tenant, JobID: f.Job, Cap: 4, Consumed: 0, CreatedAt: now},
		&models.Conversation{ID: f.WaitConversation, TenantID: f.Tenant, ChannelID: f.Channel, ExternalConversationID: f.WaitConversation, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
		&models.Conversation{ID: f.DenyConversation, TenantID: f.Tenant, ChannelID: deniedChannel, ExternalConversationID: f.DenyConversation, Metadata: "{}", CreatedAt: now, UpdatedAt: now},
	}
	for _, conversation := range f.Conversations {
		rows = append(rows, &models.Conversation{ID: conversation, TenantID: f.Tenant, ChannelID: f.Channel, ExternalConversationID: conversation, Metadata: "{}", CreatedAt: now, UpdatedAt: now})
	}
	for _, row := range rows {
		if err := db.DB.Create(row).Error; err != nil {
			return f, err
		}
	}
	for i, content := range [4]string{"Tôi khiếu nại đơn SYN-201 giao sai hàng.", "Tôi khiếu nại đơn SYN-202 bị giao thiếu.", "Tôi khiếu nại đơn SYN-203 giao nhầm địa chỉ.", "Tôi khiếu nại đơn SYN-204 đến trễ ba ngày."} {
		conversation := f.Conversations[i]
		if err := db.DB.Create(&models.Message{ID: pkg.NewUUID(), TenantID: f.Tenant, ConversationID: conversation, ExternalMessageID: pkg.NewUUID(), SenderType: "customer", Content: content, ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: now, CreatedAt: now}).Error; err != nil {
			return f, err
		}
	}
	return f, nil
}

func r100Classify(result r100Result, conversation string, slot int) string {
	if !result.DBReadOK || result.DetailStatus != 200 || result.PositiveStatus != 200 || result.Wire.Attempts != 1 || result.Wire.Status != 200 || result.Wire.TransportCode != "" || result.BudgetCap != 4 || result.BudgetConsumed != slot || result.Case == nil || result.Attempt == nil || result.Snapshot == nil || result.Case.ConversationID != conversation || result.Snapshot.ConversationID != conversation || result.Case.SnapshotDigest != result.Snapshot.Digest {
		return "HARD_STOP_OBSERVATION"
	}
	if result.Case.TenantID != result.Fixture.Tenant || result.Case.JobID != result.Fixture.Job || result.Case.RunID == "" || result.Case.SnapshotID == nil || *result.Case.SnapshotID != result.Snapshot.ID || result.Snapshot.TenantID != result.Fixture.Tenant || result.Snapshot.JobRunID != result.Case.RunID || result.Attempt.TenantID != result.Fixture.Tenant || result.Attempt.JobID != result.Fixture.Job || result.Attempt.RunID != result.Case.RunID || len(result.Messages) != 1 || result.Messages[0].TenantID != result.Fixture.Tenant || result.Messages[0].ConversationID != conversation || result.Messages[0].SenderType != "customer" {
		return "HARD_STOP_IDENTITY"
	}
	manifestSHA := sha256.Sum256([]byte(result.Snapshot.Manifest))
	if hex.EncodeToString(manifestSHA[:]) != result.Snapshot.Digest {
		return "HARD_STOP_SNAPSHOT"
	}
	var wire struct {
		Model string `json:"model"`
		Usage struct {
			Prompt     *int64 `json:"prompt_tokens"`
			Completion *int64 `json:"completion_tokens"`
		} `json:"usage"`
		Choices json.RawMessage `json:"choices"`
	}
	if json.Unmarshal([]byte(result.Wire.Response), &wire) != nil || wire.Model != r100Model || wire.Usage.Prompt == nil || wire.Usage.Completion == nil || *wire.Usage.Prompt < 0 || *wire.Usage.Completion < 0 || *wire.Usage.Completion > 768 || result.Attempt.State != "RESPONSE_RECEIVED" || result.Attempt.UsageState != "KNOWN" || result.Attempt.InputUsagePresence != "known" || result.Attempt.OutputUsagePresence != "known" || result.Attempt.InputTokens == nil || result.Attempt.OutputTokens == nil || *result.Attempt.InputTokens != *wire.Usage.Prompt || *result.Attempt.OutputTokens != *wire.Usage.Completion || result.Attempt.ResponseModel != r100Model {
		return "HARD_STOP_METADATA"
	}
	if result.Case.Execution == "ERROR" && result.Case.ReasonCode == "OUTPUT_INVALID" && len(result.Proposals) == 0 && len(result.Receipts) == 1 && result.Receipts[0].Code == "OUTPUT_INVALID" && result.Receipts[0].CaseID == result.Case.ID && result.Receipts[0].RunID == result.Case.RunID && result.Receipts[0].UsageState == "KNOWN" {
		return "SEMANTIC_INVALID"
	}
	var choices []struct {
		Finish  string `json:"finish_reason"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(wire.Choices, &choices) != nil || result.Case.Execution != "LLM_SUCCEEDED" || result.Case.Disposition != "PENDING_REVIEW" || len(result.Proposals) != 1 || len(choices) != 1 || choices[0].Finish != "stop" || strings.TrimSpace(choices[0].Message.Content) == "" {
		return "HARD_STOP_CASE"
	}
	p := result.Proposals[0]
	if p.CaseID != result.Case.ID || p.RunID != result.Case.RunID || p.SnapshotID != result.Snapshot.ID || p.SnapshotDigest != result.Snapshot.Digest || p.Model != r100Model || p.UsageState != "KNOWN" || p.SchemaVersion != "ccma.intervention.v1" || p.Kind != "complaint" || strings.TrimSpace(p.Explanation) == "" || strings.TrimSpace(p.NextAction) == "" {
		return "HARD_STOP_PROPOSAL"
	}
	var refs []struct {
		MessageID string `json:"message_id"`
		Quote     string `json:"quote"`
	}
	if json.Unmarshal([]byte(p.Evidence), &refs) != nil || len(refs) != 1 {
		return "HARD_STOP_EVIDENCE"
	}
	for _, m := range result.Messages {
		if m.ID == refs[0].MessageID && m.SenderType == "customer" && strings.Contains(m.Content, refs[0].Quote) && strings.Contains(strings.ToLower(refs[0].Quote), "khiếu nại") {
			return "VALID"
		}
	}
	return "HARD_STOP_EVIDENCE"
}

// The ready marker is written only after the full sanitized receipt is fsynced.
func r100PersistAttempt(result r100Result, key string) (string, bool) {
	clean, safe := r100SafeResult(result, key)
	if !safe {
		clean = r100MinimalSecretFailure(result)
	}
	name := fmt.Sprintf(".cvf/runtime/r100_live/attempt-%d.json", result.Slot)
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", false
	}
	_, writeErr := file.Write(clean)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return "", false
	}
	h := sha256.Sum256(clean)
	sha := hex.EncodeToString(h[:])
	ready, err := os.OpenFile(fmt.Sprintf(".cvf/runtime/r100_live/attempt-%d.ready", result.Slot), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", false
	}
	_, writeErr = ready.Write([]byte(sha))
	syncErr = ready.Sync()
	closeErr = ready.Close()
	return sha, safe && writeErr == nil && syncErr == nil && closeErr == nil
}

func r100WaitContinuation(slot int, nextConversation, receiptSHA string) bool {
	approvalRaw, err := os.ReadFile(".cvf/runtime/r100_live/root_approval.json")
	if err != nil {
		return false
	}
	approvalHash := sha256.Sum256(approvalRaw)
	parentHash, ok := r100ParentKnown()
	if !ok {
		return false
	}
	path := fmt.Sprintf(".cvf/runtime/r100_live/continuation-%d.json", slot)
	for r100CampaignOpen() {
		var token struct {
			ApprovedBy           string  `json:"approvedBy"`
			PriorSlot            int     `json:"priorSlot"`
			NextSlot             int     `json:"nextSlot"`
			NextConversation     string  `json:"nextConversation"`
			ReceiptSha256        string  `json:"receiptSha256"`
			ApprovalSha256       string  `json:"approvalSha256"`
			ParentManifestSha256 string  `json:"parentManifestSha256"`
			ExpiresEpoch         float64 `json:"expiresEpoch"`
		}
		if _, statErr := os.Stat(path); statErr == nil {
			if !r100LocalObject(path, &token) || token.ApprovedBy != "Codex /root" || token.PriorSlot != slot || token.NextSlot != slot+1 || token.NextConversation != nextConversation || token.ReceiptSha256 != receiptSHA || token.ApprovalSha256 != hex.EncodeToString(approvalHash[:]) || token.ParentManifestSha256 != parentHash || time.Now().UnixNano() >= int64(token.ExpiresEpoch*float64(time.Second)) || !r100ApprovedBinary() {
				return false
			}
			return true
		} else if !os.IsNotExist(statErr) {
			return false
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

func TestR100Application(t *testing.T) {
	if os.Getenv("R100_EXPLICIT_BINARY") != "1" {
		t.Skip("NOT_RUN: explicit R100 binary invocation required")
	}
	result := r100Result{SchemaVersion: 1, ErrorCode: "PRECONDITION_FAILED"}
	var in r100Input
	defer func() { r100Output(result, in.Key) }()
	// Approval is checked before stdin, which may contain a provider key.
	if !r100ApprovedBinary() {
		result.ErrorCode = "ROOT_APPROVAL_MISSING"
		return
	}
	if err := json.NewDecoder(io.LimitReader(os.Stdin, 20000)).Decode(&in); err != nil {
		result.ErrorCode = "INPUT_INVALID"
		return
	}
	result.Role = in.Role
	result.Fixture = in.Fixture
	if !r100Boundary(in) {
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
		fixture, err := r100Seed()
		if err != nil {
			result.ErrorCode = "SEED_FAILED"
			return
		}
		result.Fixture = fixture
		result.ErrorCode = ""
		return
	}
	if in.Mode != "live" || in.Role != "worker" || len(in.Key) < 16 || in.Endpoint == "" {
		result.ErrorCode = "LIVE_INPUT_INVALID"
		return
	}
	f := in.Fixture
	var budget models.InterventionBudget
	if err := db.DB.Where("tenant_id = ? AND job_id = ?", f.Tenant, f.Job).First(&budget).Error; err != nil || budget.Cap != 4 || budget.Consumed != 0 {
		result.ErrorCode = "BUDGET_PRECONDITION_FAILED"
		return
	}
	result.BudgetCap = budget.Cap
	isolatedTransport, err := r096IsolatedTransport()
	if err != nil {
		result.ErrorCode = "TRANSPORT_UNSUPPORTED"
		return
	}
	defer isolatedTransport.CloseIdleConnections()
	recorder := &r100Recorder{next: isolatedTransport, key: in.Key, endpoint: in.Endpoint, slot: 1}
	previousTransport := http.DefaultTransport
	http.DefaultTransport = recorder
	defer func() { http.DefaultTransport = previousTransport }()
	defer func() { result.Wire = recorder.wire }()
	provider, err := ai.NewOpenAIProvider(in.Key, r100Model, r100Tokens, in.Endpoint).WithAlibabaQwen37OnceThinkingDisabled()
	if err != nil {
		result.ErrorCode = "THINKING_OPTION_REJECTED"
		return
	}
	const fixedPrompt = "Analyze only the fabricated Vietnamese customer complaint. Return exactly one compact JSON object without markdown or aliases. Required top-level keys: schema, conversation_id, snapshot_digest, kind, explanation, next_action, evidence_refs. Set schema to ccma.intervention.v1; kind to complaint. Set conversation_id and snapshot_digest to the exact values in the next instruction. evidence_refs must be an array with exactly one object using keys message_id and quote. Copy the canonical customer message_id and a short exact Unicode quote containing 'khiếu nại' from the transcript. Give a short explanation and next_action. Do not invent IDs or quotes."
	handlers.InterventionRunner = &engine.InterventionService{Transport: provider, PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "r100-live-v1", Prompt: fixedPrompt, MaxRequestBytes: 6000, Timeout: 60 * time.Second}
	defer func() { handlers.InterventionRunner = nil }()
	middleware.SetJWTSecret(strings.Repeat("r", 32))
	token, err := middleware.GenerateAccessToken(f.User, "r100@example.invalid", false)
	if err != nil {
		result.ErrorCode = "AUTH_FAILED"
		return
	}
	router := api.SetupRouter(&config.Config{RateLimitPerIP: 1000, RateLimitPerUser: 1000})
	prefix := "/api/v1/tenants/" + f.Tenant
	request := func(job, conversation string) *httptest.ResponseRecorder {
		return r100Send(router, token, "POST", prefix+"/jobs/"+job+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, conversation))
	}
	wait := request(f.Job, f.WaitConversation)
	result.WaitStatus = wait.Code
	deny := request(f.DenyJob, f.DenyConversation)
	result.DenyStatus = deny.Code
	if wait.Code != 200 || !bytes.Contains(wait.Body.Bytes(), []byte(`"eligibility":"WAIT_DATA"`)) || deny.Code != 403 || recorder.wire.Attempts != 0 {
		result.ErrorCode = "NEGATIVE_PRECONDITION_FAILED"
		return
	}
	for slot, conversation := range f.Conversations {
		index := slot + 1
		deadline, deadlineOK := r100CampaignDeadline()
		if !r100ApprovedBinary() || !deadlineOK || !time.Now().Before(deadline) {
			result.ErrorCode = "BOUNDARY_CHANGED_OR_CAMPAIGN_EXPIRED"
			return
		}
		recorder.slot, recorder.wire = index, r100Wire{}
		result.Slot = index
		result.Case, result.Attempt, result.Snapshot = nil, nil, nil
		result.Proposals, result.Receipts, result.Messages = nil, nil, nil
		result.BudgetConsumed, result.DetailStatus, result.DBReadOK = -1, 0, false
		positive := r100SendBound(router, token, "POST", prefix+"/jobs/"+f.Job+"/intervention-run", fmt.Sprintf(`{"conversation_id":%q}`, conversation), deadline)
		result.PositiveStatus = positive.Code
		result.PositiveBody = r100Redact(positive.Body.String(), in.Key)
		var caseRow models.InterventionCase
		if db.DB.Where("tenant_id = ? AND job_id = ? AND conversation_id = ?", f.Tenant, f.Job, conversation).First(&caseRow).Error == nil {
			readOK := true
			result.Case = &caseRow
			detail := r100Send(router, token, "GET", prefix+"/intervention-cases/"+caseRow.ID, "")
			result.DetailStatus = detail.Code
			result.DetailBody = r100Redact(detail.Body.String(), in.Key)
			readOK = readOK && detail.Code == 200
			if db.DB.Where("case_id = ?", caseRow.ID).Order("revision ASC").Find(&result.Proposals).Error != nil {
				readOK = false
			}
			if db.DB.Where("case_id = ?", caseRow.ID).Order("created_at ASC").Find(&result.Receipts).Error != nil {
				readOK = false
			}
			var attempt models.InterventionAttempt
			if db.DB.Where("run_id = ?", caseRow.RunID).First(&attempt).Error == nil {
				result.Attempt = &attempt
			} else {
				readOK = false
			}
			var snapshot models.AnalysisSnapshot
			if db.DB.Where("job_run_id = ? AND conversation_id = ?", caseRow.RunID, conversation).First(&snapshot).Error == nil {
				result.Snapshot = &snapshot
			} else {
				readOK = false
			}
			if db.DB.Where("tenant_id = ? AND conversation_id = ?", f.Tenant, conversation).Order("sent_at ASC, id ASC").Find(&result.Messages).Error != nil || len(result.Messages) != 1 {
				readOK = false
			}
			result.DBReadOK = readOK
		}
		if db.DB.Where("tenant_id = ? AND job_id = ?", f.Tenant, f.Job).First(&budget).Error == nil && budget.Cap == 4 {
			result.BudgetConsumed = budget.Consumed
		} else {
			result.DBReadOK = false
		}
		result.Wire = recorder.wire
		result.Decision = r100Classify(result, conversation, index)
		if result.Decision == "VALID" {
			result.ErrorCode = ""
		} else {
			result.ErrorCode = result.Decision
		}
		receiptSHA, wrote := r100PersistAttempt(result, in.Key)
		if !wrote {
			result.ErrorCode = "RECEIPT_WRITE_FAILED"
			return
		}
		if result.Decision == "VALID" {
			return
		}
		if result.Decision != "SEMANTIC_INVALID" || index == 4 {
			return
		}
		if !r100WaitContinuation(index, f.Conversations[index], receiptSHA) {
			result.ErrorCode = "ROOT_CONTINUATION_MISSING_OR_INVALID"
			return
		}
	}
}

func TestR100RecorderOffline(t *testing.T) {
	key := "sk-R100-synthetic-offline-key" // cvf-allow-secret-fixture
	for _, echoed := range []string{
		`{"echo":"sk-R100-synthetic-offline-key"}`,      // cvf-allow-secret-fixture
		`{"echo":"\u0073k-R100-synthetic-offline-key"}`, // cvf-allow-secret-fixture
	} {
		result := r100Result{Slot: 1, Wire: r100Wire{Response: echoed, Attempts: 1}}
		if _, safe := r100SafeResult(result, key); safe {
			t.Fatal("credential echo accepted")
		}
		if bytes.Contains(r100MinimalSecretFailure(result), []byte(key)) {
			t.Fatal("credential in minimal receipt")
		}
	}
}
