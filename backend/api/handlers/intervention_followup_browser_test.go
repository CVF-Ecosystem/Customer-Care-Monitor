package handlers_test

// R133 is an opt-in synthetic product-flow fixture. It installs no provider.
import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/handlers"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type r133Identity struct {
	TrancheID string `json:"trancheId"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Database  string `json:"database"`
	Nonce     string `json:"nonce"`
}

func r133NewJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return reviewFixWriteNew(path, append(b, '\n'))
}

// Admission runs before the compatibility helper can connect, migrate or seed.
func r133Admit(t *testing.T) (string, string, string, r133Identity) {
	t.Helper()
	if os.Getenv("R133_WAITDATA_BROWSER") != "1" {
		t.Skip("R133 opt-in absent")
	}
	if handlers.InterventionRunner != nil || engine.WaitDataExecutor != nil {
		t.Fatal("R133 provider/executor must be nil")
	}
	directory, frontend := os.Getenv("R133_EVIDENCE_DIR"), os.Getenv("R133_FRONTEND_DIST")
	if !filepath.IsAbs(directory) || !filepath.IsAbs(frontend) || filepath.Clean(directory) != directory || filepath.Clean(frontend) != frontend {
		t.Fatal("R133 absolute clean evidence/dist paths required")
	}
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	projectRoot := filepath.Clean(filepath.Join(root, "..", "..", ".."))
	expectedEvidence := filepath.Join(filepath.Dir(projectRoot), ".ccmai_local_evidence", "r133_waitdata_repair_20261019", "fixture")
	if directory != expectedEvidence {
		t.Fatal("R133 exact external evidence directory required")
	}
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil || resolved != directory {
		t.Fatal("R133 evidence path must exist without symlinks")
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("R133 evidence directory invalid")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 || entries[0].Name() != "fixture_identity.json" || entries[0].Type()&os.ModeSymlink != 0 {
		t.Fatal("R133 fresh directory must contain only fixture_identity.json")
	}
	if file, err := os.Lstat(filepath.Join(directory, "fixture_identity.json")); err != nil || !file.Mode().IsRegular() {
		t.Fatal("R133 identity must be regular file")
	}
	if info, err := os.Stat(filepath.Join(frontend, "index.html")); err != nil || !info.Mode().IsRegular() {
		t.Fatal("R133 built frontend missing")
	}
	data, err := os.ReadFile(filepath.Join(directory, "fixture_identity.json"))
	var identity r133Identity
	if err != nil || json.Unmarshal(data, &identity) != nil || identity.TrancheID != "CCMAI-RUNTIME-133" || identity.Host != "127.0.0.1" || identity.Port < 1024 || identity.Port > 65535 || !regexp.MustCompile(`^ccma_r133_[a-z0-9_]{1,32}$`).MatchString(identity.Database) || !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(identity.Nonce) {
		t.Fatal("R133 owned identity invalid")
	}
	dsn := os.Getenv("R133_MYSQL_DSN")
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil || parsed == nil || parsed.Net != "tcp" || parsed.Addr != net.JoinHostPort(identity.Host, strconv.Itoa(identity.Port)) || parsed.DBName != identity.Database || parsed.Params["init_command"] != "" {
		t.Fatal("R133 DSN does not match identity")
	}
	if os.Getenv("R101_MYSQL_DSN") != "" {
		t.Fatal("R101 compatibility DSN must be unset before R133 admission")
	}
	probe, err := sql.Open("mysql", parsed.FormatDSN())
	if err != nil {
		t.Fatal("R133 ownership probe open failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	var nonce string
	var count int
	err = probe.QueryRowContext(ctx, "SELECT COUNT(*), MIN(nonce) FROM ccma_r133_fixture_owner").Scan(&count, &nonce)
	cancel()
	_ = probe.Close()
	if err != nil || count != 1 || nonce != identity.Nonce {
		t.Fatal("R133 preexisting nonce-table ownership proof failed")
	}
	return directory, frontend, dsn, identity
}

func r133LoadCase(id string) (models.InterventionCase, error) {
	var row models.InterventionCase
	if err := db.DB.Where("id = ?", id).Take(&row).Error; err != nil {
		return row, err
	}
	return row, nil
}

func r133Case(t *testing.T, id string) models.InterventionCase {
	t.Helper()
	row, err := r133LoadCase(id)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func r133Counts(caseID string) (map[string]int64, error) {
	tables := map[string]any{
		"revisions": &models.InterventionDataRevision{}, "requests": &models.InterventionRerunRequest{}, "human_events": &models.InterventionHumanEvent{},
		"attempts": &models.InterventionAttempt{}, "budgets": &models.InterventionBudget{}, "grants": &models.InterventionExecutionGrant{}, "executions_claims": &models.InterventionExecution{}, "receipts": &models.InterventionReceipt{}, "runs": &models.JobRun{}, "proposals": &models.InterventionProposal{}, "snapshots": &models.AnalysisSnapshot{}, "ai_usage_logs": &models.AIUsageLog{},
	}
	out := map[string]int64{}
	for key, model := range tables {
		query := db.DB.Model(model)
		if key == "revisions" || key == "requests" || key == "human_events" {
			query = query.Where("case_id = ?", caseID)
		}
		var count int64
		if err := query.Count(&count).Error; err != nil {
			return out, fmt.Errorf("%s count: %w", key, err)
		}
		out[key] = count
	}
	return out, nil
}

func r133MustCounts(t *testing.T, caseID string) map[string]int64 {
	t.Helper()
	counts, err := r133Counts(caseID)
	if err != nil {
		t.Fatal(err)
	}
	return counts
}

type r133StatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *r133StatusWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *r133StatusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func TestR133WaitDataFollowupBrowserFixture(t *testing.T) {
	directory, frontend, dsn, identity := r133Admit(t)
	t.Setenv("R101_MYSQL_DSN", dsn)
	f := m3FixtureStart(t)
	t.Setenv("R101_MYSQL_DSN", "")
	wait := r133Case(t, f.waitID)
	if wait.WorkState != "WAITING_ON_DATA" || wait.Disposition != "WAIT_DATA" || wait.Execution != "NOT_RUN" || wait.SnapshotID == nil {
		t.Fatal("R133 helper WAIT_DATA seed unexpected")
	}
	// New local content is seeded before the effect baseline; no channel import runs.
	newMessage := models.Message{ID: pkg.NewUUID(), TenantID: f.tenant, ConversationID: wait.ConversationID, ExternalMessageID: pkg.NewUUID(), SenderType: "customer", Content: "Tôi khiếu nại đơn hàng minh họa R133, đã chờ phản hồi và gửi thêm mã đơn hàng.", ContentType: "text", Attachments: "[]", RawData: "{}", SentAt: time.Now().UTC(), CreatedAt: time.Now().UTC()}
	if err := db.DB.Create(&newMessage).Error; err != nil {
		t.Fatal(err)
	}
	var originalMessage models.Message
	if err := db.DB.Where("id = ?", newMessage.ID).Take(&originalMessage).Error; err != nil {
		t.Fatal(err)
	}
	originalCase := r133Case(t, f.waitID)
	originalOtherCase := r133Case(t, f.caseID)
	var originalSeedProposal models.InterventionProposal
	if err := db.DB.Where("id = ?", f.proposal).Take(&originalSeedProposal).Error; err != nil {
		t.Fatal(err)
	}
	var originalSnapshot models.AnalysisSnapshot
	if err := db.DB.Where("id = ?", *wait.SnapshotID).Take(&originalSnapshot).Error; err != nil {
		t.Fatal(err)
	}
	var originalProposals []models.InterventionProposal
	if err := db.DB.Where("case_id = ?", f.waitID).Order("revision").Find(&originalProposals).Error; err != nil {
		t.Fatal(err)
	}
	baseline := r133MustCounts(t, f.waitID)
	casePath := "/intervention-cases/" + f.waitID
	capture := map[string]any{"expected_version": 1, "expected_snapshot_id": *wait.SnapshotID, "expected_snapshot_digest": wait.SnapshotDigest, "reason": "R133 synthetic source observed"}
	deniedReader := f.send(http.MethodPost, f.tenant, f.reader, casePath+"/data-revisions", capture)
	deniedNoJobs := f.send(http.MethodPost, f.tenant, f.member, casePath+"/rerun-requests", map[string]any{"expected_version": 2, "expected_snapshot_id": *wait.SnapshotID, "expected_snapshot_digest": wait.SnapshotDigest, "data_revision_id": pkg.NewUUID(), "data_revision_digest": strings.Repeat("a", 64), "reason": "R133 permission probe", "request_key": "r133-denied"})
	if deniedReader.Code != http.StatusForbidden || deniedNoJobs.Code != http.StatusForbidden || !reflect.DeepEqual(baseline, r133MustCounts(t, f.waitID)) || !reflect.DeepEqual(originalCase, r133Case(t, f.waitID)) {
		t.Fatal("R133 native denial or no-write proof failed")
	}

	// One request-scoped, one-shot synthetic source-read failure exercises the
	// real capture handler's sanitized 503. It never changes provider behavior.
	type r133CaptureFaultKey struct{}
	var injected atomic.Bool
	callbackName := "r133:synthetic-source-read-once"
	if err := db.DB.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "messages" && tx.Statement.Context != nil && tx.Statement.Context.Value(r133CaptureFaultKey{}) == true && injected.CompareAndSwap(false, true) {
			tx.AddError(fmt.Errorf("r133 synthetic source read unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.DB.Callback().Query().Remove(callbackName); err != nil {
			t.Error(err)
		}
	})

	journal, err := os.OpenFile(filepath.Join(directory, "route_journal.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	base := "/api/v1/tenants/" + f.tenant
	caseAPI := base + casePath
	var mu sync.Mutex
	authPosts, capturePosts, requestPosts, deniedRoutes := 0, 0, 0, 0
	captureStatuses := []int{}
	firstCaptureNoWrite := false
	var observationErrors []string
	browserHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, method := r.URL.EscapedPath(), r.Method
		apiRoute := strings.HasPrefix(path, "/api/") || path == "/health"
		allowed := false
		firstCapture := false
		if method == http.MethodGet || method == http.MethodHead {
			if apiRoute {
				allowed = path == "/health" || path == "/api/v1/setup/status" || path == "/api/v1/profile" || path == "/api/v1/tenants" || path == base || path == base+"/me" || path == base+"/demo/status" || path == base+"/onboarding-status" || path == base+"/dashboard" || path == base+"/intervention-capability" || path == base+"/intervention-assignees" || path == base+"/intervention-cases" || path == caseAPI || path == caseAPI+"/data-revisions" || path == caseAPI+"/rerun-requests" || strings.HasPrefix(path, caseAPI+"/data-revisions/")
			} else {
				allowed = path == "/" || path == "/login" || path == "/"+f.tenant+"/intervention-cases" || path == "/"+f.tenant+"/intervention-cases/"+f.waitID || path == "/favicon.png" || (strings.HasPrefix(path, "/assets/") && !strings.Contains(path, ".."))
			}
		} else if method == http.MethodPost {
			mu.Lock()
			switch path {
			case "/api/v1/auth/login", "/api/v1/auth/logout", "/api/v1/auth/refresh":
				authPosts++
				allowed = authPosts <= 4
			case caseAPI + "/data-revisions":
				capturePosts++
				allowed = capturePosts <= 2
				firstCapture = capturePosts == 1 && allowed
			case caseAPI + "/rerun-requests":
				requestPosts++
				allowed = requestPosts <= 1
			}
			mu.Unlock()
		}
		writer := &r133StatusWriter{ResponseWriter: w}
		if firstCapture {
			r = r.WithContext(context.WithValue(r.Context(), r133CaptureFaultKey{}, true))
		}
		if !allowed {
			http.Error(writer, "R133 route denied", http.StatusForbidden)
		} else if apiRoute {
			f.router.ServeHTTP(writer, r)
		} else if path == "/" || path == "/login" || strings.HasPrefix(path, "/"+f.tenant+"/intervention-cases") {
			http.ServeFile(writer, r, filepath.Join(frontend, "index.html"))
		} else {
			http.FileServer(http.Dir(frontend)).ServeHTTP(writer, r)
		}
		if firstCapture {
			counts, countErr := r133Counts(f.waitID)
			row, caseErr := r133LoadCase(f.waitID)
			mu.Lock()
			firstCaptureNoWrite = countErr == nil && caseErr == nil && reflect.DeepEqual(baseline, counts) && reflect.DeepEqual(originalCase, row)
			if countErr != nil {
				observationErrors = append(observationErrors, "first_capture_counts: "+countErr.Error())
			}
			if caseErr != nil {
				observationErrors = append(observationErrors, "first_capture_case: "+caseErr.Error())
			}
			mu.Unlock()
		}
		mu.Lock()
		if method == http.MethodPost && path == caseAPI+"/data-revisions" {
			captureStatuses = append(captureStatuses, writer.status)
		}
		if !allowed {
			deniedRoutes++
		}
		_, writeErr := fmt.Fprintf(journal, "{\"method\":%q,\"path\":%q,\"status\":%d}\n", method, path, writer.status)
		if writeErr != nil {
			observationErrors = append(observationErrors, "route_journal_write: "+writeErr.Error())
		}
		mu.Unlock()
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: browserHandler, ReadHeaderTimeout: 5 * time.Second}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	serveJoined := false
	serveResult := ""
	terminalWritten := false
	stopReason := "startup_or_assertion_failure"
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = server.Shutdown(ctx)
		cancel()
		_ = server.Close()
		_ = listener.Close()
		if !serveJoined {
			select {
			case err := <-served:
				serveResult = fmt.Sprint(err)
				serveJoined = true
			case <-time.After(5 * time.Second):
				serveResult = "serve_join_timeout"
				t.Error("R133 Serve did not join")
			}
		}
		if !terminalWritten {
			counts, countErr := r133Counts(f.waitID)
			if countErr != nil {
				t.Error("R133 terminal counts failed: ", countErr)
			}
			mu.Lock()
			errors := append([]string(nil), observationErrors...)
			mu.Unlock()
			if countErr != nil {
				errors = append(errors, "terminal_counts: "+countErr.Error())
			}
			result := map[string]any{"trancheId": "CCMAI-RUNTIME-133", "nonce": identity.Nonce, "stopReason": stopReason, "serveJoined": serveJoined, "serveResult": serveResult, "testFailed": t.Failed() || len(errors) > 0, "observationErrors": errors, "authPosts": authPosts, "capturePosts": capturePosts, "captureStatuses": captureStatuses, "injectedSourceRead": injected.Load(), "firstCaptureNoWrite": firstCaptureNoWrite, "requestPosts": requestPosts, "deniedRoutes": deniedRoutes, "nativeReaderCaptureStatus": deniedReader.Code, "nativeNoJobsRequestStatus": deniedNoJobs.Code, "counts": counts, "finishedAt": time.Now().UTC().Format(time.RFC3339Nano)}
			if err := r133NewJSON(filepath.Join(directory, "backend_terminal.json"), result); err != nil {
				t.Error("R133 terminal write failed: ", err)
			} else {
				terminalWritten = true
			}
		}
	}()
	origin := "http://" + listener.Addr().String()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(origin + "/health")
	if err != nil {
		t.Fatal("R133 health unavailable")
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal("R133 health failed")
	}
	ready := map[string]any{"trancheId": "CCMAI-RUNTIME-133", "nonce": identity.Nonce, "origin": origin, "tenantId": f.tenant, "caseId": f.waitID, "ownerEmail": f.owner + "@example.invalid", "readerEmail": f.reader + "@example.invalid", "password": "R101SyntheticOnly!123", "synthetic": true, "browserCaps": map[string]int{"authPosts": 4, "capturePosts": 2, "requestPosts": 1}}
	if err := r133NewJSON(filepath.Join(directory, "ready.json"), ready); err != nil {
		t.Fatal(err)
	}
	fmt.Println("R133_BROWSER_READY " + origin)
	stopPath := filepath.Join(directory, "stop.json")
	deadline := time.NewTimer(20 * time.Minute)
	defer deadline.Stop()
	for {
		select {
		case err := <-served:
			serveJoined = true
			serveResult = fmt.Sprint(err)
			t.Error("R133 Serve exited before owned stop")
			stopReason = "early_serve_exit"
			goto final
		case <-deadline.C:
			t.Error("R133 browser stop timeout")
			stopReason = "timeout"
			goto final
		case <-time.After(200 * time.Millisecond):
			data, err := os.ReadFile(stopPath)
			if os.IsNotExist(err) {
				continue
			}
			var stop struct {
				TrancheID string `json:"trancheId"`
				Nonce     string `json:"nonce"`
			}
			if err != nil || json.Unmarshal(data, &stop) != nil || stop.TrancheID != "CCMAI-RUNTIME-133" || stop.Nonce != identity.Nonce {
				t.Error("R133 stop ownership invalid")
				stopReason = "invalid_stop"
				goto final
			}
			stopReason = "owned_stop"
			goto final
		}
	}
final:
	// Close and join before final durable queries, while the deferred finalizer
	// still covers every fatal path after Serve starts.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = server.Shutdown(ctx)
	cancel()
	_ = server.Close()
	_ = listener.Close()
	if !serveJoined {
		select {
		case err := <-served:
			serveResult = fmt.Sprint(err)
			serveJoined = true
		case <-time.After(5 * time.Second):
			t.Error("R133 Serve join timeout")
		}
	}
	var revisions []models.InterventionDataRevision
	var requests []models.InterventionRerunRequest
	var events []models.InterventionHumanEvent
	if err := db.DB.Where("case_id = ?", f.waitID).Find(&revisions).Error; err != nil {
		t.Error(err)
	}
	if err := db.DB.Where("case_id = ?", f.waitID).Find(&requests).Error; err != nil {
		t.Error(err)
	}
	if err := db.DB.Where("case_id = ?", f.waitID).Order("case_version").Find(&events).Error; err != nil {
		t.Error(err)
	}
	current := r133Case(t, f.waitID)
	var snapshot models.AnalysisSnapshot
	if err := db.DB.Where("id = ?", *wait.SnapshotID).Take(&snapshot).Error; err != nil {
		t.Error(err)
	}
	var proposals []models.InterventionProposal
	if err := db.DB.Where("case_id = ?", f.waitID).Order("revision").Find(&proposals).Error; err != nil {
		t.Error(err)
	}
	var sourceMessage models.Message
	if err := db.DB.Where("id = ?", newMessage.ID).Take(&sourceMessage).Error; err != nil {
		t.Error(err)
	}
	var seedProposal models.InterventionProposal
	if err := db.DB.Where("id = ?", f.proposal).Take(&seedProposal).Error; err != nil {
		t.Error(err)
	}
	counts, countErr := r133Counts(f.waitID)
	if countErr != nil {
		t.Error("R133 final counts failed: ", countErr)
	}
	if stopReason != "owned_stop" || !serveJoined || len(revisions) != 1 || len(requests) != 1 || len(events) != 2 || current.Version != 3 || current.WorkState != "WAITING_ON_DATA" || current.Disposition != "WAIT_DATA" || current.Execution != "NOT_RUN" || !reflect.DeepEqual(originalSnapshot, snapshot) || !reflect.DeepEqual(originalMessage, sourceMessage) || !reflect.DeepEqual(originalProposals, proposals) || !reflect.DeepEqual(originalSeedProposal, seedProposal) || !reflect.DeepEqual(originalOtherCase, r133Case(t, f.caseID)) || current.RunID != originalCase.RunID || current.SnapshotDigest != originalCase.SnapshotDigest || !reflect.DeepEqual(current.SnapshotID, originalCase.SnapshotID) {
		t.Error("R133 final request-only invariant failed")
	}
	if len(revisions) == 1 && len(requests) == 1 && (revisions[0].OriginatingCaseVersion != 1 || revisions[0].CaseVersion != 2 || revisions[0].Coverage != "complete" || revisions[0].OriginalSnapshotID != *wait.SnapshotID || revisions[0].OriginalSnapshotDigest != wait.SnapshotDigest || requests[0].CaseVersion != 3 || requests[0].DataRevisionID != revisions[0].ID || requests[0].DataRevisionDigest != revisions[0].Digest || requests[0].Status != "AWAITING_EXECUTION_AUTHORITY" || requests[0].Execution != "NOT_RUN" || requests[0].OriginalSnapshotID != *wait.SnapshotID || requests[0].OriginalSnapshotDigest != wait.SnapshotDigest) {
		t.Error("R133 revision/request binding failed")
	}
	if len(events) == 2 && (events[0].Action != "OBSERVE_DATA" || events[0].CaseVersion != 2 || events[1].Action != "REQUEST_RERUN" || events[1].CaseVersion != 3) {
		t.Error("R133 human event sequence failed")
	}
	for key, before := range baseline {
		if key != "revisions" && key != "requests" && key != "human_events" && counts[key] != before {
			t.Errorf("R133 protected effect %s changed", key)
		}
	}
	if counts["revisions"] != baseline["revisions"]+1 || counts["requests"] != baseline["requests"]+1 || counts["human_events"] != baseline["human_events"]+2 {
		t.Error("R133 count delta failed")
	}
	if authPosts > 4 || capturePosts != 2 || requestPosts != 1 || !injected.Load() || !firstCaptureNoWrite || !reflect.DeepEqual(captureStatuses, []int{503, 201}) {
		t.Error("R133 browser source-error/capture budget failed")
	}
	mu.Lock()
	errors := append([]string(nil), observationErrors...)
	mu.Unlock()
	if len(errors) > 0 {
		t.Error("R133 HTTP observation errors: ", errors)
	}
	result := map[string]any{"trancheId": "CCMAI-RUNTIME-133", "nonce": identity.Nonce, "stopReason": stopReason, "serveJoined": serveJoined, "serveResult": serveResult, "testFailed": t.Failed(), "observationErrors": errors, "authPosts": authPosts, "capturePosts": capturePosts, "captureStatuses": captureStatuses, "injectedSourceRead": injected.Load(), "firstCaptureNoWrite": firstCaptureNoWrite, "requestPosts": requestPosts, "deniedRoutes": deniedRoutes, "nativeReaderCaptureStatus": deniedReader.Code, "nativeNoJobsRequestStatus": deniedNoJobs.Code, "countsBefore": baseline, "countsAfter": counts, "caseVersion": current.Version, "workState": current.WorkState, "disposition": current.Disposition, "execution": current.Execution, "requestStatus": func() string {
		if len(requests) == 1 {
			return requests[0].Status
		}
		return "missing"
	}(), "finishedAt": time.Now().UTC().Format(time.RFC3339Nano)}
	if err := r133NewJSON(filepath.Join(directory, "backend_terminal.json"), result); err != nil {
		t.Error("R133 terminal write failed: ", err)
	} else {
		terminalWritten = true
	}
}
