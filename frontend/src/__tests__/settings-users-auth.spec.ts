// @vitest-environment happy-dom
// CCMAI-UX-014: UI-structure and interaction tests for Settings, Users and Login. The API is
// mocked; no provider, storage or auth service is contacted and nothing here is governance evidence.
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
const apiPost = vi.fn()
const apiPut = vi.fn()
vi.mock('../api', () => ({
  default: {
    get: (...a: unknown[]) => apiGet(...a),
    post: (...a: unknown[]) => apiPost(...a),
    put: (...a: unknown[]) => apiPut(...a),
    delete: vi.fn(),
  },
}))

import Settings from '../views/Settings.vue'
import Users from '../views/Users.vue'
import Login from '../views/Login.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

let failSettings = false
let failUsers = false
let storageS3 = false
const users = [
  { user_id: 'u1', email: 'chu@example.invalid', name: 'Chủ', role: 'owner', permissions: '' },
  { user_id: 'u2', email: 'nv@example.invalid', name: 'Nhân viên', role: 'member', permissions: '{"channels":"r"}' },
]

function setup() {
  apiGet.mockImplementation((url: string) => {
    if (url.endsWith('/settings')) return failSettings ? Promise.reject(new Error('x')) : Promise.resolve({ data: { settings: { ai_provider: 'claude', ai_model: 'claude-sonnet-5', ai_api_key: '••••••••' }, tenant: { name: 'Cửa hàng mẫu', timezone: 'Asia/Ho_Chi_Minh', language: 'vi' } } })
    if (url.endsWith('/settings/ai/models')) return Promise.reject(new Error('offline'))
    if (url.endsWith('/settings/storage')) return Promise.resolve({ data: storageS3 ? { backend: 's3', endpoint: 'https://s3.example.invalid', bucket: 'synthetic-bucket', region: 'auto', access_key: 'SYNTHETIC', prefix: '', force_path_style: true } : { backend: 'local', local_bytes: 0, local_files: 0 } })
    if (url.endsWith('/users')) return failUsers ? Promise.reject(new Error('x')) : Promise.resolve({ data: users.map((u) => ({ ...u })) })
    return Promise.resolve({ data: {} })
  })
  apiPost.mockResolvedValue({ data: { user_id: 'u3', email: 'm@example.invalid', name: 'Mới', role: 'member', permissions: '' } })
  apiPut.mockResolvedValue({ data: {} })
}

const mounted: VueWrapper[] = []

async function mountAt(component: unknown, path: string, pattern: string) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const auth = useAuthStore()
  auth.tenantPerms = { role: 'owner', permissions: {} } as never
  auth.user = { id: 'u1', email: 'chu@example.invalid', name: 'Chủ' } as never
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

describe('Settings, Users, Login (UX-014)', () => {
  beforeEach(() => {
    apiGet.mockReset(); apiPost.mockReset(); apiPut.mockReset()
    failSettings = false
    failUsers = false
    storageS3 = false
    setup()
  })
  afterEach(() => {
    while (mounted.length) mounted.pop()!.unmount()
    document.body.innerHTML = ''
  })

  it('settings: a failed load shows no form, so defaults can never be saved over real settings', async () => {
    failSettings = true
    const w = await mountAt(Settings, '/t1/settings', '/:tenantId/settings')
    expect(w.find('[data-testid="st-load-error"]').attributes('role')).toBe('alert')
    expect(w.find('[data-testid="st-save-ai"]').exists()).toBe(false)
    expect(apiPut).not.toHaveBeenCalled()
    failSettings = false
    await w.find('[data-testid="st-load-error"] button').trigger('click')
    for (let i = 0; i < 4; i++) await flushPromises()
    expect(w.find('[data-testid="st-save-ai"]').exists()).toBe(true)
    expect(apiPut).not.toHaveBeenCalled()
  })

  it('settings: the key test says it uses the saved key, and the saved-key note is shown', async () => {
    const w = await mountAt(Settings, '/t1/settings', '/:tenantId/settings')
    expect(w.find('[data-testid="st-test-key"]').text()).toBe(viMessages.st_test_saved_key)
    expect(w.text()).toContain(viMessages.st_test_saved_note)
    expect(w.text()).toContain(viMessages.st_key_saved_note)
    expect(apiPost).not.toHaveBeenCalled()
  })

  it('settings: the S3-off dialog shows the Compose command and still asks before saving (UX014-R1)', async () => {
    storageS3 = true
    const w = await mountAt(Settings, '/t1/settings', '/:tenantId/settings')
    await w.findAll('button').find((b) => b.text().includes(viMessages.storage_settings))!.trigger('click')
    for (let i = 0; i < 3; i++) await flushPromises()
    w.find('[data-testid="st-storage"]').findComponent({ name: 'VSwitch' }).vm.$emit('update:modelValue', false)
    for (let i = 0; i < 3; i++) await flushPromises()
    const cmd = document.body.querySelector('[data-testid="st-s3-off-cmd"]')!
    expect(cmd.textContent).toBe('docker compose exec app /app/cqa-server migrate-files -down -apply')
    expect(document.body.textContent).not.toContain('cqa-app')
    expect(document.body.textContent).toContain(viMessages.st_s3_off_cmd_where)
    expect(apiPut).not.toHaveBeenCalled()
    expect(apiPost).not.toHaveBeenCalled()
  })

  it('users: Vietnamese role names, "(bạn)" for self, and no actions on self', async () => {
    const w = await mountAt(Users, '/t1/users', '/:tenantId/users')
    const rows = w.findAll('[data-testid="us-row"]')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('(bạn)')
    expect(rows[0].text()).toContain('Chủ sở hữu')
    expect(rows[1].text()).toContain('Thành viên')
    expect(rows[0].find('[data-testid="action-menu"]').exists()).toBe(false)
    expect(rows[1].find('[data-testid="action-menu"]').exists()).toBe(true)
    expect(w.text()).not.toMatch(/\bOwner\b|\bMember\b/)
  })

  it('users: the create rule matches the server (8 chars, uppercase, digit)', async () => {
    const w = await mountAt(Users, '/t1/users', '/:tenantId/users')
    await w.findAll('button').find((b) => b.text().includes(viMessages.create_user))!.trigger('click')
    await flushPromises()
    const inputs = [...document.body.querySelectorAll('.v-overlay--active input')] as HTMLInputElement[]
    const set = async (el: HTMLInputElement, v: string) => { el.value = v; el.dispatchEvent(new Event('input')); await flushPromises() }
    await set(inputs[0], 'Mới')
    await set(inputs[1], 'm@example.invalid')
    await set(document.body.querySelector('[data-testid="us-new-password"] input') as HTMLInputElement, 'abcdef12')
    ;(document.body.querySelector('[data-testid="us-invite"]') as HTMLElement).click()
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(apiPost).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain(viMessages.us_err_weak)
    await set(document.body.querySelector('[data-testid="us-new-password"] input') as HTMLInputElement, 'Abcdefg1')
    ;(document.body.querySelector('[data-testid="us-invite"]') as HTMLElement).click()
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(apiPost).toHaveBeenCalledWith('/tenants/t1/users/invite', expect.objectContaining({ email: 'm@example.invalid', password: 'Abcdefg1', role: 'member' }))
  })

  it('users: load error shows an alert with retry', async () => {
    failUsers = true
    const w = await mountAt(Users, '/t1/users', '/:tenantId/users')
    expect(w.find('[data-testid="us-load-error"]').exists()).toBe(true)
    failUsers = false
    await w.find('[data-testid="us-load-error"] button').trigger('click')
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(w.findAll('[data-testid="us-row"]')).toHaveLength(2)
  })

  it('login: a 401 shows the invalid-credentials alert', async () => {
    apiPost.mockRejectedValue({ response: { status: 401 } })
    const w = await mountAt(Login, '/login', '/login')
    await w.find('input[type="email"]').setValue('a@b.c')
    await w.find('input[type="password"]').setValue('x')
    await w.find('form').trigger('submit')
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(w.find('[data-testid="au-error"]').text()).toBe(viMessages.invalid_credentials)
    expect(w.find('[data-testid="au-error"]').attributes('role')).toBe('alert')
  })
})
