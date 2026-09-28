// @vitest-environment happy-dom
// CCMAI-UX-011: UI-structure and interaction tests for the Results screen. The API is mocked;
// nothing here is governance evidence.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { createI18n } from 'vue-i18n'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import viMessages from '../i18n/vi'
import enMessages from '../i18n/en'
import { lightColors } from '../styles/tokens'

const apiGet = vi.fn()
vi.mock('../api', () => ({ default: { get: (...args: unknown[]) => apiGet(...args), post: vi.fn(), delete: vi.fn() } }))

import Results from '../views/Results.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

const qcFacets = {
  types: { qc_analysis: { jobs: 1, results: 3 }, classification: { jobs: 0, results: 0 } },
  jobs: [{ id: 'j1', name: 'QC mẫu', job_type: 'qc_analysis' }],
  channels: [{ id: 'c1', name: 'Kênh mẫu' }],
  tags: [],
}
const clsFacets = {
  types: { qc_analysis: { jobs: 0, results: 0 }, classification: { jobs: 1, results: 1 } },
  jobs: [{ id: 'j2', name: 'Phân loại mẫu', job_type: 'classification' }],
  channels: [{ id: 'c1', name: 'Kênh mẫu' }],
  tags: ['Khiếu nại'],
}

function row(id: string, status: string, extra: Record<string, unknown> = {}) {
  return {
    id,
    job_id: 'j1',
    job_name: 'QC mẫu',
    channel_id: 'c1',
    channel_name: 'Kênh mẫu',
    conversation_id: `conv-${id}`,
    customer_name: `Khách ${id}`,
    conversation_at: '2026-09-27T10:00:00+07:00',
    evaluated_at: '2026-09-27T10:30:00+07:00',
    severity: 'PASS',
    review: 'Tổng hợp',
    score: 90,
    issues: [],
    tags: [],
    source_integrity_status: status,
    ...extra,
  }
}

const defaultRows = [
  row('1', 'legacy_unverified'),
  row('2', 'changed_since_analysis', { severity: 'FAIL', score: 40, issues: [{ rule_name: 'Giải đáp đầy đủ', evidence: 'Chỉ trả lời "Có".', severity: 'CAN_CAI_THIEN' }] }),
  row('3', 'verification_unavailable'),
]
const defaultCounts = { all: 60, pass: 40, fail: 15, skip: 5, classified: 55 }

type Opts = { facets?: unknown; items?: unknown[]; total?: number; counts?: unknown; failResults?: boolean; failFacets?: boolean }
let opts: Opts = {}

function setup(o: Opts = {}) {
  opts = o
  apiGet.mockImplementation((url: string) => {
    if (url.endsWith('/results/facets')) {
      if (opts.failFacets) return Promise.reject(new Error('boom'))
      return Promise.resolve({ data: opts.facets ?? qcFacets })
    }
    if (url.endsWith('/results')) {
      if (opts.failResults) return Promise.reject(new Error('boom'))
      const items = opts.items ?? defaultRows
      return Promise.resolve({ data: { items, total: opts.total ?? 60, counts: opts.counts ?? defaultCounts } })
    }
    if (url.endsWith('/results/export')) return Promise.resolve({ data: new Blob(['x']) })
    if (url.includes('/messages')) return Promise.resolve({ data: { messages: [{ id: 'm1', sender_type: 'customer', sender_name: 'Khách', content: 'Xin chào shop', sent_at: '2026-09-27T09:00:00Z' }] } })
    return Promise.resolve({ data: {} })
  })
}

const mounted: VueWrapper[] = []

async function mountResults(view: 'table' | 'card' = 'table') {
  localStorage.setItem('cqa_results_view', view)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:tenantId/results', component: Results }, { path: '/:p(.*)*', component: { template: '<div />' } }],
  })
  await router.push('/t1/results')
  await router.isReady()
  const w = mount(Results, {
    attachTo: document.body,
    global: {
      plugins: [
        createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } }),
        createI18n({ legacy: false, locale: 'vi', fallbackLocale: 'en', messages: { vi: viMessages, en: enMessages } }),
        createPinia(),
        router,
      ],
    },
  })
  mounted.push(w)
  for (let i = 0; i < 3; i++) await flushPromises()
  return w
}

function resultCalls() {
  return apiGet.mock.calls.filter((c) => String(c[0]).endsWith('/results'))
}
const debounce = () => new Promise((r) => setTimeout(r, 300))

async function openRow(w: VueWrapper, index: number) {
  await w.findAll('tbody tr')[index].trigger('click')
  for (let i = 0; i < 3; i++) await flushPromises()
  return document.body.querySelector('[data-testid="results-dialog"]') as HTMLElement
}

describe('Results screen (UX-011)', () => {
  beforeEach(() => {
    apiGet.mockReset()
    localStorage.clear()
  })
  afterEach(() => {
    while (mounted.length) mounted.pop()!.unmount()
    document.body.innerHTML = ''
  })

  it('source panel counts only this page and says so; the note stays visible', async () => {
    setup()
    const w = await mountResults()
    const panel = w.find('[data-testid="results-source-panel"]')
    expect(panel.find('[data-testid="source-scope"]').text()).toBe('· trên trang này (3 kết quả)')
    expect(panel.find('[data-testid="source-note"]').text()).toBe(viMessages.results_source_note)
    const statuses = panel.findAll('li').map((li) => [li.attributes('data-status'), li.text()])
    expect(statuses).toEqual([
      ['changed_since_analysis', `${viMessages.results_source_changed}1`],
      ['verification_unavailable', `${viMessages.results_source_unavailable}1`],
      ['legacy_unverified', `${viMessages.results_source_legacy}1`],
    ])
    expect(panel.classes()).toContain('ccma-src-panel--alert')
  })

  it('table shows the source label as text on every row and marks changed rows', async () => {
    setup()
    const w = await mountResults()
    const rows = w.findAll('tbody tr')
    expect(rows).toHaveLength(3)
    expect(rows[0].find('td').text()).toBe(viMessages.results_source_legacy)
    expect(rows[1].find('td').text()).toBe(viMessages.results_source_changed)
    expect(rows[1].classes()).toContain('rs-row--changed_since_analysis')
    expect(rows[2].classes()).toContain('rs-row--verification_unavailable')
    expect(rows[0].attributes('tabindex')).toBe('0')
  })

  it('names the scope of every count: chips over all pages, range from the server total, export over the full filter', async () => {
    setup()
    const w = await mountResults()
    expect(w.find('[data-testid="counts-scope"]').text()).toBe(viMessages.results_counts_scope)
    expect(w.find('[data-testid="verdict-chips"]').text()).toContain('Không đạt · 15')
    expect(w.find('[data-testid="results-range"]').text()).toBe('Hiển thị 1–3 trên 60 kết quả')
    expect(w.find('[data-testid="export-scope"]').text()).toBe(viMessages.results_export_scope)
  })

  it('range on a later page counts from that page and requests it from the server', async () => {
    setup()
    const w = await mountResults()
    await w.find('.v-pagination').findAll('button').find((b) => b.text() === '2')!.trigger('click')
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(resultCalls().at(-1)![1].params.page).toBe(2)
    expect(w.find('[data-testid="results-range"]').text()).toBe('Hiển thị 26–28 trên 60 kết quả')
  })

  it('keeps the list and export request parameters of the server contract', async () => {
    setup()
    const w = await mountResults()
    expect(resultCalls()[0][1]).toEqual({
      params: { job_type: 'qc_analysis', verdict: 'all', date_field: 'conv', sort: 'recent', page: 1, page_size: 25 },
    })

    const chips = w.find('[data-testid="verdict-chips"]').findAll('.v-chip')
    await chips[1].trigger('click') // Không đạt
    await debounce()
    await flushPromises()
    expect(resultCalls().at(-1)![1].params).toEqual({ job_type: 'qc_analysis', verdict: 'fail', date_field: 'conv', sort: 'recent', page: 1, page_size: 25 })

    const urlApi = URL as unknown as { createObjectURL: unknown; revokeObjectURL: unknown }
    urlApi.createObjectURL = vi.fn(() => 'blob:x')
    urlApi.revokeObjectURL = vi.fn()
    await w.find('[data-testid="export-csv"]').trigger('click')
    await flushPromises()
    const exportCall = apiGet.mock.calls.find((c) => String(c[0]).endsWith('/results/export'))!
    expect(exportCall[1]).toEqual({
      params: { job_type: 'qc_analysis', verdict: 'fail', date_field: 'conv', sort: 'recent', format: 'csv' },
      responseType: 'blob',
    })
  })

  it('QC dialog: explains a changed source, lists issues, shows no confidence and loads the transcript', async () => {
    setup()
    const w = await mountResults()
    const dialog = await openRow(w, 1)
    expect(dialog).toBeTruthy()
    const text = dialog.textContent || ''
    expect(dialog.querySelector('[data-testid="source-hint"]')!.textContent).toBe(viMessages.results_source_changed_hint)
    expect(dialog.querySelector('[data-testid="results-dialog-source-note"]')!.textContent).toBe(viMessages.results_source_note)
    expect(text).toContain('Vấn đề (1)')
    expect(text).toContain('Giải đáp đầy đủ')
    expect(text).not.toContain(viMessages.results_tag_evidence)
    expect(dialog.querySelector('[data-testid="confidence"]')).toBeNull()
    expect(document.body.querySelector('[data-testid="ai-generated"]')).toBeTruthy()
    expect(apiGet.mock.calls.some((c) => String(c[0]) === '/tenants/t1/conversations/conv-2/messages')).toBe(true)
    expect(text).toContain('Xin chào shop')
  })

  it('QC dialog: explains an unavailable check and adds nothing for a legacy result', async () => {
    setup()
    const w = await mountResults()
    let dialog = await openRow(w, 2)
    expect(dialog.querySelector('[data-testid="source-hint"]')!.textContent).toBe(viMessages.results_source_unavailable_hint)
    dialog = await openRow(w, 0)
    expect(dialog.querySelector('[data-testid="source-hint"]')).toBeNull()
    expect(dialog.querySelector('[data-testid="results-dialog-source-note"]')).toBeTruthy()
  })

  it('classification: tags and tag evidence are never labelled as QC issues', async () => {
    setup({
      facets: clsFacets,
      items: [row('9', 'bound_currentness_unverified', { severity: 'INFO', score: null, tags: ['Khiếu nại'], issues: [{ rule_name: 'Khiếu nại', evidence: 'Đơn tới trễ 2 tiếng.', severity: 'INFO' }] })],
      counts: { all: 1, pass: 0, fail: 1, skip: 0, classified: 1 },
      total: 1,
    })
    const w = await mountResults()
    expect(w.find('thead').text()).toContain(viMessages.results_col_tags)
    expect(w.find('thead').text()).not.toContain(viMessages.results_col_issues)
    expect(w.find('[data-testid="verdict-chips"]').text()).toContain('Đã phân loại · 1')
    const dialog = await openRow(w, 0)
    const text = dialog.textContent || ''
    expect(dialog.querySelector('[data-testid="dialog-tags"]')!.textContent).toContain('Nhãn (1)')
    expect(dialog.querySelector('[data-testid="dialog-tag-evidence"]')!.textContent).toContain('Đơn tới trễ 2 tiếng.')
    expect(dialog.querySelector('[data-testid="dialog-issues"]')).toBeNull()
    expect(text).not.toContain('Vấn đề')
    expect(text).not.toContain(viMessages.verdict_fail)
  })

  it('card view: one ResultCard per row, opening the same dialog', async () => {
    setup()
    const w = await mountResults('card')
    const cards = w.findAll('.ccma-rcard')
    expect(cards).toHaveLength(3)
    expect(cards[1].classes()).toContain('ccma-rcard--changed')
    expect(cards[1].text()).toContain('1 vấn đề · Giải đáp đầy đủ')
    await cards[1].trigger('click')
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(document.body.querySelector('[data-testid="results-dialog"]')).toBeTruthy()
  })

  it('load error: inline alert with retry that repeats the request', async () => {
    setup({ failResults: true })
    const w = await mountResults()
    const alert = w.find('[data-testid="results-load-error"]')
    expect(alert.attributes('role')).toBe('alert')
    expect(w.find('[data-testid="results-source-panel"]').exists()).toBe(false)
    const before = resultCalls().length
    opts.failResults = false
    await alert.find('button').trigger('click')
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(resultCalls().length).toBe(before + 1)
    expect(w.find('[data-testid="results-load-error"]').exists()).toBe(false)
    expect(w.findAll('tbody tr')).toHaveLength(3)
  })

  it('facets error is not shown as "no jobs"', async () => {
    setup({ failFacets: true })
    const w = await mountResults()
    expect(w.find('[data-testid="results-load-error"]').exists()).toBe(true)
    expect(w.text()).not.toContain(viMessages.results_empty_title)
    opts.failFacets = false
    await w.find('[data-testid="results-load-error"] button').trigger('click')
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(w.findAll('tbody tr')).toHaveLength(3)
  })

  it('no match: offers "Xóa lọc", which drops the filter from the next request', async () => {
    setup()
    const w = await mountResults()
    opts.items = []
    await w.find('input[aria-label="Tìm tên khách"]').setValue('zzz')
    await debounce()
    await flushPromises()
    expect(resultCalls().at(-1)![1].params.q).toBe('zzz')
    expect(w.find('[data-testid="results-no-match"]').exists()).toBe(true)
    expect(w.find('[data-testid="results-source-panel"]').exists()).toBe(false)
    await w.find('[data-testid="no-match-clear"]').trigger('click')
    await debounce()
    await flushPromises()
    expect(resultCalls().at(-1)![1].params.q).toBeUndefined()
  })
})
