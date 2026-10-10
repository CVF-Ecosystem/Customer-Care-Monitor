package engine

import "math"

// Interface values and local calculations only; never provider usage-presence or billing proof.
type usageObservationReceipt struct {
	Version              string                       `json:"version"`
	Scope                string                       `json:"scope"`
	TokenBasis           string                       `json:"token_basis"`
	Billing              string                       `json:"billing"`
	PriceRevision        string                       `json:"price_revision"`
	Responses            int64                        `json:"responses"`
	InvalidTokens        int64                        `json:"invalid_tokens"`
	Priced               int64                        `json:"priced"`
	Unpriced             int64                        `json:"unpriced"`
	InvalidCosts         int64                        `json:"invalid_costs"`
	TokenOverflow        bool                         `json:"token_overflow"`
	CostOverflow         bool                         `json:"cost_overflow"`
	CounterOverflow      bool                         `json:"counter_overflow"`
	InputTokens          *int64                       `json:"input_tokens"`
	OutputTokens         *int64                       `json:"output_tokens"`
	LocalEstimate        *float64                     `json:"local_estimate_usd"`
	TokensComplete       bool                         `json:"tokens_complete"`
	CostComplete         bool                         `json:"cost_complete"`
	AdapterUsagePresence *adapterUsagePresenceReceipt `json:"adapter_usage_presence,omitempty"`
}

type usageObservationCollector struct {
	r               usageObservationReceipt
	input, output   int64
	cost            float64
	adapterPresence *adapterUsagePresenceCollector
}

func (u *usageObservationCollector) increment(n *int64) {
	if *n == math.MaxInt64 {
		u.r.CounterOverflow = true
		return
	}
	*n++
}

func (c *executionCollector) observeUsage(input, output int, cost float64, priceKnown bool) {
	if c == nil || c.active == nil || c.active.Invocation != "RESPONSE_RETURNED" || c.active.usageObserved {
		return
	}
	c.active.usageObserved = true
	u := &c.usage
	u.increment(&u.r.Responses)
	if input < 0 || output < 0 {
		u.increment(&u.r.InvalidTokens)
	} else if int64(input) > math.MaxInt64-u.input || int64(output) > math.MaxInt64-u.output {
		u.r.TokenOverflow = true
	} else if !u.r.TokenOverflow {
		u.input += int64(input)
		u.output += int64(output)
	}
	if !priceKnown {
		u.increment(&u.r.Unpriced)
		return
	}
	u.increment(&u.r.Priced)
	if cost < 0 || math.IsNaN(cost) || math.IsInf(cost, 0) {
		u.increment(&u.r.InvalidCosts)
	} else if total := u.cost + cost; math.IsInf(total, 0) || math.IsNaN(total) {
		u.r.CostOverflow = true
	} else if !u.r.CostOverflow {
		u.cost = total
	}
}

func (u *usageObservationCollector) freeze(calls int) usageObservationReceipt {
	r := u.r
	r.Version = "ccmai.usage-observation.v1"
	r.Scope = "successful_interface_response_local_estimate"
	r.TokenBasis = "INTERFACE_VALUES_PRESENCE_UNAVAILABLE"
	r.Billing = "NOT_OBSERVED"
	r.PriceRevision = "NOT_CAPTURED"
	r.TokensComplete = calls > 0 && r.Responses == int64(calls) && r.InvalidTokens == 0 && !r.TokenOverflow && !r.CounterOverflow
	r.CostComplete = r.TokensComplete && r.Unpriced == 0 && r.Priced == r.Responses && r.InvalidCosts == 0 && !r.CostOverflow
	if r.TokensComplete {
		input, output := u.input, u.output
		r.InputTokens, r.OutputTokens = &input, &output
	}
	if r.CostComplete {
		cost := u.cost
		r.LocalEstimate = &cost
	}
	if u.adapterPresence != nil {
		r.AdapterUsagePresence = u.adapterPresence.freeze(calls)
	}
	return r
}
