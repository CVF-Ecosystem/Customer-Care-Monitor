package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-018 (handlers): the fresh demo import marks its channels with
// the server-owned marker, the marker is invisible and unforgeable through the
// channel API, and manual sync of a marked fixture is refused with 409 before
// configuration loading or worker dispatch. The worker launcher is a stub and
// no adapter, credential or provider is ever called.

func setupDemoFixtureTenant(t *testing.T) (tenantID string) {
	t.Helper()
	connectChannelsTestDB(t)
	tenantID = "fxsync-" + pkg.NewUUID()[:8]
	if err := db.DB.Exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Fixture Sync', ?, '{}', NOW(), NOW())`, tenantID, tenantID).Error; err != nil {
		t.Fatalf("fixture tenant: %v", err)
	}
	t.Cleanup(func() {
		for _, table := range []string{"job_results", "analysis_snapshots", "ai_usage_logs", "messages", "conversations", "job_runs", "jobs", "channels", "activity_logs"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenantID)
		}
		db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
	})
	return tenantID
}

func demoImport(t *testing.T, tenantID string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenantID)
	c.Request = httptest.NewRequest("POST", "/api/v1/demo/import", nil)
	ImportDemoData(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("ImportDemoData status %d, body %s", rec.Code, rec.Body.String())
	}
}

func callChannelHandler(handler gin.HandlerFunc, method, tenantID, channelID, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenantID)
	c.Set("user_id", "u-test")
	c.Set("user_email", "test@example.com")
	c.Params = gin.Params{{Key: "channelId", Value: channelID}}
	c.Request = httptest.NewRequest(method, "/api/v1/channels/"+channelID, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	return rec
}

type fxChannelRow struct {
	Marked                         bool
	Status, Error, RunID, External string
}

func readFxChannel(t *testing.T, id string) fxChannelRow {
	t.Helper()
	var r struct {
		IsDemoFixture                            bool
		LastSyncStatus, LastSyncError, SyncRunID *string
		ExternalID                               string
	}
	if err := db.DB.Raw("SELECT is_demo_fixture, last_sync_status, last_sync_error, sync_run_id, external_id FROM channels WHERE id = ?", id).Scan(&r).Error; err != nil {
		t.Fatal(err)
	}
	s := func(p *string) string {
		if p == nil {
			return "<NULL>"
		}
		return *p
	}
	return fxChannelRow{r.IsDemoFixture, s(r.LastSyncStatus), s(r.LastSyncError), s(r.SyncRunID), r.ExternalID}
}

func TestFreshDemoImportMarksBothFixturesAndTheMarkerIsNotExposedOrForgeable(t *testing.T) {
	tenantID := setupDemoFixtureTenant(t)
	demoImport(t, tenantID)

	var ids []string
	if err := db.DB.Raw("SELECT id FROM channels WHERE tenant_id = ? ORDER BY external_id", tenantID).Scan(&ids).Error; err != nil || len(ids) != 2 {
		t.Fatalf("demo import created %d channels (err %v), want 2", len(ids), err)
	}
	for _, id := range ids {
		row := readFxChannel(t, id)
		if !row.Marked || (row.Status != "" && row.Status != "<NULL>") {
			t.Fatalf("fresh fixture %s: %+v, want marked and never synced", row.External, row)
		}
	}

	// The marker is not serialized through the channel API ...
	rec := callChannelHandler(GetChannel, "GET", tenantID, ids[0], "")
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "demo_fixture") {
		t.Fatalf("GetChannel %d leaks the marker or fails: %s", rec.Code, rec.Body.String())
	}
	// ... and cannot be changed through update input or metadata.
	for _, body := range []string{
		`{"is_demo_fixture":false,"name":"Renamed fixture"}`,
		`{"metadata":"{\"is_demo_fixture\":false,\"demo\":false}"}`,
	} {
		if rec := callChannelHandler(UpdateChannel, "PUT", tenantID, ids[0], body); rec.Code != http.StatusOK {
			t.Fatalf("UpdateChannel %s: %d %s", body, rec.Code, rec.Body.String())
		}
		if !readFxChannel(t, ids[0]).Marked {
			t.Fatalf("update %s cleared the marker", body)
		}
	}
}

func TestManualSyncRefusesMarkedFixtureBeforeConfigAndDispatch(t *testing.T) {
	tenantID := setupDemoFixtureTenant(t)
	demoImport(t, tenantID)
	var fx string
	if err := db.DB.Raw("SELECT id FROM channels WHERE tenant_id = ? AND external_id = 'demo-zalo-oa'", tenantID).Scan(&fx).Error; err != nil || fx == "" {
		t.Fatalf("fixture lookup: %q %v", fx, err)
	}
	// A real channel in the same tenant, created as the API would create it.
	realID := "ch-fxreal-" + pkg.NewUUID()[:8]
	if err := db.DB.Exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, last_sync_status, last_sync_error, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh that', 'real-page', X'00', true, 'success', '', '{}', NOW(), NOW())`, realID, tenantID).Error; err != nil {
		t.Fatal(err)
	}
	if readFxChannel(t, realID).Marked {
		t.Fatal("a real channel in a demo tenant must not be marked")
	}

	loads, launches := 0, 0
	origLoad, origStart := loadManualSyncConfig, startManualSync
	loadManualSyncConfig = func() (*config.Config, error) { loads++; return &config.Config{Env: "test"}, nil }
	startManualSync = func(engine.SyncReservation, models.Channel, *config.Config) { launches++ }
	t.Cleanup(func() { loadManualSyncConfig, startManualSync = origLoad, origStart })

	before := readFxChannel(t, fx)
	rec := callChannelHandler(SyncChannelNow, "POST", tenantID, fx, "")
	if rec.Code != http.StatusConflict || strings.TrimSpace(rec.Body.String()) != `{"error":"demo_channel_not_syncable"}` {
		t.Fatalf("marked fixture: %d %s, want 409 demo_channel_not_syncable", rec.Code, rec.Body.String())
	}
	if loads != 0 || launches != 0 {
		t.Fatalf("config loaded %d times and worker launched %d times for a refused fixture", loads, launches)
	}
	if after := readFxChannel(t, fx); after != before {
		t.Fatalf("fixture row changed: %+v -> %+v", before, after)
	}

	// Positive detector: the real channel in the same tenant is still admitted.
	rec = callChannelHandler(SyncChannelNow, "POST", tenantID, realID, "")
	if rec.Code != http.StatusAccepted || loads != 1 || launches != 1 {
		t.Fatalf("real channel: %d %s (loads %d, launches %d), want 202 with one launch", rec.Code, rec.Body.String(), loads, launches)
	}
	if got := readFxChannel(t, realID); got.Status != "syncing" || got.RunID == "<NULL>" {
		t.Fatalf("real channel row after admission: %+v", got)
	}
}

func TestManualSyncRejectsMarkerSetBetweenReadAndReservation(t *testing.T) {
	tenantID := setupDemoFixtureTenant(t)
	chID := "ch-fxrace-" + pkg.NewUUID()[:8]
	if err := db.DB.Exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, last_sync_status, last_sync_error, metadata, created_at, updated_at) VALUES (?, ?, 'zalo_oa', 'Kenh', 'race-1', X'00', true, 'success', '', '{}', NOW(), NOW())`, chID, tenantID).Error; err != nil {
		t.Fatal(err)
	}
	launches := 0
	origLoad, origStart := loadManualSyncConfig, startManualSync
	// The handler read an unmarked row; the marker is set while the config loads.
	loadManualSyncConfig = func() (*config.Config, error) {
		if err := db.DB.Exec("UPDATE channels SET is_demo_fixture = TRUE WHERE id = ?", chID).Error; err != nil {
			t.Fatal(err)
		}
		return &config.Config{Env: "test"}, nil
	}
	startManualSync = func(engine.SyncReservation, models.Channel, *config.Config) { launches++ }
	t.Cleanup(func() { loadManualSyncConfig, startManualSync = origLoad, origStart })

	rec := callChannelHandler(SyncChannelNow, "POST", tenantID, chID, "")
	var body map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusConflict || body["error"] != "demo_channel_not_syncable" || launches != 0 {
		t.Fatalf("race: %d %s (launches %d), want 409 demo_channel_not_syncable and no worker", rec.Code, rec.Body.String(), launches)
	}
	if got := readFxChannel(t, chID); got.Status != "success" || got.RunID != "<NULL>" {
		t.Fatalf("row written despite denial: %+v", got)
	}
}
