package handlers_test

// Opt-in, disposable MySQL browser fixture for an enacted human workflow.
// No provider or background intervention runner is installed.
import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

type m3WorkflowStatusWriter struct {
	http.ResponseWriter
	code int
}

func (w *m3WorkflowStatusWriter) WriteHeader(code int) {
	if w.code != 0 {
		return
	}
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *m3WorkflowStatusWriter) Write(p []byte) (int, error) {
	if w.code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func m3WorkflowObserve(t *testing.T, caseID string) (models.InterventionCase, []models.InterventionHumanEvent) {
	t.Helper()
	var row models.InterventionCase
	var events []models.InterventionHumanEvent
	if err := db.DB.Where("id = ?", caseID).Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Where("case_id = ?", caseID).Order("case_version ASC").Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	return row, events
}

func m3WorkflowHash(value any) string {
	b, _ := json.Marshal(value)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func m3WorkflowSafeCase(c models.InterventionCase, events []models.InterventionHumanEvent) map[string]any {
	history := make([]map[string]any, 0, len(events))
	for _, event := range events {
		var before map[string]any
		_ = json.Unmarshal([]byte(event.Before), &before)
		outcome, _ := before["outcome_text"].(string)
		history = append(history, map[string]any{"version": event.CaseVersion, "action": event.Action, "priorOutcomePresent": outcome != ""})
	}
	return map[string]any{"id": c.ID, "version": c.Version, "workState": c.WorkState, "disposition": c.Disposition, "assessment": c.HumanAssessment, "assigneeId": c.AssigneeID, "dataOwnerId": c.DataOwnerID, "outcomePresent": c.OutcomeText != "", "history": history}
}

func TestM3WorkflowBrowserFixture(t *testing.T) {
	if os.Getenv("M3_WORKFLOW_BROWSER") != "1" {
		t.Skip("explicit synthetic workflow browser opt-in required")
	}
	directory, frontend := os.Getenv("M3_WORKFLOW_DIRECTORY"), os.Getenv("M3_WORKFLOW_FRONTEND")
	if !filepath.IsAbs(directory) || !filepath.IsAbs(frontend) {
		t.Fatal("absolute evidence and frontend directories required")
	}
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(directory) {
		t.Fatal("fresh non-symlink evidence directory required")
	}
	for _, name := range []string{"ready.json", "backend_terminal.json", "route_journal.jsonl", "stop"} {
		if _, err := os.Lstat(filepath.Join(directory, name)); !os.IsNotExist(err) {
			t.Fatal("evidence output already exists: " + name)
		}
	}
	if info, err := os.Stat(filepath.Join(frontend, "index.html")); err != nil || !info.Mode().IsRegular() {
		t.Fatal("frontend build missing")
	}
	var identity struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Database string `json:"database"`
		Nonce    string `json:"nonce"`
	}
	identityBytes, err := os.ReadFile(filepath.Join(directory, "fixture_identity.json"))
	if err != nil || json.Unmarshal(identityBytes, &identity) != nil || identity.Host != "ccma-r123-mysql-54bc229d" || identity.Port != 3306 || identity.Database != "review_m3_workflow" || len(identity.Nonce) != 32 {
		t.Fatal("owned workflow fixture identity required")
	}
	parsed, err := mysqlDriver.ParseDSN(os.Getenv("R101_MYSQL_DSN"))
	if err != nil || parsed == nil || parsed.Net != "tcp" || parsed.Addr != net.JoinHostPort(identity.Host, strconv.Itoa(identity.Port)) || parsed.DBName != identity.Database || parsed.Params["init_command"] != "" {
		t.Fatal("workflow DSN does not match owned fixture")
	}
	probe, err := sql.Open("mysql", parsed.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	ctxProbe, cancelProbe := context.WithTimeout(context.Background(), 5*time.Second)
	var nonce string
	err = probe.QueryRowContext(ctxProbe, "SELECT nonce FROM review_fix_owner LIMIT 1").Scan(&nonce)
	cancelProbe()
	_ = probe.Close()
	if err != nil || nonce != identity.Nonce {
		t.Fatal("workflow fixture ownership not proven")
	}

	f := m3FixtureStart(t)
	if err := db.DB.Model(&models.InterventionCase{}).Where("id = ?", f.caseID).Update("active_proposal_id", f.proposal).Error; err != nil {
		t.Fatal(err)
	}
	for id, name := range map[string]string{f.assignee: "Người xử lý minh họa", f.reader: "Người xem minh họa"} {
		if err := db.DB.Model(&models.User{}).Where("id = ?", id).Update("name", name).Error; err != nil {
			t.Fatal(err)
		}
	}
	initialCase, _ := m3WorkflowObserve(t, f.caseID)
	initialWait, _ := m3WorkflowObserve(t, f.waitID)
	var proposal models.InterventionProposal
	if err := db.DB.Where("id = ?", f.proposal).Take(&proposal).Error; err != nil {
		t.Fatal(err)
	}
	initialProposalHash := m3WorkflowHash(proposal)
	initialEffects := r121EffectCounts(t)
	initialFingerprints := map[string]string{"case": r121Fingerprint(t, f.caseID), "waitCase": r121Fingerprint(t, f.waitID)}
	readerWriteDeniedNative := f.send(http.MethodPost, f.tenant, f.reader, "/intervention-cases/"+f.caseID+"/actions", f.action(f.reader, f.caseID, "CONFIRM", 1, nil)).Code == http.StatusForbidden && r121Fingerprint(t, f.caseID) == initialFingerprints["case"] && r121EffectCounts(t)["humanEvents"] == initialEffects["humanEvents"]

	journal, err := os.OpenFile(filepath.Join(directory, "route_journal.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	var mu sync.Mutex
	authCount, actionCount := 0, 0
	ownerLogin, readerLogin, readerDetail, staleConflict := false, false, false, false
	var journalErr error
	base := "/api/v1/tenants/" + f.tenant
	caseBase := base + "/intervention-cases"
	casePath, waitPath := caseBase+"/"+f.caseID, caseBase+"/"+f.waitID
	staticFiles := http.FileServer(http.Dir(frontend))
	guard := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.EscapedPath()
		method := r.Method
		isRead := method == http.MethodGet || method == http.MethodHead
		api := strings.HasPrefix(path, "/api/") || path == "/health"
		action := ""
		loginEmail := ""
		allowed := false
		if isRead {
			if api {
				allowed = path == "/health" || path == "/api/v1/setup/status" || path == "/api/v1/profile" || path == "/api/v1/tenants" || path == base || path == base+"/me" || path == base+"/demo/status" || path == base+"/onboarding-status" || path == base+"/dashboard" || path == base+"/intervention-capability" || path == caseBase || path == casePath || path == waitPath || path == base+"/intervention-assignees"
			} else {
				allowed = path == "/" || path == "/login" || path == "/"+f.tenant+"/intervention-cases" || path == "/"+f.tenant+"/intervention-cases/"+f.caseID || path == "/"+f.tenant+"/intervention-cases/"+f.waitID
				if strings.HasPrefix(path, "/assets/") && !strings.Contains(path, "..") {
					name := filepath.Join(frontend, filepath.FromSlash(strings.TrimPrefix(path, "/")))
					if info, err := os.Stat(name); err == nil && info.Mode().IsRegular() {
						allowed = true
					}
				}
				if path == "/favicon.png" {
					if info, err := os.Stat(filepath.Join(frontend, "favicon.png")); err == nil && info.Mode().IsRegular() {
						allowed = true
					}
				}
			}
		}
		if method == http.MethodPost {
			if path == "/api/v1/auth/login" || path == "/api/v1/auth/logout" || path == "/api/v1/auth/refresh" {
				body, readErr := io.ReadAll(io.LimitReader(r.Body, 16*1024+1))
				if readErr == nil && len(body) <= 16*1024 {
					r.Body = io.NopCloser(bytes.NewReader(body))
					if path == "/api/v1/auth/login" {
						var login struct {
							Email string `json:"email"`
						}
						if json.Unmarshal(body, &login) == nil {
							loginEmail = login.Email
						}
					}
					mu.Lock()
					authCount++
					allowed = authCount <= 5
					mu.Unlock()
				}
			} else if path == casePath+"/actions" || path == waitPath+"/actions" {
				body, readErr := io.ReadAll(io.LimitReader(r.Body, 16*1024+1))
				if readErr == nil && len(body) <= 16*1024 {
					var request struct {
						Action string `json:"action"`
					}
					if json.Unmarshal(body, &request) == nil {
						action = request.Action
						validCase := path == casePath+"/actions" && (action == "CORRECT" || action == "CONFIRM" || action == "ASSIGN" || action == "RESOLVE" || action == "REOPEN")
						validWait := path == waitPath+"/actions" && (action == "ASSIGN_DATA" || action == "ESCALATE_DATA")
						mu.Lock()
						actionCount++
						allowed = actionCount <= 10 && (validCase || validWait)
						mu.Unlock()
					}
				}
				r.Body = io.NopCloser(bytes.NewReader(body))
			}
		}
		writer := &m3WorkflowStatusWriter{ResponseWriter: w}
		if !allowed {
			http.Error(writer, "workflow fixture route denied", http.StatusForbidden)
		} else if api {
			f.router.ServeHTTP(writer, r)
		} else if path == "/" || path == "/login" || strings.HasPrefix(path, "/"+f.tenant+"/intervention-cases") {
			http.ServeFile(writer, r, filepath.Join(frontend, "index.html"))
		} else {
			staticFiles.ServeHTTP(writer, r)
		}
		if allowed && writer.code == http.StatusOK {
			mu.Lock()
			if path == "/api/v1/auth/login" {
				if loginEmail == f.owner+"@example.invalid" {
					ownerLogin = true
				}
				if loginEmail == f.reader+"@example.invalid" {
					readerLogin = true
				}
			}
			if path == casePath || path == waitPath {
				if token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); token != r.Header.Get("Authorization") {
					if claims, err := middleware.ParseToken(token); err == nil && claims.UserID == f.reader {
						readerDetail = true
					}
				}
			}
			mu.Unlock()
		}
		if allowed && method == http.MethodPost && path == casePath+"/actions" && writer.code == http.StatusConflict {
			mu.Lock()
			staleConflict = true
			mu.Unlock()
		}
		entry, _ := json.Marshal(map[string]any{"method": method, "path": path, "action": action, "status": writer.code, "allowed": allowed})
		mu.Lock()
		if journalErr == nil {
			_, journalErr = journal.Write(append(entry, '\n'))
			if journalErr == nil {
				journalErr = journal.Sync()
			}
		}
		mu.Unlock()
	})
	ln, err := net.Listen("tcp", "0.0.0.0:51112")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	server := &http.Server{Handler: guard, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(ln) }()
	origin := "http://127.0.0.1:51112"
	client := &http.Client{Timeout: 3 * time.Second}
	health, err := client.Get(origin + "/health")
	if err != nil {
		t.Fatal("workflow fixture health failed: " + err.Error())
	}
	_ = health.Body.Close()
	if health.StatusCode != http.StatusOK {
		t.Fatal("workflow fixture health status failed")
	}
	select {
	case err := <-done:
		t.Fatalf("server exited before readiness: %v", err)
	default:
	}
	ready, _ := json.Marshal(map[string]any{"apiOrigin": origin, "tenant": f.tenant, "case": f.caseID, "waitCase": f.waitID, "assigneeID": f.assignee, "ownerEmail": f.owner + "@example.invalid", "readerEmail": f.reader + "@example.invalid", "synthetic": true})
	if err := reviewFixWriteNew(filepath.Join(directory, "ready.json"), ready); err != nil {
		t.Fatal(err)
	}
	fmt.Println("M3_WORKFLOW_BROWSER_READY " + string(ready))
	stop := filepath.Join(directory, "stop")
	deadline := time.NewTimer(15 * time.Minute)
	defer deadline.Stop()
	serveEnded := false
	var serveErr error
	stopReason := "stop_file"
loop:
	for {
		select {
		case serveErr = <-done:
			serveEnded = true
			stopReason = "early_server_exit"
			break loop
		case <-deadline.C:
			stopReason = "deadline"
			break loop
		case <-time.After(200 * time.Millisecond):
			if _, err := os.Stat(stop); err == nil {
				break loop
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	shutdownErr := server.Shutdown(ctx)
	if !serveEnded {
		select {
		case serveErr = <-done:
			serveEnded = true
		case <-ctx.Done():
		}
	}
	cancel()
	finalCase, caseEvents := m3WorkflowObserve(t, f.caseID)
	finalWait, waitEvents := m3WorkflowObserve(t, f.waitID)
	var finalProposal models.InterventionProposal
	if err := db.DB.Where("id = ?", f.proposal).Take(&finalProposal).Error; err != nil {
		t.Fatal(err)
	}
	finalEffects := r121EffectCounts(t)
	finalFingerprints := map[string]string{"case": r121Fingerprint(t, f.caseID), "waitCase": r121Fingerprint(t, f.waitID)}
	caseActions, waitActions := []string{"CORRECT", "ASSIGN", "RESOLVE", "REOPEN"}, []string{"ASSIGN_DATA", "ESCALATE_DATA"}
	caseOK := finalCase.Version == 5 && finalCase.WorkState == "ASSIGNED" && finalCase.Disposition == "HUMAN_CORRECT" && finalCase.HumanAssessment == "CORRECT" && finalCase.AssigneeID != nil && *finalCase.AssigneeID == f.assignee && finalCase.DueAt != nil && finalCase.DueAt.After(time.Now()) && finalCase.OutcomeText == "" && len(caseEvents) == len(caseActions)
	for i, action := range caseActions {
		if i < len(caseEvents) && (caseEvents[i].Action != action || caseEvents[i].CaseVersion != i+2) {
			caseOK = false
		}
	}
	if len(caseEvents) == 4 {
		var before, after map[string]any
		_ = json.Unmarshal([]byte(caseEvents[3].Before), &before)
		_ = json.Unmarshal([]byte(caseEvents[3].After), &after)
		if outcome, _ := before["outcome_text"].(string); outcome == "" || after["outcome_text"] != "" {
			caseOK = false
		}
	}
	waitOK := finalWait.Version == 3 && finalWait.WorkState == "WAITING_ON_DATA" && finalWait.Disposition == "WAIT_DATA" && finalWait.DataOwnerID != nil && *finalWait.DataOwnerID == f.assignee && finalWait.DueAt != nil && finalWait.DueAt.After(time.Now()) && strings.TrimSpace(finalWait.HumanNextAction) != "" && finalWait.NextAction == initialWait.NextAction && len(waitEvents) == len(waitActions)
	for i, action := range waitActions {
		if i < len(waitEvents) && (waitEvents[i].Action != action || waitEvents[i].CaseVersion != i+2) {
			waitOK = false
		}
	}
	for _, event := range caseEvents {
		caseOK = caseOK && event.ActorID == f.owner && strings.TrimSpace(event.Reason) != "" && !event.CreatedAt.IsZero()
	}
	for _, event := range waitEvents {
		waitOK = waitOK && event.ActorID == f.owner && strings.TrimSpace(event.Reason) != "" && !event.CreatedAt.IsZero()
	}
	sourceIdentityOK := m3WorkflowHash([]any{initialCase.ID, initialCase.TenantID, initialCase.JobID, initialCase.RunID, initialCase.ConversationID, initialCase.SnapshotID, initialCase.SnapshotDigest, initialCase.ActiveProposalID, proposal.ID, proposal.RunID, proposal.SnapshotID, proposal.SnapshotDigest}) == m3WorkflowHash([]any{finalCase.ID, finalCase.TenantID, finalCase.JobID, finalCase.RunID, finalCase.ConversationID, finalCase.SnapshotID, finalCase.SnapshotDigest, finalCase.ActiveProposalID, finalProposal.ID, finalProposal.RunID, finalProposal.SnapshotID, finalProposal.SnapshotDigest}) && m3WorkflowHash([]any{initialWait.ID, initialWait.RunID, initialWait.SnapshotID, initialWait.SnapshotDigest}) == m3WorkflowHash([]any{finalWait.ID, finalWait.RunID, finalWait.SnapshotID, finalWait.SnapshotDigest})
	effectsOK := finalEffects["humanEvents"] == initialEffects["humanEvents"]+6
	for _, key := range []string{"attempts", "executions", "budgets", "grants", "runs", "receipts", "snapshots"} {
		effectsOK = effectsOK && finalEffects[key] == initialEffects[key]
	}
	mu.Lock()
	loggedErr, totalAuth, totalActions, sawOwnerLogin, sawReaderLogin, sawReaderDetail, sawStaleConflict := journalErr, authCount, actionCount, ownerLogin, readerLogin, readerDetail, staleConflict
	mu.Unlock()
	serverStopped := shutdownErr == nil && serveEnded && serveErr == http.ErrServerClosed
	pass := stopReason == "stop_file" && serverStopped && loggedErr == nil && caseOK && waitOK && effectsOK && sourceIdentityOK && readerWriteDeniedNative && sawStaleConflict && sawOwnerLogin && sawReaderLogin && sawReaderDetail && initialProposalHash == m3WorkflowHash(finalProposal) && initialCase.Version == 1 && initialWait.Version == 1
	receipt, _ := json.Marshal(map[string]any{"pass": pass, "stopReason": stopReason, "serverStopped": serverStopped, "journalError": fmt.Sprint(loggedErr), "authPosts": totalAuth, "actionPosts": totalActions, "ownerLogin": sawOwnerLogin, "readerLogin": sawReaderLogin, "readerDetailRead": sawReaderDetail, "readerWriteDeniedNative": readerWriteDeniedNative, "staleActionConflict": sawStaleConflict, "sourceIdentityUnchanged": sourceIdentityOK, "initialFingerprints": initialFingerprints, "finalFingerprints": finalFingerprints, "initialEffects": initialEffects, "finalEffects": finalEffects, "proposalUnchanged": initialProposalHash == m3WorkflowHash(finalProposal), "case": m3WorkflowSafeCase(finalCase, caseEvents), "waitCase": m3WorkflowSafeCase(finalWait, waitEvents)})
	if err := reviewFixWriteNew(filepath.Join(directory, "backend_terminal.json"), receipt); err != nil {
		t.Error(err)
	}
	if !pass {
		t.Error("workflow browser fixture did not meet durable acceptance")
	}
}
