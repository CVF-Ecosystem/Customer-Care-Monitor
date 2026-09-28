<!-- CCMAI-UX-013: bước 6 — tóm tắt trước khi tạo. Chỉ đọc form, không đổi gì. -->
<template>
  <div class="ws-stack">
    <div>
      <h2 class="ws-h2">{{ $t('job_wizard_step_confirm') }}</h2>
      <p class="ws-intro">{{ $t('jw_confirm_intro') }}</p>
    </div>
    <dl class="ws-summary" data-testid="jw-summary">
      <div><dt>{{ $t('jw_sum_name') }}</dt><dd>{{ form.name || '—' }}</dd></div>
      <div><dt>{{ $t('jw_sum_type') }}</dt><dd>{{ form.job_type === 'classification' ? $t('jw_type_class') : $t('jw_type_qc') }}</dd></div>
      <div><dt>{{ $t('jw_sum_channels') }}</dt><dd>{{ $t('jl_channels', { n: form.input_channel_ids?.length || 0 }) }}</dd></div>
      <div><dt>{{ $t('jw_sum_rules') }}</dt><dd>{{ rulesSummary }}</dd></div>
      <div><dt>{{ $t('jw_sum_outputs') }}</dt><dd>{{ outputCount ? $t('jw_sum_outputs_n', { n: outputCount }) : $t('jw_sum_outputs_none') }}</dd></div>
      <div><dt>{{ $t('jw_sum_analysis') }}</dt><dd>{{ scheduleText(form.schedule_type, form.schedule_cron) }}</dd></div>
      <div>
        <dt>{{ $t('jw_sum_send') }}</dt>
        <dd>
          <template v-if="form.output_schedule === 'none'">{{ $t('output_none') }}</template>
          <template v-else-if="form.output_schedule === 'instant'">{{ $t('output_instant') }}</template>
          <template v-else-if="form.output_schedule === 'cron'">{{ $t('output_scheduled') }} · {{ scheduleText('cron', form.output_cron) }}</template>
          <template v-else>{{ $t('output_once') }} · {{ form.output_at || '—' }}</template>
        </dd>
      </div>
    </dl>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { describeSchedule } from '../../views/Jobs/job-list/schedule'

const form = defineModel<Record<string, any>>('form', { required: true })
const { t } = useI18n()

const outputCount = computed(() => {
  try {
    return JSON.parse(form.value.outputs || '[]').length
  } catch {
    return 0
  }
})

const rulesSummary = computed(() => {
  if (form.value.job_type === 'classification') {
    try {
      return t('jw_sum_rules_class', { n: JSON.parse(form.value.rules_config || '[]').length })
    } catch {
      return '—'
    }
  }
  return t('jw_sum_rules_qc', { n: (form.value.rules_content || '').trim().length })
})

function scheduleText(type: string, cron: string) {
  const s = describeSchedule(type, cron)
  switch (s.kind) {
    case 'after_sync': return t('jl_sched_after_sync')
    case 'manual': return t('jl_sched_manual')
    case 'daily': return t('jl_sched_daily', { time: s.time })
    case 'weekly': return t('jl_sched_weekly', { days: s.days.map((d) => t(`jl_day_${d}`)).join(', '), time: s.time })
    case 'monthly': return t('jl_sched_monthly', { day: s.day, time: s.time })
    default: return t('jl_sched_raw', { cron: s.cron || '—' })
  }
}
</script>

<style scoped>
.ws-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.ws-h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}
.ws-intro {
  margin: 4px 0 0;
  font-size: 14px;
  color: rgb(var(--v-theme-text-muted));
}
.ws-summary {
  margin: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  overflow: hidden;
}
.ws-summary > div {
  display: grid;
  grid-template-columns: minmax(120px, 200px) minmax(0, 1fr);
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid rgb(var(--v-theme-border));
  font-size: 14px;
}
.ws-summary > div:last-child {
  border-bottom: 0;
}
.ws-summary dt {
  color: rgb(var(--v-theme-text-muted));
}
.ws-summary dd {
  margin: 0;
  font-weight: 600;
  overflow-wrap: anywhere;
}
</style>
