// CCMAI-RUNTIME-088 (F06) UI04: the real Pinia conversations store protects its own publication
// boundary. Only api.get is faked (synthetic ids, controlled deferred replies); the functions under
// judgment are the real store actions. Nothing here is governance evidence.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const apiGet = vi.fn()
vi.mock('../api', () => ({ default: { get: (...args: unknown[]) => apiGet(...args), post: vi.fn(), delete: vi.fn(), put: vi.fn() } }))

import { useConversationStore } from '../stores/conversations'

interface Pending { url: string; params: unknown; resolve: (data: unknown) => void; reject: (e: unknown) => void }
const pendingCalls: Pending[] = []

function installApi() {
  pendingCalls.length = 0
  apiGet.mockReset()
  apiGet.mockImplementation((url: string, config?: { params?: unknown }) => new Promise((resolve, reject) => {
    pendingCalls.push({ url, params: config?.params, resolve: (data) => resolve({ data }), reject })
  }))
}
const detailCall = (conv: string, nth = 0) => pendingCalls.filter((c) => c.url.endsWith(`/conversations/${conv}/messages`))[nth]
const listCalls = () => pendingCalls.filter((c) => c.url.endsWith('/conversations'))
const flush = async () => { for (let i = 0; i < 4; i++) await Promise.resolve() }

const detail = (id: string, name: string, texts: string[]) => ({
  conversation: { id, customer_name: name, message_count: texts.length },
  messages: texts.map((t, i) => ({ id: `${id}-${name}-${i}`, sender_type: 'customer', sender_name: name, content: t, content_type: 'text', attachments: '[]', sent_at: '2026-09-27T09:00:00Z' })),
})
const listBody = (names: string[], total: number) => ({
  data: names.map((n, i) => ({ id: `l-${n}-${i}`, channel_id: 'ch', channel_name: 'k', channel_type: 'facebook', customer_name: n, last_message_at: null, message_count: 1, created_at: '' })),
  total,
})
const swallow = (p: Promise<unknown>) => p.catch((e: unknown) => e)

describe('conversations store: detail publication boundary (UI04)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    installApi()
  })

  it('overlapping detail calls: the later call wins whatever order the replies arrive in', async () => {
    for (const order of ['A-first', 'B-first'] as const) {
      setActivePinia(createPinia())
      installApi()
      const store = useConversationStore()
      void store.fetchMessages('t1', 'A')
      void store.fetchMessages('t1', 'B')
      const a = detailCall('A')
      const b = detailCall('B')
      if (order === 'A-first') { a.resolve(detail('A', 'Khách A', ['tin của A'])); await flush(); b.resolve(detail('B', 'Khách B', ['tin của B'])) }
      else { b.resolve(detail('B', 'Khách B', ['tin của B'])); await flush(); a.resolve(detail('A', 'Khách A', ['tin của A'])) }
      await flush()
      expect(store.currentConversation?.id, order).toBe('B')
      expect(store.messages.map((m) => m.content), order).toEqual(['tin của B'])
    }
  })

  it('A -> B -> A: an old A reply cannot win because the id matches', async () => {
    const store = useConversationStore()
    void store.fetchMessages('t1', 'A')
    void store.fetchMessages('t1', 'B')
    void store.fetchMessages('t1', 'A')
    const oldA = detailCall('A', 0)
    const newA = detailCall('A', 1)
    oldA.resolve(detail('A', 'Khách A', ['A-CŨ']))
    await flush()
    expect(store.messages, 'old A must not publish while new A is pending').toEqual([])
    newA.resolve(detail('A', 'Khách A', ['A-MỚI']))
    await flush()
    expect(store.messages.map((m) => m.content)).toEqual(['A-MỚI'])
    // and the reverse order: new A settles first, old A arrives late
    setActivePinia(createPinia())
    installApi()
    const s2 = useConversationStore()
    void s2.fetchMessages('t1', 'A')
    void s2.fetchMessages('t1', 'B')
    void s2.fetchMessages('t1', 'A')
    detailCall('A', 1).resolve(detail('A', 'Khách A', ['A-MỚI']))
    await flush()
    detailCall('A', 0).resolve(detail('A', 'Khách A', ['A-CŨ']))
    await flush()
    expect(s2.messages.map((m) => m.content)).toEqual(['A-MỚI'])
  })

  it('a new detail call hides the previous conversation immediately', async () => {
    const store = useConversationStore()
    void store.fetchMessages('t1', 'A')
    detailCall('A').resolve(detail('A', 'Khách A', ['tin của A']))
    await flush()
    expect(store.messages).toHaveLength(1)
    void store.fetchMessages('t1', 'B')
    expect(store.messages, 'A data must not stay visible while B is pending').toEqual([])
    expect(store.currentConversation).toBeNull()
  })

  it('a late rejection of a superseded detail call neither throws to its caller nor touches current data', async () => {
    const store = useConversationStore()
    const first = swallow(store.fetchMessages('t1', 'A'))
    void store.fetchMessages('t1', 'B')
    detailCall('B').resolve(detail('B', 'Khách B', ['tin của B']))
    await flush()
    detailCall('A').reject(new Error('mạng lỗi muộn'))
    const outcome = await first
    expect(outcome, 'a superseded call must not surface its failure').not.toBeInstanceOf(Error)
    expect(store.messages.map((m) => m.content)).toEqual(['tin của B'])
    expect(store.currentConversation?.id).toBe('B')
  })

  it('a failure of the current detail call still reaches its caller (healthy control)', async () => {
    const store = useConversationStore()
    const current = swallow(store.fetchMessages('t1', 'A'))
    detailCall('A').reject(new Error('lỗi hiện tại'))
    expect(await current).toBeInstanceOf(Error)
    expect(store.messages).toEqual([])
  })

  it('a single uncontested detail call publishes normally (healthy control)', async () => {
    const store = useConversationStore()
    const done = store.fetchMessages('t1', 'A')
    detailCall('A').resolve(detail('A', 'Khách A', ['x', 'y']))
    await done
    expect(store.messages).toHaveLength(2)
    expect(store.currentConversation).toEqual({ id: 'A', customer_name: 'Khách A', message_count: 2 })
  })
})

describe('conversations store: list publication boundary (UI04/UI06)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    installApi()
  })

  it('overlapping list calls: the latest request wins list and total in either reply order', async () => {
    for (const order of ['old-first', 'new-first'] as const) {
      setActivePinia(createPinia())
      installApi()
      const store = useConversationStore()
      void store.fetchConversations('t1', { page: 1 })
      void store.fetchConversations('t1', { page: 2, search: 'x' })
      const [older, newer] = listCalls()
      if (order === 'old-first') { older.resolve(listBody(['trang-1'], 100)); await flush(); newer.resolve(listBody(['trang-2'], 7)) }
      else { newer.resolve(listBody(['trang-2'], 7)); await flush(); older.resolve(listBody(['trang-1'], 100)) }
      await flush()
      expect(store.conversations.map((c) => c.customer_name), order).toEqual(['trang-2'])
      expect(store.total, order).toBe(7)
    }
  })

  it('a late rejection of an older list call is swallowed and leaves the newer result alone', async () => {
    const store = useConversationStore()
    const older = swallow(store.fetchConversations('t1', { page: 1 }))
    void store.fetchConversations('t1', { page: 2 })
    const [o, n] = listCalls()
    n.resolve(listBody(['trang-2'], 7))
    await flush()
    o.reject(new Error('lỗi muộn'))
    expect(await older).not.toBeInstanceOf(Error)
    expect(store.conversations.map((c) => c.customer_name)).toEqual(['trang-2'])
  })

  it('a failure of the current list call still reaches its caller (healthy control)', async () => {
    const store = useConversationStore()
    const current = swallow(store.fetchConversations('t1', { page: 1 }))
    listCalls()[0].reject(new Error('lỗi hiện tại'))
    expect(await current).toBeInstanceOf(Error)
  })

  it('list and detail generations are independent: a list call does not invalidate a pending detail call', async () => {
    const store = useConversationStore()
    void store.fetchMessages('t1', 'A')
    void store.fetchConversations('t1', { page: 1 })
    listCalls()[0].resolve(listBody(['x'], 1))
    await flush()
    detailCall('A').resolve(detail('A', 'Khách A', ['tin của A']))
    await flush()
    expect(store.messages.map((m) => m.content)).toEqual(['tin của A'])
    expect(store.total).toBe(1)
  })
})

describe('conversations store: owner-scoped cleanup (UI04/UI05 contract)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    installApi()
  })

  it('cleanup by an old owner cannot invalidate or clear a newer owner\'s detail work', async () => {
    const store = useConversationStore()
    const oldOwner = {}
    const newOwner = {}
    void store.fetchMessages('t1', 'A', oldOwner)
    void store.fetchMessages('t1', 'A', newOwner)
    store.resetDetail(oldOwner)
    detailCall('A', 1).resolve(detail('A', 'Khách A', ['của chủ mới']))
    await flush()
    expect(store.messages.map((m) => m.content)).toEqual(['của chủ mới'])
    store.reset(oldOwner)
    expect(store.messages.map((m) => m.content), 'old owner reset must not clear newer data').toEqual(['của chủ mới'])
  })

  it('cleanup by the current owner clears data and drops its pending replies', async () => {
    const store = useConversationStore()
    const owner = {}
    void store.fetchMessages('t1', 'A', owner)
    void store.fetchConversations('t1', {}, owner)
    store.reset(owner)
    detailCall('A').resolve(detail('A', 'Khách A', ['muộn']))
    listCalls()[0].resolve(listBody(['muộn'], 3))
    await flush()
    expect(store.messages).toEqual([])
    expect(store.currentConversation).toBeNull()
    expect(store.conversations).toEqual([])
    expect(store.total).toBe(0)
  })

  it('cleanup by an old owner cannot invalidate a newer owner\'s list work', async () => {
    const store = useConversationStore()
    const oldOwner = {}
    const newOwner = {}
    void store.fetchConversations('t1', { page: 1 }, oldOwner)
    void store.fetchConversations('t1', { page: 1 }, newOwner)
    store.reset(oldOwner)
    listCalls()[1].resolve(listBody(['của chủ mới'], 4))
    await flush()
    expect(store.conversations.map((c) => c.customer_name)).toEqual(['của chủ mới'])
    expect(store.total).toBe(4)
  })

  it('fetch calls report whether they published', async () => {
    const store = useConversationStore()
    const first = store.fetchMessages('t1', 'A')
    const second = store.fetchMessages('t1', 'B')
    detailCall('A').resolve(detail('A', 'Khách A', ['a']))
    detailCall('B').resolve(detail('B', 'Khách B', ['b']))
    expect(await first).toBe(false)
    expect(await second).toBe(true)
  })
})
