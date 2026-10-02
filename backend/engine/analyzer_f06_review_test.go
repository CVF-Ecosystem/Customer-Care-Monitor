package engine

import (
	"context"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"testing"
	"time"
)

// Independent R028 review probes: accepted cancellation must govern every terminal path.
func TestF06ReviewAcceptedCancelBeforeEarlyFailure(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	sentinel := f.setSentinel(t)
	job := f.job(t)
	job.InputChannelIDs = "{"
	res, err := ReserveJobRun(context.Background(), job, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = CancelJobRun(job.TenantID, job.ID, res.RunID()); err != nil {
		t.Fatal(err)
	}
	p := &incProvider{}
	_, _ = f.analyzerWith(p).RunReserved(res, job, "unanalyzed", 0, "", "")
	stored := f.runRow(t, res.RunID())
	if stored.Status != "cancelled" || stored.FinishedAt == nil || f.job(t).LastRunStatus != "cancelled" {
		t.Errorf("accepted cancel lost to early failure: run=%s job=%s finished=%v", stored.Status, f.job(t).LastRunStatus, stored.FinishedAt)
	}
	if cp := f.checkpoint(t); cp == nil || !cp.Equal(sentinel) || p.callCount() != 0 {
		t.Errorf("checkpoint=%v provider calls=%d", cp, p.callCount())
	}
}

func TestF06ReviewAcceptedCancelBeforeProviderPanic(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "review", []time.Time{f.clock.Add(-time.Hour)})
	job := f.job(t)
	p := &incProvider{}
	p.onCall = func(n int) {
		if _, err := CancelJobRun(job.TenantID, job.ID, ""); err != nil {
			t.Errorf("cancel: %v", err)
		}
		panic("synthetic review panic")
	}
	_, _ = f.analyzerWith(p).RunJob(context.Background(), job)
	var rows []models.JobRun
	if err := db.DB.Where("job_id = ? AND tenant_id = ?", job.ID, job.TenantID).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Status != "cancelled" || f.job(t).LastRunStatus != "cancelled" {
		t.Errorf("accepted cancel lost to panic: rows=%+v job=%s", rows, f.job(t).LastRunStatus)
	}
}

func TestF06ReviewReservationCannotExecuteAnotherJob(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "review", []time.Time{f.clock.Add(-time.Hour)})
	a := f.job(t)
	b := a
	b.ID = pkg.NewUUID()
	b.Name = "review other job"
	if err := db.DB.Create(&b).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.DB.Where("job_id = ?", b.ID).Delete(&models.JobRun{})
		db.DB.Where("id = ?", b.ID).Delete(&models.Job{})
	})
	res, err := ReserveJobRun(context.Background(), a, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Abort(a, "review cleanup")
	p := &incProvider{}
	_, err = f.analyzerWith(p).RunReserved(res, b, "unanalyzed", 0, "", "")
	var results int64
	db.DB.Model(&models.JobResult{}).Where("job_run_id = ?", res.RunID()).Count(&results)
	if err == nil || p.callCount() != 0 || results != 0 {
		t.Errorf("reservation A executed B: err=%v provider=%d published=%d", err, p.callCount(), results)
	}
}
