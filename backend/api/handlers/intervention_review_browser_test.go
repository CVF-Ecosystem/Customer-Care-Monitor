package handlers_test

// Current-source, opt-in synthetic presentation acceptance. This fixture has
// no provider and is independent of immutable historical campaign admissions.
import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

func reviewFixWriteNew(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write(data); err != nil {
		return err
	}
	return f.Sync()
}

func TestReviewFixBrowserFixture(t *testing.T) {
	if os.Getenv("REVIEW_FIX_BROWSER") != "1" {
		t.Skip("explicit synthetic browser opt-in required")
	}
	directory := os.Getenv("REVIEW_FIX_DIRECTORY")
	if directory == "" || !filepath.IsAbs(directory) {
		t.Fatal("absolute fresh evidence directory required")
	}

	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(directory) {
		t.Fatal("fresh non-symlink evidence directory required")
	}
	for _, name := range []string{"ready.json", "backend_terminal.json"} {
		if _, err := os.Lstat(filepath.Join(directory, name)); !os.IsNotExist(err) {
			t.Fatal("evidence outputs must not already exist")
		}
	}
	stop := filepath.Join(directory, "stop")
	if _, err := os.Stat(stop); !os.IsNotExist(err) {
		t.Fatal("stop must not exist before start")
	}

	// Validate an owned temporary schema and its nonce BEFORE the helper may
	// migrate or seed. An arbitrary R101_MYSQL_DSN must never be accepted here.
	var identity struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Database string `json:"database"`
		Nonce    string `json:"nonce"`
	}
	identityBytes, err := os.ReadFile(filepath.Join(directory, "fixture_identity.json"))
	if err != nil || json.Unmarshal(identityBytes, &identity) != nil || (identity.Host != "127.0.0.1" || identity.Port < 1024) && !(os.Getenv("REVIEW_FIX_DOCKER") == "1" && regexp.MustCompile(`^ccma-r123-mysql-[a-f0-9]{8}$`).MatchString(identity.Host) && identity.Port == 3306) || identity.Database != "review_fix_ui" || len(identity.Nonce) != 32 {
		t.Fatal("owned disposable fixture identity required")
	}
	parsed, err := mysqlDriver.ParseDSN(os.Getenv("R101_MYSQL_DSN"))
	if err != nil || parsed.Net != "tcp" || parsed.Addr != net.JoinHostPort(identity.Host, strconv.Itoa(identity.Port)) || parsed.DBName != identity.Database {
		t.Fatal("browser DSN is not the admitted loopback fixture")
	}
	// Reject parameters that could change connection routing or run session SQL.
	if parsed.Params["init_command"] != "" {
		t.Fatal("unexpected DSN session command")
	}
	probe, err := sql.Open("mysql", parsed.FormatDSN())
	if err != nil {
		t.Fatal("fixture ownership connection failed")
	}
	ctxProbe, cancelProbe := context.WithTimeout(context.Background(), 5*time.Second)
	var nonce string
	err = probe.QueryRowContext(ctxProbe, "SELECT nonce FROM review_fix_owner LIMIT 1").Scan(&nonce)
	cancelProbe()
	probe.Close()
	if err != nil || nonce != identity.Nonce {
		t.Fatal("disposable fixture ownership not proven")
	}
	f := m3FixtureStart(t)
	var current models.InterventionCase
	var original models.InterventionProposal
	if err := db.DB.Where("id = ?", f.caseID).Take(&current).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Where("id = ?", f.proposal).Take(&original).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Model(&original).Update("run_id", pkg.NewUUID()).Error; err != nil {
		t.Fatal(err)
	}
	active := original
	active.ID = pkg.NewUUID()
	active.Revision = 2
	active.RunID = current.RunID
	active.Explanation = "Minh họa: khách đang chờ phản hồi về đơn hàng."
	active.NextAction = "Kiểm tra đơn hàng và phản hồi khách."
	later := active
	later.ID = pkg.NewUUID()
	later.Revision = 3
	later.RunID = pkg.NewUUID()
	later.Explanation = "Minh họa lịch sử: đề xuất khác, không phải đề xuất hiện tại."
	if err := db.DB.Create(&active).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&later).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Model(&current).Updates(map[string]any{"active_proposal_id": active.ID, "version": 4}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i, action := range []string{"OBSERVE_DATA", "REQUEST_RERUN", "EXECUTE_RERUN"} {
		before, _ := json.Marshal(map[string]any{"work_state": "WAITING_ON_DATA", "disposition": "WAIT_DATA"})
		after, _ := json.Marshal(map[string]any{"work_state": "OPEN", "disposition": "PENDING_REVIEW"})
		reason := "Minh họa: kiểm tra thông tin bổ sung."
		if action == "EXECUTE_RERUN" {
			reason = "approved execution"
		}
		event := models.InterventionHumanEvent{ID: pkg.NewUUID(), TenantID: f.tenant, CaseID: f.caseID, CaseVersion: i + 2, ActorID: f.owner, Action: action, Reason: reason, ProposalID: &active.ID, SnapshotID: f.snapshot, SnapshotDigest: current.SnapshotDigest, SourceStatus: "LOCAL_MATCH_CURRENTNESS_UNVERIFIED", Before: string(before), After: string(after), CreatedAt: now.Add(time.Duration(i) * time.Second)}
		if err := db.DB.Create(&event).Error; err != nil {
			t.Fatal(err)
		}
	}
	initial := r121Fingerprint(t, f.caseID)
	effects := r121EffectCounts(t)
	listenAddress := "127.0.0.1:0"
	if os.Getenv("REVIEW_FIX_DOCKER") == "1" {
		listenAddress = "0.0.0.0:51111"
	}
	ln, err := net.Listen("tcp", listenAddress)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	var handler http.Handler = f.router
	if root := os.Getenv("REVIEW_FIX_FRONTEND"); root != "" {
		if !filepath.IsAbs(root) {
			t.Fatal("absolute frontend build directory required")
		}
		if info, err := os.Stat(filepath.Join(root, "index.html")); err != nil || !info.Mode().IsRegular() {
			t.Fatal("frontend build missing")
		}
		files := http.FileServer(http.Dir(root))
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" {
				f.router.ServeHTTP(w, r)
				return
			}
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			target := filepath.Join(root, filepath.FromSlash(r.URL.Path))
			if info, err := os.Stat(target); err == nil && info.Mode().IsRegular() {
				files.ServeHTTP(w, r)
				return
			}
			http.ServeFile(w, r, filepath.Join(root, "index.html"))
		})
	}
	server := &http.Server{Handler: r121GuardForCase(handler, f.tenant, f.caseID), ReadHeaderTimeout: 5 * time.Second}
	defer server.Close()
	done := make(chan error, 1)
	go func() { done <- server.Serve(ln) }()
	origin := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(ln.Addr().(*net.TCPAddr).Port))
	manifest := map[string]any{"apiOrigin": origin, "tenant": f.tenant, "case": f.caseID, "ownerEmail": f.owner + "@example.invalid", "activeProposalId": active.ID, "offLineageProposalId": later.ID, "synthetic": true}
	b, _ := json.Marshal(manifest)
	client := &http.Client{Timeout: 3 * time.Second}
	health, err := client.Get(origin + "/health")
	if err != nil {
		t.Fatal("browser API did not become healthy")
	}
	health.Body.Close()
	if health.StatusCode != http.StatusOK {
		t.Fatal("browser health failed")
	}
	select {
	case err := <-done:
		t.Fatalf("server exited before readiness: %v", err)
	default:
	}
	if err := reviewFixWriteNew(filepath.Join(directory, "ready.json"), b); err != nil {
		t.Fatal(err)
	}
	fmt.Println("REVIEW_FIX_BROWSER_READY " + string(b))
	deadline := time.After(5 * time.Minute)
	var serveErr error
	serveEnded := false
loop:
	for {
		select {
		case serveErr = <-done:
			serveEnded = true
			t.Errorf("server ended during browser session: %v", serveErr)
			break loop
		case <-deadline:
			t.Error("browser deadline exceeded")
			break loop
		case <-time.After(200 * time.Millisecond):
			if _, err := os.Stat(stop); err == nil {
				break loop
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	shutdownErr := server.Shutdown(ctx)
	if shutdownErr != nil {
		t.Error(shutdownErr)
	}
	if !serveEnded {
		select {
		case serveErr = <-done:
			serveEnded = true
		case <-ctx.Done():
			t.Error("server completion not observed")
		}
	}
	if serveErr != http.ErrServerClosed {
		t.Errorf("serve exit: %v", serveErr)
	}
	final := r121Fingerprint(t, f.caseID)
	finalEffects := r121EffectCounts(t)
	a, _ := json.Marshal(effects)
	z, _ := json.Marshal(finalEffects)
	if initial != final || string(a) != string(z) {
		t.Error("read-only presentation changed durable state")
	}
	receipt, _ := json.Marshal(map[string]any{"initialFingerprint": initial, "finalFingerprint": final, "initialEffects": effects, "finalEffects": finalEffects, "serverStopped": shutdownErr == nil && serveEnded && serveErr == http.ErrServerClosed})
	if err := reviewFixWriteNew(filepath.Join(directory, "backend_terminal.json"), receipt); err != nil {
		t.Error(err)
	}
}
