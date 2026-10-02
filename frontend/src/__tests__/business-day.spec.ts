// CCMAI-RUNTIME-026 (F04): the Vietnam business-day calendar for date controls. Pure helper
// tests; the same named boundary instants and dates drive the backend handler fixtures
// (backend/api/handlers/business_dates_test.go) and the mounted view specs.
import { afterEach, describe, expect, it } from 'vitest'
import { addDays, isoWeekday, lastDays, presetRange, startOfMonth, startOfQuarter, startOfWeek, startOfYear, vnToday, type DatePreset } from '../utils/businessDay'

// The frontend has no Node typings; the test runner's process.env.TZ is reached through globalThis.
const env = (globalThis as unknown as { process: { env: Record<string, string | undefined> } }).process.env
const originalTz = env.TZ
afterEach(() => {
  if (originalTz === undefined) delete env.TZ
  else env.TZ = originalTz
})

// Browser zones to run every case in: UTC, Vietnam itself, a western zone behind UTC by 4-5 h and
// an eastern zone ahead of Vietnam. Results must not depend on which one is active.
const BROWSER_ZONES = ['UTC', 'Asia/Ho_Chi_Minh', 'America/New_York', 'Pacific/Auckland']

// Named instants (UTC) -> Vietnam date, shared with the backend boundary fixtures.
const NAMED_INSTANTS: Array<[string, string, string]> = [
  ['one millisecond before VN midnight (23:59:59.999 VN)', '2026-10-01T16:59:59.999Z', '2026-10-01'],
  ['VN midnight (00:00:00.000 VN)', '2026-10-01T17:00:00.000Z', '2026-10-02'],
  ['00:30 VN', '2026-10-01T17:30:00.000Z', '2026-10-02'],
  ['06:59 VN', '2026-10-01T23:59:00.000Z', '2026-10-02'],
  ['07:00 VN', '2026-10-02T00:00:00.000Z', '2026-10-02'],
  ['23:59:59.999 VN', '2026-10-02T16:59:59.999Z', '2026-10-02'],
  ['next VN midnight', '2026-10-02T17:00:00.000Z', '2026-10-03'],
]

describe('Vietnam business-day calendar (F04)', () => {
  for (const zone of BROWSER_ZONES) {
    it(`vnToday follows Asia/Ho_Chi_Minh when the browser zone is ${zone}`, () => {
      env.TZ = zone
      for (const [name, iso, want] of NAMED_INSTANTS) {
        expect(vnToday(new Date(iso)), `${name} in ${zone}`).toBe(want)
      }
    })
  }

  it('calendar arithmetic is pure: month, year and leap-day rollovers', () => {
    expect(addDays('2026-10-02', -27)).toBe('2026-09-05')
    expect(addDays('2026-03-01', -1)).toBe('2026-02-28')
    expect(addDays('2024-03-01', -1)).toBe('2024-02-29')
    expect(addDays('2024-02-29', 1)).toBe('2024-03-01')
    expect(addDays('2026-12-31', 1)).toBe('2027-01-01')
    expect(addDays('2027-01-01', -1)).toBe('2026-12-31')
    expect(addDays('2100-02-28', 1)).toBe('2100-03-01') // century is not a leap year
  })

  it('ISO weekday and week start: Monday first, Sunday last', () => {
    expect(isoWeekday('2026-10-02')).toBe(5) // Friday
    expect(isoWeekday('2026-10-04')).toBe(7) // Sunday
    expect(isoWeekday('2026-10-05')).toBe(1) // Monday
    expect(startOfWeek('2026-10-02')).toBe('2026-09-28')
    expect(startOfWeek('2026-10-04')).toBe('2026-09-28') // Sunday belongs to the week that began Monday
    expect(startOfWeek('2026-10-05')).toBe('2026-10-05') // Monday starts its own week
    expect(startOfWeek('2026-01-01')).toBe('2025-12-29') // across the year boundary
  })

  it('period starts are the first Vietnam date of the month, quarter and year', () => {
    expect(startOfMonth('2026-10-02')).toBe('2026-10-01')
    expect(startOfMonth('2024-02-29')).toBe('2024-02-01')
    expect(startOfQuarter('2026-10-02')).toBe('2026-10-01')
    expect(startOfQuarter('2026-09-30')).toBe('2026-07-01')
    expect(startOfQuarter('2026-03-31')).toBe('2026-01-01')
    expect(startOfQuarter('2026-06-01')).toBe('2026-04-01')
    expect(startOfYear('2026-10-02')).toBe('2026-01-01')
  })

  it('7days and 28days are exactly seven and twenty-eight dates including today', () => {
    const now = new Date('2026-10-01T17:30:00.000Z') // 00:30 VN on 2026-10-02
    expect(lastDays(7, now)).toEqual({ from: '2026-09-26', to: '2026-10-02' })
    expect(lastDays(28, now)).toEqual({ from: '2026-09-05', to: '2026-10-02' })
    const count = (r: { from: string; to: string }) => {
      let n = 1
      for (let d = r.from; d !== r.to; d = addDays(d, 1)) n++
      return n
    }
    expect(count(presetRange('7days', now))).toBe(7)
    expect(count(presetRange('28days', now))).toBe(28)
    expect(presetRange('today', now)).toEqual({ from: '2026-10-02', to: '2026-10-02' })
  })

  it('every preset is identical in every browser zone, at early Vietnam hours and edge days', () => {
    const presets: DatePreset[] = ['today', '7days', '28days', 'week', 'month', 'quarter', 'year']
    // Early VN hours of a month start, a quarter start, a year start and a leap day.
    const clocks = [
      '2026-10-01T17:00:00.000Z', // 2026-10-02 00:00 VN
      '2026-09-30T17:05:00.000Z', // 2026-10-01 00:05 VN: first day of month and quarter
      '2026-12-31T17:30:00.000Z', // 2027-01-01 00:30 VN: first day of the year
      '2024-02-28T18:00:00.000Z', // 2024-02-29 01:00 VN: leap day
      '2026-10-01T16:59:59.999Z', // 2026-10-01 23:59:59.999 VN
    ]
    for (const iso of clocks) {
      const results: Record<string, string> = {}
      for (const zone of BROWSER_ZONES) {
        env.TZ = zone
        for (const p of presets) {
          const r = presetRange(p, new Date(iso))
          const key = `${p}`
          const val = `${r.from}..${r.to}`
          if (results[key] === undefined) results[key] = val
          expect(val, `${p} at ${iso} in ${zone}`).toBe(results[key])
        }
      }
    }
  })

  it('named presets at the early-hour boundary clock', () => {
    const now = new Date('2026-10-01T17:00:00.000Z') // 2026-10-02 00:00:00.000 VN
    expect(presetRange('today', now)).toEqual({ from: '2026-10-02', to: '2026-10-02' })
    expect(presetRange('7days', now)).toEqual({ from: '2026-09-26', to: '2026-10-02' })
    expect(presetRange('28days', now)).toEqual({ from: '2026-09-05', to: '2026-10-02' })
    expect(presetRange('week', now)).toEqual({ from: '2026-09-28', to: '2026-10-02' })
    expect(presetRange('month', now)).toEqual({ from: '2026-10-01', to: '2026-10-02' })
    expect(presetRange('quarter', now)).toEqual({ from: '2026-10-01', to: '2026-10-02' })
    expect(presetRange('year', now)).toEqual({ from: '2026-01-01', to: '2026-10-02' })
    // One millisecond earlier it is still the previous Vietnam day.
    const before = new Date('2026-10-01T16:59:59.999Z')
    expect(presetRange('today', before)).toEqual({ from: '2026-10-01', to: '2026-10-01' })
    expect(presetRange('month', before)).toEqual({ from: '2026-10-01', to: '2026-10-01' })
  })

  it('a browser in New York at 23:30 local on Oct 1 is already Oct 2 in Vietnam', () => {
    env.TZ = 'America/New_York'
    const now = new Date('2026-10-02T03:30:00.000Z') // 23:30 EDT on Oct 1, 10:30 VN on Oct 2
    expect(now.getDate()).toBe(1) // the browser-local calendar would say Oct 1
    expect(presetRange('today', now)).toEqual({ from: '2026-10-02', to: '2026-10-02' })
  })
})
