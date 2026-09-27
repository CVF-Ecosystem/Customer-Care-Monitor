<!--
  CCMAI-UX-000: channel sync status. "syncing" means started, not finished (R007).
  An unknown value never reads as success; error and partial are stronger than success.
-->
<template>
  <span class="ccma-sync" :class="`ccma-sync--${kind}`" :data-sync="kind" :title="kind === 'syncing' ? t('ui_sync_started_hint') : undefined">
    <v-icon size="16" aria-hidden="true">{{ ICONS[kind] }}</v-icon>
    <span>{{ t(`ui_sync_${kind}`) }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { syncKind, type SyncKind } from '../../utils/review'

const props = defineProps<{ status?: string | null }>()
const { t } = useI18n()

const ICONS: Record<SyncKind, string> = {
  never: 'mdi-sync-off',
  syncing: 'mdi-progress-clock',
  success: 'mdi-check',
  partial: 'mdi-alert-outline',
  error: 'mdi-alert-circle-outline',
  unknown: 'mdi-help-circle-outline',
}

const kind = computed(() => syncKind(props.status))
</script>

<style scoped>
.ccma-sync {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 10px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  line-height: 1;
  white-space: nowrap;
}
.ccma-sync--never,
.ccma-sync--unknown {
  background: rgb(var(--v-theme-skip-bg));
  color: rgb(var(--v-theme-skip));
}
.ccma-sync--syncing {
  background: rgba(var(--v-theme-primary), 0.1);
  color: rgb(var(--v-theme-primary));
}
.ccma-sync--success {
  color: rgb(var(--v-theme-pass));
  padding-left: 2px;
}
.ccma-sync--partial {
  background: rgb(var(--v-theme-src-changed-bg));
  color: rgb(var(--v-theme-src-changed));
  font-weight: 600;
}
.ccma-sync--error {
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
  font-weight: 600;
}
</style>
