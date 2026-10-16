import { describe, expect, it } from 'vitest'
import {
  interventionActionKey,
  interventionAssessmentKey,
  interventionCaseReference,
  interventionDispositionKey,
  interventionHistoryChanges,
  interventionProposalKindKey,
  interventionSourceKey,
  interventionStateKey,
  type InterventionHistoryLabels,
} from '../utils/interventionPresentation'

const labels: InterventionHistoryLabels = {
  fields: {
    disposition: 'Review outcome',
    work_state: 'Work status',
    human_assessment: 'Reviewer assessment',
    human_explanation: 'Explanation',
    human_next_action: 'Next step',
    assignee_id: 'Assignee',
    data_owner_id: 'Data owner',
    due_at: 'Deadline',
    outcome_text: 'Outcome',
  },
  notSet: 'Not provided',
  state: value => value === 'RESOLVED' ? 'Resolved' : value === 'ASSIGNED' ? 'Assigned' : 'Other status',
  disposition: value => value === 'HUMAN_CONFIRM' ? 'Confirmed' : 'Other outcome',
  assessment: value => value === 'CONFIRM' ? 'Confirmed' : 'Other assessment',
  person: value => value === 'user-1' ? 'Alex' : 'Assigned team member',
  date: value => value.startsWith('2026-10-20') ? 'October 20' : 'Other date',
}

describe('intervention presentation helpers', () => {
  it('maps known values and never returns an unrecognized wire value', () => {
    expect(interventionStateKey('OPEN')).toBe('iv_state_open')
    expect(interventionStateKey('INTERNAL_NEW_STATE')).toBe('iv_state_other')
    expect(interventionAssessmentKey('INTERNAL_ASSESSMENT')).toBe('iv_assessment_other')
    expect(interventionSourceKey('INTERNAL_SOURCE')).toBe('iv_source_unavailable')
    expect(interventionActionKey('INTERNAL_ACTION')).toBe('iv_action_other')
    expect(interventionDispositionKey('WAIT_DATA')).toBe('iv_disposition_waiting')
    expect(interventionDispositionKey('HUMAN_CONFIRM')).toBe('iv_disposition_confirmed')
    expect(interventionDispositionKey('HUMAN_CORRECT')).toBe('iv_disposition_corrected')
    expect(interventionDispositionKey('HUMAN_REJECT')).toBe('iv_disposition_rejected')
    expect(interventionDispositionKey('INTERNAL_DISPOSITION')).toBe('iv_disposition_other')
    expect(interventionProposalKindKey('complaint')).toBe('iv_kind_complaint')
    expect(interventionProposalKindKey('INTERNAL_KIND')).toBe('iv_kind_other')
  })

  it('keeps readable case identifiers distinct and shortens UUID references', () => {
    expect(interventionCaseReference('CASE-A', 'case-1')).toEqual({ kind: 'direct', value: 'CASE-A' })
    expect(interventionCaseReference('CASE-B', 'case-2')).toEqual({ kind: 'direct', value: 'CASE-B' })
    expect(interventionCaseReference('12345678-abcd-4def-8abc-1234567890ab', 'case-3'))
      .toEqual({ kind: 'short', value: '12345678' })
  })

  it('renders only labeled business-field changes with the previous outcome intact', () => {
    const before = JSON.stringify({
      version: 4, disposition: 'HUMAN_CONFIRM', work_state: 'RESOLVED', human_assessment: 'CONFIRM',
      human_explanation: 'Reviewed', human_next_action: 'Follow up', assignee_id: 'user-1', data_owner_id: null,
      due_at: '2026-10-20T08:00:00Z', outcome_text: 'Replacement arranged with the customer',
    })
    const after = JSON.stringify({
      version: 5, disposition: 'HUMAN_CONFIRM', work_state: 'ASSIGNED', human_assessment: 'CONFIRM',
      human_explanation: 'Reviewed', human_next_action: 'Follow up', assignee_id: 'user-1', data_owner_id: null,
      due_at: '2026-10-20T08:00:00Z', outcome_text: '',
    })
    const result = interventionHistoryChanges(before, after, labels)
    expect(result.valid).toBe(true)
    expect(result.changes).toEqual(expect.arrayContaining([
      { field: 'Work status', before: 'Resolved', after: 'Assigned' },
      { field: 'Outcome', before: 'Replacement arranged with the customer', after: 'Not provided' },
    ]))
    expect(result.changes.some(change => change.before.includes('HUMAN_'))).toBe(false)
    expect(result.changes.some(change => change.before.includes('user-1'))).toBe(false)
  })

  it('rejects malformed history and unsupported values without exposing raw JSON', () => {
    expect(interventionHistoryChanges('{not-json', '{}', labels)).toEqual({ valid: false, changes: [] })
    expect(interventionHistoryChanges('[]', '{}', labels)).toEqual({ valid: false, changes: [] })
    expect(interventionHistoryChanges('{"work_state":{"internal":"value"}}', '{}', labels))
      .toEqual({ valid: false, changes: [] })
  })
})
