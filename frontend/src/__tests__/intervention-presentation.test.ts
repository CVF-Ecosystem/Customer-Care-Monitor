import { describe, expect, it } from 'vitest'
import enMessages from '../i18n/en'
import viMessages from '../i18n/vi'
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
  it('explains incomplete channel data without claiming that records match', () => {
    expect(viMessages.iv_source_local_match_currentness_unverified).not.toMatch(/khớp/i)
    expect(viMessages.iv_source_local_match_currentness_unverified).toContain('Chưa xác nhận')
    expect(viMessages.iv_local_only).not.toMatch(/khớp/i)
    expect(viMessages.iv_local_only).toContain('Chưa xác nhận')
    expect(enMessages.iv_source_local_match_currentness_unverified).not.toMatch(/matches/i)
    expect(enMessages.iv_source_local_match_currentness_unverified).toContain('not fully verified')
    expect(enMessages.iv_local_only).not.toMatch(/matches/i)
    expect(enMessages.iv_local_only).toContain('not confirmed')
  })

  it('uses plain language for common AI and assistant-connection controls', () => {
    for (const messages of [viMessages, enMessages]) {
      expect(messages.ai_provider).not.toMatch(/provider/i)
      expect(messages.api_key).not.toMatch(/api/i)
      expect(messages.test_api_key).not.toMatch(/api/i)
      expect(messages.nav_mcp).not.toMatch(/mcp/i)
      expect(messages.mcp_connections).not.toMatch(/mcp/i)
      expect(messages.mc_sub).not.toMatch(/mcp/i)
      expect(messages.st_intro_ai).not.toMatch(/provider|api key/i)
      expect(messages.st_need_key).not.toMatch(/api key/i)
      expect(messages.job_ai_service_missing_title).not.toMatch(/provider/i)
      expect(messages.job_ai_service_missing_text).not.toMatch(/provider|api key/i)
    }
    expect(viMessages.nav_mcp).toContain('trợ lý AI')
    expect(viMessages.mcp_connections).toContain('trợ lý AI')
    expect(enMessages.nav_mcp).toContain('assistant')
    expect(enMessages.mcp_connections).toContain('assistant')
    expect(viMessages.ai_model).toContain('Mô hình')
    expect(viMessages.refresh_model_list).toContain('mô hình')
    expect(viMessages.st_intro_ai).toContain('mô hình')
    expect(viMessages.st_models_updated).toContain('mô hình')
    expect(viMessages.st_key_saved_note).toContain('khóa truy cập')
  })

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
