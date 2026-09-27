<!--
  CCMAI-UX-000: confidence display (R005). Unknown renders "Không có", never 0% or 100%;
  a value always carries the "model self-estimate, not calibrated" qualifier.
-->
<template>
  <span class="ccma-conf" data-testid="confidence">
    <span class="ccma-conf__label">{{ t('ui_confidence') }}:</span>
    <template v-if="percent === null">
      <span class="ccma-conf__value">{{ t('ui_confidence_none') }}</span>
    </template>
    <template v-else>
      <span class="ccma-conf__value tabular-nums">{{ percent }}%</span>
      <span class="ccma-conf__qualifier">· {{ t('ui_confidence_qualifier') }}</span>
    </template>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { confidencePercent } from '../../utils/review'

const props = defineProps<{ confidence: number | null | undefined; basis?: string | null }>()
const { t } = useI18n()

const percent = computed(() => confidencePercent(props.confidence, props.basis))
</script>

<style scoped>
.ccma-conf {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 4px;
  font-size: 13px;
}
.ccma-conf__label,
.ccma-conf__qualifier {
  color: rgb(var(--v-theme-text-muted));
}
.ccma-conf__value {
  font-weight: 600;
}
</style>
