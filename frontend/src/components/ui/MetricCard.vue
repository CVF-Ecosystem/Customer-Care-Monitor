<!--
  CCMAI-UX-000: metric card. A null value renders "—" (unknown), never 0.
  With `to`, the whole card is a real link to the filtered list (drill-down).
-->
<template>
  <component
    :is="to ? RouterLink : 'div'"
    v-bind="to ? { to } : {}"
    class="ccma-metric"
    :class="[`ccma-metric--${tone ?? 'neutral'}`, { 'ccma-metric--link': !!to }]"
  >
    <span class="ccma-metric__label">{{ label }}</span>
    <span class="ccma-metric__value tabular-nums" data-testid="metric-value">{{ display }}<span v-if="suffix && display !== '—'" class="ccma-metric__suffix">{{ suffix }}</span></span>
    <span v-if="hint" class="ccma-metric__hint">{{ hint }}</span>
    <v-icon v-if="to" class="ccma-metric__arrow" size="18" aria-hidden="true">mdi-arrow-right</v-icon>
  </component>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { formatNumber, type UiLocale } from '../../utils/format'

const props = defineProps<{
  label: string
  value: number | null | undefined
  to?: RouteLocationRaw
  hint?: string
  tone?: 'neutral' | 'fail' | 'warning' | 'pass'
  fractionDigits?: number
  // Unit shown after a known value only, e.g. "%" or "/100" (never after "—").
  suffix?: string
}>()
const { locale } = useI18n()

const display = computed(() => formatNumber(props.value ?? null, locale.value as UiLocale, props.fractionDigits ?? 0))
</script>

<style scoped>
.ccma-metric {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-height: 96px;
  padding: 16px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
  color: rgb(var(--v-theme-on-surface));
  text-decoration: none;
}
.ccma-metric--link:hover {
  border-color: rgb(var(--v-theme-primary));
}
.ccma-metric--link:focus-visible {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: 2px;
}
.ccma-metric__label {
  font-size: 13px;
  font-weight: 500;
  color: rgb(var(--v-theme-text-muted));
  padding-right: 24px;
}
.ccma-metric__value {
  font-size: 30px;
  font-weight: 700;
  line-height: 1.15;
}
.ccma-metric__suffix {
  margin-left: 2px;
  font-size: 16px;
  font-weight: 500;
  color: rgb(var(--v-theme-text-muted));
}
.ccma-metric--fail .ccma-metric__value {
  color: rgb(var(--v-theme-fail));
}
.ccma-metric--warning .ccma-metric__value {
  color: rgb(var(--v-theme-src-changed));
}
.ccma-metric--pass .ccma-metric__value {
  color: rgb(var(--v-theme-pass));
}
.ccma-metric__hint {
  font-size: 12px;
  color: rgb(var(--v-theme-text-muted));
}
.ccma-metric__arrow {
  position: absolute;
  top: 16px;
  right: 14px;
  color: rgb(var(--v-theme-text-muted));
}
</style>
