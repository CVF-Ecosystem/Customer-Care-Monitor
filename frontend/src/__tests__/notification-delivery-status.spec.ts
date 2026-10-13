// R091: status labels expose ambiguous in-flight outcomes with a neutral visual treatment.
// @vitest-environment happy-dom
import { afterEach, describe, expect, it, vi } from 'vitest'
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
vi.mock('../api', () => ({ default: { get: (...args: unknown[]) => apiGet(...args) } }))

import NotificationLogs from '../views/NotificationLogs.vue'

const mounted: VueWrapper[] = []

describe('notification delivery status (R091)', () => {
  afterEach(() => {
    mounted.forEach((wrapper) => wrapper.unmount())
    mounted.length = 0
    document.body.innerHTML = ''
  })

  it('labels sending and unknown distinctly while keeping them neutral', async () => {
    apiGet.mockReset().mockResolvedValue({ data: { total: 3, data: [
      { id: 'r-sending', channel_type: 'email', recipient: 'a@example.invalid', status: 'sending', sent_at: '2026-10-13T00:00:00Z', body: 'synthetic' },
      { id: 'r-unknown', channel_type: 'telegram', recipient: 'synthetic', status: 'unknown', sent_at: '2026-10-13T00:00:00Z', body: 'synthetic' },
      { id: 'r-failed', channel_type: 'email', recipient: 'b@example.invalid', status: 'failed', sent_at: '2026-10-13T00:00:00Z', body: 'synthetic' },
    ] } })
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:tenantId/notifications', component: NotificationLogs }] })
    await router.push('/t1/notifications')
    await router.isReady()
    const wrapper = mount(NotificationLogs, { attachTo: document.body, global: { plugins: [
      createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } }),
      createI18n({ legacy: false, locale: 'en', fallbackLocale: 'vi', messages: { vi: viMessages, en: enMessages } }),
      router,
    ] } })
    mounted.push(wrapper)
    await flushPromises()
    const rows = wrapper.findAll('[data-testid="nl-row"]')
    expect(rows).toHaveLength(3)
    expect(rows[0].get('[data-testid="nl-status"]').text()).toBe('Sending')
    expect(rows[0].get('[data-testid="nl-status"]').classes()).toContain('nl-chip--neutral')
    expect(rows[1].get('[data-testid="nl-status"]').text()).toBe('Unknown outcome')
    expect(rows[1].get('[data-testid="nl-status"]').classes()).toContain('nl-chip--neutral')
    expect(rows[2].get('[data-testid="nl-status"]').text()).toBe('Failed')
    expect(rows[2].get('[data-testid="nl-status"]').classes()).toContain('nl-chip--bad')
  })
})
