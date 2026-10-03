package mcp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// CCMAI-RUNTIME-035 + CCMAI-RUNTIME-044: the unchanged authorization and own-tenant lookup in front
// of cqa_trigger_job. R035's "unavailable" success contract is superseded by R044 for ADMITTED
// success only: an admitted call now returns job_triggered with the persisted run_id (hermetic
// dispatcher here; real shared worker in trigger_execution_test.go). Denial, not-found and forced
// error paths keep their R035 no-effect checks and additionally prove no configuration load,
// reservation or worker start. Disposable MySQL and synthetic rows only. Observation limits: write
// callbacks, table checksums and a replaced http.DefaultTransport see GORM writes and
// default-transport HTTP; they do not observe raw sockets, other transports or goroutines that use
// other clients.

func (f *mcpFixture) jobName(tenant string) string { return "JOB-" + forbiddenMarker + "-" + tenant }

// assertAccepted checks an admitted trigger outcome: a non-error ToolResult whose text is exactly
// {message: job_triggered, run_id}, never the retired R035 unavailable text and no tenant/job data.
func (f *mcpFixture) assertAccepted(t *testing.T, label string, got toolOutcome) string {
	t.Helper()
	if got.rpcErr != nil || got.isErr {
		t.Fatalf("%s: want an accepted trigger, got isErr=%v rpcErr=%v text=%q", label, got.isErr, got.rpcErr, got.text)
	}
	if got.text == triggerUnavailableWant || strings.Contains(got.text, "unavailable") {
		t.Fatalf("%s: the retired unavailable contract is back: %q", label, got.text)
	}
	dec := json.NewDecoder(strings.NewReader(got.text))
	dec.DisallowUnknownFields()
	var acc struct {
		Message string `json:"message"`
		RunID   string `json:"run_id"`
	}
	if err := dec.Decode(&acc); err != nil || acc.Message != "job_triggered" || acc.RunID == "" {
		t.Fatalf("%s: result %q is not exactly {message: job_triggered, run_id}: %v", label, got.text, err)
	}
	for _, leak := range []string{f.jobName(f.tenantA), "job-" + f.tenantA, f.tenantA, f.tenantB, secretSQLMarker, forbiddenMarker} {
		if strings.Contains(got.text, leak) {
			t.Fatalf("%s: response disclosed %q", label, leak)
		}
	}
	return acc.RunID
}

// assertNoDispatch proves the hermetic dispatcher saw no configuration load or start since reset.
func (f *mcpFixture) assertNoDispatch(t *testing.T, label string) {
	t.Helper()
	if loads, starts := f.disp.counts(); loads != 0 || starts != 0 {
		t.Fatalf("%s: dispatcher saw %d config loads and %d starts; a rejected call must reach neither", label, loads, starts)
	}
}

// sqlObserver records the SQL text and bind variables of queries on one table.
type sqlObserver struct {
	mu   sync.Mutex
	rows []sqlSeen
}

type sqlSeen struct {
	sql  string
	vars []interface{}
}

func (o *sqlObserver) register(t *testing.T, gdb *gorm.DB, table string) {
	t.Helper()
	name := "r035:sql:" + table
	err := gdb.Callback().Query().After("gorm:query").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table != table {
			return
		}
		o.mu.Lock()
		defer o.mu.Unlock()
		o.rows = append(o.rows, sqlSeen{sql: tx.Statement.SQL.String(), vars: append([]interface{}{}, tx.Statement.Vars...)})
	})
	if err != nil {
		t.Fatalf("sql observer: %v", err)
	}
	t.Cleanup(func() { _ = gdb.Callback().Query().Remove(name) })
}

func (o *sqlObserver) reset() { o.mu.Lock(); o.rows = nil; o.mu.Unlock() }

func (o *sqlObserver) snapshot() []sqlSeen {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]sqlSeen{}, o.rows...)
}

// effectProbe counts GORM write callbacks, default-transport HTTP attempts and table checksums.
type effectProbe struct {
	mu     sync.Mutex
	writes map[string]int // GORM write callbacks by table
	http   int
}

type trapTransport struct{ p *effectProbe }

func (tt trapTransport) RoundTrip(*http.Request) (*http.Response, error) {
	tt.p.mu.Lock()
	tt.p.http++
	tt.p.mu.Unlock()
	return nil, errors.New("r035 outbound trap: no request may leave the process")
}

var effectTables = []string{"jobs", "job_runs", "job_results", "messages", "conversations", "channels", "notification_logs", "tenants", "user_tenants"}

func (p *effectProbe) install(t *testing.T, gdb *gorm.DB) {
	t.Helper()
	count := func(tx *gorm.DB) {
		p.mu.Lock()
		if p.writes == nil {
			p.writes = map[string]int{}
		}
		p.writes[tx.Statement.Table]++
		p.mu.Unlock()
	}
	cb := gdb.Callback()
	for name, reg := range map[string]func() error{
		"r035:w:create": func() error { return cb.Create().Before("gorm:create").Register("r035:w:create", count) },
		"r035:w:update": func() error { return cb.Update().Before("gorm:update").Register("r035:w:update", count) },
		"r035:w:delete": func() error { return cb.Delete().Before("gorm:delete").Register("r035:w:delete", count) },
	} {
		if err := reg(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	old, oldClient := http.DefaultTransport, http.DefaultClient.Transport
	http.DefaultTransport = trapTransport{p}
	http.DefaultClient.Transport = trapTransport{p}
	t.Cleanup(func() {
		http.DefaultTransport, http.DefaultClient.Transport = old, oldClient
		cb.Create().Remove("r035:w:create")
		cb.Update().Remove("r035:w:update")
		cb.Delete().Remove("r035:w:delete")
	})
}

func checksums(t *testing.T) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	for _, table := range effectTables {
		rows, err := db.DB.Raw("CHECKSUM TABLE " + table).Rows()
		if err != nil {
			t.Fatalf("checksum %s: %v", table, err)
		}
		for rows.Next() {
			var name string
			var sum sql.NullInt64
			if err := rows.Scan(&name, &sum); err != nil {
				rows.Close()
				t.Fatalf("checksum scan %s: %v", table, err)
			}
			out[table] = sum.Int64
		}
		rows.Close()
	}
	return out
}

// MT-01: every admitted principal gets the same accepted shape through the direct and the mounted
// path; each call reserves its own run, so run ids are distinct and persisted.
func TestTriggerAdmittedDirectAndMounted(t *testing.T) {
	f := newMCPFixture(t)
	trigger := toolMatrix[9]
	if trigger.name != "cqa_trigger_job" {
		t.Fatalf("matrix order changed")
	}
	body := callBody(trigger.name, trigger.args(f, f.tenantA))
	seen := map[string]bool{}
	for _, step := range []struct{ label, role, perms string }{
		{"member with exactly jobs:w+messages:r", "member", permsOnly(trigger.needs)},
		{"member with all rights", "member", rightsJSON(allRights)},
		{"owner with empty permissions", "owner", ""},
		{"admin with malformed permissions", "admin", "not json"},
	} {
		f.setMember(t, step.role, step.perms)
		direct := f.direct(t, f.userID, trigger.name, trigger.args(f, f.tenantA))
		directRun := f.assertAccepted(t, step.label+"/direct", direct)
		// R044: the tool's own-tenant lookup plus the reservation's parent lock and terminal write also
		// read jobs, so an admitted call is observed at least once (rejected calls: exactly zero or one).
		if f.obs.count("jobs") < 1 {
			t.Fatalf("%s: the own-tenant job lookup must be observed, saw %d", step.label, f.obs.count("jobs"))
		}

		rec, resp := f.rpc(t, f.token, body)
		if rec.Code != http.StatusOK || resp["error"] != nil || resp["result"] == nil {
			t.Fatalf("%s: mounted call must be a normal JSON-RPC result: %d %v", step.label, rec.Code, resp)
		}
		res := resp["result"].(map[string]interface{})
		content := res["content"].([]interface{})
		if len(content) != 1 || res["isError"] == true {
			t.Fatalf("%s: want exactly one content item and no isError, got %v", step.label, res)
		}
		item := content[0].(map[string]interface{})
		if item["type"] != "text" {
			t.Fatalf("%s: mounted content %v", step.label, item)
		}
		mountedRun := f.assertAccepted(t, step.label+"/mounted", toolOutcome{text: item["text"].(string)})
		for _, id := range []string{directRun, mountedRun} {
			if seen[id] {
				t.Fatalf("%s: run id %s reused", step.label, id)
			}
			seen[id] = true
			var status, tenant string
			db.DB.Raw("SELECT status, tenant_id FROM job_runs WHERE id = ?", id).Row().Scan(&status, &tenant)
			if tenant != f.tenantA || status == "" {
				t.Fatalf("%s: returned run id %s is not a persisted own-tenant run (tenant %q)", step.label, id, tenant)
			}
		}
		if strings.Contains(rec.Body.String(), f.jobName(f.tenantA)) {
			t.Fatalf("%s: mounted body carries the job name", step.label)
		}
	}
	if loads, starts := f.disp.counts(); loads != 8 || starts != 8 {
		t.Fatalf("dispatcher saw %d loads and %d starts for 8 admitted calls", loads, starts)
	}
}

// MT-02: admission still precedes the job lookup for every way of not being admitted.
func TestTriggerAdmissionPrecedesLookup(t *testing.T) {
	f := newMCPFixture(t)
	trigger := toolMatrix[9]
	args := trigger.args(f, f.tenantA)
	type denial struct {
		label  string
		prep   func()
		user   string
		args   map[string]interface{}
		wantTx string
	}
	generic, tenantDenied := "permission denied", "access denied: you don't have access to this tenant"
	cases := []denial{
		{"missing jobs:w", func() { f.setMember(t, "member", permsExcept("jobs", "w")) }, f.userID, args, generic},
		{"missing messages:r", func() { f.setMember(t, "member", permsExcept("messages", "r")) }, f.userID, args, generic},
		{"only jobs:w", func() { f.setMember(t, "member", permsOnly([]string{"jobs:w"})) }, f.userID, args, generic},
		{"only messages:r", func() { f.setMember(t, "member", permsOnly([]string{"messages:r"})) }, f.userID, args, generic},
		{"malformed permissions", func() { f.setMember(t, "member", "not json") }, f.userID, args, generic},
		{"unknown role", func() { f.setMember(t, "guest", rightsJSON(allRights)) }, f.userID, args, generic},
		{"uppercase role", func() { f.setMember(t, "OWNER", rightsJSON(allRights)) }, f.userID, args, generic},
		{"absent membership", func() { f.setMember(t, "owner", "") }, f.outsiderID, args, tenantDenied},
		{"cross tenant", func() { f.setMember(t, "owner", "") }, f.userID, trigger.args(f, f.tenantB), tenantDenied},
	}
	for _, c := range cases {
		c.prep()
		user := c.user
		if user == "" {
			user = f.userID
		}
		got := f.direct(t, user, trigger.name, c.args) // exactly one call: an admitted pre-call would dispatch (R044)
		f.assertDenied(t, c.label, got, c.wantTx)
		if f.obs.count("jobs") != 0 {
			t.Fatalf("%s: a denied call queried jobs", c.label)
		}
		if strings.Contains(got.text, "job_triggered") {
			t.Fatalf("%s: denial must differ from an accepted trigger", c.label)
		}
		f.assertNoDispatch(t, c.label) // denied before lookup, configuration, reservation and start
		if n := f.jobRuns(t); n != 0 {
			t.Fatalf("%s: a denied call reserved %d runs", c.label, n)
		}
	}
	// Admitted again: now the lookup and the dispatch run and the response is an accepted trigger.
	f.setMember(t, "member", permsOnly(trigger.needs))
	f.assertAccepted(t, "re-admitted", f.direct(t, f.userID, trigger.name, args))
	if loads, starts := f.disp.counts(); loads != 1 || starts != 1 {
		t.Fatalf("re-admitted call: %d loads %d starts, want 1 and 1 (the control that proves the counters work)", loads, starts)
	}
}

// MT-03: the lookup keeps both predicates; other-tenant, unknown, empty and forced-error cases
// return the generic Job not found with no data.
func TestTriggerLookupKeepsTenantPredicateAndGenericErrors(t *testing.T) {
	f := newMCPFixture(t)
	trigger := toolMatrix[9]
	f.setMember(t, "member", permsOnly(trigger.needs))
	sqlObs := &sqlObserver{}
	sqlObs.register(t, db.DB, "jobs")

	own := f.direct(t, f.userID, trigger.name, trigger.args(f, f.tenantA))
	f.assertAccepted(t, "own job", own)
	seen := sqlObs.snapshot()
	// R044: the tool's own lookup is the first jobs query; the reservation's parent lock is a second,
	// locking one (id and tenant predicates as well). The lookup is checked here.
	if len(seen) < 1 {
		t.Fatalf("expected the jobs lookup, saw none")
	}
	f.disp.mu.Lock()
	f.disp.loads, f.disp.starts = 0, 0
	f.disp.mu.Unlock()
	sql := strings.ToLower(seen[0].sql)
	if !strings.Contains(sql, "id = ?") || !strings.Contains(sql, "tenant_id = ?") || strings.Count(sql, "?") < 2 {
		t.Fatalf("lookup must carry both id and tenant predicates: %s", seen[0].sql)
	}
	vars := map[string]bool{}
	for _, v := range seen[0].vars {
		if s, ok := v.(string); ok {
			vars[s] = true
		}
	}
	if !vars["job-"+f.tenantA] || !vars[f.tenantA] {
		t.Fatalf("lookup bind variables %v must contain the job id and the requested tenant", seen[0].vars)
	}

	// The other tenant's job exists in the database but is never reachable through tenant A.
	var n int64
	db.DB.Raw("SELECT COUNT(*) FROM jobs WHERE id = ?", "job-"+f.tenantB).Scan(&n)
	if n != 1 {
		t.Fatalf("fixture: the other-tenant job must exist")
	}
	notFound := []struct {
		label string
		args  map[string]interface{}
	}{
		{"other tenant's job id", map[string]interface{}{"tenant_id": f.tenantA, "job_id": "job-" + f.tenantB}},
		{"unknown job id", map[string]interface{}{"tenant_id": f.tenantA, "job_id": "job-does-not-exist"}},
		{"empty job id", map[string]interface{}{"tenant_id": f.tenantA, "job_id": ""}},
		{"missing job id", map[string]interface{}{"tenant_id": f.tenantA}},
		{"non-string job id", map[string]interface{}{"tenant_id": f.tenantA, "job_id": 12345}},
	}
	for _, c := range notFound {
		got := f.direct(t, f.userID, trigger.name, c.args)
		if got.rpcErr != nil || !got.isErr || got.text != "Job not found" {
			t.Fatalf("%s: want generic Job not found, got isErr=%v rpcErr=%v %q", c.label, got.isErr, got.rpcErr, got.text)
		}
		if strings.Contains(got.text, f.tenantB) || strings.Contains(got.text, forbiddenMarker) {
			t.Fatalf("%s: disclosure %q", c.label, got.text)
		}
		f.assertNoDispatch(t, c.label) // not found never reaches configuration, reservation or start
	}

	// Forced job-query failure: the same generic result, no SQL or driver text, no job data.
	got := f.directFailing(t, f.userID, trigger.name, trigger.args(f, f.tenantA), "jobs")
	if got.rpcErr != nil || !got.isErr || got.text != "Job not found" {
		t.Fatalf("forced read error: got isErr=%v rpcErr=%v %q", got.isErr, got.rpcErr, got.text)
	}
	if f.obs.count("jobs") != 1 {
		t.Fatalf("the failing jobs lookup must have been attempted once, saw %d", f.obs.count("jobs"))
	}
	for _, leak := range []string{secretSQLMarker, "injected", "SELECT", "jobs", f.jobName(f.tenantA), triggerUnavailableWant} {
		if strings.Contains(got.text, leak) {
			t.Fatalf("forced read error disclosed %q", leak)
		}
	}
	f.assertNoDispatch(t, "forced read error")
	f.obs.reset()
}

// directFailing is direct() with a query failure injected on one table for this call only.
func (f *mcpFixture) directFailing(t *testing.T, userID, tool string, args map[string]interface{}, table string) toolOutcome {
	t.Helper()
	f.obs.reset()
	f.obs.failOn(table)
	params, _ := json.Marshal(ToolCallParams{Name: tool, Arguments: args})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("mcp_user_id", userID)
	res, rpcErr := handleToolsCall(c, params)
	out := toolOutcome{rpcErr: rpcErr}
	if tr, ok := res.(ToolResult); ok {
		out.isErr = tr.IsError
		if len(tr.Content) > 0 {
			out.text = tr.Content[0].Text
		}
	}
	return out
}

// MT-04: rejected requests (denied rights, absent membership, foreign tenant, unknown job, extra
// arguments) cause no write, run, dispatch, outbound request or state change. The accepted call is
// the positive control that the probes can see a reservation.
func TestTriggerRejectionsHaveNoWritesRunsDispatchOrOutboundRequests(t *testing.T) {
	f := newMCPFixture(t)
	trigger := toolMatrix[9]
	probe := &effectProbe{}
	probe.install(t, db.DB)
	before := checksums(t)

	run := func(setup func(), userID string, args map[string]interface{}) {
		setup()
		f.direct(t, userID, trigger.name, args)
	}
	args := trigger.args(f, f.tenantA)
	extra := trigger.args(f, f.tenantA)
	extra["mode"] = "unanalyzed"
	// membership changes are fixture UPDATEs (raw Exec, outside these callbacks) before each call
	run(func() { f.setMember(t, "member", permsOnly(trigger.needs)) }, f.userID,
		map[string]interface{}{"tenant_id": f.tenantA, "job_id": "job-" + f.tenantB}) // unknown to tenant A
	run(func() {}, f.userID, extra) // admitted, own job, extra argument
	run(func() { f.setMember(t, "member", permsExcept("jobs", "w")) }, f.userID, args)
	run(func() { f.setMember(t, "owner", "") }, f.userID, trigger.args(f, f.tenantB))
	run(func() { f.setMember(t, "owner", "") }, f.outsiderID, args)
	f.setMember(t, "member", permsOnly(trigger.needs))
	// forced lookup failure on the mounted route is covered by TestTriggerForcedReadErrorHasNoEffects

	probe.mu.Lock()
	writes, outbound := probe.writes, probe.http
	probe.mu.Unlock()
	if outbound != 0 {
		t.Fatalf("%d outbound HTTP attempts on the default transport", outbound)
	}
	for table, n := range writes {
		if table != tokenTable {
			t.Fatalf("%d GORM write callbacks on %s during rejected trigger calls", n, table)
		}
	}
	f.assertNoDispatch(t, "rejections")
	after := checksums(t)
	for table, sum := range before {
		if table == "user_tenants" {
			continue // the test itself changes the membership row between calls
		}
		if after[table] != sum {
			t.Fatalf("table %s changed during rejected trigger calls (%d -> %d)", table, sum, after[table])
		}
	}
	if n := f.jobRuns(t); n != 0 {
		t.Fatalf("%d job runs were created by rejected calls", n)
	}

	// Positive controls: the probes can see a GORM write, a default-transport request and a reservation.
	probe.mu.Lock()
	probe.writes, probe.http = nil, 0
	probe.mu.Unlock()
	if err := db.DB.Model(&models.Job{}).Where("id = ?", "job-"+f.tenantB).Update("name", f.jobName(f.tenantB)).Error; err != nil {
		t.Fatalf("control write: %v", err)
	}
	if _, err := http.Get("http://127.0.0.1:1/r035-control"); err == nil {
		t.Fatalf("the outbound trap must fail the control request")
	}
	probe.mu.Lock()
	controlWrites, controlHTTP := probe.writes["jobs"], probe.http
	probe.mu.Unlock()
	if controlWrites != 1 || controlHTTP != 1 {
		t.Fatalf("probes did not observe the controls: writes=%d http=%d", controlWrites, controlHTTP)
	}
	f.assertAccepted(t, "dispatch control", f.direct(t, f.userID, trigger.name, args))
	if f.jobRuns(t) != 1 {
		t.Fatalf("the reservation control left %d runs", f.jobRuns(t))
	}
	probe.mu.Lock()
	reserved := probe.writes["job_runs"]
	probe.mu.Unlock()
	if reserved == 0 {
		t.Fatalf("the write probe did not see the reservation's job_runs insert")
	}
}

// R035-R1-01: the forced jobs-read-error path has no effects either. Membership is prepared
// before the observation starts, so the checksums (including user_tenants) compare equal only
// if the tool itself changed nothing; the lookup must still have been attempted exactly once.
func TestTriggerForcedReadErrorHasNoEffects(t *testing.T) {
	f := newMCPFixture(t)
	trigger := toolMatrix[9]
	f.setMember(t, "member", permsOnly(trigger.needs))
	probe := &effectProbe{}
	probe.install(t, db.DB)
	before := checksums(t)
	runsBefore := f.jobRuns(t)

	got := f.directFailing(t, f.userID, trigger.name, trigger.args(f, f.tenantA), "jobs")
	if got.rpcErr != nil || !got.isErr || got.text != "Job not found" {
		t.Fatalf("forced read error must stay the generic result, got isErr=%v rpcErr=%v %q", got.isErr, got.rpcErr, got.text)
	}
	if f.obs.count("jobs") != 1 {
		t.Fatalf("the failing jobs lookup must have been attempted once, saw %d", f.obs.count("jobs"))
	}
	probe.mu.Lock()
	writes, outbound := probe.writes, probe.http
	probe.mu.Unlock()
	if len(writes) != 0 || outbound != 0 {
		t.Fatalf("FORCED_ERROR_EFFECT: gorm writes=%v outbound=%d", writes, outbound)
	}
	after := checksums(t)
	for table, sum := range before {
		if after[table] != sum {
			t.Fatalf("FORCED_ERROR_STATE_CHANGE: table %s changed (%d -> %d)", table, sum, after[table])
		}
	}
	if n := f.jobRuns(t); n != runsBefore || n != 0 {
		t.Fatalf("job runs changed on the forced-error path: %d -> %d", runsBefore, n)
	}
	f.assertNoDispatch(t, "forced read error")
}

// goldenDescriptions are the other eleven published descriptions, copied independently of tools.go.
var goldenDescriptions = map[string]string{
	"cqa_list_tenants":          "List the companies the user has access to (id, name, slug only).",
	"cqa_get_tenant":            "Get the id, name and slug of a specific company the user has access to.",
	"cqa_list_channels":         "List chat channels for a company with status, last sync time, and message count.",
	"cqa_list_conversations":    "List conversations, optionally filtered by channel, date range, or customer name.",
	"cqa_get_messages":          "Get messages for a specific conversation with pagination.",
	"cqa_search_messages":       "Search messages by keyword across all conversations.",
	"cqa_list_jobs":             "List analysis jobs for a company with status and last run info.",
	"cqa_get_job_results":       "Get analysis results for a specific job run.",
	"cqa_search_violations":     "Search QC violations by severity, date, channel, or keyword.",
	"cqa_get_stats":             "Get overall statistics: conversations, issues, tags by time period.",
	"cqa_get_notification_logs": "Get notification history filtered by date, channel type, or status.",
}

// MT-05: the published description tells the truth, the schema is preserved and nothing else moved.
func TestTriggerDescriptionAndToolsList(t *testing.T) {
	f := newMCPFixture(t)
	f.setMember(t, "member", "{}")
	_, resp := f.rpc(t, f.token, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	tools := resp["result"].(map[string]interface{})["tools"].([]interface{})
	if len(tools) != 12 {
		t.Fatalf("tools/list returned %d tools, want 12", len(tools))
	}
	seen := 0
	for _, raw := range tools {
		tool := raw.(map[string]interface{})
		name, desc := tool["name"].(string), tool["description"].(string)
		if name != "cqa_trigger_job" {
			if golden, ok := goldenDescriptions[name]; !ok || desc != golden {
				t.Errorf("description of %s changed: %q", name, desc)
			}
			continue
		}
		seen++
		lower := strings.ToLower(desc)
		// R044: the description states background acceptance (not completion), the default-only
		// semantics and the extra-argument rejection; the retired unavailable text is gone.
		for _, must := range []string{"background", "since_last", "no date range", "no conversation cap", "job_triggered", "run_id", "does not mean the analysis has finished", "invalid_run_parameters", "job_already_running", "job_start_failed"} {
			if !strings.Contains(lower, strings.ToLower(must)) {
				t.Errorf("description must mention %q: %q", must, desc)
			}
		}
		for _, retired := range []string{"unavailable", "does not start", triggerUnavailableWant} {
			if strings.Contains(lower, retired) {
				t.Errorf("description still carries the retired %q: %q", retired, desc)
			}
		}
		for _, promise := range []string{"immediately", "manually trigger", "run now", "will run", "queued for", "completed analysis", "durable", "guarantee"} {
			if strings.Contains(lower, promise) {
				t.Errorf("description over-promises (%q): %q", promise, desc)
			}
		}
		schema := tool["inputSchema"].(map[string]interface{})
		props := schema["properties"].(map[string]interface{})
		req := schema["required"].([]interface{})
		if schema["type"] != "object" || len(props) != 2 || props["tenant_id"] == nil || props["job_id"] == nil || len(req) != 2 || req[0] != "tenant_id" || req[1] != "job_id" {
			t.Errorf("input schema changed: %v", schema)
		}
		if p := props["job_id"].(map[string]interface{}); p["type"] != "string" || p["description"] != "Job UUID to trigger" {
			t.Errorf("job_id property changed: %v", p)
		}
	}
	if seen != 1 {
		t.Fatalf("cqa_trigger_job published %d times", seen)
	}
	// the policy is unchanged: both rights, tenant scoped
	pol := toolPolicies["cqa_trigger_job"]
	if !pol.tenantScoped || len(pol.rights) != 2 || pol.rights[0] != (toolRight{"jobs", "w"}) || pol.rights[1] != (toolRight{"messages", "r"}) {
		t.Fatalf("trigger policy changed: %+v", pol)
	}
}

// MT-06 guard: the shared positive helper accepts only a non-error result and the trigger's own
// predicate accepts only the exact accepted shape; the retired unavailable error, arbitrary errors
// and malformed or extended bodies never count as an admitted trigger.
func TestAdmittedOutcomeHelperRejectsArbitraryErrors(t *testing.T) {
	trigger := toolMatrix[9]
	other := toolMatrix[0]
	if trigger.name != "cqa_trigger_job" || other.name != "cqa_list_channels" {
		t.Fatalf("matrix order changed")
	}
	good := `{"message":"job_triggered","run_id":"abc"}`
	if p := admittedOutcomeProblem(trigger, toolOutcome{text: good}); p != "" || !trigger.own(good, "t") {
		t.Fatalf("the accepted trigger must be accepted: %q own=%v", p, trigger.own(good, "t"))
	}
	for _, bad := range []toolOutcome{
		{isErr: true, text: "permission denied"}, {isErr: true, text: "Job not found"}, {isErr: true, text: "record not found"},
		{isErr: true, text: triggerUnavailableWant}, {isErr: true, text: good}, {isErr: true, text: ""},
		{text: good, rpcErr: &RPCError{Code: -32602}},
	} {
		if admittedOutcomeProblem(trigger, bad) == "" {
			t.Errorf("trigger outcome %+v must not count as admitted", bad)
		}
	}
	for _, text := range []string{
		triggerUnavailableWant, "", "{}", `{"message":"job_triggered"}`, `{"message":"job_triggered","run_id":""}`,
		`{"message":"queued","run_id":"abc"}`, `{"message":"job_triggered","run_id":"abc","extra":"x"}`, "job_triggered",
	} {
		if trigger.own(text, "t") {
			t.Errorf("trigger predicate accepted %q", text)
		}
	}
	if admittedOutcomeProblem(other, toolOutcome{isErr: true, text: "permission denied"}) == "" {
		t.Errorf("a read tool must not accept an error as success")
	}
	if p := admittedOutcomeProblem(other, toolOutcome{text: "ok"}); p != "" {
		t.Errorf("a normal success must be accepted: %s", p)
	}
}
