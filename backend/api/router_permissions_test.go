package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	appapi "github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const r090JWTSecret = "r090-synthetic-test-secret-never-for-a-real-account"

type r090Rig struct {
	router    http.Handler
	tenantID  string
	ownerID   string
	adminID   string
	memberID  string
	unknownID string
	targetIDs map[string]string
	userIDs   []string
}

func newR090Rig(t *testing.T) *r090Rig {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Fatal("R090 route tests require TEST_DB_DSN; these tests do not permit a database skip")
	}
	if err := db.Connect(dsn, true); err != nil {
		t.Fatalf("connect admitted synthetic MySQL: %v", err)
	}
	t.Cleanup(db.Close)
	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("migrate admitted synthetic MySQL: %v", err)
	}
	middleware.SetJWTSecret(r090JWTSecret)
	ginModeForTests()
	r := &r090Rig{
		tenantID:  pkg.NewUUID(),
		ownerID:   pkg.NewUUID(),
		adminID:   pkg.NewUUID(),
		memberID:  pkg.NewUUID(),
		unknownID: pkg.NewUUID(),
		targetIDs: map[string]string{"member": pkg.NewUUID(), "admin": pkg.NewUUID(), "owner": pkg.NewUUID()},
	}
	cfg := &config.Config{Env: "test", RateLimitPerIP: 10000}
	r.router = appapi.SetupRouter(cfg)
	if err := db.DB.Create(&models.Tenant{ID: r.tenantID, Name: "R090 synthetic", Slug: "r090-" + r.tenantID, Settings: "{}", CreatedAt: time.Now(), UpdatedAt: time.Now()}).Error; err != nil {
		t.Fatalf("insert synthetic tenant: %v", err)
	}
	for _, actor := range []struct{ id, role, perms string }{
		{r.ownerID, "owner", "{}"},
		{r.adminID, "admin", "{}"},
		{r.memberID, "member", "{}"},
		{r.unknownID, "auditor", `{"channels":"r"}`},
	} {
		r.addUser(t, actor.id, actor.role, actor.perms)
	}
	for role, id := range r.targetIDs {
		r.addUser(t, id, role, "{}")
	}
	for _, id := range append([]string{r.ownerID, r.adminID, r.memberID, r.unknownID}, r.targetIDs["member"], r.targetIDs["admin"], r.targetIDs["owner"]) {
		r.userIDs = append(r.userIDs, id)
	}
	t.Cleanup(func() {
		for _, table := range []string{"notification_logs", "ai_usage_logs", "app_settings", "job_results", "job_runs", "jobs", "messages", "conversations", "channels", "user_tenants"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", r.tenantID)
		}
		if len(r.userIDs) > 0 {
			db.DB.Where("id IN ?", r.userIDs).Delete(&models.User{})
		}
		db.DB.Where("id = ?", r.tenantID).Delete(&models.Tenant{})
	})
	return r
}

func ginModeForTests() { gin.SetMode(gin.TestMode) }

func (r *r090Rig) addUser(t *testing.T, id, role, perms string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("OriginalPass9!"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash synthetic password: %v", err)
	}
	user := models.User{ID: id, Email: id + "@example.invalid", Name: "R090 synthetic", PasswordHash: string(hash), TokenVersion: 4, Language: "vi", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("insert synthetic user: %v", err)
	}
	ut := models.UserTenant{UserID: id, TenantID: r.tenantID, Role: role, Permissions: perms}
	if err := db.DB.Create(&ut).Error; err != nil {
		t.Fatalf("insert synthetic membership: %v", err)
	}
}

func (r *r090Rig) accessToken(t *testing.T, userID string) string {
	t.Helper()
	token, err := middleware.GenerateAccessToken(userID, userID+"@example.invalid", false)
	if err != nil {
		t.Fatalf("generate synthetic access token: %v", err)
	}
	return token
}

func (r *r090Rig) request(method, path, actor string, payload any, refreshCookie string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	if payload != nil {
		_ = json.NewEncoder(&body).Encode(payload)
	}
	req := httptest.NewRequest(method, path, &body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if actor != "" {
		token, _ := middleware.GenerateAccessToken(actor, actor+"@example.invalid", false)
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if refreshCookie != "" {
		req.AddCookie(&http.Cookie{Name: "cqa_refresh_token", Value: refreshCookie})
	}
	w := httptest.NewRecorder()
	r.router.ServeHTTP(w, req)
	return w
}

func (r *r090Rig) setPermissions(t *testing.T, userID, permissions string) {
	t.Helper()
	if err := db.DB.Model(&models.UserTenant{}).Where("user_id = ? AND tenant_id = ?", userID, r.tenantID).Update("permissions", permissions).Error; err != nil {
		t.Fatalf("set synthetic permissions: %v", err)
	}
}

func (r *r090Rig) addDashboardAndNotificationMarkers(t *testing.T) {
	t.Helper()
	now := time.Now()
	channelID := pkg.NewUUID()
	jobID := pkg.NewUUID()
	convID := pkg.NewUUID()
	runID := pkg.NewUUID()
	if err := db.DB.Create(&models.Channel{ID: channelID, TenantID: r.tenantID, ChannelType: "pancake", Name: "R090 synthetic channel", ExternalID: "r090-" + channelID, CredentialsEncrypted: []byte{0x01}, IsActive: true, Metadata: "{}", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("insert synthetic channel: %v", err)
	}
	if err := db.DB.Create(&models.Job{ID: jobID, TenantID: r.tenantID, Name: "R090-JOB-FORBIDDEN", JobType: "qc_analysis", InputChannelIDs: "[]", RulesConfig: "[]", Outputs: "[]", ScheduleType: "manual", IsActive: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("insert synthetic job: %v", err)
	}
	if err := db.DB.Create(&models.Conversation{ID: convID, TenantID: r.tenantID, ChannelID: channelID, ExternalConversationID: "r090-" + convID, CustomerName: "R090-MESSAGE-SYNTHETIC", LastMessageAt: &now, MessageCount: 1, Metadata: "{}", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("insert synthetic conversation: %v", err)
	}
	if err := db.DB.Create(&models.JobResult{ID: pkg.NewUUID(), JobRunID: runID, TenantID: r.tenantID, ConversationID: convID, ResultType: "qc_violation", Severity: "NGHIEM_TRONG", RuleName: "R090-JOB-FORBIDDEN", Evidence: "R090-FORBIDDEN-EVIDENCE", Detail: "{}", CreatedAt: now}).Error; err != nil {
		t.Fatalf("insert synthetic result: %v", err)
	}
	if err := db.DB.Create(&models.AIUsageLog{ID: pkg.NewUUID(), TenantID: r.tenantID, Provider: "synthetic", Model: "synthetic", InputTokens: 1, OutputTokens: 1, CostUSD: 87.65, CreatedAt: now}).Error; err != nil {
		t.Fatalf("insert synthetic usage row: %v", err)
	}
	if err := db.DB.Create(&models.AppSetting{ID: pkg.NewUUID(), TenantID: r.tenantID, SettingKey: "exchange_rate_vnd", ValuePlain: "31415", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("insert synthetic setting: %v", err)
	}
	if err := db.DB.Create(&models.NotificationLog{ID: pkg.NewUUID(), TenantID: r.tenantID, ChannelType: "email", Recipient: "synthetic@example.invalid", Subject: "R090-NOTIFICATION-FORBIDDEN", Body: "R090-NOTIFICATION-FORBIDDEN-BODY", Status: "failed", SentAt: now, CreatedAt: now}).Error; err != nil {
		t.Fatalf("insert synthetic notification log: %v", err)
	}
}

func keysOfJSON(t *testing.T, body []byte) map[string]bool {
	t.Helper()
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("decode JSON response: %v: %s", err, body)
	}
	keys := make(map[string]bool, len(data))
	for key := range data {
		keys[key] = true
	}
	return keys
}

func assertKeySet(t *testing.T, got map[string]bool, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("response keys = %v, want exactly %v", got, want)
		return
	}
	for _, key := range want {
		if !got[key] {
			t.Errorf("response is missing authorized key %q; got %v", key, got)
		}
	}
}

func captureR090Queries(t *testing.T, fn func() *httptest.ResponseRecorder) (*httptest.ResponseRecorder, map[string]int) {
	t.Helper()
	var mu sync.Mutex
	tables := map[string]int{}
	capture := func(tx *gorm.DB) {
		mu.Lock()
		tables[tx.Statement.Table]++
		mu.Unlock()
	}
	queryName := "r090_capture_queries"
	rowName := "r090_capture_rows"
	if err := db.DB.Callback().Query().After("gorm:query").Before("gorm:preload").Register(queryName, capture); err != nil {
		t.Fatalf("register query capture: %v", err)
	}
	if err := db.DB.Callback().Row().After("gorm:row").Register(rowName, capture); err != nil {
		db.DB.Callback().Query().Remove(queryName)
		t.Fatalf("register row capture: %v", err)
	}
	defer db.DB.Callback().Query().Remove(queryName)
	defer db.DB.Callback().Row().Remove(rowName)
	response := fn()
	mu.Lock()
	defer mu.Unlock()
	copy := make(map[string]int, len(tables))
	for table, count := range tables {
		copy[table] = count
	}
	return response, copy
}

func TestR090DashboardProjectsOnlyAuthorizedResourcesAndQueries(t *testing.T) {
	r := newR090Rig(t)
	r.addDashboardAndNotificationMarkers(t)
	const dashboardPathFmt = "/api/v1/tenants/%s/dashboard"
	cases := []struct {
		name, permissions string
		wantKeys          []string
		allowedTables     []string
		allowedMarkers    []string
		forbiddenMarkers  []string
	}{
		{"channels", `{"channels":"r"}`, []string{"active_channels"}, []string{"channels"}, nil, []string{"R090-JOB-FORBIDDEN", "R090-FORBIDDEN-EVIDENCE", "87.65", "31415"}},
		{"messages", `{"messages":"r"}`, []string{"total_conversations", "conversations_by_channel", "messages_by_day"}, []string{"conversations", "messages"}, nil, []string{"R090-JOB-FORBIDDEN", "R090-FORBIDDEN-EVIDENCE", "87.65", "31415"}},
		{"jobs", `{"jobs":"r"}`, []string{"active_jobs", "issues", "qc_violation_count", "qc_alerts", "classification_recent"}, []string{"jobs", "job_results"}, []string{"R090-JOB-FORBIDDEN", "R090-FORBIDDEN-EVIDENCE"}, []string{"87.65", "31415"}},
		{"settings", `{"settings":"r"}`, []string{"cost_period", "cost_today", "cost_this_month", "cost_by_day", "exchange_rate"}, []string{"ai_usage_logs", "app_settings"}, []string{"87.65", "31415"}, []string{"R090-JOB-FORBIDDEN", "R090-FORBIDDEN-EVIDENCE"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r.setPermissions(t, r.memberID, tc.permissions)
			path := fmt.Sprintf(dashboardPathFmt, r.tenantID)
			response, tables := captureR090Queries(t, func() *httptest.ResponseRecorder {
				return r.request(http.MethodGet, path, r.memberID, nil, "")
			})
			if response.Code != http.StatusOK {
				t.Fatalf("dashboard status=%d body=%s", response.Code, response.Body.String())
			}
			assertKeySet(t, keysOfJSON(t, response.Body.Bytes()), tc.wantKeys)
			for _, marker := range tc.forbiddenMarkers {
				if strings.Contains(response.Body.String(), marker) {
					t.Errorf("restricted marker %q leaked in %s dashboard: %s", marker, tc.name, response.Body.String())
				}
			}
			for _, marker := range tc.allowedMarkers {
				if !strings.Contains(response.Body.String(), marker) {
					t.Errorf("authorized marker %q missing from %s dashboard: %s", marker, tc.name, response.Body.String())
				}
			}
			allowed := map[string]bool{"user_tenants": true}
			for _, table := range tc.allowedTables {
				allowed[table] = true
			}
			for table, count := range tables {
				if count > 0 && !allowed[table] {
					t.Errorf("%s dashboard issued unauthorized query against %q (%d queries); observed=%v", tc.name, table, count, tables)
				}
			}
		})
	}
	for _, tc := range []struct {
		name, role, permissions string
	}{
		{"no-rights member", "member", "{}"},
		{"malformed member permissions", "member", "{bad"},
		{"unknown role with valid member-like right", "auditor", `{"channels":"r"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actor := r.memberID
			if tc.role == "auditor" {
				actor = r.unknownID
			} else {
				r.setPermissions(t, actor, tc.permissions)
			}
			response := r.request(http.MethodGet, fmt.Sprintf(dashboardPathFmt, r.tenantID), actor, nil, "")
			if response.Code != http.StatusForbidden {
				t.Errorf("dashboard status=%d, want 403 fail-closed; body=%s", response.Code, response.Body.String())
			}
		})
	}
	for _, actor := range []string{r.ownerID, r.adminID} {
		response := r.request(http.MethodGet, fmt.Sprintf(dashboardPathFmt, r.tenantID), actor, nil, "")
		if response.Code != http.StatusOK {
			t.Errorf("%s dashboard status=%d, want owner/admin control 200", actor, response.Code)
		}
		assertKeySet(t, keysOfJSON(t, response.Body.Bytes()), []string{"total_conversations", "active_channels", "active_jobs", "issues", "qc_violation_count", "conversations_by_channel", "qc_alerts", "classification_recent", "cost_period", "cost_today", "cost_this_month", "cost_by_day", "messages_by_day", "exchange_rate"})
	}
}

func TestR090NotificationLogsRequireSettingsReadAtRegisteredRoute(t *testing.T) {
	r := newR090Rig(t)
	r.addDashboardAndNotificationMarkers(t)
	path := fmt.Sprintf("/api/v1/tenants/%s/notification-logs", r.tenantID)
	r.setPermissions(t, r.memberID, `{"jobs":"r"}`)
	denied := r.request(http.MethodGet, path, r.memberID, nil, "")
	if denied.Code != http.StatusForbidden {
		t.Errorf("jobs-only member notification logs status=%d, want 403; body=%s", denied.Code, denied.Body.String())
	}
	if strings.Contains(denied.Body.String(), "R090-NOTIFICATION-FORBIDDEN") {
		t.Errorf("notification marker leaked to jobs-only member: %s", denied.Body.String())
	}
	r.setPermissions(t, r.memberID, `{"settings":"r"}`)
	allowed := r.request(http.MethodGet, path, r.memberID, nil, "")
	if allowed.Code != http.StatusOK || !strings.Contains(allowed.Body.String(), "R090-NOTIFICATION-FORBIDDEN-BODY") {
		t.Errorf("settings:r member control failed: status=%d body=%s", allowed.Code, allowed.Body.String())
	}
	r.setPermissions(t, r.unknownID, `{"settings":"r"}`)
	unknown := r.request(http.MethodGet, path, r.unknownID, nil, "")
	if unknown.Code != http.StatusForbidden || strings.Contains(unknown.Body.String(), "R090-NOTIFICATION-FORBIDDEN") {
		t.Errorf("unknown-role settings:r route must fail closed: status=%d body=%s", unknown.Code, unknown.Body.String())
	}
	for _, actor := range []string{r.ownerID, r.adminID} {
		control := r.request(http.MethodGet, path, actor, nil, "")
		if control.Code != http.StatusOK {
			t.Errorf("owner/admin notification-log control status=%d", control.Code)
		}
	}
}

func readUser(t *testing.T, id string) models.User {
	t.Helper()
	var user models.User
	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		t.Fatalf("read synthetic user %s: %v", id, err)
	}
	return user
}

func resetRequest(r *r090Rig, actor, target string) *httptest.ResponseRecorder {
	return r.request(http.MethodPut, fmt.Sprintf("/api/v1/tenants/%s/users/%s/reset-password", r.tenantID, target), actor, map[string]string{"password": "ResetPass9!"}, "")
}

func TestR090AdminResetTargetHierarchyAndOwnerControls(t *testing.T) {
	r := newR090Rig(t)
	for _, role := range []string{"owner", "admin"} {
		target := r.targetIDs[role]
		before := readUser(t, target)
		response := resetRequest(r, r.adminID, target)
		if response.Code != http.StatusForbidden {
			t.Errorf("admin reset %s status=%d, want 403 hierarchy denial; body=%s", role, response.Code, response.Body.String())
		}
		after := readUser(t, target)
		if after.PasswordHash != before.PasswordHash || after.TokenVersion != before.TokenVersion {
			t.Errorf("denied admin reset changed protected %s: hashChanged=%v tokenVersion=%d->%d", role, after.PasswordHash != before.PasswordHash, before.TokenVersion, after.TokenVersion)
		}
	}
	memberTarget := r.targetIDs["member"]
	memberBefore := readUser(t, memberTarget)
	adminMemberReset := resetRequest(r, r.adminID, memberTarget)
	if adminMemberReset.Code != http.StatusOK {
		t.Errorf("admin reset member status=%d body=%s, want allowed 200", adminMemberReset.Code, adminMemberReset.Body.String())
	}
	memberAfter := readUser(t, memberTarget)
	if memberAfter.TokenVersion != memberBefore.TokenVersion+1 {
		t.Errorf("admin reset member version=%d, want one increment from %d", memberAfter.TokenVersion, memberBefore.TokenVersion)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(memberAfter.PasswordHash), []byte("ResetPass9!")); err != nil {
		t.Errorf("admin reset member did not persist password: %v", err)
	}
	for _, role := range []string{"member", "admin", "owner"} {
		target := r.targetIDs[role]
		before := readUser(t, target)
		response := resetRequest(r, r.ownerID, target)
		if response.Code != http.StatusOK {
			t.Errorf("owner reset %s status=%d body=%s, want 200", role, response.Code, response.Body.String())
		}
		after := readUser(t, target)
		if after.TokenVersion != before.TokenVersion+1 {
			t.Errorf("owner reset %s token version=%d, want one increment from %d", role, after.TokenVersion, before.TokenVersion)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(after.PasswordHash), []byte("ResetPass9!")); err != nil {
			t.Errorf("owner reset %s did not persist new password: %v", role, err)
		}
	}
	outsiderID := pkg.NewUUID()
	hash, _ := bcrypt.GenerateFromPassword([]byte("OriginalPass9!"), bcrypt.MinCost)
	if err := db.DB.Create(&models.User{ID: outsiderID, Email: outsiderID + "@example.invalid", PasswordHash: string(hash), TokenVersion: 3, CreatedAt: time.Now(), UpdatedAt: time.Now()}).Error; err != nil {
		t.Fatalf("insert nonmember target: %v", err)
	}
	t.Cleanup(func() { db.DB.Delete(&models.User{}, "id = ?", outsiderID) })
	response := resetRequest(r, r.adminID, outsiderID)
	if response.Code != http.StatusNotFound {
		t.Errorf("nonmember reset status=%d, want 404; body=%s", response.Code, response.Body.String())
	}
	if after := readUser(t, outsiderID); after.PasswordHash != string(hash) || after.TokenVersion != 3 {
		t.Errorf("nonmember reset changed password or version")
	}
}

func TestR090ResetLocksTargetRoleAgainstConcurrentPromotion(t *testing.T) {
	r := newR090Rig(t)
	target := r.targetIDs["member"]
	ready := make(chan struct{}, 1)
	release := make(chan struct{})
	var claimed atomic.Bool
	name := "r090_reset_target_read_barrier"
	if err := db.DB.Callback().Query().After("gorm:query").Before("gorm:preload").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table != "user_tenants" || !containsR090ID(tx.Statement.Vars, target) {
			return
		}
		if claimed.CompareAndSwap(false, true) {
			ready <- struct{}{}
			<-release
		}
	}); err != nil {
		t.Fatalf("register reset race barrier: %v", err)
	}
	defer db.DB.Callback().Query().Remove(name)
	resetDone := make(chan *httptest.ResponseRecorder, 1)
	go func() { resetDone <- resetRequest(r, r.adminID, target) }()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("reset never reached the target membership read")
	}
	promoteDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		promoteDone <- r.request(http.MethodPut, fmt.Sprintf("/api/v1/tenants/%s/users/%s/role", r.tenantID, target), r.ownerID, map[string]string{"role": "admin"}, "")
	}()
	promotionWonBeforeResetFinished := false
	var earlyPromotion *httptest.ResponseRecorder
	select {
	case earlyPromotion = <-promoteDone:
		promotionWonBeforeResetFinished = true
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	resetResponse := <-resetDone
	var promotionResponse *httptest.ResponseRecorder
	if promotionWonBeforeResetFinished {
		promotionResponse = earlyPromotion
	} else {
		select {
		case promotionResponse = <-promoteDone:
		case <-time.After(5 * time.Second):
			t.Fatal("role promotion did not finish after reset transaction released its lock")
		}
	}
	if promotionWonBeforeResetFinished {
		t.Errorf("role promotion completed while reset was paused after target-role read; authorization was not serialized")
	}
	if resetResponse.Code != http.StatusOK {
		t.Errorf("reset ordered before promotion status=%d body=%s, want serialized 200", resetResponse.Code, resetResponse.Body.String())
	}
	if promotionResponse.Code != http.StatusOK {
		t.Errorf("owner promotion control status=%d body=%s", promotionResponse.Code, promotionResponse.Body.String())
	}
	var membership models.UserTenant
	if err := db.DB.First(&membership, "user_id = ? AND tenant_id = ?", target, r.tenantID).Error; err != nil {
		t.Fatalf("read promoted target: %v", err)
	}
	if membership.Role != "admin" {
		t.Errorf("target role=%q, want promoted admin", membership.Role)
	}
	if got := readUser(t, target).TokenVersion; got != 5 {
		t.Errorf("serialized reset token version=%d, want 5", got)
	}
}

func containsR090ID(values []any, id string) bool {
	for _, value := range values {
		if fmt.Sprint(value) == id {
			return true
		}
	}
	return false
}

func makeRefreshCookie(t *testing.T, userID string, version int) string {
	t.Helper()
	token, err := middleware.GenerateRefreshToken(userID, version)
	if err != nil {
		t.Fatalf("generate synthetic refresh token: %v", err)
	}
	return token
}

func successfulRefreshCookies(response *httptest.ResponseRecorder) int {
	count := 0
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "cqa_refresh_token" && cookie.Value != "" && cookie.MaxAge > 0 {
			count++
		}
	}
	return count
}

func TestR090RefreshCookieHasExactlyOneConcurrentWinner(t *testing.T) {
	r := newR090Rig(t)
	user := readUser(t, r.memberID)
	const requests = 2
	ready := make(chan struct{}, requests)
	release := make(chan struct{})
	var reached atomic.Int32
	name := "r090_refresh_read_barrier"
	if err := db.DB.Callback().Query().After("gorm:query").Before("gorm:preload").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table != "users" || !containsR090ID(tx.Statement.Vars, user.ID) {
			return
		}
		if reached.Add(1) <= requests {
			ready <- struct{}{}
			<-release
		}
	}); err != nil {
		t.Fatalf("register refresh race barrier: %v", err)
	}
	defer db.DB.Callback().Query().Remove(name)
	cookie := makeRefreshCookie(t, user.ID, user.TokenVersion)
	results := make(chan *httptest.ResponseRecorder, requests)
	for i := 0; i < requests; i++ {
		go func() { results <- r.request(http.MethodPost, "/api/v1/auth/refresh", "", nil, cookie) }()
	}
	for i := 0; i < requests; i++ {
		select {
		case <-ready:
		case <-time.After(5 * time.Second):
			close(release)
			t.Fatalf("only %d refresh requests read the same token version", i)
		}
	}
	close(release)
	responses := []*httptest.ResponseRecorder{<-results, <-results}
	winners := 0
	for _, response := range responses {
		if response.Code == http.StatusOK {
			winners++
			if successfulRefreshCookies(response) != 1 {
				t.Errorf("successful refresh must set exactly one successor cookie; cookies=%v", response.Result().Cookies())
			}
		} else if response.Code != http.StatusUnauthorized {
			t.Errorf("refresh loser status=%d body=%s, want 401", response.Code, response.Body.String())
		} else if successfulRefreshCookies(response) != 0 {
			t.Errorf("refresh loser returned a successful successor cookie: %v", response.Result().Cookies())
		}
	}
	if winners != 1 {
		t.Errorf("same-cookie concurrent refresh winners=%d, want exactly one", winners)
	}
	if got := readUser(t, user.ID).TokenVersion; got != user.TokenVersion+1 {
		t.Errorf("refresh version=%d, want one increment from %d", got, user.TokenVersion)
	}
}

func TestR090ResetWinsAgainstRefreshAlreadyReadAtOldVersion(t *testing.T) {
	r := newR090Rig(t)
	user := readUser(t, r.memberID)
	ready := make(chan struct{}, 1)
	release := make(chan struct{})
	var claimed atomic.Bool
	name := "r090_refresh_reset_race_barrier"
	if err := db.DB.Callback().Query().After("gorm:query").Before("gorm:preload").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == "users" && containsR090ID(tx.Statement.Vars, user.ID) {
			if claimed.CompareAndSwap(false, true) {
				ready <- struct{}{}
				<-release
			}
		}
	}); err != nil {
		t.Fatalf("register reset-refresh barrier: %v", err)
	}
	defer db.DB.Callback().Query().Remove(name)
	refreshDone := make(chan *httptest.ResponseRecorder, 1)
	cookie := makeRefreshCookie(t, user.ID, user.TokenVersion)
	go func() { refreshDone <- r.request(http.MethodPost, "/api/v1/auth/refresh", "", nil, cookie) }()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("refresh did not read the pre-reset token version")
	}
	for n := 0; n < 2; n++ {
		reset := resetRequest(r, r.ownerID, user.ID)
		if reset.Code != http.StatusOK {
			close(release)
			t.Fatalf("password reset %d status=%d body=%s", n+1, reset.Code, reset.Body.String())
		}
	}
	close(release)
	refresh := <-refreshDone
	if refresh.Code != http.StatusUnauthorized {
		t.Errorf("refresh racing password reset status=%d, want 401; body=%s", refresh.Code, refresh.Body.String())
	}
	if successfulRefreshCookies(refresh) != 0 {
		t.Errorf("stale refresh produced a successor cookie after reset: %v", refresh.Result().Cookies())
	}
	if got := readUser(t, user.ID).TokenVersion; got != user.TokenVersion+2 {
		t.Errorf("two resets token version=%d, want exactly %d after stale refresh loses", got, user.TokenVersion+2)
	}
}

func TestR090RefreshDatabaseErrorCannotMintTokensOrCookie(t *testing.T) {
	r := newR090Rig(t)
	user := readUser(t, r.memberID)
	trigger := "r090_fail_refresh_" + strings.ReplaceAll(user.ID, "-", "")[:12]
	if err := db.DB.Exec(fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE ON users FOR EACH ROW BEGIN IF OLD.id = '%s' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'synthetic R090 update failure'; END IF; END", trigger, user.ID)).Error; err != nil {
		t.Fatalf("create synthetic failure trigger: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trigger) })
	response := r.request(http.MethodPost, "/api/v1/auth/refresh", "", nil, makeRefreshCookie(t, user.ID, user.TokenVersion))
	if response.Code != http.StatusInternalServerError {
		t.Errorf("refresh update error status=%d, want 500; body=%s", response.Code, response.Body.String())
	}
	if successfulRefreshCookies(response) != 0 {
		t.Errorf("failed refresh update returned successor cookie: %v", response.Result().Cookies())
	}
	var tokenBody map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &tokenBody); err == nil && tokenBody["access_token"] != nil {
		t.Errorf("failed refresh update returned an access token: %s", response.Body.String())
	}
	if got := readUser(t, user.ID).TokenVersion; got != user.TokenVersion {
		t.Errorf("failed refresh update changed version to %d, want %d", got, user.TokenVersion)
	}
}
