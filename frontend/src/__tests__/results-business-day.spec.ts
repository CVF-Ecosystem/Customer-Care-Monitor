// @vitest-environment happy-dom
// CCMAI-RUNTIME-026 (F04): the real Results component sends Vietnam calendar dates for list and
// export. Mocked API: request-contract evidence only (the same named cases drive the backend
// handler fixtures).
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
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
vi.mock('../api', () => ({ default: { get: (...a: unknown[]) => apiGet(...a), post: vi.fn(), delete: vi.fn() } }))

import Results from '../views/Results.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

const facets = {
  types: { qc_analysis: { jobs: 1, results: 3 }, classification: { jobs: 0, results: 0 } },
  jobs: [{ id: 'j1', name: 'QC mẫu', job_type: 'qc_analysis' }],
  channels: [{ id: 'c1', name: 'Kênh mẫu' }],
  tags: [],
}
// Mounting Vuetify screens and stepping through several presets can exceed the 5 s default when
// many spec files start at once; the cases below are not timing-sensitive otherwise.
vi.setConfig({ testTimeout: 30_000 })
// The frontend has no Node typings; the test runner's process.env.TZ is reached through globalThis.
const env = (globalThis as unknown as { process: { env: Record<string, string | undefined> } }).process.env
const originalTz = env.TZ
const mounted: VueWrapper[] = []

async function mountResults() {
  localStorage.setItem('cqa_results_view', 'table')
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:tenantId/results', component: Results }, { path: '/:p(.*)*', component: { template: '<div />' } }] })
  await router.push('/t1/results')
  await router.isReady()
  const w = mount(Results, {
    attachTo: document.body,
    global: { plugins: [
      createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } }),
      createI18n({ legacy: false, locale: 'vi', fallbackLocale: 'en', messages: { vi: viMessages, en: enMessages } }),
      createPinia(),
      router,
    ] },
  })
  mounted.push(w)
  for (let i = 0; i < 3; i++) await flushPromises()
  return w
}

const listCalls = () => apiGet.mock.calls.filter((c) => String(c[0]).endsWith('/results'))
const lastListParams = () => (listCalls()[listCalls().length - 1][1] as { params: Record<string, string> }).params
const settle = async () => { await new Promise((r) => setTimeout(r, 350)); for (let i = 0; i < 3; i++) await flushPromises() }

// Polls (real timers) until the latest list request carries the expected parameters: the screen
// debounces its reload, and a fixed sleep is flaky when the machine is busy.
async function waitParams(expected: Record<string, string>) {
  const deadline = Date.now() + 8000
  let last: Record<string, string> = {}
  for (;;) {
    for (let i = 0; i < 2; i++) await flushPromises()
    if (listCalls().length) {
      last = lastListParams()
      if (Object.entries(expected).every(([k, v]) => last[k] === v)) return
    }
    if (Date.now() > deadline) break
    await new Promise((r) => setTimeout(r, 50))
  }
  expect(last).toMatchObject(expected)
}

async function openDateMenu(w: VueWrapper) {
  const button = w.findAll('.loc-nut').find((b) => b.html().includes('mdi-calendar'))
  if (!button) throw new Error('date menu button missing')
  await button.trigger('click')
  await flushPromises()
}

async function clickPreset(label: string) {
  const chip = Array.from(document.body.querySelectorAll('.v-chip')).find((c) => c.textContent?.trim() === label) as HTMLElement | undefined
  if (!chip) throw new Error(`preset ${label} missing`)
  chip.click()
  await settle()
}

describe('Results date controls on the Vietnam calendar (F04)', () => {
  beforeEach(() => {
    apiGet.mockReset()
    localStorage.clear()
    apiGet.mockImplementation((url: string) => {
      if (url.endsWith('/results/facets')) return Promise.resolve({ data: facets })
      if (url.endsWith('/results')) return Promise.resolve({ data: { items: [], total: 0, counts: { all: 0, pass: 0, fail: 0, skip: 0, classified: 0 } } })
      if (url.endsWith('/results/export')) return Promise.resolve({ data: new Blob(['x']) })
      return Promise.resolve({ data: {} })
    })
  })
  afterEach(() => {
    mounted.forEach((w) => w.unmount())
    mounted.length = 0
    document.body.innerHTML = ''
    vi.useRealTimers()
    if (originalTz === undefined) delete env.TZ
    else env.TZ = originalTz
  })

  for (const zone of ['UTC', 'Asia/Ho_Chi_Minh', 'America/New_York']) {
    it(`list and export submit the preset's Vietnam dates, date_field and filters (browser zone ${zone})`, async () => {
      env.TZ = zone
      vi.useFakeTimers({ toFake: ['Date'] })
      vi.setSystemTime(new Date('2026-10-01T17:30:00.000Z')) // 00:30 VN on 2026-10-02
      const w = await mountResults()
      // Results keeps "all" (no dates) as its default.
      expect(lastListParams().from).toBeUndefined()
      expect(lastListParams().to).toBeUndefined()

      await openDateMenu(w)
      await clickPreset(viMessages.today)
      await waitParams({ from: '2026-10-02', to: '2026-10-02', date_field: 'conv' })
      await clickPreset(viMessages.results_preset_7days)
      await waitParams({ from: '2026-09-26', to: '2026-10-02' }) // seven dates including today
      await clickPreset(viMessages.results_preset_28days)
      await waitParams({ from: '2026-09-05', to: '2026-10-02' }) // twenty-eight dates
      await clickPreset(viMessages.results_preset_month)
      await waitParams({ from: '2026-10-01', to: '2026-10-02' })

      // The export submits exactly the selected dates plus date_field and the other filters.
      await w.find('[data-testid="export-csv"]').trigger('click')
      for (let i = 0; i < 3; i++) await flushPromises()
      const exportCall = apiGet.mock.calls.find((c) => String(c[0]).endsWith('/results/export'))!
      expect((exportCall[1] as { params: Record<string, string> }).params).toMatchObject({
        from: '2026-10-01', to: '2026-10-02', date_field: 'conv', job_type: 'qc_analysis', verdict: 'all', format: 'csv',
      })
    })
  }

  it('shows the Vietnam time label next to the date inputs', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date('2026-10-01T17:30:00.000Z'))
    const w = await mountResults()
    await openDateMenu(w)
    const note = document.body.querySelector('[data-testid="vn-date-note"]')
    expect(note?.textContent).toBe(viMessages.vn_date_note)
  })
})
