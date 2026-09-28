// @vitest-environment happy-dom
// CCMAI-UX-002: UI-structure test only. The API is mocked; nothing here is governance evidence.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { createI18n } from 'vue-i18n'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import viMessages from '../i18n/vi'
import enMessages from '../i18n/en'

const apiGet = vi.fn()
vi.mock('../api', () => ({ default: { get: (...args: unknown[]) => apiGet(...args), post: vi.fn(), delete: vi.fn() } }))

import Results from '../views/Results.vue'

// CCMAI-UX-011 moved the note into the shared SourceStatusPanel above the list, and cards
// into ResultCard. Selectors follow the new structure; the three assertions are unchanged.
const NOTE = '[data-testid="results-source-panel"] [data-testid="source-note"]'

const facets = {
  types: { qc_analysis: { jobs: 1, results: 2 } },
  jobs: [{ id: 'j1', name: 'QC mẫu', job_type: 'qc_analysis' }],
  channels: [{ id: 'c1', name: 'Kênh mẫu' }],
  tags: [],
}

function row(id: string, status: string) {
  return {
    id,
    job_id: 'j1',
    job_name: 'QC mẫu',
    job_type: 'qc_analysis',
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
  }
}

function setup(items: unknown[]) {
  apiGet.mockImplementation((url: string) => {
    if (url.endsWith('/results/facets')) return Promise.resolve({ data: facets })
    if (url.endsWith('/results'))
      return Promise.resolve({ data: { items, total: items.length, counts: { all: items.length, pass: items.length, fail: 0, skip: 0, classified: 0 } } })
    return Promise.resolve({ data: {} })
  })
}

async function mountResults() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:tenantId/results', component: Results }],
  })
  await router.push('/t1/results')
  await router.isReady()
  const w = mount(Results, {
    global: {
      plugins: [
        createVuetify({ components, directives }),
        createI18n({ legacy: false, locale: 'vi', fallbackLocale: 'en', messages: { vi: viMessages, en: enMessages } }),
        createPinia(),
        router,
      ],
    },
  })
  await flushPromises()
  await flushPromises()
  return w
}

describe('Results source note (UX-07)', () => {
  beforeEach(() => {
    apiGet.mockReset()
    localStorage.clear()
  })

  it('table view: the note sits above the table and the source column has a visible header', async () => {
    localStorage.setItem('cqa_results_view', 'table')
    setup([row('1', 'legacy_unverified'), row('2', 'changed_since_analysis')])
    const w = await mountResults()
    const note = w.find(NOTE)
    expect(note.exists()).toBe(true)
    expect(note.text()).toBe(viMessages.results_source_note)
    const table = w.find('table')
    expect(table.exists()).toBe(true)
    // The note precedes the table in document order.
    expect(note.element.compareDocumentPosition(table.element) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(w.find('[data-testid="results-source-header"]').text()).toBe(viMessages.results_col_source)
    expect(w.findAll('tbody tr')).toHaveLength(2)
  })

  it('card view: the note sits above the first card', async () => {
    localStorage.setItem('cqa_results_view', 'card')
    setup([row('1', 'verification_unavailable')])
    const w = await mountResults()
    const note = w.find(NOTE)
    expect(note.exists()).toBe(true)
    expect(w.find('table').exists()).toBe(false)
    const firstCard = w.find('.ccma-rcard')
    expect(firstCard.exists()).toBe(true)
    expect(note.element.compareDocumentPosition(firstCard.element) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('no rows: no note, because there is no status to qualify', async () => {
    setup([])
    const w = await mountResults()
    expect(w.find(NOTE).exists()).toBe(false)
  })
})
