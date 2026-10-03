package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/jobdispatch"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-044: cqa_trigger_job executes through the shared jobdispatch service. Disposable
// MySQL and synthetic rows only. Configuration loading and Analyzer creation are injected before any
// real environment or provider access; the only provider is the synthetic test double below, so a
// completed run here is APPLICATION evidence, never CVF governance proof. Notifications: the engine's
// notification seam is unexported, so these jobs configure no outputs (output_schedule none) and the
// tests assert notification_logs stays empty and the default HTTP transport is never used.

// ---- hermetic dispatcher (every fixture) ----

// dispatchProbe is a jobdispatch.Service whose configuration load and worker start are observable
// and which starts no analysis. By default Start closes the reservation at once through the real
// Abort so a permission-matrix call leaves no owner behind; hold keeps the reservation owned.
type dispatchProbe struct {
	mu       sync.Mutex
	loads    int
	starts   int
	cfgErr   error
	nilCfg   bool
	startErr error
	hold     bool
	onLoad   func()
	params   []jobdispatch.Params
	held     []*engine.JobRunReservation
}

func (p *dispatchProbe) counts() (loads, starts int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.loads, p.starts
}

func (p *dispatchProbe) load() (*config.Config, error) {
	p.mu.Lock()
	p.loads++
	err, nilCfg, hook := p.cfgErr, p.nilCfg, p.onLoad
	p.mu.Unlock()
	if hook != nil {
		hook()
	}
	if err != nil {
		return nil, err
	}
	if nilCfg {
		return nil, nil
	}
	return &config.Config{Env: "test"}, nil
}

func (p *dispatchProbe) start(job models.Job, _ *config.Config, params jobdispatch.Params, res *engine.JobRunReservation) error {
	p.mu.Lock()
	p.starts++
	p.params = append(p.params, params)
	err, hold := p.startErr, p.hold
	if hold {
		p.held = append(p.held, res)
	}
	p.mu.Unlock()
	if err != nil {
		return err
	}
	if !hold {
		_ = res.Abort(job, "hermetic fixture: no worker is started")
	}
	return nil
}

func installHermeticDispatcher(t *testing.T) *dispatchProbe {
	t.Helper()
	p := &dispatchProbe{}
	orig := jobdispatch.Default
	jobdispatch.Default = &jobdispatch.Service{Label: "hermetic mcp", Timeout: time.Minute, LoadConfig: p.load, Start: p.start}
	t.Cleanup(func() {
		jobdispatch.Default = orig
		p.mu.Lock()
		held := p.held
		p.mu.Unlock()
		for _, res := range held {
			_ = res.Abort(models.Job{}, "test cleanup")
		}
	})
	return p
}

// ---- synthetic provider and real-worker environment ----

const passContent = `{"verdict":"PASS","score":95,"review":"Tot.","summary":"Dat.","violations":[]}`

// synthProvider is a synthetic AI double. gate parks the call (or the context ends it); mode selects
// the outcome after the gate: "" pass, "error" returns an error, "panic" panics.
type synthProvider struct {
	mu       sync.Mutex
	calls    int
	entered  chan struct{}
	gate     chan struct{}
	mode     string
	ctxAfter error
}

func newSynthProvider(gated bool) *synthProvider {
	p := &synthProvider{entered: make(chan struct{}, 16)}
	if gated {
		p.gate = make(chan struct{})
	}
	return p
}

func (p *synthProvider) AnalyzeChat(ctx context.Context, _ string, _ string) (ai.AIResponse, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	p.entered <- struct{}{}
	if p.gate != nil {
		select {
		case <-p.gate:
		case <-ctx.Done():
			return ai.AIResponse{}, ctx.Err()
		}
	}
	p.mu.Lock()
	p.ctxAfter = ctx.Err()
	mode := p.mode
	p.mu.Unlock()
	switch mode {
	case "error":
		return ai.AIResponse{}, errors.New("synthetic provider failure")
	case "panic":
		panic("synthetic provider panic")
	}
	return ai.AIResponse{Content: passContent, Model: "test-double", Provider: "test-double"}, nil
}

func (p *synthProvider) AnalyzeChatBatch(context.Context, string, []ai.BatchItem) (ai.AIResponse, error) {
	return ai.AIResponse{}, context.Canceled
}

func (p *synthProvider) callCount() int { p.mu.Lock(); defer p.mu.Unlock(); return p.calls }

func (p *synthProvider) waitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-p.entered:
	case <-time.After(30 * time.Second):
		t.Fatal("the synthetic provider call never started")
	}
}

func (p *synthProvider) release() {
	if p.gate == nil {
		return
	}
	select {
	case <-p.gate:
	default:
		close(p.gate)
	}
}

// realEnv routes the shared service through the real Analyzer with the synthetic provider and
// records the worker parameters. Start is the real StartWorker.
type realEnv struct {
	f      *mcpFixture
	prov   *synthProvider
	mu     sync.Mutex
	params []jobdispatch.Params
	starts int
}

func (e *realEnv) startCount() int { e.mu.Lock(); defer e.mu.Unlock(); return e.starts }

func (f *mcpFixture) jobID() string { return "job-" + f.tenantA }

// useRealWorker makes tenant A's job runnable (one channel conversation from the fixture seed),
// installs the real shared worker with the synthetic provider and registers the joins/cleanups so a
// worker never outlives the fixture rows.
func (f *mcpFixture) useRealWorker(t *testing.T, prov *synthProvider) *realEnv {
	t.Helper()
	f.exec(t, `UPDATE jobs SET input_channel_ids = ?, rules_content = 'Phan hoi dung han.', output_schedule = 'none' WHERE id = ?`, `["ch-`+f.tenantA+`"]`, f.jobID())
	f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at) VALUES (?, ?, 'ai_batch_mode', 'false', NOW(), NOW())`, pkg.NewUUID(), f.tenantA)
	t.Cleanup(func() {
		for _, table := range []string{"analysis_snapshots", "ai_usage_logs", "app_settings", "activity_logs"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", f.tenantA)
		}
	})
	env := &realEnv{f: f, prov: prov}
	orig := jobdispatch.Default
	jobdispatch.Default = &jobdispatch.Service{
		Label: "mcp trigger", Timeout: time.Minute,
		LoadConfig: func() (*config.Config, error) { return &config.Config{Env: "test"}, nil },
		Start: func(job models.Job, cfg *config.Config, p jobdispatch.Params, res *engine.JobRunReservation) error {
			env.mu.Lock()
			env.starts++
			env.params = append(env.params, p)
			env.mu.Unlock()
			return jobdispatch.StartWorker("mcp trigger", func(c *config.Config) *engine.Analyzer {
				return engine.NewAnalyzerWithProvider(c, prov)
			}, job, cfg, p, res)
		},
	}
	t.Cleanup(func() {
		prov.release() // never leave a worker parked on the barrier
		f.waitIdle(t, f.tenantA, f.jobID())
		jobdispatch.Default = orig
	})
	return env
}

func (f *mcpFixture) waitIdle(t *testing.T, tenant, job string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for engine.JobRunActive(tenant, job) {
		if time.Now().After(deadline) {
			t.Fatal("the ownership slot was never released")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (f *mcpFixture) waitTerminal(t *testing.T, runID string) models.JobRun {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		var r models.JobRun
		err := db.DB.First(&r, "id = ?", runID).Error
		if err == nil && r.Status != "running" {
			return r
		}
		if time.Now().After(deadline) {
			t.Fatalf("run %s never became terminal: %v %+v", runID, err, r)
		}
		time.Sleep(30 * time.Millisecond)
	}
}

func (f *mcpFixture) runRow(t *testing.T, runID string) models.JobRun {
	t.Helper()
	var r models.JobRun
	if err := db.DB.First(&r, "id = ?", runID).Error; err != nil {
		t.Fatalf("run %s: %v", runID, err)
	}
	return r
}

// mountedCall posts one JSON-RPC body to the mounted /mcp route with the given request context.
func (f *mcpFixture) mountedCall(t *testing.T, ctx context.Context, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	r := gin.New()
	SetupMCPRoutes(r)
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec, resp
}

type triggerAccepted struct {
	Message string `json:"message"`
	RunID   string `json:"run_id"`
}

// decodeAccepted requires a non-error ToolResult whose single text item is JSON with exactly the
// message and run_id fields.
func decodeAccepted(t *testing.T, label string, resp map[string]interface{}) triggerAccepted {
	t.Helper()
	if resp["error"] != nil {
		t.Fatalf("%s: JSON-RPC error %v", label, resp["error"])
	}
	text, isErr := resultText(t, resp)
	if isErr {
		t.Fatalf("%s: tool error %q", label, text)
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(text)))
	dec.DisallowUnknownFields()
	var got triggerAccepted
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("%s: result %q is not exactly {message, run_id}: %v", label, text, err)
	}
	var raw map[string]interface{}
	_ = json.Unmarshal([]byte(text), &raw)
	if got.Message != "job_triggered" || got.RunID == "" || len(raw) != 2 {
		t.Fatalf("%s: result %q", label, text)
	}
	return got
}

func (f *mcpFixture) triggerBody() string {
	return callBody("cqa_trigger_job", map[string]interface{}{"tenant_id": f.tenantA, "job_id": f.jobID()})
}

func (f *mcpFixture) admit(t *testing.T) { f.setMember(t, "member", permsOnly(toolMatrix[9].needs)) }

func (f *mcpFixture) countRows(t *testing.T, sql string, args ...interface{}) int64 {
	t.Helper()
	var n int64
	if err := db.DB.Raw(sql, args...).Scan(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// JE-03/JE-10/JE-07: a mounted authenticated call returns the persisted reservation's identity, the
// real shared worker runs the real Analyzer once with the default parameters, and the persisted
// results carry that same run id. Ownership is held until the worker is done.
func TestMountedTriggerRunsSharedWorkerWithPersistedIdentity(t *testing.T) {
	f := newMCPFixture(t)
	f.admit(t)
	prov := newSynthProvider(true)
	env := f.useRealWorker(t, prov)
	probe := &effectProbe{}
	probe.install(t, db.DB)

	rec, resp := f.mountedCall(t, context.Background(), f.triggerBody())
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	acc := decodeAccepted(t, "mounted", resp)
	row := f.runRow(t, acc.RunID)
	if row.Status != "running" || row.JobID != f.jobID() || row.TenantID != f.tenantA {
		t.Fatalf("persisted reservation %+v: the returned run_id must be the stored running row", row)
	}
	if !engine.JobRunActive(f.tenantA, f.jobID()) || f.countRows(t, `SELECT COUNT(*) FROM job_runs WHERE job_id = ?`, f.jobID()) != 1 {
		t.Fatal("202-equivalent acceptance must already hold the owner and exactly one row")
	}
	prov.waitEntered(t) // the worker really reached the synthetic provider
	prov.release()
	done := f.waitTerminal(t, acc.RunID)
	f.waitIdle(t, f.tenantA, f.jobID())
	if done.Status != "success" {
		t.Fatalf("run %+v", done)
	}
	if env.startCount() != 1 || len(env.params) != 1 || env.params[0] != (jobdispatch.Params{Mode: "since_last"}) {
		t.Fatalf("worker starts %d params %+v, want one since_last/0/empty-dates start", env.startCount(), env.params)
	}
	if prov.callCount() != 1 || f.countRows(t, `SELECT COUNT(*) FROM job_runs WHERE job_id = ?`, f.jobID()) != 1 {
		t.Fatalf("provider calls %d: RunReserved must consume the one reservation once", prov.callCount())
	}
	mine := f.countRows(t, `SELECT COUNT(*) FROM job_results WHERE job_run_id = ? AND result_type = 'conversation_evaluation'`, acc.RunID)
	foreign := f.countRows(t, `SELECT COUNT(*) FROM job_results WHERE tenant_id = ? AND job_run_id = ? AND result_type = 'conversation_evaluation'`, f.tenantB, acc.RunID)
	if mine != 1 || foreign != 0 {
		t.Fatalf("results for the accepted run id: %d own, %d foreign", mine, foreign)
	}
	if f.countRows(t, `SELECT COUNT(*) FROM notification_logs WHERE tenant_id = ? AND job_id = ?`, f.tenantA, f.jobID()) != 1 { // the fixture's own seeded log only
		t.Fatalf("a notification log was written by the run")
	}
	probe.mu.Lock()
	outbound := probe.http
	probe.mu.Unlock()
	if outbound != 0 {
		t.Fatalf("%d default-transport requests during the run (observation limit: other transports/raw sockets are not seen)", outbound)
	}
}

// JE-07: the accepted worker does not depend on the request. The request context is cancelled as
// soon as the response is written; the provider still sees a live context and the run completes.
func TestMountedTriggerWorkerOutlivesRequestContext(t *testing.T) {
	f := newMCPFixture(t)
	f.admit(t)
	prov := newSynthProvider(true)
	f.useRealWorker(t, prov)

	ctx, cancel := context.WithCancel(context.Background())
	_, resp := f.mountedCall(t, ctx, f.triggerBody())
	acc := decodeAccepted(t, "mounted", resp)
	prov.waitEntered(t)
	cancel() // the request is over; the worker is parked in the provider
	time.Sleep(100 * time.Millisecond)
	if f.runRow(t, acc.RunID).Status != "running" || !engine.JobRunActive(f.tenantA, f.jobID()) {
		t.Fatal("cancelling the request context must not stop or release the accepted run")
	}
	prov.release()
	done := f.waitTerminal(t, acc.RunID)
	f.waitIdle(t, f.tenantA, f.jobID())
	prov.mu.Lock()
	after := prov.ctxAfter
	prov.mu.Unlock()
	if done.Status != "success" || after != nil {
		t.Fatalf("run %s (%s), provider context error after the request ended: %v", done.Status, done.ErrorMessage, after)
	}
}

// JE-07: provider error, panic and an owner cancellation each end the run in a terminal state,
// release ownership and leave exactly one row; a panic value is never logged or stored.
func TestMountedTriggerWorkerFailureModes(t *testing.T) {
	for _, mode := range []string{"error", "panic", "cancel"} {
		mode := mode
		t.Run(mode, func(t *testing.T) {
			f := newMCPFixture(t)
			f.admit(t)
			prov := newSynthProvider(mode == "cancel")
			if mode != "cancel" {
				prov.mode = mode
			}
			f.useRealWorker(t, prov)
			_, resp := f.mountedCall(t, context.Background(), f.triggerBody())
			acc := decodeAccepted(t, mode, resp)
			prov.waitEntered(t)
			if mode == "cancel" {
				got, err := engine.CancelJobRun(f.tenantA, f.jobID(), acc.RunID)
				if err != nil || got != acc.RunID {
					t.Fatalf("cancel: %q %v", got, err)
				}
			}
			done := f.waitTerminal(t, acc.RunID)
			f.waitIdle(t, f.tenantA, f.jobID())
			t.Logf("%s: terminal status %q message %q", mode, done.Status, done.ErrorMessage)
			if done.FinishedAt == nil || f.countRows(t, `SELECT COUNT(*) FROM job_runs WHERE job_id = ?`, f.jobID()) != 1 {
				t.Fatalf("%s: run %+v", mode, done)
			}
			if mode == "cancel" && done.Status != "cancelled" {
				t.Fatalf("cancelled run closed as %q", done.Status)
			}
			if mode != "cancel" && done.Status == "success" {
				t.Fatalf("%s: a failing provider produced a success run", mode)
			}
			if strings.Contains(done.ErrorMessage, "synthetic provider panic") {
				t.Fatalf("panic value stored: %q", done.ErrorMessage)
			}
			// the slot is reusable once the failed run is over
			f2 := f.mountedCallStatus(t, f.triggerBody())
			if f2 != "job_triggered" && f2 != "job_already_running" {
				t.Fatalf("follow-up call: %q", f2)
			}
		})
	}
}

// mountedCallStatus returns the message of an accepted call or the error text of a rejected one.
func (f *mcpFixture) mountedCallStatus(t *testing.T, body string) string {
	t.Helper()
	_, resp := f.mountedCall(t, context.Background(), body)
	text, isErr := resultText(t, resp)
	if isErr {
		return text
	}
	var got triggerAccepted
	_ = json.Unmarshal([]byte(text), &got)
	return got.Message
}

// JE-07: ownership stays held through the completion tail. The terminal row is already stored while
// the worker is parked in its last activity write; a second mounted call is still refused as busy.
func TestMountedTriggerOwnershipHeldThroughCompletionTail(t *testing.T) {
	f := newMCPFixture(t)
	f.admit(t)
	prov := newSynthProvider(false)
	f.useRealWorker(t, prov)

	parked := make(chan struct{}, 1)
	release := make(chan struct{})
	var once sync.Once
	const cb = "r044:tail"
	if err := db.DB.Callback().Create().Before("gorm:create").Register(cb, func(tx *gorm.DB) {
		if a, ok := tx.Statement.Dest.(*models.ActivityLog); ok && a.Action == "job.run.completed" && a.TenantID == f.tenantA {
			once.Do(func() {
				parked <- struct{}{}
				<-release
			})
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		db.DB.Callback().Create().Remove(cb)
	})

	_, resp := f.mountedCall(t, context.Background(), f.triggerBody())
	acc := decodeAccepted(t, "first", resp)
	select {
	case <-parked:
	case <-time.After(30 * time.Second):
		t.Fatal("the worker never reached its completion tail")
	}
	row := f.runRow(t, acc.RunID)
	if row.Status == "running" || !engine.JobRunActive(f.tenantA, f.jobID()) {
		t.Fatalf("at the tail the run is %q (must be terminal) and owner active=%v (must be held)", row.Status, engine.JobRunActive(f.tenantA, f.jobID()))
	}
	runsBefore := f.countRows(t, `SELECT COUNT(*) FROM job_runs WHERE job_id = ?`, f.jobID())
	if got := f.mountedCallStatus(t, f.triggerBody()); got != "job_already_running" {
		t.Fatalf("a second call during the tail got %q, want job_already_running", got)
	}
	if f.countRows(t, `SELECT COUNT(*) FROM job_runs WHERE job_id = ?`, f.jobID()) != runsBefore {
		t.Fatal("the refused call created a row")
	}
	close(release)
	f.waitIdle(t, f.tenantA, f.jobID())
	if got := f.mountedCallStatus(t, f.triggerBody()); got != "job_triggered" {
		t.Fatalf("after the last effect the slot must be free, got %q", got)
	}
}

// JE-02: after authorization and the own-tenant lookup, any argument other than tenant_id/job_id is
// rejected before configuration, reservation or start. Authorization and "Job not found" win over it.
func TestTriggerExtraArgumentsAreRejectedBeforeDispatch(t *testing.T) {
	f := newMCPFixture(t)
	f.admit(t)
	base := func() map[string]interface{} {
		return map[string]interface{}{"tenant_id": f.tenantA, "job_id": f.jobID()}
	}
	for _, extra := range []struct {
		name string
		val  interface{}
	}{
		{"mode", "unanalyzed"}, {"mode", "conditional"}, {"mode", ""}, {"full", "true"}, {"full", true}, {"from", "2026-10-01"},
		{"to", "2026-10-02"}, {"limit", "5"}, {"limit", 5}, {"limit", nil}, {"run_id", "x"}, {"unknown_option", "1"},
	} {
		args := base()
		args[extra.name] = extra.val
		got := f.direct(t, f.userID, "cqa_trigger_job", args)
		if got.rpcErr != nil || !got.isErr || got.text != "invalid_run_parameters" {
			t.Fatalf("extra %s=%v: got isErr=%v rpcErr=%v %q", extra.name, extra.val, got.isErr, got.rpcErr, got.text)
		}
		if loads, starts := f.disp.counts(); loads != 0 || starts != 0 {
			t.Fatalf("extra %s: config loads %d starts %d before rejection", extra.name, loads, starts)
		}
		if n := f.jobRuns(t); n != 0 {
			t.Fatalf("extra %s: %d runs reserved", extra.name, n)
		}
	}
	// Authorization still comes first: a denied caller never learns whether the arguments were valid.
	f.setMember(t, "member", permsExcept("jobs", "w"))
	args := base()
	args["mode"] = "full"
	if got := f.direct(t, f.userID, "cqa_trigger_job", args); got.text != "permission denied" {
		t.Fatalf("denied call with extra args got %q", got.text)
	}
	// A job that is not in the tenant is the generic not-found, never the validation error.
	f.admit(t)
	args = map[string]interface{}{"tenant_id": f.tenantA, "job_id": "job-" + f.tenantB, "limit": "3"}
	if got := f.direct(t, f.userID, "cqa_trigger_job", args); got.text != "Job not found" {
		t.Fatalf("foreign job with extra args got %q", got.text)
	}
	if loads, starts := f.disp.counts(); loads != 0 || starts != 0 || f.jobRuns(t) != 0 {
		t.Fatalf("loads %d starts %d runs %d after the rejections", loads, starts, f.jobRuns(t))
	}
	// The default-only call is the admitted contract and carries the default parameters unchanged.
	got := f.direct(t, f.userID, "cqa_trigger_job", base())
	if got.isErr || got.rpcErr != nil {
		t.Fatalf("plain call: %+v", got)
	}
	f.disp.mu.Lock()
	params := f.disp.params
	f.disp.mu.Unlock()
	if len(params) != 1 || params[0] != (jobdispatch.Params{Mode: "since_last"}) {
		t.Fatalf("dispatch parameters %+v, want exactly since_last/0/empty dates", params)
	}
}

// JE-04: every dispatch failure is a ToolResult error with a fixed generic text, never a success,
// never a run_id and never a JSON-RPC protocol error; no configuration or driver detail escapes.
func TestTriggerDispatchFailuresAreGenericToolErrors(t *testing.T) {
	f := newMCPFixture(t)
	f.admit(t)
	const secret = "SECRET-CONFIG-AND-DRIVER-DETAIL" // cvf-allow-secret-fixture: synthetic leak marker
	check := func(label, want string, wantLoads, wantStarts int, wantRuns int64) {
		t.Helper()
		_, resp := f.rpc(t, f.token, f.triggerBody())
		if resp["error"] != nil {
			t.Fatalf("%s: JSON-RPC protocol error %v", label, resp["error"])
		}
		text, isErr := resultText(t, resp)
		if !isErr || text != want {
			t.Fatalf("%s: got %q isErr=%v, want tool error %q", label, text, isErr, want)
		}
		for _, bad := range []string{"run_id", "job_triggered", secret, "SELECT", f.tenantA, f.jobName(f.tenantA)} {
			if strings.Contains(text, bad) {
				t.Fatalf("%s: response carries %q", label, bad)
			}
		}
		if loads, starts := f.disp.counts(); loads != wantLoads || starts != wantStarts {
			t.Fatalf("%s: config loads %d starts %d, want %d and %d", label, loads, starts, wantLoads, wantStarts)
		}
		if n := f.jobRuns(t); n != wantRuns {
			t.Fatalf("%s: %d job runs, want %d", label, n, wantRuns)
		}
		f.disp.mu.Lock()
		f.disp.loads, f.disp.starts = 0, 0
		f.disp.mu.Unlock()
	}

	f.disp.cfgErr = errors.New("ENCRYPTION_KEY=" + secret)
	check("config error", "job_start_failed", 1, 0, 0) // refused before any reservation
	f.disp.cfgErr, f.disp.nilCfg = nil, true
	check("nil config", "job_start_failed", 1, 0, 0)
	f.disp.nilCfg = false

	// busy: a stored running row without a local owner blocks (config was already validated)
	f.exec(t, `INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'running', '{}', NOW())`, pkg.NewUUID(), f.jobID(), f.tenantA)
	check("busy", "job_already_running", 1, 0, 1)
	f.exec(t, `DELETE FROM job_runs WHERE job_id = ?`, f.jobID())

	// the job vanishes between the lookup and the reservation
	f.disp.onLoad = func() { db.DB.Exec(`DELETE FROM jobs WHERE id = ?`, f.jobID()) }
	check("vanished job", "Job not found", 1, 0, 0)
	f.disp.onLoad = nil
	f.exec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, ?, 'qc_analysis', '[]', '', '[]', 'manual', true, '[]', NOW(), NOW())`, f.jobID(), f.tenantA, f.jobName(f.tenantA))

	// forced admission failure (job_runs reads fail inside the reservation transaction)
	const failName = "r044:mcp:admission"
	if err := db.DB.Callback().Query().Before("gorm:query").Register(failName, func(tx *gorm.DB) {
		if tx.Statement.Table == "job_runs" {
			tx.AddError(errors.New("injected " + secret))
		}
	}); err != nil {
		t.Fatal(err)
	}
	check("admission failure", "job_start_failed", 1, 0, 0)
	db.DB.Callback().Query().Remove(failName)

	// synchronous start failure: the reservation is aborted (closed as error) and nothing is accepted
	f.disp.startErr = errors.New("synthetic start failure " + secret)
	check("start failure", "job_start_failed", 1, 1, 1)
	f.disp.startErr = nil
	var status string
	db.DB.Raw(`SELECT status FROM job_runs WHERE job_id = ?`, f.jobID()).Scan(&status)
	if status != "error" || engine.JobRunActive(f.tenantA, f.jobID()) {
		t.Fatalf("aborted start left status %q owner %v", status, engine.JobRunActive(f.tenantA, f.jobID()))
	}

	// protocol behavior is unchanged: an unknown tool is still a JSON-RPC error
	_, resp := f.rpc(t, f.token, callBody("cqa_trigger_job2", map[string]interface{}{"tenant_id": f.tenantA}))
	if e, _ := resp["error"].(map[string]interface{}); e == nil || e["code"].(float64) != -32602 {
		t.Fatalf("unknown tool: %v", resp)
	}
}

// JE-06 through the transport: a failed abort leaves the stored running row blocking later MCP
// admission even though the local owner is gone; the row is not deleted to simulate cleanup.
func TestTriggerFailedAbortKeepsBlockingThroughMCP(t *testing.T) {
	f := newMCPFixture(t)
	f.admit(t)
	var runID string
	f.disp.startErr = errors.New("synthetic start failure")
	const failName = "r044:mcp:abort"
	orig := jobdispatch.Default
	svc := *orig
	inner := svc.Start
	svc.Start = func(job models.Job, cfg *config.Config, p jobdispatch.Params, res *engine.JobRunReservation) error {
		runID = res.RunID()
		if err := db.DB.Callback().Update().Before("gorm:update").Register(failName, func(tx *gorm.DB) {
			if tx.Statement.Table == "job_runs" {
				tx.AddError(errors.New("injected abort failure"))
			}
		}); err != nil {
			t.Fatal(err)
		}
		return inner(job, cfg, p, res)
	}
	jobdispatch.Default = &svc
	t.Cleanup(func() { jobdispatch.Default = orig; db.DB.Callback().Update().Remove(failName) })

	_, resp := f.rpc(t, f.token, f.triggerBody())
	if text, isErr := resultText(t, resp); !isErr || text != "job_start_failed" {
		t.Fatalf("got %q isErr=%v", text, isErr)
	}
	db.DB.Callback().Update().Remove(failName)
	if runID == "" || f.runRow(t, runID).Status != "running" {
		t.Fatalf("a failed abort must leave the stored row running (run %q)", runID)
	}
	if engine.JobRunActive(f.tenantA, f.jobID()) {
		t.Fatal("the local owner must be released after the failed abort")
	}
	f.disp.startErr = nil
	if got := f.mountedCallStatus(t, f.triggerBody()); got != "job_already_running" {
		t.Fatalf("later MCP admission got %q, want the stored row to block", got)
	}
	if f.countRows(t, `SELECT COUNT(*) FROM job_runs WHERE job_id = ?`, f.jobID()) != 1 {
		t.Fatal("the blocked call created a row or the stuck row was deleted")
	}
}

// JE-05: concurrent mounted calls for one tenant/job reserve and start exactly once; the others are
// busy. Different jobs and tenants stay independent while the first reservation is held. Finite
// barrier, no timing sleeps.
func TestTriggerConcurrentMountedCallsHaveOneOwner(t *testing.T) {
	f := newMCPFixture(t)
	f.admit(t)
	f.disp.hold = true
	// a second job in tenant A and a job in tenant C (the user has full rights in C)
	f.exec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, 'second', 'qc_analysis', '[]', '', '[]', 'manual', true, '[]', NOW(), NOW())`, "job2-"+f.tenantA, f.tenantA)
	f.exec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, 'third', 'qc_analysis', '[]', '', '[]', 'manual', true, '[]', NOW(), NOW())`, "job-"+f.tenantC, f.tenantC)

	const n = 6
	gate := make(chan struct{})
	out := make(chan string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			out <- f.mountedCallStatus(t, f.triggerBody())
		}()
	}
	close(gate)
	wg.Wait()
	close(out)
	ok, busy := 0, 0
	for s := range out {
		switch s {
		case "job_triggered":
			ok++
		case "job_already_running":
			busy++
		default:
			t.Fatalf("unexpected outcome %q", s)
		}
	}
	_, starts := f.disp.counts()
	if ok != 1 || busy != n-1 || starts != 1 || f.countRows(t, `SELECT COUNT(*) FROM job_runs WHERE job_id = ?`, f.jobID()) != 1 {
		t.Fatalf("ok %d busy %d starts %d, want exactly one winner", ok, busy, starts)
	}
	// independent jobs and tenants are not blocked by the held reservation
	if got := f.mountedCallStatus(t, callBody("cqa_trigger_job", map[string]interface{}{"tenant_id": f.tenantA, "job_id": "job2-" + f.tenantA})); got != "job_triggered" {
		t.Fatalf("second job of the tenant: %q", got)
	}
	if got := f.mountedCallStatus(t, callBody("cqa_trigger_job", map[string]interface{}{"tenant_id": f.tenantC, "job_id": "job-" + f.tenantC})); got != "job_triggered" {
		t.Fatalf("job of another tenant: %q", got)
	}
}

// Real-worker slot exclusion over MCP: while the accepted run is parked, a second call is busy and
// starts nothing (one worker, one provider call).
func TestMountedTriggerSecondCallWhileRunningIsBusy(t *testing.T) {
	f := newMCPFixture(t)
	f.admit(t)
	prov := newSynthProvider(true)
	env := f.useRealWorker(t, prov)
	_, resp := f.mountedCall(t, context.Background(), f.triggerBody())
	acc := decodeAccepted(t, "first", resp)
	prov.waitEntered(t)
	if got := f.mountedCallStatus(t, f.triggerBody()); got != "job_already_running" {
		t.Fatalf("second call got %q", got)
	}
	if env.startCount() != 1 || prov.callCount() != 1 || f.countRows(t, `SELECT COUNT(*) FROM job_runs WHERE job_id = ?`, f.jobID()) != 1 {
		t.Fatalf("starts %d provider calls %d: the busy call must start nothing", env.startCount(), prov.callCount())
	}
	prov.release()
	f.waitTerminal(t, acc.RunID)
}

// JE-09: forced configuration, admission and extra-argument rejections on the MOUNTED route write
// nothing (write callbacks and table checksums, fixture setup outside the observation), start no
// worker and use no default-transport HTTP. Positive control: the accepted call is seen by the same
// probes in TestTriggerRejectionsHaveNoWritesRunsDispatchOrOutboundRequests.
func TestTriggerForcedErrorsOnMountedRouteHaveNoEffects(t *testing.T) {
	f := newMCPFixture(t)
	f.admit(t)
	const secret = "SECRET-FORCED-ERROR-DETAIL" // cvf-allow-secret-fixture: synthetic leak marker
	const failName = "r044:mcp:effects"
	probe := &effectProbe{}
	probe.install(t, db.DB)
	before := checksums(t)

	post := func(label, body, want string) {
		t.Helper()
		_, resp := f.rpc(t, f.token, body)
		text, isErr := resultText(t, resp)
		if !isErr || text != want || strings.Contains(text, secret) {
			t.Fatalf("%s: got %q isErr=%v, want %q", label, text, isErr, want)
		}
	}
	f.disp.cfgErr = errors.New(secret)
	post("config error", f.triggerBody(), "job_start_failed")
	f.disp.cfgErr, f.disp.nilCfg = nil, true
	post("nil config", f.triggerBody(), "job_start_failed")
	f.disp.nilCfg = false
	if err := db.DB.Callback().Query().Before("gorm:query").Register(failName, func(tx *gorm.DB) {
		if tx.Statement.Table == "job_runs" {
			tx.AddError(errors.New(secret))
		}
	}); err != nil {
		t.Fatal(err)
	}
	post("admission error", f.triggerBody(), "job_start_failed")
	db.DB.Callback().Query().Remove(failName)
	post("extra argument", callBody("cqa_trigger_job", map[string]interface{}{"tenant_id": f.tenantA, "job_id": f.jobID(), "mode": "unanalyzed"}), "invalid_run_parameters")

	probe.mu.Lock()
	writes, outbound := probe.writes, probe.http
	probe.mu.Unlock()
	for table, n := range writes {
		if table != tokenTable {
			t.Fatalf("%d GORM writes on %s during forced-error trigger calls", n, table)
		}
	}
	if outbound != 0 {
		t.Fatalf("%d default-transport HTTP attempts", outbound)
	}
	if _, starts := f.disp.counts(); starts != 0 {
		t.Fatalf("%d worker starts", starts)
	}
	if loads, _ := f.disp.counts(); loads != 3 { // config error, nil config, then the admission error (config loads first)
		t.Fatalf("config loads %d, want 3", loads)
	}
	after := checksums(t)
	for table, sum := range before {
		if table == "user_tenants" {
			continue
		}
		if after[table] != sum {
			t.Fatalf("table %s changed during forced-error calls (%d -> %d)", table, sum, after[table])
		}
	}
	if n := f.jobRuns(t); n != 0 {
		t.Fatalf("%d job runs reserved", n)
	}
}
