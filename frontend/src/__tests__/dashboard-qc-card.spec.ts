// @vitest-environment happy-dom
// CCMAI-UX-016: synthetic API responses verify UI contract rendering only.
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

const mounted: VueWrapper[] = []
let dashboardResponse: () => Promise<{ data: Record<string, unknown> }>

function response(qc: unknown, issues = 91) {
  return { data: { total_conversations: 5, active_channels: 2, active_jobs: 1, issues, qc_violation_count: qc, qc_alerts: [], classification_recent: [], cost_by_day: [], messages_by_day: [], conversations_by_channel: [] } }
}

async function mountDashboard() {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:tenantId/dashboard', component: Dashboard }] })
  await router.push('/t1/dashboard')
  await router.isReady()
  const w = mount(Dashboard, {
    attachTo: document.body,
    global: { plugins: [
      createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } }),
      createI18n({ legacy: false, locale: 'vi', fallbackLocale: 'en', messages: { vi: viMessages, en: enMessages } }),
      router,
    ] },
  })
  mounted.push(w)
  await flushPromises()
  return w
}

function qcCard(w: VueWrapper) {
  const card = w.findAll('.v-card').find(c => c.text().includes('Vi phạm QC'))
  if (!card) throw new Error('QC card missing')
  return card
}

describe('Dashboard QC card (UX-016)', () => {
  beforeEach(() => {
    apiGet.mockReset()
    dashboardResponse = () => Promise.resolve(response(4))
    apiGet.mockImplementation((url: string) => url.endsWith('/dashboard') ? dashboardResponse() : Promise.resolve({ data: { has_data: true, is_demo: false } }))
  })
  afterEach(() => {
    mounted.forEach(w => w.unmount())
    mounted.length = 0
    document.body.innerHTML = ''
  })

  it('shows reviewed QC rows rather than the legacy all-results total', async () => {
    const w = await mountDashboard()
    expect(qcCard(w).text()).toContain('4')
    expect(qcCard(w).text()).not.toContain('91')
    expect(qcCard(w).html()).toContain('Số dòng vi phạm QC')
  })

  it('shows a real zero, but no invented zero or issues fallback for missing and invalid values', async () => {
    for (const [value, shown] of [[0, '0'], [undefined, '—'], ['4', '—'], [-1, '—'], [1.5, '—']] as const) {
      dashboardResponse = () => Promise.resolve(response(value))
      const w = await mountDashboard()
      expect(qcCard(w).find('.text-h5').text()).toBe(shown)
      w.unmount()
      mounted.pop()
    }
  })

  it('clears the old interval while refreshing and stays unavailable after request failure', async () => {
    const w = await mountDashboard()
    expect(qcCard(w).find('.text-h5').text()).toBe('4')
    let rejectRequest!: (reason: Error) => void
    dashboardResponse = () => new Promise((_resolve, reject) => { rejectRequest = reject })
    const dateInput = w.find('input[type="date"]')
    expect(dateInput.exists()).toBe(true)
    await dateInput.setValue('2026-03-10')
    expect(qcCard(w).find('.text-h5').text()).toBe('—')
    rejectRequest(new Error('temporary API failure'))
    await flushPromises()
    expect(qcCard(w).find('.text-h5').text()).toBe('—')
  })

  it('keeps the newer date selection when an older response resolves last', async () => {
    const w = await mountDashboard()
    expect(qcCard(w).find('.text-h5').text()).toBe('4')
    const pending: Array<{ params: Record<string, string>, resolve: (v: { data: Record<string, unknown> }) => void }> = []
    apiGet.mockImplementation((url: string, config?: { params?: Record<string, string> }) => {
      if (!url.endsWith('/dashboard')) return Promise.resolve({ data: { has_data: true, is_demo: false } })
      return new Promise(resolve => { pending.push({ params: config?.params ?? {}, resolve }) })
    })
    const dateInput = w.find('input[type="date"]')
    await dateInput.setValue('2026-03-10')
    await dateInput.setValue('2026-03-11')
    expect(pending).toHaveLength(2)
    expect(pending[0].params.from).toBe('2026-03-10')
    expect(pending[1].params.from).toBe('2026-03-11')
    expect(qcCard(w).find('.text-h5').text()).toBe('—')

    pending[1].resolve(response(7, 13))
    await flushPromises()
    expect(qcCard(w).find('.text-h5').text()).toBe('7')

    pending[0].resolve(response(99, 55))
    await flushPromises()
    expect(qcCard(w).find('.text-h5').text()).toBe('7')
    expect(qcCard(w).text()).not.toContain('99')
  })
})
