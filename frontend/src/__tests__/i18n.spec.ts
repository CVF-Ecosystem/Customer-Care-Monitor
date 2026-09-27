import { describe, it, expect } from 'vitest'
import vi from '../i18n/vi'
import en from '../i18n/en'
import { SOURCE_INTEGRITY_LABEL_KEY, SOURCE_INTEGRITY_ORDER, distinctSourceIntegrity } from '../stores/jobs'

describe('i18n completeness', () => {
  const viKeys = Object.keys(vi).sort()
  const enKeys = Object.keys(en).sort()

  it('should have the same number of keys in vi and en', () => {
    expect(viKeys.length).toBe(enKeys.length)
  })

  it('all vi keys should exist in en', () => {
    const missingInEn = viKeys.filter((key) => !enKeys.includes(key))
    expect(missingInEn).toEqual([])
  })

  it('all en keys should exist in vi', () => {
    const missingInVi = enKeys.filter((key) => !viKeys.includes(key))
    expect(missingInVi).toEqual([])
  })

  it('no empty values in vi', () => {
    const emptyVi = viKeys.filter((key) => !(vi as Record<string, string>)[key])
    expect(emptyVi).toEqual([])
  })

  it('no empty values in en', () => {
    const emptyEn = enKeys.filter((key) => !(en as Record<string, string>)[key])
    expect(emptyEn).toEqual([])
  })
})

// CCMAI-RUNTIME-006: Job Detail source-integrity badges reuse the aggregate
// Results labels and must never let a mixed group hide a changed result.
describe('job source-integrity labels and grouping', () => {
  it('every status has a non-empty label in both languages, plus the column header', () => {
    const keys = [...Object.values(SOURCE_INTEGRITY_LABEL_KEY), 'job_source_col', 'results_source_note']
    for (const key of keys) {
      expect((vi as Record<string, string>)[key], key).toBeTruthy()
      expect((en as Record<string, string>)[key], key).toBeTruthy()
    }
    expect(Object.keys(SOURCE_INTEGRITY_LABEL_KEY).sort()).toEqual([...SOURCE_INTEGRITY_ORDER].sort())
  })

  it('a mixed group lists every distinct status with changed first', () => {
    expect(
      distinctSourceIntegrity([
        { source_integrity_status: 'bound_currentness_unverified' },
        { source_integrity_status: 'legacy_unverified' },
        { source_integrity_status: 'bound_currentness_unverified' },
        { source_integrity_status: 'changed_since_analysis' },
      ]),
    ).toEqual(['changed_since_analysis', 'legacy_unverified', 'bound_currentness_unverified'])
  })

  it('missing or unknown statuses surface as unavailable, never disappear', () => {
    expect(distinctSourceIntegrity([{}, { source_integrity_status: 'fresh' }])).toEqual(['verification_unavailable'])
    expect(distinctSourceIntegrity([{ source_integrity_status: 'bound_currentness_unverified' }])).toEqual([
      'bound_currentness_unverified',
    ])
  })
})
