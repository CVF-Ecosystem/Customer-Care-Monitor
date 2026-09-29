package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-013: at most one admitted sync per tenant/channel across the
// manual, scheduler and agent entry paths, arbitrated by a conditional MySQL
// update. The adapter factory and the after-sync trigger are replaced by
// recording fakes; no real channel or provider is contacted and every
// credential is synthetic.

const sfKey = "synthetic-32-byte-key-0123456789" // exactly 32 bytes

type sfFixture struct {
	tenantID, otherTenantID string
	chA, chB, chX           string // chA, chB: tenantID; chX: otherTenantID
	cfg                     *config.Config

	mu       sync.Mutex
	fetches  map[string]int
	triggers []string
	// behavior knobs
	fetchErr error
	msgErr   error
	onFetch  func(channelID string)
	gateFor  string        // channel whose fetch blocks until release is closed
	entered  chan struct{} // signalled when the gated fetch starts
	release  chan struct{}
}

type sfAdapter struct {
	f  *sfFixture
	id string
}

func (a *sfAdapter) FetchRecentConversations(_ context.Context, _ time.Time, _ int) ([]channels.SyncedConversation, error) {
	a.f.mu.Lock()
	a.f.fetches[a.id]++
	a.f.mu.Unlock()
	if a.f.onFetch != nil {
		a.f.onFetch(a.id)
	}
	if a.f.gateFor == a.id {
		a.f.entered <- struct{}{}
		select { // bounded, so a broken admission fails the test instead of hanging it
		case <-a.f.release:
		case <-time.After(10 * time.Second):
			return nil, errors.New("gated fetch not released")
		}
	}
	if a.f.fetchErr != nil {
		return nil, a.f.fetchErr
	}
	return []channels.SyncedConversation{{ExternalID: "ext-" + a.id, ExternalUserID: "u", CustomerName: "Khach", LastMessageAt: time.Now(), Metadata: map[string]interface{}{}}}, nil
}

func (a *sfAdapter) FetchMessages(_ context.Context, _ string, _ time.Time) ([]channels.SyncedMessage, error) {
	if a.f.msgErr != nil {
		return nil, a.f.msgErr
	}
	return []channels.SyncedMessage{{ExternalID: "m-" + a.id, SenderType: "customer", SenderName: "Khach", Content: "xin chao", ContentType: "text", SentAt: time.Now()}}, nil
}

func (a *sfAdapter) HealthCheck(context.Context) error { return nil }

func setupSFFixture(t *testing.T) *sfFixture {
	t.Helper()
	connectTestDB(t)
	s := pkg.NewUUID()[:8]
	f := &sfFixture{
		tenantID: "sf-" + s, otherTenantID: "sf-other-" + s,
		chA: "ch-sf-a-" + s, chB: "ch-sf-b-" + s, chX: "ch-sf-x-" + s,
		cfg:     &config.Config{Env: "test", EncryptionKey: sfKey, StorageLocalDir: t.TempDir()},
		fetches: map[string]int{},
		entered: make(chan struct{}, 4),
		release: make(chan struct{}),
	}
	exec := func(sql string, args ...interface{}) {
		if err := db.DB.Exec(sql, args...).Error; err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	for _, tenant := range []string{f.tenantID, f.otherTenantID} {
		exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Single Flight', ?, '{}', NOW(), NOW())`, tenant, tenant)
	}
	for id, tenant := range map[string]string{f.chA: f.tenantID, f.chB: f.tenantID, f.chX: f.otherTenantID} {
		creds, err := pkg.Encrypt([]byte(fmt.Sprintf(`{"id":%q}`, id)), sfKey)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, last_sync_status, last_sync_error, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh', ?, ?, true, 'idle', '', '{}', '2026-01-01 00:00:00', '2026-01-01 00:00:00')`,
			id, tenant, "ext-"+id, creds)
	}
	t.Cleanup(func() {
		for _, tenant := range []string{f.tenantID, f.otherTenantID} {
			db.DB.Exec("DELETE FROM messages WHERE tenant_id = ?", tenant)
			db.DB.Exec("DELETE FROM conversations WHERE tenant_id = ?", tenant)
			db.DB.Exec("DELETE FROM channels WHERE tenant_id = ?", tenant)
			db.DB.Exec("DELETE FROM activity_logs WHERE tenant_id = ?", tenant)
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenant)
		}
	})

	origAdapter, origTrigger := newSyncAdapter, triggerAfterSync
	newSyncAdapter = func(_ string, credsJSON []byte) (channels.ChannelAdapter, error) {
		s := string(credsJSON)
		for _, id := range []string{f.chA, f.chB, f.chX} {
			if strings.Contains(s, id) {
				return &sfAdapter{f: f, id: id}, nil
			}
		}
		return nil, errors.New("unknown synthetic channel")
	}
	triggerAfterSync = func(tenantID, channelID string) {
		f.mu.Lock()
		f.triggers = append(f.triggers, tenantID+"/"+channelID+":"+f.statusOf(t, channelID).Status)
		f.mu.Unlock()
	}
	t.Cleanup(func() { newSyncAdapter, triggerAfterSync = origAdapter, origTrigger })
	return f
}

type sfStatus struct {
	Status, Error, Tenant string
	LastSyncAt            *time.Time
	UpdatedAt             string
}

func (f *sfFixture) statusOf(t *testing.T, id string) sfStatus {
	t.Helper()
	var r struct {
		LastSyncStatus *string
		LastSyncError  *string
		TenantID       string
		LastSyncAt     *time.Time
		UpdatedAt      string
	}
	if err := db.DB.Raw("SELECT last_sync_status, last_sync_error, tenant_id, last_sync_at, CAST(updated_at AS CHAR) AS updated_at FROM channels WHERE id = ?", id).Scan(&r).Error; err != nil {
		t.Fatalf("read channel: %v", err)
	}
	out := sfStatus{Tenant: r.TenantID, LastSyncAt: r.LastSyncAt, UpdatedAt: r.UpdatedAt}
	if r.LastSyncStatus != nil {
		out.Status = *r.LastSyncStatus
	}
	if r.LastSyncError != nil {
		out.Error = *r.LastSyncError
	}
	return out
}

func (f *sfFixture) setStatus(t *testing.T, id, status string) {
	t.Helper()
	if err := db.DB.Exec("UPDATE channels SET last_sync_status = ? WHERE id = ?", status, id).Error; err != nil {
		t.Fatalf("set status: %v", err)
	}
}

func (f *sfFixture) fetchCount(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fetches[id]
}

func (f *sfFixture) triggerCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.triggers)
}

func (f *sfFixture) channel(t *testing.T, id string) models.Channel {
	t.Helper()
	var ch models.Channel
	if err := db.DB.Where("id = ?", id).First(&ch).Error; err != nil {
		t.Fatalf("load channel: %v", err)
	}
	return ch
}

func (f *sfFixture) addTrigger(t *testing.T, id, body string) {
	t.Helper()
	name := "trg_sf_" + strings.ReplaceAll(id, "-", "_")
	// id is a generated alphanumeric string; trigger bodies cannot bind parameters.
	sql := fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE ON channels FOR EACH ROW BEGIN IF OLD.id = '%s' THEN %s END IF; END", name, id, body)
	if err := db.DB.Exec(sql).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + name) })
}

func TestReserveChannelSyncSameChannelHasExactlyOneWinner(t *testing.T) {
	f := setupSFFixture(t)
	const n = 16
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i] = ReserveChannelSync(f.tenantID, f.chA)
		}(i)
	}
	close(start)
	wg.Wait()

	winners, busy := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			winners++
		case errors.Is(err, ErrSyncAlreadyRunning):
			busy++
		default:
			t.Fatalf("unexpected reservation error: %v", err)
		}
	}
	if winners != 1 || busy != n-1 {
		t.Fatalf("winners %d, busy %d; want 1 and %d", winners, busy, n-1)
	}
	if got := f.statusOf(t, f.chA); got.Status != "syncing" || got.Error != "" {
		t.Fatalf("status %+v, want syncing with no error", got)
	}
}

func TestReserveChannelSyncIsTenantAndChannelScoped(t *testing.T) {
	f := setupSFFixture(t)
	before := f.statusOf(t, f.chA)

	if err := ReserveChannelSync(f.otherTenantID, f.chA); !errors.Is(err, ErrSyncChannelMissing) {
		t.Fatalf("other tenant got %v, want ErrSyncChannelMissing", err)
	}
	if after := f.statusOf(t, f.chA); after != before {
		t.Fatalf("another tenant changed the channel: %+v -> %+v", before, after)
	}
	if err := ReserveChannelSync(f.tenantID, "ch-does-not-exist"); !errors.Is(err, ErrSyncChannelMissing) {
		t.Fatalf("missing channel got %v", err)
	}

	if err := ReserveChannelSync(f.tenantID, f.chA); err != nil {
		t.Fatalf("first reservation: %v", err)
	}
	if err := ReserveChannelSync(f.tenantID, f.chB); err != nil {
		t.Fatalf("independent channel of the same tenant must admit: %v", err)
	}
	if err := ReserveChannelSync(f.otherTenantID, f.chX); err != nil {
		t.Fatalf("channel of another tenant must admit independently: %v", err)
	}
}

func TestReserveChannelSyncAdmitsEmptyAndNullStatus(t *testing.T) {
	f := setupSFFixture(t)
	f.setStatus(t, f.chA, "")
	if err := ReserveChannelSync(f.tenantID, f.chA); err != nil {
		t.Fatalf("empty status: %v", err)
	}
	if err := db.DB.Exec("UPDATE channels SET last_sync_status = NULL WHERE id = ?", f.chB).Error; err != nil {
		t.Skipf("column does not accept NULL here: %v", err)
	}
	if err := ReserveChannelSync(f.tenantID, f.chB); err != nil {
		t.Fatalf("NULL status must be admitted: %v", err)
	}
}

func TestReserveChannelSyncWriteFailureAndZeroRowsAdmitNothing(t *testing.T) {
	t.Run("db error", func(t *testing.T) {
		f := setupSFFixture(t)
		f.addTrigger(t, f.chA, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced reservation failure';")
		before := f.statusOf(t, f.chA)
		logs := captureEngineLog(t)

		err := ReserveChannelSync(f.tenantID, f.chA)
		if !errors.Is(err, ErrSyncNotAdmitted) || strings.Contains(err.Error(), "forced") || strings.Contains(logs.String(), "forced") {
			t.Fatalf("got %v (log %q), want a generic ErrSyncNotAdmitted", err, logs.String())
		}
		if after := f.statusOf(t, f.chA); after != before {
			t.Fatalf("status changed despite failed write: %+v -> %+v", before, after)
		}
	})
	t.Run("zero rows on an idle channel", func(t *testing.T) {
		f := setupSFFixture(t)
		f.addTrigger(t, f.chA, "SET NEW.last_sync_status = OLD.last_sync_status; SET NEW.last_sync_error = OLD.last_sync_error; SET NEW.updated_at = OLD.updated_at;")
		if err := ReserveChannelSync(f.tenantID, f.chA); !errors.Is(err, ErrSyncNotAdmitted) {
			t.Fatalf("got %v, want ErrSyncNotAdmitted (not busy, not admitted)", err)
		}
	})
}

// SyncChannel is the engine entry shared by the scheduler and both agent
// actions: while another run holds the channel, no adapter is created or
// called, nothing is triggered and the status is not overwritten.
func TestSyncChannelBusyDoesNoWork(t *testing.T) {
	f := setupSFFixture(t)
	f.setStatus(t, f.chA, "syncing")
	before := f.statusOf(t, f.chA)

	err := NewSyncEngine(f.cfg).SyncChannel(context.Background(), f.channel(t, f.chA))

	if !errors.Is(err, ErrSyncAlreadyRunning) {
		t.Fatalf("got %v, want ErrSyncAlreadyRunning", err)
	}
	if f.fetchCount(f.chA) != 0 || f.triggerCount() != 0 {
		t.Fatalf("busy channel did work: fetches %d, triggers %d", f.fetchCount(f.chA), f.triggerCount())
	}
	if after := f.statusOf(t, f.chA); after != before {
		t.Fatalf("busy channel status overwritten: %+v -> %+v", before, after)
	}
}

// Accepted-run detector: the same fixture observes the adapter, final status,
// checkpoint and trigger of an admitted run, so the zero-work assertions above
// are meaningful.
func TestSyncChannelAdmittedRunSucceedsAndAdvancesCheckpoint(t *testing.T) {
	f := setupSFFixture(t)
	if err := NewSyncEngine(f.cfg).SyncChannel(context.Background(), f.channel(t, f.chA)); err != nil {
		t.Fatalf("admitted run: %v", err)
	}
	got := f.statusOf(t, f.chA)
	if got.Status != "success" || got.Error != "" || got.LastSyncAt == nil {
		t.Fatalf("final state %+v, want success with an advanced checkpoint", got)
	}
	if f.fetchCount(f.chA) != 1 {
		t.Fatalf("adapter fetched %d times, want 1", f.fetchCount(f.chA))
	}
	want := f.tenantID + "/" + f.chA + ":success"
	if f.triggerCount() != 1 || f.triggers[0] != want {
		t.Fatalf("triggers %v, want one %q recorded after the success status", f.triggers, want)
	}
}

// The manual worker enters through SyncReservedChannel; an already-reserved
// channel must run rather than report busy (a second acquire would).
func TestSyncReservedChannelDoesNotReserveAgain(t *testing.T) {
	f := setupSFFixture(t)
	if err := ReserveChannelSync(f.tenantID, f.chA); err != nil {
		t.Fatal(err)
	}
	if err := NewSyncEngine(f.cfg).SyncReservedChannel(context.Background(), f.channel(t, f.chA)); err != nil {
		t.Fatalf("reserved run: %v", err)
	}
	if got := f.statusOf(t, f.chA); got.Status != "success" || f.fetchCount(f.chA) != 1 {
		t.Fatalf("state %+v, fetches %d", got, f.fetchCount(f.chA))
	}
}

func TestSyncChannelConcurrentEntryIsBusyWhileIndependentChannelRuns(t *testing.T) {
	f := setupSFFixture(t)
	f.gateFor = f.chA
	eng := NewSyncEngine(f.cfg)

	done := make(chan error, 1)
	go func() { done <- eng.SyncChannel(context.Background(), f.channel(t, f.chA)) }()
	select {
	case <-f.entered: // first run is inside the adapter
	case <-time.After(10 * time.Second):
		t.Fatal("first run never reached the adapter")
	}

	if err := eng.SyncChannel(context.Background(), f.channel(t, f.chA)); !errors.Is(err, ErrSyncAlreadyRunning) {
		t.Fatalf("second entry got %v, want busy", err)
	}
	if err := eng.SyncChannel(context.Background(), f.channel(t, f.chB)); err != nil {
		t.Fatalf("independent channel: %v", err)
	}
	if f.fetchCount(f.chA) != 1 {
		t.Fatalf("channel A fetched %d times while running, want 1", f.fetchCount(f.chA))
	}
	close(f.release)
	if err := <-done; err != nil {
		t.Fatalf("first run: %v", err)
	}
	if got := f.statusOf(t, f.chA).Status; got != "success" {
		t.Fatalf("channel A status %q, want success", got)
	}
}

func TestSyncAllChannelsDoesNotHideBusyMember(t *testing.T) {
	f := setupSFFixture(t)
	f.setStatus(t, f.chA, "syncing")

	err := NewSyncEngine(f.cfg).SyncAllChannels(context.Background(), f.tenantID)

	if err == nil || !errors.Is(err, ErrSyncAlreadyRunning) {
		t.Fatalf("got %v, want an aggregate error wrapping busy", err)
	}
	if f.fetchCount(f.chA) != 0 {
		t.Fatalf("busy member was fetched %d times", f.fetchCount(f.chA))
	}
	if f.fetchCount(f.chB) != 1 || f.statusOf(t, f.chB).Status != "success" {
		t.Fatalf("the idle member must still run: fetches %d, %+v", f.fetchCount(f.chB), f.statusOf(t, f.chB))
	}
	if f.fetchCount(f.chX) != 0 {
		t.Fatal("another tenant's channel was synced")
	}
}

func TestSchedulerSkipsBusyChannelAndRunsOthers(t *testing.T) {
	f := setupSFFixture(t)
	f.setStatus(t, f.chA, "syncing")
	before := f.statusOf(t, f.chA)
	logs := captureEngineLog(t)

	sched := &Scheduler{syncEngine: NewSyncEngine(f.cfg), cfg: f.cfg}
	sched.syncAllChannelsTask()

	if f.fetchCount(f.chA) != 0 {
		t.Fatalf("scheduler fetched a busy channel %d times", f.fetchCount(f.chA))
	}
	if after := f.statusOf(t, f.chA); after != before {
		t.Fatalf("scheduler overwrote the busy status: %+v -> %+v", before, after)
	}
	if !strings.Contains(logs.String(), "already syncing") {
		t.Fatalf("busy skip should be logged as a skip: %s", logs.String())
	}
	for _, id := range []string{f.chB, f.chX} {
		if f.fetchCount(id) != 1 || f.statusOf(t, id).Status != "success" {
			t.Fatalf("eligible channel %s: fetches %d, %+v", id, f.fetchCount(id), f.statusOf(t, id))
		}
	}
}

func TestSyncRunFinalStatesAndCheckpoint(t *testing.T) {
	t.Run("partial keeps the checkpoint and skips triggers", func(t *testing.T) {
		f := setupSFFixture(t)
		f.msgErr = errors.New("synthetic message failure")
		err := NewSyncEngine(f.cfg).SyncChannel(context.Background(), f.channel(t, f.chA))
		got := f.statusOf(t, f.chA)
		if err == nil || got.Status != "partial" || got.LastSyncAt != nil || f.triggerCount() != 0 {
			t.Fatalf("err %v, state %+v, triggers %d; want partial, no checkpoint, no trigger", err, got, f.triggerCount())
		}
	})
	t.Run("fetch error keeps the checkpoint and skips triggers", func(t *testing.T) {
		f := setupSFFixture(t)
		f.fetchErr = errors.New("synthetic fetch failure")
		err := NewSyncEngine(f.cfg).SyncChannel(context.Background(), f.channel(t, f.chA))
		got := f.statusOf(t, f.chA)
		if err == nil || got.Status != "error" || got.LastSyncAt != nil || f.triggerCount() != 0 {
			t.Fatalf("err %v, state %+v, triggers %d; want error, no checkpoint, no trigger", err, got, f.triggerCount())
		}
	})
}

func TestSyncRunFinalWriteFailureIsNotSuccess(t *testing.T) {
	t.Run("db error on the final write", func(t *testing.T) {
		f := setupSFFixture(t)
		f.addTrigger(t, f.chA, "IF OLD.last_sync_status = 'syncing' AND NEW.last_sync_status <> 'syncing' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced final write failure'; END IF;")
		logs := captureEngineLog(t)

		err := NewSyncEngine(f.cfg).SyncChannel(context.Background(), f.channel(t, f.chA))

		if err == nil {
			t.Fatal("a run whose final status was not recorded reported success")
		}
		got := f.statusOf(t, f.chA)
		if got.LastSyncAt != nil || f.triggerCount() != 0 {
			t.Fatalf("checkpoint %v / triggers %d advanced without a recorded success", got.LastSyncAt, f.triggerCount())
		}
		if !strings.Contains(logs.String(), "could not leave syncing state") {
			t.Fatalf("failed transition must stay observable in the log: %s", logs.String())
		}
	})
	t.Run("zero rows after tenant reassignment", func(t *testing.T) {
		f := setupSFFixture(t)
		f.onFetch = func(id string) {
			if id == f.chA {
				if err := db.DB.Exec("UPDATE channels SET tenant_id = ? WHERE id = ?", f.otherTenantID, f.chA).Error; err != nil {
					t.Errorf("reassign: %v", err)
				}
			}
		}
		err := NewSyncEngine(f.cfg).SyncChannel(context.Background(), f.channel(t, f.chA))

		if err == nil || !strings.Contains(err.Error(), "0 rows affected") {
			t.Fatalf("got %v, want the zero-row write reported", err)
		}
		got := f.statusOf(t, f.chA)
		if got.Tenant != f.otherTenantID || got.Status != "syncing" || got.LastSyncAt != nil || f.triggerCount() != 0 {
			t.Fatalf("transferred row was written or triggered: %+v, triggers %d", got, f.triggerCount())
		}
	})
	t.Run("finished row is never overwritten", func(t *testing.T) {
		f := setupSFFixture(t)
		f.onFetch = func(id string) {
			if id == f.chA {
				f.setStatus(t, f.chA, "error") // something else finished the row
			}
		}
		if err := NewSyncEngine(f.cfg).SyncChannel(context.Background(), f.channel(t, f.chA)); err == nil {
			t.Fatal("reported success although the syncing row was no longer owned")
		}
		if got := f.statusOf(t, f.chA); got.Status != "error" || got.LastSyncAt != nil {
			t.Fatalf("non-syncing row overwritten: %+v", got)
		}
	})
}

func TestSyncRunPanicLeavesSyncingAndRepanics(t *testing.T) {
	f := setupSFFixture(t)
	f.onFetch = func(id string) { panic("synthetic adapter panic PANIC-SECRET") }

	var recovered interface{}
	func() {
		defer func() { recovered = recover() }()
		_ = NewSyncEngine(f.cfg).SyncChannel(context.Background(), f.channel(t, f.chA))
	}()

	if recovered == nil {
		t.Fatal("the original panic was swallowed")
	}
	got := f.statusOf(t, f.chA)
	if got.Status != "error" || strings.Contains(got.Error, "PANIC-SECRET") || got.LastSyncAt != nil || f.triggerCount() != 0 {
		t.Fatalf("state after panic %+v, triggers %d; want a bounded error state", got, f.triggerCount())
	}
}

func captureEngineLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}
