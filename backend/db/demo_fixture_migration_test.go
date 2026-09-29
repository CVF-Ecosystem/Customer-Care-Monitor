package db

import (
	"reflect"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-018: the internal is_demo_fixture column arrives through the
// ordinary AutoMigrate and the exact legacy fixture rows are marked, once and
// idempotently. All rows are synthetic; no channel is contacted.

type channelOld018 struct {
	ID string `gorm:"type:char(36);primaryKey"`
}

func (channelOld018) TableName() string { return "channels_018_old" }

type channelNew018 struct {
	ID            string `gorm:"type:char(36);primaryKey"`
	IsDemoFixture bool   `gorm:"not null;default:false"`
}

func (channelNew018) TableName() string { return "channels_018_old" }

func TestDemoFixtureColumnOnFreshSchemaAndOldSchemaUpgrade(t *testing.T) {
	connectMigrationTestDB(t)
	col, ok := columnOf(t, "channels", "is_demo_fixture")
	if !ok || col.IsNullable != "NO" {
		t.Fatalf("channels.is_demo_fixture = %+v (present %v), want a NOT NULL column", col, ok)
	}
	real, _ := reflect.TypeOf(models.Channel{}).FieldByName("IsDemoFixture")
	emulated, _ := reflect.TypeOf(channelNew018{}).FieldByName("IsDemoFixture")
	if real.Tag.Get("gorm") != emulated.Tag.Get("gorm") || real.Tag.Get("json") != "-" {
		t.Fatalf("model tags differ: real %q/%q, emulated %q", real.Tag.Get("gorm"), real.Tag.Get("json"), emulated.Tag.Get("gorm"))
	}

	const old = "channels_018_old"
	DB.Exec("DROP TABLE IF EXISTS " + old)
	t.Cleanup(func() { DB.Exec("DROP TABLE IF EXISTS " + old) })
	if err := DB.AutoMigrate(&channelOld018{}); err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&channelOld018{ID: "legacy-row"}).Error; err != nil {
		t.Fatal(err)
	}
	for pass := 1; pass <= 2; pass++ {
		if err := DB.AutoMigrate(&channelNew018{}); err != nil {
			t.Fatalf("pass %d: %v", pass, err)
		}
	}
	var rows []channelNew018
	if err := DB.Find(&rows).Error; err != nil || len(rows) != 1 || rows[0].IsDemoFixture {
		t.Fatalf("existing row after column add: %+v (err %v), want unmarked", rows, err)
	}
}

type legacyChannelCase struct {
	name       string
	demoTenant bool
	chType     string
	external   string
	creds      []byte
	status     string
	errText    string
	syncedAt   string // "" = NULL
	runID      string // "" = NULL
	leaseSet   bool
	wantMarked bool
	wantStatus string // status after backfill
	wantError  string
}

type channelState struct {
	Marked                bool
	Status, Error, RunID  string
	LastSyncAt, UpdatedAt string
	LeaseSet              bool
}

func readChannelState(t *testing.T, id string) channelState {
	t.Helper()
	var r struct {
		IsDemoFixture                            bool
		LastSyncStatus, LastSyncError, SyncRunID *string
		LastSyncAt, UpdatedAt                    *string
		LeaseSet                                 bool
	}
	if err := DB.Raw(`SELECT is_demo_fixture, last_sync_status, last_sync_error, sync_run_id, CAST(last_sync_at AS CHAR) AS last_sync_at,
		CAST(updated_at AS CHAR) AS updated_at, sync_lease_until IS NOT NULL AS lease_set FROM channels WHERE id = ?`, id).Scan(&r).Error; err != nil {
		t.Fatal(err)
	}
	s := func(p *string) string {
		if p == nil {
			return "<NULL>"
		}
		return *p
	}
	return channelState{r.IsDemoFixture, s(r.LastSyncStatus), s(r.LastSyncError), s(r.SyncRunID), s(r.LastSyncAt), s(r.UpdatedAt), r.LeaseSet}
}

func TestLegacyDemoFixtureBackfillIsExactGuardedAndIdempotent(t *testing.T) {
	connectMigrationTestDB(t)
	suffix := pkg.NewUUID()[:8]
	plain := []byte(`{"demo":true}`)
	enc := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	const decryptErr = "decrypt failed: decrypt: cipher: message authentication failed"
	cases := []legacyChannelCase{
		{name: "zalo fixture, old decrypt error", demoTenant: true, chType: "zalo_oa", external: "demo-zalo-oa", creds: plain, status: "error", errText: decryptErr, wantMarked: true, wantStatus: ""},
		{name: "facebook fixture, old decrypt error", demoTenant: true, chType: "facebook", external: "demo-fb-page", creds: plain, status: "error", errText: decryptErr, wantMarked: true, wantStatus: ""},
		{name: "fixture never attempted", demoTenant: true, chType: "zalo_oa", external: "demo-zalo-oa", creds: plain, status: "", wantMarked: true, wantStatus: ""},
		{name: "fixture with an active run", demoTenant: true, chType: "zalo_oa", external: "demo-zalo-oa", creds: plain, status: "syncing", runID: "run-active", leaseSet: true, wantMarked: true, wantStatus: "syncing"},
		{name: "fixture error but a checkpoint exists", demoTenant: true, chType: "zalo_oa", external: "demo-zalo-oa", creds: plain, status: "error", errText: decryptErr, syncedAt: "2026-03-01 10:00:00", wantMarked: true, wantStatus: "error", wantError: decryptErr},
		{name: "fixture with a different error", demoTenant: true, chType: "zalo_oa", external: "demo-zalo-oa", creds: plain, status: "error", errText: "timeout talking to upstream", wantMarked: true, wantStatus: "error", wantError: "timeout talking to upstream"},
		{name: "fixture error with a stale run id", demoTenant: true, chType: "zalo_oa", external: "demo-zalo-oa", creds: plain, status: "error", errText: decryptErr, runID: "run-stale", wantMarked: true, wantStatus: "error", wantError: decryptErr},
		{name: "real channel in a demo tenant", demoTenant: true, chType: "pancake", external: "real-page-1", creds: enc, status: "error", errText: decryptErr, wantMarked: false, wantStatus: "error", wantError: decryptErr},
		{name: "same name but encrypted credentials", demoTenant: true, chType: "zalo_oa", external: "demo-zalo-oa", creds: enc, status: "error", errText: decryptErr, wantMarked: false, wantStatus: "error", wantError: decryptErr},
		{name: "exact identity in a non-demo tenant", demoTenant: false, chType: "zalo_oa", external: "demo-zalo-oa", creds: plain, status: "error", errText: decryptErr, wantMarked: false, wantStatus: "error", wantError: decryptErr},
		{name: "demo id with the wrong channel type", demoTenant: true, chType: "facebook", external: "demo-zalo-oa", creds: plain, status: "error", errText: decryptErr, wantMarked: false, wantStatus: "error", wantError: decryptErr},
		// R018-R1: the columns are case-insensitive, so wrong-case near matches
		// must stay untouched (unmarked, or marked with the error kept).
		{name: "wrong-case channel type", demoTenant: true, chType: "Zalo_OA", external: "demo-zalo-oa", creds: plain, status: "error", errText: decryptErr, wantMarked: false, wantStatus: "error", wantError: decryptErr},
		{name: "wrong-case external id", demoTenant: true, chType: "zalo_oa", external: "Demo-Zalo-OA", creds: plain, status: "error", errText: decryptErr, wantMarked: false, wantStatus: "error", wantError: decryptErr},
		{name: "wrong-case facebook identity", demoTenant: true, chType: "FACEBOOK", external: "DEMO-FB-PAGE", creds: plain, status: "error", errText: decryptErr, wantMarked: false, wantStatus: "error", wantError: decryptErr},
		{name: "fixture with wrong-case status", demoTenant: true, chType: "zalo_oa", external: "demo-zalo-oa", creds: plain, status: "Error", errText: decryptErr, wantMarked: true, wantStatus: "Error", wantError: decryptErr},
		{name: "fixture with wrong-case error prefix", demoTenant: true, chType: "zalo_oa", external: "demo-zalo-oa", creds: plain, status: "error", errText: "Decrypt failed: decrypt: cipher: message authentication failed", wantMarked: true, wantStatus: "error", wantError: "Decrypt failed: decrypt: cipher: message authentication failed"},
		{name: "near-miss credential bytes", demoTenant: true, chType: "zalo_oa", external: "demo-zalo-oa", creds: []byte(`{"demo": true}`), status: "error", errText: decryptErr, wantMarked: false, wantStatus: "error", wantError: decryptErr},
	}
	ids := make([]string, len(cases))
	tenants := make([]string, len(cases))
	t.Cleanup(func() {
		for i := range cases {
			DB.Exec("DELETE FROM channels WHERE id = ?", ids[i])
			DB.Exec("DELETE FROM tenants WHERE id = ?", tenants[i])
		}
	})
	for i, c := range cases {
		tenants[i] = "fxmig-" + suffix + "-" + string(rune('a'+i))
		ids[i] = "ch-fxmig-" + suffix + "-" + string(rune('a'+i))
		settings := `{}`
		if c.demoTenant {
			settings = `{"is_demo_data":true}`
		}
		if err := DB.Exec(`INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Fixture Migration', ?, ?, NOW(), NOW())`, tenants[i], tenants[i], settings).Error; err != nil {
			t.Fatal(err)
		}
		var syncedAt interface{}
		if c.syncedAt != "" {
			syncedAt = c.syncedAt
		}
		var runID interface{}
		if c.runID != "" {
			runID = c.runID
		}
		lease := "NULL"
		if c.leaseSet {
			lease = "NOW(3) + INTERVAL 5 MINUTE"
		}
		if err := DB.Exec(`INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, is_demo_fixture, last_sync_at, last_sync_status, last_sync_error, sync_run_id, sync_lease_until, metadata, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, true, false, ?, ?, ?, ?, `+lease+`, '{}', '2026-01-01 00:00:00', '2026-01-02 00:00:00')`,
			ids[i], tenants[i], c.chType, c.name, c.external, c.creds, syncedAt, c.status, c.errText, runID).Error; err != nil {
			t.Fatalf("seed %q: %v", c.name, err)
		}
	}
	before := make([]channelState, len(cases))
	for i := range cases {
		before[i] = readChannelState(t, ids[i])
	}

	var afterFirst []channelState
	for pass := 1; pass <= 2; pass++ {
		if err := AutoMigrate(); err != nil {
			t.Fatalf("pass %d: %v", pass, err)
		}
		states := make([]channelState, len(cases))
		for i, c := range cases {
			got := readChannelState(t, ids[i])
			states[i] = got
			if got.Marked != c.wantMarked {
				t.Fatalf("pass %d %q: marked = %v, want %v", pass, c.name, got.Marked, c.wantMarked)
			}
			if got.Status != c.wantStatus || got.Error != c.wantError {
				t.Fatalf("pass %d %q: status/error = %q/%q, want %q/%q", pass, c.name, got.Status, got.Error, c.wantStatus, c.wantError)
			}
			// Never touched: checkpoint, run id, lease and updated_at.
			if got.LastSyncAt != before[i].LastSyncAt || got.RunID != before[i].RunID || got.LeaseSet != before[i].LeaseSet || got.UpdatedAt != before[i].UpdatedAt {
				t.Fatalf("pass %d %q: checkpoint/run/lease/updated_at changed: %+v -> %+v", pass, c.name, before[i], got)
			}
		}
		if pass == 1 {
			afterFirst = states
		} else if !reflect.DeepEqual(afterFirst, states) {
			t.Fatalf("second AutoMigrate changed rows: %+v -> %+v", afterFirst, states)
		}
	}
}
