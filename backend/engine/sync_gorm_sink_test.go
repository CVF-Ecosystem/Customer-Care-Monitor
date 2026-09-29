package engine

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm/logger"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// CCMAI-RUNTIME-014 R014-R1: the run ID and the SQL values of the reservation
// and the run-ID-fenced terminal writes must not reach GORM's real output
// sink. GORM's logger has its own writer, separate from the application's
// log.Writer(), so these tests replace db.DB.Logger with a logger of the same
// kind whose writer is a buffer, and run each scenario at the three levels at
// which the default logger prints SQL: Info (development), Warn on errors
// (production) and Warn on slow queries.

type gormSinkConfig struct {
	name  string
	level logger.LogLevel
	slow  time.Duration
	// probeFails: the positive detector needs a failing statement when the
	// level only prints errors.
	probeFails bool
}

var gormSinkConfigs = []gormSinkConfig{
	{"info", logger.Info, 200 * time.Millisecond, false},
	{"warn on error", logger.Warn, time.Hour, true},
	{"warn on slow query", logger.Warn, time.Nanosecond, false},
}

// withGormSink runs fn with db.DB.Logger writing into the returned buffer and
// restores the previous logger afterwards. It must not wrap fixture setup or
// assertions, only the calls under observation.
func withGormSink(cfg gormSinkConfig, fn func()) string {
	var buf bytes.Buffer
	previous := db.DB.Logger
	db.DB.Logger = logger.New(log.New(&buf, "", 0), logger.Config{
		LogLevel:                  cfg.level,
		SlowThreshold:             cfg.slow,
		IgnoreRecordNotFoundError: true,
	})
	defer func() { db.DB.Logger = previous }()
	fn()
	return buf.String()
}

// Positive detector: the same sink, on an ordinary db.DB statement carrying a
// marker value, does show the value. Without this the absence assertions below
// would prove nothing.
func TestGormSinkObserverSeesOrdinaryStatements(t *testing.T) {
	f := setupSFFixture(t)
	const marker = "sink-detector-marker-014"
	for _, cfg := range gormSinkConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			sink := withGormSink(cfg, func() {
				var n int64
				if cfg.probeFails {
					_ = db.DB.Model(&models.Channel{}).Where("no_such_column = ? AND id = ?", marker, f.chA).Count(&n).Error
				} else {
					_ = db.DB.Model(&models.Channel{}).Where("sync_run_id = ? AND id = ?", marker, f.chA).Count(&n).Error
				}
			})
			if !strings.Contains(sink, marker) {
				t.Fatalf("the observer did not capture an ordinary statement at %s; sink: %q", cfg.name, sink)
			}
		})
	}
}

func TestRunIDWritesNeverReachTheGormSink(t *testing.T) {
	for _, cfg := range gormSinkConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			f := setupSFFixture(t)
			// chB: a failed final write; chX: a failed reservation.
			f.addTrigger(t, f.chB, "IF OLD.last_sync_status = 'syncing' AND NEW.last_sync_status <> 'syncing' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced final write failure'; END IF;")
			f.addTrigger(t, f.chX, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced reservation failure';")
			eng := NewSyncEngine(f.cfg)
			chA, chB := f.channel(t, f.chA), f.channel(t, f.chB)
			var runIDs []string
			var visibleErrs []string
			appLogs := captureEngineLog(t)

			sink := withGormSink(cfg, func() {
				// accepted run: reservation + success terminal write
				resA, err := ReserveChannelSync(f.tenantID, f.chA)
				if err != nil {
					t.Errorf("reserve A: %v", err)
					return
				}
				runIDs = append(runIDs, resA.RunID)
				if err := eng.SyncReservedChannel(context.Background(), chA, resA); err != nil {
					t.Errorf("run A: %v", err)
				}
				// failed terminal + deferred error writes
				resB, err := ReserveChannelSync(f.tenantID, f.chB)
				if err != nil {
					t.Errorf("reserve B: %v", err)
					return
				}
				runIDs = append(runIDs, resB.RunID)
				if err := eng.SyncReservedChannel(context.Background(), chB, resB); err != nil {
					visibleErrs = append(visibleErrs, err.Error())
				} else {
					t.Error("run B reported success although its final write failed")
				}
				// forged-ID write (zero rows) and failed reservation
				forged := resB
				forged.RunID = "forged-run-id-014"
				if _, err := eng.recordSyncStatus(forged, "success", ""); err != nil {
					visibleErrs = append(visibleErrs, err.Error())
				}
				if _, err := ReserveChannelSync(f.otherTenantID, f.chX); err != nil {
					visibleErrs = append(visibleErrs, err.Error())
				} else {
					t.Error("reservation with a failing write was admitted")
				}
			})

			forbidden := append([]string{"sync_run_id", "forged-run-id-014", "forced", "Error 1644"}, runIDs...)
			for _, needle := range forbidden {
				if strings.Contains(sink, needle) {
					t.Fatalf("%q reached the GORM sink at %s: %q", needle, cfg.name, sink)
				}
				for _, text := range append(visibleErrs, appLogs.String()) {
					if strings.Contains(text, needle) {
						t.Fatalf("%q reached an API-visible error or application log: %q", needle, text)
					}
				}
			}
			if len(runIDs) != 2 || len(visibleErrs) != 3 {
				t.Fatalf("scenario did not run fully: %d ids, %d errors", len(runIDs), len(visibleErrs))
			}
		})
	}
}

// The database failure keeps a bounded, generic class at the engine boundary
// that agents and workers print.
func TestRunIDWriteFailuresAreBoundedClasses(t *testing.T) {
	f := setupSFFixture(t)
	f.addTrigger(t, f.chA, "IF OLD.last_sync_status = 'syncing' AND NEW.last_sync_status <> 'syncing' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced final write failure'; END IF;")
	res := f.mustReserve(t, f.tenantID, f.chA)

	wrote, err := NewSyncEngine(f.cfg).recordSyncStatus(res, "success", "")

	if wrote || err == nil || err.Error() != "update sync status success: write failed" {
		t.Fatalf("wrote %v, err %v; want the bounded write-failed class", wrote, err)
	}
}
