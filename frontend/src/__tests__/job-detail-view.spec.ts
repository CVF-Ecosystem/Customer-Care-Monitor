// @vitest-environment happy-dom
// CCMAI-UX-010: UI-structure test of the Job Detail screen. The API is mocked with synthetic
// data; nothing here is governance evidence.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import viMessages from '../i18n/vi'
import enMessages from '../i18n/en'
import { lightColors } from '../styles/tokens'
import { useAuthStore } from '../stores/auth'

vi.mock('vue-chartjs', () => ({ Line: { name: 'Line', render: () => null } }))
const apiGet = vi.fn()
vi.mock('../api', () => ({ default: { get: (...a: unknown[]) => apiGet(...a), post: vi.fn(), delete: vi.fn() } }))

import JobDetail from '../views/Jobs/JobDetail.vue'

const g = globalThis as unknown as { visualViewport?: unknown; ResizeObserver?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

const job = { id: 'j1', name: 'QC mẫu', job_type: 'qc_analysis', schedule_type: 'manual', schedule_cron: '', input_channel_ids: '["c1"]', last_run_at: '2026-09-27T15:36:00Z', last_run_status: 'success' }
const runs = [
  { id: 'new', job_id: 'j1', started_at: '2026-09-27T15:30:00Z', finished_at: '2026-09-27T15:36:12Z', status: 'success', summary: '{"conversations_analyzed":3,"conversations_passed":1,"issues_found":1}', error_message: '' },
  { id: 'old', job_id: 'j1', started_at: '2026-09-20T15:30:00Z', finished_at: '2026-09-20T15:35:00Z', status: 'success', summary: '{}', error_message: '' },
]
let k = 0
const r = (conv: string, run: string, type: string, severity: string, status = 'legacy_unverified', extra = {}) => ({
  id: `x${++k}`, job_run_id: run, conversation_id: conv, result_type: type, severity, rule_name: type === 'qc_violation' ? 'Chào hỏi' : '',
  evidence: 'Nhận xét', detail: '{"score":70}', confidence: null, confidence_basis: 'unavailable', source_integrity_status: status,
  created_at: `2026-09-2${run === 'new' ? 7 : 0}T15:3${k % 10}:00Z`, conversation_date: '2026-09-27T09:00:00Z', customer_name: `Khách ${conv}`, ...extra,
})
const results = [
  r('a', 'new', 'conversation_evaluation', 'FAIL', 'changed_since_analysis'),
  r('a', 'new', 'qc_violation', 'CAN_CAI_THIEN', 'changed_since_analysis'),
  r('b', 'new', 'conversation_evaluation', 'PASS'),
  r('c', 'new', 'conversation_evaluation', 'SKIP'),
  r('d', 'old', 'conversation_evaluation', 'PASS'), // only in the older run
]

function setupApi(opts: { runs?: unknown[]; results?: unknown[] } = {}) {
  apiGet.mockImplementation((url: string) => {
    if (url.endsWith('/jobs/j1')) return Promise.resolve({ data: job })
    if (url.endsWith('/jobs/j1/runs')) return Promise.resolve({ data: opts.runs ?? runs })
    if (url.endsWith('/jobs/j1/results')) return Promise.resolve({ data: opts.results ?? results })
    if (url.endsWith('/settings')) return Promise.resolve({ data: { settings: { ai_provider: 'claude' } } })
    return Promise.resolve({ data: {} })
  })
}

async function mountView() {
  const pinia = createPinia()
  setActivePinia(pinia)
  useAuthStore().tenantPerms = { role: 'owner', permissions: {} } as never
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:tenantId/jobs/:jobId', component: JobDetail }, { path: '/:p(.*)*', component: { template: '<div />' } }] })
  await router.push('/t1/jobs/j1')
  await router.isReady()
  const w = mount(JobDetail, {
    global: {
      plugins: [
        createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } }),
        createI18n({ legacy: false, locale: 'vi', fallbackLocale: 'en', messages: { vi: viMessages, en: enMessages } }),
        pinia,
        router,
      ],
    },
  })
  for (let i = 0; i < 4; i++) await flushPromises()
  return w
}

describe('Job Detail (UX-010)', () => {
  beforeEach(() => {
    apiGet.mockReset()
    k = 0
  })

  it('defaults to the latest run with results and names that scope', async () => {
    setupApi()
    const w = await mountView()
    const text = w.text()
    expect(text).toContain('Số liệu và danh sách của lần chạy 27/09/2026')
    // Scope = run "new": 3 conversations, the older run's conversation "d" is not listed.
    expect(text).not.toContain('Khách d')
    // UX-04: evaluated excludes the skipped conversation and says so.
    expect(text).toContain('3 hội thoại, không tính 1 bỏ qua')
  })

  it('defaults the filter to "Cần xem lại" when something needs review, and keeps the local-only note', async () => {
    setupApi()
    const w = await mountView()
    const selected = w.findAll('.v-chip').find((c) => c.classes().includes('v-chip--variant-flat'))
    expect(selected?.text()).toContain(viMessages.ui_needs_review)
    expect(w.text()).toContain(viMessages.results_source_note)
    expect(w.text()).toContain('Khách a')
    expect(w.text()).not.toContain('Khách b') // passed and legacy: not in the review list
  })

  it('labels exports as all runs and keeps destructive actions out of the header', async () => {
    setupApi()
    const w = await mountView()
    expect(w.text()).toContain(viMessages.jd_export_all_csv)
    expect(w.text()).toContain(viMessages.jd_export_all_xlsx)
    const headerText = w.find('.jd-actions').text()
    expect(headerText).not.toContain(viMessages.jd_action_clear_results)
    expect(w.find('[data-testid="action-menu"]').exists()).toBe(true)
  })

  it('shows the never-run state with actions when there is no run and no result', async () => {
    setupApi({ runs: [], results: [] })
    const w = await mountView()
    expect(w.text()).toContain(viMessages.jd_never_run_title)
    expect(w.find('.jd-metrics').exists()).toBe(false)
  })

  it('shows a load error with retry instead of zeros', async () => {
    apiGet.mockImplementation((url: string) => (url.endsWith('/jobs/j1') ? Promise.reject(new Error('down')) : Promise.resolve({ data: [] })))
    const w = await mountView()
    expect(w.find('[role="alert"]').text()).toContain(viMessages.jd_load_error)
    expect(w.text()).toContain(viMessages.jd_retry)
  })
})
