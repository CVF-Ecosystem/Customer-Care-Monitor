// @vitest-environment happy-dom
// CCMAI-UX-013: UI-structure and interaction tests for the AI job list, create and edit screens.
// The API is mocked; nothing here is governance evidence and no output test is really sent.
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
import { describeSchedule, channelCount } from '../views/Jobs/job-list/schedule'

const apiGet = vi.fn()
const apiPost = vi.fn()
const apiPut = vi.fn()
const apiDelete = vi.fn()
vi.mock('../api', () => ({
  default: {
    get: (...a: unknown[]) => apiGet(...a),
    post: (...a: unknown[]) => apiPost(...a),
    put: (...a: unknown[]) => apiPut(...a),
    delete: (...a: unknown[]) => apiDelete(...a),
  },
}))

import JobList from '../views/Jobs/JobList.vue'
import JobCreate from '../views/Jobs/JobCreate.vue'
import JobEdit from '../views/Jobs/JobEdit.vue'
import StepOutput from '../components/JobWizard/StepOutput.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

const jobs = [
  { id: 'j1', name: 'QC mẫu', description: 'Quy định chuẩn', job_type: 'qc_analysis', schedule_type: 'cron', schedule_cron: '0 7 * * *', input_channel_ids: '["a","b"]', is_active: true, last_run_at: '2026-09-28T11:19:00Z', last_run_status: 'success' },
  { id: 'j2', name: 'Phân loại mẫu', description: '', job_type: 'classification', schedule_type: 'after_sync', schedule_cron: '', input_channel_ids: '["a"]', is_active: true, last_run_at: '2026-09-28T09:02:00Z', last_run_status: 'failed' },
  { id: 'j3', name: 'Thủ công', description: '', job_type: 'qc_analysis', schedule_type: 'manual', schedule_cron: '', input_channel_ids: 'oops', is_active: false, last_run_at: null, last_run_status: '' },
]
const channels = [{ id: 'a', name: 'Kênh A', channel_type: 'facebook', is_active: true }]

let failJobs = false
let failJob = false

function setup() {
  apiGet.mockImplementation((url: string) => {
    if (url.endsWith('/jobs')) return failJobs ? Promise.reject(new Error('x')) : Promise.resolve({ data: jobs })
    if (url.endsWith('/jobs/j1')) return failJob ? Promise.reject(new Error('x')) : Promise.resolve({ data: { ...jobs[0], rules_content: 'R', rules_config: '[]', outputs: '[]' } })
    if (url.endsWith('/channels')) return Promise.resolve({ data: channels })
    return Promise.resolve({ data: {} })
  })
  apiDelete.mockResolvedValue({ data: {} })
}

const mounted: VueWrapper[] = []

async function mountAt(component: unknown, path: string, pattern: string, props: Record<string, unknown> = {}) {
  const pinia = createPinia()
  setActivePinia(pinia)
  useAuthStore().tenantPerms = { role: 'owner', permissions: {} } as never
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: pattern, component: component as never }, { path: '/:p(.*)*', component: { template: '<div />' } }] })
  await router.push(path)
  await router.isReady()
  const w = mount(component as never, {
    attachTo: document.body,
    props: props as never,
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

describe('describeSchedule / channelCount (UX-013)', () => {
  it('reads the three CronPicker shapes and never guesses others', () => {
    expect(describeSchedule('cron', '0 7 * * *')).toEqual({ kind: 'daily', time: '07:00' })
    expect(describeSchedule('cron', '30 18 * * 1,3,5')).toEqual({ kind: 'weekly', days: [1, 3, 5], time: '18:30' })
    expect(describeSchedule('cron', '5 6 15 * *')).toEqual({ kind: 'monthly', day: 15, time: '06:05' })
    expect(describeSchedule('after_sync', '0 7 * * *')).toEqual({ kind: 'after_sync' })
    expect(describeSchedule('manual', '')).toEqual({ kind: 'manual' })
    expect(describeSchedule('cron', '*/5 * * * *')).toEqual({ kind: 'raw', cron: '*/5 * * * *' })
    expect(describeSchedule('cron', '0 7 * 1 *')).toEqual({ kind: 'raw', cron: '0 7 * 1 *' })
    expect(describeSchedule('cron', '')).toEqual({ kind: 'raw', cron: '' })
  })
  it('counts channels, unknown when malformed (never 0)', () => {
    expect(channelCount('["a","b"]')).toBe(2)
    expect(channelCount('[]')).toBe(0)
    expect(channelCount('oops')).toBeNull()
  })
})

describe('AI job screens (UX-013)', () => {
  beforeEach(() => {
    apiGet.mockReset(); apiPost.mockReset(); apiPut.mockReset(); apiDelete.mockReset()
    failJobs = false
    failJob = false
    setup()
  })
  afterEach(() => {
    while (mounted.length) mounted.pop()!.unmount()
    document.body.innerHTML = ''
  })

  it('list shows type, schedule in words, channel count, Vietnamese run labels and dd/mm/yyyy (UX-12/13/14)', async () => {
    const w = await mountAt(JobList, '/t1/jobs', '/:tenantId/jobs')
    expect(w.find('h1').text()).toBe('Tác vụ AI')
    expect(w.find('[data-testid="jl-subtitle"]').text()).toContain('3 tác vụ')
    expect(w.findAll('[data-testid="jl-schedule"]').map((s) => s.text())).toEqual(['Mỗi ngày lúc 07:00', 'Sau mỗi lần đồng bộ', 'Chạy thủ công'])
    expect(w.findAll('[data-testid="jl-run"]').map((s) => s.text())).toEqual(['Thành công', 'Lỗi', 'Chưa chạy'])
    expect(w.find('[data-testid="jl-run-time"]').text()).toMatch(/^\d{2}\/\d{2}\/2026 \d{2}:\d{2}$/)
    expect(w.text()).toContain('2 kênh')
    expect(w.text()).toContain('Kênh: không rõ')
    expect(w.text()).not.toContain('success')
  })

  it('delete asks first, names the cascade, and only deletes on confirm', async () => {
    const w = await mountAt(JobList, '/t1/jobs', '/:tenantId/jobs')
    await w.findAll('[data-testid="action-menu"]')[0].trigger('click')
    await flushPromises()
    ;(document.body.querySelector('[data-key="delete"]') as HTMLElement).click()
    await flushPromises()
    expect(document.body.textContent).toContain('nhật ký chi phí AI và lịch sử thông báo')
    expect(document.body.querySelector('[data-testid="confirm-ok"]')!.textContent!.trim()).toBe('Xóa tác vụ')
    expect(apiDelete).not.toHaveBeenCalled()
    ;(document.body.querySelector('[data-testid="confirm-cancel"]') as HTMLElement).click()
    await flushPromises()
    expect(apiDelete).not.toHaveBeenCalled()
    await w.findAll('[data-testid="action-menu"]')[0].trigger('click')
    await flushPromises()
    ;(document.body.querySelector('[data-key="delete"]') as HTMLElement).click()
    await flushPromises()
    ;(document.body.querySelector('[data-testid="confirm-ok"]') as HTMLElement).click()
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(apiDelete).toHaveBeenCalledWith('/tenants/t1/jobs/j1')
  })

  it('list load error shows an alert and retries', async () => {
    failJobs = true
    const w = await mountAt(JobList, '/t1/jobs', '/:tenantId/jobs')
    expect(w.find('[data-testid="jl-load-error"]').attributes('role')).toBe('alert')
    failJobs = false
    await w.find('[data-testid="jl-load-error"] button').trigger('click')
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(w.findAll('[data-testid="jl-row"]')).toHaveLength(3)
  })

  it('edit: a failed load shows no form and no Save, so defaults can never overwrite the job', async () => {
    failJob = true
    const w = await mountAt(JobEdit, '/t1/jobs/j1/edit', '/:tenantId/jobs/:jobId/edit')
    expect(w.find('[data-testid="jw-edit-load-error"]').exists()).toBe(true)
    expect(w.find('[data-testid="jw-edit-form"]').exists()).toBe(false)
    expect(w.find('[data-testid="jw-save"]').exists()).toBe(false)
    expect(apiPut).not.toHaveBeenCalled()
    failJob = false
    await w.find('[data-testid="jw-edit-load-error"] button').trigger('click')
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(w.find('[data-testid="jw-save"]').exists()).toBe(true)
  })

  it('email output explains the missing test support and keeps the gate closed', async () => {
    const form = { outputs: '[]', outputs_validated: true }
    const w = await mountAt(StepOutput, '/t1/jobs/create', '/:tenantId/jobs/create', { form, 'onUpdate:form': () => {} })
    await w.findAll('button').find((b) => b.text().includes('Thêm đầu ra'))!.trigger('click')
    await flushPromises()
    expect(w.find('[data-testid="jw-output-test"]').exists()).toBe(true)
    expect(form.outputs_validated).toBe(false)
    // switch the output to email through the component state
    const outputs = JSON.parse(form.outputs)
    expect(outputs[0].type).toBe('telegram')
    const vm = w.vm as unknown as { $: { setupState: { outputs: { type: string }[] } } }
    vm.$.setupState.outputs[0].type = 'email'
    await flushPromises()
    expect(w.find('[data-testid="jw-email-unsupported"]').text()).toBe(viMessages.jw_output_email_unsupported)
    expect(w.find('[data-testid="jw-output-test"]').exists()).toBe(false)
    expect(form.outputs_validated).toBe(false)
  })

  it('create: the disabled Next says why, and revisiting the output step keeps an untested output untested', async () => {
    const w = await mountAt(JobCreate, '/t1/jobs/create', '/:tenantId/jobs/create')
    expect(w.find('[data-testid="jw-reason"]').text()).toBe('Tối thiểu 2 ký tự')
    expect(w.find('[data-testid="jw-next"]').attributes('disabled')).toBeDefined()
    await w.find('input[type="text"]').setValue('Tác vụ thử')
    await w.find('[data-testid="jw-next"]').trigger('click') // -> 2
    await flushPromises()
    await w.find('input[type="checkbox"]').setValue(true)
    await w.find('[data-testid="jw-next"]').trigger('click') // -> 3
    await flushPromises()
    const textareas = w.findAll('textarea').filter((t) => t.isVisible())
    await textareas[0].setValue('Quy định')
    await w.find('[data-testid="jw-next"]').trigger('click') // -> 4
    await flushPromises()
    await w.findAll('button').filter((b) => b.isVisible()).find((b) => b.text().includes('Thêm đầu ra'))!.trigger('click')
    await flushPromises()
    expect(w.find('[data-testid="jw-next"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="jw-reason"]').text()).toBe(viMessages.jw_output_invalid)
    await w.findAll('button').filter((b) => b.isVisible()).find((b) => b.text().includes('Quay lại'))!.trigger('click') // -> 3
    await flushPromises()
    await w.find('[data-testid="jw-next"]').trigger('click') // -> 4 again
    await flushPromises()
    expect(w.find('[data-testid="jw-output-tested"]').exists()).toBe(false)
    expect(w.find('[data-testid="jw-next"]').attributes('disabled')).toBeDefined()
  })
})
