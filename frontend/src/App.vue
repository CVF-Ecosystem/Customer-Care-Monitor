<template>
  <v-app>
    <!-- CCMAI-RUNTIME-036: until the first navigation settled (setup status confirmed or found
         unavailable) only the neutral auth layout is mounted; the default layout and every
         protected view wait for a confirmed state. -->
    <AuthLayout v-if="!ready">
      <div class="text-center py-6" role="status" aria-live="polite" data-testid="setup-loading">
        <v-progress-circular indeterminate color="primary" class="mb-3" />
        <div class="text-body-2">{{ $t('setup_status_loading') }}</div>
      </div>
    </AuthLayout>
    <AuthLayout v-else-if="route.meta.layout === 'auth'">
      <router-view />
    </AuthLayout>
    <DefaultLayout v-else>
      <router-view :key="(route.params.tenantId as string) || 'home'" />
    </DefaultLayout>
  </v-app>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DefaultLayout from './layouts/DefaultLayout.vue'
import AuthLayout from './layouts/AuthLayout.vue'
import { useAuthStore } from './stores/auth'
import { setupStatus } from './router/setupStatus'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const ready = ref(false)
let profileRequested = false

// Hồ sơ chỉ được nạp sau khi trạng thái cài đặt được máy chủ xác nhận là đã cấu hình: nạp sớm
// hơn thì token cũ bị 401, interceptor refresh thất bại rồi chuyển cả trang về /login
// (CCMAI-AUTH-001). Một lần duy nhất dù có nhiều thông báo đồng thời (CCMAI-RUNTIME-036).
async function loadProfileOnce() {
  if (profileRequested || !ready.value || setupStatus.state !== 'configured') return
  if (!authStore.accessToken || authStore.user) return
  profileRequested = true
  try {
    await authStore.fetchProfile()
  } catch {
    // Token expired — will redirect via router guard
  }
}

onMounted(async () => {
  // Chờ lượt điều hướng đầu tiên: router guard hỏi /setup/status (hoặc chuyển sang trang
  // không xác định được trạng thái) và xóa token của bản cài trước nếu máy chủ chưa thiết lập.
  await router.isReady()
  ready.value = true
  await loadProfileOnce()
})

// App can be mounted while the status is unavailable; a later confirmed "configured" resumes it.
watch(() => setupStatus.state, () => { void loadProfileOnce() })
</script>
