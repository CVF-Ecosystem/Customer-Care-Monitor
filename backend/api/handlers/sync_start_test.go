package handlers

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-007: a manual sync is acknowledged only after its tenant-scoped
// "syncing" state is persisted. The worker launcher is stubbed, so no channel
// adapter, credential or provider is ever called.

type syncStartFixture struct {
	tenantID, otherTenantID, channelID string
	launches                           int
	statusAtLaunch                     string
	respondedBeforeLaunch              bool
	rec                                *httptest.ResponseRecorder

	// CCMAI-RUNTIME-008: the stubbed config loader returns cfg (or cfgErr) and
	// counts calls; the stubbed launcher records the config it received.
	cfg         *config.Config
	cfgErr      error
	cfgLoads    int
	launchedCfg *config.Config
}

func setupSyncStartFixture(t *testing.T) *syncStartFixture {
	t.Helper()
	connectChannelsTestDB(t)
	s := pkg.NewUUID()[:8]
	f := &syncStartFixture{tenantID: "syncst-" + s, otherTenantID: "syncst-other-" + s, channelID: "ch-syncst-" + s}
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	for _, tenant := range []string{f.tenantID, f.otherTenantID} {
		exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Sync Start', ?, '{}', NOW(), NOW())`, tenant, tenant)
	}
	exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, last_sync_status, last_sync_error, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh', 'fake', X'00', true, 'success', '', '{}', NOW(), NOW())`,
		f.channelID, f.tenantID)
	t.Cleanup(func() {
		db.DB.Exec("DELETE FROM channels WHERE id = ?", f.channelID)
		for _, tenant := range []string{f.tenantID, f.otherTenantID} {
			db.DB.Exec("DELETE FROM activity_logs WHERE tenant_id = ?", tenant)
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenant)
		}
	})

	original := startManualSync
	startManualSync = func(_ string, ch models.Channel, cfg *config.Config) {
		f.launches++
		f.launchedCfg = cfg
		f.statusAtLaunch = f.channelStatus(t).LastSyncStatus
		f.respondedBeforeLaunch = f.rec != nil && f.rec.Body.Len() > 0
	}
	t.Cleanup(func() { startManualSync = original })

	f.cfg = &config.Config{Env: "test"} // synthetic; no real secrets or credentials
	originalLoad := loadManualSyncConfig
	loadManualSyncConfig = func() (*config.Config, error) {
		f.cfgLoads++
		if f.cfgErr != nil {
			return nil, f.cfgErr
		}
		return f.cfg, nil
	}
	t.Cleanup(func() { loadManualSyncConfig = originalLoad })
	return f
}

func (f *syncStartFixture) channelStatus(t *testing.T) models.Channel {
	t.Helper()
	var ch models.Channel
	if err := db.DB.Where("id = ?", f.channelID).First(&ch).Error; err != nil {
		t.Fatalf("load channel: %v", err)
	}
	return ch
}

func (f *syncStartFixture) callSync(tenantID string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	f.rec = httptest.NewRecorder()
	c, _ := gin.CreateTestContext(f.rec)
	c.Set("tenant_id", tenantID)
	c.Params = gin.Params{{Key: "channelId", Value: f.channelID}}
	c.Request = httptest.NewRequest("POST", "/api/v1/channels/"+f.channelID+"/sync", nil)
	SyncChannelNow(c)
	return f.rec
}

// addChannelTrigger installs a BEFORE UPDATE trigger for this fixture's channel
// only. The channel ID is a generated alphanumeric string, safe to inline
// because a trigger body cannot bind parameters.
func (f *syncStartFixture) addChannelTrigger(t *testing.T, body string) {
	t.Helper()
	name := "trg_syncst_" + strings.ReplaceAll(f.channelID[len("ch-syncst-"):], "-", "")
	if err := db.DB.Exec(fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE ON channels FOR EACH ROW BEGIN IF OLD.id = '%s' THEN %s END IF; END", name, f.channelID, body)).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + name) })
}

func (f *syncStartFixture) assertNoStart(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code < 400 {
		t.Fatalf("status %d, want non-2xx; body %s", rec.Code, rec.Body.String())
	}
	if f.launches != 0 {
		t.Fatalf("worker launched %d times despite no recorded start", f.launches)
	}
	if got := f.channelStatus(t).LastSyncStatus; got != "success" {
		t.Fatalf("visible status changed to %q", got)
	}
}

func TestSyncChannelNowRecordsStartBeforeAcknowledging(t *testing.T) {
	f := setupSyncStartFixture(t)
	rec := f.callSync(f.tenantID)

	if rec.Code != http.StatusAccepted || strings.TrimSpace(rec.Body.String()) != `{"message":"sync_started"}` {
		t.Fatalf("got %d %s, want 202 sync_started", rec.Code, rec.Body.String())
	}
	if f.launches != 1 {
		t.Fatalf("worker launched %d times, want 1", f.launches)
	}
	if f.statusAtLaunch != "syncing" || f.respondedBeforeLaunch {
		t.Fatalf("at launch: status %q, responded already %v; want persisted syncing before any response", f.statusAtLaunch, f.respondedBeforeLaunch)
	}
	if ch := f.channelStatus(t); ch.LastSyncStatus != "syncing" || ch.LastSyncError != "" {
		t.Fatalf("persisted status %q / %q", ch.LastSyncStatus, ch.LastSyncError)
	}
}

func TestSyncChannelNowWriteFailureStartsNoWorker(t *testing.T) {
	f := setupSyncStartFixture(t)
	f.addChannelTrigger(t, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced syncing write failure';")
	rec := f.callSync(f.tenantID)

	f.assertNoStart(t, rec)
	body := rec.Body.String()
	if strings.TrimSpace(body) != `{"error":"sync_start_failed"}` || strings.Contains(body, "forced") || strings.Contains(body, "Error") {
		t.Fatalf("response leaks or differs: %s", body)
	}
}

// A write that matches the row but changes nothing reports zero affected rows;
// it must not be treated as a recorded start.
func TestSyncChannelNowZeroRowUpdateStartsNoWorker(t *testing.T) {
	f := setupSyncStartFixture(t)
	f.addChannelTrigger(t, "SET NEW.last_sync_status = OLD.last_sync_status; SET NEW.last_sync_error = OLD.last_sync_error; SET NEW.updated_at = OLD.updated_at;")
	f.assertNoStart(t, f.callSync(f.tenantID))
}

func TestSyncChannelNowOtherTenantCannotStart(t *testing.T) {
	f := setupSyncStartFixture(t)
	rec := f.callSync(f.otherTenantID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("other tenant got %d, want 404", rec.Code)
	}
	f.assertNoStart(t, rec)
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

const panicSecret = "access_token=SECRET-DO-NOT-LEAK"

func TestHandleManualSyncPanicRecordsBoundedTenantScopedStatus(t *testing.T) {
	f := setupSyncStartFixture(t)
	logs := captureLog(t)

	if err := handleManualSyncPanic(f.tenantID, f.channelID, panicSecret); err != nil {
		t.Fatalf("recording panic status: %v", err)
	}
	ch := f.channelStatus(t)
	if ch.LastSyncStatus != "error" || ch.LastSyncError != manualSyncPanicMessage {
		t.Fatalf("persisted %q / %q", ch.LastSyncStatus, ch.LastSyncError)
	}
	if strings.Contains(ch.LastSyncError, "SECRET") || strings.Contains(logs.String(), "SECRET") {
		t.Fatalf("raw panic value leaked; log: %s", logs.String())
	}
	if !strings.Contains(logs.String(), f.channelID) || !strings.Contains(logs.String(), "(string)") {
		t.Fatalf("log should name the channel and panic type: %s", logs.String())
	}
}

func TestHandleManualSyncPanicWrongTenantIsNotSilent(t *testing.T) {
	f := setupSyncStartFixture(t)
	logs := captureLog(t)

	if err := handleManualSyncPanic(f.otherTenantID, f.channelID, panicSecret); err == nil {
		t.Fatal("another tenant's recovery reported success")
	}
	if got := f.channelStatus(t).LastSyncStatus; got != "success" {
		t.Fatalf("another tenant changed the channel to %q", got)
	}
	if !strings.Contains(logs.String(), "not recorded") || strings.Contains(logs.String(), "SECRET") {
		t.Fatalf("failure not logged safely: %s", logs.String())
	}
}

func TestHandleManualSyncPanicWriteFailureIsReturnedAndLogged(t *testing.T) {
	f := setupSyncStartFixture(t)
	f.addChannelTrigger(t, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced panic status write failure';")
	logs := captureLog(t)

	if err := handleManualSyncPanic(f.tenantID, f.channelID, panicSecret); err == nil {
		t.Fatal("failed status write reported success")
	}
	if !strings.Contains(logs.String(), "not recorded") || strings.Contains(logs.String(), "SECRET") {
		t.Fatalf("failure not logged safely: %s", logs.String())
	}
	if got := f.channelStatus(t).LastSyncStatus; got != "success" {
		t.Fatalf("status changed to %q despite failed write", got)
	}
}
