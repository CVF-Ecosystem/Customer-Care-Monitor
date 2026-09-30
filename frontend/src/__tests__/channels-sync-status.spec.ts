// @vitest-environment happy-dom
// CCMAI-UX-017: synthetic API responses verify Channels sync-status presentation only.
// Nothing here is live-sync or CVF governance evidence.
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

const apiGet = vi.fn()
const apiPost = vi.fn()
vi.mock('../api', () => ({ default: { get: (...a: unknown[]) => apiGet(...a), post: (...a: unknown[]) => apiPost(...a), put: vi.fn(), delete: vi.fn() } }))

import Channels from '../views/Channels.vue'
import ChannelDetail from '../views/Channels/ChannelDetail.vue'

const g = globalThis as unknown as { visualViewport?: unknown }
g.visualViewport ??= { width: 1280, height: 800, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1, addEventListener() {}, removeEventListener() {} }

const OLD_SUCCESS = '2026-08-01T03:04:00Z'
type Ch = Record<string, unknown>
function channel(over: Ch = {}): Ch {
  return { id: 'c1', tenant_id: 't1', channel_type: 'facebook', name: 'Kênh mẫu', external_id: '', is_active: true, metadata: '{}', last_sync_at: null, last_sync_status: '', conversation_count: 3, created_at: '2026-07-01T00:00:00Z', ...over }
}

const mounted: VueWrapper[] = []
async function mountView(view: unknown, path: string, route: string, locale = 'vi') {
  setActivePinia(createPinia())
  useAuthStore().tenantPerms = { role: 'owner', permissions: {} } as never
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: route, component: view as never }] })
  await router.push(path)
  await router.isReady()
  const w = mount(view as never, {
    attachTo: document.body,
    global: { plugins: [
      createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } }),
      createI18n({ legacy: false, locale, fallbackLocale: 'en', messages: { vi: viMessages, en: enMessages } }),
      router,
    ] },
  })
  mounted.push(w)
  await flushPromises()
  return w
}
const mountList = (channels: Ch[], locale = 'vi') => {
  apiGet.mockImplementation((url: string) => url.endsWith('/channels') ? Promise.resolve({ data: channels }) : Promise.resolve({ data: {} }))
  return mountView(Channels, '/t1/channels', '/:tenantId/channels', locale)
}
const syncBtn = (w: VueWrapper) => w.findAll('button').find(b => b.text().includes('Đồng bộ ngay'))!

beforeEach(() => { apiGet.mockReset(); apiPost.mockReset() })
afterEach(() => {
  mounted.forEach(w => w.unmount())
  mounted.length = 0
  document.body.innerHTML = ''
  vi.useRealTimers()
})

describe('Channels list sync status (UX-017)', () => {
  it('shows all six states through the shared chip; empty is never a dash', async () => {
    const w = await mountList([
      channel({ id: 'a', last_sync_status: '' }),
      channel({ id: 'b', last_sync_status: 'syncing' }),
      channel({ id: 'c', last_sync_status: 'success', last_sync_at: OLD_SUCCESS }),
      channel({ id: 'd', last_sync_status: 'partial', last_sync_at: OLD_SUCCESS }),
      channel({ id: 'e', last_sync_status: 'error', last_sync_at: OLD_SUCCESS }),
      channel({ id: 'f', last_sync_status: 'surprise' }),
    ])
    const chips = w.findAll('.ccma-sync')
    expect(chips.map(c => c.attributes('data-sync'))).toEqual(['never', 'syncing', 'success', 'partial', 'error', 'unknown'])
    expect(chips.map(c => c.text())).toEqual(['Chưa đồng bộ', 'Đang đồng bộ', 'Đã đồng bộ', 'Đồng bộ một phần', 'Đồng bộ lỗi', 'Không rõ trạng thái'])
    // negative detectors: the legacy dash for empty status and a success reading of unknown must not return
    expect(chips[0].text()).not.toBe('—')
    expect(chips[5].text()).not.toBe('Đã đồng bộ')
  })

  it('labels the timestamp as last successful sync and keeps it beside a later partial/error', async () => {
    const w = await mountList([
      channel({ id: 'a', last_sync_status: '', last_sync_at: null }),
      channel({ id: 'b', last_sync_status: 'partial', last_sync_at: OLD_SUCCESS }),
      channel({ id: 'c', last_sync_status: 'error', last_sync_at: OLD_SUCCESS }),
      channel({ id: 'd', last_sync_status: 'syncing', last_sync_at: null }),
    ])
    const times = w.findAll('[data-test="last-success"]').map(t => t.text())
    const shown = new Date(OLD_SUCCESS).toLocaleString()
    expect(times).toEqual(['Chưa có lần đồng bộ thành công', shown, shown, 'Chưa có lần đồng bộ thành công'])
    expect(w.text()).toContain('Lần đồng bộ thành công')
    // negative detector: the old label implied the latest attempt
    expect(w.text()).not.toContain('Đồng bộ lần cuối')
    const partialCard = w.findAll('.v-card')[1]
    expect(partialCard.find('.ccma-sync').attributes('data-sync')).toBe('partial')
    expect(partialCard.text()).toContain(shown)
  })

  it('keeps activity and sync status independent for an inactive channel', async () => {
    const w = await mountList([channel({ is_active: false, last_sync_status: 'error', last_sync_at: OLD_SUCCESS })])
    const card = w.find('.v-card')
    expect(card.text()).toContain('Không hoạt động')
    expect(card.find('.ccma-sync').attributes('data-sync')).toBe('error')
  })

  it('renders English labels for the new text', async () => {
    const en = await mountList([channel({ last_sync_status: 'partial' })], 'en')
    expect(en.text()).toContain('Last successful sync')
    expect(en.text()).toContain('No successful sync yet')
    expect(en.find('.ccma-sync').text()).toBe('Partially synced')
    expect(en.text()).not.toContain('Last Synced')
  })

  it('says sync started (not finished) when the manual request is accepted', async () => {
    const w = await mountList([channel({ last_sync_status: 'success', last_sync_at: OLD_SUCCESS })])
    apiPost.mockResolvedValue({ data: { status: 'started' }, status: 202 })
    await syncBtn(w).trigger('click')
    await flushPromises()
    expect(apiPost).toHaveBeenCalledWith('/tenants/t1/channels/c1/sync')
    const text = document.body.textContent ?? ''
    expect(text).toContain('Đã bắt đầu đồng bộ')
    expect(text).not.toContain('Đồng bộ thành công')
  })

  it('shows reauth only as an optional neutral action after a generic error, within its channel-type boundary', async () => {
    const w = await mountList([
      channel({ id: 'f', channel_type: 'facebook', last_sync_status: 'error' }),
      channel({ id: 'z', channel_type: 'zalo_oa', last_sync_status: 'error' }),
      channel({ id: 'p', channel_type: 'pancake', last_sync_status: 'error' }),
      channel({ id: 's', channel_type: 'facebook', last_sync_status: 'success' }),
    ])
    const reauth = w.findAll('[data-test="reauth"]')
    expect(reauth).toHaveLength(2) // facebook and zalo error rows only; pancake and success rows have none
    for (const b of reauth) {
      expect(b.text()).toContain('tùy chọn')
      expect(b.text()).not.toBe('Kết nối lại')
      expect(b.classes().join(' ')).not.toMatch(/warning|error/)
      expect(b.attributes('title')).toContain('xác nhận')
    }
    const en = await mountList([channel({ last_sync_status: 'error' })], 'en')
    expect(en.find('[data-test="reauth"]').text()).toBe('Reconnect (optional)')
  })
})

describe('Channel detail sync status (UX-017)', () => {
  let current: Ch
  let onChannelGet: () => Promise<{ data: Ch }>
  async function mountDetail(ch: Ch, locale = 'vi') {
    current = ch
    onChannelGet = () => Promise.resolve({ data: current })
    apiGet.mockImplementation((url: string) => {
      if (url.endsWith('/sync-history')) return Promise.resolve({ data: { data: [], total: 0 } })
      return onChannelGet()
    })
    return mountView(ChannelDetail, '/t1/channels/c1', '/:tenantId/channels/:channelId', locale)
  }
  const chip = (w: VueWrapper) => w.find('.ccma-sync')

  it('uses the shared chip and successful-sync label with an older success beside partial/error', async () => {
    const shown = /^\d{2}\/\d{2}\/\d{4} \d{2}:\d{2}$/
    const rows = [['', null, 'never'], ['syncing', null, 'syncing'], ['success', OLD_SUCCESS, 'success'], ['partial', OLD_SUCCESS, 'partial'], ['error', OLD_SUCCESS, 'error'], ['surprise', null, 'unknown']] as const
    for (const [status, at, kind] of rows) {
      const w = await mountDetail(channel({ last_sync_status: status, last_sync_at: at }))
      expect(chip(w).attributes('data-sync')).toBe(kind)
      const time = w.find('[data-test="last-success"]').text()
      if (at) expect(time).toMatch(shown)
      else expect(time).toBe('Chưa có lần đồng bộ thành công')
      expect(w.text()).toContain('Lần đồng bộ thành công')
      expect(w.text()).not.toContain('Đồng bộ lần cuối')
      w.unmount()
      mounted.pop()
    }
  })

  it('renders English labels and keeps inactive independent of sync status', async () => {
    const w = await mountDetail(channel({ is_active: false, last_sync_status: 'error', last_sync_at: null }), 'en')
    expect(w.text()).toContain('Last successful sync')
    expect(w.find('[data-test="last-success"]').text()).toBe('No successful sync yet')
    expect(chip(w).attributes('data-sync')).toBe('error')
    expect(chip(w).text()).toBe('Sync failed')
  })

  it('reports an unconfirmed result, not a failure, when polling times out', async () => {
    vi.useFakeTimers()
    const w = await mountDetail(channel({ last_sync_status: 'success', last_sync_at: OLD_SUCCESS }))
    apiPost.mockResolvedValue({ data: {} })
    current = channel({ last_sync_status: 'syncing', last_sync_at: OLD_SUCCESS })
    await syncBtn(w).trigger('click')
    await vi.advanceTimersByTimeAsync(60 * 3000 + 100)
    await flushPromises()
    const alert = w.find('.v-alert')
    expect(alert.text()).toContain('Chưa xác nhận được kết quả đồng bộ')
    expect(alert.text()).not.toContain('Đồng bộ thất bại')
    expect(alert.text()).not.toContain('Đồng bộ thành công')
    expect(alert.classes().join(' ')).toMatch(/warning/)
    expect(chip(w).attributes('data-sync')).toBe('syncing')
  })

  it('reports an unconfirmed result when a refresh fails while polling', async () => {
    vi.useFakeTimers()
    const w = await mountDetail(channel({ last_sync_status: 'success', last_sync_at: OLD_SUCCESS }))
    apiPost.mockResolvedValue({ data: {} })
    onChannelGet = () => Promise.reject(new Error('network down'))
    await syncBtn(w).trigger('click')
    await vi.advanceTimersByTimeAsync(3100)
    await flushPromises()
    const alert = w.find('.v-alert')
    expect(alert.text()).toContain('Chưa xác nhận được kết quả đồng bộ')
    expect(alert.text()).not.toContain('Đồng bộ thất bại')
    expect(alert.classes().join(' ')).toMatch(/warning/)
  })

  it('keeps observed terminal success, partial and error distinct', async () => {
    const cases = [
      ['success', 'Đồng bộ thành công', /success/],
      ['partial', 'Đồng bộ hoàn tất một phần', /warning/],
      ['error', 'Đồng bộ thất bại', /error/],
    ] as const
    for (const [status, message, cls] of cases) {
      vi.useFakeTimers()
      const w = await mountDetail(channel({ last_sync_status: 'syncing', last_sync_at: OLD_SUCCESS }))
      apiPost.mockResolvedValue({ data: {} })
      current = channel({ last_sync_status: status, last_sync_at: OLD_SUCCESS })
      await syncBtn(w).trigger('click')
      await vi.advanceTimersByTimeAsync(3100)
      await flushPromises()
      const alert = w.find('.v-alert')
      expect(alert.text()).toContain(message)
      expect(alert.text()).not.toContain('Chưa xác nhận')
      expect(alert.classes().join(' ')).toMatch(cls)
      expect(chip(w).attributes('data-sync')).toBe(status)
      w.unmount()
      mounted.pop()
      vi.useRealTimers()
    }
  })

  it('treats an empty, null, never or unknown poll response as unconfirmed, not a failure', async () => {
    for (const status of ['', null, 'never', 'surprise']) {
      vi.useFakeTimers()
      const w = await mountDetail(channel({ last_sync_status: 'syncing', last_sync_at: OLD_SUCCESS }))
      apiPost.mockResolvedValue({ data: {} })
      current = channel({ last_sync_status: status, last_sync_at: OLD_SUCCESS })
      apiGet.mockClear()
      await syncBtn(w).trigger('click')
      await vi.advanceTimersByTimeAsync(3100)
      await flushPromises()
      const alert = w.find('.v-alert')
      expect(alert.text(), `status=${String(status)}`).toContain('Chưa xác nhận được kết quả đồng bộ')
      expect(alert.text()).not.toContain('Đồng bộ thất bại')
      expect(alert.text()).not.toContain('Đồng bộ thành công')
      expect(alert.classes().join(' ')).toMatch(/warning/)
      // polling stopped after the first non-terminal answer: one channel refresh, no history fetch
      const urls = apiGet.mock.calls.map(c => String(c[0]))
      expect(urls.filter(u => u.endsWith('/channels/c1'))).toHaveLength(1)
      expect(urls.some(u => u.endsWith('/sync-history'))).toBe(false)
      await vi.advanceTimersByTimeAsync(10000)
      expect(apiGet.mock.calls.filter(c => String(c[0]).endsWith('/channels/c1'))).toHaveLength(1)
      w.unmount()
      mounted.pop()
      vi.useRealTimers()
    }
  })

  it('keeps polling while the status is syncing and then reports the observed terminal outcome', async () => {
    vi.useFakeTimers()
    const w = await mountDetail(channel({ last_sync_status: 'syncing', last_sync_at: OLD_SUCCESS }))
    apiPost.mockResolvedValue({ data: {} })
    await syncBtn(w).trigger('click')
    await vi.advanceTimersByTimeAsync(3100)
    expect(w.find('.v-alert').exists()).toBe(false)
    current = channel({ last_sync_status: 'error', last_sync_at: OLD_SUCCESS })
    await vi.advanceTimersByTimeAsync(3100)
    await flushPromises()
    expect(w.find('.v-alert').text()).toContain('Đồng bộ thất bại')
    expect(w.find('.v-alert').text()).not.toContain('Chưa xác nhận')
  })

  it('reports a rejected start request as an error, without polling', async () => {
    const w = await mountDetail(channel({ last_sync_status: 'success', last_sync_at: OLD_SUCCESS }))
    apiPost.mockRejectedValue({ response: { data: { error: 'sync_in_progress' } } })
    apiGet.mockClear()
    await syncBtn(w).trigger('click')
    await flushPromises()
    expect(w.find('.v-alert').text()).toContain('sync_in_progress')
    expect(w.find('.v-alert').text()).not.toContain('Chưa xác nhận')
    expect(apiGet).not.toHaveBeenCalled()
  })

  it('has no reauth control on the detail screen', async () => {
    const w = await mountDetail(channel({ last_sync_status: 'error' }))
    expect(w.find('[data-test="reauth"]').exists()).toBe(false)
  })
})
