package engine

import (
	"encoding/json"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"sync"
	"testing"
	"time"
)

// This source-negative detector pauses at the second actual batch-provider call and
// inspects the live run row after the first provider error. The old implementation
// continues before writing the common progress summary.
func TestR091OldBatchProviderErrorPersistsProgressBeforeNextBatch(t *testing.T) {
	f := setupIncFixture(t, true, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "first", []time.Time{f.clock.Add(-2 * time.Hour)})
	f.addConv(t, f.tenantID, f.channelID, "second", []time.Time{f.clock.Add(-time.Hour)})
	f.exec(t, "UPDATE app_settings SET value_plain = '1' WHERE tenant_id = ? AND setting_key = 'ai_batch_size'", f.tenantID)
	enteredSecond := make(chan struct{})
	releaseSecond := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(releaseSecond) })
	var firstCallAt time.Time
	var secondCallDelay time.Duration
	p := &incProvider{fail: true, onCall: func(n int) {
		if n == 1 {
			firstCallAt = time.Now()
		}
		if n == 2 {
			secondCallDelay = time.Since(firstCallAt)
			close(enteredSecond)
			<-releaseSecond
		}
	}}
	done := make(chan struct{})
	go func() { defer close(done); _, _ = f.run(t, p) }()
	select {
	case <-enteredSecond:
	case <-time.After(40 * time.Second):
		t.Fatal("synthetic provider did not reach the second batch; this is a setup/timeout failure, not the source detector")
	}
	if secondCallDelay < 8*time.Second {
		t.Errorf("behavioral detector: provider-error batch skipped the common 10-second delay; next call began after %s", secondCallDelay)
	}
	var run models.JobRun
	if err := db.DB.Where("job_id = ?", f.jobID).Order("started_at DESC").First(&run).Error; err != nil {
		t.Fatalf("read active synthetic run: %v", err)
	}
	var summary map[string]any
	if err := json.Unmarshal([]byte(run.Summary), &summary); err != nil {
		t.Fatalf("decode active progress summary %q: %v", run.Summary, err)
	}
	if n, _ := summary["conversations_errors"].(float64); n < 1 {
		t.Errorf("behavioral detector: first provider batch failed but progress before batch two reports conversations_errors=%v", summary["conversations_errors"])
	}
	releaseOnce.Do(func() { close(releaseSecond) })
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("synthetic analyzer did not finish after releasing the provider; outer timeout is not detector evidence")
	}
}
