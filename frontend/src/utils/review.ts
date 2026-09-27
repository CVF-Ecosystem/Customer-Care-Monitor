// CCMAI-UX-000: shared display semantics for verdicts and the "Cần xem lại" (needs review) rule.
import { SOURCE_INTEGRITY_ORDER, type SourceIntegrityStatus } from '../stores/jobs'

export type Verdict = 'pass' | 'fail' | 'skip' | 'classified'

// Same mapping as the current Results page: anything that is not PASS or SKIP is a failure,
// so an unexpected severity never reads as a pass.
export function verdictFromSeverity(severity: string | null | undefined): Verdict {
  if (severity === 'PASS') return 'pass'
  if (severity === 'SKIP') return 'skip'
  return 'fail'
}

// Outline icons; the changed status uses the warning triangle. None is a check mark.
export const SOURCE_STATUS_ICON: Record<SourceIntegrityStatus, string> = {
  changed_since_analysis: 'mdi-alert-outline',
  verification_unavailable: 'mdi-help-circle-outline',
  legacy_unverified: 'mdi-clock-outline',
  bound_currentness_unverified: 'mdi-shield-alert-outline',
}

// A missing or unknown value is unavailable, never silently dropped or treated as fine.
export function normalizeSourceStatus(status: string | null | undefined): SourceIntegrityStatus {
  return SOURCE_INTEGRITY_ORDER.includes(status as SourceIntegrityStatus)
    ? (status as SourceIntegrityStatus)
    : 'verification_unavailable'
}

// Count per status, in SOURCE_INTEGRITY_ORDER (most concerning first); zero counts omitted.
export function countSourceStatuses(statuses: (string | null | undefined)[]): { status: SourceIntegrityStatus; count: number }[] {
  const counts = new Map<SourceIntegrityStatus, number>()
  for (const s of statuses) {
    const n = normalizeSourceStatus(s)
    counts.set(n, (counts.get(n) ?? 0) + 1)
  }
  return SOURCE_INTEGRITY_ORDER.filter((s) => counts.has(s)).map((s) => ({ status: s, count: counts.get(s)! }))
}

// "Cần xem lại": failed, or the source changed after analysis, or it could not be verified.
export function needsReview(result: { severity?: string | null; source_integrity_status?: string | null }): boolean {
  if (verdictFromSeverity(result.severity) === 'fail') return true
  const s = normalizeSourceStatus(result.source_integrity_status)
  return s === 'changed_since_analysis' || s === 'verification_unavailable'
}

export type SyncKind = 'never' | 'syncing' | 'success' | 'partial' | 'error' | 'unknown'

// Channel last_sync_status for display. Empty means never synced; any unexpected value is
// "unknown" rather than success. "syncing" means started, not completed (R007).
export function syncKind(status: string | null | undefined): SyncKind {
  if (status === null || status === undefined || status === '' || status === 'never') return 'never'
  if (status === 'syncing' || status === 'success' || status === 'partial' || status === 'error') return status
  return 'unknown'
}

// R005: a percentage is shown only for a model-reported, uncalibrated value in [0, 1].
// Anything else (null, unavailable basis, out of range) is "not available", never 0% or 100%.
export function confidencePercent(confidence: number | null | undefined, basis: string | null | undefined): number | null {
  if (basis !== 'model_reported_uncalibrated') return null
  if (confidence === null || confidence === undefined || !Number.isFinite(confidence)) return null
  if (confidence < 0 || confidence > 1) return null
  return Math.round(confidence * 100)
}
