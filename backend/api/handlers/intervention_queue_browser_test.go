package handlers_test

// R135 is an opt-in synthetic queue and human-review fixture; no provider is installed.
import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/handlers"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

type r135Identity struct {
	TrancheID string `json:"trancheId"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Database  string `json:"database"`
	Nonce     string `json:"nonce"`
}

func r135JSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return reviewFixWriteNew(path, append(b, '\n'))
}

// Verify ownership before the compatibility helper can connect, migrate or seed.
func r135Admit(t *testing.T) (string, string, string, r135Identity) {
	t.Helper()
	if os.Getenv("R135_QUEUE_BROWSER") != "1" {
		t.Skip("R135 opt-in absent")
	}
	if handlers.InterventionRunner != nil || engine.WaitDataExecutor != nil {
		t.Fatal("R135 provider/executor must be nil")
	}
	directory, frontend := os.Getenv("R135_EVIDENCE_DIR"), os.Getenv("R135_FRONTEND_DIST")
	if !filepath.IsAbs(directory) || !filepath.IsAbs(frontend) || filepath.Clean(directory) != directory || filepath.Clean(frontend) != frontend {
		t.Fatal("R135 absolute clean evidence/dist paths required")
	}
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	projectRoot := filepath.Clean(filepath.Join(root, "..", "..", ".."))
	expected := filepath.Join(filepath.Dir(projectRoot), ".ccmai_local_evidence", "r135_queue_reject_20261020", "fixture")
	if directory != expected {
		t.Fatal("R135 exact external evidence directory required")
	}
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil || resolved != directory {
		t.Fatal("R135 evidence path must exist without symlinks")
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("R135 evidence directory invalid")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 || entries[0].Name() != "fixture_identity.json" || entries[0].Type()&os.ModeSymlink != 0 {
		t.Fatal("R135 fresh directory must contain only fixture_identity.json")
	}
	if file, err := os.Lstat(filepath.Join(directory, "fixture_identity.json")); err != nil || !file.Mode().IsRegular() {
		t.Fatal("R135 identity must be regular file")
	}
	if info, err := os.Stat(filepath.Join(frontend, "index.html")); err != nil || !info.Mode().IsRegular() {
		t.Fatal("R135 built frontend missing")
	}
	data, err := os.ReadFile(filepath.Join(directory, "fixture_identity.json"))
	var identity r135Identity
	if err != nil || json.Unmarshal(data, &identity) != nil || identity.TrancheID != "CCMAI-RUNTIME-135" || identity.Host != "127.0.0.1" || identity.Port < 1024 || identity.Port > 65535 || !regexp.MustCompile(`^ccma_r135_[a-z0-9_]{1,32}$`).MatchString(identity.Database) || !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(identity.Nonce) {
		t.Fatal("R135 owned identity invalid")
	}
	dsn := os.Getenv("R135_MYSQL_DSN")
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil || parsed == nil || parsed.Net != "tcp" || parsed.Addr != net.JoinHostPort(identity.Host, strconv.Itoa(identity.Port)) || parsed.DBName != identity.Database || parsed.Params["init_command"] != "" {
		t.Fatal("R135 DSN does not match identity")
	}
	if os.Getenv("R101_MYSQL_DSN") != "" {
		t.Fatal("R101 compatibility DSN must be unset before R135 admission")
	}
	probe, err := sql.Open("mysql", parsed.FormatDSN())
	if err != nil {
		t.Fatal("R135 ownership probe open failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	var nonce string
	var count int
	err = probe.QueryRowContext(ctx, "SELECT COUNT(*), MIN(nonce) FROM ccma_r135_fixture_owner").Scan(&count, &nonce)
	cancel()
	_ = probe.Close()
	if err != nil || count != 1 || nonce != identity.Nonce {
		t.Fatal("R135 preexisting nonce-table ownership proof failed")
	}
	return directory, frontend, dsn, identity
}

type r135Page struct {
	Items []models.InterventionCase `json:"items"`
	Next  string                    `json:"next_cursor"`
	More  bool                      `json:"has_more"`
	Total int                       `json:"total"`
	Order string                    `json:"order"`
}

func r135IDs(rows []models.InterventionCase) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func r135Expected(t *testing.T, tenant, state string) []models.InterventionCase {
	t.Helper()
	var rows []models.InterventionCase
	q := db.DB.Where("tenant_id = ?", tenant)
	if state == "" {
		q = q.Where("work_state IN ?", []string{"OPEN", "ASSIGNED", "WAITING_ON_DATA"})
	} else {
		q = q.Where("work_state = ?", state)
	}
	if err := q.Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return rows
}

// Native GETs use the actual authenticated router; no positive action precedes the browser.
func r135NativePages(t *testing.T, f *m3Fixture, state string, expected []models.InterventionCase, cap *int) {
	t.Helper()
	ids, cursor, visited := []string{}, "", map[string]bool{}
	for page := 0; page < 30; page++ {
		query := "/intervention-cases?limit=7"
		if state != "" {
			query += "&state=" + url.QueryEscape(state)
		}
		if cursor != "" {
			query += "&cursor=" + url.QueryEscape(cursor)
		}
		*cap++
		if *cap > 30 {
			t.Fatal("R135 native GET cap exceeded")
		}
		rec := f.send(http.MethodGet, f.tenant, f.reader, query, nil)
		var got r135Page
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &got) != nil {
			t.Fatalf("R135 native queue %s page%d: %d %s", state, page, rec.Code, rec.Body.String())
		}
		if got.Total != len(expected) || got.Order != "created_at_asc_id_asc" || len(got.Items) == 0 || len(got.Items) > 7 {
			t.Fatalf("R135 native queue metadata %s page%d: %+v", state, page, got)
		}
		for _, row := range got.Items {
			if row.TenantID != f.tenant || visited[row.ID] {
				t.Fatal("R135 queue leaked or repeated case")
			}
			visited[row.ID] = true
			ids = append(ids, row.ID)
		}
		if got.More != (len(ids) < len(expected)) || (got.More && got.Next == "") || (!got.More && got.Next != "") || (got.More && got.Next == cursor) {
			t.Fatalf("R135 cursor/termination mismatch %s page%d", state, page)
		}
		if !got.More {
			if !reflect.DeepEqual(ids, r135IDs(expected)) {
				t.Fatalf("R135 queue order/membership %s: got %v want %v", state, ids, r135IDs(expected))
			}
			return
		}
		cursor = got.Next
	}
	t.Fatal("R135 queue did not terminate")
}

func r135Counts(caseID string) (map[string]int64, error) {
	tables := map[string]any{"human_events": &models.InterventionHumanEvent{}, "attempts": &models.InterventionAttempt{}, "budgets": &models.InterventionBudget{}, "grants": &models.InterventionExecutionGrant{}, "executions": &models.InterventionExecution{}, "receipts": &models.InterventionReceipt{}, "runs": &models.JobRun{}, "proposals": &models.InterventionProposal{}, "snapshots": &models.AnalysisSnapshot{}, "messages": &models.Message{}, "ai_usage_logs": &models.AIUsageLog{}, "activity_logs": &models.ActivityLog{}}
	out := map[string]int64{}
	for key, model := range tables {
		q := db.DB.Model(model)
		if key == "human_events" {
			q = q.Where("case_id = ?", caseID)
		}
		var n int64
		if err := q.Count(&n).Error; err != nil {
			return out, fmt.Errorf("%s count: %w", key, err)
		}
		out[key] = n
	}
	return out, nil
}

func r135Projection(c models.InterventionCase) map[string]any {
	return map[string]any{"version": c.Version, "disposition": c.Disposition, "work_state": c.WorkState, "human_assessment": c.HumanAssessment, "human_explanation": c.HumanExplanation, "human_next_action": c.HumanNextAction, "assignee_id": c.AssigneeID, "data_owner_id": c.DataOwnerID, "due_at": c.DueAt, "outcome_text": c.OutcomeText, "reviewed_data_revision_id": c.ReviewedDataRevisionID}
}

type r135Writer struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *r135Writer) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
		w.ResponseWriter.WriteHeader(code)
	}
}
func (w *r135Writer) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	_, _ = w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func TestR135QueueRejectBrowserFixture(t *testing.T) {
	directory, frontend, dsn, identity := r135Admit(t)
	t.Setenv("R101_MYSQL_DSN", dsn)
	f := m3FixtureStart(t)
	t.Setenv("R101_MYSQL_DSN", "")
	var target models.InterventionCase
	if err := db.DB.Where("id = ?", f.caseID).Take(&target).Error; err != nil {
		t.Fatal(err)
	}
	if target.WorkState != "OPEN" || target.Version != 1 || target.SnapshotID == nil {
		t.Fatal("R135 target seed unexpected")
	}
	if err := db.DB.Model(&models.InterventionCase{}).Where("id = ?", f.caseID).UpdateColumn("active_proposal_id", f.proposal).Error; err != nil {
		t.Fatal(err)
	}
	baseAt := target.CreatedAt.UTC().Truncate(time.Millisecond)
	if err := db.DB.Model(&models.InterventionCase{}).Where("id = ?", f.caseID).UpdateColumns(map[string]any{"created_at": baseAt, "updated_at": baseAt}).Error; err != nil {
		t.Fatal(err)
	}
	var source models.Message
	var snapshot models.AnalysisSnapshot
	var proposal models.InterventionProposal
	if err := db.DB.Where("id = ?", f.message).Take(&source).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Where("id = ?", f.snapshot).Take(&snapshot).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Where("id = ?", f.proposal).Take(&proposal).Error; err != nil {
		t.Fatal(err)
	}
	var parent models.Conversation
	if err := db.DB.Where("id = ?", f.conversation).Take(&parent).Error; err != nil {
		t.Fatal(err)
	}
	// Twenty-three extra OPEN cases tie with the target across both 7-row and 20-row page boundaries.
	for i := 0; i < 23; i++ {
		id, conv := pkg.NewUUID(), pkg.NewUUID()
		m3MustCreate(t, &models.Conversation{ID: conv, TenantID: f.tenant, ChannelID: parent.ChannelID, ExternalConversationID: conv, Metadata: "{}", CreatedAt: baseAt, UpdatedAt: baseAt}, &models.InterventionCase{ID: id, TenantID: f.tenant, JobID: f.job, RunID: pkg.NewUUID(), ConversationID: conv, UseCase: "pancake_complaint_wait", Version: 1, Eligibility: "ELIGIBLE", Execution: "NOT_RUN", Disposition: "PENDING_REVIEW", WorkState: "OPEN", ActorID: f.owner, PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "r135-direct-seed", SchemaVersion: "ccma.intervention.v1", CreatedAt: baseAt, UpdatedAt: baseAt})
	}
	for _, state := range []string{"ASSIGNED", "REJECTED", "RESOLVED"} {
		id, conv := pkg.NewUUID(), pkg.NewUUID()
		m3MustCreate(t, &models.Conversation{ID: conv, TenantID: f.tenant, ChannelID: parent.ChannelID, ExternalConversationID: conv, Metadata: "{}", CreatedAt: baseAt, UpdatedAt: baseAt}, &models.InterventionCase{ID: id, TenantID: f.tenant, JobID: f.job, RunID: pkg.NewUUID(), ConversationID: conv, UseCase: "pancake_complaint_wait", Version: 1, Eligibility: "ELIGIBLE", Execution: "NOT_RUN", Disposition: "PENDING_REVIEW", WorkState: state, ActorID: f.owner, PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "r135-direct-seed", SchemaVersion: "ccma.intervention.v1", CreatedAt: baseAt, UpdatedAt: baseAt})
	}
	foreignID := pkg.NewUUID()
	m3MustCreate(t, &models.InterventionCase{ID: foreignID, TenantID: f.otherTenant, JobID: pkg.NewUUID(), RunID: pkg.NewUUID(), ConversationID: pkg.NewUUID(), UseCase: "pancake_complaint_wait", Version: 1, Eligibility: "ELIGIBLE", Execution: "NOT_RUN", Disposition: "PENDING_REVIEW", WorkState: "OPEN", ActorID: f.owner, PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "r135-direct-seed", SchemaVersion: "ccma.intervention.v1", CreatedAt: baseAt, UpdatedAt: baseAt})
	active, open := r135Expected(t, f.tenant, ""), r135Expected(t, f.tenant, "OPEN")
	waiting, rejected, resolved := r135Expected(t, f.tenant, "WAITING_ON_DATA"), r135Expected(t, f.tenant, "REJECTED"), r135Expected(t, f.tenant, "RESOLVED")
	if len(active) != 26 || len(open) != 24 || len(waiting) != 1 || len(rejected) != 1 || len(resolved) != 1 {
		t.Fatalf("R135 unexpected queue counts: %d/%d/%d/%d/%d", len(active), len(open), len(waiting), len(rejected), len(resolved))
	}
	if !sort.SliceIsSorted(open, func(i, j int) bool {
		if open[i].CreatedAt.Equal(open[j].CreatedAt) {
			return open[i].ID < open[j].ID
		}
		return open[i].CreatedAt.Before(open[j].CreatedAt)
	}) {
		t.Fatal("R135 expected order invalid")
	}
	var beforeTarget models.InterventionCase
	if err := db.DB.Where("id = ?", f.caseID).Take(&beforeTarget).Error; err != nil {
		t.Fatal(err)
	}
	if beforeTarget.ActiveProposalID == nil || *beforeTarget.ActiveProposalID != f.proposal {
		t.Fatal("R135 target proposal pointer missing")
	}
	var beforeCases []models.InterventionCase
	if err := db.DB.Order("id ASC").Find(&beforeCases).Error; err != nil {
		t.Fatal(err)
	}
	var beforeSnapshots []models.AnalysisSnapshot
	var beforeProposals []models.InterventionProposal
	var beforeMessages []models.Message
	if err := db.DB.Order("id ASC").Find(&beforeSnapshots).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Order("id ASC").Find(&beforeProposals).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Order("id ASC").Find(&beforeMessages).Error; err != nil {
		t.Fatal(err)
	}
	beforeCounts, err := r135Counts(f.caseID)
	if err != nil {
		t.Fatal(err)
	}
	nativeGETs := 0
	for _, item := range []struct {
		state string
		rows  []models.InterventionCase
	}{{"", active}, {"OPEN", open}, {"WAITING_ON_DATA", waiting}, {"REJECTED", rejected}, {"RESOLVED", resolved}} {
		r135NativePages(t, f, item.state, item.rows, &nativeGETs)
	}
	if nativeGETs > 30 {
		t.Fatal("R135 native request cap")
	}
	foreign := f.send(http.MethodGet, f.tenant, f.reader, "/intervention-cases/"+foreignID, nil)
	nativeGETs++
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("R135 foreign case leaked: %d", foreign.Code)
	}
	for _, row := range active {
		if row.ID == foreignID {
			t.Fatal("R135 foreign queue leaked")
		}
	}
	if rec := f.send(http.MethodGet, f.otherTenant, f.owner, "/intervention-cases?limit=7", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("R135 foreign tenant route accepted: %d", rec.Code)
	}
	nativeGETs++
	if nativeGETs > 30 {
		t.Fatal("R135 native request cap")
	}
	if afterNative, err := r135Counts(f.caseID); err != nil || !reflect.DeepEqual(beforeCounts, afterNative) {
		t.Fatal("R135 native GET mutated effects")
	}
	if err := r135JSON(filepath.Join(directory, "expected_queue.json"), map[string]any{"trancheId": "CCMAI-RUNTIME-135", "nonce": identity.Nonce, "order": "created_at_asc_id_asc", "before": map[string]any{"default": r135IDs(active), "OPEN": r135IDs(open), "WAITING_ON_DATA": r135IDs(waiting), "REJECTED": r135IDs(rejected), "RESOLVED": r135IDs(resolved)}, "afterExpected": map[string]any{"defaultTotal": len(active) - 1, "openTotal": len(open) - 1, "rejectedTotal": len(rejected) + 1}, "targetCaseId": f.caseID, "foreignCaseId": foreignID, "nativeGETs": nativeGETs}); err != nil {
		t.Fatal(err)
	}
	journal, err := os.OpenFile(filepath.Join(directory, "route_journal.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	base := "/api/v1/tenants/" + f.tenant
	caseAPI := base + "/intervention-cases/" + f.caseID
	var mu sync.Mutex
	authPosts, actionPosts, deniedRoutes := 0, 0, 0
	var actionStatuses []int
	var observationErrors []string
	browserHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, method := r.URL.EscapedPath(), r.Method
		apiRoute := strings.HasPrefix(path, "/api/") || path == "/health"
		allowed := false
		if method == http.MethodGet || method == http.MethodHead {
			if apiRoute {
				allowed = path == "/health" || path == "/api/v1/setup/status" || path == "/api/v1/profile" || path == "/api/v1/tenants" || path == base || path == base+"/me" || path == base+"/demo/status" || path == base+"/onboarding-status" || path == base+"/dashboard" || path == base+"/intervention-capability" || path == base+"/intervention-assignees" || path == base+"/intervention-cases" || path == caseAPI
			}
			if !apiRoute {
				allowed = path == "/" || path == "/login" || path == "/"+f.tenant+"/intervention-cases" || path == "/"+f.tenant+"/intervention-cases/"+f.caseID || path == "/favicon.png" || (strings.HasPrefix(path, "/assets/") && !strings.Contains(path, ".."))
			}
		} else if method == http.MethodPost {
			mu.Lock()
			switch path {
			case "/api/v1/auth/login", "/api/v1/auth/logout", "/api/v1/auth/refresh":
				authPosts++
				allowed = authPosts <= 4
			case caseAPI + "/actions":
				actionPosts++
				allowed = actionPosts <= 1
			}
			mu.Unlock()
		}
		writer := &r135Writer{ResponseWriter: w}
		if !allowed {
			http.Error(writer, "R135 route denied", http.StatusForbidden)
		} else if apiRoute {
			f.router.ServeHTTP(writer, r)
		} else if path == "/" || path == "/login" || strings.HasPrefix(path, "/"+f.tenant+"/intervention-cases") {
			http.ServeFile(writer, r, filepath.Join(frontend, "index.html"))
		} else {
			http.FileServer(http.Dir(frontend)).ServeHTTP(writer, r)
		}
		mu.Lock()
		if method == http.MethodPost && path == caseAPI+"/actions" {
			actionStatuses = append(actionStatuses, writer.status)
		}
		if !allowed {
			deniedRoutes++
		}
		entry := map[string]any{"method": method, "path": path, "query": r.URL.RawQuery, "status": writer.status}
		if method == http.MethodGet && path == base+"/intervention-cases" && writer.status == http.StatusOK {
			var page r135Page
			if json.Unmarshal(writer.body.Bytes(), &page) != nil {
				observationErrors = append(observationErrors, "queue_journal_decode")
			} else {
				entry["state"] = r.URL.Query().Get("state")
				entry["cursor"] = r.URL.Query().Get("cursor")
				entry["ids"] = r135IDs(page.Items)
				entry["total"] = page.Total
				entry["order"] = page.Order
				entry["has_more"] = page.More
				entry["next_cursor"] = page.Next
			}
		}
		b, e := json.Marshal(entry)
		if e == nil {
			_, e = journal.Write(append(b, '\n'))
		}
		if e != nil {
			observationErrors = append(observationErrors, "route_journal_write: "+e.Error())
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
	serveJoined, terminalWritten := false, false
	serveResult, stopReason := "", "startup_or_assertion_failure"
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = server.Shutdown(ctx)
		cancel()
		_ = server.Close()
		_ = listener.Close()
		if !serveJoined {
			select {
			case e := <-served:
				serveResult = fmt.Sprint(e)
				serveJoined = true
			case <-time.After(5 * time.Second):
				serveResult = "serve_join_timeout"
				t.Error("R135 Serve did not join")
			}
		}
		if !terminalWritten {
			counts, countErr := r135Counts(f.caseID)
			mu.Lock()
			errors := append([]string(nil), observationErrors...)
			mu.Unlock()
			if countErr != nil {
				errors = append(errors, "terminal_counts: "+countErr.Error())
			}
			result := map[string]any{"trancheId": "CCMAI-RUNTIME-135", "nonce": identity.Nonce, "stopReason": stopReason, "serveJoined": serveJoined, "serveResult": serveResult, "testFailed": t.Failed() || len(errors) > 0, "observationErrors": errors, "authPosts": authPosts, "actionPosts": actionPosts, "actionStatuses": actionStatuses, "deniedRoutes": deniedRoutes, "counts": counts, "finishedAt": time.Now().UTC().Format(time.RFC3339Nano)}
			if err := r135JSON(filepath.Join(directory, "backend_terminal.json"), result); err != nil {
				t.Error("R135 terminal write failed: ", err)
			} else {
				terminalWritten = true
			}
		}
	}()
	origin := "http://" + listener.Addr().String()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(origin + "/health")
	if err != nil {
		t.Fatal("R135 health unavailable: ", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal("R135 health failed")
	}
	ready := map[string]any{"trancheId": "CCMAI-RUNTIME-135", "nonce": identity.Nonce, "origin": origin, "tenantId": f.tenant, "caseId": f.caseID, "ownerEmail": f.owner + "@example.invalid", "readerEmail": f.reader + "@example.invalid", "password": "R101SyntheticOnly!123", "synthetic": true, "browserCaps": map[string]int{"authPosts": 4, "actionPosts": 1}, "expected": map[string]any{"default": r135IDs(active), "OPEN": r135IDs(open), "WAITING_ON_DATA": r135IDs(waiting), "REJECTED": r135IDs(rejected), "RESOLVED": r135IDs(resolved), "defaultTotal": len(active), "openTotal": len(open), "rejectedTotal": len(rejected), "order": "created_at_asc_id_asc"}}
	if err := r135JSON(filepath.Join(directory, "ready.json"), ready); err != nil {
		t.Fatal(err)
	}
	fmt.Println("R135_BROWSER_READY " + origin)
	stopPath := filepath.Join(directory, "stop.json")
	deadline := time.NewTimer(20 * time.Minute)
	defer deadline.Stop()
	for {
		select {
		case e := <-served:
			serveJoined = true
			serveResult = fmt.Sprint(e)
			t.Error("R135 Serve exited before owned stop")
			stopReason = "early_serve_exit"
			goto final
		case <-deadline.C:
			t.Error("R135 browser stop timeout")
			stopReason = "timeout"
			goto final
		case <-time.After(200 * time.Millisecond):
			data, e := os.ReadFile(stopPath)
			if os.IsNotExist(e) {
				continue
			}
			var stop struct {
				TrancheID string `json:"trancheId"`
				Nonce     string `json:"nonce"`
			}
			if e != nil || json.Unmarshal(data, &stop) != nil || stop.TrancheID != "CCMAI-RUNTIME-135" || stop.Nonce != identity.Nonce {
				t.Error("R135 stop ownership invalid")
				stopReason = "invalid_stop"
				goto final
			}
			stopReason = "owned_stop"
			goto final
		}
	}
final:
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = server.Shutdown(ctx)
	cancel()
	_ = server.Close()
	_ = listener.Close()
	if !serveJoined {
		select {
		case e := <-served:
			serveResult = fmt.Sprint(e)
			serveJoined = true
		case <-time.After(5 * time.Second):
			t.Error("R135 Serve join timeout")
		}
	}
	var current models.InterventionCase
	var afterSource models.Message
	var afterSnapshot models.AnalysisSnapshot
	var afterProposal models.InterventionProposal
	var afterCases []models.InterventionCase
	var afterSnapshots []models.AnalysisSnapshot
	var afterProposals []models.InterventionProposal
	var afterMessages []models.Message
	var events []models.InterventionHumanEvent
	for _, check := range []struct {
		label string
		err   error
	}{{"target", db.DB.Where("id = ?", f.caseID).Take(&current).Error}, {"message", db.DB.Where("id = ?", f.message).Take(&afterSource).Error}, {"snapshot", db.DB.Where("id = ?", f.snapshot).Take(&afterSnapshot).Error}, {"proposal", db.DB.Where("id = ?", f.proposal).Take(&afterProposal).Error}, {"cases", db.DB.Order("id ASC").Find(&afterCases).Error}, {"snapshots", db.DB.Order("id ASC").Find(&afterSnapshots).Error}, {"proposals", db.DB.Order("id ASC").Find(&afterProposals).Error}, {"messages", db.DB.Order("id ASC").Find(&afterMessages).Error}, {"events", db.DB.Where("case_id = ?", f.caseID).Order("case_version ASC").Find(&events).Error}} {
		if check.err != nil {
			t.Errorf("R135 final %s: %v", check.label, check.err)
		}
	}
	counts, countErr := r135Counts(f.caseID)
	if countErr != nil {
		t.Error("R135 final counts: ", countErr)
	}
	if stopReason != "owned_stop" || !serveJoined || !reflect.DeepEqual(source, afterSource) || !reflect.DeepEqual(snapshot, afterSnapshot) || !reflect.DeepEqual(proposal, afterProposal) {
		t.Error("R135 lifecycle/source invariant failed")
	}
	if !reflect.DeepEqual(beforeSnapshots, afterSnapshots) || !reflect.DeepEqual(beforeProposals, afterProposals) || !reflect.DeepEqual(beforeMessages, afterMessages) {
		t.Error("R135 non-target or target source row changed")
	}
	if len(events) != 1 {
		t.Error("R135 exact reject event count failed")
	} else {
		event := events[0]
		var beforeEvent, afterEvent map[string]any
		beforeErr, afterErr := json.Unmarshal([]byte(event.Before), &beforeEvent), json.Unmarshal([]byte(event.After), &afterEvent)
		beforeJSON, _ := json.Marshal(r135Projection(beforeTarget))
		afterJSON, _ := json.Marshal(r135Projection(current))
		var expectedBefore, expectedAfter map[string]any
		_ = json.Unmarshal(beforeJSON, &expectedBefore)
		_ = json.Unmarshal(afterJSON, &expectedAfter)
		if event.Action != "REJECT" || event.CaseVersion != 2 || strings.TrimSpace(event.Reason) == "" || event.ActorID != f.owner || event.ProposalID == nil || *event.ProposalID != f.proposal || event.SnapshotID != f.snapshot || event.SnapshotDigest != beforeTarget.SnapshotDigest || beforeErr != nil || afterErr != nil || !reflect.DeepEqual(beforeEvent, expectedBefore) || !reflect.DeepEqual(afterEvent, expectedAfter) {
			t.Error("R135 exact reject event/projection failed")
		}
	}
	if current.Version != 2 || current.WorkState != "REJECTED" || current.Disposition != "HUMAN_REJECT" || current.HumanAssessment != "REJECT" || current.Execution != beforeTarget.Execution || current.RunID != beforeTarget.RunID || current.SnapshotDigest != beforeTarget.SnapshotDigest || !reflect.DeepEqual(current.SnapshotID, beforeTarget.SnapshotID) || !reflect.DeepEqual(current.ActiveProposalID, beforeTarget.ActiveProposalID) {
		t.Error("R135 target reject projection failed")
	}
	if len(afterCases) != len(beforeCases) {
		t.Error("R135 case count changed")
	} else {
		for i := range beforeCases {
			if beforeCases[i].ID != afterCases[i].ID || (beforeCases[i].ID != f.caseID && !reflect.DeepEqual(beforeCases[i], afterCases[i])) {
				t.Error("R135 non-target case changed: ", beforeCases[i].ID)
			}
		}
	}
	for key, before := range beforeCounts {
		if key != "human_events" && key != "activity_logs" && counts[key] != before {
			t.Errorf("R135 protected effect %s changed", key)
		}
	}
	if counts["human_events"] != beforeCounts["human_events"]+1 {
		t.Error("R135 event count delta failed")
	}
	if !reflect.DeepEqual(actionStatuses, []int{http.StatusOK}) || actionPosts != 1 || authPosts > 4 {
		t.Error("R135 browser action/auth cap failed")
	}
	mu.Lock()
	errors := append([]string(nil), observationErrors...)
	mu.Unlock()
	if len(errors) > 0 {
		t.Error("R135 route observations: ", errors)
	}
	result := map[string]any{"trancheId": "CCMAI-RUNTIME-135", "nonce": identity.Nonce, "stopReason": stopReason, "serveJoined": serveJoined, "serveResult": serveResult, "testFailed": t.Failed(), "observationErrors": errors, "authPosts": authPosts, "actionPosts": actionPosts, "actionStatuses": actionStatuses, "deniedRoutes": deniedRoutes, "nativeGETs": nativeGETs, "countsBefore": beforeCounts, "countsAfter": counts, "auditActivityBefore": beforeCounts["activity_logs"], "auditActivityAfter": counts["activity_logs"], "caseVersion": current.Version, "workState": current.WorkState, "disposition": current.Disposition, "eventCount": len(events), "finishedAt": time.Now().UTC().Format(time.RFC3339Nano)}
	if err := r135JSON(filepath.Join(directory, "backend_terminal.json"), result); err != nil {
		t.Error("R135 terminal write failed: ", err)
	} else {
		terminalWritten = true
	}
}
