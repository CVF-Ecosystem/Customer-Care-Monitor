// CCMAI-RUNTIME-026 (F04): Vietnam business-day calendar for date controls.
//
// All arithmetic is done on "YYYY-MM-DD" calendar keys (never on local-time Date objects), so the
// result is the same whatever the browser's time zone is. "Today" is the current instant in
// Asia/Ho_Chi_Minh (UTC+07:00, no daylight saving). Date-only values are carried as strings and
// are never parsed as midnight instants and re-serialized through the browser zone.

import { vnDateKey } from './format'

export const BUSINESS_TIME_ZONE = 'Asia/Ho_Chi_Minh'

export type DatePreset = 'today' | '7days' | '28days' | 'week' | 'month' | 'quarter' | 'year'

export interface DateRange {
  from: string
  to: string
}

interface Ymd {
  y: number
  m: number
  d: number
}

function parseKey(key: string): Ymd {
  const [y, m, d] = key.split('-').map(Number)
  return { y, m, d }
}

function pad(n: number) {
  return String(n).padStart(2, '0')
}

function toKey(y: number, m: number, d: number) {
  return `${String(y).padStart(4, '0')}-${pad(m)}-${pad(d)}`
}

// Calendar day in Vietnam for an instant (the existing helper, reused).
export function vnToday(now: Date = new Date()): string {
  const key = vnDateKey(now)
  if (key === null) throw new Error('invalid instant')
  return key
}

// Shifts a calendar key by whole days using UTC calendar arithmetic only.
export function addDays(key: string, days: number): string {
  const { y, m, d } = parseKey(key)
  const t = new Date(Date.UTC(y, m - 1, d + days))
  return toKey(t.getUTCFullYear(), t.getUTCMonth() + 1, t.getUTCDate())
}

// ISO weekday of a key: Monday = 1 ... Sunday = 7.
export function isoWeekday(key: string): number {
  const { y, m, d } = parseKey(key)
  const wd = new Date(Date.UTC(y, m - 1, d)).getUTCDay()
  return wd === 0 ? 7 : wd
}

export function startOfWeek(key: string): string {
  return addDays(key, -(isoWeekday(key) - 1))
}

export function startOfMonth(key: string): string {
  const { y, m } = parseKey(key)
  return toKey(y, m, 1)
}

export function startOfQuarter(key: string): string {
  const { y, m } = parseKey(key)
  return toKey(y, Math.floor((m - 1) / 3) * 3 + 1, 1)
}

export function startOfYear(key: string): string {
  const { y } = parseKey(key)
  return toKey(y, 1, 1)
}

// The last `n` calendar dates including today: exactly n dates, from = today - (n - 1).
export function lastDays(n: number, now: Date = new Date()): DateRange {
  const to = vnToday(now)
  return { from: addDays(to, -(n - 1)), to }
}

// A preset as an inclusive date pair, evaluated at one captured instant. Today means the same
// date on both bounds; 7days and 28days are exactly seven / twenty-eight dates including today;
// week begins Monday; month, quarter and year begin on the first Vietnam date and end today.
export function presetRange(preset: DatePreset, now: Date = new Date()): DateRange {
  const to = vnToday(now)
  switch (preset) {
    case 'today':
      return { from: to, to }
    case '7days':
      return lastDays(7, now)
    case '28days':
      return lastDays(28, now)
    case 'week':
      return { from: startOfWeek(to), to }
    case 'month':
      return { from: startOfMonth(to), to }
    case 'quarter':
      return { from: startOfQuarter(to), to }
    case 'year':
      return { from: startOfYear(to), to }
  }
}
