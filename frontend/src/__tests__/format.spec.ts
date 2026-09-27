import { describe, it, expect } from 'vitest'
import { formatCurrency, formatDate, formatDateTime, formatNumber, formatRelative } from '../utils/format'

// CCMAI-UX-000: shared formatting rules (UX-01, UX-13, UX-15).
describe('format helpers', () => {
  const d = new Date(2026, 8, 5, 7, 4) // 05/09/2026 07:04 local time

  it('formats Vietnamese dates as dd/mm/yyyy and 24-hour time', () => {
    expect(formatDate(d, 'vi')).toBe('05/09/2026')
    expect(formatDateTime(d, 'vi')).toBe('05/09/2026 07:04')
    expect(formatDateTime(new Date(2026, 8, 5, 19, 30), 'vi')).toBe('05/09/2026 19:30')
  })

  it('renders invalid or empty input as a dash', () => {
    for (const v of [null, undefined, '', 'not a date']) {
      expect(formatDate(v as never)).toBe('—')
      expect(formatDateTime(v as never)).toBe('—')
      expect(formatRelative(v as never)).toBe('—')
    }
    expect(formatNumber(null)).toBe('—')
    expect(formatNumber(Number.NaN)).toBe('—')
    expect(formatCurrency(undefined, 'VND')).toBe('—')
  })

  it('never shows negative relative time and switches units', () => {
    const now = new Date('2026-09-28T10:00:00Z')
    const at = (mins: number) => new Date(now.getTime() - mins * 60000)
    expect(formatRelative(at(-180), 'vi', now)).toBe('vừa xong')
    expect(formatRelative(at(-1), 'en', now)).toBe('just now')
    expect(formatRelative(at(0), 'vi', now)).toBe('vừa xong')
    expect(formatRelative(at(5), 'vi', now)).toBe('5 phút trước')
    expect(formatRelative(at(59), 'vi', now)).toBe('59 phút trước')
    expect(formatRelative(at(60), 'vi', now)).toBe('1 giờ trước')
    expect(formatRelative(at(125), 'en', now)).toBe('2 h ago')
    expect(formatRelative(at(60 * 24), 'vi', now)).toBe('1 ngày trước')
    expect(formatRelative(at(60 * 24), 'en', now)).toBe('1 day ago')
    expect(formatRelative(at(60 * 24 * 3), 'en', now)).toBe('3 days ago')
  })

  it('formats numbers and currency per locale', () => {
    expect(formatNumber(1234567, 'vi')).toBe('1.234.567')
    expect(formatNumber(1234567, 'en')).toBe('1,234,567')
    expect(formatNumber(0, 'vi')).toBe('0')
    expect(formatCurrency(125000, 'VND', 'vi')).toBe('125.000 ₫')
    expect(formatCurrency(1.5, 'USD', 'en')).toBe('$1.50')
    expect(formatCurrency(1.5, 'USD', 'vi')).toBe('1,50 US$')
  })
})
