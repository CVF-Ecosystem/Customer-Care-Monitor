import { describe, it, expect } from 'vitest'
import {
  confidencePercent,
  countSourceStatuses,
  needsReview,
  normalizeSourceStatus,
  syncKind,
  verdictFromSeverity,
} from '../utils/review'

// CCMAI-UX-000: display semantics must match reviewed R004/R005/R007 behavior.
describe('review semantics', () => {
  it('maps severity to verdict without ever defaulting to pass', () => {
    expect(verdictFromSeverity('PASS')).toBe('pass')
    expect(verdictFromSeverity('SKIP')).toBe('skip')
    expect(verdictFromSeverity('NGHIEM_TRONG')).toBe('fail')
    expect(verdictFromSeverity('CAN_CAI_THIEN')).toBe('fail')
    expect(verdictFromSeverity(undefined)).toBe('fail')
    expect(verdictFromSeverity('')).toBe('fail')
  })

  it('treats a missing or unknown source status as unavailable', () => {
    expect(normalizeSourceStatus(undefined)).toBe('verification_unavailable')
    expect(normalizeSourceStatus(null)).toBe('verification_unavailable')
    expect(normalizeSourceStatus('fresh')).toBe('verification_unavailable')
    expect(normalizeSourceStatus('legacy_unverified')).toBe('legacy_unverified')
  })

  it('counts statuses most-concerning first and keeps unknowns visible', () => {
    expect(
      countSourceStatuses(['legacy_unverified', 'bound_currentness_unverified', 'changed_since_analysis', 'x', undefined, 'legacy_unverified']),
    ).toEqual([
      { status: 'changed_since_analysis', count: 1 },
      { status: 'verification_unavailable', count: 2 },
      { status: 'legacy_unverified', count: 2 },
      { status: 'bound_currentness_unverified', count: 1 },
    ])
    expect(countSourceStatuses([])).toEqual([])
  })

  it('flags failed, changed or unverifiable results as needing review', () => {
    expect(needsReview({ severity: 'CAN_CAI_THIEN', source_integrity_status: 'bound_currentness_unverified' })).toBe(true)
    expect(needsReview({ severity: 'PASS', source_integrity_status: 'changed_since_analysis' })).toBe(true)
    expect(needsReview({ severity: 'PASS', source_integrity_status: 'verification_unavailable' })).toBe(true)
    expect(needsReview({ severity: 'PASS' })).toBe(true) // unknown source counts as unavailable
    expect(needsReview({ severity: 'PASS', source_integrity_status: 'bound_currentness_unverified' })).toBe(false)
    expect(needsReview({ severity: 'SKIP', source_integrity_status: 'legacy_unverified' })).toBe(false)
  })

  it('never reads an unknown or started sync as success', () => {
    expect(syncKind('')).toBe('never')
    expect(syncKind(null)).toBe('never')
    expect(syncKind('syncing')).toBe('syncing')
    expect(syncKind('partial')).toBe('partial')
    expect(syncKind('error')).toBe('error')
    expect(syncKind('done')).toBe('unknown')
  })

  it('shows confidence only for model-reported uncalibrated values in [0, 1]', () => {
    expect(confidencePercent(null, 'unavailable')).toBeNull()
    expect(confidencePercent(1, 'unavailable')).toBeNull()
    expect(confidencePercent(1, undefined)).toBeNull()
    expect(confidencePercent(null, 'model_reported_uncalibrated')).toBeNull()
    expect(confidencePercent(1.2, 'model_reported_uncalibrated')).toBeNull()
    expect(confidencePercent(Number.NaN, 'model_reported_uncalibrated')).toBeNull()
    expect(confidencePercent(0.724, 'model_reported_uncalibrated')).toBe(72)
    expect(confidencePercent(0, 'model_reported_uncalibrated')).toBe(0)
  })
})
