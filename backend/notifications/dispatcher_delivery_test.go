package notifications

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm"
)

type r091SMTPGate struct {
	ch   chan struct{}
	once sync.Once
}

func r091NewSMTPGate() *r091SMTPGate { return &r091SMTPGate{ch: make(chan struct{})} }
func (g *r091SMTPGate) release() {
	if g != nil {
		g.once.Do(func() { close(g.ch) })
	}
}

type r091DeliverySMTPPlan struct {
	noGreeting          bool
	rejectEHLO          bool
	noAuth              bool
	stallCommand        string
	holdCommand         *r091SMTPGate
	replyByRecipient    map[string][]string
	defaultDataReply    string
	holdAfterData       *r091SMTPGate
	closeConnectionQuit bool
}

type r091DeliverySMTPServer struct {
	listener         net.Listener
	plans            []r091DeliverySMTPPlan
	connected        chan int
	commandSeen      chan string
	connectionClosed chan int
	dataReady        chan string
	accepted         chan string
	done             chan struct{}
	mu               sync.Mutex
	conns            []net.Conn
	count            int
	recipients       map[string]int
}

func r091StartDeliverySMTP(t *testing.T, plans ...r091DeliverySMTPPlan) *r091DeliverySMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on synthetic outbound SMTP: %v", err)
	}
	s := &r091DeliverySMTPServer{
		listener: ln, plans: plans, connected: make(chan int, 16), commandSeen: make(chan string, 16),
		connectionClosed: make(chan int, 16), dataReady: make(chan string, 16),
		accepted: make(chan string, 16), done: make(chan struct{}), recipients: make(map[string]int),
	}
	go func() {
		var handlers sync.WaitGroup
		defer close(s.done)
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				break
			}
			s.mu.Lock()
			s.count++
			attempt := s.count
			s.conns = append(s.conns, conn)
			s.mu.Unlock()
			planIndex := attempt - 1
			if planIndex >= len(plans) {
				planIndex = len(plans) - 1
			}
			plan := r091DeliverySMTPPlan{}
			if planIndex >= 0 {
				plan = plans[planIndex]
			}
			handlers.Add(1)
			go func() {
				defer handlers.Done()
				s.serve(conn, attempt, plan)
			}()
		}
		handlers.Wait()
	}()
	t.Cleanup(func() { s.closeAndJoin(t) })
	return s
}

func (s *r091DeliverySMTPServer) port() int { return s.listener.Addr().(*net.TCPAddr).Port }

func (s *r091DeliverySMTPServer) connectionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

func (s *r091DeliverySMTPServer) serve(conn net.Conn, attempt int, plan r091DeliverySMTPPlan) {
	defer conn.Close()
	s.connected <- attempt
	if plan.noGreeting {
		_, _ = io.Copy(io.Discard, conn)
		return
	}
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	write := func(value string) bool {
		if _, err := w.WriteString(value); err != nil {
			return false
		}
		return w.Flush() == nil
	}
	if !write("220 synthetic.local ESMTP\r\n") {
		return
	}
	recipient := ""
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.TrimSpace(line)
		upper := strings.ToUpper(command)
		if plan.stallCommand != "" && strings.HasPrefix(upper, strings.ToUpper(plan.stallCommand)) {
			s.commandSeen <- upper
			if !s.waitForGateOrClientClose(conn, r, plan.holdCommand, attempt) {
				return
			}
		}
		switch {
		case strings.HasPrefix(upper, "EHLO "):
			if plan.rejectEHLO {
				if !write("502 EHLO is not supported\r\n") {
					return
				}
			} else if plan.noAuth {
				if !write("250 synthetic.local\r\n") {
					return
				}
			} else if !write("250-synthetic.local\r\n250 AUTH PLAIN\r\n") {
				return
			}
		case strings.HasPrefix(upper, "HELO "), strings.HasPrefix(upper, "MAIL FROM:"), upper == "RSET", upper == "NOOP":
			if !write("250 OK\r\n") {
				return
			}
		case strings.HasPrefix(upper, "AUTH "):
			if !write("235 synthetic authentication accepted\r\n") {
				return
			}
		case strings.HasPrefix(upper, "RCPT TO:"):
			recipient = strings.Trim(strings.TrimSpace(command[len("RCPT TO:"):]), "<>")
			if !write("250 recipient accepted\r\n") {
				return
			}
		case upper == "DATA":
			if !write("354 send synthetic message\r\n") {
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
			s.dataReady <- recipient
			if plan.holdAfterData != nil {
				if !s.waitForGateOrClientClose(conn, r, plan.holdAfterData, attempt) {
					return
				}
			}
			reply := plan.defaultDataReply
			if choices, ok := plan.replyByRecipient[recipient]; ok && len(choices) > 0 {
				s.mu.Lock()
				index := s.recipients[recipient]
				s.recipients[recipient]++
				s.mu.Unlock()
				if index >= len(choices) {
					index = len(choices) - 1
				}
				reply = choices[index]
			}
			if reply == "" {
				return // server closes after DATA terminator; client cannot know acceptance
			}
			if !write(reply + "\r\n") {
				return
			}
			if strings.HasPrefix(reply, "250") {
				s.accepted <- recipient
			}
		case upper == "QUIT":
			if plan.closeConnectionQuit {
				return
			}
			_ = write("221 synthetic bye\r\n")
			return
		default:
			if !write("250 OK\r\n") {
				return
			}
		}
	}
}

func (s *r091DeliverySMTPServer) waitForGateOrClientClose(conn net.Conn, reader *bufio.Reader, gate *r091SMTPGate, attempt int) bool {
	if gate == nil {
		return true
	}
	for {
		select {
		case <-gate.ch:
			_ = conn.SetReadDeadline(time.Time{})
			return true
		default:
		}
		_ = conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
		_, err := reader.ReadByte()
		if err == nil {
			continue
		}
		if networkErr, ok := err.(net.Error); ok && networkErr.Timeout() {
			continue
		}
		s.connectionClosed <- attempt
		return false
	}
}

func (s *r091DeliverySMTPServer) closeAndJoin(t *testing.T) {
	t.Helper()
	_ = s.listener.Close()
	for _, plan := range s.plans {
		plan.holdAfterData.release()
	}
	s.mu.Lock()
	conns := append([]net.Conn(nil), s.conns...)
	s.mu.Unlock()
	for _, conn := range conns {
		_ = conn.Close()
	}
	select {
	case <-s.done:
	case <-time.After(3 * time.Second):
		t.Errorf("synthetic SMTP accept/handler goroutines did not join")
	}
}

type r091OutboundFixture struct {
	job    models.Job
	run    models.JobRun
	result models.JobResult
}

func r091NewOutboundFixture(t *testing.T, outputs string) *r091OutboundFixture {
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
	f := &r091OutboundFixture{
		job:    models.Job{ID: jobID, TenantID: tenantID, Name: "R091 synthetic job", JobType: "qc_analysis", InputChannelIDs: "[]", RulesConfig: "[]", Outputs: outputs, IsActive: true, CreatedAt: now, UpdatedAt: now},
		run:    models.JobRun{ID: runID, TenantID: tenantID, JobID: jobID, StartedAt: now, Status: "success", Summary: `{"conversations_analyzed":1,"conversations_passed":0,"issues_found":1}`, CreatedAt: now},
		result: models.JobResult{ID: resultID, JobRunID: runID, TenantID: tenantID, ConversationID: pkg.NewUUID(), ResultType: "qc_violation", Severity: "HIGH", RuleName: "Synthetic", Evidence: "synthetic evidence", Detail: `{}`, CreatedAt: now},
	}
	tenant := models.Tenant{ID: tenantID, Name: "R091 synthetic", Slug: "r091-" + strings.ReplaceAll(tenantID, "-", ""), Settings: "{}", CreatedAt: now, UpdatedAt: now}
	for _, row := range []interface{}{&tenant, &f.job, &f.run, &f.result} {
		if err := db.DB.Create(row).Error; err != nil {
			t.Fatalf("create synthetic %T: %v", row, err)
		}
	}
	t.Cleanup(func() {
		if db.DB == nil {
			return
		}
		for _, table := range []string{"notification_logs", "notification_delivery_receipts", "notification_delivery_manifests", "job_results", "job_runs", "jobs"} {
			db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenantID)
		}
		db.DB.Where("id = ?", tenantID).Delete(&models.Tenant{})
	})
	return f
}

func r091EmailOutputs(port int, recipients ...string) string {
	outputs := make([]OutputConfig, 0, len(recipients))
	for _, recipient := range recipients {
		outputs = append(outputs, OutputConfig{Type: "email", SMTPHost: "127.0.0.1", SMTPPort: port, From: "sender@example.invalid", To: recipient})
	}
	encoded, _ := json.Marshal(outputs)
	return string(encoded)
}

func r091InjectUpdateFailure(t *testing.T, table, status string) *bool {
	t.Helper()
	name := "r091_fail_update_" + pkg.NewUUID()
	fired := false
	if err := db.DB.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
		if fired || tx.Statement.Table != table {
			return
		}
		dest, ok := tx.Statement.Dest.(map[string]interface{})
		if !ok || dest["status"] != status {
			return
		}
		fired = true
		tx.AddError(errors.New("synthetic R091 persistence failure"))
	}); err != nil {
		t.Fatalf("register synthetic GORM update failure: %v", err)
	}
	t.Cleanup(func() {
		if err := db.DB.Callback().Update().Remove(name); err != nil {
			t.Errorf("remove synthetic GORM update failure: %v", err)
		}
	})
	return &fired
}

func r091ReceiptByRecipient(t *testing.T, f *r091OutboundFixture, recipient string) models.NotificationDeliveryReceipt {
	t.Helper()
	configs, err := parseOutputConfigs(f.job.Outputs)
	if err != nil {
		t.Fatal(err)
	}
	outputs, _, err := resolveOutputs(configs)
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range outputs {
		if output.Config.To == recipient {
			var receipt models.NotificationDeliveryReceipt
			if err := db.DB.Where("tenant_id = ? AND job_run_id = ? AND output_key = ?", f.job.TenantID, f.run.ID, output.Key).First(&receipt).Error; err != nil {
				t.Fatal(err)
			}
			return receipt
		}
	}
	t.Fatalf("no output for recipient %s", recipient)
	return models.NotificationDeliveryReceipt{}
}

func TestR091DeliveryMixedRetrySkipsSentAndPreservesFrozenResultSet(t *testing.T) {
	first := "first@example.invalid"
	second := "second@example.invalid"
	server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{replyByRecipient: map[string][]string{
		first: {"250 accepted"}, second: {"550 synthetic explicit rejection", "250 accepted"},
	}})
	f := r091NewOutboundFixture(t, r091EmailOutputs(server.port(), first, second))
	d := NewDispatcher()
	if err := d.SendJobResults(context.Background(), f.job, f.run); err == nil {
		t.Fatal("mixed confirmed/rejected outputs must remain incomplete")
	}
	firstReceipt := r091ReceiptByRecipient(t, f, first)
	secondReceipt := r091ReceiptByRecipient(t, f, second)
	if firstReceipt.Status != deliverySent || firstReceipt.AttemptCount != 1 || secondReceipt.Status != deliveryFailedStatus || secondReceipt.AttemptCount != 1 {
		t.Fatalf("first-attempt receipts: first=%+v second=%+v", firstReceipt, secondReceipt)
	}
	var added models.JobResult
	added = models.JobResult{ID: pkg.NewUUID(), TenantID: f.job.TenantID, JobRunID: f.run.ID, ConversationID: pkg.NewUUID(), ResultType: "qc_violation", Severity: "LOW", RuleName: "Appended after capture", Evidence: "synthetic", Detail: `{}`, CreatedAt: time.Now().UTC()}
	if err := db.DB.Create(&added).Error; err != nil {
		t.Fatalf("append a result after the manifest was frozen: %v", err)
	}
	if err := d.SendJobResults(context.Background(), f.job, f.run); err != nil {
		t.Fatalf("retry proven failed output: %v", err)
	}
	if got := server.connectionCount(); got != 3 {
		t.Fatalf("confirmed output was resent or failed output was not retried: SMTP connections=%d, want 3", got)
	}
	firstReceipt = r091ReceiptByRecipient(t, f, first)
	secondReceipt = r091ReceiptByRecipient(t, f, second)
	if firstReceipt.Status != deliverySent || firstReceipt.AttemptCount != 1 || secondReceipt.Status != deliverySent || secondReceipt.AttemptCount != 2 {
		t.Fatalf("final receipts: first=%+v second=%+v", firstReceipt, secondReceipt)
	}
	var original, appended models.JobResult
	if err := db.DB.First(&original, "id = ?", f.result.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.First(&appended, "id = ?", added.ID).Error; err != nil {
		t.Fatal(err)
	}
	if original.NotifiedAt == nil || appended.NotifiedAt != nil {
		t.Fatalf("marker did not follow frozen result set: original=%v appended=%v", original.NotifiedAt, appended.NotifiedAt)
	}
	var manifest models.NotificationDeliveryManifest
	if err := db.DB.Where("job_run_id = ?", f.run.ID).First(&manifest).Error; err != nil {
		t.Fatal(err)
	}
	var captured []string
	if err := json.Unmarshal([]byte(manifest.CapturedResultIDs), &captured); err != nil || len(captured) != 1 || captured[0] != f.result.ID {
		t.Fatalf("frozen result manifest changed after later insert: %v (%v)", captured, err)
	}
}

func TestR091DeliveryRejectsConfigChangeAndLeavesAllFailedUnnotified(t *testing.T) {
	f := r091NewOutboundFixture(t, `[{"type":"unsupported"}]`)
	d := NewDispatcher()
	if err := d.SendJobResults(context.Background(), f.job, f.run); err == nil {
		t.Fatal("unsupported output must be a truthful failed attempt")
	}
	var added models.JobResult
	added = models.JobResult{ID: pkg.NewUUID(), TenantID: f.job.TenantID, JobRunID: f.run.ID, ConversationID: pkg.NewUUID(), ResultType: "qc_violation", Severity: "LOW", RuleName: "later", Evidence: "synthetic", Detail: `{}`, CreatedAt: time.Now().UTC()}
	if err := db.DB.Create(&added).Error; err != nil {
		t.Fatal(err)
	}
	if err := d.SendJobResults(context.Background(), f.job, f.run); err == nil {
		t.Fatal("failed output retry should still report an incomplete set")
	}
	var manifest models.NotificationDeliveryManifest
	if err := db.DB.Where("job_run_id = ?", f.run.ID).First(&manifest).Error; err != nil {
		t.Fatal(err)
	}
	var captured []string
	if err := json.Unmarshal([]byte(manifest.CapturedResultIDs), &captured); err != nil || len(captured) != 1 || captured[0] != f.result.ID {
		t.Fatalf("re-entry recaptured a newly appended result: %v (%v)", captured, err)
	}
	if err := db.DB.Model(&models.Job{}).Where("id = ?", f.job.ID).Update("outputs", `[]`).Error; err != nil {
		t.Fatal(err)
	}
	if err := d.SendJobResults(context.Background(), f.job, f.run); err == nil || !strings.Contains(err.Error(), "configuration conflicts") {
		t.Fatalf("changed output identity bypassed frozen manifest: %v", err)
	}
	var original, appended models.JobResult
	_ = db.DB.First(&original, "id = ?", f.result.ID).Error
	_ = db.DB.First(&appended, "id = ?", added.ID).Error
	if original.NotifiedAt != nil || appended.NotifiedAt != nil {
		t.Fatalf("all-failed or config-conflict path marked results: %v %v", original.NotifiedAt, appended.NotifiedAt)
	}
}

func TestR091DeliveryUnknownAndCrashLeftSendingAreNotRetried(t *testing.T) {
	recipient := "ambiguous@example.invalid"
	server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{replyByRecipient: map[string][]string{recipient: {""}}})
	f := r091NewOutboundFixture(t, r091EmailOutputs(server.port(), recipient))
	d := NewDispatcher()
	if err := d.SendJobResults(context.Background(), f.job, f.run); err == nil {
		t.Fatal("connection closed after DATA terminator must remain ambiguous")
	}
	unknown := r091ReceiptByRecipient(t, f, recipient)
	if unknown.Status != deliveryUnknown {
		t.Fatalf("ambiguous SMTP close state=%s, want unknown", unknown.Status)
	}
	if err := d.SendJobResults(context.Background(), f.job, f.run); err == nil {
		t.Fatal("unknown receipt must keep the output set incomplete")
	}
	if server.connectionCount() != 1 {
		t.Fatalf("unknown delivery was automatically retried: connections=%d", server.connectionCount())
	}

	secondServer := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{defaultDataReply: "250 accepted"})
	crashed := r091NewOutboundFixture(t, r091EmailOutputs(secondServer.port(), "crash@example.invalid"))
	set, err := freezeDeliverySet(context.Background(), crashed.job, crashed.run)
	if err != nil || set == nil {
		t.Fatalf("create frozen manifest before simulated crash: %v", err)
	}
	var receipt models.NotificationDeliveryReceipt
	if err := db.DB.Where("job_run_id = ?", crashed.run.ID).First(&receipt).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Model(&receipt).Updates(map[string]interface{}{"status": deliverySending, "attempt_count": 1, "active_attempt_id": pkg.NewUUID()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := d.SendJobResults(context.Background(), crashed.job, crashed.run); err == nil {
		t.Fatal("crash-left sending receipt must not be treated as complete")
	}
	if secondServer.connectionCount() != 0 {
		t.Fatalf("crash-left sending receipt initiated an automatic resend: %d", secondServer.connectionCount())
	}
}

func TestR091DeliveryClaimsOutputOnceUnderConcurrentNotify(t *testing.T) {
	gate := r091NewSMTPGate()
	server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{defaultDataReply: "250 accepted", holdAfterData: gate})
	recipient := "concurrent@example.invalid"
	f := r091NewOutboundFixture(t, r091EmailOutputs(server.port(), recipient))
	d := NewDispatcher()
	firstDone := make(chan error, 1)
	go func() { firstDone <- d.SendJobResults(context.Background(), f.job, f.run) }()
	select {
	case <-server.dataReady:
	case <-time.After(3 * time.Second):
		t.Fatal("first local delivery did not reach final DATA boundary")
	}
	if err := d.SendJobResults(context.Background(), f.job, f.run); err == nil {
		t.Fatal("concurrent caller must report the still-sending output as incomplete")
	}
	if got := server.connectionCount(); got != 1 {
		t.Fatalf("concurrent Notify claimed the same output twice: %d SMTP connections", got)
	}
	gate.release()
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first delivery: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("first delivery did not finish after the synthetic server released it")
	}
	var stored models.JobResult
	if err := db.DB.First(&stored, "id = ?", f.result.ID).Error; err != nil || stored.NotifiedAt == nil {
		t.Fatalf("confirmed concurrent delivery did not mark the captured result: %v, %v", stored.NotifiedAt, err)
	}
}

func TestR091DeliveryPersistenceErrorsRemainTruthful(t *testing.T) {
	server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{defaultDataReply: "250 accepted"})
	f := r091NewOutboundFixture(t, r091EmailOutputs(server.port(), "persist@example.invalid"))
	trigger := "r091_log_" + strings.ReplaceAll(f.job.ID, "-", "")[:12]
	createTrigger := fmt.Sprintf("CREATE TRIGGER %s BEFORE INSERT ON notification_logs FOR EACH ROW BEGIN IF NEW.job_id = '%s' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'synthetic log insert failure'; END IF; END", trigger, f.job.ID)
	if err := db.DB.Exec(createTrigger).Error; err != nil {
		t.Fatalf("create synthetic log trigger: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trigger) })
	if err := NewDispatcher().SendJobResults(context.Background(), f.job, f.run); err == nil || strings.Contains(err.Error(), "synthetic") {
		t.Fatalf("log persistence failure must return a safe truthful error: %v", err)
	}
	if server.connectionCount() != 0 {
		t.Fatalf("provider was called before attempt log commit: %d", server.connectionCount())
	}
	var result models.JobResult
	if err := db.DB.First(&result, "id = ?", f.result.ID).Error; err != nil || result.NotifiedAt != nil {
		t.Fatalf("log failure marked result notified: %v, %v", result.NotifiedAt, err)
	}
	var receipt models.NotificationDeliveryReceipt
	if err := db.DB.Where("job_run_id = ?", f.run.ID).First(&receipt).Error; err != nil || receipt.Status != deliveryPending || receipt.AttemptCount != 0 {
		t.Fatalf("log failure left an unclaimed send state: %+v, %v", receipt, err)
	}
}

func TestR091DeliveryQueryAndMarkerFailuresAreNotReportedAsSuccess(t *testing.T) {
	// A missing results table is a real synthetic query failure; restore the exact table before cleanup.
	queryFixture := r091NewOutboundFixture(t, `[{"type":"unsupported"}]`)
	if err := db.DB.Exec("RENAME TABLE job_results TO job_results_r091_hold").Error; err != nil {
		t.Fatal(err)
	}
	restored := false
	defer func() {
		if !restored {
			db.DB.Exec("RENAME TABLE job_results_r091_hold TO job_results")
		}
	}()
	queryErr := NewDispatcher().SendJobResults(context.Background(), queryFixture.job, queryFixture.run)
	restoreErr := db.DB.Exec("RENAME TABLE job_results_r091_hold TO job_results").Error
	restored = restoreErr == nil
	if queryErr == nil || restoreErr != nil {
		t.Fatalf("query failure/restore: dispatch=%v restore=%v", queryErr, restoreErr)
	}

	server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{defaultDataReply: "250 accepted"})
	f := r091NewOutboundFixture(t, r091EmailOutputs(server.port(), "marker@example.invalid"))
	trigger := "r091_marker_" + strings.ReplaceAll(f.job.ID, "-", "")[:12]
	createTrigger := fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE ON job_results FOR EACH ROW BEGIN IF NEW.id = '%s' AND OLD.notified_at IS NULL AND NEW.notified_at IS NOT NULL THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'synthetic marker failure'; END IF; END", trigger, f.result.ID)
	if err := db.DB.Exec(createTrigger).Error; err != nil {
		t.Fatalf("create synthetic marker trigger: %v", err)
	}
	t.Cleanup(func() { db.DB.Exec("DROP TRIGGER IF EXISTS " + trigger) })
	if err := NewDispatcher().SendJobResults(context.Background(), f.job, f.run); err == nil || strings.Contains(err.Error(), "synthetic") {
		t.Fatalf("marker persistence failure must return safe error: %v", err)
	}
	var receipt models.NotificationDeliveryReceipt
	if err := db.DB.Where("job_run_id = ?", f.run.ID).First(&receipt).Error; err != nil || receipt.Status != deliverySent {
		t.Fatalf("accepted output receipt should remain confirmed after marker failure: %+v %v", receipt, err)
	}
	var result models.JobResult
	if err := db.DB.First(&result, "id = ?", f.result.ID).Error; err != nil || result.NotifiedAt != nil {
		t.Fatalf("failed marker write falsely marked result: %v, %v", result.NotifiedAt, err)
	}
	if err := db.DB.Exec("DROP TRIGGER IF EXISTS " + trigger).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewDispatcher().SendJobResults(context.Background(), f.job, f.run); err != nil {
		t.Fatalf("re-enter and retry only the failed marker write: %v", err)
	}
	if server.connectionCount() != 1 {
		t.Fatalf("confirmed output was resent after marker failure: %d", server.connectionCount())
	}
	if err := db.DB.First(&result, "id = ?", f.result.ID).Error; err != nil || result.NotifiedAt == nil {
		t.Fatalf("captured marker was not repaired on re-entry: %v, %v", result.NotifiedAt, err)
	}
}

func TestR091OutputIdentityIsStableAcrossReorderingAndCredentialRotation(t *testing.T) {
	a := OutputConfig{Type: "email", SMTPHost: "SMTP.EXAMPLE.INVALID", SMTPPort: 587, SMTPUser: "user-a", SMTPPass: "secret-a", From: "sender@example.invalid", To: "b@example.invalid, a@example.invalid"}
	b := OutputConfig{Type: "telegram", BotToken: "token-a", ChatID: "chat-1"}
	first, firstHash, err := resolveOutputs([]OutputConfig{a, b})
	if err != nil {
		t.Fatal(err)
	}
	a.SMTPUser, a.SMTPPass = "user-b", "secret-b"
	b.BotToken = "token-b"
	second, secondHash, err := resolveOutputs([]OutputConfig{b, a})
	if err != nil {
		t.Fatal(err)
	}
	if firstHash != secondHash || len(first) != len(second) {
		t.Fatalf("reordering or credential rotation changed output identity: %s != %s", firstHash, secondHash)
	}
	for _, old := range first {
		found := false
		for _, current := range second {
			if old.Key == current.Key {
				found = true
			}
		}
		if !found {
			t.Fatalf("stable output key changed for %#v", old.Identity)
		}
	}
	a.To = "different@example.invalid"
	changed, changedHash, err := resolveOutputs([]OutputConfig{a, b})
	if err != nil || changedHash == firstHash || len(changed) != 2 {
		t.Fatalf("destination change failed to change configuration identity: %s %v", changedHash, err)
	}
}

func TestR091ZeroOutputsDoNotCreateCompletionOrManifest(t *testing.T) {
	f := r091NewOutboundFixture(t, `[]`)
	if err := NewDispatcher().SendJobResults(context.Background(), f.job, f.run); err != nil {
		t.Fatalf("zero-output re-entry should be a no-op: %v", err)
	}
	var manifests int64
	if err := db.DB.Model(&models.NotificationDeliveryManifest{}).Where("job_run_id = ?", f.run.ID).Count(&manifests).Error; err != nil {
		t.Fatal(err)
	}
	var result models.JobResult
	if err := db.DB.First(&result, "id = ?", f.result.ID).Error; err != nil || result.NotifiedAt != nil || manifests != 0 {
		t.Fatalf("zero outputs synthesized completion: notified_at=%v manifests=%d err=%v", result.NotifiedAt, manifests, err)
	}
}

func TestR091DeliveryErrorContractUsesOnlySafeTypedOutcomes(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{nil, deliverySent}, {failedDelivery("recipient_550"), deliveryFailedStatus}, {unknownDelivery("transport"), deliveryUnknown},
		{errors.New("arbitrary provider detail with a token"), deliveryUnknown},
	} {
		status, safe := deliveryOutcome(test.err)
		if status != test.want {
			t.Fatalf("deliveryOutcome(%v)=%s, want %s", test.err, status, test.want)
		}
		if strings.Contains(safe, "token") {
			t.Fatalf("unsafe error text escaped into receipt: %q", safe)
		}
	}
}

func TestR091R1MarkerRejectsCapturedResultDeletedAfterUnlockedSnapshot(t *testing.T) {
	f := r091NewOutboundFixture(t, `[]`)
	set := &deliverySet{job: f.job, run: f.run, results: []models.JobResult{f.result}}
	name := "r091_delete_after_count_" + pkg.NewUUID()
	deleted := false
	var deleteErr error
	if err := db.DB.Callback().Query().After("gorm:query").Register(name, func(tx *gorm.DB) {
		if deleted || tx.Statement.Table != "job_results" || !strings.Contains(strings.ToLower(tx.Statement.SQL.String()), "count(") {
			return
		}
		deleted = true
		deleteErr = db.DB.Exec("DELETE FROM job_results WHERE id = ?", f.result.ID).Error
	}); err != nil {
		t.Fatalf("register deterministic marker race hook: %v", err)
	}
	t.Cleanup(func() {
		if err := db.DB.Callback().Query().Remove(name); err != nil {
			t.Errorf("remove deterministic marker race hook: %v", err)
		}
	})

	err := updateRunNotifiedAt(set)
	if deleted {
		if deleteErr != nil {
			t.Fatalf("concurrent synthetic result delete failed: %v", deleteErr)
		}
		if err == nil {
			t.Fatalf("behavioral detector: marker returned nil after the captured result disappeared between count and update")
		}
		return
	}
	if err != nil {
		t.Fatalf("locked marker update failed on a result that remained present: %v", err)
	}
	var saved models.JobResult
	if err := db.DB.First(&saved, "id = ?", f.result.ID).Error; err != nil || saved.NotifiedAt == nil {
		t.Fatalf("marker did not update the row protected by the captured-set lock: notified_at=%v err=%v", saved.NotifiedAt, err)
	}
}

func TestR091R1ClaimPersistenceFailureMakesNoOutboundCall(t *testing.T) {
	server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{defaultDataReply: "250 synthetic accepted"})
	recipient := "claim-failure@example.invalid"
	f := r091NewOutboundFixture(t, r091EmailOutputs(server.port(), recipient))
	fired := r091InjectUpdateFailure(t, "notification_delivery_receipts", deliverySending)
	err := NewDispatcher().SendJobResults(context.Background(), f.job, f.run)
	if err == nil || !*fired {
		t.Fatalf("claim update failure was not surfaced: err=%v injected=%v", err, *fired)
	}
	if server.connectionCount() != 0 {
		t.Fatalf("provider was called before durable claim: connections=%d", server.connectionCount())
	}
	receipt := r091ReceiptByRecipient(t, f, recipient)
	if receipt.Status != deliveryPending || receipt.AttemptCount != 0 {
		t.Fatalf("failed claim did not roll back to pending: %+v", receipt)
	}
	var logs []models.NotificationLog
	if err := db.DB.Where("job_run_id = ?", f.run.ID).Find(&logs).Error; err != nil || len(logs) != 0 {
		t.Fatalf("failed claim left an attempt log: rows=%d err=%v", len(logs), err)
	}
}

func TestR091R1OutcomeReceiptAndLogFailuresRemainSendingWithoutResend(t *testing.T) {
	for _, target := range []struct {
		name  string
		table string
	}{
		{name: "receipt update", table: "notification_delivery_receipts"},
		{name: "log update", table: "notification_logs"},
	} {
		t.Run(target.name, func(t *testing.T) {
			recipient := "outcome-failure@example.invalid"
			server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{defaultDataReply: "250 synthetic accepted"})
			f := r091NewOutboundFixture(t, r091EmailOutputs(server.port(), recipient))
			fired := r091InjectUpdateFailure(t, target.table, deliverySent)
			if err := NewDispatcher().SendJobResults(context.Background(), f.job, f.run); err == nil || !*fired {
				t.Fatalf("injected outcome persistence failure was not surfaced: err=%v injected=%v", err, *fired)
			}
			select {
			case <-server.accepted:
			case <-time.After(time.Second):
				t.Fatal("synthetic SMTP server did not confirm DATA acceptance")
			}
			receipt := r091ReceiptByRecipient(t, f, recipient)
			if receipt.Status != deliverySending || receipt.ActiveAttemptID == "" || receipt.AttemptCount != 1 {
				t.Fatalf("outcome transaction did not roll the receipt back to durable sending: %+v", receipt)
			}
			var log models.NotificationLog
			if err := db.DB.Where("id = ?", receipt.ActiveAttemptID).First(&log).Error; err != nil || log.Status != deliverySending {
				t.Fatalf("outcome transaction did not roll the log back to sending: status=%s err=%v", log.Status, err)
			}
			var result models.JobResult
			if err := db.DB.First(&result, "id = ?", f.result.ID).Error; err != nil || result.NotifiedAt != nil {
				t.Fatalf("possible acceptance was falsely marked complete: notified_at=%v err=%v", result.NotifiedAt, err)
			}
			_ = NewDispatcher().SendJobResults(context.Background(), f.job, f.run)
			if server.connectionCount() != 1 {
				t.Fatalf("ambiguous crash-left sending state was resent: connections=%d", server.connectionCount())
			}
		})
	}
}
