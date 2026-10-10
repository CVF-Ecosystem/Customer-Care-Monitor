package engine

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

func adapterPresenceCount(status ai.AIUsagePresenceStatus, value *int64) ai.AIUsageCount {
	return ai.AIUsageCount{Status: status, Value: value}
}

func adapterPresence(schema string, source ai.AIUsagePresenceSource, input, output ai.AIUsageCount, complete bool) *ai.AIUsagePresence {
	return &ai.AIUsagePresence{SchemaVersion: schema, Source: source, InputTokens: input, OutputTokens: output, Complete: complete}
}

func adapterObserveReturned(c *executionCollector, presence *ai.AIUsagePresence) {
	c.begin("SINGLE", nil)
	c.returned(nil)
	c.observeUsagePresence(presence)
}

func TestAdapterUsagePresenceOmittedBeforeNewHook(t *testing.T) {
	c, _ := exCollector()
	c.begin("SINGLE", nil)
	c.returned(nil)
	c.observeUsage(11, 12, .25, true)
	r := c.freeze().UsageObservation
	b, err := json.Marshal(r)
	if err != nil || len(b) > 600 || strings.Contains(string(b), "adapter_usage_presence") {
		t.Fatalf("legacy usage envelope changed: len=%d err=%v", len(b), err)
	}
	if r.Responses != 1 || r.InputTokens == nil || *r.InputTokens != 11 || r.OutputTokens == nil || *r.OutputTokens != 12 {
		t.Fatalf("legacy scalar observation changed: %+v", r)
	}
}

func TestAdapterUsagePresencePositiveAggregation(t *testing.T) {
	c, _ := exCollector()
	input := int64(9007199254740993)
	output := int64(0)
	presence := adapterPresence("1", ai.AIUsagePresenceSourceOpenAIWire,
		adapterPresenceCount(ai.AIUsagePresenceKnown, &input), adapterPresenceCount(ai.AIUsagePresenceKnown, &output), false)
	c.begin("SINGLE", nil)
	c.returned(nil)
	c.observeUsage(3, 4, .1, true)
	c.observeUsagePresence(presence)
	input = 1
	output = 1
	r := c.freeze().UsageObservation
	aggregate := r.AdapterUsagePresence
	if aggregate == nil || aggregate.Version != adapterUsagePresenceVersion || aggregate.Basis != adapterUsagePresenceBasis || !aggregate.Complete || aggregate.Responses != 1 {
		t.Fatalf("adapter aggregate missing or incomplete: %+v", aggregate)
	}
	if aggregate.Input.Known != 1 || aggregate.Input.Total == nil || *aggregate.Input.Total != 9007199254740993 || aggregate.Output.Known != 1 || aggregate.Output.Total == nil || *aggregate.Output.Total != 0 {
		t.Fatalf("exact known adapter values lost: %+v", aggregate)
	}
	if r.InputTokens == nil || *r.InputTokens != 3 || r.OutputTokens == nil || *r.OutputTokens != 4 || !r.TokensComplete {
		t.Fatalf("adapter metadata changed legacy scalar accounting: %+v", r)
	}
	encoded, err := json.Marshal(aggregate)
	if err != nil || len(encoded) > 1200 {
		t.Fatalf("adapter aggregate exceeded fixed bound: len=%d err=%v", len(encoded), err)
	}
	*aggregate.Input.Total = 7
	fresh := c.freeze().UsageObservation.AdapterUsagePresence
	if fresh == nil || fresh.Input.Total == nil || *fresh.Input.Total != 9007199254740993 {
		t.Fatal("frozen aggregate aliases collector state")
	}
}

func TestAdapterUsagePresenceM01NilUnobservedDetector(t *testing.T) {
	c, _ := exCollector()
	c.begin("SINGLE", nil)
	c.returned(nil)
	c.observeUsage(19, 23, .5, true)
	c.observeUsagePresence(nil)
	receipt := c.freeze().UsageObservation
	aggregate := receipt.AdapterUsagePresence
	if aggregate == nil || aggregate.Responses != 1 || aggregate.Input.Unobserved != 1 || aggregate.Output.Unobserved != 1 || aggregate.Input.Known != 0 || aggregate.Output.Known != 0 || aggregate.Input.Total != nil || aggregate.Output.Total != nil || receipt.InputTokens == nil || *receipt.InputTokens != 19 || receipt.OutputTokens == nil || *receipt.OutputTokens != 23 {
		t.Fatalf("APO:M01 nil metadata must remain unobserved: %+v", aggregate)
	}
}

func TestAdapterUsagePresenceM01KnownZeroHealthyControl(t *testing.T) {
	c, _ := exCollector()
	zero := int64(0)
	adapterObserveReturned(c, adapterPresence("1", ai.AIUsagePresenceSourceAnthropicRawJSON,
		adapterPresenceCount(ai.AIUsagePresenceKnown, &zero), adapterPresenceCount(ai.AIUsagePresenceKnown, &zero), true))
	aggregate := c.freeze().UsageObservation.AdapterUsagePresence
	if aggregate == nil || !aggregate.Complete || aggregate.Input.Known != 1 || aggregate.Input.Unobserved != 0 || aggregate.Input.Total == nil || *aggregate.Input.Total != 0 || aggregate.Output.Total == nil || *aggregate.Output.Total != 0 {
		t.Fatalf("explicit known zero was not retained: %+v", aggregate)
	}
}

func TestAdapterUsagePresenceM02MalformedKnownDetector(t *testing.T) {
	c, _ := exCollector()
	valid := int64(7)
	adapterObserveReturned(c, adapterPresence("1", ai.AIUsagePresenceSourceOpenAIWire,
		adapterPresenceCount(ai.AIUsagePresenceKnown, nil), adapterPresenceCount(ai.AIUsagePresenceKnown, &valid), true))
	aggregate := c.freeze().UsageObservation.AdapterUsagePresence
	if aggregate == nil || aggregate.Input.Invalid != 1 || aggregate.Input.Known != 0 || aggregate.Input.Total != nil || aggregate.Output.Known != 1 || aggregate.Output.Total == nil || *aggregate.Output.Total != valid || aggregate.Complete {
		t.Fatalf("APO:M02 malformed known count shape must be invalid without erasing sibling status: %+v", aggregate)
	}
}

func TestAdapterUsagePresenceM02ValidIntegerHealthyControl(t *testing.T) {
	c, _ := exCollector()
	input, output := int64(9007199254740993), int64(math.MaxInt64)
	adapterObserveReturned(c, adapterPresence("1", ai.AIUsagePresenceSourceGeminiHTTPBody,
		adapterPresenceCount(ai.AIUsagePresenceKnown, &input), adapterPresenceCount(ai.AIUsagePresenceKnown, &output), false))
	aggregate := c.freeze().UsageObservation.AdapterUsagePresence
	if aggregate == nil || !aggregate.Complete || aggregate.Input.Total == nil || *aggregate.Input.Total != input || aggregate.Output.Total == nil || *aggregate.Output.Total != output {
		t.Fatalf("valid exact int64 metadata was not retained: %+v", aggregate)
	}
}

func TestAdapterUsagePresenceStrictNormalizationAndIndependentSides(t *testing.T) {
	large := int64(9)
	negative := int64(-1)
	badSource := adapterPresence("1", ai.AIUsagePresenceSource("private-provider/prompt/secret"), adapterPresenceCount(ai.AIUsagePresenceAbsent, nil), adapterPresenceCount(ai.AIUsagePresenceNull, nil), false)
	badSource.Diagnostic = "customer content diagnostic sentinel"
	cases := []struct {
		name          string
		presence      *ai.AIUsagePresence
		inputKnown    int64
		inputAbsent   int64
		inputNull     int64
		inputInvalid  int64
		inputUnavail  int64
		inputUnobserv int64
		outputKnown   int64
		outputInvalid int64
	}{
		{name: "bad_schema_invalidates_pair", presence: adapterPresence("2", ai.AIUsagePresenceSourceOpenAIWire, adapterPresenceCount(ai.AIUsagePresenceKnown, &large), adapterPresenceCount(ai.AIUsagePresenceKnown, &large), true), inputInvalid: 1, outputInvalid: 1},
		{name: "raw_source_invalidates_pair", presence: badSource, inputInvalid: 1, outputInvalid: 1},
		{name: "explicit_absent", presence: adapterPresence("1", ai.AIUsagePresenceSourceOpenAIWire, adapterPresenceCount(ai.AIUsagePresenceAbsent, nil), adapterPresenceCount(ai.AIUsagePresenceKnown, &large), true), inputAbsent: 1, outputKnown: 1},
		{name: "absent_with_value_invalid", presence: adapterPresence("1", ai.AIUsagePresenceSourceOpenAIWire, adapterPresenceCount(ai.AIUsagePresenceAbsent, &large), adapterPresenceCount(ai.AIUsagePresenceKnown, &large), false), inputInvalid: 1, outputKnown: 1},
		{name: "null_with_value_invalid_but_sibling_survives", presence: adapterPresence("1", ai.AIUsagePresenceSourceAnthropicRawJSON, adapterPresenceCount(ai.AIUsagePresenceNull, &large), adapterPresenceCount(ai.AIUsagePresenceKnown, &large), false), inputInvalid: 1, outputKnown: 1},
		{name: "negative_known_invalid", presence: adapterPresence("1", ai.AIUsagePresenceSourceOpenAIWire, adapterPresenceCount(ai.AIUsagePresenceKnown, &negative), adapterPresenceCount(ai.AIUsagePresenceKnown, &large), true), inputInvalid: 1, outputKnown: 1},
		{name: "invalid_status_with_value_invalid", presence: adapterPresence("1", ai.AIUsagePresenceSourceOpenAIWire, adapterPresenceCount(ai.AIUsagePresenceInvalid, &large), adapterPresenceCount(ai.AIUsagePresenceAbsent, nil), false), inputInvalid: 1},
		{name: "unavailable_with_value_invalid", presence: adapterPresence("1", ai.AIUsagePresenceSourceOpenAIWire, adapterPresenceCount(ai.AIUsagePresenceUnavailable, &large), adapterPresenceCount(ai.AIUsagePresenceAbsent, nil), false), inputInvalid: 1},
		{name: "unknown_status_invalid", presence: adapterPresence("1", ai.AIUsagePresenceSourceOpenAIWire, adapterPresenceCount(ai.AIUsagePresenceStatus("prompt secret"), nil), adapterPresenceCount(ai.AIUsagePresenceAbsent, nil), false), inputInvalid: 1, inputAbsent: 0},
		{name: "unavailable_source_valid", presence: adapterPresence("1", ai.AIUsagePresenceSourceGeminiUnavailable, adapterPresenceCount(ai.AIUsagePresenceUnavailable, nil), adapterPresenceCount(ai.AIUsagePresenceUnavailable, nil), true), inputUnavail: 1},
		{name: "unavailable_source_contradiction_invalidates_pair", presence: adapterPresence("1", ai.AIUsagePresenceSourceGeminiUnavailable, adapterPresenceCount(ai.AIUsagePresenceKnown, &large), adapterPresenceCount(ai.AIUsagePresenceUnavailable, nil), true), inputInvalid: 1, outputInvalid: 1},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			c, _ := exCollector()
			adapterObserveReturned(c, test.presence)
			aggregate := c.freeze().UsageObservation.AdapterUsagePresence
			if aggregate == nil || aggregate.Responses != 1 || aggregate.Input.Known != test.inputKnown || aggregate.Input.Absent != test.inputAbsent || aggregate.Input.Null != test.inputNull || aggregate.Input.Invalid != test.inputInvalid || aggregate.Input.Unavailable != test.inputUnavail || aggregate.Input.Unobserved != test.inputUnobserv || aggregate.Output.Known != test.outputKnown || aggregate.Output.Invalid != test.outputInvalid {
				t.Fatalf("normalization mismatch: %+v", aggregate)
			}
			encoded, err := json.Marshal(aggregate)
			if err != nil || len(encoded) > 1200 || strings.Contains(string(encoded), "private-provider") || strings.Contains(string(encoded), "prompt secret") || strings.Contains(string(encoded), "customer content diagnostic sentinel") {
				t.Fatalf("aggregate retained unbounded/raw metadata: %s err=%v", encoded, err)
			}
		})
	}
}

func TestAdapterUsagePresenceReturnedGuardAndIncompleteCoverage(t *testing.T) {
	c, _ := exCollector()
	valid := int64(4)
	presence := adapterPresence("1", ai.AIUsagePresenceSourceOpenAIWire,
		adapterPresenceCount(ai.AIUsagePresenceKnown, &valid), adapterPresenceCount(ai.AIUsagePresenceKnown, &valid), true)
	c.observeUsagePresence(presence)
	c.begin("SINGLE", nil)
	c.observeUsagePresence(presence)
	c.returned(nil)
	c.observeUsagePresence(nil)
	c.observeUsagePresence(presence)
	first := c.freeze().UsageObservation.AdapterUsagePresence
	if first == nil || first.Responses != 1 || first.Input.Known != 0 || first.Input.Unobserved != 1 {
		t.Fatalf("in-flight or duplicate hook changed the observation: %+v", first)
	}
	c.begin("BATCH", nil)
	c.returned(nil)
	// A successful response without the new hook leaves the aggregate prefix incomplete.
	incomplete := c.freeze().UsageObservation.AdapterUsagePresence
	if incomplete == nil || incomplete.Responses != 1 || incomplete.Input.Total != nil || incomplete.Output.Total != nil || incomplete.Complete {
		t.Fatalf("unobserved successful call fabricated totals: %+v", incomplete)
	}
	c.begin("SINGLE", nil)
	c.returned(errors.New("private-error-must-not-be-serialized"))
	c.observeUsagePresence(presence)
	if c.freeze().UsageObservation.AdapterUsagePresence.Responses != 1 {
		t.Fatal("error-returned call presence was counted")
	}
}

func TestAdapterUsagePresenceOverflowAndOwnedSnapshots(t *testing.T) {
	t.Run("sum_overflow_is_side_independent", func(t *testing.T) {
		c, _ := exCollector()
		c.usage.adapterPresence = &adapterUsagePresenceCollector{inputSum: math.MaxInt64}
		zero := int64(0)
		adapterObserveReturned(c, adapterPresence("1", ai.AIUsagePresenceSourceOpenAIWire,
			adapterPresenceCount(ai.AIUsagePresenceKnown, &zero), adapterPresenceCount(ai.AIUsagePresenceKnown, &zero), true))
		aggregate := c.freeze().UsageObservation.AdapterUsagePresence
		if aggregate == nil || !aggregate.Input.SumOverflow || aggregate.Input.Total != nil || aggregate.Output.SumOverflow || aggregate.Output.Total == nil || *aggregate.Output.Total != 0 || aggregate.Complete {
			t.Fatalf("side sum overflow handling mismatch: %+v", aggregate)
		}
	})
	t.Run("counter_saturates", func(t *testing.T) {
		c, _ := exCollector()
		c.usage.adapterPresence = &adapterUsagePresenceCollector{}
		c.usage.adapterPresence.r.Input.Known = math.MaxInt64
		one := int64(1)
		adapterObserveReturned(c, adapterPresence("1", ai.AIUsagePresenceSourceOpenAIWire,
			adapterPresenceCount(ai.AIUsagePresenceKnown, &one), adapterPresenceCount(ai.AIUsagePresenceAbsent, nil), true))
		aggregate := c.freeze().UsageObservation.AdapterUsagePresence
		if aggregate == nil || !aggregate.CounterOverflow || aggregate.Input.Known != math.MaxInt64 || aggregate.Input.Total != nil || aggregate.Output.Total != nil {
			t.Fatalf("counter wrapped or overflow totals escaped: %+v", aggregate)
		}
		encoded, err := json.Marshal(aggregate)
		if err != nil || len(encoded) > 1200 {
			t.Fatalf("maximum counter receipt exceeded fixed bound: len=%d err=%v", len(encoded), err)
		}
	})
}

func TestAdapterUsagePresenceTrimSurvivesCalls(t *testing.T) {
	c, member := exCollector()
	c.byteLimit = 3000
	one := int64(1)
	presence := adapterPresence("1", ai.AIUsagePresenceSourceOpenAIWire,
		adapterPresenceCount(ai.AIUsagePresenceKnown, &one), adapterPresenceCount(ai.AIUsagePresenceKnown, &one), true)
	for i := 0; i < 205; i++ {
		c.begin("SINGLE", []models.Conversation{member})
		c.returned(nil)
		c.observeUsagePresence(presence)
	}
	r := c.freeze()
	b, err := json.Marshal(r)
	if err != nil || len(b) > 3000 || r.CallsBegun != 205 || len(r.Calls) >= 200 || r.OmittedCalls == 0 {
		t.Fatalf("new aggregate did not survive bounded retained-call trimming: bytes=%d calls=%d omitted=%d err=%v", len(b), len(r.Calls), r.OmittedCalls, err)
	}
	aggregate := r.UsageObservation.AdapterUsagePresence
	if aggregate == nil || aggregate.Responses != 205 || !aggregate.Complete || aggregate.Input.Total == nil || *aggregate.Input.Total != 205 || aggregate.Output.Total == nil || *aggregate.Output.Total != 205 {
		t.Fatalf("trimmed receipt lost adapter aggregate: %+v", aggregate)
	}
}
