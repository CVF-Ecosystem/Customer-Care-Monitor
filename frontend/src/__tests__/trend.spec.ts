import { describe, it, expect } from 'vitest'
import { dayMonthLabel, formatRelative, vnDateKey } from '../utils/format'
import { qualityTrendByDay } from '../utils/trend'
import vi from '../i18n/vi'
import en from '../i18n/en'

// CCMAI-UX-001a: UX-01, UX-02 label, UX-03 wording, UX-05 day grouping.
describe('vnDateKey', () => {
  it('uses the Vietnam calendar day on both sides of UTC midnight', () => {
    expect(vnDateKey('2026-09-15T16:59:00Z')).toBe('2026-09-15') // 23:59 in Vietnam
    expect(vnDateKey('2026-09-15T17:00:00Z')).toBe('2026-09-16') // 00:00 next day in Vietnam
    expect(vnDateKey('2026-09-16T00:30:00+07:00')).toBe('2026-09-16')
    expect(vnDateKey('2026-09-15T23:30:00+07:00')).toBe('2026-09-15')
    expect(vnDateKey('not a date')).toBeNull()
    expect(dayMonthLabel('2026-09-05')).toBe('05/09')
  })
})

describe('qualityTrendByDay', () => {
  const r = (conv: string, date: string, type: string, severity = 'PASS') => ({
    conversation_id: conv,
    conversation_date: date,
    created_at: date,
    result_type: type,
    severity,
  })

  it('gives one point per Vietnam day, sorted, with no duplicate labels', () => {
    const points = qualityTrendByDay([
      // 15/09 16:30Z is 23:30 on 15/09 in Vietnam
      r('a', '2026-09-15T16:30:00Z', 'conversation_evaluation'),
      // 16/09 00:30 VN is 15/09 17:30Z — the old code keyed this by the UTC date 15/09
      r('b', '2026-09-15T17:30:00Z', 'conversation_evaluation'),
      r('b', '2026-09-15T17:30:00Z', 'qc_violation', 'CAN_CAI_THIEN'),
      r('c', '2026-09-16T10:00:00+07:00', 'conversation_evaluation'),
      r('d', '2026-09-14T09:00:00+07:00', 'conversation_evaluation', 'SKIP'),
    ])
    expect(points.map((p) => p.day)).toEqual(['2026-09-14', '2026-09-15', '2026-09-16'])
    const labels = points.map((p) => p.label)
    expect(new Set(labels).size).toBe(labels.length)
    expect(points.find((p) => p.day === '2026-09-15')).toMatchObject({ passed: 1, failed: 0 })
    expect(points.find((p) => p.day === '2026-09-16')).toMatchObject({ passed: 1, failed: 1 })
    expect(points.find((p) => p.day === '2026-09-14')).toMatchObject({ passed: 0, failed: 0 }) // SKIP excluded
  })

  it('counts a conversation once even with several results', () => {
    const points = qualityTrendByDay([
      r('a', '2026-09-20T09:00:00+07:00', 'conversation_evaluation'),
      r('a', '2026-09-20T09:00:00+07:00', 'qc_violation', 'NGHIEM_TRONG'),
      r('a', '2026-09-20T09:00:00+07:00', 'qc_violation', 'CAN_CAI_THIEN'),
    ])
    expect(points).toEqual([{ day: '2026-09-20', label: '20/09', passed: 0, failed: 1 }])
  })
})

describe('dashboard and job detail wording', () => {
  it('labels the dashboard card for what the API counts, and tags as tags', () => {
    expect(vi.dash_results_total).toBe('Kết quả đánh giá')
    expect(vi.dash_results_total_hint).toContain('nhãn')
    expect(en.dash_results_total).toBeTruthy()
    expect(vi.tags_count_label).toBe('nhãn')
    expect(en.tags_count_label).toBe('tags')
  })

  it('never shows negative relative time for a future timestamp (UX-01)', () => {
    const now = new Date('2026-09-28T10:00:00+07:00')
    expect(formatRelative('2026-09-28T13:00:00+07:00', 'vi', now)).toBe('vừa xong')
    expect(formatRelative('2026-09-28T08:00:00+07:00', 'vi', now)).toBe('2 giờ trước')
  })
})
