import { describe, expect, it } from 'vitest'
import { projectRunObservation, type RunObservationContext } from '../views/Jobs/job-detail/run-observation'

const ids = {
  tenant: '11111111-1111-4111-8111-111111111111',
  job: '22222222-2222-4222-8222-222222222222',
  run: '33333333-3333-4333-8333-333333333333',
  conversation: '44444444-4444-4444-8444-444444444444',
  member: '55555555-5555-4555-8555-555555555555',
  evaluation: '66666666-6666-4666-8666-666666666666',
}

const context: RunObservationContext = { tenantId: ids.tenant, jobId: ids.job, runId: ids.run, runJobId: ids.job }

function preparation(runId = ids.run) {
  return {
    version: 'ccmai.source-preparation.v1', scope: 'preparation_only', tenant_id: ids.tenant, job_id: ids.job, run_id: runId,
    mode: 'conditional', metadata_incomplete: false, selection_status: 'COMPLETE', selected: 1, visited: 1, unvisited: 0,
    counts: { PREPARED_FOR_INFERENCE: 1 },
    entries: [{
      conversation_id: ids.conversation, outcome: 'PREPARED_FOR_INFERENCE', snapshot_schema: 'ccma.snapshot.v1',
      snapshot_digest: 'a'.repeat(64), coverage: 'complete', reference: { state: 'NONE' }, metadata_incomplete: false,
    }],
    omitted_entries: 0, entries_complete: true, scan_complete: true,
  }
}

function execution(runId = ids.run) {
  return {
    version: 'ccmai.source-execution.v1', scope: 'analyzer_provider_interface_only', tenant_id: ids.tenant, job_id: ids.job,
    run_id: runId, mode: 'conditional', metadata_incomplete: false, calls_begun: 1, response_returned: 1,
    error_returned: 0, interrupted: 0, in_flight: 0, item_count: 1, items_saved: 1, items_save_failed: 0,
    items_not_published: 0, items_pending: 0,
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
    usage_observation: usageObservation(),
  }
}

function usageObservation(overrides: Record<string, unknown> = {}) {
  return {
    version: 'ccmai.usage-observation.v1', scope: 'successful_interface_response_local_estimate',
    token_basis: 'INTERFACE_VALUES_PRESENCE_UNAVAILABLE', billing: 'NOT_OBSERVED', price_revision: 'NOT_CAPTURED',
    responses: 1, invalid_tokens: 0, priced: 1, unpriced: 0, invalid_costs: 0,
    token_overflow: false, cost_overflow: false, counter_overflow: false,
    input_tokens: 0, output_tokens: 0, local_estimate_usd: 0, tokens_complete: true, cost_complete: true,
    ...overrides,
  }
}

function rules(runId = ids.run) {
  return {
    version: 'ccmai.rule-observation.v1', scope: 'analyzer_job_input_only', tenant_id: ids.tenant, job_id: ids.job,
    run_id: runId, job_type: 'qc_analysis', metadata_incomplete: false, rule_authority: 'JOB_INPUT_OBSERVED',
    policy_version_status: 'NOT_AVAILABLE', permission_status: 'NOT_OBSERVED_AT_ANALYZER', wait_data_status: 'NOT_IMPLEMENTED',
    fingerprint_status: 'OBSERVED', fingerprint: 'b'.repeat(64),
  }
}

function summary(overrides: Record<string, unknown> = {}): string {
  return JSON.stringify({ source_preparation: preparation(), source_execution: execution(), rule_observation: rules(), ...overrides })
}

function project(value: string, runContext = context) { return projectRunObservation(value, runContext) }

describe('saved-run observation projection', () => {
  it('accepts source-shaped conditional-mode receipts and preserves a known local zero', () => {
    const result = project(summary())
    expect(result.preparation.available).toBe(true)
    expect(result.execution.available).toBe(true)
    expect(result.rules.available).toBe(true)
    expect(result.usage.available).toBe(true)
    if (result.usage.available) expect(result.usage.values.local_estimate_usd).toBe(0)
  })

  it('keeps independent receipt sections visible when another section is absent or malformed', () => {
    const withoutRules = JSON.parse(summary()) as Record<string, unknown>
    delete withoutRules.rule_observation
    const noRules = project(JSON.stringify(withoutRules))
    expect(noRules.preparation.available).toBe(true)
    expect(noRules.execution.available).toBe(true)
    expect(noRules.rules.available).toBe(false)
    expect(noRules.usage.available).toBe(true)

    const badUsage = JSON.parse(summary()) as { source_execution: Record<string, unknown> }
    badUsage.source_execution.usage_observation = { version: 'future', scope: 'unknown' }
    const noUsage = project(JSON.stringify(badUsage))
    expect(noUsage.execution.available).toBe(true)
    expect(noUsage.usage.available).toBe(false)
    expect(noUsage.preparation.available).toBe(true)
    expect(noUsage.rules.available).toBe(true)

    const legacyExecution = JSON.parse(summary()) as { source_execution: Record<string, unknown> }
    delete legacyExecution.source_execution.usage_observation
    const legacy = project(JSON.stringify(legacyExecution))
    expect(legacy.execution.available).toBe(true)
    expect(legacy.usage.available).toBe(false)
  })

  it('rejects malformed, primitive, oversized, future-version and top-level-incomplete summaries', () => {
    expect(project('{')).toMatchObject({ preparation: { available: false }, execution: { available: false } })
    expect(project('null')).toMatchObject({ preparation: { available: false }, execution: { available: false } })
    expect(project('"text"')).toMatchObject({ preparation: { available: false }, rules: { available: false } })
    expect(project(' '.repeat(1024 * 1024 + 1))).toMatchObject({ preparation: { available: false }, usage: { available: false } })

    const future = JSON.parse(summary()) as { source_preparation: Record<string, unknown> }
    future.source_preparation.version = 'ccmai.source-preparation.v2'
    expect(project(JSON.stringify(future)).preparation.available).toBe(false)

    const incomplete = JSON.parse(summary()) as { source_execution: Record<string, unknown> }
    incomplete.source_execution.metadata_incomplete = true
    expect(project(JSON.stringify(incomplete)).execution.available).toBe(false)
  })

  it('rejects a saved receipt bound to another run', () => {
    const wrongRun = JSON.parse(summary()) as { source_preparation: Record<string, unknown>; source_execution: Record<string, unknown>; rule_observation: Record<string, unknown> }
    wrongRun.source_preparation.run_id = '77777777-7777-4777-8777-777777777777'
    wrongRun.source_execution.run_id = '77777777-7777-4777-8777-777777777777'
    wrongRun.rule_observation.run_id = '77777777-7777-4777-8777-777777777777'
    const mismatched = project(JSON.stringify(wrongRun))
    expect(mismatched.preparation.available).toBe(false)
    expect(mismatched.execution.available).toBe(false)
    expect(mismatched.rules.available).toBe(false)
  })

  it('accepts the exact tenant, job, run, and run.job_id context', () => {
    const mismatchedRunJob = project(summary(), { ...context, runJobId: '77777777-7777-4777-8777-777777777777' })
    expect(mismatchedRunJob.execution.available).toBe(false)
    expect(project(summary()).execution.available).toBe(true)
  })

  it('rejects receipts bound to another tenant or job', () => {
    const wrongTenant = JSON.parse(summary()) as { source_preparation: Record<string, unknown> }
    wrongTenant.source_preparation.tenant_id = '77777777-7777-4777-8777-777777777777'
    expect(project(JSON.stringify(wrongTenant)).preparation.available).toBe(false)

    const wrongJob = JSON.parse(summary()) as { source_execution: Record<string, unknown> }
    wrongJob.source_execution.job_id = '77777777-7777-4777-8777-777777777777'
    expect(project(JSON.stringify(wrongJob)).execution.available).toBe(false)

    const wrongRuleTenant = JSON.parse(summary()) as { rule_observation: Record<string, unknown> }
    wrongRuleTenant.rule_observation.tenant_id = '77777777-7777-4777-8777-777777777777'
    expect(project(JSON.stringify(wrongRuleTenant)).rules.available).toBe(false)
  })

  it('requires exact string enums instead of accepting array coercion or unknown values', () => {
    const badMode = JSON.parse(summary()) as { source_preparation: Record<string, unknown> }
    badMode.source_preparation.mode = ['conditional']
    expect(project(JSON.stringify(badMode)).preparation.available).toBe(false)

    const badSelection = JSON.parse(summary()) as { source_preparation: Record<string, unknown> }
    badSelection.source_preparation.selection_status = ['COMPLETE']
    expect(project(JSON.stringify(badSelection)).preparation.available).toBe(false)

    const badStop = JSON.parse(summary()) as { source_preparation: Record<string, unknown> }
    badStop.source_preparation.stop_reason = ['CONTEXT_CANCELLED']
    expect(project(JSON.stringify(badStop)).preparation.available).toBe(false)

    const badJobType = JSON.parse(summary()) as { rule_observation: Record<string, unknown> }
    badJobType.rule_observation.job_type = ['qc_analysis']
    expect(project(JSON.stringify(badJobType)).rules.available).toBe(false)
  })

  it('keeps unsupported or unavailable fingerprints unknown without exposing a fingerprint', () => {
    const unsupported = JSON.parse(summary()) as { rule_observation: Record<string, unknown> }
    delete unsupported.rule_observation.job_type
    delete unsupported.rule_observation.fingerprint
    unsupported.rule_observation.fingerprint_status = 'UNSUPPORTED_JOB_TYPE'
    expect(project(JSON.stringify(unsupported)).rules.available).toBe(false)

    const missingHash = JSON.parse(summary()) as { rule_observation: Record<string, unknown> }
    delete missingHash.rule_observation.fingerprint
    expect(project(JSON.stringify(missingHash)).rules.available).toBe(false)
  })

  it('reconciles partial preparation entries without converting unknown fields to zero', () => {
    const partial = JSON.parse(summary()) as { source_preparation: Record<string, unknown> }
    partial.source_preparation.selected = 2
    partial.source_preparation.visited = 2
    partial.source_preparation.counts = { PREPARED_FOR_INFERENCE: 2 }
    partial.source_preparation.entries = [{
      outcome: 'PREPARED_FOR_INFERENCE', metadata_incomplete: true,
      snapshot_schema: 'ccma.snapshot.v1', snapshot_digest: 'a'.repeat(64), coverage: 'complete',
      reference: { state: 'VERIFIED', evaluation_id: ids.evaluation },
    }]
    partial.source_preparation.omitted_entries = 1
    partial.source_preparation.entries_complete = false
    const projected = project(JSON.stringify(partial))
    expect(projected.preparation.available).toBe(true)
    if (projected.preparation.available) {
      expect(projected.preparation.values.omitted_entries).toBe(1)
      expect(projected.preparation.values.entries_complete).toBe(false)
      expect(projected.preparation.values.SNAPSHOT_ERROR).toBe(0)
    }

    const notAttempted = JSON.parse(summary()) as { source_preparation: Record<string, unknown> }
    Object.assign(notAttempted.source_preparation, {
      selection_status: 'NOT_ATTEMPTED',
      selected: null,
      visited: 0,
      unvisited: null,
      counts: {},
      entries: [],
      omitted_entries: 0,
      entries_complete: true,
      scan_complete: false,
    })
    const unknownSelection = project(JSON.stringify(notAttempted))
    expect(unknownSelection.preparation.available).toBe(true)
    if (unknownSelection.preparation.available) {
      expect(unknownSelection.preparation.values.selected).toBeNull()
      expect(unknownSelection.preparation.values.unvisited).toBeNull()
    }
  })

  it('rejects count gaps, malformed nested preparation records, and coerced child enums', () => {
    const countGap = JSON.parse(summary()) as { source_preparation: Record<string, unknown> }
    countGap.source_preparation.visited = 2
    countGap.source_preparation.selected = 2
    countGap.source_preparation.unvisited = 0
    countGap.source_preparation.omitted_entries = 1
    countGap.source_preparation.entries_complete = false
    expect(project(JSON.stringify(countGap)).preparation.available).toBe(false)

    const badChild = JSON.parse(summary()) as { source_preparation: { entries: Record<string, unknown>[] } }
    badChild.source_preparation.entries[0].metadata_incomplete = 'false'
    expect(project(JSON.stringify(badChild)).preparation.available).toBe(false)

    const coercedOutcome = JSON.parse(summary()) as { source_preparation: { entries: Record<string, unknown>[] } }
    coercedOutcome.source_preparation.entries[0].outcome = ['PREPARED_FOR_INFERENCE']
    expect(project(JSON.stringify(coercedOutcome)).preparation.available).toBe(false)
  })

  it('validates retained execution calls, bounded omissions, member completeness, and aggregate counters', () => {
    const partial = JSON.parse(summary()) as { source_execution: Record<string, unknown> }
    const ex = partial.source_execution
    ex.calls_begun = 2
    ex.response_returned = 1
    ex.error_returned = 1
    ex.item_count = 2
    ex.items_not_published = 1
    ex.usage_writes = { NOT_ATTEMPTED: 1, WRITE_SUCCEEDED: 1, WRITE_FAILED: 0, WRITE_OUTCOME_UNKNOWN: 0 }
    ex.parsing = { NOT_ATTEMPTED: 0, ACCEPTED: 0, REJECTED: 0, NOT_SEPARATELY_OBSERVABLE: 2 }
    ex.calls = [{
      sequence: 1, method: 'SINGLE', item_count: 1, member_ids: [], omitted_members: 1,
      members_complete: false, metadata_incomplete: true, invocation_outcome: 'RESPONSE_RETURNED',
      usage_write_outcome: 'WRITE_SUCCEEDED', parsing_outcome: 'NOT_SEPARATELY_OBSERVABLE',
      items_saved: 1, items_save_failed: 0, items_not_published: 0, items_pending: 0,
    }]
    ex.omitted_calls = 1
    ex.omitted_members = 2
    ex.entries_complete = false
    ex.members_complete = false
    ex.execution_complete = false
    ex.stop_reason = 'EXECUTION_ERROR'
    ex.usage_observation = usageObservation({
      responses: 1, priced: 0, unpriced: 1, input_tokens: null, output_tokens: null,
      local_estimate_usd: null, tokens_complete: false, cost_complete: false,
    })
    const projected = project(JSON.stringify(partial))
    expect(projected.execution.available).toBe(true)
    expect(projected.usage.available).toBe(true)
    if (projected.execution.available) {
      expect(projected.execution.values.omitted_calls).toBe(1)
      expect(projected.execution.values.omitted_members).toBe(2)
    }

    const malformed = JSON.parse(summary()) as { source_execution: Record<string, unknown> }
    malformed.source_execution.calls = [null]
    expect(project(JSON.stringify(malformed)).execution.available).toBe(false)
    expect(project(JSON.stringify(malformed)).usage.available).toBe(false)
  })

  it('rejects unsafe and negative counts, overflow, completeness contradictions, malformed usage, and bad call enums', () => {
    const unsafe = JSON.parse(summary()) as { source_execution: Record<string, unknown> }
    unsafe.source_execution.calls_begun = Number.MAX_SAFE_INTEGER + 1
    expect(project(JSON.stringify(unsafe)).execution.available).toBe(false)

    const negative = JSON.parse(summary()) as { source_execution: Record<string, unknown> }
    negative.source_execution.calls_begun = -1
    expect(project(JSON.stringify(negative)).execution.available).toBe(false)

    const nonFinite = project(summary().replace('"local_estimate_usd":0', '"local_estimate_usd":1e309'))
    expect(nonFinite.execution.available).toBe(true)
    expect(nonFinite.usage.available).toBe(false)

    const negativeTokens = JSON.parse(summary()) as { source_execution: { usage_observation: Record<string, unknown> } }
    negativeTokens.source_execution.usage_observation.input_tokens = -1
    const negativeTokenResult = project(JSON.stringify(negativeTokens))
    expect(negativeTokenResult.execution.available).toBe(true)
    expect(negativeTokenResult.usage.available).toBe(false)

    const overflowContradiction = JSON.parse(summary()) as { source_execution: { usage_observation: Record<string, unknown> } }
    overflowContradiction.source_execution.usage_observation.token_overflow = true
    const contradictory = project(JSON.stringify(overflowContradiction))
    expect(contradictory.execution.available).toBe(true)
    expect(contradictory.usage.available).toBe(false)

    const badUsage = JSON.parse(summary()) as { source_execution: Record<string, unknown> }
    badUsage.source_execution.usage_observation = usageObservation({ local_estimate_usd: '0' })
    const malformedUsage = project(JSON.stringify(badUsage))
    expect(malformedUsage.execution.available).toBe(true)
    expect(malformedUsage.usage.available).toBe(false)

    const badCall = JSON.parse(summary()) as { source_execution: { calls: Record<string, unknown>[] } }
    badCall.source_execution.calls[0].invocation_outcome = ['RESPONSE_RETURNED']
    expect(project(JSON.stringify(badCall)).execution.available).toBe(false)
  })
})
