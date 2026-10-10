// @vitest-environment happy-dom
// Synthetic saved-receipt UI fixtures only; this is not provider or billing proof.
import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import RunObservationPanel from '../components/ui/RunObservationPanel.vue'
import enMessages from '../i18n/en'
import viMessages from '../i18n/vi'
import { projectRunObservation, type RunObservationContext } from '../views/Jobs/job-detail/run-observation'
import { projectAdapterUsagePresence } from '../views/Jobs/job-detail/adapter-usage-presence'

const ids = {
  tenant: '11111111-1111-4111-8111-111111111111',
  job: '22222222-2222-4222-8222-222222222222',
  run: '33333333-3333-4333-8333-333333333333',
  member: '44444444-4444-4444-8444-444444444444',
}
const context: RunObservationContext = { tenantId: ids.tenant, jobId: ids.job, runId: ids.run, runJobId: ids.job }
type RecordValue = Record<string, unknown>
type PresenceSide = {
  known: number; absent: number; null: number; invalid: number; unavailable: number; unobserved: number
  sum_overflow: boolean; total: number | null
}
type PresenceAggregate = {
  version: string; basis: string; responses: number; input: PresenceSide; output: PresenceSide
  counter_overflow: boolean; complete: boolean
}
type RunShape = { callsBegun?: number; responseReturned?: number; inFlight?: number }

const mounted: Array<{ unmount: () => void }> = []

function side(overrides: Partial<PresenceSide> = {}): PresenceSide {
  return { known: 1, absent: 0, null: 0, invalid: 0, unavailable: 0, unobserved: 0, sum_overflow: false, total: 0, ...overrides }
}

function knownZeroAggregate(): PresenceAggregate {
  return {
    version: 'ccmai.adapter-usage-presence.v1',
    basis: 'ADAPTER_REPORTED_TOKEN_COUNTS',
    responses: 1,
    input: side(),
    output: side(),
    counter_overflow: false,
    complete: true,
  }
}

function baseExecution(shape: RunShape = {}): RecordValue {
  const callsBegun = shape.callsBegun ?? 1
  const responseReturned = shape.responseReturned ?? callsBegun
  const inFlight = shape.inFlight ?? callsBegun - responseReturned
  const calls = callsBegun === 0 ? [] : [{
    sequence: 1,
    method: 'SINGLE',
    item_count: responseReturned > 0 ? 1 : 0,
    member_ids: responseReturned > 0 ? [ids.member] : [],
    omitted_members: 0,
    members_complete: true,
    metadata_incomplete: false,
    invocation_outcome: responseReturned > 0 ? 'RESPONSE_RETURNED' : inFlight > 0 ? 'IN_FLIGHT' : 'ERROR_RETURNED',
    usage_write_outcome: responseReturned > 0 ? 'WRITE_SUCCEEDED' : 'NOT_ATTEMPTED',
    parsing_outcome: responseReturned > 0 ? 'NOT_SEPARATELY_OBSERVABLE' : 'NOT_ATTEMPTED',
    items_saved: responseReturned > 0 ? 1 : 0,
    items_save_failed: 0,
    items_not_published: 0,
    items_pending: 0,
  }]
  const omittedCalls = callsBegun - calls.length
  const usageComplete = callsBegun > 0 && responseReturned === callsBegun
  const usageResponses = responseReturned
  const usageObservation = {
    version: 'ccmai.usage-observation.v1',
    scope: 'successful_interface_response_local_estimate',
    token_basis: 'INTERFACE_VALUES_PRESENCE_UNAVAILABLE',
    billing: 'NOT_OBSERVED',
    price_revision: 'NOT_CAPTURED',
    responses: usageResponses,
    invalid_tokens: 0,
    priced: usageResponses,
    unpriced: 0,
    invalid_costs: 0,
    token_overflow: false,
    cost_overflow: false,
    counter_overflow: false,
    input_tokens: usageComplete ? 0 : null,
    output_tokens: usageComplete ? 0 : null,
    local_estimate_usd: usageComplete ? 0 : null,
    tokens_complete: usageComplete,
    cost_complete: usageComplete,
  }
  return {
    version: 'ccmai.source-execution.v1',
    scope: 'analyzer_provider_interface_only',
    tenant_id: ids.tenant,
    job_id: ids.job,
    run_id: ids.run,
    mode: 'conditional',
    metadata_incomplete: false,
    calls_begun: callsBegun,
    response_returned: responseReturned,
    error_returned: 0,
    interrupted: 0,
    in_flight: inFlight,
    item_count: responseReturned > 0 ? 1 : 0,
    items_saved: responseReturned > 0 ? 1 : 0,
    items_save_failed: 0,
    items_not_published: 0,
    items_pending: 0,
    usage_writes: {
      NOT_ATTEMPTED: callsBegun - responseReturned,
      WRITE_SUCCEEDED: responseReturned,
      WRITE_FAILED: 0,
      WRITE_OUTCOME_UNKNOWN: 0,
    },
    parsing: {
      NOT_ATTEMPTED: callsBegun - responseReturned,
      ACCEPTED: 0,
      REJECTED: 0,
      NOT_SEPARATELY_OBSERVABLE: responseReturned,
    },
    calls,
    omitted_calls: omittedCalls,
    omitted_members: 0,
    entries_complete: omittedCalls === 0,
    members_complete: true,
    execution_complete: callsBegun > 0 && responseReturned === callsBegun,
    stop_reason: 'NONE',
    usage_observation: usageObservation,
  }
}

function savedSummary(shape: RunShape = {}): string {
  return JSON.stringify({ source_execution: baseExecution(shape) })
}

function savedSummaryWithPresence(aggregate: unknown, shape: RunShape = {}): string {
  const sourceExecution = baseExecution(shape)
  const usage = sourceExecution.usage_observation as RecordValue
  usage.adapter_usage_presence = aggregate
  return JSON.stringify({ source_execution: sourceExecution })
}

function mutatePresence(change: (aggregate: RecordValue) => void, shape: RunShape = {}): string {
  const summary = JSON.parse(savedSummaryWithPresence(knownZeroAggregate(), shape)) as RecordValue
  const sourceExecution = summary.source_execution as RecordValue
  const usage = sourceExecution.usage_observation as RecordValue
  const aggregate = usage.adapter_usage_presence as RecordValue
  change(aggregate)
  return JSON.stringify(summary)
}

const invalidNumericMutations: Array<{ label: string; change: (aggregate: RecordValue) => void }> = [
  { label: 'string response count', change: aggregate => { aggregate.responses = '1' } },
  { label: 'negative response count', change: aggregate => { aggregate.responses = -1 } },
  { label: 'fractional response count', change: aggregate => { aggregate.responses = 1.5 } },
  { label: 'unsafe response count', change: aggregate => { aggregate.responses = Number.MAX_SAFE_INTEGER + 1 } },
  { label: 'string side counter', change: aggregate => { (aggregate.input as RecordValue).known = '1' } },
  { label: 'negative side counter', change: aggregate => { (aggregate.input as RecordValue).known = -1 } },
  { label: 'fractional side counter', change: aggregate => { (aggregate.input as RecordValue).known = 0.5 } },
  { label: 'unsafe side counter', change: aggregate => { (aggregate.input as RecordValue).known = Number.MAX_SAFE_INTEGER + 1 } },
]

const invalidShapeMutations: Array<{ label: string; change: (aggregate: RecordValue) => void }> = [
  { label: 'missing basis', change: aggregate => { delete aggregate.basis } },
  { label: 'wrong basis', change: aggregate => { aggregate.basis = 'INTERFACE_VALUES' } },
  { label: 'missing side total', change: aggregate => { delete (aggregate.input as RecordValue).total } },
  { label: 'extra side key', change: aggregate => { (aggregate.output as RecordValue).future_field = true } },
]

const expectedKnownZero = {
  available: true,
  values: {
    adapter_responses: 1,
    input_known: 1, input_absent: 0, input_null: 0, input_invalid: 0, input_unavailable: 0, input_unobserved: 0,
    input_sum_overflow: false, input_total: 0,
    output_known: 1, output_absent: 0, output_null: 0, output_invalid: 0, output_unavailable: 0, output_unobserved: 0,
    output_sum_overflow: false, output_total: 0,
    adapter_counter_overflow: false,
    complete: true,
  },
}

function mountPanel(summary: string, locale: 'en' | 'vi' = 'en') {
  const i18n = createI18n({ legacy: false, locale, fallbackLocale: 'en', messages: { en: enMessages, vi: viMessages } })
  const wrapper = mount(RunObservationPanel, {
    attachTo: document.body,
    props: { summary, context },
    global: { plugins: [i18n] },
  })
  mounted.push(wrapper)
  return wrapper
}

describe('saved adapter usage-presence projector and panel', () => {
  afterEach(() => {
    for (const wrapper of mounted.splice(0)) wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('keeps the R072 projection shape and hides the optional section when the extension is absent', () => {
    const summary = savedSummary()
    const legacy = projectRunObservation(summary, context)
    expect(Object.keys(legacy)).toEqual(['preparation', 'execution', 'rules', 'usage'])
    expect(legacy.execution.available).toBe(true)
    expect(legacy.usage.available).toBe(true)
    expect(projectAdapterUsagePresence(summary, context)).toBeNull()

    const panel = mountPanel(summary)
    expect(panel.findAll('section')).toHaveLength(4)
    expect(panel.find('[data-section="adapter_presence"]').exists()).toBe(false)
  })

  it('projects a literal fixed allowlist and preserves known zero totals', () => {
    const summary = savedSummaryWithPresence(knownZeroAggregate())
    const legacy = projectRunObservation(summary, context)
    expect(legacy.execution.available).toBe(true)
    expect(legacy.usage.available).toBe(true)
    expect(projectAdapterUsagePresence(summary, context)).toEqual(expectedKnownZero)
  })

  it('keeps side totals independent and preserves null for a side with a null count', () => {
    const aggregate = knownZeroAggregate()
    aggregate.output = side({ known: 0, null: 1, total: null })
    aggregate.complete = false
    const summary = savedSummaryWithPresence(aggregate)
    const legacy = projectRunObservation(summary, context)
    expect(legacy.execution.available).toBe(true)
    expect(legacy.usage.available).toBe(true)
    expect(projectAdapterUsagePresence(summary, context)).toEqual({
      available: true,
      values: {
        adapter_responses: 1,
        input_known: 1, input_absent: 0, input_null: 0, input_invalid: 0, input_unavailable: 0, input_unobserved: 0,
        input_sum_overflow: false, input_total: 0,
        output_known: 0, output_absent: 0, output_null: 1, output_invalid: 0, output_unavailable: 0, output_unobserved: 0,
        output_sum_overflow: false, output_total: null,
        adapter_counter_overflow: false,
        complete: false,
      },
    })
  })

  it('withholds prefix totals while begun calls remain in flight', () => {
    const aggregate = knownZeroAggregate()
    aggregate.input.total = null
    aggregate.output.total = null
    aggregate.complete = false
    const summary = savedSummaryWithPresence(aggregate, { callsBegun: 2, responseReturned: 1, inFlight: 1 })
    const legacy = projectRunObservation(summary, context)
    expect(legacy.execution.available).toBe(true)
    expect(legacy.execution.available && legacy.execution.values.calls_begun).toBe(2)
    expect(legacy.usage.available).toBe(true)
    expect(legacy.usage.available && legacy.usage.values.input_tokens).toBeNull()
    const projected = projectAdapterUsagePresence(summary, context)
    expect(projected).toMatchObject({ available: true, values: { input_total: null, output_total: null, complete: false } })
  })

  it('shows no-call and overflow states without inventing totals', () => {
    const empty = knownZeroAggregate()
    empty.responses = 0
    empty.input = side({ known: 0, total: null })
    empty.output = side({ known: 0, total: null })
    empty.complete = false
    const noCallSummary = savedSummaryWithPresence(empty, { callsBegun: 0, responseReturned: 0, inFlight: 0 })
    expect(projectRunObservation(noCallSummary, context).usage.available).toBe(true)
    expect(projectAdapterUsagePresence(noCallSummary, context)).toMatchObject({
      available: true, values: { adapter_responses: 0, input_total: null, output_total: null, complete: false },
    })

    const sideOverflow = knownZeroAggregate()
    sideOverflow.input = side({ sum_overflow: true, total: null })
    sideOverflow.complete = false
    const sideOverflowSummary = savedSummaryWithPresence(sideOverflow)
    expect(projectRunObservation(sideOverflowSummary, context).usage.available).toBe(true)
    expect(projectAdapterUsagePresence(sideOverflowSummary, context)).toMatchObject({
      available: true, values: { input_sum_overflow: true, input_total: null, output_total: 0, complete: false },
    })

    const counterOverflow = knownZeroAggregate()
    counterOverflow.input.total = null
    counterOverflow.output.total = null
    counterOverflow.counter_overflow = true
    counterOverflow.complete = false
    const counterOverflowSummary = savedSummaryWithPresence(counterOverflow)
    expect(projectRunObservation(counterOverflowSummary, context).usage.available).toBe(true)
    expect(projectAdapterUsagePresence(counterOverflowSummary, context)).toMatchObject({
      available: true, values: { adapter_counter_overflow: true, input_total: null, output_total: null, complete: false },
    })
  })

  it('accepts the safe-integer boundary and rejects unsafe values or unsafe counter sums', () => {
    const maximum = Number.MAX_SAFE_INTEGER
    const boundary = knownZeroAggregate()
    boundary.responses = maximum
    boundary.input = side({ known: maximum, total: maximum })
    boundary.output = side({ known: maximum, total: maximum })
    const boundarySummary = savedSummaryWithPresence(boundary, { callsBegun: maximum, responseReturned: maximum })
    expect(projectRunObservation(boundarySummary, context).execution.available).toBe(true)
    expect(projectRunObservation(boundarySummary, context).usage.available).toBe(true)
    expect(projectAdapterUsagePresence(boundarySummary, context)).toMatchObject({
      available: true, values: { adapter_responses: maximum, input_total: maximum, output_total: maximum, complete: true },
    })

    const unsafeSummary = mutatePresence(aggregate => {
      const input = aggregate.input as RecordValue
      input.total = maximum + 1
    }, { callsBegun: maximum, responseReturned: maximum })
    expect(projectRunObservation(unsafeSummary, context).usage.available).toBe(true)
    expect(projectAdapterUsagePresence(unsafeSummary, context)).toEqual({ available: false })

    const unsafeSumSummary = mutatePresence(aggregate => {
      const input = aggregate.input as RecordValue
      input.absent = 1
    }, { callsBegun: maximum, responseReturned: maximum })
    expect(projectRunObservation(unsafeSumSummary, context).usage.available).toBe(true)
    expect(projectAdapterUsagePresence(unsafeSumSummary, context)).toEqual({ available: false })
  })

  it.each(invalidNumericMutations)('rejects $label while preserving valid legacy usage', ({ change }) => {
    const healthy = savedSummaryWithPresence(knownZeroAggregate())
    expect(projectAdapterUsagePresence(healthy, context)).toEqual(expectedKnownZero)

    const summary = mutatePresence(change)
    expect(projectRunObservation(summary, context).usage.available).toBe(true)
    expect(projectAdapterUsagePresence(summary, context)).toEqual({ available: false })
  })

  it.each(invalidShapeMutations)('rejects $label without changing the legacy projection', ({ change }) => {
    const healthy = savedSummaryWithPresence(knownZeroAggregate())
    expect(projectAdapterUsagePresence(healthy, context)).toEqual(expectedKnownZero)

    const summary = mutatePresence(change)
    expect(projectRunObservation(summary, context).usage.available).toBe(true)
    expect(projectAdapterUsagePresence(summary, context)).toEqual({ available: false })
  })

  it('rejects unknown, future, raw, missing, and wrongly typed extension fields while leaving legacy usage available', () => {
    const invalidExtensions: unknown[] = [
      null,
      'PRIVATE_EXTENSION_CANARY',
      42,
      false,
      ['PRIVATE_EXTENSION_CANARY'],
      { ...knownZeroAggregate(), version: 'ccmai.adapter-usage-presence.v99' },
      { ...knownZeroAggregate(), raw_provider_model: 'PRIVATE_EXTENSION_CANARY' },
    ]
    for (const invalid of invalidExtensions) {
      const healthy = savedSummaryWithPresence(knownZeroAggregate())
      expect(projectAdapterUsagePresence(healthy, context)).toEqual(expectedKnownZero)
      const summary = savedSummaryWithPresence(invalid)
      expect(projectRunObservation(summary, context).usage.available).toBe(true)
      const projected = projectAdapterUsagePresence(summary, context)
      expect(projected).toEqual({ available: false })
      expect(JSON.stringify(projected)).not.toContain('PRIVATE_EXTENSION_CANARY')
    }

    const badSide = mutatePresence(aggregate => { aggregate.input = [] })
    expect(projectAdapterUsagePresence(savedSummaryWithPresence(knownZeroAggregate()), context)).toEqual(expectedKnownZero)
    expect(projectRunObservation(badSide, context).usage.available).toBe(true)
    expect(projectAdapterUsagePresence(badSide, context)).toEqual({ available: false })
  })

  it('requires safe boolean flags, exact status sums, bounded responses, and derived completeness', () => {
    const mutations: Array<(aggregate: RecordValue) => void> = [
      aggregate => { aggregate.counter_overflow = 'false' },
      aggregate => { aggregate.complete = 1 },
      aggregate => { (aggregate.input as RecordValue).sum_overflow = 0 },
      aggregate => { (aggregate.input as RecordValue).known = 0 },
      aggregate => { aggregate.responses = 2 },
      aggregate => { aggregate.complete = false },
      aggregate => { (aggregate.input as RecordValue).future_field = 'PRIVATE_EXTENSION_CANARY' },
    ]
    for (const change of mutations) {
      const healthy = savedSummaryWithPresence(knownZeroAggregate())
      expect(projectAdapterUsagePresence(healthy, context)).toEqual(expectedKnownZero)
      const summary = mutatePresence(change)
      expect(projectRunObservation(summary, context).usage.available).toBe(true)
      expect(projectAdapterUsagePresence(summary, context)).toEqual({ available: false })
      expect(JSON.stringify(projectAdapterUsagePresence(summary, context))).not.toContain('PRIVATE_EXTENSION_CANARY')
    }

    const validUnobserved = knownZeroAggregate()
    validUnobserved.input = side({ known: 0, unobserved: 1, total: null })
    validUnobserved.complete = false
    const unobservedSummary = savedSummaryWithPresence(validUnobserved)
    expect(projectRunObservation(unobservedSummary, context).usage.available).toBe(true)
    expect(projectAdapterUsagePresence(unobservedSummary, context)).toMatchObject({
      available: true, values: { input_known: 0, input_unobserved: 1, input_total: null, complete: false },
    })

    const mismatchButBounded = knownZeroAggregate()
    mismatchButBounded.responses = 2
    mismatchButBounded.input = side({ known: 2, total: null })
    mismatchButBounded.output = side({ known: 2, total: null })
    mismatchButBounded.complete = false
    const boundedSummary = savedSummaryWithPresence(mismatchButBounded)
    expect(projectRunObservation(boundedSummary, context).execution.available).toBe(true)
    expect(projectRunObservation(boundedSummary, context).usage.available).toBe(true)
    expect(projectAdapterUsagePresence(boundedSummary, context)).toEqual({ available: false })

    const prefix = knownZeroAggregate()
    prefix.input.total = null
    prefix.output.total = null
    prefix.complete = false
    const prefixSummary = savedSummaryWithPresence(prefix, { callsBegun: 2, responseReturned: 1, inFlight: 1 })
    expect(projectRunObservation(prefixSummary, context).execution.available).toBe(true)
    expect(projectRunObservation(prefixSummary, context).usage.available).toBe(true)
    expect(projectAdapterUsagePresence(prefixSummary, context)).toMatchObject({
      available: true, values: { input_total: null, output_total: null, complete: false },
    })
  })

  it('gates on the existing run binding and usage projection without a scalar fallback', () => {
    const valid = savedSummaryWithPresence(knownZeroAggregate())
    expect(projectAdapterUsagePresence(valid, context)).toEqual(expectedKnownZero)

    const wrongRun = JSON.parse(valid) as RecordValue
    const execution = wrongRun.source_execution as RecordValue
    execution.run_id = '77777777-7777-4777-8777-777777777777'
    const wrongRunSummary = JSON.stringify(wrongRun)
    expect(projectRunObservation(wrongRunSummary, context).usage.available).toBe(false)
    expect(projectAdapterUsagePresence(wrongRunSummary, context)).toEqual({ available: false })

    const badLegacyUsage = JSON.parse(valid) as RecordValue
    const badExecution = badLegacyUsage.source_execution as RecordValue
    const usage = badExecution.usage_observation as RecordValue
    usage.input_tokens = null
    const badLegacySummary = JSON.stringify(badLegacyUsage)
    expect(projectRunObservation(badLegacySummary, context).execution.available).toBe(true)
    expect(projectRunObservation(badLegacySummary, context).usage.available).toBe(false)
    expect(projectAdapterUsagePresence(badLegacySummary, context)).toEqual({ available: false })
  })

  it('renders translated zero, unknown side, and note in English and Vietnamese', () => {
    const aggregate = knownZeroAggregate()
    aggregate.output = side({ known: 0, absent: 1, total: null })
    aggregate.complete = false
    const summary = savedSummaryWithPresence(aggregate)

    for (const locale of ['en', 'vi'] as const) {
      const messages = locale === 'en' ? enMessages : viMessages
      const panel = mountPanel(summary, locale)
      const section = panel.find('[data-section="adapter_presence"]')
      expect(section.exists()).toBe(true)
      expect(section.find('h4').text()).toBe(messages.run_obs_adapter_presence)
      expect(section.find('[data-field="input_total"]').text()).toBe('0')
      expect(section.find('[data-field="output_total"]').text()).toBe(messages.run_obs_unknown)
      expect(section.find('[data-field="complete"]').text()).toBe(messages.run_obs_no)
      expect(section.text()).toContain(messages.run_obs_adapter_presence_note)
      expect(section.findAll('dt').map(label => label.text())).toEqual([
        messages.run_obs_field_adapter_responses,
        messages.run_obs_field_input_known,
        messages.run_obs_field_input_absent,
        messages.run_obs_field_input_null,
        messages.run_obs_field_input_invalid,
        messages.run_obs_field_input_unavailable,
        messages.run_obs_field_input_unobserved,
        messages.run_obs_field_input_sum_overflow,
        messages.run_obs_field_input_total,
        messages.run_obs_field_output_known,
        messages.run_obs_field_output_absent,
        messages.run_obs_field_output_null,
        messages.run_obs_field_output_invalid,
        messages.run_obs_field_output_unavailable,
        messages.run_obs_field_output_unobserved,
        messages.run_obs_field_output_sum_overflow,
        messages.run_obs_field_output_total,
        messages.run_obs_field_adapter_counter_overflow,
        messages.run_obs_field_complete,
      ])
    }
  })

  it('shows invalid present extensions as unavailable without hiding legacy usage or leaking private content', () => {
    const privateMarker = 'PRIVATE_EXTENSION_CANARY'
    const summary = savedSummaryWithPresence({ ...knownZeroAggregate(), raw_provider_model: privateMarker })
    const legacy = projectRunObservation(summary, context)
    expect(legacy.execution.available).toBe(true)
    expect(legacy.usage.available).toBe(true)

    const panel = mountPanel(summary, 'en')
    const presence = panel.find('[data-section="adapter_presence"]')
    expect(presence.exists()).toBe(true)
    expect(presence.find('[data-testid="observation-unavailable"]').text()).toBe(enMessages.run_obs_unavailable)
    expect(panel.find('[data-section="usage"] [data-field="local_estimate_usd"]').text()).toBe('0 USD')
    expect(panel.text()).not.toContain(privateMarker)
    expect(panel.html()).not.toContain(privateMarker)
  })
})
