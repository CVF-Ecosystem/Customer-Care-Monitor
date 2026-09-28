// CCMAI-UX-013: pure display helpers for the AI job list. They only read job fields the
// API already returns; nothing here changes how or when a job runs.

export type ScheduleText =
  | { kind: 'after_sync' }
  | { kind: 'manual' }
  | { kind: 'daily'; time: string }
  | { kind: 'weekly'; days: number[]; time: string }
  | { kind: 'monthly'; day: number; time: string }
  | { kind: 'raw'; cron: string }

const pad = (n: number) => String(n).padStart(2, '0')

function isInt(s: string) {
  return /^\d+$/.test(s)
}

// Reads the same three cron shapes the CronPicker writes (daily, weekly, monthly). Anything
// else is shown as the raw expression rather than guessed into words.
export function describeSchedule(scheduleType: string | null | undefined, cron: string | null | undefined): ScheduleText {
  if (scheduleType === 'after_sync') return { kind: 'after_sync' }
  if (scheduleType === 'manual') return { kind: 'manual' }
  const expr = (cron || '').trim()
  const parts = expr.split(/\s+/)
  if (parts.length !== 5) return { kind: 'raw', cron: expr }
  const [min, hr, dom, mon, dow] = parts
  if (!isInt(min) || !isInt(hr) || mon !== '*') return { kind: 'raw', cron: expr }
  const m = Number(min)
  const h = Number(hr)
  if (m > 59 || h > 23) return { kind: 'raw', cron: expr }
  const time = `${pad(h)}:${pad(m)}`
  if (dom === '*' && dow === '*') return { kind: 'daily', time }
  if (dom === '*' && /^\d(,\d)*$/.test(dow)) {
    const days = [...new Set(dow.split(',').map(Number))].sort((a, b) => a - b)
    if (days.every((d) => d >= 0 && d <= 6)) return { kind: 'weekly', days, time }
  }
  if (dow === '*' && isInt(dom) && Number(dom) >= 1 && Number(dom) <= 31) return { kind: 'monthly', day: Number(dom), time }
  return { kind: 'raw', cron: expr }
}

// input_channel_ids is JSON text; a malformed value counts as unknown (null), never 0.
export function channelCount(inputChannelIds: string | null | undefined): number | null {
  try {
    const v = JSON.parse(inputChannelIds || '[]')
    return Array.isArray(v) ? v.length : null
  } catch {
    return null
  }
}

