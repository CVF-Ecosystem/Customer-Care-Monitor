// The human write path makes one request only. A 401 is returned to the view;
// the shared Axios refresh interceptor must never silently replay an action.
export interface InterventionCase {
  id: string; tenant_id: string; job_id: string; conversation_id: string
  version: number; run_id: string; snapshot_id: string | null; snapshot_digest: string
  active_execution_id?: string | null; active_proposal_id?: string | null
  reviewed_data_revision_id?: string | null
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
  active_proposal_id?: string | null; human_events: InterventionEvent[]; source_status: string
  participants?: {user_id: string; name: string}[]
  current_source_digest?: string; reviewed_source_current?: boolean
}
export interface InterventionAction {
  action: string; expected_version: number; expected_snapshot_id: string
  expected_snapshot_digest: string; expected_proposal_id: string; expected_revision: number
  reason: string; explanation?: string; next_action?: string
  expected_current_digest?: string
  assignee_id?: string; due_at?: string; outcome?: string
}
export interface InterventionDataRevision {
  id: string; case_id: string; originating_case_version: number; case_version: number
  original_snapshot_id: string; original_snapshot_digest: string; digest: string
  coverage: string; coverage_reasons: string[]; message_count: number
  observation_status: string; reason: string; created_at: string
}
export interface InterventionRerunRequest {
  id: string; case_id: string; data_revision_id: string; data_revision_digest: string
  original_snapshot_id: string; original_snapshot_digest: string; reason: string
  status: string; execution: string; case_version: number; created_at: string
}
export interface InterventionWaitDataPage<T> {
  items: T[]; has_more: boolean; next_cursor: string
}
export type InterventionReadinessState = 'READY' | 'AWAITING_OPERATOR_AUTHORITY' | 'PROCESS_DISABLED' | 'STALE_OR_INELIGIBLE' | 'ALREADY_STARTED'
export interface InterventionExecutionReadiness {
  state: InterventionReadinessState; request_id: string; server_now: string
  expected_case_version?: number; expires_at?: string; grant_handle?: string
}
export type InterventionExecutionPhase = 'NOT_STARTED' | 'CLAIMED' | 'CALL_MAY_HAVE_STARTED' | 'RESPONSE_RECORDED' | 'PUBLISHED' | 'MANUAL_HOLD'
export interface InterventionExecutionStatus {
  request_id: string; request_intent: string; execution_present: boolean
  phase: InterventionExecutionPhase; observed_at: string
  terminal_code?: string; published_case_version?: number; proposal_available?: boolean
}
export interface InterventionExecuteInput {
  expected_case_version: number; grant_id: string
}
export interface CaptureInterventionDataRevisionInput {
  expected_version: number; expected_snapshot_id: string; expected_snapshot_digest: string; reason: string
}
export interface RequestInterventionRerunInput {
  expected_version: number; expected_snapshot_id: string; expected_snapshot_digest: string
  data_revision_id: string; data_revision_digest: string; reason: string; request_key: string
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
const waitDataCase = (tenant: string, id: string) => `${root(tenant)}/intervention-cases/${encodeURIComponent(id)}`
function waitDataListPath(tenant: string, id: string, collection: 'data-revisions' | 'rerun-requests', cursor = '') {
  const q = new URLSearchParams({ limit: '20' })
  if (cursor) q.set('cursor', cursor)
  return `${waitDataCase(tenant,id)}/${collection}?${q}`
}
export function listInterventionDataRevisions(tenant: string, id: string, token: string, signal: AbortSignal, cursor = '') {
  return once<InterventionWaitDataPage<InterventionDataRevision>>(waitDataListPath(tenant,id,'data-revisions',cursor), token, signal)
}
export function createInterventionDataRevision(tenant: string, id: string, token: string, signal: AbortSignal, input: CaptureInterventionDataRevisionInput) {
  return once<{revision: InterventionDataRevision; case_version: number}>(`${waitDataCase(tenant,id)}/data-revisions`, token, signal, input)
}
export function listInterventionRerunRequests(tenant: string, id: string, token: string, signal: AbortSignal, cursor = '') {
  return once<InterventionWaitDataPage<InterventionRerunRequest>>(waitDataListPath(tenant,id,'rerun-requests',cursor), token, signal)
}
export function createInterventionRerunRequest(tenant: string, id: string, token: string, signal: AbortSignal, input: RequestInterventionRerunInput) {
  return once<{request: InterventionRerunRequest; case_version: number; replayed: boolean}>(`${waitDataCase(tenant,id)}/rerun-requests`, token, signal, input)
}
const executionPath = (tenant: string, id: string, requestId: string) => `${waitDataCase(tenant,id)}/rerun-requests/${encodeURIComponent(requestId)}`
export function getInterventionExecutionReadiness(tenant: string, id: string, requestId: string, token: string, signal: AbortSignal) {
  return once<InterventionExecutionReadiness>(`${executionPath(tenant,id,requestId)}/execution-readiness`, token, signal)
}
export function getInterventionExecutionStatus(tenant: string, id: string, requestId: string, token: string, signal: AbortSignal) {
  return once<InterventionExecutionStatus>(`${executionPath(tenant,id,requestId)}/execution-status`, token, signal)
}
export function executeInterventionRerun(tenant: string, id: string, requestId: string, token: string, signal: AbortSignal, input: InterventionExecuteInput) {
  return once<{execution: {id: string; state: string; terminal_code: string; run_id: string; snapshot_id: string}; replayed: boolean}>(`${executionPath(tenant,id,requestId)}/execute`, token, signal, {expected_case_version: input.expected_case_version, grant_id: input.grant_id})
}
