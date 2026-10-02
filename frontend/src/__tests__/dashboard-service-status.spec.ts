// @vitest-environment happy-dom
// CCMAI-RUNTIME-029 (F07): the real Dashboard never shows measured service health. The Service Status
// card keeps the API Server / Database / Scheduler rows as an explicit, neutral "no health-check
// data" state whatever the Dashboard request does. Synthetic API responses verify UI rendering only;
// nothing here measures a real API, database or scheduler and nothing is governance evidence.
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
const apiPost = vi.fn()
vi.mock('../api', () => ({ default: { get: (...a: unknown[]) => apiGet(...a), post: (...a: unknown[]) => apiPost(...a) } }))

import Dashboard from '../views/Dashboard.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }
vi.setConfig({ testTimeout: 30_000 })

const SERVICES = ['API Server', 'Database', 'Scheduler']
const mounted: VueWrapper[] = []
let dashboardResponse: () => Promise<{ data: Record<string, unknown> }>

const payload = (extra: Record<string, unknown> = {}) => ({
  data: { total_conversations: 4242, active_channels: 2, active_jobs: 3, issues: 1, qc_violation_count: 1, qc_alerts: [], classification_recent: [], cost_by_day: [], messages_by_day: [], conversations_by_channel: [], ...extra },
})

async function mountDashboard(locale: 'vi' | 'en' = 'vi', settle = true) {
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
  if (settle) await flushPromises()
  return w
}

// The Service Status card, located by its (existing) title so the same lookup works on old source.
function serviceCard(w: VueWrapper, locale: 'vi' | 'en' = 'vi') {
  const title = (locale === 'vi' ? viMessages : enMessages).service_status
  const card = w.findAll('.v-card').find((c) => c.text().includes(title))
  if (!card) throw new Error('Service Status card missing')
  return card
}

const UNKNOWN = { vi: 'Chưa có dữ liệu kiểm tra', en: 'No health-check data' }
const NOTE = { vi: 'Chưa có phép kiểm tra sức khỏe cho các dịch vụ này.', en: 'Health checks are not available for these services.' }

// All three rows are an explicit unknown state, neutral in text, color and icon.
function expectUnknownAndNeutral(w: VueWrapper, locale: 'vi' | 'en' = 'vi') {
  const card = serviceCard(w, locale)
  const text = card.text()
  // names and order persist
  let last = -1
  for (const name of SERVICES) {
    const at = text.indexOf(name)
    expect(at, `${name} present`).toBeGreaterThan(-1)
    expect(at, `${name} order`).toBeGreaterThan(last)
    last = at
  }
  // exactly three unknown chips and one explanation, with visible text (not color or a tooltip)
  expect((text.match(new RegExp(UNKNOWN[locale], 'g')) ?? []).length).toBe(3)
  expect((text.match(new RegExp(NOTE[locale].replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g')) ?? []).length).toBe(1)
  // no healthy / error wording for the rows
  const other = locale === 'vi' ? enMessages : viMessages
  for (const word of [viMessages.normal, viMessages.error, enMessages.normal, enMessages.error, other.normal]) {
    expect(text.includes(word) && word.length > 0 ? word : '', `status word "${word}" must not appear`).toBe('')
  }
  // neutral styling: no success/error/warning utility classes and no check/outage icon
  const html = card.html()
  expect(html).not.toMatch(/(text|bg)-(success|error|warning)/)
  expect(html).not.toContain('mdi-check-circle')
  expect(html).not.toContain('mdi-alert')
  expect(html).not.toContain('mdi-close-circle')
  expect(html).toContain('mdi-help-circle-outline')
}

describe('Dashboard service status is an explicit unknown (F07)', () => {
  beforeEach(() => {
    apiGet.mockReset()
    apiPost.mockReset()
    dashboardResponse = () => Promise.resolve(payload())
    apiGet.mockImplementation((url: string) => url.endsWith('/dashboard') ? dashboardResponse() : Promise.resolve({ data: { has_data: true, is_demo: false } }))
  })
  afterEach(() => {
    mounted.forEach((w) => w.unmount())
    mounted.length = 0
    document.body.innerHTML = ''
    vi.useRealTimers()
  })

  // A: initial and pending
  it('A: shows the unknown state at first render and while the Dashboard request is pending', async () => {
    let resolve!: (v: { data: Record<string, unknown> }) => void
    dashboardResponse = () => new Promise((r) => { resolve = r })
    const w = await mountDashboard('vi', false)
    expectUnknownAndNeutral(w) // before anything settles: no green frame
    await flushPromises()
    expectUnknownAndNeutral(w) // still pending
    resolve(payload())
    await flushPromises()
    expectUnknownAndNeutral(w)
  })

  // B: ordinary responses
  for (const [name, extra] of [
    ['populated with positive job and channel counts', { active_jobs: 5, active_channels: 3 }],
    ['populated with zero job and channel counts', { active_jobs: 0, active_channels: 0 }],
    ['empty', { total_conversations: 0, active_jobs: 0, active_channels: 0, issues: 0, qc_violation_count: 0 }],
  ] as const) {
    it(`B: stays unknown for a ${name} response and still updates the metrics`, async () => {
      dashboardResponse = () => Promise.resolve(payload(extra))
      const w = await mountDashboard()
      expectUnknownAndNeutral(w)
      if (name.startsWith('populated with positive')) expect(w.text()).toContain('4242') // existing metrics unchanged
    })
  }

  // C: failure and repeat
  for (const [name, make] of [
    ['network error', () => Promise.reject(new Error('Network Error'))],
    ['timeout', () => Promise.reject(Object.assign(new Error('timeout of 10000ms exceeded'), { code: 'ECONNABORTED' }))],
    ['HTTP 500', () => Promise.reject({ response: { status: 500, data: { error: 'internal_error' } } })],
    ['HTTP 503 (database-failure shaped)', () => Promise.reject({ response: { status: 503, data: { error: 'database_unavailable' } } })],
  ] as const) {
    it(`C: stays unknown after a ${name}`, async () => {
      dashboardResponse = make as () => Promise<{ data: Record<string, unknown> }>
      const w = await mountDashboard()
      expectUnknownAndNeutral(w)
    })
  }

  it('C: success then failure then success through the existing refresh never changes the rows', async () => {
    const w = await mountDashboard()
    expectUnknownAndNeutral(w)
    const dateInput = w.find('input[type="date"]')
    expect(dateInput.exists()).toBe(true)
    for (const next of [
      () => Promise.reject({ response: { status: 503, data: { error: 'database_unavailable' } } }),
      () => Promise.resolve(payload({ active_jobs: 0 })),
      () => Promise.reject(new Error('Network Error')),
      () => Promise.resolve(payload()),
    ]) {
      dashboardResponse = next as () => Promise<{ data: Record<string, unknown> }>
      await dateInput.setValue('2026-03-10')
      await flushPromises()
      expectUnknownAndNeutral(w)
      await dateInput.setValue('2026-03-11')
      await flushPromises()
      expectUnknownAndNeutral(w)
    }
  })

  // D: untrusted hints, scheduler metrics and the request boundary
  it('D: ignores incidental health-like payload fields and scheduler metrics, and makes no health call', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'setInterval', 'clearTimeout', 'clearInterval'] })
    for (const hint of [
      { health: 'down', services: { api: 'ok', database: 'down', scheduler: 'down' }, scheduler_status: 'failed' },
      { health: 'ok', services: { api: 'ok', database: 'ok', scheduler: 'ok' }, scheduler_status: 'active', active_jobs: 9 },
      { scheduler_status: 'idle', active_jobs: 0, active_channels: 0 },
      { status: 'healthy', database: { ok: true }, scheduler: { running: true } },
    ]) {
      dashboardResponse = () => Promise.resolve(payload(hint))
      const w = await mountDashboard()
      expectUnknownAndNeutral(w)
      w.unmount()
      mounted.pop()
    }
    // let any existing timer run for a long time: the request inventory stays Dashboard/demo only
    const w = await mountDashboard()
    await vi.advanceTimersByTimeAsync(10 * 60 * 1000)
    expectUnknownAndNeutral(w)
    const urls = apiGet.mock.calls.map((c) => String(c[0]))
    expect(urls.length).toBeGreaterThan(0)
    for (const u of urls) expect(u, `unexpected request ${u}`).toMatch(/\/tenants\/t1\/(dashboard|demo\/status)$/)
    expect(urls.some((u) => /health/i.test(u))).toBe(false)
    expect(apiPost.mock.calls.length).toBe(0)
  })

  // E: locale and appearance
  for (const locale of ['vi', 'en'] as const) {
    it(`E: ${locale} chip and explanation resolve without raw keys`, async () => {
      const w = await mountDashboard(locale)
      expectUnknownAndNeutral(w, locale)
      const text = serviceCard(w, locale).text()
      expect(text).not.toMatch(/service_health|service_status/)
      expect(text).toContain((locale === 'vi' ? viMessages : enMessages).service_status)
    })
  }

  it('E: the unknown chips and heading carry neutral computed Vuetify classes', async () => {
    const w = await mountDashboard()
    const card = serviceCard(w)
    const chips = card.findAll('.v-chip')
    expect(chips).toHaveLength(3)
    for (const chip of chips) {
      expect(chip.classes().join(' ')).not.toMatch(/text-(success|error|warning)|bg-(success|error|warning)/)
      expect(chip.find('.mdi-help-circle-outline').exists()).toBe(true)
    }
    const heading = card.find('.text-subtitle-1')
    expect(heading.find('.mdi-help-circle-outline').exists()).toBe(true)
    expect(heading.html()).not.toMatch(/text-(success|error)/)
  })
})
