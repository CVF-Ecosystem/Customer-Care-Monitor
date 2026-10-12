// @vitest-environment happy-dom
// R090: mounted Dashboard permission projection and tenant-change clearing with a synthetic API.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory, type Router } from 'vue-router'
import viMessages from '../i18n/vi'
import enMessages from '../i18n/en'
import { lightColors } from '../styles/tokens'
import { useAuthStore } from '../stores/auth'

const apiGet = vi.fn()
const apiPost = vi.fn()
const apiDelete = vi.fn()
vi.mock('../api', () => ({
  default: {
    get: (...args: unknown[]) => apiGet(...args),
    post: (...args: unknown[]) => apiPost(...args),
    delete: (...args: unknown[]) => apiDelete(...args),
  },
}))

import Dashboard from '../views/Dashboard.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

const mounted: VueWrapper[] = []

function fullDashboard() {
  return { data: {
    total_conversations: 5,
    active_channels: 2,
    active_jobs: 1,
    issues: 91,
    qc_violation_count: 7,
    conversations_by_channel: [{ channel_type: 'pancake', count: 5 }],
    qc_alerts: [{ id: 'qc-r090', evidence: 'R090-JOB-FORBIDDEN-EVIDENCE', rule_name: 'R090-JOB-FORBIDDEN', severity: 'NGHIEM_TRONG', created_at: '2026-10-13T10:00:00Z', conversation_id: 'conv-r090' }],
    classification_recent: [{ id: 'class-r090', rule_name: 'R090-CLASSIFICATION-PRIVATE', customer_name: 'Private', created_at: '2026-10-13T09:00:00Z', conversation_id: 'conv-r090' }],
    cost_period: 87.65,
    cost_today: 12.34,
    cost_this_month: 56.78,
    cost_by_day: [{ date: '2026-10-13', total_cost: 87.65, input_tokens: 9, output_tokens: 7 }],
    exchange_rate: 31415,
    messages_by_day: [{ date: '2026-10-13', count: 99, chat_count: 5, reply_count: 2 }],
  } }
}

let resolveStaleTenant!: (value: { data: { role: string; permissions: string } }) => void
let holdStaleTenantPermissions = false
let firstTenantPermissions = { role: 'owner', permissions: '{}' }

function setupAPI() {
  apiGet.mockImplementation((url: string) => {
    if (url.endsWith('/me')) {
      if (url.includes('/t2/')) {
        if (holdStaleTenantPermissions) return new Promise(resolve => { resolveStaleTenant = resolve })
        return Promise.resolve({ data: { role: 'member', permissions: '{"channels":"r"}' } })
      }
      if (url.includes('/t3/')) return Promise.resolve({ data: { role: 'member', permissions: '{"channels":"r"}' } })
      return Promise.resolve({ data: firstTenantPermissions })
    }
    if (url.endsWith('/dashboard')) return Promise.resolve(fullDashboard())
    if (url.endsWith('/demo/status')) return Promise.resolve({ data: { has_data: true, is_demo: false } })
    return Promise.resolve({ data: {} })
  })
  apiPost.mockResolvedValue({ data: {} })
  apiDelete.mockResolvedValue({ data: {} })
}

async function mountDashboard(): Promise<{ wrapper: VueWrapper; router: Router }> {
  const pinia = createPinia()
  setActivePinia(pinia)
  const auth = useAuthStore(pinia)
  auth.user = { id: 'synthetic', email: 'r090@example.invalid', name: 'R090', is_admin: false, language: 'vi' }
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:tenantId/dashboard', component: Dashboard }] })
  await router.push('/t1/dashboard')
  await router.isReady()
  const wrapper = mount(Dashboard, {
    attachTo: document.body,
    global: { plugins: [
      createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } }),
      createI18n({ legacy: false, locale: 'vi', fallbackLocale: 'en', messages: { vi: viMessages, en: enMessages } }),
      pinia,
      router,
    ] },
  })
  mounted.push(wrapper)
  await flushPromises()
  await flushPromises()
  return { wrapper, router }
}

describe('Dashboard permission projection (R090)', () => {
  beforeEach(() => {
    apiGet.mockReset(); apiPost.mockReset(); apiDelete.mockReset()
    holdStaleTenantPermissions = false
    firstTenantPermissions = { role: 'owner', permissions: '{}' }
    setupAPI()
  })
  afterEach(() => {
    while (mounted.length) mounted.pop()!.unmount()
    document.body.innerHTML = ''
  })

  it('renders only the channel section for channels:r even when the API returns a full payload', async () => {
    firstTenantPermissions = { role: 'member', permissions: '{"channels":"r"}' }
    const { wrapper: member } = await mountDashboard()
    expect(member.text()).toContain('2')
    expect(member.text()).not.toContain('R090-JOB-FORBIDDEN')
    expect(member.text()).not.toContain('R090-CLASSIFICATION-PRIVATE')
    expect(member.text()).not.toContain('99')
    expect(member.text()).not.toContain('31415')
    expect(member.text()).not.toContain('Pancake')
    expect(member.find('[data-testid="dashboard-messages-channel-breakdown"]').exists()).toBe(false)
    expect(member.find('[data-testid="dashboard-settings-section"]').exists()).toBe(false)
    expect(member.find('[data-testid="dashboard-messages-section"]').exists()).toBe(false)
    expect(member.find('[data-testid="dashboard-jobs-section"]').exists()).toBe(false)
  })

  it('shows conversation channel counts to messages:r without exposing channels-only totals', async () => {
    firstTenantPermissions = { role: 'member', permissions: '{"messages":"r"}' }
    const { wrapper } = await mountDashboard()
    expect(wrapper.find('[data-testid="dashboard-messages-channel-breakdown"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Pancake')
    expect(wrapper.text()).toContain('99')
    expect(wrapper.text()).not.toContain('31415')
    expect(wrapper.find('[data-testid="dashboard-settings-section"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="dashboard-jobs-section"]').exists()).toBe(false)
  })

  it('clears old content and ignores a late old-tenant owner response after the current member resolves', async () => {
    const { wrapper, router } = await mountDashboard()
    expect(wrapper.text()).toContain('R090-JOB-FORBIDDEN-EVIDENCE')
    holdStaleTenantPermissions = true
    await router.push('/t2/dashboard')
    await flushPromises()
    expect(wrapper.text()).not.toContain('R090-JOB-FORBIDDEN-EVIDENCE')
    expect(wrapper.text()).not.toContain('R090-CLASSIFICATION-PRIVATE')
    expect(wrapper.text()).not.toContain('31415')
    await router.push('/t3/dashboard')
    await flushPromises()
    await flushPromises()
    expect(wrapper.text()).not.toContain('Pancake')
    expect(wrapper.text()).not.toContain('R090-JOB-FORBIDDEN-EVIDENCE')
    expect(wrapper.find('[data-testid="dashboard-settings-section"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="dashboard-messages-section"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="dashboard-messages-channel-breakdown"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="dashboard-jobs-section"]').exists()).toBe(false)

    resolveStaleTenant({ data: { role: 'owner', permissions: '{}' } })
    await flushPromises()
    await flushPromises()
    expect(wrapper.text()).not.toContain('R090-JOB-FORBIDDEN-EVIDENCE')
    expect(wrapper.text()).not.toContain('R090-CLASSIFICATION-PRIVATE')
    expect(wrapper.find('[data-testid="dashboard-settings-section"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="dashboard-messages-section"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="dashboard-jobs-section"]').exists()).toBe(false)
  })
})
