package handlers

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm/logger"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// CCMAI-RUNTIME-014 R014-R1 (handler side): the manual panic write and the
// manual reservation must not print the run ID or SQL values into GORM's real
// output sink, and a failed write reaches workers and agents only as a bounded
// class. The sink is a buffer behind a logger of GORM's own kind, observed at
// the three levels where the default logger prints SQL.

type handlerSinkConfig struct {
	name       string
	level      logger.LogLevel
	slow       time.Duration
	probeFails bool
}

var handlerSinkConfigs = []handlerSinkConfig{
	{"info", logger.Info, 200 * time.Millisecond, false},
	{"warn on error", logger.Warn, time.Hour, true},
	{"warn on slow query", logger.Warn, time.Nanosecond, false},
}

func withHandlerGormSink(cfg handlerSinkConfig, fn func()) string {
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

func TestHandlerGormSinkObserverSeesOrdinaryStatements(t *testing.T) {
	f := setupSyncStartFixture(t)
	const marker = "handler-sink-detector-marker-014"
	for _, cfg := range handlerSinkConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			sink := withHandlerGormSink(cfg, func() {
				var n int64
				if cfg.probeFails {
					_ = db.DB.Model(&models.Channel{}).Where("no_such_column = ? AND id = ?", marker, f.channelID).Count(&n).Error
				} else {
					_ = db.DB.Model(&models.Channel{}).Where("sync_run_id = ? AND id = ?", marker, f.channelID).Count(&n).Error
				}
			})
			if !strings.Contains(sink, marker) {
				t.Fatalf("the observer did not capture an ordinary statement at %s; sink: %q", cfg.name, sink)
			}
		})
	}
}

func TestManualRunIDWritesNeverReachTheGormSink(t *testing.T) {
	cfgEngine := &config.Config{Env: "test", EncryptionKey: "synthetic-32-byte-key-0123456789"}
	for _, cfg := range handlerSinkConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			f := setupSyncStartFixture(t)
			appLogs := captureLog(t)
			var runIDs, visible []string

			sink := withHandlerGormSink(cfg, func() {
				// accepted manual request: reservation write through the handler
				if rec := f.callSync(f.tenantID); rec.Code != 202 {
					t.Errorf("manual sync: %d %s", rec.Code, rec.Body.String())
				}
				runIDs = append(runIDs, f.launchedRes.RunID)
				// successful panic write with the owner's ID
				if err := handleManualSyncPanic(f.launchedRes, panicSecret); err != nil {
					t.Errorf("owner panic write: %v", err)
				}
			})
			for _, needle := range append([]string{"sync_run_id"}, runIDs...) {
				if strings.Contains(sink, needle) {
					t.Fatalf("%q reached the GORM sink at %s: %q", needle, cfg.name, sink)
				}
			}
			if f.launchedRes.RunID == "" {
				t.Fatal("no run id was issued; the scenario is vacuous")
			}

			// failed panic write (forced DB error) and a stale write
			res := f.reserve(t)
			f.addChannelTrigger(t, "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced panic status write failure';")
			sink = withHandlerGormSink(cfg, func() {
				if err := handleManualSyncPanic(res, panicSecret); err != nil {
					visible = append(visible, err.Error())
				} else {
					t.Error("failed panic write reported success")
				}
				// a worker run with the same reservation: its terminal writes fail too
				runManualSync(res, f.channelStatus(t), cfgEngine)
			})
			for _, needle := range []string{"sync_run_id", res.RunID, "forced", "Error 1644"} {
				for name, text := range map[string]string{"gorm sink": sink, "application log": appLogs.String(), "returned error": strings.Join(visible, ";")} {
					if strings.Contains(text, needle) {
						t.Fatalf("%q reached the %s at %s: %q", needle, name, cfg.name, text)
					}
				}
			}
			if len(visible) != 1 || visible[0] != "update sync status: write failed" {
				t.Fatalf("returned errors %v, want the bounded write-failed class", visible)
			}
		})
	}
}

// Agent results print engine errors; a failed terminal write must surface only
// the bounded class there.
func TestAgentResultNeverCarriesRawWriteErrors(t *testing.T) {
	f := setupSyncStartFixture(t)
	f.addChannelTrigger(t, "IF OLD.last_sync_status = 'syncing' AND NEW.last_sync_status <> 'syncing' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'forced final write failure'; END IF;")
	cfg := &config.Config{Env: "test", EncryptionKey: "synthetic-32-byte-key-0123456789"}

	res := handleSyncAgent(context.Background(), cfg, AgentRunRequest{
		TenantID: f.tenantID, Action: "sync_channel", Params: map[string]interface{}{"channel_id": f.channelID},
	})

	if res.Status != "error" || len(res.Errors) != 1 {
		t.Fatalf("got %+v, want one error", res)
	}
	for _, needle := range []string{"forced", "Error 1644", "sync_run_id"} {
		if strings.Contains(res.Errors[0], needle) {
			t.Fatalf("%q in the agent-visible error %q", needle, res.Errors[0])
		}
	}
	if !strings.Contains(res.Errors[0], "write failed") {
		t.Fatalf("agent error %q does not name the bounded class", res.Errors[0])
	}
}
