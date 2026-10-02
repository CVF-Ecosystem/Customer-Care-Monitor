package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
)

// CCMAI-RUNTIME-028-R2 (F06-R2-02): through the real handlers and the real worker, a run whose
// terminal state is already stored keeps ownership until the worker has exited (completion
// activity, notification and cleanup). A launch inside that tail is 409; once the worker releases
// ownership the same launch is 202. The tail is blocked deterministically: a BEFORE INSERT trigger
// on activity_logs waits on a named MySQL lock that the test holds on a dedicated connection and
// releases explicitly. The only polling is observation of the stored terminal state; no sleep or retry
// stands in for the ordering being asserted.

func TestRouteTerminalRunHoldsOwnershipUntilTheWorkerExits(t *testing.T) {
	fx := setupOwnership(t, true, 1)
	lockName := "r028r2_tail_" + fx.jobID[len(fx.jobID)-8:]

	// dedicated connection that holds the named lock
	sqlDB, err := db.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	released := false
	release := func() {
		if released {
			return
		}
		released = true
		var ok sql.NullInt64
		_ = conn.QueryRowContext(context.Background(), "SELECT RELEASE_LOCK(?)", lockName).Scan(&ok)
		_ = conn.Close()
	}
	t.Cleanup(release)
	var got int
	if err := conn.QueryRowContext(context.Background(), "SELECT GET_LOCK(?, 10)", lockName).Scan(&got); err != nil || got != 1 {
		t.Fatalf("could not take the barrier lock: %d %v", got, err)
	}
	trig := "r028r2_tail_" + fx.jobID[len(fx.jobID)-6:]
	if err := db.DB.Exec(fmt.Sprintf(`CREATE TRIGGER %s BEFORE INSERT ON activity_logs FOR EACH ROW
BEGIN
  IF NEW.action = 'job.run.completed' AND NEW.tenant_id = '%s' THEN
    SET @r028r2_got = GET_LOCK('%s', 60);
    DO RELEASE_LOCK('%s');
  END IF;
END`, trig, fx.tenantID, lockName, lockName)).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trig) })
	// registered last, so it runs first: free the worker, then join it before the fixture teardown
	t.Cleanup(func() {
		release()
		fx.prov.release()
		fx.waitIdle(t)
	})

	rec := fx.callTrigger(fx.tenantID, "mode=unanalyzed")
	runID := bodyField(t, rec, "run_id")
	if rec.Code != http.StatusAccepted || runID == "" {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	fx.prov.waitEntered(t)
	fx.prov.release()

	// the terminal state is stored (terminal commit precedes the completion activity)...
	deadline := time.Now().Add(30 * time.Second)
	for fx.runStatus(t, runID) == "running" {
		if time.Now().After(deadline) {
			t.Fatal("the run never reached its terminal state")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := fx.runStatus(t, runID); got != "success" {
		t.Fatalf("terminal state %q", got)
	}
	// ...but the worker is blocked in its tail, so ownership is still held and a launch is refused
	if !engine.JobRunActive(fx.tenantID, fx.jobID) {
		t.Fatal("ownership was released before the worker finished its tail")
	}
	rows := fx.jobRunCount(t)
	for name, r := range map[string]*httptest.ResponseRecorder{
		"trigger":  fx.callTrigger(fx.tenantID, "mode=unanalyzed"),
		"test-run": fx.callTestRun(fx.tenantID),
	} {
		if r.Code != http.StatusConflict || bodyField(t, r, "error") != "job_already_running" {
			t.Fatalf("%s during the tail: %d %s", name, r.Code, r.Body.String())
		}
	}
	if fx.jobRunCount(t) != rows {
		t.Fatal("a launch during the tail created a run row")
	}

	// release the barrier: the worker exits, ownership is released, the same launch is accepted
	release()
	fx.waitIdle(t)
	if next := fx.callTrigger(fx.tenantID, "mode=unanalyzed"); next.Code != http.StatusAccepted {
		t.Fatalf("launch after the worker exited: %d %s", next.Code, next.Body.String())
	}
	var completed int64
	db.DB.Model(&models.ActivityLog{}).Where("tenant_id = ? AND action = 'job.run.completed'", fx.tenantID).Count(&completed)
	if completed < 1 {
		t.Fatal("the completion activity of the first run was never written")
	}
}
