package handlers

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
)

// CCMAI-RUNTIME-008: a manual sync is admitted only with a validated
// configuration, checked after the tenant-scoped lookup and before the R007
// start write. Launcher and (except where noted) config loader are stubbed by
// setupSyncStartFixture; no adapter, credential or provider is called.

func TestSyncChannelNowConfigFailureStartsNoWorker(t *testing.T) {
	f := setupSyncStartFixture(t)
	f.cfgErr = errors.New("ENCRYPTION_KEY=SECRET-DO-NOT-LEAK is invalid")
	logs := captureLog(t)

	rec := f.callSync(f.tenantID)

	f.assertNoStart(t, rec)
	if f.cfgLoads != 1 {
		t.Fatalf("config loaded %d times, want 1", f.cfgLoads)
	}
	body := rec.Body.String()
	if strings.TrimSpace(body) != `{"error":"sync_start_failed"}` {
		t.Fatalf("response %q, want generic sync_start_failed", body)
	}
	for _, leak := range []string{"SECRET", "ENCRYPTION_KEY"} {
		if strings.Contains(body, leak) || strings.Contains(logs.String(), leak) {
			t.Fatalf("config error detail %q leaked; body %q, log %q", leak, body, logs.String())
		}
	}
	if !strings.Contains(logs.String(), f.channelID) || !strings.Contains(logs.String(), "configuration invalid") {
		t.Fatalf("log should name the channel and the failure class: %s", logs.String())
	}
}

func TestSyncChannelNowNilConfigIsNotAdmitted(t *testing.T) {
	f := setupSyncStartFixture(t)
	f.cfg = nil
	f.assertNoStart(t, f.callSync(f.tenantID))
}

func TestSyncChannelNowPassesValidatedConfigToWorker(t *testing.T) {
	f := setupSyncStartFixture(t)
	rec := f.callSync(f.tenantID)

	if rec.Code != http.StatusAccepted || strings.TrimSpace(rec.Body.String()) != `{"message":"sync_started"}` {
		t.Fatalf("got %d %s, want unchanged 202 sync_started", rec.Code, rec.Body.String())
	}
	if f.cfgLoads != 1 || f.launches != 1 {
		t.Fatalf("config loads %d, launches %d; want 1 and 1", f.cfgLoads, f.launches)
	}
	if f.launchedCfg != f.cfg {
		t.Fatalf("worker got config %p, want the validated config %p", f.launchedCfg, f.cfg)
	}
	if f.statusAtLaunch != "syncing" || f.respondedBeforeLaunch {
		t.Fatalf("at launch: status %q, responded already %v", f.statusAtLaunch, f.respondedBeforeLaunch)
	}
}

func TestSyncChannelNowWrongTenantSkipsConfigLoad(t *testing.T) {
	f := setupSyncStartFixture(t)
	rec := f.callSync(f.otherTenantID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("other tenant got %d, want 404", rec.Code)
	}
	if f.cfgLoads != 0 {
		t.Fatalf("config loaded %d times before the tenant check", f.cfgLoads)
	}
	f.assertNoStart(t, rec)
}

// TestSyncChannelNowUsesRealConfigValidation runs the real config.Load with
// synthetic environment values (t.Setenv restores them; no t.Parallel).
func TestSyncChannelNowUsesRealConfigValidation(t *testing.T) {
	const jwt = "synthetic-jwt-secret-for-tests-0123456789"
	const encKey = "synthetic-32-byte-key-0123456789" // exactly 32 bytes

	t.Run("invalid", func(t *testing.T) {
		f := setupSyncStartFixture(t)
		loadManualSyncConfig = config.Load // restored by the fixture cleanup
		t.Setenv("JWT_SECRET", "too-short")
		t.Setenv("ENCRYPTION_KEY", encKey)
		t.Setenv("DB_PASSWORD", "synthetic")
		logs := captureLog(t)

		rec := f.callSync(f.tenantID)
		f.assertNoStart(t, rec)
		if strings.Contains(rec.Body.String(), "JWT") || strings.Contains(logs.String(), "JWT") {
			t.Fatalf("validation detail leaked; body %q, log %q", rec.Body.String(), logs.String())
		}
	})

	t.Run("valid", func(t *testing.T) {
		f := setupSyncStartFixture(t)
		loadManualSyncConfig = config.Load
		t.Setenv("JWT_SECRET", jwt)
		t.Setenv("ENCRYPTION_KEY", encKey)
		t.Setenv("DB_PASSWORD", "synthetic")

		rec := f.callSync(f.tenantID)
		if rec.Code != http.StatusAccepted || f.launches != 1 {
			t.Fatalf("got %d with %d launches, want 202 and 1", rec.Code, f.launches)
		}
		if f.launchedCfg == nil || f.launchedCfg.JWTSecret != jwt || f.launchedCfg.EncryptionKey != encKey {
			t.Fatalf("worker did not receive the config validated from the synthetic environment")
		}
		if got := f.channelStatus(t).LastSyncStatus; got != "syncing" {
			t.Fatalf("status %q, want syncing", got)
		}
	})
}
