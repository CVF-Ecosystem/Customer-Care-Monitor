package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// Application fault fixtures only. These tests do not call any real provider.
type exProvider struct {
	base   incProvider
	single func(context.Context, string, string) (ai.AIResponse, error)
	batch  func(context.Context, string, []ai.BatchItem) (ai.AIResponse, error)
}

func (p *exProvider) AnalyzeChat(ctx context.Context, prompt, transcript string) (ai.AIResponse, error) {
	if p.single != nil {
		return p.single(ctx, prompt, transcript)
	}
	return p.base.AnalyzeChat(ctx, prompt, transcript)
}
func (p *exProvider) AnalyzeChatBatch(ctx context.Context, prompt string, items []ai.BatchItem) (ai.AIResponse, error) {
	if p.batch != nil {
		return p.batch(ctx, prompt, items)
	}
	return p.base.AnalyzeChatBatch(ctx, prompt, items)
}
func exRead(t *testing.T, summary string) *executionReceipt {
	t.Helper()
	var env map[string]json.RawMessage
	if err := json.Unmarshal([]byte(summary), &env); err != nil {
		t.Fatal(err)
	}
	if len(env["source_execution"]) == 0 {
		t.Fatal("missing source_execution receipt")
	}
	var r executionReceipt
	if err := json.Unmarshal(env["source_execution"], &r); err != nil {
		t.Fatal(err)
	}
	if r.Version != executionVersion || r.Scope != "analyzer_provider_interface_only" {
		t.Fatal("execution receipt identity")
	}
	exReconcile(t, &r)
	return &r
}
func exReload(t *testing.T, run *models.JobRun) *executionReceipt {
	t.Helper()
	var stored models.JobRun
	if err := db.DB.Where("id = ? AND tenant_id = ?", run.ID, run.TenantID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	var env map[string]json.RawMessage
	json.Unmarshal([]byte(stored.Summary), &env)
	if len(env["source_execution"]) == 0 {
		t.Fatal("stored terminal execution receipt missing")
	}
	r := exRead(t, stored.Summary)
	returned := exRead(t, run.Summary)
	a, _ := json.Marshal(r)
	b, _ := json.Marshal(returned)
	if string(a) != string(b) {
		t.Fatal("stored terminal execution receipt differs from returned observation")
	}
	if stored.Status != run.Status {
		t.Fatal("terminal status mismatch")
	}
	return r
}

// UUID-positive fixture is local to EX tests. Construct parents before children;
// no old fixture identity is rewritten and no ON UPDATE CASCADE is assumed.
func setupEXFixture(t *testing.T, batch bool, jobType string) *incFixture {
	t.Helper()
	db.Close()
	connectIncDB(t)
	f := &incFixture{tenantID: pkg.NewUUID(), otherTenantID: pkg.NewUUID(), channelID: pkg.NewUUID(), otherChannelID: pkg.NewUUID(), otherTenantChannelID: pkg.NewUUID(), jobID: pkg.NewUUID(), batch: batch}
	for _, tenant := range []string{f.tenantID, f.otherTenantID} {
		f.exec(t, `INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'EX', ?, '{}', NOW(), NOW())`, tenant, tenant)
	}
	for channel, tenant := range map[string]string{f.channelID: f.tenantID, f.otherChannelID: f.tenantID, f.otherTenantChannelID: f.otherTenantID} {
		f.exec(t, `INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'EX', ?, X'00', true, '{}', NOW(), NOW())`, channel, tenant, "ext-"+channel)
	}
	channels, _ := json.Marshal([]string{f.channelID})
	f.exec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, schedule_cron, is_active, outputs, output_schedule, created_at, updated_at) VALUES (?, ?, 'EX', ?, ?, 'Phan hoi dung han.', '[{"name":"Hoi don"}]', 'manual', '', true, '[]', 'none', NOW(), NOW())`, f.jobID, f.tenantID, jobType, string(channels))
	mode := "false"
	if batch {
		mode = "true"
	}
	for key, value := range map[string]string{"ai_batch_mode": mode, "ai_batch_size": "30"} {
		f.exec(t, `INSERT INTO app_settings (id, tenant_id, setting_key, value_plain, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW())`, pkg.NewUUID(), f.tenantID, key, value)
	}
	f.clock = time.Now().UTC().Truncate(time.Second).Add(-10 * time.Minute).Add(437 * time.Millisecond)
	origNow, origDelay, origNotify := analyzerNow, ordinaryFinalizeRetryDelay, sendJobNotifications
	analyzerNow = func() time.Time { return f.clock }
	ordinaryFinalizeRetryDelay = 10 * time.Millisecond
	sendJobNotifications = func(context.Context, models.Job, models.JobRun) error {
		t.Fatal("EX fixture unexpectedly notified")
		return nil
	}
	t.Cleanup(func() {
		analyzerNow, ordinaryFinalizeRetryDelay, sendJobNotifications = origNow, origDelay, origNotify
		for _, tenant := range []string{f.tenantID, f.otherTenantID} {
			for _, table := range []string{"job_results", "analysis_snapshots", "job_runs", "ai_usage_logs", "messages", "conversations", "app_settings", "activity_logs", "jobs", "channels"} {
				if err := db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenant).Error; err != nil {
					t.Errorf("EX cleanup %s: %v", table, err)
				}
			}
			if err := db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenant).Error; err != nil {
				t.Errorf("EX tenant cleanup: %v", err)
			}
		}
	})
	return f
}
func exAdd(t *testing.T, f *incFixture, n int) []string {
	t.Helper()
	ids := []string{}
	for i := 0; i < n; i++ {
		id := pkg.NewUUID()
		at := f.clock.Add(-time.Hour)
		f.exec(t, `INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, 'EX', ?, 1, '{}', NOW(), NOW())`, id, f.tenantID, f.channelID, "ext-"+id, at.UTC())
		f.addMsg(t, f.tenantID, id, fmt.Sprintf("ex-%d-m0", i), at)
		ids = append(ids, id)
	}
	// The ordinary candidate query orders equal timestamps by ID.
	sort.Strings(ids)
	return ids
}
func exCreateHook(t *testing.T, table string, fn func(*gorm.DB)) {
	t.Helper()
	name := "ex_create_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
	if err := db.DB.Callback().Create().Before("gorm:begin_transaction").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == table {
			fn(tx)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.DB.Callback().Create().Remove(name) })
}
func TestEXNoCallConstructionAndPreparation(t *testing.T) {
	for _, scenario := range []string{"empty", "unchanged", "setup_error", "preparation_error", "cancel_before_call"} {
		t.Run(scenario, func(t *testing.T) {
			f := setupEXFixture(t, false, "qc_analysis")
			p := &incProvider{verdict: "PASS"}
			a := NewAnalyzer(&config.Config{})
			constructed := 0
			a.providerResolver = func(models.Job) (ai.AIProvider, error) {
				constructed++
				if scenario == "setup_error" {
					return nil, errors.New("secret setup")
				}
				return p, nil
			}
			if scenario != "empty" {
				exAdd(t, f, 1)
			}
			if scenario == "unchanged" {
				f.mustRun(t, p)
				p.reset()
			}
			if scenario == "preparation_error" {
				name := "ex_query_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
				db.DB.Callback().Query().After("gorm:query").Register(name, func(tx *gorm.DB) {
					if tx.Statement.Table == "messages" {
						tx.AddError(errors.New("secret preparation"))
					}
				})
				t.Cleanup(func() { db.DB.Callback().Query().Remove(name) })
			}
			if scenario == "cancel_before_call" {
				a.providerResolver = func(job models.Job) (ai.AIProvider, error) {
					constructed++
					_, err := CancelJobRun(job.TenantID, job.ID, "")
					if err != nil {
						t.Fatal(err)
					}
					return p, nil
				}
			}
			run, err := a.RunJob(context.Background(), f.job(t))
			if run == nil {
				t.Fatalf("no run %v", err)
			}
			r := exReload(t, run)
			if r.CallsBegun != 0 || r.ItemCount != 0 || p.callCount() != 0 {
				t.Fatal("logical invocation began without provider method")
			}
			want := 0
			if scenario == "setup_error" || scenario == "cancel_before_call" {
				want = 1
			}
			if constructed != want {
				t.Fatalf("construction %d != %d", constructed, want)
			}
			prep := spReload(t, run)
			if scenario == "setup_error" && (r.StopReason != "PROVIDER_SETUP_FAILED" || prep.Counts["PREPARED_FOR_INFERENCE"] != 1) {
				t.Fatal("setup preparation context")
			}
			if scenario == "preparation_error" && r.StopReason != "PREPARATION_STOPPED" {
				t.Fatal("preparation stop misclassified")
			}
			if scenario == "cancel_before_call" && (run.Status != "cancelled" || r.StopReason != "CANCELLED") {
				t.Fatal("cancel boundary")
			}
		})
	}
}
func TestEXStoredTerminalSingleSuccess(t *testing.T) {
	f := setupEXFixture(t, false, "qc_analysis")
	ids := exAdd(t, f, 1)
	run := f.mustRun(t, &incProvider{verdict: "PASS"})
	r := exReload(t, run)
	if len(r.Calls) != 1 || len(r.Calls[0].MemberIDs) != 1 {
		t.Fatalf("single binding missing: %+v", r)
	}
	if r.TenantID != f.tenantID || r.JobID != f.jobID || r.RunID != run.ID || r.MetadataIncomplete || r.Calls[0].MetadataIncomplete {
		t.Fatal("valid UUID fixture envelope binding incomplete")
	}
	if r.CallsBegun != 1 || r.ResponseReturned != 1 || r.ItemsSaved != 1 || r.ItemsPending != 0 || r.UsageWrites["WRITE_SUCCEEDED"] != 1 || !r.ExecutionComplete || r.Calls[0].MemberIDs[0] != ids[0] || r.Calls[0].Parsing != "NOT_SEPARATELY_OBSERVABLE" {
		t.Fatalf("single %+v", r)
	}
	var usage int64
	db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&usage)
	if usage != 1 || run.Status != "success" || f.checkpoint(t) == nil {
		t.Fatal("legacy single result/usage/checkpoint")
	}
}
func TestEXBatchBindingAndParserRejection(t *testing.T) {
	for _, bad := range []bool{false, true} {
		t.Run(fmt.Sprint(bad), func(t *testing.T) {
			f := setupEXFixture(t, true, "qc_analysis")
			ids := exAdd(t, f, 3)
			p := &exProvider{base: incProvider{verdict: "PASS"}}
			if bad {
				p.batch = func(context.Context, string, []ai.BatchItem) (ai.AIResponse, error) {
					return ai.AIResponse{Content: "malformed secret response", Model: "test-double"}, nil
				}
			}
			run, err := f.run(t, p)
			if err != nil {
				t.Fatal(err)
			}
			r := exReload(t, run)
			if len(r.Calls) != 1 || len(r.Calls[0].MemberIDs) != len(ids) {
				t.Fatal("batch bindings missing")
			}
			if strings.Join(r.Calls[0].MemberIDs, ",") != strings.Join(ids, ",") || r.TenantID != f.tenantID || r.JobID != f.jobID || r.RunID != run.ID || r.MetadataIncomplete || r.Calls[0].MetadataIncomplete {
				t.Fatal("ordered batch UUID binding changed")
			}
			if r.CallsBegun != 1 || r.ResponseReturned != 1 || r.ItemCount != 3 || len(r.Calls[0].MemberIDs) != 3 || r.Calls[0].Method != "BATCH" || r.UsageWrites["WRITE_SUCCEEDED"] != 1 {
				t.Fatalf("batch %+v", r)
			}
			var usage int64
			db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&usage)
			if usage != 1 {
				t.Fatal("batch usage duplicated")
			}
			if bad {
				if r.Parsing["REJECTED"] != 1 || r.ItemsNotPublished != 3 || r.ItemsSaved != 0 {
					t.Fatal("parser rejection publication")
				}
			} else if r.Parsing["ACCEPTED"] != 1 || r.ItemsSaved != 3 {
				t.Fatal("batch success")
			}
		})
	}
}
func TestEXResponseAndErrorWinsInBothModes(t *testing.T) {
	for _, scenario := range []struct{ batch, explicit bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		t.Run(fmt.Sprintf("batch%t_explicit%t", scenario.batch, scenario.explicit), func(t *testing.T) {
			batch := scenario.batch
			f := setupEXFixture(t, batch, "qc_analysis")
			exAdd(t, f, 2)
			p := &exProvider{}
			answer := ai.AIResponse{Content: "secret response with error"}
			p.single = func(context.Context, string, string) (ai.AIResponse, error) {
				return answer, errors.New("secret provider error")
			}
			p.batch = func(context.Context, string, []ai.BatchItem) (ai.AIResponse, error) {
				return answer, errors.New("secret provider error")
			}
			var run *models.JobRun
			var err error
			if scenario.explicit {
				run, err = NewAnalyzerWithProvider(&config.Config{}, p).RunJobFull(context.Background(), f.job(t))
			} else {
				run, err = f.run(t, p)
			}
			if err != nil {
				t.Fatal(err)
			}
			r := exReload(t, run)
			want := 2
			if batch {
				want = 1
			}
			if r.CallsBegun != want || r.ErrorReturned != want || r.ResponseReturned != 0 || r.ItemsNotPublished != 2 || r.ItemsSaved != 0 || r.UsageWrites["NOT_ATTEMPTED"] != want || run.Status != "error" {
				t.Fatalf("response+error %+v", r)
			}
			var n int64
			db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&n)
			if n != 0 {
				t.Fatal("usage fabricated for error")
			}
		})
	}
}
func TestEXSaveDatabaseFailureIsPublicationFailure(t *testing.T) {
	f := setupEXFixture(t, false, "qc_analysis")
	exAdd(t, f, 1)
	exCreateHook(t, "job_results", func(tx *gorm.DB) { tx.AddError(errors.New("synthetic save database failure")) })
	run := f.mustRun(t, &incProvider{verdict: "PASS"})
	r := exReload(t, run)
	if len(r.Calls) != 1 {
		t.Fatal("save-failure call binding missing")
	}
	if r.ResponseReturned != 1 || r.ErrorReturned != 0 || r.ItemsSaveFailed != 1 || r.ItemsSaved != 0 || r.UsageWrites["WRITE_SUCCEEDED"] != 1 || r.Calls[0].Parsing != "NOT_SEPARATELY_OBSERVABLE" || run.Status != "error" || f.checkpoint(t) != nil {
		t.Fatalf("database save failure conflated %+v", r)
	}
	for _, table := range []string{"job_results", "analysis_snapshots"} {
		var n int64
		if err := db.DB.Table(table).Where("job_run_id = ?", run.ID).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatal("failed result transaction persisted")
		}
	}
}
func TestEXMixedSaveAndProviderFailures(t *testing.T) {
	f := setupEXFixture(t, false, "qc_analysis")
	exAdd(t, f, 3)
	n := 0
	base := &incProvider{verdict: "PASS"}
	p := &exProvider{single: func(ctx context.Context, prompt, transcript string) (ai.AIResponse, error) {
		n++
		switch n {
		case 2:
			return ai.AIResponse{}, errors.New("synthetic provider")
		case 3:
			return ai.AIResponse{Content: "invalid output", Model: "test-double"}, nil
		}
		return base.AnalyzeChat(ctx, prompt, transcript)
	}}
	run, err := f.run(t, p)
	if err != nil {
		t.Fatal(err)
	}
	r := exReload(t, run)
	if r.CallsBegun != 3 || r.ResponseReturned != 2 || r.ErrorReturned != 1 || r.ItemsSaved != 1 || r.ItemsSaveFailed != 1 || r.ItemsNotPublished != 1 || r.Parsing["NOT_SEPARATELY_OBSERVABLE"] != 3 || run.Status != "partial" || f.checkpoint(t) != nil {
		t.Fatalf("mixed %+v", r)
	}
}
func TestEXUsageWriteFailurePreservesSavedResult(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprint(batch), func(t *testing.T) {
			f := setupEXFixture(t, batch, "qc_analysis")
			exAdd(t, f, 2)
			exCreateHook(t, "ai_usage_logs", func(tx *gorm.DB) { tx.AddError(errors.New("synthetic usage failure")) })
			run := f.mustRun(t, &incProvider{verdict: "PASS"})
			r := exReload(t, run)
			want := 2
			if batch {
				want = 1
			}
			if r.ResponseReturned != want || r.UsageWrites["WRITE_FAILED"] != want || r.ItemsSaved != 2 || run.Status != "success" {
				t.Fatalf("usage failure %+v", r)
			}
			var n int64
			db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&n)
			if n != 0 {
				t.Fatal("failed usage persisted")
			}
		})
	}
}
func TestEXCancellationDuringProviderAndBatchPrefix(t *testing.T) {
	for _, scenario := range []string{"single_running", "batch_running", "batch_prefix"} {
		t.Run(scenario, func(t *testing.T) {
			batch := scenario != "single_running"
			f := setupEXFixture(t, batch, "qc_analysis")
			exAdd(t, f, 3)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			job := f.job(t)
			res, reserveErr := ReserveJobRun(ctx, job, 0)
			if reserveErr != nil {
				t.Fatal(reserveErr)
			}
			p := &incProvider{verdict: "PASS"}
			if scenario != "batch_prefix" {
				p.onCall = func(int) {
					if _, err := CancelJobRun(job.TenantID, job.ID, res.RunID()); err != nil {
						t.Fatal(err)
					}
				}
			} else {
				seen := 0
				exCreateHook(t, "job_results", func(*gorm.DB) {
					seen++
					if seen == 1 {
						// This callback executes inside owner.publish's held lock. Install the
						// accepted-cancel state there to make the exact published prefix finite;
						// a public CancelJobRun call here would deadlock on that same lock.
						res.owner.cancelRequested = true
						res.owner.cancel()
					}
				})
			}
			run, err := NewAnalyzerWithProvider(&config.Config{}, p).executeReserved(res, job, ordinaryPlan(), nil)
			if err != nil {
				t.Fatal(err)
			}
			r := exReload(t, run)
			if r.CallsBegun != 1 || r.ResponseReturned != 1 || r.UsageWrites["WRITE_SUCCEEDED"] != 1 || r.StopReason != "CANCELLED" || r.ExecutionComplete || f.checkpoint(t) != nil {
				t.Fatalf("cancel %+v", r)
			}
			if scenario == "batch_prefix" {
				if r.ItemsSaved != 1 || r.ItemsNotPublished != 2 || r.ItemsPending != 0 {
					t.Fatal("actual batch publication prefix lost")
				}
			} else {
				want := 1
				if batch {
					want = 3
				}
				if r.ItemsSaved != 0 || r.ItemsNotPublished != want {
					t.Fatal("cancelled response falsely published")
				}
			}
		})
	}
}
func TestEXActualPanicBoundaries(t *testing.T) {
	for _, stage := range []string{"provider", "usage", "publication"} {
		t.Run(stage, func(t *testing.T) {
			f := setupEXFixture(t, false, "qc_analysis")
			exAdd(t, f, 1)
			p := &incProvider{verdict: "PASS"}
			if stage == "provider" {
				p.onCall = func(int) { panic("secret provider panic") }
			}
			if stage == "usage" {
				exCreateHook(t, "ai_usage_logs", func(*gorm.DB) { panic("secret usage panic") })
			}
			if stage == "publication" {
				exCreateHook(t, "analysis_snapshots", func(*gorm.DB) { panic("secret publication panic") })
			}
			run, err := f.run(t, p)
			if !errors.Is(err, ErrJobRunPanic) {
				t.Fatalf("panic error %v", err)
			}
			r := exReload(t, run)
			if r.StopReason != "PANIC" || r.ItemsPending != 1 || r.ItemsSaved != 0 || r.InFlight != 0 || r.ExecutionComplete || run.Status != "error" || JobRunActive(f.tenantID, f.jobID) {
				t.Fatalf("panic %+v", r)
			}
			if stage == "provider" {
				if r.Interrupted != 1 || r.ResponseReturned != 0 || r.UsageWrites["NOT_ATTEMPTED"] != 1 {
					t.Fatal("provider panic observation")
				}
			} else if r.Interrupted != 0 || r.ResponseReturned != 1 {
				t.Fatal("publication panic erased returned response")
			}
			if stage == "usage" && r.UsageWrites["WRITE_OUTCOME_UNKNOWN"] != 1 {
				t.Fatal("usage panic fabricated write result")
			}
		})
	}
}
func TestEXInitialProgressAndTerminalWriteFaults(t *testing.T) {
	for _, stage := range []string{"initial", "progress", "terminal", "early_terminal"} {
		t.Run(stage, func(t *testing.T) {
			f := setupEXFixture(t, false, "qc_analysis")
			exAdd(t, f, 1)
			job := f.job(t)
			writes := 0
			name := "ex_update_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
			db.DB.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
				if tx.Statement.Table != "job_runs" {
					return
				}
				m, ok := tx.Statement.Dest.(map[string]interface{})
				if !ok {
					return
				}
				if _, ok := m["summary"]; !ok {
					return
				}
				writes++
				terminal := m["status"] != nil
				if (stage == "initial" && writes == 1) || (stage == "progress" && writes == 2) || ((stage == "terminal" || stage == "early_terminal") && terminal) {
					tx.AddError(errors.New("synthetic summary failure"))
				}
			})
			t.Cleanup(func() { db.DB.Callback().Update().Remove(name) })
			a := NewAnalyzerWithProvider(&config.Config{}, &incProvider{verdict: "PASS"})
			if stage == "early_terminal" {
				a = NewAnalyzer(&config.Config{})
			}
			run, err := a.RunJob(context.Background(), job)
			if run == nil {
				t.Fatal("missing returned row")
			}
			returned := exRead(t, run.Summary)
			if stage == "initial" || stage == "progress" {
				if err != nil || run.Status != "success" || returned.ItemsSaved != 1 {
					t.Fatalf("observation write changed behavior %v", err)
				}
				exReload(t, run)
			} else {
				if err == nil {
					t.Fatal("terminal fault hidden")
				}
				var stored models.JobRun
				db.DB.First(&stored, "id = ?", run.ID)
				if stored.Status != "running" {
					t.Fatal("failed terminal claimed durable status")
				}
				if stage == "terminal" && returned.ItemsSaved != 1 {
					t.Fatal("failed terminal lost returned observation")
				}
				if stage == "early_terminal" && returned.CallsBegun != 0 {
					t.Fatal("early terminal invocation fabricated")
				}
				db.DB.Callback().Update().Remove(name)
				f.exec(t, "UPDATE job_runs SET status='error' WHERE id = ?", run.ID)
			}
		})
	}
}
func TestEXProgressPrefixAndPreparationUnchanged(t *testing.T) {
	f := setupEXFixture(t, false, "qc_analysis")
	exAdd(t, f, 2)
	var summaries []string
	name := "ex_progress_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
	db.DB.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == "job_runs" {
			if m, ok := tx.Statement.Dest.(map[string]interface{}); ok {
				if s, ok := m["summary"].(string); ok {
					summaries = append(summaries, s)
				}
			}
		}
	})
	t.Cleanup(func() { db.DB.Callback().Update().Remove(name) })
	run := f.mustRun(t, &incProvider{verdict: "PASS"})
	final := exReload(t, run)
	if len(summaries) != 4 {
		t.Fatalf("writer count %d", len(summaries))
	}
	prep, _ := json.Marshal(spRead(t, run.Summary))
	for i, s := range summaries {
		r := exRead(t, s)
		want := i
		if want > 2 {
			want = 2
		}
		if r.CallsBegun != want || r.ItemsSaved != want {
			t.Fatal("progress execution prefix mismatch")
		}
		pr, _ := json.Marshal(spRead(t, s))
		if string(pr) != string(prep) {
			t.Fatal("preparation receipt changed with execution prefix")
		}
	}
	if final.ItemsSaved != 2 {
		t.Fatal("terminal prefix")
	}
}
func TestEXAllExplicitModesRetainScalars(t *testing.T) {
	for _, mode := range []analysisMode{modeTestRun, modeFull, modeUnanalyzed, modeSinceLast} {
		t.Run(string(mode), func(t *testing.T) {
			f := setupEXFixture(t, false, "qc_analysis")
			exAdd(t, f, 1)
			plan, err := newPlan(mode, 1, pkg.BusinessRange{})
			if err != nil {
				t.Fatal(err)
			}
			run, err := NewAnalyzerWithProvider(&config.Config{}, &incProvider{verdict: "PASS"}).execute(context.Background(), f.job(t), plan, nil)
			if err != nil {
				t.Fatal(err)
			}
			r := exReload(t, run)
			if r.Mode != string(mode) || r.ItemsSaved != 1 || f.checkpoint(t) != nil {
				t.Fatalf("mode %+v", r)
			}
			var scalars map[string]interface{}
			json.Unmarshal([]byte(run.Summary), &scalars)
			delete(scalars, "source_preparation")
			delete(scalars, "source_execution")
			if len(scalars) != 5 || scalars["conversations_analyzed"] != float64(1) || scalars["conversations_errors"] != float64(0) {
				t.Fatal("legacy mode scalars")
			}
		})
	}
}
