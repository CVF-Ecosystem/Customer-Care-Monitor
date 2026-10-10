// @vitest-environment happy-dom
import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import RunObservationPanel from '../components/ui/RunObservationPanel.vue'
import enMessages from '../i18n/en'
import viMessages from '../i18n/vi'

const ids = {
  tenant: '11111111-1111-4111-8111-111111111111',
  job: '22222222-2222-4222-8222-222222222222',
  run: '33333333-3333-4333-8333-333333333333',
  conversation: '44444444-4444-4444-8444-444444444444',
  member: '55555555-5555-4555-8555-555555555555',
}
const context = { tenantId: ids.tenant, jobId: ids.job, runId: ids.run, runJobId: ids.job }
const mounted: Array<{ unmount: () => void }> = []

function sourceExecution(usageOverrides: Record<string, unknown> = {}) {
  return {
    version: 'ccmai.source-execution.v1', scope: 'analyzer_provider_interface_only', tenant_id: ids.tenant,
    job_id: ids.job, run_id: ids.run, mode: 'conditional', metadata_incomplete: false,
    calls_begun: 1, response_returned: 1, error_returned: 0, interrupted: 0, in_flight: 0,
    item_count: 1, items_saved: 1, items_save_failed: 0, items_not_published: 0, items_pending: 0,
    usage_writes: { NOT_ATTEMPTED: 0, WRITE_SUCCEEDED: 1, WRITE_FAILED: 0, WRITE_OUTCOME_UNKNOWN: 0 },
    parsing: { NOT_ATTEMPTED: 0, ACCEPTED: 0, REJECTED: 0, NOT_SEPARATELY_OBSERVABLE: 1 },
    calls: [{ sequence: 1, method: 'SINGLE', item_count: 1, member_ids: [ids.member], omitted_members: 0,
      members_complete: true, metadata_incomplete: false, invocation_outcome: 'RESPONSE_RETURNED',
      usage_write_outcome: 'WRITE_SUCCEEDED', parsing_outcome: 'NOT_SEPARATELY_OBSERVABLE',
      items_saved: 1, items_save_failed: 0, items_not_published: 0, items_pending: 0 }],
    omitted_calls: 0, omitted_members: 0, entries_complete: true, members_complete: true,
    execution_complete: true, stop_reason: 'NONE',
    usage_observation: {
      version: 'ccmai.usage-observation.v1', scope: 'successful_interface_response_local_estimate',
      token_basis: 'INTERFACE_VALUES_PRESENCE_UNAVAILABLE', billing: 'NOT_OBSERVED', price_revision: 'NOT_CAPTURED',
      responses: 1, invalid_tokens: 0, priced: 1, unpriced: 0, invalid_costs: 0,
      token_overflow: false, cost_overflow: false, counter_overflow: false,
      input_tokens: 0, output_tokens: 0, local_estimate_usd: 0, tokens_complete: true, cost_complete: true,
      ...usageOverrides,
    },
  }
}

function savedSummary(usageOverrides: Record<string, unknown> = {}) {
  return JSON.stringify({
    source_execution: sourceExecution(usageOverrides),
    prompt: 'PRIVATE PROMPT MUST NOT BE DISPLAYED',
    provider_response: 'PRIVATE PROVIDER CONTENT MUST NOT BE DISPLAYED',
  })
}

function mountPanel(summary: string, locale: 'en' | 'vi' = 'en') {
  const i18n = createI18n({ legacy: false, locale, fallbackLocale: 'en', messages: { en: enMessages, vi: viMessages } })
  const wrapper = mount(RunObservationPanel, { attachTo: document.body, props: { summary, context }, global: { plugins: [i18n] } })
  mounted.push(wrapper)
  return wrapper
}

describe('RunObservationPanel', () => {
  afterEach(() => {
    for (const wrapper of mounted.splice(0)) wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('renders an unknown local estimate without showing zero', () => {
    const unknown = mountPanel(savedSummary({ priced: 0, unpriced: 1, input_tokens: 0, output_tokens: 0,
      local_estimate_usd: null, cost_complete: false }))
    const unknownEstimate = unknown.find('[data-field="local_estimate_usd"]')
    expect(unknownEstimate.text()).toBe(enMessages.run_obs_unknown)
    expect(unknown.text()).not.toContain('0 USD')
  })

  it('shows a recorded local zero as zero', () => {
    const zero = mountPanel(savedSummary({ local_estimate_usd: 0 }))
    expect(zero.find('[data-field="local_estimate_usd"]').text()).toBe('0 USD')
    expect(zero.find('[data-field="tokens_complete"]').text()).toBe(enMessages.run_obs_yes)
  })

  it('uses translated labels and native keyboard-accessible details in English and Vietnamese', async () => {
    const english = mountPanel(savedSummary(), 'en')
    const vietnamese = mountPanel(savedSummary(), 'vi')
    expect(english.find('summary').text()).toBe(enMessages.run_obs_title)
    expect(vietnamese.find('summary').text()).toBe(viMessages.run_obs_title)
    expect(english.find('[data-section="execution"]').attributes('aria-label')).toBe(enMessages.run_obs_execution)
    expect(vietnamese.find('[data-section="execution"]').attributes('aria-label')).toBe(viMessages.run_obs_execution)

    const details = english.find('details')
    const summary = english.find('summary')
    expect(details.element.tagName).toBe('DETAILS')
    expect(summary.element.tagName).toBe('SUMMARY')
    const summaryElement = summary.element as HTMLElement
    summaryElement.focus()
    expect(document.activeElement).toBe(summary.element)
    summaryElement.click()
    expect((details.element as HTMLDetailsElement).open).toBe(true)
    expect(english.find('button').exists()).toBe(false)
  })

  it('does not render raw summary content or arbitrary unsupported enum strings', () => {
    const bad = sourceExecution()
    bad.stop_reason = '<img src=x onerror=alert(1)>'
    const wrapper = mountPanel(JSON.stringify({ source_execution: bad, prompt: 'PRIVATE PROMPT MUST NOT BE DISPLAYED' }))
    expect(wrapper.text()).not.toContain('PRIVATE PROMPT')
    expect(wrapper.text()).not.toContain('PRIVATE PROVIDER CONTENT')
    expect(wrapper.text()).not.toContain('<img')
    expect(wrapper.find('[data-section="execution"] [data-testid="observation-unavailable"]').exists()).toBe(true)
    expect(wrapper.find('img').exists()).toBe(false)
  })
})
