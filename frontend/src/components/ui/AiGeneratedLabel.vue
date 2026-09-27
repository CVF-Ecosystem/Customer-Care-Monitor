<!-- CCMAI-UX-000: marks AI-authored reviews and verdicts. -->
<template>
  <span class="ccma-ai-label" data-testid="ai-generated">
    <v-icon size="14" aria-hidden="true">mdi-robot-outline</v-icon>
    <span>{{ t('ui_ai_generated') }}</span>
  </span>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>

<style scoped>
.ccma-ai-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 500;
  color: rgb(var(--v-theme-text-muted));
}
</style>
