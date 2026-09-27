// CCMAI-UX-010: pure logic behind the Job Detail screen, kept out of the view so it can be tested.
import { distinctSourceIntegrity, type JobResult, type JobRun, type SourceIntegrityStatus } from '../../../stores/jobs'
import { verdictFromSeverity } from '../../../utils/review'

export interface ConversationGroup {
  conversationId: string
  runId: string
  customerName: string
  conversationDate: string
  verdict: string // PASS | SKIP | anything else (a failure)
  score: number | null
  review: string
  confidence: number | null
  confidenceBasis: string
  violations: JobResult[] // QC violations, or classification tags
  tags: string[]
  // Every distinct source status among the group's results, most concerning first.
  sourceStatuses: SourceIntegrityStatus[]
}

export function parseDetail(s: string | null | undefined): Record<string, any> {
  try {
    const v = JSON.parse(s || '{}')
    return v && typeof v === 'object' ? v : {}
  } catch {
    return {}
  }
}

// Results limited to one run, or every run when runId is null ("Mọi lần chạy").
export function scopeResults(results: JobResult[], runId: string | null): JobResult[] {
  return runId ? results.filter((r) => r.job_run_id === runId) : results
}

// Default scope: the most recent run that has results; none -> every run.
export function defaultScopeRunId(runs: JobRun[], results: JobResult[]): string | null {
  const withResults = new Set(results.map((r) => r.job_run_id))
  const sorted = [...runs].sort((a, b) => (b.started_at || '').localeCompare(a.started_at || ''))
  return sorted.find((r) => withResults.has(r.id))?.id ?? null
}

// One group per conversation, built from that conversation's latest run within the given results.
// Behavior carried over from the previous view; within a single-run scope it is simply that run.
export function groupResults(results: JobResult[]): ConversationGroup[] {
  if (!results.length) return []
  const latest = new Map<string, { runId: string; at: string }>()
  for (const r of results) {
    const cur = latest.get(r.conversation_id)
    if (!cur || r.created_at > cur.at) latest.set(r.conversation_id, { runId: r.job_run_id, at: r.created_at })
  }

  const groups = new Map<string, ConversationGroup>()
  const members = new Map<string, JobResult[]>()
  for (const r of results) {
    const cid = r.conversation_id
    if (r.job_run_id !== latest.get(cid)!.runId) continue
    if (!members.has(cid)) members.set(cid, [])
    members.get(cid)!.push(r)
    if (!groups.has(cid)) {
      groups.set(cid, {
        conversationId: cid,
        runId: r.job_run_id,
        customerName: r.customer_name || '',
        conversationDate: r.conversation_date || r.created_at,
        verdict: 'PASS',
        score: null,
        review: '',
        confidence: null,
        confidenceBasis: 'unavailable',
        violations: [],
        tags: [],
        sourceStatuses: [],
      })
    }
    const g = groups.get(cid)!
    if (r.result_type === 'conversation_evaluation') {
      g.verdict = r.severity
      g.review = r.evidence
      g.score = parseDetail(r.detail).score ?? null
      g.confidence = r.confidence ?? null
      g.confidenceBasis = r.confidence_basis || 'unavailable'
    } else if (r.result_type === 'classification_tag') {
      g.tags.push(r.rule_name)
      g.violations.push(r)
      if (!g.review) {
        const summary = parseDetail(r.detail).summary
        if (summary) g.review = summary
      }
    } else {
      g.violations.push(r)
    }
  }
  for (const [cid, g] of groups) g.sourceStatuses = distinctSourceIntegrity(members.get(cid) || [])
  return [...groups.values()].sort((a, b) => b.conversationDate.localeCompare(a.conversationDate))
}

// The status a group is counted under in the source panel: its most concerning one.
export function primarySourceStatus(g: ConversationGroup): SourceIntegrityStatus {
  return g.sourceStatuses[0] ?? 'verification_unavailable'
}

// "Cần xem lại": a QC failure, or a source that changed or cannot be verified.
// Classification has no pass/fail, so only the source status counts there.
export function groupNeedsReview(g: ConversationGroup, isClassification: boolean): boolean {
  if (!isClassification && g.verdict !== 'SKIP' && verdictFromSeverity(g.verdict) === 'fail') return true
  return g.sourceStatuses.some((s) => s === 'changed_since_analysis' || s === 'verification_unavailable')
}

export interface QcMetrics {
  total: number
  skipped: number
  evaluated: number // excludes SKIP
  passed: number
  passRate: number | null
  issues: number
  avgScore: number | null
}

// SKIP-aware: evaluated, pass rate, issues and score exclude skipped conversations, and the
// screen says so (this was the real cause of the baseline "110 vs 100", finding UX-04).
export function qcMetrics(groups: ConversationGroup[]): QcMetrics {
  const skipped = groups.filter((g) => g.verdict === 'SKIP').length
  const evaluated = groups.filter((g) => g.verdict !== 'SKIP')
  const passed = evaluated.filter((g) => g.verdict === 'PASS').length
  const scores = evaluated.map((g) => g.score).filter((s): s is number => typeof s === 'number')
  return {
    total: groups.length,
    skipped,
    evaluated: evaluated.length,
    passed,
    passRate: evaluated.length ? Math.round((passed / evaluated.length) * 100) : null,
    issues: evaluated.reduce((n, g) => n + g.violations.length, 0),
    avgScore: scores.length ? Math.round(scores.reduce((a, b) => a + b, 0) / scores.length) : null,
  }
}

export interface ClassificationMetrics {
  total: number
  classified: number
  skipped: number
  topTag: { name: string; count: number } | null
  tagCounts: { name: string; count: number }[]
}

export function classificationMetrics(groups: ConversationGroup[]): ClassificationMetrics {
  const counts = new Map<string, number>()
  for (const g of groups) for (const t of new Set(g.tags)) counts.set(t, (counts.get(t) ?? 0) + 1)
  const tagCounts = [...counts.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name))
  const skipped = groups.filter((g) => g.verdict === 'SKIP').length
  return { total: groups.length, classified: groups.length - skipped, skipped, topTag: tagCounts[0] ?? null, tagCounts }
}

export type RunKind = 'running' | 'success' | 'partial' | 'error' | 'cancelled' | 'unknown'

export function runStatusKind(status: string | null | undefined): RunKind {
  if (status === 'running' || status === 'success' || status === 'partial' || status === 'cancelled') return status
  if (status === 'error' || status === 'failed') return 'error'
  return 'unknown'
}

// Progress of a running run, from the counters the analyzer writes into its summary.
export function runProgress(run: JobRun | undefined): { analyzed: number; found: number; errors: number; passed: number } | null {
  if (!run || run.status !== 'running') return null
  const s = parseDetail(run.summary)
  if (!s.conversations_found) return null
  return {
    found: s.conversations_found,
    analyzed: (s.conversations_analyzed || 0) + (s.conversations_errors || 0),
    errors: s.conversations_errors || 0,
    passed: s.conversations_passed || 0,
  }
}

// Whole seconds between start and finish; null while running or when a timestamp is missing.
export function runDurationSeconds(run: JobRun): number | null {
  if (!run.started_at || !run.finished_at) return null
  const ms = new Date(run.finished_at).getTime() - new Date(run.started_at).getTime()
  return Number.isFinite(ms) && ms >= 0 ? Math.round(ms / 1000) : null
}

export interface EvidenceRef {
  message_id: string
  quote: string
}

export function evidenceRefs(result: JobResult): EvidenceRef[] {
  const refs = parseDetail(result.detail).evidence_refs
  if (!Array.isArray(refs)) return []
  return refs.filter((r): r is EvidenceRef => !!r && typeof r.message_id === 'string' && typeof r.quote === 'string' && r.quote !== '')
}

// A reference is shown only when its message is loaded and still contains the exact quote.
// Nothing is guessed: an unresolved reference highlights nothing.
export function resolveEvidence(result: JobResult, messages: { id: string; content?: string | null }[]): EvidenceRef[] {
  const byId = new Map(messages.map((m) => [m.id, m.content || '']))
  return evidenceRefs(result).filter((ref) => (byId.get(ref.message_id) ?? '').includes(ref.quote))
}
