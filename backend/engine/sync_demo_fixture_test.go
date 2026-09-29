package engine

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"gorm.io/gorm"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-018: server-marked demo fixture channels can never be admitted
// to a sync by any entry path, while a real channel in the same tenant keeps
// syncing. Adapters are recording fakes; nothing real is contacted.

// addFixtureChannel inserts a marked demo fixture with the exact plaintext demo
// credential, as the demo importer writes it.
func (f *sfFixture) addFixtureChannel(t *testing.T, channelType, name string) string {
	t.Helper()
	id := "ch-fx-" + pkg.NewUUID()[:12]
	if err := db.DB.Exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, is_demo_fixture, last_sync_status, last_sync_error, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, true, true, '', '', '{}', '2026-01-01 00:00:00', '2026-01-01 00:00:00')`,
		id, f.tenantID, channelType, name, "demo-"+id, []byte(`{"demo":true}`)).Error; err != nil {
		t.Fatalf("add fixture: %v", err)
	}
	return id
}

// countAdapterCalls wraps the fixture's adapter factory to count constructions.
func countAdapterCalls(t *testing.T) *int32 {
	t.Helper()
	var calls int32
	inner := newSyncAdapter
	newSyncAdapter = func(channelType string, credsJSON []byte) (channels.ChannelAdapter, error) {
		atomic.AddInt32(&calls, 1)
		return inner(channelType, credsJSON)
	}
	t.Cleanup(func() { newSyncAdapter = inner })
	return &calls
}

type fxSnapshot struct {
	Status, Error, RunID, UpdatedAt string
	LeaseSet                        bool
	LastSyncAt                      string
}

func (f *sfFixture) fxSnapshot(t *testing.T, id string) fxSnapshot {
	t.Helper()
	var r struct {
		LastSyncStatus, LastSyncError, SyncRunID *string
		LeaseSet                                 bool
		UpdatedAt, LastSyncAt                    *string
	}
	if err := db.DB.Raw(`SELECT last_sync_status, last_sync_error, sync_run_id, sync_lease_until IS NOT NULL AS lease_set,
		CAST(updated_at AS CHAR) AS updated_at, CAST(last_sync_at AS CHAR) AS last_sync_at FROM channels WHERE id = ?`, id).Scan(&r).Error; err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	str := func(p *string) string {
		if p == nil {
			return "<NULL>"
		}
		return *p
	}
	return fxSnapshot{str(r.LastSyncStatus), str(r.LastSyncError), str(r.SyncRunID), str(r.UpdatedAt), r.LeaseSet, str(r.LastSyncAt)}
}

func TestReserveRefusesMarkedFixtureWithoutAnyWrite(t *testing.T) {
	f := setupSFFixture(t)
	fx := f.addFixtureChannel(t, "zalo_oa", "Fixture")
	before := f.fxSnapshot(t, fx)

	if _, err := ReserveChannelSync(f.tenantID, fx); !errors.Is(err, ErrDemoFixture) {
		t.Fatalf("reserve marked fixture: err = %v, want ErrDemoFixture", err)
	}
	if after := f.fxSnapshot(t, fx); after != before {
		t.Fatalf("denial wrote to the fixture row: %+v -> %+v", before, after)
	}
	// A wrong tenant must not learn that the row is a fixture.
	if _, err := ReserveChannelSync(f.otherTenantID, fx); !errors.Is(err, ErrSyncChannelMissing) {
		t.Fatalf("wrong tenant: err = %v, want ErrSyncChannelMissing", err)
	}
	// Positive detector: an unmarked real channel in the same tenant is admitted.
	res, err := ReserveChannelSync(f.tenantID, f.chA)
	if err != nil || res.RunID == "" {
		t.Fatalf("real channel in the same tenant not admitted: %+v, %v", res, err)
	}
	if snap := f.fxSnapshot(t, f.chA); snap.Status != "syncing" || snap.RunID != res.RunID {
		t.Fatalf("real channel row after admission: %+v", snap)
	}
}

func TestSyncChannelOnMarkedFixtureCallsNoAdapterAndNoTrigger(t *testing.T) {
	f := setupSFFixture(t)
	fx := f.addFixtureChannel(t, "facebook", "Fixture")
	calls := countAdapterCalls(t)
	before := f.fxSnapshot(t, fx)

	err := NewSyncEngine(f.cfg).SyncChannel(context.Background(), f.channel(t, fx))
	if !errors.Is(err, ErrDemoFixture) {
		t.Fatalf("SyncChannel err = %v, want ErrDemoFixture", err)
	}
	if atomic.LoadInt32(calls) != 0 || f.triggerCount() != 0 {
		t.Fatalf("denied fixture reached the adapter (%d) or the after-sync trigger (%d)", atomic.LoadInt32(calls), f.triggerCount())
	}
	if after := f.fxSnapshot(t, fx); after != before {
		t.Fatalf("row changed: %+v -> %+v", before, after)
	}
	if strings.Contains(err.Error(), "decrypt") || strings.Contains(err.Error(), "SELECT") {
		t.Fatalf("denial text is not bounded: %v", err)
	}
}

func TestSchedulerNeverTouchesFixturesAndStillSyncsRealChannels(t *testing.T) {
	f := setupSFFixture(t)
	name := "FixtureName-" + pkg.NewUUID()[:8]
	fxZalo := f.addFixtureChannel(t, "zalo_oa", name)
	fxFB := f.addFixtureChannel(t, "facebook", name+"-fb")
	calls := countAdapterCalls(t)
	logs := captureEngineLog(t)
	beforeZalo, beforeFB := f.fxSnapshot(t, fxZalo), f.fxSnapshot(t, fxFB)

	sched := &Scheduler{syncEngine: NewSyncEngine(f.cfg), cfg: f.cfg}
	sched.syncAllChannelsTask()

	if s := logs.String(); strings.Contains(s, name) {
		t.Fatalf("scheduler attempted a fixture channel: %s", s)
	}
	for id, before := range map[string]fxSnapshot{fxZalo: beforeZalo, fxFB: beforeFB} {
		if after := f.fxSnapshot(t, id); after != before {
			t.Fatalf("scheduler rewrote fixture %s: %+v -> %+v", id, before, after)
		}
	}
	// Positive detector: the real channels of the same tenant were synced, and
	// only they touched the adapter factory.
	for _, id := range []string{f.chA, f.chB} {
		if f.fetchCount(id) != 1 || f.statusOf(t, id).Status != "success" {
			t.Fatalf("real channel %s: fetches %d, %+v", id, f.fetchCount(id), f.statusOf(t, id))
		}
	}
	if got := atomic.LoadInt32(calls); got != 3 { // chA, chB, chX only
		t.Fatalf("adapter constructed %d times, want 3 (real channels only)", got)
	}
}

func TestAgentSyncAllSkipsFixturesWithoutFailingRealChannels(t *testing.T) {
	f := setupSFFixture(t)
	fx := f.addFixtureChannel(t, "zalo_oa", "Fixture")
	before := f.fxSnapshot(t, fx)

	if err := NewSyncEngine(f.cfg).SyncAllChannels(context.Background(), f.tenantID); err != nil {
		t.Fatalf("SyncAllChannels on a mixed tenant: %v", err)
	}
	if after := f.fxSnapshot(t, fx); after != before {
		t.Fatalf("fixture changed: %+v -> %+v", before, after)
	}
	for _, id := range []string{f.chA, f.chB} {
		if f.statusOf(t, id).Status != "success" {
			t.Fatalf("real channel %s: %+v", id, f.statusOf(t, id))
		}
	}
}

func TestLeaseRecoveryIgnoresMarkedFixtures(t *testing.T) {
	f := setupSFFixture(t)
	fx := f.addFixtureChannel(t, "facebook", "Fixture")
	// Same shape as an expired leased run, but the row is a fixture.
	if err := db.DB.Exec(`UPDATE channels SET last_sync_status = 'syncing', sync_run_id = 'run-fx', sync_lease_until = NOW(3) - INTERVAL 1 SECOND WHERE id = ?`, fx).Error; err != nil {
		t.Fatal(err)
	}
	// Positive detector: an unmarked expired lease is released by the same call.
	res, err := ReserveChannelSync(f.tenantID, f.chA)
	if err != nil {
		t.Fatal(err)
	}
	f.expireLease(t, f.chA)
	before := f.fxSnapshot(t, fx)

	released, err := RecoverExpiredSyncLeases(10)
	if err != nil || released != 1 {
		t.Fatalf("recovery released %d (err %v), want exactly the real channel", released, err)
	}
	if f.statusOf(t, f.chA).Status != "error" || f.lease(t, f.chA).RunID != "" {
		t.Fatalf("real channel %s was not released: %+v", res.RunID, f.lease(t, f.chA))
	}
	if after := f.fxSnapshot(t, fx); after != before {
		t.Fatalf("recovery touched a fixture: %+v -> %+v", before, after)
	}
}

func TestReserveFailsClosedOnDatabaseErrors(t *testing.T) {
	f := setupSFFixture(t)
	fx := f.addFixtureChannel(t, "zalo_oa", "Fixture")
	before := f.fxSnapshot(t, fx)
	realBefore := f.fxSnapshot(t, f.chA)

	const upd, qry = "ccma:test_fail_reserve_update", "ccma:test_fail_classify_read"
	db.DB.Callback().Update().Before("gorm:update").Register(upd, func(tx *gorm.DB) {
		if m, ok := tx.Statement.Dest.(map[string]interface{}); ok && m["sync_run_id"] != nil {
			tx.AddError(errors.New("forced reservation write failure"))
		}
	})
	t.Cleanup(func() { db.DB.Callback().Update().Remove(upd) })
	if _, err := ReserveChannelSync(f.tenantID, f.chA); !errors.Is(err, ErrSyncNotAdmitted) {
		t.Fatalf("write failure: err = %v, want ErrSyncNotAdmitted", err)
	}
	if after := f.fxSnapshot(t, f.chA); after != realBefore {
		t.Fatalf("failed write left a trace: %+v -> %+v", realBefore, after)
	}
	db.DB.Callback().Update().Remove(upd)

	db.DB.Callback().Row().After("gorm:row").Register(qry, func(tx *gorm.DB) {
		if strings.Contains(tx.Statement.SQL.String(), "is_demo_fixture") {
			tx.AddError(errors.New("forced classification read failure"))
		}
	})
	t.Cleanup(func() { db.DB.Callback().Row().Remove(qry) })
	_, err := ReserveChannelSync(f.tenantID, fx)
	if !errors.Is(err, ErrSyncNotAdmitted) || errors.Is(err, ErrDemoFixture) {
		t.Fatalf("classification read failure: err = %v, want ErrSyncNotAdmitted", err)
	}
	if after := f.fxSnapshot(t, fx); after != before {
		t.Fatalf("fixture row changed: %+v -> %+v", before, after)
	}
}

