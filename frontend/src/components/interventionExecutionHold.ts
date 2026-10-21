export type ExecutionHoldState = 'IN_FLIGHT' | 'UNCERTAIN' | 'OBSERVED'
export interface ExecutionHold { state: ExecutionHoldState; at: string }

// Identity is in the key; values contain no token, grant, request body or secret.
export function executionHoldKey(tenant: string, caseId: string, requestId: string, actor: string) {
  return `ccmai:execution:${tenant.toLowerCase()}:${caseId.toLowerCase()}:${requestId.toLowerCase()}:${actor.toLowerCase()}`
}

export function readExecutionHold(storage: Storage, key: string): ExecutionHold | null {
  const raw = storage.getItem(key)
  if (raw === null) return null
  try {
    const value: unknown = JSON.parse(raw)
    if (value && typeof value === 'object' && ['IN_FLIGHT','UNCERTAIN','OBSERVED'].includes((value as ExecutionHold).state) && typeof (value as ExecutionHold).at === 'string') return value as ExecutionHold
  } catch { /* A malformed marker still blocks another POST. */ }
  return {state:'UNCERTAIN',at:''}
}

export function beginExecutionHold(storage: Storage, key: string): boolean {
  try {
    if (storage.getItem(key) !== null) return false
    const value = JSON.stringify({state:'IN_FLIGHT',at:new Date().toISOString()} satisfies ExecutionHold)
    storage.setItem(key,value)
    return storage.getItem(key) === value
  } catch { return false }
}

export function updateExecutionHold(storage: Storage, key: string, state: Exclude<ExecutionHoldState,'IN_FLIGHT'>): void {
  try {
    const prior = readExecutionHold(storage,key)
    if (prior) storage.setItem(key,JSON.stringify({state,at:prior.at} satisfies ExecutionHold))
  } catch { /* A failed update leaves the original hold in place. */ }
}
