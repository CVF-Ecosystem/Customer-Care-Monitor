package engine

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm"
	"strings"
	"testing"
)

func roRead(t *testing.T, summary string) *ruleObservationReceipt {
	t.Helper()
	var env map[string]json.RawMessage
	if err := json.Unmarshal([]byte(summary), &env); err != nil {
		t.Fatal(err)
	}
	if len(env["rule_observation"]) == 0 {
		t.Fatal("stored terminal rule observation missing")
	}
	var r ruleObservationReceipt
	if err := json.Unmarshal(env["rule_observation"], &r); err != nil {
		t.Fatal(err)
	}
	roHonest(t, &r)
	return &r
}
func roReload(t *testing.T, run *models.JobRun) *ruleObservationReceipt {
	t.Helper()
	var stored models.JobRun
	if err := db.DB.First(&stored, "id = ? AND tenant_id = ? AND job_id = ?", run.ID, run.TenantID, run.JobID).Error; err != nil {
		t.Fatal(err)
	}
	r := roRead(t, stored.Summary)
	if *r != *roRead(t, run.Summary) || stored.Status != run.Status {
		t.Fatal("durable/returned rule observation mismatch")
	}
	return r
}
func roSummaryHook(t *testing.T, before func(*gorm.DB), after func(*gorm.DB)) func() {
	t.Helper()
	registered := []string{}
	removed := false
	remove := func() {
		if removed {
			return
		}
		for _, name := range registered {
			if err := db.DB.Callback().Update().Remove(name); err != nil {
				t.Errorf("remove RO observer: %v", err)
			}
		}
		removed = true
	}
	t.Cleanup(remove)
	for _, stage := range []string{"before", "after"} {
		fn := before
		if stage == "after" {
			fn = after
		}
		if fn == nil {
			continue
		}
		name := "ro_" + stage + strings.ReplaceAll(pkg.NewUUID(), "-", "")
		cb := db.DB.Callback().Update().Before("gorm:update")
		if stage == "after" {
			cb = db.DB.Callback().Update().After("gorm:update")
		}
		if err := cb.Register(name, func(tx *gorm.DB) {
			if tx.Statement.Table == "job_runs" {
				fn(tx)
			}
		}); err != nil {
			t.Fatal(err)
		}
		registered = append(registered, name)
	}
	return remove
}
func TestROCapturedJobPromptsSingleAndBatch(t *testing.T) {
	for _, kind := range []string{"qc_analysis", "classification"} {
		for _, batch := range []bool{false, true} {
			name := kind + "_single"
			if batch {
				name = kind + "_batch"
			}
			t.Run(name, func(t *testing.T) {
				f := setupEXFixture(t, batch, kind)
				exAdd(t, f, 1)
				job := f.job(t)
				job.RulesContent = "R"
				job.SkipConditions = "S"
				job.RulesConfig = `{"a":1,"b":2}`
				f.exec(t, "UPDATE jobs SET rules_content=?, skip_conditions=?, rules_config=? WHERE id=?", job.RulesContent, job.SkipConditions, job.RulesConfig, job.ID)
				p := &exProvider{base: incProvider{jobType: kind, verdict: "PASS"}}
				prompts := []string{}
				collect := func(prompt string) { prompts = append(prompts, prompt) }
				p.single = func(ctx context.Context, prompt, transcript string) (ai.AIResponse, error) {
					collect(prompt)
					return p.base.AnalyzeChat(ctx, prompt, transcript)
				}
				p.batch = func(ctx context.Context, prompt string, items []ai.BatchItem) (ai.AIResponse, error) {
					collect(prompt)
					return p.base.AnalyzeChatBatch(ctx, prompt, items)
				}
				a := NewAnalyzer(&config.Config{})
				constructed := 0
				a.providerResolver = func(captured models.Job) (ai.AIProvider, error) {
					constructed++
					if captured.RulesContent != job.RulesContent || captured.SkipConditions != job.SkipConditions || captured.RulesConfig != job.RulesConfig {
						t.Fatal("provider resolver did not receive captured rules")
					}
					f.exec(t, "UPDATE jobs SET rules_content='LATEST', skip_conditions='LATEST', rules_config='[]' WHERE id=?", job.ID)
					return p, nil
				}
				writes := []string{}
				remove := roSummaryHook(t, nil, func(tx *gorm.DB) {
					if tx.Error == nil && tx.RowsAffected > 0 {
						if m, ok := tx.Statement.Dest.(map[string]interface{}); ok {
							if s, ok := m["summary"].(string); ok {
								writes = append(writes, s)
							}
						}
					}
				})
				run, err := a.RunJob(context.Background(), job)
				remove()
				if err != nil || run == nil || run.Status != "success" || constructed != 1 || p.base.callCount() != 1 || len(prompts) != 1 {
					t.Fatalf("captured-run behavior changed: %v", err)
				}
				wantPrompt := ai.BuildQCPrompt(job.RulesContent, job.SkipConditions)
				wantDigest := "92c65713689d82e31486696ed77e4540f15af5d6da34535bc535cb0d333f5942"
				if kind == "classification" {
					wantPrompt = ai.BuildClassificationPrompt(job.RulesConfig)
					wantDigest = "9c1834f351909d856361a609093a80b7f04c2a84efd09e17381937f11a41c5dd"
				}
				if prompts[0] != wantPrompt {
					t.Fatal("prompt replaced captured rules with DB latest")
				}
				r := roReload(t, run)
				if r.Fingerprint != wantDigest || r.JobID != job.ID || r.TenantID != job.TenantID || r.RunID != run.ID || r.MetadataIncomplete {
					t.Fatal("captured rule digest or binding mismatch")
				}
				if len(writes) < 3 {
					t.Fatal("missing actual initial/progress/terminal Summary observations")
				}
				for _, s := range writes {
					if *roRead(t, s) != *r {
						t.Fatal("rule input changed across Summary writes")
					}
				}
			})
		}
	}
}
func TestROStoredTerminalSingleSuccess(t *testing.T) {
	f := setupEXFixture(t, false, "qc_analysis")
	exAdd(t, f, 1)
	job := f.job(t)
	run, err := NewAnalyzerWithProvider(&config.Config{}, &incProvider{verdict: "PASS"}).RunJob(context.Background(), job)
	if err != nil || run == nil || run.Status != "success" {
		t.Fatalf("ordinary success changed %v", err)
	}
	r := roReload(t, run)
	if *r != *newRuleObservation(job, *run).freeze() {
		t.Fatal("stored terminal rule observation differs from captured input")
	}
	exReload(t, run)
	spReload(t, run)
}
func TestRONoWorkAndFailureObservations(t *testing.T) {
	for _, scenario := range []string{"empty", "unchanged_rule_edit", "setup_error", "provider_error", "provider_panic", "input_error", "selection_error", "preparation_panic", "unsupported"} {
		t.Run(scenario, func(t *testing.T) {
			f := setupEXFixture(t, false, "qc_analysis")
			if scenario != "empty" {
				exAdd(t, f, 1)
			}
			p := &exProvider{base: incProvider{verdict: "PASS"}}
			job := f.job(t)
			if scenario == "unchanged_rule_edit" {
				f.mustRun(t, &incProvider{verdict: "PASS"})
				f.exec(t, "UPDATE jobs SET rules_content='changed rules' WHERE id=?", job.ID)
				job = f.job(t)
			}
			if scenario == "input_error" {
				job.InputChannelIDs = "invalid"
			}
			if scenario == "unsupported" {
				job.JobType = "unrecognized private type"
			}
			if scenario == "provider_error" {
				p.base.fail = true
			}
			if scenario == "provider_panic" {
				p.single = func(context.Context, string, string) (ai.AIResponse, error) { panic("private panic") }
			}
			a := NewAnalyzer(&config.Config{})
			constructed := 0
			a.providerResolver = func(models.Job) (ai.AIProvider, error) {
				constructed++
				if scenario == "setup_error" {
					return nil, errors.New("private setup")
				}
				return p, nil
			}
			remove := func() {}
			if scenario == "selection_error" || scenario == "preparation_panic" {
				name := "ro_query_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
				if err := db.DB.Callback().Query().After("gorm:query").Register(name, func(tx *gorm.DB) {
					if scenario == "selection_error" && tx.Statement.Table == "conversations" {
						tx.AddError(errors.New("private selection"))
					}
					if scenario == "preparation_panic" && tx.Statement.Table == "messages" {
						panic("private preparation")
					}
				}); err != nil {
					t.Fatal(err)
				}
				removed := false
				remove = func() {
					if !removed {
						if err := db.DB.Callback().Query().Remove(name); err != nil {
							t.Errorf("remove RO query: %v", err)
						}
						removed = true
					}
				}
				t.Cleanup(remove)
			}
			run, err := a.RunJob(context.Background(), job)
			remove()
			if run == nil {
				t.Fatalf("no returned observation %v", err)
			}
			r := roReload(t, run)
			if *r != *newRuleObservation(job, *run).freeze() {
				t.Fatal("no-work/failure rule input changed")
			}
			ex := exReload(t, run)
			switch scenario {
			case "empty", "unchanged_rule_edit":
				if err != nil || run.Status != "success" || constructed != 0 || ex.CallsBegun != 0 {
					t.Fatal("rule receipt altered no-work/cache behavior")
				}
			case "input_error", "selection_error", "preparation_panic":
				if err == nil || constructed != 0 || ex.CallsBegun != 0 {
					t.Fatal("preparation failure admitted provider")
				}
			case "setup_error":
				if err == nil || constructed != 1 || ex.CallsBegun != 0 {
					t.Fatal("setup failure invocation changed")
				}
			case "provider_error":
				if run.Status != "error" || ex.CallsBegun != 1 {
					t.Fatal("provider error behavior changed")
				}
			case "provider_panic":
				if !errors.Is(err, ErrJobRunPanic) || ex.CallsBegun != 1 {
					t.Fatal("provider panic changed")
				}
			case "unsupported":
				if r.FingerprintStatus != "UNSUPPORTED_JOB_TYPE" || r.Fingerprint != "" || r.JobType != "" {
					t.Fatal("unsupported type leaked")
				}
			}
		})
	}
}
func TestRORejectedReservationHasNoReceiptEffects(t *testing.T) {
	f := setupEXFixture(t, false, "qc_analysis")
	job := f.job(t)
	res, err := ReserveJobRun(context.Background(), job, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer res.owner.release()
	a := NewAnalyzerWithProvider(&config.Config{}, &incProvider{verdict: "PASS"})
	wrong := job
	wrong.ID = pkg.NewUUID()
	plan, err := newPlan(modeOrdinary, 0, pkg.BusinessRange{})
	if err != nil {
		t.Fatal(err)
	}
	if run, err := a.executeReserved(res, wrong, plan, nil); run != nil || !errors.Is(err, ErrReservationMismatch) {
		t.Fatal("mismatched reservation accepted")
	}
	var stored models.JobRun
	if err := db.DB.First(&stored, "id = ?", res.RunID()).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored.Summary, "rule_observation") {
		t.Fatal("rejected reservation fabricated receipt")
	}
	run, err := a.executeReserved(res, job, plan, nil)
	if err != nil {
		t.Fatal(err)
	}
	roReload(t, run)
	if next, err := a.executeReserved(res, job, plan, nil); next != nil || !errors.Is(err, ErrJobAdmission) {
		t.Fatal("reservation reuse accepted")
	}
}

func TestROInitialProgressAndTerminalWriteFaults(t *testing.T) {
	for _, stage := range []string{"initial", "progress", "terminal", "early_terminal", "terminal_fallback_failed", "early_terminal_fallback_failed"} {
		t.Run(stage, func(t *testing.T) {
			f := setupEXFixture(t, false, "qc_analysis")
			exAdd(t, f, 1)
			job := f.job(t)
			early := strings.HasPrefix(stage, "early_terminal")
			blockFallback := strings.HasSuffix(stage, "_fallback_failed")
			writes := 0
			lastSuccessfulSummary := ""
			name := "ro_fault_" + strings.ReplaceAll(pkg.NewUUID(), "-", "")
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
			a := NewAnalyzerWithProvider(&config.Config{}, &incProvider{verdict: "PASS"})
			if early {
				a = NewAnalyzer(&config.Config{})
			}
			run, err := a.RunJob(context.Background(), job)
			if run == nil {
				t.Fatal("missing returned row")
			}
			returned := exRead(t, run.Summary)
			returnedRule := roRead(t, run.Summary)
			if *returnedRule != *newRuleObservation(job, *run).freeze() {
				t.Fatal("returned fault rule input mismatch")
			}
			if stage == "initial" || stage == "progress" {
				if err != nil || run.Status != "success" || returned.ItemsSaved != 1 {
					t.Fatalf("observation write changed behavior %v", err)
				}
				exReload(t, run)
				roReload(t, run)
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
				if *roRead(t, stored.Summary) != *returnedRule {
					t.Fatal("durable prefix lost immutable rule observation")
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
