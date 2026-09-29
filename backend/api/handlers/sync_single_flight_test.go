package handlers

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
)

// CCMAI-RUNTIME-013: the manual handler and both agent sync actions share the
// engine's conditional reservation. The launcher is stubbed by
// setupSyncStartFixture; the fixture channel carries an undecryptable
// credential blob, so a run that gets past admission ends in a visible
// "error" status without any adapter or outbound call.

const busyBody = `{"error":"sync_already_running"}`

func TestSyncChannelNowSecondRequestWhileBusyIsConflict(t *testing.T) {
	f := setupSyncStartFixture(t)

	first := f.callSync(f.tenantID)
	if first.Code != http.StatusAccepted || f.launches != 1 || f.statusAtLaunch != "syncing" {
		t.Fatalf("first: %d %s, launches %d, status at launch %q", first.Code, first.Body.String(), f.launches, f.statusAtLaunch)
	}

	second := f.callSync(f.tenantID)
	if second.Code != http.StatusConflict || strings.TrimSpace(second.Body.String()) != busyBody {
		t.Fatalf("second: %d %s, want exact 409 %s", second.Code, second.Body.String(), busyBody)
	}
	if f.launches != 1 {
		t.Fatalf("a busy request launched a worker: launches %d", f.launches)
	}
	if got := f.channelStatus(t).LastSyncStatus; got != "syncing" {
		t.Fatalf("status %q, want the first run's syncing", got)
	}
}

func TestSyncChannelNowBusyChannelIsNotDisturbedByPresetSyncing(t *testing.T) {
	f := setupSyncStartFixture(t)
	f.markSyncing(t)
	before := f.channelStatus(t)

	rec := f.callSync(f.tenantID)

	if rec.Code != http.StatusConflict || strings.TrimSpace(rec.Body.String()) != busyBody || f.launches != 0 {
		t.Fatalf("%d %s, launches %d; want 409 and no dispatch", rec.Code, rec.Body.String(), f.launches)
	}
	if after := f.channelStatus(t); after.UpdatedAt != before.UpdatedAt || after.LastSyncStatus != "syncing" {
		t.Fatalf("busy request changed the row: %+v -> %+v", before, after)
	}
}

func TestSyncChannelNowReservationFailureKeepsGenericResponses(t *testing.T) {
	f := setupSyncStartFixture(t)
	rec := f.callSync(f.otherTenantID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("wrong tenant got %d, want 404", rec.Code)
	}
	f.assertNoStart(t, rec)
}

// The manual worker enters through the already-reserved path. If it reserved a
// second time it would find its own "syncing" and leave the status untouched.
func TestRunManualSyncDoesNotReserveAgain(t *testing.T) {
	f := setupSyncStartFixture(t)
	cfg := &config.Config{Env: "test", EncryptionKey: "synthetic-32-byte-key-0123456789"}
	res := f.reserve(t)

	runManualSync(res, f.channelStatus(t), cfg)

	ch := f.channelStatus(t)
	if ch.LastSyncStatus != "error" || !strings.Contains(ch.LastSyncError, "decrypt failed") {
		t.Fatalf("worker ended in %q / %q; want the run's own error state (a second reservation would leave syncing)", ch.LastSyncStatus, ch.LastSyncError)
	}
}

func TestHandleManualSyncPanicDoesNotOverwriteFinishedRow(t *testing.T) {
	f := setupSyncStartFixture(t) // fixture status is "success", not syncing
	forged := engine.SyncReservation{TenantID: f.tenantID, ChannelID: f.channelID, RunID: "forged-run-id"}
	if err := handleManualSyncPanic(forged, panicSecret); err == nil {
		t.Fatal("a panic recovery reported success on a row that was not syncing")
	}
	if got := f.channelStatus(t).LastSyncStatus; got != "success" {
		t.Fatalf("finished row overwritten with %q", got)
	}
}

func TestAgentSyncChannelBusyIsNonSuccessWithBoundedReason(t *testing.T) {
	f := setupSyncStartFixture(t)
	f.markSyncing(t)
	before := f.channelStatus(t)
	cfg := &config.Config{Env: "test", EncryptionKey: "synthetic-32-byte-key-0123456789"}

	res := handleSyncAgent(context.Background(), cfg, AgentRunRequest{
		TenantID: f.tenantID, Action: "sync_channel", Params: map[string]interface{}{"channel_id": f.channelID},
	})

	if res.Status != "error" || len(res.Errors) != 1 || res.Errors[0] != "sync_already_running" {
		t.Fatalf("got %+v, want a non-success result with reason sync_already_running", res)
	}
	if after := f.channelStatus(t); after.UpdatedAt != before.UpdatedAt || after.LastSyncStatus != "syncing" {
		t.Fatalf("busy agent action changed the row: %+v -> %+v", before, after)
	}
}

// Accepted-path detector for the agent actions: an idle channel reaches the
// engine past admission (its status leaves "success" and the run reports its
// own failure), so the busy assertions are not vacuous.
func TestAgentSyncActionsAdmitIdleChannel(t *testing.T) {
	cfg := &config.Config{Env: "test", EncryptionKey: "synthetic-32-byte-key-0123456789"}
	for _, action := range []string{"sync_channel", "sync_all"} {
		t.Run(action, func(t *testing.T) {
			f := setupSyncStartFixture(t)
			res := handleSyncAgent(context.Background(), cfg, AgentRunRequest{
				TenantID: f.tenantID, Action: action, Params: map[string]interface{}{"channel_id": f.channelID},
			})
			if res.Status != "error" || strings.Contains(strings.Join(res.Errors, ";"), "sync_already_running") {
				t.Fatalf("got %+v, want the run's own failure, not busy", res)
			}
			ch := f.channelStatus(t)
			if ch.LastSyncStatus != "error" || !strings.Contains(ch.LastSyncError, "decrypt failed") {
				t.Fatalf("status %q / %q; the engine was not reached past admission", ch.LastSyncStatus, ch.LastSyncError)
			}
		})
	}
}

func TestAgentSyncAllDoesNotReportSuccessForBusyMember(t *testing.T) {
	f := setupSyncStartFixture(t)
	f.markSyncing(t)
	cfg := &config.Config{Env: "test", EncryptionKey: "synthetic-32-byte-key-0123456789"}

	res := handleSyncAgent(context.Background(), cfg, AgentRunRequest{TenantID: f.tenantID, Action: "sync_all"})

	if res.Status == "success" || !strings.Contains(strings.Join(res.Errors, ";"), "sync_already_running") {
		t.Fatalf("got %+v, want a non-success aggregate naming the busy channel", res)
	}
	var status string
	if err := db.DB.Raw("SELECT last_sync_status FROM channels WHERE id = ?", f.channelID).Row().Scan(&status); err != nil || status != "syncing" {
		t.Fatalf("busy member status %q (%v), want untouched syncing", status, err)
	}
}
