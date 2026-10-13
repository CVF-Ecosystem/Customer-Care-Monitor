package notifications

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

func r091DeliveryFixture(t *testing.T, outputs string) (models.Job, models.JobRun, models.JobResult) {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Fatal("R091 outbound tests require TEST_DB_DSN; database setup failure is not a skip")
	}
	db.Close()
	if err := db.Connect(dsn, true); err != nil {
		t.Fatalf("connect owned synthetic MySQL: %v", err)
	}
	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("migrate owned synthetic MySQL: %v", err)
	}
	tenantID, jobID, runID, resultID := pkg.NewUUID(), pkg.NewUUID(), pkg.NewUUID(), pkg.NewUUID()
	now := time.Now().UTC()
	tenant := models.Tenant{ID: tenantID, Name: "R091 synthetic", Slug: "r091-" + strings.ReplaceAll(tenantID, "-", ""), Settings: "{}", CreatedAt: now, UpdatedAt: now}
	job := models.Job{ID: jobID, TenantID: tenantID, Name: "R091 synthetic job", JobType: "qc_analysis", InputChannelIDs: "[]", RulesConfig: "[]", Outputs: outputs, IsActive: true, CreatedAt: now, UpdatedAt: now}
	run := models.JobRun{ID: runID, TenantID: tenantID, JobID: jobID, StartedAt: now, Status: "success", Summary: `{"conversations_analyzed":1,"conversations_passed":0,"issues_found":1}`, CreatedAt: now}
	result := models.JobResult{ID: resultID, JobRunID: runID, TenantID: tenantID, ConversationID: pkg.NewUUID(), ResultType: "qc_violation", Severity: "HIGH", RuleName: "Synthetic", Evidence: "synthetic evidence", Detail: `{}`, CreatedAt: now}
	if err := db.DB.Create(&tenant).Error; err != nil {
		t.Fatalf("create synthetic tenant: %v", err)
	}
	if err := db.DB.Create(&job).Error; err != nil {
		t.Fatalf("create synthetic job: %v", err)
	}
	if err := db.DB.Create(&run).Error; err != nil {
		t.Fatalf("create synthetic run: %v", err)
	}
	if err := db.DB.Create(&result).Error; err != nil {
		t.Fatalf("create synthetic result: %v", err)
	}
	t.Cleanup(func() {
		if db.DB != nil {
			for _, table := range []string{"notification_logs", "job_results", "job_runs", "jobs"} {
				db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenantID)
			}
			db.DB.Where("id = ?", tenantID).Delete(&models.Tenant{})
		}
		db.Close()
	})
	return job, run, result
}

func r091SMTPServer(t *testing.T) (int, <-chan struct{}) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on synthetic SMTP loopback: %v", err)
	}
	accepted := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		w := bufio.NewWriter(conn)
		write := func(s string) bool {
			if _, e := w.WriteString(s); e != nil {
				return false
			}
			return w.Flush() == nil
		}
		if !write("220 synthetic.local ESMTP\r\n") {
			return
		}
		for {
			line, e := r.ReadString('\n')
			if e != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO "):
				if !write("250-synthetic.local\r\n250-AUTH PLAIN\r\n250 OK\r\n") {
					return
				}
			case strings.HasPrefix(cmd, "HELO "), strings.HasPrefix(cmd, "MAIL FROM:"), strings.HasPrefix(cmd, "RCPT TO:"), cmd == "RSET", cmd == "NOOP":
				if !write("250 OK\r\n") {
					return
				}
			case strings.HasPrefix(cmd, "AUTH "):
				if !write("235 Authentication successful\r\n") {
					return
				}
			case cmd == "DATA":
				if !write("354 end with dot\r\n") {
					return
				}
				for {
					dataLine, readErr := r.ReadString('\n')
					if readErr != nil {
						return
					}
					if strings.TrimRight(dataLine, "\r\n") == "." {
						break
					}
				}
				if !write("250 accepted\r\n") {
					return
				}
				close(accepted)
			case cmd == "QUIT":
				write("221 bye\r\n")
				return
			default:
				if !write("250 OK\r\n") {
					return
				}
			}
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Errorf("synthetic SMTP server goroutine did not join")
		}
	})
	return ln.Addr().(*net.TCPAddr).Port, accepted
}

func TestR091OldDispatcherDoesNotMarkAllFailedOutputDelivered(t *testing.T) {
	job, run, result := r091DeliveryFixture(t, `[{"type":"unsupported"}]`)
	err := NewDispatcher().SendJobResults(context.Background(), job, run)
	if err == nil {
		t.Errorf("behavioral detector: all outputs were rejected but SendJobResults returned nil")
	}
	var stored models.JobResult
	if e := db.DB.First(&stored, "id = ?", result.ID).Error; e != nil {
		t.Fatalf("read synthetic result: %v", e)
	}
	if stored.NotifiedAt != nil {
		t.Errorf("behavioral detector: rejected output marked result notified at %s", stored.NotifiedAt)
	}
}

func TestR091OldDispatcherDoesNotMarkMixedDeliverySetComplete(t *testing.T) {
	port, accepted := r091SMTPServer(t)
	job, run, result := r091DeliveryFixture(t, fmt.Sprintf(`[{"type":"email","smtp_host":"127.0.0.1","smtp_port":%d,"from":"sender@example.invalid","to":"recipient@example.invalid"},{"type":"unsupported"}]`, port))
	err := NewDispatcher().SendJobResults(context.Background(), job, run)
	if err == nil {
		t.Errorf("behavioral detector: mixed accepted/rejected outputs returned nil")
	}
	select {
	case <-accepted:
	case <-time.After(2 * time.Second):
		t.Fatalf("synthetic SMTP acceptance control did not execute")
	}
	var stored models.JobResult
	if e := db.DB.First(&stored, "id = ?", result.ID).Error; e != nil {
		t.Fatalf("read synthetic result: %v", e)
	}
	if stored.NotifiedAt != nil {
		t.Errorf("behavioral detector: one accepted plus one rejected output marked result notified at %s", stored.NotifiedAt)
	}
	if reentryErr := NewDispatcher().SendJobResults(context.Background(), job, run); reentryErr == nil {
		t.Errorf("behavioral detector: re-entry did not retry the proven failed output while preserving the accepted output receipt")
	}
}

func TestR091OldTelegramErrorDoesNotExposeToken(t *testing.T) {
	const token = "SYNTHETIC_TOKEN_R091"
	n := NewTelegramNotifier(token, "synthetic-chat")
	n.client = &http.Client{Transport: r091RoundTripper(func(*http.Request) (*http.Response, error) { return nil, fmt.Errorf("synthetic transport unavailable") })}
	err := n.Send(context.Background(), "subject", "body")
	if err == nil {
		t.Fatal("expected synthetic transport failure")
	}
	if strings.Contains(err.Error(), token) {
		t.Errorf("behavioral detector: notifier error exposed the bot token")
	}
}

type r091RoundTripper func(*http.Request) (*http.Response, error)

func (fn r091RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }
