// @vitest-environment happy-dom
// CCMAI-RUNTIME-026 (F04): the real Cost Logs screen keeps empty/manual date defaults, carries
// date-only strings unchanged to the API (no browser-zone re-serialization) and labels the dates
// as Vietnam time. Mocked API: request-contract evidence only.
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
vi.mock('../api', () => ({ default: { get: (...a: unknown[]) => apiGet(...a), post: vi.fn(), put: vi.fn(), delete: vi.fn() } }))

import CostLogs from '../views/CostLogs.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

// Mounting Vuetify screens and stepping through several presets can exceed the 5 s default when
// many spec files start at once; the cases below are not timing-sensitive otherwise.
vi.setConfig({ testTimeout: 30_000 })
// The frontend has no Node typings; the test runner's process.env.TZ is reached through globalThis.
const env = (globalThis as unknown as { process: { env: Record<string, string | undefined> } }).process.env
const originalTz = env.TZ
const mounted: VueWrapper[] = []

async function mountCostLogs() {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:tenantId/cost-logs', component: CostLogs }] })
  await router.push('/t1/cost-logs')
  await router.isReady()
  const w = mount(CostLogs, {
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

const lastParams = () => {
  const calls = apiGet.mock.calls.filter((c) => String(c[0]).endsWith('/cost-logs'))
  return (calls[calls.length - 1][1] as { params: Record<string, unknown> }).params
}

describe('Cost Logs date controls on the Vietnam calendar (F04)', () => {
  beforeEach(() => {
    apiGet.mockReset()
    apiGet.mockImplementation(() => Promise.resolve({ data: { data: [], total: 0, exchange_rate: 26000 } }))
  })
  afterEach(() => {
    mounted.forEach((w) => w.unmount())
    mounted.length = 0
    document.body.innerHTML = ''
    vi.useRealTimers()
    if (originalTz === undefined) delete env.TZ
    else env.TZ = originalTz
  })

  it('has empty date defaults and labels the controls as Vietnam time', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date('2026-10-01T17:30:00.000Z'))
    env.TZ = 'America/New_York'
    const w = await mountCostLogs()
    expect(lastParams().from).toBeUndefined()
    expect(lastParams().to).toBeUndefined()
    expect(w.find('[data-testid="vn-date-note"]').text()).toBe(viMessages.vn_date_note)
    const inputs = Array.from(document.body.querySelectorAll('input[type="date"]')) as HTMLInputElement[]
    expect(inputs.map((i) => i.value)).toEqual(['', ''])
  })

  for (const zone of ['UTC', 'Asia/Ho_Chi_Minh', 'America/New_York']) {
    it(`a chosen pair reaches the API as the same date-only strings (browser zone ${zone})`, async () => {
      env.TZ = zone
      const w = await mountCostLogs()
      const inputs = Array.from(document.body.querySelectorAll('input[type="date"]')) as HTMLInputElement[]
      inputs[0].value = '2026-10-02'
      inputs[0].dispatchEvent(new Event('input'))
      await flushPromises()
      inputs[1].value = '2026-10-02'
      inputs[1].dispatchEvent(new Event('input'))
      await flushPromises()
      expect(lastParams()).toMatchObject({ from: '2026-10-02', to: '2026-10-02' })
      expect(w.exists()).toBe(true)
    })
  }
})
