// @vitest-environment happy-dom
// Synthetic UI fixture only; saved observations are presentation data, not governance evidence.
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
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

vi.mock('vue-chartjs', () => ({ Line: { name: 'Line', render: () => null } }))
const apiGet = vi.fn()
const apiPost = vi.fn()
const apiDelete = vi.fn()
vi.mock('../api', () => ({ default: { get: (...args: unknown[]) => apiGet(...args), post: (...args: unknown[]) => apiPost(...args), delete: (...args: unknown[]) => apiDelete(...args) } }))

import JobDetail from '../views/Jobs/JobDetail.vue'

const ids = {
  tenant: '11111111-1111-4111-8111-111111111111',
  job: '22222222-2222-4222-8222-222222222222',
  success: '33333333-3333-4333-8333-333333333333',
  running: '77777777-7777-4777-8777-777777777777',
  error: '88888888-8888-4888-8888-888888888888',
  cancelled: '99999999-9999-4999-8999-999999999999',
  noResults: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
  conversation: '44444444-4444-4444-8444-444444444444',
  member: '55555555-5555-4555-8555-555555555555',
}

function savedSummary(runId: string, estimate: number | null = 0, state: 'complete' | 'running' | 'error' | 'cancelled' = 'complete') {
  const costComplete = estimate !== null
  return JSON.stringify({
    conversations_found: 1,
    source_preparation: {
      version: 'ccmai.source-preparation.v1', scope: 'preparation_only', tenant_id: ids.tenant, job_id: ids.job,
      run_id: runId, mode: 'conditional', metadata_incomplete: false, selection_status: 'COMPLETE', selected: 1,
      visited: 1, unvisited: 0, counts: { PREPARED_FOR_INFERENCE: 1 },
      entries: [{ conversation_id: ids.conversation, outcome: 'PREPARED_FOR_INFERENCE', snapshot_schema: 'ccma.snapshot.v1',
        snapshot_digest: 'a'.repeat(64), coverage: 'complete', reference: { state: 'NONE' }, metadata_incomplete: false }],
      omitted_entries: 0, entries_complete: true, scan_complete: true,
    },
    source_execution: {
      version: 'ccmai.source-execution.v1', scope: 'analyzer_provider_interface_only', tenant_id: ids.tenant,
      job_id: ids.job, run_id: runId, mode: 'conditional', metadata_incomplete: false,
      calls_begun: 1, response_returned: 1, error_returned: 0, interrupted: 0, in_flight: 0,
      item_count: 1, items_saved: 1, items_save_failed: 0, items_not_published: 0, items_pending: 0,
      usage_writes: { NOT_ATTEMPTED: 0, WRITE_SUCCEEDED: 1, WRITE_FAILED: 0, WRITE_OUTCOME_UNKNOWN: 0 },
      parsing: { NOT_ATTEMPTED: 0, ACCEPTED: 0, REJECTED: 0, NOT_SEPARATELY_OBSERVABLE: 1 },
      calls: [{ sequence: 1, method: 'SINGLE', item_count: 1, member_ids: [ids.member], omitted_members: 0,
        members_complete: true, metadata_incomplete: false, invocation_outcome: 'RESPONSE_RETURNED',
        usage_write_outcome: 'WRITE_SUCCEEDED', parsing_outcome: 'NOT_SEPARATELY_OBSERVABLE',
        items_saved: 1, items_save_failed: 0, items_not_published: 0, items_pending: 0 }],
      omitted_calls: 0, omitted_members: 0, entries_complete: true, members_complete: true,
      execution_complete: state === 'complete' || state === 'error',
      stop_reason: state === 'error' ? 'EXECUTION_ERROR' : state === 'cancelled' ? 'CANCELLED' : 'NONE',
      usage_observation: {
        version: 'ccmai.usage-observation.v1', scope: 'successful_interface_response_local_estimate',
        token_basis: 'INTERFACE_VALUES_PRESENCE_UNAVAILABLE', billing: 'NOT_OBSERVED', price_revision: 'NOT_CAPTURED',
        responses: 1, invalid_tokens: 0, priced: costComplete ? 1 : 0, unpriced: costComplete ? 0 : 1, invalid_costs: 0,
        token_overflow: false, cost_overflow: false, counter_overflow: false,
        input_tokens: 0, output_tokens: 0, local_estimate_usd: estimate, tokens_complete: true, cost_complete: costComplete,
      },
    },
  })
}

const job = { id: ids.job, tenant_id: ids.tenant, name: 'Saved run fixture', job_type: 'qc_analysis', schedule_type: 'manual',
  schedule_cron: '', input_channel_ids: '[]', last_run_at: null, last_run_status: 'success' }
const runs = [
  { id: ids.success, job_id: ids.job, started_at: '2026-10-08T08:00:00Z', finished_at: '2026-10-08T08:01:00Z', status: 'success', summary: savedSummary(ids.success), error_message: '' },
  { id: ids.running, job_id: ids.job, started_at: '2026-10-08T08:02:00Z', finished_at: null, status: 'running', summary: savedSummary(ids.running, null, 'running'), error_message: '' },
  { id: ids.error, job_id: ids.job, started_at: '2026-10-08T08:03:00Z', finished_at: '2026-10-08T08:04:00Z', status: 'error', summary: savedSummary(ids.error, 0, 'error'), error_message: 'Synthetic failed run' },
  { id: ids.cancelled, job_id: ids.job, started_at: '2026-10-08T08:05:00Z', finished_at: '2026-10-08T08:06:00Z', status: 'cancelled', summary: savedSummary(ids.success, 0, 'cancelled'), error_message: 'Synthetic cancellation' },
  { id: ids.noResults, job_id: ids.job, started_at: '2026-10-08T08:07:00Z', finished_at: '2026-10-08T08:08:00Z', status: 'partial', summary: '{}', error_message: '' },
]
const results = [{ id: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb', job_run_id: ids.success, conversation_id: ids.conversation,
  result_type: 'conversation_evaluation', severity: 'PASS', rule_name: '', evidence: 'Synthetic result', detail: '{}',
  confidence: null, confidence_basis: 'unavailable', source_integrity_status: 'legacy_unverified',
  created_at: '2026-10-08T08:01:00Z', conversation_date: '2026-10-08T08:00:00Z', customer_name: 'Synthetic customer' }]

function setupApi() {
  apiGet.mockImplementation((url: string) => {
    if (url.endsWith(`/jobs/${ids.job}`)) return Promise.resolve({ data: job })
    if (url.endsWith(`/jobs/${ids.job}/runs`)) return Promise.resolve({ data: runs })
    if (url.endsWith(`/jobs/${ids.job}/results`)) return Promise.resolve({ data: results })
    if (url.endsWith('/settings')) return Promise.resolve({ data: { settings: { ai_provider: 'synthetic' } } })
    return Promise.resolve({ data: {} })
  })
}

async function mountView() {
  const pinia = createPinia()
  setActivePinia(pinia)
  useAuthStore().tenantPerms = { role: 'owner', permissions: {} } as never
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: '/:tenantId/jobs/:jobId', component: JobDetail },
    { path: '/:pathMatch(.*)*', component: { template: '<div />' } },
  ] })
  await router.push(`/${ids.tenant}/jobs/${ids.job}`)
  await router.isReady()
  const wrapper = mount(JobDetail, {
    global: { plugins: [
      createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } }),
      createI18n({ legacy: false, locale: 'vi', fallbackLocale: 'en', messages: { vi: viMessages, en: enMessages } }),
      pinia, router,
    ] },
  })
  for (let i = 0; i < 4; i++) await flushPromises()
  return wrapper
}

describe('JobDetail saved-run observation integration', () => {
  beforeEach(() => {
    apiGet.mockReset()
    apiPost.mockReset()
    apiDelete.mockReset()
  })

  it('shows receipt access on every loaded history row without adding requests, actions, or changing row state', async () => {
    setupApi()
    const wrapper = await mountView()
    const expectedInitialGets = [
      `/tenants/${ids.tenant}/jobs/${ids.job}`,
      `/tenants/${ids.tenant}/jobs/${ids.job}/runs`,
      `/tenants/${ids.tenant}/jobs/${ids.job}/results`,
      `/tenants/${ids.tenant}/settings`,
    ]
    expect(apiGet.mock.calls.map(([url]) => url).sort()).toEqual(expectedInitialGets.sort())
    const historyTab = wrapper.findAll('.v-tab').find(tab => tab.text() === viMessages.run_history)
    expect(historyTab).toBeTruthy()
    await historyTab!.trigger('click')
    await flushPromises()

    expect(wrapper.findAll('details.run-observation')).toHaveLength(5)
    const success = wrapper.find(`details[data-run-id="${ids.success}"]`)
    const running = wrapper.find(`details[data-run-id="${ids.running}"]`)
    const failed = wrapper.find(`details[data-run-id="${ids.error}"]`)
    const mismatched = wrapper.find(`details[data-run-id="${ids.cancelled}"]`)
    const noResults = wrapper.find(`details[data-run-id="${ids.noResults}"]`)
    const successfulRow = success.element.closest('tr')
    const emptyRow = noResults.element.closest('tr')
    expect(successfulRow).not.toBeNull()
    expect(emptyRow).not.toBeNull()
    expect(success.find('[data-field="local_estimate_usd"]').text()).toBe('0 USD')
    expect(running.find('[data-field="local_estimate_usd"]').text()).toBe(viMessages.run_obs_unknown)
    expect(failed.find('[data-field="local_estimate_usd"]').text()).toBe('0 USD')
    expect(mismatched.find('[data-section="execution"] [data-testid="observation-unavailable"]').exists()).toBe(true)
    expect(noResults.findAll('[data-testid="observation-unavailable"]')).toHaveLength(4)
    expect(wrapper.text()).toContain('Synthetic failed run')
    expect(wrapper.text()).toContain(viMessages.jd_run_running)
    expect(Array.from(successfulRow!.querySelectorAll('button')).some(button => button.textContent?.includes(viMessages.jd_run_view))).toBe(true)
    expect(emptyRow!.querySelectorAll('button')).toHaveLength(0)

    const beforeExpand = apiGet.mock.calls.length
    await success.find('summary').trigger('click')
    await failed.find('summary').trigger('click')
    expect(apiGet).toHaveBeenCalledTimes(beforeExpand)
    expect(apiPost).not.toHaveBeenCalled()
    expect(apiDelete).not.toHaveBeenCalled()
    expect(wrapper.findAll('details.run-observation')).toHaveLength(5)
    wrapper.unmount()
  })
})
