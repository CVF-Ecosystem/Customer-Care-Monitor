package handlers_test

// R120 is a synthetic, read-only browser fixture. Its records are presentation
// examples, not provider output or evidence of an enacted human workflow.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

type r120Admission struct {
	TrancheID string `json:"trancheId"`
	Database  string `json:"database"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
}

func r120StrictDSN(t *testing.T) string {
	t.Helper()
	path := os.Getenv("R120_RESOURCE_ADMISSION")
	if filepath.Clean(path) != filepath.Join("docs", "reviews", "probes", "r120_root", "resource_admission.json") {
		t.Fatal("R120 exact resource admission path required")
	}
	data, err := os.ReadFile(path)
	if err != nil { t.Fatal("R120 resource admission missing") }
	var admission r120Admission
	if json.Unmarshal(data, &admission) != nil || admission.TrancheID != "CCMAI-RUNTIME-120" || admission.Database != "r120_fixture" || admission.Host != "127.0.0.1" || admission.Port < 1024 || admission.Port > 65535 {
		t.Fatal("R120 resource admission invalid")
	}
	dsn := os.Getenv("R120_MYSQL_DSN")
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil || parsed == nil || parsed.Net != "tcp" || parsed.Addr != net.JoinHostPort(admission.Host, strconv.Itoa(admission.Port)) || parsed.DBName != admission.Database || parsed.User != "root" || parsed.Passwd == "" {
		t.Fatal("R120 owned loopback DSN mismatch")
	}
	return dsn
}

func r120ValidatedStopFile(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil { t.Fatal("R120 project root unavailable") }
	stop := os.Getenv("R120_STOP_FILE")
	expected := filepath.Join(wd,".cvf","runtime","r120_ui","stop_requested")
	if filepath.Clean(stop) != filepath.Clean(expected) { t.Fatal("R120 exact stop file required") }
	parent, err := filepath.EvalSymlinks(filepath.Dir(stop))
	if err != nil || filepath.Clean(parent) != filepath.Clean(filepath.Dir(expected)) { t.Fatal("R120 stop directory is not the exact runtime directory") }
	if _, err := os.Lstat(stop); err == nil || !os.IsNotExist(err) { t.Fatal("R120 stop file already exists") }
	return stop
}

func r120SelectFields(source map[string]any, keys ...string) map[string]any {
	selected := map[string]any{}
	for _, key := range keys { if value, ok := source[key]; ok { selected[key] = value } }
	return selected
}

func r120SafeDetailCapture(body []byte) (map[string]any, error) {
	var detail map[string]any
	if err := json.Unmarshal(body,&detail); err != nil { return nil, err }
	caseRow, ok := detail["case"].(map[string]any)
	if !ok { return nil, fmt.Errorf("case missing") }
	proposals, ok := detail["proposals"].([]any)
	if !ok { return nil, fmt.Errorf("proposals missing") }
	events, ok := detail["human_events"].([]any)
	if !ok { return nil, fmt.Errorf("events missing") }
	safeProposals := make([]map[string]any,0,len(proposals))
	for _, item := range proposals {
		proposal, ok := item.(map[string]any); if !ok { return nil, fmt.Errorf("proposal invalid") }
		safeProposals = append(safeProposals,r120SelectFields(proposal,"id","revision","run_id","snapshot_id","snapshot_digest","explanation","next_action"))
	}
	safeEvents := make([]map[string]any,0,len(events))
	for _, item := range events {
		event, ok := item.(map[string]any); if !ok { return nil, fmt.Errorf("event invalid") }
		safeEvents = append(safeEvents,r120SelectFields(event,"id","case_version","action","reason","created_at"))
	}
	return map[string]any{
		"case":r120SelectFields(caseRow,"id","version","run_id","snapshot_id","snapshot_digest","active_proposal_id","disposition","work_state"),
		"active_proposal_id":detail["active_proposal_id"],
		"proposals":safeProposals,
		"human_events":safeEvents,
	},nil
}

type r120StatusWriter struct { http.ResponseWriter; code int }
func (w *r120StatusWriter) WriteHeader(code int) { w.code = code; w.ResponseWriter.WriteHeader(code) }
func (w *r120StatusWriter) Write(p []byte) (int,error) { if w.code == 0 { w.code = http.StatusOK }; return w.ResponseWriter.Write(p) }

func r120Guard(next http.Handler) http.Handler {
	var mu sync.Mutex
	authAttempts, authAllowed := 0, 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.EscapedPath()
		allowed := r.Method == http.MethodGet || r.Method == http.MethodHead
		if r.Method == http.MethodPost && (path == "/api/v1/auth/login" || path == "/api/v1/auth/logout") {
			mu.Lock()
			authAttempts++
			if authAllowed < 2 { authAllowed++; allowed = true }
			attempt, allowedCount := authAttempts, authAllowed
			mu.Unlock()
			fmt.Printf("R120_AUTH_ATTEMPT method=%s path=%s attempt=%d allowedCount=%d allowed=%t\n",r.Method,path,attempt,allowedCount,allowed)
		}
		writer := &r120StatusWriter{ResponseWriter: w}
		if !allowed { http.Error(writer, "R120 read-only fixture", http.StatusMethodNotAllowed) } else { next.ServeHTTP(writer,r) }
		fmt.Printf("R120_ROUTE method=%s path=%s allowed=%t status=%d\n", r.Method, path, allowed, writer.code)
	})
}

func r120GuardForCase(next http.Handler, tenantID, caseID string) http.Handler {
	guard := r120Guard(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		path := r.URL.EscapedPath()
		if r.Method == http.MethodGet && path == "/api/v1/tenants/"+tenantID+"/intervention-cases/"+caseID {
			recorder := httptest.NewRecorder()
			next.ServeHTTP(recorder,r)
			response := recorder.Result()
			for key,values := range response.Header { for _,value := range values { w.Header().Add(key,value) } }
			w.WriteHeader(response.StatusCode)
			body := recorder.Body.Bytes()
			_,_ = w.Write(body)
			if response.StatusCode == http.StatusOK {
				capture, err := r120SafeDetailCapture(body)
				if err != nil { fmt.Printf("R120_DETAIL_CAPTURE_INVALID\n") } else if encoded, err := json.Marshal(capture); err == nil { fmt.Printf("R120_DETAIL_CAPTURE %s\n",encoded) }
			}
			return
		}
		next.ServeHTTP(w,r)
	}))
	return guard
}

func r120Fingerprint(t *testing.T, caseID string) string {
	t.Helper()
	var row models.InterventionCase
	var proposals []models.InterventionProposal
	var events []models.InterventionHumanEvent
	if db.DB.Where("id = ?", caseID).Take(&row).Error != nil || db.DB.Where("case_id = ?", caseID).Order("revision ASC").Find(&proposals).Error != nil || db.DB.Where("case_id = ?", caseID).Order("case_version ASC").Find(&events).Error != nil { t.Fatal("R120 durable fingerprint query failed") }
	encoded, err := json.Marshal([]any{row,proposals,events})
	if err != nil { t.Fatal(err) }
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func r120EffectCounts(t *testing.T) map[string]int64 {
	t.Helper()
	modelsByName := map[string]any{"attempts":&models.InterventionAttempt{},"executions":&models.InterventionExecution{},"grants":&models.InterventionExecutionGrant{},"budgets":&models.InterventionBudget{},"receipts":&models.InterventionReceipt{},"runs":&models.JobRun{},"snapshots":&models.AnalysisSnapshot{},"humanEvents":&models.InterventionHumanEvent{}}
	result := map[string]int64{}
	for name,model := range modelsByName {
		var count int64
		if err := db.DB.Model(model).Count(&count).Error; err != nil { t.Fatal(err) }
		result[name] = count
	}
	return result
}

func TestR120BrowserFixture(t *testing.T) {
	if os.Getenv("R120_BROWSER_FIXTURE") != "1" { t.Skip("R120 fixture requires explicit opt-in") }
	stop := r120ValidatedStopFile(t)
	dsn := r120StrictDSN(t)
	if os.Getenv("R101_MYSQL_DSN") != "" { t.Fatal("R101 compatibility DSN must start empty") }
	t.Setenv("R101_MYSQL_DSN",dsn)
	f := m3FixtureStart(t)
	t.Setenv("R101_MYSQL_DSN","")
	if err := db.DB.Model(&models.Tenant{}).Where("id = ?",f.tenant).Update("name","Minh họa R120").Error; err != nil { t.Fatal(err) }
	for id,name := range map[string]string{f.owner:"Người duyệt minh họa",f.reader:"Người xem minh họa",f.member:"Nhân viên minh họa",f.assignee:"Người nhận minh họa"} {
		if err := db.DB.Model(&models.User{}).Where("id = ?",id).Update("name",name).Error; err != nil { t.Fatal(err) }
	}
	var current models.InterventionCase
	var original models.InterventionProposal
	if err := db.DB.Where("id = ?",f.caseID).Take(&current).Error; err != nil { t.Fatal(err) }
	if err := db.DB.Where("id = ?",f.proposal).Take(&original).Error; err != nil { t.Fatal(err) }
	oldRun := pkg.NewUUID()
	if err := db.DB.Model(&original).Update("run_id",oldRun).Error; err != nil { t.Fatal(err) }
	activeID, laterID := pkg.NewUUID(), pkg.NewUUID()
	now := time.Now().UTC().Truncate(time.Millisecond)
	active := original
	active.ID, active.Revision, active.RunID = activeID, 2, current.RunID
	active.PromptVersion = "r101-direct-seed"
	active.Explanation = "Minh họa: khách đang chờ phản hồi về đơn hàng."
	active.NextAction = "Kiểm tra đơn hàng và phản hồi khách."
	active.Provider, active.Model = "synthetic-seed-not-provider", "none"
	active.CreatedAt = now.Add(time.Second)
	later := active
	later.ID, later.Revision, later.RunID = laterID, 3, pkg.NewUUID()
	later.Explanation = "Minh họa lịch sử: đề xuất khác, không phải đề xuất hiện tại."
	later.NextAction = "Không sử dụng nội dung lịch sử này."
	later.CreatedAt = now.Add(2*time.Second)
	if err := db.DB.Create(&active).Error; err != nil { t.Fatal(err) }
	if err := db.DB.Create(&later).Error; err != nil { t.Fatal(err) }
	if err := db.DB.Model(&current).Updates(map[string]any{"active_proposal_id":activeID,"version":4,"work_state":"OPEN","disposition":"PENDING_REVIEW"}).Error; err != nil { t.Fatal(err) }
	actions := []string{"OBSERVE_DATA","REQUEST_RERUN","EXECUTE_RERUN"}
	reasons := []string{"Minh họa: đã kiểm tra dữ liệu.","Minh họa: đã yêu cầu xem lại.","approved execution"}
	for i, action := range actions {
		before,_ := json.Marshal(map[string]any{"work_state":"WAITING_ON_DATA","disposition":"WAIT_DATA","human_next_action":fmt.Sprintf("Bước minh họa %d",i)})
		after,_ := json.Marshal(map[string]any{"work_state":"OPEN","disposition":"PENDING_REVIEW","human_next_action":fmt.Sprintf("Bước minh họa %d",i+1)})
		event := &models.InterventionHumanEvent{ID:pkg.NewUUID(),TenantID:f.tenant,CaseID:f.caseID,CaseVersion:i+2,ActorID:f.owner,Action:action,Reason:reasons[i],ProposalID:&activeID,SnapshotID:f.snapshot,SnapshotDigest:current.SnapshotDigest,SourceStatus:"bound_currentness_unverified",Before:string(before),After:string(after),CreatedAt:now.Add(time.Duration(i+3)*time.Second)}
		if err := db.DB.Create(event).Error; err != nil { t.Fatal(err) }
	}
	initial := r120Fingerprint(t,f.caseID)
	initialEffects := r120EffectCounts(t)
	listener, err := net.Listen("tcp","127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	server := &http.Server{Handler:r120GuardForCase(f.router,f.tenant,f.caseID)}
	defer server.Close()
	go func(){ _ = server.Serve(listener) }()
	manifest := map[string]any{"tranche":"CCMAI-RUNTIME-120","synthetic":true,"tenant":f.tenant,"case":f.caseID,"reader":f.reader,"ownerEmail":f.owner+"@example.invalid","activeProposalId":activeID,"historicalProposalId":f.proposal,"laterOffLineageProposalId":laterID,"activeExplanation":active.Explanation,"laterExplanation":later.Explanation,"eventActions":actions,"initialFingerprint":initial,"initialEffectCounts":initialEffects}
	encoded,_ := json.Marshal(manifest)
	fmt.Printf("R120_BROWSER_READY http://127.0.0.1:%d %s\n",listener.Addr().(*net.TCPAddr).Port,encoded)
	deadline := time.After(20*time.Minute)
	for {
		if _,err := os.Stat(stop); err == nil { break }
		select { case <-deadline: t.Error("R120 stop deadline"); goto stopped; case <-time.After(250*time.Millisecond): }
	}
stopped:
	_ = server.Close()
	_ = listener.Close()
	final := r120Fingerprint(t,f.caseID)
	finalEffects := r120EffectCounts(t)
	fmt.Printf("R120_BROWSER_FINAL initial=%s final=%s equal=%t\n",initial,final,strings.EqualFold(initial,final))
	counts,_ := json.Marshal(map[string]any{"initial":initialEffects,"final":finalEffects})
	fmt.Printf("R120_EFFECT_COUNTS %s\n",counts)
	if final != initial { t.Fatal("R120 case/proposal/event data changed") }
	for key,expected := range initialEffects { if finalEffects[key] != expected { t.Fatalf("R120 effect count changed: %s",key) } }
}
