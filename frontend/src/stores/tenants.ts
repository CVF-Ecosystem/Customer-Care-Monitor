import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../api'

interface Tenant {
  id: string
  name: string
  slug: string
  channels_count?: number
  jobs_count?: number
}

export const useTenantStore = defineStore('tenants', () => {
  const tenants = ref<Tenant[]>([])
  const currentTenant = ref<Tenant | null>(null)

  async function fetchTenants() {
    const { data } = await api.get('/tenants')
    tenants.value = data
  }

  function setCurrentTenant(tenant: Tenant) {
    currentTenant.value = tenant
  }

  return { tenants, currentTenant, fetchTenants, setCurrentTenant }
})
