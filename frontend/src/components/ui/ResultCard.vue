<!--
  CCMAI-UX-000: one result as a mobile card (list-style, one column). The whole card is a
  button; the source-status chip is always shown, including for legacy results.
-->
<template>
  <button type="button" class="ccma-rcard" :class="{ 'ccma-rcard--changed': changed }" @click="emit('open')">
    <span class="ccma-rcard__top">
      <span class="ccma-rcard__name">{{ customerName || '—' }}</span>
      <span class="ccma-rcard__time">{{ time }}</span>
    </span>
    <span class="ccma-rcard__chips">
      <VerdictChip :verdict="verdict" small />
      <SourceStatusChip :status="sourceStatus" small />
      <span v-if="score !== null && score !== undefined" class="ccma-rcard__score tabular-nums">{{ score }}/100</span>
    </span>
    <span v-if="summary" class="ccma-rcard__summary">{{ summary }}</span>
    <span v-if="meta" class="ccma-rcard__meta">{{ meta }}</span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import VerdictChip from './VerdictChip.vue'
import SourceStatusChip from './SourceStatusChip.vue'
import { normalizeSourceStatus, type Verdict } from '../../utils/review'

const props = defineProps<{
  customerName?: string | null
  time: string
  verdict: Verdict
  sourceStatus?: string | null
  summary?: string | null
  score?: number | null
  meta?: string | null
}>()
const emit = defineEmits<{ open: [] }>()

const changed = computed(() => normalizeSourceStatus(props.sourceStatus) === 'changed_since_analysis')
</script>

<style scoped>
.ccma-rcard {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
  min-height: 44px;
  padding: 14px 16px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
  color: rgb(var(--v-theme-on-surface));
  text-align: left;
  font: inherit;
  cursor: pointer;
}
.ccma-rcard:focus-visible {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: 2px;
}
.ccma-rcard--changed {
  border-left: 4px solid rgb(var(--v-theme-src-changed));
}
.ccma-rcard__top {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}
.ccma-rcard__name {
  font-size: 15px;
  font-weight: 600;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ccma-rcard__time {
  flex-shrink: 0;
  font-size: 12px;
  color: rgb(var(--v-theme-text-muted));
}
.ccma-rcard__chips {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.ccma-rcard__score {
  font-size: 13px;
  font-weight: 600;
}
.ccma-rcard__summary {
  font-size: 14px;
  line-height: 1.45;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.ccma-rcard__meta {
  font-size: 12px;
  color: rgb(var(--v-theme-text-muted));
}
</style>
