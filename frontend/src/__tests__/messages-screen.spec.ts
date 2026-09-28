// @vitest-environment happy-dom
// CCMAI-UX-012: UI-structure and interaction tests for the Messages screen. The API is mocked;
// nothing here is governance evidence.
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
import { useAuthStore } from '../stores/auth'

const apiGet = vi.fn()
vi.mock('../api', () => ({ default: { get: (...args: unknown[]) => apiGet(...args), post: vi.fn(), delete: vi.fn(), put: vi.fn() } }))

import Messages from '../views/Messages.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

const convs = [
  { id: 'c1', channel_id: 'ch1', channel_name: 'Facebook mẫu', channel_type: 'facebook', customer_name: 'Khách Một', last_message_at: '2026-09-27T10:00:00Z', message_count: 8, created_at: '' },
  { id: 'c2', channel_id: 'ch1', channel_name: 'Facebook mẫu', channel_type: 'facebook', customer_name: 'Khách Hai', last_message_at: '2026-09-27T09:00:00Z', message_count: 10, created_at: '' },
  { id: 'c3', channel_id: 'ch2', channel_name: 'Zalo mẫu', channel_type: 'zalo_oa', customer_name: 'Khách Ba', last_message_at: '2026-09-27T08:00:00Z', message_count: 2, created_at: '' },
  { id: 'c4', channel_id: 'ch2', channel_name: 'Zalo mẫu', channel_type: 'zalo_oa', customer_name: '', last_message_at: '2026-09-27T07:00:00Z', message_count: 5, created_at: '' },
]
const evalMap = { c1: 'PASS', c2: 'FAIL', c3: 'SKIP' }
const evaluations = {
  has_evaluation: true,
  groups: [
    {
      job_run_id: 'r1', job_name: 'QC mẫu', job_type: 'qc_analysis', evaluated_at: '2026-09-27T10:30:00Z',
      results: [
        { result_type: 'conversation_evaluation', severity: 'FAIL', evidence: 'Trả lời thiếu', detail: '{"score":40}' },
        { result_type: 'qc_violation', severity: 'NGHIEM_TRONG', rule_name: 'Chào hỏi', evidence: 'Không chào' },
      ],
    },
    {
      job_run_id: 'r2', job_name: 'Phân loại mẫu', job_type: 'classification', evaluated_at: '2026-09-27T10:40:00Z',
      results: [
        { result_type: 'conversation_evaluation', severity: 'PASS', evidence: 'Hỏi giờ mở cửa', detail: '{"summary":"Hỏi giờ mở cửa"}' },
        { result_type: 'classification_tag', severity: 'INFO', rule_name: 'Hỗ trợ chung', evidence: 'Khách hỏi giờ' },
      ],
    },
  ],
}

type Opts = { failList?: boolean; items?: unknown[]; total?: number; exportBody?: string }
let opts: Opts = {}

function setup(o: Opts = {}) {
  opts = o
  apiGet.mockImplementation((url: string) => {
    if (url.endsWith('/channels')) return Promise.resolve({ data: [] })
    if (url.endsWith('/conversations')) {
      if (opts.failList) return Promise.reject(new Error('boom'))
      const items = opts.items ?? convs
      return Promise.resolve({ data: { data: items, total: opts.total ?? 210 } })
    }
    if (url.endsWith('/conversations/evaluated')) return Promise.resolve({ data: evalMap })
    if (url.endsWith('/messages')) return Promise.resolve({ data: { conversation: { id: 'c2', customer_name: 'Khách Hai', message_count: 10 }, messages: [{ id: 'm1', sender_type: 'customer', sender_name: 'Khách Hai', content: 'Xin chào', content_type: 'text', attachments: '[]', sent_at: '2026-09-27T09:00:00Z' }] } })
    if (url.endsWith('/evaluations')) return Promise.resolve({ data: evaluations })
    if (url.includes('/conversations/export')) return Promise.resolve({ data: opts.exportBody ?? 'nội dung' })
    return Promise.resolve({ data: {} })
  })
}

const mounted: VueWrapper[] = []

async function mountView() {
  const pinia = createPinia()
  setActivePinia(pinia)
  useAuthStore().tenantPerms = { role: 'owner', permissions: {} } as never
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:tenantId/messages', component: Messages }, { path: '/:p(.*)*', component: { template: '<div />' } }] })
  await router.push('/t1/messages')
  await router.isReady()
  const w = mount(Messages, {
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

const listCalls = () => apiGet.mock.calls.filter((c) => String(c[0]).endsWith('/conversations'))
const wait = (ms: number) => new Promise((r) => setTimeout(r, ms))

describe('Messages screen (UX-012)', () => {
  beforeEach(() => {
    apiGet.mockReset()
  })
  afterEach(() => {
    while (mounted.length) mounted.pop()!.unmount()
    document.body.innerHTML = ''
  })

  it('counts conversations, not messages (UX-16)', async () => {
    setup()
    const w = await mountView()
    expect(w.find('[data-testid="msgs-subtitle"]').text()).toContain('210 hội thoại')
    expect(w.find('h1').text()).toBe(viMessages.nav_messages)
    expect(w.find('[data-testid="msgs-range"]').text()).toBe('Hiển thị 1–4 trên 210 hội thoại')
  })

  it('names the latest analysis honestly: PASS may be classification', async () => {
    setup()
    const w = await mountView()
    const chips = w.findAll('[data-testid="msgs-chip"]').map((c) => c.text())
    expect(chips).toEqual([viMessages.msgs_chip_pass, viMessages.msgs_chip_fail, viMessages.msgs_chip_skip, viMessages.msgs_chip_none])
    expect(chips).not.toContain('Đạt')
  })

  it('keeps list request parameters and debounced search', async () => {
    setup()
    const w = await mountView()
    expect(listCalls()[0][1]).toEqual({ params: { page: 1, per_page: 9 } })
    await w.find('input[aria-label="Tìm tên khách"]').setValue('Hai')
    await wait(350)
    await flushPromises()
    expect(listCalls().at(-1)![1]).toEqual({ params: { page: 1, per_page: 9, search: 'Hai' } })
    await w.find('[data-testid="msgs-clear"]').trigger('click')
    await flushPromises()
    expect(listCalls().at(-1)![1]).toEqual({ params: { page: 1, per_page: 9 } })
  })

  it('list load error shows an alert with retry', async () => {
    setup({ failList: true })
    const w = await mountView()
    const alert = w.find('[data-testid="msgs-list-error"]')
    expect(alert.attributes('role')).toBe('alert')
    opts.failList = false
    await alert.find('button').trigger('click')
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(w.find('[data-testid="msgs-list-error"]').exists()).toBe(false)
    expect(w.findAll('[data-testid="msgs-row"]')).toHaveLength(4)
  })

  it('empty tenant vs no match are different states', async () => {
    setup({ items: [], total: 0 })
    const w = await mountView()
    expect(w.find('[data-testid="msgs-empty"]').exists()).toBe(true)
    await w.find('input[aria-label="Tìm tên khách"]').setValue('zzz')
    await wait(350)
    await flushPromises()
    expect(w.find('[data-testid="msgs-no-match"]').exists()).toBe(true)
  })

  it('conversation tabs: QC shows issues, classification shows tags and never "Vấn đề"; both carry the source note', async () => {
    setup()
    const w = await mountView()
    await w.findAll('[data-testid="msgs-row"]')[1].trigger('click')
    for (let i = 0; i < 4; i++) await flushPromises()
    const tabs = w.findAll('.v-tab')
    expect(tabs.map((t) => t.text())).toEqual(['Tin nhắn', 'Đánh giá chất lượng · 1', 'Phân loại · 1'])
    await tabs[1].trigger('click')
    await flushPromises()
    const qc = w.find('[data-testid="msgs-qc"]')
    expect(qc.find('[data-testid="msgs-source-note"]').text()).toContain('chưa đối chiếu nguồn')
    expect(qc.text()).toContain('Vấn đề (1)')
    expect(qc.text()).toContain('40/100')
    expect(qc.find('[data-verdict="fail"]').exists()).toBe(true)
    await tabs[2].trigger('click')
    await flushPromises()
    const cls = w.find('[data-testid="msgs-class"]')
    expect(cls.text()).toContain('Hỗ trợ chung')
    expect(cls.text()).toContain('chưa đối chiếu nguồn')
    expect(cls.text()).not.toContain('Vấn đề')
    expect(cls.find('[data-verdict]').exists()).toBe(false)
  })

  it('export: a 200 JSON {error} body is shown, not downloaded; real content is downloaded', async () => {
    const urlApi = URL as unknown as { createObjectURL: unknown; revokeObjectURL: unknown }
    const create = vi.fn(() => 'blob:x')
    urlApi.createObjectURL = create
    urlApi.revokeObjectURL = vi.fn()
    setup({ exportBody: '{"error":"Không có cuộc chat nào trong khoảng thời gian này"}' })
    const w = await mountView()
    await w.find('[data-testid="msgs-export-open"]').trigger('click')
    await flushPromises()
    expect(document.body.querySelector('[data-testid="msgs-export-desc"]')!.textContent).toBe(viMessages.msgs_export_desc)
    ;(document.body.querySelector('[data-testid="msgs-export-download"]') as HTMLElement).click()
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(create).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain('Không có cuộc chat nào trong khoảng thời gian này')
    opts.exportBody = 'Cuộc chat: A'
    ;(document.body.querySelector('[data-testid="msgs-export-download"]') as HTMLElement).click()
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(create).toHaveBeenCalledTimes(1)
    const call = apiGet.mock.calls.find((c) => String(c[0]).includes('/conversations/export'))!
    expect(String(call[0])).toMatch(/\/tenants\/t1\/conversations\/export\?from=\d{4}-\d{2}-\d{2}&to=\d{4}-\d{2}-\d{2}&format=txt$/)
  })
})
