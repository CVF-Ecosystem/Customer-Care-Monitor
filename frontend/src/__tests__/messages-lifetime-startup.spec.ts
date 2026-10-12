// @vitest-environment happy-dom
// CCMAI-RUNTIME-088 (F06) UI06-UI07: list/filter/search request ordering, the tenant-bound status
// map, and mounted initialization/deep-link continuations after their lifetime ended. Real
// Messages.vue and store; synthetic deferred API only. Nothing here is governance evidence.
import { describe, it, expect, vi, afterEach } from 'vitest'
import {
  apiGet, resetApi, standardRules, URLS, requested, issued, messagesCallFor, evaluationsCallFor,
  convRow, listPayload, messagesPayload, evaluationPayload, calls,
  mountMessages, unmountOne, cleanup, settle, wait,
  rows, listErrorShown, chipTexts, qcPane, qcText, hasDetail, detailTitle, clickRow,
} from './fixtures/conversation-lifetime/harness'
import viMessages from '../i18n/vi'

vi.mock('../api', () => ({ default: { get: (...args: unknown[]) => apiGet(...args), post: vi.fn(), delete: vi.fn(), put: vi.fn() } }))

vi.setConfig({ testTimeout: 30_000 })

const listRule = { test: URLS.list, mode: 'deferred' as const }
const SEARCH = 'input[aria-label="Tìm tên khách"]'
const listCallsDeferred = () => calls.filter((c) => URLS.list.test(c.url))
const rowNames = (w: Parameters<typeof rows>[0]) => rows(w).map((r) => r.text()).join('|')

async function typeSearch(w: Parameters<typeof rows>[0], text: string) {
  await w.find(SEARCH).setValue(text)
  await wait(350)
  await settle(2)
}

describe('Messages list: the latest list request wins (UI06)', () => {
  afterEach(cleanup)

  it('reversed success: an older (initial) list reply cannot restore the previous query\'s rows', async () => {
    resetApi(standardRules([listRule]))
    const { w } = await mountMessages()
    const [initial] = listCallsDeferred()
    expect(initial).toBeDefined()
    await typeSearch(w, 'Hai')
    const searched = listCallsDeferred()[1]
    expect(searched).toBeDefined()
    expect((searched.config as { params: Record<string, unknown> }).params.search).toBe('Hai')
    searched.resolve(listPayload([convRow('s1', 'Kết quả tìm')], 1))
    await settle()
    expect(rowNames(w)).toContain('Kết quả tìm')
    initial.resolve(listPayload([convRow('i1', 'Danh sách đầu')], 50))
    await settle()
    expect(rowNames(w), 'late older list must not publish').toContain('Kết quả tìm')
    expect(rowNames(w)).not.toContain('Danh sách đầu')
    expect(w.find('[data-testid="msgs-subtitle"]').text(), 'total must follow the winning request').toContain('1 hội thoại')
  })

  it('a late rejection of an older list request shows no list error over the newer success', async () => {
    resetApi(standardRules([listRule]))
    const { w } = await mountMessages()
    const [initial] = listCallsDeferred()
    await typeSearch(w, 'Hai')
    listCallsDeferred()[1].resolve(listPayload([convRow('s1', 'Kết quả tìm')], 1))
    await settle()
    initial.reject(new Error('lỗi muộn của yêu cầu cũ'))
    await settle()
    expect(listErrorShown(w), 'stale catch must not become the list error').toBe(false)
    expect(rowNames(w)).toContain('Kết quả tìm')
  })

  it('an older request settling while a newer one is pending neither publishes nor ends the busy state', async () => {
    resetApi(standardRules([listRule]))
    const { w } = await mountMessages()
    listCallsDeferred()[0].resolve(listPayload([convRow('i1', 'Danh sách đầu')], 50))
    await settle()
    expect(rowNames(w)).toContain('Danh sách đầu')
    await typeSearch(w, 'a')
    await typeSearch(w, 'ab')
    const all = listCallsDeferred()
    expect(all.length).toBeGreaterThanOrEqual(3)
    const [, stale, current] = all
    stale.resolve(listPayload([convRow('s1', 'Kết quả a')], 5))
    await settle()
    expect(rowNames(w), 'stale list must not replace rows').not.toContain('Kết quả a')
    expect(w.find('.mg-rows--busy').exists(), 'newer request is still loading').toBe(true)
    current.resolve(listPayload([convRow('s2', 'Kết quả ab')], 2))
    await settle()
    expect(rowNames(w)).toContain('Kết quả ab')
    expect(w.find('.mg-rows--busy').exists()).toBe(false)
  })

  it('healthy control: a current list failure shows the alert and retry recovers', async () => {
    resetApi(standardRules([listRule]))
    const { w } = await mountMessages()
    listCallsDeferred()[0].reject(new Error('lỗi hiện tại'))
    await settle()
    expect(listErrorShown(w)).toBe(true)
    await w.find('[data-testid="msgs-list-error"] button').trigger('click')
    await settle(2)
    listCallsDeferred()[1].resolve(listPayload([convRow('r1', 'Sau khi thử lại')], 1))
    await settle()
    expect(listErrorShown(w)).toBe(false)
    expect(rowNames(w)).toContain('Sau khi thử lại')
  })

  it('a valid selection survives a filter change of the same tenant', async () => {
    resetApi(standardRules())
    const { w } = await mountMessages()
    await w.findAll('[data-testid="msgs-row"]')[0].trigger('click')
    await settle(2)
    messagesCallFor('t1', 'cA')[0].resolve(messagesPayload('cA', 'Khách A', ['TIN-A-1'], 3))
    await settle()
    await typeSearch(w, 'zzz')
    expect(hasDetail(w), 'a list/search change must not clear the selected conversation').toBe(true)
    expect(detailTitle(w)).toContain('Khách A')
  })
})

describe('Messages status map is tenant-bound (UI06)', () => {
  afterEach(cleanup)

  it('t1 -> t2: a late t1 map reply cannot overwrite the t2 map', async () => {
    resetApi(standardRules([{ test: URLS.evaluated, mode: 'deferred' }], [convRow('c1', 'Khách chung')]))
    const { w, router } = await mountMessages({ tenant: 't1' })
    const t1Map = calls.filter((c) => c.url === '/tenants/t1/conversations/evaluated')[0]
    expect(t1Map).toBeDefined()
    await router.push('/t2/messages')
    await settle(6)
    const t2Map = calls.filter((c) => c.url === '/tenants/t2/conversations/evaluated')[0]
    expect(t2Map, 'the new tenant must load its own status map').toBeDefined()
    t2Map.resolve({ c1: 'PASS' })
    await settle()
    expect(chipTexts(w)).toEqual([viMessages.msgs_chip_pass])
    t1Map.resolve({ c1: 'FAIL' })
    await settle()
    expect(chipTexts(w), 'late t1 map must not relabel t2').toEqual([viMessages.msgs_chip_pass])
  })

  it('t1 -> t2: a late t1 map failure cannot turn the t2 chips into "unknown"', async () => {
    resetApi(standardRules([{ test: URLS.evaluated, mode: 'deferred' }], [convRow('c1', 'Khách chung')]))
    const { w, router } = await mountMessages({ tenant: 't1' })
    const t1Map = calls.filter((c) => c.url === '/tenants/t1/conversations/evaluated')[0]
    await router.push('/t2/messages')
    await settle(6)
    calls.filter((c) => c.url === '/tenants/t2/conversations/evaluated')[0].resolve({ c1: 'PASS' })
    await settle()
    t1Map.reject(new Error('lỗi map muộn'))
    await settle()
    expect(chipTexts(w)).toEqual([viMessages.msgs_chip_pass])
  })
})

describe('Messages initialization and deep link stop when their lifetime ended (UI07)', () => {
  afterEach(cleanup)

  it('unmount while the channel list is pending: no list request follows', async () => {
    resetApi(standardRules([{ test: URLS.channels, mode: 'deferred' }]))
    const { w } = await mountMessages()
    expect(requested(URLS.list)).toHaveLength(0)
    unmountOne(w)
    calls.filter((c) => URLS.channels.test(c.url))[0].resolve([])
    await settle(6)
    expect(requested(URLS.list), 'a dead lifetime must not continue its initialization').toHaveLength(0)
  })

  it('t1 -> t2 -> t1: the first t1 lifetime does not continue although the tenant string matches again', async () => {
    resetApi(standardRules([{ test: URLS.channels, mode: 'deferred' }]))
    const { router } = await mountMessages({ tenant: 't1' })
    await router.push('/t2/messages')
    await settle(4)
    await router.push('/t1/messages')
    await settle(4)
    const channelCalls = calls.filter((c) => URLS.channels.test(c.url))
    expect(channelCalls.map((c) => c.url)).toEqual(['/tenants/t1/channels', '/tenants/t2/channels', '/tenants/t1/channels'])
    channelCalls[0].resolve([]) // the oldest lifetime finishes first
    await settle(6)
    expect(requested(URLS.list), 'a finished lifetime must not continue').toHaveLength(0)
    channelCalls[2].resolve([])
    await settle(6)
    expect(requested(URLS.list).map((r) => r.url)).toEqual(['/tenants/t1/conversations'])
  })

  it('unmount while the first list is pending: no status-map request follows', async () => {
    resetApi(standardRules([listRule]))
    const { w } = await mountMessages()
    unmountOne(w)
    listCallsDeferred()[0].resolve(listPayload([convRow('i1', 'Danh sách đầu')], 1))
    await settle(6)
    expect(requested(URLS.evaluated), 'initialization must stop after unmount').toHaveLength(0)
  })

  it('unmount at the deep-link page lookup: the old conversation is never selected', async () => {
    resetApi(standardRules([{ test: URLS.page, mode: 'deferred' }]))
    const { w } = await mountMessages({ query: '?conv=cA&tab=qc' })
    const lookup = calls.filter((c) => URLS.page.test(c.url))[0]
    expect(lookup).toBeDefined()
    unmountOne(w)
    lookup.resolve({ page: 1 })
    await settle(6)
    expect(issued(URLS.messages), 'no stale detail chain may start for any tenant').toHaveLength(0)
  })

  it('unmount at a deep-link lookup that points to another page: that page is never loaded', async () => {
    resetApi(standardRules([{ test: URLS.page, mode: 'deferred' }]))
    const { w } = await mountMessages({ query: '?conv=cA' })
    const lookup = calls.filter((c) => URLS.page.test(c.url))[0]
    const listBefore = requested(URLS.list).length
    unmountOne(w)
    lookup.resolve({ page: 2 })
    await settle(6)
    expect(requested(URLS.list).length, 'a dead lifetime must not reload a list page').toBe(listBefore)
    expect(issued(URLS.messages)).toHaveLength(0)
  })

  it('tenant change at the deep-link page lookup: the old tenant\'s continuation does not select', async () => {
    resetApi(standardRules([{ test: URLS.page, mode: 'deferred' }]))
    const { w, router } = await mountMessages({ tenant: 't1', query: '?conv=cA' })
    const lookup = calls.filter((c) => URLS.page.test(c.url))[0]
    await router.push('/t2/messages')
    await settle(6)
    expect(requested(URLS.list).some((r) => r.url === '/tenants/t2/conversations'), 'the new tenant must be loaded').toBe(true)
    lookup.resolve({ page: 1 })
    await settle(6)
    expect(issued(URLS.messages), 'no detail request for any tenant').toHaveLength(0)
    expect(hasDetail(w)).toBe(false)
  })

  it('healthy control: an active deep link selects the conversation on the requested tab', async () => {
    resetApi(standardRules())
    const { w } = await mountMessages({ query: '?conv=cB&tab=qc' })
    await settle(4)
    const detail = messagesCallFor('t1', 'cB')
    expect(detail).toHaveLength(1)
    detail[0].resolve(messagesPayload('cB', 'Khách B', ['TIN-B-1'], 5))
    await settle()
    evaluationsCallFor('t1', 'cB')[0].resolve(evaluationPayload('QC-CỦA-B'))
    await settle()
    expect(hasDetail(w)).toBe(true)
    expect(qcPane(w).exists(), 'tab=qc opens the evaluation tab').toBe(true)
    expect(qcText(w)).toContain('QC-CỦA-B')
  })

  it('healthy control: a deep link on another page reloads that page then selects', async () => {
    resetApi(standardRules([{ test: URLS.page, mode: 'auto', reply: () => ({ page: 2 }) }]))
    const { w } = await mountMessages({ query: '?conv=cA' })
    await settle(6)
    expect(requested(URLS.list).filter((r) => (r.config as { params: Record<string, number> }).params.page === 2).length).toBeGreaterThan(0)
    expect(messagesCallFor('t1', 'cA')).toHaveLength(1)
    expect(hasDetail(w)).toBe(true)
  })

  it('debounced search cannot publish after teardown', async () => {
    resetApi(standardRules())
    const { w } = await mountMessages()
    const before = requested(URLS.list).length
    await w.find(SEARCH).setValue('Hai')
    unmountOne(w)
    await wait(400)
    await settle(2)
    expect(requested(URLS.list).length, 'debounce timer must be cleared by teardown').toBe(before)
  })
})

// R088-R1-F01 (round1): a pending automatic deep link is not a user action. Once the user explicitly
// selects (any conversation, including the deep-linked one) or deselects (mobile back), the older
// initialization continuation must not change the page, select, or fetch anything for the old link.
describe('A pending deep link yields to a newer explicit selection or back (R088-R1-F01)', () => {
  afterEach(cleanup)

  const pageRule = { test: URLS.page, mode: 'deferred' as const }
  const lookupCalls = () => issued(URLS.page)
  const backButton = (w: Parameters<typeof rows>[0]) => w.find(`button[aria-label="${viMessages.msgs_back}"]`)
  const pageParams = () => requested(URLS.list).map((r) => (r.config as { params: Record<string, number> }).params.page)

  it('healthy control: an untouched pending deep link still selects A on the requested tab', async () => {
    resetApi(standardRules([pageRule]))
    const { w } = await mountMessages({ query: '?conv=cA&tab=qc' })
    lookupCalls()[0].resolve({ page: 1 })
    await settle()
    expect(messagesCallFor('t1', 'cA')).toHaveLength(1)
    messagesCallFor('t1', 'cA')[0].resolve(messagesPayload('cA', 'Khách A', ['TIN-A-1'], 3))
    await settle()
    evaluationsCallFor('t1', 'cA')[0].resolve(evaluationPayload('QC-CỦA-A'))
    await settle()
    expect(detailTitle(w)).toContain('Khách A')
    expect(qcText(w)).toContain('QC-CỦA-A')
  })

  it('healthy control: a page lookup failure with no user action falls back to the current page and still selects', async () => {
    resetApi(standardRules([pageRule]))
    const { w } = await mountMessages({ query: '?conv=cA' })
    lookupCalls()[0].reject(new Error('lỗi tra cứu trang'))
    await settle()
    expect(messagesCallFor('t1', 'cA')).toHaveLength(1)
    expect(hasDetail(w)).toBe(true)
    expect(detailTitle(w)).toContain('Khách A')
  })

  it('page success: selecting B while the lookup is pending keeps B and never fetches the old link', async () => {
    resetApi(standardRules([pageRule]))
    const { w } = await mountMessages({ query: '?conv=cA' })
    await clickRow(w, 'Khách B')
    lookupCalls()[0].resolve({ page: 1 })
    await settle()
    messagesCallFor('t1', 'cB')[0].resolve(messagesPayload('cB', 'Khách B', ['TIN-B-1'], 5))
    await settle()
    expect(detailTitle(w), 'header must stay on the explicit B selection').toContain('Khách B')
    expect(w.find('.mg-transcript').text()).toContain('TIN-B-1')
    expect(messagesCallFor('t1', 'cA'), 'no stale detail request for the old link').toHaveLength(0)
    expect(evaluationsCallFor('t1', 'cA'), 'no stale evaluation request for the old link').toHaveLength(0)
  })

  it('page success: explicitly selecting the deep-linked conversation itself is not selected a second time', async () => {
    resetApi(standardRules([pageRule]))
    const { w } = await mountMessages({ query: '?conv=cA' })
    await clickRow(w, 'Khách A')
    expect(messagesCallFor('t1', 'cA')).toHaveLength(1)
    lookupCalls()[0].resolve({ page: 1 })
    await settle()
    expect(messagesCallFor('t1', 'cA'), 'the pending link must not start a second lifetime for A').toHaveLength(1)
    messagesCallFor('t1', 'cA')[0].resolve(messagesPayload('cA', 'Khách A', ['TIN-A-1'], 3))
    await settle()
    expect(w.find('.mg-transcript').text()).toContain('TIN-A-1')
  })

  it('page success: mobile back while the lookup is pending leaves the list view and does not reopen A', async () => {
    resetApi(standardRules([pageRule]))
    const { w } = await mountMessages({ query: '?conv=cA', width: 375 })
    await clickRow(w, 'Khách B')
    await backButton(w).trigger('click')
    await settle()
    expect(hasDetail(w)).toBe(false)
    lookupCalls()[0].resolve({ page: 1 })
    await settle()
    expect(hasDetail(w), 'back must end the pending automatic selection').toBe(false)
    expect(messagesCallFor('t1', 'cA')).toHaveLength(0)
    expect(evaluationsCallFor('t1', 'cA')).toHaveLength(0)
  })

  it('page rejection: selecting B keeps B instead of falling back to the old link', async () => {
    resetApi(standardRules([pageRule]))
    const { w } = await mountMessages({ query: '?conv=cA' })
    await clickRow(w, 'Khách B')
    lookupCalls()[0].reject(new Error('lỗi tra cứu trang'))
    await settle()
    expect(detailTitle(w)).toContain('Khách B')
    expect(messagesCallFor('t1', 'cA'), 'the fallback path must also yield to the newer selection').toHaveLength(0)
  })

  it('page rejection: mobile back stays deselected', async () => {
    resetApi(standardRules([pageRule]))
    const { w } = await mountMessages({ query: '?conv=cA', width: 375 })
    await clickRow(w, 'Khách B')
    await backButton(w).trigger('click')
    await settle()
    lookupCalls()[0].reject(new Error('lỗi tra cứu trang'))
    await settle()
    expect(hasDetail(w)).toBe(false)
    expect(messagesCallFor('t1', 'cA')).toHaveLength(0)
  })

  it('other-page lookup: selecting B first means the old link neither reloads another page nor selects', async () => {
    resetApi(standardRules([pageRule]))
    const { w } = await mountMessages({ query: '?conv=cA' })
    const listBefore = requested(URLS.list).length
    await clickRow(w, 'Khách B')
    lookupCalls()[0].resolve({ page: 2 })
    await settle(6)
    expect(requested(URLS.list).length, 'the stale link must not reload the list').toBe(listBefore)
    expect(pageParams().includes(2), 'no request for the old link\'s page').toBe(false)
    messagesCallFor('t1', 'cB')[0].resolve(messagesPayload('cB', 'Khách B', ['TIN-B-1'], 5))
    await settle()
    expect(detailTitle(w)).toContain('Khách B')
    expect(messagesCallFor('t1', 'cA')).toHaveLength(0)
  })

  it('other-page reload pending: selecting B during the reload keeps B after the reload settles', async () => {
    resetApi(standardRules([pageRule, { test: URLS.list, mode: 'deferred' }]))
    const { w } = await mountMessages({ query: '?conv=cA' })
    const lists = () => issued(URLS.list)
    const settleReload = () => lists().forEach((c) => { if (!c.settled) c.resolve(listPayload([convRow('cA', 'Khách A', 3), convRow('cB', 'Khách B', 5)], 50)) })
    lists()[0].resolve(listPayload([convRow('cA', 'Khách A', 3), convRow('cB', 'Khách B', 5)], 50))
    await settle(6)
    lookupCalls()[0].resolve({ page: 2 })
    await settle(4)
    expect(lists().length, 'the link reloads page 2 while the user has not acted').toBeGreaterThanOrEqual(2)
    await clickRow(w, 'Khách B')
    settleReload()
    await settle(6)
    messagesCallFor('t1', 'cB')[0].resolve(messagesPayload('cB', 'Khách B', ['TIN-B-1'], 5))
    await settle()
    expect(detailTitle(w)).toContain('Khách B')
    expect(messagesCallFor('t1', 'cA'), 'the continuation after the reload must yield').toHaveLength(0)
  })

  it('other-page reload pending: mobile back during the reload stays deselected after it settles', async () => {
    resetApi(standardRules([pageRule, { test: URLS.list, mode: 'deferred' }]))
    const { w } = await mountMessages({ query: '?conv=cA', width: 375 })
    const lists = () => issued(URLS.list)
    const settleReload = () => lists().forEach((c) => { if (!c.settled) c.resolve(listPayload([convRow('cA', 'Khách A', 3), convRow('cB', 'Khách B', 5)], 50)) })
    lists()[0].resolve(listPayload([convRow('cA', 'Khách A', 3), convRow('cB', 'Khách B', 5)], 50))
    await settle(6)
    lookupCalls()[0].resolve({ page: 2 })
    await settle(4)
    await clickRow(w, 'Khách B')
    await backButton(w).trigger('click')
    await settle()
    settleReload()
    await settle(6)
    expect(hasDetail(w)).toBe(false)
    expect(messagesCallFor('t1', 'cA')).toHaveLength(0)
  })

  it('selection made before the lookup is even issued: the link issues no stale detail', async () => {
    resetApi(standardRules([pageRule, { test: URLS.evaluated, mode: 'deferred' }]))
    const { w } = await mountMessages({ query: '?conv=cA' })
    await clickRow(w, 'Khách B')
    issued(URLS.evaluated)[0].resolve({})
    await settle(6)
    lookupCalls().forEach((c) => { if (!c.settled) c.resolve({ page: 1 }) })
    await settle(6)
    expect(detailTitle(w)).toContain('Khách B')
    expect(messagesCallFor('t1', 'cA')).toHaveLength(0)
    expect(evaluationsCallFor('t1', 'cA')).toHaveLength(0)
  })

  it('a tenant change still starts the new tenant\'s deep link after an explicit selection in the old one', async () => {
    resetApi(standardRules([pageRule]))
    const { w, router } = await mountMessages({ tenant: 't1', query: '?conv=cA' })
    await clickRow(w, 'Khách B')
    await router.push('/t2/messages?conv=cA')
    await settle(6)
    const t2Lookup = lookupCalls().find((c) => c.url === '/tenants/t2/conversations/cA/page')
    expect(t2Lookup, 'the new tenant\'s own deep link must still run').toBeDefined()
    t2Lookup!.resolve({ page: 1 })
    await settle(6)
    expect(messagesCallFor('t2', 'cA'), 'the new lifetime deep link is a fresh automatic selection').toHaveLength(1)
    expect(hasDetail(w)).toBe(true)
  })
})
