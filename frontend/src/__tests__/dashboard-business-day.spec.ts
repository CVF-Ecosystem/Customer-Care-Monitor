// @vitest-environment happy-dom
// CCMAI-RUNTIME-026 (F04): the real Dashboard component captures the date query it sends. The
// mocked API proves only the request contract; the same named cases drive the backend handler
// fixtures, so this is linked UI/API/DB contract evidence, not a browser-to-server run.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { createI18n } from 'vue-i18n'
import { createRouter, createMemoryHistory } from 'vue-router'
import viMessages from '../i18n/vi'
import enMessages from '../i18n/en'
import { lightColors } from '../styles/tokens'

const apiGet = vi.fn()
vi.mock('../api', () => ({ default: { get: (...a: unknown[]) => apiGet(...a) } }))

import Dashboard from '../views/Dashboard.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

// Mounting Vuetify screens and stepping through several presets can exceed the 5 s default when
// many spec files start at once; the cases below are not timing-sensitive otherwise.
vi.setConfig({ testTimeout: 30_000 })
// The frontend has no Node typings; the test runner's process.env.TZ is reached through globalThis.
const env = (globalThis as unknown as { process: { env: Record<string, string | undefined> } }).process.env
const originalTz = env.TZ
const mounted: VueWrapper[] = []

async function mountDashboard(locale: 'vi' | 'en' = 'vi') {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:tenantId/dashboard', component: Dashboard }] })
  await router.push('/t1/dashboard')
  await router.isReady()
  const w = mount(Dashboard, {
    attachTo: document.body,
    global: { plugins: [
      createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } }),
      createI18n({ legacy: false, locale, fallbackLocale: 'en', messages: { vi: viMessages, en: enMessages } }),
      router,
    ] },
  })
  mounted.push(w)
  await flushPromises()
  await flushPromises()
  return w
}

const dashboardParams = () => {
  const calls = apiGet.mock.calls.filter((c) => String(c[0]).endsWith('/dashboard'))
  return (calls[calls.length - 1][1] as { params: Record<string, string> }).params
}

async function clickPreset(w: VueWrapper, label: string) {
  const chip = w.findAll('.v-chip').find((c) => c.text() === label)
  if (!chip) throw new Error(`preset ${label} missing`)
  await chip.trigger('click')
  await flushPromises()
}

describe('Dashboard date controls on the Vietnam calendar (F04)', () => {
  beforeEach(() => {
    apiGet.mockReset()
    apiGet.mockImplementation((url: string) => url.endsWith('/dashboard')
      ? Promise.resolve({ data: { total_conversations: 0, active_channels: 0, active_jobs: 0, issues: 0, qc_violation_count: 0, qc_alerts: [], classification_recent: [], cost_by_day: [], messages_by_day: [], conversations_by_channel: [] } })
      : Promise.resolve({ data: { has_data: true, is_demo: false } }))
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
    it(`default is exactly 28 Vietnam dates ending today and presets follow the same calendar (browser zone ${zone})`, async () => {
      env.TZ = zone
      vi.useFakeTimers({ toFake: ['Date'] })
      vi.setSystemTime(new Date('2026-10-01T17:30:00.000Z')) // 00:30 VN on 2026-10-02
      const w = await mountDashboard()
      expect(dashboardParams()).toEqual({ from: '2026-09-05', to: '2026-10-02' }) // 28 dates including today

      await clickPreset(w, viMessages.today)
      expect(dashboardParams()).toEqual({ from: '2026-10-02', to: '2026-10-02' })
      await clickPreset(w, '7 ngày')
      expect(dashboardParams()).toEqual({ from: '2026-09-26', to: '2026-10-02' }) // seven dates including today
      await clickPreset(w, '28 ngày')
      expect(dashboardParams()).toEqual({ from: '2026-09-05', to: '2026-10-02' })
      await clickPreset(w, 'Tháng này')
      expect(dashboardParams()).toEqual({ from: '2026-10-01', to: '2026-10-02' })
      await clickPreset(w, 'Quý này')
      expect(dashboardParams()).toEqual({ from: '2026-10-01', to: '2026-10-02' })
      await clickPreset(w, 'Năm này')
      expect(dashboardParams()).toEqual({ from: '2026-01-01', to: '2026-10-02' })
    })
  }

  it('at one millisecond before Vietnam midnight it is still the previous Vietnam day', async () => {
    env.TZ = 'Pacific/Auckland'
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date('2026-10-01T16:59:59.999Z'))
    const w = await mountDashboard()
    expect(dashboardParams()).toEqual({ from: '2026-09-04', to: '2026-10-01' })
    await clickPreset(w, viMessages.today)
    expect(dashboardParams()).toEqual({ from: '2026-10-01', to: '2026-10-01' })
  })

  it('shows the Vietnam time label next to the date controls in both languages', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date('2026-10-01T17:30:00.000Z'))
    const vi_ = await mountDashboard('vi')
    expect(vi_.find('[data-testid="vn-date-note"]').text()).toBe(viMessages.vn_date_note)
    expect(viMessages.vn_date_note).toContain('UTC+7')
    const en = await mountDashboard('en')
    expect(en.findAll('[data-testid="vn-date-note"]').some((n) => n.text() === enMessages.vn_date_note)).toBe(true)
  })
})
