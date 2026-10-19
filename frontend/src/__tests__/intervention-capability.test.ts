// @vitest-environment happy-dom
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { afterEach, describe, expect, it, vi } from 'vitest'
import Notice from '../components/InterventionCapabilityNotice.vue'

const response = (generation = false, rerun = false) => ({ ok: true, redirected: false, json: async () => ({ scope: 'synthetic_only', generation_configured: generation, rerun_configured: rerun }) })
const mountNotice = () => mount(Notice, { props: { tenantId: 'tenant-a', token: 'synthetic-session' }, global: {
  plugins: [createI18n({ legacy: false, locale: 'vi' })], stubs: { 'v-alert': { template: '<div role="status"><slot /></div>' } },
} })
afterEach(() => vi.unstubAllGlobals())
describe('intervention configuration notice', () => {
  it('shows disabled creation truth without disabling human case handling', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response()))
    const w = mountNotice(); await flushPromises()
    expect(w.text()).toContain('chưa được bật')
    expect(w.text()).toContain('xử lý các vụ việc hiện có')
    w.unmount()
  })
  it('labels a configured transport synthetic, not production ready', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(true, true)))
    const w = mountNotice(); await flushPromises()
    expect(w.text()).toContain('chỉ dành cho dữ liệu minh họa')
    expect(w.text()).toContain('hạn mức')
    w.unmount()
  })
  it('does not imply both features are enabled when only one is configured', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(true, false)))
    const w = mountNotice(); await flushPromises()
    expect(w.text()).toContain('cấu hình một phần')
    w.unmount()
  })
  it.each([401, 403, 503])('never reports enabled after HTTP %s', async status => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status }))
    const w = mountNotice(); await flushPromises()
    expect(w.text()).toContain('Chưa kiểm tra được')
    w.unmount()
  })
  it('rejects a capability from a different scope', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ scope: 'real', generation_configured: true, rerun_configured: true }) }))
    const w = mountNotice(); await flushPromises()
    expect(w.text()).toContain('Chưa kiểm tra được')
    w.unmount()
  })
  it('discards a late capability response after tenant/token change', async () => {
    let oldResolve!: (value: unknown) => void
    vi.stubGlobal('fetch', vi.fn().mockImplementationOnce(() => new Promise(resolve => { oldResolve = resolve })).mockResolvedValueOnce(response()))
    const w = mountNotice()
    await w.setProps({ tenantId: 'tenant-b', token: 'other-session' }); await flushPromises()
    oldResolve(response(true, true)); await flushPromises()
    expect(w.text()).toContain('chưa được bật')
    w.unmount()
  })
})
