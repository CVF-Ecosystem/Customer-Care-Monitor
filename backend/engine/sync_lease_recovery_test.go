package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-016: a run admitted for a GET-only channel type (facebook,
// pancake) carries a database-time lease that it extends by heartbeat; a lease
// that expires by database time can be released by a bounded conditional
// recovery. Zalo, unknown types, rows without an R016 lease and rows without
// a run ID stay blocked. All adapters, transports and "recovery-like" clock
// moves are synthetic; lease expiry is simulated by moving the stored deadline
// into the database's past, never by waiting.

type leaseRow struct {
	Type, Status, Error string
	RunID               string // "" when NULL
	LeaseSet            bool
	LeaseFuture         bool
	LastSyncAt          *time.Time
	Credentials         string
	UpdatedAt           string
}

func (f *sfFixture) lease(t *testing.T, id string) leaseRow {
	t.Helper()
	var r struct {
		ChannelType    string
		LastSyncStatus *string
		LastSyncError  *string
		SyncRunID      *string
		LeaseSet       bool
		LeaseFuture    bool
		LastSyncAt     *time.Time
		Creds          string
		UpdatedAt      string
	}
	if err := db.DB.Raw(`SELECT channel_type, last_sync_status, last_sync_error, sync_run_id,
		sync_lease_until IS NOT NULL AS lease_set, COALESCE(sync_lease_until > NOW(3), FALSE) AS lease_future,
		last_sync_at, HEX(credentials_encrypted) AS creds, CAST(updated_at AS CHAR) AS updated_at
		FROM channels WHERE id = ?`, id).Scan(&r).Error; err != nil {
		t.Fatalf("read lease row: %v", err)
	}
	out := leaseRow{Type: r.ChannelType, LeaseSet: r.LeaseSet, LeaseFuture: r.LeaseFuture,
		LastSyncAt: r.LastSyncAt, Credentials: r.Creds, UpdatedAt: r.UpdatedAt}
	if r.LastSyncStatus != nil {
		out.Status = *r.LastSyncStatus
	}
	if r.LastSyncError != nil {
		out.Error = *r.LastSyncError
	}
	if r.SyncRunID != nil {
		out.RunID = *r.SyncRunID
	}
	return out
}

func (f *sfFixture) setType(t *testing.T, id, channelType string) {
	t.Helper()
	if err := db.DB.Exec("UPDATE channels SET channel_type = ? WHERE id = ?", channelType, id).Error; err != nil {
		t.Fatalf("set type: %v", err)
	}
}

// expireLease moves the stored deadline into the database's past.
func (f *sfFixture) expireLease(t *testing.T, id string) {
	t.Helper()
	if err := db.DB.Exec("UPDATE channels SET sync_lease_until = NOW(3) - INTERVAL 1 SECOND WHERE id = ?", id).Error; err != nil {
		t.Fatalf("expire lease: %v", err)
	}
}

// addChannel inserts an extra synthetic channel for this fixture's tenant.
func (f *sfFixture) addChannel(t *testing.T, channelType string) string {
	t.Helper()
	id := "ch-sf-" + pkg.NewUUID()[:12]
	creds, err := pkg.Encrypt([]byte(fmt.Sprintf(`{"id":%q}`, id)), sfKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, last_sync_status, last_sync_error, metadata, created_at, updated_at) VALUES (?, ?, ?, 'Kenh', ?, ?, true, 'idle', '', '{}', '2026-01-01 00:00:00', '2026-01-01 00:00:00')`,
		id, f.tenantID, channelType, "ext-"+id, creds).Error; err != nil {
		t.Fatalf("add channel: %v", err)
	}
	return id
}

func withLeaseRecoveryHook(t *testing.T, hook func(channelID string)) {
	t.Helper()
	prev := leaseRecoveryBeforeRelease
	leaseRecoveryBeforeRelease = hook
	t.Cleanup(func() { leaseRecoveryBeforeRelease = prev })
}

func withLeaseTiming(t *testing.T, lease, heartbeat time.Duration) {
	t.Helper()
	prevLease, prevBeat := syncLeaseDuration, syncHeartbeatInterval
	syncLeaseDuration, syncHeartbeatInterval = lease, heartbeat
	t.Cleanup(func() { syncLeaseDuration, syncHeartbeatInterval = prevLease, prevBeat })
}

func TestReservationWritesLeaseMarkerOnlyForGetOnlyTypes(t *testing.T) {
	for _, tc := range []struct {
		typ    string
		leased bool
	}{{"facebook", true}, {"pancake", true}, {"zalo_oa", false}, {"unknown_type", false}} {
		t.Run(tc.typ, func(t *testing.T) {
			f := setupSFFixture(t)
			f.setType(t, f.chA, tc.typ)
			r := f.mustReserve(t, f.tenantID, f.chA)
			row := f.lease(t, f.chA)
			if r.Leased != tc.leased || row.LeaseSet != tc.leased || row.LeaseFuture != tc.leased {
				t.Fatalf("reservation leased=%v, row set=%v future=%v; want %v", r.Leased, row.LeaseSet, row.LeaseFuture, tc.leased)
			}
			if row.RunID != r.RunID || row.Status != "syncing" {
				t.Fatalf("admission not atomic with the run id: %+v", row)
			}
		})
	}
}

func TestHeartbeatExtendsOnlyTheExactLeasedRun(t *testing.T) {
	f := setupSFFixture(t)
	owner := f.mustReserve(t, f.tenantID, f.chA)
	f.expireLease(t, f.chA)
	before := f.lease(t, f.chA)

	for name, bad := range map[string]SyncReservation{
		"forged id":    {TenantID: owner.TenantID, ChannelID: owner.ChannelID, RunID: "forged-run-id"},
		"wrong tenant": {TenantID: f.otherTenantID, ChannelID: owner.ChannelID, RunID: owner.RunID},
	} {
		if err := extendSyncLease(bad); !errors.Is(err, ErrSyncOwnershipLost) {
			t.Fatalf("%s: got %v, want ownership lost", name, err)
		}
	}
	if after := f.lease(t, f.chA); after != before {
		t.Fatalf("a non-owner heartbeat changed the row: %+v -> %+v", before, after)
	}
	if err := extendSyncLease(owner); err != nil {
		t.Fatalf("owner heartbeat: %v", err)
	}
	if row := f.lease(t, f.chA); !row.LeaseFuture || row.UpdatedAt != before.UpdatedAt {
		t.Fatalf("owner heartbeat did not extend the lease by DB time (or touched updated_at): %+v", row)
	}

	// A Zalo run has no lease and can never gain one through a heartbeat.
	f.setType(t, f.chB, "zalo_oa")
	zalo := f.mustReserve(t, f.tenantID, f.chB)
	if err := extendSyncLease(zalo); !errors.Is(err, ErrSyncOwnershipLost) {
		t.Fatalf("zalo heartbeat: %v", err)
	}
	if row := f.lease(t, f.chB); row.LeaseSet {
		t.Fatal("a zalo run gained a lease")
	}

	// A database failure is a bounded write-failure class.
	f.addTrigger(t, f.chA, "IF NEW.sync_lease_until IS NOT NULL AND NOT (NEW.sync_lease_until <=> OLD.sync_lease_until) THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced heartbeat failure'; END IF;")
	if err := extendSyncLease(owner); !errors.Is(err, ErrSyncWriteFailed) || strings.Contains(err.Error(), "forced") {
		t.Fatalf("heartbeat DB error: %v", err)
	}
}

func TestRecoveryReleasesOnlyExpiredAllowlistedR016Runs(t *testing.T) {
	f := setupSFFixture(t)
	// released: pancake and facebook, leased and expired
	expPancake := f.chA
	resP := f.mustReserve(t, f.tenantID, expPancake)
	f.expireLease(t, expPancake)
	expFacebook := f.addChannel(t, "facebook")
	f.mustReserve(t, f.tenantID, expFacebook)
	f.expireLease(t, expFacebook)
	// kept: lease still in the future
	live := f.chB
	f.mustReserve(t, f.tenantID, live)
	// kept: zalo with a forced expired lease (allowlist is the only guard)
	zalo := f.addChannel(t, "zalo_oa")
	f.mustReserve(t, f.tenantID, zalo)
	if err := db.DB.Exec("UPDATE channels SET sync_lease_until = NOW(3) - INTERVAL 1 SECOND WHERE id = ?", zalo).Error; err != nil {
		t.Fatal(err)
	}
	// kept: old-binary run (run id, NULL lease)
	oldBinary := f.addChannel(t, "pancake")
	if err := db.DB.Exec("UPDATE channels SET last_sync_status = 'syncing', sync_run_id = ?, sync_lease_until = NULL WHERE id = ?", pkg.NewUUID(), oldBinary).Error; err != nil {
		t.Fatal(err)
	}
	// kept: legacy row (no run id) with an expired lease value
	legacy := f.addChannel(t, "facebook")
	if err := db.DB.Exec("UPDATE channels SET last_sync_status = 'syncing', sync_run_id = NULL, sync_lease_until = NOW(3) - INTERVAL 1 SECOND WHERE id = ?", legacy).Error; err != nil {
		t.Fatal(err)
	}
	// seed a checkpoint so we can prove recovery leaves it alone
	if err := db.DB.Exec("UPDATE channels SET last_sync_at = '2026-01-02 03:04:05' WHERE id = ?", expPancake).Error; err != nil {
		t.Fatal(err)
	}
	kept := map[string]leaseRow{}
	for _, id := range []string{live, zalo, oldBinary, legacy} {
		kept[id] = f.lease(t, id)
	}
	beforeP := f.lease(t, expPancake)

	if _, err := RecoverExpiredSyncLeases(syncLeaseRecoveryBatch); err != nil {
		t.Fatalf("recovery: %v", err)
	}

	for _, id := range []string{expPancake, expFacebook} {
		row := f.lease(t, id)
		if row.Status != "error" || row.Error != syncLeaseReleasedMessage || row.RunID != "" || row.LeaseSet {
			t.Fatalf("expired %s not released: %+v", row.Type, row)
		}
	}
	afterP := f.lease(t, expPancake)
	if afterP.LastSyncAt == nil || beforeP.LastSyncAt == nil || !afterP.LastSyncAt.Equal(*beforeP.LastSyncAt) || afterP.Credentials != beforeP.Credentials {
		t.Fatalf("recovery changed the checkpoint or credentials: %+v -> %+v", beforeP, afterP)
	}
	for id, before := range kept {
		if after := f.lease(t, id); after != before {
			t.Fatalf("%s (%s) must stay blocked: %+v -> %+v", id, before.Type, before, after)
		}
	}
	if f.triggerCount() != 0 || f.fetchCount(expPancake) != 0 {
		t.Fatal("recovery launched work or analysis")
	}
	// The released run's old reservation is fenced.
	if err := checkSyncOwnership(resP); !errors.Is(err, ErrSyncOwnershipLost) {
		t.Fatalf("released run still owns the channel: %v", err)
	}
}

func TestRecoveryRacesResolveAtomically(t *testing.T) {
	t.Run("heartbeat wins", func(t *testing.T) {
		f := setupSFFixture(t)
		owner := f.mustReserve(t, f.tenantID, f.chA)
		f.expireLease(t, f.chA)
		withLeaseRecoveryHook(t, func(id string) {
			if id == f.chA {
				if err := extendSyncLease(owner); err != nil {
					t.Errorf("heartbeat: %v", err)
				}
			}
		})
		if _, err := RecoverExpiredSyncLeases(syncLeaseRecoveryBatch); err != nil {
			t.Fatal(err)
		}
		if row := f.lease(t, f.chA); row.Status != "syncing" || row.RunID != owner.RunID || !row.LeaseFuture {
			t.Fatalf("release overrode a heartbeat that extended the lease: %+v", row)
		}
		if err := checkSyncOwnership(owner); err != nil {
			t.Fatalf("owner lost the run: %v", err)
		}
	})
	t.Run("final status wins", func(t *testing.T) {
		f := setupSFFixture(t)
		owner := f.mustReserve(t, f.tenantID, f.chA)
		f.expireLease(t, f.chA)
		eng := NewSyncEngine(f.cfg)
		withLeaseRecoveryHook(t, func(id string) {
			if id == f.chA {
				if wrote, err := eng.recordSyncStatus(owner, "success", ""); !wrote || err != nil {
					t.Errorf("final status: %v %v", wrote, err)
				}
			}
		})
		if _, err := RecoverExpiredSyncLeases(syncLeaseRecoveryBatch); err != nil {
			t.Fatal(err)
		}
		if row := f.lease(t, f.chA); row.Status != "success" || row.LastSyncAt == nil || row.RunID != "" || row.LeaseSet {
			t.Fatalf("release overwrote the recorded success: %+v", row)
		}
	})
	t.Run("newer generation with an expired lease is not released under the old id", func(t *testing.T) {
		f := setupSFFixture(t)
		owner := f.mustReserve(t, f.tenantID, f.chA)
		f.expireLease(t, f.chA)
		eng := NewSyncEngine(f.cfg)
		var newer SyncReservation
		withLeaseRecoveryHook(t, func(id string) {
			if id != f.chA {
				return
			}
			if wrote, err := eng.recordSyncStatus(owner, "success", ""); !wrote || err != nil {
				t.Errorf("final status: %v %v", wrote, err)
			}
			newer = f.mustReserve(t, f.tenantID, f.chA)
			f.expireLease(t, f.chA) // the new run's lease is also expired
		})
		if _, err := RecoverExpiredSyncLeases(syncLeaseRecoveryBatch); err != nil {
			t.Fatal(err)
		}
		if row := f.lease(t, f.chA); row.Status != "syncing" || row.RunID != newer.RunID {
			t.Fatalf("release under the observed old run id hit the newer run: %+v", row)
		}
	})
	t.Run("zero rows and db error", func(t *testing.T) {
		f := setupSFFixture(t)
		f.mustReserve(t, f.tenantID, f.chA)
		f.expireLease(t, f.chA)
		f.addTrigger(t, f.chA, "IF OLD.last_sync_status = 'syncing' AND NEW.last_sync_status <> 'syncing' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced release failure'; END IF;")
		logs := captureEngineLog(t)
		_, err := RecoverExpiredSyncLeases(syncLeaseRecoveryBatch)
		if !errors.Is(err, ErrSyncWriteFailed) || strings.Contains(err.Error()+logs.String(), "forced") {
			t.Fatalf("release DB error: %v (log %q)", err, logs.String())
		}
		if row := f.lease(t, f.chA); row.Status != "syncing" || row.RunID == "" {
			t.Fatalf("failed release changed the row: %+v", row)
		}
	})
}

// ctxAdapter blocks inside its GET-like fetch until released or its context
// ends, then returns one conversation with one message.
type ctxAdapter struct {
	id      string
	entered chan struct{}
	release chan struct{}
	fetches atomic.Int32
}

func (a *ctxAdapter) FetchRecentConversations(ctx context.Context, _ time.Time, _ int) ([]channels.SyncedConversation, error) {
	a.fetches.Add(1)
	if a.entered != nil {
		close(a.entered)
		select {
		case <-a.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
			return nil, errors.New("gated fetch not released")
		}
	}
	return []channels.SyncedConversation{{ExternalID: "conv-" + a.id, CustomerName: "Khach", LastMessageAt: time.Now()}}, nil
}

func (a *ctxAdapter) FetchMessages(_ context.Context, _ string, _ time.Time) ([]channels.SyncedMessage, error) {
	return []channels.SyncedMessage{{ExternalID: "msg-" + a.id, SenderType: "customer", Content: "xin chao", ContentType: "text", SentAt: time.Now()}}, nil
}

func (a *ctxAdapter) HealthCheck(context.Context) error { return nil }

func (f *sfFixture) conversationCount(t *testing.T, externalID string) int64 {
	t.Helper()
	var n int64
	if err := db.DB.Model(&models.Conversation{}).Where("tenant_id = ? AND external_conversation_id = ?", f.tenantID, externalID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestHeartbeatLossCancelsTheRun(t *testing.T) {
	t.Run("released lease", func(t *testing.T) {
		f := setupSFFixture(t)
		withLeaseTiming(t, 5*time.Minute, 30*time.Millisecond)
		a := &ctxAdapter{id: "A", entered: make(chan struct{}), release: make(chan struct{})}
		newSyncAdapter = func(string, []byte) (channels.ChannelAdapter, error) { return a, nil }
		owner := f.mustReserve(t, f.tenantID, f.chA)
		done := make(chan error, 1)
		go func() {
			done <- NewSyncEngine(f.cfg).SyncReservedChannel(context.Background(), f.channel(t, f.chA), owner)
		}()
		<-a.entered
		// One atomic recovery-like release (a 30ms heartbeat could otherwise
		// extend the lease between an expiry and a recovery scan).
		if err := db.DB.Exec("UPDATE channels SET last_sync_status = 'error', last_sync_error = ?, sync_run_id = NULL, sync_lease_until = NULL WHERE id = ?", syncLeaseReleasedMessage, f.chA).Error; err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-done:
			if !errors.Is(err, ErrSyncOwnershipLost) {
				t.Fatalf("run returned %v, want ownership lost (not partial)", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("a failed heartbeat did not cancel the run")
		}
		if f.conversationCount(t, "conv-A") != 0 || f.triggerCount() != 0 {
			t.Fatal("the cancelled run wrote data or triggered analysis")
		}
		if row := f.lease(t, f.chA); row.Status != "error" || row.Error != syncLeaseReleasedMessage {
			t.Fatalf("recovery's state was overwritten: %+v", row)
		}
	})
	t.Run("db error", func(t *testing.T) {
		f := setupSFFixture(t)
		withLeaseTiming(t, 5*time.Minute, 30*time.Millisecond)
		a := &ctxAdapter{id: "A", entered: make(chan struct{}), release: make(chan struct{})}
		newSyncAdapter = func(string, []byte) (channels.ChannelAdapter, error) { return a, nil }
		owner := f.mustReserve(t, f.tenantID, f.chA)
		f.addTrigger(t, f.chA, "IF NEW.sync_lease_until IS NOT NULL AND NOT (NEW.sync_lease_until <=> OLD.sync_lease_until) THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced heartbeat failure'; END IF;")
		logs := captureEngineLog(t)
		done := make(chan error, 1)
		go func() {
			done <- NewSyncEngine(f.cfg).SyncReservedChannel(context.Background(), f.channel(t, f.chA), owner)
		}()
		<-a.entered
		select {
		case err := <-done:
			if !errors.Is(err, ErrSyncWriteFailed) {
				t.Fatalf("run returned %v, want the bounded write-failure class", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("a failed heartbeat did not cancel the run")
		}
		row := f.lease(t, f.chA)
		if row.Status != "error" || row.RunID != "" || row.LeaseSet || row.LastSyncAt != nil {
			t.Fatalf("owned run did not leave syncing through its own error transition: %+v", row)
		}
		if f.conversationCount(t, "conv-A") != 0 || strings.Contains(logs.String(), "forced") || strings.Contains(logs.String(), owner.RunID) {
			t.Fatalf("data written or detail leaked; log %q", logs.String())
		}
	})
}

// Paused A-to-B handoff across full SyncReservedChannel runs. A's heartbeat is
// slow enough not to notice, so only R015's write fence can stop it.
func TestExpiredGenerationCannotTouchTheNewerRun(t *testing.T) {
	f := setupSFFixture(t)
	a := &ctxAdapter{id: "A", entered: make(chan struct{}), release: make(chan struct{})}
	b := &ctxAdapter{id: "B"}
	calls := 0
	newSyncAdapter = func(string, []byte) (channels.ChannelAdapter, error) {
		calls++
		if calls == 1 {
			return a, nil
		}
		return b, nil
	}
	eng := NewSyncEngine(f.cfg)
	oldRun := f.mustReserve(t, f.tenantID, f.chA)
	done := make(chan error, 1)
	go func() { done <- eng.SyncReservedChannel(context.Background(), f.channel(t, f.chA), oldRun) }()
	<-a.entered // A is inside its GET

	f.expireLease(t, f.chA)
	if n, err := RecoverExpiredSyncLeases(syncLeaseRecoveryBatch); err != nil || n < 1 {
		t.Fatalf("release: %d %v", n, err)
	}
	newRun := f.mustReserve(t, f.tenantID, f.chA)
	if !newRun.Leased || newRun.RunID == oldRun.RunID {
		t.Fatalf("new generation %+v", newRun)
	}
	if err := eng.persistZaloRefreshedTokens(newRun, "b-access", "b-refresh"); err != nil {
		t.Fatalf("B credential write: %v", err)
	}
	bBefore := f.lease(t, f.chA)

	close(a.release) // A's GET completes after takeover
	if err := <-done; !errors.Is(err, ErrSyncOwnershipLost) {
		t.Fatalf("stale run returned %v, want ownership lost", err)
	}
	if f.conversationCount(t, "conv-A") != 0 || f.triggerCount() != 0 {
		t.Fatal("the stale run wrote data or triggered analysis")
	}
	if after := f.lease(t, f.chA); after != bBefore {
		t.Fatalf("the stale run changed B's row (status/lease/checkpoint/credentials): %+v -> %+v", bBefore, after)
	}

	if err := eng.SyncReservedChannel(context.Background(), f.channel(t, f.chA), newRun); err != nil {
		t.Fatalf("B run: %v", err)
	}
	final := f.lease(t, f.chA)
	if final.Status != "success" || final.LastSyncAt == nil || final.RunID != "" || final.LeaseSet || f.triggerCount() != 1 || f.conversationCount(t, "conv-B") != 1 {
		t.Fatalf("B did not complete normally: %+v, triggers %d", final, f.triggerCount())
	}
}

func TestTerminalWritesClearTheLease(t *testing.T) {
	for _, mode := range []string{"success", "partial", "error"} {
		t.Run(mode, func(t *testing.T) {
			f := setupSFFixture(t)
			switch mode {
			case "partial":
				f.msgErr = errors.New("synthetic message failure")
			case "error":
				f.fetchErr = errors.New("synthetic fetch failure")
			}
			owner := f.mustReserve(t, f.tenantID, f.chA)
			_ = NewSyncEngine(f.cfg).SyncReservedChannel(context.Background(), f.channel(t, f.chA), owner)
			if row := f.lease(t, f.chA); row.Status != mode || row.LeaseSet || row.RunID != "" {
				t.Fatalf("terminal %s left %+v", mode, row)
			}
		})
	}
}

func TestSchedulerRecoversExpiredLeaseAndReadmitsAfterThrottle(t *testing.T) {
	f := setupSFFixture(t)
	f.mustReserve(t, f.tenantID, f.chA)
	f.expireLease(t, f.chA)
	zalo := f.addChannel(t, "zalo_oa")
	f.mustReserve(t, f.tenantID, zalo)
	if err := db.DB.Exec("UPDATE channels SET sync_lease_until = NOW(3) - INTERVAL 1 SECOND WHERE id = ?", zalo).Error; err != nil {
		t.Fatal(err)
	}
	sched := &Scheduler{syncEngine: NewSyncEngine(f.cfg), cfg: f.cfg}

	sched.recoverExpiredSyncLeasesTask()
	if row := f.lease(t, f.chA); row.Status != "error" {
		t.Fatalf("scheduler did not release the expired lease: %+v", row)
	}
	if row := f.lease(t, zalo); row.Status != "syncing" {
		t.Fatalf("scheduler released a zalo run: %+v", row)
	}

	sched.syncAllChannelsTask()
	if f.fetchCount(f.chA) != 0 {
		t.Fatal("recovery bypassed the normal throttle")
	}
	if err := db.DB.Exec("UPDATE channels SET updated_at = NOW() - INTERVAL 1 HOUR WHERE id = ?", f.chA).Error; err != nil {
		t.Fatal(err)
	}
	sched.syncAllChannelsTask()
	if row := f.lease(t, f.chA); f.fetchCount(f.chA) != 1 || row.Status != "success" || row.LeaseSet {
		t.Fatalf("released channel not re-admitted normally: fetches %d, %+v", f.fetchCount(f.chA), row)
	}
}

func TestLeaseWritesNeverReachTheGormSink(t *testing.T) {
	f := setupSFFixture(t)
	owner := f.mustReserve(t, f.tenantID, f.chA)
	f.expireLease(t, f.chA)
	for _, cfg := range gormSinkConfigs {
		sink := withGormSink(cfg, func() {
			_ = extendSyncLease(owner)
			f.expireLease(t, f.chA)
			_, _ = RecoverExpiredSyncLeases(syncLeaseRecoveryBatch)
		})
		if strings.Contains(sink, owner.RunID) || strings.Contains(sink, "sync_run_id") {
			t.Fatalf("run id or predicate reached the GORM sink at %s: %q", cfg.name, sink)
		}
	}
}
