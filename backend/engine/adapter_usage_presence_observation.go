package engine

import (
	"math"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
)

const adapterUsagePresenceSchemaVersion = "1"
const adapterUsagePresenceVersion = "ccmai.adapter-usage-presence.v1"
const adapterUsagePresenceBasis = "ADAPTER_REPORTED_TOKEN_COUNTS"

const (
	adapterUsageKnown       = "known"
	adapterUsageAbsent      = "absent"
	adapterUsageNull        = "null"
	adapterUsageInvalid     = "invalid"
	adapterUsageUnavailable = "unavailable"
	adapterUsageUnobserved  = "unobserved"
)

// Fixed aggregate only. It contains no provider-specific strings or per-call records.
type adapterUsagePresenceReceipt struct {
	Version         string                          `json:"version"`
	Basis           string                          `json:"basis"`
	Responses       int64                           `json:"responses"`
	Input           adapterUsagePresenceSideReceipt `json:"input"`
	Output          adapterUsagePresenceSideReceipt `json:"output"`
	CounterOverflow bool                            `json:"counter_overflow"`
	Complete        bool                            `json:"complete"`
}

type adapterUsagePresenceSideReceipt struct {
	Known       int64  `json:"known"`
	Absent      int64  `json:"absent"`
	Null        int64  `json:"null"`
	Invalid     int64  `json:"invalid"`
	Unavailable int64  `json:"unavailable"`
	Unobserved  int64  `json:"unobserved"`
	SumOverflow bool   `json:"sum_overflow"`
	Total       *int64 `json:"total"`
}

type adapterUsagePresenceCollector struct {
	r                   adapterUsagePresenceReceipt
	inputSum, outputSum int64
}

type normalizedAdapterUsagePresence struct {
	input  normalizedAdapterUsageCount
	output normalizedAdapterUsageCount
}

type normalizedAdapterUsageCount struct {
	status string
	value  int64
}

func (c *executionCollector) observeUsagePresence(presence *ai.AIUsagePresence) {
	if c == nil || c.active == nil || c.active.Invocation != "RESPONSE_RETURNED" || c.active.usagePresenceObserved {
		return
	}
	c.active.usagePresenceObserved = true
	c.usage.observeAdapterUsagePresence(presence)
}

func (u *usageObservationCollector) observeAdapterUsagePresence(presence *ai.AIUsagePresence) {
	if u == nil {
		return
	}
	if u.adapterPresence == nil {
		u.adapterPresence = &adapterUsagePresenceCollector{}
	}
	u.adapterPresence.observe(presence)
}

func (c *adapterUsagePresenceCollector) observe(presence *ai.AIUsagePresence) {
	if c == nil {
		return
	}
	incrementAdapterUsageCounter(&c.r.Responses, &c.r.CounterOverflow)
	normalized := normalizeAdapterUsagePresence(presence)
	c.observeSide(&c.r.Input, normalized.input, &c.inputSum)
	c.observeSide(&c.r.Output, normalized.output, &c.outputSum)
}

func (c *adapterUsagePresenceCollector) observeSide(side *adapterUsagePresenceSideReceipt, count normalizedAdapterUsageCount, sum *int64) {
	switch count.status {
	case adapterUsageKnown:
		incrementAdapterUsageCounter(&side.Known, &c.r.CounterOverflow)
		if !side.SumOverflow {
			if count.value > math.MaxInt64-*sum {
				side.SumOverflow = true
			} else {
				*sum += count.value
			}
		}
	case adapterUsageAbsent:
		incrementAdapterUsageCounter(&side.Absent, &c.r.CounterOverflow)
	case adapterUsageNull:
		incrementAdapterUsageCounter(&side.Null, &c.r.CounterOverflow)
	case adapterUsageUnavailable:
		incrementAdapterUsageCounter(&side.Unavailable, &c.r.CounterOverflow)
	case adapterUsageUnobserved:
		incrementAdapterUsageCounter(&side.Unobserved, &c.r.CounterOverflow)
	default:
		incrementAdapterUsageCounter(&side.Invalid, &c.r.CounterOverflow)
	}
}

func incrementAdapterUsageCounter(counter *int64, overflow *bool) {
	if *counter == math.MaxInt64 {
		*overflow = true
		return
	}
	*counter++
}

func normalizeAdapterUsagePresence(presence *ai.AIUsagePresence) normalizedAdapterUsagePresence {
	invalid := normalizedAdapterUsagePresence{
		input:  normalizedAdapterUsageCount{status: adapterUsageInvalid},
		output: normalizedAdapterUsageCount{status: adapterUsageInvalid},
	}
	if presence == nil {
		return normalizedAdapterUsagePresence{
			input:  normalizedAdapterUsageCount{status: adapterUsageUnobserved},
			output: normalizedAdapterUsageCount{status: adapterUsageUnobserved},
		}
	}
	if presence.SchemaVersion != adapterUsagePresenceSchemaVersion || !allowedAdapterUsagePresenceSource(presence.Source) {
		return invalid
	}
	if presence.Source == ai.AIUsagePresenceSourceGeminiUnavailable {
		if presence.InputTokens.Status == ai.AIUsagePresenceUnavailable && presence.InputTokens.Value == nil &&
			presence.OutputTokens.Status == ai.AIUsagePresenceUnavailable && presence.OutputTokens.Value == nil {
			return normalizedAdapterUsagePresence{
				input:  normalizedAdapterUsageCount{status: adapterUsageUnavailable},
				output: normalizedAdapterUsageCount{status: adapterUsageUnavailable},
			}
		}
		return invalid
	}
	return normalizedAdapterUsagePresence{
		input:  normalizeAdapterUsageCount(presence.InputTokens),
		output: normalizeAdapterUsageCount(presence.OutputTokens),
	}
}

func allowedAdapterUsagePresenceSource(source ai.AIUsagePresenceSource) bool {
	switch source {
	case ai.AIUsagePresenceSourceOpenAIWire,
		ai.AIUsagePresenceSourceAnthropicRawJSON,
		ai.AIUsagePresenceSourceGeminiHTTPBody,
		ai.AIUsagePresenceSourceGeminiUnavailable:
		return true
	default:
		return false
	}
}

func normalizeAdapterUsageCount(count ai.AIUsageCount) normalizedAdapterUsageCount {
	switch count.Status {
	case ai.AIUsagePresenceKnown:
		value := int64(0)
		if count.Value != nil {
			value = *count.Value
		}
		if count.Value == nil || value < 0 {
			return normalizedAdapterUsageCount{status: adapterUsageInvalid}
		}
		return normalizedAdapterUsageCount{status: adapterUsageKnown, value: value}
	case ai.AIUsagePresenceAbsent:
		if count.Value == nil {
			return normalizedAdapterUsageCount{status: adapterUsageAbsent}
		}
	case ai.AIUsagePresenceNull:
		if count.Value == nil {
			return normalizedAdapterUsageCount{status: adapterUsageNull}
		}
	case ai.AIUsagePresenceInvalid:
		if count.Value == nil {
			return normalizedAdapterUsageCount{status: adapterUsageInvalid}
		}
	case ai.AIUsagePresenceUnavailable:
		if count.Value == nil {
			return normalizedAdapterUsageCount{status: adapterUsageUnavailable}
		}
	}
	return normalizedAdapterUsageCount{status: adapterUsageInvalid}
}

func (c *adapterUsagePresenceCollector) freeze(calls int) *adapterUsagePresenceReceipt {
	if c == nil {
		return nil
	}
	r := c.r
	r.Version = adapterUsagePresenceVersion
	r.Basis = adapterUsagePresenceBasis
	allResponsesObserved := calls > 0 && int64(calls) == r.Responses && !r.CounterOverflow
	if allResponsesObserved && adapterUsageSideAllKnown(r.Input, r.Responses) && !r.Input.SumOverflow {
		total := c.inputSum
		r.Input.Total = &total
	}
	if allResponsesObserved && adapterUsageSideAllKnown(r.Output, r.Responses) && !r.Output.SumOverflow {
		total := c.outputSum
		r.Output.Total = &total
	}
	r.Complete = r.Input.Total != nil && r.Output.Total != nil
	return &r
}

func adapterUsageSideAllKnown(side adapterUsagePresenceSideReceipt, responses int64) bool {
	return responses > 0 && side.Known == responses && side.Absent == 0 && side.Null == 0 && side.Invalid == 0 && side.Unavailable == 0 && side.Unobserved == 0
}
