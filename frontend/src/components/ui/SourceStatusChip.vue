<!--
  CCMAI-UX-000: source-integrity chip for the four reviewed R004 statuses.
  Outlined style with icon and text; never green, never a check mark. A missing or
  unknown value renders as "verification_unavailable" instead of disappearing.
-->
<template>
  <span
    class="ccma-src"
    :class="[`ccma-src--${normalized}`, { 'ccma-src--small': small }]"
    :data-status="normalized"
    :title="note ? t('results_source_note') : undefined"
  >
    <v-icon :size="small ? 14 : 16" aria-hidden="true">{{ icon }}</v-icon>
    <span>{{ label }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { SOURCE_INTEGRITY_LABEL_KEY } from '../../stores/jobs'
import { SOURCE_STATUS_ICON, normalizeSourceStatus } from '../../utils/review'

const props = defineProps<{
  status?: string | null
  small?: boolean
  // Show the local-only note as a native tooltip (the note must also be visible elsewhere on the screen).
  note?: boolean
}>()
const { t } = useI18n()

const normalized = computed(() => normalizeSourceStatus(props.status))
const icon = computed(() => SOURCE_STATUS_ICON[normalized.value])
const label = computed(() => t(SOURCE_INTEGRITY_LABEL_KEY[normalized.value]))
</script>

<style scoped>
.ccma-src {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid currentColor;
  border-radius: 8px;
  background: rgb(var(--v-theme-surface));
  font-size: 13px;
  font-weight: 500;
  line-height: 1;
  white-space: nowrap;
}
.ccma-src--small {
  height: 24px;
  padding: 0 8px;
  font-size: 12px;
}
.ccma-src--changed_since_analysis {
  color: rgb(var(--v-theme-src-changed));
  border-width: 2px;
  font-weight: 600;
}
.ccma-src--verification_unavailable {
  color: rgb(var(--v-theme-src-unavailable));
}
.ccma-src--legacy_unverified {
  color: rgb(var(--v-theme-src-legacy));
  border-style: dashed;
}
.ccma-src--bound_currentness_unverified {
  color: rgb(var(--v-theme-src-legacy));
}
</style>
