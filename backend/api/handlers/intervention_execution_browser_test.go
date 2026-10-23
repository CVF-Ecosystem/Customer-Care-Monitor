package handlers_test

// R139 is opt-in. Ordinary `go test` compiles this file and performs no DB,
// browser, credential, or provider work. Runtime admission is checked before
// the first fixture connection; the source-only campaign cannot set R139_RUN.
import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type r139Admission struct {
	TrancheID            string            `json:"trancheId"`
	Kind                 string            `json:"kind"`
	SourceCommit         string            `json:"sourceCommit"`
	FixtureName          string            `json:"fixtureName"`
	BrowserReceiptSHA256 string            `json:"browserReceiptSha256"`
	BrowserReceiptPath   string            `json:"browserReceiptPath"`
	GoTestBinarySHA256   string            `json:"goTestBinarySha256"`
	SourceSHA256         map[string]string `json:"sourceSha256"`
	RuntimeRoot          string            `json:"runtimeRoot"`
	ExternalEvidenceDir  string            `json:"externalEvidenceDir"`
	BrowserSupported     bool              `json:"browserSupported"`
	OwnerApproved        bool              `json:"ownerApproved"`
	NativeExecutePOST    int               `json:"nativeExecutePOST"`
	BrowserExecutePOST   int               `json:"browserExecutePOST"`
	GrantCap             int               `json:"grantCap"`
	ProviderCap          int               `json:"providerCap"`
	GrantDeadlineUTC     string            `json:"grantDeadlineUTC"`
	Provider             string            `json:"provider"`
	Model                string            `json:"model"`
	Endpoint             string            `json:"endpoint"`
	CompletionTokens     int               `json:"completionTokens"`
	MaxRequestBytes      int64             `json:"maxRequestBytes"`
	Thinking             bool              `json:"thinking"`
	SyntheticNamespace   string            `json:"syntheticNamespace"`
	IssuerRole           string            `json:"issuerRole"`
	ActorRole            string            `json:"actorRole"`
	ConfigRef            string            `json:"configRef"`
	PromptVersion        string            `json:"promptVersion"`
	PromptSHA256         string            `json:"promptSha256"`
	BackendTree          string            `json:"backendTree"`
	FrontendTree         string            `json:"frontendTree"`
}

func r139BrowserRoute(method, path string) (auth, execute, read bool) {
	auth = method == http.MethodPost && (path == "/api/v1/auth/login" || path == "/api/v1/auth/logout" || path == "/api/v1/auth/refresh")
	execute = method == http.MethodPost && regexp.MustCompile(`^/api/v1/tenants/[^/]+/intervention-cases/[^/]+/rerun-requests/[^/]+/execute$`).MatchString(path)
	read = method == http.MethodGet || method == http.MethodHead
	return
}

func TestR139PureRouteAdmission(t *testing.T) {
	for _, path := range []string{"/api/v1/auth/login", "/api/v1/auth/logout", "/api/v1/auth/refresh"} {
		auth, execute, read := r139BrowserRoute(http.MethodPost, path)
		if !auth || execute || read { t.Fatalf("R139 auth route %q", path) }
	}
	for _, path := range []string{"/api/v1/auth/refresh/extra", "/api/v1/auth/reset", "/api/v1/tenants/t/intervention-cases/c/rerun-requests/r/execute/extra"} {
		auth, execute, read := r139BrowserRoute(http.MethodPost, path)
		if auth || execute || read { t.Fatalf("R139 mutation route %q", path) }
	}
	auth, execute, read := r139BrowserRoute(http.MethodPost, "/api/v1/tenants/t/intervention-cases/c/rerun-requests/r/execute")
	if auth || !execute || read { t.Fatal("R139 execute route") }
}

func r139Admit(t *testing.T) (r139Admission, string) {
	t.Helper()
	if os.Getenv("R139_RUN") != "1" {
		t.Skip("NOT_RUN: R139 runtime admission absent")
	}
	path := os.Getenv("R139_RUNTIME_ADMISSION")
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		t.Fatal("R139 admission path invalid")
	}
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	pathRoot, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(pathRoot, "..") || !regexp.MustCompile(`^CVF_SESSION[/\\]authority[/\\]CCMAI-RUNTIME-139-[A-Za-z0-9_-]+-ADMISSION\.json$`).MatchString(pathRoot) {
		t.Fatal("R139 admission outside authority")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		t.Fatal("R139 admission symlink")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("R139 admission missing")
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != os.Getenv("R139_RUNTIME_ADMISSION_SHA256") {
		t.Fatal("R139 admission hash mismatch")
	}
	var a r139Admission
	if json.Unmarshal(data, &a) != nil || a.TrancheID != "CCMAI-RUNTIME-139" || a.Kind != "OWNER_APPROVED_RUNTIME_EFFECT_ADMISSION" || !a.OwnerApproved || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(a.SourceCommit) || a.SourceCommit != os.Getenv("R139_SOURCE_COMMIT") || a.NativeExecutePOST != 8 || a.BrowserExecutePOST != 1 || a.GrantCap != 3 || a.ProviderCap != 1 || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(a.BrowserReceiptSHA256) || a.GrantDeadlineUTC == "" || os.Getenv("R139_GRANT_DEADLINE_UTC") != a.GrantDeadlineUTC || a.Provider != "alibaba_maas" || a.Model != "qwen3.7-flash" || a.Endpoint == "" || os.Getenv("R139_ALIBABA_ENDPOINT") != a.Endpoint || a.CompletionTokens != 768 || a.MaxRequestBytes != 6000 || a.Thinking || !regexp.MustCompile(`^ccma_r139_[a-z0-9_-]{1,64}$`).MatchString(a.SyntheticNamespace) || a.IssuerRole != "owner" || a.ActorRole != "writer" || a.ConfigRef != "r139-owned-config" || a.PromptVersion != "r139-synthetic-v1" {
		t.Fatal("R139 effect authority incomplete")
	}
	endpointDigest := sha256.Sum256([]byte(a.Endpoint))
	if hex.EncodeToString(endpointDigest[:]) != "b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f" {
		t.Fatal("R139 admitted endpoint mismatch")
	}
	promptDigest := sha256.Sum256([]byte(r139PromptText))
	if a.PromptSHA256 != hex.EncodeToString(promptDigest[:]) {
		t.Fatal("R139 admitted prompt mismatch")
	}
	deadline, deadlineErr := time.Parse(time.RFC3339, a.GrantDeadlineUTC)
	now := time.Now().UTC()
	if deadlineErr != nil || !strings.HasSuffix(a.GrantDeadlineUTC, "Z") || deadline.Format(time.RFC3339) != a.GrantDeadlineUTC || deadline.Before(now.Add(time.Minute)) || deadline.After(now.Add(24*time.Hour)) {
		t.Fatal("R139 owner grant deadline invalid")
	}
	show := exec.Command("git", "show", "HEAD:"+filepath.ToSlash(pathRoot))
	show.Dir = root
	committed, err := show.Output()
	if err != nil || !bytes.Equal(bytes.ReplaceAll(committed, []byte("\r\n"), []byte("\n")), bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))) {
		t.Fatal("R139 runtime admission uncommitted")
	}
	ancestor := exec.Command("git", "merge-base", "--is-ancestor", a.SourceCommit, "HEAD")
	ancestor.Dir = root
	if err := ancestor.Run(); err != nil {
		t.Fatal("R139 source commit not ancestor")
	}
	dirty := exec.Command("git", "status", "--porcelain", "--untracked-files=all")
	dirty.Dir = root
	dirtyRaw, dirtyErr := dirty.Output()
	if dirtyErr != nil || len(bytes.TrimSpace(dirtyRaw)) != 0 {
		t.Fatal("R139 source worktree dirty")
	}
	required := []string{"backend/api/handlers/intervention_execution_browser_test.go", "scripts/r139_execution_supervisor.py", "scripts/tests/test_r139_execution_supervisor.py", "scripts/r139_vite_config.mjs.template", "backend/go.mod", "backend/go.sum", "frontend/package.json", "frontend/package-lock.json", "frontend/vite.config.ts"}
	for _, relative := range required {
		if a.SourceSHA256[relative] == "" {
			t.Fatal("R139 required source pin absent")
		}
	}
	for relative, expected := range a.SourceSHA256 {
		if filepath.IsAbs(relative) || strings.Contains(relative, "\\") || strings.HasPrefix(relative, "../") || strings.Contains(relative, "/../") {
			t.Fatal("R139 source pin path invalid")
		}
		raw, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		sum := sha256.Sum256(raw)
		if readErr != nil || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(expected) || hex.EncodeToString(sum[:]) != expected {
			t.Fatal("R139 source pin mismatch")
		}
	}
	for _, tree := range []struct{ name, expected string }{{"backend", a.BackendTree}, {"frontend", a.FrontendTree}} {
		original := exec.Command("git", "rev-parse", a.SourceCommit+":"+tree.name)
		original.Dir = root
		current := exec.Command("git", "rev-parse", "HEAD:"+tree.name)
		current.Dir = root
		originalRaw, originalErr := original.Output()
		currentRaw, currentErr := current.Output()
		if originalErr != nil || currentErr != nil || !bytes.Equal(originalRaw, currentRaw) || strings.TrimSpace(string(currentRaw)) != tree.expected {
			t.Fatal("R139 product source tree changed")
		}
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	selfRaw, err := os.ReadFile(self)
	selfSum := sha256.Sum256(selfRaw)
	if err != nil || hex.EncodeToString(selfSum[:]) != a.GoTestBinarySHA256 {
		t.Fatal("R139 binary pin mismatch")
	}
	browserPath := filepath.Join(root, filepath.FromSlash(a.BrowserReceiptPath))
	browserResolved, err := filepath.EvalSymlinks(browserPath)
	browserRaw, readErr := os.ReadFile(browserPath)
	browserSum := sha256.Sum256(browserRaw)
	if err != nil || readErr != nil || browserResolved != browserPath || hex.EncodeToString(browserSum[:]) != a.BrowserReceiptSHA256 {
		t.Fatal("R139 browser proof receipt mismatch")
	}
	var browserProof map[string]any
	if json.Unmarshal(browserRaw, &browserProof) != nil || !a.BrowserSupported || browserProof["status"] != "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE" || browserProof["browserId"] != "iab" || browserProof["controlObserved"] != true || browserProof["captureObserved"] != true || browserProof["closeObserved"] != true || browserProof["ownedTabsFinal"] != float64(0) || (browserProof["protocol"] != "http" && browserProof["protocol"] != "https") {
		t.Fatal("R139 supported browser proof absent")
	}
	if !filepath.IsAbs(a.RuntimeRoot) || filepath.Clean(a.RuntimeRoot) != a.RuntimeRoot || os.Getenv("R139_RUNTIME_ROOT") != a.RuntimeRoot {
		t.Fatal("R139 runtime evidence root mismatch")
	}
	runtimeRel, err := filepath.Rel(filepath.Join(root, ".cvf", "runtime"), a.RuntimeRoot)
	if err != nil || runtimeRel == "." || strings.HasPrefix(runtimeRel, "..") {
		t.Fatal("R139 runtime root outside ignored area")
	}
	expectedEvidence := filepath.Join(filepath.Dir(root), ".ccmai_local_evidence", "r139_execution_proof_20261023", "fixture_r4")
	if a.ExternalEvidenceDir != expectedEvidence || os.Getenv("R139_EVIDENCE_DIR") != expectedEvidence {
		t.Fatal("R139 external evidence location mismatch")
	}
	evidenceResolved, err := filepath.EvalSymlinks(expectedEvidence)
	if err != nil || evidenceResolved != expectedEvidence {
		t.Fatal("R139 external evidence symlink")
	}
	entries, err := os.ReadDir(expectedEvidence)
	if err != nil || len(entries) != 1 || entries[0].Name() != "fixture_identity.json" {
		t.Fatal("R139 external evidence is not fresh identity-only")
	}
	if engine.WaitDataExecutor != nil {
		t.Fatal("R139 inherited executor")
	}
	if os.Getenv("R101_MYSQL_DSN") != "" || os.Getenv("R102_MYSQL_DSN") != "" {
		t.Fatal("R139 inherited DSN")
	}
	identityPath := os.Getenv("R139_FIXTURE_IDENTITY")
	if identityPath != filepath.Join(expectedEvidence, "fixture_identity.json") {
		t.Fatal("R139 identity not external")
	}
	identityResolved, err := filepath.EvalSymlinks(identityPath)
	if err != nil || identityResolved != identityPath || !filepath.IsAbs(identityPath) {
		t.Fatal("R139 identity path invalid")
	}
	identityRaw, err := os.ReadFile(identityPath)
	if err != nil {
		t.Fatal("R139 fixture identity unavailable")
	}
	var identity struct {
		TrancheID string `json:"trancheId"`
		Host      string `json:"host"`
		Port      int    `json:"port"`
		Database  string `json:"database"`
		Name      string `json:"name"`
		Nonce     string `json:"nonce"`
	}
	if json.Unmarshal(identityRaw, &identity) != nil || identity.TrancheID != a.TrancheID || identity.Name != a.FixtureName || identity.Host != "127.0.0.1" || identity.Port < 1024 || identity.Port > 65535 || !regexp.MustCompile(`^ccma_r139_[a-z0-9_]{1,32}$`).MatchString(identity.Database) || !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(identity.Nonce) {
		t.Fatal("R139 fixture identity mismatch")
	}
	dsn := os.Getenv("R139_MYSQL_DSN")
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil || parsed == nil || parsed.Net != "tcp" || parsed.Addr != fmt.Sprintf("127.0.0.1:%d", identity.Port) || parsed.DBName != identity.Database {
		t.Fatal("R139 DSN identity mismatch")
	}
	// Supervisor independently proves the committed admission and Docker owner.
	// This test repeats the exact nonce table check before compatibility helpers.
	probe, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("R139 nonce probe open")
	}
	defer probe.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var count int
	var nonce string
	if err := probe.QueryRowContext(ctx, "SELECT COUNT(*), MIN(nonce) FROM ccma_r139_fixture_owner").Scan(&count, &nonce); err != nil || count != 1 || nonce != identity.Nonce {
		t.Fatal("R139 nonce table does not prove ownership")
	}
	return a, dsn
}

type r139Trap struct{ calls atomic.Int32 }

func (p *r139Trap) AnalyzeChatOnce(context.Context, string, string, int64) (ai.AIResponse, error) {
	p.calls.Add(1)
	return ai.AIResponse{}, errors.New("r139_native_call_trap_no_ai_output")
}

const r139PromptText = "Analyze only fabricated Vietnamese complaint; return the required JSON."

func r139Service(transport engine.InterventionTransport, admission r139Admission) *engine.InterventionService {
	return &engine.InterventionService{Transport: transport, Provider: admission.Provider, Model: admission.Model, ConfigRef: admission.ConfigRef, PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: admission.PromptVersion, Prompt: r139PromptText, MaxRequestBytes: admission.MaxRequestBytes, Timeout: 60 * time.Second}
}

func r139Request(t *testing.T, f *r102Fixture) models.InterventionRerunRequest {
	t.Helper()
	f.source("Tôi khiếu nại và chờ đơn hàng", "text")
	if rec := f.send(http.MethodPost, f.tenant, f.writer, "/data-revisions", f.revisionBody(1)); rec.Code != 201 {
		t.Fatalf("R139 revision: %d", rec.Code)
	}
	var revision models.InterventionDataRevision
	r102DB(t, db.DB.Where("tenant_id = ? AND case_id = ?", f.tenant, f.caseID).Take(&revision).Error)
	if rec := f.send(http.MethodPost, f.tenant, f.writer, "/rerun-requests", f.requestBody(2, revision, "r139-"+pkg.NewUUID())); rec.Code != 201 {
		t.Fatalf("R139 request: %d", rec.Code)
	}
	var request models.InterventionRerunRequest
	r102DB(t, db.DB.Where("tenant_id = ? AND case_id = ?", f.tenant, f.caseID).Take(&request).Error)
	return request
}

func r139SendRaw(f *r102Fixture, actor, tenant, caseID, requestID, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/"+tenant+"/intervention-cases/"+caseID+"/rerun-requests/"+requestID+"/execute", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.token[actor])
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func r139JSONNew(t *testing.T, path string, value any) {
	t.Helper()
	if err := r139WriteNew(path, value); err != nil {
		t.Fatal(err)
	}
}

func r139WriteNew(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(append(raw, '\n')); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// A final receipt reports only observations already made. Incomplete tests
// keep a separate HOLD receipt; they cannot produce the completed terminal.
type r139FinalEvidence struct {
	Phase                 string `json:"phase"`
	Disposition           string `json:"disposition"`
	Failed                bool   `json:"failed"`
	PhysicalSlot          string `json:"physicalSlot"`
	RequestCaptured       bool   `json:"requestCaptured"`
	ResponseCaptured      bool   `json:"responseCaptured"`
	ProviderHTTPStatus    int    `json:"providerHTTPStatus"`
	ExecutePOSTObserved   int32  `json:"executePOSTObserved"`
	AuthPOSTObserved      int32  `json:"authPOSTObserved"`
	NativeExecutePOST     int    `json:"nativeExecutePOST"`
	NativeNegativeReads   int    `json:"nativeNegativeReads"`
	NativePositiveReads   int    `json:"nativePositiveReads"`
	GrantIssuances        int    `json:"grantIssuances"`
	DurableFingerprint    string `json:"durableFingerprint,omitempty"`
	FingerprintError      string `json:"fingerprintError,omitempty"`
	ExecutionQueryPresent bool   `json:"executionQueryPresent"`
	CaseQueryPresent      bool   `json:"caseQueryPresent"`
	ExecutionState        string `json:"executionState,omitempty"`
	ExecutionTerminal     string `json:"executionTerminal,omitempty"`
	CaseWorkState         string `json:"caseWorkState,omitempty"`
}

func r139WriteFinalEvidence(path string, evidence r139FinalEvidence) error {
	if evidence.Phase == "" || evidence.Disposition == "" || evidence.PhysicalSlot == "" {
		return errors.New("r139 final evidence incomplete")
	}
	return r139WriteNew(path, evidence)
}

func TestR139PureFinalEvidenceSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "final.json")
	evidence := r139FinalEvidence{Phase: "VALIDATED_PUBLICATION", Disposition: "PASS", PhysicalSlot: "MAY_HAVE_STARTED", RequestCaptured: true, ResponseCaptured: true, ExecutionState: "PUBLISHED"}
	if err := r139WriteFinalEvidence(path, evidence); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved r139FinalEvidence
	if json.Unmarshal(raw, &saved) != nil || saved != evidence {
		t.Fatal("R139 pure final success evidence mismatch")
	}
}

func TestR139PureFinalEvidenceFatalGoexit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "final.json")
	done := make(chan error, 1)
	go func() {
		defer close(done)
		defer func() {
			done <- r139WriteFinalEvidence(path, r139FinalEvidence{Phase: "BROWSER_STOPPED_TERMINAL_OBSERVATION", Disposition: "HOLD", Failed: true, PhysicalSlot: "MAY_HAVE_STARTED", RequestCaptured: true, ResponseCaptured: false, ExecutionState: "MANUAL_HOLD"})
		}()
		runtime.Goexit()
	}()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved r139FinalEvidence
	if json.Unmarshal(raw, &saved) != nil || saved.Disposition != "HOLD" || !saved.Failed || saved.ExecutionState != "MANUAL_HOLD" || saved.ResponseCaptured {
		t.Fatal("R139 pure fatal evidence mismatch")
	}
}

func TestR139PureFinalEvidenceIOFailure(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "final.json")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r139WriteFinalEvidence(path, r139FinalEvidence{Phase: "ANY", Disposition: "HOLD", PhysicalSlot: "NOT_RESERVED"}); !errors.Is(err, os.ErrExist) {
		t.Fatalf("R139 pure final IO failure missing: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "original" {
		t.Fatal("R139 pure final IO failure changed original")
	}
}

type r139PureRoundTrip func(*http.Request) (*http.Response, error)

func (fn r139PureRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestR139PureJournalRequestIOFailureBlocksNext(t *testing.T) {
	dir := t.TempDir()
	requestPath := filepath.Join(dir, "provider_request_captured_private.json")
	if err := os.WriteFile(requestPath, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	journal := r139JournalTransport{evidenceDir: dir, key: "synthetic-secret", next: r139PureRoundTrip(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, errors.New("should not execute")
	})}
	req := httptest.NewRequest(http.MethodPost, "https://example.invalid/chat/completions", strings.NewReader(`{"synthetic":true}`))
	if response, err := journal.RoundTrip(req); err == nil || response != nil || called {
		t.Fatal("R139 pure journal IO failure allowed next effect")
	}
}

func TestR139PureJournalCapturedResponseSurvivesGoexit(t *testing.T) {
	dir := t.TempDir()
	done := make(chan error, 1)
	go func() {
		defer close(done)
		journal := r139JournalTransport{evidenceDir: dir, key: "synthetic-secret", next: r139PureRoundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader(`{"actual":"captured"}`))}, nil
		})}
		req := httptest.NewRequest(http.MethodPost, "https://example.invalid/chat/completions", strings.NewReader(`{"synthetic":true}`))
		response, err := journal.RoundTrip(req)
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		done <- err
		runtime.Goexit()
	}()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "provider_response_captured_private.json"))
	var saved struct {
		HTTPStatus int    `json:"httpStatus"`
		Response   string `json:"response"`
	}
	if err != nil || json.Unmarshal(raw, &saved) != nil || saved.HTTPStatus != 202 || saved.Response != `{"actual":"captured"}` {
		t.Fatal("R139 pure journal lost captured response after fatal exit")
	}
}

func TestR139PureJournalUnsafeResponseHasNoInventedReceipt(t *testing.T) {
	dir := t.TempDir()
	journal := r139JournalTransport{evidenceDir: dir, key: "synthetic-secret", next: r139PureRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("not-json"))}, nil
	})}
	req := httptest.NewRequest(http.MethodPost, "https://example.invalid/chat/completions", strings.NewReader(`{"synthetic":true}`))
	if response, err := journal.RoundTrip(req); err == nil || response != nil {
		t.Fatal("R139 pure journal accepted unsafe response")
	}
	if _, err := os.Stat(filepath.Join(dir, "provider_response_captured_private.json")); !os.IsNotExist(err) {
		t.Fatal("R139 pure journal invented unsafe response")
	}
}

// The inner transport is invoked only after a durable, secret-free request
// journal. A response is journaled only when its actual bytes are readable
// and safe. The historical one-slot transport still owns slot reservation.
type r139JournalTransport struct {
	next        http.RoundTripper
	evidenceDir string
	key         string
}

func (j r139JournalTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, err := io.ReadAll(io.LimitReader(req.Body, 6001))
	if err != nil || len(raw) > 6000 || !r116SafeJSON(raw, j.key) {
		return nil, errors.New("r139 journal request invalid")
	}
	h := sha256.Sum256(raw)
	if err := r139WriteNew(filepath.Join(j.evidenceDir, "provider_request_captured_private.json"), map[string]any{"provider": "alibaba_maas", "model": "qwen3.7-flash", "request": string(raw), "requestSha256": hex.EncodeToString(h[:]), "state": "MAY_HAVE_STARTED"}); err != nil {
		return nil, fmt.Errorf("r139 request journal: %w", err)
	}
	req.Body = io.NopCloser(bytes.NewReader(raw))
	response, err := j.next.RoundTrip(req)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, err
	}
	if response == nil || response.Body == nil {
		return nil, errors.New("r139 response missing")
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	_ = response.Body.Close()
	if readErr != nil || !r116SafeJSON(body, j.key) {
		return nil, errors.New("r139 response unreadable or unsafe")
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	responseHash := sha256.Sum256(body)
	if err := r139WriteNew(filepath.Join(j.evidenceDir, "provider_response_captured_private.json"), map[string]any{"provider": "alibaba_maas", "model": "qwen3.7-flash", "httpStatus": response.StatusCode, "response": string(body), "responseSha256": hex.EncodeToString(responseHash[:])}); err != nil {
		return nil, fmt.Errorf("r139 response journal: %w", err)
	}
	return response, nil
}

func r139Append(path string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write(append(raw, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

type r139ResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *r139ResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func r139Grant(t *testing.T, f *r102Fixture, request models.InterventionRerunRequest, service *engine.InterventionService, deadlineUTC string) models.InterventionExecutionGrant {
	t.Helper()
	deadline, err := time.Parse(time.RFC3339, deadlineUTC)
	if err != nil || !strings.HasSuffix(deadlineUTC, "Z") || deadline.Format(time.RFC3339) != deadlineUTC {
		t.Fatal("R139 owner grant deadline invalid")
	}
	r102Must(t, &models.InterventionBudget{ID: pkg.NewUUID(), TenantID: f.tenant, JobID: f.job, Cap: 1, Consumed: 0, CreatedAt: time.Now().UTC()})
	grant, replayed, err := engine.IssueSyntheticInterventionGrant(context.Background(), f.tenant, f.owner, f.writer, f.caseID, request.ID, "R139 admitted synthetic fixture", deadline.UTC(), service)
	if err != nil || replayed || grant == nil {
		t.Fatalf("R139 server issuer: %v", err)
	}
	return *grant
}

// This uses the existing synthetic M3 seed without its old browser entrypoint.
// The WAIT_DATA case is projected into the R102 request helpers; M3's separate
// historical OPEN/proposal seed is presentation data and never AI proof.
// Unlike r102StartWithDSN, this fixture does not delete the final tenant graph.
func r139Start(t *testing.T, dsn string) *r102Fixture {
	t.Helper()
	t.Setenv("R101_MYSQL_DSN", dsn)
	m := m3FixtureStart(t)
	var wait models.InterventionCase
	r102DB(t, db.DB.Where("id = ?", m.waitID).Take(&wait).Error)
	if wait.SnapshotID == nil || wait.WorkState != "WAITING_ON_DATA" {
		t.Fatal("R139 WAIT_DATA seed missing")
	}
	f := &r102Fixture{t: t, tenant: m.tenant, other: m.otherTenant, owner: m.owner, writer: m.member, noJobs: m.assignee, reader: m.reader, conv: wait.ConversationID, job: wait.JobID, caseID: wait.ID, snap: *wait.SnapshotID, run: wait.RunID, originalDigest: wait.SnapshotDigest, token: m.token, router: m.router}
	r102DB(t, db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.writer).Update("permissions", `{"intervention_cases":"rw","jobs":"w"}`).Error)
	r102DB(t, db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.noJobs).Update("permissions", `{"intervention_cases":"rw"}`).Error)
	r102Must(t, &models.UserTenant{UserID: f.owner, TenantID: f.other, Role: "owner", Permissions: "{}"})
	return f
}

type r139BrandedTransport struct{ inner engine.InterventionTransport }

func (p r139BrandedTransport) AnalyzeChatOnce(ctx context.Context, prompt, transcript string, maxBytes int64) (ai.AIResponse, error) {
	response, err := p.inner.AnalyzeChatOnce(ctx, prompt, transcript, maxBytes)
	if response.Provider != "" {
		response.Provider = "alibaba_maas"
	}
	return response, err
}

func r139Count(t *testing.T, model any, clause string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.DB.Model(model).Where(clause, args...).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func r139FingerprintRaw(tenant string) (string, error) {
	tables := []string{"intervention_cases", "intervention_data_revisions", "intervention_rerun_requests", "intervention_execution_grants", "intervention_executions", "intervention_attempts", "intervention_budgets", "intervention_receipts", "intervention_proposals", "intervention_human_events", "job_runs", "analysis_snapshots", "messages", "user_tenants"}
	all := map[string][]map[string]any{}
	for _, table := range tables {
		var rows []map[string]any
		order := "id ASC"
		if table == "user_tenants" {
			order = "user_id ASC"
		}
		if err := db.DB.Table(table).Where("tenant_id = ?", tenant).Order(order).Find(&rows).Error; err != nil {
			return "", fmt.Errorf("fingerprint %s: %w", table, err)
		}
		all[table] = rows
	}
	raw, err := json.Marshal(all)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), nil
}

func r139Fingerprint(t *testing.T, tenant string) string {
	t.Helper()
	fingerprint, err := r139FingerprintRaw(tenant)
	if err != nil {
		t.Fatal(err)
	}
	return fingerprint
}

func r139ReadNoWrite(t *testing.T, f *r102Fixture, actor, tenant, path string, want int, handle string, permitHandle bool, expectedBody ...string) map[string]any {
	t.Helper()
	before := r139Fingerprint(t, f.tenant)
	rec := f.send(http.MethodGet, tenant, actor, path, nil)
	after := r139Fingerprint(t, f.tenant)
	if before != after || rec.Code != want || !strings.Contains(rec.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("R139 read status=%d want=%d durableEqual=%v", rec.Code, want, before == after)
	}
	if permitHandle {
		if handle == "" || !bytes.Contains(rec.Body.Bytes(), []byte(handle)) {
			t.Fatal("R139 current writer handle absent")
		}
	} else if bytes.Contains(rec.Body.Bytes(), []byte(`"grant_handle"`)) || (handle != "" && bytes.Contains(rec.Body.Bytes(), []byte(handle))) {
		t.Fatal("R139 denied/read-only handle leaked")
	}
	for _, expected := range expectedBody {
		if !bytes.Contains(rec.Body.Bytes(), []byte(expected)) {
			t.Fatalf("R139 read response missing expected field %q", expected)
		}
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("R139 read response JSON invalid: %v", err)
	}
	h := sha256.Sum256(rec.Body.Bytes())
	return map[string]any{"status": rec.Code, "durableBefore": before, "durableAfter": after, "bodySha256": hex.EncodeToString(h[:]), "state": response["state"], "error": response["error"], "executionPresent": response["execution_present"], "phase": response["phase"], "publishedCaseVersion": response["published_case_version"], "proposalAvailable": response["proposal_available"], "grantHandlePresent": bytes.Contains(rec.Body.Bytes(), []byte(`"grant_handle"`)), "deniedHandleAbsent": !bytes.Contains(rec.Body.Bytes(), []byte(`"grant_handle"`)) && (handle == "" || !bytes.Contains(rec.Body.Bytes(), []byte(handle))), "noStore": strings.Contains(rec.Header().Get("Cache-Control"), "no-store")}
}

func r139PostDeniedNoWrite(t *testing.T, f *r102Fixture, actor, tenant, caseID, requestID, body, name string, want int, expectedError string) map[string]any {
	t.Helper()
	before := r139Fingerprint(t, f.tenant)
	rec := r139SendRaw(f, actor, tenant, caseID, requestID, body)
	after := r139Fingerprint(t, f.tenant)
	var response map[string]any
	if json.Unmarshal(rec.Body.Bytes(), &response) != nil || len(response) != 1 || response["error"] != expectedError || before != after || rec.Code != want {
		t.Fatalf("R139 rejected POST %s status=%d want=%d durableEqual=%v", name, rec.Code, want, before == after)
	}
	h := sha256.Sum256(rec.Body.Bytes())
	return map[string]any{"slot": 0, "name": name, "status": rec.Code, "error": expectedError, "durableBefore": before, "durableAfter": after, "bodySha256": hex.EncodeToString(h[:])}
}

func TestR139IntegratedExecutionProof(t *testing.T) {
	a, dsn := r139Admit(t)
	evidenceDir := os.Getenv("R139_EVIDENCE_DIR")
	final := r139FinalEvidence{Phase: "ADMITTED_BEFORE_NATIVE", Disposition: "HOLD", PhysicalSlot: "NOT_RESERVED"}
	r139JSONNew(t, filepath.Join(evidenceDir, "go_evidence_started.json"), final)
	var executeCount, authCount atomic.Int32
	var wire *r116WireTransport
	nativeNegativeReadCount, nativePositiveReadCount := 0, 0
	nativeGrantIssuances, nativeExecutePostCount := 0, 0
	t.Cleanup(func() {
		final.Failed = t.Failed()
		final.ExecutePOSTObserved = executeCount.Load()
		final.AuthPOSTObserved = authCount.Load()
		final.NativeExecutePOST = nativeExecutePostCount
		final.NativeNegativeReads = nativeNegativeReadCount
		final.NativePositiveReads = nativePositiveReadCount
		final.GrantIssuances = nativeGrantIssuances
		if _, err := os.Stat(filepath.Join(os.Getenv("R139_EFFECT_DIR"), "slot-1.effect")); err == nil {
			final.PhysicalSlot = "MAY_HAVE_STARTED"
		} else if !os.IsNotExist(err) {
			final.PhysicalSlot = "UNKNOWN"
		}
		if _, err := os.Stat(filepath.Join(evidenceDir, "provider_request_captured_private.json")); err == nil {
			final.RequestCaptured = true
		}
		if _, err := os.Stat(filepath.Join(evidenceDir, "provider_response_captured_private.json")); err == nil {
			final.ResponseCaptured = true
		}
		if wire != nil {
			wire.mu.Lock()
			final.ProviderHTTPStatus = wire.status
			wire.mu.Unlock()
		}
		if final.Failed {
			final.Disposition = "HOLD"
		}
		if err := r139WriteFinalEvidence(filepath.Join(evidenceDir, "go_evidence_final.json"), final); err != nil {
			t.Errorf("R139 final evidence write: %v", err)
		}
	})
	if a.FixtureName == "" {
		t.Fatal("R139 fixture identity absent")
	}
	trap := &r139Trap{}
	issueGrant := func(t *testing.T, f *r102Fixture, request models.InterventionRerunRequest, service *engine.InterventionService) models.InterventionExecutionGrant {
		grant := r139Grant(t, f, request, service, a.GrantDeadlineUTC)
		nativeGrantIssuances++
		return grant
	}
	deniedPost := func(t *testing.T, f *r102Fixture, actor, tenant, caseID, requestID, body, name string, want int, expectedError string) map[string]any {
		nativeExecutePostCount++
		return r139PostDeniedNoWrite(t, f, actor, tenant, caseID, requestID, body, name, want, expectedError)
	}
	// The first native group is intentionally software-only: transport returns
	// an error with no AI output, and every reserved rejection must precede it.
	// Positive publication requires the separately admitted one-call transport.
	if !t.Run("native-negative", func(t *testing.T) {
		f := r139Start(t, dsn)
		request := r139Request(t, f)
		engine.WaitDataExecutor = r139Service(trap, a)
		defer func() { engine.WaitDataExecutor = nil }()
		readinessPath := "/rerun-requests/" + request.ID + "/execution-readiness"
		readLedger := []map[string]any{}
		read := func(name string, want int, handle, expected string) {
			row := r139ReadNoWrite(t, f, f.writer, f.tenant, readinessPath, want, handle, false, expected)
			row["name"] = name
			readLedger = append(readLedger, row)
		}
		read("grant-absent", 200, "", `"state":"AWAITING_OPERATOR_AUTHORITY"`)
		base := fmt.Sprintf(`{"expected_case_version":3,"grant_id":%q}`, pkg.NewUUID())
		cases := []struct {
			name, body                       string
			actor, tenant, caseID, requestID string
			want                             int
			errorText                        string
		}{
			{"malformed-json", `{"expected_case_version":`, f.writer, f.tenant, f.caseID, request.ID, 400, "invalid_request"},
			{"unknown-field", `{"expected_case_version":3,"grant_id":"x","extra":1}`, f.writer, f.tenant, f.caseID, request.ID, 400, "invalid_request"},
			{"missing-version", `{"grant_id":"x"}`, f.writer, f.tenant, f.caseID, request.ID, 400, "invalid_request"},
			{"stale-version", strings.Replace(base, `"expected_case_version":3`, `"expected_case_version":2`, 1), f.writer, f.tenant, f.caseID, request.ID, 409, "case_conflict"},
		}
		ledger := make([]map[string]any, 0, len(cases))
		for _, c := range cases {
			row := deniedPost(t, f, c.actor, c.tenant, c.caseID, c.requestID, c.body, c.name, c.want, c.errorText)
			row["slot"] = len(ledger) + 1
			ledger = append(ledger, row)
		}
		if trap.calls.Load() != 0 {
			t.Fatal("R139 native call trap reached")
		}
		// One sacrificial grant is bound to a different actor for slot five,
		// then restored and expired for slot six. Neither can call transport.
		grant := issueGrant(t, f, request, engine.WaitDataExecutor)
		// Every read below compares complete tenant rows before and after the
		// GET. Mutation of each prerequisite is test setup, then restored.
		r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", grant.ID).Update("state", "CONSUMED").Error)
		read("consumed-grant", 200, grant.ID, `"state":"AWAITING_OPERATOR_AUTHORITY"`)
		r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", grant.ID).Update("state", "ISSUED").Error)
		engine.WaitDataExecutor.ConfigRef = "r139-invalid-config"
		read("config-mismatch", 200, grant.ID, `"state":"AWAITING_OPERATOR_AUTHORITY"`)
		engine.WaitDataExecutor.ConfigRef = "r139-owned-config"
		r102DB(t, db.DB.Model(&models.InterventionBudget{}).Where("tenant_id = ? AND job_id = ?", f.tenant, f.job).Update("cap", 0).Error)
		read("budget-exhausted", 200, grant.ID, `"state":"STALE_OR_INELIGIBLE"`)
		r102DB(t, db.DB.Model(&models.InterventionBudget{}).Where("tenant_id = ? AND job_id = ?", f.tenant, f.job).Update("cap", 1).Error)
		r102DB(t, db.DB.Model(&models.InterventionCase{}).Where("id = ?", f.caseID).Update("version", 4).Error)
		read("case-version-changed", 200, grant.ID, `"state":"STALE_OR_INELIGIBLE"`)
		r102DB(t, db.DB.Model(&models.InterventionCase{}).Where("id = ?", f.caseID).Update("version", 3).Error)
		r102DB(t, db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.writer).Update("permissions", `{"intervention_cases":"r","jobs":"w"}`).Error)
		read("writer-permission-revoked", 403, grant.ID, `"error":"permission_denied"`)
		r102DB(t, db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.writer).Update("permissions", `{"intervention_cases":"rw","jobs":"w"}`).Error)
		var message models.Message
		r102DB(t, db.DB.Where("tenant_id = ? AND conversation_id = ?", f.tenant, f.conv).Take(&message).Error)
		originalContent := message.Content
		r102DB(t, db.DB.Model(&models.Message{}).Where("id = ?", message.ID).Update("content", "Đổi nội dung tổng hợp").Error)
		read("source-changed", 200, grant.ID, `"state":"STALE_OR_INELIGIBLE"`)
		r102DB(t, db.DB.Model(&models.Message{}).Where("id = ?", message.ID).Update("content", originalContent).Error)
		r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", grant.ID).Update("actor_id", f.reader).Error)
		read("foreign-actor-grant", 200, grant.ID, `"state":"AWAITING_OPERATOR_AUTHORITY"`)
		foreignActor := deniedPost(t, f, f.writer, f.tenant, f.caseID, request.ID, fmt.Sprintf(`{"expected_case_version":3,"grant_id":%q}`, grant.ID), "foreign-actor-grant", 409, "case_conflict")
		foreignActor["slot"] = 5
		ledger = append(ledger, foreignActor)
		r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", grant.ID).Update("actor_id", f.writer).Error)
		r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", grant.ID).Update("deadline", time.Now().UTC().Add(-time.Minute)).Error)
		read("expired-grant", 200, grant.ID, `"state":"AWAITING_OPERATOR_AUTHORITY"`)
		expired := deniedPost(t, f, f.writer, f.tenant, f.caseID, request.ID, fmt.Sprintf(`{"expected_case_version":3,"grant_id":%q}`, grant.ID), "expired-grant", 409, "case_conflict")
		expired["slot"] = 6
		ledger = append(ledger, expired)
		foreignCase := deniedPost(t, f, f.owner, f.other, f.caseID, request.ID, fmt.Sprintf(`{"expected_case_version":3,"grant_id":%q}`, grant.ID), "foreign-case", 404, "not_found")
		foreignCase["slot"] = 7
		ledger = append(ledger, foreignCase)
		if trap.calls.Load() != 0 {
			t.Fatal("R139 native call trap reached")
		}
		r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "native_rejections.json"), map[string]any{"kind": "SOFTWARE_ONLY_NO_AI_OUTPUT", "slotsUsed": len(ledger), "nativeExecutePOST": nativeExecutePostCount, "grantIssuances": nativeGrantIssuances, "cases": ledger, "trapCalls": trap.calls.Load()})
		nativeNegativeReadCount = len(readLedger)
		r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "native_reads_negative.json"), map[string]any{"kind": "SOFTWARE_ONLY_NO_AI_OUTPUT", "reads": readLedger})
	}) {
		t.Fatal("R139 G1 native rejection stage failed; G2 forbidden")
	}
	if !t.Run("native-precall-denial", func(t *testing.T) {
		f := r139Start(t, dsn)
		request := r139Request(t, f)
		service := r139Service(trap, a)
		engine.WaitDataExecutor = service
		defer func() { engine.WaitDataExecutor = nil }()
		grant := issueGrant(t, f, request, service)
		callbackName := "r139:after_execution_create"
		fired := false
		if err := db.DB.Callback().Create().After("gorm:create").Register(callbackName, func(tx *gorm.DB) {
			if row, ok := tx.Statement.Dest.(*models.InterventionExecution); ok && row.RequestID == request.ID {
				service.PromptVersion = "r139-changed-after-claim"
				fired = true
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := db.DB.Callback().Create().Remove(callbackName); err != nil {
				t.Error(err)
			}
		}()
		body := fmt.Sprintf(`{"expected_case_version":3,"grant_id":%q}`, grant.ID)
		before := r139Fingerprint(t, f.tenant)
		nativeExecutePostCount++
		rec := r139SendRaw(f, f.writer, f.tenant, f.caseID, request.ID, body)
		if rec.Code != 202 || !fired {
			t.Fatalf("R139 pre-call race claim=%d callback=%v", rec.Code, fired)
		}
		var accepted map[string]any
		if json.Unmarshal(rec.Body.Bytes(), &accepted) != nil {
			t.Fatal("R139 pre-call accepted response JSON invalid")
		}
		acceptedExecution, _ := accepted["execution"].(map[string]any)
		acceptedID, _ := acceptedExecution["id"].(string)
		acceptedState, _ := acceptedExecution["state"].(string)
		if acceptedID == "" || acceptedState != "CLAIMED" {
			t.Fatalf("R139 pre-call accepted response identity/state invalid: %q %q", acceptedID, acceptedState)
		}
		var execution models.InterventionExecution
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			if db.DB.Where("tenant_id = ? AND request_id = ?", f.tenant, request.ID).Take(&execution).Error == nil && execution.TerminalCode != "" {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if execution.TerminalCode != "DENIED" || trap.calls.Load() != 0 || r139Count(t, &models.InterventionAttempt{}, "execution_id = ?", execution.ID) != 0 {
			t.Fatalf("R139 pre-call denial=%q calls=%d", execution.TerminalCode, trap.calls.Load())
		}
		var budget models.InterventionBudget
		r102DB(t, db.DB.Where("tenant_id = ? AND job_id = ?", f.tenant, f.job).Take(&budget).Error)
		if budget.Consumed != 0 {
			t.Fatal("R139 denied pre-call debited budget")
		}
		after := r139Fingerprint(t, f.tenant)
		if before == after {
			t.Fatal("R139 accepted pre-call denial did not record its terminal state")
		}
		responseHash := sha256.Sum256(rec.Body.Bytes())
		r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "native_precall_denied.json"), map[string]any{"slot": 8, "status": rec.Code, "acceptedState": acceptedState, "acceptedExecutionId": acceptedID, "terminal": execution.TerminalCode, "executionId": execution.ID, "durableBefore": before, "durableAfter": after, "bodySha256": hex.EncodeToString(responseHash[:]), "trapCalls": trap.calls.Load(), "attempts": 0, "budgetConsumed": budget.Consumed, "nativeExecutePOSTBeforeBrowser": nativeExecutePostCount, "grantIssuancesBeforeBrowser": nativeGrantIssuances, "kind": "SOFTWARE_ONLY_NO_AI_OUTPUT"})
	}) {
		t.Fatal("R139 G1 pre-call stage failed; G2 forbidden")
	}
	if trap.calls.Load() != 0 {
		t.Fatal("R139 native call trap reached")
	}
	final.Phase = "NATIVE_SOFTWARE_MATRIX_COMPLETE"
	// The positive case is a new tenant/request/grant. Only its browser POST
	// reaches the real one-call wire wrapper. Its server is held in this process.
	f := r139Start(t, dsn)
	request := r139Request(t, f)
	var originalSnapshot models.AnalysisSnapshot
	r102DB(t, db.DB.Where("id = ?", f.snap).Take(&originalSnapshot).Error)
	key := os.Getenv("R139_ALIBABA_KEY")
	endpoint := os.Getenv("R139_ALIBABA_ENDPOINT")
	endpointDigest := sha256.Sum256([]byte(endpoint))
	effectDir := filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "effect_slots")
	if len(key) < 16 || hex.EncodeToString(endpointDigest[:]) != "b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f" || os.Getenv("R139_EFFECT_DIR") != effectDir {
		t.Fatal("R139 admitted provider material missing")
	}
	if err := os.Mkdir(effectDir, 0700); err != nil {
		t.Fatal("R139 effect slot directory not fresh")
	}
	isolation, err := r096IsolatedTransport()
	if err != nil {
		t.Fatal(err)
	}
	defer isolation.CloseIdleConnections()
	wire = &r116WireTransport{next: r139JournalTransport{next: isolation, evidenceDir: evidenceDir, key: key}, key: key, endpoint: endpoint, effectDir: os.Getenv("R139_EFFECT_DIR")}
	oldTransport := http.DefaultTransport
	http.DefaultTransport = wire
	defer func() { http.DefaultTransport = oldTransport }()
	provider, err := ai.NewOpenAIProvider(key, a.Model, a.CompletionTokens, endpoint).WithAlibabaQwen37OnceThinkingDisabled()
	if err != nil {
		t.Fatal(err)
	}
	service := r139Service(r139BrandedTransport{inner: provider}, a)
	engine.WaitDataExecutor = service
	defer func() { engine.WaitDataExecutor = nil }()
	grant := issueGrant(t, f, request, service)
	readinessPath := "/rerun-requests/" + request.ID + "/execution-readiness"
	statusPath := "/rerun-requests/" + request.ID + "/execution-status"
	positiveReads := []map[string]any{}
	addRead := func(name string, row map[string]any) { row["name"] = name; positiveReads = append(positiveReads, row) }
	for _, denied := range []struct{ actor, name string }{{f.reader, "readiness-denied-reader"}, {f.noJobs, "readiness-denied-nojobs"}} {
		addRead(denied.name, r139ReadNoWrite(t, f, denied.actor, f.tenant, readinessPath, 403, grant.ID, false, `"error":"permission_denied"`))
	}
	addRead("readiness-foreign-case", r139ReadNoWrite(t, f, f.owner, f.other, readinessPath, 404, grant.ID, false, `"error":"not_found"`))
	addRead("writer-READY", r139ReadNoWrite(t, f, f.writer, f.tenant, readinessPath, 200, grant.ID, true, `"state":"READY"`))
	status := r139ReadNoWrite(t, f, f.reader, f.tenant, statusPath, 200, grant.ID, false, `"execution_present":false`, `"phase":"NOT_STARTED"`)
	addRead("reader-status-no-row", status)
	engine.WaitDataExecutor = nil
	disabledStatus := r139ReadNoWrite(t, f, f.reader, f.tenant, statusPath, 200, grant.ID, false, `"execution_present":false`, `"phase":"NOT_STARTED"`)
	addRead("process-disabled-status", disabledStatus)
	addRead("process-disabled-readiness", r139ReadNoWrite(t, f, f.writer, f.tenant, readinessPath, 200, grant.ID, false, `"state":"PROCESS_DISABLED"`))
	engine.WaitDataExecutor = service
	beforeFault := r139Fingerprint(t, f.tenant)
	callbackName := "r139:unreadable_status"
	if err := db.DB.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "intervention_executions" {
			tx.AddError(errors.New("r139_controlled_storage_unreadable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	unreadable := f.send(http.MethodGet, f.tenant, f.reader, statusPath, nil)
	unreadableReadiness := f.send(http.MethodGet, f.tenant, f.writer, readinessPath, nil)
	if err := db.DB.Callback().Query().Remove(callbackName); err != nil {
		t.Fatal(err)
	}
	afterFault := r139Fingerprint(t, f.tenant)
	if unreadable.Code != 503 || unreadableReadiness.Code != 503 || beforeFault != afterFault || bytes.Contains(unreadable.Body.Bytes(), []byte(grant.ID)) || bytes.Contains(unreadableReadiness.Body.Bytes(), []byte(grant.ID)) || !strings.Contains(unreadable.Header().Get("Cache-Control"), "no-store") || !strings.Contains(unreadableReadiness.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("R139 unreadable status=%d readiness=%d durableEqual=%v", unreadable.Code, unreadableReadiness.Code, beforeFault == afterFault)
	}
	statusFaultHash := sha256.Sum256(unreadable.Body.Bytes())
	readinessFaultHash := sha256.Sum256(unreadableReadiness.Body.Bytes())
	addRead("status-unreadable-503", map[string]any{"status": unreadable.Code, "error": "status_unavailable", "durableBefore": beforeFault, "durableAfter": afterFault, "bodySha256": hex.EncodeToString(statusFaultHash[:]), "grantHandlePresent": false, "deniedHandleAbsent": !bytes.Contains(unreadable.Body.Bytes(), []byte(grant.ID)), "noStore": strings.Contains(unreadable.Header().Get("Cache-Control"), "no-store")})
	addRead("readiness-unreadable-503", map[string]any{"status": unreadableReadiness.Code, "error": "readiness_unavailable", "durableBefore": beforeFault, "durableAfter": afterFault, "bodySha256": hex.EncodeToString(readinessFaultHash[:]), "grantHandlePresent": false, "deniedHandleAbsent": !bytes.Contains(unreadableReadiness.Body.Bytes(), []byte(grant.ID)), "noStore": strings.Contains(unreadableReadiness.Header().Get("Cache-Control"), "no-store")})
	r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "native_reads_positive_pre_browser.json"), map[string]any{"kind": "SOFTWARE_ONLY_NO_AI_OUTPUT", "reads": positiveReads})
	var routeSequence atomic.Int64
	var routeLock sync.Mutex
	routePath := filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "go_routes.jsonl")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := &r139ResponseWriter{ResponseWriter: w, status: 200}
		path := r.URL.Path
		id := routeSequence.Add(1)
		auth, execute, allowed := r139BrowserRoute(r.Method, path)
		if auth {
			allowed = authCount.Add(1) <= 4
		}
		if execute {
			allowed = executeCount.Add(1) <= 1
		}
		routeLock.Lock()
		logErr := r139Append(routePath, map[string]any{"phase": "attempt", "id": id, "method": r.Method, "path": path, "allowed": allowed})
		routeLock.Unlock()
		if logErr != nil {
			http.Error(wrapped, "r139_route_ledger_unavailable", 503)
			return
		}
		defer func() {
			routeLock.Lock()
			defer routeLock.Unlock()
			if err := r139Append(routePath, map[string]any{"phase": "response", "id": id, "method": r.Method, "path": path, "status": wrapped.status}); err != nil {
				t.Errorf("R139 route response ledger: %v", err)
			}
		}()
		if !allowed {
			http.Error(wrapped, "r139_route_not_allowed", 405)
			return
		}
		if execute {
			body, err := io.ReadAll(io.LimitReader(r.Body, 513))
			if err != nil || len(body) > 512 {
				http.Error(wrapped, "r139_body_invalid", http.StatusBadRequest)
				return
			}
			var fields map[string]json.RawMessage
			if json.Unmarshal(body, &fields) != nil || len(fields) != 2 || fields["grant_id"] == nil || fields["expected_case_version"] == nil {
				http.Error(wrapped, "r139_body_invalid", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		f.router.ServeHTTP(wrapped, r)
	}))
	defer server.Close()
	if !strings.HasPrefix(server.URL, "http://127.0.0.1:") {
		t.Fatal("R139 browser origin not loopback")
	}
	manifest := map[string]any{"tranche": "CCMAI-RUNTIME-139", "apiOrigin": server.URL, "tenant": f.tenant, "writerEmail": f.writer + "@example.invalid", "readerEmail": f.reader + "@example.invalid", "syntheticPassword": "R101SyntheticOnly!123", "caseId": f.caseID, "requestId": request.ID, "expectedCaseVersion": 3, "browserExecuteCap": 1}
	r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "browser_manifest_private.json"), manifest)
	final.Phase = "BROWSER_READY_AWAITING_STOP"
	fmt.Printf("R139_BROWSER_READY %s\n", server.URL)
	stop := os.Getenv("R139_STOP_FILE")
	if !filepath.IsAbs(stop) {
		t.Fatal("R139 stop file invalid")
	}
	end := time.Now().Add(20 * time.Minute)
	for time.Now().Before(end) {
		if _, err := os.Stat(stop); err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if _, err := os.Stat(stop); err != nil {
		t.Error("R139 supervisor stop deadline")
	}
	final.Phase = "BROWSER_STOPPED_TERMINAL_OBSERVATION"
	wire.mu.Lock()
	capturedCalls, capturedRequest, capturedResponse, capturedStatus := wire.attempts, wire.request, wire.response, wire.status
	wire.mu.Unlock()
	if capturedCalls > 0 && capturedRequest != "" && capturedResponse != "" {
		r139JSONNew(t, filepath.Join(evidenceDir, "provider_wire_private.json"), map[string]any{"provider": "alibaba_maas", "model": "qwen3.7-flash", "httpStatus": capturedStatus, "request": capturedRequest, "response": capturedResponse, "physicalCalls": capturedCalls})
	}
	var row models.InterventionExecution
	rowErr := db.DB.Where("tenant_id = ? AND request_id = ?", f.tenant, request.ID).Take(&row).Error
	var storedRequest models.InterventionRerunRequest
	requestErr := db.DB.Where("id = ?", request.ID).Take(&storedRequest).Error
	var current models.InterventionCase
	caseErr := db.DB.Where("id = ?", f.caseID).Take(&current).Error
	final.ExecutionQueryPresent = rowErr == nil
	final.CaseQueryPresent = caseErr == nil
	if rowErr == nil {
		final.ExecutionState = row.State
		final.ExecutionTerminal = row.TerminalCode
	}
	if caseErr == nil {
		final.CaseWorkState = current.WorkState
	}
	var snapshotAfter models.AnalysisSnapshot
	snapshotErr := db.DB.Where("id = ?", f.snap).Take(&snapshotAfter).Error
	var run models.JobRun
	runErr := db.DB.Where("id = ?", row.RunID).Take(&run).Error
	var newSnapshot models.AnalysisSnapshot
	newSnapshotErr := db.DB.Where("id = ?", row.SnapshotID).Take(&newSnapshot).Error
	var proposal models.InterventionProposal
	proposalErr := db.DB.Where("tenant_id = ? AND case_id = ? AND run_id = ?", f.tenant, f.caseID, row.RunID).Take(&proposal).Error
	var event models.InterventionHumanEvent
	eventErr := db.DB.Where("tenant_id = ? AND case_id = ? AND rerun_request_id = ? AND action = ?", f.tenant, f.caseID, request.ID, "EXECUTE_RERUN").Take(&event).Error
	var receipt models.InterventionReceipt
	receiptErr := db.DB.Where("tenant_id = ? AND run_id = ?", f.tenant, row.RunID).Take(&receipt).Error
	var attempt models.InterventionAttempt
	attemptErr := db.DB.Where("tenant_id = ? AND execution_id = ?", f.tenant, row.ID).Take(&attempt).Error
	var budget models.InterventionBudget
	budgetErr := db.DB.Where("tenant_id = ? AND job_id = ?", f.tenant, f.job).Take(&budget).Error
	var after map[string]any
	jsonErr := json.Unmarshal([]byte(event.After), &after)
	originalRequestUnchanged := requestErr == nil && reflect.DeepEqual(storedRequest, request)
	originalSnapshotUnchanged := snapshotErr == nil && reflect.DeepEqual(snapshotAfter, originalSnapshot)
	lineage := rowErr == nil && runErr == nil && newSnapshotErr == nil && proposalErr == nil && eventErr == nil && receiptErr == nil && attemptErr == nil && budgetErr == nil && jsonErr == nil && caseErr == nil && row.RequestID == request.ID && row.CaseID == f.caseID && row.GrantID == grant.ID && row.RunID == run.ID && row.SnapshotID == newSnapshot.ID && row.SnapshotDigest == newSnapshot.Digest && proposal.RunID == row.RunID && proposal.SnapshotID == row.SnapshotID && event.ProposalID != nil && *event.ProposalID == proposal.ID && event.RerunRequestID != nil && *event.RerunRequestID == request.ID && event.CaseVersion == proposal.Revision && current.ActiveProposalID != nil && *current.ActiveProposalID == proposal.ID && current.Version == proposal.Revision && run.Status == "success" && run.FinishedAt != nil && receipt.RunID == run.ID && attempt.RunID == run.ID && attempt.ExecutionID != nil && *attempt.ExecutionID == row.ID && budget.Consumed == 1 && after["execution_id"] == row.ID && after["request_id"] == request.ID && after["run_id"] == run.ID && after["snapshot_id"] == newSnapshot.ID
	// Export the observed state before any publication assertion can terminate
	// this test. A failed expectation leaves the separate partial HOLD receipt.
	fingerprint, fingerprintErr := r139FingerprintRaw(f.tenant)
	final.DurableFingerprint = fingerprint
	if fingerprintErr != nil {
		final.FingerprintError = fingerprintErr.Error()
	}
	readerStatus := r139ReadNoWrite(t, f, f.reader, f.tenant, statusPath, 200, grant.ID, false, `"phase":"PUBLISHED"`, `"proposal_available":true`, fmt.Sprintf(`"published_case_version":%d`, proposal.Revision))
	writerStatus := r139ReadNoWrite(t, f, f.writer, f.tenant, statusPath, 200, grant.ID, false, `"phase":"PUBLISHED"`)
	history := r139ReadNoWrite(t, f, f.reader, f.tenant, "/rerun-requests?limit=10", 200, grant.ID, false, request.ID)
	detail := r139ReadNoWrite(t, f, f.reader, f.tenant, "", 200, grant.ID, false, proposal.ID)
	addRead("reader-published-status", readerStatus)
	addRead("writer-published-status", writerStatus)
	addRead("reader-history", history)
	addRead("reader-open-detail", detail)
	nativePositiveReadCount = len(positiveReads)
	r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "native_reads_positive.json"), map[string]any{"kind": "SOFTWARE_ONLY_NO_AI_OUTPUT", "reads": positiveReads})
	statusPublished := readerStatus["status"] == 200 && writerStatus["status"] == 200
	historyVisible := history["status"] == 200 && detail["status"] == 200
	wire.mu.Lock()
	calls, requestWire, responseWire, providerStatus := wire.attempts, wire.request, wire.response, wire.status
	wire.mu.Unlock()
	terminal := map[string]any{"nativeExecutePOST": nativeExecutePostCount, "nativeNegativeReadCount": nativeNegativeReadCount, "nativePositiveReadCount": nativePositiveReadCount, "grantIssuances": nativeGrantIssuances, "executePOST": executeCount.Load(), "authPOST": authCount.Load(), "providerPhysicalCalls": calls, "provider": service.Provider, "model": service.Model, "configRef": service.ConfigRef, "promptVersion": service.PromptVersion, "promptSha256": a.PromptSHA256, "completionTokens": a.CompletionTokens, "maxRequestBytes": a.MaxRequestBytes, "thinkingDisabled": true, "syntheticNamespace": a.SyntheticNamespace, "issuerRole": a.IssuerRole, "actorRole": a.ActorRole, "grantDeadlineUTC": a.GrantDeadlineUTC, "grantId": grant.ID, "requestSha256": func() string { h := sha256.Sum256([]byte(requestWire)); return hex.EncodeToString(h[:]) }(), "responseSha256": func() string { h := sha256.Sum256([]byte(responseWire)); return hex.EncodeToString(h[:]) }(), "providerHTTPStatus": providerStatus, "originalRequestUnchanged": originalRequestUnchanged, "originalSnapshotUnchanged": originalSnapshotUnchanged, "lineageVerified": lineage, "readerWriterPublishedStatus": statusPublished, "readerHistoryAndOpenDetail": historyVisible, "caseWorkState": current.WorkState, "executionPresent": rowErr == nil, "executionTerminal": row.TerminalCode, "executionState": row.State, "caseId": f.caseID, "requestId": request.ID, "storedRequestId": storedRequest.ID, "storedRequestCaseId": storedRequest.CaseID, "executionId": row.ID, "executionRequestId": row.RequestID, "executionCaseId": row.CaseID, "executionGrantId": row.GrantID, "executionRunId": row.RunID, "executionSnapshotId": row.SnapshotID, "jobRunId": run.ID, "snapshotId": newSnapshot.ID, "snapshotDigest": newSnapshot.Digest, "proposalId": proposal.ID, "proposalRunId": proposal.RunID, "proposalSnapshotId": proposal.SnapshotID, "proposalRevision": proposal.Revision, "eventId": event.ID, "eventRequestId": event.RerunRequestID, "eventProposalId": event.ProposalID, "eventCaseVersion": event.CaseVersion, "attemptId": attempt.ID, "attemptRunId": attempt.RunID, "attemptExecutionId": attempt.ExecutionID, "receiptId": receipt.ID, "receiptRunId": receipt.RunID, "budgetConsumed": budget.Consumed, "usageState": attempt.UsageState, "inputUsagePresence": attempt.InputUsagePresence, "outputUsagePresence": attempt.OutputUsagePresence, "inputTokens": attempt.InputTokens, "outputTokens": attempt.OutputTokens, "responseModel": attempt.ResponseModel, "durableFingerprint": fingerprint, "fingerprintError": fingerprintErr != nil}
	if executeCount.Load() != 1 || calls != 1 || responseWire == "" || requestWire == "" || providerStatus != 200 || rowErr != nil || row.State != "PUBLISHED" || current.WorkState != "OPEN" || !originalRequestUnchanged || !originalSnapshotUnchanged || !lineage || !statusPublished || !historyVisible || fingerprintErr != nil {
		t.Fatal("R139 positive publication HOLD")
	}
	r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "go_terminal.json"), terminal)
	final.Phase = "VALIDATED_PUBLICATION"
	final.Disposition = "PASS"
}
