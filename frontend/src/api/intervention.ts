// The human write path makes one request only. A 401 is returned to the view;
// the shared Axios refresh interceptor must never silently replay an action.
export interface InterventionCase {
  id: string; tenant_id: string; job_id: string; conversation_id: string
  version: number; snapshot_id: string | null; snapshot_digest: string
  disposition: string; work_state: string; reason_code: string; next_action: string
  human_assessment: string; human_explanation: string; human_next_action: string
  data_owner_id: string | null; assignee_id: string | null; due_at: string | null
  outcome_text: string; created_at: string; updated_at: string
}
export interface InterventionProposal {
  id: string; revision: number; run_id: string; snapshot_id: string
  snapshot_digest: string; kind: string; explanation: string; next_action: string
  evidence: string; provider: string; model: string; prompt_version: string; created_at: string
}
export interface InterventionEvent {
  id: string; case_version: number; actor_id: string; action: string; reason: string
  before: string; after: string; source_status: string; created_at: string
}
export interface InterventionDetail {
  case: InterventionCase; proposals: InterventionProposal[]
  human_events: InterventionEvent[]; source_status: string
}
export interface InterventionAction {
  action: string; expected_version: number; expected_snapshot_id: string
  expected_snapshot_digest: string; expected_proposal_id: string; expected_revision: number
  reason: string; explanation?: string; next_action?: string
  assignee_id?: string; due_at?: string; outcome?: string
}
export class InterventionHttpError extends Error {
  readonly status: number
  readonly code: string
  constructor(status: number, code: string) { super(code); this.status = status; this.code = code }
}
async function boundedJSON(response: Response, maxBytes: number): Promise<Record<string, unknown>> {
  const declared = Number(response.headers.get('content-length'))
  if (Number.isFinite(declared) && declared > maxBytes) throw new InterventionHttpError(0, 'response_too_large')
  if (!response.body) throw new InterventionHttpError(0, 'response_unavailable')
  const reader = response.body.getReader(), chunks: Uint8Array[] = []
  let size = 0
  try {
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      size += value.byteLength
      if (size > maxBytes) throw new InterventionHttpError(0, 'response_too_large')
      chunks.push(value)
    }
  } finally { await reader.cancel().catch(() => undefined) }
  const bytes = new Uint8Array(size)
  let offset = 0
  for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.byteLength }
  const parsed: unknown = JSON.parse(new TextDecoder().decode(bytes))
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new InterventionHttpError(0, 'response_invalid')
  return parsed as Record<string, unknown>
}
async function once<T>(path: string, token: string, signal: AbortSignal, body?: unknown): Promise<T> {
  const write = body !== undefined, request = new AbortController()
  const abort = () => request.abort()
  signal.addEventListener('abort', abort, { once:true })
  if (signal.aborted) request.abort()
  const timer = setTimeout(abort, write ? 20_000 : 15_000)
  try {
    const response = await fetch(`/api/v1${path}`, {
      method: write ? 'POST' : 'GET',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: write ? JSON.stringify(body) : undefined,
      signal: request.signal,
      credentials: 'same-origin',
      redirect: 'error',
      cache: 'no-store',
    })
    if (response.redirected) throw new InterventionHttpError(0, 'redirect_rejected')
    const data = await boundedJSON(response, write ? 64 * 1024 : 1024 * 1024)
    if (!response.ok) throw new InterventionHttpError(response.status, typeof data.error === 'string' ? data.error : 'request_failed')
    return data as unknown as T
  } catch (error) {
    if (signal.aborted) throw error
    if (write && !(error instanceof InterventionHttpError && error.status >= 400)) throw new InterventionHttpError(0, 'completion_unknown_reload')
    throw error
  } finally {
    clearTimeout(timer)
    signal.removeEventListener('abort', abort)
  }
}
const root = (tenant: string) => `/tenants/${encodeURIComponent(tenant)}`
export function listInterventionCases(tenant: string, token: string, signal: AbortSignal, state: string, cursor = '') {
  const q = new URLSearchParams({ limit: '20' })
  if (state) q.set('state', state)
  if (cursor) q.set('cursor', cursor)
  return once<{items: InterventionCase[]; next_cursor: string; has_more: boolean; total: number; order: string}>(`${root(tenant)}/intervention-cases?${q}`, token, signal)
}
export function getInterventionCase(tenant: string, id: string, token: string, signal: AbortSignal) {
  return once<InterventionDetail>(`${root(tenant)}/intervention-cases/${encodeURIComponent(id)}`, token, signal)
}
export function listInterventionAssignees(tenant: string, token: string, signal: AbortSignal) {
  return once<{items: {user_id: string; name: string}[]}>(`${root(tenant)}/intervention-assignees`, token, signal)
}
export function applyInterventionAction(tenant: string, id: string, token: string, signal: AbortSignal, action: InterventionAction) {
  return once<{case: InterventionCase}>(`${root(tenant)}/intervention-cases/${encodeURIComponent(id)}/actions`, token, signal, action)
}
