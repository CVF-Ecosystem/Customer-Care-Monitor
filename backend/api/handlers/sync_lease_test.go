package handlers

import (
	"net/http"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
)

// CCMAI-RUNTIME-016 (handler side): a manual reservation of the fixture's
// pancake channel carries a lease, the 202/409 contract is unchanged, and the
// manual panic write clears the lease together with the run ID. The launcher
// is stubbed; no adapter or outbound request runs.

func (f *syncStartFixture) leaseSet(t *testing.T) bool {
	t.Helper()
	var rows []struct{ Set bool }
	if err := db.DB.Raw("SELECT sync_lease_until IS NOT NULL AS `set` FROM channels WHERE id = ?", f.channelID).Scan(&rows).Error; err != nil || len(rows) != 1 {
		t.Fatalf("read lease: %v", err)
	}
	return rows[0].Set
}

func TestManualSyncCarriesLeaseAndPanicWriteClearsIt(t *testing.T) {
	f := setupSyncStartFixture(t)

	if rec := f.callSync(f.tenantID); rec.Code != http.StatusAccepted || f.launches != 1 {
		t.Fatalf("first request: %d %s", rec.Code, rec.Body.String())
	}
	if !f.launchedRes.Leased || !f.leaseSet(t) {
		t.Fatal("a pancake manual run was admitted without a lease")
	}
	if rec := f.callSync(f.tenantID); rec.Code != http.StatusConflict || f.launches != 1 {
		t.Fatalf("busy request: %d %s, launches %d", rec.Code, rec.Body.String(), f.launches)
	}

	if err := handleManualSyncPanic(f.launchedRes, panicSecret); err != nil {
		t.Fatalf("panic write: %v", err)
	}
	if ch := f.channelStatus(t); ch.LastSyncStatus != "error" || f.runID(t) != nil || f.leaseSet(t) {
		t.Fatalf("panic write left status %q, run id %v, lease %v", ch.LastSyncStatus, f.runID(t), f.leaseSet(t))
	}
}
