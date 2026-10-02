package engine

import (
	"bytes"
	"context"
	"errors"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"gorm.io/gorm"
	"log"
	"strings"
	"testing"
)

// The F06-07/R028-R1-01 error contract applies to the application log sink too.
// This injects harmless synthetic driver detail, never a real credential or provider response.
func TestReviewF06EarlyFailureBoundsDriverDetailInAppLogs(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	const detail = "SYNTHETIC_SQL_DRIVER_DETAIL"
	const callback = "r028_review_driver_detail"
	if err := db.DB.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "conversations" {
			tx.AddError(errors.New(detail))
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer db.DB.Callback().Query().Remove(callback)
	originalWriter := log.Writer()
	var sink bytes.Buffer
	log.SetOutput(&sink)
	defer log.SetOutput(originalWriter)
	p := &incProvider{}
	run, err := f.analyzerWith(p).RunJob(context.Background(), f.job(t))
	if err == nil || run == nil || run.Status != "error" || p.callCount() != 0 {
		t.Fatalf("fault did not reach early failure: err=%v run=%v calls=%d", err, run, p.callCount())
	}
	if strings.Contains(err.Error(), detail) || strings.Contains(f.runRow(t, run.ID).ErrorMessage, detail) {
		t.Error("raw driver detail reached public/stored outcome")
	}
	if strings.Contains(sink.String(), detail) {
		t.Error("raw driver detail reached standard application log through failOwnedRun")
	}
}
