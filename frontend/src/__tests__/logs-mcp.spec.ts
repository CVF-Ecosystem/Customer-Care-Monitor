// @vitest-environment happy-dom
// CCMAI-UX-015: UI-structure and interaction tests for the activity log, cost log, notification
// history and MCP connection screens. The API is mocked; nothing is sent to a provider, a
// notification channel or an OAuth client, and nothing here is governance evidence.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
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

const apiGet = vi.fn()
const apiPost = vi.fn()
const apiDelete = vi.fn()
vi.mock('../api', () => ({
  default: {
    get: (...a: unknown[]) => apiGet(...a),
    post: (...a: unknown[]) => apiPost(...a),
    put: vi.fn(),
    delete: (...a: unknown[]) => apiDelete(...a),
  },
}))

import ActivityLogs from '../views/ActivityLogs.vue'
import CostLogs from '../views/CostLogs.vue'
import NotificationLogs from '../views/NotificationLogs.vue'
import MCPConnections from '../views/MCPConnections.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

let failNotif = false
let failMcp = false

const activity = [
  { id: 'a1', action: 'job.run.completed', user_email: 'system', detail: "Job 'Mẫu': 10 analyzed", error_message: '', created_at: '2026-09-28T11:19:00Z' },
  { id: 'a2', action: 'sync.error', user_email: '', detail: 'Sync failed: Kênh mẫu', error_message: 'token expired', created_at: '2026-09-28T10:40:00Z' },
  { id: 'a3', action: 'custom.future', user_email: 'chu@example.invalid', detail: 'x', error_message: '', created_at: '2026-09-28T09:00:00Z' },
]
const costs = [
  { id: 'c1', provider: 'claude', model: 'claude-sonnet-5', input_tokens: 5140, output_tokens: 812, cost_usd: 0.0276, created_at: '2026-09-28T11:19:00Z' },
  { id: 'c2', provider: 'openai', model: 'gpt-5-mini', input_tokens: 3890, output_tokens: 512, cost_usd: 0.0019, created_at: '2026-09-27T13:44:00Z' },
]
const notifs = [
  { id: 'n1', channel_type: 'telegram', recipient: '-100123', status: 'sent', body: 'Kết quả phân tích mẫu', subject: '', sent_at: '2026-09-28T11:20:00Z' },
  { id: 'n2', channel_type: 'email', recipient: 'cskh@example.invalid', status: 'failed', body: 'Nội dung', subject: 'Báo cáo', error_message: 'connection refused', sent_at: '2026-09-27T00:00:00Z' },
]
const mcpClients = [
  { id: 'm1', client_id: 'cqa_synthetic', name: 'Claude Desktop mẫu', redirect_uris: '[]', scopes: '["read","write"]', created_at: '2026-09-28T08:00:00Z' },
]

function setup() {
  apiGet.mockImplementation((url: string) => {
    if (url.endsWith('/activity-logs')) return Promise.resolve({ data: { data: activity, total: 45 } })
    if (url.endsWith('/cost-logs')) return Promise.resolve({ data: { data: costs, total: 312, exchange_rate: 26000 } })
    if (url.endsWith('/notification-logs')) return failNotif ? Promise.reject(new Error('x')) : Promise.resolve({ data: { data: notifs, total: 2 } })
    if (url === '/mcp/clients') return failMcp ? Promise.reject(new Error('x')) : Promise.resolve({ data: mcpClients.map((c) => ({ ...c })) })
    return Promise.resolve({ data: {} })
  })
  apiPost.mockResolvedValue({ data: { id: 'm2', client_id: 'cqa_new', client_secret: 'sk_synthetic_only_in_test', name: 'Mới', scopes: '["read"]' } })
  apiDelete.mockResolvedValue({ data: { message: 'deleted' } })
}

const mounted: VueWrapper[] = []

async function mountAt(component: unknown, path: string, pattern: string) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: pattern, component: component as never }, { path: '/:p(.*)*', component: { template: '<div />' } }] })
  await router.push(path)
  await router.isReady()
  const w = mount(component as never, {
    attachTo: document.body,
    global: {
      plugins: [
        createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } }),
        createI18n({ legacy: false, locale: 'vi', fallbackLocale: 'en', messages: { vi: viMessages, en: enMessages } }),
        pinia,
        router,
      ],
    },
  })
  mounted.push(w)
  for (let i = 0; i < 4; i++) await flushPromises()
  return w
}

const settle = async () => { for (let i = 0; i < 4; i++) await flushPromises() }
const lastParams = (suffix: string) => [...apiGet.mock.calls].reverse().find((c) => String(c[0]).endsWith(suffix))![1].params

describe('Logs, cost, notifications and MCP (UX-015)', () => {
  beforeEach(() => {
    apiGet.mockReset(); apiPost.mockReset(); apiDelete.mockReset()
    failNotif = false
    failMcp = false
    setup()
  })
  afterEach(() => {
    while (mounted.length) mounted.pop()!.unmount()
    document.body.innerHTML = ''
  })

  it('activity: Vietnamese action labels, raw unknown action, and "Hệ thống" for system', async () => {
    const w = await mountAt(ActivityLogs, '/t1/activity-logs', '/:tenantId/activity-logs')
    const actions = w.findAll('[data-testid="lg-action"]').map((e) => e.text())
    expect(actions).toEqual([viMessages.lg_act_job_run_completed, viMessages.lg_act_sync_error, 'custom.future'])
    const actors = w.findAll('[data-testid="lg-actor"]').map((e) => e.text())
    expect(actors).toEqual([viMessages.lg_system, viMessages.lg_system, 'chu@example.invalid'])
    expect(w.text()).toContain('Lỗi: token expired')
  })

  it('activity: changing the filter goes back to page 1 with the action prefix; settings is offered', async () => {
    const w = await mountAt(ActivityLogs, '/t1/activity-logs', '/:tenantId/activity-logs')
    w.findComponent({ name: 'VPagination' }).vm.$emit('update:modelValue', 2)
    await settle()
    expect(lastParams('/activity-logs')).toMatchObject({ page: 2 })
    const select = w.findComponent({ name: 'VSelect' })
    expect((select.props('items') as { value: string }[]).map((i) => i.value)).toContain('settings')
    select.vm.$emit('update:modelValue', 'sync')
    await settle()
    expect(lastParams('/activity-logs')).toEqual({ page: 1, per_page: 20, action: 'sync' })
  })

  it('cost: the footer is the page total and says it is not the filtered total', async () => {
    const w = await mountAt(CostLogs, '/t1/cost-logs', '/:tenantId/cost-logs')
    const footer = w.find('[data-testid="cl-total"]').text()
    expect(footer).toContain('Tổng của trang này (2 lần gọi)')
    expect(footer).toContain('0,0295 US$')
    expect(footer).toContain('767 ₫')
    expect(w.find('[data-testid="cl-total-note"]').text()).toContain('không phải toàn bộ 312 lần gọi')
    const providers = (w.findComponent({ name: 'VSelect' }).props('items') as { value: string }[]).map((i) => i.value)
    expect(providers).toEqual(['', 'claude', 'gemini', 'openai', 'xai'])
    expect(w.text()).toContain('ChatGPT')
  })

  it('notifications: a failed load is an alert, not "no notifications", and retry loads', async () => {
    failNotif = true
    const w = await mountAt(NotificationLogs, '/t1/notifications', '/:tenantId/notifications')
    expect(w.find('[data-testid="nl-load-error"]').attributes('role')).toBe('alert')
    expect(w.find('[data-testid="nl-empty"]').exists()).toBe(false)
    expect(w.text()).not.toContain(viMessages.lg_notif_empty)
    failNotif = false
    await w.find('[data-testid="nl-retry"]').trigger('click')
    await settle()
    expect(w.findAll('[data-testid="nl-row"]')).toHaveLength(2)
    expect(w.findAll('[data-testid="nl-status"]').map((e) => e.text())).toEqual([viMessages.lg_sent, viMessages.lg_failed])
  })

  it('notifications: the toggle reveals the body and reports aria-expanded', async () => {
    const w = await mountAt(NotificationLogs, '/t1/notifications', '/:tenantId/notifications')
    const toggle = w.findAll('[data-testid="nl-toggle"]')[1]
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(w.find('[data-testid="nl-body"]').exists()).toBe(false)
    await toggle.trigger('click')
    await settle()
    expect(w.findAll('[data-testid="nl-toggle"]')[1].attributes('aria-expanded')).toBe('true')
    expect(w.find('[data-testid="nl-body"]').text()).toBe('Nội dung')
    expect(w.text()).toContain('Tiêu đề: Báo cáo')
  })

  it('mcp: a failed load is an alert, not the empty state', async () => {
    failMcp = true
    const w = await mountAt(MCPConnections, '/mcp', '/mcp')
    expect(w.find('[data-testid="mc-load-error"]').exists()).toBe(true)
    expect(w.find('[data-testid="mc-empty"]').exists()).toBe(false)
  })

  it('mcp: revoke asks through ConfirmDialog before DELETE; scopes are Vietnamese', async () => {
    const w = await mountAt(MCPConnections, '/mcp', '/mcp')
    expect(w.findAll('[data-testid="mc-scope"]').map((e) => e.text())).toEqual(['Đọc', 'Ghi'])
    await w.find('[data-testid="mc-revoke"]').trigger('click')
    await settle()
    expect(apiDelete).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain('Thu hồi kết nối “Claude Desktop mẫu”?')
    ;(document.body.querySelector('[data-testid="confirm-ok"]') as HTMLElement).click()
    await settle()
    expect(apiDelete).toHaveBeenCalledWith('/mcp/clients/m1')
  })

  it('mcp: the created secret is shown once with the warning and cleared on close', async () => {
    const w = await mountAt(MCPConnections, '/mcp', '/mcp')
    await w.find('[data-testid="mc-open-create"]').trigger('click')
    await settle()
    const name = document.body.querySelector('[data-testid="mc-name"] input') as HTMLInputElement
    name.value = 'Mới'
    name.dispatchEvent(new Event('input'))
    await settle()
    ;(document.body.querySelector('[data-testid="mc-create"]') as HTMLElement).click()
    await settle()
    expect(apiPost).toHaveBeenCalledWith('/mcp/clients', { name: 'Mới', redirect_uris: [], scopes: ['read', 'write'] })
    expect(document.body.querySelector('[data-testid="mc-secret"]')!.textContent).toBe('sk_synthetic_only_in_test')
    expect(document.body.textContent).toContain(viMessages.mc_secret_warn)
    ;(document.body.querySelector('[data-testid="mc-done"]') as HTMLElement).click()
    await settle()
    await w.find('[data-testid="mc-open-create"]').trigger('click')
    await settle()
    expect(document.body.querySelector('[data-testid="mc-secret"]')).toBeNull()
  })
})
