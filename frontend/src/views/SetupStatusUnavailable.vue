<template>
  <v-card class="pa-6 text-center" variant="outlined" role="alert" data-testid="setup-unavailable">
    <v-icon size="40" color="warning" class="mb-3" aria-hidden="true">mdi-lan-disconnect</v-icon>
    <h2 class="text-h6 mb-2">{{ $t('setup_status_unavailable_title') }}</h2>
    <p class="text-body-2 mb-4">{{ $t('setup_status_unavailable_desc') }}</p>
    <v-btn
      color="primary"
      data-testid="setup-retry"
      :disabled="setupStatus.retrying"
      :loading="setupStatus.retrying"
      :aria-busy="setupStatus.retrying ? 'true' : 'false'"
      @click="retry"
    >
      {{ setupStatus.retrying ? $t('setup_status_retrying') : $t('setup_status_retry') }}
    </v-btn>
  </v-card>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { retrySetupStatus, setupStatus } from '../router/setupStatus'

const router = useRouter()

// Explicit user action only. A confirmed answer continues through the fixed local entry route
// ("/") and the existing guards; a failure stays here with credentials untouched.
async function retry() {
  if (setupStatus.retrying) return
  const state = await retrySetupStatus()
  if (state !== 'unavailable') await router.replace('/')
}
</script>
