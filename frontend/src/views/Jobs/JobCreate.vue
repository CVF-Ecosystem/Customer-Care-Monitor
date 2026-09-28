<!--
  Tạo tác vụ AI — wizard 6 bước.
  CCMAI-UX-013: trình bày lại; dữ liệu gửi đi, điều kiện qua từng bước và cổng "gửi thử
  đầu ra" giữ nguyên.
-->
<template>
  <div class="jw-page">
    <div>
      <router-link :to="`/${tenantId}/jobs`" class="jw-back"><v-icon size="18">mdi-chevron-left</v-icon>{{ $t('jw_back_list') }}</router-link>
      <h1 class="jw-title">{{ $t('jw_create_title') }}</h1>
    </div>

    <section class="jw-card">
      <!-- Thanh bước: desktop đủ 6 nhãn, mobile "Bước n/6" + thanh tiến độ -->
      <ol v-if="mdAndUp" class="jw-steps" data-testid="jw-steps">
        <li
          v-for="s in stepItems"
          :key="s.value"
          class="jw-step"
          :class="{ 'jw-step--done': s.value < step, 'jw-step--now': s.value === step }"
          :aria-current="s.value === step ? 'step' : undefined"
        >
          <span class="jw-step__bar" aria-hidden="true" />
          <span>{{ s.value }} · {{ s.title }}</span>
        </li>
      </ol>
      <div v-else class="jw-steps-mobile" data-testid="jw-steps">
        <span class="jw-muted">{{ $t('jw_step_of', { n: step, total: stepItems.length }) }} · <b class="jw-strong">{{ stepItems[step - 1].title }}</b></span>
        <v-progress-linear :model-value="(step / stepItems.length) * 100" color="primary" height="6" rounded />
      </div>

      <!-- v-show, không v-if: mỗi bước giữ trạng thái khi quay lại. StepOutput khi mount lại
           sẽ coi mọi đầu ra đã có là đã gửi thử, nên không được mount lại giữa chừng. -->
      <div class="jw-body">
        <StepType v-show="step === 1" v-model:form="form" />
        <StepInput v-show="step === 2" v-model:form="form" />
        <StepRules v-show="step === 3" v-model:form="form" />
        <StepOutput v-show="step === 4" v-model:form="form" />
        <StepOutputSchedule v-show="step === 5" v-model:form="form" />
        <StepConfirm v-show="step === 6" v-model:form="form" />
      </div>

      <div v-if="createError" class="jw-error" role="alert" data-testid="jw-create-error">{{ createError }}</div>

      <footer class="jw-footer" :class="{ 'jw-footer--mobile': !mdAndUp }">
        <span v-if="!canProceed && step < 6" class="jw-reason" role="status" data-testid="jw-reason">{{ validationMessage }}</span>
        <div class="jw-footer__buttons">
          <v-btn v-if="step > 1" variant="text" height="44" prepend-icon="mdi-chevron-left" @click="step--">{{ $t('jw_back') }}</v-btn>
          <v-spacer />
          <v-btn v-if="step < 6" color="primary" height="44" :disabled="!canProceed" append-icon="mdi-chevron-right" data-testid="jw-next" @click="step++">
            {{ $t('jw_next') }}
          </v-btn>
          <v-btn v-else color="primary" height="44" :loading="creating" prepend-icon="mdi-check" data-testid="jw-submit" @click="submitJob">
            {{ $t('jw_submit') }}
          </v-btn>
        </div>
      </footer>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDisplay } from 'vuetify'
import { useI18n } from 'vue-i18n'
import { useJobStore } from '../../stores/jobs'
import StepType from '../../components/JobWizard/StepType.vue'
import StepInput from '../../components/JobWizard/StepInput.vue'
import StepRules from '../../components/JobWizard/StepRules.vue'
import StepOutput from '../../components/JobWizard/StepOutput.vue'
import StepOutputSchedule from '../../components/JobWizard/StepOutputSchedule.vue'
import StepConfirm from '../../components/JobWizard/StepConfirm.vue'

const route = useRoute()
const router = useRouter()
const { mdAndUp } = useDisplay()
const { t } = useI18n()
const jobStore = useJobStore()

const tenantId = computed(() => route.params.tenantId as string)
const step = ref(1)
const creating = ref(false)
const createError = ref('')

const canProceed = computed(() => {
  switch (step.value) {
    case 1: return form.value.name.trim().length >= 2
    case 2: return form.value.input_channel_ids.length > 0
    case 3: {
      if (form.value.job_type === 'classification') {
        try { return JSON.parse(form.value.rules_config || '[]').length > 0 } catch { return false }
      }
      return form.value.rules_content.trim().length > 0
    }
    case 4: {
      try {
        const outputs = JSON.parse(form.value.outputs || '[]')
        if (outputs.length === 0) return true // optional — no output configured
        // All outputs must have valid fields AND pass test
        return form.value.outputs_validated === true
      } catch { return false }
    }
    case 5: {
      if (form.value.schedule_type === 'cron' && !form.value.schedule_cron.trim()) return false
      if (form.value.output_schedule === 'cron' && !form.value.output_cron.trim()) return false
      return true
    }
    default: return true
  }
})

const validationMessage = computed(() => {
  switch (step.value) {
    case 1: return t('validation_min_chars', { min: 2 })
    case 2: return t('validation_select_channel')
    case 3: return t('validation_enter_rules')
    case 4: return t('jw_output_invalid')
    default: return ''
  }
})

const form = ref({
  name: '',
  description: '',
  job_type: 'qc_analysis',
  input_channel_ids: [] as string[],
  rules_content: '',
  rules_config: '[]',
  skip_conditions: '',
  ai_provider: '',
  ai_model: '',
  outputs: '[]',
  outputs_validated: true,
  output_schedule: 'instant',
  output_cron: '',
  output_at: '',
  schedule_type: 'cron',
  schedule_cron: '0 7 * * *',
})

const stepItems = computed(() => [
  { title: t('jw_step_type'), value: 1 },
  { title: t('job_wizard_step_input'), value: 2 },
  { title: t('job_wizard_step_rules'), value: 3 },
  { title: t('job_wizard_step_output'), value: 4 },
  { title: t('job_wizard_step_schedule'), value: 5 },
  { title: t('job_wizard_step_confirm'), value: 6 },
])

async function submitJob() {
  creating.value = true
  createError.value = ''
  try {
    const payload = {
      ...form.value,
      outputs: JSON.parse(form.value.outputs || '[]'),
      rules_config: form.value.job_type === 'classification' ? JSON.parse(form.value.rules_config) : undefined,
      output_at: form.value.output_at || undefined,
    }
    await jobStore.createJob(tenantId.value, payload)
    router.push(`/${tenantId.value}/jobs`)
  } catch (err: any) {
    const msg = err?.response?.data?.error
    createError.value = msg ? t('jw_create_error', { msg }) : t('jw_create_error_generic')
  } finally {
    creating.value = false
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
.jw-card {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 20px 24px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.jw-steps {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.jw-step {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  color: rgb(var(--v-theme-text-muted));
}
.jw-step__bar {
  height: 4px;
  border-radius: 2px;
  background: rgb(var(--v-theme-border));
}
.jw-step--done {
  color: rgb(var(--v-theme-on-surface));
}
.jw-step--done .jw-step__bar,
.jw-step--now .jw-step__bar {
  background: rgb(var(--v-theme-primary));
}
.jw-step--now {
  color: rgb(var(--v-theme-primary));
  font-weight: 700;
}
.jw-steps-mobile {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 14px;
}
.jw-muted {
  color: rgb(var(--v-theme-text-muted));
}
.jw-strong {
  color: rgb(var(--v-theme-on-surface));
}
.jw-error {
  padding: 12px 14px;
  border: 1px solid rgb(var(--v-theme-fail));
  border-radius: 10px;
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
  font-size: 14px;
  font-weight: 600;
}
.jw-footer {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid rgb(var(--v-theme-border));
}
.jw-footer__buttons {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
}
.jw-footer--mobile {
  flex-direction: column;
  align-items: stretch;
}
.jw-reason {
  order: 0;
  font-size: 13px;
  color: rgb(var(--v-theme-fail));
}
.jw-footer:not(.jw-footer--mobile) .jw-reason {
  order: 1;
  max-width: 360px;
  text-align: right;
}
.jw-footer:not(.jw-footer--mobile) .jw-footer__buttons {
  order: 0;
}
</style>
