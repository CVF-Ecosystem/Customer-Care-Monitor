package mcp

// Independent R044 reviewer probe. Replay only in an isolated exact-BUILD archive's mcp package.
// Synthetic provider, disposable MySQL and an in-process HTTP transport; no Telegram call occurs.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
)

type r044NotificationTransport struct {
	entered chan struct{}
	release chan struct{}
	once sync.Once
	mu sync.Mutex
	calls int
}

func (p *r044NotificationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "POST" || req.URL.Host != "api.telegram.org" || req.URL.Path != "/botR044_SYNTHETIC/sendMessage" {
		return nil, context.Canceled // never delegate to a real transport
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || payload["chat_id"] != "R044_LOCAL" {
		return nil, context.Canceled
	}
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	p.once.Do(func() { close(p.entered) })
	select {
	case <-p.release:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Request: req}, nil
}

func TestR044ReviewerNotificationTail(t *testing.T) {
	f := newMCPFixture(t)
	f.admit(t)
	transport := &r044NotificationTransport{entered: make(chan struct{}), release: make(chan struct{})}
	original := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })
	f.useRealWorker(t, newSynthProvider(false))
	t.Cleanup(func() {
		select { case <-transport.release: default: close(transport.release) }
	})
	f.exec(t, `UPDATE jobs SET output_schedule = 'immediate', outputs = ? WHERE id = ?`, `[{"type":"telegram","bot_token":"R044_SYNTHETIC","chat_id":"R044_LOCAL"}]`, f.jobID())
	f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at) VALUES (?, ?, 'app_url', 'http://r044.invalid', NOW(), NOW())`, "url-"+f.tenantA, f.tenantA)
	_, response := f.mountedCall(t, context.Background(), f.triggerBody())
	accepted := decodeAccepted(t, "notification-enabled", response)
	select {
	case <-transport.entered:
	case <-time.After(30*time.Second): t.Fatal("configured notification never reached the in-process transport")
	}
	if f.runRow(t, accepted.RunID).Status != "success" || !engine.JobRunActive(f.tenantA, f.jobID()) {
		t.Fatal("terminal run must retain ownership while notification is blocked")
	}
	before := f.countRows(t, `SELECT COUNT(*) FROM job_runs WHERE job_id = ?`, f.jobID())
	if got := f.mountedCallStatus(t, f.triggerBody()); got != "job_already_running" {
		t.Fatalf("during notification got %q, want busy", got)
	}
	if f.countRows(t, `SELECT COUNT(*) FROM job_runs WHERE job_id = ?`, f.jobID()) != before {
		t.Fatal("notification-tail denial created another run")
	}
	close(transport.release)
	f.waitIdle(t, f.tenantA, f.jobID())
	if got := f.countRows(t, `SELECT COUNT(*) FROM notification_logs WHERE job_run_id = ? AND tenant_id = ? AND status = 'sent'`, accepted.RunID, f.tenantA); got != 1 {
		t.Fatalf("sent notification rows for accepted run: %d, want 1", got)
	}
	var notified int64
	if err := db.DB.Raw(`SELECT COUNT(*) FROM job_results WHERE job_run_id = ? AND notified_at IS NOT NULL`, accepted.RunID).Scan(&notified).Error; err != nil || notified < 1 {
		t.Fatalf("notified results: %d, error %v", notified, err)
	}
	transport.mu.Lock()
	calls := transport.calls
	transport.mu.Unlock()
	if calls != 1 { t.Fatalf("intercepted requests %d, want 1", calls) }
	t.Log("synthetic notification intercepted once; terminal owner held until send/log/result-mark tail completed; no external transport used")
}
