// @vitest-environment happy-dom
// CCMAI-RUNTIME-027 (F05): the real Job Detail run dialog sends the chosen mode, the count cap and
// date-only Vietnam dates as separate query parameters; the cap never changes the mode and dates
// reach the API as typed (no browser-zone re-serialization). Mocked API: request-contract
// evidence only, not governance evidence.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
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
const apiPost = vi.fn()
vi.mock('../api', () => ({ default: { get: (...a: unknown[]) => apiGet(...a), post: (...a: unknown[]) => apiPost(...a), delete: vi.fn() } }))

import JobDetail from '../views/Jobs/JobDetail.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

vi.setConfig({ testTimeout: 30_000 })
const env = (globalThis as unknown as { process: { env: Record<string, string | undefined> } }).process.env
const originalTz = env.TZ

const job = { id: 'j1', name: 'QC mẫu', job_type: 'qc_analysis', schedule_type: 'manual', schedule_cron: '', input_channel_ids: '["c1"]', last_run_at: null, last_run_status: '' }
const mounted: VueWrapper[] = []

type RunState = { runMode: string; runDateFrom: string; runDateTo: string; runLimit: number | null }

async function mountDialog() {
  apiGet.mockImplementation((url: string) => {
    if (url.endsWith('/jobs/j1')) return Promise.resolve({ data: job })
    if (url.endsWith('/jobs/j1/runs')) return Promise.resolve({ data: [] })
    if (url.endsWith('/jobs/j1/results')) return Promise.resolve({ data: [] })
    if (url.endsWith('/settings')) return Promise.resolve({ data: { settings: { ai_api_key: 'configured' } } })
    return Promise.resolve({ data: {} })
  })
  apiPost.mockResolvedValue({ data: {} })
  const pinia = createPinia()
  setActivePinia(pinia)
  useAuthStore().tenantPerms = { role: 'owner', permissions: {} } as never
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:tenantId/jobs/:jobId', component: JobDetail }, { path: '/:p(.*)*', component: { template: '<div />' } }] })
  await router.push('/t1/jobs/j1')
  await router.isReady()
  const w = mount(JobDetail, {
    attachTo: document.body,
    global: { plugins: [
      createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } }),
      createI18n({ legacy: false, locale: 'vi', fallbackLocale: 'en', messages: { vi: viMessages, en: enMessages } }),
      pinia,
      router,
    ] },
  })
  mounted.push(w)
  for (let i = 0; i < 4; i++) await flushPromises()
  // Open the dialog through the same handler the "Run now" button uses.
  const state = (w.vm as unknown as { $: { setupState: Record<string, unknown> } }).$.setupState
  await (state.openRunDialog as () => Promise<void>)()
  for (let i = 0; i < 3; i++) await flushPromises()
  return { w, state: state as unknown as RunState }
}

async function fill(testId: string, value: string) {
  const input = document.body.querySelector(`[data-testid="${testId}"] input`) as HTMLInputElement
  expect(input, testId).toBeTruthy()
  input.value = value
  input.dispatchEvent(new Event('input'))
  await flushPromises()
}

const confirmBtn = () => document.body.querySelector('[data-testid="run-confirm"]') as HTMLButtonElement
const lastUrl = () => String(apiPost.mock.calls[apiPost.mock.calls.length - 1][0])

describe('Job run dialog request contract (F05)', () => {
  beforeEach(() => {
    apiGet.mockReset()
    apiPost.mockReset()
  })
  afterEach(() => {
    mounted.forEach((w) => w.unmount())
    mounted.length = 0
    document.body.innerHTML = ''
    if (originalTz === undefined) delete env.TZ
    else env.TZ = originalTz
  })

  it('since-last with a cap sends the cap and keeps the mode', async () => {
    const { state } = await mountDialog()
    state.runMode = 'since_last'
    await flushPromises()
    await fill('run-limit', '3')
    confirmBtn().click()
    await flushPromises()
    expect(lastUrl()).toBe('/tenants/t1/jobs/j1/trigger?mode=since_last&limit=3')
  })

  it('unanalyzed without a cap sends only the mode', async () => {
    const { state } = await mountDialog()
    state.runMode = 'unanalyzed'
    await flushPromises()
    confirmBtn().click()
    await flushPromises()
    expect(lastUrl()).toBe('/tenants/t1/jobs/j1/trigger?mode=unanalyzed')
  })

  for (const zone of ['UTC', 'Asia/Ho_Chi_Minh', 'America/New_York']) {
    it(`conditional dates reach the API as the typed date-only strings (browser zone ${zone})`, async () => {
      env.TZ = zone
      const { state } = await mountDialog()
      state.runMode = 'conditional'
      await flushPromises()
      await fill('run-date-from', '2026-10-02')
      await fill('run-date-to', '2026-10-02')
      confirmBtn().click()
      await flushPromises()
      expect(lastUrl()).toBe('/tenants/t1/jobs/j1/trigger?mode=conditional&from=2026-10-02&to=2026-10-02')
    })
  }

  it('conditional with dates and a cap sends all three', async () => {
    const { state } = await mountDialog()
    state.runMode = 'conditional'
    await flushPromises()
    await fill('run-date-from', '2026-10-01')
    await fill('run-date-to', '2026-10-02')
    await fill('run-limit', '5')
    confirmBtn().click()
    await flushPromises()
    expect(lastUrl()).toBe('/tenants/t1/jobs/j1/trigger?mode=conditional&from=2026-10-01&to=2026-10-02&limit=5')
  })

  it('dates left over from the conditional mode are not sent in another mode', async () => {
    const { state } = await mountDialog()
    state.runMode = 'conditional'
    await flushPromises()
    await fill('run-date-from', '2026-10-01')
    await fill('run-date-to', '2026-10-02')
    state.runMode = 'unanalyzed'
    await flushPromises()
    confirmBtn().click()
    await flushPromises()
    expect(lastUrl()).toBe('/tenants/t1/jobs/j1/trigger?mode=unanalyzed')
  })

  for (const endpoint of ['from', 'to']) {
    it(`independent review: permits ${endpoint}-only conditional range`, async () => {
      const { state } = await mountDialog()
      state.runMode = 'conditional'
      await flushPromises()
      await fill(`run-date-${endpoint}`, '2026-10-02')
      expect(confirmBtn().disabled, 'SPEC permits an open endpoint').toBe(false)
      confirmBtn().click()
      await flushPromises()
      expect(lastUrl()).toBe(`/tenants/t1/jobs/j1/trigger?mode=conditional&${endpoint}=2026-10-02`)
    })
  }

  it('blocks invalid input without any request', async () => {
    const { state } = await mountDialog()
    state.runMode = 'since_last'
    await flushPromises()
    for (const bad of ['1.5', '0', '-2']) {
      await fill('run-limit', bad)
      expect(confirmBtn().disabled, `limit ${bad}`).toBe(true)
      confirmBtn().click()
      await flushPromises()
    }
    state.runMode = 'conditional'
    await fill('run-limit', '')
    await flushPromises()
    expect(confirmBtn().disabled, 'conditional without any condition').toBe(true)
    await fill('run-date-from', '2026-10-03')
    await fill('run-date-to', '2026-10-02')
    expect(confirmBtn().disabled, 'reversed dates').toBe(true)
    confirmBtn().click()
    await flushPromises()
    expect(apiPost).not.toHaveBeenCalled()
  })
})
