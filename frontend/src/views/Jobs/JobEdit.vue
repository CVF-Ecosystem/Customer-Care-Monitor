<!--
  Sửa tác vụ AI.
  CCMAI-UX-013: nếu không tải được tác vụ thì KHÔNG hiện biểu mẫu — trước đây biểu mẫu
  hiện giá trị mặc định và bấm Lưu sẽ ghi đè tác vụ bằng dữ liệu trống. Dữ liệu gửi đi
  khi lưu giữ nguyên.
-->
<template>
  <div class="jw-page">
    <div>
      <router-link :to="`/${tenantId}/jobs/${jobId}`" class="jw-back"><v-icon size="18">mdi-chevron-left</v-icon>{{ loadedName || $t('jw_back_list') }}</router-link>
      <h1 class="jw-title">{{ $t('jw_edit_title') }}</h1>
    </div>

    <v-skeleton-loader v-if="loading" type="article, article" />

    <div v-else-if="loadError" class="jw-error jw-error--row" role="alert" data-testid="jw-edit-load-error">
      <span>{{ $t('jw_edit_load_error') }}</span>
      <v-btn variant="outlined" color="error" height="44" @click="load">{{ $t('jl_retry') }}</v-btn>
    </div>

    <template v-else>
      <v-expansion-panels v-model="openPanels" multiple class="jw-panels" data-testid="jw-edit-form">
        <v-expansion-panel value="info">
          <v-expansion-panel-title>{{ $t('jw_step_type') }}</v-expansion-panel-title>
          <v-expansion-panel-text><StepType v-model:form="form" /></v-expansion-panel-text>
        </v-expansion-panel>
        <v-expansion-panel value="input">
          <v-expansion-panel-title>
            {{ $t('job_wizard_step_input') }}
            <span class="jw-count">{{ $t('jw_selected', { n: form.input_channel_ids?.length || 0 }) }}</span>
          </v-expansion-panel-title>
          <v-expansion-panel-text><StepInput v-model:form="form" /></v-expansion-panel-text>
        </v-expansion-panel>
        <v-expansion-panel value="rules">
          <v-expansion-panel-title>{{ $t('job_wizard_step_rules') }}</v-expansion-panel-title>
          <v-expansion-panel-text><StepRules v-model:form="form" /></v-expansion-panel-text>
        </v-expansion-panel>
        <v-expansion-panel value="output">
          <v-expansion-panel-title>
            {{ $t('job_wizard_step_output') }}
            <span class="jw-count">{{ $t('jw_sum_outputs_n', { n: outputCount }) }}</span>
          </v-expansion-panel-title>
          <v-expansion-panel-text><StepOutput v-model:form="form" /></v-expansion-panel-text>
        </v-expansion-panel>
        <v-expansion-panel value="schedule">
          <v-expansion-panel-title>{{ $t('job_wizard_step_schedule') }}</v-expansion-panel-title>
          <v-expansion-panel-text><StepOutputSchedule v-model:form="form" /></v-expansion-panel-text>
        </v-expansion-panel>
      </v-expansion-panels>

      <div v-if="saveError" class="jw-error" role="alert" data-testid="jw-save-error">{{ $t('jw_save_error') }}</div>

      <div class="jw-actions">
        <v-btn variant="text" height="44" :to="`/${tenantId}/jobs/${jobId}`">{{ $t('jw_cancel') }}</v-btn>
        <v-btn color="primary" height="44" :loading="saving" prepend-icon="mdi-content-save-outline" data-testid="jw-save" @click="saveJob">
          {{ $t('jw_save') }}
        </v-btn>
      </div>
    </template>

    <v-snackbar v-model="showSuccess" :timeout="2000">{{ $t('jw_updated') }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useJobStore } from '../../stores/jobs'
import StepType from '../../components/JobWizard/StepType.vue'
import StepInput from '../../components/JobWizard/StepInput.vue'
import StepRules from '../../components/JobWizard/StepRules.vue'
import StepOutput from '../../components/JobWizard/StepOutput.vue'
import StepOutputSchedule from '../../components/JobWizard/StepOutputSchedule.vue'

const route = useRoute()
const router = useRouter()
const jobStore = useJobStore()

const tenantId = computed(() => route.params.tenantId as string)
const jobId = computed(() => route.params.jobId as string)

const loading = ref(true)
const loadError = ref(false)
const saving = ref(false)
const saveError = ref(false)
const showSuccess = ref(false)
const openPanels = ref(['info'])
const loadedName = ref('')

const form = ref<Record<string, any>>({
  name: '',
  description: '',
  job_type: 'qc_analysis',
  input_channel_ids: [] as string[],
  rules_content: '',
  rules_config: '[]',
  skip_conditions: '',
  ai_provider: 'claude',
  ai_model: '',
  outputs: '[]',
  output_schedule: 'instant',
  output_cron: '',
  output_at: '',
  schedule_type: 'cron',
  schedule_cron: '0 7 * * *',
})

const outputCount = computed(() => {
  try {
    return JSON.parse(form.value.outputs || '[]').length
  } catch { return 0 }
})

async function load() {
  loading.value = true
  loadError.value = false
  try {
    const job = await jobStore.fetchJob(tenantId.value, jobId.value)
    // Map job data to form
    form.value = {
      name: job.name || '',
      description: job.description || '',
      job_type: job.job_type || 'qc_analysis',
      input_channel_ids: JSON.parse(job.input_channel_ids || '[]'),
      rules_content: job.rules_content || '',
      rules_config: job.rules_config || '[]',
      skip_conditions: job.skip_conditions || '',
      ai_provider: job.ai_provider || 'claude',
      ai_model: job.ai_model || '',
      outputs: job.outputs || '[]',
      output_schedule: job.output_schedule || 'instant',
      output_cron: job.output_cron || '',
      output_at: job.output_at || '',
      schedule_type: job.schedule_type || 'cron',
      schedule_cron: job.schedule_cron || '0 7 * * *',
    }
    loadedName.value = job.name || ''
  } catch {
    loadError.value = true
  } finally {
    loading.value = false
  }
}

onMounted(load)

async function saveJob() {
  saving.value = true
  saveError.value = false
  try {
    await jobStore.updateJob(tenantId.value, jobId.value, {
      name: form.value.name,
      description: form.value.description,
      rules_content: form.value.rules_content,
      rules_config: form.value.rules_config,
      skip_conditions: form.value.skip_conditions,
      input_channel_ids: Array.isArray(form.value.input_channel_ids)
        ? form.value.input_channel_ids
        : JSON.parse(form.value.input_channel_ids || '[]'),
      outputs: typeof form.value.outputs === 'string'
        ? JSON.parse(form.value.outputs || '[]')
        : form.value.outputs || [],
      output_schedule: form.value.output_schedule,
      output_cron: form.value.output_cron,
      output_at: form.value.output_at || null,
      schedule_type: form.value.schedule_type,
      schedule_cron: form.value.schedule_cron,
    })
    showSuccess.value = true
    setTimeout(() => router.push(`/${tenantId.value}/jobs/${jobId.value}`), 1500)
  } catch {
    saveError.value = true
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.jw-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.jw-back {
  display: inline-flex;
  align-items: center;
  min-height: 32px;
  color: rgb(var(--v-theme-primary));
  font-size: 14px;
  text-decoration: none;
}
.jw-title {
  margin: 2px 0 0;
  font-size: 26px;
  font-weight: 700;
  line-height: 1.25;
}
.jw-panels :deep(.v-expansion-panel-title) {
  font-weight: 600;
  min-height: 56px;
}
.jw-count {
  margin-left: 10px;
  font-size: 12px;
  font-weight: 500;
  color: rgb(var(--v-theme-text-muted));
}
.jw-error {
  padding: 12px 14px;
  border: 1px solid rgb(var(--v-theme-fail));
  border-radius: 10px;
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
  font-size: 14px;
  font-weight: 600;
  line-height: 1.45;
}
.jw-error--row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
}
.jw-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
