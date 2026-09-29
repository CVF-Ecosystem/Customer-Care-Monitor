package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
)

// CCMAI-RUNTIME-014: the manual path carries the reservation's run ID from
// admission into the worker and its panic recovery, and a worker holding an
// old or forged ID cannot finish a channel. The launcher is stubbed by
// setupSyncStartFixture; the fixture channel's credential blob cannot be
// decrypted, so a worker that runs ends in a visible "error" status without
// any adapter or outbound call.

func (f *syncStartFixture) runID(t *testing.T) *string {
	t.Helper()
	var rows []struct{ SyncRunID *string }
	if err := db.DB.Raw("SELECT sync_run_id FROM channels WHERE id = ?", f.channelID).Scan(&rows).Error; err != nil || len(rows) != 1 {
		t.Fatalf("read run id: %v (%d rows)", err, len(rows))
	}
	return rows[0].SyncRunID
}

func TestSyncChannelNowLaunchesWorkerWithTheStoredRunID(t *testing.T) {
	f := setupSyncStartFixture(t)

	rec := f.callSync(f.tenantID)

	if rec.Code != http.StatusAccepted || f.launches != 1 {
		t.Fatalf("%d %s, launches %d", rec.Code, rec.Body.String(), f.launches)
	}
	stored := f.runID(t)
	if stored == nil || *stored == "" || f.launchedRes.RunID != *stored {
		t.Fatalf("launcher got run id %q, stored %v", f.launchedRes.RunID, stored)
	}
	if f.launchedRes.TenantID != f.tenantID || f.launchedRes.ChannelID != f.channelID {
		t.Fatalf("reservation %+v does not name the request's tenant/channel", f.launchedRes)
	}

	busy := f.callSync(f.tenantID)
	if busy.Code != http.StatusConflict || strings.TrimSpace(busy.Body.String()) != busyBody || f.launches != 1 {
		t.Fatalf("busy: %d %s, launches %d", busy.Code, busy.Body.String(), f.launches)
	}
	if again := f.runID(t); again == nil || *again != *stored {
		t.Fatalf("a busy request changed the owner's run id: %v", again)
	}
}

func TestSyncRunIDIsInternal(t *testing.T) {
	f := setupSyncStartFixture(t)
	res := f.reserve(t)

	raw, err := json.Marshal(f.channelStatus(t))
	if err != nil {
		t.Fatal(err)
	}
	list := serveChannelHandler(ListChannels, f.tenantID, "GET", "/api/v1/channels", "", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list: %d %s", list.Code, list.Body.String())
	}
	for name, body := range map[string]string{"channel model": string(raw), "channel list": list.Body.String()} {
		if strings.Contains(body, res.RunID) || strings.Contains(body, "sync_run_id") {
			t.Fatalf("%s exposes the run id: %s", name, body)
		}
	}
}

// If the worker dropped or replaced the run ID it was given, its terminal
// write would match no row and the channel would stay syncing. With the ID it
// was handed, the same worker ends its own run.
func TestRunManualSyncFinishesOnlyWithItsOwnRunID(t *testing.T) {
	cfg := &config.Config{Env: "test", EncryptionKey: "synthetic-32-byte-key-0123456789"}

	t.Run("own id", func(t *testing.T) {
		f := setupSyncStartFixture(t)
		res := f.reserve(t)
		runManualSync(res, f.channelStatus(t), cfg)
		if ch := f.channelStatus(t); ch.LastSyncStatus != "error" || f.runID(t) != nil {
			t.Fatalf("status %q, run id %v; want the run finished and its id cleared", ch.LastSyncStatus, f.runID(t))
		}
	})
	t.Run("dropped or forged id", func(t *testing.T) {
		f := setupSyncStartFixture(t)
		res := f.reserve(t)
		forged := res
		forged.RunID = "forged-run-id"
		runManualSync(forged, f.channelStatus(t), cfg)
		if ch := f.channelStatus(t); ch.LastSyncStatus != "syncing" {
			t.Fatalf("status %q, want the owner's syncing left alone", ch.LastSyncStatus)
		}
		if id := f.runID(t); id == nil || *id != res.RunID {
			t.Fatalf("owner's run id changed: %v", id)
		}
	})
}

// A stale manual worker (and its panic handler) finishing after a
// recovery-like release and a newer reservation must not touch the new run.
func TestStaleManualWorkerAndPanicHandlerCannotClearNewerRun(t *testing.T) {
	f := setupSyncStartFixture(t)
	cfg := &config.Config{Env: "test", EncryptionKey: "synthetic-32-byte-key-0123456789"}
	oldRun := f.reserve(t)

	// Fixture-only stand-in for a future recovery.
	if err := db.DB.Exec("UPDATE channels SET last_sync_status = 'idle', sync_run_id = NULL WHERE id = ?", f.channelID).Error; err != nil {
		t.Fatal(err)
	}
	newRun := f.reserve(t)
	before := f.channelStatus(t)

	runManualSync(oldRun, f.channelStatus(t), cfg)
	if err := handleManualSyncPanic(oldRun, panicSecret); err == nil {
		t.Fatal("the stale panic handler reported success")
	}

	after := f.channelStatus(t)
	if after.LastSyncStatus != "syncing" || after.LastSyncError != "" || after.LastSyncAt != before.LastSyncAt || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("stale worker changed the new run: %+v -> %+v", before, after)
	}
	if id := f.runID(t); id == nil || *id != newRun.RunID {
		t.Fatalf("new run id replaced: %v", id)
	}
	// The owner's panic handler still works with the right ID.
	if err := handleManualSyncPanic(newRun, panicSecret); err != nil {
		t.Fatalf("owner's panic handler: %v", err)
	}
	if got := f.channelStatus(t); got.LastSyncStatus != "error" || f.runID(t) != nil {
		t.Fatalf("owner's panic write not applied: %+v", got)
	}
}
