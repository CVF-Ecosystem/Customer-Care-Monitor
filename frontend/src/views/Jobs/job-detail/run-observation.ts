// Presentation of persisted observations only; no authority or currentness decision.
export interface RunObservationContext { tenantId: string; jobId: string; runId: string; runJobId: string }
type RecordValue = Record<string, unknown>
export type ObservationValue = number | string | boolean | null
export type ObservationSection = { available: false } | { available: true; values: Record<string, ObservationValue> }
export interface RunObservation { preparation: ObservationSection; execution: ObservationSection; rules: ObservationSection; usage: ObservationSection }

const unavailable = (): ObservationSection => ({ available: false })
const object = (v: unknown): v is RecordValue => v !== null && typeof v === 'object' && !Array.isArray(v)
const has = (v: RecordValue, key: string): boolean => Object.prototype.hasOwnProperty.call(v, key)
const count = (v: unknown): v is number => typeof v === 'number' && Number.isSafeInteger(v) && v >= 0
const sum = (values: number[]): number => values.reduce((total, value) => total + value, 0)
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const digest = /^[0-9a-f]{64}$/i
const modes = ['ordinary', 'test_run', 'conditional', 'unanalyzed', 'since_last']
const preparationOutcomes = ['EMPTY_SOURCE', 'UNCHANGED_VERIFIED', 'PREPARED_FOR_INFERENCE', 'SNAPSHOT_ERROR', 'SOURCE_VERSION_ERROR', 'PREPARATION_INTERRUPTED']
const preparationStopReasons = ['CONTEXT_CANCELLED', 'PREPARATION_INTERRUPTED', 'SELECTION_FAILED']
const selectionStatuses = ['NOT_ATTEMPTED', 'COMPLETE', 'FAILED']
const referenceStates = ['NONE', 'LEGACY_NO_SNAPSHOT', 'VERIFIED', 'LOOKUP_FAILED', 'PROVENANCE_UNVERIFIABLE']
const coverageValues = ['complete', 'partial', 'empty']
const coverageReasons = ['NO_MESSAGES', 'HISTORY_WINDOWED', 'UNSUPPORTED_CONTENT_TYPE', 'ATTACHMENT_NOT_REPRESENTED', 'ATTACHMENT_JSON_INVALID']
const usageWrites = ['NOT_ATTEMPTED', 'WRITE_SUCCEEDED', 'WRITE_FAILED', 'WRITE_OUTCOME_UNKNOWN']
const parsingOutcomes = ['NOT_ATTEMPTED', 'ACCEPTED', 'REJECTED', 'NOT_SEPARATELY_OBSERVABLE']
const invocationOutcomes = ['IN_FLIGHT', 'ERROR_RETURNED', 'RESPONSE_RETURNED', 'INTERRUPTED']
const executionStopReasons = ['NONE', 'CANCELLED', 'OWNERSHIP_NOT_PUBLISHED', 'PANIC', 'PROVIDER_SETUP_FAILED', 'PREPARATION_STOPPED', 'EXECUTION_ERROR']
const jobTypes = ['qc_analysis', 'classification']

function enumString(value: unknown, allowed: string[]): value is string {
  return typeof value === 'string' && allowed.includes(value)
}

function exactKeys(value: RecordValue, required: string[], optional: string[] = []): boolean {
  const allowed = [...required, ...optional]
  return required.every(key => has(value, key)) && Object.keys(value).every(key => allowed.includes(key))
}

function counters(value: unknown, keys: string[], sparse = false): Record<string, number> | null {
  if (!object(value) || Object.keys(value).some(key => !keys.includes(key))) return null
  const result: Record<string, number> = {}
  for (const key of keys) {
    if (!has(value, key) && sparse) {
      result[key] = 0
      continue
    }
    const amount = value[key]
    if (!count(amount)) return null
    result[key] = amount
  }
  return Number.isSafeInteger(sum(Object.values(result))) ? result : null
}

function bound(value: RecordValue, context: RunObservationContext): boolean {
  // BINDING_GUARD: never trust another saved run's receipt.
  return value.metadata_incomplete === false && value.tenant_id === context.tenantId && value.job_id === context.jobId && value.run_id === context.runId
}

function header(value: unknown, version: string, scope: string, context: RunObservationContext, required: string[], optional: string[] = []): value is RecordValue {
  return object(value) && exactKeys(value, required, optional) && value.version === version && value.scope === scope && bound(value, context)
}

const preparationEntryRequired = ['outcome', 'reference', 'metadata_incomplete']
const preparationEntryOptional = ['conversation_id', 'snapshot_schema', 'snapshot_digest', 'coverage', 'coverage_reasons']

function validPreparationEntry(value: unknown): value is RecordValue {
  if (!object(value) || !exactKeys(value, preparationEntryRequired, preparationEntryOptional) ||
      !enumString(value.outcome, preparationOutcomes) || typeof value.metadata_incomplete !== 'boolean') return false
  if (has(value, 'conversation_id') && (typeof value.conversation_id !== 'string' || !uuid.test(value.conversation_id))) return false
  if (!value.metadata_incomplete && !has(value, 'conversation_id')) return false
  if (has(value, 'snapshot_schema') && value.snapshot_schema !== 'ccma.snapshot.v1') return false
  if (has(value, 'snapshot_digest') && (typeof value.snapshot_digest !== 'string' || !digest.test(value.snapshot_digest))) return false
  if (has(value, 'coverage') && !enumString(value.coverage, coverageValues)) return false
  if (has(value, 'coverage_reasons') && (!Array.isArray(value.coverage_reasons) || !value.coverage_reasons.every(reason => enumString(reason, coverageReasons)))) return false

  const reference = value.reference
  const referenceIDs = ['evaluation_id', 'run_id', 'snapshot_id']
  if (!object(reference) || !exactKeys(reference, ['state'], referenceIDs) || !enumString(reference.state, referenceStates)) return false
  const presentIDs = referenceIDs.filter(key => has(reference, key))
  if (presentIDs.some(key => typeof reference[key] !== 'string' || !uuid.test(reference[key] as string))) return false
  if (reference.state !== 'VERIFIED' && presentIDs.length > 0) return false
  if (reference.state === 'VERIFIED' && !value.metadata_incomplete && presentIDs.length !== referenceIDs.length) return false
  return true
}

function preparation(value: unknown, context: RunObservationContext): ObservationSection {
  const required = ['version', 'scope', 'tenant_id', 'job_id', 'run_id', 'mode', 'metadata_incomplete', 'selection_status', 'selected', 'visited', 'unvisited', 'counts', 'entries', 'omitted_entries', 'entries_complete', 'scan_complete']
  if (!header(value, 'ccmai.source-preparation.v1', 'preparation_only', context, required, ['stop_reason']) ||
      !enumString(value.mode, modes) || !enumString(value.selection_status, selectionStatuses)) return unavailable()

  const totals = counters(value.counts, preparationOutcomes, true)
  const stopReason = has(value, 'stop_reason') ? value.stop_reason : ''
  if (!totals || !count(value.visited) || !count(value.omitted_entries) || !Array.isArray(value.entries) || value.entries.length > 200 ||
      !value.entries.every(validPreparationEntry) || value.entries.length + value.omitted_entries !== value.visited ||
      value.entries_complete !== (value.omitted_entries === 0) || typeof value.scan_complete !== 'boolean' ||
      (stopReason !== '' && !enumString(stopReason, preparationStopReasons)) || typeof stopReason !== 'string') return unavailable()

  const visited = value.visited as number
  const completed = sum(Object.values(totals))
  if (completed !== visited) return unavailable()
  const retainedOutcomes = countersFromPreparationEntries(value.entries as RecordValue[])
  if (!retainedOutcomes || Object.keys(retainedOutcomes).some(key => retainedOutcomes[key] > totals[key])) return unavailable()
  if (value.entries_complete && Object.keys(totals).some(key => retainedOutcomes[key] !== totals[key])) return unavailable()

  if (value.selection_status === 'COMPLETE') {
    if (!count(value.selected) || !count(value.unvisited) || sum([visited, value.unvisited as number]) !== value.selected) return unavailable()
  } else if (value.selected !== null || value.unvisited !== null || visited !== 0) return unavailable()

  return { available: true, values: {
    selected: value.selected as number | null,
    visited,
    unvisited: value.unvisited as number | null,
    scan_complete: value.scan_complete as boolean,
    omitted_entries: value.omitted_entries as number,
    entries_complete: value.entries_complete as boolean,
    ...totals,
  } }
}

function countersFromPreparationEntries(entries: RecordValue[]): Record<string, number> | null {
  const result = Object.fromEntries(preparationOutcomes.map(outcome => [outcome, 0])) as Record<string, number>
  for (const entry of entries) {
    const outcome = entry.outcome as string
    if (!Object.prototype.hasOwnProperty.call(result, outcome)) return null
    result[outcome]++
  }
  return result
}

const executionCallRequired = ['sequence', 'method', 'item_count', 'member_ids', 'omitted_members', 'members_complete', 'metadata_incomplete', 'invocation_outcome', 'usage_write_outcome', 'parsing_outcome', 'items_saved', 'items_save_failed', 'items_not_published', 'items_pending']

interface ExecutionCallCounts {
  calls: number
  responses: number
  errors: number
  interrupted: number
  inFlight: number
  items: number
  saved: number
  saveFailed: number
  notPublished: number
  pending: number
  omittedMembers: number
  usageWrites: Record<string, number>
  parsing: Record<string, number>
}

function validExecutionCall(value: unknown, sequence: number): value is RecordValue {
  if (!object(value) || !exactKeys(value, executionCallRequired) || value.sequence !== sequence ||
      !enumString(value.method, ['SINGLE', 'BATCH']) || !Array.isArray(value.member_ids) ||
      !value.member_ids.every(id => typeof id === 'string' && uuid.test(id)) ||
      typeof value.members_complete !== 'boolean' || typeof value.metadata_incomplete !== 'boolean' ||
      !enumString(value.invocation_outcome, invocationOutcomes) || !enumString(value.usage_write_outcome, usageWrites) ||
      !enumString(value.parsing_outcome, parsingOutcomes)) return false

  const numeric = ['item_count', 'omitted_members', 'items_saved', 'items_save_failed', 'items_not_published', 'items_pending']
  if (numeric.some(key => !count(value[key])) || value.member_ids.length > (value.item_count as number) ||
      value.omitted_members !== (value.item_count as number) - value.member_ids.length ||
      value.members_complete !== (value.omitted_members === 0) ||
      sum([value.items_saved as number, value.items_save_failed as number, value.items_not_published as number, value.items_pending as number]) !== value.item_count) return false
  if (value.method === 'SINGLE' && value.parsing_outcome !== 'NOT_SEPARATELY_OBSERVABLE') return false
  return true
}

function retainedExecutionCalls(calls: RecordValue[]): ExecutionCallCounts | null {
  const result: ExecutionCallCounts = {
    calls: calls.length, responses: 0, errors: 0, interrupted: 0, inFlight: 0, items: 0,
    saved: 0, saveFailed: 0, notPublished: 0, pending: 0, omittedMembers: 0,
    usageWrites: Object.fromEntries(usageWrites.map(value => [value, 0])),
    parsing: Object.fromEntries(parsingOutcomes.map(value => [value, 0])),
  }
  for (const call of calls) {
    result.items += call.item_count as number
    result.saved += call.items_saved as number
    result.saveFailed += call.items_save_failed as number
    result.notPublished += call.items_not_published as number
    result.pending += call.items_pending as number
    result.omittedMembers += call.omitted_members as number
    switch (call.invocation_outcome) {
      case 'RESPONSE_RETURNED': result.responses++; break
      case 'ERROR_RETURNED': result.errors++; break
      case 'INTERRUPTED': result.interrupted++; break
      case 'IN_FLIGHT': result.inFlight++; break
    }
    result.usageWrites[call.usage_write_outcome as string]++
    result.parsing[call.parsing_outcome as string]++
  }
  return Object.values(result.usageWrites).every(count) && Object.values(result.parsing).every(count) ? result : null
}

function atMost(retained: Record<string, number>, aggregate: Record<string, number>): boolean {
  return Object.keys(retained).every(key => retained[key] <= aggregate[key])
}

function execution(value: unknown, context: RunObservationContext): ObservationSection {
  const required = ['version', 'scope', 'tenant_id', 'job_id', 'run_id', 'mode', 'metadata_incomplete', 'calls_begun', 'response_returned', 'error_returned', 'interrupted', 'in_flight', 'item_count', 'items_saved', 'items_save_failed', 'items_not_published', 'items_pending', 'usage_writes', 'parsing', 'calls', 'omitted_calls', 'omitted_members', 'entries_complete', 'members_complete', 'execution_complete', 'stop_reason']
  if (!header(value, 'ccmai.source-execution.v1', 'analyzer_provider_interface_only', context, required, ['usage_observation']) || !enumString(value.mode, modes)) return unavailable()

  const fields = ['calls_begun', 'response_returned', 'error_returned', 'interrupted', 'in_flight', 'item_count', 'items_saved', 'items_save_failed', 'items_not_published', 'items_pending', 'omitted_calls', 'omitted_members']
  if (fields.some(key => !count(value[key])) || typeof value.entries_complete !== 'boolean' || typeof value.members_complete !== 'boolean' ||
      typeof value.execution_complete !== 'boolean' || !enumString(value.stop_reason, executionStopReasons) ||
      !Array.isArray(value.calls) || value.calls.length > 200 || !value.calls.every((call, index) => validExecutionCall(call, index + 1))) return unavailable()

  const totals = Object.fromEntries(fields.map(key => [key, value[key] as number])) as Record<string, number>
  const writes = counters(value.usage_writes, usageWrites)
  const parses = counters(value.parsing, parsingOutcomes)
  const retained = retainedExecutionCalls(value.calls as RecordValue[])
  if (!writes || !parses || !retained ||
      sum([totals.response_returned, totals.error_returned, totals.interrupted, totals.in_flight]) !== totals.calls_begun ||
      sum([totals.items_saved, totals.items_save_failed, totals.items_not_published, totals.items_pending]) !== totals.item_count ||
      sum(Object.values(writes)) !== totals.calls_begun || sum(Object.values(parses)) !== totals.calls_begun ||
      value.calls.length + totals.omitted_calls !== totals.calls_begun ||
      totals.omitted_members !== totals.item_count - (value.calls as RecordValue[]).reduce((n, call) => n + (call.member_ids as unknown[]).length, 0) ||
      value.entries_complete !== (totals.omitted_calls === 0) || value.members_complete !== (totals.omitted_members === 0) ||
      retained.omittedMembers > totals.omitted_members ||
      !atMost(retained.usageWrites, writes) || !atMost(retained.parsing, parses)) return unavailable()

  const retainedInvocations: Record<string, number> = {
    response_returned: retained.responses,
    error_returned: retained.errors,
    interrupted: retained.interrupted,
    in_flight: retained.inFlight,
  }
  const retainedItems: Record<string, number> = {
    items_saved: retained.saved,
    items_save_failed: retained.saveFailed,
    items_not_published: retained.notPublished,
    items_pending: retained.pending,
  }
  if (!atMost(retainedInvocations, totals) || !atMost(retainedItems, totals) || retained.items > totals.item_count) return unavailable()
  if (value.entries_complete && (
    retained.items !== totals.item_count || retained.omittedMembers !== totals.omitted_members ||
    Object.keys(retainedInvocations).some(key => retainedInvocations[key] !== totals[key]) ||
    Object.keys(retainedItems).some(key => retainedItems[key] !== totals[key]) ||
    Object.keys(writes).some(key => retained.usageWrites[key] !== writes[key]) ||
    Object.keys(parses).some(key => retained.parsing[key] !== parses[key])
  )) return unavailable()

  return { available: true, values: {
    ...totals,
    ...writes,
    execution_complete: value.execution_complete,
    entries_complete: value.entries_complete,
    members_complete: value.members_complete,
    stop_reason: value.stop_reason,
  } }
}

function rules(value: unknown, context: RunObservationContext): ObservationSection {
  const required = ['version', 'scope', 'tenant_id', 'job_id', 'run_id', 'metadata_incomplete', 'rule_authority', 'policy_version_status', 'permission_status', 'wait_data_status', 'fingerprint_status']
  const optional = ['job_type', 'fingerprint']
  if (!header(value, 'ccmai.rule-observation.v1', 'analyzer_job_input_only', context, required, optional) ||
      value.rule_authority !== 'JOB_INPUT_OBSERVED' || value.policy_version_status !== 'NOT_AVAILABLE' ||
      value.permission_status !== 'NOT_OBSERVED_AT_ANALYZER' || value.wait_data_status !== 'NOT_IMPLEMENTED' ||
      !enumString(value.fingerprint_status, ['OBSERVED', 'METADATA_INVALID', 'UNSUPPORTED_JOB_TYPE'])) return unavailable()
  if (value.fingerprint_status !== 'OBSERVED') return unavailable()
  if (!enumString(value.job_type, jobTypes) || typeof value.fingerprint !== 'string' || !digest.test(value.fingerprint)) return unavailable()
  return { available: true, values: { fingerprint: value.fingerprint } }
}

function usage(value: unknown, observedExecution: ObservationSection): ObservationSection {
  const required = ['version', 'scope', 'token_basis', 'billing', 'price_revision', 'responses', 'invalid_tokens', 'priced', 'unpriced', 'invalid_costs', 'token_overflow', 'cost_overflow', 'counter_overflow', 'input_tokens', 'output_tokens', 'local_estimate_usd', 'tokens_complete', 'cost_complete']
  if (!observedExecution.available || !object(value) || !exactKeys(value, required, ['adapter_usage_presence']) ||
      value.version !== 'ccmai.usage-observation.v1' || value.scope !== 'successful_interface_response_local_estimate' ||
      value.token_basis !== 'INTERFACE_VALUES_PRESENCE_UNAVAILABLE' || value.billing !== 'NOT_OBSERVED' || value.price_revision !== 'NOT_CAPTURED') return unavailable()

  const numericFields = ['responses', 'invalid_tokens', 'priced', 'unpriced', 'invalid_costs']
  const tokenOverflow = value.token_overflow
  const costOverflow = value.cost_overflow
  const counterOverflow = value.counter_overflow
  const tokensCompleteFlag = value.tokens_complete
  const costCompleteFlag = value.cost_complete
  if (numericFields.some(key => !count(value[key])) ||
      typeof tokenOverflow !== 'boolean' || typeof costOverflow !== 'boolean' || typeof counterOverflow !== 'boolean' ||
      typeof tokensCompleteFlag !== 'boolean' || typeof costCompleteFlag !== 'boolean') return unavailable()
  const numbers = Object.fromEntries(numericFields.map(key => [key, value[key] as number])) as Record<string, number>
  const callsBegun = observedExecution.values.calls_begun as number
  const responsesReturned = observedExecution.values.response_returned as number
  if (numbers.responses > responsesReturned || sum([numbers.priced, numbers.unpriced]) !== numbers.responses ||
      numbers.invalid_tokens > numbers.responses || numbers.invalid_costs > numbers.priced) return unavailable()

  const tokensComplete = callsBegun > 0 && numbers.responses === callsBegun && numbers.invalid_tokens === 0 && !tokenOverflow && !counterOverflow
  const costComplete = tokensComplete && numbers.unpriced === 0 && numbers.priced === numbers.responses && numbers.invalid_costs === 0 && !costOverflow
  if (tokensCompleteFlag !== tokensComplete || costCompleteFlag !== costComplete ||
      (tokensComplete ? !count(value.input_tokens) || !count(value.output_tokens) : value.input_tokens !== null || value.output_tokens !== null) ||
      (costComplete ? typeof value.local_estimate_usd !== 'number' || !Number.isFinite(value.local_estimate_usd) || value.local_estimate_usd < 0 : value.local_estimate_usd !== null)) return unavailable()

  // UNKNOWN_TOTAL_GUARD: null is unavailable, never a free or billed zero.
  return { available: true, values: {
    ...numbers,
    input_tokens: value.input_tokens as number | null,
    output_tokens: value.output_tokens as number | null,
    local_estimate_usd: value.local_estimate_usd as number | null,
    tokens_complete: tokensComplete,
    cost_complete: costComplete,
    token_overflow: tokenOverflow,
    cost_overflow: costOverflow,
    counter_overflow: counterOverflow,
  } }
}

export function projectRunObservation(summary: unknown, context: RunObservationContext): RunObservation {
  const result: RunObservation = { preparation: unavailable(), execution: unavailable(), rules: unavailable(), usage: unavailable() }
  if (typeof summary !== 'string' || summary.length > 1024 * 1024 || !object(context) ||
      ![context.tenantId, context.jobId, context.runId, context.runJobId].every(id => typeof id === 'string' && uuid.test(id)) ||
      context.runJobId !== context.jobId) return result
  let parsed: unknown
  try { parsed = JSON.parse(summary) } catch { return result }
  if (!object(parsed)) return result

  result.preparation = preparation(parsed.source_preparation, context)
  result.execution = execution(parsed.source_execution, context)
  result.rules = rules(parsed.rule_observation, context)
  result.usage = usage(object(parsed.source_execution) ? parsed.source_execution.usage_observation : undefined, result.execution)
  return result
}
