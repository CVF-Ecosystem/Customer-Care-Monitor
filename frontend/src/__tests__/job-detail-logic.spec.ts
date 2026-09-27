import { describe, it, expect } from 'vitest'
import type { JobResult, JobRun } from '../stores/jobs'
import {
  classificationMetrics,
  defaultScopeRunId,
  groupNeedsReview,
  groupResults,
  primarySourceStatus,
  qcMetrics,
  resolveEvidence,
  runDurationSeconds,
  runProgress,
  runStatusKind,
  scopeResults,
} from '../views/Jobs/job-detail/logic'

// CCMAI-UX-010: Job Detail logic. Synthetic fixtures only.
let n = 0
function res(p: Partial<JobResult> & { conversation_id: string; job_run_id: string; result_type: string }): JobResult {
  n++
  return {
    id: `r${n}`,
    severity: 'PASS',
    rule_name: '',
    evidence: '',
    detail: '{}',
    confidence: null,
    confidence_basis: 'unavailable',
    source_integrity_status: 'legacy_unverified',
    created_at: `2026-09-27T10:${String(n).padStart(2, '0')}:00Z`,
    ...p,
  } as JobResult
}
function run(id: string, started: string, status = 'success', extra: Partial<JobRun> = {}): JobRun {
  return { id, job_id: 'j', started_at: started, finished_at: null, status, summary: '{}', error_message: '', ...extra }
}

const evalRes = (conv: string, runId: string, severity: string, score?: number, status = 'legacy_unverified') =>
  res({ conversation_id: conv, job_run_id: runId, result_type: 'conversation_evaluation', severity, detail: JSON.stringify({ score }), source_integrity_status: status as never })
const violation = (conv: string, runId: string, detail: object = {}, status = 'legacy_unverified') =>
  res({ conversation_id: conv, job_run_id: runId, result_type: 'qc_violation', severity: 'CAN_CAI_THIEN', rule_name: 'Chào hỏi', detail: JSON.stringify(detail), source_integrity_status: status as never })

describe('scope', () => {
  const runs = [run('old', '2026-09-20T10:00:00Z'), run('new', '2026-09-27T10:00:00Z'), run('running', '2026-09-28T10:00:00Z', 'running')]
  const results = [evalRes('a', 'old', 'PASS'), evalRes('a', 'new', 'FAIL'), evalRes('b', 'old', 'PASS')]

  it('defaults to the most recent run that has results, not a running run without results', () => {
    expect(defaultScopeRunId(runs, results)).toBe('new')
    expect(defaultScopeRunId(runs, [])).toBeNull()
  })

  it('filters by run, or keeps every run for "Mọi lần chạy"', () => {
    expect(scopeResults(results, 'new').map((r) => r.conversation_id)).toEqual(['a'])
    expect(scopeResults(results, null)).toHaveLength(3)
  })

  it('all-runs grouping keeps each conversation once, from its latest run', () => {
    const groups = groupResults(results)
    expect(groups).toHaveLength(2)
    expect(groups.find((g) => g.conversationId === 'a')!.verdict).toBe('FAIL')
    expect(groups.find((g) => g.conversationId === 'a')!.runId).toBe('new')
  })
})

describe('metrics (UX-04: skipped conversations are excluded and counted separately)', () => {
  it('matches the baseline shape: 110 total = 80 pass + 20 fail + 10 skip, 100 evaluated', () => {
    const rs: JobResult[] = []
    for (let i = 0; i < 80; i++) rs.push(evalRes(`p${i}`, 'x', 'PASS', 90))
    for (let i = 0; i < 20; i++) rs.push(evalRes(`f${i}`, 'x', 'FAIL', 40), violation(`f${i}`, 'x'))
    for (let i = 0; i < 10; i++) rs.push(evalRes(`s${i}`, 'x', 'SKIP'))
    const m = qcMetrics(groupResults(rs))
    expect(m).toMatchObject({ total: 110, skipped: 10, evaluated: 100, passed: 80, passRate: 80, issues: 20 })
    expect(m.avgScore).toBe(80)
  })

  it('returns null, never 0, when nothing was evaluated or scored', () => {
    expect(qcMetrics(groupResults([evalRes('s', 'x', 'SKIP')]))).toMatchObject({ evaluated: 0, passRate: null, avgScore: null })
    expect(qcMetrics(groupResults([evalRes('a', 'x', 'PASS')])).avgScore).toBeNull()
  })

  it('classification counts each tag once per conversation', () => {
    const tag = (conv: string, name: string) => res({ conversation_id: conv, job_run_id: 'x', result_type: 'classification_tag', rule_name: name })
    const m = classificationMetrics(groupResults([tag('a', 'Góp ý'), tag('a', 'Góp ý'), tag('b', 'Góp ý'), tag('b', 'Khiếu nại'), evalRes('c', 'x', 'SKIP')]))
    expect(m).toMatchObject({ total: 3, classified: 2, skipped: 1, topTag: { name: 'Góp ý', count: 2 } })
  })
})

describe('needs review and source status', () => {
  it('flags QC failures and changed/unverifiable sources, not skips or legacy passes', () => {
    const [fail] = groupResults([evalRes('a', 'x', 'FAIL')])
    const [skip] = groupResults([evalRes('b', 'x', 'SKIP')])
    const [legacyPass] = groupResults([evalRes('c', 'x', 'PASS')])
    const [changed] = groupResults([evalRes('d', 'x', 'PASS', 90, 'changed_since_analysis')])
    expect(groupNeedsReview(fail, false)).toBe(true)
    expect(groupNeedsReview(skip, false)).toBe(false)
    expect(groupNeedsReview(legacyPass, false)).toBe(false)
    expect(groupNeedsReview(changed, false)).toBe(true)
    // Classification has no pass/fail; only the source matters.
    expect(groupNeedsReview(fail, true)).toBe(false)
    expect(groupNeedsReview(changed, true)).toBe(true)
  })

  it('a mixed group is counted under its most concerning status', () => {
    const [g] = groupResults([evalRes('a', 'x', 'FAIL', 40, 'bound_currentness_unverified'), violation('a', 'x', {}, 'changed_since_analysis')])
    expect(g.sourceStatuses).toEqual(['changed_since_analysis', 'bound_currentness_unverified'])
    expect(primarySourceStatus(g)).toBe('changed_since_analysis')
  })
})

describe('runs', () => {
  it('maps statuses without inventing success', () => {
    expect(runStatusKind('running')).toBe('running')
    expect(runStatusKind('failed')).toBe('error')
    expect(runStatusKind('error')).toBe('error')
    expect(runStatusKind('cancelled')).toBe('cancelled')
    expect(runStatusKind('partial')).toBe('partial')
    expect(runStatusKind('weird')).toBe('unknown')
  })

  it('reads live progress only for a running run', () => {
    const summary = JSON.stringify({ conversations_found: 100, conversations_analyzed: 30, conversations_errors: 2, conversations_passed: 25 })
    expect(runProgress(run('r', '2026-09-28T10:00:00Z', 'running', { summary }))).toEqual({ found: 100, analyzed: 32, errors: 2, passed: 25 })
    expect(runProgress(run('r', '2026-09-28T10:00:00Z', 'success', { summary }))).toBeNull()
  })

  it('duration is null while running', () => {
    expect(runDurationSeconds(run('r', '2026-09-28T10:00:00Z'))).toBeNull()
    expect(runDurationSeconds(run('r', '2026-09-28T10:00:00Z', 'success', { finished_at: '2026-09-28T10:06:12Z' }))).toBe(372)
  })
})

describe('evidence resolution', () => {
  const v = violation('a', 'x', {
    evidence_refs: [
      { message_id: 'm1', quote: 'Mã đơn?' },
      { message_id: 'm2', quote: 'Đơn đang giao.' },
      { message_id: 'gone', quote: 'không còn' },
      { message_id: 'm3', quote: '' },
    ],
  })
  it('keeps only references whose message exists and still contains the exact quote', () => {
    const messages = [
      { id: 'm1', content: 'Mã đơn?' },
      { id: 'm2', content: 'Đơn đã giao xong.' }, // edited after analysis
      { id: 'm3', content: 'x' },
    ]
    expect(resolveEvidence(v, messages).map((r) => r.message_id)).toEqual(['m1'])
  })
  it('legacy results without references resolve to nothing', () => {
    expect(resolveEvidence(violation('a', 'x', { explanation: 'x' }), [{ id: 'm1', content: 'Mã đơn?' }])).toEqual([])
    expect(resolveEvidence(violation('a', 'x'), [])).toEqual([])
  })
})
