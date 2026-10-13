// @vitest-environment happy-dom
// R091: exercise the registered router guard for notification history using synthetic membership.
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const apiGet = vi.fn()
vi.mock('../api', () => ({ default: { get: (...args: unknown[]) => apiGet(...args) } }))

import router, { clearPermissionDeniedMsg, clearTenantCache, markSetupComplete, permissionDeniedMsg } from '../router'
import { useAuthStore } from '../stores/auth'

describe('notification route permission (R091)', () => {
  let role = 'member'
  let permissions = '{"settings":"r"}'
  let tenantID = ''
  let tenantNumber = 0

  beforeEach(() => {
    role = 'member'
    permissions = '{"settings":"r"}'
    tenantID = `r091-${++tenantNumber}`
    apiGet.mockReset()
    localStorage.setItem('cqa_access_token', 'r091-synthetic-token')
    apiGet.mockImplementation((url: string) => {
      if (url === `/tenants/${tenantID}`) return Promise.resolve({ data: { id: tenantID } })
      if (url === `/tenants/${tenantID}/me`) return Promise.resolve({ data: { role, permissions } })
      return Promise.resolve({ data: {} })
    })
    setActivePinia(createPinia())
    useAuthStore().tenantPerms = { role: '', permissions: {} }
    clearTenantCache()
    clearPermissionDeniedMsg()
    markSetupComplete()
  })

  it('allows settings:r and checks the registered notifications route', async () => {
    await router.push('/')
    await router.push(`/${tenantID}/notifications`)
    expect(router.currentRoute.value.name).toBe('notifications')
    expect(apiGet).toHaveBeenCalledWith(`/tenants/${tenantID}/me`)
    expect(permissionDeniedMsg).toBe('')
  })

  it('denies jobs:r without settings:r at the registered notifications route', async () => {
    permissions = '{"jobs":"r"}'
    await router.push('/')
    await router.push(`/${tenantID}/notifications`)
    expect(router.currentRoute.value.path).toBe(`/${tenantID}`)
    expect(permissionDeniedMsg).not.toBe('')
  })

  it('denies a member when settings permission is missing', async () => {
    permissions = '{}'
    await router.push('/')
    await router.push(`/${tenantID}/notifications`)
    expect(router.currentRoute.value.path).toBe(`/${tenantID}`)
    expect(permissionDeniedMsg).not.toBe('')
  })

  it.each(['owner', 'admin'])('allows the %s role at the registered notifications route', async (memberRole) => {
    role = memberRole
    permissions = '{}'
    await router.push('/')
    await router.push(`/${tenantID}/notifications`)
    expect(router.currentRoute.value.name).toBe('notifications')
    expect(permissionDeniedMsg).toBe('')
  })
})
