// @vitest-environment happy-dom
// CCMAI-AUTH-001: first-time Setup must not hang when the browser still holds a token from a
// previous installation. Uses the real application router guard and auth store with a mocked
// API; synthetic tokens only. Frontend routing evidence, not governance evidence.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

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

const STALE = 'stale-synthetic-token'
const FRESH = 'fresh-synthetic-token'

type Status = { needs_setup: boolean } | 'error'
let status: Status = { needs_setup: true }

function setupApi() {
  apiGet.mockImplementation((url: string) => {
    if (url === '/setup/status') return status === 'error' ? Promise.reject(new Error('network')) : Promise.resolve({ data: status })
    if (url === '/profile') {
      // Only the token issued by Setup is valid; the stale one is what a real server rejects.
      return localStorage.getItem('cqa_access_token') === FRESH
        ? Promise.resolve({ data: { id: 'u1', email: 'owner@example.invalid', name: 'Owner', is_admin: true, language: 'vi' } })
        : Promise.reject({ response: { status: 401 } })
    }
    if (url === '/tenants') return Promise.resolve({ data: [] })
    return Promise.resolve({ data: {} })
  })
  apiPost.mockImplementation((url: string) => {
    if (url === '/setup') return Promise.resolve({ data: { access_token: FRESH } })
    return Promise.reject(new Error(`unexpected POST ${url}`))
  })
}

// Fresh router module per test: the guard caches the setup status at module level.
async function freshApp() {
  vi.resetModules()
  const pinia = createPinia()
  setActivePinia(pinia)
  const routerMod = await import('../router')
  const { useAuthStore } = await import('../stores/auth')
  return { router: routerMod.default, markSetupComplete: routerMod.markSetupComplete, pinia, useAuthStore }
}

// A redirect loop runs entirely in microtasks, so it blocks the event loop and a timer could
// never fire (this is how the tab hangs). Bound it instead: the guard reads the token on every
// pass, so a navigation that reads it more than LOOP_LIMIT times throws and reports a loop.
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

const calls = (fn: typeof apiGet, url: string) => fn.mock.calls.filter((c) => c[0] === url).length

describe('Setup with a stale browser token (AUTH-001)', () => {
  beforeEach(() => {
    apiGet.mockReset()
    apiPost.mockReset()
    localStorage.clear()
    status = { needs_setup: true }
    setupApi()
  })
  afterEach(() => localStorage.clear())

  for (const entry of ['/setup', '/login', '/']) {
    it(`needs_setup=true with a stale token: entering ${entry} settles on Setup without a loop`, async () => {
      localStorage.setItem('cqa_access_token', STALE)
      localStorage.setItem('cqa_refresh_token', 'legacy-synthetic')
      const { router, useAuthStore } = await freshApp()
      const outcome = await navigate(router, entry)
      expect(outcome).toEqual({ kind: 'done', failure: undefined })
      expect(router.currentRoute.value.name).toBe('setup')
      // Old installation's credentials are gone from storage and from the store.
      expect(localStorage.getItem('cqa_access_token')).toBeNull()
      expect(localStorage.getItem('cqa_refresh_token')).toBeNull()
      const auth = useAuthStore()
      expect(auth.accessToken).toBe('')
      expect(auth.user).toBeNull()
      expect(calls(apiGet, '/setup/status')).toBe(1)
      expect(calls(apiGet, '/profile')).toBe(0)
    })
  }

  it('App does not load a profile with the stale token before the setup status is known', async () => {
    localStorage.setItem('cqa_access_token', STALE)
    const { router, pinia } = await freshApp()
    const App = (await import('../App.vue')).default
    const w = mount(App, {
      global: {
        plugins: [pinia, router],
        stubs: { 'v-app': { template: '<div><slot /></div>' }, AuthLayout: { template: '<div><slot /></div>' }, DefaultLayout: { template: '<div><slot /></div>' }, RouterView: true },
      },
    })
    const outcome = await navigate(router, '/setup')
    for (let i = 0; i < 5; i++) await flushPromises()
    expect(outcome).toEqual({ kind: 'done', failure: undefined })
    expect(router.currentRoute.value.name).toBe('setup')
    expect(calls(apiGet, '/profile')).toBe(0)
    w.unmount()
  })

  it('successful Setup keeps the new token, loads the profile and reaches the authenticated route', async () => {
    localStorage.setItem('cqa_access_token', STALE)
    const { router, markSetupComplete, useAuthStore } = await freshApp()
    await navigate(router, '/setup')
    const auth = useAuthStore()
    await auth.setup('Owner', 'owner@example.invalid', 'Abcdefg1', 'Cửa hàng mẫu')
    markSetupComplete()
    const outcome = await navigate(router, '/')
    expect(outcome.kind).toBe('done')
    expect(router.currentRoute.value.name).toBe('tenants')
    expect(localStorage.getItem('cqa_access_token')).toBe(FRESH)
    expect(auth.accessToken).toBe(FRESH)
    expect(auth.user?.email).toBe('owner@example.invalid')
  })

  it('configured installation (needs_setup=false): guest and protected routes behave as before', async () => {
    status = { needs_setup: false }
    localStorage.setItem('cqa_access_token', 'valid-synthetic-token')
    let app = await freshApp()
    await navigate(app.router, '/login')
    expect(app.router.currentRoute.value.name).toBe('tenants')
    await navigate(app.router, '/setup')
    expect(app.router.currentRoute.value.name).toBe('tenants')
    expect(localStorage.getItem('cqa_access_token')).toBe('valid-synthetic-token')

    localStorage.clear()
    app = await freshApp()
    await navigate(app.router, '/')
    expect(app.router.currentRoute.value.name).toBe('login')
    await navigate(app.router, '/setup')
    expect(app.router.currentRoute.value.name).toBe('login')
  })

  it('a failed setup-status request neither clears the credential nor admits Setup', async () => {
    status = 'error'
    localStorage.setItem('cqa_access_token', 'valid-synthetic-token')
    const { router } = await freshApp()
    await navigate(router, '/setup')
    expect(router.currentRoute.value.name).not.toBe('setup')
    expect(localStorage.getItem('cqa_access_token')).toBe('valid-synthetic-token')
  })
})
