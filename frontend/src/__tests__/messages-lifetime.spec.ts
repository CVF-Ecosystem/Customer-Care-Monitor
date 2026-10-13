// @vitest-environment happy-dom
// CCMAI-RUNTIME-088 (F06) UI01-UI05: the real Messages.vue and the real conversations store with a
// synthetic, deferred API boundary. Assertions are about what the user can see for the CURRENT
// tenant/selection, not about completion order. Nothing here is governance evidence.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createPinia } from 'pinia'
import viMessages from '../i18n/vi'
import { useConversationStore } from '../stores/conversations'
import {
  apiGet, resetApi, standardRules, URLS, pending, requested, messagesCallFor, evaluationsCallFor,
  convRow, listPayload, messagesPayload, evaluationPayload,
  mountMessages, unmountOne, cleanup, settle, clickRow, openTab,
  hasDetail, detailTitle, detailSubtitle, transcriptText, transcriptSpinner, convErrorShown,
  downloadDisabled, qcText, qcSpinner, rows, chipTexts,
} from './fixtures/conversation-lifetime/harness'

vi.mock('../api', () => ({ default: { get: (...args: unknown[]) => apiGet(...args), post: vi.fn(), delete: vi.fn(), put: vi.fn() } }))

vi.setConfig({ testTimeout: 30_000 })

const A = 'Khách A'
const B = 'Khách B'

async function selectAndSettleA(w: Awaited<ReturnType<typeof mountMessages>>['w']) {
  await clickRow(w, A)
  messagesCallFor('t1', 'cA')[0].resolve(messagesPayload('cA', A, ['TIN-A-1', 'TIN-A-2'], 3))
  await settle()
  evaluationsCallFor('t1', 'cA')[0].resolve(evaluationPayload('QC-CỦA-A'))
  await settle()
}

describe('Messages detail: pending selection must not expose the previous conversation (UI01)', () => {
  beforeEach(() => resetApi(standardRules()))
  afterEach(cleanup)

  it('settled A, then pending B: header, count, messages, evaluation and download source are not A\'s', async () => {
    const { w } = await mountMessages()
    await selectAndSettleA(w)
    expect(transcriptText(w)).toContain('TIN-A-1')
    expect(detailSubtitle(w)).toContain('3 tin nhắn')
    expect(downloadDisabled(w)).toBe(false)
    await openTab(w, 'qc')
    expect(qcText(w)).toContain('QC-CỦA-A')
    await openTab(w, 'messages')

    await clickRow(w, B) // B stays pending
    expect(detailTitle(w), 'header must not show A while B is pending').not.toContain(A)
    expect(detailSubtitle(w), 'A\'s message count must not label B').not.toContain('3 tin nhắn')
    expect(transcriptText(w)).not.toContain('TIN-A-1')
    expect(transcriptSpinner(w)).toBe(true)
    expect(downloadDisabled(w), 'download must not export A while B is pending').toBe(true)
    await openTab(w, 'qc')
    expect(qcText(w), 'evaluation must not show A while B is pending').not.toContain('QC-CỦA-A')
    await openTab(w, 'messages')

    // healthy control: B settles and shows its own data
    messagesCallFor('t1', 'cB')[0].resolve(messagesPayload('cB', B, ['TIN-B-1'], 5))
    await settle()
    expect(detailTitle(w)).toContain(B)
    expect(detailSubtitle(w)).toContain('5 tin nhắn')
    expect(transcriptText(w)).toContain('TIN-B-1')
    expect(transcriptText(w)).not.toContain('TIN-A-1')
    expect(downloadDisabled(w)).toBe(false)
    evaluationsCallFor('t1', 'cB')[0].resolve(evaluationPayload('QC-CỦA-B'))
    await settle()
    await openTab(w, 'qc')
    expect(qcText(w)).toContain('QC-CỦA-B')
    expect(qcText(w)).not.toContain('QC-CỦA-A')
  })

  it('a reply whose conversation id is not the selected one is never shown as the selection\'s header or count', async () => {
    const { w } = await mountMessages()
    await clickRow(w, A)
    messagesCallFor('t1', 'cA')[0].resolve(messagesPayload('someone-else', 'Người khác hẳn', ['TIN-LẠ'], 99))
    await settle()
    expect(detailTitle(w)).toContain(A)
    expect(detailTitle(w)).not.toContain('Người khác hẳn')
    expect(detailSubtitle(w)).not.toContain('99 tin nhắn')
  })

  it('healthy control: a single selection loads messages then evaluation and tabs keep their data', async () => {
    const { w } = await mountMessages()
    await selectAndSettleA(w)
    expect(detailTitle(w)).toContain(A)
    await openTab(w, 'qc')
    expect(qcText(w)).toContain('QC-CỦA-A')
    await openTab(w, 'classification')
    await openTab(w, 'messages')
    expect(transcriptText(w), 'tab switching must keep the valid selection data').toContain('TIN-A-1')
    expect(messagesCallFor('t1', 'cA')).toHaveLength(1)
  })
})

describe('Messages detail: A -> B -> A is a new lifetime (UI02)', () => {
  beforeEach(() => resetApi(standardRules()))
  afterEach(cleanup)

  for (const order of ['old A first', 'new A first'] as const) {
    it(`an old A reply cannot win because the ids match (${order})`, async () => {
      const { w } = await mountMessages()
      await clickRow(w, A)
      await clickRow(w, B)
      await clickRow(w, A)
      const [oldA, newA] = messagesCallFor('t1', 'cA')
      expect(oldA).toBeDefined()
      expect(newA).toBeDefined()
      if (order === 'old A first') {
        oldA.resolve(messagesPayload('cA', A, ['A-CŨ'], 1))
        await settle()
        expect(transcriptText(w), 'old A must not publish while new A is pending').not.toContain('A-CŨ')
        expect(transcriptSpinner(w)).toBe(true)
        newA.resolve(messagesPayload('cA', A, ['A-MỚI'], 1))
      } else {
        newA.resolve(messagesPayload('cA', A, ['A-MỚI'], 1))
        await settle()
        oldA.resolve(messagesPayload('cA', A, ['A-CŨ'], 1))
      }
      await settle()
      expect(transcriptText(w)).toContain('A-MỚI')
      expect(transcriptText(w)).not.toContain('A-CŨ')
      expect(transcriptSpinner(w)).toBe(false)

      // evaluation: only the current lifetime may start downstream work; resolve everything out of order
      const evalCalls = evaluationsCallFor('t1', 'cA')
      expect(evalCalls, 'a superseded lifetime must not start evaluation work').toHaveLength(1)
      evalCalls[evalCalls.length - 1].resolve(evaluationPayload('QC-MỚI'))
      await settle()
      for (const extra of evalCalls.slice(0, -1)) extra.resolve(evaluationPayload('QC-CŨ'))
      await settle()
      await openTab(w, 'qc')
      expect(qcText(w)).toContain('QC-MỚI')
      expect(qcText(w)).not.toContain('QC-CŨ')
    })
  }

  it('reversed evaluation replies for the same id: the newer lifetime keeps its evaluation', async () => {
    const { w } = await mountMessages()
    await clickRow(w, A)
    messagesCallFor('t1', 'cA')[0].resolve(messagesPayload('cA', A, ['A-CŨ'], 1))
    await settle()
    await clickRow(w, B)
    await clickRow(w, A)
    messagesCallFor('t1', 'cA')[1].resolve(messagesPayload('cA', A, ['A-MỚI'], 1))
    await settle()
    const evals = evaluationsCallFor('t1', 'cA')
    // newest evaluation answers first, the old lifetime's evaluation (issued earlier) arrives late
    evals[evals.length - 1].resolve(evaluationPayload('QC-MỚI'))
    await settle()
    for (const old of evals.slice(0, -1)) old.resolve(evaluationPayload('QC-CŨ'))
    await settle()
    await openTab(w, 'qc')
    expect(qcText(w)).toContain('QC-MỚI')
    expect(qcText(w)).not.toContain('QC-CŨ')
  })
})

describe('Messages detail: superseded success, failure and finally cannot change the current view (UI03)', () => {
  beforeEach(() => resetApi(standardRules()))
  afterEach(cleanup)

  it('a stale success does not publish, does not clear the current spinner and starts no evaluation', async () => {
    const { w } = await mountMessages()
    await clickRow(w, A)
    await clickRow(w, B)
    messagesCallFor('t1', 'cA')[0].resolve(messagesPayload('cA', A, ['TIN-A-MUỘN'], 1))
    await settle()
    expect(transcriptSpinner(w), 'B is still pending: its spinner must stay').toBe(true)
    expect(transcriptText(w)).not.toContain('TIN-A-MUỘN')
    expect(detailTitle(w)).not.toContain(A)
    expect(evaluationsCallFor('t1', 'cA'), 'stale chain must not start downstream evaluation').toHaveLength(0)
    messagesCallFor('t1', 'cB')[0].resolve(messagesPayload('cB', B, ['TIN-B-1'], 5))
    await settle()
    expect(transcriptText(w)).toContain('TIN-B-1')
    expect(transcriptText(w)).not.toContain('TIN-A-MUỘN')
  })

  it('a stale messages success leaves the current evaluation loading state alone', async () => {
    const { w } = await mountMessages()
    await clickRow(w, A)
    await clickRow(w, B)
    messagesCallFor('t1', 'cA')[0].resolve(messagesPayload('cA', A, ['TIN-A-MUỘN'], 1))
    await settle()
    await openTab(w, 'qc')
    expect(qcSpinner(w), 'B is still loading: A\'s late reply must not end the evaluation spinner').toBe(true)
    expect(qcText(w)).not.toContain(viMessages.msgs_qc_empty)
  })

  it('a stale rejection shows no error and leaves the current spinner and result alone', async () => {
    const { w } = await mountMessages()
    await clickRow(w, A)
    await clickRow(w, B)
    messagesCallFor('t1', 'cA')[0].reject(new Error('lỗi của A, đến muộn'))
    await settle()
    expect(convErrorShown(w), 'a superseded failure must not become the current error').toBe(false)
    expect(transcriptSpinner(w)).toBe(true)
    messagesCallFor('t1', 'cB')[0].resolve(messagesPayload('cB', B, ['TIN-B-1'], 5))
    await settle()
    expect(convErrorShown(w)).toBe(false)
    expect(transcriptText(w)).toContain('TIN-B-1')
  })

  it('healthy control: a current failure shows the error and retry recovers', async () => {
    const { w } = await mountMessages()
    await clickRow(w, A)
    messagesCallFor('t1', 'cA')[0].reject(new Error('lỗi hiện tại'))
    await settle()
    expect(convErrorShown(w)).toBe(true)
    await w.find('[data-testid="msgs-conv-error"] button').trigger('click')
    await settle(2)
    expect(messagesCallFor('t1', 'cA')).toHaveLength(2)
    messagesCallFor('t1', 'cA')[1].resolve(messagesPayload('cA', A, ['TIN-A-1'], 3))
    await settle()
    expect(convErrorShown(w)).toBe(false)
    expect(transcriptText(w)).toContain('TIN-A-1')
  })

  it('a stale evaluation success or failure cannot change the current evaluation spinner, data or error', async () => {
    const { w } = await mountMessages()
    await clickRow(w, A)
    messagesCallFor('t1', 'cA')[0].resolve(messagesPayload('cA', A, ['TIN-A-1'], 3))
    await settle()
    const evalA = evaluationsCallFor('t1', 'cA')[0]
    await clickRow(w, B) // A's evaluation request is still pending
    await openTab(w, 'qc')
    evalA.resolve(evaluationPayload('QC-CỦA-A-MUỘN'))
    await settle()
    expect(qcText(w), 'stale evaluation must not appear for B').not.toContain('QC-CỦA-A-MUỘN')
    expect(qcSpinner(w), 'B has not finished loading: stale finally must not clear its spinner').toBe(true)

    // same with a rejection of a superseded evaluation
    await openTab(w, 'messages')
    messagesCallFor('t1', 'cB')[0].resolve(messagesPayload('cB', B, ['TIN-B-1'], 5))
    await settle()
    await clickRow(w, A)
    messagesCallFor('t1', 'cA')[1].resolve(messagesPayload('cA', A, ['TIN-A-2'], 3))
    await settle()
    const evalBStale = evaluationsCallFor('t1', 'cB')[0]
    const evalA2 = evaluationsCallFor('t1', 'cA')[1]
    expect(evalA2).toBeDefined()
    evalBStale.reject(new Error('lỗi đánh giá của B, muộn'))
    await settle()
    await openTab(w, 'qc')
    expect(qcSpinner(w), 'A\'s evaluation is still pending').toBe(true)
    evalA2.resolve(evaluationPayload('QC-A-HIỆN-TẠI'))
    await settle()
    expect(qcText(w)).toContain('QC-A-HIỆN-TẠI')
  })

  it('healthy control: a current evaluation failure leaves messages intact and shows the honest empty state', async () => {
    const { w } = await mountMessages()
    await clickRow(w, A)
    messagesCallFor('t1', 'cA')[0].resolve(messagesPayload('cA', A, ['TIN-A-1'], 3))
    await settle()
    evaluationsCallFor('t1', 'cA')[0].reject(new Error('lỗi đánh giá hiện tại'))
    await settle()
    expect(transcriptText(w)).toContain('TIN-A-1')
    await openTab(w, 'qc')
    expect(qcSpinner(w)).toBe(false)
    expect(qcText(w)).toContain(viMessages.msgs_qc_empty)
  })
})

describe('Messages detail: reset, tenant change and remount (UI05)', () => {
  afterEach(cleanup)

  it('mobile back before completion clears the detail and late replies cannot bring it back', async () => {
    resetApi(standardRules())
    const { w, pinia } = await mountMessages({ width: 500 })
    await clickRow(w, A)
    expect(hasDetail(w)).toBe(true)
    await w.find(`button[aria-label="${viMessages.msgs_back}"]`).trigger('click')
    await settle(2)
    expect(hasDetail(w), 'back must leave the list view').toBe(false)
    expect(rows(w).length).toBeGreaterThan(0)
    messagesCallFor('t1', 'cA')[0].resolve(messagesPayload('cA', A, ['TIN-A-MUỘN'], 3))
    await settle()
    expect(hasDetail(w)).toBe(false)
    const store = useConversationStore(pinia)
    expect(store.messages, 'old detail data must stay cleared').toEqual([])
    expect(store.currentConversation).toBeNull()
    expect(evaluationsCallFor('t1', 'cA'), 'a reset lifetime starts no evaluation').toHaveLength(0)
  })

  it('mobile back then a late rejection and a new selection: no stale error and the new selection is intact', async () => {
    resetApi(standardRules())
    const { w } = await mountMessages({ width: 500 })
    await clickRow(w, A)
    await w.find(`button[aria-label="${viMessages.msgs_back}"]`).trigger('click')
    await settle(2)
    await clickRow(w, B)
    messagesCallFor('t1', 'cA')[0].reject(new Error('lỗi muộn của A'))
    await settle()
    expect(convErrorShown(w)).toBe(false)
    expect(transcriptSpinner(w)).toBe(true)
    messagesCallFor('t1', 'cB')[0].resolve(messagesPayload('cB', B, ['TIN-B-1'], 5))
    await settle()
    expect(transcriptText(w)).toContain('TIN-B-1')
    expect(detailTitle(w)).toContain(B)
  })

  it('tenant t1 -> t2 with the same conversation id: the old tenant\'s reply cannot appear in the new tenant', async () => {
    resetApi(standardRules([], []).map((r) => (r.test === URLS.list
      ? { ...r, reply: (url: string) => listPayload([convRow('c1', url.includes('/t2/') ? 'Khách của T2' : 'Khách của T1')]) }
      : r)))
    const { w, router, pinia } = await mountMessages({ tenant: 't1' })
    await clickRow(w, 'Khách của T1')
    const t1Detail = messagesCallFor('t1', 'c1')[0]
    expect(t1Detail).toBeDefined()
    await router.push('/t2/messages')
    await settle(6)
    expect(hasDetail(w), 'tenant change must clear the selection').toBe(false)
    expect(requested(URLS.list).some((r) => r.url === '/tenants/t2/conversations'), 'the new tenant\'s list must be loaded').toBe(true)
    expect(rows(w).map((r) => r.text()).join('|')).toContain('Khách của T2')
    t1Detail.resolve(messagesPayload('c1', 'Khách của T1', ['TIN-CỦA-T1'], 1))
    await settle()
    expect(hasDetail(w)).toBe(false)
    expect(useConversationStore(pinia).messages).toEqual([])
    await clickRow(w, 'Khách của T2')
    expect(messagesCallFor('t2', 'c1')).toHaveLength(1)
    messagesCallFor('t2', 'c1')[0].resolve(messagesPayload('c1', 'Khách của T2', ['TIN-CỦA-T2'], 1))
    await settle()
    expect(transcriptText(w)).toContain('TIN-CỦA-T2')
    expect(transcriptText(w)).not.toContain('TIN-CỦA-T1')
  })

  it('tenant t1 -> t2: t2 selected first, then the old t1 reply arrives late and loses', async () => {
    resetApi(standardRules([], []).map((r) => (r.test === URLS.list
      ? { ...r, reply: (url: string) => listPayload([convRow('c1', url.includes('/t2/') ? 'Khách của T2' : 'Khách của T1')]) }
      : r)))
    const { w, router } = await mountMessages({ tenant: 't1' })
    await clickRow(w, 'Khách của T1')
    const t1Detail = messagesCallFor('t1', 'c1')[0]
    await router.push('/t2/messages')
    await settle(6)
    await clickRow(w, 'Khách của T2')
    t1Detail.resolve(messagesPayload('c1', 'Khách của T1', ['TIN-CỦA-T1'], 1))
    await settle()
    expect(transcriptText(w)).not.toContain('TIN-CỦA-T1')
    expect(transcriptSpinner(w), 't2 is still pending').toBe(true)
    messagesCallFor('t2', 'c1')[0].resolve(messagesPayload('c1', 'Khách của T2', ['TIN-CỦA-T2'], 1))
    await settle()
    expect(transcriptText(w)).toContain('TIN-CỦA-T2')
    expect(transcriptText(w)).not.toContain('TIN-CỦA-T1')
  })

  for (const outcome of ['success-before-new', 'success-after-new', 'error'] as const) {
    it(`unmount then remount on one Pinia: the old lifetime's reply (${outcome}) cannot touch the new one`, async () => {
      resetApi(standardRules())
      const pinia = createPinia()
      const first = await mountMessages({ pinia })
      await clickRow(first.w, A)
      const oldReply = messagesCallFor('t1', 'cA')[0]
      unmountOne(first.w)

      const second = await mountMessages({ pinia })
      await clickRow(second.w, A)
      const newReply = messagesCallFor('t1', 'cA')[1]
      expect(newReply).toBeDefined()
      if (outcome === 'success-before-new') oldReply.resolve(messagesPayload('cA', A, ['TIN-CỦA-LẦN-CŨ'], 1))
      else if (outcome === 'error') oldReply.reject(new Error('lỗi của lần mount cũ'))
      await settle()
      expect(transcriptText(second.w)).not.toContain('TIN-CỦA-LẦN-CŨ')
      expect(convErrorShown(second.w)).toBe(false)
      expect(transcriptSpinner(second.w), 'the new lifetime is still pending').toBe(true)
      newReply.resolve(messagesPayload('cA', A, ['TIN-CỦA-LẦN-MỚI'], 1))
      await settle()
      if (outcome === 'success-after-new') {
        oldReply.resolve(messagesPayload('cA', A, ['TIN-CỦA-LẦN-CŨ'], 1)) // arrives late, after the new data is visible
        await settle()
      }
      expect(transcriptText(second.w)).toContain('TIN-CỦA-LẦN-MỚI')
      expect(transcriptText(second.w)).not.toContain('TIN-CỦA-LẦN-CŨ')
      expect(convErrorShown(second.w)).toBe(false)
    })
  }

  it('an old owner unmounting AFTER the new one started must not cancel the new owner\'s request', async () => {
    resetApi(standardRules())
    const pinia = createPinia()
    const first = await mountMessages({ pinia })
    await clickRow(first.w, A)
    const second = await mountMessages({ pinia })
    await clickRow(second.w, A)
    const newReply = messagesCallFor('t1', 'cA').at(-1)!
    unmountOne(first.w) // old owner cleanup runs now
    newReply.resolve(messagesPayload('cA', A, ['TIN-CỦA-CHỦ-MỚI'], 1))
    await settle()
    expect(transcriptText(second.w), 'cleanup by the old owner must not invalidate the new owner').toContain('TIN-CỦA-CHỦ-MỚI')
    expect(pending(URLS.evaluations).length + evaluationsCallFor('t1', 'cA').length).toBeGreaterThan(0)
  })

  it('healthy control: switching tenants and selecting again works end to end', async () => {
    resetApi(standardRules([], []).map((r) => (r.test === URLS.list
      ? { ...r, reply: (url: string) => listPayload([convRow('c1', url.includes('/t2/') ? 'Khách của T2' : 'Khách của T1')]) }
      : r)))
    const { w, router } = await mountMessages({ tenant: 't1' })
    await router.push('/t2/messages')
    await settle(6)
    expect(chipTexts(w).length).toBeGreaterThan(0)
    await clickRow(w, 'Khách của T2')
    messagesCallFor('t2', 'c1')[0].resolve(messagesPayload('c1', 'Khách của T2', ['TIN-CỦA-T2'], 1))
    await settle()
    expect(transcriptText(w)).toContain('TIN-CỦA-T2')
  })
})
