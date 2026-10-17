<template>
  <v-sheet v-if="visible" color="indigo-lighten-5" rounded class="d-flex align-start pa-2 mb-4 ga-2">
    <v-icon color="primary" class="flex-shrink-0">mdi-rocket-launch</v-icon>
    <div class="onboarding-steps d-flex align-center ga-1">
      <span class="text-body-2 font-weight-medium" style="white-space: nowrap;">Bắt đầu:</span>
      <v-chip
        v-for="step in steps"
        :key="step.key"
        size="small"
        :color="step.done ? 'success' : 'default'"
        :variant="step.done ? 'flat' : 'outlined'"
        :prepend-icon="step.done ? 'mdi-check-circle' : 'mdi-circle-outline'"
        class="onboarding-step-chip"
        @click="goToStep(step)"
      >
        <span class="onboarding-step-chip__label">{{ displayStepTitle(step) }}</span>
      </v-chip>
      <v-chip
        v-if="completedCount === steps.length"
        color="success"
        size="small"
        variant="flat"
        prepend-icon="mdi-party-popper"
      >
        Hoàn thành!
      </v-chip>
    </div>
    <v-btn icon="mdi-close" variant="text" size="x-small" class="flex-shrink-0" @click="dismiss" />
  </v-sheet>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api from '../api'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const tenantId = computed(() => route.params.tenantId as string)

interface OnboardingStep {
  key: string
  title: string
  done: boolean
  link: string
}

const steps = ref<OnboardingStep[]>([])
const dismissed = ref(false)
const loaded = ref(false)

const visible = computed(() => {
  if (!loaded.value || dismissed.value) return false
  return steps.value.length > 0
})

const completedCount = computed(() => steps.value.filter(s => s.done).length)

function displayStepTitle(step: OnboardingStep) {
  return step.key === 'ai' ? t('ai_config') : step.title
}

async function loadOnboarding() {
  if (!tenantId.value) return
  loaded.value = false
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/onboarding-status`)
    steps.value = data.steps || []
    dismissed.value = data.dismissed || false
    loaded.value = true
  } catch {
    loaded.value = false
  }
}

onMounted(loadOnboarding)
// Reload when tenant changes (switch company)
watch(tenantId, loadOnboarding)
// Reload when navigating between pages (steps may have been completed)
watch(() => route.path, loadOnboarding)

function goToStep(step: OnboardingStep) {
  if (!step.done) {
    router.push(`/${tenantId.value}/${step.link}`)
  }
}

async function dismiss() {
  dismissed.value = true
  try {
    await api.put(`/tenants/${tenantId.value}/settings`, {
      key: 'onboarding_dismissed',
      value: 'true',
    })
  } catch { /* ignore */ }
}
</script>

<style scoped>
.onboarding-steps {
  flex: 1 1 0;
  flex-wrap: wrap;
  overflow: visible;
  min-width: 0;
}

.onboarding-step-chip {
  flex: 0 1 auto;
  height: auto !important;
  max-width: 100%;
  min-width: 0;
  min-height: var(--v-chip-height, 24px);
}

.onboarding-step-chip :deep(.v-chip__content) {
  max-width: 100%;
  min-width: 0;
  white-space: normal;
}

.onboarding-step-chip__label {
  min-width: 0;
  overflow-wrap: anywhere;
  white-space: normal;
}
</style>
