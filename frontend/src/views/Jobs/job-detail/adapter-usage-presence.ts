// Saved adapter reports only; these values do not verify provider wire fields or billing.
import { projectRunObservation, type ObservationSection, type RunObservationContext } from './run-observation'

type RecordValue = Record<string, unknown>

type SideProjection = {
  known: number
  absent: number
  nullCount: number
  invalid: number
  unavailable: number
  unobserved: number
  sumOverflow: boolean
  total: number | null
}

const unavailable = (): ObservationSection => ({ available: false })
const object = (value: unknown): value is RecordValue => value !== null && typeof value === 'object' && !Array.isArray(value)
const has = (value: RecordValue, key: string): boolean => Object.prototype.hasOwnProperty.call(value, key)
const safeCount = (value: unknown): value is number => typeof value === 'number' && Number.isSafeInteger(value) && value >= 0

function exactKeys(value: RecordValue, required: string[]): boolean {
  return required.every(key => has(value, key)) && Object.keys(value).every(key => required.includes(key))
}

function safeSum(values: number[]): number | null {
  let total = 0
  for (const value of values) {
    const next = total + value
    if (!Number.isSafeInteger(next)) return null
    total = next
  }
  return total
}

function sideProjection(
  value: unknown,
  prefix: 'input' | 'output',
  responses: number,
  callsBegun: number,
  counterOverflow: boolean,
): { side: SideProjection; values: Record<string, number | boolean | null> } | null {
  const keys = ['known', 'absent', 'null', 'invalid', 'unavailable', 'unobserved', 'sum_overflow', 'total']
  if (!object(value) || !exactKeys(value, keys)) return null

  const numeric = ['known', 'absent', 'null', 'invalid', 'unavailable', 'unobserved']
  if (numeric.some(key => !safeCount(value[key])) || typeof value.sum_overflow !== 'boolean' ||
      (value.total !== null && !safeCount(value.total))) return null

  const counts = numeric.map(key => value[key] as number)
  const statusSum = safeSum(counts)
  if (statusSum === null || statusSum !== responses) return null

  const side: SideProjection = {
    known: value.known as number,
    absent: value.absent as number,
    nullCount: value.null as number,
    invalid: value.invalid as number,
    unavailable: value.unavailable as number,
    unobserved: value.unobserved as number,
    sumOverflow: value.sum_overflow,
    total: value.total as number | null,
  }
  const knownOnly = side.known === responses && side.absent === 0 && side.nullCount === 0 &&
    side.invalid === 0 && side.unavailable === 0 && side.unobserved === 0
  const totalExpected = callsBegun > 0 && responses === callsBegun && !counterOverflow && knownOnly && !side.sumOverflow
  if (totalExpected !== (side.total !== null)) return null

  const values: Record<string, number | boolean | null> = {
    [`${prefix}_known`]: side.known,
    [`${prefix}_absent`]: side.absent,
    [`${prefix}_null`]: side.nullCount,
    [`${prefix}_invalid`]: side.invalid,
    [`${prefix}_unavailable`]: side.unavailable,
    [`${prefix}_unobserved`]: side.unobserved,
    [`${prefix}_sum_overflow`]: side.sumOverflow,
    [`${prefix}_total`]: side.total,
  }
  return { side, values }
}

/**
 * Projects only the named R071 aggregate. The legacy saved-run projection remains
 * authoritative for parent binding and for the existing execution/usage contracts.
 */
export function projectAdapterUsagePresence(summary: unknown, context: RunObservationContext): ObservationSection | null {
  if (typeof summary !== 'string' || summary.length > 1024 * 1024) return null

  let parsed: unknown
  try { parsed = JSON.parse(summary) } catch { return null }
  if (!object(parsed) || !object(parsed.source_execution) || !object(parsed.source_execution.usage_observation) ||
      !has(parsed.source_execution.usage_observation, 'adapter_usage_presence')) return null

  const legacy = projectRunObservation(summary, context)
  if (!legacy.execution.available || !legacy.usage.available) return unavailable()

  const aggregate = parsed.source_execution.usage_observation.adapter_usage_presence
  const keys = ['version', 'basis', 'responses', 'input', 'output', 'counter_overflow', 'complete']
  if (!object(aggregate) || !exactKeys(aggregate, keys) ||
      aggregate.version !== 'ccmai.adapter-usage-presence.v1' || aggregate.basis !== 'ADAPTER_REPORTED_TOKEN_COUNTS' ||
      !safeCount(aggregate.responses) || typeof aggregate.counter_overflow !== 'boolean' ||
      typeof aggregate.complete !== 'boolean') return unavailable()

  const executionValues = legacy.execution.values
  const callsBegun = executionValues.calls_begun
  const responseReturned = executionValues.response_returned
  if (!safeCount(callsBegun) || !safeCount(responseReturned) || aggregate.responses > responseReturned) return unavailable()

  const input = sideProjection(aggregate.input, 'input', aggregate.responses, callsBegun, aggregate.counter_overflow)
  const output = sideProjection(aggregate.output, 'output', aggregate.responses, callsBegun, aggregate.counter_overflow)
  if (!input || !output) return unavailable()

  const complete = input.side.total !== null && output.side.total !== null
  if (aggregate.complete !== complete) return unavailable()

  // Fixed flattened allowlist. No extension object, source string, or dynamic key escapes.
  return { available: true, values: {
    adapter_responses: aggregate.responses,
    ...input.values,
    ...output.values,
    adapter_counter_overflow: aggregate.counter_overflow,
    complete,
  } }
}
