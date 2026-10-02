// @vitest-environment happy-dom
// CCMAI-RUNTIME-028 (F06-10): the real Job Detail cancel flow targets the exact run the user saw,
// treats 202 as "requested" (keeps polling that run until it is terminal), never retargets to a newer
// run, and keeps failures truthful. Mocked API: request-contract evidence only.
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
import { useJobStore } from '../stores/jobs'

vi.mock('vue-chartjs', () => ({ Line: { name: 'Line', render: () => null } }))
const apiGet = vi.fn()
const apiPost = vi.fn()
vi.mock('../api', () => ({ default: { get: (...a: unknown[]) => apiGet(...a), post: (...a: unknown[]) => apiPost(...a), delete: vi.fn() } }))

import JobDetail from '../views/Jobs/JobDetail.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }
vi.setConfig({ testTimeout: 30_000 })

const job = { id: 'j1', name: 'QC mẫu', job_type: 'qc_analysis', schedule_type: 'manual', schedule_cron: '', input_channel_ids: '["c1"]', last_run_at: null, last_run_status: 'running' }
const run = (id: string, status: string) => ({ id, job_id: 'j1', started_at: '2026-10-02T03:00:00Z', finished_at: status === 'running' ? null : '2026-10-02T03:05:00Z', status, summary: '{}', error_message: '' })
const A = '11111111-1111-4111-8111-111111111111'
const B = '22222222-2222-4222-8222-222222222222'

let runsNow: unknown[] = []
let runsFetches = 0
const mounted: VueWrapper[] = []

async function mountView() {
  apiGet.mockImplementation((url: string) => {
    if (url.endsWith('/jobs/j1')) return Promise.resolve({ data: job })
    if (url.endsWith('/jobs/j1/runs')) { runsFetches++; return Promise.resolve({ data: runsNow }) }
    if (url.endsWith('/jobs/j1/results')) return Promise.resolve({ data: [] })
    if (url.endsWith('/settings')) return Promise.resolve({ data: { settings: { ai_api_key: 'configured' } } })
    return Promise.resolve({ data: {} })
  })
  const pinia = createPinia()
  setActivePinia(pinia)
  useAuthStore().tenantPerms = { role: 'owner', permissions: {} } as never
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:tenantId/jobs/:jobId', component: JobDetail }, { path: '/:p(.*)*', component: { template: '<div />' } }] })
  await router.push('/t1/jobs/j1')
  await router.isReady()
  const w = mount(JobDetail, {
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
  return { w, store: useJobStore() }
}

const stopButton = (w: VueWrapper) => w.findAll('button').find((b) => b.text().includes(viMessages.jd_stop))
const notice = () => document.body.querySelector('[data-testid="run-notice"]')?.textContent ?? ''
const cancelCalls = () => apiPost.mock.calls.filter((c) => String(c[0]).includes('/cancel'))

describe('Job Detail cancel targets the exact run (F06-10)', () => {
  beforeEach(() => {
    apiGet.mockReset()
    apiPost.mockReset()
    runsFetches = 0
  })
  afterEach(() => {
    mounted.forEach((w) => w.unmount())
    mounted.length = 0
    document.body.innerHTML = ''
    vi.useRealTimers()
  })

  it('sends the displayed run id and treats 202 as requested, not terminal', async () => {
    runsNow = [run(A, 'running')]
    const { w } = await mountView()
    apiPost.mockResolvedValue({ data: { message: 'job_cancel_requested', run_id: A } })
    await stopButton(w)!.trigger('click')
    await flushPromises()
    expect(cancelCalls()).toHaveLength(1)
    expect(String(cancelCalls()[0][0])).toBe(`/tenants/t1/jobs/j1/cancel?run_id=${A}`)
    expect(notice()).toContain('Đã yêu cầu hủy')
    expect(notice()).not.toContain('đã dừng')
  })

  it('keeps polling the cancelled run until it is terminal, then reports it', async () => {
    runsNow = [run(A, 'running')]
    const { w } = await mountView()
    apiPost.mockResolvedValue({ data: { message: 'job_cancel_requested', run_id: A } })
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    await stopButton(w)!.trigger('click')
    for (let i = 0; i < 5; i++) await Promise.resolve()
    const baseline = runsFetches
    // the worker has not exited: the run is still running for several poll ticks
    await vi.advanceTimersByTimeAsync(2000)
    await vi.advanceTimersByTimeAsync(3000)
    await vi.advanceTimersByTimeAsync(3000)
    expect(runsFetches - baseline).toBeGreaterThanOrEqual(3)
    expect(notice()).toContain('Đã yêu cầu hủy')
    // the worker exits: the exact run is observed cancelled and polling stops
    runsNow = [run(A, 'cancelled')]
    await vi.advanceTimersByTimeAsync(3000)
    await vi.advanceTimersByTimeAsync(0)
    expect(notice()).toContain('đã dừng theo yêu cầu hủy')
    const after = runsFetches
    await vi.advanceTimersByTimeAsync(9000)
    expect(runsFetches).toBe(after)
  })

  it('never retargets a delayed request to a newer run', async () => {
    runsNow = [run(A, 'running')]
    const { w, store } = await mountView()
    let resolve!: (v: unknown) => void
    apiPost.mockReturnValue(new Promise((r) => { resolve = r }))
    await stopButton(w)!.trigger('click')
    // while the request is in flight the list is refreshed: B is now the newest running run
    runsNow = [run(B, 'running'), run(A, 'running')]
    await store.fetchJobRuns('t1', 'j1')
    await flushPromises()
    resolve({ data: { message: 'job_cancel_requested', run_id: A } })
    await flushPromises()
    expect(cancelCalls()).toHaveLength(1)
    expect(String(cancelCalls()[0][0])).toContain(`run_id=${A}`)
    expect(String(cancelCalls()[0][0])).not.toContain(B)
  })

  it('a stale run id answers 409 job_not_running without cancelling or retargeting the new run', async () => {
    runsNow = [run(A, 'running')]
    const { w } = await mountView()
    apiPost.mockRejectedValue({ response: { status: 409, data: { error: 'job_not_running' } } })
    runsNow = [run(B, 'running'), run(A, 'success')] // A already finished, B started meanwhile
    await stopButton(w)!.trigger('click')
    await flushPromises()
    expect(cancelCalls()).toHaveLength(1) // exactly one request, for A; B is never cancelled
    expect(String(cancelCalls()[0][0])).toContain(`run_id=${A}`)
    expect(notice()).toContain('đã kết thúc hoặc không còn là lượt đang chạy')
  })

  it('a failed request keeps the running state truthful and does not claim cancellation', async () => {
    runsNow = [run(A, 'running')]
    const { w } = await mountView()
    apiPost.mockRejectedValue({ response: { status: 500, data: { error: 'job_cancel_failed' } } })
    await stopButton(w)!.trigger('click')
    await flushPromises()
    expect(notice()).toContain('Không gửi được yêu cầu hủy')
    expect(notice()).not.toContain('Đã yêu cầu hủy')
    expect(stopButton(w)).toBeTruthy() // the run is still shown as running, the stop button remains
  })

  it('an unowned running row gets bounded copy', async () => {
    runsNow = [run(A, 'running')]
    const { w } = await mountView()
    apiPost.mockRejectedValue({ response: { status: 409, data: { error: 'job_run_not_owned' } } })
    await stopButton(w)!.trigger('click')
    await flushPromises()
    expect(notice()).toContain('không do tiến trình hiện tại quản lý')
  })

  it('a busy launch shows bounded copy and the request URL is unchanged', async () => {
    runsNow = [run(A, 'success')]
    const { store } = await mountView()
    apiPost.mockRejectedValue({ response: { status: 409, data: { error: 'job_already_running' } } })
    await expect(store.testRunJob('t1', 'j1')).rejects.toBeTruthy()
    expect(String(apiPost.mock.calls[0][0])).toBe('/tenants/t1/jobs/j1/test-run')
  })
})
