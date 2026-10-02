package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-028-R2: the early-failure lifecycle path logs and stores a fixed bounded class only
// (F06-R2-01), and a terminal run keeps its slot until the worker has exited (F06-R2-02, engine
// layer; the handler layer has the matching route test).

// captureAppLog redirects the standard application log for the test and returns the buffer.
func captureAppLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	orig := log.Writer()
	var sink bytes.Buffer
	log.SetOutput(&sink)
	t.Cleanup(func() { log.SetOutput(orig) })
	return &sink
}

// failTable makes every query on table fail with a harmless synthetic driver detail.
func failTable(t *testing.T, table, detail string) {
	t.Helper()
	name := "r028r2_" + table
	if err := db.DB.Callback().Query().Before("gorm:query").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == table {
			tx.AddError(errors.New(detail))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.DB.Callback().Query().Remove(name) })
}

func TestEarlyFailureClassesNeverLogOrStoreTheUnderlyingCause(t *testing.T) {
	const detail = "SYNTHETIC_SQL_DRIVER_DETAIL"
	cases := []struct {
		name  string
		class string
		setup func(t *testing.T, f *incFixture) func() (*models.JobRun, error)
		// raw cause fragments that must not reach the log, the stored message or the returned error
		forbidden func(t *testing.T) []string
	}{
		{"candidate selection (ordinary)", "candidate_selection_failed",
			func(t *testing.T, f *incFixture) func() (*models.JobRun, error) {
				failTable(t, "conversations", detail)
				return func() (*models.JobRun, error) {
					return f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t))
				}
			}, func(*testing.T) []string { return []string{detail} }},
		{"candidate selection (explicit mode)", "candidate_selection_failed",
			func(t *testing.T, f *incFixture) func() (*models.JobRun, error) {
				failTable(t, "conversations", detail)
				return func() (*models.JobRun, error) {
					return f.analyzerWith(&incProvider{}).RunJobUnanalyzed(context.Background(), f.job(t), 0)
				}
			}, func(*testing.T) []string { return []string{detail} }},
		{"invalid input list", "input_channels_invalid",
			func(t *testing.T, f *incFixture) func() (*models.JobRun, error) {
				f.exec(t, `UPDATE jobs SET input_channel_ids = '{"a":1}' WHERE id = ?`, f.jobID)
				return func() (*models.JobRun, error) {
					return f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t))
				}
			}, func(*testing.T) []string { return []string{"cannot unmarshal", "json:"} }},
		{"provider selection (no key)", "provider_unavailable",
			func(t *testing.T, f *incFixture) func() (*models.JobRun, error) {
				return func() (*models.JobRun, error) {
					return NewAnalyzer(&config.Config{}).RunJob(context.Background(), f.job(t))
				}
			}, func(*testing.T) []string { return []string{"Settings > AI Config"} }},
		{"provider selection (undecryptable key)", "provider_unavailable",
			func(t *testing.T, f *incFixture) func() (*models.JobRun, error) {
				f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, value_encrypted, created_at, updated_at) VALUES (?, ?, 'ai_api_key', '', X'0102', NOW(), NOW())`, pkg.NewUUID(), f.tenantID)
				return func() (*models.JobRun, error) {
					return NewAnalyzer(&config.Config{}).RunJob(context.Background(), f.job(t))
				}
			}, func(*testing.T) []string {
				_, derr := pkg.Decrypt([]byte{1, 2}, "")
				if derr == nil {
					return []string{"failed to decrypt"}
				}
				return []string{derr.Error(), "failed to decrypt"}
			}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			f := setupIncFixture(t, false, "qc_analysis")
			f.twoConvs(t)
			call := c.setup(t, f)
			forbidden := c.forbidden(t)
			sink := captureAppLog(t)
			run, err := call()
			if err == nil || run == nil || run.Status != "error" || JobRunActive(f.tenantID, f.jobID) {
				t.Fatalf("the fault did not reach the early-failure path: run %+v err %v", run, err)
			}
			stored := f.runRow(t, run.ID)
			for _, bad := range forbidden {
				if strings.Contains(err.Error(), bad) || strings.Contains(stored.ErrorMessage, bad) {
					t.Errorf("raw cause %q reached the returned/stored outcome: %v / %q", bad, err, stored.ErrorMessage)
				}
				if strings.Contains(sink.String(), bad) {
					t.Errorf("raw cause %q reached the application log: %s", bad, sink.String())
				}
			}
			// the log still correlates: job id, run id and the fixed class
			want := fmt.Sprintf("job %s: run %s failed before analysis (class=%s)", f.jobID, run.ID, c.class)
			if !strings.Contains(sink.String(), want) {
				t.Errorf("correlation line %q missing from the log: %s", want, sink.String())
			}
			if stored.Status != "error" || stored.ErrorMessage == "" {
				t.Errorf("stored %+v", stored)
			}
		})
	}
}

// A terminal run keeps the slot while the completion tail (activity/notification/cleanup) runs: a
// launch during the tail is refused with ErrJobBusy although the row is already terminal, and is
// admitted only after the worker exits. Barrier-controlled, no sleeps.
func TestTerminalRunHoldsTheSlotUntilTheTailFinishes(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	nt := installNotifier(t)
	f.exec(t, "UPDATE jobs SET output_schedule = 'instant' WHERE id = ?", f.jobID)
	f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-time.Hour)})
	inTail, releaseTail := make(chan struct{}), make(chan struct{})
	nt.onSend = func(job models.Job, run models.JobRun) {
		close(inTail)
		<-releaseTail
	}
	bg := goRun(func() (*models.JobRun, error) {
		return f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t))
	})
	select {
	case <-inTail:
	case <-time.After(30 * time.Second):
		t.Fatal("the completion tail was never reached")
	}
	var stored models.JobRun
	if err := db.DB.Where("job_id = ?", f.jobID).Order("started_at DESC").First(&stored).Error; err != nil || stored.Status != "success" {
		t.Fatalf("the run must already be terminal in the tail: %v %+v", err, stored)
	}
	before := f.runsOf(t, f.jobID)
	if _, err := f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t)); !errors.Is(err, ErrJobBusy) {
		t.Fatalf("a launch during the tail must be refused: %v", err)
	}
	if f.runsOf(t, f.jobID) != before || !JobRunActive(f.tenantID, f.jobID) {
		t.Fatal("the refused launch left a row, or ownership was dropped while the tail ran")
	}
	close(releaseTail)
	bg.wait(t)
	if bg.err != nil || bg.run.Status != "success" || JobRunActive(f.tenantID, f.jobID) {
		t.Fatalf("after the tail: %v %+v active=%v", bg.err, bg.run, JobRunActive(f.tenantID, f.jobID))
	}
	if _, err := f.analyzerWith(&incProvider{}).RunJob(context.Background(), f.job(t)); err != nil {
		t.Fatalf("admission after the worker exited: %v", err)
	}
}
