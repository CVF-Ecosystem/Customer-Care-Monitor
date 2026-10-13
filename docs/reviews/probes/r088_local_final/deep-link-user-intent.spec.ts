// @vitest-environment happy-dom
import { it, expect, vi, afterEach } from 'vitest'
import { apiGet, resetApi, standardRules, URLS, issued, mountMessages, cleanup, settle, clickRow, hasDetail, detailTitle, messagesCallFor } from './fixtures/conversation-lifetime/harness'
import viMessages from '../i18n/vi'
vi.mock('../api', () => ({ default: { get: (...args: unknown[]) => apiGet(...args) } }))
afterEach(cleanup)
it('healthy deep link still selects A when no newer user action occurred', async () => {
  resetApi(standardRules([{ test: URLS.page, mode: 'deferred' }]))
  const { w } = await mountMessages({ query: '?conv=cA' })
  issued(URLS.page)[0].resolve({ page: 1 }); await settle()
  expect(hasDetail(w)).toBe(true)
  expect(detailTitle(w)).toContain('Khách A')
  expect(messagesCallFor('t1', 'cA')).toHaveLength(1)
})
it('pending initial deep link cannot override the newer explicit selection B', async () => {
  resetApi(standardRules([{ test: URLS.page, mode: 'deferred' }]))
  const { w } = await mountMessages({ query: '?conv=cA' })
  await clickRow(w, 'Khách B')
  expect(detailTitle(w)).toContain('Khách B')
  issued(URLS.page)[0].resolve({ page: 1 }); await settle()
  expect(detailTitle(w), 'older initialization must preserve the explicit B selection').toContain('Khách B')
  expect(messagesCallFor('t1', 'cA'), 'older initialization must not issue stale detail').toHaveLength(0)
})
it('mobile back cancels pending deep-link selection instead of reopening A', async () => {
  resetApi(standardRules([{ test: URLS.page, mode: 'deferred' }]))
  const { w } = await mountMessages({ query: '?conv=cA', width: 375 })
  await clickRow(w, 'Khách B')
  await w.find(`button[aria-label="${viMessages.msgs_back}"]`).trigger('click'); await settle()
  expect(hasDetail(w)).toBe(false)
  issued(URLS.page)[0].resolve({ page: 1 }); await settle()
  expect(hasDetail(w), 'back must invalidate pending initialization selection').toBe(false)
  expect(messagesCallFor('t1', 'cA')).toHaveLength(0)
})
