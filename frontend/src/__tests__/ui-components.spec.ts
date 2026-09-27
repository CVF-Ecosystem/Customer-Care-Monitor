// @vitest-environment happy-dom
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { createI18n } from 'vue-i18n'
import { createRouter, createMemoryHistory } from 'vue-router'
import vi from '../i18n/vi'
import en from '../i18n/en'
import SourceStatusChip from '../components/ui/SourceStatusChip.vue'
import SourceStatusPanel from '../components/ui/SourceStatusPanel.vue'
import SyncStatusChip from '../components/ui/SyncStatusChip.vue'
import MetricCard from '../components/ui/MetricCard.vue'
import ConfidenceText from '../components/ui/ConfidenceText.vue'
import VerdictChip from '../components/ui/VerdictChip.vue'
import ActionMenu from '../components/ui/ActionMenu.vue'
import ResultCard from '../components/ui/ResultCard.vue'
import { lightColors } from '../styles/tokens'

// CCMAI-UX-000: shared components must render the reviewed semantics, not soften them.
function plugins() {
  const vuetify = createVuetify({ components, directives, theme: { themes: { light: { colors: lightColors } } } })
  const i18n = createI18n({ legacy: false, locale: 'vi', fallbackLocale: 'en', messages: { vi, en } })
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { template: '<div />' } }] })
  return [vuetify, i18n, router]
}

// Props are checked by each component at runtime; the helper stays untyped on purpose.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyComponent = any

function mountUi(component: AnyComponent, props: Record<string, unknown>) {
  return mount(component as AnyComponent, { props, global: { plugins: plugins() } })
}

describe('SourceStatusChip', () => {
  it('renders each reviewed status with its existing label', () => {
    const cases: [string, string][] = [
      ['changed_since_analysis', vi.results_source_changed],
      ['verification_unavailable', vi.results_source_unavailable],
      ['legacy_unverified', vi.results_source_legacy],
      ['bound_currentness_unverified', vi.results_source_unverified],
    ]
    for (const [status, label] of cases) {
      const w = mountUi(SourceStatusChip, { status })
      expect(w.text()).toContain(label)
      expect(w.attributes('data-status')).toBe(status)
    }
  })

  it('renders a missing or unknown status as unavailable instead of hiding it', () => {
    for (const status of [undefined, null, '', 'fresh']) {
      const w = mountUi(SourceStatusChip, { status })
      expect(w.attributes('data-status')).toBe('verification_unavailable')
      expect(w.text()).toContain(vi.results_source_unavailable)
    }
  })

  it('never uses a check icon', () => {
    for (const status of ['changed_since_analysis', 'verification_unavailable', 'legacy_unverified', 'bound_currentness_unverified']) {
      expect(mountUi(SourceStatusChip, { status }).html()).not.toContain('mdi-check')
    }
  })
})

describe('SourceStatusPanel', () => {
  it('always shows the local-only note, even with no results', () => {
    for (const statuses of [[], ['legacy_unverified'], ['changed_since_analysis', 'x']]) {
      const w = mountUi(SourceStatusPanel, { statuses })
      expect(w.find('[data-testid="source-note"]').text()).toBe(vi.results_source_note)
    }
  })

  it('lists counts most-concerning first and counts unknown values as unavailable', () => {
    const w = mountUi(SourceStatusPanel, {
      statuses: ['bound_currentness_unverified', 'changed_since_analysis', 'weird', 'changed_since_analysis'],
    })
    const items = w.findAll('li')
    expect(items.map((i) => i.attributes('data-status'))).toEqual([
      'changed_since_analysis',
      'verification_unavailable',
      'bound_currentness_unverified',
    ])
    expect(items[0].text()).toContain('2')
    expect(w.classes()).toContain('ccma-src-panel--alert')
  })
})

describe('SyncStatusChip', () => {
  it('says syncing has started, not finished, and never shows unknown as success', () => {
    const syncing = mountUi(SyncStatusChip, { status: 'syncing' })
    expect(syncing.text()).toContain(vi.ui_sync_syncing)
    expect(syncing.attributes('title')).toBe(vi.ui_sync_started_hint)
    expect(mountUi(SyncStatusChip, { status: '' }).text()).toContain(vi.ui_sync_never)
    const odd = mountUi(SyncStatusChip, { status: 'done' })
    expect(odd.attributes('data-sync')).toBe('unknown')
    expect(odd.text()).not.toContain(vi.ui_sync_success)
  })
})

describe('MetricCard', () => {
  it('renders null as a dash, never 0', () => {
    const w = mountUi(MetricCard, { label: 'Điểm trung bình', value: null })
    expect(w.find('[data-testid="metric-value"]').text()).toBe('—')
    const zero = mountUi(MetricCard, { label: 'Không đạt', value: 0 })
    expect(zero.find('[data-testid="metric-value"]').text()).toBe('0')
  })

  it('becomes a real link when it drills down', () => {
    const w = mountUi(MetricCard, { label: 'Không đạt', value: 4, to: '/results?verdict=fail' })
    expect(w.element.tagName).toBe('A')
    expect(w.attributes('href')).toBe('/results?verdict=fail')
  })
})

describe('ConfidenceText', () => {
  it('shows "Không có" for missing or unavailable confidence, never a percentage', () => {
    for (const [confidence, basis] of [[null, 'unavailable'], [1, 'unavailable'], [0.8, undefined]] as const) {
      const w = mountUi(ConfidenceText, { confidence, basis })
      expect(w.text()).toContain(vi.ui_confidence_none)
      expect(w.text()).not.toMatch(/\d+%/)
    }
  })

  it('always qualifies a model-reported value as uncalibrated', () => {
    const w = mountUi(ConfidenceText, { confidence: 0.72, basis: 'model_reported_uncalibrated' })
    expect(w.text()).toContain('72%')
    expect(w.text()).toContain(vi.ui_confidence_qualifier)
  })
})

describe('VerdictChip', () => {
  it('renders icon and text for each verdict', () => {
    expect(mountUi(VerdictChip, { verdict: 'pass' }).text()).toContain(vi.verdict_pass)
    expect(mountUi(VerdictChip, { verdict: 'fail' }).text()).toContain(vi.verdict_fail)
    expect(mountUi(VerdictChip, { verdict: 'skip' }).text()).toContain(vi.verdict_skip)
    expect(mountUi(VerdictChip, { verdict: 'fail' }).find('.v-icon').exists()).toBe(true)
  })
})

describe('ActionMenu', () => {
  it('has an accessible 44px trigger and only emits the selected key', async () => {
    // happy-dom has no visualViewport; Vuetify's overlay positioning reads it.
    const g = globalThis as unknown as { visualViewport?: unknown }
    g.visualViewport ??= {
      width: 1024, height: 768, offsetLeft: 0, offsetTop: 0, pageLeft: 0, pageTop: 0, scale: 1,
      addEventListener() {}, removeEventListener() {},
    }
    const w = mount(ActionMenu as AnyComponent, {
      props: {
        items: [
          { key: 'delete', label: 'Xóa', danger: true },
          { key: 'export', label: 'Xuất' },
        ],
      },
      global: { plugins: plugins() },
      attachTo: document.body,
    })
    const trigger = w.find('[data-testid="action-menu"]')
    expect(trigger.attributes('aria-label')).toBe(vi.ui_more_actions)
    expect(trigger.attributes('style')).toContain('width: 44px')
    expect(trigger.attributes('style')).toContain('height: 44px')
    await trigger.trigger('click')
    await new Promise((r) => setTimeout(r, 0))
    const keys = Array.from(document.querySelectorAll('[data-key]')).map((el) => el.getAttribute('data-key'))
    expect(keys).toEqual(['export', 'delete']) // danger items last
    ;(document.querySelector('[data-key="delete"]') as HTMLElement).click()
    expect(w.emitted('select')).toEqual([['delete']])
    w.unmount()
  })
})

describe('ResultCard', () => {
  it('always shows a source chip, including an unknown status', () => {
    const w = mountUi(ResultCard, { customerName: 'Lê Thu Hà', time: '5 phút trước', verdict: 'pass' })
    expect(w.find('[data-status="verification_unavailable"]').exists()).toBe(true)
    expect(w.element.tagName).toBe('BUTTON')
  })
})
