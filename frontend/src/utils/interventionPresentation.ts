import type { InterventionDetail, InterventionProposal } from '../api/intervention'

export function currentInterventionProposal(detail: InterventionDetail | null): InterventionProposal | null {
  if (!detail || !Array.isArray(detail.proposals)) return null
  const current = detail.case
  const top = detail.active_proposal_id
  const nested = current.active_proposal_id
  if (top !== undefined && nested !== undefined && top !== nested) return null
  const pointer = top !== undefined ? top : nested
  if (!current.run_id || !current.snapshot_id || !current.snapshot_digest) return null
  const matches = detail.proposals.filter(proposal =>
    proposal && typeof proposal.id === 'string' && !!proposal.id &&
    Number.isInteger(proposal.revision) && proposal.revision > 0 &&
    proposal.run_id === current.run_id && proposal.snapshot_id === current.snapshot_id &&
    proposal.snapshot_digest === current.snapshot_digest &&
    (pointer == null || proposal.id === pointer))
  if (pointer != null && (typeof pointer !== 'string' || !pointer)) return null
  if (pointer == null && current.active_execution_id != null) return null
  return matches.length === 1 ? matches[0] : null
}

export type InterventionHistoryField =
  | 'disposition'
  | 'work_state'
  | 'human_assessment'
  | 'human_explanation'
  | 'human_next_action'
  | 'assignee_id'
  | 'data_owner_id'
  | 'due_at'
  | 'outcome_text'

export interface InterventionHistoryChange {
  field: string
  before: string
  after: string
}

export interface InterventionHistoryLabels {
  fields: Record<InterventionHistoryField, string>
  notSet: string
  state: (value: string) => string
  disposition: (value: string) => string
  assessment: (value: string) => string
  person: (value: string) => string
  date: (value: string) => string
}

export interface InterventionHistoryChanges {
  valid: boolean
  changes: InterventionHistoryChange[]
}

const STATE_KEYS: Record<string, string> = {
  OPEN: 'iv_state_open',
  ASSIGNED: 'iv_state_assigned',
  WAITING_ON_DATA: 'iv_state_waiting_on_data',
  RESOLVED: 'iv_state_resolved',
  REJECTED: 'iv_state_rejected',
}

const ASSESSMENT_KEYS: Record<string, string> = {
  CONFIRM: 'iv_assessment_confirm',
  CORRECT: 'iv_assessment_correct',
  REJECT: 'iv_assessment_reject',
}

const SOURCE_KEYS: Record<string, string> = {
  LOCAL_MATCH_CURRENTNESS_UNVERIFIED: 'iv_source_local_match_currentness_unverified',
  BOUND_CURRENTNESS_UNVERIFIED: 'iv_source_local_match_currentness_unverified',
  CHANGED: 'iv_source_changed',
  INCOMPLETE: 'iv_source_incomplete',
  UNAVAILABLE: 'iv_source_unavailable',
}

const ACTION_KEYS: Record<string, string> = {
  CONFIRM: 'iv_act_confirm',
  CORRECT: 'iv_act_correct',
  REJECT: 'iv_act_reject',
  ASSIGN: 'iv_act_assign',
  RESOLVE: 'iv_act_resolve',
  REOPEN: 'iv_act_reopen',
  ASSIGN_DATA: 'iv_act_assign_data',
  ESCALATE_DATA: 'iv_act_escalate_data',
  OBSERVE_DATA: 'iv_act_observe_data',
  REQUEST_RERUN: 'iv_act_request_rerun',
  EXECUTE_RERUN: 'iv_act_execute_rerun',
  ACKNOWLEDGE_SOURCE_CHANGE: 'iv_act_acknowledge_source_change',
}

export const interventionEventReasonKey = (action: string, reason: string): string | null =>
  action === 'EXECUTE_RERUN' && reason === 'approved execution' ? 'iv_reason_approved_execution' : null

const DISPOSITION_KEYS: Record<string, string> = {
  PENDING_REVIEW: 'iv_disposition_pending',
  NOT_CREATED: 'iv_disposition_not_created',
  WAIT_DATA: 'iv_disposition_waiting',
  HUMAN_CONFIRM: 'iv_disposition_confirmed',
  HUMAN_CORRECT: 'iv_disposition_corrected',
  HUMAN_REJECT: 'iv_disposition_rejected',
}

const PROPOSAL_KIND_KEYS: Record<string, string> = {
  COMPLAINT: 'iv_kind_complaint',
  PRAISE: 'iv_kind_praise',
  QUESTION: 'iv_kind_question',
  SPAM: 'iv_kind_spam',
  OTHER: 'iv_kind_other',
}

function safeKey(value: unknown, known: Record<string, string>, fallback: string): string {
  if (typeof value !== 'string') return fallback
  return known[value.trim().toUpperCase()] || fallback
}

export const interventionStateKey = (value: unknown) => safeKey(value, STATE_KEYS, 'iv_state_other')
export const interventionAssessmentKey = (value: unknown) =>
  value === '' || value === null || value === undefined
    ? 'iv_pending'
    : safeKey(value, ASSESSMENT_KEYS, 'iv_assessment_other')
export const interventionSourceKey = (value: unknown) => safeKey(value, SOURCE_KEYS, 'iv_source_unavailable')
export const interventionActionKey = (value: unknown) => safeKey(value, ACTION_KEYS, 'iv_action_other')
export const interventionDispositionKey = (value: unknown) => safeKey(value, DISPOSITION_KEYS, 'iv_disposition_other')
export const interventionProposalKindKey = (value: unknown) => safeKey(value, PROPOSAL_KIND_KEYS, 'iv_kind_other')

const UUID = /^(?:[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}|[\da-f]{32})$/i

export type InterventionCaseReference = { kind: 'direct'; value: string } | { kind: 'short'; value: string }

export function interventionCaseReference(conversationId: string, caseId: string): InterventionCaseReference {
  const conversation = typeof conversationId === 'string' ? conversationId.trim() : ''
  if (conversation && !UUID.test(conversation)) return { kind: 'direct', value: conversation }

  const identifier = conversation || (typeof caseId === 'string' ? caseId.trim() : '')
  const short = identifier.replace(/[^\da-z]/gi, '').slice(0, 8).toUpperCase()
  return { kind: 'short', value: short }
}

type Projection = Record<string, unknown>

function parseProjection(value: string): Projection | null {
  try {
    const parsed: unknown = JSON.parse(value)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null
    return parsed as Projection
  } catch {
    return null
  }
}

function scalar(value: unknown): string | null {
  if (value === null || value === undefined) return null
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') return String(value)
  return null
}

function formatHistoryValue(field: InterventionHistoryField, raw: unknown, labels: InterventionHistoryLabels): string {
  const value = scalar(raw)
  if (value === null || value === '') return labels.notSet
  switch (field) {
    case 'work_state': return labels.state(value)
    case 'disposition': return labels.disposition(value)
    case 'human_assessment': return labels.assessment(value)
    case 'assignee_id':
    case 'data_owner_id': return labels.person(value)
    case 'due_at': return labels.date(value)
    default: return value
  }
}

const HISTORY_FIELDS: InterventionHistoryField[] = [
  'disposition',
  'work_state',
  'human_assessment',
  'human_explanation',
  'human_next_action',
  'assignee_id',
  'data_owner_id',
  'due_at',
  'outcome_text',
]

export function interventionHistoryChanges(before: string, after: string, labels: InterventionHistoryLabels): InterventionHistoryChanges {
  const oldProjection = parseProjection(before)
  const newProjection = parseProjection(after)
  if (!oldProjection || !newProjection) return { valid: false, changes: [] }

  for (const field of HISTORY_FIELDS) {
    for (const projection of [oldProjection, newProjection]) {
      const value = projection[field]
      if (value !== null && value !== undefined && scalar(value) === null) {
        return { valid: false, changes: [] }
      }
    }
  }

  const changes: InterventionHistoryChange[] = []
  for (const field of HISTORY_FIELDS) {
    const oldValue = scalar(oldProjection[field])
    const newValue = scalar(newProjection[field])
    if (oldValue === newValue) continue
    changes.push({
      field: labels.fields[field],
      before: formatHistoryValue(field, oldProjection[field], labels),
      after: formatHistoryValue(field, newProjection[field], labels),
    })
  }
  return { valid: true, changes }
}
