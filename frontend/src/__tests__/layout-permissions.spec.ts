// @vitest-environment happy-dom
// R090: mounted navigation permission contract with synthetic tenant membership responses.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { createI18n } from 'vue-i18n'
import { createPinia, type Pinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import viMessages from '../i18n/vi'
import enMessages from '../i18n/en'
import { lightColors } from '../styles/tokens'
import { useAuthStore } from '../stores/auth'

const apiGet = vi.fn()
const apiPut = vi.fn()
const apiPost = vi.fn()
vi.mock('../api', () => ({
  default: {
    get: (...args: unknown[]) => apiGet(...args),
    put: (...args: unknown[]) => apiPut(...args),
    post: (...args: unknown[]) => apiPost(...args),
  },
}))

import DefaultLayout from '../layouts/DefaultLayout.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

const mounted: VueWrapper[] = []
let permissions = '{"jobs":"r"}'
let holdStaleTenantPermissions = false
let holdFirstTenantPermissions = false
let resolveStaleTenant!: (value: { data: { role: string; permissions: string } }) => void

async function mountLayout(role = 'member', pinia: Pinia = createPinia()): Promise<{ wrapper: VueWrapper; router: ReturnType<typeof createRouter>; auth: ReturnType<typeof useAuthStore> }> {
  const auth = useAuthStore(pinia)
  auth.user = { id: 'r090-layout-user', email: 'layout@example.invalid', name: 'R090', is_admin: false, language: 'vi' }
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:tenantId', component: DefaultLayout }] })
  await router.push('/t1')
  await router.isReady()
  apiGet.mockImplementation((url: string) => {
    if (url.endsWith('/me')) {
      if (url.includes('/t1/') && holdFirstTenantPermissions) {
        holdFirstTenantPermissions = false
        return new Promise(resolve => { resolveStaleTenant = resolve })
      }
      if (url.includes('/t2/')) {
        if (holdStaleTenantPermissions) return new Promise(resolve => { resolveStaleTenant = resolve })
        return Promise.resolve({ data: { role: 'member', permissions: '{"jobs":"r"}' } })
      }
      if (url.includes('/t3/')) return Promise.resolve({ data: { role: 'member', permissions: '{"channels":"r"}' } })
      return Promise.resolve({ data: { role, permissions } })
    }
    if (url === '/tenants') return Promise.resolve({ data: [{ id: 't1', name: 'Synthetic', slug: 'synthetic' }] })
    return Promise.resolve({ data: {} })
  })
  const wrapper = mount({
    components: { DefaultLayout },
    template: '<v-app><DefaultLayout /></v-app>',
  }, {
    attachTo: document.body,
    global: {
      plugins: [
        createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } }),
        createI18n({ legacy: false, locale: 'vi', fallbackLocale: 'en', messages: { vi: viMessages, en: enMessages } }),
        pinia,
        router,
      ],
      stubs: { LanguageSwitcher: true, OnboardingWizard: true },
    },
  })
  mounted.push(wrapper)
  await flushPromises()
  await flushPromises()
  return { wrapper, router, auth }
}

describe('DefaultLayout notification-log navigation permission (R090)', () => {
  beforeEach(() => {
    apiGet.mockReset(); apiPut.mockReset(); apiPost.mockReset()
    permissions = '{"jobs":"r"}'
    holdStaleTenantPermissions = false
    holdFirstTenantPermissions = false
  })
  afterEach(() => {
    while (mounted.length) mounted.pop()!.unmount()
    document.body.innerHTML = ''
  })

  it('hides notification logs from a default jobs:r member', async () => {
    const { wrapper } = await mountLayout('member')
    expect(wrapper.text()).toContain(viMessages.nav_jobs)
    expect(wrapper.text()).not.toContain(viMessages.nav_notification_logs)
  })

  it('shows notification logs to settings:r members and owner/admin', async () => {
    permissions = '{"settings":"r"}'
    const { wrapper: settings } = await mountLayout('member')
    expect(settings.text()).toContain(viMessages.nav_notification_logs)
    settings.unmount(); mounted.pop()

    for (const role of ['owner', 'admin']) {
      const { wrapper: ownerOrAdmin } = await mountLayout(role)
      expect(ownerOrAdmin.text()).toContain(viMessages.nav_notification_logs)
      ownerOrAdmin.unmount(); mounted.pop()
    }
  })

  it('fails closed for unknown roles and malformed permission values', async () => {
    permissions = '{"settings":{"read":true}}'
    const { wrapper: malformed, auth: malformedAuth } = await mountLayout('member')
    expect(malformed.text()).not.toContain(viMessages.nav_notification_logs)
    expect(malformedAuth.tenantPerms).toEqual({ role: 'member', permissions: {} })
    malformed.unmount(); mounted.pop()

    permissions = '{"settings":"r"}'
    const { wrapper: unknown, auth: unknownAuth } = await mountLayout('unknown')
    expect(unknown.text()).not.toContain(viMessages.nav_notification_logs)
    expect(unknownAuth.tenantPerms.role).toBe('')
    expect(unknownAuth.tenantPerms.permissions).toEqual({})
    expect(unknownAuth.canView('settings')).toBe(false)
    expect(unknownAuth.canEdit('settings')).toBe(false)
  })

  it('ignores a late old-tenant owner permission response after the current channels:r member resolves', async () => {
    permissions = '{"channels":"r"}'
    const { wrapper, router, auth } = await mountLayout('member')
    expect(wrapper.text()).not.toContain(viMessages.nav_notification_logs)

    holdStaleTenantPermissions = true
    await router.push('/t2')
    await flushPromises()
    await router.push('/t3')
    await flushPromises()
    await flushPromises()
    expect(wrapper.text()).toContain(viMessages.nav_channels)
    expect(wrapper.text()).not.toContain(viMessages.nav_notification_logs)

    resolveStaleTenant({ data: { role: 'owner', permissions: '{}' } })
    await flushPromises()
    await flushPromises()
    expect(wrapper.text()).toContain(viMessages.nav_channels)
    expect(wrapper.text()).not.toContain(viMessages.nav_notification_logs)
    expect(auth.tenantPerms).toEqual({ role: 'member', permissions: { channels: 'r' } })
  })

  it('invalidates a delayed same-tenant permission response after unmount and remount', async () => {
    const sharedPinia = createPinia()
    holdFirstTenantPermissions = true
    const { wrapper: oldLayout, auth } = await mountLayout('member', sharedPinia)
    oldLayout.unmount(); mounted.pop()

    permissions = '{"channels":"r"}'
    const { wrapper: currentLayout } = await mountLayout('member', sharedPinia)
    expect(auth.tenantPerms).toEqual({ role: 'member', permissions: { channels: 'r' } })

    resolveStaleTenant({ data: { role: 'owner', permissions: '{"settings":"rw"}' } })
    await flushPromises()
    await flushPromises()
    expect(currentLayout.text()).not.toContain(viMessages.nav_notification_logs)
    expect(auth.tenantPerms).toEqual({ role: 'member', permissions: { channels: 'r' } })
    expect(auth.canView('settings')).toBe(false)
    expect(auth.canEdit('settings')).toBe(false)
  })
})
