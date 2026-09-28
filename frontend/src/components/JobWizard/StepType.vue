<!-- CCMAI-UX-013: bước 1 — tên và loại tác vụ. Hai lựa chọn là thẻ bấm được (ngữ nghĩa radio). -->
<template>
  <div class="ws-stack">
    <p class="ws-intro">{{ $t('jw_type_intro') }}</p>
    <v-text-field
      v-model="form.name"
      :label="$t('jw_name')"
      :rules="[v => !!v || $t('validation_required'), v => (v && v.length >= 2) || $t('validation_min_chars', { min: 2 })]"
      variant="outlined"
    />
    <v-textarea v-model="form.description" :label="$t('jw_desc')" rows="2" auto-grow variant="outlined" />

    <fieldset class="ws-fieldset">
      <legend class="ws-legend">{{ $t('jw_type_label') }}</legend>
      <v-radio-group v-model="form.job_type" hide-details class="ws-options">
        <label v-for="o in options" :key="o.value" class="ws-option" :class="{ 'ws-option--on': form.job_type === o.value }">
          <v-radio :value="o.value" :aria-label="o.title" />
          <span class="ws-option__text">
            <span class="ws-option__title">{{ o.title }}</span>
            <span class="ws-option__desc">{{ o.desc }}</span>
          </span>
        </label>
      </v-radio-group>
    </fieldset>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const form = defineModel<Record<string, any>>('form', { required: true })
const { t } = useI18n()
const options = computed(() => [
  { value: 'qc_analysis', title: t('jw_type_qc'), desc: t('jw_type_qc_desc') },
  { value: 'classification', title: t('jw_type_class'), desc: t('jw_type_class_desc') },
])
</script>

<style scoped>
.ws-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.ws-intro {
  margin: 0;
  font-size: 14px;
  color: rgb(var(--v-theme-text-muted));
}
.ws-fieldset {
  border: 0;
  margin: 0;
  padding: 0;
}
.ws-legend {
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 600;
}
.ws-options :deep(.v-selection-control-group) {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 10px;
}
.ws-option {
  display: flex;
  align-items: flex-start;
  gap: 4px;
  padding: 10px 14px 12px 6px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  cursor: pointer;
}
.ws-option--on {
  border: 2px solid rgb(var(--v-theme-primary));
  background: rgba(var(--v-theme-primary), 0.05);
}
.ws-option__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-top: 10px;
}
.ws-option__title {
  font-size: 15px;
  font-weight: 600;
}
.ws-option__desc {
  font-size: 13px;
  line-height: 1.45;
  color: rgb(var(--v-theme-text-muted));
}
</style>
