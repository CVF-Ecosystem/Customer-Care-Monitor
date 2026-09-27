// CCMAI-UX-000: one place for date, time, number and currency display.
// Vietnamese: dd/mm/yyyy and 24-hour time. Invalid input always renders "—".

export type UiLocale = 'vi' | 'en'

const EMPTY = '—'

function toDate(value: string | number | Date | null | undefined): Date | null {
  if (value === null || value === undefined || value === '') return null
  const d = value instanceof Date ? value : new Date(value)
  return Number.isNaN(d.getTime()) ? null : d
}

function pad(n: number) {
  return String(n).padStart(2, '0')
}

export function formatDate(value: string | number | Date | null | undefined, locale: UiLocale = 'vi'): string {
  const d = toDate(value)
  if (!d) return EMPTY
  if (locale === 'vi') return `${pad(d.getDate())}/${pad(d.getMonth() + 1)}/${d.getFullYear()}`
  return d.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })
}

export function formatDateTime(value: string | number | Date | null | undefined, locale: UiLocale = 'vi'): string {
  const d = toDate(value)
  if (!d) return EMPTY
  return `${formatDate(d, locale)} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// Relative time is never negative: a timestamp in the future (clock skew, demo data)
// reads as "just now" instead of "-3 minutes ago".
export function formatRelative(
  value: string | number | Date | null | undefined,
  locale: UiLocale = 'vi',
  now: Date = new Date(),
): string {
  const d = toDate(value)
  if (!d) return EMPTY
  const minutes = Math.floor((now.getTime() - d.getTime()) / 60000)
  if (minutes < 1) return locale === 'vi' ? 'vừa xong' : 'just now'
  if (minutes < 60) return locale === 'vi' ? `${minutes} phút trước` : `${minutes} min ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return locale === 'vi' ? `${hours} giờ trước` : `${hours} h ago`
  const days = Math.floor(hours / 24)
  if (locale === 'vi') return `${days} ngày trước`
  return days === 1 ? '1 day ago' : `${days} days ago`
}

export function formatNumber(value: number | null | undefined, locale: UiLocale = 'vi', fractionDigits = 0): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return EMPTY
  return value.toLocaleString(locale === 'vi' ? 'vi-VN' : 'en-US', {
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  })
}

export function formatCurrency(value: number | null | undefined, currency: 'VND' | 'USD', locale: UiLocale = 'vi'): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return EMPTY
  const digits = currency === 'VND' ? 0 : 2
  const amount = formatNumber(value, locale, digits)
  if (currency === 'VND') return `${amount} ₫`
  return locale === 'vi' ? `${amount} US$` : `$${amount}`
}
