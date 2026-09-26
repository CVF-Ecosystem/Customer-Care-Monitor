<template>
  <div class="d-flex justify-center align-center pa-8">
    <v-progress-circular v-if="loading" indeterminate color="primary" />
    <v-alert v-else type="error" variant="tonal" max-width="560">
      {{ errorMessage }}
    </v-alert>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useTenantStore } from '../stores/tenants'

const router = useRouter()
const workspaceStore = useTenantStore()
const loading = ref(true)
const errorMessage = ref('Không tìm thấy không gian làm việc. Vui lòng kiểm tra cài đặt.')

onMounted(async () => {
  try {
    await workspaceStore.fetchTenants()
    if (workspaceStore.tenants.length !== 1) {
      errorMessage.value = 'Dữ liệu không hợp lệ: ứng dụng chỉ hỗ trợ một không gian làm việc.'
      return
    }
    const workspace = workspaceStore.tenants[0]
    workspaceStore.setCurrentTenant(workspace)
    await router.replace(`/${workspace.id}`)
  } catch {
    errorMessage.value = 'Không tải được không gian làm việc.'
  } finally {
    loading.value = false
  }
})
</script>
