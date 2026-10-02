package engine

import (
	"context"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
)

// Independent R027 probe: every explicit mode preserves last_run_at but records job terminal status.
func TestF05ReviewTestRunRecordsTerminalJobStatus(t *testing.T) {
	forModes(t, func(t *testing.T, batch bool) {
		f := setupIncFixture(t, batch, "qc_analysis")
		f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-time.Hour)})
		sentinel := f.clock.Add(-48 * time.Hour).Truncate(time.Second)
		f.exec(t, "UPDATE jobs SET last_run_at = ?, last_run_status = 'prior', updated_at = ? WHERE id = ?", sentinel, sentinel, f.jobID)
		an := NewAnalyzerWithProvider(&config.Config{}, &incProvider{})
		run, err := an.RunJobWithLimit(context.Background(), f.job(t), 1)
		if err != nil || run == nil || run.Status != "success" {
			t.Fatalf("accepted test run: %v %+v", err, run)
		}
		after := f.job(t)
		if after.LastRunStatus != "success" || !after.UpdatedAt.After(sentinel) {
			t.Errorf("SPEC terminal bookkeeping missing: last_run_status=%q updated_at=%v", after.LastRunStatus, after.UpdatedAt)
		}
		if after.LastRunAt == nil || !after.LastRunAt.Equal(sentinel) {
			t.Errorf("test run changed ordinary checkpoint: %v", after.LastRunAt)
		}
	})
}
