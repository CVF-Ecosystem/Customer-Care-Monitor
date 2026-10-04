// @vitest-environment happy-dom
// CCMAI-RUNTIME-036: /setup/status must be confirmed before the app opens. Uses the real router
// guard, the real App lifecycle, the real unavailable view and the real auth store with a mocked
// API and synthetic tokens. Frontend routing and request-sequencing evidence only: it makes no
// backend-security, AI or governance assertion.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import axios, { AxiosError } from 'axios'
import { createI18n } from 'vue-i18n'
import viMessages from '../i18n/vi'
import enMessages from '../i18n/en'

const apiGet = vi.fn()
const apiPost = vi.fn()
vi.mock('../api', () => ({
  default: {
    get: (...a: unknown[]) => apiGet(...a),
    post: (...a: unknown[]) => apiPost(...a),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

const TOKEN = 'existing-synthetic-token' // cvf-allow-secret-fixture: synthetic test value
const LEGACY = 'legacy-synthetic-refresh' // cvf-allow-secret-fixture: synthetic test value

// ---- in-memory storage that also bounds the token reads of one navigation (microtask-loop detector)
const LOOP_LIMIT = 50
let tokenReads = 0
let armed = false
const store = new Map<string, string>()
const memoryStorage = {
  getItem(key: string) {
    if (armed && key === 'cqa_access_token' && ++tokenReads > LOOP_LIMIT) throw new Error('redirect loop detected')
    return store.has(key) ? store.get(key)! : null
  },
  setItem(key: string, value: string) { store.set(key, String(value)) },
  removeItem(key: string) { store.delete(key) },
  clear() { store.clear() },
  key(i: number) { return [...store.keys()][i] ?? null },
  get length() { return store.size },
}
Object.defineProperty(globalThis, 'localStorage', { value: memoryStorage, configurable: true })
Object.defineProperty(window, 'localStorage', { value: memoryStorage, configurable: true })

// ---- controllable /setup/status
type Cfg = { timeout?: number; signal?: AbortSignal } | undefined
interface Pending { resolve: (v: unknown) => void; reject: (e: unknown) => void; cfg: Cfg }
let pending: Pending[] = []
let statusImpl: (cfg: Cfg) => Promise<unknown>
const deferStatus = () => (cfg: Cfg) => new Promise<unknown>((resolve, reject) => { pending.push({ resolve, reject, cfg }) })
const answer = (data: unknown) => () => Promise.resolve({ data })

function installApi() {
  apiGet.mockImplementation((url: string, cfg?: Cfg) => {
    if (url === '/setup/status') return statusImpl(cfg)
    if (url === '/profile') return Promise.resolve({ data: { id: 'u1', email: 'owner@example.invalid', name: 'Owner', is_admin: true, language: 'vi' } })
    if (url === '/tenants') return Promise.resolve({ data: [] })
    return Promise.resolve({ data: {} })
  })
  apiPost.mockImplementation((url: string) => Promise.reject(new Error(`unexpected POST ${url}`)))
}

const calls = (url: string) => apiGet.mock.calls.filter((c) => c[0] === url).length
const urls = () => apiGet.mock.calls.map((c) => c[0] as string)

async function freshApp() {
  vi.resetModules()
  const pinia = createPinia()
  setActivePinia(pinia)
  const routerMod = await import('../router')
  const status = await import('../router/setupStatus')
  const { useAuthStore } = await import('../stores/auth')
  return { router: routerMod.default, pinia, status, useAuthStore, markSetupComplete: routerMod.markSetupComplete }
}

async function navigate(router: { push: (p: string) => Promise<unknown> }, path: string) {
  tokenReads = 0
  armed = true
  try {
    return await router.push(path).then(
      (failure) => ({ kind: 'done' as const, failure }),
      (error: Error) => ({ kind: 'error' as const, error: error.message }),
    )
  } finally {
    armed = false
  }
}

function plugins(pinia: ReturnType<typeof createPinia>, router: unknown, locale = 'vi') {
  const vuetify = createVuetify({ components, directives })
  const i18n = createI18n({ legacy: false, locale, fallbackLocale: 'vi', messages: { vi: viMessages, en: enMessages } })
  return [pinia, router, vuetify, i18n] as never[]
}

async function mountApp(pinia: ReturnType<typeof createPinia>, router: unknown, locale = 'vi') {
  const App = (await import('../App.vue')).default
  return mount(App, {
    global: {
      plugins: plugins(pinia, router, locale),
      stubs: {
        DefaultLayout: { template: '<div data-testid="default-layout"><slot /></div>' },
        AuthLayout: { template: '<div data-testid="auth-layout"><slot /></div>' },
        RouterView: { template: '<div data-testid="router-view" />' },
      },
    },
  })
}

beforeEach(() => {
  apiGet.mockReset()
  apiPost.mockReset()
  pending = []
  localStorage.clear()
  statusImpl = answer({ needs_setup: false })
  installApi()
})
afterEach(() => {
  localStorage.clear()
  vi.useRealTimers()
})

const seedCredentials = (auth?: { accessToken: string }) => {
  localStorage.setItem('cqa_access_token', TOKEN)
  localStorage.setItem('cqa_refresh_token', LEGACY)
  if (auth) auth.accessToken = TOKEN
}

describe('SS-01 pending status shows loading and opens nothing', () => {
  it('renders the localized loading state under the auth layout, no default layout, no profile or tenant calls', async () => {
    seedCredentials()
    statusImpl = deferStatus()
    const { router, pinia } = await freshApp()
    const w = await mountApp(pinia, router)
    const first = navigate(router, '/')
    const second = navigate(router, '/login') // concurrent navigation
    await flushPromises()
    expect(w.find('[data-testid="setup-loading"]').exists()).toBe(true)
    expect(w.text()).toContain(viMessages.setup_status_loading)
    expect(w.find('[data-testid="default-layout"]').exists()).toBe(false)
    expect(w.find('[data-testid="auth-layout"]').exists()).toBe(true)
    expect(urls()).toEqual(['/setup/status']) // one shared request, nothing else yet
    expect(apiPost).not.toHaveBeenCalled()

    pending[0].resolve({ data: { needs_setup: false } })
    await Promise.all([first, second])
    for (let i = 0; i < 5; i++) await flushPromises()
    expect(calls('/setup/status')).toBe(1)
    expect(w.find('[data-testid="setup-loading"]').exists()).toBe(false)
    w.unmount()
  })

  it('the loading text follows the locale', async () => {
    statusImpl = deferStatus()
    const { router, pinia } = await freshApp()
    const w = await mountApp(pinia, router, 'en')
    void navigate(router, '/')
    await flushPromises()
    expect(w.text()).toContain(enMessages.setup_status_loading)
    expect(enMessages.setup_status_loading).not.toBe(viMessages.setup_status_loading)
    pending[0].resolve({ data: { needs_setup: false } })
    await flushPromises()
    w.unmount()
  })
})

describe('SS-02 an unconfirmed status settles unavailable without side effects', () => {
  const failures: Array<[string, () => Promise<unknown>]> = [
    ['network rejection', () => Promise.reject(new Error('Network Error'))],
    ['axios timeout rejection', () => Promise.reject({ code: 'ECONNABORTED', message: 'timeout of 8000ms exceeded' })],
    ['HTTP 401', () => Promise.reject({ response: { status: 401 } })],
    ['HTTP 500', () => Promise.reject({ response: { status: 500, data: { error: 'SECRET-SERVER-TEXT' } } })],
    ['null body', () => Promise.resolve({ data: null })],
    ['undefined response', () => Promise.resolve(undefined)],
    ['empty object', () => Promise.resolve({ data: {} })],
    ['string payload', () => Promise.resolve({ data: 'needs_setup' })],
    ['array payload', () => Promise.resolve({ data: [{ needs_setup: true }] })],
    ['needs_setup as string', () => Promise.resolve({ data: { needs_setup: 'true' } })],
    ['needs_setup as string false', () => Promise.resolve({ data: { needs_setup: 'false' } })],
    ['needs_setup as number', () => Promise.resolve({ data: { needs_setup: 1 } })],
    ['needs_setup as zero', () => Promise.resolve({ data: { needs_setup: 0 } })],
    ['needs_setup null', () => Promise.resolve({ data: { needs_setup: null } })],
  ]
  for (const [label, impl] of failures) {
    it(`${label}: every entry lands on the unavailable page and credentials stay`, async () => {
      statusImpl = impl
      const { router, pinia, status, useAuthStore } = await freshApp()
      const auth = useAuthStore()
      seedCredentials(auth)
      for (const entry of ['/setup', '/login', '/', '/tenant-x/jobs', '/setup-unavailable']) {
        const outcome = await navigate(router, entry)
        expect(outcome.kind).toBe('done')
        expect(router.currentRoute.value.name).toBe('setup-unavailable')
      }
      expect(status.setupStatus.state).toBe('unavailable')
      expect(urls()).toEqual(['/setup/status']) // no profile, tenant, permission or refresh request
      expect(apiPost).not.toHaveBeenCalled()
      expect(localStorage.getItem('cqa_access_token')).toBe(TOKEN)
      expect(localStorage.getItem('cqa_refresh_token')).toBe(LEGACY)
      expect(auth.accessToken).toBe(TOKEN)
      void pinia
    })
  }

  it('a request that never settles ends unavailable within the status timeout and is aborted', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    statusImpl = deferStatus() // never answered
    const { router, status } = await freshApp()
    seedCredentials()
    const nav = navigate(router, '/')
    await vi.advanceTimersByTimeAsync(0)
    expect(pending).toHaveLength(1)
    const cfg = pending[0].cfg!
    expect(cfg.timeout).toBeGreaterThan(0)
    expect(cfg.timeout!).toBeLessThanOrEqual(10_000)
    expect(status.SETUP_STATUS_TIMEOUT_MS).toBeLessThanOrEqual(10_000)
    expect(cfg.signal!.aborted).toBe(false)
    await vi.advanceTimersByTimeAsync(status.SETUP_STATUS_TIMEOUT_MS + 1)
    // assert before awaiting the navigation so a missing deadline fails here, not by hanging
    expect(status.setupStatus.state).toBe('unavailable')
    const outcome = await nav
    expect(outcome.kind).toBe('done')
    expect(router.currentRoute.value.name).toBe('setup-unavailable')
    expect(cfg.signal!.aborted).toBe(true)
    expect(localStorage.getItem('cqa_access_token')).toBe(TOKEN)
    // a late answer after the deadline is ignored
    pending[0].resolve({ data: { needs_setup: false } })
    await flushPromises()
    expect(status.setupStatus.state).toBe('unavailable')
  })

  it('real client + mocked adapter: an HTTP 401 on /setup/status triggers no refresh, redirect or token change', async () => {
    const real = (await vi.importActual('../api')) as { default: { get: (u: string, c?: unknown) => Promise<unknown>; defaults: { adapter: unknown } } }
    const seen: string[] = []
    const deny = (config: { url?: string }) => {
      seen.push(config.url || '')
      return Promise.reject(new AxiosError('unauthorized', 'ERR_BAD_REQUEST', config as never, null, { status: 401, statusText: 'Unauthorized', data: {}, headers: {}, config: config as never }))
    }
    real.default.defaults.adapter = deny
    const savedGlobal = axios.defaults.adapter
    axios.defaults.adapter = deny as never
    try {
      statusImpl = (cfg) => real.default.get('/setup/status', cfg)
      const { router } = await freshApp()
      seedCredentials()
      const hrefBefore = window.location.href
      await navigate(router, '/')
      expect(router.currentRoute.value.name).toBe('setup-unavailable')
      expect(seen).toEqual(['/setup/status']) // exactly one request, no /auth/refresh
      expect(window.location.href).toBe(hrefBefore)
      expect(localStorage.getItem('cqa_access_token')).toBe(TOKEN)
    } finally {
      axios.defaults.adapter = savedGlobal
    }
  })
})

describe('SS-03 the unavailable view and explicit single-flight Retry', () => {
  async function unavailableView(locale: string) {
    statusImpl = () => Promise.reject(new Error('network'))
    const ctx = await freshApp()
    const auth = ctx.useAuthStore()
    seedCredentials(auth)
    await navigate(ctx.router, '/')
    expect(ctx.router.currentRoute.value.name).toBe('setup-unavailable')
    const View = (await import('../views/SetupStatusUnavailable.vue')).default
    const w = mount(View, { global: { plugins: plugins(ctx.pinia, ctx.router, locale) } })
    return { ...ctx, auth, w }
  }

  for (const [locale, msgs] of [['vi', viMessages], ['en', enMessages]] as const) {
    it(`${locale}: clear message, accessible Retry, no raw error or form`, async () => {
      const { w } = await unavailableView(locale)
      const text = w.text()
      expect(text).toContain(msgs.setup_status_unavailable_title)
      expect(text).toContain(msgs.setup_status_unavailable_desc)
      const btn = w.find('[data-testid="setup-retry"]')
      expect(btn.exists()).toBe(true)
      expect(btn.element.tagName).toBe('BUTTON')
      expect(btn.text()).toBe(msgs.setup_status_retry)
      expect(btn.attributes('disabled')).toBeUndefined()
      expect(w.find('[role="alert"]').exists()).toBe(true)
      expect(w.find('input').exists()).toBe(false)
      expect(text).not.toMatch(/network|SECRET|axios|stack/i)
      w.unmount()
    })
  }

  it('Retry is visibly disabled while deferred; a double activation makes exactly one new request', async () => {
    const { w, status, auth } = await unavailableView('vi')
    expect(calls('/setup/status')).toBe(1)
    statusImpl = deferStatus()
    const btn = w.find('[data-testid="setup-retry"]')
    await btn.trigger('click')
    await flushPromises()
    expect(calls('/setup/status')).toBe(2)
    expect(status.setupStatus.retrying).toBe(true)
    const busy = w.find('[data-testid="setup-retry"]')
    expect(busy.attributes('disabled')).toBeDefined()
    expect(busy.attributes('aria-busy')).toBe('true')
    expect(busy.text()).toBe(viMessages.setup_status_retrying)
    await busy.trigger('click')
    void status.retrySetupStatus() // another caller while in flight shares the same request
    await flushPromises()
    expect(calls('/setup/status')).toBe(2)

    pending[0].reject(new Error('still down'))
    await flushPromises()
    await flushPromises()
    expect(status.setupStatus.state).toBe('unavailable')
    expect(status.setupStatus.retrying).toBe(false)
    const again = w.find('[data-testid="setup-retry"]')
    expect(again.attributes('disabled')).toBeUndefined()
    expect(again.text()).toBe(viMessages.setup_status_retry)
    expect(localStorage.getItem('cqa_access_token')).toBe(TOKEN)
    expect(auth.accessToken).toBe(TOKEN)
    expect(urls().filter((u) => u !== '/setup/status')).toEqual([])
    w.unmount()
  })

  it('repeated navigation never retries on its own; only the explicit action does', async () => {
    const { router, status } = await freshApp()
    statusImpl = () => Promise.reject(new Error('network'))
    seedCredentials()
    for (const entry of ['/', '/login', '/setup', '/', '/x/jobs']) await navigate(router, entry)
    expect(calls('/setup/status')).toBe(1)
    await status.retrySetupStatus()
    expect(calls('/setup/status')).toBe(2)
    expect(router.currentRoute.value.name).toBe('setup-unavailable')
  })

  it('a failing Retry repeated several times is handled without unhandled rejections', async () => {
    const { w, status } = await unavailableView('en')
    for (let i = 0; i < 3; i++) {
      await w.find('[data-testid="setup-retry"]').trigger('click')
      await flushPromises()
      await flushPromises()
      expect(status.setupStatus.state).toBe('unavailable')
    }
    expect(calls('/setup/status')).toBe(4)
    w.unmount()
  })
})

describe('SS-04 Retry confirms Setup is required', () => {
  it('clears the stale local session through the existing AUTH-001 path and settles on Setup', async () => {
    statusImpl = () => Promise.reject(new Error('network'))
    const { router, status, useAuthStore } = await freshApp()
    const auth = useAuthStore()
    seedCredentials(auth)
    await navigate(router, '/')
    expect(router.currentRoute.value.name).toBe('setup-unavailable')
    statusImpl = answer({ needs_setup: true })
    tokenReads = 0
    armed = true
    try {
      expect(await status.retrySetupStatus()).toBe('required')
      await router.replace('/')
    } finally {
      armed = false
    }
    expect(router.currentRoute.value.name).toBe('setup')
    expect(localStorage.getItem('cqa_access_token')).toBeNull()
    expect(localStorage.getItem('cqa_refresh_token')).toBeNull()
    expect(auth.accessToken).toBe('')
    expect(auth.user).toBeNull()
    expect(calls('/profile')).toBe(0)
    expect(apiPost).not.toHaveBeenCalled()
  })

  it('also through the view: the Retry button ends on Setup without a loop', async () => {
    statusImpl = () => Promise.reject(new Error('network'))
    const { router, pinia, useAuthStore } = await freshApp()
    const auth = useAuthStore()
    seedCredentials(auth)
    await navigate(router, '/login')
    const View = (await import('../views/SetupStatusUnavailable.vue')).default
    const w = mount(View, { global: { plugins: plugins(pinia, router) } })
    statusImpl = answer({ needs_setup: true })
    await w.find('[data-testid="setup-retry"]').trigger('click')
    for (let i = 0; i < 5; i++) await flushPromises()
    expect(router.currentRoute.value.name).toBe('setup')
    expect(auth.accessToken).toBe('')
    expect(calls('/profile')).toBe(0)
    w.unmount()
  })
})

describe('SS-05 confirmed configured states', () => {
  it('Retry false with a token continues to the normal entry route; without one to Login', async () => {
    for (const withToken of [true, false]) {
      apiGet.mockClear()
      localStorage.clear()
      statusImpl = () => Promise.reject(new Error('network'))
      const { router, status } = await freshApp()
      if (withToken) localStorage.setItem('cqa_access_token', TOKEN)
      await navigate(router, '/')
      statusImpl = answer({ needs_setup: false })
      expect(await status.retrySetupStatus()).toBe('configured')
      await router.replace('/')
      expect(router.currentRoute.value.name).toBe(withToken ? 'tenants' : 'login')
      if (withToken) expect(localStorage.getItem('cqa_access_token')).toBe(TOKEN)
    }
  })

  it('leaving /setup-unavailable after confirmation uses the fixed local route, no return URL', async () => {
    statusImpl = () => Promise.reject(new Error('network'))
    const { router, status } = await freshApp()
    await navigate(router, '/some/deep/path?next=https://evil.invalid')
    expect(router.currentRoute.value.name).toBe('setup-unavailable')
    expect(router.currentRoute.value.fullPath).toBe('/setup-unavailable')
    statusImpl = answer({ needs_setup: false })
    await status.retrySetupStatus()
    await navigate(router, '/login') // leave the page first so the next entry is a real navigation
    expect(router.currentRoute.value.name).toBe('login')
    await navigate(router, '/setup-unavailable?next=https://evil.invalid')
    expect(router.currentRoute.value.fullPath).toBe('/login')
  })

  it('an App already mounted on the unavailable page resumes the profile load once, despite concurrent notifications', async () => {
    statusImpl = () => Promise.reject(new Error('network'))
    const { router, pinia, status, useAuthStore } = await freshApp()
    const auth = useAuthStore()
    seedCredentials(auth)
    const w = await mountApp(pinia, router)
    await navigate(router, '/')
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(router.currentRoute.value.name).toBe('setup-unavailable')
    expect(w.find('[data-testid="default-layout"]').exists()).toBe(false)
    expect(calls('/profile')).toBe(0)

    statusImpl = answer({ needs_setup: false })
    await Promise.all([status.retrySetupStatus(), status.retrySetupStatus(), status.retrySetupStatus()])
    status.markSetupConfigured() // a further notification of the same state
    await router.replace('/')
    for (let i = 0; i < 5; i++) await flushPromises()
    expect(calls('/profile')).toBe(1)
    expect(auth.user?.email).toBe('owner@example.invalid')
    expect(w.find('[data-testid="default-layout"]').exists()).toBe(true)
    w.unmount()
  })

  it('initially configured loads the profile once after the first navigation, never before', async () => {
    statusImpl = deferStatus()
    const { router, pinia, useAuthStore } = await freshApp()
    const auth = useAuthStore()
    seedCredentials(auth)
    const w = await mountApp(pinia, router)
    const nav = navigate(router, '/')
    for (let i = 0; i < 3; i++) await flushPromises()
    expect(calls('/profile')).toBe(0)
    pending[0].resolve({ data: { needs_setup: false } })
    await nav
    for (let i = 0; i < 5; i++) await flushPromises()
    expect(calls('/profile')).toBe(1)
    expect(urls()[0]).toBe('/setup/status')
    w.unmount()
  })

  it('confirmed answers are cached: later navigations never ask again, for both outcomes', async () => {
    for (const needs of [false, true]) {
      apiGet.mockClear()
      localStorage.clear()
      statusImpl = answer({ needs_setup: needs })
      const { router } = await freshApp()
      for (const entry of ['/', '/login', '/setup', '/']) await navigate(router, entry)
      expect(calls('/setup/status')).toBe(1)
    }
  })

  it('successful Setup afterwards marks configured with no further status request', async () => {
    statusImpl = answer({ needs_setup: true })
    const { router, status, markSetupComplete } = await freshApp()
    await navigate(router, '/setup')
    expect(router.currentRoute.value.name).toBe('setup')
    markSetupComplete()
    expect(status.setupStatus.state).toBe('configured')
    await navigate(router, '/login')
    expect(calls('/setup/status')).toBe(1)
  })
})
