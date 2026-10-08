package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm"
)

// Synthetic interface/DB tests only; no provider call or AI governance assertion.
func uoRead(t *testing.T, summary string) usageObservationReceipt {
	t.Helper()
	var env map[string]json.RawMessage
	if err := json.Unmarshal([]byte(summary), &env); err != nil {
		t.Fatal(err)
	}
	var execution map[string]json.RawMessage
	if err := json.Unmarshal(env["source_execution"], &execution); err != nil {
		t.Fatal(err)
	}
	if len(execution["usage_observation"]) == 0 {
		t.Fatal("stored terminal usage observation missing")
	}
	var r usageObservationReceipt
	if err := json.Unmarshal(execution["usage_observation"], &r); err != nil {
		t.Fatal(err)
	}
	uoCheck(t, r)
	return r
}

func uoReload(t *testing.T, run *models.JobRun) usageObservationReceipt {
	t.Helper()
	var stored models.JobRun
	if err := db.DB.Where("id = ? AND tenant_id = ? AND job_id = ?", run.ID, run.TenantID, run.JobID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	r := uoRead(t, stored.Summary)
	returned := uoRead(t, run.Summary)
	a, _ := json.Marshal(r)
	b, _ := json.Marshal(returned)
	if string(a) != string(b) || stored.Status != run.Status {
		t.Fatal("stored and returned usage observation differ")
	}
	return r
}

func uoProvider(known bool) *exProvider {
	p := &exProvider{base: incProvider{verdict: "PASS"}}
	decorate := func(r ai.AIResponse) ai.AIResponse {
		r.InputTokens = 1000
		r.OutputTokens = 2000
		r.Provider = "private-provider"
		r.Model = "gemini-2.5-flash"
		if !known {
			r.Model = "private-model-unknown"
		}
		return r
	}
	p.single = func(ctx context.Context, prompt, transcript string) (ai.AIResponse, error) {
		r, err := p.base.AnalyzeChat(ctx, prompt, transcript)
		return decorate(r), err
	}
	p.batch = func(ctx context.Context, prompt string, items []ai.BatchItem) (ai.AIResponse, error) {
		r, err := p.base.AnalyzeChatBatch(ctx, prompt, items)
		return decorate(r), err
	}
	return p
}

// Removable callbacks are always removed explicitly before fixture cleanup writes.
func uoCreateHook(t *testing.T, table string, fn func(*gorm.DB)) func() {
	t.Helper()
	name := "uo_create_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
	removed := false
	if err := db.DB.Callback().Create().Before("gorm:begin_transaction").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == table {
			fn(tx)
		}
	}); err != nil {
		t.Fatal(err)
	}
	remove := func() {
		if !removed {
			removed = true
			if err := db.DB.Callback().Create().Remove(name); err != nil {
				t.Error(err)
			}
		}
	}
	t.Cleanup(remove)
	return remove
}

func uoStoredSuccess(t *testing.T, batch, known bool) {
	t.Helper()
	f := setupEXFixture(t, batch, "qc_analysis")
	count := 1
	if batch {
		count = 3
	}
	exAdd(t, f, count)
	run := f.mustRun(t, uoProvider(known))
	r := uoReload(t, run)
	execution := exReload(t, run)
	if execution.TenantID != f.tenantID || execution.JobID != f.jobID || execution.RunID != run.ID || execution.MetadataIncomplete || execution.ItemsSaved != count || !execution.ExecutionComplete || r.Responses != 1 || !r.TokensComplete || *r.InputTokens != 1000 || *r.OutputTokens != 2000 {
		t.Fatal("stored usage response/binding mismatch")
	}
	if known {
		if !r.CostComplete || r.Priced != 1 || r.Unpriced != 0 || math.Abs(*r.LocalEstimate-.0053) > 1e-12 {
			t.Fatal("stored known estimate literal mismatch")
		}
	} else if r.CostComplete || r.LocalEstimate != nil || r.Unpriced != 1 || r.Priced != 0 {
		t.Fatal("stored unknown price became free")
	}
	var rows []models.AIUsageLog
	if err := db.DB.Where("tenant_id = ? AND job_id = ? AND job_run_id = ?", f.tenantID, f.jobID, run.ID).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	wantModel, wantCost := "gemini-2.5-flash", .0053
	if !known {
		wantModel = "private-model-unknown"
		wantCost = 0
	}
	if len(rows) != 1 || rows[0].InputTokens != 1000 || rows[0].OutputTokens != 2000 || rows[0].Provider != "private-provider" || rows[0].Model != wantModel || math.Abs(rows[0].CostUSD-wantCost) > 1e-12 || run.Status != "success" || f.checkpoint(t) == nil {
		t.Fatal("legacy usage row/result/checkpoint changed")
	}
	if strings.Contains(run.Summary, "private-provider") || strings.Contains(run.Summary, "private-model") {
		t.Fatal("response metadata leaked into Summary")
	}
}

func TestUOStoredTerminalSingleSuccess(t *testing.T) { uoStoredSuccess(t, false, true) }
func TestUOStoredTerminalBatchSuccess(t *testing.T)  { uoStoredSuccess(t, true, true) }

func TestUOMultipleResponsesBothJobTypes(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, kind := range []string{"qc_analysis", "classification"} {
			t.Run(fmt.Sprintf("batch%t_%s", batch, kind), func(t *testing.T) {
				f := setupEXFixture(t, batch, kind)
				exAdd(t, f, 3)
				f.exec(t, "UPDATE app_settings SET value_plain='1' WHERE tenant_id=? AND setting_key='ai_batch_size'", f.tenantID)
				p := uoProvider(true)
				p.base.jobType = kind
				run := f.mustRun(t, p)
				r := uoReload(t, run)
				e := exReload(t, run)
				if r.Responses != 3 || e.CallsBegun != 3 || e.ItemsSaved != 3 || !r.CostComplete || *r.InputTokens != 3000 || *r.OutputTokens != 6000 || math.Abs(*r.LocalEstimate-.0159) > 1e-12 {
					t.Fatal("multiple logical responses did not aggregate")
				}
				var rows []models.AIUsageLog
				if err := db.DB.Where("tenant_id=? AND job_id=? AND job_run_id=?", f.tenantID, f.jobID, run.ID).Find(&rows).Error; err != nil {
					t.Fatal(err)
				}
				if len(rows) != 3 {
					t.Fatal("usage rows no longer one per response")
				}
				for _, row := range rows {
					if row.InputTokens != 1000 || row.OutputTokens != 2000 || math.Abs(row.CostUSD-.0053) > 1e-12 {
						t.Fatal("multiple usage row values changed")
					}
				}
			})
		}
	}
}
func TestUOStoredUnknownPriceBothModes(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprint(batch), func(t *testing.T) { uoStoredSuccess(t, batch, false) })
	}
}

func TestUOExplicitModesKeepScalarsAndUsage(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, mode := range []analysisMode{modeTestRun, modeFull, modeUnanalyzed, modeSinceLast} {
			t.Run(fmt.Sprintf("batch%t_%s", batch, mode), func(t *testing.T) {
				f := setupEXFixture(t, batch, "qc_analysis")
				exAdd(t, f, 1)
				plan, err := newPlan(mode, 1, pkg.BusinessRange{})
				if err != nil {
					t.Fatal(err)
				}
				run, err := NewAnalyzerWithProvider(&config.Config{}, uoProvider(true)).execute(context.Background(), f.job(t), plan, nil)
				if err != nil {
					t.Fatal(err)
				}
				r := uoReload(t, run)
				e := exReload(t, run)
				if r.Responses != 1 || !r.CostComplete || e.Mode != string(mode) || e.ItemsSaved != 1 || f.checkpoint(t) != nil {
					t.Fatal("explicit usage/mode/checkpoint changed")
				}
				var scalars map[string]interface{}
				if err := json.Unmarshal([]byte(run.Summary), &scalars); err != nil {
					t.Fatal(err)
				}
				for _, key := range []string{"source_preparation", "source_execution", "rule_observation"} {
					delete(scalars, key)
				}
				if len(scalars) != 5 || scalars["conversations_analyzed"] != float64(1) || scalars["conversations_errors"] != float64(0) {
					t.Fatal("explicit legacy scalars changed")
				}
			})
		}
	}
}

func TestUONoCallAndErrorResponses(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, stage := range []string{"empty", "setup", "error", "response_error", "panic"} {
			t.Run(fmt.Sprintf("batch%t_%s", batch, stage), func(t *testing.T) {
				f := setupEXFixture(t, batch, "qc_analysis")
				if stage != "empty" {
					exAdd(t, f, 1)
				}
				p := uoProvider(true)
				if stage == "error" || stage == "response_error" {
					answer := ai.AIResponse{}
					if stage == "response_error" {
						answer = ai.AIResponse{InputTokens: 999, OutputTokens: 999, Model: "gemini-2.5-flash", Content: "private-response"}
					}
					p.single = func(context.Context, string, string) (ai.AIResponse, error) {
						return answer, errors.New("synthetic private-error")
					}
					p.batch = func(context.Context, string, []ai.BatchItem) (ai.AIResponse, error) {
						return answer, errors.New("synthetic private-error")
					}
				}
				if stage == "panic" {
					p.base.onCall = func(int) { panic("private provider panic") }
				}
				a := NewAnalyzerWithProvider(&config.Config{}, p)
				if stage == "setup" {
					a = NewAnalyzer(&config.Config{})
					a.providerResolver = func(models.Job) (ai.AIProvider, error) { return nil, errors.New("synthetic setup failure") }
				}
				run, err := a.RunJob(context.Background(), f.job(t))
				if run == nil {
					t.Fatal("no returned row")
				}
				if stage == "panic" || stage == "setup" {
					if err == nil {
						t.Fatal("provider panic hidden")
					}
				} else if err != nil {
					t.Fatal(err)
				}
				r := uoReload(t, run)
				e := exReload(t, run)
				if r.Responses != 0 || r.TokensComplete || r.CostComplete || r.InputTokens != nil || r.LocalEstimate != nil {
					t.Fatal("no-call/error usage fabricated")
				}
				if stage == "empty" || stage == "setup" {
					if e.CallsBegun != 0 {
						t.Fatal("no-call invocation fabricated")
					}
				} else if stage == "panic" {
					if e.Interrupted != 1 {
						t.Fatal("provider panic not interrupted")
					}
				} else if e.ErrorReturned != 1 || e.UsageWrites["NOT_ATTEMPTED"] != 1 {
					t.Fatal("err precedence changed")
				}
				var n int64
				if err := db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&n).Error; err != nil {
					t.Fatal(err)
				}
				if n != 0 {
					t.Fatal("no-call/error usage row fabricated")
				}
			})
		}
	}
}

func TestUOUsageFailureAndPanicBoundaries(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, stage := range []string{"usage_error", "usage_panic", "publication_panic", "parser_rejected"} {
			t.Run(fmt.Sprintf("batch%t_%s", batch, stage), func(t *testing.T) {
				if stage == "parser_rejected" && !batch {
					return
				}
				f := setupEXFixture(t, batch, "qc_analysis")
				exAdd(t, f, 1)
				p := uoProvider(true)
				remove := func() {}
				if strings.HasPrefix(stage, "usage") {
					remove = uoCreateHook(t, "ai_usage_logs", func(tx *gorm.DB) {
						if stage == "usage_panic" {
							panic("private usage panic")
						}
						tx.AddError(errors.New("synthetic usage failure"))
					})
				}
				if stage == "publication_panic" {
					remove = uoCreateHook(t, "analysis_snapshots", func(*gorm.DB) { panic("private publication panic") })
				}
				if stage == "parser_rejected" {
					p.batch = func(context.Context, string, []ai.BatchItem) (ai.AIResponse, error) {
						return ai.AIResponse{Content: "private malformed", InputTokens: 1000, OutputTokens: 2000, Model: "gemini-2.5-flash"}, nil
					}
				}
				run, err := f.run(t, p)
				if run == nil {
					t.Fatal("no returned panic row")
				}
				if strings.HasSuffix(stage, "panic") {
					if err == nil {
						t.Fatal("actual panic hidden")
					}
				} else if err != nil {
					t.Fatal(err)
				}
				r := uoReload(t, run)
				e := exReload(t, run)
				if r.Responses != 1 || !r.CostComplete || *r.InputTokens != 1000 || math.Abs(*r.LocalEstimate-.0053) > 1e-12 {
					t.Fatal("post-response failure erased observed usage")
				}
				var n int64
				if err := db.DB.Model(&models.AIUsageLog{}).Where("job_run_id = ?", run.ID).Count(&n).Error; err != nil {
					t.Fatal(err)
				}
				switch stage {
				case "usage_error":
					if e.UsageWrites["WRITE_FAILED"] != 1 || n != 0 || e.ItemsSaved != 1 || run.Status != "success" || f.checkpoint(t) == nil {
						t.Fatal("usage failure changed result semantics")
					}
				case "usage_panic":
					if e.UsageWrites["WRITE_OUTCOME_UNKNOWN"] != 1 || n != 0 || e.ItemsSaved != 0 || e.StopReason != "PANIC" || f.checkpoint(t) != nil {
						t.Fatal("usage panic fabricated write/publication")
					}
				case "publication_panic":
					if e.UsageWrites["WRITE_SUCCEEDED"] != 1 || n != 1 || e.ItemsSaved != 0 || e.StopReason != "PANIC" || f.checkpoint(t) != nil {
						t.Fatal("publication panic changed usage")
					}
				case "parser_rejected":
					if e.Parsing["REJECTED"] != 1 || e.ItemsSaved != 0 || n != 1 || run.Status != "error" || f.checkpoint(t) != nil {
						t.Fatal("parser rejection changed usage")
					}
				}
				remove()
			})
		}
	}
}

func TestUOCancellationKeepsBegunUsagePrefix(t *testing.T) {
	for _, scenario := range []string{"single", "batch", "batch_prefix"} {
		t.Run(scenario, func(t *testing.T) {
			batch := scenario != "single"
			f := setupEXFixture(t, batch, "qc_analysis")
			exAdd(t, f, 3)
			job := f.job(t)
			res, err := ReserveJobRun(context.Background(), job, 0)
			if err != nil {
				t.Fatal(err)
			}
			p := uoProvider(true)
			remove := func() {}
			if scenario == "batch_prefix" {
				seen := 0
				remove = uoCreateHook(t, "job_results", func(*gorm.DB) {
					seen++
					if seen == 1 {
						res.owner.cancelRequested = true
						res.owner.cancel()
					}
				})
			} else {
				p.base.onCall = func(int) {
					if _, err := CancelJobRun(job.TenantID, job.ID, res.RunID()); err != nil {
						t.Fatal(err)
					}
				}
			}
			run, err := NewAnalyzerWithProvider(&config.Config{}, p).executeReserved(res, job, ordinaryPlan(), nil)
			if err != nil {
				t.Fatal(err)
			}
			r := uoReload(t, run)
			e := exReload(t, run)
			if r.Responses != 1 || !r.TokensComplete || !r.CostComplete || e.StopReason != "CANCELLED" || e.ExecutionComplete || e.UsageWrites["WRITE_SUCCEEDED"] != 1 || f.checkpoint(t) != nil {
				t.Fatal("cancel erased usage or certified full execution")
			}
			if scenario == "batch_prefix" {
				if e.ItemsSaved != 1 || e.ItemsNotPublished != 2 {
					t.Fatal("cancel batch publication prefix lost")
				}
			} else if e.ItemsSaved != 0 {
				t.Fatal("cancelled response published")
			}
			remove()
		})
	}
}

func TestUOSuccessfulSummaryPrefixes(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprint(batch), func(t *testing.T) {
			f := setupEXFixture(t, batch, "qc_analysis")
			exAdd(t, f, 2)
			var summaries []string
			remove := roSummaryHook(t, nil, func(tx *gorm.DB) {
				if tx.Error == nil && tx.RowsAffected > 0 {
					if m, ok := tx.Statement.Dest.(map[string]interface{}); ok {
						if s, ok := m["summary"].(string); ok {
							summaries = append(summaries, s)
						}
					}
				}
			})
			run := f.mustRun(t, uoProvider(true))
			final := uoReload(t, run)
			wantCalls := int64(2)
			if batch {
				wantCalls = 1
			}
			if final.Responses != wantCalls || len(summaries) != int(wantCalls)+2 {
				t.Fatal("successful Summary prefixes missing")
			}
			prep, _ := json.Marshal(spRead(t, run.Summary))
			rule, _ := json.Marshal(roRead(t, run.Summary))
			for i, s := range summaries {
				r := uoRead(t, s)
				e := exRead(t, s)
				want := int64(i)
				if want > wantCalls {
					want = wantCalls
				}
				if r.Responses != want || int64(e.CallsBegun) != want || r.TokensComplete != (want > 0) || r.CostComplete != (want > 0) {
					t.Fatal("actual Summary usage prefix mismatch")
				}
				pr, _ := json.Marshal(spRead(t, s))
				rr, _ := json.Marshal(roRead(t, s))
				if string(pr) != string(prep) || string(rr) != string(rule) {
					t.Fatal("usage changed immutable preparation/rule bytes")
				}
			}
			remove()
		})
	}
}

func TestUOInitialProgressAndTerminalWriteFaults(t *testing.T) {
	for _, stage := range []string{"initial", "progress", "terminal", "early_terminal", "terminal_fallback_failed", "early_terminal_fallback_failed"} {
		t.Run(stage, func(t *testing.T) {
			f := setupEXFixture(t, false, "qc_analysis")
			exAdd(t, f, 1)
			job := f.job(t)
			early := strings.HasPrefix(stage, "early_terminal")
			blockFallback := strings.HasSuffix(stage, "_fallback_failed")
			writes := 0
			lastSuccessfulSummary := ""
			name := "uo_update_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
			captureName := name + "_capture"
			registered := []string{}
			removed := false
			removeCallbacks := func() {
				if removed {
					return
				}
				for _, callback := range registered {
					if err := db.DB.Callback().Update().Remove(callback); err != nil {
						t.Errorf("remove fault callback: %v", err)
					}
				}
				removed = true
			}
			t.Cleanup(removeCallbacks)
			if err := db.DB.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
				if tx.Statement.Table != "job_runs" {
					return
				}
				m, ok := tx.Statement.Dest.(map[string]interface{})
				if !ok {
					return
				}
				terminal := m["status"] != nil
				// The explicit fallback-failed fixtures fault status writes even when
				// the best-effort fallback intentionally omits Summary.
				if blockFallback && terminal {
					tx.AddError(errors.New("synthetic fallback failure"))
					return
				}
				if _, ok := m["summary"]; !ok {
					return
				}
				writes++
				if (stage == "initial" && writes == 1) || (stage == "progress" && writes == 2) || ((stage == "terminal" || early) && terminal) {
					tx.AddError(errors.New("synthetic summary failure"))
				}
			}); err != nil {
				t.Fatal(err)
			}
			registered = append(registered, name)
			if err := db.DB.Callback().Update().After("gorm:update").Register(captureName, func(tx *gorm.DB) {
				if tx.Statement.Table != "job_runs" || tx.Error != nil || tx.RowsAffected <= 0 {
					return
				}
				if m, ok := tx.Statement.Dest.(map[string]interface{}); ok {
					if summary, ok := m["summary"].(string); ok {
						lastSuccessfulSummary = summary
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			registered = append(registered, captureName)
			a := NewAnalyzerWithProvider(&config.Config{}, uoProvider(true))
			if early {
				a = NewAnalyzer(&config.Config{})
				a.providerResolver = func(models.Job) (ai.AIProvider, error) { return nil, errors.New("synthetic setup failure") }
			}
			run, err := a.RunJob(context.Background(), job)
			if run == nil {
				t.Fatal("missing returned row")
			}
			returned := exRead(t, run.Summary)
			returnedUsage := uoRead(t, run.Summary)
			if early {
				if returnedUsage.Responses != 0 || returnedUsage.TokensComplete || returnedUsage.CostComplete {
					t.Fatal("early terminal usage fabricated")
				}
			} else if returnedUsage.Responses != 1 || !returnedUsage.CostComplete {
				t.Fatal("returned terminal usage lost")
			}
			if stage == "initial" || stage == "progress" {
				if err != nil || run.Status != "success" || returned.ItemsSaved != 1 {
					t.Fatalf("observation write changed behavior %v", err)
				}
				uoReload(t, run)
				exReload(t, run)
			} else {
				if err == nil {
					t.Fatal("terminal fault hidden")
				}
				var stored models.JobRun
				if readErr := db.DB.First(&stored, "id = ? AND tenant_id = ? AND job_id = ?", run.ID, job.TenantID, job.ID).Error; readErr != nil {
					t.Fatal(readErr)
				}
				if blockFallback {
					if stored.Status != "running" || stored.FinishedAt != nil {
						t.Fatal("blocked fallback claimed durable status")
					}
				} else if stored.Status != "error" || stored.FinishedAt == nil || stored.ErrorMessage != "Không ghi nhận được kết quả cuối của lượt chạy; mốc quét giữ nguyên." {
					t.Fatal("existing fallback error/finished marker lost")
				}
				// MySQL JSON re-encodes whitespace/key order; compare decoded payloads,
				// separately from the earlier successful-update observation.
				canonical := func(summary string) string {
					var value interface{}
					if parseErr := json.Unmarshal([]byte(summary), &value); parseErr != nil {
						t.Fatal(parseErr)
					}
					encoded, encodeErr := json.Marshal(value)
					if encodeErr != nil {
						t.Fatal(encodeErr)
					}
					return string(encoded)
				}
				if lastSuccessfulSummary == "" || canonical(stored.Summary) != canonical(lastSuccessfulSummary) {
					t.Fatal("failed terminal did not retain last successful Summary prefix")
				}
				if canonical(stored.Summary) == canonical(run.Summary) {
					t.Fatal("failed terminal falsely persisted returned terminal observation")
				}
				prefix := exRead(t, stored.Summary)
				prefixUsage := uoRead(t, stored.Summary)
				if early {
					if prefixUsage.Responses != 0 || prefixUsage.TokensComplete || prefixUsage.CostComplete {
						t.Fatal("early durable prefix usage fabricated")
					}
				} else if prefixUsage.Responses != 1 || !prefixUsage.CostComplete || prefixUsage.InputTokens == nil || *prefixUsage.InputTokens != 1000 {
					t.Fatal("durable progress prefix usage lost")
				}
				if prefix.ExecutionComplete || f.checkpoint(t) != nil {
					t.Fatal("failed terminal certified receipt completion or checkpoint")
				}
				if !early {
					if returned.ItemsSaved != 1 || !returned.ExecutionComplete || prefix.CallsBegun != 1 || prefix.ItemsSaved != 1 {
						t.Fatal("failed terminal lost returned observation or durable progress prefix")
					}
				} else if returned.CallsBegun != 0 || returned.StopReason != "PROVIDER_SETUP_FAILED" || prefix.CallsBegun != 0 || prefix.StopReason != "NONE" {
					t.Fatal("early terminal invocation fabricated or initial prefix changed")
				}
			}
			removeCallbacks() // both observers are removed before any fixture cleanup write
			if blockFallback {
				f.exec(t, "UPDATE job_runs SET status='error' WHERE id = ?", run.ID)
			}
		})
	}
}
