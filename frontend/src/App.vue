<template>
  <v-app>
    <AuthLayout v-if="route.meta.layout === 'auth'">
      <router-view />
    </AuthLayout>
    <DefaultLayout v-else>
      <router-view :key="(route.params.tenantId as string) || 'home'" />
    </DefaultLayout>
  </v-app>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DefaultLayout from './layouts/DefaultLayout.vue'
import AuthLayout from './layouts/AuthLayout.vue'
import { useAuthStore } from './stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

onMounted(async () => {
  // Chờ lượt điều hướng đầu tiên: router guard hỏi /setup/status và xóa token của bản cài
  // trước nếu máy chủ chưa thiết lập. Nạp hồ sơ sớm hơn thì token cũ bị 401, interceptor
  // refresh thất bại rồi chuyển cả trang về /login (CCMAI-AUTH-001).
  await router.isReady()
  if (authStore.accessToken && !authStore.user) {
    try {
      await authStore.fetchProfile()
    } catch {
      // Token expired — will redirect via router guard
    }
  }
})
</script>
