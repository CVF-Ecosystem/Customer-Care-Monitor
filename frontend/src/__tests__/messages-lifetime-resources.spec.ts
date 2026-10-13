// @vitest-environment happy-dom
// CCMAI-RUNTIME-088 (F06) UI08: authenticated image fetches and object URLs follow the lifetime of
// the selection/tenant/mount that requested them. fetch, Blob bytes and URL.createObjectURL are
// synthetic; the real Messages.vue and store run. Nothing here is governance evidence.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import viMessages from '../i18n/vi'
import {
  apiGet, resetApi, standardRules, messagesCallFor, messagesPayload, imageAttachment,
  mountMessages, unmountOne, cleanup, settle, clickRow, hasDetail,
  spyObjectUrls, stubImageFetch, type BlobUrlSpy, type FetchStub,
} from './fixtures/conversation-lifetime/harness'

vi.mock('../api', () => ({ default: { get: (...args: unknown[]) => apiGet(...args), post: vi.fn(), delete: vi.fn(), put: vi.fn() } }))

vi.setConfig({ testTimeout: 30_000 })

const IMG = imageAttachment('shared/anh-mau.png')
const IMG_URL = '/api/v1/files/shared/anh-mau.png'
let urls: BlobUrlSpy
let net: FetchStub

beforeEach(() => {
  urls = spyObjectUrls()
  net = stubImageFetch()
  resetApi(standardRules())
})
afterEach(() => {
  cleanup()
  urls.restore()
  net.restore()
})

async function openWithImage(w: Awaited<ReturnType<typeof mountMessages>>['w'], row: string, convId: string, nth = 0, tenant = 't1') {
  await clickRow(w, row)
  messagesCallFor(tenant, convId)[nth].resolve(messagesPayload(convId, row, ['có ảnh'], 1, [IMG]))
  await settle()
}

describe('Messages image resources follow the lifetime that requested them (UI08)', () => {
  it('healthy control: an image loaded for the current selection becomes one live object URL shown in the transcript', async () => {
    const { w } = await mountMessages()
    await openWithImage(w, 'Khách A', 'cA')
    expect(net.calls).toHaveLength(1)
    expect(net.calls[0].url).toBe(IMG_URL)
    net.calls[0].resolveOk()
    await settle()
    expect(urls.live()).toHaveLength(1)
    expect(w.find('.mg-msg__img').attributes('src')).toBe(urls.live()[0])
  })

  it('unmount with a pending image fetch: the late result leaves no live object URL', async () => {
    const { w } = await mountMessages()
    await openWithImage(w, 'Khách A', 'cA')
    expect(net.calls).toHaveLength(1)
    unmountOne(w)
    net.calls[0].resolveOk()
    await settle(6)
    expect(urls.live(), 'an allocation after teardown would leak').toEqual([])
  })

  it('unmount while the response body is still downloading: finishing it leaves no live object URL', async () => {
    const { w } = await mountMessages()
    await openWithImage(w, 'Khách A', 'cA')
    const finishBody = net.calls[0].resolveOkWithPendingBlob()
    await settle()
    unmountOne(w)
    finishBody()
    await settle(6)
    expect(urls.live()).toEqual([])
  })

  it('reset (mobile back) with a pending image fetch: the late result leaves no live object URL', async () => {
    const { w } = await mountMessages({ width: 500 })
    await openWithImage(w, 'Khách A', 'cA')
    await w.find(`button[aria-label="${viMessages.msgs_back}"]`).trigger('click')
    await settle(2)
    expect(hasDetail(w)).toBe(false)
    net.calls[0].resolveOk()
    await settle(6)
    expect(urls.live(), 'a result for a reset lifetime must not stay allocated').toEqual([])
  })

  it('a later lifetime that needs the same URL loads it itself and the old fetch cannot own it', async () => {
    const { w } = await mountMessages({ width: 500 })
    await openWithImage(w, 'Khách A', 'cA')
    const oldFetch = net.calls[0]
    await w.find(`button[aria-label="${viMessages.msgs_back}"]`).trigger('click')
    await settle(2)
    await openWithImage(w, 'Khách B', 'cB')
    expect(net.calls, 'the new lifetime must issue its own fetch for the shared URL').toHaveLength(2)
    oldFetch.resolveOk('bytes-của-lần-cũ')
    await settle()
    expect(urls.live(), 'the old fetch must not allocate').toEqual([])
    net.calls[1].resolveOk('bytes-của-lần-mới')
    await settle()
    expect(urls.live()).toHaveLength(1)
    expect(w.find('.mg-msg__img').attributes('src')).toBe(urls.live()[0])
  })

  it('a stale failed fetch cannot delete the cache entry a newer lifetime already filled', async () => {
    const { w } = await mountMessages({ width: 500 })
    await openWithImage(w, 'Khách A', 'cA')
    const oldFetch = net.calls[0]
    await w.find(`button[aria-label="${viMessages.msgs_back}"]`).trigger('click')
    await settle(2)
    await openWithImage(w, 'Khách B', 'cB')
    net.calls[1].resolveOk()
    await settle()
    const active = urls.live()[0]
    expect(active).toBeDefined()
    oldFetch.resolveNotOk() // the previous lifetime fails late, for the very same URL
    await settle()
    expect(urls.live()).toEqual([active])
    expect(w.find('.mg-msg__img').attributes('src'), 'the current lifetime\'s image must stay').toBe(active)
  })

  it('tenant change releases the previous tenant\'s object URLs', async () => {
    const { w, router } = await mountMessages({ tenant: 't1' })
    await openWithImage(w, 'Khách A', 'cA')
    net.calls[0].resolveOk()
    await settle()
    expect(urls.live()).toHaveLength(1)
    await router.push('/t2/messages')
    await settle(6)
    expect(urls.live(), 'the old tenant\'s resource must be released').toEqual([])
  })

  it('tenant change with a pending image fetch: the old result allocates nothing', async () => {
    const { w, router } = await mountMessages({ tenant: 't1' })
    await openWithImage(w, 'Khách A', 'cA')
    await router.push('/t2/messages')
    await settle(6)
    net.calls[0].resolveOk()
    await settle(6)
    expect(urls.live()).toEqual([])
  })

  it('an active resource survives cleanup of an earlier selection (cleanup is not global)', async () => {
    const { w } = await mountMessages()
    await openWithImage(w, 'Khách A', 'cA')
    net.calls[0].resolveOk()
    await settle()
    const active = urls.live()[0]
    await clickRow(w, 'Khách B')
    messagesCallFor('t1', 'cB')[0].resolve(messagesPayload('cB', 'Khách B', ['cũng có ảnh'], 1, [IMG]))
    await settle()
    expect(urls.revoked, 'switching selection must not revoke a still-valid resource').not.toContain(active)
    expect(w.find('.mg-msg__img').attributes('src')).toBe(active)
    expect(net.calls, 'the cached image is reused, not fetched again').toHaveLength(1)
  })

  it('failed image fetch of the current lifetime stays retryable', async () => {
    const { w } = await mountMessages()
    await openWithImage(w, 'Khách A', 'cA')
    net.calls[0].resolveNotOk()
    await settle()
    expect(urls.live()).toEqual([])
    await clickRow(w, 'Khách B')
    messagesCallFor('t1', 'cB')[0].resolve(messagesPayload('cB', 'Khách B', ['ảnh'], 1, [IMG]))
    await settle()
    expect(net.calls, 'a failed load is retried when the image is needed again').toHaveLength(2)
  })

  it('unmount revokes every object URL the lifetime still owns', async () => {
    const { w } = await mountMessages()
    await openWithImage(w, 'Khách A', 'cA')
    net.calls[0].resolveOk()
    await settle()
    expect(urls.live()).toHaveLength(1)
    unmountOne(w)
    expect(urls.live()).toEqual([])
  })
})
