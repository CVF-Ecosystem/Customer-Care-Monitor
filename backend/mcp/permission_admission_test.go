package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-021 (F01-B): every MCP tool is admitted by the authenticated user's stored
// membership for the requested tenant and the exact rights below. The matrix is written out
// here independently of toolPolicies so drift in either is caught. Disposable MySQL and
// synthetic rows only; the observer below records which tables a call queried, so a denied call
// is proved to stop after the membership lookup while a permitted call proves the observer can
// see the protected lookup. No provider, channel or job dispatch is involved.

const forbiddenMarker = "R021-FORBIDDEN"

// tokenTable is the table the bearer-token check reads (the GORM default name of OAuthToken).
const tokenTable = "o_auth_tokens"

type toolCase struct {
	name  string
	needs []string // resource:letter, every one required
	table string   // protected table the handler queries
	args  func(f *mcpFixture, tenant string) map[string]interface{}
	// own reports whether the response text carries the tenant's own seeded data.
	own func(text, tenant string) bool
}

func containsMarker(kind string) func(text, tenant string) bool {
	return func(text, tenant string) bool { return strings.Contains(text, kind+"-"+tenant) }
}

var toolMatrix = []toolCase{
	{"cqa_list_channels", []string{"channels:r"}, "channels",
		func(f *mcpFixture, tn string) map[string]interface{} { return map[string]interface{}{"tenant_id": tn} },
		containsMarker("CH-" + forbiddenMarker)},
	{"cqa_list_conversations", []string{"messages:r"}, "conversations",
		func(f *mcpFixture, tn string) map[string]interface{} { return map[string]interface{}{"tenant_id": tn} },
		containsMarker("CUST-" + forbiddenMarker)},
	{"cqa_get_messages", []string{"messages:r"}, "messages",
		func(f *mcpFixture, tn string) map[string]interface{} {
			return map[string]interface{}{"tenant_id": tn, "conversation_id": "conv-" + tn + "-0"}
		},
		containsMarker("MSG-" + forbiddenMarker)},
	{"cqa_search_messages", []string{"messages:r"}, "messages",
		func(f *mcpFixture, tn string) map[string]interface{} {
			return map[string]interface{}{"tenant_id": tn, "query": forbiddenMarker}
		},
		containsMarker("MSG-" + forbiddenMarker)},
	{"cqa_list_jobs", []string{"jobs:r"}, "jobs",
		func(f *mcpFixture, tn string) map[string]interface{} { return map[string]interface{}{"tenant_id": tn} },
		containsMarker("JOB-" + forbiddenMarker)},
	{"cqa_get_job_results", []string{"jobs:r"}, "job_results",
		func(f *mcpFixture, tn string) map[string]interface{} {
			return map[string]interface{}{"tenant_id": tn, "job_run_id": "run-" + tn}
		},
		containsMarker("EVID-" + forbiddenMarker)},
	{"cqa_search_violations", []string{"jobs:r"}, "job_results",
		func(f *mcpFixture, tn string) map[string]interface{} { return map[string]interface{}{"tenant_id": tn} },
		containsMarker("EVID-" + forbiddenMarker)},
	{"cqa_get_stats", []string{"messages:r", "jobs:r"}, "conversations",
		func(f *mcpFixture, tn string) map[string]interface{} {
			return map[string]interface{}{"tenant_id": tn, "period": "month"}
		},
		// Tenant A has 1 message/violation, tenant B has 3: only A's own counts match.
		func(text, tenant string) bool {
			var got struct{ Conversations, Messages, Violations int }
			return json.Unmarshal([]byte(text), &got) == nil && got.Conversations == 1 && got.Messages == 1 && got.Violations == 1
		}},
	{"cqa_get_notification_logs", []string{"settings:r"}, "notification_logs",
		func(f *mcpFixture, tn string) map[string]interface{} { return map[string]interface{}{"tenant_id": tn} },
		containsMarker("NOTE-" + forbiddenMarker)},
	{"cqa_trigger_job", []string{"jobs:w", "messages:r"}, "jobs",
		func(f *mcpFixture, tn string) map[string]interface{} {
			return map[string]interface{}{"tenant_id": tn, "job_id": "job-" + tn}
		},
		// R035: an admitted trigger reaches the own-tenant lookup and then reports the fixed unavailable error.
		func(text, tenant string) bool { return text == triggerUnavailableWant }},
}

// triggerUnavailableWant is written independently of the production constant on purpose.
const triggerUnavailableWant = "job_trigger_unavailable"

// expectedToolError lists the only admitted tool whose success path is a fixed tool error.
// assertAllowed accepts an error result solely for these tools and only with the exact text.
var expectedToolError = map[string]string{"cqa_trigger_job": triggerUnavailableWant}

// ---- observer ----

type queryObserver struct {
	mu        sync.Mutex
	tables    map[string]int
	failTable string
}

func (o *queryObserver) reset() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.tables = map[string]int{}
	o.failTable = ""
}

func (o *queryObserver) failOn(table string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.failTable = table
}

func (o *queryObserver) seen() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	var out []string
	for table := range o.tables {
		out = append(out, table)
	}
	sort.Strings(out)
	return out
}

func (o *queryObserver) count(table string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.tables[table]
}

func (o *queryObserver) register(t *testing.T, gdb *gorm.DB) {
	t.Helper()
	err := gdb.Callback().Query().Before("gorm:query").Register("r021:observe", func(tx *gorm.DB) {
		o.mu.Lock()
		defer o.mu.Unlock()
		o.tables[tx.Statement.Table]++
		if o.failTable != "" && tx.Statement.Table == o.failTable {
			tx.AddError(errors.New("injected r021 failure " + secretSQLMarker))
		}
	})
	if err != nil {
		t.Fatalf("observer: %v", err)
	}
}

const secretSQLMarker = "R021-SQL-DIAGNOSTIC"

// ---- fixture ----

type mcpFixture struct {
	obs                 *queryObserver
	userID, outsiderID  string
	tenantA, tenantB    string
	tenantC             string
	token, expiredToken string
}

var allRights = map[string]string{"channels": "rwd", "messages": "rwd", "jobs": "rwd", "settings": "rwd"}

func rightsJSON(m map[string]string) string {
	b, _ := json.Marshal(m)
	return string(b)
}

// permsExcept grants every letter of every resource except one right.
func permsExcept(resource, letter string) string {
	m := map[string]string{}
	for k, v := range allRights {
		m[k] = v
	}
	m[resource] = strings.ReplaceAll(m[resource], letter, "")
	return rightsJSON(m)
}

// permsOnly grants exactly the listed resource:letter rights.
func permsOnly(needs []string) string {
	m := map[string]string{}
	for _, n := range needs {
		parts := strings.SplitN(n, ":", 2)
		m[parts[0]] += parts[1]
	}
	return rightsJSON(m)
}

func newMCPFixture(t *testing.T) *mcpFixture {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("bo qua: TEST_DB_DSN chua duoc thiet lap")
	}
	// db.Connect never closes its predecessor; close it so repeated fixtures do not exhaust
	// the disposable server's connection limit for later packages.
	db.Close()
	if err := db.Connect(dsn, false); err != nil {
		t.Skipf("bo qua: khong ket noi duoc DB test: %v", err)
	}
	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate loi: %v", err)
	}
	gin.SetMode(gin.TestMode)

	s := pkg.NewUUID()[:8]
	f := &mcpFixture{
		obs:    &queryObserver{},
		userID: pkg.NewUUID(), outsiderID: pkg.NewUUID(),
		tenantA: "r021-a-" + s, tenantB: "r021-b-" + s, tenantC: "r021-c-" + s,
		token: "tok-" + pkg.NewUUID(), expiredToken: "tok-old-" + pkg.NewUUID(),
	}
	f.obs.reset()
	f.exec(t, `INSERT INTO users (id, email, password_hash, name, is_admin, token_version, language, created_at, updated_at) VALUES (?, ?, 'x', 'R021', false, 0, 'vi', NOW(), NOW())`, f.userID, "r021-"+s+"@example.invalid")
	f.exec(t, `INSERT INTO users (id, email, password_hash, name, is_admin, token_version, language, created_at, updated_at) VALUES (?, ?, 'x', 'R021 out', false, 0, 'vi', NOW(), NOW())`, f.outsiderID, "r021-out-"+s+"@example.invalid")
	for _, tn := range []string{f.tenantA, f.tenantB, f.tenantC} {
		f.exec(t, `INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW())`,
			tn, "Name-"+tn, tn, `{"settings_marker":"`+forbiddenMarker+`-SETTINGS-`+tn+`"}`)
	}
	f.exec(t, `INSERT INTO user_tenants (user_id, tenant_id, role, permissions) VALUES (?, ?, 'member', ?)`, f.userID, f.tenantA, rightsJSON(allRights))
	f.exec(t, `INSERT INTO user_tenants (user_id, tenant_id, role, permissions) VALUES (?, ?, 'member', ?)`, f.userID, f.tenantC, rightsJSON(allRights))
	f.seed(t, f.tenantA, 1)
	f.seed(t, f.tenantB, 3)
	for tok, exp := range map[string]time.Duration{f.token: time.Hour, f.expiredToken: -time.Hour} {
		row := models.OAuthToken{
			ID: pkg.NewUUID(), ClientID: "r021-client", UserID: f.userID, AccessTokenHash: sha256Hash(tok),
			Scopes: "[]", ExpiresAt: time.Now().Add(exp).UTC(), CreatedAt: time.Now().UTC(),
		}
		if err := db.DB.Create(&row).Error; err != nil {
			t.Fatalf("fixture token: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, tn := range []string{f.tenantA, f.tenantB, f.tenantC} {
			for _, table := range []string{"notification_logs", "job_results", "job_runs", "messages", "conversations", "channels", "jobs", "user_tenants", "tenants"} {
				col := "tenant_id"
				if table == "tenants" {
					col = "id"
				}
				db.DB.Exec("DELETE FROM "+table+" WHERE "+col+" = ?", tn)
			}
		}
		db.DB.Where("user_id = ?", f.userID).Delete(&models.OAuthToken{})
		db.DB.Exec("DELETE FROM user_tenants WHERE user_id IN (?, ?)", f.userID, f.outsiderID)
		db.DB.Exec("DELETE FROM users WHERE id IN (?, ?)", f.userID, f.outsiderID)
	})
	f.obs.register(t, db.DB)
	return f
}

func (f *mcpFixture) exec(t *testing.T, sql string, args ...interface{}) {
	t.Helper()
	if err := db.DB.Exec(sql, args...).Error; err != nil {
		t.Fatalf("fixture: %v", err)
	}
}

// seed inserts n conversations/messages for a tenant plus one channel, job, qc_violation and
// notification; every text field carries the forbidden marker and the tenant ID.
func (f *mcpFixture) seed(t *testing.T, tn string, n int) {
	t.Helper()
	chID := "ch-" + tn
	f.exec(t, `INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', ?, ?, X'00', true, '{}', NOW(), NOW())`, chID, tn, "CH-"+forbiddenMarker+"-"+tn, "ext-"+tn)
	for i := 0; i < n; i++ {
		convID := "conv-" + tn + "-" + string(rune('0'+i))
		f.exec(t, `INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NOW(), 1, '{}', NOW(), NOW())`, convID, tn, chID, "x-"+convID, "CUST-"+forbiddenMarker+"-"+tn)
		f.exec(t, `INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, ?, 'customer', 'K', ?, 'text', '[]', NOW(), NOW())`, pkg.NewUUID(), tn, convID, "m-"+convID, "MSG-"+forbiddenMarker+"-"+tn)
		f.exec(t, `INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, confidence, created_at) VALUES (?, ?, ?, ?, 'qc_violation', 'NGHIEM_TRONG', 'RULE', ?, '{}', 1, NOW())`, pkg.NewUUID(), "run-"+tn, tn, convID, "EVID-"+forbiddenMarker+"-"+tn)
	}
	f.exec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, created_at, updated_at) VALUES (?, ?, ?, 'qc_analysis', '[]', '', '[]', 'manual', true, '[]', NOW(), NOW())`, "job-"+tn, tn, "JOB-"+forbiddenMarker+"-"+tn)
	f.exec(t, `INSERT INTO notification_logs (id, tenant_id, job_id, job_run_id, channel_type, recipient, subject, body, status, error_message, sent_at, created_at) VALUES (?, ?, ?, ?, 'email', 'a@example.invalid', 's', ?, 'sent', '', NOW(), NOW())`, pkg.NewUUID(), tn, "job-"+tn, "run-"+tn, "NOTE-"+forbiddenMarker+"-"+tn)
}

func (f *mcpFixture) setMember(t *testing.T, role, perms string) {
	t.Helper()
	f.exec(t, `UPDATE user_tenants SET role = ?, permissions = ? WHERE user_id = ? AND tenant_id = ?`, role, perms, f.userID, f.tenantA)
}

func (f *mcpFixture) jobRuns(t *testing.T) int64 {
	t.Helper()
	var n int64
	if err := db.DB.Raw(`SELECT COUNT(*) FROM job_runs WHERE tenant_id IN (?, ?, ?)`, f.tenantA, f.tenantB, f.tenantC).Scan(&n).Error; err != nil {
		t.Fatalf("count job_runs: %v", err)
	}
	return n
}

type toolOutcome struct {
	text   string
	isErr  bool
	rpcErr *RPCError
	tables []string
}

// direct runs handleToolsCall for an authenticated user; the observer is reset first so
// tables lists exactly what this call queried.
func (f *mcpFixture) direct(t *testing.T, userID, tool string, args map[string]interface{}) toolOutcome {
	t.Helper()
	f.obs.reset()
	params, _ := json.Marshal(ToolCallParams{Name: tool, Arguments: args})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("mcp_user_id", userID)
	res, rpcErr := handleToolsCall(c, params)
	out := toolOutcome{rpcErr: rpcErr, tables: f.obs.seen()}
	if tr, ok := res.(ToolResult); ok {
		out.isErr = tr.IsError
		if len(tr.Content) > 0 {
			out.text = tr.Content[0].Text
		}
	}
	return out
}

// protectedTables are the queried tables other than the membership lookup.
func protectedTables(tables []string) []string {
	var out []string
	for _, tb := range tables {
		if tb != "user_tenants" {
			out = append(out, tb)
		}
	}
	return out
}

func (f *mcpFixture) assertDenied(t *testing.T, label string, got toolOutcome, wantText string) {
	t.Helper()
	if got.rpcErr != nil || !got.isErr || got.text != wantText {
		t.Fatalf("%s: got isErr=%v rpcErr=%v text=%q, want error result %q", label, got.isErr, got.rpcErr, got.text, wantText)
	}
	if strings.Contains(got.text, forbiddenMarker) || strings.Contains(got.text, secretSQLMarker) {
		t.Fatalf("%s: denial leaked data: %q", label, got.text)
	}
	if p := protectedTables(got.tables); len(p) != 0 {
		t.Fatalf("%s: a denied call queried protected tables %v", label, p)
	}
}

// admittedOutcomeProblem returns "" when got is the admitted outcome of tc: success for every tool
// except one listed in expectedToolError, which must return exactly its fixed tool error.
func admittedOutcomeProblem(tc toolCase, got toolOutcome) string {
	if want, ok := expectedToolError[tc.name]; ok {
		if got.rpcErr != nil || !got.isErr || got.text != want {
			return fmt.Sprintf("expected the fixed tool error %q, got isErr=%v rpcErr=%v text=%q", want, got.isErr, got.rpcErr, got.text)
		}
		return ""
	}
	if got.rpcErr != nil || got.isErr {
		return fmt.Sprintf("expected success, got isErr=%v rpcErr=%v text=%q", got.isErr, got.rpcErr, got.text)
	}
	return ""
}

func (f *mcpFixture) assertAllowed(t *testing.T, label string, tc toolCase, tenant string, got toolOutcome) {
	t.Helper()
	if problem := admittedOutcomeProblem(tc, got); problem != "" {
		t.Fatalf("%s: %s", label, problem)
	}
	if !tc.own(got.text, tenant) {
		t.Fatalf("%s: response lacks the tenant's own data: %s", label, got.text)
	}
	if other := f.tenantB; tenant == f.tenantA && strings.Contains(got.text, other) {
		t.Fatalf("%s: response leaked another tenant's data: %s", label, got.text)
	}
	// Positive observer: the membership lookup and the protected handler lookup both ran.
	found := false
	for _, tb := range got.tables {
		if tb == tc.table {
			found = true
		}
	}
	if !found {
		t.Fatalf("%s: observer did not see the protected %s lookup (saw %v)", label, tc.table, got.tables)
	}
}

// ---- tests ----

// The matrix here must name exactly the tools the server publishes, and every published tool
// must have a policy: nothing can be added without an explicit decision.
func TestMCPToolMatrixCoversEveryPublishedTool(t *testing.T) {
	published := map[string]bool{}
	for _, tool := range getAllTools() {
		published[tool.Name] = true
	}
	if len(published) != 12 {
		t.Fatalf("expected 12 published tools, got %d", len(published))
	}
	covered := map[string]bool{"cqa_list_tenants": true, "cqa_get_tenant": true}
	for _, tc := range toolMatrix {
		covered[tc.name] = true
	}
	for name := range published {
		if !covered[name] {
			t.Errorf("published tool %s is missing from the test matrix", name)
		}
		if _, ok := toolPolicies[name]; !ok {
			t.Errorf("published tool %s has no authorization policy", name)
		}
	}
	for name := range covered {
		if !published[name] {
			t.Errorf("matrix tool %s is not published", name)
		}
	}
	for name := range toolPolicies {
		if !published[name] {
			t.Errorf("policy for unpublished tool %s", name)
		}
	}
}

// Every protected tool: complete rights admit and return only the requested tenant's data;
// each missing right (with every other letter granted) is denied generically before any
// protected lookup, and the denied call causes no job run.
func TestMCPEachToolRequiresExactlyItsRights(t *testing.T) {
	f := newMCPFixture(t)
	for _, tc := range toolMatrix {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			f.setMember(t, "member", permsOnly(tc.needs))
			f.assertAllowed(t, "exact rights", tc, f.tenantA, f.direct(t, f.userID, tc.name, tc.args(f, f.tenantA)))

			for _, need := range tc.needs {
				parts := strings.SplitN(need, ":", 2)
				f.setMember(t, "member", permsExcept(parts[0], parts[1]))
				got := f.direct(t, f.userID, tc.name, tc.args(f, f.tenantA))
				f.assertDenied(t, "missing "+need, got, "permission denied")
				if got.rpcErr != nil {
					t.Fatalf("denial must be an MCP error result, got RPC error %v", got.rpcErr)
				}
			}
			// Only the other half of a conjunctive rule is not enough.
			if len(tc.needs) > 1 {
				for _, need := range tc.needs {
					f.setMember(t, "member", permsOnly([]string{need}))
					f.assertDenied(t, "only "+need, f.direct(t, f.userID, tc.name, tc.args(f, f.tenantA)), "permission denied")
				}
			}
			if n := f.jobRuns(t); n != 0 {
				t.Fatalf("%d job runs exist; no tool may dispatch a job", n)
			}
		})
	}
}

// Owner and admin bypass the permission letters only, with empty or broken permission data.
func TestMCPOwnerAndAdminBypassLettersButNotMembership(t *testing.T) {
	f := newMCPFixture(t)
	for _, role := range []string{"owner", "admin"} {
		for _, perms := range []string{"", "{}", "not json"} {
			f.setMember(t, role, perms)
			for _, tc := range toolMatrix {
				f.assertAllowed(t, role+"/"+perms+"/"+tc.name, tc, f.tenantA, f.direct(t, f.userID, tc.name, tc.args(f, f.tenantA)))
			}
		}
	}
	// The same owner row grants nothing in a tenant where the user has no membership.
	f.setMember(t, "owner", "")
	for _, tc := range toolMatrix {
		f.assertDenied(t, "owner of A on B/"+tc.name, f.direct(t, f.userID, tc.name, tc.args(f, f.tenantB)),
			"access denied: you don't have access to this tenant")
	}
	if n := f.jobRuns(t); n != 0 {
		t.Fatalf("%d job runs exist", n)
	}
}

// Missing/malformed permissions and unrecognized roles never admit a call.
func TestMCPBadPermissionDataAndRolesFailClosed(t *testing.T) {
	f := newMCPFixture(t)
	for _, perms := range []string{"", "{}", "not json", `["messages"]`, `{"messages":5}`, "null", `{"messages":["r"],"jobs":["r"],"channels":["r"],"settings":["r"]}`} {
		f.setMember(t, "member", perms)
		for _, tc := range toolMatrix {
			f.assertDenied(t, "perms "+perms+"/"+tc.name, f.direct(t, f.userID, tc.name, tc.args(f, f.tenantA)), "permission denied")
		}
	}
	for _, role := range []string{"guest", "", "OWNER", "Admin", "viewer"} {
		f.setMember(t, role, rightsJSON(allRights))
		for _, tc := range toolMatrix {
			f.assertDenied(t, "role "+role+"/"+tc.name, f.direct(t, f.userID, tc.name, tc.args(f, f.tenantA)), "permission denied")
		}
		f.assertDenied(t, "role "+role+"/get_tenant", f.direct(t, f.userID, "cqa_get_tenant", map[string]interface{}{"tenant_id": f.tenantA}), "permission denied")
	}
}

// Rights claimed in the arguments, and rights held in another tenant, never count.
func TestMCPSpoofedArgumentsAndCrossTenantRightsAreIgnored(t *testing.T) {
	f := newMCPFixture(t)
	f.setMember(t, "member", "{}")
	for _, tc := range toolMatrix {
		args := tc.args(f, f.tenantA)
		args["role"] = "owner"
		args["permissions"] = rightsJSON(allRights)
		args["user_id"] = pkg.NewUUID()
		f.assertDenied(t, "spoof/"+tc.name, f.direct(t, f.userID, tc.name, args), "permission denied")
	}
	// Full rights in A (and tenant C) do not reach tenant B or a nonexistent tenant.
	f.setMember(t, "member", rightsJSON(allRights))
	for _, tc := range toolMatrix {
		for _, other := range []string{f.tenantB, "r021-missing"} {
			f.assertDenied(t, "cross/"+other+"/"+tc.name, f.direct(t, f.userID, tc.name, tc.args(f, other)),
				"access denied: you don't have access to this tenant")
		}
		f.assertDenied(t, "outsider/"+tc.name, f.direct(t, f.outsiderID, tc.name, tc.args(f, f.tenantA)),
			"access denied: you don't have access to this tenant")
	}
	// A user with a zero-rights membership in B but full rights in A stays denied in B.
	f.exec(t, `INSERT INTO user_tenants (user_id, tenant_id, role, permissions) VALUES (?, ?, 'member', '{}')`, f.userID, f.tenantB)
	for _, tc := range toolMatrix {
		f.assertDenied(t, "weak B membership/"+tc.name, f.direct(t, f.userID, tc.name, tc.args(f, f.tenantB)), "permission denied")
	}
}

// get_tenant is membership only and returns id/name/slug, never the settings fixture.
func TestMCPGetTenantIsMembershipOnlyAndProjected(t *testing.T) {
	f := newMCPFixture(t)
	f.setMember(t, "member", "{}")
	got := f.direct(t, f.userID, "cqa_get_tenant", map[string]interface{}{"tenant_id": f.tenantA})
	if got.isErr || got.rpcErr != nil {
		t.Fatalf("membership alone must admit get_tenant: %+v", got)
	}
	var fields map[string]interface{}
	if err := json.Unmarshal([]byte(got.text), &fields); err != nil {
		t.Fatalf("unparseable response %q: %v", got.text, err)
	}
	keys := []string{}
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if strings.Join(keys, ",") != "id,name,slug" || fields["id"] != f.tenantA || fields["name"] != "Name-"+f.tenantA || fields["slug"] != f.tenantA {
		t.Fatalf("get_tenant fields = %v", fields)
	}
	if strings.Contains(got.text, forbiddenMarker) || strings.Contains(got.text, "settings") {
		t.Fatalf("get_tenant exposed tenant settings: %s", got.text)
	}
	f.assertDenied(t, "nonmember", f.direct(t, f.outsiderID, "cqa_get_tenant", map[string]interface{}{"tenant_id": f.tenantA}),
		"access denied: you don't have access to this tenant")
	f.assertDenied(t, "cross-tenant", f.direct(t, f.userID, "cqa_get_tenant", map[string]interface{}{"tenant_id": f.tenantB}),
		"access denied: you don't have access to this tenant")
}

// list_tenants: only the caller's recognized memberships, only id/name/slug, DB failure is an
// error and not an empty success.
func TestMCPListTenantsProjectionAndFailure(t *testing.T) {
	f := newMCPFixture(t)
	f.setMember(t, "member", "{}") // no data rights: list_tenants needs membership only
	got := f.direct(t, f.userID, "cqa_list_tenants", map[string]interface{}{})
	if got.isErr || got.rpcErr != nil {
		t.Fatalf("list_tenants: %+v", got)
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(got.text), &rows); err != nil {
		t.Fatalf("unparseable response %q: %v", got.text, err)
	}
	ids := []string{}
	for _, r := range rows {
		ids = append(ids, r["id"].(string))
		if len(r) != 3 || r["name"] == nil || r["slug"] == nil {
			t.Fatalf("row must be exactly id/name/slug: %v", r)
		}
	}
	sort.Strings(ids)
	if want := []string{f.tenantA, f.tenantC}; strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("tenants = %v, want %v (memberships only; %s must not appear)", ids, want, f.tenantB)
	}
	for _, banned := range []string{f.tenantB, forbiddenMarker, "settings", "count"} {
		if strings.Contains(got.text, banned) {
			t.Fatalf("list_tenants exposed %q: %s", banned, got.text)
		}
	}
	// No data tables beyond the membership/tenant lookups are touched (no per-tenant counts).
	if p := protectedTables(got.tables); strings.Join(p, ",") != "tenants" {
		t.Fatalf("list_tenants queried %v, want only tenants", p)
	}

	if out := f.direct(t, f.outsiderID, "cqa_list_tenants", nil); out.isErr || strings.TrimSpace(out.text) != "[]" {
		t.Fatalf("a user with no membership must get an empty list, got %+v", out)
	}
	// A membership with an unrecognized role is not listed.
	f.setMember(t, "guest", "{}")
	rolesOut := f.direct(t, f.userID, "cqa_list_tenants", nil)
	if strings.Contains(rolesOut.text, f.tenantA) || !strings.Contains(rolesOut.text, f.tenantC) {
		t.Fatalf("unrecognized-role membership must not be listed: %s", rolesOut.text)
	}

	// Injected failures: an error result, never an empty or partial success.
	for _, table := range []string{"user_tenants", "tenants"} {
		f.obs.reset()
		f.obs.failOn(table)
		params, _ := json.Marshal(ToolCallParams{Name: "cqa_list_tenants", Arguments: map[string]interface{}{}})
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("mcp_user_id", f.userID)
		res, rpcErr := handleToolsCall(c, params)
		tr, _ := res.(ToolResult)
		if rpcErr != nil || !tr.IsError || len(tr.Content) == 0 || tr.Content[0].Text != "Unable to list tenants" {
			t.Fatalf("failure on %s: res=%+v rpcErr=%v", table, res, rpcErr)
		}
		if strings.Contains(tr.Content[0].Text, secretSQLMarker) {
			t.Fatalf("failure on %s leaked diagnostics", table)
		}
	}
}

// A failing membership lookup denies instead of admitting, before any protected lookup, and
// leaks no SQL diagnostics to the response or the log.
func TestMCPMembershipLookupErrorFailsClosedWithoutLeaks(t *testing.T) {
	f := newMCPFixture(t)
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	for _, tc := range toolMatrix {
		f.obs.reset()
		f.obs.failOn("user_tenants")
		params, _ := json.Marshal(ToolCallParams{Name: tc.name, Arguments: tc.args(f, f.tenantA)})
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("mcp_user_id", f.userID)
		res, rpcErr := handleToolsCall(c, params)
		tr, _ := res.(ToolResult)
		out := toolOutcome{rpcErr: rpcErr, isErr: tr.IsError, tables: f.obs.seen()}
		if len(tr.Content) > 0 {
			out.text = tr.Content[0].Text
		}
		f.assertDenied(t, "lookup error/"+tc.name, out, "authorization unavailable")
	}
	if strings.Contains(logBuf.String(), secretSQLMarker) {
		t.Fatalf("log leaked SQL diagnostics: %s", logBuf.String())
	}
	if !strings.Contains(logBuf.String(), "membership_lookup_failed") {
		t.Fatalf("lookup failure should be logged by class: %q", logBuf.String())
	}
}

// Denial logs name user/tenant/tool/reason only; stored permission JSON never appears.
func TestMCPDenialLogsOmitPermissionJSON(t *testing.T) {
	f := newMCPFixture(t)
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	f.setMember(t, "member", `{"zz_log_probe":"R021-PERM-JSON","channels":"w"}`)
	f.assertDenied(t, "denied", f.direct(t, f.userID, "cqa_list_channels", map[string]interface{}{"tenant_id": f.tenantA}), "permission denied")
	got := logBuf.String()
	if !strings.Contains(got, "tool=cqa_list_channels") || !strings.Contains(got, "reason=permission_denied") {
		t.Fatalf("denial was not logged by class: %q", got)
	}
	for _, banned := range []string{"R021-PERM-JSON", "zz_log_probe", f.token} {
		if strings.Contains(got, banned) {
			t.Fatalf("log leaked %q: %s", banned, got)
		}
	}
}

// Unknown tools get the JSON-RPC unknown-tool error and no database access at all; missing
// tenant IDs and malformed params keep their bounded protocol errors.
func TestMCPUnknownToolMissingTenantAndMalformedParams(t *testing.T) {
	f := newMCPFixture(t)
	f.setMember(t, "owner", "")
	for _, name := range []string{"cqa_nope", "", "CQA_LIST_TENANTS", "cqa_list_tenants ", "cqa_trigger_job2"} {
		got := f.direct(t, f.userID, name, map[string]interface{}{"tenant_id": f.tenantA})
		if got.rpcErr == nil || got.rpcErr.Code != -32602 || got.rpcErr.Message != "Unknown tool: "+name {
			t.Fatalf("unknown tool %q: %+v", name, got)
		}
		if len(got.tables) != 0 {
			t.Fatalf("unknown tool %q reached the database: %v", name, got.tables)
		}
	}
	for _, tc := range toolMatrix {
		got := f.direct(t, f.userID, tc.name, map[string]interface{}{})
		f.assertDenied(t, "no tenant/"+tc.name, got, "tenant_id is required")
		if len(got.tables) != 0 {
			t.Fatalf("missing tenant_id reached the database: %v", got.tables)
		}
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("mcp_user_id", f.userID)
	if _, rpcErr := handleToolsCall(c, json.RawMessage(`{"name": 5}`)); rpcErr == nil || rpcErr.Code != -32602 || rpcErr.Message != "Invalid params" {
		t.Fatalf("malformed params: %v", rpcErr)
	}
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	res, _ := handleToolsCall(c2, json.RawMessage(`{"name":"cqa_list_channels","arguments":{"tenant_id":"`+f.tenantA+`"}}`))
	if tr, _ := res.(ToolResult); !tr.IsError || tr.Content[0].Text != "authentication required" {
		t.Fatalf("a call without an authenticated user must be refused, got %+v", res)
	}
}

// ---- mounted /mcp route with OAuth bearer tokens ----

func (f *mcpFixture) rpc(t *testing.T, token string, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	f.obs.reset()
	r := gin.New()
	SetupMCPRoutes(r)
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec, resp
}

func callBody(tool string, args map[string]interface{}) string {
	b, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]interface{}{"name": tool, "arguments": args},
	})
	return string(b)
}

func resultText(t *testing.T, resp map[string]interface{}) (string, bool) {
	t.Helper()
	res, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("no result in %v", resp)
	}
	content := res["content"].([]interface{})[0].(map[string]interface{})
	isErr, _ := res["isError"].(bool)
	return content["text"].(string), isErr
}

func TestMCPMountedRouteEnforcesTokenAndToolPermissions(t *testing.T) {
	f := newMCPFixture(t)
	read := toolMatrix[0]    // cqa_list_channels
	trigger := toolMatrix[9] // cqa_trigger_job
	if read.name != "cqa_list_channels" || trigger.name != "cqa_trigger_job" {
		t.Fatalf("matrix order changed: %s %s", read.name, trigger.name)
	}
	readBody := callBody(read.name, read.args(f, f.tenantA))
	triggerBody := callBody(trigger.name, trigger.args(f, f.tenantA))

	// Token admission: none, unknown and expired tokens stop at the bearer check with no tool or
	// tenant lookup (the only table touched is oauth_tokens).
	for label, tok := range map[string]string{"missing": "", "invalid": "tok-unknown", "expired": f.expiredToken} {
		rec, _ := f.rpc(t, tok, readBody)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s token: status %d, want 401", label, rec.Code)
		}
		for _, tb := range f.obs.seen() {
			if tb != tokenTable {
				t.Fatalf("%s token reached %s", label, tb)
			}
		}
		if strings.Contains(rec.Body.String(), forbiddenMarker) {
			t.Fatalf("%s token response leaked data", label)
		}
	}

	// Member without the rights: generic MCP error, no protected lookup, no job run.
	f.setMember(t, "member", permsExcept("channels", "r"))
	rec, resp := f.rpc(t, f.token, readBody)
	if text, isErr := resultText(t, resp); rec.Code != http.StatusOK || !isErr || text != "permission denied" {
		t.Fatalf("read denied: %d %v", rec.Code, resp)
	}
	if p := protectedTables(f.obs.seen()); len(p) != 1 || p[0] != tokenTable {
		t.Fatalf("denied read queried %v", p)
	}
	f.setMember(t, "member", permsExcept("jobs", "w"))
	rec, resp = f.rpc(t, f.token, triggerBody)
	if text, isErr := resultText(t, resp); !isErr || text != "permission denied" {
		t.Fatalf("trigger denied: %v", resp)
	}
	if p := protectedTables(f.obs.seen()); len(p) != 1 || p[0] != tokenTable {
		t.Fatalf("denied trigger looked up the job: %v", p)
	}
	if f.obs.count("jobs") != 0 {
		t.Fatalf("denied trigger queried jobs")
	}

	// Member with exact rights, then owner and admin with empty permissions, are admitted.
	for _, step := range []struct{ role, perms string }{
		{"member", permsOnly(trigger.needs)}, {"owner", ""}, {"admin", "not json"},
	} {
		f.setMember(t, step.role, step.perms)
		_, resp = f.rpc(t, f.token, triggerBody)
		text, isErr := resultText(t, resp)
		if !isErr || text != triggerUnavailableWant {
			t.Fatalf("%s trigger: %q isErr=%v, want the fixed unavailable error", step.role, text, isErr)
		}
		if f.obs.count("jobs") != 1 {
			t.Fatalf("%s: the permitted lookup was not observed (jobs queried %d times)", step.role, f.obs.count("jobs"))
		}
		// R035: the call reports unavailable and dispatches nothing.
		if n := f.jobRuns(t); n != 0 {
			t.Fatalf("%s: cqa_trigger_job created %d job runs; it must not dispatch", step.role, n)
		}
	}
	f.setMember(t, "member", permsOnly(read.needs))
	_, resp = f.rpc(t, f.token, readBody)
	if text, isErr := resultText(t, resp); isErr || !strings.Contains(text, "CH-"+forbiddenMarker+"-"+f.tenantA) || strings.Contains(text, f.tenantB) {
		t.Fatalf("permitted read: %q isErr=%v", text, isErr)
	}

	// Cross-tenant through the mounted route, even for an owner of A.
	f.setMember(t, "owner", "")
	for _, tc := range []toolCase{read, trigger} {
		_, resp = f.rpc(t, f.token, callBody(tc.name, tc.args(f, f.tenantB)))
		if text, isErr := resultText(t, resp); !isErr || text != "access denied: you don't have access to this tenant" {
			t.Fatalf("cross-tenant %s: %q", tc.name, text)
		}
		if f.obs.count("channels")+f.obs.count("jobs") != 0 {
			t.Fatalf("cross-tenant %s reached protected tables: %v", tc.name, f.obs.seen())
		}
	}

	// Unknown tool stays a JSON-RPC error even for an owner.
	_, resp = f.rpc(t, f.token, callBody("cqa_nope", map[string]interface{}{"tenant_id": f.tenantA}))
	if e, _ := resp["error"].(map[string]interface{}); e == nil || e["code"].(float64) != -32602 {
		t.Fatalf("unknown tool: %v", resp)
	}

	// tools/list and initialize remain metadata-only and need no tenant rights.
	f.setMember(t, "member", "{}")
	_, resp = f.rpc(t, f.token, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	tools := resp["result"].(map[string]interface{})["tools"].([]interface{})
	if len(tools) != 12 {
		t.Fatalf("tools/list returned %d tools, want 12", len(tools))
	}
	if p := protectedTables(f.obs.seen()); len(p) != 1 || p[0] != tokenTable {
		t.Fatalf("tools/list queried tenant data: %v", f.obs.seen())
	}
	for _, tl := range tools {
		d := tl.(map[string]interface{})["description"].(string)
		if tl.(map[string]interface{})["name"] == "cqa_list_tenants" && (strings.Contains(d, "stats") || strings.Contains(d, "conversations")) {
			t.Fatalf("list_tenants description still advertises counts: %s", d)
		}
		if tl.(map[string]interface{})["name"] == "cqa_get_tenant" && strings.Contains(d, "settings") {
			t.Fatalf("get_tenant description still advertises settings: %s", d)
		}
	}
	_, resp = f.rpc(t, f.token, `{"jsonrpc":"2.0","id":3,"method":"initialize"}`)
	if resp["result"] == nil {
		t.Fatalf("initialize failed: %v", resp)
	}
}
