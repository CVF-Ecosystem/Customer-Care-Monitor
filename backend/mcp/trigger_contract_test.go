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

// CCMAI-RUNTIME-035: cqa_trigger_job reports a fixed unavailable tool error after the unchanged
// authorization and own-tenant lookup, never "triggered"/"queued". Disposable MySQL and synthetic
// rows only; nothing is dispatched. Observation limits: write callbacks, table checksums and a
// replaced http.DefaultTransport see GORM writes and default-transport HTTP; they do not observe
// raw sockets, other transports or goroutines that use other clients.

// triggerHistory lists strings the retired placeholder response could carry.
var triggerHistory = []string{"triggered", "queued", "accepted", "started", "run_id", "job_run", "has been", "status"}

func (f *mcpFixture) jobName(tenant string) string { return "JOB-" + forbiddenMarker + "-" + tenant }

func (f *mcpFixture) assertUnavailable(t *testing.T, label string, got toolOutcome) {
	t.Helper()
	if got.rpcErr != nil || !got.isErr || got.text != triggerUnavailableWant {
		t.Fatalf("%s: want tool error %q, got isErr=%v rpcErr=%v text=%q", label, triggerUnavailableWant, got.isErr, got.rpcErr, got.text)
	}
	lower := strings.ToLower(got.text)
	for _, bad := range triggerHistory {
		if strings.Contains(lower, bad) {
			t.Fatalf("%s: response still mentions %q: %q", label, bad, got.text)
		}
	}
	for _, leak := range []string{f.jobName(f.tenantA), "job-" + f.tenantA, f.tenantA, f.tenantB, secretSQLMarker, forbiddenMarker} {
		if strings.Contains(got.text, leak) {
			t.Fatalf("%s: response disclosed %q", label, leak)
		}
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

// MT-01: every admitted principal gets the same fixed tool error, never a protocol error, and the
// direct and mounted paths agree.
func TestTriggerUnavailableDirectAndMounted(t *testing.T) {
	f := newMCPFixture(t)
	trigger := toolMatrix[9]
	if trigger.name != "cqa_trigger_job" {
		t.Fatalf("matrix order changed")
	}
	body := callBody(trigger.name, trigger.args(f, f.tenantA))
	for _, step := range []struct{ label, role, perms string }{
		{"member with exactly jobs:w+messages:r", "member", permsOnly(trigger.needs)},
		{"member with all rights", "member", rightsJSON(allRights)},
		{"owner with empty permissions", "owner", ""},
		{"admin with malformed permissions", "admin", "not json"},
	} {
		f.setMember(t, step.role, step.perms)
		direct := f.direct(t, f.userID, trigger.name, trigger.args(f, f.tenantA))
		f.assertUnavailable(t, step.label+"/direct", direct)
		if f.obs.count("jobs") != 1 {
			t.Fatalf("%s: the own-tenant job lookup must be observed once, saw %d", step.label, f.obs.count("jobs"))
		}

		rec, resp := f.rpc(t, f.token, body)
		if rec.Code != http.StatusOK || resp["error"] != nil || resp["result"] == nil {
			t.Fatalf("%s: mounted call must be a normal JSON-RPC result: %d %v", step.label, rec.Code, resp)
		}
		res := resp["result"].(map[string]interface{})
		content := res["content"].([]interface{})
		if len(content) != 1 || res["isError"] != true {
			t.Fatalf("%s: want exactly one content item with isError=true, got %v", step.label, res)
		}
		item := content[0].(map[string]interface{})
		if item["type"] != "text" || item["text"] != triggerUnavailableWant {
			t.Fatalf("%s: mounted content %v", step.label, item)
		}
		f.assertUnavailable(t, step.label+"/mounted", toolOutcome{text: item["text"].(string), isErr: true})
		if mounted, _ := resultText(t, resp); mounted != direct.text {
			t.Fatalf("%s: direct %q and mounted %q disagree", step.label, direct.text, mounted)
		}
		if strings.Contains(rec.Body.String(), f.jobName(f.tenantA)) {
			t.Fatalf("%s: mounted body carries the job name", step.label)
		}
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
		got := f.direct(t, f.userID, trigger.name, c.args)
		if c.user != f.userID {
			got = f.direct(t, c.user, trigger.name, c.args)
		}
		f.assertDenied(t, c.label, got, c.wantTx)
		if f.obs.count("jobs") != 0 {
			t.Fatalf("%s: a denied call queried jobs", c.label)
		}
		if got.text == triggerUnavailableWant {
			t.Fatalf("%s: denial must differ from the admitted unavailable error", c.label)
		}
	}
	// Admitted again: now the lookup runs and the response is the distinct unavailable error.
	f.setMember(t, "member", permsOnly(trigger.needs))
	f.assertUnavailable(t, "re-admitted", f.direct(t, f.userID, trigger.name, args))
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
	f.assertUnavailable(t, "own job", own)
	seen := sqlObs.snapshot()
	if len(seen) != 1 {
		t.Fatalf("expected one jobs query, saw %d", len(seen))
	}
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

// MT-04: no write, run, outbound request or state change on any path.
func TestTriggerNoWritesRunsOrOutboundRequests(t *testing.T) {
	f := newMCPFixture(t)
	trigger := toolMatrix[9]
	probe := &effectProbe{}
	probe.install(t, db.DB)
	before := checksums(t)

	run := func(label string, setup func(), userID string, args map[string]interface{}) {
		setup()
		f.direct(t, userID, trigger.name, args)
	}
	args := trigger.args(f, f.tenantA)
	run("unavailable/member", func() { f.setMember(t, "member", permsOnly(trigger.needs)) }, f.userID, args)
	run("unavailable/owner", func() { f.setMember(t, "owner", "") }, f.userID, args)
	run("not found", func() { f.setMember(t, "member", permsOnly(trigger.needs)) }, f.userID,
		map[string]interface{}{"tenant_id": f.tenantA, "job_id": "job-" + f.tenantB})
	run("denied rights", func() { f.setMember(t, "member", permsExcept("jobs", "w")) }, f.userID, args)
	run("denied tenant", func() { f.setMember(t, "owner", "") }, f.userID, trigger.args(f, f.tenantB))
	// the mounted route as well
	f.setMember(t, "member", permsOnly(trigger.needs))
	f.rpc(t, f.token, callBody(trigger.name, args))

	// setMember is a fixture UPDATE; those are the only writes and happen outside the tool call.
	probe.mu.Lock()
	writes, outbound := probe.writes, probe.http
	probe.mu.Unlock()
	if outbound != 0 {
		t.Fatalf("%d outbound HTTP attempts on the default transport", outbound)
	}
	for table, n := range writes { // fixture setMember uses raw Exec, which these callbacks do not see
		if table != tokenTable {
			t.Fatalf("%d GORM write callbacks on %s during trigger calls", n, table)
		}
	}
	t.Logf("write callbacks by table (bearer-token bookkeeping only is tolerated): %v", writes)
	after := checksums(t)
	for table, sum := range before {
		if table == "user_tenants" {
			continue // the test itself changes the membership row between calls
		}
		if after[table] != sum {
			t.Fatalf("table %s changed during trigger calls (%d -> %d)", table, sum, after[table])
		}
	}
	if n := f.jobRuns(t); n != 0 {
		t.Fatalf("%d job runs were created", n)
	}

	// Positive controls: the probes are able to see a GORM write and a default-transport request.
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
	defer probe.mu.Unlock()
	if probe.writes["jobs"] != 1 || probe.http != 1 {
		t.Fatalf("probes did not observe the controls: writes=%v http=%d", probe.writes, probe.http)
	}
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
		if !strings.Contains(lower, "unavailable") || !strings.Contains(lower, "does not start or queue") || !strings.Contains(desc, triggerUnavailableWant) {
			t.Errorf("description must state the tool is unavailable and starts nothing: %q", desc)
		}
		for _, promise := range []string{"immediately", "manually trigger", "run now", "will run", "queued for"} {
			if strings.Contains(lower, promise) {
				t.Errorf("description still promises execution (%q): %q", promise, desc)
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

// MT-06 guard: the shared positive helper accepts an error only for the listed tool and only with
// its exact text; arbitrary tool errors never count as an admitted outcome.
func TestAdmittedOutcomeHelperRejectsArbitraryErrors(t *testing.T) {
	trigger := toolMatrix[9]
	other := toolMatrix[0]
	if trigger.name != "cqa_trigger_job" || other.name != "cqa_list_channels" {
		t.Fatalf("matrix order changed")
	}
	if p := admittedOutcomeProblem(trigger, toolOutcome{isErr: true, text: triggerUnavailableWant}); p != "" {
		t.Fatalf("the fixed unavailable error must be accepted for the trigger: %s", p)
	}
	for _, bad := range []toolOutcome{
		{isErr: true, text: "permission denied"}, {isErr: true, text: "Job not found"}, {isErr: true, text: "record not found"},
		{isErr: true, text: triggerUnavailableWant + " "}, {isErr: true, text: ""}, {isErr: false, text: triggerUnavailableWant},
		{isErr: true, text: triggerUnavailableWant, rpcErr: &RPCError{Code: -32602}},
	} {
		if admittedOutcomeProblem(trigger, bad) == "" {
			t.Errorf("trigger outcome %+v must not count as admitted", bad)
		}
	}
	if admittedOutcomeProblem(other, toolOutcome{isErr: true, text: triggerUnavailableWant}) == "" {
		t.Errorf("a read tool must never be allowed to return the trigger error")
	}
	if admittedOutcomeProblem(other, toolOutcome{isErr: true, text: "permission denied"}) == "" {
		t.Errorf("a read tool must not accept an error as success")
	}
	if p := admittedOutcomeProblem(other, toolOutcome{text: "ok"}); p != "" {
		t.Errorf("a normal success must be accepted: %s", p)
	}
	if len(expectedToolError) != 1 {
		t.Errorf("exactly one tool may have an expected error, got %v", expectedToolError)
	}
}
