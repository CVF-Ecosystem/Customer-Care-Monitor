import { describe, expect, it } from 'vitest'
import { projectRunObservation, type RunObservation, type RunObservationContext } from '../views/Jobs/job-detail/run-observation'

const ids = {
  tenant: '11111111-1111-4111-8111-111111111111',
  job: '22222222-2222-4222-8222-222222222222',
  run: '33333333-3333-4333-8333-333333333333',
  conversation: '44444444-4444-4444-8444-444444444444',
  member: '55555555-5555-4555-8555-555555555555',
}

const context: RunObservationContext = { tenantId: ids.tenant, jobId: ids.job, runId: ids.run, runJobId: ids.job }

type ParsedSummary = {
  source_preparation: Record<string, unknown>
  source_execution: Record<string, unknown> & { usage_observation: Record<string, unknown> }
  rule_observation: Record<string, unknown>
}

function legacySummary(): string {
  return JSON.stringify({
    source_preparation: {
      version: 'ccmai.source-preparation.v1', scope: 'preparation_only', tenant_id: ids.tenant, job_id: ids.job,
      run_id: ids.run, mode: 'conditional', metadata_incomplete: false, selection_status: 'COMPLETE',
      selected: 1, visited: 1, unvisited: 0, counts: { PREPARED_FOR_INFERENCE: 1 },
      entries: [{
        conversation_id: ids.conversation, outcome: 'PREPARED_FOR_INFERENCE', snapshot_schema: 'ccma.snapshot.v1',
        snapshot_digest: 'a'.repeat(64), coverage: 'complete', reference: { state: 'NONE' }, metadata_incomplete: false,
      }],
      omitted_entries: 0, entries_complete: true, scan_complete: true,
    },
    source_execution: {
      version: 'ccmai.source-execution.v1', scope: 'analyzer_provider_interface_only', tenant_id: ids.tenant,
      job_id: ids.job, run_id: ids.run, mode: 'conditional', metadata_incomplete: false,
      calls_begun: 1, response_returned: 1, error_returned: 0, interrupted: 0, in_flight: 0,
      item_count: 1, items_saved: 1, items_save_failed: 0, items_not_published: 0, items_pending: 0,
      usage_writes: { NOT_ATTEMPTED: 0, WRITE_SUCCEEDED: 1, WRITE_FAILED: 0, WRITE_OUTCOME_UNKNOWN: 0 },
      parsing: { NOT_ATTEMPTED: 0, ACCEPTED: 0, REJECTED: 0, NOT_SEPARATELY_OBSERVABLE: 1 },
      calls: [{
        sequence: 1, method: 'SINGLE', item_count: 1, member_ids: [ids.member], omitted_members: 0,
        members_complete: true, metadata_incomplete: false, invocation_outcome: 'RESPONSE_RETURNED',
        usage_write_outcome: 'WRITE_SUCCEEDED', parsing_outcome: 'NOT_SEPARATELY_OBSERVABLE',
        items_saved: 1, items_save_failed: 0, items_not_published: 0, items_pending: 0,
      }],
      omitted_calls: 0, omitted_members: 0, entries_complete: true, members_complete: true,
      execution_complete: true, stop_reason: 'NONE',
      usage_observation: {
        version: 'ccmai.usage-observation.v1', scope: 'successful_interface_response_local_estimate',
        token_basis: 'INTERFACE_VALUES_PRESENCE_UNAVAILABLE', billing: 'NOT_OBSERVED', price_revision: 'NOT_CAPTURED',
        responses: 1, invalid_tokens: 0, priced: 1, unpriced: 0, invalid_costs: 0,
        token_overflow: false, cost_overflow: false, counter_overflow: false,
        input_tokens: 0, output_tokens: 0, local_estimate_usd: 0, tokens_complete: true, cost_complete: true,
      },
    },
    rule_observation: {
      version: 'ccmai.rule-observation.v1', scope: 'analyzer_job_input_only', tenant_id: ids.tenant, job_id: ids.job,
      run_id: ids.run, job_type: 'qc_analysis', metadata_incomplete: false, rule_authority: 'JOB_INPUT_OBSERVED',
      policy_version_status: 'NOT_AVAILABLE', permission_status: 'NOT_OBSERVED_AT_ANALYZER',
      wait_data_status: 'NOT_IMPLEMENTED', fingerprint_status: 'OBSERVED', fingerprint: 'b'.repeat(64),
    },
  })
}

function changeSummary(change: (summary: ParsedSummary) => void): string {
  const summary = JSON.parse(legacySummary()) as ParsedSummary
  change(summary)
  return JSON.stringify(summary)
}

const expectedLegacyProjection: RunObservation = {
  preparation: { available: true, values: {
    selected: 1, visited: 1, unvisited: 0, scan_complete: true, omitted_entries: 0, entries_complete: true,
    EMPTY_SOURCE: 0, UNCHANGED_VERIFIED: 0, PREPARED_FOR_INFERENCE: 1, SNAPSHOT_ERROR: 0,
    SOURCE_VERSION_ERROR: 0, PREPARATION_INTERRUPTED: 0,
  } },
  execution: { available: true, values: {
    calls_begun: 1, response_returned: 1, error_returned: 0, interrupted: 0, in_flight: 0,
    item_count: 1, items_saved: 1, items_save_failed: 0, items_not_published: 0, items_pending: 0,
    omitted_calls: 0, omitted_members: 0,
    NOT_ATTEMPTED: 0, WRITE_SUCCEEDED: 1, WRITE_FAILED: 0, WRITE_OUTCOME_UNKNOWN: 0,
    execution_complete: true, entries_complete: true, members_complete: true, stop_reason: 'NONE',
  } },
  rules: { available: true, values: { fingerprint: 'b'.repeat(64) } },
  usage: { available: true, values: {
    responses: 1, invalid_tokens: 0, priced: 1, unpriced: 0, invalid_costs: 0,
    input_tokens: 0, output_tokens: 0, local_estimate_usd: 0,
    tokens_complete: true, cost_complete: true,
    token_overflow: false, cost_overflow: false, counter_overflow: false,
  } },
}

const representativeR071Aggregate = {
  version: 'ccmai.adapter-usage-presence.v1',
  basis: 'ADAPTER_REPORTED_TOKEN_COUNTS',
  responses: 1,
  input: { known: 1, absent: 0, null: 0, invalid: 0, unavailable: 0, unobserved: 0, sum_overflow: false, total: 0 },
  output: { known: 1, absent: 0, null: 0, invalid: 0, unavailable: 0, unobserved: 0, sum_overflow: false, total: 0 },
  counter_overflow: false,
  complete: true,
}

describe('saved-run adapter usage extension compatibility', () => {
  it('preserves the exact full legacy projection with the extension absent or source-shaped', () => {
    const legacy = projectRunObservation(legacySummary(), context)
    const extended = projectRunObservation(changeSummary(summary => {
      summary.source_execution.usage_observation.adapter_usage_presence = representativeR071Aggregate
    }), context)

    expect(legacy).toEqual(expectedLegacyProjection)
    expect(extended).toEqual(expectedLegacyProjection)
    expect(extended).toEqual(legacy)
  })

  it('ignores null, malformed, future, primitive, array, and private extension content without projecting it', () => {
    const privateMarker = 'PRIVATE_ADAPTER_USAGE_MARKER_MUST_NOT_LEAK'
    const opaqueExtensions: unknown[] = [
      null,
      { version: null, input: { total: 'malformed' } },
      { version: 'ccmai.adapter-usage-presence.v99', futureField: privateMarker },
      'arbitrary extension string',
      42,
      false,
      [privateMarker, { nested: privateMarker }],
    ]

    for (const extension of opaqueExtensions) {
      const projected = projectRunObservation(changeSummary(summary => {
        summary.source_execution.usage_observation.adapter_usage_presence = extension
      }), context)
      const output = JSON.stringify(projected)
      expect(projected).toEqual(expectedLegacyProjection)
      expect(output).not.toContain('adapter_usage_presence')
      expect(output).not.toContain(privateMarker)
    }
  })

  it('still rejects unrelated keys and malformed, unsafe, or contradictory legacy usage', () => {
    const invalidLegacyCases: Array<{ name: string; mutate: (usage: Record<string, unknown>) => void }> = [
      { name: 'unrelated key', mutate: usage => { usage.unrelated_extension = { marker: 'must reject' } } },
      { name: 'missing legacy field', mutate: usage => { delete usage.token_basis } },
      { name: 'future legacy version', mutate: usage => { usage.version = 'ccmai.usage-observation.v2' } },
      { name: 'unsafe legacy number', mutate: usage => { usage.input_tokens = Number.MAX_SAFE_INTEGER + 1 } },
      { name: 'contradictory totals', mutate: usage => { usage.input_tokens = null } },
    ]

    for (const withExtension of [false, true]) {
      for (const invalidCase of invalidLegacyCases) {
        const projected = projectRunObservation(changeSummary(summary => {
          const usage = summary.source_execution.usage_observation
          invalidCase.mutate(usage)
          if (withExtension) usage.adapter_usage_presence = representativeR071Aggregate
        }), context)

        expect(projected.execution.available, `${invalidCase.name}; extension ${withExtension ? 'present' : 'absent'}`).toBe(true)
        expect(projected.usage, `${invalidCase.name}; extension ${withExtension ? 'present' : 'absent'}`).toEqual({ available: false })
      }
    }
  })

  it('preserves execution binding, known zero, and legacy null totals despite extension totals', () => {
    const wrongBinding = projectRunObservation(changeSummary(summary => {
      summary.source_execution.tenant_id = '77777777-7777-4777-8777-777777777777'
      summary.source_execution.usage_observation.adapter_usage_presence = representativeR071Aggregate
    }), context)
    expect(wrongBinding.preparation.available).toBe(true)
    expect(wrongBinding.execution).toEqual({ available: false })
    expect(wrongBinding.usage).toEqual({ available: false })

    const knownZero = projectRunObservation(legacySummary(), context)
    expect(knownZero.usage).toEqual(expectedLegacyProjection.usage)
    if (knownZero.usage.available) {
      expect(knownZero.usage.values.input_tokens).toBe(0)
      expect(knownZero.usage.values.output_tokens).toBe(0)
    }

    const nullTotals = projectRunObservation(changeSummary(summary => {
      const usage = summary.source_execution.usage_observation
      usage.invalid_tokens = 1
      usage.input_tokens = null
      usage.output_tokens = null
      usage.local_estimate_usd = null
      usage.tokens_complete = false
      usage.cost_complete = false
      usage.adapter_usage_presence = representativeR071Aggregate
    }), context)
    expect(nullTotals.execution.available).toBe(true)
    expect(nullTotals.usage).toEqual({ available: true, values: {
      responses: 1, invalid_tokens: 1, priced: 1, unpriced: 0, invalid_costs: 0,
      input_tokens: null, output_tokens: null, local_estimate_usd: null,
      tokens_complete: false, cost_complete: false,
      token_overflow: false, cost_overflow: false, counter_overflow: false,
    } })
  })
})
