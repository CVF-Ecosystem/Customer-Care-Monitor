// CCMAI-RUNTIME-088 (F06): synthetic harness for the Messages request-lifetime tests.
// Only the HTTP boundary (api.get) and the browser fetch/URL boundary are faked, with synthetic
// tenants, customers and bytes. The real Messages.vue and the real Pinia conversations store run.
// Nothing here is governance evidence and no real endpoint, token or customer data is touched.
import { vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { createRouter, createMemoryHistory, type Router } from 'vue-router'
import viMessages from '../../../i18n/vi'
import enMessages from '../../../i18n/en'
import { lightColors } from '../../../styles/tokens'
import { useAuthStore } from '../../../stores/auth'
import Messages from '../../../views/Messages.vue'

// ---------------------------------------------------------------------------------------------
// Fake HTTP boundary
// ---------------------------------------------------------------------------------------------
export interface PendingCall {
  url: string
  config: unknown
  settled: boolean
  resolve: (data: unknown) => void
  reject: (error: unknown) => void
}
export interface ApiRule {
  test: RegExp
  mode: 'auto' | 'deferred'
  reply?: (url: string, config: unknown) => unknown
}

export const apiGet = vi.fn()
export const calls: PendingCall[] = []
export const requests: { url: string; config: unknown }[] = []
let rules: ApiRule[] = []

export function resetApi(next: ApiRule[]) {
  rules = next
  calls.length = 0
  requests.length = 0
  apiGet.mockReset()
  apiGet.mockImplementation((url: string, config?: unknown) => {
    requests.push({ url, config })
    const rule = rules.find((r) => r.test.test(url))
    if (!rule) return Promise.resolve({ data: {} })
    if (rule.mode === 'auto') {
      try {
        return Promise.resolve({ data: rule.reply ? rule.reply(url, config) : {} })
      } catch (e) {
        return Promise.reject(e)
      }
    }
    return new Promise((resolve, reject) => {
      const call: PendingCall = {
        url, config, settled: false,
        resolve: (data) => { call.settled = true; resolve({ data }) },
        reject: (error) => { call.settled = true; reject(error) },
      }
      calls.push(call)
    })
  })
}

export const URLS = {
  channels: /\/tenants\/[^/]+\/channels$/,
  list: /\/tenants\/[^/]+\/conversations$/,
  evaluated: /\/tenants\/[^/]+\/conversations\/evaluated$/,
  messages: /\/tenants\/[^/]+\/conversations\/[^/]+\/messages$/,
  evaluations: /\/tenants\/[^/]+\/conversations\/[^/]+\/evaluations$/,
  page: /\/tenants\/[^/]+\/conversations\/[^/]+\/page$/,
}

/** Pending (unsettled) calls whose URL matches. */
export const pending = (re: RegExp) => calls.filter((c) => !c.settled && re.test(c.url))
/** Every call ever issued to a matching URL, settled or not (deferred rules only). */
export const issued = (re: RegExp) => calls.filter((c) => re.test(c.url))
/** Every request to a matching URL including auto-replied ones. */
export const requested = (re: RegExp) => requests.filter((r) => re.test(r.url))
export const messagesCallFor = (tenant: string, conv: string) =>
  issued(URLS.messages).filter((c) => c.url === `/tenants/${tenant}/conversations/${conv}/messages`)
export const evaluationsCallFor = (tenant: string, conv: string) =>
  issued(URLS.evaluations).filter((c) => c.url === `/tenants/${tenant}/conversations/${conv}/evaluations`)

// ---------------------------------------------------------------------------------------------
// Synthetic data
// ---------------------------------------------------------------------------------------------
export function convRow(id: string, customer: string, count = 3) {
  return { id, channel_id: 'ch1', channel_name: 'Kênh mẫu', channel_type: 'facebook', customer_name: customer, last_message_at: '2026-09-27T10:00:00Z', message_count: count, created_at: '' }
}
export const listPayload = (items: unknown[], total = items.length) => ({ data: items, total })

export function message(id: string, text: string, attachments = '[]') {
  return { id, sender_type: 'customer', sender_name: 'Người gửi', content: text, content_type: 'text', attachments, sent_at: '2026-09-27T09:00:00Z' }
}
export function messagesPayload(convId: string, customer: string, texts: string[], count = texts.length, attachments: string[] = []) {
  return {
    conversation: { id: convId, customer_name: customer, message_count: count },
    messages: texts.map((t, i) => message(`${convId}-${customer}-${i}`, t, attachments[i] ?? '[]')),
  }
}
export function evaluationPayload(jobName: string, severity = 'FAIL') {
  return {
    has_evaluation: true,
    groups: [{
      job_run_id: `run-${jobName}`, job_name: jobName, job_type: 'qc_analysis', evaluated_at: '2026-09-27T10:30:00Z',
      results: [{ result_type: 'conversation_evaluation', severity, evidence: 'x', detail: '{"score":40}' }],
    }],
  }
}
export const imageAttachment = (path: string) => JSON.stringify([{ type: 'image', local_path: path, name: 'ảnh mẫu' }])

/** Healthy defaults: channels and the status map answer immediately; list, detail and evaluation are deferred. */
export function standardRules(extra: ApiRule[] = [], listItems: unknown[] = [convRow('cA', 'Khách A', 3), convRow('cB', 'Khách B', 5)]): ApiRule[] {
  return [
    ...extra,
    { test: URLS.channels, mode: 'auto', reply: () => [] },
    { test: URLS.evaluated, mode: 'auto', reply: () => ({}) },
    { test: URLS.list, mode: 'auto', reply: () => listPayload(listItems) },
    { test: URLS.messages, mode: 'deferred' },
    { test: URLS.evaluations, mode: 'deferred' },
    { test: URLS.page, mode: 'auto', reply: () => ({ page: 1 }) },
  ]
}

// ---------------------------------------------------------------------------------------------
// Mounting
// ---------------------------------------------------------------------------------------------
const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

const mounted: VueWrapper[] = []
export interface Mounted { w: VueWrapper; router: Router; pinia: Pinia }

export async function settle(rounds = 4) {
  for (let i = 0; i < rounds; i++) await flushPromises()
}
export const wait = (ms: number) => new Promise((r) => setTimeout(r, ms))

export async function mountMessages(opts: { tenant?: string; query?: string; pinia?: Pinia; width?: number; settleRounds?: number } = {}): Promise<Mounted> {
  const win = window as unknown as { innerWidth: number }
  const previousWidth = win.innerWidth
  if (opts.width) win.innerWidth = opts.width
  const pinia = opts.pinia ?? createPinia()
  setActivePinia(pinia)
  useAuthStore().tenantPerms = { role: 'owner', permissions: {} } as never
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:tenantId/messages', component: Messages }, { path: '/:p(.*)*', component: { template: '<div />' } }],
  })
  await router.push(`/${opts.tenant ?? 't1'}/messages${opts.query ?? ''}`)
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
  win.innerWidth = previousWidth
  mounted.push(w)
  await settle(opts.settleRounds ?? 4)
  return { w, router, pinia }
}

export function unmountOne(w: VueWrapper) {
  const at = mounted.indexOf(w)
  if (at >= 0) mounted.splice(at, 1)
  w.unmount()
}

export function cleanup() {
  while (mounted.length) {
    try { mounted.pop()!.unmount() } catch { /* already unmounted */ }
  }
  document.body.innerHTML = ''
}

// ---------------------------------------------------------------------------------------------
// Readers of what the user can see
// ---------------------------------------------------------------------------------------------
export const rows = (w: VueWrapper) => w.findAll('[data-testid="msgs-row"]')
export function rowByName(w: VueWrapper, name: string) {
  const row = rows(w).find((r) => r.text().includes(name))
  if (!row) throw new Error(`no conversation row containing ${name}`)
  return row
}
export const hasDetail = (w: VueWrapper) => w.find('.mg-conv').exists()
export const detailTitle = (w: VueWrapper) => w.find('.mg-conv__title').text()
export const detailSubtitle = (w: VueWrapper) => w.find('.mg-conv__titles p').text()
export const transcript = (w: VueWrapper) => w.find('.mg-transcript')
export const transcriptText = (w: VueWrapper) => (w.find('.mg-transcript').exists() ? w.find('.mg-transcript').text() : '')
export const transcriptSpinner = (w: VueWrapper) => w.find('.mg-transcript .v-progress-circular').exists()
export const convErrorShown = (w: VueWrapper) => w.find('[data-testid="msgs-conv-error"]').exists()
export const listErrorShown = (w: VueWrapper) => w.find('[data-testid="msgs-list-error"]').exists()
export const downloadButton = (w: VueWrapper) => {
  const b = w.findAll('button').find((x) => x.text().includes(viMessages.msgs_download_txt))
  if (!b) throw new Error('download button not found')
  return b
}
export const downloadDisabled = (w: VueWrapper) => downloadButton(w).attributes('disabled') !== undefined
export async function openTab(w: VueWrapper, which: 'messages' | 'qc' | 'classification') {
  const idx = which === 'messages' ? 0 : which === 'qc' ? 1 : 2
  await w.findAll('.v-tab')[idx].trigger('click')
  await settle(2)
}
export const qcPane = (w: VueWrapper) => w.find('[data-testid="msgs-qc"]')
export const qcText = (w: VueWrapper) => (qcPane(w).exists() ? qcPane(w).text() : '')
export const qcSpinner = (w: VueWrapper) => qcPane(w).exists() && qcPane(w).find('.v-progress-circular').exists()
export const chipTexts = (w: VueWrapper) => w.findAll('[data-testid="msgs-chip"]').map((c) => c.text())
export async function clickRow(w: VueWrapper, name: string) {
  await rowByName(w, name).trigger('click')
  await settle(2)
}

// ---------------------------------------------------------------------------------------------
// Synthetic browser boundary for object URLs and authenticated image fetches
// ---------------------------------------------------------------------------------------------
export interface BlobUrlSpy { created: string[]; revoked: string[]; live: () => string[]; restore: () => void }
export function spyObjectUrls(): BlobUrlSpy {
  const urlApi = URL as unknown as { createObjectURL: unknown; revokeObjectURL: unknown }
  const original = { create: urlApi.createObjectURL, revoke: urlApi.revokeObjectURL }
  const created: string[] = []
  const revoked: string[] = []
  let n = 0
  urlApi.createObjectURL = vi.fn(() => {
    const u = `blob:synthetic-${++n}`
    created.push(u)
    return u
  })
  urlApi.revokeObjectURL = vi.fn((u: string) => { revoked.push(u) })
  return {
    created, revoked,
    live: () => created.filter((u) => !revoked.includes(u)),
    restore: () => { urlApi.createObjectURL = original.create; urlApi.revokeObjectURL = original.revoke },
  }
}

export interface FetchCall { url: string; settled: boolean; resolveOk: (bytes?: string) => void; resolveOkWithPendingBlob: () => () => void; resolveNotOk: () => void; reject: () => void }
export interface FetchStub { calls: FetchCall[]; restore: () => void }
export function stubImageFetch(): FetchStub {
  const calls: FetchCall[] = []
  const original = (globalThis as unknown as { fetch?: unknown }).fetch
  ;(globalThis as unknown as { fetch: unknown }).fetch = vi.fn((url: string) => new Promise((resolve, reject) => {
    const call: FetchCall = {
      url, settled: false,
      resolveOk: (bytes = 'synthetic-bytes') => { call.settled = true; resolve({ ok: true, blob: async () => new Blob([bytes]) }) },
      // headers arrive now, the body (blob) only when the returned function is called
      resolveOkWithPendingBlob: () => {
        let finish: () => void = () => {}
        const body = new Promise<Blob>((done) => { finish = () => done(new Blob(['synthetic-bytes'])) })
        call.settled = true
        resolve({ ok: true, blob: () => body })
        return finish
      },
      resolveNotOk: () => { call.settled = true; resolve({ ok: false, blob: async () => new Blob([]) }) },
      reject: () => { call.settled = true; reject(new Error('synthetic network failure')) },
    }
    calls.push(call)
  }))
  return { calls, restore: () => { (globalThis as unknown as { fetch: unknown }).fetch = original } }
}
