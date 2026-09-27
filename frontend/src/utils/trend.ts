// CCMAI-UX-001a: Job Detail quality trend, grouped by Vietnam calendar day.
// Logic moved unchanged from JobDetail.vue except the day key: grouping and label now
// come from the same Vietnam-time day, so a day can no longer appear twice or go missing.
import { dayMonthLabel, vnDateKey } from './format'

export interface TrendInput {
  conversation_id: string
  conversation_date?: string
  created_at: string
  result_type: string
  severity: string
}

export interface TrendPoint {
  day: string // YYYY-MM-DD in Vietnam time
  label: string // dd/mm
  passed: number
  failed: number
}

export function qualityTrendByDay(results: TrendInput[]): TrendPoint[] {
  // Step 1: one verdict per conversation (a violation fails it; SKIP is excluded).
  const convMap = new Map<string, { date: string; hasViolation: boolean; isSkip: boolean }>()
  for (const r of results) {
    const existing = convMap.get(r.conversation_id) || {
      date: r.conversation_date || r.created_at,
      hasViolation: false,
      isSkip: false,
    }
    if (r.result_type === 'qc_violation') existing.hasViolation = true
    if (r.result_type === 'conversation_evaluation' && r.severity === 'SKIP') existing.isSkip = true
    convMap.set(r.conversation_id, existing)
  }
  // Step 2: count conversations per Vietnam day.
  const byDay = new Map<string, TrendPoint>()
  for (const [, conv] of convMap) {
    const day = vnDateKey(conv.date)
    if (!day) continue
    const point = byDay.get(day) || { day, label: dayMonthLabel(day), passed: 0, failed: 0 }
    if (conv.isSkip) {
      // excluded from the trend
    } else if (conv.hasViolation) {
      point.failed += 1
    } else {
      point.passed += 1
    }
    byDay.set(day, point)
  }
  return [...byDay.values()].sort((a, b) => a.day.localeCompare(b.day))
}
