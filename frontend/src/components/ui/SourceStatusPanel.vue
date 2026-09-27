<!--
  CCMAI-UX-000: "Trạng thái nguồn dữ liệu" panel. The local-only note is always rendered;
  there is deliberately no prop to hide it. Counts follow SOURCE_INTEGRITY_ORDER.
-->
<template>
  <section class="ccma-src-panel" :class="{ 'ccma-src-panel--alert': hasChanged }" :aria-label="t('ui_source_panel_title')">
    <div class="ccma-src-panel__head">
      <v-icon size="20" aria-hidden="true">{{ hasChanged ? 'mdi-alert-outline' : 'mdi-information-outline' }}</v-icon>
      <span class="ccma-src-panel__title">{{ t('ui_source_panel_title') }}</span>
    </div>
    <p class="ccma-src-panel__note" data-testid="source-note">{{ t('results_source_note') }}</p>
    <ul v-if="counts.length" class="ccma-src-panel__counts">
      <li v-for="c in counts" :key="c.status" :data-status="c.status">
        <SourceStatusChip :status="c.status" small />
        <span class="ccma-src-panel__count tabular-nums">{{ c.count }}</span>
      </li>
    </ul>
    <p v-else class="ccma-src-panel__empty">{{ t('ui_source_panel_empty') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import SourceStatusChip from './SourceStatusChip.vue'
import { countSourceStatuses } from '../../utils/review'

const props = defineProps<{ statuses: (string | null | undefined)[] }>()
const { t } = useI18n()

const counts = computed(() => countSourceStatuses(props.statuses))
const hasChanged = computed(() => counts.value.some((c) => c.status === 'changed_since_analysis'))
</script>

<style scoped>
.ccma-src-panel {
  border: 1px solid rgb(var(--v-theme-border));
  border-left: 4px solid rgb(var(--v-theme-text-muted));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
  padding: 12px 16px;
}
.ccma-src-panel--alert {
  border-left-color: rgb(var(--v-theme-src-changed));
}
.ccma-src-panel__head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.ccma-src-panel--alert .ccma-src-panel__head {
  color: rgb(var(--v-theme-src-changed));
}
.ccma-src-panel__title {
  font-size: 14px;
  font-weight: 600;
}
.ccma-src-panel__note {
  margin: 4px 0 0;
  font-size: 13px;
  line-height: 1.5;
  color: rgb(var(--v-theme-text-muted));
}
.ccma-src-panel__counts {
  list-style: none;
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;
  margin: 10px 0 0;
  padding: 0;
}
.ccma-src-panel__counts li {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.ccma-src-panel__count {
  font-size: 14px;
  font-weight: 600;
}
.ccma-src-panel__empty {
  margin: 8px 0 0;
  font-size: 13px;
  color: rgb(var(--v-theme-text-muted));
}
</style>
