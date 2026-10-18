// @vitest-environment happy-dom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import api from '../api'
import OnboardingWizard from '../components/OnboardingWizard.vue'
import viLocale from '../i18n/vi'
import enLocale from '../i18n/en'

vi.mock('../api', () => ({
  default: {
    get: vi.fn(),
    put: vi.fn(),
  },
}))

interface Step {
  key: string
  title: string
  done: boolean
  link: string
}

const mounted: Array<{ unmount: () => void }> = []

afterEach(() => {
  while (mounted.length) mounted.pop()?.unmount()
  vi.clearAllMocks()
})

async function mountWizard(steps: Step[]) {
  vi.mocked(api.get).mockResolvedValue({ data: { steps, dismissed: false } } as never)
  vi.mocked(api.put).mockResolvedValue({} as never)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:tenantId/:page', component: { template: '<div />' } }],
  })
  await router.push('/tenant-a/home')
  const push = vi.spyOn(router, 'push').mockResolvedValue(undefined)
  const wrapper = mount(OnboardingWizard, {
    global: {
      plugins: [
        createVuetify({ components, directives }),
        createI18n({ legacy: false, locale: 'vi', fallbackLocale: 'en', messages: { vi: viLocale, en: enLocale } }),
        router,
      ],
    },
  })
  mounted.push(wrapper)
  await flushPromises()
  return { wrapper, router, push }
}

describe('OnboardingWizard', () => {
  it('uses the existing localized AI settings label and keeps other backend titles', async () => {
    const { wrapper } = await mountWizard([
      { key: 'ai', title: 'Cấu hình AI Provider', done: false, link: 'settings' },
      { key: 'channel', title: 'Kết nối kênh chat', done: false, link: 'channels' },
      { key: 'job', title: 'Create analysis job', done: false, link: 'jobs/create' },
    ])

    expect(wrapper.text()).toContain(viLocale.ai_config)
    expect(wrapper.text()).not.toContain('Cấu hình AI Provider')
    expect(wrapper.text()).toContain('Kết nối kênh chat')
    expect(wrapper.text()).toContain('Create analysis job')
  })

  it('navigates for a pending step and leaves completed steps in place', async () => {
    const { wrapper, push } = await mountWizard([
      { key: 'channel', title: 'Kết nối kênh chat', done: false, link: 'channels' },
      { key: 'ai', title: 'Cấu hình AI Provider', done: true, link: 'settings' },
    ])
    const chip = (title: string) => wrapper.findAll('.v-chip').find(item => item.text().trim() === title)

    await chip('Kết nối kênh chat')!.trigger('click')
    expect(push).toHaveBeenCalledWith('/tenant-a/channels')

    await chip(viLocale.ai_config)!.trigger('click')
    expect(push).toHaveBeenCalledTimes(1)
  })

  it('dismisses locally and preserves the existing settings request', async () => {
    const { wrapper } = await mountWizard([
      { key: 'channel', title: 'Kết nối kênh chat', done: false, link: 'channels' },
    ])

    const dismissButton = wrapper.findAll('.v-btn').at(-1)
    expect(dismissButton).toBeDefined()
    await dismissButton!.trigger('click')
    await flushPromises()

    expect(api.put).toHaveBeenCalledWith('/tenants/tenant-a/settings', {
      key: 'onboarding_dismissed',
      value: 'true',
    })
    expect(wrapper.find('.v-sheet').exists()).toBe(false)
  })
})
