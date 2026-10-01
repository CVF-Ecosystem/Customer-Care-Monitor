package engine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

func TestOrdinaryReviewCancellationAtCompletion(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, empty := range []bool{false, true} {
			t.Run(fmt.Sprintf("batch=%v/empty=%v", batch, empty), func(t *testing.T) {
				f := setupIncFixture(t, batch, "qc_analysis")
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				p := &incProvider{}
				if empty {
					cancel()
				} else {
					f.addConv(t, f.tenantID, f.channelID, "last", []time.Time{f.clock.Add(-time.Hour)})
					p.onCall = func(int) { cancel() }
				}
				before := f.job(t).LastRunAt
				run, err := NewAnalyzer(&config.Config{}).RunJobWithProvider(ctx, f.job(t), 0, p)
				after := f.job(t).LastRunAt
				if err != nil {
					t.Fatalf("run error: %v", err)
				}
				if run.Status == "success" || (before == nil) != (after == nil) || (before != nil && !before.Equal(*after)) {
					t.Fatalf("canceled run advanced or succeeded: status=%s before=%v after=%v", run.Status, before, after)
				}
			})
		}
	}
}

func TestOrdinaryReviewSuppressedCheckpointWrite(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	job := f.job(t)
	prior := f.clock.Add(-48 * time.Hour).Truncate(time.Second)
	f.exec(t, "UPDATE jobs SET last_run_at=?,last_run_status='idle' WHERE id=?", prior, f.jobID)
	job = f.job(t)
	trigger := "r025_review_" + f.jobID[len(f.jobID)-6:]
	f.exec(t, fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE ON jobs FOR EACH ROW BEGIN IF NEW.id='%s' THEN SET NEW.last_run_at=OLD.last_run_at, NEW.last_run_status=OLD.last_run_status, NEW.updated_at=OLD.updated_at; END IF; END", trigger, f.jobID))
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trigger) })
	run, err := NewAnalyzer(&config.Config{}).RunJobWithProvider(context.Background(), job, 0, &incProvider{})
	if err == nil || run.Status == "success" {
		t.Fatalf("suppressed checkpoint write accepted: status=%s error=%v", run.Status, err)
	}
	if after := f.job(t).LastRunAt; after == nil || !after.Equal(prior) {
		t.Fatal("checkpoint changed")
	}
}

func TestOrdinaryReviewSameSecondReplaySucceeds(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	a := NewAnalyzer(&config.Config{})
	for i := 0; i < 2; i++ {
		run, err := a.RunJobWithProvider(context.Background(), f.job(t), 0, &incProvider{})
		if err != nil || run.Status != "success" {
			t.Fatalf("identical same-second job update must succeed: status=%s err=%v", run.Status, err)
		}
		var stored models.JobRun
		if err := db.DB.First(&stored, "id = ?", run.ID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.FinishedAt == nil || stored.FinishedAt.Before(stored.StartedAt) {
			t.Fatalf("completion precedes start: start=%v finish=%v", stored.StartedAt, stored.FinishedAt)
		}
	}
}
