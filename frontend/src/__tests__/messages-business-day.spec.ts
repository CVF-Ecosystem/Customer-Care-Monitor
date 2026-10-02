// @vitest-environment happy-dom
// CCMAI-RUNTIME-026 (F04): the real Messages screen's export dialog defaults to exactly seven
// Vietnam calendar dates including today and submits date-only strings. Mocked API: request
// contract evidence only.
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

const apiGet = vi.fn()
vi.mock('../api', () => ({ default: { get: (...a: unknown[]) => apiGet(...a), post: vi.fn(), delete: vi.fn(), put: vi.fn() } }))

import Messages from '../views/Messages.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

// Mounting Vuetify screens and stepping through several presets can exceed the 5 s default when
// many spec files start at once; the cases below are not timing-sensitive otherwise.
vi.setConfig({ testTimeout: 30_000 })
// The frontend has no Node typings; the test runner's process.env.TZ is reached through globalThis.
const env = (globalThis as unknown as { process: { env: Record<string, string | undefined> } }).process.env
const originalTz = env.TZ
const mounted: VueWrapper[] = []

async function mountMessages() {
  const pinia = createPinia()
  setActivePinia(pinia)
  useAuthStore().tenantPerms = { role: 'owner', permissions: {} } as never
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:tenantId/messages', component: Messages }, { path: '/:p(.*)*', component: { template: '<div />' } }] })
  await router.push('/t1/messages')
  await router.isReady()
  const w = mount(Messages, {
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
  return w
}

const dateInputs = () => Array.from(document.body.querySelectorAll('input[type="date"]')) as HTMLInputElement[]

describe('Message export dates on the Vietnam calendar (F04)', () => {
  beforeEach(() => {
    apiGet.mockReset()
    apiGet.mockImplementation((url: string) => {
      if (url.endsWith('/channels')) return Promise.resolve({ data: [] })
      if (url.endsWith('/conversations')) return Promise.resolve({ data: { data: [], total: 0 } })
      if (url.endsWith('/conversations/evaluated')) return Promise.resolve({ data: {} })
      if (url.includes('/conversations/export')) return Promise.resolve({ data: 'noi dung xuat' })
      return Promise.resolve({ data: {} })
    })
    URL.createObjectURL = vi.fn(() => 'blob:x')
    URL.revokeObjectURL = vi.fn()
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
    it(`default range is exactly seven Vietnam dates ending today and is submitted as date-only strings (browser zone ${zone})`, async () => {
      env.TZ = zone
      vi.useFakeTimers({ toFake: ['Date'] })
      vi.setSystemTime(new Date('2026-10-01T17:30:00.000Z')) // 00:30 VN on 2026-10-02 (still Oct 1 in UTC/New York)
      const w = await mountMessages()
      await w.find('[data-testid="msgs-export-open"]').trigger('click')
      await flushPromises()
      const [from, to] = dateInputs()
      expect(from.value).toBe('2026-09-26') // seven dates: 26, 27, 28, 29, 30 Sep, 1, 2 Oct
      expect(to.value).toBe('2026-10-02')
      expect(document.body.querySelector('[data-testid="vn-date-note"]')?.textContent).toBe(viMessages.vn_date_note)

      ;(document.body.querySelector('[data-testid="msgs-export-download"]') as HTMLElement).click()
      for (let i = 0; i < 3; i++) await flushPromises()
      const call = apiGet.mock.calls.find((c) => String(c[0]).includes('/conversations/export'))!
      expect(String(call[0])).toContain('from=2026-09-26&to=2026-10-02&format=txt')
    })
  }

  it('a user-chosen date pair is submitted unchanged', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date('2026-10-01T17:30:00.000Z'))
    env.TZ = 'America/New_York'
    const w = await mountMessages()
    await w.find('[data-testid="msgs-export-open"]').trigger('click')
    await flushPromises()
    const [from, to] = dateInputs()
    from.value = '2024-02-29'
    from.dispatchEvent(new Event('input'))
    to.value = '2024-03-01'
    to.dispatchEvent(new Event('input'))
    await flushPromises()
    ;(document.body.querySelector('[data-testid="msgs-export-download"]') as HTMLElement).click()
    for (let i = 0; i < 3; i++) await flushPromises()
    const call = apiGet.mock.calls.find((c) => String(c[0]).includes('/conversations/export'))!
    expect(String(call[0])).toContain('from=2024-02-29&to=2024-03-01')
  })
})
