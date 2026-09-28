<!-- CCMAI-UX-013: bước 5 — lịch phân tích và lịch gửi kết quả. Giá trị lưu giữ nguyên. -->
<template>
  <div class="ws-stack">
    <section class="ws-section">
      <h2 class="ws-h2">{{ $t('job_wizard_step_analysis_schedule') }}</h2>
      <p class="ws-intro">{{ $t('analysis_schedule_desc') }}</p>
      <v-radio-group v-model="form.schedule_type" hide-details>
        <v-radio value="cron" :label="$t('schedule_cron')" />
        <v-radio value="after_sync" :label="$t('schedule_after_sync')" />
        <v-radio value="manual" :label="$t('schedule_manual')" />
      </v-radio-group>
      <CronPicker v-if="form.schedule_type === 'cron'" v-model="form.schedule_cron" class="ws-indent" />
    </section>

    <v-divider />

    <section class="ws-section">
      <h2 class="ws-h2">{{ $t('job_wizard_step_output_schedule') }}</h2>
      <p class="ws-intro">{{ $t('output_schedule_desc') }}</p>
      <v-radio-group v-model="form.output_schedule" hide-details>
        <v-radio value="none" :label="$t('output_none')" />
        <v-radio value="instant" :label="$t('output_instant')" />
        <v-radio value="cron" :label="$t('output_scheduled')" />
        <v-radio value="scheduled" :label="$t('output_once')" />
      </v-radio-group>
      <CronPicker v-if="form.output_schedule === 'cron'" v-model="form.output_cron" class="ws-indent" />
      <v-text-field
        v-if="form.output_schedule === 'scheduled'"
        v-model="form.output_at"
        :label="$t('send_at')"
        type="datetime-local"
        variant="outlined"
        class="ws-indent"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import CronPicker from '../CronPicker.vue'

const form = defineModel<Record<string, any>>('form', { required: true })
</script>

<style scoped>
.ws-stack {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.ws-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.ws-h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}
.ws-intro {
  margin: 0;
  font-size: 14px;
  color: rgb(var(--v-theme-text-muted));
}
.ws-indent {
  margin-left: 40px;
  max-width: 520px;
}
@media (max-width: 599px) {
  .ws-indent {
    margin-left: 0;
  }
}
</style>
