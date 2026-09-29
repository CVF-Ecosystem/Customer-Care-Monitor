package engine

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
)

// CCMAI-RUNTIME-014: every admitted sync run owns a run ID stored on the
// channel row, and every terminal write must present it. The fixture, fake
// adapter and trigger recorder come from the R013 tests; nothing here contacts
// a real channel or provider and recovery-like releases exist only inside the
// fixture.

// runIDOf returns the stored sync_run_id (nil when NULL).
func (f *sfFixture) runIDOf(t *testing.T, channelID string) *string {
	t.Helper()
	var rows []struct{ SyncRunID *string }
	if err := db.DB.Raw("SELECT sync_run_id FROM channels WHERE id = ?", channelID).Scan(&rows).Error; err != nil || len(rows) != 1 {
		t.Fatalf("read run id: %v (%d rows)", err, len(rows))
	}
	return rows[0].SyncRunID
}

func (f *sfFixture) mustReserve(t *testing.T, tenantID, channelID string) SyncReservation {
	t.Helper()
	r, err := ReserveChannelSync(tenantID, channelID)
	if err != nil {
		t.Fatalf("reserve %s: %v", channelID, err)
	}
	return r
}

// simulateRecoveryRelease is the fixture-only stand-in for a future recovery:
// it puts the row back to idle without any worker cooperation.
func (f *sfFixture) simulateRecoveryRelease(t *testing.T, channelID string) {
	t.Helper()
	if err := db.DB.Exec("UPDATE channels SET last_sync_status = 'idle', sync_run_id = NULL WHERE id = ?", channelID).Error; err != nil {
		t.Fatalf("simulated release: %v", err)
	}
}

func TestReserveChannelSyncAssignsDistinctInternalRunIDs(t *testing.T) {
	f := setupSFFixture(t)

	first := f.mustReserve(t, f.tenantID, f.chA)
	if first.RunID == "" || first.TenantID != f.tenantID || first.ChannelID != f.chA {
		t.Fatalf("reservation %+v", first)
	}
	if got := f.runIDOf(t, f.chA); got == nil || *got != first.RunID {
		t.Fatalf("stored run id %v, want %q", got, first.RunID)
	}
	if err := NewSyncEngine(f.cfg).SyncReservedChannel(context.Background(), f.channel(t, f.chA), first); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := f.runIDOf(t, f.chA); got != nil {
		t.Fatalf("terminal write left run id %q, want NULL", *got)
	}

	second := f.mustReserve(t, f.tenantID, f.chA)
	if second.RunID == first.RunID {
		t.Fatalf("two accepted generations share run id %q", first.RunID)
	}
}

func TestReserveChannelSyncConcurrentWinnerOwnsTheStoredRunID(t *testing.T) {
	f := setupSFFixture(t)
	const n = 16
	var wg sync.WaitGroup
	start := make(chan struct{})
	reservations := make([]SyncReservation, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			reservations[i], errs[i] = ReserveChannelSync(f.tenantID, f.chA)
		}(i)
	}
	close(start)
	wg.Wait()

	winner := -1
	for i, err := range errs {
		if err == nil {
			if winner != -1 {
				t.Fatal("two winners")
			}
			winner = i
		} else if !errors.Is(err, ErrSyncAlreadyRunning) || reservations[i] != (SyncReservation{}) {
			t.Fatalf("loser %d: %v with reservation %+v", i, err, reservations[i])
		}
	}
	if winner == -1 {
		t.Fatal("no winner")
	}
	if got := f.runIDOf(t, f.chA); got == nil || *got != reservations[winner].RunID {
		t.Fatalf("stored run id %v is not the winner's %q", got, reservations[winner].RunID)
	}
}

func TestReserveChannelSyncKeepsLegacyNullIDSyncingRowBlockedAndUnchanged(t *testing.T) {
	f := setupSFFixture(t)
	f.setStatus(t, f.chA, "syncing") // a pre-014 row: syncing, sync_run_id NULL
	before := f.statusOf(t, f.chA)

	r, err := ReserveChannelSync(f.tenantID, f.chA)

	if !errors.Is(err, ErrSyncAlreadyRunning) || r != (SyncReservation{}) {
		t.Fatalf("got %+v, %v; want busy with no reservation", r, err)
	}
	if after := f.statusOf(t, f.chA); after != before || f.runIDOf(t, f.chA) != nil {
		t.Fatalf("legacy row adopted or changed: %+v -> %+v, id %v", before, after, f.runIDOf(t, f.chA))
	}
	// An idle legacy row (NULL id) is admitted and receives an ID.
	if r := f.mustReserve(t, f.tenantID, f.chB); r.RunID == "" {
		t.Fatal("idle row was admitted without a run id")
	}
}

func TestReserveChannelSyncFailuresReturnNoReservationAndNoID(t *testing.T) {
	f := setupSFFixture(t)
	if r, err := ReserveChannelSync(f.otherTenantID, f.chA); err == nil || r != (SyncReservation{}) {
		t.Fatalf("other tenant: %+v, %v", r, err)
	}
	if got := f.runIDOf(t, f.chA); got != nil {
		t.Fatalf("another tenant's attempt stored run id %q", *got)
	}
	f.addTrigger(t, f.chB, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced reservation failure';")
	if r, err := ReserveChannelSync(f.tenantID, f.chB); err == nil || r != (SyncReservation{}) {
		t.Fatalf("db error: %+v, %v", r, err)
	}
	if got := f.runIDOf(t, f.chB); got != nil {
		t.Fatalf("failed write stored run id %q", *got)
	}
}

func TestSyncReservedChannelRejectsMismatchedReservationBeforeAdapterWork(t *testing.T) {
	f := setupSFFixture(t)
	resA := f.mustReserve(t, f.tenantID, f.chA)
	eng := NewSyncEngine(f.cfg)
	chA := f.channel(t, f.chA)

	for name, bad := range map[string]SyncReservation{
		"empty":         {},
		"no run id":     {TenantID: resA.TenantID, ChannelID: resA.ChannelID},
		"other channel": {TenantID: resA.TenantID, ChannelID: f.chB, RunID: resA.RunID},
		"other tenant":  {TenantID: f.otherTenantID, ChannelID: resA.ChannelID, RunID: resA.RunID},
	} {
		if err := eng.SyncReservedChannel(context.Background(), chA, bad); !errors.Is(err, ErrSyncReservationMismatch) {
			t.Fatalf("%s: got %v, want ErrSyncReservationMismatch", name, err)
		}
	}
	if f.fetchCount(f.chA) != 0 || f.triggerCount() != 0 {
		t.Fatalf("rejected runs did work: fetches %d, triggers %d", f.fetchCount(f.chA), f.triggerCount())
	}
	if got := f.statusOf(t, f.chA); got.Status != "syncing" || f.runIDOf(t, f.chA) == nil || *f.runIDOf(t, f.chA) != resA.RunID {
		t.Fatalf("a rejected run changed the owner's row: %+v", got)
	}
}

// Final-write predicate: a final status needs tenant, channel, syncing and the
// exact run ID. Every other combination changes nothing.
func TestRecordSyncStatusRequiresTheOwningRunID(t *testing.T) {
	f := setupSFFixture(t)
	owner := f.mustReserve(t, f.tenantID, f.chA)
	before := f.statusOf(t, f.chA)
	eng := NewSyncEngine(f.cfg)

	for name, r := range map[string]SyncReservation{
		"forged id":     {TenantID: owner.TenantID, ChannelID: owner.ChannelID, RunID: "forged-run-id"},
		"wrong tenant":  {TenantID: f.otherTenantID, ChannelID: owner.ChannelID, RunID: owner.RunID},
		"missing":       {TenantID: owner.TenantID, ChannelID: "ch-missing", RunID: owner.RunID},
		"other channel": {TenantID: owner.TenantID, ChannelID: f.chB, RunID: owner.RunID},
	} {
		wrote, err := eng.recordSyncStatus(r, "success", "")
		if wrote || err == nil || !strings.Contains(err.Error(), "0 rows affected") {
			t.Fatalf("%s: wrote %v, err %v; want zero rows", name, wrote, err)
		}
	}
	if after := f.statusOf(t, f.chA); after != before {
		t.Fatalf("a non-owner write changed the row: %+v -> %+v", before, after)
	}
	if got := f.runIDOf(t, f.chA); got == nil || *got != owner.RunID {
		t.Fatalf("owner id %v lost", got)
	}

	// A legacy syncing row has NULL id: nothing can finish it by ID.
	f.setStatus(t, f.chB, "syncing")
	legacy := SyncReservation{TenantID: f.tenantID, ChannelID: f.chB, RunID: owner.RunID}
	if wrote, err := eng.recordSyncStatus(legacy, "success", ""); wrote || err == nil {
		t.Fatalf("legacy NULL-id row was finished: %v %v", wrote, err)
	}
	if got := f.statusOf(t, f.chB).Status; got != "syncing" {
		t.Fatalf("legacy row now %q", got)
	}
}

func TestSyncTerminalWritesClearOnlyTheOwningRunID(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(f *sfFixture)
		status string
		errOK  bool
		cp     bool
	}{
		{"success", func(f *sfFixture) {}, "success", false, true},
		{"partial", func(f *sfFixture) { f.msgErr = errors.New("synthetic message failure") }, "partial", true, false},
		{"error", func(f *sfFixture) { f.fetchErr = errors.New("synthetic fetch failure") }, "error", true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := setupSFFixture(t)
			c.setup(f)
			owner := f.mustReserve(t, f.tenantID, f.chA)
			other := f.mustReserve(t, f.tenantID, f.chB) // must stay untouched

			err := NewSyncEngine(f.cfg).SyncReservedChannel(context.Background(), f.channel(t, f.chA), owner)

			if (err != nil) != c.errOK {
				t.Fatalf("err %v, want error=%v", err, c.errOK)
			}
			got := f.statusOf(t, f.chA)
			if got.Status != c.status || (got.LastSyncAt != nil) != c.cp || f.runIDOf(t, f.chA) != nil {
				t.Fatalf("state %+v id %v; want %s, checkpoint %v, id cleared", got, f.runIDOf(t, f.chA), c.status, c.cp)
			}
			if (f.triggerCount() == 1) != c.cp {
				t.Fatalf("triggers %d, want one only after success", f.triggerCount())
			}
			if id := f.runIDOf(t, f.chB); id == nil || *id != other.RunID || f.statusOf(t, f.chB).Status != "syncing" {
				t.Fatalf("another channel's run was disturbed: %v", id)
			}
		})
	}
}

func TestSyncFailureTextsNeverRevealTheRunID(t *testing.T) {
	f := setupSFFixture(t)
	owner := f.mustReserve(t, f.tenantID, f.chA)
	f.addTrigger(t, f.chA, "IF OLD.last_sync_status = 'syncing' AND NEW.last_sync_status <> 'syncing' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced final write failure'; END IF;")
	logs := captureEngineLog(t)

	err := NewSyncEngine(f.cfg).SyncReservedChannel(context.Background(), f.channel(t, f.chA), owner)

	if err == nil {
		t.Fatal("unrecorded final status reported success")
	}
	visible := err.Error() + logs.String()
	if strings.Contains(visible, owner.RunID) {
		t.Fatalf("run id exposed: %s", visible)
	}
}

// Deterministic two-generation race. A is reserved and blocked inside its
// adapter; a recovery-like release then lets B reserve the same channel; A
// finishes afterwards. A must not touch B's status, error, checkpoint or ID.
func TestStaleWorkerCannotFinishOverNewerRun(t *testing.T) {
	for _, mode := range []string{"success", "partial", "error", "panic"} {
		t.Run(mode, func(t *testing.T) {
			f := setupSFFixture(t)
			switch mode {
			case "partial":
				f.msgErr = errors.New("synthetic message failure")
			case "error":
				f.fetchErr = errors.New("synthetic fetch failure")
			case "panic":
				f.onFetch = func(id string) {
					if id == f.chA {
						panic("synthetic stale worker panic")
					}
				}
			}
			f.gateFor = f.chA
			eng := NewSyncEngine(f.cfg)
			oldRun := f.mustReserve(t, f.tenantID, f.chA)

			done := make(chan error, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						done <- errors.New("recovered panic")
					}
				}()
				done <- eng.SyncReservedChannel(context.Background(), f.channel(t, f.chA), oldRun)
			}()
			select {
			case <-f.entered:
			case <-time.After(10 * time.Second):
				t.Fatal("stale worker never reached the adapter")
			}

			f.simulateRecoveryRelease(t, f.chA)
			newRun := f.mustReserve(t, f.tenantID, f.chA)
			if newRun.RunID == oldRun.RunID {
				t.Fatal("new generation reused the old run id")
			}
			newBefore := f.statusOf(t, f.chA)

			close(f.release) // the stale worker now finishes/fails/panics
			if err := <-done; err == nil {
				t.Fatal("the stale worker reported a clean completion")
			}

			got := f.statusOf(t, f.chA)
			if got.Status != "syncing" || got.Error != "" || got.LastSyncAt != nil || got.UpdatedAt != newBefore.UpdatedAt {
				t.Fatalf("stale worker changed the new run's row: %+v -> %+v", newBefore, got)
			}
			if id := f.runIDOf(t, f.chA); id == nil || *id != newRun.RunID {
				t.Fatalf("new run's id replaced: %v", id)
			}
			if f.triggerCount() != 0 {
				t.Fatalf("stale worker triggered after-sync work: %v", f.triggers)
			}

			// The new generation still completes normally.
			f.msgErr, f.fetchErr, f.onFetch = nil, nil, nil
			if err := eng.SyncReservedChannel(context.Background(), f.channel(t, f.chA), newRun); err != nil {
				t.Fatalf("new run: %v", err)
			}
			final := f.statusOf(t, f.chA)
			if final.Status != "success" || final.LastSyncAt == nil || f.runIDOf(t, f.chA) != nil || f.triggerCount() != 1 {
				t.Fatalf("new run final state %+v, triggers %d", final, f.triggerCount())
			}
		})
	}
}
