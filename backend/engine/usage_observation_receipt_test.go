package engine

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

func uoCall(c *executionCollector, input, output int, cost float64, known bool) {
	c.begin("SINGLE", nil)
	c.returned(nil)
	c.observeUsage(input, output, cost, known)
}

func uoCheck(t *testing.T, r usageObservationReceipt) {
	t.Helper()
	if r.Version != "ccmai.usage-observation.v1" || r.Scope != "successful_interface_response_local_estimate" || r.TokenBasis != "INTERFACE_VALUES_PRESENCE_UNAVAILABLE" || r.Billing != "NOT_OBSERVED" || r.PriceRevision != "NOT_CAPTURED" {
		t.Fatal("usage observation evidence boundary")
	}
	if (r.InputTokens != nil) != r.TokensComplete || (r.OutputTokens != nil) != r.TokensComplete || (r.LocalEstimate != nil) != r.CostComplete {
		t.Fatal("usage observation optional totals disagree with completeness")
	}
	if _, err := json.Marshal(r); err != nil {
		t.Fatalf("usage observation not JSON safe: %v", err)
	}
}

func TestUOInitialUnknownAndZeroPresence(t *testing.T) {
	c, _ := exCollector()
	r := c.freeze().UsageObservation
	uoCheck(t, r)
	if r.Responses != 0 || r.TokensComplete || r.CostComplete || r.InputTokens != nil || r.LocalEstimate != nil {
		t.Fatal("no-call usage fabricated")
	}
	uoCall(c, 0, 0, 0, true)
	r = c.freeze().UsageObservation
	uoCheck(t, r)
	if r.Responses != 1 || !r.TokensComplete || !r.CostComplete || *r.InputTokens != 0 || *r.OutputTokens != 0 || *r.LocalEstimate != 0 {
		t.Fatal("zero interface observation lost")
	}
	// The fixed token basis still discloses missing presence evidence; zero is never free proof.
}

func TestUOKnownPriceLiteralControl(t *testing.T) {
	c, _ := exCollector()
	uoCall(c, 1000, 2000, .0103, true)
	uoCall(c, 3000, 4000, .0209, true)
	r := c.freeze().UsageObservation
	uoCheck(t, r)
	if r.Responses != 2 || r.Priced != 2 || r.Unpriced != 0 || !r.CostComplete || *r.InputTokens != 4000 || *r.OutputTokens != 6000 || math.Abs(*r.LocalEstimate-.0312) > 1e-12 {
		t.Fatal("known local estimate literal mismatch")
	}
}

func TestUOUnknownPriceNeverFree(t *testing.T) {
	c, _ := exCollector()
	uoCall(c, 1000, 2000, 0, false)
	r := c.freeze().UsageObservation
	uoCheck(t, r)
	if r.Unpriced != 1 || r.Priced != 0 || r.LocalEstimate != nil || r.CostComplete || !r.TokensComplete || *r.InputTokens != 1000 {
		t.Fatal("unknown price fabricated complete/free estimate")
	}
	uoCall(c, 3000, 4000, .0209, true)
	r = c.freeze().UsageObservation
	if r.Unpriced != 1 || r.Priced != 1 || r.Responses != 2 || r.LocalEstimate != nil || r.CostComplete || !r.TokensComplete || *r.OutputTokens != 6000 {
		t.Fatal("mixed unknown price fabricated complete estimate")
	}
}

func TestUOInvalidValuesAndOverflow(t *testing.T) {
	for _, v := range []struct {
		name    string
		in, out int
		cost    float64
	}{
		{"input", -1, 2, 1}, {"output", 1, -2, 1}, {"negative_cost", 1, 2, -1}, {"nan", 1, 2, math.NaN()}, {"positive_inf", 1, 2, math.Inf(1)}, {"negative_inf", 1, 2, math.Inf(-1)},
	} {
		t.Run(v.name, func(t *testing.T) {
			c, _ := exCollector()
			uoCall(c, v.in, v.out, v.cost, true)
			r := c.freeze().UsageObservation
			uoCheck(t, r)
			if r.LocalEstimate != nil || r.CostComplete {
				t.Fatal("invalid value fabricated complete cost")
			}
			if v.in < 0 || v.out < 0 {
				if r.InvalidTokens != 1 || r.TokensComplete || r.InputTokens != nil {
					t.Fatal("negative tokens accepted")
				}
			} else if r.InvalidCosts != 1 || !r.TokensComplete {
				t.Fatal("invalid cost classification")
			}
		})
	}
	t.Run("token_overflow", func(t *testing.T) {
		c, _ := exCollector()
		c.usage.input = math.MaxInt64
		uoCall(c, 1, 0, 1, true)
		r := c.freeze().UsageObservation
		uoCheck(t, r)
		if !r.TokenOverflow || r.TokensComplete || r.CostComplete {
			t.Fatal("token overflow wrapped")
		}
	})
	t.Run("output_overflow", func(t *testing.T) {
		c, _ := exCollector()
		c.usage.output = math.MaxInt64
		uoCall(c, 0, 1, 1, true)
		r := c.freeze().UsageObservation
		uoCheck(t, r)
		if !r.TokenOverflow || r.OutputTokens != nil {
			t.Fatal("output overflow wrapped")
		}
	})
	t.Run("cost_overflow", func(t *testing.T) {
		c, _ := exCollector()
		uoCall(c, 1, 1, math.MaxFloat64, true)
		uoCall(c, 1, 1, math.MaxFloat64, true)
		r := c.freeze().UsageObservation
		uoCheck(t, r)
		if !r.CostOverflow || !r.TokensComplete || r.CostComplete {
			t.Fatal("cost overflow accepted")
		}
	})
	for _, field := range []string{"responses", "invalid_tokens", "priced", "unpriced", "invalid_costs"} {
		t.Run("counter_"+field, func(t *testing.T) {
			c, _ := exCollector()
			switch field {
			case "responses":
				c.usage.r.Responses = math.MaxInt64
			case "invalid_tokens":
				c.usage.r.InvalidTokens = math.MaxInt64
			case "priced":
				c.usage.r.Priced = math.MaxInt64
			case "unpriced":
				c.usage.r.Unpriced = math.MaxInt64
			case "invalid_costs":
				c.usage.r.InvalidCosts = math.MaxInt64
			}
			in, cost, known := 1, 1.0, true
			if field == "invalid_tokens" {
				in = -1
			}
			if field == "unpriced" {
				known = false
			}
			if field == "invalid_costs" {
				cost = math.NaN()
			}
			uoCall(c, in, 1, cost, known)
			r := c.freeze().UsageObservation
			uoCheck(t, r)
			if !r.CounterOverflow || r.TokensComplete || r.CostComplete {
				t.Fatal("counter overflow accepted")
			}
		})
	}
}

func TestUOActiveReturnGuardAndIncompletePrefix(t *testing.T) {
	c, m := exCollector()
	c.observeUsage(1, 2, 1, true)
	c.begin("SINGLE", []models.Conversation{m})
	c.observeUsage(1, 2, 1, true)
	if c.freeze().UsageObservation.Responses != 0 {
		t.Fatal("in-flight usage fabricated")
	}
	c.returned(errors.New("private response+error"))
	c.observeUsage(1, 2, 1, true)
	if c.freeze().UsageObservation.Responses != 0 {
		t.Fatal("error response usage fabricated")
	}
	uoCall(c, 1, 2, 1, true)
	c.observeUsage(99, 99, 99, true)
	r := c.freeze().UsageObservation
	uoCheck(t, r)
	if r.Responses != 1 || r.TokensComplete || r.CostComplete {
		t.Fatal("duplicate or missing-call coverage fabricated")
	}
	c.begin("SINGLE", nil)
	c.panicStop()
	c.observeUsage(1, 2, 1, true)
	if c.freeze().UsageObservation.Responses != 1 {
		t.Fatal("interrupted usage fabricated")
	}
	var nilCollector *executionCollector
	nilCollector.observeUsage(1, 1, 1, true)
}

func TestUOFreezePrivacyBoundsAndComposition(t *testing.T) {
	c, m := exCollector()
	uoCall(c, 1, 2, .1, true)
	prefix := c.freeze()
	*prefix.UsageObservation.InputTokens = 99
	*prefix.UsageObservation.LocalEstimate = 99
	uoCall(c, 3, 4, .2, true)
	fresh := c.freeze().UsageObservation
	if *fresh.InputTokens != 4 || math.Abs(*fresh.LocalEstimate-.3) > 1e-12 {
		t.Fatal("frozen usage pointers alias collector")
	}
	for i := 2; i < 205; i++ {
		uoCall(c, 1, 2, 0, true)
	}
	c.byteLimit = 1500
	r := c.freeze()
	b, err := json.Marshal(r)
	if err != nil || len(b) > 1500 || r.UsageObservation.Responses != 205 || !r.UsageObservation.TokensComplete || *r.UsageObservation.InputTokens != 207 || r.OmittedCalls == 0 {
		t.Fatal("bounded trimming lost usage aggregate")
	}
	u, err := json.Marshal(newExecutionCollector(models.Job{}, models.JobRun{}, ordinaryPlan()).freeze().UsageObservation)
	if err != nil || len(u) > 600 {
		t.Fatalf("usage fixed envelope too large %d", len(u))
	}
	job := models.Job{ID: c.receipt.JobID, TenantID: c.tenant}
	run := models.JobRun{ID: c.receipt.RunID, TenantID: c.tenant, JobID: job.ID}
	prep := newPreparationCollector(job, run, ordinaryPlan())
	prep.start(m)
	prep.finish("PREPARED_FOR_INFERENCE")
	prep.complete()
	before, _ := json.Marshal(prep.freeze())
	summary := observedSummary(map[string]interface{}{"count": 1}, prep.freeze(), r)
	var env map[string]json.RawMessage
	if json.Unmarshal([]byte(summary), &env) != nil || len(env) != 3 || string(env["source_preparation"]) != string(before) || len(env["usage_observation"]) != 0 {
		t.Fatal("usage changed composer/preparation/top-level keys")
	}
	for _, secret := range []string{"provider", "model", "prompt", "rules", "secret", "transcript", "content"} {
		if strings.Contains(string(u), secret) {
			t.Fatal("usage retained raw identity/content")
		}
	}
	bad := newExecutionCollector(models.Job{ID: "private-secret", TenantID: "private-model"}, models.JobRun{ID: "private-content", TenantID: "different"}, ordinaryPlan())
	uoCall(bad, 1, 2, 1, true)
	encoded, _ := json.Marshal(bad.freeze())
	if strings.Contains(string(encoded), "private-") || !bad.freeze().MetadataIncomplete {
		t.Fatal("invalid binding leaked identity")
	}
}
