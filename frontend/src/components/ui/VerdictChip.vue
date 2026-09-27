<!--
  CCMAI-UX-000: verdict chip. Filled style (status background + status text),
  always icon plus text, so it is told apart from the outlined source-status chip.
-->
<template>
  <span class="ccma-chip ccma-verdict" :class="[`ccma-verdict--${verdict}`, { 'ccma-chip--small': small }]" :data-verdict="verdict">
    <v-icon :size="small ? 14 : 16" aria-hidden="true">{{ icon }}</v-icon>
    <span>{{ label }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Verdict } from '../../utils/review'

const props = defineProps<{ verdict: Verdict; small?: boolean }>()
const { t } = useI18n()

const ICONS: Record<Verdict, string> = {
  pass: 'mdi-check-circle-outline',
  fail: 'mdi-close-circle-outline',
  skip: 'mdi-minus-circle-outline',
  classified: 'mdi-tag-outline',
}
const LABEL_KEYS: Record<Verdict, string> = {
  pass: 'verdict_pass',
  fail: 'verdict_fail',
  skip: 'verdict_skip',
  classified: 'ui_verdict_classified',
}

const icon = computed(() => ICONS[props.verdict] ?? ICONS.fail)
const label = computed(() => t(LABEL_KEYS[props.verdict] ?? LABEL_KEYS.fail))
</script>

<style scoped>
.ccma-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 10px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  line-height: 1;
  white-space: nowrap;
}
.ccma-chip--small {
  height: 24px;
  padding: 0 8px;
  font-size: 12px;
}
.ccma-verdict--pass {
  background: rgb(var(--v-theme-pass-bg));
  color: rgb(var(--v-theme-pass));
}
.ccma-verdict--fail {
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
}
.ccma-verdict--skip {
  background: rgb(var(--v-theme-skip-bg));
  color: rgb(var(--v-theme-skip));
}
.ccma-verdict--classified {
  background: rgba(var(--v-theme-primary), 0.1);
  color: rgb(var(--v-theme-primary));
}
</style>
