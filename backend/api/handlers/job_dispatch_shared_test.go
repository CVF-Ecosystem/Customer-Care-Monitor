package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/jobdispatch"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/mcp"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-044: the HTTP trigger and the MCP cqa_trigger_job tool dispatch through ONE shared
// service on the same coordinator and database. Disposable MySQL and synthetic rows; configuration
// loading and Analyzer creation are injected (no real environment, provider or channel). Synthetic
// provider execution is application evidence only, never CVF governance proof.

type mcpSide struct {
	userID, token string
}

// newMCPSide adds an authenticated MCP principal (member with exactly jobs:w + messages:r) in the
// fixture tenant, with a valid bearer token, and removes it afterwards.
func newMCPSide(t *testing.T, tenantID string) *mcpSide {
	t.Helper()
	m := &mcpSide{userID: pkg.NewUUID(), token: "tok-" + pkg.NewUUID()}
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	exec(`INSERT INTO users (id, email, password_hash, name, is_admin, token_version, language, created_at, updated_at) VALUES (?, ?, 'x', 'R044', false, 0, 'vi', NOW(), NOW())`, m.userID, "r044-"+m.userID[:8]+"@example.invalid")
	exec(`INSERT INTO user_tenants (user_id, tenant_id, role, permissions) VALUES (?, ?, 'member', ?)`, m.userID, tenantID, `{"jobs":"w","messages":"r"}`)
	sum := sha256.Sum256([]byte(m.token))
	row := models.OAuthToken{ID: pkg.NewUUID(), ClientID: "r044-client", UserID: m.userID, AccessTokenHash: hex.EncodeToString(sum[:]),
		Scopes: "[]", ExpiresAt: time.Now().Add(time.Hour).UTC(), CreatedAt: time.Now().UTC()}
	if err := db.DB.Create(&row).Error; err != nil {
		t.Fatalf("fixture token: %v", err)
	}
	t.Cleanup(func() {
		db.DB.Where("user_id = ?", m.userID).Delete(&models.OAuthToken{})
		db.DB.Exec("DELETE FROM user_tenants WHERE user_id = ?", m.userID)
		db.DB.Exec("DELETE FROM users WHERE id = ?", m.userID)
	})
	return m
}

// call posts tools/call for cqa_trigger_job through the mounted /mcp route and returns the tool text.
func (m *mcpSide) call(t *testing.T, tenantID, jobID string) (text string, isErr bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp.SetupMCPRoutes(r)
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]interface{}{"name": "cqa_trigger_job", "arguments": map[string]interface{}{"tenant_id": tenantID, "job_id": jobID}},
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var resp struct {
		Result struct {
			Content []struct{ Text string } `json:"content"`
			IsError bool                    `json:"isError"`
		} `json:"result"`
		Error interface{} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || resp.Error != nil || len(resp.Result.Content) != 1 {
		t.Fatalf("MCP response %d %q: %v", rec.Code, rec.Body.String(), err)
	}
	return resp.Result.Content[0].Text, resp.Result.IsError
}

func mcpRunID(t *testing.T, text string) string {
	t.Helper()
	var got struct {
		Message string `json:"message"`
		RunID   string `json:"run_id"`
	}
	if err := json.Unmarshal([]byte(text), &got); err != nil || got.Message != "job_triggered" || got.RunID == "" {
		t.Fatalf("MCP result %q is not an accepted trigger: %v", text, err)
	}
	return got.RunID
}

// httpTrigger posts the HTTP trigger through a mounted gin route that carries the tenant like the
// real middleware does.
func httpTrigger(tenantID, jobID, query string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenant_id", tenantID) })
	r.POST("/api/v1/jobs/:jobId/trigger", TriggerJob)
	url := "/api/v1/jobs/" + jobID + "/trigger"
	if query != "" {
		url += "?" + query
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, url, nil))
	return rec
}

// useSharedWorkerFor routes the MCP side of the shared service through the real Analyzer with the
// fixture's synthetic provider, with injected configuration (no real config read).
func useSharedWorkerFor(t *testing.T, fx *ownFx) {
	t.Helper()
	orig := jobdispatch.Default
	jobdispatch.Default = &jobdispatch.Service{
		Label: "mcp trigger", Timeout: time.Minute,
		LoadConfig:  func() (*config.Config, error) { return &config.Config{Env: "test"}, nil },
		NewAnalyzer: func(cfg *config.Config) *engine.Analyzer { return engine.NewAnalyzerWithProvider(cfg, fx.prov) },
	}
	t.Cleanup(func() {
		fx.prov.release()
		fx.waitIdle(t) // join any shared worker before the fixture rows are deleted
		jobdispatch.Default = orig
	})
}

// JE-05/JE-10: one real Analyzer run holds the single slot for both transports. While the run
// started through MCP is parked in the synthetic provider, the HTTP trigger and a second MCP call
// are busy and start nothing; after completion the HTTP trigger is admitted and its run identity
// matches its persisted results. The reverse order is checked too.
func TestMixedTransportsShareOneOwnerWithRealAnalyzer(t *testing.T) {
	for _, first := range []string{"mcp-first", "http-first"} {
		first := first
		t.Run(first, func(t *testing.T) {
			fx := setupOwnership(t, true, 2)
			useSharedWorkerFor(t, fx)
			m := newMCPSide(t, fx.tenantID)

			mcpCall := func() (string, bool) { return m.call(t, fx.tenantID, fx.jobID) }
			var firstRun string
			if first == "mcp-first" {
				text, isErr := mcpCall()
				if isErr {
					t.Fatalf("MCP first call: %q", text)
				}
				firstRun = mcpRunID(t, text)
			} else {
				rec := httpTrigger(fx.tenantID, fx.jobID, "mode=unanalyzed")
				if rec.Code != http.StatusAccepted {
					t.Fatalf("HTTP first call: %d %s", rec.Code, rec.Body.String())
				}
				firstRun = bodyField(t, rec, "run_id")
			}
			fx.prov.waitEntered(t)
			calls, rows := fx.prov.callCount(), fx.jobRunCount(t)

			// both transports are refused while the first run holds the slot
			if rec := httpTrigger(fx.tenantID, fx.jobID, ""); rec.Code != http.StatusConflict || bodyField(t, rec, "error") != "job_already_running" {
				t.Fatalf("HTTP during the run: %d %s", rec.Code, rec.Body.String())
			}
			if text, isErr := mcpCall(); !isErr || text != "job_already_running" {
				t.Fatalf("MCP during the run: %q isErr=%v", text, isErr)
			}
			if fx.prov.callCount() != calls || fx.jobRunCount(t) != rows || rows != 1 {
				t.Fatalf("a busy request started work: provider calls %d->%d rows %d->%d", calls, fx.prov.callCount(), rows, fx.jobRunCount(t))
			}

			fx.prov.release()
			done := fx.waitRunDone(t, firstRun)
			fx.waitIdle(t)
			var mine int64
			db.DB.Model(&models.JobResult{}).Where("job_run_id = ? AND tenant_id = ?", firstRun, fx.tenantID).Count(&mine)
			if done.Status != "success" || mine == 0 {
				t.Fatalf("first run %+v with %d results under its own run id", done, mine)
			}

			// the other transport is admitted afterwards and gets its own persisted run
			var secondRun string
			if first == "mcp-first" {
				rec := httpTrigger(fx.tenantID, fx.jobID, "mode=unanalyzed")
				if rec.Code != http.StatusAccepted {
					t.Fatalf("HTTP after completion: %d %s", rec.Code, rec.Body.String())
				}
				secondRun = bodyField(t, rec, "run_id")
			} else {
				text, isErr := mcpCall()
				if isErr {
					t.Fatalf("MCP after completion: %q", text)
				}
				secondRun = mcpRunID(t, text)
			}
			second := fx.waitRunDone(t, secondRun)
			fx.waitIdle(t)
			if secondRun == firstRun || second.Status == "running" || fx.jobRunCount(t) != 2 {
				t.Fatalf("second run %+v (first %s), rows %d", second, firstRun, fx.jobRunCount(t))
			}
			var foreign int64
			db.DB.Model(&models.JobResult{}).Where("tenant_id = ? AND job_run_id NOT IN (?, ?)", fx.tenantID, firstRun, secondRun).Count(&foreign)
			if foreign != 0 {
				t.Fatalf("%d results carry neither run id", foreign)
			}
		})
	}
}

// JE-05: simultaneous MCP and HTTP requests for one tenant/job reserve and start exactly once; the
// others are busy. Finite barrier, repeated rounds, a mutex-safe parked launcher on both transports.
func TestMixedTransportsConcurrentAdmissionHasOneWinner(t *testing.T) {
	db.Close()
	fx := setupJobDispatchFixture(t)
	m := newMCPSide(t, fx.tenantID)

	var mu sync.Mutex
	var parked []*engine.JobRunReservation
	var httpStarts, mcpStarts int
	park := func(counter *int) func(models.Job, *config.Config, jobdispatch.Params, *engine.JobRunReservation) error {
		return func(_ models.Job, _ *config.Config, _ jobdispatch.Params, res *engine.JobRunReservation) error {
			mu.Lock()
			defer mu.Unlock()
			*counter++
			parked = append(parked, res)
			return nil
		}
	}
	mcpPark := park(&mcpStarts)
	origDefault := jobdispatch.Default
	jobdispatch.Default = &jobdispatch.Service{
		Label: "mcp trigger", Timeout: time.Minute,
		LoadConfig: func() (*config.Config, error) { return &config.Config{Env: "test"}, nil },
		Start:      mcpPark,
	}
	t.Cleanup(func() { jobdispatch.Default = origDefault })
	httpPark := park(&httpStarts)
	startTriggerJob = func(job models.Job, cfg *config.Config, p triggerJobParams, res *engine.JobRunReservation) error {
		return httpPark(job, cfg, jobdispatch.Params{Mode: p.mode, DateFrom: p.dateFrom, DateTo: p.dateTo, Limit: p.maxConv}, res)
	}

	const rounds, perTransport = 12, 4
	winners := map[string]int{}
	for round := 0; round < rounds; round++ {
		gate := make(chan struct{})
		type outcome struct {
			transport string
			ok, busy  bool
		}
		out := make(chan outcome, 2*perTransport)
		var wg sync.WaitGroup
		for i := 0; i < perTransport; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				<-gate
				text, isErr := m.call(t, fx.tenantID, fx.jobID)
				out <- outcome{"mcp", !isErr && strings.Contains(text, "job_triggered"), isErr && text == "job_already_running"}
			}()
			go func() {
				defer wg.Done()
				<-gate
				rec := httpTrigger(fx.tenantID, fx.jobID, "")
				out <- outcome{"http", rec.Code == http.StatusAccepted, rec.Code == http.StatusConflict}
			}()
		}
		close(gate)
		wg.Wait()
		close(out)
		ok, busy := 0, 0
		for o := range out {
			switch {
			case o.ok:
				ok++
				winners[o.transport]++
			case o.busy:
				busy++
			default:
				t.Fatalf("round %d: %s request was neither accepted nor busy", round, o.transport)
			}
		}
		mu.Lock()
		held := len(parked)
		mu.Unlock()
		// rows accumulate: each earlier round's aborted run stays as a closed error row
		if ok != 1 || busy != 2*perTransport-1 || held != 1 || fx.jobRunCount(t) != int64(round+1) {
			t.Fatalf("round %d: accepted %d busy %d parked %d rows %d, want exactly one new owner/row", round, ok, busy, held, fx.jobRunCount(t))
		}
		// free the slot for the next round through the real abort (closed as error), never by deleting
		mu.Lock()
		res := parked[0]
		parked = nil
		mu.Unlock()
		if err := res.Abort(models.Job{}, "round cleanup"); err != nil {
			t.Fatalf("round %d abort: %v", round, err)
		}
	}
	t.Logf("winners over %d rounds: %v (starts: mcp %d http %d)", rounds, winners, mcpStarts, httpStarts)
	if mcpStarts+httpStarts != rounds {
		t.Fatalf("starts %d, want exactly one per round", mcpStarts+httpStarts)
	}
}

// JE-05: a held reservation on one job does not block another job or tenant over either transport.
func TestMixedTransportsDifferentJobsAreIndependent(t *testing.T) {
	db.Close()
	fx := setupJobDispatchFixture(t)
	m := newMCPSide(t, fx.tenantID)
	job2 := "job2-" + fx.jobID
	if err := db.DB.Exec(`INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, 'second', 'qc_analysis', '[]', '', '[]', 'manual', true, '[]', NOW(), NOW())`, job2, fx.tenantID).Error; err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var parked []*engine.JobRunReservation
	origDefault := jobdispatch.Default
	jobdispatch.Default = &jobdispatch.Service{
		LoadConfig: func() (*config.Config, error) { return &config.Config{Env: "test"}, nil },
		Start: func(_ models.Job, _ *config.Config, _ jobdispatch.Params, res *engine.JobRunReservation) error {
			mu.Lock()
			parked = append(parked, res)
			mu.Unlock()
			return nil
		},
	}
	t.Cleanup(func() {
		jobdispatch.Default = origDefault
		mu.Lock()
		for _, res := range parked {
			_ = res.Abort(models.Job{}, "test cleanup")
		}
		mu.Unlock()
		db.DB.Exec("DELETE FROM job_runs WHERE job_id = ?", job2)
		db.DB.Exec("DELETE FROM jobs WHERE id = ?", job2)
	})

	if rec := httpTrigger(fx.tenantID, fx.jobID, ""); rec.Code != http.StatusAccepted {
		t.Fatalf("HTTP on job 1: %d %s", rec.Code, rec.Body.String())
	}
	if text, isErr := m.call(t, fx.tenantID, job2); isErr {
		t.Fatalf("MCP on job 2 while job 1 is held: %q", text)
	}
	if text, isErr := m.call(t, fx.tenantID, fx.jobID); !isErr || text != "job_already_running" {
		t.Fatalf("MCP on the held job 1: %q isErr=%v", text, isErr)
	}
	if rec := httpTrigger(fx.tenantID, job2, ""); rec.Code != http.StatusConflict {
		t.Fatalf("HTTP on the MCP-held job 2: %d %s", rec.Code, rec.Body.String())
	}
}

// Production wiring is the real configuration loader and Analyzer on both transports; the tests above
// replace them only through the injected seams. (Checked as function identity: no configuration is
// read and no provider is created.)
func TestProductionDispatchWiringUsesRealLoaderAndAnalyzer(t *testing.T) {
	same := func(a, b interface{}) bool { return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer() }
	if !same(loadJobDispatchConfig, config.Load) {
		t.Fatal("the HTTP trigger configuration loader is not config.Load")
	}
	if !same(newTriggerAnalyzer, engine.NewAnalyzer) {
		t.Fatal("the HTTP trigger Analyzer factory is not engine.NewAnalyzer")
	}
	d := jobdispatch.Default
	if d == nil || d.LoadConfig != nil || d.NewAnalyzer != nil || d.Start != nil {
		t.Fatalf("jobdispatch.Default must use the production defaults (config.Load, engine.NewAnalyzer, StartWorker): %+v", d)
	}
	svc := triggerDispatcher()
	if svc.Timeout != jobRunTimeout || svc.Timeout <= 0 || jobRunTimeout != jobdispatch.DefaultTimeout {
		t.Fatalf("the HTTP dispatcher timeout %v / jobRunTimeout %v must be the bounded default %v", svc.Timeout, jobRunTimeout, jobdispatch.DefaultTimeout)
	}
}

// rendezvous holds each of two callers at its configuration load until both transports are inside
// dispatch, so the reservation race is real for the pair (finite barrier: no timing sleeps).
type rendezvous struct {
	mu      sync.Mutex
	arrived int
	open    chan struct{}
}

func newRendezvous() *rendezvous { return &rendezvous{open: make(chan struct{})} }

func (r *rendezvous) arrive(t *testing.T) {
	r.mu.Lock()
	r.arrived++
	if r.arrived == 2 {
		close(r.open)
	}
	ch := r.open
	r.mu.Unlock()
	select {
	case <-ch:
	case <-time.After(30 * time.Second):
		t.Error("the other transport never reached dispatch")
	}
}

// JE-05: one MCP and one HTTP request are both inside dispatch (past authorization, lookup and
// configuration) before either reserves. Exactly one is accepted and the other is busy, every round;
// the winning transport is recorded because it depends on scheduling.
func TestMixedTransportsSimultaneousDispatchHasOneWinner(t *testing.T) {
	db.Close()
	fx := setupJobDispatchFixture(t)
	m := newMCPSide(t, fx.tenantID)

	var mu sync.Mutex
	var parked []*engine.JobRunReservation
	var current *rendezvous
	cfg := func() (*config.Config, error) {
		mu.Lock()
		r := current
		mu.Unlock()
		r.arrive(t)
		return &config.Config{Env: "test"}, nil
	}
	park := func(res *engine.JobRunReservation) error {
		mu.Lock()
		defer mu.Unlock()
		parked = append(parked, res)
		return nil
	}
	origDefault, origLoad := jobdispatch.Default, loadJobDispatchConfig
	jobdispatch.Default = &jobdispatch.Service{
		Label: "mcp trigger", Timeout: time.Minute, LoadConfig: cfg,
		Start: func(_ models.Job, _ *config.Config, _ jobdispatch.Params, res *engine.JobRunReservation) error {
			return park(res)
		},
	}
	loadJobDispatchConfig = cfg
	startTriggerJob = func(_ models.Job, _ *config.Config, _ triggerJobParams, res *engine.JobRunReservation) error {
		return park(res)
	}
	t.Cleanup(func() { jobdispatch.Default, loadJobDispatchConfig = origDefault, origLoad })

	winners := map[string]int{}
	const rounds = 30
	for round := 0; round < rounds; round++ {
		mu.Lock()
		current = newRendezvous()
		mu.Unlock()
		type outcome struct {
			transport string
			ok, busy  bool
		}
		out := make(chan outcome, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			text, isErr := m.call(t, fx.tenantID, fx.jobID)
			out <- outcome{"mcp", !isErr && strings.Contains(text, "job_triggered"), isErr && text == "job_already_running"}
		}()
		go func() {
			defer wg.Done()
			rec := httpTrigger(fx.tenantID, fx.jobID, "")
			out <- outcome{"http", rec.Code == http.StatusAccepted, rec.Code == http.StatusConflict}
		}()
		wg.Wait()
		close(out)
		ok, busy := 0, 0
		for o := range out {
			switch {
			case o.ok:
				ok++
				winners[o.transport]++
			case o.busy:
				busy++
			default:
				t.Fatalf("round %d: %s request was neither accepted nor busy", round, o.transport)
			}
		}
		mu.Lock()
		held := len(parked)
		var res *engine.JobRunReservation
		if held > 0 {
			res = parked[0]
		}
		parked = nil
		mu.Unlock()
		if ok != 1 || busy != 1 || held != 1 || fx.jobRunCount(t) != int64(round+1) {
			t.Fatalf("round %d: accepted %d busy %d parked %d rows %d, want one winner and one busy", round, ok, busy, held, fx.jobRunCount(t))
		}
		if err := res.Abort(models.Job{}, "round cleanup"); err != nil {
			t.Fatalf("round %d abort: %v", round, err)
		}
	}
	t.Logf("simultaneous-dispatch winners over %d rounds: %v", rounds, winners)
}

// ---- notification-tail exclusion across transports (R044-R1-01) ----

const (
	hNotifyToken = "R044_SYNTHETIC" // synthetic Telegram bot token: never a credential
	hNotifyChat  = "R044_LOCAL"
)

// hNotifyTransport replaces http.DefaultTransport with an in-process responder for exactly one
// synthetic Telegram destination. The matching send parks until open() or its request context ends;
// every other request is counted as foreign and fails. Nothing is delegated to a real transport.
type hNotifyTransport struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	relOnce sync.Once
	mu      sync.Mutex
	sends   int
	foreign int
}

func newHNotifyTransport() *hNotifyTransport {
	return &hNotifyTransport{entered: make(chan struct{}), release: make(chan struct{})}
}

func (n *hNotifyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var payload map[string]interface{}
	matches := req.Method == http.MethodPost && req.URL.Host == "api.telegram.org" &&
		req.URL.Path == "/bot"+hNotifyToken+"/sendMessage" && req.Body != nil &&
		json.NewDecoder(req.Body).Decode(&payload) == nil && payload["chat_id"] == hNotifyChat
	n.mu.Lock()
	if matches {
		n.sends++
	} else {
		n.foreign++
	}
	n.mu.Unlock()
	if !matches {
		return nil, errors.New("r044 notification trap: unexpected outbound request")
	}
	n.once.Do(func() { close(n.entered) })
	select {
	case <-n.release:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Request: req}, nil
}

func (n *hNotifyTransport) open() { n.relOnce.Do(func() { close(n.release) }) }

func (n *hNotifyTransport) counts() (sends, foreign int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.sends, n.foreign
}

// waitSend is the positive-send detector: true once a configured notification reached the transport.
func (n *hNotifyTransport) waitSend(d time.Duration) bool {
	select {
	case <-n.entered:
		return true
	case <-time.After(d):
		return false
	}
}

// JE-05/JE-07/JE-10: a run accepted through MCP with a configured (synthetic) notification is already
// terminal while its send is in flight but still owns the slot. The HTTP trigger and a second MCP call
// are both busy and reserve, start and call the provider nothing. After the send returns the log and
// notified marks are bound to the accepted run, the slot is released and HTTP is admitted.
func TestHTTPTriggerIsBusyWhileMCPHoldsTheNotificationTail(t *testing.T) {
	fx := setupOwnership(t, true, 1)
	tr := newHNotifyTransport()
	original := http.DefaultTransport
	http.DefaultTransport = tr
	t.Cleanup(func() { http.DefaultTransport = original }) // runs after the worker join below
	t.Cleanup(func() {
		db.DB.Exec("DELETE FROM notification_logs WHERE tenant_id = ?", fx.tenantID) // after the join, before the fixture rows
	})
	useSharedWorkerFor(t, fx)
	t.Cleanup(tr.open) // never leave a send parked when the join runs
	for _, q := range []struct {
		sql  string
		args []interface{}
	}{
		{`UPDATE jobs SET output_schedule = 'immediate', outputs = ? WHERE id = ?`, []interface{}{`[{"type":"telegram","bot_token":"` + hNotifyToken + `","chat_id":"` + hNotifyChat + `"}]`, fx.jobID}},
		{`INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at) VALUES (?, ?, 'app_url', 'http://r044.invalid', NOW(), NOW())`, []interface{}{pkg.NewUUID(), fx.tenantID}},
	} {
		if err := db.DB.Exec(q.sql, q.args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	m := newMCPSide(t, fx.tenantID)

	text, isErr := m.call(t, fx.tenantID, fx.jobID)
	if isErr {
		t.Fatalf("MCP call: %q", text)
	}
	runID := mcpRunID(t, text)
	fx.prov.waitEntered(t)
	fx.prov.release()
	if !tr.waitSend(30 * time.Second) {
		t.Fatal("the configured notification never reached the in-process transport")
	}
	if status := fx.runStatus(t, runID); status != "success" || !engine.JobRunActive(fx.tenantID, fx.jobID) {
		t.Fatalf("while the send is parked the accepted run must be terminal (%q) and the owner held (active=%v)", status, engine.JobRunActive(fx.tenantID, fx.jobID))
	}
	rows, calls := fx.jobRunCount(t), fx.prov.callCount()
	if rec := httpTrigger(fx.tenantID, fx.jobID, ""); rec.Code != http.StatusConflict || bodyField(t, rec, "error") != "job_already_running" {
		t.Fatalf("HTTP during the notification send: %d %s", rec.Code, rec.Body.String())
	}
	if text, isErr := m.call(t, fx.tenantID, fx.jobID); !isErr || text != "job_already_running" {
		t.Fatalf("MCP during the notification send: %q isErr=%v", text, isErr)
	}
	if fx.jobRunCount(t) != rows || rows != 1 || fx.prov.callCount() != calls || calls != 1 {
		t.Fatalf("a busy request changed state: runs %d->%d provider calls %d->%d (want 1/1)", rows, fx.jobRunCount(t), calls, fx.prov.callCount())
	}
	var logged, marked int64
	db.DB.Raw(`SELECT COUNT(*) FROM notification_logs WHERE job_run_id = ?`, runID).Scan(&logged)
	db.DB.Raw(`SELECT COUNT(*) FROM job_results WHERE job_run_id = ? AND notified_at IS NOT NULL`, runID).Scan(&marked)
	if logged != 0 || marked != 0 {
		t.Fatalf("a log (%d) or notified mark (%d) exists before the send returned", logged, marked)
	}

	tr.open()
	fx.waitIdle(t)
	var total int64
	db.DB.Raw(`SELECT COUNT(*) FROM notification_logs WHERE job_run_id = ? AND tenant_id = ? AND job_id = ? AND channel_type = 'telegram' AND recipient = ? AND status = 'sent'`, runID, fx.tenantID, fx.jobID, hNotifyChat).Scan(&logged)
	db.DB.Raw(`SELECT COUNT(*) FROM job_results WHERE job_run_id = ?`, runID).Scan(&total)
	db.DB.Raw(`SELECT COUNT(*) FROM job_results WHERE job_run_id = ? AND notified_at IS NOT NULL`, runID).Scan(&marked)
	if logged != 1 || total < 1 || marked != total {
		t.Fatalf("sent logs bound to the accepted run %d (want 1), notified results %d of %d", logged, marked, total)
	}
	if sends, foreign := tr.counts(); sends != 1 || foreign != 0 {
		t.Fatalf("transport saw %d matching sends and %d foreign requests, want 1 and 0", sends, foreign)
	}
	rec := httpTrigger(fx.tenantID, fx.jobID, "mode=unanalyzed")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("HTTP after the last effect: %d %s", rec.Code, rec.Body.String())
	}
	second := bodyField(t, rec, "run_id")
	fx.waitRunDone(t, second)
	fx.waitIdle(t)
	if _, foreign := tr.counts(); foreign != 0 || second == runID || fx.jobRunCount(t) != 2 {
		t.Fatalf("follow-up run %s (first %s): foreign requests %d, rows %d", second, runID, foreign, fx.jobRunCount(t))
	}
}
